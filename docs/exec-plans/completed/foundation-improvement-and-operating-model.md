# Foundation Improvement And Operating Model (Completed)

**Status:** Completed
**Priority:** P0
**Depends On:** None
**Blocks:** Plan promotions until this file names the next slice
**Related Tickets:** MH-034, MH-035, MH-037, MH-031, MH-030, MH-038, MH-039, MH-040, MH-041, MH-042, MH-043, MH-044, MH-045, MH-046, MH-047, MH-048, MH-049, MH-050, MH-051, MH-052, MH-053, MH-054, MH-055, MH-056, MH-057, MH-058, MH-059, MH-060, MH-061, T-001, T-002, T-003, T-004, T-005, T-006, T-009, T-010, T-011, T-012, T-013, T-014, T-015, T-016, T-019, T-020, T-021, T-022, T-023
**Goals:** G-001, G-002, G-003, G-004
**BDD Feature:** F-001, F-004, F-005, F-006, F-007, F-009, F-010, F-012
**Hypothesis:** Treating factory pace as measured intervention debt, using the `demo-123` replay series as concrete evidence, will reduce avoidable turns without hiding productive long-running work.
**Success Evidence:** The 2026-05-20 `demo-123-run11` replay reached product planning, ticketing, Engineer completion, QA, Security, Dogfood, and local release notes with zero intervention-debt tickets; release-blocked publication stopped dispatch without remote mutation or a Dogfood loop. The 2026-05-20 `demo-api-run1`, `demo-api-run2`, `demo-api-run4`, `demo-api-run5`, `demo-api-run6`, `demo-api-run7`, `demo-api-run8`, `demo-api-run9`, `demo-api-run10`, `demo-api-run11`, `demo-api-run12`, `demo-api-run13`, `demo-api-run14`, `demo-api-run15`, `demo-api-run16`, `demo-api-run17`, `demo-api-run18`, `demo-api-run19`, and `demo-api-run20` non-static replays reached product-specific planning and ticketing without intervention-debt ticket amplification; `demo-api-run9` reached local release notes, `demo-api-run11` reached product implementation, QA, Security, Dogfood validation evidence, and quality-score export, `demo-api-run13` confirmed tracked background kill cleanup lets the `/tmp` validation binary start without manual port cleanup, `demo-api-run14` confirmed no-op shell guidance lets Engineer complete the ticket lifecycle, `demo-api-run15` confirmed runtime failures stay quarantined when Engineer fails before review, `demo-api-run16` confirmed source write DocSync metadata can be present at first write, `demo-api-run17` confirmed the handoff reaches Engineer but shell work must be mechanically claim-first, `demo-api-run18` confirmed claim-first shell policy plus source DocSync preflight recovery in the live implementation path, `demo-api-run19` confirmed managed background API validation, ticket completion, QA approval, quality-score export, and zero target intervention-debt tickets, and `demo-api-run20` confirmed bounded Security terminal evidence, Dogfood validation, target release-note generation, overall quality grade `A`, and zero target intervention-debt tickets. The 2026-05-20 `demo-cli-run1` Note Stats CLI replay confirmed product-specific planning, ticketing, implementation, QA, external build-output recovery, and zero intervention-debt tickets for a CLI archetype. The 2026-05-20 `demo-cli-run2` replay confirmed Security no longer mutates product code, root scratch probes are blocked, target quality export reaches grade `B`, and runtime failures remain foundation telemetry with zero target intervention-debt tickets. The 2026-05-20 `demo-cli-run3` replay confirmed the patched generated guidance, claim-first implementation start, root scratch-probe blocking, and max-turn containment with zero target intervention-debt tickets. The 2026-05-20 `demo-cli-run4` replay confirmed contract-first implementation fixed the empty-text semantic drift and produced a committed product slice before the next closure blocker. The 2026-05-20 `demo-cli-run5` replay confirmed closure-before-packaging moved `T-001` to done and reached QA/Security/Dogfood without repo-local packaging artifacts before exposing Dogfood finding-handoff drift. The 2026-05-20 `demo-cli-run6` replay confirmed Dogfood creates one committed finding and Orchestrator routes Engineer rework without intervention-debt amplification; Engineer fixed the target tests, while QA/Security validation approval and remediation-ticket closure became the next source gates.
**Falsification Evidence:** Pace remains unmeasured, max-turn limits are raised blindly, future clean target replays still route autonomous follow-up after a terminal release blocker, or Dogfood/Engineer tool recovery prevents useful validation from reaching a terminal outcome.
**Scenario Schedule:** F-012-S010, F-001-S015, F-004-S007, F-012-S006, F-012-S007, F-009-S013, F-005-S010, F-005-S011, F-005-S016, F-005-S035, F-006-S018, F-006-S019, F-006-S020, F-006-S021, F-006-S022, F-006-S042, F-007-S014
**Current Failing Scenario:** As of 2026-05-21, `demo-temp-run56` confirms AD-209 moved the clean CLI lifecycle beyond the prior Engineer `max_turns` blocker: product implementation, QA-requested test rework, Engineer test addition, ticket completion, and a second QA handoff all occurred. The remaining live blocker is review terminal convergence firing before QA can run required `docsync_audit` evidence, causing a `circle_detected` failure after successful tests. The next check is AD-210 docsync-aware review terminal convergence in a clean CLI canary.
**Walking Skeleton Slice:** Expand the live validation matrix across distinct target archetypes, record pace and guardrail-tax deltas for each, and only promote fixes that repeat across targets or protect a clear foundation invariant.
**Learning Or MVP Outcome:** Future agents inherit the foundation/deployed architecture decision, generated target mirror, drift review, skill/tool decision, and a refreshed path back to runtime remediation work.
**Created:** 2026-05-02
**Owner:** Mars Harness maintainers
**Source:** Exec-plan review and repository state audit on 2026-05-02

## Purpose

This is the current execution map for Mars Harness. It exists because the
original master plan and delivery schedule are historical baseline
documents: useful for lineage, but stale as a status source.

Future agents should use this file, the ticket tree, and prioritized backlog
plans to decide what to do next.

## Current Truth

- Current source version is recorded in `VERSION`.
- Current branch: `main`
- Active goals live in `docs/goals/active.md`; the current plan references `G-001`, `G-002`, `G-003`, and `G-004`.
- BDD feature contracts live in `docs/features/`; the current operating-model feature is `F-001`, target-harness mirroring is `F-004`, release publication discipline is `F-009`, and feedback/self-improvement routing is `F-012`.
- Ticket state:
  - `docs/tickets/in-progress/` is empty.
  - `docs/tickets/backlog/` contains `T-010`, `T-013`, `MH-051` through
    `MH-061`, and any newly created live-loop follow-up tickets.
  - `docs/tickets/done/` contains `MH-001` through `MH-050`, `T-001`
    through `T-043`, and `T-011`.
- Exec-plan state:
  - `docs/exec-plans/active/` contains exactly one active plan: this file.
  - `docs/exec-plans/backlog/` contains prioritized waiting plans with dependencies and blockers.
  - `docs/exec-plans/superseded/` contains historical plans that must not drive current work.
- GitHub release notes are published for semantic versions generated from `VERSION`.
- Release binary assets are published for `v0.14.5`; `MH-031` is done.
- The per-version release publication and asset-verification blocker ledger
  (`v0.21.0` through `v0.42.20`) was extracted on 2026-06-11 to
  [docs/validation/release-blockers.md](../../validation/release-blockers.md)
  under `T-022`; as of 2026-05-21 the most recent releases (`v0.42.19`,
  `v0.42.20`) published full binary assets after the GitHub Actions budget
  was restored.
