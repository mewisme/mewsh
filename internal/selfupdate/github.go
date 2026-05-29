package selfupdate

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"time"
)

const githubRepo = "mewisme/mewsh"

type ghRelease struct {
	TagName string    `json:"tag_name"`
	Assets  []ghAsset `json:"assets"`
}

type ghAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

func fetchLatestRelease() (ghRelease, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest(http.MethodGet, "https://api.github.com/repos/"+githubRepo+"/releases/latest", nil)
	if err != nil {
		return ghRelease{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "mewsh-selfupdate")

	resp, err := client.Do(req)
	if err != nil {
		return ghRelease{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return ghRelease{}, fmt.Errorf("no releases found for %s", githubRepo)
	}
	if resp.StatusCode != http.StatusOK {
		return ghRelease{}, fmt.Errorf("github API: %s", resp.Status)
	}

	var rel ghRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return ghRelease{}, err
	}
	if rel.TagName == "" {
		return ghRelease{}, fmt.Errorf("release has no tag")
	}
	return rel, nil
}

func assetFileName(tag string) string {
	v := strings.TrimPrefix(tag, "v")
	ext := "tar.gz"
	if runtime.GOOS == "windows" {
		ext = "zip"
	}
	return fmt.Sprintf("mewsh_%s_%s_%s.%s", v, runtime.GOOS, runtime.GOARCH, ext)
}

func findAssetURL(rel ghRelease) (string, error) {
	want := assetFileName(rel.TagName)
	for _, a := range rel.Assets {
		if a.Name == want {
			if a.BrowserDownloadURL == "" {
				break
			}
			return a.BrowserDownloadURL, nil
		}
	}
	return "", fmt.Errorf("no release asset %q for %s/%s", want, runtime.GOOS, runtime.GOARCH)
}
