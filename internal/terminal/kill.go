package terminal

// KillProcess terminates a process by PID (best-effort).
func KillProcess(pid int) {
	if pid <= 0 {
		return
	}
	killProcessPlatform(pid)
}

// FindSSHProcessPID returns one ssh PID for host/port, skipping exclude PIDs.
func FindSSHProcessPID(host, port string, exclude []int) (int, error) {
	pids, err := listSSHProcessPIDs(host, port)
	if err != nil {
		return 0, err
	}
	skip := pidSet(exclude)
	for _, pid := range pids {
		if !skip[pid] {
			return pid, nil
		}
	}
	return 0, errSSHNotFound()
}

// KillSSH stops one ssh session matching argv, skipping PIDs used by other sessions.
func KillSSH(argv []string, exclude []int) {
	host, port := sshProcessMarkers(argv)
	if pid, err := FindSSHProcessPID(host, port, exclude); err == nil && pid > 0 {
		KillProcess(pid)
		return
	}
	killSSHByPattern(host, port, exclude)
}

func pidSet(pids []int) map[int]bool {
	m := make(map[int]bool, len(pids))
	for _, pid := range pids {
		if pid > 0 {
			m[pid] = true
		}
	}
	return m
}
