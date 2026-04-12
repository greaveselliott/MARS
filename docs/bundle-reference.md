# Bundle Reference

A bundle is the `.harness/` directory in your repository. It contains everything Mars Harness needs to operate on your codebase.

## Directory Structure

```
.harness/
├── manifest.yaml       # Required: role definitions and metadata
├── roles/              # Role prompt files (markdown)
│   ├── engineer.md
│   └── pipeline-fixer.md
├── guardrails/         # Guardrail rule files (YAML)
│   └── safety.yaml
└── knowledge/          # Context files injected into prompts
    ├── architecture.md
    └── api-conventions.md
```

## manifest.yaml

The manifest is the entry point. It declares the bundle name and all roles.

```yaml
name: my-project
description: Mars Harness bundle for my-project

roles:
  pipeline-fixer:
    prompt: roles/pipeline-fixer.md
    model: ""
    tools:
      - file_read
      - file_write
      - shell_exec
      - grep
    guardrails:
      - guardrails/safety.yaml
    knowledge:
      - knowledge/architecture.md
    triggers:
      - workflow_run.conclusion == "failure"
```

### Top-level fields

| Field | Required | Description |
|-------|----------|-------------|
| `name` | Yes | Bundle identifier. Used in job IDs and logs. |
| `description` | No | Human-readable description. |
| `roles` | Yes | Map of role name → role config. At least one required. |

### Role configuration

| Field | Required | Description |
|-------|----------|-------------|
| `prompt` | Yes | Path to the role's markdown prompt file, relative to `.harness/`. |
| `model` | No | Model hint (e.g. `gemma-4-27b`). Empty string uses the default. |
| `tools` | No | List of tools the role is allowed to use. |
| `guardrails` | No | List of guardrail files to load for this role. |
| `knowledge` | No | List of knowledge files to inject into the context. |
| `triggers` | No | Events that activate this role. |

### Available tools

| Tool | Description |
|------|-------------|
| `file_read` | Read file contents |
| `file_write` | Write or create files |
| `shell_exec` | Execute shell commands |
| `grep` | Search file contents with regex |

### Trigger syntax

Triggers use the format `<event>.<field> == "<value>"` or shorthand names:

```yaml
triggers:
  - workflow_run.conclusion == "failure"   # CI failure
  - pull_request.opened                    # PR opened
  - pull_request.synchronize               # PR updated
  - pull_request.merged                    # PR merged
  - schedule.weekly                        # Weekly cron
  - schedule.daily                         # Daily cron
  - workflow_dispatch                      # Manual trigger
  - ticket.assigned                        # Ticket assigned
  - alert.fired                            # Alert triggered
```

## Role Prompts

Role prompts are markdown files containing the system instructions for the agent. They should include:

1. **Identity** — who the agent is and what it does
2. **Workflow** — step-by-step instructions
3. **Constraints** — what the agent must not do
4. **Output format** — expected response structure

See `examples/roles/` for complete examples of all 11 roles.

## Guardrails

Guardrail files define safety rules enforced during execution. See [guardrails-guide.md](guardrails-guide.md) for the full format.

```yaml
rules:
  - id: no-secrets
    name: No hardcoded secrets
    severity: hard
    scope: global
    pattern: '(?i)(password|secret|api_key)\s*[:=]\s*["\x27][^"\x27]{8,}'
    message: Do not hardcode secrets in source files
```

## Knowledge Routes

Knowledge files are plain markdown or text files injected into the role's context during assembly. Use them for:

- Architecture overviews the agent needs to respect
- API conventions and naming standards
- Domain-specific terminology

Keep knowledge files concise (under 500 lines). The context assembly engine (MH-004) manages token budgets automatically.

## Example: Minimal Bundle

```yaml
name: my-app
description: Minimal harness bundle
roles:
  pipeline-fixer:
    prompt: roles/pipeline-fixer.md
    tools:
      - file_read
      - file_write
      - shell_exec
      - grep
    triggers:
      - workflow_run.conclusion == "failure"
```

With a single file at `.harness/roles/pipeline-fixer.md` containing the role prompt.
