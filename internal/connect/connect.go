package connect

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"
	"syscall"

	"github.com/mewisme/mewsh/internal/cloudflared"
	"github.com/mewisme/mewsh/internal/config"
	"github.com/mewisme/mewsh/internal/profile"
	"github.com/mewisme/mewsh/internal/secret"
	"github.com/mewisme/mewsh/internal/terminal"
	"github.com/mewisme/mewsh/internal/tunnel"
)

func Profile(cfg *config.Config, alias string, opts ...Option) error {
	o := defaultOptions()
	if !o.quiet && o.status == nil {
		o.status = os.Stderr
	}
	for _, opt := range opts {
		opt(&o)
	}

	p, idx := cfg.FindByAlias(alias)
	if idx < 0 {
		return fmt.Errorf("profile %q not found", alias)
	}
	aliases := profileAliases(cfg)
	if err := p.Validate(aliases, true); err != nil {
		return err
	}

	if p.ConnectionType == profile.ConnectionCloudflareAccess {
		if o.detached {
			return connectCloudflareDetached(cfg, *p, o)
		}
		return connectCloudflare(cfg, *p, o)
	}
	return connectDirect(*p)
}

func connectDirect(p profile.Profile) error {
	argv, err := buildLaunchArgs(p, p.Host, p.Port)
	if err != nil {
		return err
	}
	if err := detachedSpawn(p, argv); err != nil {
		return err
	}
	sessionID := registerSSHSession(p.Alias, "", argv)
	argvCopy := append([]string(nil), argv...)
	go monitorDetachedSSH(sessionID, argvCopy)
	return nil
}

func connectCloudflare(cfg *config.Config, p profile.Profile, o Options) error {
	hostname := p.CFHostname
	tun, _, err := acquireSharedTunnel(cfg, hostname, o)
	if err != nil {
		return err
	}

	argv, err := buildLaunchArgs(p, "127.0.0.1", tun.LocalPort)
	if err != nil {
		releaseSharedTunnel(hostname)
		return err
	}

	sessionID := registerSSHSession(p.Alias, hostname, argv)
	defer removeSSHSession(sessionID, false)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	go func() {
		<-sigCh
		CleanupActive()
	}()

	err = terminal.RunInteractive(argv, func(pid int) {
		setSSHSessionPID(sessionID, pid)
	})
	return err
}

func connectCloudflareDetached(cfg *config.Config, p profile.Profile, o Options) error {
	// Windows OpenSSH + password auth hangs when using a pre-started localhost tunnel;
	// use cloudflared as ProxyCommand instead (matches Cloudflare's recommended ssh config).
	if runtime.GOOS == "windows" {
		return connectCloudflareDetachedProxy(cfg, p)
	}

	hostname := p.CFHostname
	tun, _, err := acquireSharedTunnel(cfg, hostname, o)
	if err != nil {
		return err
	}

	argv, err := buildLaunchArgs(p, "127.0.0.1", tun.LocalPort)
	if err != nil {
		releaseSharedTunnel(hostname)
		return err
	}

	if err := detachedSpawn(p, argv); err != nil {
		releaseSharedTunnel(hostname)
		return err
	}

	sessionID := registerSSHSession(p.Alias, hostname, argv)
	argvCopy := append([]string(nil), argv...)
	go monitorDetachedSSH(sessionID, argvCopy)
	return nil
}

func connectCloudflareDetachedProxy(cfg *config.Config, p profile.Profile) error {
	argv, _, err := buildCloudflareProxyArgs(cfg, p)
	if err != nil {
		return err
	}
	if err := detachedSpawn(p, argv); err != nil {
		return err
	}
	sessionID := registerSSHSession(p.Alias, p.CFHostname, argv)
	argvCopy := append([]string(nil), argv...)
	go monitorDetachedSSH(sessionID, argvCopy)
	return nil
}

func startCloudflareTunnel(cfg *config.Config, hostname string, o Options) (*tunnel.Tunnel, context.CancelFunc, error) {
	cfPath, err := cloudflared.ResolvePathForConnect(cfg)
	if err != nil {
		return nil, nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())

	o.say("Starting Cloudflare Access tunnel...\n")
	tun, err := tunnel.StartCloudflareAccessTunnel(ctx, cfPath, hostname)
	if err != nil {
		cancel()
		return nil, nil, err
	}

	if err := tun.EnsureReady(); err != nil {
		_ = tun.Close()
		cancel()
		return nil, nil, fmt.Errorf("tunnel not ready before ssh: %w", err)
	}

	o.say(fmt.Sprintf("Tunnel ready on 127.0.0.1:%d, opening SSH...\n", tun.LocalPort))
	return tun, cancel, nil
}

