package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var binaryPath string

func TestMain(m *testing.M) {
	// Build the binary before running tests
	tmpDir, err := os.MkdirTemp("", "devsandbox-e2e-*")
	if err != nil {
		panic("failed to create temp dir: " + err.Error())
	}
	defer os.RemoveAll(tmpDir)

	binaryPath = filepath.Join(tmpDir, "devsandbox")

	// Get project root (parent of e2e directory)
	wd, err := os.Getwd()
	if err != nil {
		panic("failed to get working directory: " + err.Error())
	}
	projectRoot := filepath.Dir(wd)

	// Build the binary
	cmd := exec.Command("go", "build", "-o", binaryPath, "./cmd/devsandbox")
	cmd.Dir = projectRoot
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		panic("failed to build binary: " + err.Error())
	}

	os.Exit(m.Run())
}

func TestSandbox_Help(t *testing.T) {
	cmd := exec.Command(binaryPath, "--help")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("--help failed: %v\nOutput: %s", err, output)
	}

	outputStr := string(output)
	expectedStrings := []string{
		"devsandbox",
		"Bubblewrap sandbox",
		"SSH: BLOCKED",
		".env files: BLOCKED",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(outputStr, expected) {
			t.Errorf("--help output missing %q", expected)
		}
	}
}

func TestSandbox_Version(t *testing.T) {
	cmd := exec.Command(binaryPath, "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("--version failed: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(string(output), "v1.0.0") {
		t.Errorf("--version output missing version: %s", output)
	}
}

func TestSandbox_Info(t *testing.T) {
	cmd := exec.Command(binaryPath, "--info")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("--info failed: %v\nOutput: %s", err, output)
	}

	outputStr := string(output)
	expectedStrings := []string{
		"Sandbox Configuration:",
		"Project:",
		"Sandbox Home:",
		"Blocked Paths:",
		"~/.ssh",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(outputStr, expected) {
			t.Errorf("--info output missing %q", expected)
		}
	}
}

func TestSandbox_EchoCommand(t *testing.T) {
	if !bwrapAvailable() {
		t.Skip("bwrap not available")
	}

	cmd := exec.Command(binaryPath, "echo", "hello from sandbox")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("echo command failed: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(string(output), "hello from sandbox") {
		t.Errorf("echo output unexpected: %s", output)
	}
}

func TestSandbox_EnvironmentVariables(t *testing.T) {
	if !bwrapAvailable() {
		t.Skip("bwrap not available")
	}

	cmd := exec.Command(binaryPath, "env")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("env command failed: %v\nOutput: %s", err, output)
	}

	outputStr := string(output)
	expectedVars := []string{
		"SANDBOX=1",
		"SANDBOX_PROJECT=",
		"XDG_CONFIG_HOME=",
		"XDG_DATA_HOME=",
	}

	for _, expected := range expectedVars {
		if !strings.Contains(outputStr, expected) {
			t.Errorf("env output missing %q", expected)
		}
	}
}

func TestSandbox_MiseAvailable(t *testing.T) {
	if !bwrapAvailable() {
		t.Skip("bwrap not available")
	}

	// Check if mise is installed on host first
	if _, err := exec.LookPath("mise"); err != nil {
		t.Skip("mise not installed on host")
	}

	cmd := exec.Command(binaryPath, "mise", "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("mise --version failed: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(string(output), "mise") {
		t.Errorf("mise version output unexpected: %s", output)
	}
}

func TestSandbox_NeovimAvailable(t *testing.T) {
	if !bwrapAvailable() {
		t.Skip("bwrap not available")
	}

	// Check if nvim is installed on host first
	if _, err := exec.LookPath("nvim"); err != nil {
		t.Skip("nvim not installed on host")
	}

	cmd := exec.Command(binaryPath, "nvim", "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("nvim --version failed: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(string(output), "NVIM") {
		t.Errorf("nvim version output unexpected: %s", output)
	}
}

func TestSandbox_ClaudeAvailable(t *testing.T) {
	if !bwrapAvailable() {
		t.Skip("bwrap not available")
	}

	// Check if claude is installed on host first
	if _, err := exec.LookPath("claude"); err != nil {
		t.Skip("claude not installed on host")
	}

	cmd := exec.Command(binaryPath, "claude", "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Claude might return non-zero for --version, check output anyway
		if !strings.Contains(string(output), "claude") && !strings.Contains(string(output), "Claude") {
			t.Fatalf("claude --version failed: %v\nOutput: %s", err, output)
		}
	}
}

func TestSandbox_SSHBlocked(t *testing.T) {
	if !bwrapAvailable() {
		t.Skip("bwrap not available")
	}

	// Try to access ~/.ssh inside sandbox - should not exist
	cmd := exec.Command(binaryPath, "ls", "-la", os.Getenv("HOME")+"/.ssh")
	output, err := cmd.CombinedOutput()

	// Should fail or show empty/nonexistent
	outputStr := string(output)
	if err == nil && strings.Contains(outputStr, "id_rsa") {
		t.Error("SSH keys should not be accessible in sandbox")
	}
}

func TestSandbox_EnvFileBlocked(t *testing.T) {
	if !bwrapAvailable() {
		t.Skip("bwrap not available")
	}

	// Create a temp .env file in a temp project directory
	tmpDir, err := os.MkdirTemp("", "sandbox-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	envFile := filepath.Join(tmpDir, ".env")
	if err := os.WriteFile(envFile, []byte("SECRET=supersecret123"), 0o644); err != nil {
		t.Fatalf("failed to create .env: %v", err)
	}

	// Run sandbox from that directory and try to read .env
	cmd := exec.Command(binaryPath, "cat", ".env")
	cmd.Dir = tmpDir
	output, err := cmd.CombinedOutput()

	// .env should be blocked (empty or error)
	if strings.Contains(string(output), "supersecret123") {
		t.Error(".env file contents should be blocked in sandbox")
	}
}

func TestSandbox_ProjectDirWritable(t *testing.T) {
	if !bwrapAvailable() {
		t.Skip("bwrap not available")
	}

	tmpDir, err := os.MkdirTemp("", "sandbox-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a file inside sandbox using touch
	testFile := "sandbox-test-file.txt"
	cmd := exec.Command(binaryPath, "touch", testFile)
	cmd.Dir = tmpDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to create file in sandbox: %v\nOutput: %s", err, output)
	}

	// Verify file exists on host
	filePath := filepath.Join(tmpDir, testFile)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Fatalf("file not created in sandbox")
	}
}

func TestSandbox_NetworkAvailable(t *testing.T) {
	if !bwrapAvailable() {
		t.Skip("bwrap not available")
	}

	// Simple network test - check if we can resolve DNS
	cmd := exec.Command(binaryPath, "cat", "/etc/resolv.conf")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to read resolv.conf: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(string(output), "nameserver") {
		t.Error("resolv.conf should be available for network access")
	}
}

func bwrapAvailable() bool {
	_, err := exec.LookPath("bwrap")
	return err == nil
}
