# Active P0 Exec Plan: Example Target Project Ways Of Working

**Status:** Active
**Priority:** P0
**Depends On:** None
**Blocks:** Board-driven Example Target Project integration claims until each sequential plan is implemented, validated, released, pushed, tagged, and asset-verified
**Related Tickets:** T-044, T-045
**Goals:** G-001, G-002, G-003, G-004
**BDD Feature:** F-013
**Related Feature Contracts:** F-001, F-013
**Hypothesis:** A default-off integrations layer can let Mars Harness adopt Example Target Project's JIRA-board-driven workflow while preserving the current CEO-led/GitHub/trunk path for every repo that does not opt in.
**Success Evidence:** No-config repos keep the 2026-06-23 baseline routes, scheduler behavior, effective role tool lists, strict-trunk policy, and CEO-led planning. Board-driven repos can progressively mirror JIRA tickets, select ready work by active sprint -> P1/P2/P3 priority -> LexoRank -> age, scope through `cto-weekly`, deliver through Engineer/QA, and open human-merged PRs.
**Falsification Evidence:** Missing config changes runtime behavior, board-driven behavior is inferred from partial config, JIRA events enqueue LLM work directly, future tools appear in generated default manifests, branch/PR delivery leaks into strict-trunk default mode, credentials leak, or clean-project validation cannot distinguish real idle from misconfiguration.
**Scenario Schedule:** F-013-S001, F-013-S002, F-013-S003, F-013-S004, F-013-S005, F-013-S006
**Current Failing Scenario:** F-013-S002 JIRA mirror and sync model. F-013-S001 Optionality foundation is complete and recorded in `docs/validation/reports/2026-06-23-example-target-project-optionality-foundation.md`.
**Walking Skeleton Slice:** Start Plan 2 only after T-045 is selected: JIRA webhook/poll ingestion, ticket materialization by `jira_key`, and safe reconciliation of JIRA-owned fields while preserving harness-owned lifecycle and evidence. Do not implement prioritisation, Figma, PR delivery, or frontier model routing in this slice.
**Learning Or MVP Outcome:** Establish a reversible, observable, default-off substrate that every later Example Target Project plan can import without changing existing target behavior.
**Created:** 2026-06-23
**Owner:** foundation-maintainer
**Source:** Operator-approved Example Target Project Ways Of Working program executed by Codex with Mars Harness role subagents.

## Primary Outcome

Add an optional board-driven Example Target Project-compatible workflow to Mars Harness:

JIRA board intake -> CTO scoping -> Engineer -> QA -> PR, with frontier gateway models and Figma context, while preserving the 2026-06-23 CEO-led/GitHub/trunk flow byte-identically when `.harness/integrations.yaml` is absent.

The external board is an intake signal. Repository artifacts remain authoritative for goals, feature contracts, tickets, implementation, validation, commits, release notes, and blockers.

## Subagent Operating Model

Codex main acts as `foundation-maintainer` and Orchestrator. Subagents assume existing Mars personas:

- COO owns active exec-plan and F-013 contract wording.
- CTO-weekly owns first-slice technical ticketing.
- Engineer owns one ticket implementation with tests, docs, and MarsDocSync.
- QA owns acceptance review and docsync/test evidence.
- Security owns secret handling, default-off behavior, and blast-radius review.
- Dogfood owns installed-binary clean-project validation and matrix reports.
- Release Manager owns release notes, backfill, tags, assets, verification, and blockers.

Subagents may run in parallel only with disjoint write scopes. Main Codex integrates outputs, resolves conflicts, commits to `main`, pushes, and owns final claims.

## Six Sequential Plans

### Plan 1: Optionality Foundation

Add `internal/integrations`, `flow_profile`, `.harness/integrations.example.yaml`, runtime section gates, profile status/logging, warm-restart schedule rebuild, and effective-tool injection hooks.

No behavior changes without config. No JIRA route, poller, prioritisation, Figma tool, PR tool, or model gateway routing ships in this plan.

### Plan 2: JIRA Mirror And Sync Model

Add `internal/jira` webhook and poll ingestion. JIRA stays the source of record. The first sighting of a `jira_key` materializes a backlog Markdown ticket; later pulls reconcile JIRA-owned fields and requirements body in place while preserving harness-owned lifecycle directory, delivery evidence fields, scoped marker, and agent progress notes.

JIRA events do not use the GitHub webhook spine, do not register `jira_issue.*` triggers, and do not enqueue LLM work per event.

### Plan 3: Board Prioritisation And Cost Guards

Extend ticket state with `jira_key`, `jira_updated`, `jira_created`, `sprint`, `sprint_active`, `rank`, `jira_status`, and `epic`; reuse existing `priority` and `blocked_by`.

The orchestrator survey selects the top ready backlog ticket by active sprint -> configured priority order -> LexoRank -> age, skipping unresolved blockers, non-ready statuses, and closed-sprint tickets, then dispatches `cto-weekly` for scoping.

This plan also ships DailyCap, interval floors, cost/turn telemetry, circuit breaker behavior, and dashboard/log operator signals so paid board dispatch cannot run unguarded.

### Plan 4: Frontier Model Parity

Wire `models.ResolveModelOverride` into executor, extend `model-overrides.yaml` with `api_key_env`, pass Bearer tokens without logging values, and populate fallback endpoint key from env. Examples stay vendor-neutral; Example Target Project model names and endpoints are not hardcoded.

