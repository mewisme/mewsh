package connect

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/mewisme/mewsh/internal/config"
	"github.com/mewisme/mewsh/internal/terminal"
	"github.com/mewisme/mewsh/internal/tunnel"
)

// SessionInfo describes a tracked SSH session for the TUI sessions list.
type SessionInfo struct {
	ID       string
	Alias    string
	Hostname string
	Target   string
	PID      int
	LogPath  string `json:"log_path,omitempty"`
}

type sharedTunnel struct {
	hostname string
	tun      *tunnel.Tunnel
	cancel   context.CancelFunc
	refCount int
}

type sshSession struct {
	id           string
	alias        string
	hostname     string
	target       string
	sshPID       int
	sshArgv      []string
	registeredAt time.Time
}

const sessionSpawnGrace = 45 * time.Second

var (
	tunnelsMu     sync.Mutex
	sharedTunnels = map[string]*sharedTunnel{}

	sshMu       sync.Mutex
	sshSessions = map[string]*sshSession{}
)

func acquireSharedTunnel(cfg *config.Config, hostname string, o Options) (*tunnel.Tunnel, context.CancelFunc, error) {
	tunnelsMu.Lock()
	if st, ok := sharedTunnels[hostname]; ok && st.tun != nil && st.tun.Alive() {
		st.refCount++
		tunnelsMu.Unlock()
		o.say(fmt.Sprintf("Reusing Cloudflare tunnel on 127.0.0.1:%d\n", st.tun.LocalPort))
		return st.tun, st.cancel, nil
	}
	tunnelsMu.Unlock()

	tun, cancel, err := startCloudflareTunnel(cfg, hostname, o)
	if err != nil {
		return nil, nil, err
	}

	tunnelsMu.Lock()
	if st, ok := sharedTunnels[hostname]; ok && st.tun != nil && st.tun.Alive() {
		st.refCount++
		tunnelsMu.Unlock()
		_ = tun.Close()
		cancel()
		o.say(fmt.Sprintf("Reusing Cloudflare tunnel on 127.0.0.1:%d\n", st.tun.LocalPort))
		return st.tun, st.cancel, nil
	}
	sharedTunnels[hostname] = &sharedTunnel{
		hostname: hostname,
		tun:      tun,
		cancel:   cancel,
		refCount: 1,
	}
	tunnelsMu.Unlock()
	return tun, cancel, nil
}

func releaseSharedTunnel(hostname string) {
	if hostname == "" {
		return
	}
	tunnelsMu.Lock()
	st, ok := sharedTunnels[hostname]
	if !ok {
		tunnelsMu.Unlock()
		return
	}
	st.refCount--
	if st.refCount > 0 {
		tunnelsMu.Unlock()
		return
	}
	delete(sharedTunnels, hostname)
	tun := st.tun
	cancel := st.cancel
	port := 0
	if tun != nil {
		port = tun.LocalPort
	}
	tunnelsMu.Unlock()

	if port > 0 {
		waitTunnelPortIdle(port)
	}
	if cancel != nil {
		cancel()
	}
	if tun != nil {
		_ = tun.Close()
	}
}

func registerSSHSession(alias, hostname string, argv []string, displayTarget string) string {
	id := os.Getenv(sessionIDEnv)
	if id == "" {
		id = newSessionID(alias)
	}
	if displayTarget == "" {
		host, port := terminal.SSHProcessMarkers(argv)
		displayTarget = host
		if port != "" {
			displayTarget = fmt.Sprintf("%s:%s", host, port)
		}
	}
	argvCopy := append([]string(nil), argv...)
	s := &sshSession{
		id:           id,
		alias:        alias,
		hostname:     hostname,
		target:       displayTarget,
		sshArgv:      argvCopy,
		registeredAt: time.Now(),
	}
	sshMu.Lock()
	sshSessions[id] = s
	sshMu.Unlock()
	persistFromSSHSession(s)
	return id
}

func setSSHSessionPID(id string, pid int) {
	sshMu.Lock()
	var s *sshSession
	if cur, ok := sshSessions[id]; ok && pid > 0 {
		cur.sshPID = pid
		s = cur
	}
	sshMu.Unlock()
	if s != nil {
		persistFromSSHSession(s)
	}
}

