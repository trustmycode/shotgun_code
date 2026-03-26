package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type ContextGenerator struct {
	app                *App
	mu                 sync.Mutex
	currentCancelFunc  context.CancelFunc
	currentCancelToken interface{}
}

func NewContextGenerator(app *App) *ContextGenerator {
	return &ContextGenerator{app: app}
}

func (cg *ContextGenerator) requestShotgunContextGenerationInternal(rootDir string, excludedPaths []string) {
	cg.mu.Lock()
	if cg.currentCancelFunc != nil {
		safeLogDebug(cg.app.ctx, "Cancelling previous context generation job.")
		cg.currentCancelFunc()
	}

	genCtx, cancel := context.WithCancel(cg.app.ctx)
	myToken := new(struct{})
	cg.currentCancelFunc = cancel
	cg.currentCancelToken = myToken
	safeLogInfof(cg.app.ctx, "Starting new shotgun context generation for: %s. Max size: %d bytes.", rootDir, maxOutputSizeBytes)
	cg.mu.Unlock()

	go func(tokenForThisJob interface{}) {
		jobStartTime := time.Now()
		defer func() {
			cg.mu.Lock()
			if cg.currentCancelToken == tokenForThisJob {
				cg.currentCancelFunc = nil
				cg.currentCancelToken = nil
				safeLogDebug(cg.app.ctx, "Cleared currentCancelFunc for completed/cancelled job (token match).")
			} else {
				safeLogDebug(cg.app.ctx, "currentCancelFunc was replaced by a newer job (token mismatch); not clearing.")
			}
			cg.mu.Unlock()
			safeLogInfof(cg.app.ctx, "Shotgun context generation goroutine finished in %s", time.Since(jobStartTime))
		}()

		if genCtx.Err() != nil {
			safeLogInfo(cg.app.ctx, fmt.Sprintf("Context generation for %s cancelled before starting: %v", rootDir, genCtx.Err()))
			return
		}

		output, err := cg.app.generateShotgunOutputWithProgress(genCtx, rootDir, excludedPaths)

		select {
		case <-genCtx.Done():
			errMsg := fmt.Sprintf("Shotgun context generation cancelled for %s: %v", rootDir, genCtx.Err())
			safeLogInfo(cg.app.ctx, errMsg)
			safeEventsEmit(cg.app.ctx, "shotgunContextError", errMsg)
		default:
			if err != nil {
				errMsg := fmt.Sprintf("Error generating shotgun output for %s: %v", rootDir, err)
				safeLogError(cg.app.ctx, errMsg)
				safeEventsEmit(cg.app.ctx, "shotgunContextError", errMsg)
			} else {
				finalSize := len(output)
				successMsg := fmt.Sprintf("Shotgun context generated successfully for %s. Size: %d bytes.", rootDir, finalSize)
				if finalSize > maxOutputSizeBytes {
					safeLogWarningf(cg.app.ctx, "Warning: Generated context size %d exceeds max %d, but was not caught by ErrContextTooLong.", finalSize, maxOutputSizeBytes)
				}
				safeLogInfo(cg.app.ctx, successMsg)
				safeEventsEmit(cg.app.ctx, "shotgunContextGenerated", output)
			}
		}
	}(myToken)
}

func (a *App) RequestShotgunContextGeneration(rootDir string, excludedPaths []string) {
	if a.contextGenerator == nil {
		safeLogError(a.ctx, "ContextGenerator not initialized")
		safeEventsEmit(a.ctx, "shotgunContextError", "Internal error: ContextGenerator not initialized")
		return
	}
	a.contextGenerator.requestShotgunContextGenerationInternal(rootDir, excludedPaths)
}

func (a *App) countProcessableItems(jobCtx context.Context, rootDir string, excludedMap map[string]bool) (int, error) {
	count := 1

	var counterHelper func(currentPath string) error
	counterHelper = func(currentPath string) error {
		select {
		case <-jobCtx.Done():
			return jobCtx.Err()
		default:
		}

		entries, err := os.ReadDir(currentPath)
		if err != nil {
			safeLogWarningf(a.ctx, "countProcessableItems: error reading dir %s: %v", currentPath, err)
			return nil
		}

		for _, entry := range entries {
			path := filepath.Join(currentPath, entry.Name())
			relPath, _ := filepath.Rel(rootDir, path)
			relPathNormalized := normalizeContextRelPath(relPath)

			if excludedMap[relPathNormalized] {
				continue
			}

			count++
			if entry.IsDir() {
				if err := counterHelper(path); err != nil {
					return err
				}
			} else {
				count++
			}
		}
		return nil
	}

	if err := counterHelper(rootDir); err != nil {
		return 0, err
	}
	return count, nil
}

type generationProgressState struct {
	processedItems   int
	totalItems       int
	lastEmittedItems int
	lastEmittedAt    time.Time
}

