package cmd

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var (
	headerStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("69"))
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
			fmt.Println("No profiles configured. Run `mewsh add` to create one.")
			return nil
		}
		fmt.Println(headerStyle.Render(fmt.Sprintf("%-16s %-24s %-8s %-10s %s", "ALIAS", "TARGET", "PORT", "AUTH", "NOTE")))
		for _, p := range cfg.Profiles {
			target := p.Host
			if p.ConnectionType == "cloudflare_access" {
				target = "cf:" + p.CFHostname
			}
			note := strings.ReplaceAll(p.Note, "\n", " ")
			fmt.Printf("%-16s %-24s %-8d %-10s %s\n", p.Alias, target, p.Port, p.AuthType, note)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
