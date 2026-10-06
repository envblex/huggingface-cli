package hf

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// FormatSize formats size in bytes into human-readable string (e.g. 116.3K, 64.9M, 1.9G).
func FormatSize(num int64) string {
	numF := float64(num)
	units := []string{"", "K", "M", "G", "T", "P", "E", "Z"}
	for _, unit := range units {
		if math.Abs(numF) < 1000.0 {
			if unit == "" {
				return fmt.Sprintf("%.0f", numF)
			}
			return fmt.Sprintf("%.1f%s", numF, unit)
		}
		numF /= 1000.0
	}
	return fmt.Sprintf("%.1fY", numF)
}

// FormatTimesince formats timestamp into human-readable relative time (e.g. "3 days ago").
func FormatTimesince(ts time.Time) string {
	delta := time.Since(ts)
	if delta < 20*time.Second {
		return "a few seconds ago"
	}

	chunks := []struct {
		label   string
		divider time.Duration
		maxVal  int
	}{
		{"second", time.Second, 60},
		{"minute", time.Minute, 60},
		{"hour", time.Hour, 24},
		{"day", 24 * time.Hour, 6},
		{"week", 7 * 24 * time.Hour, 6},
		{"month", 30 * 24 * time.Hour, 11},
		{"year", 365 * 24 * time.Hour, 0},
	}

	for _, c := range chunks {
		val := int(math.Round(float64(delta) / float64(c.divider)))
		if c.maxVal == 0 || val <= c.maxVal {
			s := ""
			if val > 1 {
				s = "s"
			}
			return fmt.Sprintf("%d %s%s ago", val, c.label, s)
		}
	}
	return "long ago"
}

// RepoFolderName serializes a repo ID and type to cache directory name (e.g. models--gpt2).
func RepoFolderName(repoID, repoType string) string {
	if repoType == "" {
		repoType = RepoTypeModel
	}
	parts := append([]string{repoType + "s"}, strings.Split(repoID, "/")...)
	return strings.Join(parts, RepoIDSeparator)
}

// CreateRelativeSymlink creates a symbolic link from dst to src using a relative path.
func CreateRelativeSymlink(src, dst string) error {
	_ = os.Remove(dst)
	_ = os.MkdirAll(filepath.Dir(dst), 0755)

	rel, err := filepath.Rel(filepath.Dir(dst), src)
	if err != nil {
		rel = src
	}
	return os.Symlink(rel, dst)
}