- Model evaluation, Ollama catalog support, model overrides, persisted reports,
  repo-backed benchmark cases, and promotion blocking shipped under `MH-030`.

## Plan State

| Plan | State | Depends On | Blocks | How to use it |
| --- | --- | --- | --- | --- |
| `active/current-operating-plan.md` | Active, P0 | None | Plan promotions until this file names the next slice | Use this file as the only top-level execution map and scenario schedule. |
| `backlog/mars-parity-supersession-plan.md` | Backlog, P1, G-001/G-002/F-001 | Ticket materialization completed on 2026-05-03; no open blocker | Supersession readiness claims | Pull slices into tickets and this active plan before execution. |
| `backlog/tanstack-dashboard-control-plane.md` | Backlog, P2, G-001/G-003/F-010 | `T-011` closed 2026-06-12; AD-279 start condition satisfied — promote via active plan when prioritized | Next-generation dashboard implementation claims | Promote from active plan task 3 when TD-009/T-013 bounded or operator reprioritizes; start with `MH-051`. |
| `backlog/model-evaluation-refresh-plan.md` | Completed initial model-refresh slice; future P4, G-003/F-001 | Higher-priority release/quality work for any future promotion ticket | Default model registry promotion until live benchmark evidence exists | Use only to seed future model-refresh tickets. |
| `superseded/master-execution-plan.md` | Superseded | None | Nothing | Historical baseline. Do not use its checkbox status as truth. |
| `superseded/delivery-schedule.md` | Superseded | None | Nothing | Historical milestone schedule; kept for lineage only. |

## Foundation Improvement Workstreams

Registered 2026-06-11 from the foundation improvement review. This is the only
active plan; the workstreams below are amendments to it, not a second plan.
Ticket IDs `T-019` through `T-023` are created; `T-024` and later IDs are
provisional until `ticket_create` assigns them when each workstream starts.

| Workstream | Scope | Tickets |
| --- | --- | --- |
| WS-A Documentation hygiene | Retire/refresh stale durable docs, quality-score cadence, slim this plan | `T-019`, `T-020`, `T-021`, `T-022` |
| WS-B Decision gates | Dashboard architecture AD and schedule-or-defer decision | `T-023` |
| WS-C Production-grade gates | Coverage ratchet (`T-024` done 2026-06-11, AD-280), govulncheck + fuzz (`T-025` done 2026-06-11, AD-281), self-verifying release pipeline (`T-026` done 2026-06-11, AD-282, `release audit`), pace/convergence telemetry (`T-027` done 2026-06-11, AD-283, Convergence And Guardrails export) | `T-024`, `T-025`, `T-026`, `T-027` |
| WS-D Convergence consolidation | Convergence state-machine AD and incremental rule-cluster migration, extending `T-011` | provisional (next IDs from `ticket_create`; originally drafted as `T-028`, `T-029`) |
| WS-E God-file decomposition | Policy-domain decomposition AD and per-domain extractions (`T-030`–`T-043` done 2026-06-12; AD-287 sequence complete) | `T-030`–`T-043` |
| WS-F Validation matrix discipline | Matrix-gating AD (`T-028` done 2026-06-11, AD-284/AD-285, provisional `T-040`) and archetype-gap baseline replays | `T-028`, provisional `T-041` |

Sequencing recorded 2026-06-11: WS-A and WS-B land first (docs/doctrine only,
no replay tax). WS-C and WS-F follow, then the measurement floor (`T-027`
telemetry plus the `T-011` dated pace baseline replay) must exist before any
WS-D or WS-E slice, because each slice is judged by a before/after replay
comparison against that baseline per the AD-138 loop.

