# huggingface-cli (Go Standalone Binary)

[![Go Report Card](https://goreportcard.com/badge/github.com/envblex/huggingface-cli)](https://goreportcard.com/report/github.com/envblex/huggingface-cli)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-%3E%3D%201.21-00ADD8.svg)](https://golang.org)

A high-performance, single standalone Go binary rewriting the Python `huggingface-cli` (from [`huggingface_hub` v0.21.2](https://huggingface.co/docs/huggingface_hub/v0.21.2/guides/cli)).

---

## Why Go?

- **Zero Runtime Dependencies**: Single self-contained static executable (~8.5MB). No Python interpreter, `pip`, or virtualenv required.
- **Instantaneous Startup**: Cold start in `< 3ms` (compared to 200ms–400ms for Python imports), making it ideal for container init scripts, CI/CD pipelines, and shell hooks.
- **100% Byte-for-Byte Cache Compatibility**: Uses the exact standard Hugging Face Hub cache hierarchy (`~/.cache/huggingface/hub/` with `blobs/`, `snapshots/<sha>/`, `refs/<rev>`, and relative symlinks). Works transparently alongside Python libraries (`transformers`, `diffusers`).
- **Resilient & Fast Downloads**: Multi-worker concurrent downloads, resume support, intelligent S3/CloudFront CDN redirect handling, and progress bars.

---

## Installation

### From Source

```bash
git clone https://github.com/envblex/huggingface-cli.git
cd huggingface-cli
make install
```

This compiles the binary with stripped symbols (`-s -w`) and installs both `huggingface-cli` and the short alias `hf` into `~/.local/bin/`.

### Using `go install`

```bash
go install github.com/envblex/huggingface-cli/cmd/huggingface-cli@latest
```

---

## Quickstart

### Authentication & Identity

```bash
# Diagnostic environment report (cache dirs, token status, whoami)
huggingface-cli env

# Login interactively or via flag
huggingface-cli login --token $HF_TOKEN --add-to-git-credential

# Verify current account identity
huggingface-cli whoami

# Logout and purge cached credentials
huggingface-cli logout
```

### Downloading Models & Datasets

```bash
# Download a single file
huggingface-cli download gpt2 config.json

# Download directly to a target folder
huggingface-cli download gpt2 config.json --local-dir ./models/gpt2

# Download an entire repository snapshot (concurrently)
huggingface-cli download HuggingFaceH4/zephyr-7b-beta

# Download a dataset or Space
huggingface-cli download HuggingFaceH4/ultrachat_200k --repo-type dataset
huggingface-cli download HuggingFaceH4/zephyr-chat --repo-type space

# Filter files with glob patterns
huggingface-cli download stabilityai/stable-diffusion-xl-base-1.0 --include "*.safetensors" --exclude "*.fp16.*"

# Quiet mode for shell automation (prints only local file path)
MODEL_PATH=$(huggingface-cli download gpt2 --quiet)
```

### Uploading Files & Directories

```bash
# Upload a single file
huggingface-cli upload my-user/my-model ./model.safetensors

# Upload an entire directory
huggingface-cli upload my-user/my-model ./models .

# Upload to a dataset
huggingface-cli upload my-user/my-dataset ./data /train --repo-type dataset

# Upload with commit message and description
huggingface-cli upload my-user/my-model ./models . --commit-message "Upload weights" --commit-description "Epoch 10 checkpoint"
```

### Cache Inspection & Safe Pruning

```bash
# Scan local Hub cache and report sizes and revisions
huggingface-cli scan-cache

# Verbose scan displaying all individual revisions
huggingface-cli scan-cache -v

# Interactively delete unneeded revisions to free disk space
huggingface-cli delete-cache
```

---

## Documentation

Detailed documentation is available in the [`docs/`](docs/) directory:

- [Command Line Reference](docs/cli-reference.md): Complete list of commands, flags, and options.
- [Cache Architecture](docs/cache-architecture.md): Technical details on cache layouts and symlinks.
- [Go vs Python Comparison](docs/comparison.md): Feature matrix, startup latency benchmarks, and memory efficiency comparison.

---

## Supported Commands

| Command | Description |
|---|---|
| `login` | Authenticate using a Hugging Face token and store Git credentials |
| `whoami` | Print authenticated user identity and organization affiliations |
| `logout` | Clear stored credentials and remove Git helper entries |
| `repo create` | Create a new model, dataset, or space repository on the Hub |
| `download` | Download files or complete snapshots from the Hub |
| `upload` | Commit files or directories to the Hub using the ndjson commit API |
| `scan-cache` | Scan cache directory and report repository and revision sizes |
| `delete-cache` | Safely prune unreferenced revisions and blobs |
| `env` | Output machine and configuration diagnostics |
| `lfs-enable-largefiles` | Configure Git repo for files > 5GB using custom transfer agent |

---

## License

This project is licensed under the Apache License, Version 2.0. See [LICENSE](LICENSE) for details.
