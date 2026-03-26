package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	gitignore "github.com/sabhiram/go-gitignore"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type FileNode struct {
	Name            string      `json:"name"`
	Path            string      `json:"path"`
	RelPath         string      `json:"relPath"`
	IsDir           bool        `json:"isDir"`
	Children        []*FileNode `json:"children,omitempty"`
	IsGitignored    bool        `json:"isGitignored"`
	IsCustomIgnored bool        `json:"isCustomIgnored"`
}

func (a *App) SelectDirectory() (string, error) {
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{})
}

func (a *App) ListFiles(dirPath string) ([]*FileNode, error) {
	safeLogDebugf(a.ctx, "ListFiles called for directory: %s", dirPath)

	a.projectGitignore = nil
	var gitIgn *gitignore.GitIgnore
	gitignorePath := filepath.Join(dirPath, ".gitignore")
	safeLogDebugf(a.ctx, "Attempting to find .gitignore at: %s", gitignorePath)
	if _, err := os.Stat(gitignorePath); err == nil {
		safeLogDebugf(a.ctx, ".gitignore found at: %s", gitignorePath)
		gitIgn, err = gitignore.CompileIgnoreFile(gitignorePath)
		if err != nil {
			safeLogWarningf(a.ctx, "Error compiling .gitignore file at %s: %v", gitignorePath, err)
			gitIgn = nil
		} else {
			a.projectGitignore = gitIgn
			safeLogDebug(a.ctx, ".gitignore compiled successfully.")
		}
	} else {
		safeLogDebugf(a.ctx, ".gitignore not found at %s (os.Stat error: %v)", gitignorePath, err)
		gitIgn = nil
	}

	rootNode := &FileNode{
		Name:            filepath.Base(dirPath),
		Path:            dirPath,
		RelPath:         ".",
		IsDir:           true,
		IsGitignored:    false,
		IsCustomIgnored: a.currentCustomIgnorePatterns != nil && a.currentCustomIgnorePatterns.MatchesPath("."),
	}

	children, err := buildTreeRecursive(context.TODO(), dirPath, dirPath, gitIgn, a.currentCustomIgnorePatterns, 0)
	if err != nil {
		return []*FileNode{rootNode}, fmt.Errorf("error building children tree for %s: %w", dirPath, err)
	}
	rootNode.Children = children

	return []*FileNode{rootNode}, nil
}

func buildTreeRecursive(ctx context.Context, currentPath, rootPath string, gitIgn *gitignore.GitIgnore, customIgn *gitignore.GitIgnore, depth int) ([]*FileNode, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	entries, err := os.ReadDir(currentPath)
	if err != nil {
		return nil, err
	}

	var nodes []*FileNode
	for _, entry := range entries {
		nodePath := filepath.Join(currentPath, entry.Name())
		relPath, _ := filepath.Rel(rootPath, nodePath)

		isGitignored := false
		isCustomIgnored := false
		pathToMatch := relPath
		if entry.IsDir() && !strings.HasSuffix(pathToMatch, string(os.PathSeparator)) {
			pathToMatch += string(os.PathSeparator)
		}

		if gitIgn != nil {
			isGitignored = gitIgn.MatchesPath(pathToMatch)
		}
		if customIgn != nil {
			isCustomIgnored = customIgn.MatchesPath(pathToMatch)
		}

		node := &FileNode{
			Name:            entry.Name(),
			Path:            nodePath,
			RelPath:         relPath,
			IsDir:           entry.IsDir(),
			IsGitignored:    isGitignored,
			IsCustomIgnored: isCustomIgnored,
		}

		if entry.IsDir() && !isCustomIgnored {
			children, err := buildTreeRecursive(ctx, nodePath, rootPath, gitIgn, customIgn, depth+1)
			if err != nil {
				if errors.Is(err, context.Canceled) {
					return nil, err
				}
				safeLogWarningf(context.Background(), "Error building subtree for %s: %v", nodePath, err)
			} else {
				node.Children = children
			}
		}

		nodes = append(nodes, node)
	}

	sort.SliceStable(nodes, func(i, j int) bool {
		if nodes[i].IsDir && !nodes[j].IsDir {
			return true
		}
		if !nodes[i].IsDir && nodes[j].IsDir {
			return false
		}
		return strings.ToLower(nodes[i].Name) < strings.ToLower(nodes[j].Name)
	})
	return nodes, nil
}

func (a *App) SaveRepoScan(rootDir string, content string) error {
	if rootDir == "" {
		return errors.New("root directory is empty")
	}
	filePath := filepath.Join(rootDir, "shotgun_reposcan.md")
	return os.WriteFile(filePath, []byte(content), 0644)
}

func (a *App) LoadRepoScan(rootDir string) (string, error) {
	if rootDir == "" {
		return "", errors.New("root directory is empty")
	}
	filePath := filepath.Join(rootDir, "shotgun_reposcan.md")
	content, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return string(content), nil
}
