package cmd

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mewisme/mewsh/internal/config"
	"github.com/mewisme/mewsh/internal/connect"
	"github.com/mewisme/mewsh/internal/terminal"
	"github.com/mewisme/mewsh/internal/tui"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "mewsh",
	Short: "SSH profile manager",
	Long:  "Manage SSH credentials and connect servers.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := config.EnsureDir(); err != nil {
			return err
		}
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		m := tui.NewModel(cfg)
		p := tea.NewProgram(m, tea.WithAltScreen())
		p.SetWindowTitle(terminal.WindowTitle)
		if _, err := p.Run(); err != nil {
			connect.CleanupActive()
			return fmt.Errorf("tui: %w", err)
		}
		connect.CleanupActive()
		return nil
	},
}

func Execute() error {
	return rootCmd.Execute()
}

var configOverride string

func init() {
	rootCmd.PersistentFlags().StringVar(&configOverride, "config", "", "override config file path")
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if configOverride != "" {
			config.SetPathOverride(configOverride)
		}
		return config.EnsureDir()
	}
}

func loadConfig() (*config.Config, error) {
	if err := config.EnsureDir(); err != nil {
		return nil, err
	}
	return config.Load()
}

func saveConfig(cfg *config.Config) error {
	return config.Save(cfg)
}

func exitErr(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
