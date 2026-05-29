//go:build windows

package tunnel

import (
	"os/exec"
	"syscall"
)

func setHiddenProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
}
