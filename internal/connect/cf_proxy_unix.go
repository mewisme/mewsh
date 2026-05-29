//go:build !windows

package connect

import (
	"fmt"

	"github.com/mewisme/mewsh/internal/cloudflared"
	"github.com/mewisme/mewsh/internal/config"
	"github.com/mewisme/mewsh/internal/profile"
)

func buildCloudflareProxyArgs(cfg *config.Config, p profile.Profile, o Options) ([]string, func(), error) {
	proxy, err := cloudflareProxyCommand(cfg, p.CFHostname)
	if err != nil {
		return nil, nil, err
	}
	argv, err := buildLaunchArgs(p, p.CFHostname, p.Port, o)
	if err != nil {
		return nil, nil, err
	}
	return insertSSHProxyCommand(argv, proxy), nil, nil
}

func cloudflareProxyCommand(cfg *config.Config, hostname string) (string, error) {
	cfPath, err := cloudflared.ResolvePathForConnect(cfg)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s access ssh --hostname %s", cfPath, hostname), nil
}
