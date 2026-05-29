//go:build windows

package terminal

import (
	"os"
	"os/exec"
	"syscall"
)

func spawnBackgroundPlatform(argv []string, extraEnv []string, logPath string) (int, error) {
	log, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return 0, err
	}

	devNull, err := os.Open(os.DevNull)
	if err != nil {
		_ = log.Close()
		return 0, err
	}

	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Env = append(os.Environ(), extraEnv...)
	cmd.Stdin = devNull
	cmd.Stdout = log
	cmd.Stderr = log
	attr := &syscall.SysProcAttr{}
	configureBackgroundCmd(attr)
	cmd.SysProcAttr = attr

	if err := cmd.Start(); err != nil {
		_ = log.Close()
		return 0, err
	}
	_ = log.Close()
	return cmd.Process.Pid, nil
}
