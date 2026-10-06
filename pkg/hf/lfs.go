package hf

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// LFSEnableLargefiles sets git config for git-lfs custom multipart transfer agent.
func LFSEnableLargefiles(repoPath string) error {
	absPath, err := filepath.Abs(repoPath)
	if err != nil {
		return err
	}
	gitDir := filepath.Join(absPath, ".git")
	if fi, err := os.Stat(gitDir); err != nil || !fi.IsDir() {
		return fmt.Errorf("this does not look like a valid git repo: %s", absPath)
	}

	cmd1 := exec.Command("git", "config", "lfs.customtransfer.multipart.path", "huggingface-cli")
	cmd1.Dir = absPath
	if out, err := cmd1.CombinedOutput(); err != nil {
		return fmt.Errorf("failed setting lfs path config (%s): %w", string(out), err)
	}

	cmd2 := exec.Command("git", "config", "lfs.customtransfer.multipart.args", "lfs-multipart-upload")
	cmd2.Dir = absPath
	if out, err := cmd2.CombinedOutput(); err != nil {
		return fmt.Errorf("failed setting lfs args config (%s): %w", string(out), err)
	}

	return nil
}

// LFSMultipartUpload runs the line-delimited JSON protocol for git-lfs upload.
func LFSMultipartUpload(in io.Reader, out io.Writer) error {
	scanner := bufio.NewScanner(in)
	writeMsg := func(v any) error {
		b, err := json.Marshal(v)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(out, "%s\n", string(b))
		return err
	}

	// 1. Initial handshake
	if !scanner.Scan() {
		return nil
	}
	var initMsg map[string]any
	if err := json.Unmarshal(scanner.Bytes(), &initMsg); err != nil {
		_ = writeMsg(map[string]any{"error": map[string]any{"code": 32, "message": "Malformed init message"}})
		return err
	}

	if initMsg["event"] != "init" || initMsg["operation"] != "upload" {
		_ = writeMsg(map[string]any{"error": map[string]any{"code": 32, "message": "Wrong lfs init operation"}})
		return fmt.Errorf("wrong lfs init operation: %+v", initMsg)
	}

	// Ack handshake
	if err := writeMsg(map[string]any{}); err != nil {
		return err
	}

	client := &http.Client{Timeout: 0}

	// 2. Process transfer requests
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var msg map[string]any
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			continue
		}

		event, _ := msg["event"].(string)
		msgType, _ := msg["type"].(string)
		if event == "terminate" || msgType == "terminate" {
			return nil
		}

		if event != "upload" {
			continue
		}

		oid, _ := msg["oid"].(string)
		path, _ := msg["path"].(string)
		action, _ := msg["action"].(map[string]any)
		if action == nil {
			continue
		}

		completionURL, _ := action["href"].(string)
		header, _ := action["header"].(map[string]any)
		if header == nil {
			continue
		}

		chunkSizeStr, _ := header["chunk_size"].(string)
		chunkSize, _ := strconv.ParseInt(chunkSizeStr, 10, 64)
		if chunkSize <= 0 {
			chunkSize = 5 * 1024 * 1024
		}

		// Sort presigned url keys
		var urlKeys []string
		for k := range header {
			if k != "chunk_size" {
				urlKeys = append(urlKeys, k)
			}
		}
		sort.Strings(urlKeys)

		// Send initial progress
		_ = writeMsg(map[string]any{
			"event":         "progress",
			"oid":           oid,
			"bytesSoFar":    1,
			"bytesSinceLast": 0,
		})

		file, err := os.Open(path)
		if err != nil {
			_ = writeMsg(map[string]any{
				"event": "complete",
				"oid":   oid,
				"error": map[string]any{"code": 1, "message": err.Error()},
			})
			continue
		}

		type partInfo struct {
			ETag       string `json:"etag"`
			PartNumber int    `json:"partNumber"`
		}
		var parts []partInfo
		var uploadErr error

		for i, k := range urlKeys {
			presignedURL, _ := header[k].(string)
			sectionReader := io.NewSectionReader(file, int64(i)*chunkSize, chunkSize)

			req, err := http.NewRequest("PUT", presignedURL, sectionReader)
			if err != nil {
				uploadErr = err
				break
			}
			req.ContentLength = sectionReader.Size()

			putResp, err := client.Do(req)
			if err != nil {
				uploadErr = err
				break
			}
			etag := CleanETag(putResp.Header.Get("ETag"))
			putResp.Body.Close()

			parts = append(parts, partInfo{
				ETag:       etag,
				PartNumber: i + 1,
			})

			_ = writeMsg(map[string]any{
				"event":         "progress",
				"oid":           oid,
				"bytesSoFar":    int64(i+1) * chunkSize,
				"bytesSinceLast": chunkSize,
			})
		}
		file.Close()

		if uploadErr != nil {
			_ = writeMsg(map[string]any{
				"event": "complete",
				"oid":   oid,
				"error": map[string]any{"code": 1, "message": uploadErr.Error()},
			})
			continue
		}

		// Complete multipart upload
		completePayload, _ := json.Marshal(map[string]any{
			"oid":   oid,
			"parts": parts,
		})
		compReq, _ := http.NewRequest("POST", completionURL, bytes.NewReader(completePayload))
		compReq.Header.Set("Content-Type", "application/json")
		compResp, err := client.Do(compReq)
		if err != nil {
			_ = writeMsg(map[string]any{
				"event": "complete",
				"oid":   oid,
				"error": map[string]any{"code": 1, "message": err.Error()},
			})
			continue
		}
		compResp.Body.Close()

		_ = writeMsg(map[string]any{
			"event": "complete",
			"oid":   oid,
		})
	}

	return scanner.Err()
}
