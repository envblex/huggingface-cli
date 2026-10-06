package hf

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type UploadOptions struct {
	RepoID            string
	LocalPath         string
	PathInRepo        string
	RepoType          string
	Revision          string
	Private           bool
	Include           []string
	Exclude           []string
	Delete            []string
	CommitMessage     string
	CommitDescription string
	CreatePR          bool
	Every             float64
	Token             string
	Quiet             bool
}

type fileUploadEntry struct {
	localPath  string
	pathInRepo string
	size       int64
	sha256     string
	sample     []byte
	uploadMode string // "lfs" or "regular"
}

// Upload uploads a file or folder to the Hub.
func (c *Client) Upload(opts UploadOptions) (string, error) {
	if opts.RepoType == "" {
		opts.RepoType = RepoTypeModel
	}
	if opts.Revision == "" {
		opts.Revision = DefaultRevision
	}
	if opts.Token == "" {
		opts.Token, _ = GetToken()
	}

	fi, err := os.Stat(opts.LocalPath)
	if err != nil {
		return "", fmt.Errorf("local path does not exist: %w", err)
	}

	// 1. Ensure repository exists
	if _, err := c.CreateRepo(CreateRepoOptions{
		RepoID:   opts.RepoID,
		RepoType: opts.RepoType,
		Private:  opts.Private,
		ExistOK:  true,
		SpaceSDK: "gradio",
		Token:    opts.Token,
	}); err != nil {
		return "", fmt.Errorf("failed to ensure repo exists: %w", err)
	}

	// 2. Ensure revision branch exists if not main and not PR
	if opts.Revision != DefaultRevision && !opts.CreatePR {
		_ = c.CreateBranch(opts.RepoID, opts.RepoType, opts.Revision, "", opts.Token, true)
	}

	// 3. Handle periodic upload if --every > 0
	if opts.Every > 0 {
		interval := time.Duration(opts.Every * float64(time.Minute))
		fmt.Fprintf(os.Stderr, "Scheduling commits every %.1f minutes to %s...\n", opts.Every, opts.RepoID)
		for {
			url, err := c.executeUpload(opts, fi)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Scheduled upload error: %v\n", err)
			} else {
				fmt.Fprintf(os.Stderr, "Uploaded snapshot: %s\n", url)
			}
			time.Sleep(interval)
		}
	}

	return c.executeUpload(opts, fi)
}

