package sandbox

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	// SandboxBaseDir is the directory under ~/.local/share for sandbox data
	SandboxBaseDir = "devsandbox"
)

// Shell represents a supported shell type
type Shell string

const (
	ShellFish Shell = "fish"
	ShellBash Shell = "bash"
	ShellZsh  Shell = "zsh"
)

type Config struct {
	HomeDir     string
	ProjectDir  string
	ProjectName string
	SandboxRoot string // ~/.local/share/devsandbox/<project>
	SandboxHome string // ~/.local/share/devsandbox/<project>/home (mounted at $HOME)
	XDGRuntime  string
	Shell       Shell  // Detected shell (fish, bash, zsh)
	ShellPath   string // Full path to shell binary

	// Proxy settings
	ProxyEnabled bool
	ProxyPort    int
	ProxyCAPath  string
	GatewayIP    string
	// True if network namespace is isolated (pasta)
	NetworkIsolated bool
}

func NewConfig() (*Config, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	projectDir, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	baseDir := filepath.Join(homeDir, ".local", "share", SandboxBaseDir)

	// Check for existing sandbox for this project directory (backward compatibility)
	var projectName string
	if existingName, found := FindExistingSandbox(baseDir, projectDir); found {
		projectName = existingName
	} else {
		// Generate new unique name with hash suffix
		projectName = GenerateSandboxName(projectDir)
	}

	sandboxRoot := filepath.Join(baseDir, projectName)
	sandboxHome := filepath.Join(sandboxRoot, "home")

	xdgRuntime := os.Getenv("XDG_RUNTIME_DIR")
	if xdgRuntime == "" {
		xdgRuntime = filepath.Join("/run/user", string(rune(os.Getuid())))
	}

	shell, shellPath := DetectShell()

	return &Config{
		HomeDir:     homeDir,
		ProjectDir:  projectDir,
		ProjectName: projectName,
		SandboxRoot: sandboxRoot,
		SandboxHome: sandboxHome,
		XDGRuntime:  xdgRuntime,
		Shell:       shell,
		ShellPath:   shellPath,
	}, nil
}

func DetectShell() (Shell, string) {
	shellEnv := os.Getenv("SHELL")
	if shellEnv == "" {
		shellEnv = "/bin/bash" // Default fallback
	}

	shellName := filepath.Base(shellEnv)

	switch {
	case strings.Contains(shellName, "fish"):
		return ShellFish, shellEnv
	case strings.Contains(shellName, "zsh"):
		return ShellZsh, shellEnv
	default:
		// Default to bash for unknown shells
		if shellEnv == "" || !strings.Contains(shellName, "bash") {
			return ShellBash, "/bin/bash"
		}
		return ShellBash, shellEnv
	}
}

var nonAlphanumericRe = regexp.MustCompile(`[^a-zA-Z0-9._-]`)

func SanitizeProjectName(name string) string {
	return nonAlphanumericRe.ReplaceAllString(name, "_")
}

// GenerateSandboxName creates a unique sandbox name from project path.
// Format: <basename>-<short-hash> (e.g., "myproject-a1b2c3d4")
func GenerateSandboxName(projectDir string) string {
	basename := SanitizeProjectName(filepath.Base(projectDir))
	hash := sha256.Sum256([]byte(projectDir))
	shortHash := hex.EncodeToString(hash[:])[:8]
	return basename + "-" + shortHash
}

// FindExistingSandbox looks for a sandbox that matches the given project directory.
// Returns the sandbox name and true if found, empty string and false otherwise.
func FindExistingSandbox(baseDir, projectDir string) (string, bool) {
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return "", false
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		sandboxRoot := filepath.Join(baseDir, entry.Name())
		m, err := LoadMetadata(sandboxRoot)
		if err != nil {
			continue
		}

		if m.ProjectDir == projectDir {
			return entry.Name(), true
		}
	}

	return "", false
}

func (c *Config) EnsureSandboxDirs() error {
	dirs := []string{
		c.SandboxHome,
		filepath.Join(c.SandboxHome, ".config"),
		filepath.Join(c.SandboxHome, ".cache"),
		filepath.Join(c.SandboxHome, ".cache", "go-build"), // Go build cache (isolated)
		filepath.Join(c.SandboxHome, ".cache", "go-mod"),   // Go module cache (isolated)
		filepath.Join(c.SandboxHome, ".local", "share"),
		filepath.Join(c.SandboxHome, ".local", "state"),
		filepath.Join(c.SandboxHome, "go"), // GOPATH (isolated)
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}

	metadataPath := filepath.Join(c.SandboxRoot, MetadataFile)
	if _, err := os.Stat(metadataPath); os.IsNotExist(err) {
		m := CreateMetadata(c)
		if err := SaveMetadata(m, c.SandboxRoot); err != nil {
			return err
		}
	} else {
		m, err := LoadMetadata(c.SandboxRoot)
		if err == nil {
			_ = m.UpdateLastUsed()
		}
	}

	return nil
}

func (c *Config) SandboxBase() string {
	return filepath.Join(c.HomeDir, ".local", "share", SandboxBaseDir)
}

// SandboxBasePath returns the base path for all sandboxes given a home directory
func SandboxBasePath(homeDir string) string {
	return filepath.Join(homeDir, ".local", "share", SandboxBaseDir)
}
