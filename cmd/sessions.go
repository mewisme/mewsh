package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/mewisme/mewsh/internal/cliui"
	"github.com/mewisme/mewsh/internal/connect"
	"github.com/spf13/cobra"
)

var (
	sessionsJSON      bool
	sessionsKillAll   bool
	sessionsKillAlias string
)

var sessionsCmd = &cobra.Command{
	Use:   "sessions",
	Short: "List and manage active SSH sessions",
	Long:  "List SSH sessions started by mewsh (including background workers) and stop them by id or profile alias.",
}

var sessionsListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List active SSH sessions",
	Aliases: []string{"ls"},
	RunE: func(cmd *cobra.Command, args []string) error {
		sessions := connect.ListSessions()
		if sessionsJSON {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(sessions)
		}
		if len(sessions) == 0 {
			cliui.Info("No active sessions.")
			return nil
		}
		rows := make([][]string, 0, len(sessions))
		for _, s := range sessions {
			pid := "-"
			if s.PID > 0 {
				pid = fmt.Sprintf("%d", s.PID)
			}
			target := sessionTarget(s)
			rows = append(rows, []string{s.ID, s.Alias, target, pid})
		}
		cliui.PrintTable(cmd.OutOrStdout(), []string{"ID", "ALIAS", "TARGET", "PID"}, rows)
		return nil
	},
}

var sessionsKillCmd = &cobra.Command{
	Use:   "kill [session-id...]",
	Short: "Stop one or more SSH sessions",
	RunE: func(cmd *cobra.Command, args []string) error {
		out := cmd.OutOrStdout()
		if sessionsKillAll {
			if len(args) > 0 || sessionsKillAlias != "" {
				return fmt.Errorf("use only one of: session ids, --all, or --alias")
			}
			connect.KillAllSessions()
			cliui.OKf(out, "Stopped all sessions.")
			return nil
		}
		if sessionsKillAlias != "" {
			if len(args) > 0 {
				return fmt.Errorf("use either session ids or --alias, not both")
			}
			if err := connect.KillSessionsByAlias(sessionsKillAlias); err != nil {
				return err
			}
			cliui.OKf(out, "Stopped session(s) for profile %q.", sessionsKillAlias)
			return nil
		}
		if len(args) == 0 {
			return fmt.Errorf("provide session id(s), or use --all / --alias")
		}
		for _, id := range args {
			if err := connect.KillSession(id); err != nil {
				return err
			}
		}
		if len(args) == 1 {
			cliui.OKf(out, "Stopped session %s.", args[0])
		} else {
			cliui.OKf(out, "Stopped %d session(s).", len(args))
		}
		return nil
	},
}

func init() {
	sessionsListCmd.Flags().BoolVar(&sessionsJSON, "json", false, "Print sessions as JSON")
	sessionsKillCmd.Flags().BoolVar(&sessionsKillAll, "all", false, "Stop every active session")
	sessionsKillCmd.Flags().StringVar(&sessionsKillAlias, "alias", "", "Stop all sessions for a profile alias")

	sessionsCmd.AddCommand(sessionsListCmd)
	sessionsCmd.AddCommand(sessionsKillCmd)
	rootCmd.AddCommand(sessionsCmd)

	sessionsCmd.RunE = sessionsListCmd.RunE
}

func sessionTarget(s connect.SessionInfo) string {
	target := s.Target
	if s.Hostname != "" && target != "" {
		return target + " (cf:" + s.Hostname + ")"
	}
	if s.Hostname != "" {
		return "cf:" + s.Hostname
	}
	return target
}