### Plan 5: Figma And PR Delivery

Add a registry injection seam, bounded `figma_fetch`, and `github_pr_open`. PR mode relaxes push policy only for sanitized branches matching `delivery.branch_pattern` under `delivery.mode: pull_request`; force-push is forbidden, base must be `main`, one PR is allowed per JIRA key, and auto-merge is out of scope.

Trust gating has two layers: the existing mutating-tool observer block plus an explicit `delivery.min_trust` rank check.

### Plan 6: Context Mirroring And Traceability

Generate deployed-harness JIRA operating-model documentation, knowledge route, and AGENTS section. The generated docs describe edit-in-JIRA -> pull -> local MD updated, the JIRA workflow status versus Mars lifecycle directory axes, and the one-way/write-back-deferred boundary.

Release notes surface `jira_key` in CHANGELOG entries beside Mars ticket IDs.

## Shared Config Surface

`.harness/integrations.yaml` is optional. Missing config or `flow_profile: ceo-led` means current behavior.

The v1 schema contains:

- `flow_profile: ceo-led | board-driven`
- `ingestion.jira`: enabled, base URL, env-var names, webhook secret env, poll interval, JQL, project-to-repo map, custom field IDs, ready statuses, priority/rank/age ordering, blocked-by handling
- `design_sources.figma`: enabled, token env, base URL
- `delivery`: `trunk | pull_request`, branch pattern, minimum trust

Model routing remains in `.harness/model-overrides.yaml`. Config stores env-var names only, never secret values. Example Target Project field IDs may appear only as commented examples in `.harness/integrations.example.yaml`, never as Go constants.

## Plan 1 Implementation Detail

- Loader mirrors `LoadModelOverrides`: missing file returns disabled config, version defaults to `1`, unknown fields are tolerated, and unknown or empty `flow_profile` fails safe to `ceo-led`.
- Startup and warm restart reload integrations config.
- Scheduler supports replacing registered schedules on restart; append-only registration would leave stale cron entries.
- Under `board-driven`, suppress schedules only for `ceo`, `coo`, `head-of-strategy`, and `cto-weekly`; role availability remains.
- Effective tool allowlists are computed late in executor. No-config/CEO-led lists are byte-identical. Future tools are appended only when profile and section gates allow and the registry contains them.
- Status, dashboard APIs, and logs surface active profile so board-driven stalls are operator-visible.
- Init and upgrade write `.harness/integrations.example.yaml` only.

## Risks And Guards

- Optionality regression: no-config invariant tests assert no JIRA route/poller, strict trunk, unchanged schedules, and unchanged effective tools.
- Runaway frontier spend: Plan 3 ships cost guards with board dispatch, not later.
- Wrong ticket picked: selector tests cover active sprint, P1/P2/P3, unknown priority last, LexoRank, age, blockers, and ready statuses.
- Sync clobbering: reconcile tests preserve lifecycle directory, evidence fields, scoped marker, and agent notes byte-for-byte.
- Secret leakage: config holds env-var names only; redaction tests ensure no token or signed URL appears in logs/traces.
- Push-policy creep: strict-trunk tests remain green; PR branch allow-cases are gated by delivery mode.
- Wrong repo ingestion: explicit `project_repo_map` is required; unmapped or ambiguous projects drop with a log and never fan out.

## Current Ticket

`T-044` completed F-013-S001. `T-045` is the first Plan 2 ticket for F-013-S002 and must be scoped before any Plan 2 implementation code starts.

## Plan 1 Evidence Snapshot

F-013-S001 is complete as of 2026-06-23. Focused Plan 1 tests pass for
`internal/integrations`, `internal/scheduler`, targeted `internal/serve`,
targeted `internal/scanner`, `internal/docsync`, and
`internal/docsconsistency`. Broad `go test ./...`, `make check`, GitHub auth,
`make install`, and release backfill all pass.

Hosted-model clean-project validation ran through a real OpenAI-compatible
endpoint. The operator provided `OPENAI_API_KEY` through `launchctl`, Codex did
not print the key, and a temporary local proxy mapped harness role model names
to `gpt-4.1-mini`. Static browser and Go API clean targets both generated
`.harness/integrations.example.yaml`, did not write
`.harness/integrations.yaml`, loaded `flow_profile="ceo-led"`, replaced 8 cron
schedules, completed CEO -> COO -> CTO -> Engineer through the installed
binary, and produced product validation evidence without local GGUF models.

Plan 2 may start only from T-045. No JIRA, Figma, PR, board-prioritisation, or
frontier model-routing implementation code shipped in Plan 1.

## Validation And Release

Each plan independently requires targeted tests, `go test ./...`, `make check`, docsync/docs consistency, installed-binary clean-project validation, and a matrix report under `docs/validation/reports/`.

AD-284 minimum for Plan 1 is orchestration plus generated defaults: static browser plus CLI/tooling or API/service, or an explicit blocker with replay command.

Each semantic commit is followed by `mars-harness release notes --repo . --bump auto`, `mars-harness release backfill-notes --repo . --check`, a `release: notes X.Y.Z` commit, push, tag `vX.Y.Z`, `mars-harness release publish-assets --repo . --version vX.Y.Z --upload auto`, and `mars-harness release verify-assets --dist dist/releases --version vX.Y.Z`. Blockers are recorded explicitly.
