package version

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
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

func String() string {
	resolve()
	v := Version
	if Commit != "" {
		v += " (" + Commit + ")"
	}
	if Date != "" {
		v += " built " + Date
	}
	return fmt.Sprintf("mewsh %s\n%s/%s", v, runtime.GOOS, runtime.GOARCH)
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
