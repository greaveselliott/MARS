# Quickstart

Zero to first run in under five minutes.

## Prerequisites

- A machine with a GPU (NVIDIA, Apple Silicon, or AMD ROCm)
- Go 1.22+ (only if building from source)
- Git

## Install

**Option A — Binary release (recommended):**

```bash
curl -sSfL https://raw.githubusercontent.com/greaveselliott/mars-harness/main/scripts/install.sh | bash
```

Or pin a specific version:

```bash
VERSION=v1.0.0 curl -sSfL https://raw.githubusercontent.com/greaveselliott/mars-harness/main/scripts/install.sh | bash
```

**Option B — Build from source:**

```bash
git clone https://github.com/greaveselliott/mars-harness.git
cd mars-harness
make install
```

This installs `mars-harness` into your Go binary directory, usually `$(go env GOPATH)/bin`, and configures Fish, Zsh, Bash, POSIX sh/Ksh, Csh, or Tcsh so new terminals can resolve `mars-harness` automatically.

For one-off source builds, prefer `go build -o build/mars-harness ./cmd/mars-harness`. Avoid `go build ./cmd/mars-harness; ./mars-harness ...`: the semicolon runs the old binary if the build fails, and the source-tree binary is easy to confuse with the installed command.

## Verify

```bash
mars-harness version
# mars-harness v1.0.0 linux/amd64 commit=abc123 built=2026-04-12T00:00:00Z
```

## Upgrade The Command

Upgrade or reinstall the installed command without changing into the source checkout:

```bash
mars-harness update check --repo ~/my-project
mars-harness update tool
```

By default this downloads the latest platform release asset, verifies
`checksums.txt`, atomically replaces the current `mars-harness` binary, and
refreshes shell PATH setup for that directory. For source-development channels:

```bash
mars-harness update tool --source --version main
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

## Setup

Run the first-time wizard. This detects your GPU, downloads a model, and creates `~/.mars-harness/`.

```bash
mars-harness setup
```

Use `--skip-download` to skip model download if you already have a compatible GGUF model. Use `--skip-github` to skip GitHub App configuration.

### Local inference speed

Mars Harness chooses a local inference profile automatically during setup.
On Apple Silicon and other unified-memory machines, this avoids loading the
largest Q8 model when a smaller model is likely to run faster overall.

Manual overrides are still available in `~/.mars-harness/config.yaml`:

```yaml
performance_profile: auto       # auto | quality | balanced | speed
llama_parallel: 1               # default strict-trunk single-agent setting
llama_flash_attention: auto
```

After changing `performance_profile`, run `mars-harness setup` once so any newly required model files are downloaded.

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

### Common flags

| Flag | Description |
|------|-------------|
| `--repo` | Path to the target repository (required) |
| `--dry-run` | Print system prompt and exit |
| `--trace` | Enable verbose trace output |
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
When GitHub release credentials are configured, push tag `vX.Y.Z` at the
release-note commit so release automation can publish the generated changelog
entry and any repo-required assets. A GitHub Release with notes but no required
assets is not complete; run the Release workflow backfill for that tag or record
the blocker. For Mars Harness source releases, verify the binary asset contract
with:

```bash
mars-harness release verify-assets --version vX.Y.Z
```

## Next Steps

- [bundle-reference.md](bundle-reference.md) — manifest.yaml format and role configuration
- [guardrails-guide.md](guardrails-guide.md) — writing safety rules
- `examples/roles/` — starter role prompts ready to adapt for your project
