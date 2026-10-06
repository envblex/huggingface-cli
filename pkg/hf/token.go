package hf

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// GetToken returns the currently active HF token, checking environment variables then cache file.
func GetToken() (string, error) {
	if tok := os.Getenv("HF_TOKEN"); tok != "" {
		return strings.TrimSpace(tok), nil
	}
	if tok := os.Getenv("HUGGING_FACE_HUB_TOKEN"); tok != "" {
		return strings.TrimSpace(tok), nil
	}

	path := HFTokenPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

// SaveToken writes the token to the HF token path.
func SaveToken(token string) error {
	path := HFTokenPath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create token dir: %w", err)
	}
	if err := os.WriteFile(path, []byte(strings.TrimSpace(token)), 0600); err != nil {
		return fmt.Errorf("failed to write token file: %w", err)
	}
	return nil
}

// DeleteToken removes the token file if it exists.
func DeleteToken() error {
	path := HFTokenPath()
	err := os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// SetGitCredential sets the token in the git credential helper store.
func SetGitCredential(token string) error {
	cmd := exec.Command("git", "credential", "approve")
	payload := fmt.Sprintf("protocol=https\nhost=huggingface.co\nusername=user\npassword=%s\n\n", strings.TrimSpace(token))
	cmd.Stdin = bytes.NewBufferString(payload)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git credential approve failed (%s): %w", strings.TrimSpace(string(out)), err)
	}
	return nil
}

// UnsetGitCredential removes the token from the git credential helper store.
func UnsetGitCredential() error {
	cmd := exec.Command("git", "credential", "reject")
	payload := "protocol=https\nhost=huggingface.co\n\n"
	cmd.Stdin = bytes.NewBufferString(payload)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git credential reject failed (%s): %w", strings.TrimSpace(string(out)), err)
	}
	return nil
}

// ListGitCredentialHelpers returns list of configured git credential helpers.
func ListGitCredentialHelpers() ([]string, error) {
	cmd := exec.Command("git", "config", "--get-all", "credential.helper")
	out, err := cmd.Output()
	if err != nil {
		return nil, nil
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var helpers []string
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l != "" {
			helpers = append(helpers, l)
		}
	}
	return helpers, nil
}
