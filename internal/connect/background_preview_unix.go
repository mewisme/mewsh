//go:build !windows

package connect

import (
	"github.com/mewisme/mewsh/internal/config"
	"github.com/mewisme/mewsh/internal/profile"
)

func previewInteractiveCloudflareSSH(cfg *config.Config, p profile.Profile, o Options) ([]string, error) {
	_ = cfg
	return buildLaunchArgs(p, p.CFHostname, p.Port, o)
}
