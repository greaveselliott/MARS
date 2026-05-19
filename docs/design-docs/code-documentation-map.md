# Code Documentation Map

**Status:** Accepted
**Date:** 2026-05-04
**Owner:** Mars Harness maintainers
**Decision:** AD-101
**Architecture:** [documentation-sync-architecture.md](documentation-sync-architecture.md)

## Purpose

This map is the durable bridge between source files, architecture, and BDD
feature contracts. Every source file carries top-of-file `MarsDocSync` metadata
with a `docs` array. When an agent changes a file, the listed docs are the
minimum documentation review set for that change. The full architecture and
universal operating model live in
[documentation-sync-architecture.md](documentation-sync-architecture.md).

The map is maintained by `internal/docsync` and checked with:

```bash
mars-harness docsync audit --repo .
mars-harness tools run docsync_audit --repo . --args-json '{}'
```

## Metadata Shape

Go, JavaScript, and CSS files use block comments:

```go
/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/features/F-001-delivery-operating-model.md
*/
```

YAML files use `#` comments and HTML templates use `<!-- ... -->`. The `docs`
field is a list of repo-relative documentation paths.

## Package Map

| Source Prefix | Architecture / Product Docs | Feature Contracts |
| --- | --- | --- |
| `.github/workflows/` | `docs/design-docs/release-versioning.md` | `docs/features/F-009-release-update-lifecycle.md` |
| `cmd/mars-harness/` | `docs/product-specs/product-surface.md`, `docs/design-docs/cli-tool-skill-sync.md`, `docs/design-docs/delivery-operating-model.md`, `docs/design-docs/documentation-sync-architecture.md`, `docs/design-docs/release-versioning.md`, `docs/design-docs/self-reflective-telemetry.md`, `docs/design-docs/dashboard.md` | F-001, F-002, F-004, F-005, F-009, F-010, F-012 |
| `examples/` | `docs/design-docs/role-customization.md` | F-004 |
| `internal/agent/` | `docs/design-docs/agent-runtime.md` | F-005 |
| `internal/buildinfo/` | `docs/design-docs/release-versioning.md` | F-009 |
| `internal/bundle/` | `docs/design-docs/context-efficiency.md`, `docs/design-docs/role-customization.md` | F-004, F-005 |
| `internal/config/` | `docs/product-specs/product-surface.md`, `docs/design-docs/release-versioning.md` | F-003, F-009 |
| `internal/context/` | `docs/design-docs/context-efficiency.md` | F-005 |
| `internal/dashboard/` | `docs/design-docs/dashboard.md` | F-010 |
| `internal/docsconsistency/` | `docs/design-docs/delivery-operating-model.md` | F-001 |
| `internal/docsync/` | `docs/design-docs/delivery-operating-model.md`, `docs/design-docs/documentation-sync-architecture.md`, this map | F-001 |
| `internal/doctor/` | `docs/product-specs/product-surface.md` | F-004 |
| `internal/evolution/` | `docs/design-docs/self-improvement.md` | F-012 |
| `internal/foundationtelemetry/` | `docs/design-docs/self-reflective-telemetry.md` | F-012 |
| `internal/github/` | `docs/product-specs/product-surface.md` | F-011 |
| `internal/githubauth/` | `docs/design-docs/release-versioning.md`, `docs/product-specs/product-surface.md` | F-009 |
| `internal/guardrails/` | `docs/design-docs/guardrails.md` | F-007 |
| `internal/hardware/` | `docs/design-docs/local-inference.md` | F-003 |
| `internal/inference/` | `docs/design-docs/local-inference.md` | F-003 |
| `internal/learnings/` | `docs/design-docs/dogfood-and-decisions.md` | F-012 |
| `internal/llm/` | `docs/design-docs/agent-runtime.md`, `docs/design-docs/local-inference.md` | F-003, F-005 |
| `internal/mcpstdio/` | `docs/design-docs/tools-glossary.md` | F-005 |
| `internal/models/` | `docs/design-docs/local-inference.md` | F-003 |
| `internal/operatingmodel/` | `docs/design-docs/harness-operating-model.md` | F-001 |
| `internal/orchestration/` | `docs/design-docs/orchestrated-organization-layer.md` | F-006 |
| `internal/orgstate/` | `docs/design-docs/orchestrated-organization-layer.md` | F-006 |
| `internal/personas/` | `docs/design-docs/harness-operating-model.md`, `docs/roles/ROLES.md` | F-001 |
| `internal/planhygiene/` | `docs/design-docs/self-improvement.md` | F-001 |
| `internal/power/` | `docs/product-specs/product-surface.md` | F-006 |
| `internal/qualityscore/` | `docs/design-docs/scoring-system.md` | F-008 |
| `internal/queue/` | `docs/design-docs/pipeline-engine.md` | F-006 |
| `internal/remediation/` | `docs/design-docs/self-reflective-telemetry.md` | F-012 |
| `internal/release/` | `docs/design-docs/release-versioning.md` | F-009 |
| `internal/roleregistry/` | `docs/design-docs/harness-operating-model.md` | F-001 |
| `internal/safety/` | `docs/design-docs/guardrails.md` | F-007 |
| `internal/sandbox/` | `docs/design-docs/guardrails.md` | F-007 |
| `internal/scanner/` | `docs/design-docs/delivery-operating-model.md` | F-004 |
| `internal/scheduler/` | `docs/design-docs/pipeline-engine.md` | F-006 |
| `internal/scoring/` | `docs/design-docs/scoring-system.md` | F-008 |
| `internal/selfupdate/` | `docs/design-docs/release-versioning.md` | F-009 |
| `internal/serve/` | `docs/design-docs/pipeline-engine.md`, `docs/design-docs/orchestrated-organization-layer.md`, `docs/design-docs/dashboard.md` | F-006, F-010 |
| `internal/setup/` | `docs/design-docs/local-inference.md`, `docs/design-docs/release-versioning.md` | F-002, F-003, F-009 |
| `internal/shellpath/` | `docs/design-docs/release-versioning.md` | F-002 |
| `internal/telemetry/` | `docs/design-docs/self-reflective-telemetry.md` | F-012 |
| `internal/tickets/` | `docs/design-docs/delivery-operating-model.md` | F-001 |
| `internal/tools/` | `docs/design-docs/tools-glossary.md` | F-005 |
| `internal/trace/` | `docs/design-docs/agent-runtime.md` | F-005 |
| `internal/trust/` | `docs/design-docs/scoring-system.md` | F-008 |
| `internal/ui/` | `docs/design-docs/agent-runtime.md`, `docs/design-docs/dashboard.md` | F-005, F-010 |
| `internal/updatecheck/` | `docs/design-docs/release-versioning.md` | F-004 |
| `pkg/testutil/` | `docs/design-docs/agent-runtime.md` | F-005 |

