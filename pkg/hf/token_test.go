package hf_test

import (
	"path/filepath"
	"testing"

	"github.com/envblex/huggingface-cli/pkg/hf"
)

func TestTokenStorage(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HF_HOME", tmpDir)

	tokenPath := hf.HFTokenPath()
	if filepath.Dir(tokenPath) != tmpDir {
		t.Fatalf("expected token in %s, got %s", tmpDir, tokenPath)
	}

	testToken := "hf_test_1234567890abcdef"
	if err := hf.SaveToken(testToken); err != nil {
		t.Fatalf("SaveToken failed: %v", err)
	}

	tok, err := hf.GetToken()
	if err != nil {
		t.Fatalf("GetToken failed: %v", err)
	}
	if tok != testToken {
		t.Fatalf("expected token %q, got %q", testToken, tok)
	}

	if err := hf.DeleteToken(); err != nil {
		t.Fatalf("DeleteToken failed: %v", err)
	}
	tok, _ = hf.GetToken()
	if tok != "" {
		t.Fatalf("expected empty token after DeleteToken, got %q", tok)
	}
}

func TestTokenEnvPrecedence(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HF_HOME", tmpDir)
	_ = hf.SaveToken("saved_token")

	t.Setenv("HF_TOKEN", "env_token")
	tok, err := hf.GetToken()
	if err != nil {
		t.Fatalf("GetToken failed: %v", err)
	}
	if tok != "env_token" {
		t.Fatalf("expected env_token, got %s", tok)
	}
}
