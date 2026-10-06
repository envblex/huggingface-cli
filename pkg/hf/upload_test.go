package hf_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/envblex/huggingface-cli/pkg/hf"
)

func TestUploadFileCommit(t *testing.T) {
	var receivedBody string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/preupload/") {
			w.Write([]byte(`{"files": [{"path": "hello.txt", "uploadMode": "regular", "shouldIgnore": false}]}`))
			return
		}
		if strings.Contains(r.URL.Path, "/commit/") {
			b, _ := io.ReadAll(r.Body)
			receivedBody = string(b)
			json.NewEncoder(w).Encode(map[string]any{
				"commitUrl": "https://huggingface.co/test/model/commit/abc",
				"commitOid": "abc12345",
			})
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "hello.txt")
	os.WriteFile(filePath, []byte("Hello Hub!"), 0644)

	client := hf.NewClient(ts.URL)
	url, err := client.Upload(hf.UploadOptions{
		RepoID:        "test/model",
		LocalPath:     filePath,
		PathInRepo:    "hello.txt",
		Token:         "my_token",
		CommitMessage: "Upload hello.txt",
		Quiet:         true,
	})
	if err != nil {
		t.Fatalf("UploadFile failed: %v", err)
	}
	if !strings.Contains(receivedBody, `"key":"file"`) {
		t.Fatalf("expected ndjson file payload, got: %s", receivedBody)
	}
	if !strings.Contains(url, "commit/abc") {
		t.Fatalf("unexpected return URL: %s", url)
	}
}
