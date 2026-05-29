package terminal

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// WindowTitle is the OS terminal window title (not the in-app TUI header).
const WindowTitle = "MewSH"

func SpawnDetached(argv []string) error {
	if len(argv) == 0 {
		return fmt.Errorf("empty command")
	}
	if err := ensureSSH(argv[0]); err != nil {
		return err
	}
	err := spawnDetachedPlatform(argv)
	if err != nil {
		return printFallback(argv, err)
	}
	return nil
}

// SSHStartedFunc is called once the ssh child process is known (Windows).
type SSHStartedFunc func(pid int)

// WaitSSHSessionEnd blocks until the ssh process for argv exits (best-effort).
func WaitSSHSessionEnd(argv []string) {
	waitSSHSessionEnd(argv)
}

func RunInteractive(argv []string, onStarted SSHStartedFunc) error {
	if len(argv) == 0 {
		return fmt.Errorf("empty command")
	}
	if err := ensureSSH(argv[0]); err != nil {
		return err
	}
	err := runInteractivePlatform(argv, onStarted)
	if err != nil {
		return printFallback(argv, err)
	}
	return nil
}

// ProcessExists reports whether a process with the given PID is still running.
func ProcessExists(pid int) bool {
	return processExists(pid)
}

func ensureSSH(name string) error {
	if name != "ssh" && name != "expect" && name != "sshpass" {
		return nil
	}
	if _, err := exec.LookPath("ssh"); err != nil {
		return fmt.Errorf("ssh is not installed or not available in PATH")
	}
	return nil
}

func printFallback(argv []string, spawnErr error) error {
	fmt.Fprintf(os.Stderr, "failed to spawn terminal: %v\n", spawnErr)
	fmt.Fprintf(os.Stderr, "run manually: %s\n", shellJoin(argv))
	return fmt.Errorf("failed to spawn terminal: %w", spawnErr)
}

// SSHProcessMarkers extracts host and port from ssh argv for session display/kill.
func SSHProcessMarkers(argv []string) (host, port string) {
	return sshProcessMarkers(argv)
}

func sshProcessMarkers(argv []string) (host, port string) {
	for i, arg := range argv {
		if arg == "-p" && i+1 < len(argv) {
			port = argv[i+1]
		}
	}
	if len(argv) > 0 {
		host = argv[len(argv)-1]
	}
	return host, port
}

func shellJoin(argv []string) string {
	parts := make([]string, len(argv))
	for i, a := range argv {
		if strings.ContainsAny(a, " \t\"'") {
			parts[i] = fmt.Sprintf("%q", a)
		} else {
			parts[i] = a
		}
	}
	return strings.Join(parts, " ")
}

func CheckSupport() error {
	if runtime.GOOS == "windows" {
		if _, err := exec.LookPath("wt"); err == nil {
			return nil
		}
		if _, err := exec.LookPath("powershell"); err == nil {
			return nil
		}
		if _, err := exec.LookPath("cmd"); err == nil {
			return nil
		}
		return fmt.Errorf("no supported terminal launcher found (wt, powershell, cmd)")
	}
	if runtime.GOOS == "darwin" {
		if _, err := exec.LookPath("osascript"); err != nil {
			return fmt.Errorf("osascript not found")
		}
		return nil
	}
	for _, term := range linuxTerminals() {
		if _, err := exec.LookPath(term); err == nil {
			return nil
		}
	}
	if term := os.Getenv("TERMINAL"); term != "" {
		if _, err := exec.LookPath(term); err == nil {
			return nil
		}
	}
	return fmt.Errorf("no supported terminal emulator found")
}

func linuxTerminals() []string {
	if term := os.Getenv("TERMINAL"); term != "" {
		return []string{term, "x-terminal-emulator", "gnome-terminal", "konsole", "kitty", "alacritty", "wezterm", "xterm"}
	}
	return []string{"x-terminal-emulator", "gnome-terminal", "konsole", "kitty", "alacritty", "wezterm", "xterm"}
}
