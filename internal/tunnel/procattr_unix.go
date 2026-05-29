//go:build !windows

package tunnel

import "os/exec"

func setHiddenProcess(cmd *exec.Cmd) {
	// Unix/macOS: run as background child; stdout/stderr already discarded.
}
