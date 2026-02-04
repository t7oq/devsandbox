package sandbox

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

// LockFileName is the name of the session lock file within a sandbox directory.
const LockFileName = ".lock"

// AcquireSessionLock acquires a shared lock on the sandbox.
// The caller must keep the returned file open for the session duration.
// The lock is automatically released when the file is closed or process exits.
func AcquireSessionLock(sandboxRoot string) (*os.File, error) {
	lockPath := filepath.Join(sandboxRoot, LockFileName)

	// Create or open the lock file
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, fmt.Errorf("failed to open lock file: %w", err)
	}

	// Acquire shared lock (non-blocking)
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_SH|syscall.LOCK_NB); err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("failed to acquire lock: %w", err)
	}

	return f, nil
}

// IsSessionActive checks if any session holds a lock on the sandbox.
// Returns true if a session is active (lock is held).
func IsSessionActive(sandboxRoot string) bool {
	lockPath := filepath.Join(sandboxRoot, LockFileName)

	// Try to open the lock file
	f, err := os.OpenFile(lockPath, os.O_RDWR, 0o644)
	if err != nil {
		// File doesn't exist or can't be opened - no active session
		return false
	}
	defer func() { _ = f.Close() }()

	// Try to acquire exclusive lock (non-blocking)
	// If this fails, someone else holds a shared lock
	err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		// EWOULDBLOCK means lock is held by another process
		return true
	}

	// We got the lock, so no one else has it - release immediately
	_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	return false
}
