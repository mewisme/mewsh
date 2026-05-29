package tunnel

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"sync/atomic"
	"time"
)

const defaultReadyTimeout = 30 * time.Second

type Tunnel struct {
	LocalPort int
	Cmd       *exec.Cmd
	Cancel    context.CancelFunc
	exited    *atomic.Bool
	stderr    *bytes.Buffer
}

func PickFreePort() (int, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	port := ln.Addr().(*net.TCPAddr).Port
	if err := ln.Close(); err != nil {
		return 0, err
	}
	return port, nil
}

func portAccepts(addr string) bool {
	conn, err := net.DialTimeout("tcp", addr, 300*time.Millisecond)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func WaitForTCP(port int, timeout time.Duration) error {
	return waitForTunnelReady(context.Background(), port, nil, timeout, nil)
}

func waitForTunnelReady(ctx context.Context, port int, exited *atomic.Bool, timeout time.Duration, stderr *bytes.Buffer) error {
	if timeout <= 0 {
		timeout = defaultReadyTimeout
	}
	deadline := time.Now().Add(timeout)
	addr := fmt.Sprintf("127.0.0.1:%d", port)

	for time.Now().Before(deadline) {
		if err := ctx.Err(); err != nil {
			return err
		}
		if exited != nil && exited.Load() {
			return cloudflaredExitError(stderr)
		}
		if portAccepts(addr) {
			// Confirm the listener stays up briefly before handing off to ssh.
			time.Sleep(150 * time.Millisecond)
			if exited != nil && exited.Load() {
				return cloudflaredExitError(stderr)
			}
			if portAccepts(addr) {
				return nil
			}
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(200 * time.Millisecond):
		}
	}
	return fmt.Errorf("tunnel not ready on %s after %s", addr, timeout)
}

func cloudflaredExitError(stderr *bytes.Buffer) error {
	if stderr != nil && stderr.Len() > 0 {
		return fmt.Errorf("cloudflared exited before tunnel was ready: %s", trimStderr(stderr.String()))
	}
	return errors.New("cloudflared exited before tunnel was ready")
}

func trimStderr(s string) string {
	const maxLen = 512
	s = string(bytes.TrimSpace([]byte(s)))
	if len(s) > maxLen {
		return s[len(s)-maxLen:]
	}
	return s
}

func StartCloudflareAccessTunnel(ctx context.Context, cloudflaredPath, hostname string) (*Tunnel, error) {
	if cloudflaredPath == "" {
		return nil, fmt.Errorf("cloudflared is not installed or not available in PATH")
	}
	if _, err := os.Stat(cloudflaredPath); err != nil {
		if _, lookErr := exec.LookPath(cloudflaredPath); lookErr != nil {
			return nil, fmt.Errorf("cloudflared is not installed or not available in PATH")
		}
	}

	port, err := PickFreePort()
	if err != nil {
		return nil, fmt.Errorf("pick free port: %w", err)
	}

	ctx, cancel := context.WithCancel(ctx)
	url := fmt.Sprintf("127.0.0.1:%d", port)
	cmd := exec.CommandContext(ctx, cloudflaredPath, "access", "ssh", "--hostname", hostname, "--url", url)

	var stderr bytes.Buffer
	cmd.Stdout = io.Discard
	cmd.Stderr = io.MultiWriter(io.Discard, &stderr)
	if devNull, err := os.Open(os.DevNull); err == nil {
		cmd.Stdin = devNull
	}
	setHiddenProcess(cmd)

	if err := cmd.Start(); err != nil {
		cancel()
		return nil, fmt.Errorf("start cloudflared: %w", err)
	}

	var exited atomic.Bool
	go func() {
		_ = cmd.Wait()
		exited.Store(true)
	}()

	if err := waitForTunnelReady(ctx, port, &exited, defaultReadyTimeout, &stderr); err != nil {
		_ = cmd.Process.Kill()
		cancel()
		if stderr.Len() > 0 {
			return nil, fmt.Errorf("%w: %s", err, trimStderr(stderr.String()))
		}
		return nil, err
	}

	return &Tunnel{
		LocalPort: port,
		Cmd:       cmd,
		Cancel:    cancel,
		exited:    &exited,
		stderr:    &stderr,
	}, nil
}

func (t *Tunnel) EnsureReady() error {
	if t == nil {
		return fmt.Errorf("tunnel is nil")
	}
	return waitForTunnelReady(context.Background(), t.LocalPort, t.exited, 5*time.Second, t.stderr)
}

func (t *Tunnel) Alive() bool {
	return t != nil && t.exited != nil && !t.exited.Load()
}

func (t *Tunnel) Close() error {
	if t == nil {
		return nil
	}
	if t.Cancel != nil {
		t.Cancel()
	}
	if t.Cmd != nil && t.Cmd.Process != nil {
		_ = t.Cmd.Process.Kill()
		_, _ = t.Cmd.Process.Wait()
	}
	return nil
}
