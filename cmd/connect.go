package cmd

import (
	"github.com/mewisme/mewsh/internal/connect"
	"github.com/spf13/cobra"
)

var connectBackground bool

var connectCmd = &cobra.Command{
	Use:   "connect <alias>",
	Short: "Connect to an SSH profile",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}
		opts := []connect.Option{}
		if connectBackground {
			opts = append(opts, connect.WithBackground(true))
		}
		return connect.Profile(cfg, args[0], opts...)
	},
}

func init() {
	connectCmd.Flags().BoolVarP(&connectBackground, "background", "b", false,
		"Run SSH in the background without a GUI terminal; survives when your shell exits (headless servers)")
	rootCmd.AddCommand(connectCmd)
}
