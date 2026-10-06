package hf_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/envblex/huggingface-cli/pkg/hf"
)

func TestDownloadFileCaching(t *testing.T) {
	cacheDir := t.TempDir()
	content := []byte(`{"model_type": "gpt2"}`)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Repo-Commit", "1111222233334444555566667777888899990000")
		w.Header().Set("ETag", `"etag-config-123"`)
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(content)))
		if r.Method == "HEAD" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.Write(content)
	}))
	defer ts.Close()

	client := hf.NewClient(ts.URL)
	outPath, err := client.DownloadFile(hf.DownloadFileOptions{
		RepoID:   "gpt2",
		RepoType: "model",
		Filename: "config.json",
		CacheDir: cacheDir,
		Revision: "main",
		Quiet:    true,
	})
	if err != nil {
		t.Fatalf("DownloadFile failed: %v", err)
	}

	if _, err := os.Stat(outPath); err != nil {
		t.Fatalf("expected file to exist at %s: %v", outPath, err)
	}

	// Verify symlink points to blobs/etag-config-123
	linkTarget, err := os.Readlink(outPath)
	if err != nil {
		t.Fatalf("expected snapshot file to be symlink: %v", err)
	}
	resolved := filepath.Clean(filepath.Join(filepath.Dir(outPath), linkTarget))
	blobPath := filepath.Join(cacheDir, "models--gpt2", "blobs", "etag-config-123")
	if resolved != blobPath {
		t.Fatalf("symlink target %s does not match expected blob %s", resolved, blobPath)
	}
}

func TestFilterFiles(t *testing.T) {
	files := []string{"config.json", "model.safetensors", "model.fp16.safetensors", "README.md"}
	filtered := hf.FilterFiles(files, []string{"*.safetensors"}, []string{"*.fp16.*"})
	if len(filtered) != 1 || filtered[0] != "model.safetensors" {
		t.Fatalf("unexpected filtered files: %v", filtered)
	}
}
