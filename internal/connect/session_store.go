package connect

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/mewisme/mewsh/internal/config"
	"github.com/mewisme/mewsh/internal/terminal"
)

const sessionIDEnv = "MEWSH_SESSION_ID"

// backgroundSessionGrace is how long finished background sessions stay in `mewsh sessions`.
const backgroundSessionGrace = 10 * time.Minute

type storedSession struct {
	ID        string    `json:"id"`
	Alias     string    `json:"alias"`
	Hostname  string    `json:"hostname,omitempty"`
	Target    string    `json:"target,omitempty"`
	WorkerPID int       `json:"worker_pid,omitempty"`
	SSHPID    int       `json:"ssh_pid,omitempty"`
	SSHArgv   []string  `json:"ssh_argv,omitempty"`
	LogPath   string    `json:"log_path,omitempty"`
	StartedAt time.Time `json:"started_at"`
}

type sessionRegistry struct {
	Sessions []storedSession `json:"sessions"`
}

var storeMu sync.Mutex

func newSessionID(alias string) string {
	return fmt.Sprintf("%s-%d", alias, time.Now().UnixNano())
}

func registryPath() (string, error) {
	dir, err := config.Dir()
	if err != nil {
		return "", err
	}
	sessionsDir := filepath.Join(dir, "sessions")
	if err := os.MkdirAll(sessionsDir, 0700); err != nil {
		return "", err
	}
	return filepath.Join(sessionsDir, "registry.json"), nil
}

func loadRegistry() (sessionRegistry, error) {
	path, err := registryPath()
	if err != nil {
		return sessionRegistry{}, err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return sessionRegistry{}, nil
		}
		return sessionRegistry{}, err
	}
	var reg sessionRegistry
	if err := json.Unmarshal(b, &reg); err != nil {
		return sessionRegistry{}, err
	}
	return reg, nil
}

func saveRegistry(reg sessionRegistry) error {
	path, err := registryPath()
	if err != nil {
		return err
	}
	b, err := json.MarshalIndent(reg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0600)
}

func sessionStoreUpsert(rec storedSession) error {
	storeMu.Lock()
	defer storeMu.Unlock()
	reg, err := loadRegistry()
	if err != nil {
		return err
	}
	for i, s := range reg.Sessions {
		if s.ID == rec.ID {
			if rec.WorkerPID == 0 {
				rec.WorkerPID = s.WorkerPID
			}
			if rec.LogPath == "" {
				rec.LogPath = s.LogPath
			}
			if rec.Hostname == "" {
				rec.Hostname = s.Hostname
			}
			reg.Sessions[i] = rec
			return saveRegistry(reg)
		}
	}
	reg.Sessions = append(reg.Sessions, rec)
	return saveRegistry(reg)
}

func sessionStoreRemove(id string) {
	storeMu.Lock()
	defer storeMu.Unlock()
	reg, err := loadRegistry()
	if err != nil {
		return
	}
	out := reg.Sessions[:0]
	for _, s := range reg.Sessions {
		if s.ID != id {
			out = append(out, s)
		}
	}
	reg.Sessions = out
	_ = saveRegistry(reg)
}

func sessionStoreGet(id string) (*storedSession, error) {
	storeMu.Lock()
	defer storeMu.Unlock()
	reg, err := loadRegistry()
	if err != nil {
		return nil, err
	}
	for i := range reg.Sessions {
		if reg.Sessions[i].ID == id {
			s := reg.Sessions[i]
			return &s, nil
		}
	}
	return nil, nil
}

func sessionStoreList() []storedSession {
	storeMu.Lock()
	defer storeMu.Unlock()
	reg, err := loadRegistry()
	if err != nil {
		return nil
	}
	return append([]storedSession(nil), reg.Sessions...)
}

func sessionStoreClear() {
	storeMu.Lock()
	defer storeMu.Unlock()
	_ = saveRegistry(sessionRegistry{})
}

func storedSessionRunning(s storedSession) bool {
	if s.WorkerPID > 0 && terminal.ProcessExists(s.WorkerPID) {
		return true
	}
	if s.SSHPID > 0 && terminal.ProcessExists(s.SSHPID) {
		return true
	}
	if len(s.SSHArgv) == 0 {
		return false
	}
	host, port := terminal.SSHProcessMarkers(s.SSHArgv)
	_, err := terminal.FindSSHProcessPID(host, port, nil)
	return err == nil
}

// shouldListStoredSession reports whether a registry entry should appear in mewsh sessions list.
func shouldListStoredSession(s storedSession) bool {
	if storedSessionRunning(s) {
		return true
	}
	grace := sessionSpawnGrace
	if s.WorkerPID > 0 || s.LogPath != "" {
		grace = backgroundSessionGrace
	}
	return time.Since(s.StartedAt) < grace
}

func sessionStorePrune() {
	storeMu.Lock()
	defer storeMu.Unlock()
	reg, err := loadRegistry()
	if err != nil {
		return
	}
	out := reg.Sessions[:0]
	for _, s := range reg.Sessions {
		if shouldListStoredSession(s) {
			out = append(out, s)
		}
	}
	reg.Sessions = out
	_ = saveRegistry(reg)
}

func persistFromSSHSession(s *sshSession) {
	if s == nil {
		return
	}
	rec := storedSession{
		ID:        s.id,
		Alias:     s.alias,
		Hostname:  s.hostname,
		Target:    s.target,
		SSHPID:    s.sshPID,
		SSHArgv:   append([]string(nil), s.sshArgv...),
		StartedAt: s.registeredAt,
	}
	if rec.StartedAt.IsZero() {
		rec.StartedAt = time.Now()
	}
	_ = sessionStoreUpsert(rec)
}

func persistBackgroundSupervisor(id, alias string, workerPID int, logPath string) error {
	return sessionStoreUpsert(storedSession{
		ID:        id,
		Alias:     alias,
		WorkerPID: workerPID,
		LogPath:   logPath,
		StartedAt: time.Now(),
	})
}

func infoFromStored(s storedSession) SessionInfo {
	target := s.Target
	if target == "" && len(s.SSHArgv) > 0 {
		host, port := terminal.SSHProcessMarkers(s.SSHArgv)
		target = host
		if port != "" {
			target = fmt.Sprintf("%s:%s", host, port)
		}
	}
	pid := s.SSHPID
	if pid == 0 {
		pid = s.WorkerPID
	}
	return SessionInfo{
		ID:       s.ID,
		Alias:    s.Alias,
		Hostname: s.Hostname,
		Target:   target,
		PID:      pid,
		LogPath:  s.LogPath,
	}
}

func enrichSessionInfo(info SessionInfo) SessionInfo {
	rec, err := sessionStoreGet(info.ID)
	if err != nil || rec == nil {
		return info
	}
	if info.LogPath == "" {
		info.LogPath = rec.LogPath
	}
	if info.PID == 0 && rec.WorkerPID > 0 {
		info.PID = rec.WorkerPID
	}
	return info
}

func killStoredSession(s storedSession) {
	if s.WorkerPID > 0 && terminal.ProcessExists(s.WorkerPID) {
		terminal.KillProcess(s.WorkerPID)
		return
	}
	if s.SSHPID > 0 && terminal.ProcessExists(s.SSHPID) {
		terminal.KillProcess(s.SSHPID)
		return
	}
	if len(s.SSHArgv) > 0 {
		terminal.KillSSH(s.SSHArgv, nil)
	}
}
