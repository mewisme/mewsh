//go:build windows

package tunnel

import (
	"os/exec"
	"syscall"
)

func setHiddenProcess(cmd *exec.Cmd) {
	// DETACHED_PROCESS + CREATE_NO_WINDOW: no console flash while the tunnel runs.
	const (
		createNewProcessGroup = 0x00000200
		detachedProcess       = 0x00000008
		createNoWindow        = 0x08000000
	)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: createNewProcessGroup | detachedProcess | createNoWindow,
		HideWindow:    true,
	}
}
