# Prompt Port Status

Tracks the migration of all 11 Mars automation prompts to harness-native role bundles.

**Source**: Mars monorepo `cursor-automations/` directory
**Target**: `examples/roles/` with manifest at `examples/roles/manifest.yaml`

## Role Checklist

| # | Role | File | Ported | Tested (dry-run) | Manifest Entry |
|---|------|------|--------|-------------------|----------------|
| 1 | Engineer | `engineer.md` | [x] | [x] | [x] |
| 2 | Pipeline Fixer | `pipeline-fixer.md` | [x] | [x] | [x] |
| 3 | QA | `qa.md` | [x] | [x] | [x] |
| 4 | Code Reviewer | `code-reviewer.md` | [x] | [x] | [x] |
| 5 | Release Manager | `release-manager.md` | [x] | [x] | [x] |
| 6 | Docs Writer | `docs-writer.md` | [x] | [x] | [x] |
| 7 | Security Auditor | `security-auditor.md` | [x] | [x] | [x] |
| 8 | Dependency Updater | `dependency-updater.md` | [x] | [x] | [x] |
| 9 | Performance Optimizer | `performance-optimizer.md` | [x] | [x] | [x] |
| 10 | Refactorer | `refactorer.md` | [x] | [x] | [x] |
| 11 | Incident Responder | `incident-responder.md` | [x] | [x] | [x] |

## Port Notes

- All prompts updated to reference harness tool names (`file_read`, `file_write`, `shell_exec`, `grep`)
- Cursor-specific UI instructions removed; replaced with tool-based equivalents
- Each prompt includes `prompt_version` and `source_mars_commit` in header comments
- GitHub-only features guarded with conditional text where applicable
- Token budget considerations deferred to context assembly (MH-004)

## Tier Assignments

| Tier | Roles | Rationale |
|------|-------|-----------|
| `autonomous` | Engineer, Pipeline Fixer, QA, Docs Writer, Incident Responder | Low blast radius, well-scoped triggers |
| `supervised` | Code Reviewer, Release Manager, Security Auditor, Dependency Updater, Performance Optimizer, Refactorer | Higher blast radius or human judgment needed |
