package proxy

import (
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRotatingFileWriter_Basic(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "filewriter-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	w, err := NewRotatingFileWriter(RotatingFileWriterConfig{
		Dir:      tmpDir,
		Prefix:   "test",
		Suffix:   ".log.gz",
		MaxSize:  1024,
		MaxFiles: 3,
	})
	if err != nil {
		t.Fatalf("NewRotatingFileWriter failed: %v", err)
	}

	// Write some data
	msg := "hello world\n"
	n, err := w.Write([]byte(msg))
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if n != len(msg) {
		t.Errorf("Write returned %d, want %d", n, len(msg))
	}

	if err := w.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Check that file was created
	files, _ := filepath.Glob(filepath.Join(tmpDir, "test_*.log.gz"))
	if len(files) != 1 {
		t.Errorf("expected 1 file, got %d", len(files))
	}

	// Read and decompress file contents
	f, _ := os.Open(files[0])
	defer func() { _ = f.Close() }()
	gz, _ := gzip.NewReader(f)
	defer func() { _ = gz.Close() }()
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, gz)

	if buf.String() != msg {
		t.Errorf("file content = %q, want %q", buf.String(), msg)
	}
}

func TestRotatingFileWriter_Rotation(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "filewriter-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Small max size to trigger rotation
	w, err := NewRotatingFileWriter(RotatingFileWriterConfig{
		Dir:      tmpDir,
		Prefix:   "test",
		Suffix:   ".log.gz",
		MaxSize:  50, // 50 bytes
		MaxFiles: 3,
	})
	if err != nil {
		t.Fatalf("NewRotatingFileWriter failed: %v", err)
	}

	// Write enough data to trigger multiple rotations
	msg := strings.Repeat("x", 30) + "\n"
	for i := 0; i < 5; i++ {
		_, err := w.Write([]byte(msg))
		if err != nil {
			t.Fatalf("Write %d failed: %v", i, err)
		}
	}

	if err := w.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Should have at most MaxFiles files due to pruning
	files, _ := filepath.Glob(filepath.Join(tmpDir, "test_*.log.gz"))
	if len(files) > 3 {
		t.Errorf("expected at most 3 files, got %d", len(files))
	}
}

func TestRotatingFileWriter_Pruning(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "filewriter-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Very small max size to ensure rotation on every write
	w, err := NewRotatingFileWriter(RotatingFileWriterConfig{
		Dir:      tmpDir,
		Prefix:   "test",
		Suffix:   ".log.gz",
		MaxSize:  10, // Very small
		MaxFiles: 2,  // Keep only 2 files
	})
	if err != nil {
		t.Fatalf("NewRotatingFileWriter failed: %v", err)
	}

	// Write multiple times to create several files
	for i := 0; i < 10; i++ {
		_, _ = w.Write([]byte("data\n"))
	}

	if err := w.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Should have at most 2 files
	files, _ := filepath.Glob(filepath.Join(tmpDir, "test_*.log.gz"))
	if len(files) > 2 {
		t.Errorf("expected at most 2 files after pruning, got %d", len(files))
	}
}
