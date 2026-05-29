package connect

import (
	"testing"
	"time"
)

func TestShouldListStoredSessionGrace(t *testing.T) {
	s := storedSession{
		ID:        "prod-1",
		Alias:     "prod",
		WorkerPID: 999999999, // unlikely to exist
		StartedAt: time.Now(),
	}
	if !shouldListStoredSession(s) {
		t.Fatal("expected new session to be listed during spawn grace")
	}
	s.StartedAt = time.Now().Add(-backgroundSessionGrace - time.Second)
	if shouldListStoredSession(s) {
		t.Fatal("expected old dead session to be pruned")
	}
}
