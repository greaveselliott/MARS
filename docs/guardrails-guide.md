# Guardrails Guide

Guardrails are safety rules that constrain what MARS agents can do. They prevent agents from making dangerous changes, leaking secrets, or violating project conventions.

## Severity Levels

| Severity | Behaviour |
|----------|-----------|
| `hard` | Blocks the operation. The agent cannot proceed until the violation is resolved. |
| `advisory` | Injected into the system prompt as guidance. The agent is warned but not blocked. |

Hard rules are enforced at runtime by the guardrails engine. Advisory rules rely on the LLM following instructions.

## Rule Format

Guardrail files are YAML and live in `.harness/guardrails/`. Reference them from the manifest:

```yaml
# .harness/manifest.yaml
roles:
  engineer:
    guardrails:
      - guardrails/safety.yaml
      - guardrails/conventions.yaml
```

### Rule structure

```yaml
rules:
  - id: unique-rule-id
    name: Human-readable name
    severity: hard          # "hard" or "advisory"
    scope: global           # "global" or a specific role name
    pattern: 'regex'        # Content regex (optional)
    file_pattern: '*.go'    # File glob (optional)
    message: Explanation shown when the rule triggers
    stale_days: 90          # Auto-expire after N days (0 = never)
```

| Field | Required | Description |
|-------|----------|-------------|
| `id` | Yes | Unique identifier for the rule. |
| `name` | Yes | Short name displayed in violations. |
| `severity` | Yes | `hard` (blocks) or `advisory` (warns). |
| `scope` | Yes | `global` applies to all roles, or specify a role name. |
| `pattern` | No | Regex matched against file content. Rule triggers on match. |
| `file_pattern` | No | Glob matched against filenames. Limits which files the rule applies to. |
| `message` | Yes | Explanation shown when the rule triggers. |
| `stale_days` | No | Days until the rule is flagged as stale. Default: 90. Set to -1 for never. |

## Examples

### Prevent hardcoded secrets

```yaml
rules:
  - id: no-hardcoded-secrets
    name: No hardcoded secrets
    severity: hard
    scope: global
    pattern: '(?i)(password|secret|api_key|token)\s*[:=]\s*["\x27][^"\x27]{8,}'
    message: Do not hardcode secrets. Use environment variables or a secrets manager.
```

### Block workflow file edits

```yaml
rules:
  - id: no-ci-edits
    name: No CI workflow modifications
    severity: hard
    scope: pipeline-fixer
    file_pattern: '*.yml'
    pattern: 'on:\s*(push|pull_request)'
    message: Pipeline Fixer must not modify CI workflow files unless the workflow itself is broken.
```

### Enforce test coverage (advisory)

```yaml
rules:
  - id: require-tests
    name: New code needs tests
    severity: advisory
    scope: engineer
    message: Every new function must have a corresponding test. Check coverage before submitting.
```

### Prevent deletions in specific paths

```yaml
rules:
  - id: no-migration-delete
    name: No migration file deletion
    severity: hard
    scope: global
    file_pattern: '*.sql'
    pattern: 'DROP TABLE'
    message: Do not delete migration files or add DROP TABLE statements without explicit approval.
```

### Role-scoped advisory

```yaml
rules:
  - id: refactor-no-api-changes
    name: Refactoring must not change public API
    severity: advisory
    scope: refactorer
    message: Do not rename exported identifiers or change function signatures during refactoring.
```

## Overrides

Hard rules can be temporarily overridden for specific operations. Overrides require a reason and optional expiry:

```go
override := guardrails.Override{
    RuleID:    "no-ci-edits",
    Principal: "admin@example.com",
    Reason:    "Fixing workflow syntax error",
    ExpiresAt: &expiry, // nil = no expiry
}
```

Overrides are logged and auditable. Use them sparingly.

## Stale Rules

Rules auto-expire after `stale_days` (default: 90 days). The `mars doctor` command flags stale rules. Review and either refresh or remove them.

Set `stale_days: -1` for permanent rules that should never expire.

## Best Practices

1. **Start with advisory rules** — observe how agents behave before adding hard blocks.
2. **Scope narrowly** — prefer role-specific rules over global ones to avoid blocking unrelated roles.
3. **Write clear messages** — the message is shown to the agent and to humans reviewing violations.
4. **Review regularly** — stale rules with outdated patterns create false positives.
5. **Test your patterns** — regex mistakes can block legitimate operations or miss violations.
6. **Use `file_pattern` to limit scope** — a broad content regex combined with a narrow file glob is safer than a broad content regex alone.