Every row also implicitly includes this map. If a file crosses package
boundaries, add the additional docs directly in that file's metadata.

Notable cross-boundary files:

- `internal/scanner/init.go` and its tests generate target doctrine, role
  registries, tools guidance, release guidance, and F-001 operating-model
  feature docs, so their metadata lists those extra docs directly.
- `internal/tools/formalized_workflows.go` and its tests own `docsync_audit`,
  so their metadata also points to the delivery operating model and F-001.
- `internal/tools/mars_harness_cli.go` and its tests mirror the CLI command
  tree into tool reference and repo-shortcut behavior, so their metadata points
  to [cli-tool-skill-sync.md](cli-tool-skill-sync.md).
- `internal/scanner/init.go` also mirrors CLI/tool/skill sync doctrine into
  generated target guidance and skills.

## Maintenance Rules

- `docsync audit` is the mechanical source-code coverage gate.
- The universal operating model and architecture live in
  [documentation-sync-architecture.md](documentation-sync-architecture.md).
- CLI changes also follow the tool/skill synchronization model in
  [cli-tool-skill-sync.md](cli-tool-skill-sync.md).
- If a source prefix moves, update this document, `internal/docsync`, and the
  affected file metadata in the same change.
- If a code change alters business behavior, update the referenced BDD feature
  contract before claiming the change is complete.
- If a code change alters architecture, generated target behavior, CLI surface,
  or operating doctrine, update the referenced design/product docs before
  claiming the change is complete.
