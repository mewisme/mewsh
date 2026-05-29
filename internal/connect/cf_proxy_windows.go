//go:build windows

package connect

import (
	"github.com/mewisme/mewsh/internal/config"
	"github.com/mewisme/mewsh/internal/profile"
)

func buildCloudflareProxyArgs(cfg *config.Config, p profile.Profile) ([]string, func(), error) {
	return buildCloudflareSSHConfigArgs(cfg, p)
}
