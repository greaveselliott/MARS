# F-019: TypeScript Monorepo Documentation Sync

- Feature ID: F-019
- Goals: G-DOCSYNC-TS-001, G-001, G-002, G-003
- Status: passing
- Owner: foundation-maintainer with CTO-weekly, Engineer, QA, and Dogfood

## Business Logic

DocSync selects source files through three target-owned manifest fields:
`include_roots`, `include_extensions`, and `exclude_globs`. When a field is
absent or empty, the corresponding built-in default is used. A non-empty field
replaces that field's defaults so target maintainers can define a smaller
audited surface deliberately.

Selection is containment-safe. Roots and globs are repository-relative,
extensions are dot-prefixed file suffixes, and parent traversal or absolute
paths are rejected before traversal. Exclusions apply before file reads and
include safe defaults for dependency, build, coverage, Expo, React Router, and
generated-file output.

The CLI, mirrored tool, file-write policy, and terminal disposition gates use
the same effective DocSync configuration. The audit still proves metadata and
path coverage rather than semantic completeness of the referenced prose.

## Step-By-Step Behavior

1. Resolve the repository root and read optional `.harness/manifest.yaml` DocSync configuration.
2. Fill each absent or empty field from the safe built-in defaults.
3. Normalize, deduplicate, and validate roots, extensions, and globs before walking the repository.
4. Scan root-level matching source files plus matching files below selected roots.
5. Skip default and target-configured excluded paths before reading source.
6. Apply the existing metadata, referenced-document, and foundation expected-document checks.
7. Return one deterministic report shape to package callers, the CLI, mirrored tool, and disposition gates.

## Scenario Schedule

1. F-019-S001 - TypeScript monorepo targets receive configurable, safe, and consistent DocSync coverage.

## Scenarios

### F-019-S001: Configurable TypeScript Monorepo Coverage

Given a deployed harness contains TypeScript or TSX source across nested application, package, worker, or test workspaces
And `.harness/manifest.yaml` either uses the generated DocSync defaults or supplies repository-relative `include_roots`, dot-prefixed `include_extensions`, and `exclude_globs`
When `mars docsync audit --repo .`, the mirrored `docsync_audit` tool, or a successful-disposition gate audits the repository
Then each selected source file is checked for valid near-top `MarsDocSync` metadata and real documentation paths
And dependency, build, coverage, Expo, React Router, generated-file, and explicitly excluded paths are skipped
And absent fields preserve the built-in Go, HTML, CSS, JavaScript, YAML, TypeScript, and TSX coverage defaults
And non-empty fields deliberately replace their corresponding defaults
And unsafe absolute paths, parent traversal, malformed extensions, or malformed glob patterns fail with an actionable error before traversal begins
And the CLI and mirrored tool report the same selected files and findings

## Out of Scope

- Semantic analysis of referenced documentation prose.
- Framework-specific Real-Time Chess behavior.
- Role, orchestration, scoring, or model changes.
- Rewriting existing target-owned manifests during upgrade.

## Descoped Scenarios

None.

## Evidence

- F-019-S001: `go test ./...`; `go test -race ./internal/docsync ./internal/bundle`; `go run ./cmd/mars docsync audit --repo .`; and [the clean generated-target CLI/tool validation](../validation/reports/2026-08-08-typescript-docsync-live-target.md) proving parity, validation, defaults, overrides, and exclusions.
