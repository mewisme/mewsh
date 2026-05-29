//go:build windows

package terminal

import (
	"syscall"
)

var (
	modKernel32      = syscall.NewLazyDLL("kernel32.dll")
	procFreeConsole  = modKernel32.NewProc("FreeConsole")
	procSetStdHandle = modKernel32.NewProc("SetStdHandle")
)

const (
	stdInputHandle  = uintptr(^uintptr(9))  // STD_INPUT_HANDLE = -10
	stdOutputHandle = uintptr(^uintptr(10)) // STD_OUTPUT_HANDLE = -11
	stdErrorHandle  = uintptr(^uintptr(11)) // STD_ERROR_HANDLE = -12
)

// DetachFromConsole releases this process from the parent console so SSH/cloudflared
// cannot write escape sequences into the user's terminal.
func DetachFromConsole() {
	_, _, _ = procFreeConsole.Call()
	nul, err := syscall.Open("NUL", syscall.O_RDWR, 0)
	if err != nil {
		return
	}
	handle := uintptr(nul)
	_, _, _ = procSetStdHandle.Call(stdOutputHandle, handle)
	_, _, _ = procSetStdHandle.Call(stdErrorHandle, handle)
	_, _, _ = procSetStdHandle.Call(stdInputHandle, handle)
}
