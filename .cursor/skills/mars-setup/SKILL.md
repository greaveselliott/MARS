---
name: mars-setup
description: >-
  First-time setup of mars. Use when the user wants to install
  mars, set up the system, detect hardware, download models,
  install llama-server, or mentions mars setup.
---

# First-Time Setup

Run the setup wizard. This is idempotent — safe to re-run.

```bash
mars setup
```

## What It Does

1. Creates `~/.mars/` directory structure
2. Writes default `config.yaml`
3. Detects GPU/CPU/RAM hardware profile
4. Downloads and installs `llama-server` binary
5. Downloads GGUF models appropriate for detected hardware
6. Configures GitHub App integration (if not skipped)

## Flags

| Flag | Effect |
|---|---|
| `--skip-download` | Skip model + llama-server download |
| `--skip-github` | Skip GitHub App setup |
| `--test-mode` | Skip all downloads and external services |
| `--dry-run` | Print steps without executing |

## Verify Setup

```bash
mars doctor
```

Expected: all checks pass. If any fail, the doctor output includes remediation commands.

## Troubleshooting

- **Disk space error**: Free at least 10 GB in `~/.mars/models/`
- **Network timeout**: The wizard downloads from HuggingFace and GitHub. Check connectivity and retry.
- **Permission denied**: Ensure write access to `~/.mars/`
- **GPU not detected**: On macOS, unified memory is used. On Linux, ensure `nvidia-smi` is in PATH.
