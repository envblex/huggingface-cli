package hf

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type CachedFileInfo struct {
	FileName         string
	FilePath         string
	BlobPath         string
	SizeOnDisk       int64
	BlobLastAccessed time.Time
	BlobLastModified time.Time
}

type CachedRevisionInfo struct {
	CommitHash   string
	SnapshotPath string
	SizeOnDisk   int64
	Files        []CachedFileInfo
	Refs         []string
	LastModified time.Time
}

type CachedRepoInfo struct {
	RepoID       string
	RepoType     string
	RepoPath     string
	SizeOnDisk   int64
	NbFiles      int
	Revisions    []CachedRevisionInfo
	LastAccessed time.Time
	LastModified time.Time
	Refs         []string
}

type HFCacheInfo struct {
	Repos      []*CachedRepoInfo
	SizeOnDisk int64
	Warnings   []string
}

type DeleteStrategy struct {
	Repos            []string
	Snapshots        []string
	Refs             []string
	Blobs            []string
	ExpectedFreed    int64
	ExpectedFreedStr string
}

// ScanCacheDir scans the HuggingFace Hub cache directory.
func ScanCacheDir(cacheDir string) (*HFCacheInfo, error) {
	if cacheDir == "" {
		cacheDir = HFHubCache()
	}

	fi, err := os.Stat(cacheDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("cache directory not found: %s", cacheDir)
		}
		return nil, err
	}
	if !fi.IsDir() {
		return nil, fmt.Errorf("cache directory is a file: %s", cacheDir)
	}

	entries, err := os.ReadDir(cacheDir)
	if err != nil {
		return nil, err
	}

	info := &HFCacheInfo{}

	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == ".locks" {
			continue
		}

		repoPath := filepath.Join(cacheDir, entry.Name())
		repoInfo, warn := scanCachedRepo(repoPath)
		if warn != "" {
			info.Warnings = append(info.Warnings, warn)
		}
		if repoInfo != nil {
			info.Repos = append(info.Repos, repoInfo)
			info.SizeOnDisk += repoInfo.SizeOnDisk
		}
	}

	// Sort repos by RepoPath
	sort.Slice(info.Repos, func(i, j int) bool {
		return info.Repos[i].RepoPath < info.Repos[j].RepoPath
	})

	return info, nil
}

