//go:build !windows

package terminal

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

func spawnDetachedPlatform(argv []string, extraEnv []string) error {
	_ = extraEnv
	if runtime.GOOS == "darwin" {
		script := fmt.Sprintf(`tell application "Terminal"
  set t to do script "%s"
  set custom title of t to "%s"
end tell`, escapeAppleScript(shellJoin(argv)), escapeAppleScript(WindowTitle))
		return exec.Command("osascript", "-e", script).Run()
	}
	return spawnLinuxTerminal(argv, false)
}

func waitSSHSessionEnd(argv []string) {
	host, port := sshProcessMarkers(argv)
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := FindSSHProcessPID(host, port, nil); err == nil {
			break
		}
		time.Sleep(250 * time.Millisecond)
	}
	for {
		if _, err := FindSSHProcessPID(host, port, nil); err != nil {
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func runInteractivePlatform(argv []string, onStarted SSHStartedFunc) error {
	_ = onStarted
	if runtime.GOOS == "darwin" {
		wrapper, done, err := writeWrapperScript(argv)
		if err != nil {
			return err
		}
		defer os.Remove(wrapper)
		defer os.Remove(done)
		script := fmt.Sprintf(`tell application "Terminal"
  set t to do script "%s"
  set custom title of t to "%s"
end tell`, escapeAppleScript(wrapper), escapeAppleScript(WindowTitle))
		if err := exec.Command("osascript", "-e", script).Run(); err != nil {
			return err
		}
		return waitForDoneFile(done, 24*time.Hour)
	}
	return spawnLinuxTerminal(argv, true)
}

func spawnLinuxTerminal(argv []string, wait bool) error {
	line := shellJoin(argv)
	terms := linuxTerminals()
	var last error
	for _, term := range terms {
		path, err := exec.LookPath(term)
		if err != nil {
			continue
		}
		base := filepath.Base(path)
		var cmd *exec.Cmd
		switch base {
		case "gnome-terminal":
			if wait {
				cmd = exec.Command(path, "--title", WindowTitle, "--wait", "--")
			} else {
				cmd = exec.Command(path, "--title", WindowTitle, "--")
			}
			cmd.Args = append(cmd.Args, argv...)
		case "konsole":
			cmd = exec.Command(path, "-p", "tabtitle="+WindowTitle, "-e")
			cmd.Args = append(cmd.Args, argv...)
		case "kitty":
			cmd = exec.Command(path, "--title", WindowTitle)
			cmd.Args = append(cmd.Args, argv...)
		case "alacritty", "wezterm":
			cmd = exec.Command(path, "--title", WindowTitle, "-e")
			cmd.Args = append(cmd.Args, argv...)
		case "xterm", "x-terminal-emulator":
			cmd = exec.Command(path, "-title", WindowTitle, "-e")
			cmd.Args = append(cmd.Args, argv...)
		default:
			cmd = exec.Command(path, "-e")
			cmd.Args = append(cmd.Args, argv...)
		}
		if wait {
			if err := cmd.Run(); err != nil {
				last = err
				continue
			}
			return nil
		}
		if err := cmd.Start(); err != nil {
			last = err
			continue
		}
		return nil
	}
	if last == nil {
		last = fmt.Errorf("no terminal found for command: %s", line)
	}
	return last
}

func writeWrapperScript(argv []string) (scriptPath, donePath string, err error) {
	f, err := os.CreateTemp("", "mewsh-ssh-*.sh")
	if err != nil {
		return "", "", err
	}
	donePath = f.Name() + ".done"
	content := "#!/bin/sh\n" + shellJoin(argv) + "\ntouch " + shellQuote(donePath) + "\n"
	if _, err := f.WriteString(content); err != nil {
		f.Close()
		os.Remove(f.Name())
		return "", "", err
	}
	if err := f.Chmod(0700); err != nil {
		f.Close()
		os.Remove(f.Name())
		return "", "", err
	}
	name := f.Name()
	if err := f.Close(); err != nil {
		os.Remove(name)
		return "", "", err
	}
	return name, donePath, nil
}

func waitForDoneFile(done string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(done); err == nil {
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("timed out waiting for SSH session to finish")
}

func shellQuote(s string) string {
	return fmt.Sprintf("%q", s)
}

func escapeAppleScript(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `"`, `\"`)
	return r.Replace(s)
}