const (
	progressEmitMinInterval  = 100 * time.Millisecond
	progressEmitItemInterval = 50
)

func shouldEmitProgress(state *generationProgressState, now time.Time, force bool) bool {
	if force || state.processedItems == 0 || state.lastEmittedAt.IsZero() {
		return true
	}
	if state.processedItems-state.lastEmittedItems >= progressEmitItemInterval {
		return true
	}
	return now.Sub(state.lastEmittedAt) >= progressEmitMinInterval
}

func (a *App) emitProgress(state *generationProgressState, force bool) {
	now := time.Now()
	if !shouldEmitProgress(state, now, force) {
		return
	}
	safeEventsEmit(a.ctx, "shotgunContextGenerationProgress", map[string]int{
		"current": state.processedItems,
		"total":   state.totalItems,
	})
	state.lastEmittedItems = state.processedItems
	state.lastEmittedAt = now
}

func readFileWithLimit(path string, maxBytes int) ([]byte, bool, error) {
	if maxBytes < 0 {
		return nil, true, nil
	}

	info, err := os.Stat(path)
	if err != nil {
		return nil, false, err
	}
	if info.Size() > int64(maxBytes) {
		return nil, true, nil
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, false, err
	}
	defer file.Close()

	reader := io.LimitReader(file, int64(maxBytes)+1)
	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, false, err
	}
	if len(content) > maxBytes {
		return nil, true, nil
	}
	return content, false, nil
}

func isTextLikeContentType(contentType string) bool {
	if strings.HasPrefix(contentType, "text/") {
		return true
	}

	textLikeTypes := []string{
		"application/json",
		"application/xml",
		"application/javascript",
		"application/x-javascript",
		"application/x-sh",
		"application/x-ndjson",
		"application/x-yaml",
		"application/yaml",
		"application/toml",
	}

	for _, t := range textLikeTypes {
		if strings.Contains(contentType, t) {
			return true
		}
	}

	return false
}

func isBinaryFile(path string) (bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer file.Close()

	header := make([]byte, 512)
	n, err := file.Read(header)
	if err != nil && !errors.Is(err, io.EOF) {
		return false, err
	}
	if n == 0 {
		return false, nil
	}

	sample := header[:n]
	contentType := http.DetectContentType(sample)
	if bytes.IndexByte(sample, 0) >= 0 && !isTextLikeContentType(contentType) {
		return true, nil
	}

	if isTextLikeContentType(contentType) {
		return false, nil
	}
	if strings.HasPrefix(contentType, "application/") {
		return true, nil
	}
	if strings.HasPrefix(contentType, "image/") || strings.HasPrefix(contentType, "audio/") || strings.HasPrefix(contentType, "video/") || strings.HasPrefix(contentType, "font/") {
		return true, nil
	}

	return false, nil
}

func normalizeContextRelPath(path string) string {
	p := strings.TrimSpace(path)
	if p == "" || p == "." {
		return ""
	}
	p = strings.ReplaceAll(p, "\\", "/")
	p = filepath.ToSlash(p)
	p = strings.TrimPrefix(p, "./")
	p = strings.TrimPrefix(p, "/")
	return p
}