func (c *Client) executeUpload(opts UploadOptions, fi os.FileInfo) (string, error) {
	var entries []*fileUploadEntry

	if !fi.IsDir() {
		// Single file
		pathInRepo := opts.PathInRepo
		if pathInRepo == "" || pathInRepo == "." {
			pathInRepo = filepath.Base(opts.LocalPath)
		}
		pathInRepo = filepath.ToSlash(pathInRepo)

		entry, err := inspectFileForUpload(opts.LocalPath, pathInRepo)
		if err != nil {
			return "", err
		}
		entries = append(entries, entry)
	} else {
		// Directory walk
		err := filepath.Walk(opts.LocalPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				if info.Name() == ".git" {
					return filepath.SkipDir
				}
				return nil
			}

			rel, err := filepath.Rel(opts.LocalPath, path)
			if err != nil {
				return err
			}
			relSlash := filepath.ToSlash(rel)

			// Exclude / Include patterns
			if len(opts.Exclude) > 0 {
				for _, pat := range opts.Exclude {
					if MatchesGlob(pat, relSlash) || MatchesGlob(pat, info.Name()) {
						return nil
					}
				}
			}
			if len(opts.Include) > 0 {
				included := false
				for _, pat := range opts.Include {
					if MatchesGlob(pat, relSlash) || MatchesGlob(pat, info.Name()) {
						included = true
						break
					}
				}
				if !included {
					return nil
				}
			}

			destPath := relSlash
			if opts.PathInRepo != "" && opts.PathInRepo != "." {
				destPath = filepath.ToSlash(filepath.Join(opts.PathInRepo, relSlash))
			}

			entry, err := inspectFileForUpload(path, destPath)
			if err != nil {
				return err
			}
			entries = append(entries, entry)
			return nil
		})
		if err != nil {
			return "", fmt.Errorf("failed walking local directory: %w", err)
		}
	}

	if len(entries) == 0 && len(opts.Delete) == 0 {
		return "", fmt.Errorf("no files to upload")
	}

	// 4. Preupload check: determine uploadMode (LFS vs regular)
	if err := c.preuploadCheck(opts.RepoID, opts.RepoType, opts.Revision, opts.Token, entries); err != nil {
		// Fallback: if preupload endpoint is unavailable, mark large files (>5MB) as LFS
		for _, e := range entries {
			if e.uploadMode == "" {
				if e.size > 5*1024*1024 {
					e.uploadMode = "lfs"
				} else {
					e.uploadMode = "regular"
				}
			}
		}
	}

	// 5. Upload LFS files
	var lfsEntries []*fileUploadEntry
	for _, e := range entries {
		if e.uploadMode == "lfs" {
			lfsEntries = append(lfsEntries, e)
		}
	}
	if len(lfsEntries) > 0 {
		if !opts.Quiet {
			fmt.Fprintf(os.Stderr, "Uploading %d LFS files...\n", len(lfsEntries))
		}
		if err := c.uploadLFSObjects(opts.RepoID, opts.RepoType, opts.Revision, opts.Token, lfsEntries, opts.Quiet); err != nil {
			return "", fmt.Errorf("failed uploading LFS objects: %w", err)
		}
	}

	// 6. Build ndjson commit payload
	var ndjsonLines []string
	summary := opts.CommitMessage
	if summary == "" {
		if fi.IsDir() {
			summary = fmt.Sprintf("Upload %s with huggingface-cli", opts.PathInRepo)
		} else {
			summary = fmt.Sprintf("Upload %s with huggingface-cli", filepath.Base(opts.LocalPath))
		}
	}

	headerPayload, _ := json.Marshal(map[string]any{
		"key": "header",
		"value": map[string]any{
			"summary":     summary,
			"description": opts.CommitDescription,
		},
	})
	ndjsonLines = append(ndjsonLines, string(headerPayload))

	// Add files
	for _, e := range entries {
		if e.uploadMode == "lfs" {
			line, _ := json.Marshal(map[string]any{
				"key": "lfsFile",
				"value": map[string]any{
					"path": e.pathInRepo,
					"algo": "sha256",
					"oid":  e.sha256,
					"size": e.size,
				},
			})
			ndjsonLines = append(ndjsonLines, string(line))
		} else {
			data, err := os.ReadFile(e.localPath)
			if err != nil {
				return "", fmt.Errorf("failed reading file %s: %w", e.localPath, err)
			}
			b64 := base64.StdEncoding.EncodeToString(data)
			line, _ := json.Marshal(map[string]any{
				"key": "file",
				"value": map[string]any{
					"path":     e.pathInRepo,
					"content":  b64,
					"encoding": "base64",
				},
			})
			ndjsonLines = append(ndjsonLines, string(line))
		}
	}

	// Delete patterns
	for _, del := range opts.Delete {
		line, _ := json.Marshal(map[string]any{
			"key": "deletedFile",
			"value": map[string]any{
				"path": del,
			},
		})
		ndjsonLines = append(ndjsonLines, string(line))
	}

	commitBody := strings.Join(ndjsonLines, "\n") + "\n"

	// 7. Post commit
	commitPath := fmt.Sprintf("/api/%ss/%s/commit/%s", opts.RepoType, opts.RepoID, url.PathEscape(opts.Revision))
	if opts.CreatePR {
		commitPath += "?create_pr=1"
	}

	req, err := http.NewRequestWithContext(context.Background(), "POST", c.Endpoint+commitPath, strings.NewReader(commitBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", UserAgent())
	req.Header.Set("Content-Type", "application/x-ndjson")
	if opts.Token != "" {
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(opts.Token))
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("commit request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("commit failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}

	var res struct {
		CommitURL      string `json:"commitUrl"`
		PullRequestURL string `json:"pullRequestUrl"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return fmt.Sprintf("%s/%s", c.Endpoint, opts.RepoID), nil
	}

	if opts.CreatePR && res.PullRequestURL != "" {
		return res.PullRequestURL, nil
	}
	if res.CommitURL != "" {
		return res.CommitURL, nil
	}
	return fmt.Sprintf("%s/%s", c.Endpoint, opts.RepoID), nil
}

func inspectFileForUpload(localPath, pathInRepo string) (*fileUploadEntry, error) {
	f, err := os.Open(localPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	h := sha256.New()
	sample := make([]byte, 512)
	n, _ := f.Read(sample)
	sample = sample[:n]

	// Reset read pos
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	size, err := io.Copy(h, f)
	if err != nil {
		return nil, err
	}

	return &fileUploadEntry{
		localPath:  localPath,
		pathInRepo: pathInRepo,
		size:       size,
		sha256:     hex.EncodeToString(h.Sum(nil)),
		sample:     sample,
	}, nil
}

func (c *Client) preuploadCheck(repoID, repoType, revision, token string, entries []*fileUploadEntry) error {
	type preuploadFile struct {
		Path   string `json:"path"`
		Size   int64  `json:"size"`
		SHA    string `json:"sha"`
		Sample string `json:"sample"`
	}

	path := fmt.Sprintf("/api/%ss/%s/preupload/%s", repoType, repoID, url.PathEscape(revision))

	for i := 0; i < len(entries); i += 256 {
		end := i + 256
		if end > len(entries) {
			end = len(entries)
		}
		chunk := entries[i:end]

		var files []preuploadFile
		for _, e := range chunk {
			files = append(files, preuploadFile{
				Path:   e.pathInRepo,
				Size:   e.size,
				SHA:    e.sha256,
				Sample: base64.StdEncoding.EncodeToString(e.sample),
			})
		}

		payload := map[string]any{"files": files}
		resp, err := c.DoRequest(context.Background(), "POST", path, payload, token)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("preupload returned status %d", resp.StatusCode)
		}

		var res struct {
			Files []struct {
				Path         string `json:"path"`
				UploadMode   string `json:"uploadMode"`
				ShouldIgnore bool   `json:"shouldIgnore"`
			} `json:"files"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
			return err
		}

		modeMap := make(map[string]string)
		for _, f := range res.Files {
			modeMap[f.Path] = f.UploadMode
		}
		for _, e := range chunk {
			if mode, ok := modeMap[e.pathInRepo]; ok {
				e.uploadMode = mode
			} else if e.size > 5*1024*1024 {
				e.uploadMode = "lfs"
			} else {
				e.uploadMode = "regular"
			}
			if e.size == 0 {
				e.uploadMode = "regular"
			}
		}
	}
	return nil
}

