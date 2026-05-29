package terminal

import "testing"

func TestSSHDestinationArg(t *testing.T) {
	argv := []string{"ssh", "-n", "-T", "-p", "55021", "mew@127.0.0.1", "sleep", "infinity"}
	if got := sshDestinationArg(argv); got != "mew@127.0.0.1" {
		t.Fatalf("got %q, want mew@127.0.0.1", got)
	}
	host, port := sshProcessMarkers(argv)
	if host != "mew@127.0.0.1" || port != "55021" {
		t.Fatalf("markers host=%q port=%q", host, port)
	}
}
