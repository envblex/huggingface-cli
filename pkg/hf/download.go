package hf

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

type DownloadFileOptions struct {
	RepoID              string
	RepoType            string
	Revision            string
	Filename            string
	CacheDir            string
	LocalDir            string
	LocalDirUseSymlinks string // "auto", "true", "false"
	ForceDownload       bool
	ResumeDownload      bool
	Token               string
	Quiet               bool
}

type FileMetadata struct {
	CommitHash string
	ETag       string
	Size       int64
	Location   string
}

func CleanETag(etag string) string {
	etag = strings.TrimSpace(etag)
	etag = strings.TrimPrefix(etag, "W/")
	etag = strings.Trim(etag, "\"")
	return etag
}

// GetFileMetadata fetches file HEAD information (commit, etag, size, redirect).
func (c *Client) GetFileMetadata(repoID, repoType, revision, filename, token string) (*FileMetadata, error) {
	if repoType == "" {
		repoType = RepoTypeModel
	}
	if revision == "" {
		revision = DefaultRevision
	}

	var resolveURL string
	if repoType == RepoTypeModel {
		resolveURL = fmt.Sprintf("%s/%s/resolve/%s/%s", c.Endpoint, repoID, revision, filename)
	} else {
		resolveURL = fmt.Sprintf("%s/%ss/%s/resolve/%s/%s", c.Endpoint, repoType, repoID, revision, filename)
	}

	req, err := http.NewRequestWithContext(context.Background(), "HEAD", resolveURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", UserAgent())
	req.Header.Set("Accept-Encoding", "identity")
	if token == "" {
		token, _ = GetToken()
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(token))
	}

	// Do not auto-follow redirects for HEAD so we inspect headers and location directly
	noRedirectClient := &http.Client{
		Timeout: c.HTTPClient.Timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	resp, err := noRedirectClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed HEAD request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("file not found on Hub: %s at revision %s", filename, revision)
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusFound && resp.StatusCode != http.StatusTemporaryRedirect {
		return nil, fmt.Errorf("metadata lookup failed with status %d", resp.StatusCode)
	}

	meta := &FileMetadata{}
	meta.CommitHash = resp.Header.Get(HeaderRepoCommit)

	etag := resp.Header.Get(HeaderLinkedETag)
	if etag == "" {
		etag = resp.Header.Get("ETag")
	}
	meta.ETag = CleanETag(etag)

	sizeStr := resp.Header.Get(HeaderLinkedSize)
	if sizeStr == "" {
		sizeStr = resp.Header.Get("Content-Length")
	}
	if s, err := strconv.ParseInt(sizeStr, 10, 64); err == nil {
		meta.Size = s
	}

	if loc := resp.Header.Get("Location"); loc != "" {
		meta.Location = loc
	} else {
		meta.Location = resolveURL
	}

	return meta, nil
}

// DownloadFile downloads a single file from the Hub into cache or local dir.
func (c *Client) DownloadFile(opts DownloadFileOptions) (string, error) {
	if opts.RepoType == "" {
		opts.RepoType = RepoTypeModel
	}
	if opts.Revision == "" {
		opts.Revision = DefaultRevision
	}
	if opts.CacheDir == "" {
		opts.CacheDir = HFHubCache()
	}

	storageFolder := filepath.Join(opts.CacheDir, RepoFolderName(opts.RepoID, opts.RepoType))
	blobsDir := filepath.Join(storageFolder, "blobs")
	snapshotsDir := filepath.Join(storageFolder, "snapshots")
	refsDir := filepath.Join(storageFolder, "refs")

	if err := os.MkdirAll(blobsDir, 0755); err != nil {
		return "", err
	}
	if err := os.MkdirAll(snapshotsDir, 0755); err != nil {
		return "", err
	}
	if err := os.MkdirAll(refsDir, 0755); err != nil {
		return "", err
	}

	// Check ref or existing commit
	commitHash := opts.Revision
	if refData, err := os.ReadFile(filepath.Join(refsDir, opts.Revision)); err == nil {
		commitHash = strings.TrimSpace(string(refData))
	}

	// Quick check if file already exists in snapshot
	snapshotFile := filepath.Join(snapshotsDir, commitHash, filepath.FromSlash(opts.Filename))
	if !opts.ForceDownload && commitHash != opts.Revision {
		if fi, err := os.Stat(snapshotFile); err == nil && !fi.IsDir() {
			if opts.LocalDir != "" {
				return placeInLocalDir(snapshotFile, opts.LocalDir, opts.Filename, opts.LocalDirUseSymlinks)
			}
			return snapshotFile, nil
		}
	}

	// Query remote metadata
	meta, err := c.GetFileMetadata(opts.RepoID, opts.RepoType, opts.Revision, opts.Filename, opts.Token)
	if err != nil {
		// Offline fallback check
		if fi, errStat := os.Stat(snapshotFile); errStat == nil && !fi.IsDir() {
			if opts.LocalDir != "" {
				return placeInLocalDir(snapshotFile, opts.LocalDir, opts.Filename, opts.LocalDirUseSymlinks)
			}
			return snapshotFile, nil
		}
		return "", err
	}

	commitHash = meta.CommitHash
	if commitHash == "" {
		commitHash = opts.Revision
	}
	snapshotFile = filepath.Join(snapshotsDir, commitHash, filepath.FromSlash(opts.Filename))

	blobPath := filepath.Join(blobsDir, meta.ETag)
	if meta.ETag == "" {
		// Fallback for empty or unversioned files
		blobPath = filepath.Join(blobsDir, commitHash+"_"+filepath.Base(opts.Filename))
	}

	// If revision is a branch or tag name (not the 40-char hash), write ref
	if opts.Revision != commitHash && len(opts.Revision) != 40 {
		refFile := filepath.Join(refsDir, opts.Revision)
		_ = os.MkdirAll(filepath.Dir(refFile), 0755)
		_ = os.WriteFile(refFile, []byte(commitHash), 0644)
	}

	// If blob already exists on disk
	if fi, err := os.Stat(blobPath); err == nil && !fi.IsDir() && !opts.ForceDownload {
		_ = CreateRelativeSymlink(blobPath, snapshotFile)
		if opts.LocalDir != "" {
			return placeInLocalDir(blobPath, opts.LocalDir, opts.Filename, opts.LocalDirUseSymlinks)
		}
		return snapshotFile, nil
	}

	// Perform actual GET download
	downloadURL := meta.Location
	req, err := http.NewRequestWithContext(context.Background(), "GET", downloadURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", UserAgent())

	// Only send Auth token if downloading directly from Hub endpoint (not external CDN)
	parsedDL, _ := url.Parse(downloadURL)
	parsedEndpoint, _ := url.Parse(c.Endpoint)
	if parsedDL != nil && parsedEndpoint != nil && parsedDL.Host == parsedEndpoint.Host {
		token := opts.Token
		if token == "" {
			token, _ = GetToken()
		}
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(token))
		}
	}

	var resumeOffset int64 = 0
	incompletePath := blobPath + ".incomplete"
	if opts.ResumeDownload {
		if fi, err := os.Stat(incompletePath); err == nil && fi.Size() > 0 && fi.Size() < meta.Size {
			resumeOffset = fi.Size()
			req.Header.Set("Range", fmt.Sprintf("bytes=%d-", resumeOffset))
		}
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("download GET failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("download failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var outFile *os.File
	if resumeOffset > 0 && resp.StatusCode == http.StatusPartialContent {
		outFile, err = os.OpenFile(incompletePath, os.O_WRONLY|os.O_APPEND, 0644)
	} else {
		outFile, err = os.Create(incompletePath)
		resumeOffset = 0
	}
	if err != nil {
		return "", fmt.Errorf("failed to create temp blob: %w", err)
	}

	pw := NewProgressWriter(outFile, meta.Size, opts.Filename, opts.Quiet)
	pw.Current = resumeOffset
	_, copyErr := io.Copy(pw, resp.Body)
	pw.Finish()
	outFile.Close()

	if copyErr != nil {
		return "", fmt.Errorf("download interrupted: %w", copyErr)
	}

	// Rename temp file into final blob path
	if err := os.Rename(incompletePath, blobPath); err != nil {
		return "", fmt.Errorf("failed to move blob into place: %w", err)
	}

	// Create relative symlink in snapshots
	if err := CreateRelativeSymlink(blobPath, snapshotFile); err != nil {
		return "", fmt.Errorf("failed to create snapshot symlink: %w", err)
	}

	if opts.LocalDir != "" {
		return placeInLocalDir(blobPath, opts.LocalDir, opts.Filename, opts.LocalDirUseSymlinks)
	}

	return snapshotFile, nil
}

func placeInLocalDir(srcBlobPath, localDir, filename, useSymlinks string) (string, error) {
	dstPath := filepath.Join(localDir, filepath.FromSlash(filename))
	if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
		return "", err
	}

	useSymlinks = strings.ToLower(strings.TrimSpace(useSymlinks))
	shouldSymlink := true

	fi, err := os.Stat(srcBlobPath)
	if err == nil {
		switch useSymlinks {
		case "false":
			shouldSymlink = false
		case "true":
			shouldSymlink = true
		default: // "auto"
			shouldSymlink = fi.Size() > AutoSymlinkThreshold()
		}
	}

	if shouldSymlink {
		if err := CreateRelativeSymlink(srcBlobPath, dstPath); err == nil {
			return dstPath, nil
		}
	}

	// Copy file if symlink is false or failed
	src, err := os.Open(srcBlobPath)
	if err != nil {
		return "", err
	}
	defer src.Close()

	dst, err := os.Create(dstPath)
	if err != nil {
		return "", err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return "", err
	}
	return dstPath, nil
}

type SnapshotDownloadOptions struct {
	RepoID              string
	RepoType            string
	Revision            string
	Filenames           []string
	AllowPatterns       []string
	IgnorePatterns      []string
	CacheDir            string
	LocalDir            string
	LocalDirUseSymlinks string
	ForceDownload       bool
	ResumeDownload      bool
	Token               string
	Quiet               bool
	MaxWorkers          int
}

// MatchesGlob reports whether name matches pattern with support for '*' and '?'
func MatchesGlob(pattern, name string) bool {
	matched, err := filepath.Match(pattern, name)
	if err == nil && matched {
		return true
	}
	// Support deep globs e.g. *.safetensors matching dir/foo.safetensors
	if strings.HasPrefix(pattern, "*") {
		suffix := strings.TrimPrefix(pattern, "*")
		if strings.HasSuffix(name, suffix) {
			return true
		}
	}
	return false
}

func FilterFiles(all []string, allow, ignore []string) []string {
	var res []string
	for _, file := range all {
		if len(allow) > 0 {
			matched := false
			for _, pat := range allow {
				if MatchesGlob(pat, file) || MatchesGlob(pat, filepath.Base(file)) {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}

		if len(ignore) > 0 {
			ignored := false
			for _, pat := range ignore {
				if MatchesGlob(pat, file) || MatchesGlob(pat, filepath.Base(file)) {
					ignored = true
					break
				}
			}
			if ignored {
				continue
			}
		}

		res = append(res, file)
	}
	return res
}

// SnapshotDownload downloads all files (or filtered subset) of a repo snapshot.
func (c *Client) SnapshotDownload(opts SnapshotDownloadOptions) (string, error) {
	if opts.MaxWorkers <= 0 {
		opts.MaxWorkers = 8
	}
	if opts.CacheDir == "" {
		opts.CacheDir = HFHubCache()
	}
	if opts.Revision == "" {
		opts.Revision = DefaultRevision
	}
	if opts.RepoType == "" {
		opts.RepoType = RepoTypeModel
	}

	info, err := c.RepoInfo(opts.RepoID, opts.RepoType, opts.Revision, opts.Token)
	if err != nil {
		return "", fmt.Errorf("failed to fetch repo info: %w", err)
	}

	var allFiles []string
	if len(opts.Filenames) > 0 {
		allFiles = opts.Filenames
	} else {
		for _, s := range info.Siblings {
			allFiles = append(allFiles, s.RFilename)
		}
	}

	filesToDownload := FilterFiles(allFiles, opts.AllowPatterns, opts.IgnorePatterns)
	if !opts.Quiet && len(filesToDownload) > 0 {
		fmt.Fprintf(os.Stderr, "Fetching %d files...\n", len(filesToDownload))
	}

	sem := make(chan struct{}, opts.MaxWorkers)
	var wg sync.WaitGroup
	var downloadErr error
	var errMu sync.Mutex

	for _, file := range filesToDownload {
		wg.Add(1)
		sem <- struct{}{}

		go func(filename string) {
			defer wg.Done()
			defer func() { <-sem }()

			_, err := c.DownloadFile(DownloadFileOptions{
				RepoID:              opts.RepoID,
				RepoType:            opts.RepoType,
				Revision:            info.SHA,
				Filename:            filename,
				CacheDir:            opts.CacheDir,
				LocalDir:            opts.LocalDir,
				LocalDirUseSymlinks: opts.LocalDirUseSymlinks,
				ForceDownload:       opts.ForceDownload,
				ResumeDownload:      opts.ResumeDownload,
				Token:               opts.Token,
				Quiet:               opts.Quiet,
			})
			if err != nil {
				errMu.Lock()
				if downloadErr == nil {
					downloadErr = fmt.Errorf("failed downloading %s: %w", filename, err)
				}
				errMu.Unlock()
			}
		}(file)
	}

	wg.Wait()
	if downloadErr != nil {
		return "", downloadErr
	}

	if opts.LocalDir != "" {
		absLocal, _ := filepath.Abs(opts.LocalDir)
		return absLocal, nil
	}

	storageFolder := filepath.Join(opts.CacheDir, RepoFolderName(opts.RepoID, opts.RepoType))
	return filepath.Join(storageFolder, "snapshots", info.SHA), nil
}
