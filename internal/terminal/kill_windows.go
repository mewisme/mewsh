//go:build windows

package terminal

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

func errSSHNotFound() error {
	return fmt.Errorf("ssh process not found")
}

func escapePSLike(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

func listSSHProcessPIDs(host, port string) ([]int, error) {
	if host == "" && port == "" {
		return nil, fmt.Errorf("missing ssh markers")
	}
	filter := fmt.Sprintf(
		`Get-CimInstance Win32_Process -Filter "Name='ssh.exe'" | Where-Object { $_.CommandLine -like '*%s*' -and $_.CommandLine -like '*%s*' } | Select-Object -ExpandProperty ProcessId`,
		escapePSLike(port),
		escapePSLike(host),
	)
	out, err := exec.Command("powershell", "-NoProfile", "-Command", filter).Output()
	if err != nil {
		return nil, err
	}
	var pids []int
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		pid, err := strconv.Atoi(line)
		if err != nil || pid <= 0 {
			continue
		}
		pids = append(pids, pid)
	}
	if len(pids) == 0 {
		return nil, errSSHNotFound()
	}
	return pids, nil
}

// WaitSSHProcessExit blocks until the given PID exits (best-effort).
func WaitSSHProcessExit(pid int) {
	if pid <= 0 {
		return
	}
	_ = waitForProcessExit(pid, 0)
}

func killSSHByPattern(host, port string, exclude []int) {
	pids, err := listSSHProcessPIDs(host, port)
	if err != nil {
		return
	}
	skip := pidSet(exclude)
	for _, pid := range pids {
		if skip[pid] {
			continue
		}
		KillProcess(pid)
		return
	}
}
