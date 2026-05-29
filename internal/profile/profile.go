package profile

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
)

const (
	AuthAgent    = "agent"
	AuthKey      = "key"
	AuthPassword = "password"

	PasswordManual = "manual"
	PasswordAuto   = "auto"

	ConnectionDirect           = "direct"
	ConnectionCloudflareAccess = "cloudflare_access"
)

type Profile struct {
	Alias          string `json:"alias"`
	Host           string `json:"host"`
	Port           int    `json:"port"`
	User           string `json:"user"`
	AuthType       string `json:"auth_type"`
	KeyPath        string `json:"key_path,omitempty"`
	PasswordMode   string `json:"password_mode,omitempty"`
	PasswordRef    string `json:"password_ref,omitempty"`
	Note           string `json:"note,omitempty"`
	ConnectionType string `json:"connection_type"`
	CFHostname     string `json:"cf_hostname,omitempty"`
}

func Default() Profile {
	return Profile{
		Port:           22,
		ConnectionType: ConnectionDirect,
		PasswordMode:   PasswordManual,
	}
}

func (p *Profile) ApplyDefaults() {
	if p.Port == 0 {
		p.Port = 22
	}
	if p.ConnectionType == "" {
		p.ConnectionType = ConnectionDirect
	}
	if p.PasswordMode == "" {
		p.PasswordMode = PasswordManual
	}
	if p.PasswordRef == "" && p.Alias != "" {
		p.PasswordRef = p.Alias
	}
}

func (p Profile) Validate(existingAliases []string, editing bool) error {
	p.ApplyDefaults()

	if p.Alias == "" {
		return fmt.Errorf("alias is required")
	}
	count := 0
	for _, a := range existingAliases {
		if a == p.Alias {
			count++
		}
	}
	if !editing && count > 0 {
		return fmt.Errorf("alias %q already exists", p.Alias)
	}
	if editing && count > 1 {
		return fmt.Errorf("alias %q already exists", p.Alias)
	}

	if p.ConnectionType == ConnectionDirect && p.Host == "" {
		return fmt.Errorf("host is required for direct connections")
	}
	if p.ConnectionType == ConnectionCloudflareAccess && p.CFHostname == "" {
		return fmt.Errorf("cf_hostname is required for cloudflare_access connections")
	}
	if p.Port < 1 || p.Port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}
	if p.User == "" {
		return fmt.Errorf("user is required")
	}

	switch p.AuthType {
	case AuthAgent, AuthKey, AuthPassword:
	default:
		return fmt.Errorf("auth_type must be agent, key, or password")
	}

	if p.AuthType == AuthKey {
		if p.KeyPath == "" {
			return fmt.Errorf("key_path is required when auth_type is key")
		}
		if _, err := os.Stat(p.KeyPath); err != nil {
			return fmt.Errorf("key_path: %w", err)
		}
	}

	if p.AuthType == AuthPassword {
		switch p.PasswordMode {
		case PasswordManual, PasswordAuto:
		default:
			return fmt.Errorf("password_mode must be manual or auto")
		}
	}

	switch p.ConnectionType {
	case ConnectionDirect, ConnectionCloudflareAccess:
	default:
		return fmt.Errorf("connection_type must be direct or cloudflare_access")
	}

	return nil
}

func (p Profile) BuildSSHArgs(host string, port int, background bool) []string {
	target := fmt.Sprintf("%s@%s", p.User, host)
	args := []string{"ssh"}

	if background {
		// Never allocate a local TTY for background workers (avoids corrupting the invoking terminal).
		args = append(args,
			"-n", "-T",
			"-o", "ServerAliveInterval=30",
			"-o", "ServerAliveCountMax=10",
		)
	}

	if runtime.GOOS == "windows" {
		args = append(args,
			"-o", "GSSAPIAuthentication=no",
			"-o", "ConnectTimeout=30",
			"-o", "StrictHostKeyChecking=accept-new",
		)
		if !background {
			if p.AuthType == AuthPassword {
				// No -tt: password comes from SSH_ASKPASS; RequestTTY keeps the remote shell interactive.
				args = append(args,
					"-o", "RequestTTY=force",
					"-o", "PreferredAuthentications=keyboard-interactive,password",
					"-o", "PubkeyAuthentication=no",
					"-o", "NumberOfPasswordPrompts=3",
				)
			} else {
				args = append(args, "-tt")
			}
		} else if p.AuthType == AuthPassword {
			args = append(args,
				"-o", "PreferredAuthentications=keyboard-interactive,password",
				"-o", "PubkeyAuthentication=no",
				"-o", "NumberOfPasswordPrompts=3",
			)
		}
	}

	args = append(args, "-p", strconv.Itoa(port))
	if p.AuthType == AuthKey && p.KeyPath != "" {
		args = append(args, "-i", p.KeyPath)
	}
	args = append(args, target)
	if background {
		// Without a local TTY, a default remote shell often exits immediately; hold the session open.
		args = append(args, "sleep", "infinity")
	}
	return args
}

func (p Profile) Summary() string {
	if p.ConnectionType == ConnectionCloudflareAccess {
		return fmt.Sprintf("%s@%s (cf:%s)", p.User, p.CFHostname, p.AuthType)
	}
	return fmt.Sprintf("%s@%s:%d (%s)", p.User, p.Host, p.Port, p.AuthType)
}

func (p Profile) DisplayHost() string {
	if p.ConnectionType == ConnectionCloudflareAccess {
		return p.CFHostname
	}
	return p.Host
}
