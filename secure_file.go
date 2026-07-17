package main

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	privateDirectoryMode = 0700
	privateFileMode      = 0600
)

// writePrivateFileAtomically replaces path without exposing a partially written
// settings or history file. The temporary file is created in the same directory
// so the final rename remains atomic on the local filesystem.
func writePrivateFileAtomically(path string, data []byte) (err error) {
	if path == "" {
		return fmt.Errorf("private file path is empty")
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, privateDirectoryMode); err != nil {
		return fmt.Errorf("create private directory: %w", err)
	}
	if err := os.Chmod(dir, privateDirectoryMode); err != nil {
		return fmt.Errorf("restrict private directory permissions: %w", err)
	}

	tmp, err := os.CreateTemp(dir, ".shotgun-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary private file: %w", err)
	}
	tmpName := tmp.Name()
	defer func() {
		_ = tmp.Close()
		if err != nil {
			_ = os.Remove(tmpName)
		}
	}()

	if err = tmp.Chmod(privateFileMode); err != nil {
		return fmt.Errorf("restrict temporary file permissions: %w", err)
	}
	if _, err = tmp.Write(data); err != nil {
		return fmt.Errorf("write temporary private file: %w", err)
	}
	if err = tmp.Sync(); err != nil {
		return fmt.Errorf("sync temporary private file: %w", err)
	}
	if err = tmp.Close(); err != nil {
		return fmt.Errorf("close temporary private file: %w", err)
	}
	if err = os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("replace private file: %w", err)
	}
	if err = os.Chmod(path, privateFileMode); err != nil {
		return fmt.Errorf("restrict private file permissions: %w", err)
	}
	return nil
}

func restrictPrivateFile(path string) error {
	if err := os.Chmod(path, privateFileMode); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("restrict private file permissions: %w", err)
	}
	return nil
}
