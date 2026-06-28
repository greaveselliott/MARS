# Active P0 Exec Plan: MARS Rename

**Status:** Active
**Priority:** P0
**Depends On:** None
**Blocks:** Nothing; MARS rename completion evidence is recorded below.
**Related Tickets:** MH-060
**Goals:** G-001; rename the source repo, product surfaces, CLI, module path, generated target doctrine, release/update pipeline, and docs from old Mars Harness naming to MARS.
**BDD Feature:** F-014
**Related Feature Contracts:** F-001, F-002, F-004, F-005, F-009, F-014
**Hypothesis:** MARS can adopt the new product identity and lowercase `mars` CLI without leaving stale old-name references, while preserving deliberate compatibility shims for existing installs, config, metadata, and release markers.
**Success Evidence:** `rg` finds no unclassified `mars-harness`, `Mars Harness`, `mars_harness`, `MARS_HARNESS`, `.mars-harness`, or old module references; targeted rename tests pass; `go test ./...`, docsync/docs consistency, generated target checks, build/smoke, release notes/backfill, and release asset verification either pass or record exact blockers.
**Falsification Evidence:** Any unclassified old-name reference remains; canonical docs or generated target guidance still instruct users to run `mars-harness`; new writes default to `~/.mars-harness`; release/update assets still target `mars-harness`; compatibility aliases are untested; module imports still use `github.com/greaveselliott/mars-harness`.
**Scenario Schedule:** F-014-S001, F-014-S002, F-014-S003, F-014-S004, F-014-S005, F-014-S006, F-014-S007, F-014-S008
**Current Failing Scenario:** None for F-014; F-014-S001 through F-014-S008 completed on 2026-06-28 under MH-060.
**Walking Skeleton Slice:** Rename module, command path, root CLI identity, tool surface, runtime state defaults, release/update paths, generated target defaults, and docs in one coordinated compatibility-aware change.
**Learning Or MVP Outcome:** The repo becomes consistently named MARS, and legacy old-name behavior is explicitly retained only where needed for migration.
**Created:** 2026-06-28
**Owner:** foundation-maintainer
**Source:** Operator request to implement the 9-agent rename plan.

## Primary Outcome

Update all references from old `mars-harness` / `Mars Harness` naming to the new MARS identity, with lowercase `mars` for the CLI, and do so exhaustively.

## Primary Pass Gate

The primary outcome passes only when every remaining old-name string is either absent or recorded in the compatibility allowlist below with a test, parser, or migration reason.

## Subagent Operating Model

Codex main acts as `foundation-maintainer` and Orchestrator/integrator. Nine role-assuming agents cover the work:

- Task Agent 0: plan/tracker validation.
- Task Agent 1: filesystem, module, imports, build paths.
- Task Agent 2: CLI, tool, and environment compatibility.
- Task Agent 3: runtime state, setup, doctor, shell path, DB defaults.
- Task Agent 4: release, update, installer, assets, markers.
- Task Agent 5: generated target defaults and role doctrine.
- Task Agent 6: docs, skills, changelog, historical tracked artifacts.
- Task Agent 7: compatibility fixture coverage.
- Orchestrator Agent: sequencing, evidence gaps, unsupported assumptions, release-risk acceptance.

Subagents audit or validate bounded slices. Main Codex integrates edits and owns final completion evidence.

## Compatibility Allowlist

Old-name strings may remain only in these categories:

- `legacy CLI alias`: accepting or detecting `mars-harness` so older installs and PATH entries fail gracefully or route to `mars`.
- `legacy tool alias`: registering `mars_harness_cli` as an alias of canonical `mars_cli` for older role prompts or traces.
- `legacy env fallback`: reading `MARS_HARNESS_*` when canonical `MARS_*` is absent.
- `legacy state fallback`: reading existing `~/.mars-harness` config, DB, model, trace, and binary paths during migration while new writes default to `~/.mars`.
- `legacy release marker`: parsing old `mars-harness-release` changelog markers while writing new `mars-release` markers.
- `legacy generated metadata`: reading old `generator: mars-harness` target metadata while writing `generator: mars`.
- `legacy integration marker`: parsing old machine markers such as JIRA mirror markers where existing target docs may contain them, while writing new markers.
- `historical Mars monorepo`: references to the separate precursor Mars monorepo when the URL or context proves it is not the old product name.

## Validation Gates