func scanCachedRepo(repoPath string) (*CachedRepoInfo, string) {
	name := filepath.Base(repoPath)
	if !strings.Contains(name, "--") {
		return nil, fmt.Sprintf("Repo path is not a valid HuggingFace cache directory: %s", repoPath)
	}

	parts := strings.SplitN(name, "--", 2)
	rawType := parts[0]
	if !strings.HasSuffix(rawType, "s") {
		return nil, fmt.Sprintf("Invalid repo type prefix: %s", rawType)
	}
	repoType := rawType[:len(rawType)-1]
	repoID := strings.ReplaceAll(parts[1], "--", "/")

	if repoType != RepoTypeModel && repoType != RepoTypeDataset && repoType != RepoTypeSpace {
		return nil, fmt.Sprintf("Unknown repo type: %s", repoType)
	}

	snapshotsDir := filepath.Join(repoPath, "snapshots")
	refsDir := filepath.Join(repoPath, "refs")
	blobsDir := filepath.Join(repoPath, "blobs")

	if _, err := os.Stat(snapshotsDir); err != nil {
		return nil, fmt.Sprintf("Snapshots dir does not exist in cached repo: %s", snapshotsDir)
	}

	// Map commitHash -> list of ref names
	refsByHash := make(map[string][]string)
	allRepoRefs := make(map[string]bool)

	if _, err := os.Stat(refsDir); err == nil {
		_ = filepath.Walk(refsDir, func(path string, fi os.FileInfo, err error) error {
			if err != nil || fi.IsDir() {
				return nil
			}
			rel, err := filepath.Rel(refsDir, path)
			if err != nil {
				return nil
			}
			data, err := os.ReadFile(path)
			if err == nil {
				hash := strings.TrimSpace(string(data))
				refName := filepath.ToSlash(rel)
				refsByHash[hash] = append(refsByHash[hash], refName)
				allRepoRefs[refName] = true
			}
			return nil
		})
	}

	// Scan blobs directory
	blobStats := make(map[string]os.FileInfo)
	if entries, err := os.ReadDir(blobsDir); err == nil {
		for _, e := range entries {
			if !e.IsDir() {
				bp := filepath.Join(blobsDir, e.Name())
				if fi, err := os.Stat(bp); err == nil {
					blobStats[bp] = fi
				}
			}
		}
	}

	var revisions []CachedRevisionInfo
	var repoLastAccess time.Time
	var repoLastMod time.Time

	revEntries, err := os.ReadDir(snapshotsDir)
	if err == nil {
		for _, revEntry := range revEntries {
			if !revEntry.IsDir() {
				continue
			}

			commitHash := revEntry.Name()
			snapshotRevDir := filepath.Join(snapshotsDir, commitHash)

			var cachedFiles []CachedFileInfo
			revBlobs := make(map[string]int64)
			var revLastMod time.Time

			_ = filepath.Walk(snapshotRevDir, func(path string, fi os.FileInfo, err error) error {
				if err != nil || fi.IsDir() {
					return nil
				}

				realBlobPath, err := filepath.EvalSymlinks(path)
				if err != nil {
					realBlobPath = path
				}

				blobFi, ok := blobStats[realBlobPath]
				if !ok {
					blobFi, _ = os.Stat(realBlobPath)
					if blobFi != nil {
						blobStats[realBlobPath] = blobFi
					}
				}

				var size int64
				var atime, mtime time.Time
				if blobFi != nil {
					size = blobFi.Size()
					mtime = blobFi.ModTime()
					atime = blobFi.ModTime() // fallback for atime
				} else {
					size = fi.Size()
					mtime = fi.ModTime()
					atime = fi.ModTime()
				}

				revBlobs[realBlobPath] = size
				if mtime.After(revLastMod) {
					revLastMod = mtime
				}

				cachedFiles = append(cachedFiles, CachedFileInfo{
					FileName:         fi.Name(),
					FilePath:         path,
					BlobPath:         realBlobPath,
					SizeOnDisk:       size,
					BlobLastAccessed: atime,
					BlobLastModified: mtime,
				})
				return nil
			})

			if revLastMod.IsZero() {
				if fi, err := os.Stat(snapshotRevDir); err == nil {
					revLastMod = fi.ModTime()
				}
			}

			var revSize int64
			for _, sz := range revBlobs {
				revSize += sz
			}

			revRefs := refsByHash[commitHash]
			sort.Strings(revRefs)

			revisions = append(revisions, CachedRevisionInfo{
				CommitHash:   commitHash,
				SnapshotPath: snapshotRevDir,
				SizeOnDisk:   revSize,
				Files:        cachedFiles,
				Refs:         revRefs,
				LastModified: revLastMod,
			})
		}
	}

	var repoTotalSize int64
	for _, fi := range blobStats {
		repoTotalSize += fi.Size()
		if fi.ModTime().After(repoLastMod) {
			repoLastMod = fi.ModTime()
		}
		if fi.ModTime().After(repoLastAccess) {
			repoLastAccess = fi.ModTime()
		}
	}

	if repoLastMod.IsZero() {
		if fi, err := os.Stat(repoPath); err == nil {
			repoLastMod = fi.ModTime()
			repoLastAccess = fi.ModTime()
		}
	}

	var repoRefsList []string
	for r := range allRepoRefs {
		repoRefsList = append(repoRefsList, r)
	}
	sort.Strings(repoRefsList)

	return &CachedRepoInfo{
		RepoID:       repoID,
		RepoType:     repoType,
		RepoPath:     repoPath,
		SizeOnDisk:   repoTotalSize,
		NbFiles:      len(blobStats),
		Revisions:    revisions,
		LastAccessed: repoLastAccess,
		LastModified: repoLastMod,
		Refs:         repoRefsList,
	}, ""
}

