// internal/sandbox/tools/setup_helpers.go
package tools

import (
	"os"
	"path/filepath"
)

// SetupConfigWithSuffix copies a config file and appends a suffix.
// Creates destination directory if needed.
// Returns nil without error if source doesn't exist.
func SetupConfigWithSuffix(srcPath, destPath, suffix string) error {
	if _, err := os.Stat(srcPath); os.IsNotExist(err) {
		return nil
	}

	// Create destination directory
	destDir := filepath.Dir(destPath)
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return err
	}

	// Read original
	original, err := os.ReadFile(srcPath)
	if err != nil {
		return err
	}

	// Write with suffix
	modified := string(original) + suffix
	return os.WriteFile(destPath, []byte(modified), 0o644)
}
