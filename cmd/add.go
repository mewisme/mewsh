package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new SSH profile interactively",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}
		aliases := aliasesFromConfig(cfg.Profiles)
		p, err := promptProfile(nil, aliases, formOptions{})
		if err != nil {
			if isFormCancelled(err) {
				printStatus("Cancelled.")
				return nil
			}
			return err
		}
		cfg.Profiles = append(cfg.Profiles, p)
		if err := saveConfig(cfg); err != nil {
			return err
		}
		printStatus(fmt.Sprintf("Profile %q added.", p.Alias))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(addCmd)
}
