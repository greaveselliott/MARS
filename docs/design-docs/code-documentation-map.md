# Code Documentation Map

**Status:** Accepted
**Date:** 2026-05-04
**Owner:** MARS maintainers
**Decision:** AD-101
**Architecture:** [documentation-sync-architecture.md](documentation-sync-architecture.md)

## Purpose

This map is the durable bridge between source files, architecture, and BDD
feature contracts. Every audited source file carries near-top `MarsDocSync`
metadata with associated documentation paths. When an agent changes a file, the
listed docs are the minimum documentation review set for that change. The full
architecture and universal operating model live in
[documentation-sync-architecture.md](documentation-sync-architecture.md).

The map is maintained by `internal/docsync` and checked with:

```bash
mars docsync audit --repo .
mars tools run docsync_audit --repo . --args-json '{}'
```

## Metadata Shape

Go, JavaScript, TypeScript, TSX, and CSS files use block comments:

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

Deployed static CSS and JavaScript may use a compact inline form when the doc
list is short:

```css
/* MarsDocSync: ["docs/features/F-001-product-walking-skeleton.md"] */
```

`internal/docsync` audits both foundation roots and deployed app roots. In the
foundation source checkout, prefix rules below define expected docs. In a
deployed target repo, product code under common layouts such as `cmd/`,
`internal/`, `pkg/`, `src/`, `app/`, `apps/`, `pages/`, `packages/`, `public/`,
`web/`, `workers/`, `static/`, and `tests/`
still needs valid metadata, and root-level source files such as `main.go` or
`index.html` are included as audited source too. Deployed targets do not
inherit this foundation package map; metadata must point to the target's local
feature contracts or design docs. The mirrored `file_write` policy rejects
source/test writes that omit valid `MarsDocSync` metadata or point at a missing
doc path, so generated target agents correct documentation routing before the
source file exists in the worktree.

Generated manifests expose `docsync.include_roots`,
`docsync.include_extensions`, and `docsync.exclude_globs`. Empty or absent
fields retain safe built-in defaults; non-empty fields replace that field's
defaults. Repository-relative path validation and generated-output exclusions
are part of the audit contract described in F-019.

## Package Map