func removeSSHSession(id string, kill bool) {
	sshMu.Lock()
	s := sshSessions[id]
	delete(sshSessions, id)
	sshMu.Unlock()
	if s == nil {
		return
	}
	if kill {
		killSSHSession(s)
		waitSessionSSHExit(s)
	}
	releaseSharedTunnel(s.hostname)
	// Natural exit: keep the registry row until sessionStorePrune (spawn grace) so
	// `mewsh sessions` still lists it briefly; explicit kill removes immediately.
	if kill {
		sessionStoreRemove(id)
	}
}

func waitSessionSSHExit(s *sshSession) {
	if s == nil {
		return
	}
	if s.sshPID > 0 {
		terminal.WaitSSHProcessExit(s.sshPID)
		return
	}
	if len(s.sshArgv) == 0 {
		return
	}
	host, port := terminal.SSHProcessMarkers(s.sshArgv)
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := terminal.FindSSHProcessPID(host, port, otherSessionPIDs(s.id)); err != nil {
			return
		}
		time.Sleep(250 * time.Millisecond)
	}
}

func waitTunnelPortIdle(port int) {
	if port <= 0 {
		return
	}
	portStr := strconv.Itoa(port)
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := terminal.FindSSHProcessPID("127.0.0.1", portStr, otherSessionPIDs("")); err != nil {
			return
		}
		time.Sleep(250 * time.Millisecond)
	}
}

func killSSHSession(s *sshSession) {
	if s == nil {
		return
	}
	if s.sshPID > 0 {
		terminal.KillProcess(s.sshPID)
		return
	}
	if len(s.sshArgv) > 0 {
		terminal.KillSSH(s.sshArgv, otherSessionPIDs(s.id))
	}
}

// CleanupActive stops every tracked SSH session and shared tunnel (e.g. on TUI quit).
func CleanupActive() {
	sshMu.Lock()
	sessions := make([]*sshSession, 0, len(sshSessions))
	for _, s := range sshSessions {
		sessions = append(sessions, s)
	}
	sshSessions = map[string]*sshSession{}
	sshMu.Unlock()

	for _, s := range sessions {
		killSSHSession(s)
	}
	for _, s := range sessions {
		waitSessionSSHExit(s)
	}

	tunnelsMu.Lock()
	tunnels := make([]*sharedTunnel, 0, len(sharedTunnels))
	for _, st := range sharedTunnels {
		tunnels = append(tunnels, st)
	}
	sharedTunnels = map[string]*sharedTunnel{}
	tunnelsMu.Unlock()

	for _, st := range tunnels {
		port := 0
		if st.tun != nil {
			port = st.tun.LocalPort
		}
		if port > 0 {
			waitTunnelPortIdle(port)
		}
		if st.cancel != nil {
			st.cancel()
		}
		if st.tun != nil {
			_ = st.tun.Close()
		}
	}
	for _, s := range sessionStoreList() {
		killStoredSession(s)
	}
	sessionStoreClear()
}

// CleanupAlias kills all SSH sessions for a profile alias (tunnel stays if other aliases share it).
func CleanupAlias(alias string) {
	sshMu.Lock()
	var ids []string
	for id, s := range sshSessions {
		if s.alias == alias {
			ids = append(ids, id)
		}
	}
	sshMu.Unlock()
	for _, id := range ids {
		removeSSHSession(id, true)
	}
}

func monitorDetachedSSH(sessionID string, argv []string) {
	resolveSessionSSHPID(sessionID, argv)
	if pid := sessionPID(sessionID); pid > 0 {
		terminal.WaitSSHProcessExit(pid)
	} else {
		terminal.WaitSSHSessionEnd(argv)
	}
	removeSSHSession(sessionID, false)
}

func resolveSessionSSHPID(sessionID string, argv []string) {
	host, port := terminal.SSHProcessMarkers(argv)
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		exclude := otherSessionPIDs(sessionID)
		pid, err := terminal.FindSSHProcessPID(host, port, exclude)
		if err == nil && pid > 0 && claimSessionPID(sessionID, pid) {
			return
		}
		time.Sleep(250 * time.Millisecond)
	}
}

func sessionPID(sessionID string) int {
	sshMu.Lock()
	defer sshMu.Unlock()
	if s, ok := sshSessions[sessionID]; ok {
		return s.sshPID
	}
	return 0
}

func claimSessionPID(sessionID string, pid int) bool {
	sshMu.Lock()
	defer sshMu.Unlock()
	for id, s := range sshSessions {
		if id != sessionID && s.sshPID == pid {
			return false
		}
	}
	s, ok := sshSessions[sessionID]
	if !ok {
		return false
	}
	s.sshPID = pid
	return true
}

