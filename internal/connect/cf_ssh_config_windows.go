//go:build windows

package connect

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/mewisme/mewsh/internal/cloudflared"
	"github.com/mewisme/mewsh/internal/config"
	"github.com/mewisme/mewsh/internal/profile"
)

var sshConfigHostSanitizer = regexp.MustCompile(`[^a-zA-Z0-9_-]+`)

func buildCloudflareSSHConfigArgs(cfg *config.Config, p profile.Profile) ([]string, func(), error) {
	cfPath, err := cloudflared.ResolvePathForConnect(cfg)
	if err != nil {
		return nil, nil, err
	}
	cfPath = filepath.Clean(cfPath)
	if _, err := os.Stat(cfPath); err != nil {
		return nil, nil, fmt.Errorf("cloudflared binary not found at %s: %w", cfPath, err)
	}

	hostAlias := sshConfigHostAlias(p.Alias)
	content := renderCloudflareSSHConfig(p, cfPath, hostAlias)

	path, err := writeSSHConfigFile(p.Alias, content)
	if err != nil {
		return nil, nil, err
	}

	// Config must outlive this function — detached ssh starts after we return.
	argv := []string{"ssh", "-F", path, hostAlias}
	return argv, nil, nil
}

func writeSSHConfigFile(alias, content string) (string, error) {
	if err := config.EnsureDir(); err != nil {
		return "", err
	}
	dir, err := config.Dir()
	if err != nil {
		return "", err
	}
	sshDir := filepath.Join(dir, "ssh-configs")
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		return "", err
	}
	safe := sshConfigHostSanitizer.ReplaceAllString(alias, "-")
	if safe == "" {
		safe = "session"
	}
	path := filepath.Join(sshDir, safe+".conf")
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		return "", err
	}
	return path, nil
}

func sshConfigHostAlias(alias string) string {
	safe := sshConfigHostSanitizer.ReplaceAllString(alias, "-")
	if safe == "" {
		safe = "session"
	}
	return "mewsh-" + safe
}

func renderCloudflareSSHConfig(p profile.Profile, cfPath, hostAlias string) string {
	p.ApplyDefaults()
	port := p.Port
	if port == 0 {
		port = 22
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Host %s\n", hostAlias)
	fmt.Fprintf(&b, "  HostName %s\n", p.CFHostname)
	fmt.Fprintf(&b, "  User %s\n", p.User)
	fmt.Fprintf(&b, "  Port %d\n", port)
	fmt.Fprintf(&b, "  ProxyCommand %s\n", sshConfigProxyCommand(cfPath, p.CFHostname))
	b.WriteString("  GSSAPIAuthentication no\n")
	b.WriteString("  ConnectTimeout 30\n")
	b.WriteString("  StrictHostKeyChecking accept-new\n")
	b.WriteString("  RequestTTY force\n")

	if p.AuthType == profile.AuthPassword {
		b.WriteString("  PreferredAuthentications keyboard-interactive,password\n")
		b.WriteString("  PubkeyAuthentication no\n")
		b.WriteString("  NumberOfPasswordPrompts 3\n")
	}
	if p.AuthType == profile.AuthKey && p.KeyPath != "" {
		fmt.Fprintf(&b, "  IdentityFile %s\n", p.KeyPath)
	}
	return b.String()
}

func sshConfigProxyCommand(cfPath, hostname string) string {
	// OpenSSH on Windows passes ProxyCommand to CreateProcessW: only the executable
	// may be quoted, not the whole "cloudflared … access ssh …" string.
	exe := filepath.Clean(cfPath)
	if strings.ContainsAny(exe, " \t") {
		exe = `"` + strings.ReplaceAll(exe, `"`, `\"`) + `"`
	}
	return fmt.Sprintf("%s access ssh --hostname %s", exe, hostname)
}
