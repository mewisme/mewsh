package profile

import (
	"os"
	"path/filepath"
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

func TestBuildSSHArgs(t *testing.T) {
	p := Profile{
		User:     "ubuntu",
		AuthType: AuthKey,
		KeyPath:  "/home/user/.ssh/id_rsa",
	}
	args := p.BuildSSHArgs("127.0.0.1", 2222)
	want := []string{"ssh", "-p", "2222", "-i", "/home/user/.ssh/id_rsa", "ubuntu@127.0.0.1"}
	if len(args) != len(want) {
		t.Fatalf("got %v want %v", args, want)
	}
	for i := range want {
		if args[i] != want[i] {
			t.Fatalf("got %v want %v", args, want)
		}
	}
}
