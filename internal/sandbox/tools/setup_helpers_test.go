// internal/sandbox/tools/setup_helpers_test.go
package tools

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSetupConfigWithSuffix(t *testing.T) {
	tmpDir := t.TempDir()

	// Create source config
	srcPath := filepath.Join(tmpDir, "source.conf")
	if err := os.WriteFile(srcPath, []byte("original content\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create dest directory
	destDir := filepath.Join(tmpDir, "sandbox", ".config")
	destPath := filepath.Join(destDir, "source.conf")

	suffix := "\n# Added by sandbox\n"

	err := SetupConfigWithSuffix(srcPath, destPath, suffix)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify content
	content, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("failed to read dest: %v", err)
	}

	expected := "original content\n" + suffix
	if string(content) != expected {
		t.Errorf("unexpected content:\ngot: %q\nwant: %q", string(content), expected)
	}
}

func TestSetupConfigWithSuffix_SourceNotExists(t *testing.T) {
	tmpDir := t.TempDir()

	srcPath := filepath.Join(tmpDir, "nonexistent.conf")
	destPath := filepath.Join(tmpDir, "dest.conf")

	err := SetupConfigWithSuffix(srcPath, destPath, "suffix")
	if err != nil {
		t.Errorf("expected nil error for missing source, got: %v", err)
	}

	// Dest should not be created
	if _, err := os.Stat(destPath); !os.IsNotExist(err) {
		t.Error("expected dest to not exist")
	}
}

func TestSetupConfigWithSuffix_CreatesDirIfNeeded(t *testing.T) {
	tmpDir := t.TempDir()

	// Create source config
	srcPath := filepath.Join(tmpDir, "source.conf")
	if err := os.WriteFile(srcPath, []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Nested destination path - directories don't exist yet
	destPath := filepath.Join(tmpDir, "a", "b", "c", "dest.conf")

	err := SetupConfigWithSuffix(srcPath, destPath, "\nappended")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(destPath); os.IsNotExist(err) {
		t.Error("expected dest file to be created")
	}
}
