//go:build windows

package terminal

import "syscall"

const (
	createNewProcessGroup = 0x00000200
	detachedProcess       = 0x00000008
	createNoWindow        = 0x08000000
)

// backgroundCreationFlags starts a child with no console window and no attachment
// to the parent's console (avoids corrupting the invoking terminal).
const backgroundCreationFlags = createNewProcessGroup | detachedProcess | createNoWindow

func configureBackgroundCmd(attr *syscall.SysProcAttr) {
	if attr == nil {
		return
	}
	attr.CreationFlags |= backgroundCreationFlags
	attr.HideWindow = true
}

// ConfigureDetachedCmd applies process attributes for background workers on Windows.
func ConfigureDetachedCmd(attr *syscall.SysProcAttr) {
	configureBackgroundCmd(attr)
}
