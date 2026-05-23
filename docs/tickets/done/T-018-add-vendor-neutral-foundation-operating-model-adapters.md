---
id: T-018
title: Add vendor-neutral foundation operating model adapters
priority: high
complexity: medium
work_type: enabler
bdd_scenarios: []
end_to_end_evidence: not_applicable
evidence_links:
  - AGENTS.md
  - CLAUDE.md
  - GEMINI.md
  - .github/copilot-instructions.md
  - internal/docsconsistency/foundation_adapters_test.go
verified_by: "go test ./internal/docsconsistency ./internal/docsync ./internal/roleregistry; go test ./cmd/mars-harness ./internal/context ./internal/scanner ./internal/personas ./internal/tools"
owner: "foundation-maintainer"
last_attempt: "2026-05-23"
blocker: "none"
blocked_by: []
trace_id: "manual-foundation-role-adapters-20260523"
next_action: "done"
dedupe_key: "public-example"
source: foundation role and vendor-neutral operating model plan
created: 2026-05-23
depends_on: []
---

# T-018: Add vendor-neutral foundation operating model adapters

## Context

The foundation operating model must work across Claude Code, Cursor, Gemini CLI, Windsurf, OpenCode, GitHub Copilot, Kiro IDE and CLI, Codex, and other AGENTS.md-compatible agents. Vendor files should adapt to the canonical doctrine rather than duplicate it.

## Requirements

- Add thin vendor adapters for Claude Code, Gemini CLI, GitHub Copilot, and existing Cursor rules.
- Document how each supported client enters foundation mode.
- Add docsconsistency coverage that adapters remain pointers/invocation guidance rather than independent doctrine.
- Add a doctrine-leak guard for validation-subject terms in generic adapter and foundation surfaces.

## Affected Files

- AGENTS.md
- CLAUDE.md
- GEMINI.md
- .github/copilot-instructions.md
- .cursor/rules/
- internal/docsconsistency/

## Design Guidance

Keep AGENTS.md and the canonical foundation role packet as source of truth. Vendor adapters are compatibility shims only.

## Acceptance Criteria

- Supported client matrix names every requested AI client.
- Adapter tests fail when unique doctrine is added to vendor shims.
- Demo/test-subject vocabulary is rejected from generic adapter surfaces.
- Evidence surfaces may still mention validation subjects.

## Completion Evidence

- Added thin adapters for Claude Code, Gemini CLI, GitHub Copilot, and existing Cursor rules.
- Added the supported-client matrix to `AGENTS.md` with Claude Code, Cursor, Gemini CLI, Windsurf, OpenCode, GitHub Copilot, Kiro IDE & CLI, and Codex / Other Agents.
- Added docsconsistency gates for adapter thinness, supported-client coverage, and validation-subject doctrine leakage.
- Kept vendor files as pointers to `AGENTS.md` and the canonical foundation role packet instead of duplicating doctrine.
