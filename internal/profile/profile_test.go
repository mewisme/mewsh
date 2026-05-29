package profile

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestValidate(t *testing.T) {
	tmp := t.TempDir()
	keyPath := filepath.Join(tmp, "id_rsa")
	if err := os.WriteFile(keyPath, []byte("key"), 0600); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		profile Profile
		aliases []string
		edit    bool
		wantErr bool
	}{
		{
			name: "valid direct key",
			profile: Profile{
				Alias:          "prod",
				Host:           "example.com",
				Port:           22,
				User:           "ubuntu",
				AuthType:       AuthKey,
				KeyPath:        keyPath,
				ConnectionType: ConnectionDirect,
			},
		},
		{
			name: "duplicate alias",
			profile: Profile{
				Alias:          "prod",
				Host:           "example.com",
				User:           "ubuntu",
				AuthType:       AuthAgent,
				ConnectionType: ConnectionDirect,
			},
			aliases: []string{"prod"},
			wantErr: true,
		},
		{
			name: "invalid port",
			profile: Profile{
				Alias:          "prod",
				Host:           "example.com",
				Port:           70000,
				User:           "ubuntu",
				AuthType:       AuthAgent,
				ConnectionType: ConnectionDirect,
			},
			wantErr: true,
		},
		{
			name: "cf requires hostname",
			profile: Profile{
				Alias:          "cf",
				User:           "ubuntu",
				AuthType:       AuthAgent,
				ConnectionType: ConnectionCloudflareAccess,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.profile.Validate(tt.aliases, tt.edit)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBuildSSHArgsBackgroundKeepsSessionOpen(t *testing.T) {
	p := Profile{User: "mew", AuthType: AuthKey, KeyPath: "/tmp/id_rsa"}
	args := p.BuildSSHArgs("127.0.0.1", 22, true)
	joined := stringsJoinArgs(args)
	for _, want := range []string{"-n", "-T", "ServerAliveInterval", "sleep", "infinity", "mew@127.0.0.1"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("missing %q in %v", want, args)
		}
	}
}

func TestBuildSSHArgs(t *testing.T) {
	p := Profile{
		User:     "ubuntu",
		AuthType: AuthKey,
		KeyPath:  "/home/user/.ssh/id_rsa",
	}
	args := p.BuildSSHArgs("127.0.0.1", 2222, false)
	if args[0] != "ssh" {
		t.Fatalf("argv[0] = %q, want ssh", args[0])
	}
	if runtime.GOOS == "windows" {
		if !strings.Contains(stringsJoinArgs(args), "-tt ") {
			t.Fatalf("windows key auth should use -tt, got %v", args)
		}
	}
	joined := stringsJoinArgs(args)
	for _, want := range []string{"-p", "2222", "-i", "/home/user/.ssh/id_rsa", "ubuntu@127.0.0.1"} {
		if !containsArg(joined, want) {
			t.Fatalf("missing %q in %v", want, args)
		}
	}
}

func stringsJoinArgs(args []string) string {
	s := ""
	for _, a := range args {
		s += a + " "
	}
	return s
}

func containsArg(haystack, needle string) bool {
	return strings.Contains(haystack, needle+" ")
}
