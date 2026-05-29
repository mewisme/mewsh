package cmd

import (
	"fmt"
	"os"

	"github.com/mewisme/mewsh/internal/cliui"
	"github.com/mewisme/mewsh/internal/config"
	"github.com/mewisme/mewsh/internal/tui"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:          "mewsh",
	Short:        "SSH profile manager",
	Long:         "Manage SSH credentials and connect servers.",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := config.EnsureDir(); err != nil {
			return err
		}
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		if err := tui.Run(cfg); err != nil {
			return fmt.Errorf("tui: %w", err)
		}
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
	cliui.Errf(os.Stderr, "%s", err)
	os.Exit(1)
}
