package connect

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSessionRegistryRoundTrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("APPDATA", dir)

	id := "prod-123"
	if err := persistBackgroundSupervisor(id, "prod", 99999, filepath.Join(dir, "sessions", "prod.log")); err != nil {
		t.Fatal(err)
	}

	got, err := sessionStoreGet(id)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.Alias != "prod" || got.WorkerPID != 99999 {
		t.Fatalf("got %#v", got)
	}

	path, err := registryPath()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}

	sessionStoreRemove(id)
	got, _ = sessionStoreGet(id)
	if got != nil {
		t.Fatalf("expected removed, got %#v", got)
	}
}