func (c *Client) uploadLFSObjects(repoID, repoType, revision, token string, entries []*fileUploadEntry, quiet bool) error {
	type lfsObjectReq struct {
		OID  string `json:"oid"`
		Size int64  `json:"size"`
	}

	var objs []lfsObjectReq
	entryMap := make(map[string]*fileUploadEntry)
	for _, e := range entries {
		objs = append(objs, lfsObjectReq{OID: e.sha256, Size: e.size})
		entryMap[e.sha256] = e
	}

	batchPayload := map[string]any{
		"operation": "upload",
		"transfers": []string{"basic"},
		"objects":   objs,
		"hash_algo": "sha256",
	}

	prefix := ""
	if repoType != RepoTypeModel {
		prefix = repoType + "s/"
	}
	batchPath := fmt.Sprintf("/%s%s.git/info/lfs/objects/batch", prefix, repoID)

	reqBody, _ := json.Marshal(batchPayload)
	req, err := http.NewRequestWithContext(context.Background(), "POST", c.Endpoint+batchPath, bytes.NewReader(reqBody))
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", UserAgent())
	req.Header.Set("Content-Type", "application/vnd.git-lfs+json")
	req.Header.Set("Accept", "application/vnd.git-lfs+json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(token))
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("lfs batch request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("lfs batch failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}

	var batchResp struct {
		Objects []struct {
			OID     string `json:"oid"`
			Actions struct {
				Upload *struct {
					Href   string            `json:"href"`
					Header map[string]string `json:"header"`
				} `json:"upload"`
				Verify *struct {
					Href   string            `json:"href"`
					Header map[string]string `json:"header"`
				} `json:"verify"`
			} `json:"actions"`
		} `json:"objects"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&batchResp); err != nil {
		return fmt.Errorf("failed decoding lfs batch response: %w", err)
	}

	for _, obj := range batchResp.Objects {
		if obj.Actions.Upload == nil {
			// Object already exists on Hub
			continue
		}

		entry := entryMap[obj.OID]
		if entry == nil {
			continue
		}

		file, err := os.Open(entry.localPath)
		if err != nil {
			return err
		}

		uploadReq, err := http.NewRequestWithContext(context.Background(), "PUT", obj.Actions.Upload.Href, file)
		if err != nil {
			file.Close()
			return err
		}
		for k, v := range obj.Actions.Upload.Header {
			uploadReq.Header.Set(k, v)
		}
		uploadReq.ContentLength = entry.size

		putResp, err := c.HTTPClient.Do(uploadReq)
		file.Close()
		if err != nil {
			return fmt.Errorf("lfs put upload failed: %w", err)
		}
		putResp.Body.Close()

		if putResp.StatusCode != http.StatusOK && putResp.StatusCode != http.StatusCreated && putResp.StatusCode != http.StatusNoContent {
			return fmt.Errorf("lfs upload failed with status %d", putResp.StatusCode)
		}

		if obj.Actions.Verify != nil {
			verifyBody, _ := json.Marshal(map[string]any{"oid": obj.OID, "size": entry.size})
			vReq, _ := http.NewRequestWithContext(context.Background(), "POST", obj.Actions.Verify.Href, bytes.NewReader(verifyBody))
			vReq.Header.Set("Content-Type", "application/vnd.git-lfs+json")
			for k, v := range obj.Actions.Verify.Header {
				vReq.Header.Set(k, v)
			}
			vResp, err := c.HTTPClient.Do(vReq)
			if err == nil {
				vResp.Body.Close()
			}
		}
	}

	return nil
}
