package hf_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/envblex/huggingface-cli/pkg/hf"
)

func TestLFSInitHandshake(t *testing.T) {
	input := `{"event": "init", "operation": "upload", "remote": "origin", "concurrent": true}` + "\n" + `{"event": "terminate"}` + "\n"
	in := strings.NewReader(input)
	out := &bytes.Buffer{}

	err := hf.LFSMultipartUpload(in, out)
	if err != nil {
		t.Fatalf("LFSMultipartUpload failed: %v", err)
	}
	if !strings.Contains(out.String(), "{}\n") {
		t.Fatalf("expected empty ack object on stdout, got: %s", out.String())
	}
}

func TestDumpEnvironmentInfo(t *testing.T) {
	buf := &bytes.Buffer{}
	hf.DumpEnvironmentInfo(buf)
	out := buf.String()
	if !strings.Contains(out, "huggingface_hub version:") || !strings.Contains(out, "ENDPOINT:") {
		t.Fatalf("unexpected env output: %s", out)
	}
}
