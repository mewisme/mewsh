package terminal

import "fmt"

// SpawnBackgroundEnv starts argv without a terminal emulator. Output is appended to
// logPath. The process is placed in a new session so it can outlive the invoking shell.
func SpawnBackgroundEnv(argv []string, extraEnv []string, logPath string) (pid int, err error) {
	if len(argv) == 0 {
		return 0, fmt.Errorf("empty command")
	}
	if err := ensureSSH(argv[0]); err != nil {
		return 0, err
	}
	return spawnBackgroundPlatform(argv, extraEnv, logPath)
}
