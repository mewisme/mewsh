package connect

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/mewisme/mewsh/internal/cliui"
	"github.com/mewisme/mewsh/internal/config"
	"github.com/mewisme/mewsh/internal/profile"
	"github.com/mewisme/mewsh/internal/terminal"
)

// LaunchBackground starts a detached mewsh worker that keeps Cloudflare tunnels alive
// and runs SSH without a terminal emulator (for headless servers and scripts).
func LaunchBackground(cfg *config.Config, alias string, o Options) error {
	p, idx := cfg.FindByAlias(alias)
	if idx < 0 {
		return fmt.Errorf("profile %q not found", alias)
	}

	logPath, err := sessionLogPath(alias)
	if err != nil {
		return err
	}

	exe, err := os.Executable()
	if err != nil {
		return err
	}

	sessionID := newSessionID(alias)
	args := backgroundWorkerArgs(alias)
	cmd := exec.Command(exe, args...)
	cmd.Env = append(os.Environ(),
		"MEWSH_BG_LOG="+logPath,
		sessionIDEnv+"="+sessionID,
	)
	devNull, err := os.Open(os.DevNull)
	if err != nil {
		return err
	}
	cmd.Stdin = devNull

	log, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return err
	}
	cmd.Stdout = log
	cmd.Stderr = log

	configureSupervisorCmd(cmd)

	if err := cmd.Start(); err != nil {
		_ = log.Close()
		return err
	}
	_ = log.Close()

	if err := persistBackgroundSupervisor(sessionID, alias, cmd.Process.Pid, logPath); err != nil {
		return fmt.Errorf("record background session: %w", err)
	}

	cliui.OKf(o.status, "Background connect %q started (session %s, worker pid %d).", alias, sessionID, cmd.Process.Pid)
	cliui.Labeled(o.status, "log", logPath)
	printBackgroundSSHHints(cfg, *p, o)
	return nil
}

func printBackgroundSSHHints(cfg *config.Config, p profile.Profile, o Options) {
	if o.quiet || o.status == nil {
		return
	}
	if argv, err := previewInteractiveSSHArgv(cfg, p); err == nil && len(argv) > 0 {
		cliui.Section(o.status, "Interactive (open a terminal)")
		cliui.Block(o.status, cliui.LevelCmd, terminal.FormatCommand(argv))
	}
	if argv, note, err := previewWorkerSSHArgv(cfg, p); err == nil && len(argv) > 0 {
		cliui.Section(o.status, "Background worker (mewsh -b)")
		cliui.Block(o.status, cliui.LevelCmd, terminal.FormatCommand(argv))
		if note != "" {
			cliui.Lines(o.status, cliui.LevelDim, note)
		}
	}
}

// previewInteractiveSSHArgv is for copy/paste into your own terminal (TTY, stays open after login).
func previewInteractiveSSHArgv(cfg *config.Config, p profile.Profile) ([]string, error) {
	o := defaultOptions()
	switch p.ConnectionType {
	case profile.ConnectionCloudflareAccess:
		if runtime.GOOS == "windows" {
			argv, _, err := buildCloudflareSSHConfigArgs(cfg, p, false)
			return argv, err
		}
		return buildLaunchArgs(p, p.CFHostname, p.Port, o)
	default:
		return buildLaunchArgs(p, p.Host, p.Port, o)
	}
}

// previewWorkerSSHArgv matches the command the background worker runs (no TTY; sleep infinity).
func previewWorkerSSHArgv(cfg *config.Config, p profile.Profile) ([]string, string, error) {
	o := backgroundOptions()
	switch p.ConnectionType {
	case profile.ConnectionCloudflareAccess:
		argv, err := buildLaunchArgs(p, "127.0.0.1", 0, o)
		if err != nil {
			return nil, "", err
		}
		for i := 0; i < len(argv)-1; i++ {
			if argv[i] == "-p" {
				argv[i+1] = "<local-port>"
				break
			}
		}
		return argv, "Replace <local-port> with the port from the log line \"Tunnel ready on 127.0.0.1:...\"", nil
	default:
		argv, err := buildLaunchArgs(p, p.Host, p.Port, o)
		return argv, "", err
	}
}

// RunBackgroundWorker is invoked by the hidden __bg-connect__ command. It blocks until
// the SSH session ends.
func backgroundOptions() Options {
	o := defaultOptions()
	o.background = true
	return o
}