Phase 3 sequencing amendment recorded 2026-06-12, after the measurement floor
landed (baseline: `docs/validation/baselines/2026-06-12-factory-pace-baseline.md`;
design ADs: AD-286 convergence state machine, AD-287 policy decomposition):
**`T-032` (engineer coding-tier context overflow, P1) is the first Phase 3
implementation slice.** Rationale: the matrix-gating table (AD-284) requires
package-managed-frontend and existing-repo-maintenance archetype replays as
confirmatory evidence for the later WS-D and WS-E slices, and as of
2026-06-12 both archetypes wedge deterministically on the same context
overflow (demo-12 and demo-13 baselines) — until T-032 lands, two of the
three matrix rows cannot
produce the before/after replay evidence those slices are judged by.
**T-032 closed 2026-06-12** (AD-288, v0.50.11): both archetype replays
passed with zero context_overflow and lifecycle reach beyond the wedged
baselines; the replay runs are the new demo-12/demo-13 archetype pace
baselines (see the factory-pace baseline's archetype table).
**T-031 closed 2026-06-12** (AD-289, v0.50.13/v0.50.14): convergence
failures get one automatic same-role retry per failure fingerprint, then a
recorded `blocked/operator_retry` escalation; the 2026-06-12 demo-14
Inventory/API replay drained its full backlog (T-001–T-010 done) with
zero operator `run-role` calls and is the archetype lifecycle-reach
reference. The T-035/post-max_turns siblings mapped in AD-286 remain
separate edges (confirmed distinct code paths 2026-06-12). T-029-style
extraction and state-machine slices follow in the AD-287 extraction order,
with dogfood review convergence (6 failures in 10 demo-14 dogfood jobs) as
the dominant remaining convergence consumer per the demo-14 report.
**T-043 closed 2026-06-12** (AD-287 final slice, v0.50.24): validation-lane
extraction plus two-archetype final checkpoint PASS (demo-12 Run 4, demo-15
Run 2). **AD-287 extraction sequence complete.**
**T-030 closed 2026-06-12:** ticket_create title dedupe requires true keyword
subset; demo-11 canary PASS (cto-weekly 2m 1s, 0 DUPLICATE). **T-011 closed
2026-06-12:** max-turn calibration from balanced-model baseline (engineer
100, qa 40, security 30) plus cumulative pace stack (T-030/T-031/AD-288).
**WS-D slice 5 landed 2026-06-12:** `DeliveryPhaseValidated` derivation plus
post-validation shell guards via `engineerInValidatedPhase()`. **WS-D slice 4
landed 2026-06-12:** file_write repair-lane guard and engineer disposition
validation-failed gates migrated to `EngineerDeliveryState()`. **WS-D slice 6
landed 2026-06-13:** browser-framework completion blockers consult
`engineerInValidatedBrowserCompletionPhase()`. **WS-D slice 7 landed
2026-06-13:** `EngineerDeliveryState()` derives committing, evidence-recording,
and closing post-validation phases. **WS-D slice 8 landed 2026-06-13:**
`ReviewDeliveryState()` plus review terminal/no-op guards consult
`reviewerInValidatedPhase()` / `reviewerInValidationFailedPhase()`. **CTO
ticket-gate loop fix landed 2026-06-13:** when CTO completes with an
implementation handoff but no open product ticket exists, dispatch advances to
QA when ordinary product tickets are already done or escalates to COO instead
of repeating CTO (demo-14 invalid canary closed as checkpoint-seed mismatch).
**WS-D AD-286 migration queue complete** — remaining work is plan closure and
optional dashboard epic (`MH-051`..`061`). **Foundation operating model
landed 2026-06-13** ([foundation-operating-model.md](../../design-docs/foundation-operating-model.md),
AD-291/AD-292): foundation runtime fixes require clean-project harness replay,
not unit tests alone.

## Next slice (plan closure)

1. [x] **`make install`** then run AD-284 batched validation for WS-D slices
   6–8 + AD-290/AD-291 on **fresh ephemeral targets** (AD-293):
   `node scripts/validation-target.mjs create --profile static-browser-todo --label wsd-closure`
   and
   `create --profile depot-supplies-api --label wsd-closure`
   per [foundation-operating-model.md](../../design-docs/foundation-operating-model.md);
   append evidence to
   `docs/validation/reports/2026-06-13-foundation-wsd-closure-replay.md`.
   Discard runs when recorded (`validation-target.mjs discard`).
2. [x] Record pass/fail against AD-284/AD-285; Run 2 blocked on COO max_turns —
   recorded with retry command and **TD-009**; Run 1 PASS; supplementary demo-14
   API evidence cited.
3. [x] Move this plan to `docs/exec-plans/completed/` (2026-06-13).

## Completion (2026-06-13)

Foundation improvement workstreams WS-A through WS-F and WS-D/WS-E landed.
Closure replay: static-browser PASS; api-service ephemeral BLOCKED (TD-009).
See [2026-06-13-foundation-wsd-closure-replay.md](../../validation/reports/2026-06-13-foundation-wsd-closure-replay.md).
Successor active plan promotes factory-pace follow-up and optional dashboard epic.

## Current Priority Order

1. **Factory pace intervention debt (`T-011`/`T-013`)**: Continue measuring role
   pace from durable traces and job outcomes, define target thresholds, and
   implement the smallest evidence-backed change that reduces avoidable turns.
   The 2026-05-20 runs 8-12 show shallow ticket globs, broad `find .`, empty
   `shell_exec`, and static app serving discovery. The `demo-api-run1`
   non-static replay adds a more general turn sink: scheduled same-repo
   same-role work stacked behind an active Engineer. The 2026-05-20 scheduler
   slice fixed that duplication. The `demo-api-run2` rerun found the next
   generic turn sink: repo-local compiled binaries block blast-radius recovery
   and cleanup. The `demo-api-run3` rerun then found a prior generic planning
   ambiguity: CEO/COO guidance still allowed duplicate `F-001` paths and
   duplicate starter scenarios, so the current `T-011` slice tightens
   canonical feature-contract reuse before the next API replay. Later reruns
   confirmed module-named artifact cleanup and cleanup hints. `demo-api-run6`
   moved the bottleneck to managed long-running server validation, and
   `demo-api-run7` confirmed that fix before exposing repo-local validation
   binary prevention. `demo-api-run8` confirmed binary prevention and moved the
   current slice to malformed bare-port command recovery.
2. **Hygiene and decision gates (WS-A `T-019`-`T-022`, WS-B `T-023`)**:
   Registered 2026-06-11. Small docs/doctrine commits that retire stale
   artifacts, define the quality-score regeneration cadence, slim this plan,
   and resolve the dashboard architecture decision. No factory runtime
   behavior changes, so no replay tax.
3. **Production gates and measurement floor (WS-C, WS-F, `T-011` baseline)**:
   Coverage ratchet, govulncheck plus fuzz targets, self-verifying release
   pipeline, matrix-gating doctrine, then pace/convergence telemetry and the
   `T-011` dated pace baseline replay. These protect and precede the refactor
   tracks.
4. **Representative live validation matrix**: Apply AD-138 by checking at least
   one non-static project archetype before claiming broad factory progress.
   Keep `demo-123` as the static canary, but judge generic changes against the
   small matrix from `docs/design-docs/delivery-operating-model.md`.
5. **Live lifecycle replay**: Run a clean target lifecycle and
   record product progress, queue health, intervention-debt count, quality
   export behavior, release-note behavior, stop/shutdown behavior, and whether
   Dogfood reaches a terminal disposition without dirty watchdog routing.
6. **Convergence consolidation and god-file decomposition (WS-D, WS-E)**:
   The measurement floor, convergence state-machine AD (AD-286), and
   decomposition AD (AD-287) landed 2026-06-12. Implementation order per the
   Phase 3 sequencing amendment above: `T-032` context overflow first (it
   unblocks the demo-12/demo-13 matrix replays the other slices need —
   **done 2026-06-12**, AD-288/v0.50.11, both archetype replays passed),
   then the `T-031` routing fix (**done 2026-06-12**, AD-289/v0.50.13,
   demo-14 replay passed with zero operator retries),
   then one rule cluster or one policy domain per
   slice with a canary replay and recorded pace delta per the AD-138 loop.
   **WS-E complete 2026-06-12** (`T-043`, AD-287); **WS-D is the active
   refactor track** starting with AD-286 state-machine clusters.
7. **Dashboard control-plane backlog (`MH-051` through `MH-061`)**: Decided
   2026-06-11 under `T-023` (WS-B): the epic is **deferred until `T-011`
   closes**, per AD-279. AD-279 also scopes the single-binary constraint to
   the core runtime and absorbs the `T-010` restyle into the epic. Start
   condition: promote `backlog/tanstack-dashboard-control-plane.md` when
   `T-011` reaches done (or explicit operator reprioritization), beginning
   with `MH-051`.
8. **Mars parity continuation**: Use
   [OpenHarness comparator](../../references/openharness-comparator.md)
   as reference input for readiness, skill metadata, compaction, and
   remediation ergonomics without creating a parallel roadmap.

## Scenario Schedule

| Scenario | Status | Evidence |
| --- | --- | --- |
| F-001-S001 | Passing | `docs/goals/active.md`, `docs/features/F-001-delivery-operating-model.md`, and this active plan are linked. |
| F-001-S002 | Passing | Engineer ticket gate validates feature ticket evidence before done. |
| F-001-S003 | Passing | `mars-harness init` emits goals, feature contracts, AD-074, role guidance, ticket metadata, exec-plan metadata, quality guidance, and knowledge routes. |
| F-001-S004 | Passing | `update check` and `doctor --repo` report operating-model drift without overwriting user-owned docs. |
| F-001-S005 | Passing | Ticket docs, quality score, and release-note generation distinguish shipped feature scenarios from enabler work when commits reference tickets. |
| F-001-S006 | Passing | Telemetry proposals can create or update active goals/observations with dedupe evidence. |
| F-001-S007 | Passing | `go test ./internal/docsconsistency -run TestFeatureContractsDeclareRequiredFields` checks feature contracts include first-class business-logic sections. |
| F-001-S008 | Passing | `go test ./internal/docsconsistency -run TestOperatingModelCodeFilesDeclareDocSyncMetadata` checks operating-model code files carry associated documentation metadata. |
| F-001-S009 | Passing | `go test ./internal/docsync ./internal/docsconsistency`, `go test ./...`, and the 2026-05-20 focused run-12 replay `go run ./cmd/mars-harness tools run docsync_audit --repo <validation-root> --args-json '{}'` verify source-wide docsync audits deployed `src/` app roots and parses compact inline static metadata. |
| F-012-S010 | Passing | `go test ./internal/remediation`, `go test ./internal/serve -run 'TestHandleJobFailed(RecordsDeterministicRemediation|ExecutesGeneratedDocs)|TestHandleRemediation(ExecutableReadyRecipe|AutoSafeWithoutExecutor|OperatorRecipe)'`, `go test ./internal/doctor -run TestCheckDeterministicRemediationHealth`, `go test ./internal/qualityscore -run TestExportRendersTelemetryAndOutcomeSignals`, `go test ./internal/docsconsistency ./internal/docsync`, `go test ./...`, and the 2026-05-19 clean `<validation-root>` replay cover the completed `MH-048` edge slice. |
| F-001-S015 | Passing | [foundation-deployed-harness-architecture.md](../../design-docs/foundation-deployed-harness-architecture.md) records the foundation/deployed doctrine boundary, feedback routing, tool/skill/runtime split, generated-target implications, doctrine-maintenance duties, and 2026-05-19 drift review. |
| F-004-S007 | Passing | `go test ./internal/scanner -run TestInit_success` verifies generated targets receive the foundation/deployed route and AD-139 core doctrine without source binary asset names. |
| F-012-S006 | Passing | [skill-evolution.md](../../design-docs/skill-evolution.md) AD-140 keeps the recursive improvement loop as operating doctrine and creates `T-006` for a foundation Release Manager skill. |
| F-012-S007 | Passing | Generated target knowledge routes and mirrored harness docs carry the reusable feedback and improvement-loop doctrine after the AD-139 source doc. |
| F-009-S013 | Passing | `go test ./internal/docsconsistency ./internal/docsync` and `gh release view v0.41.29 --repo greaveselliott/mars-harness` cover the release-object gate and notes-only fallback. `mars-harness release verify-assets --version v0.41.29` records the separate missing-asset blocker. |
| F-010-S003 | Passing | `go test ./internal/serve -run 'TestServer_(dashboardStopEndpointStopsStart|startStop)'`, `go test ./internal/dashboard -run 'TestDashboard_(stopEndpoint|controlEndpoints_methodNotAllowed|controlEndpoints_nilCallbacks)'`, and the 2026-05-19 clean `demo-123-stop-check2` replay verify dashboard stop returns success and exits `start` without manual kill. |
| F-005-S006 | Passing | `go test ./cmd/mars-harness -run 'TestRunCommand(NoInit|AutoInit|RejectsRepoLocalLogFile)|TestMarsHarnessCLI'` and `go run ./cmd/mars-harness run engineer --repo /path/to/local-redacted --dry-run --trace --no-init` verify observer-safe dry-run exits without scaffolding an uninitialized target. |
| F-006-S001 | Passing | `go test ./pkg/testutil ./internal/queue ./internal/telemetry ./internal/foundationtelemetry ./internal/trace ./internal/scoring ./internal/trust ./internal/evolution ./internal/orgstate ./internal/serve`, `go test ./internal/docsconsistency ./internal/docsync`, and `go test ./...` verify legacy SQLite fixture coverage across persistent stores. |
| F-008-S005 | Passing | `go test ./internal/qualityscore -run 'TestExport(CreatesOutcomeSignalTickets|RendersTelemetryAndOutcomeSignals)'`, `go run ./cmd/mars-harness scores export --repo .`, and refreshed `docs/QUALITY_SCORE.md` verify guardrail outcome signals stay quality evidence by default and require `--create-intervention-debt` for ticket materialization. |
| F-008-S008 | In progress | `go test ./internal/qualityscore -run TestExportRendersFactoryPaceFromTraceSummaries` verifies quality export renders trace-derived Factory Pace rows. The 2026-05-20 run-12 export recorded Engineer at 92 turns / 45 tool invocations and Dogfood at 66 turns / 32 tool invocations; the `demo-api-run1` export recorded Engineer at 102 trace turns / 50 tool invocations with a max-turn stop. |
| F-006-S015 | In progress | `go test ./internal/scheduler -run TestScheduler_skipsWhenRepoRoleAlreadyActive` and `go test ./internal/queue -run TestQueue_activeJobForRepoRole` cover the scheduled duplicate-work prevention added after `demo-api-run1`. The `demo-api-run2` rerun preserved product progress but did not coincide with a default cron boundary, so a cron-boundary live proof remains open. |
| F-007-S010 | In progress | `go test ./internal/tools -run 'TestShellExec(AllowsUntrackedRootBuildArtifactCleanup|AllowsUntrackedGoModuleBuildArtifactCleanup|StillBlocksRemovalOfOrdinaryFiles|StillBlocksGoModuleNamedTextFileRemoval)'` covers bounded repo/module-local build artifact cleanup; `demo-api-run5` showed the remaining gap is cleanup discoverability, so the 2026-05-20 fix makes blast-radius errors name the exact safe `rm <artifact>` remediation. |
| F-007-S011 | In progress | `go test ./internal/tools -run 'TestShellExec(BlocksGoBuildOutputInsideRepoBeforeArtifact|BlocksDefaultGoBuildInsideRepoBeforeArtifact|BlocksDefaultGoBuildInShellCommandBeforeArtifact|AllowsGoBuildOutputOutsideRepo|NoopArgsNotMaskedByDirtyArtifact)'` covers pre-execution blocking for explicit and implicit Go build outputs inside the target repo and malformed shell payload surfacing; the 2026-05-20 `demo-api-run11` replay showed implicit `go build ./...` artifact creation now has a source fix awaiting live rerun. |
| F-006-S016 | Passing | `go test ./internal/scanner -run TestInit_success` covers generated CEO/COO canonical feature-contract reuse, and `demo-api-run4` confirmed the live API canary reached CTO ticketing and Engineer implementation without duplicate `F-001` paths. |
| F-005-S015 | In progress | `demo-api-run6` confirmed cleanup hints let Engineer recover from `task-notes-api`, then exposed foreground server and shell-background process leaks during API validation. `demo-api-run7` confirmed the managed-background fix, `demo-api-run10` exposed job-boundary wrapper-child cleanup, and `demo-api-run12` exposed same-job `kill <tracked-wrapper-pid>` child leakage. `go test ./internal/tools -run 'TestShellExec(RejectsShellCommandBackgroundOperator|BackgroundReportsEarlyExit|BackgroundReturnsPIDForLongRunningProcess|RejectsBarePortCommands|KillTrackedBackgroundPIDKillsDescendant)'` covers the current tool contract; live rerun evidence remains open for same-job kill interception. |

## Quality State

Checks recorded during the 2026-05-02 review:

- `go test ./...` passes.
- `go test -cover ./...` passes after making update-check fixtures
  release-agnostic.
- Coverage remains below the stated 70 percent target in several packages,
  including `internal/inference`, `internal/setup`, `internal/ui`,
  `internal/serve`, `internal/hardware`, and command entrypoint code.
- As of 2026-06-11 (`T-024`, AD-280), per-package coverage floors are enforced
  by `scripts/check-coverage.sh` inside `make check`; regressions below the
  seeded floors fail mechanically and floors are ratchet-only.
- `golangci-lint` was not installed in the local environment during the 2026-05-02
  review, so local lint status is unknown.

## Operating Rules

- In-progress tickets are still highest priority. If any appear, drain them
  before taking backlog work.
- Intervention-debt tickets do not outrank ordinary product work by default;
  surface them ahead of product backlog only when an active product ticket names
  them in `blocked_by` or an operator explicitly asks for intervention-debt work.
- Superseded plans should not remain silently active. Either reconcile them,
  move them, or mark them with a clear status and pointer to the current plan.
- Large strategic plans must be materialized into ticket files before agents
  are expected to execute them autonomously.
- Every non-release semantic commit still requires generated release notes, a
  matching release commit, a pushed `vX.Y.Z` tag, `gh release view vX.Y.Z`, a
  notes-only GitHub Release from `CHANGELOG.md` if the tag workflow did not
  create one, and `mars-harness release verify-assets --version vX.Y.Z` before
  release work can be claimed complete. Missing binary assets are recorded as a
  blocker instead of allowing the GitHub Releases page to stay stale.

## Next Ticket Work

- `MH-048`: deterministic remediation recipes are done. The completed slices
  add the remediation registry, applicability planner, trace-linked score
  evidence in `serve`, generated-docs execution through `scanner.Upgrade`,
  doctor recipe output, score-export summaries, destructive-git negative-path
  coverage, dirty-worktree blockers, and missing-optional-tool guidance.
- `MH-049`: dogfood matrix supersession benchmark is done as of 2026-05-19. The
  2026-05-19 slices add broader fake-LLM loop coverage, observer-trust mutation
  blocking, a live `demo-123` lifecycle report that reached product
  implementation, QA, Security, Dogfood, and Release Manager, and the first
  Mars observer validation report. This ticket is done.
- `T-007`: deployed `mars_harness_cli` binary resolution is done as of
  2026-05-19. The tool prefers the active harness executable before stale PATH
  binaries and adds actionable stale-binary guidance.
- `T-008`: dashboard stop is done as of 2026-05-19. Stop requests now route
  through the server loop, the regression test covers `/api/stop`, and the
  clean `demo-123-stop-check2` replay exited `start` without manual kill.
- `T-002`: foundation/deployed architecture source doc is done and should be
  used as the input for mirroring and drift review.
- `T-003`: generated target mirroring is done and should be used as input for
  the drift review.
- `T-004`: doctrine drift review is done and found no unowned mismatch across
  source and generated target surfaces.
- `T-005`: skill/tool/doctrine evaluation is done; the recursive improvement
  loop remains operating doctrine and `T-006` captures the foundation Release
  Manager skill implementation.
- `T-006`: foundation release publication skill is done as of 2026-05-19.
  `.harness/skills/release-publication/SKILL.md` covers release-note
  commit, push, tag, GitHub Release object, notes-only fallback, asset
  verification, token safety, and blocker recording. Generated target mirroring
  is deferred with rationale because target publication modes differ from the
  source binary-release workflow.
- `T-009`: observer-safe dry-run is done as of 2026-05-19. `run --dry-run
  --no-init` reports the missing-harness boundary without writing `.harness/`,
  the Mars observer profile uses it, and the live Mars checkout stayed clean.
- `MH-050`: persistent store upgrade fixtures are done as of 2026-05-19.
  Legacy-open fixture tests cover queue, telemetry, foundation telemetry, trace,
  scoring, trust, evolution, orgstate, repo registry, and the shared SQLite
  fixture helper.
- `T-001`: next intervention-debt slice. Calibrate the quality-score guardrail
  workflow signal is done as of 2026-05-19. The original 2026-05-03 signal is
  stale and no longer traceable in the current local DB, current default export
  reports insufficient live score evidence without creating tickets, and tests
  assert outcome signals need explicit `--create-intervention-debt` before
  materializing backlog work.
- `T-012`: done as of 2026-05-20. The `demo-123-run5` live lifecycle
  replay found that Dogfood-created target tickets must be committed before
  `changes_requested` handoff, otherwise Engineer receives an untracked ticket
  and gets trapped behind claim guardrails. The `demo-123-run6` replay confirmed
  terminal clean-tree handoffs help normal role completion, but found that
  Dogfood can still create a target-owned ticket on its final turn, hit
  `max_turns`, and let the watchdog route Engineer while the ticket is
  uncommitted. As of 2026-05-20, the bounded fix pauses orchestrator survey
  routing for dirty target workspaces except runtime-only
  `.harness/learnings.yaml`, and the patched `run6` replay confirmed no new
  Engineer job was routed while `T-002` was uncommitted.
- `T-013`: backlog as of 2026-05-20. The `demo-123-run6` replay found
  Dogfood/static-serving turn waste: repeated root-server retries, broad process
  inspection, broad find attempts, and final-turn ticket creation before
  `max_turns`. Use this as concrete candidate input for `T-011` factory pace
  measurement.
- `T-014`: done as of 2026-05-20. The `demo-123-run10` replay found no-remote
  release publication could mutate remotes and route back to Dogfood after local
  release evidence. The `demo-123-run11` replay confirmed remote mutation is
  blocked and `release_blocked` stops dispatch with no pending follow-up jobs.
- `T-011`: in progress as of 2026-05-21. The clean `demo-cli-run7` replay
  confirmed product-specific CEO and COO planning on a generic Note Stats CLI
  target with zero target intervention-debt tickets, then exposed a fresh
  ticketing blocker: COO wrote `F-002-SNNN` headings inside the existing `F-001`
  contract, causing CTO ticket creation to stall behind the feature-contract
  planning-order guardrail. The clean `demo-cli-run8` replay confirmed the
  scenario/file ID alignment fix reaches CTO ticket creation and a committed
  working CLI, then exposed post-validation convergence drift: Engineer kept
  running exploratory shell probes after successful validation and an
  implementation commit until `context_overflow`. The clean `demo-cli-run9`
  replay confirmed ticket closure reaches QA and QA validation gates catch
  missing tests, then exposed review rework against a ticket still marked done.
  The clean `demo-cli-run10` replay confirmed rework-ticket reopening
  enforcement, product implementation, passing tests, and docsync, then exposed
  the completion-path edge case: the final product-plus-ticket-done commit was
  blocked as if it were hidden rework. The clean `demo-cli-run11` replay
  reached implementation and successful external validation, then exposed
  repeated empty shell no-ops before commit and ticket close. The clean
  `demo-cli-run12` replay confirmed Engineer reached QA with a completed
  product ticket, then exposed a review-policy/tool-surface mismatch: QA
  requires in-job validation evidence but lacked `shell_exec`, and `/tmp`
  validation binaries could be reused without same-session freshness proof. The
  clean `demo-cli-run13` replay confirmed product-specific planning, ticketing,
  and a committed Note Stats CLI implementation, then exposed that direct
  `go run` product probes were not counted as validation evidence, so Engineer
  repeated no-op shell placeholders until `circle_detected` with `T-001` still
  in progress. The 2026-05-21 source slice gives QA bounded validation-only
  `shell_exec`, records same-session external validation artifacts, blocks
  stale `<validation-root>` executions, allows fresh validation artifact
  execution through the Engineer convergence gate, and counts successful direct
  runtime probes as validation evidence. The clean `demo-cli-run14` replay
  confirmed Engineer closes the ticket after direct runtime proof and reaches
  QA, then exposed that expected non-zero runtime probes for invalid input were
  treated like failed builds/tests and blocked QA approval. The clean
  `demo-cli-run15` replay then exposed an earlier completion blocker: Engineer
  reached committed implementation plus passing external runtime proof, but
  repeated shell placeholders after the completion gate instead of updating
  ticket evidence through `file_read`/`file_write` and moving `T-001` to done.
  The clean `demo-cli-run16` replay confirmed ticket evidence convergence
  reaches QA, then exposed a QA false approval after an unexpected empty-text
  runtime failure contradicted the brief while tests encoded the bug. The clean
  `demo-cli-run17` replay confirmed unexpected runtime failures no longer
  approve by default and the empty-text behavior was fixed, then exposed the
  2026-05-21 failing scenario: QA ran a failing `go test ./...` on the final turn
  and hit `max_turns` before recording `changes_requested`, while Engineer used
  `<validation-root>` instead of the freshness-tracked
  `<validation-root>` path. The 2026-05-21 source slice stops QA/Security
  shell validation after the first failing build/test/unexpected runtime
  command, gives dispatch jobs one terminal-tool grace prompt at the
  turn-budget edge, and blocks external Go validation builds that do not use
  `<validation-root>`. The clean `demo-cli-run18` replay confirmed that failing
  QA validation became structured `changes_requested`, Orchestrator routes
  rework back to Engineer, Engineer reopens done tickets before mutation, and
  same-session validation artifact freshness works across role boundaries. The
  2026-05-21 failing scenario is narrower: an expected-negative missing-argument
  probe run once without `expected_exit_code` could not be corrected because the
  shell-stop rule blocked the exact rerun. The 2026-05-21 source slice allows one
  exact matching `expected_exit_code` correction for runtime probes while keeping
  failing builds/tests and uncorrected runtime failures on the rework path. The
  clean `demo-cli-run19` replay confirmed fresh product-specific bootstrap and
  implementation still proceed without target intervention-debt tickets, then
  exposed completion outrunning evidence: Engineer observed an empty-text
  runtime failure, marked the acceptance path complete, moved `T-001` to done,
  and ended as `max_turns` after spending the terminal grace turn on a
  non-terminal lifecycle command. The 2026-05-21 source slice now blocks
  Engineer ticket completion while current-job runtime validation failures
  remain unrepaired, and restricts the budget-edge grace turn to the configured
  terminal disposition tool. The clean `demo-cli-run20` replay confirmed the
  done move is blocked while the empty-text runtime failure remains unresolved,
  with no target intervention-debt tickets. It then exposed an Engineer
  expected-exit bypass loop: the role retried the failed acceptance path with
  `expected_exit_code: 1` until `circle_detected`. The 2026-05-21 source slice
  makes retroactive expected-exit correction review-only for QA/Security;
  Engineer must make the exact failed runtime command pass before completion.
  The clean `demo-cli-run21` replay confirmed that bypass is closed and
  product planning still works, then exposed a repeat-failure loop: Engineer
  kept running runtime probes after the empty-text acceptance command failed.
  The 2026-05-21 source slice now requires a post-failure implementation edit
  before Engineer can rerun runtime probes, and still requires the exact failed
  command to pass before completion. The clean `demo-cli-run22` replay
  confirmed that behavior: Engineer edited after the blocked repeat and made
  the exact empty-text command pass. It then exposed an expected-negative
  correction gap for missing required arguments; as of 2026-05-21, source
  policy allows Engineer to rerun an obvious missing-argument probe with
  matching `expected_exit_code` while preserving strict rework for positive
  acceptance failures. The clean `demo-cli-run23` replay confirmed that
  correction path, then exposed zero-exit error stderr being counted as
  successful runtime evidence; as of 2026-05-21, source policy treats
  conservative error-shaped stderr from direct runtime validation as failed
  evidence until the exact command passes cleanly. The clean `demo-cli-run24`
  replay confirmed product-first planning, feature-contract update, ticket
  creation, and intervention-debt quarantine, then exposed an Engineer
  claimed-ticket no-op loop before implementation; as of 2026-05-21, repeated
  pre-validation no-op shell calls after claim are blocked and routed to
  ticket/feature reading plus product `file_write` implementation or a blocked
  disposition. The clean `demo-cli-run25` replay confirmed that fix, with
  Engineer writing product files and repairing the empty-text path, then exposed
  missing-argument correction wording; as of 2026-05-21, unresolved runtime
  blocker messages explicitly name the exact-command `expected_exit_code`
  correction for no-argument or missing-required-input probes. The clean
  `demo-cli-run26` replay confirmed CEO/COO still produce product-specific
  planning, then exposed false CTO progress after failed ticket creation; as
  of 2026-05-21, unresolved failed `ticket_create` and ticket-file bypass
  attempts block successful dispositions until a later `ticket_create`
  succeeds.
  The clean `demo-cli-run27` replay confirmed that ticket-creation fix: CTO
  created a real product ticket and Engineer started implementation. It exposed
  that a missing-argument runtime probe first run without `expected_exit_code`
  still allowed adjacent work before correction; as of 2026-05-21, session
  state stores the exact correction and policy blocks unrelated Engineer
  mutations until that correction runs or an honest blocked disposition is
  recorded.
  The clean `demo-cli-run28` replay confirmed CTO ticketing and Engineer
  implementation handoff, then exposed stale `<validation-root>` reuse after
  a positive acceptance failure and source edit; as of 2026-05-21, external
  validation artifacts must be rebuilt after post-failure implementation edits
  before rerun evidence is trusted.
  The clean `demo-cli-run29` replay confirmed fresh product-specific planning,
  feature contract, ticketing, and Engineer source work, then exposed ticket
  evidence outrunning validation: Engineer wrote in-progress `evidence_links`
  and `verified_by` before any successful validation in the job. As of
  2026-05-21, Engineer ticket evidence writes are blocked until the same job
  records successful validation.
  The clean `demo-cli-run30` replay confirmed that evidence-first behavior:
  Engineer validated, updated ticket evidence, moved `T-001` to done, and
  handed to QA. It then exposed review recovery drift: QA hit the same-session
  `<validation-root>` artifact guard and stalled instead of rebuilding the
  binary in its own session. As of 2026-05-21, external artifact freshness
  errors name the exact `shell_exec argv` rebuild command for QA/Security.
  The clean `demo-cli-run31` replay confirmed product-first delivery reached
  QA with ordinary product artifacts and zero target intervention-debt tickets,
  and confirmed exact missing-argument correction state in a live Engineer
  run. It then exposed two validation-quality gaps: Engineer accepted
  exit-zero runtime output that contradicted the empty-text JSON contract, and
  QA guessed an invalid root package after a repo-local `go build
  ./cmd/note-stats` guardrail block. As of 2026-05-21, Go build-output
  guardrails emit exact corrected `shell_exec argv` commands that preserve the
  package target, and generated role guidance requires automated assertions
  for explicit expected-output examples; QA approval is mechanically blocked
  for Go source changes when no `_test.go` files exist.
  The clean `demo-cli-run32` replay confirmed product-first delivery still
  reached a committed product ticket with Go tests and exact build-output
  correction, then exposed the next generic fault: after a missing-input
  expected-exit repro still panicked, policy trapped Engineer before repair.
  It also exposed deployed/foundation naming drift through `cmd/mars-harness`
  and `module mars-harness` in a Note Stats target. As of 2026-05-21, failed
  missing-input correction attempts unlock implementation edits while
  completion stays blocked, and generated CTO/Engineer guidance requires
  target-derived command, module, and binary names.
  The clean `demo-cli-run33` replay confirmed target-derived `cmd/note-stats`
  and `module note-stats` behavior plus implementation repair after runtime
  failure. It then exposed stale runtime-failure accounting: repeated failures
  of the same exact `--text ""` command left multiple outstanding counters,
  and one later exact success only cleared one. As of 2026-05-21, exact runtime
  repair clears all unmatched failures for the same command fingerprint in the
  current job.
  The clean `demo-cli-run34` replay confirmed the repeated-runtime repair:
  Engineer fixed the empty-text path, corrected the omitted-flag negative path
  with `expected_exit_code`, wrote tests, passed docsync, and reached QA without
  intervention-debt amplification. It then exposed review and traceability
  drift: QA ran package initialization during review, missed the first-run
  `expected_exit_code` on an intentional negative probe, Orchestrator treated
  ticket README examples as live backlog, and product code was bundled into the
  ticket done move commit. As of 2026-05-21, reviewer shell access is
  validation-only, ticket done moves require non-ticket product changes to be
  committed first, and Orchestrator routing must ignore README examples as live
  work.
  The clean `demo-cli-run35` replay confirmed the ticket-closure fix: Engineer
  committed product source, README, and `go.mod` separately before a
  lifecycle-only done-ticket commit. QA then built a fresh validation binary
  and passed docsync plus runtime probes, but repeated empty `shell_exec`
  placeholders instead of recording `job_disposition_record`, ending in
  `circle_detected`. As of 2026-05-21, required terminal-tool jobs get one
  circle-grace reminder, reviewer no-op placeholders after successful
  validation route directly to structured disposition, and policy-blocked
  no-op shell calls are counted as no-op failures for telemetry.
  The clean `demo-cli-run36` replay confirmed product-first planning and
  ticketing still work without target intervention-debt tickets, but exposed
  that an unresolved empty-string acceptance failure could still be bypassed
  with shell-wrapper probes, unrelated validation, ticket evidence edits, and
  an implementation commit. As of 2026-05-21, Engineer runtime acceptance
  failures freeze unrelated shell paths and product commits until the exact
  failed command passes, while stale `<validation-root>` artifact rebuilds
  remain available after source edits.
  The clean `demo-temp-run37` replay used a different Temperature JSON CLI
  target. It confirmed product-specific planning, ticket creation,
  implementation, tests, exact omitted-flag `expected_exit_code` correction,
  product commit, evidence update, docsync, and a lifecycle-only done-ticket
  commit. It then exposed stale ticket-creation state: a blocked Engineer
  pre-validation evidence write later prevented an otherwise valid successful
  disposition. As of 2026-05-21, Engineer evidence-write failures no longer
  count as ticket-creation debt; failed `ticket_create` and non-Engineer
  ticket-file bypass attempts still block false planning handoffs.
  The clean `demo-temp-run38` replay on 2026-05-21 confirmed the alternate CLI canary
  reaches product implementation, evidence update, and done-ticket closure, but
  exposed two remaining generic loop leaks: COO tried alternate ticket creation
  paths despite not owning ticketing, and QA alternated no-op shell placeholders
  after validation until `circle_detected` instead of recording disposition or
  requesting missing tests. As of 2026-05-21, non-ticket-owning planners hand
  off `ticket_breakdown` to CTO without alternate ticket writes, review no-op
  recovery after successful validation is terminal-only, and QA routes Go
  source without `_test.go` files to `changes_requested`.
  The clean `demo-temp-run39` replay on 2026-05-21 confirmed the planning
  handoff fix and durable-test expectation on the alternate CLI target, then
  exposed failing-test bypass: Engineer observed a failing `go test`, proved
  narrower runtime probes, attempted forbidden cleanup, and committed product
  work while the authoritative test command still failed. As of 2026-05-21,
  Engineer failing test/build evidence creates a repair lane: source/test
  edits remain available, but runtime side probes, ticket evidence, ticket
  completion, successful disposition, and product commits are blocked until the
  exact failing command passes.
  The clean `demo-temp-run40` replay on 2026-05-21 confirmed early planning
  again, then exposed CTO role-boundary drift before Engineer validation: CTO
  created the ticket but also wrote `go.mod`, attempted source/test writes,
  updated README usage, and committed product-adjacent state. As of 2026-05-21,
  CTO file writes are limited to technical planning artifacts, while package,
  README usage, source, test, build, config, and root product-file changes
  belong to ticket-backed Engineer delivery.
  The clean `demo-temp-run41` replay on 2026-05-21 confirmed CTO now creates
  and commits only the implementation ticket before Engineer claims it. It then
  exposed that the test/build repair lane was too exact-command-bound:
  Engineer could not run a focused same-lane `go test` after the original
  package-pattern test failed, and started trying workaround paths such as a
  root verification script. As of 2026-05-21, test/build repair lanes accept
  bounded source/test/fixture/build-config edits followed by same-lane focused
  validation, while runtime probes, helper scripts, ticket evidence,
  completion, disposition, and product commits stay blocked until validation
  passes.
  The clean `demo-temp-run42` replay on 2026-05-21 confirmed AD-196's
  blocking side: runtime probes, build substitution, helper paths, and commits
  stayed blocked while the test lane was unresolved. The same run also exposed that
  focused shell validation shaped as `cd cmd/temperature-json-cli && go test
  -v .` was not classified as a test command. As of 2026-05-21, AD-197
  recognizes that narrow `cd <dir> && <test/build>` form for same-lane repair
  while keeping arbitrary shell wrappers blocked.
  The clean `demo-temp-run43` replay on 2026-05-21 confirmed the product path
  through Engineer and QA on the alternate target: Engineer corrected an
  expected missing-input probe, repaired a failing test, committed product
  work, updated evidence, closed the ticket, and QA approved. Security then
  gathered clean read plus validation evidence but spent more than five minutes
  in the next model turn instead of recording disposition. As of 2026-05-21,
  AD-198 makes clean QA/Security review evidence a terminal-only boundary with
  a short grace timeout for the required `job_disposition_record` response.
  The clean `demo-temp-run44` replay on 2026-05-21 confirmed product-first
  planning and Engineer delivery again, but QA falsely routed implementation
  rework after a validation command procedure mistake: `go build ... cmd/...`
  lacked the required `./cmd/...` package prefix. As of 2026-05-21, AD-199
  records obvious QA/Security Go build/test command-target mistakes separately
  from target validation failures so reviewers can correct their command and
  continue without creating false product rework.
  The clean `demo-temp-run45` replay on 2026-05-21 then exposed the same
  validation shape one layer earlier: Engineer tried the safe focused nested
  module test as `argv:["cd","cmd/temperature-json-cli","&&","go","test",
  "./..."]`, which argv mode rejected before the model fell into a root
  `go test ./cmd/...` failure. As of 2026-05-21, AD-200 normalizes only the
  validation-only `cd <dir> && <test/build>` argv shape into `shell_command`
  while keeping arbitrary shell syntax rejected.
  The clean `demo-temp-run46` replay on 2026-05-21 confirmed the external
  build-artifact correction and positive runtime checks, then exposed a
  missing-input CLI validation loop: Engineer ran the validation binary with
  no arguments, received the expected `--celsius flag is required` error, but
  the runtime repair guardrail treated the absent `expected_exit_code` as an
  unresolved failure and blocked completion. As of 2026-05-21, AD-201 treats
  clear missing-input CLI probes with required/usage output and no crash
  markers as expected negative-path evidence on the first run.
  The clean `demo-temp-run47` replay on 2026-05-21 confirmed that
  missing-input validation no longer poisoned Engineer completion, then
  exposed the sibling invalid-input case: `go run ... invalid` correctly
  produced `Must be a number`, but the policy still treated it as unexpected.
  As of 2026-05-21, AD-201 covers deliberate invalid-input probes as well as
  missing-input probes, while valid positive inputs rejected as invalid remain
  failures.
  The clean `demo-temp-run48` replay on 2026-05-21 confirmed AD-201 through
  Engineer implementation, positive runtime validation, missing-input and
  invalid-input validation, product commit, ticket closure, and QA handoff.
  QA requested rework because the Go CLI lacked `_test.go` coverage, and the
  rework Engineer then hit a missing `./` Go package-target procedure failure
  that incorrectly poisoned the product repair lane. As of 2026-05-21, AD-202
  classifies obvious Engineer Go validation-procedure mistakes separately
  from real product build failures so corrected validation can continue.
  The clean `demo-temp-run49` replay on 2026-05-21 confirmed the alternate
  CLI target through product planning, ticket creation, Engineer claim, source
  write with DocSync metadata, external validation build, positive runtime
  validation, and missing-input validation. It then exposed the next
  negative-path classifier gap: `<validation-root> 25 30`
  correctly returned `error: too many arguments provided`, but the runtime
  guardrail treated the surplus-argument rejection as unexpected. As of
  2026-05-21, AD-203 treats clear surplus-argument CLI probes as expected
  negative-path evidence when the output names too many or surplus arguments
  and no crash markers are present.
  The clean `demo-temp-run50` replay on 2026-05-21 confirmed another
  product-first path through Engineer implementation, external validation,
  positive runtime checks, and explicit expected-exit negative checks. It then
  exposed the next repair-lane gap: after bad same-job test files caused a real
  `go test` compile failure, the guardrail correctly blocked unrelated work
  but also blocked the `rm` cleanup needed to delete those bad same-job test
  files. As of 2026-05-21, AD-204 allows only non-recursive removal of
  test-like files written by the same Engineer job after the test/build
  failure began; source files, unmarked tests, and recursive cleanup remain
  blocked.
  The clean `demo-temp-run51` replay on 2026-05-21 confirmed full
  product-first delivery through Engineer: plan, ticket, implementation,
  tests, runtime evidence, ticket closure, and QA handoff. QA corrected the
  familiar `cmd/...` versus `./cmd/...` Go build procedure mistake, then had
  sufficient review evidence but missed the required terminal
  `job_disposition_record` call and ended as `circle_detected`. As of
  2026-05-21, AD-205 rejects the first non-terminal response after a clean
  review-evidence reminder without executing it and gives one stronger
  terminal-only correction before repeated misses fail.
  The clean `demo-temp-run52` replay on 2026-05-21 exercised a different
  Engineer repair path. After a failing `go test ./cmd/temperature-json-cli/...`
  command, the repair lane blocked runtime probes, destructive cleanup, commits,
  and ticket moves, but still allowed root `main.go` and `main_test.go` writes.
  Engineer started validating a parallel root implementation instead of the
  failed package. As of 2026-05-21, AD-206 records narrow Go package
  test/build repair scopes and blocks source/test/fixture writes outside that
  scope until the lane is repaired.
  The clean `demo-temp-run55` replay on 2026-05-21 confirmed run metadata
  injection but exposed the same-job cleanup gap one layer later: Engineer
  wrote duplicate or placeholder test files before the first failing package
  test, then could not remove them because AD-204 only covered files written
  after the failure began. As of 2026-05-21, AD-209 records every successful
  Engineer `file_write` path so same-job generated test files can be removed
  during unresolved test/build repair while pre-existing tests and source files
  stay protected.
  The clean `demo-temp-run56` replay on 2026-05-21 validated that direction by
  reaching QA, receiving test-coverage rework, adding focused Go tests, closing
  `T-001`, and handing back to QA. The next blocker was review terminal
  convergence: QA successfully ran `go test ./cmd/temperature-json-cli/` but
  tried required `docsync_audit` after the runtime had already forced
  `job_disposition_record`, so the job ended with `circle_detected`. As of
  2026-05-21, AD-210 requires docsync evidence before the review terminal
  boundary fires.
  The clean `demo-temp-run57` replay on 2026-05-21 confirmed docsync can now
  run before the terminal boundary, then exposed an adjacent evidence-ordering
  bug: QA was forced to terminal disposition after external build evidence
  even though `_test.go` files existed and the test command had not run. As of
  2026-05-21, AD-211 requires review terminal convergence to wait for a
  successful test command when test files exist.
  The clean `demo-temp-run58` replay on 2026-05-21 confirmed the direct
  convergence check but exposed no-op recovery as a stale terminal trigger: an
  empty review `shell_exec` after build evidence forced approval guidance
  before tests had passed. As of 2026-05-21, AD-212 keeps no-op recovery
  aligned with the same evidence gates as approval and points reviewers to
  missing tests or docsync before terminal approval guidance.
  The clean `demo-temp-run59` replay on 2026-05-21 validated that review path
  and reached local release artifacts: product planning, ticket creation,
  Engineer implementation, QA, Security, Dogfood, `release: notes 0.2.0`, and
  tag `v0.2.0`. The remaining rough edge was Release Manager using
  `shell_exec mars-harness release notes`, which resolved a stale installed
  binary before recovering through a second release pass. As of 2026-05-21,
  AD-213 blocks direct `mars-harness` shell invocations in agent jobs and
  routes Mars Harness CLI workflows through `mars_harness_cli`.
  The clean `demo-temp-run60` replay on 2026-05-21 broadened the canary matrix
  with a Word Count JSON CLI. It validated AD-213 in the live release path:
  Release Manager used `mars_harness_cli`, committed `release: notes 0.2.0`,
  created tag `v0.2.0`, and stopped only on the real missing-remote blocker.
  The new finding was startup retry persistence: a bind-failed sandboxed start
  registered and queued bootstrap state, then automatic cleanup deleted SQLite
  `-wal`/`-shm` sidecars before the retry. As of 2026-05-21, AD-214 preserves
  SQLite sidecars and lets SQLite recover/checkpoint them instead of deleting
  queue or repo registry state.
  The clean `demo-slug-run61` replay on 2026-05-21 validated AD-214 with a
  Slugify JSON CLI: the retry reused the same repo ID and CEO job after a
  bind-failed first start. It then exposed the next pace issue. QA correctly
  requested test rework, Orchestrator routed it to Engineer, and Engineer added
  a failing contract-shaped test, but the unresolved test/build guardrail only
  repeated the command and the role churned for 9m44s without repairing the
  implementation. As of 2026-05-21, AD-215 repeats the latest failing
  test/build output in guardrail guidance and tells Engineer to edit
  implementation when the assertion matches the contract.
  The clean `demo-slug-run62` replay on 2026-05-21 validated AD-215 in the
  live lifecycle: the same Slugify JSON CLI shape completed CEO, COO,
  CTO-weekly, Engineer, QA, Security, Dogfood, and Release Manager in one local
  run; generated `release: notes 0.2.0`; created tag `v0.2.0`; kept guardrail
  blocks as foundation telemetry; and created no target intervention-debt
  tickets. The remaining terminal blocker was the expected missing remote for
  publication in the temporary target. Next canaries should broaden beyond
  small CLI targets and include a remote-backed release path.
  The clean `demo-notes-api-run63` replay on 2026-05-21 broadened the matrix
  with a Go HTTP JSON API and a local bare `origin`. The lifecycle again reached
  product-specific CEO, COO, and CTO-weekly work, created one ordinary product
  ticket, and pushed Engineer's ticket-claim commit to the remote. The new
  blocker was bootstrap repair policy: after `go test ./internal/note` failed
  because no Go module existed, the unresolved test/build guardrail blocked the
  direct `go mod init` repair and the role drifted into workaround attempts. As
  of 2026-05-21, AD-216 allows `go mod init` only when missing-module output
  and an absent `go.mod` prove it is the direct package-config repair.
  The patched `demo-notes-api-run64` replay on 2026-05-21 avoided the module
  trap by writing `go.mod` before validation, then exposed two adjacent
  evidence holes on the HTTP-service shape: raw `go get` escaped dependency
  sync, and same-job test cleanup could delete assertion evidence after a
  focused test failure. As of 2026-05-21, AD-217 blocks raw `go get` and
  reserves same-job test cleanup for duplicate/generated-test shaped failures.
  The clean `demo-inventory-api-run65` replay on 2026-05-21 broadened the
  matrix with a different Go HTTP JSON API. It completed product planning,
  feature contract update, ordinary product ticketing, implementation, QA,
  Security, Dogfood finding creation, Orchestrator rework routing, and Engineer
  route repair validation. The terminal gap was post-runtime-validation
  convergence: after successful background server validation and `/health`
  probe, Engineer repeated no-op placeholders instead of killing the tracked
  PID, committing the dirty repair, updating evidence, moving `T-002` to done,
  and handing back to QA. As of 2026-05-21, AD-218 blocks the first such
  no-op with exact convergence steps.
