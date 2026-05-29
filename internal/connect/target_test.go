package connect

import (
	"testing"

	"github.com/mewisme/mewsh/internal/profile"
)

func TestSessionDisplayTarget(t *testing.T) {
	p := profile.Profile{
		User:           "mew",
		Host:           "10.0.0.1",
		Port:           22,
		ConnectionType: profile.ConnectionCloudflareAccess,
		CFHostname:     "ssh.example.com",
	}
	if got := sessionDisplayTarget(p, "127.0.0.1", 55021); got != "mew@127.0.0.1:55021" {
		t.Fatalf("got %q", got)
	}
	if got := sessionDisplayTarget(p, "", 0); got != "mew@ssh.example.com" {
		t.Fatalf("cf fallback got %q", got)
	}
}
