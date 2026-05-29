package selfupdate

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"strings"
)

// InstallMethod describes how mewsh was installed.
type InstallMethod int

const (
	InstallUnknown InstallMethod = iota
	InstallBinary
	InstallHomebrew
	InstallGo
)

func (m InstallMethod) String() string {
	switch m {
	case InstallHomebrew:
		return "Homebrew"
	case InstallGo:
		return "go install"
	case InstallBinary:
		return "binary"
	default:
		return "unknown"
	}
}

// InstallInfo describes the running binary and how it was installed.
type InstallInfo struct {
	Method InstallMethod
	Exe    string
}

// DetectInstall inspects the executable path and build metadata.
func DetectInstall() (InstallInfo, error) {
	exe, err := os.Executable()
	if err != nil {
		return InstallInfo{}, err
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return InstallInfo{}, err
	}

	info := InstallInfo{Exe: exe, Method: InstallBinary}

	if method, ok := detectHomebrew(exe); ok {
		info.Method = method
		return info, nil
	}
	if detectGoInstall() {
		info.Method = InstallGo
		return info, nil
	}
	return info, nil
}

func detectHomebrew(exe string) (InstallMethod, bool) {
	lower := strings.ToLower(filepath.ToSlash(exe))
	if strings.Contains(lower, "/cellar/mewsh/") {
		return InstallHomebrew, true
	}
	brew, err := exec.LookPath("brew")
	if err != nil {
		return InstallUnknown, false
	}
	out, err := exec.Command(brew, "--prefix", "mewsh").Output()
	if err != nil {
		return InstallUnknown, false
	}
	prefix := strings.TrimSpace(string(out))
	if prefix == "" {
		return InstallUnknown, false
	}
	rel, err := filepath.Rel(prefix, exe)
	if err != nil {
		return InstallUnknown, false
	}
	if rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return InstallHomebrew, true
	}
	return InstallUnknown, false
}

func detectGoInstall() bool {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return false
	}
	return bi.Main.Path == "github.com/mewisme/mewsh" || strings.HasPrefix(bi.Main.Path, "github.com/mewisme/mewsh/")
}

func UpdateCommand(method InstallMethod) string {
	switch method {
	case InstallHomebrew:
		return "brew upgrade mewsh"
	case InstallGo:
		return "go install github.com/mewisme/mewsh@latest"
	default:
		return ""
	}
}
