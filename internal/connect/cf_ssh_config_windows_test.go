//go:build windows

package connect

import "testing"

func TestSSHConfigProxyCommand(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		cfPath   string
		hostname string
		want     string
	}{
		{
			name:     "no spaces",
			cfPath:   `C:\Users\Mew\AppData\mewsh\bin\cloudflared.exe`,
			hostname: "ssh.example.com",
			want:     `C:\Users\Mew\AppData\mewsh\bin\cloudflared.exe access ssh --hostname ssh.example.com`,
		},
		{
			name:     "spaces in path",
			cfPath:   `C:\Program Files\cloudflared\cloudflared.exe`,
			hostname: "ssh.example.com",
			want:     `"C:\Program Files\cloudflared\cloudflared.exe" access ssh --hostname ssh.example.com`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sshConfigProxyCommand(tt.cfPath, tt.hostname)
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}
