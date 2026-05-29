package connect

import (
	"github.com/mewisme/mewsh/internal/profile"
	"github.com/mewisme/mewsh/internal/terminal"
)

func detachedSpawn(p profile.Profile, argv []string) error {
	extraEnv, cleanup, err := spawnExtraEnv(p)
	if err != nil {
		return err
	}
	if cleanup != nil {
		defer cleanup()
	}

	return terminal.SpawnDetachedEnv(argv, extraEnv)
}
