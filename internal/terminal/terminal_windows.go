//go:build windows

package terminal

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func spawnDetachedPlatform(argv []string) error {
	sshLine := shellJoin(argv)
	attempts := []func() error{}

	if script, err := writeSSHBatch(argv); err == nil {
		scriptPath := script
		attempts = append(attempts, func() error {
			return exec.Command("cmd", "/C", "start", WindowTitle, "cmd", "/K", scriptPath).Run()
		})
		go func() {
			time.Sleep(time.Hour)
			_ = os.Remove(scriptPath)
		}()
	}

	attempts = append(attempts,
		func() error {
			if _, err := exec.LookPath("wt"); err != nil {
				return err
			}
			// START "" wt — empty title so wt.exe is the command. --title is an nt flag, not global.
			args := append([]string{"/C", "start", "", "wt", "-w", "0", "nt", "--title", WindowTitle, "--"}, argv...)
			return exec.Command("cmd", args...).Run()
		},
		func() error {
			return exec.Command("cmd", "/C", "start", WindowTitle, "cmd", "/K", "title "+WindowTitle+" & "+sshLine).Run()
		},
	)

	var last error
	for _, fn := range attempts {
		if err := fn(); err != nil {
			last = err
			continue
		}
		return nil
	}
	return last
}

func runInteractivePlatform(argv []string, onStarted SSHStartedFunc) error {
	attempts := []func() error{
		runInteractiveWT(argv, onStarted),
		runInteractiveBatch(argv),
		runInteractivePowerShell(argv),
		runInteractiveCmd(argv),
	}
	var last error
	for _, fn := range attempts {
		if err := fn(); err != nil {
			last = err
			continue
		}
		return nil
	}
	return last
}

func runInteractiveWT(argv []string, onStarted SSHStartedFunc) func() error {
	return func() error {
		if _, err := exec.LookPath("wt"); err != nil {
			return err
		}
		args := append([]string{"-w", "0", "nt", "--title", WindowTitle, "--"}, argv...)
		if err := exec.Command("wt", args...).Start(); err != nil {
			return err
		}
		host, port := sshProcessMarkers(argv)
		pid, err := waitForSSHProcessStart(host, port, 20*time.Second)
		if err != nil {
			return err
		}
		if onStarted != nil {
			onStarted(pid)
		}
		return waitForProcessExit(pid, 0)
	}
}

func runInteractiveBatch(argv []string) func() error {
	return func() error {
		script, err := writeSSHBatch(argv)
		if err != nil {
			return err
		}
		defer os.Remove(script)
		return exec.Command("cmd", "/C", "start", "/wait", WindowTitle, "cmd", "/C", script).Run()
	}
}

func runInteractivePowerShell(argv []string) func() error {
	return func() error {
		sshLine := shellJoin(argv)
		ps := fmt.Sprintf("& {%s}; exit $LASTEXITCODE", sshLine)
		return exec.Command("cmd", "/C", "start", "/wait", WindowTitle, "powershell", "-NoProfile", "-Command", "title "+WindowTitle+"; "+ps).Run()
	}
}

func runInteractiveCmd(argv []string) func() error {
	return func() error {
		sshLine := shellJoin(argv)
		return exec.Command("cmd", "/C", "start", "/wait", WindowTitle, "cmd", "/C", "title "+WindowTitle+" && "+sshLine).Run()
	}
}

func waitSSHSessionEnd(argv []string) {
	host, port := sshProcessMarkers(argv)
	pid, err := waitForSSHProcessStart(host, port, 30*time.Second)
	if err != nil {
		return
	}
	_ = waitForProcessExit(pid, 0)
}

func waitForSSHProcessStart(host, port string, timeout time.Duration) (int, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		pid, err := FindSSHProcessPID(host, port, nil)
		if err == nil && pid > 0 {
			return pid, nil
		}
		time.Sleep(250 * time.Millisecond)
	}
	return 0, fmt.Errorf("ssh process not found within %s", timeout)
}

func waitForProcessExit(pid int, timeout time.Duration) error {
	if pid <= 0 {
		return fmt.Errorf("invalid pid")
	}
	if timeout > 0 {
		cmd := exec.Command("powershell", "-NoProfile", "-Command",
			fmt.Sprintf("Wait-Process -Id %d -Timeout %d", pid, int(timeout.Seconds())))
		return cmd.Run()
	}
	for {
		if !processExists(pid) {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func processExists(pid int) bool {
	cmd := exec.Command("tasklist", "/FI", fmt.Sprintf("PID eq %d", pid), "/NH")
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(out), strconv.Itoa(pid))
}

func writeSSHBatch(argv []string) (string, error) {
	dir := os.TempDir()
	f, err := os.CreateTemp(dir, "mewsh-ssh-*.cmd")
	if err != nil {
		return "", err
	}
	name := f.Name()
	if !strings.HasSuffix(strings.ToLower(name), ".cmd") {
		newName := name + ".cmd"
		f.Close()
		if err := os.Rename(name, newName); err != nil {
			os.Remove(name)
			return "", err
		}
		name = newName
		f, err = os.OpenFile(name, os.O_WRONLY|os.O_TRUNC, 0700)
		if err != nil {
			os.Remove(name)
			return "", err
		}
	}

	line := shellJoin(argv)
	content := "@echo off\r\ntitle " + WindowTitle + "\r\n" + line + "\r\nexit /b %ERRORLEVEL%\r\n"
	if _, err := f.WriteString(content); err != nil {
		f.Close()
		os.Remove(name)
		return "", err
	}
	if err := f.Close(); err != nil {
		os.Remove(name)
		return "", err
	}
	return filepath.Clean(name), nil
}