func (a *App) generateShotgunOutputWithProgress(jobCtx context.Context, rootDir string, excludedPaths []string) (string, error) {
	if err := jobCtx.Err(); err != nil {
		return "", err
	}

	excludedMap := make(map[string]bool)
	for _, p := range excludedPaths {
		normalized := normalizeContextRelPath(p)
		if normalized != "" {
			excludedMap[normalized] = true
		}
	}

	totalItems, err := a.countProcessableItems(jobCtx, rootDir, excludedMap)
	if err != nil {
		return "", fmt.Errorf("failed to count processable items: %w", err)
	}
	progressState := &generationProgressState{processedItems: 0, totalItems: totalItems}
	a.emitProgress(progressState, false)

	var output strings.Builder
	var fileContents strings.Builder

	output.WriteString(filepath.Base(rootDir) + string(os.PathSeparator) + "\n")
	progressState.processedItems++
	a.emitProgress(progressState, progressState.processedItems >= progressState.totalItems)
	if output.Len() > maxOutputSizeBytes {
		return "", fmt.Errorf("%w: content limit of %d bytes exceeded after root dir line (size: %d bytes)", ErrContextTooLong, maxOutputSizeBytes, output.Len())
	}

	var buildShotgunTreeRecursive func(pCtx context.Context, currentPath, prefix string) error
	buildShotgunTreeRecursive = func(pCtx context.Context, currentPath, prefix string) error {
		select {
		case <-pCtx.Done():
			return pCtx.Err()
		default:
		}

		entries, err := os.ReadDir(currentPath)
		if err != nil {
			safeLogWarningf(a.ctx, "buildShotgunTreeRecursive: error reading dir %s: %v", currentPath, err)
			return nil
		}

		sort.SliceStable(entries, func(i, j int) bool {
			entryI := entries[i]
			entryJ := entries[j]
			isDirI := entryI.IsDir()
			isDirJ := entryJ.IsDir()
			if isDirI && !isDirJ {
				return true
			}
			if !isDirI && isDirJ {
				return false
			}
			return strings.ToLower(entryI.Name()) < strings.ToLower(entryJ.Name())
		})

		var visibleEntries []fs.DirEntry
		for _, entry := range entries {
			path := filepath.Join(currentPath, entry.Name())
			relPath, _ := filepath.Rel(rootDir, path)
			relPathNormalized := normalizeContextRelPath(relPath)
			if !excludedMap[relPathNormalized] {
				visibleEntries = append(visibleEntries, entry)
			}
		}

		for i, entry := range visibleEntries {
			select {
			case <-pCtx.Done():
				return pCtx.Err()
			default:
			}

			path := filepath.Join(currentPath, entry.Name())
			relPath, _ := filepath.Rel(rootDir, path)

			isLast := i == len(visibleEntries)-1
			branch := "├── "
			nextPrefix := prefix + "│   "
			if isLast {
				branch = "└── "
				nextPrefix = prefix + "    "
			}
			output.WriteString(prefix + branch + entry.Name() + "\n")

			progressState.processedItems++
			a.emitProgress(progressState, progressState.processedItems >= progressState.totalItems)

			if output.Len()+fileContents.Len() > maxOutputSizeBytes {
				return fmt.Errorf("%w: content limit of %d bytes exceeded during tree generation (size: %d bytes)", ErrContextTooLong, maxOutputSizeBytes, output.Len()+fileContents.Len())
			}

			if entry.IsDir() {
				err := buildShotgunTreeRecursive(pCtx, path, nextPrefix)
				if err != nil {
					if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
						return err
					}
					safeLogWarningf(a.ctx, "Error processing subdirectory %s: %v", path, err)
				}
			} else {
				select {
				case <-pCtx.Done():
					return pCtx.Err()
				default:
				}

				relPathForwardSlash := filepath.ToSlash(relPath)
				fileOpenTag := fmt.Sprintf("<file path=\"%s\">\n", relPathForwardSlash)
				fileCloseTag := "\n</file>\n"
				currentSize := output.Len() + fileContents.Len()
				remainingBudget := maxOutputSizeBytes - currentSize
				contentBudget := remainingBudget - len(fileOpenTag) - len(fileCloseTag)
				if contentBudget < 0 {
					return fmt.Errorf("%w: content limit of %d bytes exceeded before appending file %s metadata (total size: %d bytes)", ErrContextTooLong, maxOutputSizeBytes, relPath, currentSize)
				}

				content := []byte{}
				isBinary, binaryErr := isBinaryFile(path)
				if binaryErr != nil {
					safeLogWarningf(a.ctx, "Error checking file type for %s: %v", path, binaryErr)
				}

				if isBinary {
					content = []byte("[Binary file content omitted]")
					if len(content) > contentBudget {
						return fmt.Errorf("%w: content limit of %d bytes exceeded while handling binary file %s (current size: %d bytes, file budget: %d bytes)", ErrContextTooLong, maxOutputSizeBytes, relPath, currentSize, contentBudget)
					}
				} else {
					readContent, tooLong, err := readFileWithLimit(path, contentBudget)
					if tooLong {
						return fmt.Errorf("%w: content limit of %d bytes exceeded while reading file %s (current size: %d bytes, file budget: %d bytes)", ErrContextTooLong, maxOutputSizeBytes, relPath, currentSize, contentBudget)
					}
					if err != nil {
						safeLogWarningf(a.ctx, "Error reading file %s: %v", path, err)
						content = []byte(fmt.Sprintf("Error reading file: %v", err))
					} else {
						content = readContent
					}
				}

				fileContents.WriteString(fileOpenTag)
				fileContents.WriteString(string(content))
				fileContents.WriteString(fileCloseTag)

				progressState.processedItems++
				a.emitProgress(progressState, progressState.processedItems >= progressState.totalItems)

				if output.Len()+fileContents.Len() > maxOutputSizeBytes {
					return fmt.Errorf("%w: content limit of %d bytes exceeded after appending file %s (total size: %d bytes)", ErrContextTooLong, maxOutputSizeBytes, relPath, output.Len()+fileContents.Len())
				}
			}
		}
		return nil
	}

	if err := buildShotgunTreeRecursive(jobCtx, rootDir, ""); err != nil {
		return "", fmt.Errorf("failed to build tree for shotgun: %w", err)
	}

	if err := jobCtx.Err(); err != nil {
		return "", err
	}

	return output.String() + "\n" + strings.TrimRight(fileContents.String(), "\n"), nil
}