- Old-name grep: `rg -n "mars-harness|Mars Harness|mars_harness|MARS_HARNESS|\\.mars-harness|github.com/greaveselliott/mars-harness" --glob '!dist/**'`
- Old-path grep: `git ls-files | rg "mars-harness|mars_harness|MARS_HARNESS"`
- Targeted tests: `go test ./cmd/mars ./internal/tools ./internal/config ./internal/shellpath ./internal/selfupdate ./internal/release ./internal/scanner ./internal/operatingmodel ./internal/codeintel ./internal/githubauth ./internal/setup ./internal/doctor`
- Broad tests: `go test ./...`
- Build smoke: `go build ./cmd/mars`
- CLI smoke: `go run ./cmd/mars version`
- Generated target smoke: `go run ./cmd/mars init --repo <clean-temp-target>`
- Docs gates: docsync/docs-consistency tests and `mars docsync audit --repo .` when available.
- Release gates: `mars release notes --repo . --bump auto`, `mars release backfill-notes --repo . --check`, tag, publish assets, and verify assets, or explicit blocker.

## Current Evidence

- Remote trunk was fetched on 2026-06-28 and local `main` matched `origin/main` before edits.
- Baseline targeted tests passed before rename for CLI/tool/config/release/scanner/update surfaces.
- After the first mechanical rename slice, `go test ./cmd/mars ./internal/tools ./internal/config ./internal/shellpath` passed.
- Compatibility-focused gates passed:
  - `GOCACHE=/private/tmp/mars-go-cache go test ./internal/tools -run 'TestMarsCLI|TestDefaultRegistry_includesMarsCLI|TestShellExecPolicyBlocksMarsBinary'`
  - `GOCACHE=/private/tmp/mars-go-cache go test ./internal/jira -run 'TestWebhookRequiresSignatureAndMirrorsMappedIssue|TestWebhookAcceptsLegacySignatureHeader|TestWebhookUsesMappedRepoSecret'`
  - `GOCACHE=/private/tmp/mars-go-cache go test ./internal/scanner -run 'TestReadHarnessMetadataAcceptsLegacyGenerator|TestInit_success'`
  - `GOCACHE=/private/tmp/mars-go-cache go test ./internal/config ./internal/shellpath ./internal/release ./internal/selfupdate ./internal/codeintel ./internal/operatingmodel ./internal/sandbox`
- Broad gate passed with local-listener/process-cleanup access: `GOCACHE=/private/tmp/mars-go-cache go test ./...`.
- Build smoke passed: `GOCACHE=/private/tmp/mars-go-cache go build -o /private/tmp/mars-rename-smoke ./cmd/mars`.
- CLI smoke passed: `/private/tmp/mars-rename-smoke version` printed `mars 0.65.9 darwin/arm64 commit=unknown built=unknown`.
- Generated target smoke passed: `/private/tmp/mars-rename-smoke init --repo /private/tmp/mars-rename-target`, followed by an old-name grep with zero hits.
- Docs sync passed: `/private/tmp/mars-rename-smoke docsync audit --repo .`.
- Full source gate passed with local-listener/process-cleanup access: `GOCACHE=/private/tmp/mars-go-cache make check`; coverage ratchets passed at total 73.6%, `internal/release` 67.0%, and `internal/serve` 66.3%.
- Old-name grep is classified to this plan/feature/ticket plus compatibility code/tests for old envs, state paths, tool alias, binary alias, release assets/markers, shell markers, JIRA signature header, and generated metadata.
- Semantic rename commit `419a5a3 feat(rename): adopt MARS identity` was pushed to `origin/main` on 2026-06-28.
- Release commit `f200f68 release: notes 0.66.0` was pushed to `origin/main` on 2026-06-28.
- Tag `v0.66.0` was pushed on 2026-06-28.
- Release asset publication passed on 2026-06-28: `GOCACHE=/private/tmp/mars-go-cache go run ./cmd/mars release publish-assets --repo . --version v0.66.0 --upload auto`.
- Local release asset verification passed on 2026-06-28: `GOCACHE=/private/tmp/mars-go-cache go run ./cmd/mars release verify-assets --dist dist/releases --version v0.66.0`.
- GitHub release verification passed on 2026-06-28: `gh release view v0.66.0 --repo greaveselliott/MARS`, showing canonical `mars-*`, legacy `mars-harness-*`, and `checksums.txt` assets.

## Residual Risks

- Full lifecycle clean-project validation beyond generated-target smoke is not run in this slice; the exact supporting smoke evidence above is recorded, and any broader lifecycle claim remains unconfirmed until a matrix replay is scheduled.
