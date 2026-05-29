package main

import (
	"fmt"
	"os"

	"github.com/mewisme/mewsh/cmd"
	"github.com/mewisme/mewsh/internal/connect"
)

func main() {
	if len(os.Args) >= 2 && os.Args[1] == connect.AskpassModeArg {
		ref, err := connect.ReadAskpassRefFile()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if err := connect.RunAskpass(ref); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if ref := os.Getenv(connect.AskpassEnvRef); ref != "" {
		if err := connect.RunAskpass(ref); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
