package main

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	gitignore "github.com/sabhiram/go-gitignore"
)

const watchmanEventDebounce = 400 * time.Millisecond

type Watchman struct {
	app         *App
	rootDir     string
	fsWatcher   *fsnotify.Watcher
	watchedDirs map[string]bool
	mu          sync.Mutex
	cancelFunc  context.CancelFunc

	currentProjectGitignore *gitignore.GitIgnore
	currentCustomPatterns   *gitignore.GitIgnore
}

func NewWatchman(app *App) *Watchman {
	return &Watchman{
		app:         app,
		watchedDirs: make(map[string]bool),
	}
}

func (a *App) StartFileWatcher(rootDirPath string) error {
	safeLogInfof(a.ctx, "StartFileWatcher called for: %s", rootDirPath)
	if a.fileWatcher == nil {
		return fmt.Errorf("file watcher not initialized")
	}
	return a.fileWatcher.Start(rootDirPath)
}

func (a *App) StopFileWatcher() error {
	safeLogInfo(a.ctx, "StopFileWatcher called")
	if a.fileWatcher == nil {
		return fmt.Errorf("file watcher not initialized")
	}
	a.fileWatcher.Stop()
	return nil
}

func (w *Watchman) Start(newRootDir string) error {
	w.Stop()

	w.mu.Lock()
	w.rootDir = newRootDir
	if w.rootDir == "" {
		w.mu.Unlock()
		safeLogInfo(w.app.ctx, "Watchman: Root directory is empty, not starting.")
		return nil
	}
	w.mu.Unlock()

	if w.app.useGitignore {
		w.currentProjectGitignore = w.app.projectGitignore
	} else {
		w.currentProjectGitignore = nil
	}
	if w.app.useCustomIgnore {
		w.currentCustomPatterns = w.app.currentCustomIgnorePatterns
	} else {
		w.currentCustomPatterns = nil
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		safeLogErrorf(w.app.ctx, "Watchman: Error creating fsnotify watcher: %v", err)
		return fmt.Errorf("failed to create fsnotify watcher: %w", err)
	}

	ctx, cancel := context.WithCancel(w.app.ctx)

	w.mu.Lock()
	w.cancelFunc = cancel
	w.fsWatcher = watcher
	w.watchedDirs = make(map[string]bool)
	w.mu.Unlock()

	safeLogInfof(w.app.ctx, "Watchman: Starting for directory %s", newRootDir)
	w.addPathsToWatcherRecursive(newRootDir)

	go w.run(ctx)
	return nil
}

func normalizeWatchmanRelPath(path string) string {
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

func matchesIgnore(ign *gitignore.GitIgnore, relPath string, isDir bool) bool {
	if ign == nil {
		return false
	}
	normalized := normalizeWatchmanRelPath(relPath)
	if normalized == "" {
		return false
	}
	if ign.MatchesPath(normalized) {
		return true
	}
	// Patterns like "node_modules/" in go-gitignore require trailing slash for the directory itself.
	if isDir && !strings.HasSuffix(normalized, "/") {
		return ign.MatchesPath(normalized + "/")
	}
	return false
}

func (w *Watchman) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.cancelFunc != nil {
		safeLogInfo(w.app.ctx, "Watchman: Stopping...")
		w.cancelFunc()
		w.cancelFunc = nil
	}
	if w.fsWatcher != nil {
		if err := w.fsWatcher.Close(); err != nil {
			safeLogWarningf(w.app.ctx, "Watchman: Error closing fsnotify watcher: %v", err)
		}
		w.fsWatcher = nil
	}
	w.rootDir = ""
	w.watchedDirs = make(map[string]bool)
}

