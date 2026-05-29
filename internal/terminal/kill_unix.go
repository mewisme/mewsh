//go:build !windows

package terminal

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func errSSHNotFound() error {
	return fmt.Errorf("ssh process not found")
}

func listSSHProcessPIDs(host, port string) ([]int, error) {
	if host == "" {
		return nil, fmt.Errorf("missing ssh host")
	}
	out, err := exec.Command("pgrep", "-f", buildSSHPattern(host, port)).Output()
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
	for {
		proc, err := os.FindProcess(pid)
		if err != nil {
			return
		}
		if err := proc.Signal(syscall.Signal(0)); err != nil {
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
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

func buildSSHPattern(host, port string) string {
	if port != "" {
		return fmt.Sprintf("ssh.*-p %s.*%s", port, host)
	}
	return fmt.Sprintf("ssh.*%s", host)
}
