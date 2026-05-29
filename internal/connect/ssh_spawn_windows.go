//go:build windows

package connect

import (
	"os"
	"path/filepath"

	"github.com/mewisme/mewsh/internal/profile"
	"github.com/mewisme/mewsh/internal/secret"
)

// spawnExtraEnv returns extra environment variables and an optional cleanup for SSH spawn.
func spawnExtraEnv(p profile.Profile) (extra []string, cleanup func(), err error) {
	if p.AuthType != profile.AuthPassword {
		return nil, nil, nil
	}
	if _, err := secret.GetPassword(p.PasswordRef); err != nil {
		return nil, nil, nil
	}

	exe, err := os.Executable()
	if err != nil {
		return nil, nil, err
	}
	exe, err = filepath.Abs(exe)
	if err != nil {
		return nil, nil, err
	}

	if err := WriteAskpassRef(p.PasswordRef); err != nil {
		return nil, nil, err
	}

	askpass, err := writeAskpassLauncher(exe)
	if err != nil {
		return nil, nil, err
	}

	extra = []string{
		"SSH_ASKPASS=" + askpass,
		"SSH_ASKPASS_REQUIRE=force",
		AskpassEnvRef + "=" + p.PasswordRef,
		"DISPLAY=localhost:0",
	}
	return extra, nil, nil
}
