package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCountProcessableItems(t *testing.T) {
	app := newTestApp(t)
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "src"), 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("ok"), 0o644); err != nil {
		t.Fatalf("write failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "src", "main.go"), []byte("package main"), 0o644); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	count, err := app.countProcessableItems(context.Background(), root, map[string]bool{"README.md": true})
	if err != nil {
		t.Fatalf("countProcessableItems failed: %v", err)
	}
	if count < 4 {
		t.Fatalf("expected count >= 4, got %d", count)
	}
}

func TestGenerateShotgunOutputWithProgress(t *testing.T) {
	app := newTestApp(t)
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "src"), 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "src", "keep.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("write failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "src", "skip.go"), []byte("skip"), 0o644); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	out, err := app.generateShotgunOutputWithProgress(context.Background(), root, []string{"src/skip.go"})
	if err != nil {
		t.Fatalf("generateShotgunOutputWithProgress failed: %v", err)
	}
	if !strings.Contains(out, "<file path=\"src/keep.go\">") {
		t.Fatalf("expected keep.go in output, got: %q", out)
	}
	if strings.Contains(out, "src/skip.go") {
		t.Fatalf("excluded file must not appear in output: %q", out)
	}
}

func TestGenerateShotgunOutputWithProgressCancelled(t *testing.T) {
	app := newTestApp(t)
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("a"), 0o644); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := app.generateShotgunOutputWithProgress(ctx, root, nil)
	if err == nil {
		t.Fatal("expected cancellation error")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestGenerateShotgunOutputWithProgressTooLong(t *testing.T) {
	app := newTestApp(t)
	root := t.TempDir()
	big := strings.Repeat("x", maxOutputSizeBytes+1024)
	if err := os.WriteFile(filepath.Join(root, "huge.txt"), []byte(big), 0o644); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	_, err := app.generateShotgunOutputWithProgress(context.Background(), root, nil)
	if err == nil {
		t.Fatal("expected ErrContextTooLong")
	}
	if !errors.Is(err, ErrContextTooLong) {
		t.Fatalf("expected ErrContextTooLong, got %v", err)
	}
}

func TestReadFileWithLimit(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "small.txt")
	if err := os.WriteFile(path, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	content, tooLong, err := readFileWithLimit(path, 5)
	if err != nil {
		t.Fatalf("readFileWithLimit failed: %v", err)
	}
	if tooLong {
		t.Fatal("did not expect file to exceed limit")
	}
	if string(content) != "hello" {
		t.Fatalf("unexpected content: %q", string(content))
	}

	_, tooLong, err = readFileWithLimit(path, 4)
	if err != nil {
		t.Fatalf("readFileWithLimit failed: %v", err)
	}
	if !tooLong {
		t.Fatal("expected file to exceed limit 4")
	}
}

func TestNormalizeContextRelPath(t *testing.T) {
	if got := normalizeContextRelPath(`.\src\main.go`); got != "src/main.go" {
		t.Fatalf("unexpected normalized path: %q", got)
	}
}

func TestIsBinaryFile(t *testing.T) {
	root := t.TempDir()
	textPath := filepath.Join(root, "main.go")
	binaryPath := filepath.Join(root, "image.bin")

	if err := os.WriteFile(textPath, []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("write text file failed: %v", err)
	}
	if err := os.WriteFile(binaryPath, []byte{0x89, 'P', 'N', 'G', 0x00, 0x0A}, 0o644); err != nil {
		t.Fatalf("write binary file failed: %v", err)
	}

	textIsBinary, err := isBinaryFile(textPath)
	if err != nil {
		t.Fatalf("isBinaryFile(text) failed: %v", err)
	}
	if textIsBinary {
		t.Fatal("expected text file to be treated as non-binary")
	}

	binaryIsBinary, err := isBinaryFile(binaryPath)
	if err != nil {
		t.Fatalf("isBinaryFile(binary) failed: %v", err)
	}
	if !binaryIsBinary {
		t.Fatal("expected binary file to be detected")
	}
}

func TestGenerateShotgunOutputWithProgressOmitsBinaryContent(t *testing.T) {
	app := newTestApp(t)
	root := t.TempDir()

	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("write text file failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "tool.bin"), []byte{0x00, 0x01, 0x02, 0x03}, 0o644); err != nil {
		t.Fatalf("write binary file failed: %v", err)
	}

	out, err := app.generateShotgunOutputWithProgress(context.Background(), root, nil)
	if err != nil {
		t.Fatalf("generateShotgunOutputWithProgress failed: %v", err)
	}

	if !strings.Contains(out, "<file path=\"tool.bin\">") {
		t.Fatalf("expected tool.bin tag in output: %q", out)
	}
	if !strings.Contains(out, "[Binary file content omitted]") {
		t.Fatalf("expected binary placeholder in output: %q", out)
	}
}

func TestShouldEmitProgress(t *testing.T) {
	now := time.Now()

	initialState := &generationProgressState{}
	if !shouldEmitProgress(initialState, now, false) {
		t.Fatal("expected initial progress update to be emitted")
	}

	throttledState := &generationProgressState{
		processedItems:   10,
		lastEmittedItems: 5,
		lastEmittedAt:    now,
	}
	if shouldEmitProgress(throttledState, now.Add(50*time.Millisecond), false) {
		t.Fatal("expected progress update to be throttled within the minimum interval")
	}

	itemThresholdState := &generationProgressState{
		processedItems:   55,
		lastEmittedItems: 5,
		lastEmittedAt:    now,
	}
	if !shouldEmitProgress(itemThresholdState, now.Add(10*time.Millisecond), false) {
		t.Fatal("expected progress update once enough items have been processed")
	}

	timeThresholdState := &generationProgressState{
		processedItems:   10,
		lastEmittedItems: 5,
		lastEmittedAt:    now,
	}
	if !shouldEmitProgress(timeThresholdState, now.Add(progressEmitMinInterval), false) {
		t.Fatal("expected progress update once the minimum interval elapses")
	}

	forcedState := &generationProgressState{
		processedItems:   10,
		lastEmittedItems: 10,
		lastEmittedAt:    now,
	}
	if !shouldEmitProgress(forcedState, now.Add(10*time.Millisecond), true) {
		t.Fatal("expected forced progress update to bypass throttling")
	}
}