func (w *Watchman) run(ctx context.Context) {
	w.mu.Lock()
	currentRootDir := w.rootDir
	watcher := w.fsWatcher
	w.mu.Unlock()
	if watcher == nil {
		return
	}

	var notifyTimer *time.Timer
	var notifyTimerC <-chan time.Time
	var pendingNotifyRootDir string
	defer func() {
		if notifyTimer == nil {
			return
		}
		if !notifyTimer.Stop() {
			select {
			case <-notifyTimer.C:
			default:
			}
		}
	}()

	safeLogInfof(w.app.ctx, "Watchman: Monitoring goroutine started for %s", currentRootDir)
	for {
		select {
		case <-ctx.Done():
			safeLogInfof(w.app.ctx, "Watchman: Context cancelled, shutting down watcher for %s.", currentRootDir)
			return
		case <-notifyTimerC:
			if pendingNotifyRootDir != "" {
				w.app.notifyFileChange(pendingNotifyRootDir)
				pendingNotifyRootDir = ""
			}
			notifyTimerC = nil
		case event, ok := <-watcher.Events:
			if !ok {
				safeLogInfo(w.app.ctx, "Watchman: fsnotify events channel closed.")
				return
			}

			w.mu.Lock()
			currentRootDir = w.rootDir
			projIgn := w.currentProjectGitignore
			custIgn := w.currentCustomPatterns
			w.mu.Unlock()
			if currentRootDir == "" {
				continue
			}

			relEventPath, err := filepath.Rel(currentRootDir, event.Name)
			if err != nil {
				safeLogWarningf(w.app.ctx, "Watchman: Could not get relative path for event %s (root: %s): %v", event.Name, currentRootDir, err)
				continue
			}
			isDir := false
			if info, statErr := os.Stat(event.Name); statErr == nil {
				isDir = info.IsDir()
			}
			if matchesIgnore(projIgn, relEventPath, isDir) || matchesIgnore(custIgn, relEventPath, isDir) {
				continue
			}

			if event.Op&fsnotify.Chmod == 0 {
				pendingNotifyRootDir = currentRootDir
				if notifyTimer == nil {
					notifyTimer = time.NewTimer(watchmanEventDebounce)
				} else {
					if !notifyTimer.Stop() {
						select {
						case <-notifyTimer.C:
						default:
						}
					}
					notifyTimer.Reset(watchmanEventDebounce)
				}
				notifyTimerC = notifyTimer.C
			}

			if event.Op&fsnotify.Create != 0 {
				if info, statErr := os.Stat(event.Name); statErr == nil && info.IsDir() {
					w.addPathsToWatcherRecursive(event.Name)
				}
			}

			if event.Op&fsnotify.Remove != 0 || event.Op&fsnotify.Rename != 0 {
				w.mu.Lock()
				if w.watchedDirs[event.Name] {
					if w.fsWatcher != nil {
						_ = w.fsWatcher.Remove(event.Name)
					}
					delete(w.watchedDirs, event.Name)
				}
				w.mu.Unlock()
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				safeLogInfo(w.app.ctx, "Watchman: fsnotify errors channel closed.")
				return
			}
			safeLogErrorf(w.app.ctx, "Watchman: fsnotify error: %v", err)
		}
	}
}

func (w *Watchman) addPathsToWatcherRecursive(baseDirToAdd string) {
	w.mu.Lock()
	fsW := w.fsWatcher
	projIgn := w.currentProjectGitignore
	custIgn := w.currentCustomPatterns
	overallRoot := w.rootDir
	w.mu.Unlock()

	if fsW == nil || overallRoot == "" {
		safeLogWarningf(w.app.ctx, "Watchman.addPathsToWatcherRecursive: fsWatcher is nil or rootDir is empty. Skipping add for %s.", baseDirToAdd)
		return
	}

	_ = filepath.WalkDir(baseDirToAdd, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			safeLogWarningf(w.app.ctx, "Watchman scan error accessing %s: %v", path, walkErr)
			if d != nil && d.IsDir() && path != overallRoot {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.IsDir() {
			return nil
		}

		relPath, errRel := filepath.Rel(overallRoot, path)
		if errRel != nil {
			return nil
		}
		if d.Name() == ".git" && filepath.Dir(path) == overallRoot {
			return filepath.SkipDir
		}
		if matchesIgnore(projIgn, relPath, true) || matchesIgnore(custIgn, relPath, true) {
			return filepath.SkipDir
		}
		if errAdd := fsW.Add(path); errAdd == nil {
			w.mu.Lock()
			w.watchedDirs[path] = true
			w.mu.Unlock()
		}
		return nil
	})
}

func (a *App) notifyFileChange(rootDir string) {
	safeEventsEmit(a.ctx, "projectFilesChanged", rootDir)
}

func (w *Watchman) RefreshIgnoresAndRescan() error {
	w.mu.Lock()
	rootDir := w.rootDir
	w.mu.Unlock()
	if rootDir == "" {
		safeLogInfo(w.app.ctx, "Watchman.RefreshIgnoresAndRescan: No rootDir, skipping.")
		return nil
	}

	if w.app.useGitignore {
		w.currentProjectGitignore = w.app.projectGitignore
	} else {
		w.currentProjectGitignore = nil
	}
	if w.app.useCustomIgnore {
		w.currentCustomPatterns = w.app.currentCustomIgnorePatterns
	} else {
		w.currentCustomPatterns = nil
	}

	w.Stop()
	return w.Start(rootDir)
}
