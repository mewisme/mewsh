package cloudflared

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/mewisme/mewsh/internal/config"
)

const latestReleaseURL = "https://api.github.com/repos/cloudflare/cloudflared/releases/latest"

type release struct {
	Assets []asset `json:"assets"`
}

type asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

func ResolvePath(cfg *config.Config) (string, error) {
	if path, err := resolveExisting(cfg); err != nil || path != "" {
		return path, err
	}
	path, err := DownloadLatest(cfg)
	if err != nil {
		return "", err
	}
	return path, nil
}

// ResolvePathForConnect returns cloudflared without downloading (connect must stay fast).
func ResolvePathForConnect(cfg *config.Config) (string, error) {
	path, err := resolveExisting(cfg)
	if err != nil {
		return "", err
	}
	if path != "" {
		return path, nil
	}
	return "", fmt.Errorf("cloudflared not found — run: mewsh cloudflared update")
}

func resolveExisting(cfg *config.Config) (string, error) {
	if cfg.CloudflaredPath != "" {
		if fileExists(cfg.CloudflaredPath) {
			return cfg.CloudflaredPath, nil
		}
	}
	bundled, err := config.BundledCloudflaredPath()
	if err != nil {
		return "", err
	}
	if fileExists(bundled) {
		return bundled, nil
	}
	if path, err := execLookPath("cloudflared"); err == nil {
		return path, nil
	}
	return "", nil
}

func Update(cfg *config.Config) (string, error) {
	return downloadAndInstall(cfg)
}

func DownloadLatest(cfg *config.Config) (string, error) {
	return downloadAndInstall(cfg)
}

func downloadAndInstall(cfg *config.Config) (string, error) {
	asset, err := fetchMatchingAsset()
	if err != nil {
		return "", err
	}
	dest, err := config.BundledCloudflaredPath()
	if err != nil {
		return "", err
	}
	if err := config.EnsureDir(); err != nil {
		return "", err
	}
	tmp, err := downloadFile(asset.BrowserDownloadURL)
	if err != nil {
		return "", err
	}
	defer os.Remove(tmp)

	if strings.HasSuffix(asset.Name, ".tgz") {
		if err := extractCloudflaredFromTGZ(tmp, dest); err != nil {
			return "", err
		}
	} else {
		if err := copyFile(tmp, dest); err != nil {
			return "", err
		}
	}

	if runtime.GOOS != "windows" {
		if err := os.Chmod(dest, 0755); err != nil {
			return "", err
		}
	}

	cfg.CloudflaredPath = dest
	if err := config.Save(cfg); err != nil {
		return "", err
	}
	return dest, nil
}

func fetchMatchingAsset() (asset, error) {
	resp, err := http.Get(latestReleaseURL)
	if err != nil {
		return asset{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return asset{}, fmt.Errorf("github api returned %s", resp.Status)
	}
	var rel release
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return asset{}, err
	}
	pattern := assetPattern()
	for _, a := range rel.Assets {
		if isPackageAsset(a.Name) {
			continue
		}
		if matchesAsset(a.Name, pattern) {
			return a, nil
		}
	}
	return asset{}, fmt.Errorf("no cloudflared release asset found for %s/%s", runtime.GOOS, runtime.GOARCH)
}

func assetPattern() string {
	switch runtime.GOOS {
	case "windows":
		switch runtime.GOARCH {
		case "386":
			return "windows-386.exe"
		default:
			return "windows-amd64.exe"
		}
	case "linux":
		switch runtime.GOARCH {
		case "arm64":
			return "linux-arm64"
		default:
			return "linux-amd64"
		}
	case "darwin":
		if runtime.GOARCH == "amd64" {
			return "darwin-amd64.tgz"
		}
		return "darwin-arm64.tgz"
	default:
		return runtime.GOOS + "-" + runtime.GOARCH
	}
}

func matchesAsset(name, pattern string) bool {
	if runtime.GOOS == "linux" && runtime.GOARCH == "amd64" {
		return name == "cloudflared-linux-amd64" || strings.Contains(name, "linux-amd64")
	}
	return strings.Contains(name, pattern)
}

func isPackageAsset(name string) bool {
	lower := strings.ToLower(name)
	for _, ext := range []string{".deb", ".rpm", ".pkg"} {
		if strings.HasSuffix(lower, ext) {
			return true
		}
	}
	return false
}

func downloadFile(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed: %s", resp.Status)
	}
	f, err := os.CreateTemp("", "cloudflared-*")
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err := io.Copy(f, resp.Body); err != nil {
		os.Remove(f.Name())
		return "", err
	}
	return f.Name(), nil
}

func extractCloudflaredFromTGZ(src, dest string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		base := filepath.Base(hdr.Name)
		if base != "cloudflared" && base != "cloudflared.exe" {
			continue
		}
		out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
		if err != nil {
			return err
		}
		if _, err := io.Copy(out, tr); err != nil {
			out.Close()
			return err
		}
		return out.Close()
	}
	return fmt.Errorf("cloudflared binary not found in archive")
}

func copyFile(src, dest string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func execLookPath(name string) (string, error) {
	return exec.LookPath(name)
}
