//go:build windows

package connect

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mewisme/mewsh/internal/config"
)

// writeAskpassLauncher installs a cmd wrapper SSH_ASKPASS can execute reliably.
func writeAskpassLauncher(mewshExe string) (string, error) {
	mewshExe, err := filepath.Abs(mewshExe)
	if err != nil {
		return "", err
	}
	if err := config.EnsureDir(); err != nil {
		return "", err
	}
	dir, err := config.Dir()
	if err != nil {
		return "", err
	}
	binDir := filepath.Join(dir, "bin")
	if err := os.MkdirAll(binDir, 0700); err != nil {
		return "", err
	}
	script := filepath.Join(binDir, "mewsh-askpass.cmd")
	content := fmt.Sprintf("@echo off\r\n%q %s\r\n", mewshExe, AskpassModeArg)
	if err := os.WriteFile(script, []byte(content), 0700); err != nil {
		return "", err
	}
	return script, nil
}
