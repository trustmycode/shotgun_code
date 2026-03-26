package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gofrs/flock"
)

const (
	fileLockTimeout      = 500 * time.Millisecond
	fileLockRetryBackoff = 25 * time.Millisecond
)

func ensureParentDir(path string) error {
	dir := filepath.Dir(path)
	if dir == "." || dir == "" {
		return nil
	}
	return os.MkdirAll(dir, os.ModePerm)
}

func withFileLock(lockPath string, shared bool, fn func() error) error {
	if err := ensureParentDir(lockPath); err != nil {
		return fmt.Errorf("failed to create lock dir for %s: %w", lockPath, err)
	}

	lock := flock.New(lockPath)
	deadline := time.Now().Add(fileLockTimeout)

	for {
		var (
			locked bool
			err    error
		)

		if shared {
			locked, err = lock.TryRLock()
		} else {
			locked, err = lock.TryLock()
		}
		if err != nil {
			return fmt.Errorf("failed to acquire lock %s: %w", lockPath, err)
		}

		if locked {
			defer func() {
				_ = lock.Unlock()
			}()
			return fn()
		}

		if time.Now().After(deadline) {
			lockKind := "exclusive"
			if shared {
				lockKind = "shared"
			}
			return fmt.Errorf("timed out acquiring %s lock for %s after %s", lockKind, lockPath, fileLockTimeout)
		}

		time.Sleep(fileLockRetryBackoff)
	}
}
