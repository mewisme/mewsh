//go:build !windows

package terminal

import "syscall"

func configureBackgroundCmd(attr *syscall.SysProcAttr) {
	if attr == nil {
		return
	}
	attr.Setsid = true
}

// ConfigureDetachedCmd applies process attributes for background workers on Unix.
func ConfigureDetachedCmd(attr *syscall.SysProcAttr) {
	configureBackgroundCmd(attr)
}
