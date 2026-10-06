package hf_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/envblex/huggingface-cli/pkg/hf"
)

func TestWhoamiAndRepoInfo(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/whoami-v2":
			auth := r.Header.Get("Authorization")
			if auth != "Bearer valid_token" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			json.NewEncoder(w).Encode(map[string]any{
				"name": "tester",
				"orgs": []map[string]string{{"name": "org-a"}, {"name": "org-b"}},
				"auth": map[string]any{
					"accessToken": map[string]string{"role": "write"},
				},
			})
		case "/api/models/test/model":
			json.NewEncoder(w).Encode(map[string]any{
				"id":  "test/model",
				"sha": "abcdef123456",
				"siblings": []map[string]string{
					{"rfilename": "config.json"},
					{"rfilename": "model.safetensors"},
				},
			})
		case "/api/repos/create":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{
				"url": "https://huggingface.co/test/new-model",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	client := hf.NewClient(ts.URL)
	info, err := client.Whoami("valid_token")
	if err != nil {
		t.Fatalf("Whoami failed: %v", err)
	}
	if info.Name != "tester" || len(info.Orgs) != 2 || info.Role != "write" {
		t.Fatalf("unexpected Whoami result: %+v", info)
	}

	repo, err := client.RepoInfo("test/model", "model", "", "")
	if err != nil {
		t.Fatalf("RepoInfo failed: %v", err)
	}
	if repo.SHA != "abcdef123456" || len(repo.Siblings) != 2 {
		t.Fatalf("unexpected RepoInfo result: %+v", repo)
	}

	url, err := client.CreateRepo(hf.CreateRepoOptions{
		RepoID: "test/new-model",
	})
	if err != nil {
		t.Fatalf("CreateRepo failed: %v", err)
	}
	if url != "https://huggingface.co/test/new-model" {
		t.Fatalf("unexpected CreateRepo URL: %s", url)
	}
}
