package tui

import (
	"fmt"
	"os"
	"runtime"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/term"
	"github.com/mewisme/mewsh/internal/config"
	"github.com/mewisme/mewsh/internal/connect"
)

// Run starts the interactive TUI.
func Run(cfg *config.Config) error {
	_ = os.Setenv("MEWSH_TUI", "1")
	defer os.Unsetenv("MEWSH_TUI")

	if !canRunInteractive() {
		return fmt.Errorf(
			"mewsh TUI needs an interactive terminal\n" +
				"  • Run from Windows Terminal, PowerShell, or cmd\n" +
				"  • For scripts, use: mewsh list, mewsh connect <alias>, etc.",
		)
	}

	opts := []tea.ProgramOption{tea.WithAltScreen()}
	opts = append(opts, platformOpts()...)

	m := NewModel(cfg)
	p := tea.NewProgram(m, opts...)
	if _, err := p.Run(); err != nil {
		connect.CleanupActive()
		return err
	}
	connect.CleanupActive()
	return nil
}

func platformOpts() []tea.ProgramOption {
	if runtime.GOOS != "windows" {
		return nil
	}
	var opts []tea.ProgramOption
	if !term.IsTerminal(os.Stdin.Fd()) {
		opts = append(opts, tea.WithInputTTY())
	}
	// Windows Terminal / ConPTY may not expose stdout as a console handle; CONOUT$ still works.
	if !term.IsTerminal(os.Stdout.Fd()) {
		conout, err := os.OpenFile("CONOUT$", os.O_WRONLY, 0o644) //nolint:gosec // Windows console device
		if err == nil {
			opts = append(opts, tea.WithOutput(conout))
		}
	}
	return opts
}

func canRunInteractive() bool {
	if term.IsTerminal(os.Stdout.Fd()) || term.IsTerminal(os.Stderr.Fd()) || term.IsTerminal(os.Stdin.Fd()) {
		return true
	}
	if runtime.GOOS == "windows" {
		f, err := os.OpenFile("CONOUT$", os.O_RDWR, 0o644) //nolint:gosec
		if err == nil {
			f.Close()
			return true
		}
	}
	return false
}
