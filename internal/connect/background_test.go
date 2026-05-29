package connect

import (
	"strings"
	"testing"
)

func TestBackgroundWorkerArgs(t *testing.T) {
	backgroundConfigOverride = func() string { return "" }
	args := backgroundWorkerArgs("prod")
	if len(args) != 2 || args[0] != "__bg-connect__" || args[1] != "prod" {
		t.Fatalf("args = %v", args)
	}

	backgroundConfigOverride = func() string { return "/tmp/mewsh.json" }
	args = backgroundWorkerArgs("prod")
	want := []string{"--config", "/tmp/mewsh.json", "__bg-connect__", "prod"}
	if len(args) != len(want) {
		t.Fatalf("args = %v, want %v", args, want)
	}
	for i := range want {
		if args[i] != want[i] {
			t.Fatalf("args[%d] = %q, want %q", i, args[i], want[i])
		}
	}
}

func TestSessionLogPathSanitize(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	path, err := sessionLogPath("my/server")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(path, "my_server-") {
		t.Fatalf("path = %q", path)
	}
}
