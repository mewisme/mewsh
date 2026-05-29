package cmd

import (
	"fmt"
	"os/exec"

	"github.com/mewisme/mewsh/internal/cloudflared"
	"github.com/mewisme/mewsh/internal/config"
	"github.com/mewisme/mewsh/internal/terminal"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check mewsh environment and dependencies",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}
		ok := true
		printCheck := func(name string, pass bool, detail string) {
			status := "ok"
			if !pass {
				status = "fail"
				ok = false
			}
			fmt.Printf("[%s] %s", status, name)
			if detail != "" {
				fmt.Printf(": %s", detail)
			}
			fmt.Println()
		}

		if path, err := exec.LookPath("ssh"); err != nil {
			printCheck("ssh", false, "not found in PATH")
		} else {
			printCheck("ssh", true, path)
		}

		if path, err := cloudflared.ResolvePath(cfg); err != nil {
			printCheck("cloudflared", false, err.Error())
		} else {
			printCheck("cloudflared", true, path)
		}

		if err := terminal.CheckSupport(); err != nil {
			printCheck("terminal spawn", false, err.Error())
		} else {
			printCheck("terminal spawn", true, "supported launcher available")
		}

		path, _ := config.Path()
		for _, issue := range config.CheckPermissions() {
			printCheck("config permissions", false, issue.Message)
		}
		if len(config.CheckPermissions()) == 0 {
			printCheck("config permissions", true, path)
		}

		if !ok {
			return fmt.Errorf("doctor found issues")
		}
		return nil
	},
}

var cloudflaredCmd = &cobra.Command{
	Use:   "cloudflared",
	Short: "Manage bundled cloudflared",
}

var cloudflaredUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Download the latest cloudflared release",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}
		path, err := cloudflared.Update(cfg)
		if err != nil {
			return err
		}
		fmt.Printf("cloudflared updated: %s\n", path)
		return nil
	},
}

func init() {
	cloudflaredCmd.AddCommand(cloudflaredUpdateCmd)
	rootCmd.AddCommand(doctorCmd)
	rootCmd.AddCommand(cloudflaredCmd)
}
