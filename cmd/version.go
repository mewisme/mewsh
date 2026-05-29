package cmd

import (
	"fmt"

	"github.com/mewisme/mewsh/internal/cliui"
	"github.com/mewisme/mewsh/internal/selfupdate"
	"github.com/mewisme/mewsh/internal/version"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the current mewsh version",
	RunE: func(cmd *cobra.Command, args []string) error {
		w := cmd.OutOrStdout()
		b := version.BuildInfo()

		cliui.Section(w, "mewsh")
		buildRows := [][2]string{
			{"version", b.Version},
			{"platform", b.GOOS + "/" + b.GOARCH},
		}
		if b.Commit != "" {
			buildRows = append(buildRows, [2]string{"commit", b.Commit})
		}
		if b.Date != "" {
			buildRows = append(buildRows, [2]string{"built", b.Date})
		}
		if b.Dev {
			buildRows = append(buildRows, [2]string{"note", "development build"})
		}
		cliui.PrintKV(w, buildRows)

		if info, err := selfupdate.DetectInstall(); err == nil {
			fmt.Fprintln(w)
			cliui.Section(w, "Install")
			installRows := [][2]string{
				{"method", info.Method.String()},
				{"binary", info.Exe},
			}
			hint := selfupdate.UpdateCommand(info.Method)
			if hint == "" {
				hint = "mewsh update"
			}
			installRows = append(installRows, [2]string{"update", hint})
			cliui.PrintKV(w, installRows)
		}

		fmt.Fprintln(w)
		cliui.Section(w, "Release")
		if b.Dev {
			cliui.PrintKV(w, [][2]string{{"status", "skipped (development build)"}})
			return nil
		}

		check, err := selfupdate.Check()
		if err != nil {
			cliui.PrintKV(w, [][2]string{{"status", "could not check: " + err.Error()}})
			return nil
		}

		releaseRows := [][2]string{{"latest", check.Latest}}
		if check.Newer {
			releaseRows = append(releaseRows, [2]string{"status", "update available"})
			cliui.PrintKV(w, releaseRows)
			hint := "mewsh update"
			if cmd := selfupdate.UpdateCommand(check.Info.Method); cmd != "" {
				hint = cmd
			}
			cliui.Warnf(w, "A newer release is available (%s → %s).", check.Current, check.Latest)
			cliui.Block(w, cliui.LevelCmd, hint)
		} else {
			releaseRows = append(releaseRows, [2]string{"status", "up to date"})
			cliui.PrintKV(w, releaseRows)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
