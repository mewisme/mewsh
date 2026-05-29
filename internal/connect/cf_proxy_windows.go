//go:build windows

package connect

import (
	"github.com/mewisme/mewsh/internal/config"
	"github.com/mewisme/mewsh/internal/profile"
)

func buildCloudflareProxyArgs(cfg *config.Config, p profile.Profile, o Options) ([]string, func(), error) {
	return buildCloudflareSSHConfigArgs(cfg, p, o.background)
}
