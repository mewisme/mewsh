package version

import "testing"

func TestFormatBuildDate(t *testing.T) {
	got := FormatBuildDate("2024-06-01T12:00:00Z")
	if got == "" || got == "2024-06-01T12:00:00Z" {
		t.Fatalf("expected formatted date, got %q", got)
	}
}

func TestBuildInfoDev(t *testing.T) {
	// Default in tests is dev unless ldflags set.
	b := BuildInfo()
	if b.GOOS == "" || b.GOARCH == "" {
		t.Fatal("expected platform")
	}
}