| Source Prefix | Architecture / Product Docs | Feature Contracts |
| --- | --- | --- |
| `cmd/mars/` | `docs/product-specs/product-surface.md`, `docs/design-docs/cli-tool-skill-sync.md`, `docs/design-docs/delivery-operating-model.md`, `docs/design-docs/documentation-sync-architecture.md`, `docs/design-docs/release-versioning.md`, `docs/design-docs/self-reflective-telemetry.md`, `docs/design-docs/dashboard.md`, `docs/design-docs/github-app-integration.md` | F-001, F-002, F-004, F-005, F-009, F-010, F-011, F-012, F-017 |
| `examples/` | `docs/design-docs/role-customization.md` | F-004 |
| `internal/agent/` | `docs/design-docs/agent-runtime.md`, `docs/design-docs/context-efficiency.md` | F-005 |
| `internal/buildinfo/` | `docs/design-docs/release-versioning.md` | F-009 |
| `internal/bundle/` | `docs/design-docs/context-efficiency.md`, `docs/design-docs/role-customization.md` | F-004, F-005 |
| `internal/config/` | `docs/product-specs/product-surface.md`, `docs/design-docs/release-versioning.md`, `docs/design-docs/github-app-integration.md` | F-003, F-009, F-011, F-017 |
| `Makefile` | `docs/design-docs/release-versioning.md`, `docs/design-docs/dogfood-matrix.md` | F-002, F-009 |
| `internal/context/` | `docs/design-docs/context-efficiency.md` | F-005 |
| `internal/dashboard/` | `docs/design-docs/dashboard.md`, `docs/design-docs/github-app-integration.md` | F-010, F-017 |
| `internal/docsconsistency/` | `docs/design-docs/delivery-operating-model.md` | F-001 |
| `internal/docsync/` | `docs/design-docs/delivery-operating-model.md`, `docs/design-docs/documentation-sync-architecture.md`, this map | F-001, F-019 |
| `internal/doctor/` | `docs/product-specs/product-surface.md`, `docs/design-docs/self-reflective-telemetry.md` | F-004, F-012 |
| `internal/evolution/` | `docs/design-docs/self-improvement.md` | F-012 |
| `internal/foundationtelemetry/` | `docs/design-docs/self-reflective-telemetry.md` | F-012 |
| `internal/github/` | `docs/product-specs/product-surface.md`, `docs/design-docs/github-app-integration.md` | F-006, F-011, F-017 |
| `internal/githubauth/` | `docs/design-docs/release-versioning.md`, `docs/product-specs/product-surface.md` | F-009 |
| `internal/guardrails/` | `docs/design-docs/guardrails.md` | F-007 |
| `internal/hardware/` | `docs/design-docs/local-inference.md` | F-003 |
| `internal/inference/` | `docs/design-docs/local-inference.md` | F-003 |
| `internal/integrations/` | `docs/design-docs/board-driven-integrations.md`, `docs/runbooks/atlassian-mcp-jira-intake.md` | F-013 |
| `internal/jira/` | `docs/design-docs/board-driven-integrations.md`, `docs/runbooks/atlassian-mcp-jira-intake.md` | F-013 |
| `internal/learnings/` | `docs/design-docs/dogfood-and-decisions.md` | F-012 |
| `internal/llm/` | `docs/design-docs/agent-runtime.md`, `docs/design-docs/local-inference.md`, `docs/design-docs/context-efficiency.md` | F-003, F-005 |
| `internal/mcpclient/` | `docs/design-docs/board-driven-integrations.md`, `docs/runbooks/atlassian-mcp-jira-intake.md` | F-013 |
| `internal/mcpstdio/` | `docs/design-docs/tools-glossary.md` | F-005 |
| `internal/models/` | `docs/design-docs/local-inference.md` | F-003 |
| `internal/operatingmodel/` | `docs/design-docs/harness-operating-model.md` | F-001 |
| `internal/orchestration/` | `docs/design-docs/orchestrated-organization-layer.md` | F-006 |
| `internal/orgstate/` | `docs/design-docs/orchestrated-organization-layer.md` | F-006 |
| `internal/personas/` | `docs/design-docs/harness-operating-model.md`, `docs/roles/ROLES.md` | F-001 |
| `internal/planhygiene/` | `docs/design-docs/self-improvement.md` | F-001 |
| `internal/power/` | `docs/product-specs/product-surface.md` | F-006 |
| `internal/qualityscore/` | `docs/design-docs/scoring-system.md`, `docs/design-docs/self-reflective-telemetry.md` | F-008, F-012 |
| `internal/network/` | `docs/design-docs/dashboard.md`, `docs/design-docs/github-app-integration.md` | F-010, F-017 |
| `internal/queue/` | `docs/design-docs/pipeline-engine.md`, `docs/design-docs/github-app-integration.md` | F-006, F-011, F-017 |
| `internal/remediation/` | `docs/design-docs/self-reflective-telemetry.md` | F-012 |
| `internal/release/` | `docs/design-docs/release-versioning.md` | F-009 |
| `internal/roleregistry/` | `docs/design-docs/harness-operating-model.md` | F-001 |
| `internal/safety/` | `docs/design-docs/guardrails.md` | F-007 |
| `internal/sandbox/` | `docs/design-docs/guardrails.md` | F-007 |
| `internal/scanner/` | `docs/design-docs/delivery-operating-model.md` | F-004 |
| `internal/scheduler/` | `docs/design-docs/pipeline-engine.md` | F-006 |
| `internal/scoring/` | `docs/design-docs/scoring-system.md` | F-008 |
| `internal/selfupdate/` | `docs/design-docs/release-versioning.md` | F-009 |
| `internal/serve/` | `docs/design-docs/pipeline-engine.md`, `docs/design-docs/orchestrated-organization-layer.md`, `docs/design-docs/dashboard.md`, `docs/design-docs/github-app-integration.md`, `docs/design-docs/self-reflective-telemetry.md` | F-006, F-010, F-011, F-012, F-017 |
| `internal/serve/remediation*.go` | `docs/design-docs/pipeline-engine.md`, `docs/design-docs/self-reflective-telemetry.md` | F-006, F-012 |
| `internal/setup/` | `docs/design-docs/local-inference.md`, `docs/design-docs/release-versioning.md` | F-002, F-003, F-009 |
| `internal/shellpath/` | `docs/design-docs/release-versioning.md` | F-002 |
| `internal/telemetry/` | `docs/design-docs/self-reflective-telemetry.md` | F-012 |
| `internal/tickets/` | `docs/design-docs/delivery-operating-model.md` | F-001 |
| `internal/tools/` | `docs/design-docs/tools-glossary.md`, `docs/design-docs/guardrails.md` | F-005, F-006, F-007 |
| `internal/trace/` | `docs/design-docs/agent-runtime.md` | F-005 |
| `internal/trust/` | `docs/design-docs/scoring-system.md` | F-008 |
| `internal/ui/` | `docs/design-docs/agent-runtime.md`, `docs/design-docs/dashboard.md` | F-005, F-010 |
| `internal/updatecheck/` | `docs/design-docs/release-versioning.md` | F-004 |
| `internal/validation/` | `docs/design-docs/foundation-operating-model.md`, `docs/design-docs/validation-matrix-gating.md`, `docs/design-docs/agent-smoke-validation.md`, `docs/validation/README.md`, `docs/validation/agent-smoke/README.md` | F-012 |
| `pkg/testutil/` | `docs/design-docs/agent-runtime.md` | F-005 |