func buildLaunchArgs(p profile.Profile, host string, port int) ([]string, error) {
	if p.AuthType == profile.AuthPassword && p.PasswordMode == profile.PasswordAuto {
		return buildAutoPasswordArgs(p, host, port)
	}
	return p.BuildSSHArgs(host, port), nil
}

func buildAutoPasswordArgs(p profile.Profile, host string, port int) ([]string, error) {
	if runtime.GOOS == "windows" {
		// Password is supplied via SSH_ASKPASS in spawnExtraEnv.
		return p.BuildSSHArgs(host, port), nil
	}
	pass, err := secret.GetPassword(p.PasswordRef)
	if err != nil {
		return nil, err
	}
	sshArgs := p.BuildSSHArgs(host, port)

	if expectPath, err := exec.LookPath("expect"); err == nil {
		script, err := writeExpectScript(sshArgs, pass)
		if err != nil {
			return nil, err
		}
		return []string{expectPath, script}, nil
	}

	if sshpassPath, err := exec.LookPath("sshpass"); err == nil {
		script, err := writeSSHPASSScript(sshpassPath, sshArgs, pass)
		if err != nil {
			return nil, err
		}
		if runtime.GOOS == "windows" {
			return nil, fmt.Errorf("auto password requires expect or sshpass; use manual mode")
		}
		return []string{"/bin/sh", script}, nil
	}

	return nil, fmt.Errorf("auto password requires expect or sshpass; use manual mode")
}

func writeExpectScript(sshArgs []string, password string) (string, error) {
	f, err := os.CreateTemp("", "mewsh-expect-*.exp")
	if err != nil {
		return "", err
	}
	escaped := escapeExpect(password)
	content := fmt.Sprintf(`#!/usr/bin/expect -f
set timeout -1
spawn ssh %s
expect {
  -re "(?i)password:" { send "%s\r" }
}
interact
`, strings.Join(quoteExpectArgs(sshArgs[1:]), " "), escaped)
	if _, err := f.WriteString(content); err != nil {
		f.Close()
		os.Remove(f.Name())
		return "", err
	}
	if err := f.Chmod(0700); err != nil {
		f.Close()
		os.Remove(f.Name())
		return "", err
	}
	if err := f.Close(); err != nil {
		os.Remove(f.Name())
		return "", err
	}
	return f.Name(), nil
}

func writeSSHPASSScript(sshpassPath string, sshArgs []string, password string) (string, error) {
	f, err := os.CreateTemp("", "mewsh-sshpass-*.sh")
	if err != nil {
		return "", err
	}
	line := fmt.Sprintf("export SSHPASS=%q\nexec %q -e ssh %s\n", password, sshpassPath, strings.Join(quoteShell(sshArgs[1:]), " "))
	if _, err := f.WriteString("#!/bin/sh\n" + line); err != nil {
		f.Close()
		os.Remove(f.Name())
		return "", err
	}
	if err := f.Chmod(0700); err != nil {
		f.Close()
		os.Remove(f.Name())
		return "", err
	}
	if err := f.Close(); err != nil {
		os.Remove(f.Name())
		return "", err
	}
	return f.Name(), nil
}

func quoteShell(args []string) []string {
	out := make([]string, len(args))
	for i, a := range args {
		out[i] = fmt.Sprintf("%q", a)
	}
	return out
}

func quoteExpectArgs(args []string) []string {
	out := make([]string, len(args))
	for i, a := range args {
		out[i] = fmt.Sprintf("{%s}", a)
	}
	return out
}

func escapeExpect(s string) string {
	repl := map[rune]string{
		'\\': `\\`,
		'"':  `\"`,
		'$':  `\$`,
		'[':  `\[`,
		']':  `\]`,
	}
	out := make([]rune, 0, len(s))
	for _, r := range s {
		if v, ok := repl[r]; ok {
			for _, c := range v {
				out = append(out, c)
			}
		} else {
			out = append(out, r)
		}
	}
	return string(out)
}

func profileAliases(cfg *config.Config) []string {
	out := make([]string, len(cfg.Profiles))
	for i, p := range cfg.Profiles {
		out[i] = p.Alias
	}
	return out
}
