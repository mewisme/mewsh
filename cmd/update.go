package cmd

import (
	"github.com/mewisme/mewsh/internal/selfupdate"
	"github.com/spf13/cobra"
)

var updateCheckOnly bool

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update mewsh to the latest release",
	Long: `Checks GitHub for a newer release and updates using the best method:

  • Homebrew — runs brew upgrade mewsh
  • go install — runs go install github.com/mewisme/mewsh@latest
  • binary     — downloads the release asset and replaces this executable`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return selfupdate.Run(updateCheckOnly)
	},
}

func init() {
	updateCmd.Flags().BoolVar(&updateCheckOnly, "check", false, "only report if an update is available")
	rootCmd.AddCommand(updateCmd)
}
