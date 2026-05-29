package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/mewisme/mewsh/internal/profile"
)

type Config struct {
	Profiles        []profile.Profile `json:"profiles"`
	CloudflaredPath string            `json:"cloudflared_path,omitempty"`
}

var configOverride string

func SetPathOverride(path string) {
	configOverride = path
}

func Dir() (string, error) {
	if configOverride != "" {
		return filepath.Dir(configOverride), nil
	}
	if runtime.GOOS == "windows" {
		appData := os.Getenv("APPDATA")
		if appData == "" {
			return "", fmt.Errorf("APPDATA is not set")
		}
		return filepath.Join(appData, "mewsh"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "mewsh"), nil
}

func Path() (string, error) {
	if configOverride != "" {
		return configOverride, nil
	}
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

func BinDir() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "bin"), nil
}

func BundledCloudflaredPath() (string, error) {
	binDir, err := BinDir()
	if err != nil {
		return "", err
	}
	name := "cloudflared"
	if runtime.GOOS == "windows" {
		name = "cloudflared.exe"
	}
	return filepath.Join(binDir, name), nil
}

func EnsureDir() error {
	dir, err := Dir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	binDir, err := BinDir()
	if err != nil {
		return err
	}
	return os.MkdirAll(binDir, 0700)
}

func Load() (*Config, error) {
	path, err := Path()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{Profiles: []profile.Profile{}}, nil
		}
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if cfg.Profiles == nil {
		cfg.Profiles = []profile.Profile{}
	}
	return &cfg, nil
}

func Save(cfg *Config) error {
	if err := EnsureDir(); err != nil {
		return err
	}
	path, err := Path()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, configFileMode()); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func configFileMode() os.FileMode {
	if runtime.GOOS == "windows" {
		return 0600
	}
	return 0600
}

func (c *Config) FindByAlias(alias string) (*profile.Profile, int) {
	for i, p := range c.Profiles {
		if p.Alias == alias {
			cp := p
			return &cp, i
		}
	}
	return nil, -1
}

func (c *Config) AliasExists(alias string) bool {
	_, i := c.FindByAlias(alias)
	return i >= 0
}

type PermissionIssue struct {
	Message string
}

func CheckPermissions() []PermissionIssue {
	if runtime.GOOS == "windows" {
		return nil
	}
	path, err := Path()
	if err != nil {
		return []PermissionIssue{{Message: err.Error()}}
	}
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return []PermissionIssue{{Message: err.Error()}}
	}
	mode := info.Mode().Perm()
	if mode&0077 != 0 {
		return []PermissionIssue{{
			Message: fmt.Sprintf("config file %s has permissions %o; recommended 0600", path, mode),
		}}
	}
	return nil
}

func FormatPathHint() string {
	path, err := Path()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(path)
}
