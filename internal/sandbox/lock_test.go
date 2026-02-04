package sandbox

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAcquireSessionLock(t *testing.T) {
	tmpDir := t.TempDir()

	lockFile, err := AcquireSessionLock(tmpDir)
	if err != nil {
		t.Fatalf("AcquireSessionLock failed: %v", err)
	}
	defer func() { _ = lockFile.Close() }()

	// Verify lock file was created
	lockPath := filepath.Join(tmpDir, LockFileName)
	if _, err := os.Stat(lockPath); os.IsNotExist(err) {
		t.Error("Lock file was not created")
	}
}

func TestIsSessionActive_NoLock(t *testing.T) {
	tmpDir := t.TempDir()

	if IsSessionActive(tmpDir) {
		t.Error("Expected no active session for new directory")
	}
}

func TestIsSessionActive_WithLock(t *testing.T) {
	tmpDir := t.TempDir()

	// Acquire lock
	lockFile, err := AcquireSessionLock(tmpDir)
	if err != nil {
		t.Fatalf("AcquireSessionLock failed: %v", err)
	}

	// Check if active
	if !IsSessionActive(tmpDir) {
		t.Error("Expected session to be active while lock is held")
	}

	// Release lock
	_ = lockFile.Close()

	// Check again - should not be active
	if IsSessionActive(tmpDir) {
		t.Error("Expected no active session after lock released")
	}
}

func TestAcquireSessionLock_MultipleSessions(t *testing.T) {
	tmpDir := t.TempDir()

	// First session
	lock1, err := AcquireSessionLock(tmpDir)
	if err != nil {
		t.Fatalf("First lock failed: %v", err)
	}
	defer func() { _ = lock1.Close() }()

	// Second session (should succeed - shared locks)
	lock2, err := AcquireSessionLock(tmpDir)
	if err != nil {
		t.Fatalf("Second lock failed: %v", err)
	}
	defer func() { _ = lock2.Close() }()

	// Both should show as active
	if !IsSessionActive(tmpDir) {
		t.Error("Expected session to be active with two locks held")
	}

	// Close first lock
	_ = lock1.Close()

	// Should still be active (second lock held)
	if !IsSessionActive(tmpDir) {
		t.Error("Expected session to be active with one lock still held")
	}
}
