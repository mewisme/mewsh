package tunnel

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestWaitForTunnelReady(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	port := ln.Addr().(*net.TCPAddr).Port

	if err := waitForTunnelReady(context.Background(), port, nil, 2*time.Second, nil); err != nil {
		t.Fatalf("expected ready, got %v", err)
	}
}

func TestWaitForTunnelReadyTimeout(t *testing.T) {
	port, err := PickFreePort()
	if err != nil {
		t.Fatal(err)
	}
	err = waitForTunnelReady(context.Background(), port, nil, 500*time.Millisecond, nil)
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestPortAccepts(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	addr := ln.Addr().String()
	if !portAccepts(addr) {
		t.Fatal("expected port to accept connections")
	}
}
