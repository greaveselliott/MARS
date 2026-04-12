---
name: mars-harness-register
description: >-
  Register a repository for autonomous management by mars-harness. Use when
  the user wants to add a repo, register a project, connect a repository,
  or mentions mars-harness register.
---

# Register a Repository

## Prerequisites

The repo must be a git repository. If `.harness/manifest.yaml` is missing, `register` runs the same scaffold as `mars-harness init` automatically. To overwrite an existing bundle, run `init --force` first.

## Register

```bash
mars-harness register --repo /path/to/repo --remote owner/repo-name
```

## Flags

| Flag | Default | Effect |
|---|---|---|
| `--repo` | Current directory | Local path to the repository |
| `--remote` | (empty) | GitHub `owner/repo` for webhook matching |
| `--branch` | `main` | Default branch name |
| `--db` | `~/.mars-harness/db/mars.db` | SQLite database path |

## What Happens

1. Ensures `.harness/manifest.yaml` exists (auto-init if missing), then validates it
2. Registers the repo in the SQLite database with a unique ID
3. If the orchestrator is running, it picks up the new repo on next trigger index rebuild

## Verify

Check the repo is registered (the register command prints the ID on success).

## Troubleshooting

- **Missing manifest**: Run `mars-harness init --repo /path` first
- **Already registered**: Re-registering the same path updates the remote/branch (upsert)
