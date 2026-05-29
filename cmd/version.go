package cmd

import (
	"fmt"

	"github.com/mewisme/mewsh/internal/selfupdate"
	"github.com/mewisme/mewsh/internal/version"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the current mewsh version",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println(version.String())
		if info, err := selfupdate.DetectInstall(); err == nil {
			fmt.Printf("install: %s\n", info.Method)
			if hint := selfupdate.UpdateCommand(info.Method); hint != "" {
				fmt.Printf("update via: %s\n", hint)
			} else {
				fmt.Println("update via: mewsh update")
			}
		}
		if check, err := selfupdate.Check(); err == nil && check.Newer {
			fmt.Printf("version latest: %s (update available — run `mewsh update`)\n", check.Latest)
		} else if err == nil && check.Latest != "" {
			fmt.Printf("version latest: %s\n", check.Latest)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
