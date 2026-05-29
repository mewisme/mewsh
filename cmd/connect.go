package cmd

import (
	"github.com/mewisme/mewsh/internal/connect"
	"github.com/spf13/cobra"
)

var connectCmd = &cobra.Command{
	Use:   "connect <alias>",
	Short: "Connect to an SSH profile",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}
		return connect.Profile(cfg, args[0])
	},
}

func init() {
	rootCmd.AddCommand(connectCmd)
}
