package sandbox

import (
	"os"
	"path/filepath"
	"regexp"
)

const (
	// SandboxBaseDir is the directory under ~/.local/share for sandbox data
	SandboxBaseDir = "devsandbox"
)

type Config struct {
	HomeDir     string
	ProjectDir  string
	ProjectName string
	SandboxRoot string // ~/.local/share/devsandbox/<project>
	SandboxHome string // ~/.local/share/devsandbox/<project>/home (mounted at $HOME)
	XDGRuntime  string

	// Proxy settings
	ProxyEnabled bool
	ProxyPort    int
	ProxyLog     bool
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

	projectName := SanitizeProjectName(filepath.Base(projectDir))
	// Use XDG-compliant path: ~/.local/share/devsandbox/<project>
	sandboxRoot := filepath.Join(homeDir, ".local", "share", SandboxBaseDir, projectName)
	sandboxHome := filepath.Join(sandboxRoot, "home")

	xdgRuntime := os.Getenv("XDG_RUNTIME_DIR")
	if xdgRuntime == "" {
		xdgRuntime = filepath.Join("/run/user", string(rune(os.Getuid())))
	}

	return &Config{
		HomeDir:     homeDir,
		ProjectDir:  projectDir,
		ProjectName: projectName,
		SandboxRoot: sandboxRoot,
		SandboxHome: sandboxHome,
		XDGRuntime:  xdgRuntime,
	}, nil
}

var nonAlphanumericRe = regexp.MustCompile(`[^a-zA-Z0-9._-]`)

func SanitizeProjectName(name string) string {
	return nonAlphanumericRe.ReplaceAllString(name, "_")
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

	return nil
}

func (c *Config) SandboxBase() string {
	return filepath.Join(c.HomeDir, ".local", "share", SandboxBaseDir)
}