Every row also implicitly includes this map. If a file crosses package
boundaries, add the additional docs directly in that file's metadata.

Notable cross-boundary files:

- `internal/bundle/bundle.go` and its focused tests expose the optional target
  DocSync manifest selection, so their metadata also points to the
  documentation-sync architecture and F-019.
- `internal/scanner/init.go` and its tests generate target doctrine, role
  registries, tools guidance, release guidance, and F-001 operating-model
  feature docs, so their metadata lists those extra docs directly.
- `internal/scanner/init.go` and its tests also generate the optional
  integrations example config, so their metadata points directly to
  [board-driven-integrations.md](board-driven-integrations.md) and F-013.
- `internal/serve/server.go`, `internal/serve/executor.go`, and their focused
  tests add board-driven profile visibility, schedule suppression, and future
  tool injection hooks, so their metadata points directly to
  [board-driven-integrations.md](board-driven-integrations.md) and F-013.
- `internal/jira/` owns board-driven JIRA webhook and poll ingestion, scoped
  project/workspace/label containment, ticket materialization by `jira_key`,
  and pull-only reconciliation.
- `internal/mcpclient/` owns short-lived outbound MCP-over-HTTP and MCP-over-stdio
  sessions for optional integration providers such as Atlassian MCP. It is
  client-side integration plumbing, separate from the MARS MCP stdio
  server exposed by `internal/mcpstdio/`.
- `internal/tools/formalized_workflows.go` and its tests own `docsync_audit`,
  so their metadata also points to the delivery operating model and F-001.
- `internal/tools/mars_cli.go` and its tests mirror the CLI command
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
  affected file metadata in the same change. If a built-in deployed root,
  extension, or exclusion changes, update the audit defaults, generated
  manifest, generated target doctrine, F-019, and tests in the same change.
  Target-specific selection belongs in that target's manifest.
- If a code change alters business behavior, update the referenced BDD feature
  contract before claiming the change is complete.
- If a code change alters architecture, generated target behavior, CLI surface,
  or operating doctrine, update the referenced design/product docs before
  claiming the change is complete.
