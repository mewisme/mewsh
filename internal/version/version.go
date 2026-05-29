package version

import (
	"fmt"
	"runtime"
	"strings"
)

// Set by GoReleaser ldflags; defaults for dev builds.
var (
	Version = "dev"
	Commit  = ""
	Date    = ""
)

func String() string {
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
	if Version == "" || Version == "dev" {
		return ""
	}
	if strings.HasPrefix(Version, "v") {
		return Version
	}
	return "v" + Version
}
