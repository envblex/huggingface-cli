package hf_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/envblex/huggingface-cli/pkg/hf"
)

func TestScanCacheAndDelete(t *testing.T) {
	cacheDir := t.TempDir()
	repoDir := filepath.Join(cacheDir, "models--test--model")
	blobsDir := filepath.Join(repoDir, "blobs")
	snapshotsDir := filepath.Join(repoDir, "snapshots", "rev1")
	refsDir := filepath.Join(repoDir, "refs")

	os.MkdirAll(blobsDir, 0755)
	os.MkdirAll(snapshotsDir, 0755)
	os.MkdirAll(refsDir, 0755)

	blob1 := filepath.Join(blobsDir, "hash1")
	os.WriteFile(blob1, []byte("data"), 0644)
	os.Symlink(blob1, filepath.Join(snapshotsDir, "config.json"))
	os.WriteFile(filepath.Join(refsDir, "main"), []byte("rev1"), 0644)

	info, err := hf.ScanCacheDir(cacheDir)
	if err != nil {
		t.Fatalf("ScanCacheDir failed: %v", err)
	}
	if len(info.Repos) != 1 {
		t.Fatalf("expected 1 repo, got %d", len(info.Repos))
	}
	repo := info.Repos[0]
	if repo.RepoID != "test/model" || repo.SizeOnDisk != 4 {
		t.Fatalf("unexpected repo info: %+v", repo)
	}

	table := hf.FormatCacheTable(info, 0)
	if !strings.Contains(table, "test/model") || !strings.Contains(table, "REPO ID") {
		t.Fatalf("table missing expected columns or content: %s", table)
	}

	strat, err := info.DeleteRevisions([]string{"rev1"})
	if err != nil {
		t.Fatalf("DeleteRevisions failed: %v", err)
	}
	if err := strat.Execute(); err != nil {
		t.Fatalf("strat.Execute failed: %v", err)
	}

	// Entire repo should be cleaned up
	if _, err := os.Stat(repoDir); !os.IsNotExist(err) {
		t.Fatalf("expected repoDir to be removed, but exists")
	}
}
