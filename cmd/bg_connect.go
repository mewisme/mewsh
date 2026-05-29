package cmd

import (
	"github.com/mewisme/mewsh/internal/connect"
	"github.com/spf13/cobra"
)

var bgConnectCmd = &cobra.Command{
	Use:    "__bg-connect__",
	Hidden: true,
	Short:  "Internal: run a background SSH worker",
	Args:   cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}
		return connect.RunBackgroundWorker(cfg, args[0])
	},
}

func init() {
	rootCmd.AddCommand(bgConnectCmd)
	connect.SetBackgroundConfigOverride(func() string { return configOverride })
}
