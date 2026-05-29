package cmd

import (
	"strconv"
	"strings"

	"github.com/mewisme/mewsh/internal/cliui"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List SSH profiles",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}
		if len(cfg.Profiles) == 0 {
			cliui.Info("No profiles configured. Run `mewsh add` to create one.")
			return nil
		}
		rows := make([][]string, 0, len(cfg.Profiles))
		for _, p := range cfg.Profiles {
			target := p.Host
			if p.ConnectionType == "cloudflare_access" {
				target = "cf:" + p.CFHostname
			}
			note := strings.ReplaceAll(p.Note, "\n", " ")
			rows = append(rows, []string{p.Alias, target, strconv.Itoa(p.Port), p.AuthType, note})
		}
		cliui.PrintTable(cmd.OutOrStdout(), []string{"ALIAS", "TARGET", "PORT", "AUTH", "NOTE"}, rows)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
