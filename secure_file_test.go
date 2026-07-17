package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWritePrivateFileAtomically(t *testing.T) {
	path := filepath.Join(t.TempDir(), "shotgun-code", "settings.json")
	want := []byte(`{"value":"secret"}`)

	if err := writePrivateFileAtomically(path, want); err != nil {
		t.Fatalf("writePrivateFileAtomically() error = %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(got) != string(want) {
		t.Fatalf("file content = %q, want %q", got, want)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if gotMode := info.Mode().Perm(); gotMode != privateFileMode {
		t.Fatalf("file mode = %o, want %o", gotMode, privateFileMode)
	}
}
