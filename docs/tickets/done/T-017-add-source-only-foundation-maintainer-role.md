---
id: T-017
title: Add source-only foundation-maintainer role
priority: high
complexity: medium
work_type: enabler
bdd_scenarios: []
end_to_end_evidence: not_applicable
evidence_links:
  - docs/roles/personas/foundation-maintainer.md
  - docs/features/F-005-agent-execution-runtime.md#F-005-S063
  - cmd/mars-harness/main_test.go
verified_by: "go test ./cmd/mars-harness ./internal/context ./internal/scanner ./internal/personas ./internal/tools; go test ./internal/docsconsistency ./internal/docsync ./internal/roleregistry"
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

# T-017: Add source-only foundation-maintainer role

## Context

Agents working on mars-harness need a role-shaped way to consume the foundation operating model without turning deployed targets into source-maintenance contexts.

## Requirements

- Define foundation-maintainer as source-only and manual/operator-invoked.
- Support mars-harness run foundation-maintainer --repo . --dry-run --no-init without scaffolding .harness/manifest.yaml into the source repo.
- Reject foundation-maintainer for non-source repositories with an actionable error.
- Keep generated target manifests free of foundation-maintainer.

## Affected Files

- cmd/mars-harness/main.go
- docs/roles/personas/foundation-maintainer.md
- docs/roles/ROLES.md
- docs/design-docs/harness-operating-model.md

## Design Guidance

Use the existing role, domain, mode, context assembly, and docsync vocabulary. The role is source-only, not mirrored.

## Acceptance Criteria

- Dry-run assembles the foundation role prompt from repo-owned docs.
- Dry-run does not create a source .harness/manifest.yaml.
- Non-source repos reject the role.
- Generated target init/upgrade output does not include the role.

## Completion Evidence

- Added the canonical source-only foundation role packet at `docs/roles/personas/foundation-maintainer.md`.
- Added `mars-harness run foundation-maintainer --repo . --dry-run --no-init` support from source context without target scaffolding.
- Added non-source rejection coverage and generated target exclusion coverage.
- Added source registry generation for source-only rows while preserving deployed target registries.
