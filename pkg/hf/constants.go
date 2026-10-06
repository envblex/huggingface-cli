package hf

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	DefaultEndpoint = "https://huggingface.co"
	DefaultRevision = "main"

	RepoTypeModel   = "model"
	RepoTypeDataset = "dataset"
	RepoTypeSpace   = "space"

	RepoIDSeparator = "--"

	HeaderRepoCommit = "X-Repo-Commit"
	HeaderLinkedETag = "X-Linked-Etag"
	HeaderLinkedSize = "X-Linked-Size"

	DefaultAutoSymlinkThreshold = 5 * 1024 * 1024 // 5MB

	Version = "0.21.2"
)

var RepoTypes = []string{RepoTypeModel, RepoTypeDataset, RepoTypeSpace}

// IsTrue returns true if s is "1", "true", "on", or "yes" (case-insensitive).
func IsTrue(s string) bool {
	s = strings.ToUpper(strings.TrimSpace(s))
	return s == "1" || s == "ON" || s == "YES" || s == "TRUE"
}

func Endpoint() string {
	if ep := os.Getenv("HF_ENDPOINT"); ep != "" {
		return strings.TrimRight(ep, "/")
	}
	return DefaultEndpoint
}

func HFHome() string {
	if home := os.Getenv("HF_HOME"); home != "" {
		return home
	}
	if xdg := os.Getenv("XDG_CACHE_HOME"); xdg != "" {
		return filepath.Join(xdg, "huggingface")
	}
	userHome, err := os.UserHomeDir()
	if err != nil {
		userHome = "."
	}
	return filepath.Join(userHome, ".cache", "huggingface")
}

func HFHubCache() string {
	if cache := os.Getenv("HF_HUB_CACHE"); cache != "" {
		return cache
	}
	return filepath.Join(HFHome(), "hub")
}

func HFAssetsCache() string {
	if assets := os.Getenv("HF_ASSETS_CACHE"); assets != "" {
		return assets
	}
	return filepath.Join(HFHome(), "assets")
}

func HFTokenPath() string {
	if tp := os.Getenv("HF_TOKEN_PATH"); tp != "" {
		return tp
	}
	return filepath.Join(HFHome(), "token")
}

func HFHubOffline() bool {
	return IsTrue(os.Getenv("HF_HUB_OFFLINE")) || IsTrue(os.Getenv("TRANSFORMERS_OFFLINE"))
}

func HFHubDisableProgressBars() bool {
	return IsTrue(os.Getenv("HF_HUB_DISABLE_PROGRESS_BARS"))
}

func AutoSymlinkThreshold() int64 {
	if v := os.Getenv("HF_HUB_LOCAL_DIR_AUTO_SYMLINK_THRESHOLD"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			return n
		}
	}
	return DefaultAutoSymlinkThreshold
}

func UserAgent() string {
	return "huggingface-cli/" + Version + " (go)"
}