func RunBackgroundWorker(cfg *config.Config, alias string) error {
	o := backgroundOptions()
	o.status = os.Stderr
	redirectBackgroundWorkerIO()
	terminal.DetachFromConsole()

	p, idx := cfg.FindByAlias(alias)
	if idx < 0 {
		return fmt.Errorf("profile %q not found", alias)
	}
	aliases := profileAliases(cfg)
	if err := p.Validate(aliases, true); err != nil {
		return err
	}

	if p.ConnectionType == profile.ConnectionCloudflareAccess {
		// Always keep one cloudflared tunnel in the worker. ProxyCommand spawns
		// cloudflared on every SSH handshake (visible window flashes on Windows).
		return connectCloudflareBackgroundHold(cfg, *p, o)
	}
	return connectDirectBackgroundHold(*p, o)
}

func redirectBackgroundWorkerIO() {
	logPath := os.Getenv("MEWSH_BG_LOG")
	if logPath == "" {
		return
	}
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return
	}
	os.Stdout = f
	os.Stderr = f
	if devNull, err := os.Open(os.DevNull); err == nil {
		os.Stdin = devNull
	}
}

func connectDirectBackgroundHold(p profile.Profile, o Options) error {
	argv, err := buildLaunchArgs(p, p.Host, p.Port, o)
	if err != nil {
		return err
	}
	return runBackgroundSSH(p, "", argv, sessionDisplayTarget(p, p.Host, p.Port), o)
}

func connectCloudflareBackgroundHold(cfg *config.Config, p profile.Profile, o Options) error {
	hostname := p.CFHostname
	tun, _, err := acquireSharedTunnel(cfg, hostname, o)
	if err != nil {
		return err
	}
	defer releaseSharedTunnel(hostname)

	argv, err := buildLaunchArgs(p, "127.0.0.1", tun.LocalPort, o)
	if err != nil {
		return err
	}
	return runBackgroundSSH(p, hostname, argv, sessionDisplayTarget(p, "127.0.0.1", tun.LocalPort), o)
}

func runBackgroundSSH(p profile.Profile, hostname string, argv []string, displayTarget string, o Options) error {
	extraEnv, cleanup, err := spawnExtraEnv(p)
	if err != nil {
		return err
	}
	if cleanup != nil {
		defer cleanup()
	}

	logPath := os.Getenv("MEWSH_BG_LOG")
	if logPath == "" {
		var err error
		logPath, err = sessionLogPath(p.Alias)
		if err != nil {
			return err
		}
	}

	cliui.Infof(o.status, "Starting SSH for %q (log %s).", p.Alias, logPath)

	sessionID := registerSSHSession(p.Alias, hostname, argv, displayTarget)
	defer removeSSHSession(sessionID, false)

	pid, err := terminal.SpawnBackgroundEnv(argv, extraEnv, logPath)
	if err != nil {
		return err
	}
	setSSHSessionPID(sessionID, pid)

	if terminal.ProcessExists(pid) {
		terminal.WaitSSHProcessExit(pid)
	} else {
		terminal.WaitSSHSessionEnd(argv)
	}
	return nil
}

func sessionLogPath(alias string) (string, error) {
	dir, err := config.Dir()
	if err != nil {
		return "", err
	}
	sessions := filepath.Join(dir, "sessions")
	if err := os.MkdirAll(sessions, 0700); err != nil {
		return "", err
	}
	safe := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		return '_'
	}, alias)
	if safe == "" {
		safe = "profile"
	}
	name := fmt.Sprintf("%s-%d.log", safe, time.Now().Unix())
	return filepath.Join(sessions, name), nil
}

// backgroundWorkerArgs builds argv for re-exec; configOverride is read from cmd package.
var backgroundConfigOverride func() string

// SetBackgroundConfigOverride registers how to pass --config to the background worker.
func SetBackgroundConfigOverride(fn func() string) {
	backgroundConfigOverride = fn
}

func backgroundWorkerArgs(alias string) []string {
	var args []string
	if backgroundConfigOverride != nil {
		if path := backgroundConfigOverride(); path != "" {
			args = append(args, "--config", path)
		}
	}
	args = append(args, "__bg-connect__", alias)
	return args
}
