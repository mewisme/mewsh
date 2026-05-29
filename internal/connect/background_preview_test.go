//go:build windows

package connect

import (
	"strings"
	"testing"

	"github.com/mewisme/mewsh/internal/config"
	"github.com/mewisme/mewsh/internal/profile"
	"github.com/mewisme/mewsh/internal/terminal"
)

func TestPreviewInteractiveSSHArgvNoBackgroundFlags(t *testing.T) {
	cfg := &config.Config{
		Profiles: []profile.Profile{{
			Alias:          "optiplex",
			User:           "mew",
			Port:           22,
			ConnectionType: profile.ConnectionCloudflareAccess,
			CFHostname:     "ssh.example.com",
			AuthType:       profile.AuthPassword,
		}},
	}
	p := cfg.Profiles[0]
	argv, err := previewInteractiveSSHArgv(cfg, p)
	if err != nil {
		t.Fatal(err)
	}
	line := terminal.FormatCommand(argv)
	for _, flag := range []string{"-n", "-T", "sleep", "infinity"} {
		if strings.Contains(line, flag) {
			t.Fatalf("interactive preview must not include %q: %s", flag, line)
		}
	}
	if !strings.Contains(line, "-F") || !strings.Contains(line, "mewsh-optiplex") {
		t.Fatalf("expected ssh config host alias, got %s", line)
	}
}

func TestPreviewWorkerSSHArgvIncludesSleep(t *testing.T) {
	p := profile.Profile{
		User:           "mew",
		Port:           22,
		ConnectionType: profile.ConnectionCloudflareAccess,
		CFHostname:     "ssh.example.com",
		AuthType:       profile.AuthKey,
		KeyPath:        "C:\\tmp\\id_rsa",
	}
	argv, _, err := previewWorkerSSHArgv(&config.Config{}, p)
	if err != nil {
		t.Fatal(err)
	}
	line := terminal.FormatCommand(argv)
	if !strings.Contains(line, "sleep") || !strings.Contains(line, "infinity") {
		t.Fatalf("worker preview missing sleep infinity: %s", line)
	}
	if !strings.Contains(line, "-n") || !strings.Contains(line, "-T") {
		t.Fatalf("worker preview missing -n -T: %s", line)
	}
}
