package cli_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/envblex/huggingface-cli/pkg/cli"
)

func TestRootHelp(t *testing.T) {
	cmd := cli.NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	out := buf.String()
	for _, sub := range []string{"env", "login", "whoami", "logout", "repo", "upload", "download", "scan-cache", "delete-cache"} {
		if !strings.Contains(out, sub) {
			t.Fatalf("expected subcommand %s in help output:\n%s", sub, out)
		}
	}
}

func TestDownloadSubcommandHelp(t *testing.T) {
	cmd := cli.NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"download", "--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	out := buf.String()
	for _, flag := range []string{"--repo-type", "--revision", "--include", "--exclude", "--cache-dir", "--local-dir", "--local-dir-use-symlinks", "--quiet"} {
		if !strings.Contains(out, flag) {
			t.Fatalf("expected flag %s in download help output:\n%s", flag, out)
		}
	}
}

func TestUploadSubcommandHelp(t *testing.T) {
	cmd := cli.NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"upload", "--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	out := buf.String()
	for _, flag := range []string{"--repo-type", "--revision", "--include", "--exclude", "--delete", "--commit-message", "--create-pr", "--every"} {
		if !strings.Contains(out, flag) {
			t.Fatalf("expected flag %s in upload help output:\n%s", flag, out)
		}
	}
}
