package main

import (
	"os"

	"github.com/mewisme/mewsh/cmd"
	"github.com/mewisme/mewsh/internal/cliui"
	"github.com/mewisme/mewsh/internal/connect"
)

func main() {
	if len(os.Args) >= 2 && os.Args[1] == connect.AskpassModeArg {
		ref, err := connect.ReadAskpassRefFile()
		if err != nil {
			cliui.Errf(os.Stderr, "%s", err)
			os.Exit(1)
		}
		if err := connect.RunAskpass(ref); err != nil {
			cliui.Errf(os.Stderr, "%s", err)
			os.Exit(1)
		}
		return
	}
	if ref := os.Getenv(connect.AskpassEnvRef); ref != "" {
		if err := connect.RunAskpass(ref); err != nil {
			cliui.Errf(os.Stderr, "%s", err)
			os.Exit(1)
		}
		return
	}
	if err := cmd.Execute(); err != nil {
		cliui.Errf(os.Stderr, "%s", err)
		os.Exit(1)
	}
}
