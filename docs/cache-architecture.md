# Hugging Face Hub Cache Architecture

This document describes the directory hierarchy and storage mechanics employed by `huggingface-cli` (Go). It maintains 100% byte-for-byte and link-for-link parity with Python's [`huggingface_hub`](https://github.com/huggingface/huggingface_hub).

---

## Directory Hierarchy Overview

By default, the cache directory resides at:
- Linux / macOS: `~/.cache/huggingface/hub/`
- Custom location: configured via `$HF_HUB_CACHE` or `$HF_HOME/hub`

The cache folder layout organizes repositories by type (`models`, `datasets`, `spaces`) and repository ID:

```text
~/.cache/huggingface/hub/
├── models--openai-community--gpt2/
│   ├── blobs/
│   │   ├── 10c66461e4c109db5a2196bff4bb59be30396ed8
│   │   └── 248dfc3911869ec493c76e65bf2fcf7f615828b0254c12b473182f0f81d3a707
│   ├── snapshots/
│   │   └── 607a30d783dfa663caf39e06633721c8d4cfcd7e/
│   │       ├── config.json -> ../../blobs/10c66461e4c109db5a2196bff4bb59be30396ed8
│   │       └── model.safetensors -> ../../blobs/248dfc3911869ec493c76e65bf2fcf7f615828b0254c12b473182f0f81d3a707
│   └── refs/
│       └── main  (contains "607a30d783dfa663caf39e06633721c8d4cfcd7e")
└── datasets--glue/
    ├── blobs/
    │   └── ...
    ├── snapshots/
    │   └── ...
    └── refs/
        └── main
```

---

## Core Components

### 1. Repository Folder Names

The directory name encodes both repository type and full repository ID:
- Formula: `<type>s--<namespace>--<name>`
- Slash (`/`) separators in repository IDs are mapped to `--`.
- Examples:
  - `gpt2` (model) -> `models--gpt2`
  - `openai-community/gpt2` (model) -> `models--openai-community--gpt2`
  - `glue` (dataset) -> `datasets--glue`
  - `HuggingFaceH4/zephyr-chat` (space) -> `spaces--HuggingFaceH4--zephyr-chat`

### 2. Blobs Directory (`blobs/`)

The `blobs/` directory contains the actual file content stored on disk:
- Filenames correspond to the server ETag (or SHA-256 for LFS objects).
- Content-addressed deduplication: files identical across multiple revisions or branches share the same underlying blob file on disk.

### 3. Snapshots Directory (`snapshots/<commit_hash>/`)

The `snapshots/` directory reconstructs the exact tree layout of a specific commit:
- Subdirectories are named after the full 40-character Git commit hash.
- Files inside snapshot folders are **relative symbolic links** pointing to `../../blobs/<etag>`.
- Relative symlinks ensure that if the user copies, backs up, or relocates their entire cache folder, symlinks do not break.

### 4. References Directory (`refs/`)

The `refs/` directory stores lightweight plain-text files mapping Git branch or tag names to commit hashes:
- `refs/main` contains the commit hash for the `main` branch.
- `refs/pr/1` contains the commit hash for pull request 1.

---

## Fast-Path Lookup & Network Resilience

When downloading or reading files:
1. **Commit Shortcut**: If the caller passes a 40-character commit hash and the snapshot file exists, the file is served immediately without any network calls.
2. **Ref Fast-Path**: If the branch ref file exists and matches the cached snapshot, downloads are resolved locally.
3. **HEAD Metadata Verification**: If network access is active, a single `HEAD` request inspects `X-Repo-Commit` and `X-Linked-ETag` / `ETag`. If the blob is already present on disk, file downloading is skipped and the pointer symlink is updated immediately.
4. **Offline Mode**: If `$HF_HUB_OFFLINE=1` is set, only local cached entries are queried.

---

## Deletion & Pruning Safety

The `delete-cache` subcommand implements a 4-stage dependency-ordered deletion:
1. **Full Repositories**: Repositories where all revisions are selected for deletion are removed in whole.
2. **Snapshots**: Selected snapshot directories are removed.
3. **References**: Corresponding ref files pointing to deleted revisions are removed.
4. **Unreferenced Blobs**: Blobs are only deleted if they are not referenced by any other surviving revision within the repository.
