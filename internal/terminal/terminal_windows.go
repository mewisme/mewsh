//go:build windows

package terminal

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/windows"
)

func spawnDetachedPlatform(argv []string, extraEnv []string) error {
	sshLine := shellJoin(argv)
	fromTUI := os.Getenv("MEWSH_TUI") == "1"
	attempts := []func() error{}

	// Prefer Windows Terminal — least likely to disturb the parent Bubble Tea console.
	attempts = append(attempts, func() error {
		if _, err := exec.LookPath("wt"); err != nil {
			return err
		}
		args := append([]string{"-w", "0", "nt", "--title", WindowTitle, "--"}, argv...)
		return startInNewConsoleEnv(extraEnv, "wt", args...)
	})

	if script, err := writeSSHBatch(argv); err == nil {
		scriptPath := script
		attempts = append(attempts, func() error {
			return startInNewConsoleEnv(extraEnv, "cmd", "/C", scriptPath)
		})
		go func() {
			time.Sleep(time.Hour)
			_ = os.Remove(scriptPath)
		}()
	}

	// cmd /k last — can confuse ConPTY when spawned from the TUI parent.
	if !fromTUI {
		attempts = append(attempts, func() error {
			return startInNewConsoleEnv(extraEnv, "cmd", "/k", "title "+WindowTitle+" & "+sshLine)
		})
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

// startInNewConsoleEnv runs a process in a fresh console without touching the parent
// terminal (required when spawning from a Bubble Tea TUI).
func startInNewConsoleEnv(extraEnv []string, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Env = mergeEnv(os.Environ(), extraEnv)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: windows.CREATE_NEW_CONSOLE,
	}
	return cmd.Start()
}

func mergeEnv(base, extra []string) []string {
	if len(extra) == 0 {
		return base
	}
	out := make([]string, 0, len(base)+len(extra))
	out = append(out, base...)
	out = append(out, extra...)
	return out
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
		if err := startInNewConsoleEnv(nil, "wt", args...); err != nil {
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
		cmd := exec.Command("cmd", "/C", script)
		cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: windows.CREATE_NEW_CONSOLE}
		return cmd.Run()
	}
}

func runInteractivePowerShell(argv []string) func() error {
	return func() error {
		sshLine := shellJoin(argv)
		ps := fmt.Sprintf("& {%s}; exit $LASTEXITCODE", sshLine)
		cmd := exec.Command("powershell", "-NoProfile", "-Command", "title "+WindowTitle+"; "+ps)
		cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: windows.CREATE_NEW_CONSOLE}
		return cmd.Run()
	}
}

func runInteractiveCmd(argv []string) func() error {
	return func() error {
		sshLine := shellJoin(argv)
		cmd := exec.Command("cmd", "/C", "title "+WindowTitle+" && "+sshLine)
		cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: windows.CREATE_NEW_CONSOLE}
		return cmd.Run()
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
	content := "@echo off\r\ntitle " + WindowTitle + "\r\n" + line + "\r\nif errorlevel 1 (\r\n  echo.\r\n  echo SSH exited with error %ERRORLEVEL%.\r\n  pause\r\n)\r\n"
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
