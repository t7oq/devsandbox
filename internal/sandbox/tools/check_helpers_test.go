// internal/sandbox/tools/check_helpers_test.go
package tools

import (
	"testing"
)

func TestCheckBinary_Found(t *testing.T) {
	// "sh" should exist on any Unix system
	result := CheckBinary("sh", "install sh")

	if !result.Available {
		t.Error("expected sh to be available")
	}
	if result.BinaryPath == "" {
		t.Error("expected BinaryPath to be set")
	}
	if len(result.Issues) != 0 {
		t.Errorf("expected no issues, got %v", result.Issues)
	}
}

func TestCheckBinary_NotFound(t *testing.T) {
	result := CheckBinary("nonexistent-binary-xyz", "apt install nonexistent")

	if result.Available {
		t.Error("expected binary to not be available")
	}
	if result.BinaryPath != "" {
		t.Error("expected BinaryPath to be empty")
	}
	if len(result.Issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(result.Issues))
	}
	if result.Issues[0] != "nonexistent-binary-xyz binary not found in PATH" {
		t.Errorf("unexpected issue message: %s", result.Issues[0])
	}
	if result.InstallHint != "apt install nonexistent" {
		t.Errorf("unexpected install hint: %s", result.InstallHint)
	}
}

func TestCheckResult_AddConfigPath_Exists(t *testing.T) {
	result := CheckResult{}

	// /tmp should exist
	result.AddConfigPath("/tmp")

	if len(result.ConfigPaths) != 1 {
		t.Fatalf("expected 1 config path, got %d", len(result.ConfigPaths))
	}
	if result.ConfigPaths[0] != "/tmp" {
		t.Errorf("unexpected config path: %s", result.ConfigPaths[0])
	}
}

func TestCheckResult_AddConfigPath_NotExists(t *testing.T) {
	result := CheckResult{}

	result.AddConfigPath("/nonexistent/path/xyz")

	if len(result.ConfigPaths) != 0 {
		t.Errorf("expected 0 config paths, got %d", len(result.ConfigPaths))
	}
}

func TestCheckResult_AddConfigPaths(t *testing.T) {
	result := CheckResult{}

	result.AddConfigPaths("/tmp", "/nonexistent/xyz", "/usr")

	if len(result.ConfigPaths) != 2 {
		t.Fatalf("expected 2 config paths, got %d: %v", len(result.ConfigPaths), result.ConfigPaths)
	}
}
