//go:build windows

package connect

import (
	"os/exec"
	"syscall"

	"github.com/mewisme/mewsh/internal/terminal"
)

func configureSupervisorCmd(cmd *exec.Cmd) {
	attr := &syscall.SysProcAttr{}
	terminal.ConfigureDetachedCmd(attr)
	cmd.SysProcAttr = attr
}
