package cmd

import (
	"github.com/mewisme/mewsh/internal/selfupdate"
	"github.com/spf13/cobra"
)

var (
	updateCheckOnly bool
	updateForce     bool
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update mewsh to the latest release",
	Long: `Checks GitHub for a newer release and updates using the best method:

  • Homebrew — runs brew upgrade mewsh
  • go install — runs go install github.com/mewisme/mewsh@<release> (latest GitHub tag)
  • binary     — downloads the release asset and replaces this executable

For go install installs, use --force to reinstall the release tag even when already up to date:
  go install -a with GOPROXY=direct (bypasses proxy cache, rebuilds packages).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return selfupdate.Run(updateCheckOnly, updateForce)
	},
}

func init() {
	updateCmd.Flags().BoolVar(&updateCheckOnly, "check", false, "only report if an update is available")
	updateCmd.Flags().BoolVar(&updateForce, "force", false, "reinstall latest even if already up to date (go install: -a and GOPROXY=direct)")
	rootCmd.AddCommand(updateCmd)
}
