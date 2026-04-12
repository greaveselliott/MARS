---
name: mars-harness-run
description: >-
  Manually execute a single agent role against a repository. Use when the
  user wants to run a specific role, execute an agent, test a role, or
  mentions mars-harness run.
---

# Run an Agent Role

## Prerequisites

1. Setup complete: `mars-harness doctor`
2. Repo has `.harness/` bundle: `mars-harness init --repo /path` (if needed)

## Run

```bash
mars-harness run <role> --repo /path/to/repo
```

Example:

```bash
mars-harness run engineer --repo /path/to/project
mars-harness run pipeline-fixer --repo .
mars-harness run reviewer --repo /path/to/project
```

## Flags

| Flag | Default | Effect |
|---|---|---|
| `--repo` | (required) | Path to the target repository |
| `--model-endpoint` | (auto) | Override LLM endpoint URL |
| `--trace` | `false` | Verbose execution trace output |
| `--dry-run` | `false` | Print system prompt without calling LLM |
| `--budget` | `0` (unlimited) | Token budget |
| `--max-turns` | `50` | Maximum LLM round-trips |

## Dry Run

Preview what the agent will see without calling the LLM:

```bash
mars-harness run engineer --repo /path --dry-run
```

## What Happens

1. Loads `.harness/manifest.yaml` and the role's prompt
2. Assembles the system prompt (role + guardrails + knowledge routes)
3. Auto-starts local inference (llama-server) for the role's model tier
4. Runs the agent conversation loop
5. Prints summary: end reason, LLM calls, tool invocations, wall time

## Troubleshooting

- **Role not found**: Check `.harness/manifest.yaml` for available roles
- **Inference failed**: Run `mars-harness setup` to install models + llama-server
- **Timeout**: Increase `--max-turns` or check model performance
