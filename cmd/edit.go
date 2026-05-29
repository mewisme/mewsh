package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var editCmd = &cobra.Command{
	Use:   "edit <alias>",
	Short: "Edit an SSH profile",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}
		alias := args[0]
		p, idx := cfg.FindByAlias(alias)
		if idx < 0 {
			return fmt.Errorf("profile %q not found", alias)
		}
		aliases := aliasesFromConfig(cfg.Profiles)
		updated, err := promptProfile(p, aliases, formOptions{Editing: true, OrigAlias: alias})
		if err != nil {
			if isFormCancelled(err) {
				printStatus("Cancelled.")
				return nil
			}
			return err
		}
		cfg.Profiles[idx] = updated
		if err := saveConfig(cfg); err != nil {
			return err
		}
		printStatus(fmt.Sprintf("Profile %q updated.", updated.Alias))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(editCmd)
}
