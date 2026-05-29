package version

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"time"
)

// Set by GoReleaser ldflags; defaults for plain go build.
var (
	Version = "dev"
	Commit  = ""
	Date    = ""
)

var resolved sync.Once

func resolve() {
	resolved.Do(func() {
		bi, ok := debug.ReadBuildInfo()
		if !ok {
			return
		}
		if Version == "" || Version == "dev" {
			if v := bi.Main.Version; v != "" && v != "(devel)" {
				Version = strings.TrimPrefix(v, "v")
			} else {
				for _, s := range bi.Settings {
					if s.Key == "vcs.tag" && s.Value != "" {
						Version = strings.TrimPrefix(s.Value, "v")
						break
					}
				}
			}
		}
		if Commit == "" {
			for _, s := range bi.Settings {
				if s.Key == "vcs.revision" && s.Value != "" {
					rev := s.Value
					if len(rev) > 7 {
						rev = rev[:7]
					}
					Commit = rev
				}
			}
		}
		if Date == "" {
			for _, s := range bi.Settings {
				if s.Key == "vcs.time" {
					Date = s.Value
				}
			}
		}
	})
}

// Build describes the running binary (from ldflags and build metadata).
type Build struct {
	Version string
	Commit  string
	Date    string
	GOOS    string
	GOARCH  string
	Dev     bool
}

// BuildInfo returns resolved version metadata for CLI display.
func BuildInfo() Build {
	resolve()
	dev := Version == "" || Version == "dev"
	ver := Version
	if ver == "" {
		ver = "dev"
	}
	if !dev && !strings.HasPrefix(ver, "v") {
		ver = "v" + ver
	}
	return Build{
		Version: ver,
		Commit:  Commit,
		Date:    FormatBuildDate(Date),
		GOOS:    runtime.GOOS,
		GOARCH:  runtime.GOARCH,
		Dev:     dev,
	}
}

// FormatBuildDate formats vcs.time for display; returns s unchanged if not parseable.
func FormatBuildDate(s string) string {
	if s == "" {
		return ""
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.Local().Format("2006-01-02 15:04 MST")
	}
	return s
}

func String() string {
	b := BuildInfo()
	v := b.Version
	if b.Commit != "" {
		v += " (" + b.Commit + ")"
	}
	if b.Date != "" {
		v += " built " + b.Date
	}
	return fmt.Sprintf("mewsh %s\n%s/%s", v, b.GOOS, b.GOARCH)
}

func Tag() string {
	resolve()
	if Version == "" || Version == "dev" {
		return ""
	}
	if strings.HasPrefix(Version, "v") {
		return Version
	}
	return "v" + Version
}