func otherSessionPIDs(skipSessionID string) []int {
	sshMu.Lock()
	defer sshMu.Unlock()
	var out []int
	for id, s := range sshSessions {
		if id != skipSessionID && s.sshPID > 0 {
			out = append(out, s.sshPID)
		}
	}
	return out
}

func sessionSSHRunning(s *sshSession) bool {
	if s.sshPID > 0 {
		return terminal.ProcessExists(s.sshPID)
	}
	if time.Since(s.registeredAt) < sessionSpawnGrace {
		host, port := terminal.SSHProcessMarkers(s.sshArgv)
		_, err := terminal.FindSSHProcessPID(host, port, otherSessionPIDs(s.id))
		return err == nil
	}
	return false
}

// pruneStaleSessions removes tracked sessions whose SSH process is no longer running.
func pruneStaleSessions() {
	type dead struct {
		id       string
		hostname string
	}
	var removed []dead
	sshMu.Lock()
	for id, s := range sshSessions {
		if !sessionSSHRunning(s) {
			removed = append(removed, dead{id: id, hostname: s.hostname})
			delete(sshSessions, id)
		}
	}
	sshMu.Unlock()
	for _, d := range removed {
		releaseSharedTunnel(d.hostname)
	}
}

// PruneStaleSessions runs session cleanup in the background (safe for the TUI thread).
func PruneStaleSessions() {
	go pruneStaleSessions()
}

// ActiveSessionCount returns tracked SSH sessions (one per spawn). Non-blocking for the TUI.
func ActiveSessionCount() int {
	sshMu.Lock()
	defer sshMu.Unlock()
	return len(sshSessions)
}

// ListSessions returns active SSH sessions managed by mewsh (in-process and on disk).
func ListSessions() []SessionInfo {
	pruneStaleSessions()
	sessionStorePrune()

	byID := map[string]SessionInfo{}

	sshMu.Lock()
	for _, s := range sshSessions {
		byID[s.id] = enrichSessionInfo(sessionInfoFrom(s))
	}
	sshMu.Unlock()

	for _, s := range sessionStoreList() {
		if _, ok := byID[s.ID]; ok {
			continue
		}
		if shouldListStoredSession(s) {
			byID[s.ID] = infoFromStored(s)
		}
	}

	out := make([]SessionInfo, 0, len(byID))
	for _, info := range byID {
		out = append(out, info)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Alias != out[j].Alias {
			return out[i].Alias < out[j].Alias
		}
		return out[i].ID < out[j].ID
	})
	return out
}

func sessionInfoFrom(s *sshSession) SessionInfo {
	target := s.target
	if target == "" {
		host, port := terminal.SSHProcessMarkers(s.sshArgv)
		target = host
		if port != "" {
			target = fmt.Sprintf("%s:%s", host, port)
		}
	}
	return SessionInfo{
		ID:       s.id,
		Alias:    s.alias,
		Hostname: s.hostname,
		Target:   target,
		PID:      s.sshPID,
	}
}

// ActiveSessionsByAlias counts tracked SSH sessions per profile alias (one per spawn).
func ActiveSessionsByAlias() map[string]int {
	sshMu.Lock()
	defer sshMu.Unlock()
	counts := make(map[string]int)
	for _, s := range sshSessions {
		counts[s.alias]++
	}
	return counts
}

// KillSession stops one session by id (in-process or persisted).
func KillSession(id string) error {
	sshMu.Lock()
	_, inMem := sshSessions[id]
	sshMu.Unlock()
	if inMem {
		removeSSHSession(id, true)
		return nil
	}
	rec, err := sessionStoreGet(id)
	if err != nil {
		return err
	}
	if rec == nil {
		return fmt.Errorf("session %q not found", id)
	}
	killStoredSession(*rec)
	sessionStoreRemove(id)
	return nil
}

// KillSessions stops multiple sessions by id.
func KillSessions(ids []string) error {
	var first error
	for _, id := range ids {
		if err := KillSession(id); err != nil && first == nil {
			first = err
		}
	}
	return first
}

// KillSessionsByAlias stops every tracked session for a profile alias.
func KillSessionsByAlias(alias string) error {
	var ids []string
	for _, s := range ListSessions() {
		if s.Alias == alias {
			ids = append(ids, s.ID)
		}
	}
	if len(ids) == 0 {
		return fmt.Errorf("no active sessions for profile %q", alias)
	}
	return KillSessions(ids)
}

// KillAllSessions stops every tracked SSH session and shared tunnels.
func KillAllSessions() {
	CleanupActive()
}
