//go:build windows

package connect

import (
	"github.com/mewisme/mewsh/internal/config"
	"github.com/mewisme/mewsh/internal/profile"
)

func previewInteractiveCloudflareSSH(cfg *config.Config, p profile.Profile, _ Options) ([]string, error) {
	argv, _, err := buildCloudflareSSHConfigArgs(cfg, p, false)
	return argv, err
}
