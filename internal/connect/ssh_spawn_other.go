//go:build !windows

package connect

import "github.com/mewisme/mewsh/internal/profile"

func spawnExtraEnv(p profile.Profile) (extra []string, cleanup func(), err error) {
	return nil, nil, nil
}
