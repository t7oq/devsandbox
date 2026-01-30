package sandbox

import (
	"fmt"
	"strings"
)

// BuildShellCommand creates the command to run inside the sandbox
func BuildShellCommand(cfg *Config, args []string) []string {
	switch cfg.Shell {
	case ShellFish:
		return buildFishCommand(cfg, args)
	case ShellZsh:
		return buildZshCommand(cfg, args)
	default:
		return buildBashCommand(cfg, args)
	}
}

func buildFishCommand(cfg *Config, args []string) []string {
	miseActivation := "if command -q mise; mise activate fish | source; end"

	if len(args) == 0 {
		greeting := fmt.Sprintf(`set -gx fish_greeting "🔒 Sandbox: %s | .env blocked | No SSH/git push"`, cfg.ProjectName)
		fishInit := miseActivation + "; " + greeting + "; exec fish"
		return []string{cfg.ShellPath, "-c", fishInit}
	}

	cmdString := strings.Join(args, " ")
	fishCmd := miseActivation + "; " + cmdString
	return []string{cfg.ShellPath, "-c", fishCmd}
}

func buildBashCommand(cfg *Config, args []string) []string {
	miseActivation := `if command -v mise &>/dev/null; then eval "$(mise activate bash)"; fi`

	if len(args) == 0 {
		// Set PS1 prompt with sandbox indicator
		ps1 := fmt.Sprintf(`PS1="🔒 [%s] \w $ "`, cfg.ProjectName)
		bashInit := miseActivation + "; " + ps1 + "; exec bash --norc --noprofile"
		return []string{cfg.ShellPath, "-c", bashInit}
	}

	cmdString := strings.Join(args, " ")
	bashCmd := miseActivation + "; " + cmdString
	return []string{cfg.ShellPath, "-c", bashCmd}
}

func buildZshCommand(cfg *Config, args []string) []string {
	miseActivation := `if command -v mise &>/dev/null; then eval "$(mise activate zsh)"; fi`

	if len(args) == 0 {
		// Set PROMPT with sandbox indicator
		prompt := fmt.Sprintf(`PROMPT="🔒 [%s] %%~ $ "`, cfg.ProjectName)
		zshInit := miseActivation + "; " + prompt + "; exec zsh --no-rcs"
		return []string{cfg.ShellPath, "-c", zshInit}
	}

	cmdString := strings.Join(args, " ")
	zshCmd := miseActivation + "; " + cmdString
	return []string{cfg.ShellPath, "-c", zshCmd}
}
