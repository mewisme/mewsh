package connect

import (
	"fmt"

	"github.com/mewisme/mewsh/internal/profile"
)

// sessionDisplayTarget formats the user-visible ssh destination for session listings.
func sessionDisplayTarget(p profile.Profile, host string, port int) string {
	p.ApplyDefaults()
	if host == "" {
		if p.ConnectionType == profile.ConnectionCloudflareAccess {
			return fmt.Sprintf("%s@%s", p.User, p.CFHostname)
		}
		host = p.Host
	}
	if port <= 0 {
		port = p.Port
	}
	if port == 22 {
		return fmt.Sprintf("%s@%s", p.User, host)
	}
	return fmt.Sprintf("%s@%s:%d", p.User, host, port)
}
