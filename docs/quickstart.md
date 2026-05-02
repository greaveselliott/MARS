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
go build ./cmd/mars-harness
sudo mv mars-harness /usr/local/bin/
```

## Verify

```bash
mars-harness version
# mars-harness v1.0.0 linux/amd64 commit=abc123 built=2026-04-12T00:00:00Z
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

Execute a role against your repository:

```bash
mars-harness run pipeline-fixer --repo ~/my-project
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

## Next Steps

- [bundle-reference.md](bundle-reference.md) — manifest.yaml format and role configuration
- [guardrails-guide.md](guardrails-guide.md) — writing safety rules
- `examples/roles/` — starter role prompts ready to adapt for your project
