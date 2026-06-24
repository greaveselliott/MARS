# Quickstart

Install from a source checkout, download local inference assets, and run Mars
Harness against a target repository.

## System Requirements

- macOS or Linux
- Git
- Go 1.22+
- Network access for setup downloads
- Disk space for multi-GB GGUF model files under `~/.mars-harness/models`
- Recommended GPU: Apple Silicon/Metal, NVIDIA CUDA, or AMD ROCm

CPU fallback exists for development and dry runs, but ordinary autonomous
operation is designed for a GPU-backed local model.

## Install From Source

```bash
git clone https://github.com/greaveselliott/software-factory.git
cd software-factory
make install
```

This installs `mars-harness` into your Go binary directory, usually
`$(go env GOPATH)/bin`, and configures Fish, Zsh, Bash, POSIX sh/Ksh, Csh, or
Tcsh so new terminals can resolve `mars-harness` automatically.

For one-off source builds, prefer `go build -o build/mars-harness ./cmd/mars-harness`. Avoid `go build ./cmd/mars-harness; ./mars-harness ...`: the semicolon runs the old binary if the build fails, and the source-tree binary is easy to confuse with the installed command.

## Verify

```bash
mars-harness version
# mars-harness 0.45.1 darwin/arm64 commit=abc123 built=2026-06-11T00:00:00Z
```

The printed version matches the repo `VERSION` file (no `v` prefix); release
tags use the `vX.Y.Z` form. Run `mars-harness --help` for the full command
surface beyond this quickstart.

## Setup Local Inference

Run the first-time setup wizard:

```bash
mars-harness setup --skip-github
mars-harness doctor
```

Setup creates `~/.mars-harness/`, writes default config, detects hardware,
installs the pinned `llama-server` binary, downloads pinned GGUF models into
`~/.mars-harness/models`, and configures shell PATH for the installed command.

Use `--skip-download` only if compatible model files are already present. Use
`--test-mode` only for dry/local setup paths that avoid downloads and external
services.

Mars Harness chooses a local inference profile automatically during setup. On
Apple Silicon and other unified-memory machines, this avoids loading the
largest Q8 model when a smaller model is likely to run faster overall.

Manual overrides are available in `~/.mars-harness/config.yaml`:

```yaml
performance_profile: auto       # auto | quality | balanced | speed
llama_parallel: 1               # default strict-trunk single-agent setting
llama_flash_attention: auto
```

After changing `performance_profile`, run `mars-harness setup --skip-github`
once so any newly required model files are downloaded.

## Update From A Clone

Run this from the Mars Harness source checkout:

```bash
make update-tool
```

`make update-tool` fast-forwards the checkout from `origin/main` when the
worktree is clean, installs the updated command with `go install`, refreshes
shell PATH setup, and prints `mars-harness version`.

If you have local source changes, commit or stash them first. To install your
current checkout without fetching remote changes, run:

```bash
make install
```

## Optional Binary Releases

Binary release assets and GitHub Release mirrors remain available for
compatibility, but they are not required for source checkout onboarding.

If you use the binary updater, configure release auth once:

```bash
gh auth login
mars-harness auth github setup
mars-harness auth github check
```

Then:

```bash
mars-harness update tool
mars-harness update tool --dry-run
```

If your current shell still sees an older binary after updating, refresh its command cache with `hash -r`, run the reload hint printed by the command, or open a new terminal.

You can rerun shell PATH setup directly:

```bash
mars-harness path setup
```

Update the harness files deployed into a target repo with the same verb:

```bash
mars-harness update harness --repo ~/my-project
```

Use JSON when another agent or automation needs to decide which update action to run:

```bash
mars-harness update check --repo ~/my-project --json
```

## Initialise a Repository

Navigate to your project and scaffold the `.harness/` bundle:

```bash
cd ~/my-project
mars-harness init
```

This creates:

```
.
├── AGENTS.md         # Compact agent entrypoint and workflow map
├── docs/             # Tickets, exec plans, design docs, references
└── .harness/
    ├── manifest.yaml # Role definitions and triggers
    ├── roles/        # Role prompt files
    ├── guardrails/   # Safety rules
    └── knowledge/    # Lightweight context routes, including the glossary route
```

Edit `.harness/manifest.yaml` and `.harness/roles/*.md` to configure the agent team for your repo. The generated roles are starter defaults, not a claim that the shipped agents are perfect for every project. See [bundle-reference.md](bundle-reference.md) for the full format.

## Run a Role

Mars Harness is a command installed on your machine. It should be runnable from any working directory; the target project is always selected with `--repo`.

Execute a role against your repository:

```bash
mars-harness run pipeline-fixer --repo ~/my-project
```

Start the full autonomous loop for one target repo:

```bash
mars-harness start --repo ~/my-project
```

### Dry-run mode

Preview the assembled system prompt without calling the LLM:

```bash
mars-harness run pipeline-fixer --repo ~/my-project --dry-run
```

Inspect an uninitialized target without scaffolding `.harness/`:

```bash
mars-harness run pipeline-fixer --repo ~/legacy-project --dry-run --no-init
```

### Common flags

| Flag | Description |
|------|-------------|
| `--repo` | Path to the target repository (required) |
| `--dry-run` | Print system prompt and exit |
| `--no-init` | Do not auto-initialize a missing target harness; pair with `--dry-run` for observer-safe inspection |
| `--debug` | Stream verbose trace and logs inline instead of the default TTY dashboard |
| `--log-file` | Write verbose command logs to a specific path |
| `--trace` | Compatibility alias for debug-style trace detail on `run` |
| `--model-endpoint` | Override LLM endpoint URL |
| `--max-turns` | Maximum LLM round-trips (default: 50) |
| `--budget` | Token budget (0 = unlimited) |

## Health Check

Verify your setup is complete:

```bash
mars-harness doctor
```

## Patch Notes

Mars Harness uses semantic versions and generated patch notes. The same command works in this repo and in repos initialized by the harness:

```bash
mars-harness release notes --repo . --bump auto --dry-run
mars-harness release notes --repo . --bump auto
```

In this source repo and in repos initialized by Mars Harness, run the release command after every non-release semantic commit and commit the generated version files as `release: notes X.Y.Z` before pushing `main`.
For Mars Harness source releases, push tag `vX.Y.Z` at the release-note commit,
publish local release assets, and verify the binary asset contract with:

```bash
mars-harness release publish-assets --repo . --version vX.Y.Z --upload auto
mars-harness release verify-assets --dist dist/releases --version vX.Y.Z
```

When GitHub release credentials are configured, `--upload auto` also mirrors the
local assets to a GitHub Release. A failed mirror is a release blocker to record,
not hidden CI state.

## Next Steps

- [bundle-reference.md](bundle-reference.md) — manifest.yaml format and role configuration
- [guardrails-guide.md](guardrails-guide.md) — writing safety rules
- `examples/roles/` — starter role prompts ready to adapt for your project
