# Hugging Face CLI (Go) Command Reference

A comprehensive guide to all subcommands, arguments, flags, and environment variables for the standalone Go `huggingface-cli`.

---

## Table of Contents

- [Global Environment Variables](#global-environment-variables)
- [Authentication Commands](#authentication-commands)
  - [`login`](#huggingface-cli-login)
  - [`whoami`](#huggingface-cli-whoami)
  - [`logout`](#huggingface-cli-logout)
- [Repository Commands](#repository-commands)
  - [`repo create`](#huggingface-cli-repo-create)
- [Transfer Commands](#transfer-commands)
  - [`download`](#huggingface-cli-download)
  - [`upload`](#huggingface-cli-upload)
- [Cache Commands](#cache-commands)
  - [`scan-cache`](#huggingface-cli-scan-cache)
  - [`delete-cache`](#huggingface-cli-delete-cache)
- [Diagnostic & Git-LFS Commands](#diagnostic--git-lfs-commands)
  - [`env`](#huggingface-cli-env)
  - [`lfs-enable-largefiles`](#huggingface-cli-lfs-enable-largefiles)

---

## Global Environment Variables

The CLI strictly respects the standard Hugging Face environment variables:

| Variable | Description | Default |
|---|---|---|
| `HF_TOKEN` | User Access Token used for Hub authentication | Read from `~/.cache/huggingface/token` if unset |
| `HUGGING_FACE_HUB_TOKEN` | Fallback environment variable for `HF_TOKEN` | Read from `~/.cache/huggingface/token` if unset |
| `HF_HOME` | Base directory for Hugging Face data | `~/.cache/huggingface` (or `$XDG_CACHE_HOME/huggingface`) |
| `HF_HUB_CACHE` | Directory where downloaded models and datasets are cached | `$HF_HOME/hub` |
| `HF_ASSETS_CACHE` | Directory for asset caches | `$HF_HOME/assets` |
| `HF_TOKEN_PATH` | Explicit path to the stored token file | `$HF_HOME/token` |
| `HF_ENDPOINT` | Custom Hub API endpoint (for enterprise/staging mirrors) | `https://huggingface.co` |
| `HF_HUB_OFFLINE` | When set to `1` or `true`, disables outgoing HTTP requests | `false` |
| `HF_HUB_DISABLE_PROGRESS_BARS` | When set to `1` or `true`, disables progress output | `false` |
| `HF_HUB_LOCAL_DIR_AUTO_SYMLINK_THRESHOLD` | Threshold in bytes for automatic symlinks vs copies in `--local-dir` | `5242880` (5 MB) |

---

## Authentication Commands

### `huggingface-cli login`

Authenticate your machine against the Hugging Face Hub using a User Access Token.

```bash
huggingface-cli login [--token <TOKEN>] [--add-to-git-credential]
```

- `--token <string>`: User Access Token generated from [huggingface.co/settings/tokens](https://huggingface.co/settings/tokens). If omitted, an interactive terminal prompt will securely prompt for the token.
- `--add-to-git-credential`: Also register the token with the configured system Git credential helper (`git credential approve`).

### `huggingface-cli whoami`

Display the currently authenticated user identity and associated organization memberships.

```bash
huggingface-cli whoami
```

Outputs:
```text
username
orgs:  org-a,org-b
```

### `huggingface-cli logout`

Remove the locally stored token from `$HF_TOKEN_PATH` and remove credentials from the system Git credential store.

```bash
huggingface-cli logout
```

---

## Repository Commands

### `huggingface-cli repo create`

Create a new repository on Hugging Face Hub.

```bash
huggingface-cli repo create <name> [--type <TYPE>] [--organization <ORG>] [--space_sdk <SDK>] [-y|--yes]
```

- `<name>`: The repository name.
- `--type <string>`: Target repository type: `model` (default), `dataset`, or `space`.
- `--organization <string>`: Organization namespace to create the repository under. If omitted, the repository is created under your personal user account.
- `--space_sdk <string>`: Required when `--type space`. Must be one of `gradio`, `streamlit`, `docker`, or `static`.
- `-y`, `--yes`: Skip confirmation prompt.

---

## Transfer Commands

### `huggingface-cli download`

Download files or an entire repository snapshot from the Hub directly to the local cache or a destination folder.

```bash
huggingface-cli download <repo_id> [<filenames>...] [flags]
```

#### Flags

- `--repo-type <string>`: `model` (default), `dataset`, or `space`.
- `--revision <string>`: Branch name, tag, or 40-character commit hash (default: `main`).
- `--include <patterns>`: Comma-separated glob patterns to include (e.g. `"*.safetensors"`).
- `--exclude <patterns>`: Comma-separated glob patterns to exclude (e.g. `"*.fp16.*"`).
- `--cache-dir <dir>`: Custom directory for cache storage (overrides `$HF_HUB_CACHE`).
- `--local-dir <dir>`: Destination folder where files should be copied or symlinked.
- `--local-dir-use-symlinks <string>`: `auto` (symlink files > 5MB, copy smaller files), `True` (always symlink), or `False` (always duplicate).
- `--force-download`: Re-download files even if already present in the cache.
- `--resume-download`: Resume interrupted download using HTTP Range requests.
- `--token <string>`: Explicit token for downloading private or gated repositories.
- `--quiet`: Silence progress bars and diagnostic output. Prints only the final local path to stdout.

#### Examples

```bash
# Download a single file
huggingface-cli download gpt2 config.json

# Download whole repository with 8 parallel worker threads
huggingface-cli download HuggingFaceH4/zephyr-7b-beta

# Download to a project folder
huggingface-cli download gpt2 --local-dir ./models/gpt2

# Download specific patterns only
huggingface-cli download stabilityai/stable-diffusion-xl-base-1.0 --include "*.safetensors" --exclude "*.fp16.*"

# Download a dataset
huggingface-cli download HuggingFaceH4/ultrachat_200k --repo-type dataset
```

---

### `huggingface-cli upload`

Upload files or an entire directory to a Hub repository. Automatically creates the repository if it does not yet exist.

```bash
huggingface-cli upload <repo_id> [<local_path>] [<path_in_repo>] [flags]
```

#### Flags

- `--repo-type <string>`: `model` (default), `dataset`, or `space`.
- `--revision <string>`: Target branch or reference (default: `main`). If the branch does not exist, it is created automatically.
- `--private`: Mark the repository as private if created automatically.
- `--include <patterns>`: Comma-separated glob patterns to include.
- `--exclude <patterns>`: Comma-separated glob patterns to exclude.
- `--delete <patterns>`: Glob patterns of files to delete remotely.
- `--commit-message <string>`: Commit title / headline.
- `--commit-description <string>`: Extended commit description body.
- `--create-pr`: Create a Pull Request instead of committing directly to the target branch.
- `--every <float>`: Periodically upload updates every N minutes.
- `--token <string>`: Explicit token with write permissions.
- `--quiet`: Silence upload progress.

#### Examples

```bash
# Upload a single weights file
huggingface-cli upload my-user/my-model ./model.safetensors

# Upload an entire directory to the repository root
huggingface-cli upload my-user/my-model ./checkpoints .

# Upload to a dataset with a custom commit message
huggingface-cli upload my-user/my-dataset ./data /train --repo-type dataset --commit-message "Upload training split"
```

---

## Cache Commands

### `huggingface-cli scan-cache`

Inspect the local Hugging Face cache directory and report disk space consumption, file counts, and revisions.

```bash
huggingface-cli scan-cache [--dir <DIR>] [-v | -vvv]
```

- `--dir <string>`: Custom cache directory to scan (default: `$HF_HUB_CACHE`).
- `-v`: Verbose output. Lists individual commit revisions instead of repository summaries.
- `-vvv`: Print warnings and details about corrupted or broken symlinks if detected.

---

### `huggingface-cli delete-cache`

Interactive and scriptable removal of cached revisions to free disk space. Calculates unreferenced blob files and deletes unused snapshots and blobs safely.

```bash
huggingface-cli delete-cache [--dir <DIR>] [--disable-tui]
```

---

## Diagnostic & Git-LFS Commands

### `huggingface-cli env`

Dump system architecture, Go version, active configuration, and environment variables for troubleshooting and GitHub issues.

```bash
huggingface-cli env
```

### `huggingface-cli lfs-enable-largefiles`

Configure a local Git repository to use `huggingface-cli` as a Git-LFS custom transfer agent for uploading files larger than 5GB.

```bash
huggingface-cli lfs-enable-largefiles <path-to-repo>
```