// FormatCacheTable generates tabular output matching huggingface-cli scan-cache.
func FormatCacheTable(cacheInfo *HFCacheInfo, verbose int) string {
	if verbose == 0 {
		headers := []string{"REPO ID", "REPO TYPE", "SIZE ON DISK", "NB FILES", "LAST_ACCESSED", "LAST_MODIFIED", "REFS", "LOCAL PATH"}
		var rows [][]string

		for _, repo := range cacheInfo.Repos {
			rows = append(rows, []string{
				repo.RepoID,
				repo.RepoType,
				FormatSize(repo.SizeOnDisk),
				fmt.Sprintf("%d", repo.NbFiles),
				FormatTimesince(repo.LastAccessed),
				FormatTimesince(repo.LastModified),
				strings.Join(repo.Refs, ", "),
				repo.RepoPath,
			})
		}
		return Tabulate(headers, rows)
	}

	// Verbose mode (per revision)
	headers := []string{"REPO ID", "REPO TYPE", "REVISION", "SIZE ON DISK", "NB FILES", "LAST_MODIFIED", "REFS", "LOCAL PATH"}
	var rows [][]string

	for _, repo := range cacheInfo.Repos {
		sort.Slice(repo.Revisions, func(i, j int) bool {
			return repo.Revisions[i].CommitHash < repo.Revisions[j].CommitHash
		})
		for _, rev := range repo.Revisions {
			rows = append(rows, []string{
				repo.RepoID,
				repo.RepoType,
				rev.CommitHash,
				FormatSize(rev.SizeOnDisk),
				fmt.Sprintf("%d", len(rev.Files)),
				FormatTimesince(rev.LastModified),
				strings.Join(rev.Refs, ", "),
				rev.SnapshotPath,
			})
		}
	}
	return Tabulate(headers, rows)
}

// DeleteRevisions computes which directories and blobs need deletion for given commit hashes.
func (info *HFCacheInfo) DeleteRevisions(hashes []string) (*DeleteStrategy, error) {
	hashSet := make(map[string]bool)
	for _, h := range hashes {
		hashSet[strings.TrimSpace(h)] = true
	}

	strategy := &DeleteStrategy{}
	deletedBlobs := make(map[string]bool)

	for _, repo := range info.Repos {
		var revsToDelete []CachedRevisionInfo
		var otherRevs []CachedRevisionInfo

		for _, rev := range repo.Revisions {
			if hashSet[rev.CommitHash] {
				revsToDelete = append(revsToDelete, rev)
			} else {
				otherRevs = append(otherRevs, rev)
			}
		}

		if len(revsToDelete) == 0 {
			continue
		}

		// If all revisions deleted, delete entire repo
		if len(otherRevs) == 0 {
			strategy.Repos = append(strategy.Repos, repo.RepoPath)
			strategy.ExpectedFreed += repo.SizeOnDisk
			continue
		}

		// Otherwise, delete snapshots, refs, and unreferenced blobs
		for _, rev := range revsToDelete {
			strategy.Snapshots = append(strategy.Snapshots, rev.SnapshotPath)

			for _, ref := range rev.Refs {
				refPath := filepath.Join(repo.RepoPath, "refs", filepath.FromSlash(ref))
				strategy.Refs = append(strategy.Refs, refPath)
			}

			// Check which blobs are exclusive to this deleted revision
			for _, file := range rev.Files {
				if deletedBlobs[file.BlobPath] {
					continue
				}
				isShared := false
				for _, other := range otherRevs {
					for _, otherFile := range other.Files {
						if otherFile.BlobPath == file.BlobPath {
							isShared = true
							break
						}
					}
					if isShared {
						break
					}
				}
				if !isShared {
					deletedBlobs[file.BlobPath] = true
					strategy.Blobs = append(strategy.Blobs, file.BlobPath)
					strategy.ExpectedFreed += file.SizeOnDisk
				}
			}
		}
	}

	strategy.ExpectedFreedStr = FormatSize(strategy.ExpectedFreed)
	return strategy, nil
}

// Execute performs deletion in safe order (repos -> snapshots -> refs -> blobs).
func (s *DeleteStrategy) Execute() error {
	for _, p := range s.Repos {
		_ = os.RemoveAll(p)
	}
	for _, p := range s.Snapshots {
		_ = os.RemoveAll(p)
	}
	for _, p := range s.Refs {
		_ = os.Remove(p)
	}
	for _, p := range s.Blobs {
		_ = os.Remove(p)
	}
	return nil
}
