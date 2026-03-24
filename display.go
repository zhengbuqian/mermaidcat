package main

import (
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Brand represents a detected terminal emulator (mirrors yazi's Brand enum).
const (
	BrandKitty     = "kitty"
	BrandKonsole   = "konsole"
	BrandIterm2    = "iterm2"
	BrandWezTerm   = "wezterm"
	BrandFoot      = "foot"
	BrandGhostty   = "ghostty"
	BrandMicrosoft = "microsoft"
	BrandWarp      = "warp"
	BrandRio       = "rio"
	BrandBlackBox  = "blackbox"
	BrandVSCode    = "vscode"
	BrandTabby     = "tabby"
	BrandHyper     = "hyper"
	BrandMintty    = "mintty"
	BrandTmux      = "tmux"
	BrandVTerm     = "vterm"
	BrandApple     = "apple"
	BrandUrxvt     = "urxvt"
	BrandBobcat    = "bobcat"
)

// detectAll detects terminal brand and theme in a single operation.
// Detection order (matches yazi): env vars → tmux env → CSI probe.
func detectAll() (string, string) {
	// 0. MERMAIDCAT_TERMINAL override (for environments without tty, e.g. Claude Code)
	if override := os.Getenv("MERMAIDCAT_TERMINAL"); override != "" {
		logf("terminal override via MERMAIDCAT_TERMINAL: %s", override)
		mermaidTheme := "dark"
		if *theme != "" {
			mermaidTheme = *theme
		}
		return override, mermaidTheme
	}

	// 1. Environment variables (highest priority)
	terminal := brandFromEnv()
	if terminal != "" {
		logf("terminal detected via env: %s", terminal)
	}

	// 2. tmux saved environment
	if terminal == "" && inTmux() {
		terminal = brandFromTmux()
		if terminal != "" {
			logf("terminal detected via tmux show-environment: %s", terminal)
		}
	}

	// Use --theme flag if specified (skip tty probe for theme)
	if *theme != "" {
		// Still need CSI probe for terminal detection if not found yet
		if terminal == "" {
			probe := probeTTY()
			if probe.terminal != "" {
				terminal = probe.terminal
				logf("terminal detected via CSI query: %s", terminal)
			}
		}
		logf("theme: %s (from --theme flag)", *theme)
		return terminal, *theme
	}

	// 3. Probe TTY: XTVERSION + OSC 11 + DA1
	probe := probeTTY()
	if terminal == "" && probe.terminal != "" {
		terminal = probe.terminal
		logf("terminal detected via CSI query: %s", terminal)
	}

	mermaidTheme := "dark"
	if probe.probed {
		if probe.isDark {
			mermaidTheme = "dark"
		} else {
			mermaidTheme = "default"
		}
		logf("theme: %s (auto-detected via OSC 11)", mermaidTheme)
	} else {
		logf("theme: dark (fallback)")
	}

	return terminal, mermaidTheme
}

// brandFromEnv detects terminal brand from environment variables.
// Matches yazi's Brand::from_env() exactly.
func brandFromEnv() string {
	// Priority 1: TERM variable
	term := os.Getenv("TERM")
	switch term {
	case "xterm-kitty":
		return BrandKitty
	case "foot", "foot-extra":
		return BrandFoot
	case "xterm-ghostty":
		return BrandGhostty
	case "rio":
		return BrandRio
	case "rxvt-unicode-256color":
		return BrandUrxvt
	}

	// Priority 2: TERM_PROGRAM variable
	termProgram := os.Getenv("TERM_PROGRAM")
	switch termProgram {
	case "iTerm.app":
		return BrandIterm2
	case "WezTerm":
		return BrandWezTerm
	case "ghostty":
		return BrandGhostty
	case "WarpTerminal":
		return BrandWarp
	case "rio":
		return BrandRio
	case "BlackBox":
		return BrandBlackBox
	case "vscode":
		return BrandVSCode
	case "Tabby":
		return BrandTabby
	case "Hyper":
		return BrandHyper
	case "mintty":
		return BrandMintty
	case "Apple_Terminal":
		return BrandApple
	}

	// Priority 3: Special environment variables
	envChecks := []struct {
		env   string
		brand string
	}{
		{"KITTY_WINDOW_ID", BrandKitty},
		{"KONSOLE_VERSION", BrandKonsole},
		{"ITERM_SESSION_ID", BrandIterm2},
		{"WEZTERM_EXECUTABLE", BrandWezTerm},
		{"GHOSTTY_RESOURCES_DIR", BrandGhostty},
		{"WT_Session", BrandMicrosoft},
		{"WARP_HONOR_PS1", BrandWarp},
		{"VSCODE_INJECTION", BrandVSCode},
		{"TABBY_CONFIG_DIRECTORY", BrandTabby},
	}
	for _, check := range envChecks {
		if os.Getenv(check.env) != "" {
			return check.brand
		}
	}

	return ""
}

// brandFromTmux recovers TERM/TERM_PROGRAM from tmux's saved environment.
// Matches yazi's Mux::term_program().
func brandFromTmux() string {
	out, err := exec.Command("tmux", "show-environment").Output()
	if err != nil {
		logf("tmux show-environment failed: %v", err)
		return ""
	}
	var termProgram, term string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if k, v, ok := strings.Cut(line, "="); ok {
			switch k {
			case "TERM_PROGRAM":
				termProgram = v
			case "TERM":
				term = v
			}
		}
	}
	logf("tmux env: TERM=%q, TERM_PROGRAM=%q", term, termProgram)

	// Re-run env detection with tmux values
	switch term {
	case "xterm-kitty":
		return BrandKitty
	case "foot", "foot-extra":
		return BrandFoot
	case "xterm-ghostty":
		return BrandGhostty
	case "rio":
		return BrandRio
	}
	switch termProgram {
	case "iTerm.app":
		return BrandIterm2
	case "WezTerm":
		return BrandWezTerm
	case "ghostty":
		return BrandGhostty
	case "WarpTerminal":
		return BrandWarp
	case "rio":
		return BrandRio
	case "BlackBox":
		return BrandBlackBox
	case "vscode":
		return BrandVSCode
	case "Tabby":
		return BrandTabby
	case "Hyper":
		return BrandHyper
	case "mintty":
		return BrandMintty
	case "Apple_Terminal":
		return BrandApple
	}
	return ""
}

// isIIPTerminal returns true if the terminal supports iTerm2 Inline Images Protocol.
// Matches yazi's Brand → Adapter mapping (brands that include Iip).
func isIIPTerminal(terminal string) bool {
	switch terminal {
	case BrandIterm2, BrandWezTerm, BrandWarp, BrandRio,
		BrandVSCode, BrandTabby, BrandHyper, BrandMintty, BrandBobcat:
		return true
	}
	return false
}

// inTmux returns true if running inside tmux.
func inTmux() bool {
	return os.Getenv("TMUX") != ""
}

// displayIIP writes image data to the terminal using iTerm2 Inline Images Protocol.
func displayIIP(imageData []byte) error {
	encoded := base64.StdEncoding.EncodeToString(imageData)

	var sb strings.Builder

	if inTmux() {
		sb.WriteString("\033Ptmux;\033\033]")
	} else {
		sb.WriteString("\033]")
	}

	sb.WriteString("1337;File=inline=1")
	sb.WriteString(fmt.Sprintf(";size=%d", len(imageData)))
	if *width != "" {
		sb.WriteString(fmt.Sprintf(";width=%s", *width))
	}
	if *height != "" {
		sb.WriteString(fmt.Sprintf(";height=%s", *height))
	}
	sb.WriteString(":")
	sb.WriteString(encoded)

	if inTmux() {
		sb.WriteString("\a\033\\")
	} else {
		sb.WriteString("\a")
	}

	sb.WriteString("\n")

	_, err := fmt.Fprint(os.Stdout, sb.String())
	return err
}
