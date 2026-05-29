package cmd

import (
	"fmt"

	"github.com/mewisme/mewsh/internal/cliui"
	"github.com/mewisme/mewsh/internal/secret"
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete <alias>",
	Short: "Delete an SSH profile",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}
		alias := args[0]
		_, idx := cfg.FindByAlias(alias)
		if idx < 0 {
			return fmt.Errorf("profile %q not found", alias)
		}
		ref := cfg.Profiles[idx].PasswordRef
		if ref == "" {
			ref = alias
		}
		cfg.Profiles = append(cfg.Profiles[:idx], cfg.Profiles[idx+1:]...)
		if err := saveConfig(cfg); err != nil {
			return err
		}
		_ = secret.DeletePassword(ref)
		cliui.OKf(cmd.OutOrStdout(), "Deleted profile %q.", alias)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(deleteCmd)
}
