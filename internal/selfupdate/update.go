package selfupdate

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/mewisme/mewsh/internal/cliui"
	"github.com/mewisme/mewsh/internal/version"
	"golang.org/x/mod/semver"
)

// CheckResult describes whether a newer release exists.
type CheckResult struct {
	Current string
	Latest  string
	Newer   bool
	Info    InstallInfo
}

// Check returns the latest GitHub release tag and whether it is newer than the running build.
func Check() (CheckResult, error) {
	info, err := DetectInstall()
	if err != nil {
		return CheckResult{}, err
	}
	rel, err := fetchLatestRelease()
	if err != nil {
		return CheckResult{}, err
	}
	cur := version.Tag()
	newer := isNewer(rel.TagName, cur)
	return CheckResult{
		Current: displayVersion(cur),
		Latest:  rel.TagName,
		Newer:   newer,
		Info:    info,
	}, nil
}

func displayVersion(tag string) string {
	if tag == "" {
		return version.Version + " (dev)"
	}
	return tag
}

func isNewer(latest, current string) bool {
	if current == "" || current == "dev" || !strings.HasPrefix(current, "v") {
		return true
	}
	if !strings.HasPrefix(latest, "v") {
		latest = "v" + latest
	}
	return semver.Compare(latest, current) > 0
}

// Run updates mewsh using the best method for how it was installed.
func Run(checkOnly, force bool) error {
	result, err := Check()
	if err != nil {
		return err
	}
	w := os.Stdout

	if !result.Newer && !force {
		cliui.OKf(w, "mewsh is up to date (%s).", result.Current)
		return nil
	}

	if result.Newer {
		cliui.Infof(w, "Update available: %s → %s (installed via %s).", result.Current, result.Latest, result.Info.Method.String())
	} else if force {
		cliui.Infof(w, "Reinstalling %s (installed via %s).", result.Current, result.Info.Method.String())
	}

	if checkOnly {
		return printUpdateHint(result.Info)
	}

	switch result.Info.Method {
	case InstallHomebrew:
		return runBrewUpgrade()
	case InstallGo:
		return runGoInstall(result.Latest, force)
	default:
		return updateBinary(result.Latest, result.Info.Exe)
	}
}

func printUpdateHint(info InstallInfo) error {
	w := os.Stdout
	switch info.Method {
	case InstallHomebrew, InstallGo:
		cliui.Labeled(w, "run", UpdateCommand(info.Method))
	default:
		cliui.Labeled(w, "run", "mewsh update")
	}
	return nil
}

func runBrewUpgrade() error {
	brew, err := exec.LookPath("brew")
	if err != nil {
		return fmt.Errorf("brew not found in PATH")
	}
	w := os.Stdout
	cliui.Info("Running: brew upgrade mewsh")
	cmd := exec.Command(brew, "upgrade", "mewsh")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("brew upgrade: %w", err)
	}
	cliui.OKf(w, "Updated via Homebrew.")
	return nil
}

func goInstallModuleRef(releaseTag string) string {
	if releaseTag == "" {
		return "github.com/mewisme/mewsh@latest"
	}
	if !strings.HasPrefix(releaseTag, "v") {
		releaseTag = "v" + releaseTag
	}
	return "github.com/mewisme/mewsh@" + releaseTag
}

func runGoInstall(releaseTag string, force bool) error {
	goBin, err := exec.LookPath("go")
	if err != nil {
		return fmt.Errorf("go not found in PATH")
	}
	moduleRef := goInstallModuleRef(releaseTag)
	args := []string{"install"}
	if force {
		args = append(args, "-a")
	}
	args = append(args, moduleRef)

	cmd := exec.Command(goBin, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	if force {
		cmd.Env = append(cmd.Env, "GOPROXY=direct")
	}

	w := os.Stdout
	cliui.Block(w, cliui.LevelCmd, goBin+" "+strings.Join(args, " "))
	if force {
		cliui.Dim("Using GOPROXY=direct and -a to avoid cached module/build artifacts.")
	}
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go install: %w", err)
	}
	cliui.OKf(w, "Updated via go install. Ensure your GOBIN is on PATH.")
	return nil
}

func updateBinary(tag, dest string) error {
	rel, err := fetchLatestRelease()
	if err != nil {
		return err
	}
	url, err := findAssetURL(rel)
	if err != nil {
		return err
	}

	w := os.Stdout
	cliui.Infof(w, "Downloading %s...", filepath.Base(url))
	data, err := download(url)
	if err != nil {
		return err
	}

	bin, err := extractBinary(data, url)
	if err != nil {
		return err
	}

	if err := replaceExecutable(bin, dest); err != nil {
		return err
	}
	cliui.OKf(w, "Updated %s to %s.", dest, tag)
	return nil
}

func download(url string) ([]byte, error) {
	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download: %s", resp.Status)
	}
	return io.ReadAll(resp.Body)
}

func extractBinary(data []byte, url string) ([]byte, error) {
	if strings.HasSuffix(strings.ToLower(url), ".zip") {
		return extractFromZip(data)
	}
	return extractFromTarGz(data)
}

func extractFromTarGz(data []byte) ([]byte, error) {
	gzr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if h.Typeflag != tar.TypeReg {
			continue
		}
		name := filepath.Base(h.Name)
		if name == "mewsh" || name == "mewsh.exe" {
			return io.ReadAll(tr)
		}
	}
	return nil, fmt.Errorf("mewsh binary not found in archive")
}

func extractFromZip(data []byte) ([]byte, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}
	for _, f := range zr.File {
		name := filepath.Base(f.Name)
		if name != "mewsh" && name != "mewsh.exe" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		bin, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return nil, err
		}
		return bin, nil
	}
	return nil, fmt.Errorf("mewsh binary not found in archive")
}

func replaceExecutable(bin []byte, dest string) error {
	mode := os.FileMode(0o755)
	if runtime.GOOS == "windows" {
		newPath := dest + ".new.exe"
		if err := os.WriteFile(newPath, bin, mode); err != nil {
			return err
		}
		if err := os.Rename(newPath, dest); err != nil {
			return fmt.Errorf("could not replace %s (close other mewsh windows and run: move /Y %q %q): %w", dest, newPath, dest, err)
		}
		return nil
	}

	tmp := dest + ".new"
	if err := os.WriteFile(tmp, bin, mode); err != nil {
		return err
	}
	if err := os.Rename(tmp, dest); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("replace executable: %w", err)
	}
	return nil
}
