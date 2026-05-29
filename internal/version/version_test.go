package version

import (
	"sync"
	"testing"
)

func TestTag(t *testing.T) {
	t.Parallel()
	orig := Version
	t.Cleanup(func() { Version = orig; resolved = sync.Once{} })

	Version = "0.0.2"
	resolved = sync.Once{}
	if got := Tag(); got != "v0.0.2" {
		t.Fatalf("Tag() = %q, want v0.0.2", got)
	}

	Version = "dev"
	resolved = sync.Once{}
	if got := Tag(); got != "" {
		t.Fatalf("Tag() = %q, want empty for dev", got)
	}
}
