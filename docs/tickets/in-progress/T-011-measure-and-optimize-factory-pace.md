---
id: T-011
title: Measure and optimize factory pace
priority: high
complexity: large
work_type: intervention-debt
bdd_scenarios:
  - F-008-S008
  - F-006-S015
  - F-006-S016
  - F-007-S010
  - F-007-S011
  - F-005-S015
end_to_end_evidence: not_applicable
evidence_links:
  - "go test ./internal/qualityscore -run TestExportRendersFactoryPaceFromTraceSummaries"
  - "go test ./internal/scheduler -run TestScheduler_skipsWhenRepoRoleAlreadyActive"
  - "go test ./internal/queue -run TestQueue_activeJobForRepoRole"
  - "go test ./internal/tools -run 'TestShellExec(AllowsUntrackedRootBuildArtifactCleanup|AllowsUntrackedGoModuleBuildArtifactCleanup|StillBlocksRemovalOfOrdinaryFiles|StillBlocksGoModuleNamedTextFileRemoval)'"
  - "go test ./internal/tools -run 'TestShellExec(RejectsShellCommandBackgroundOperator|AllowsShellCommandNonBackgroundAmpersands|RejectsBarePortCommands|BackgroundReportsEarlyExit|BackgroundReturnsPIDForLongRunningProcess)'"
  - "go test ./internal/tools -run 'TestShellExec(BlocksGoBuildOutputInsideRepoBeforeArtifact|AllowsGoBuildOutputOutsideRepo|MalformedArgsNotMaskedByDirtyArtifact)'"
  - "go test ./internal/scanner -run TestInit_success"
  - docs/validation/reports/2026-05-19-demo-123-live-lifecycle.md#run-12-tool-argument-and-matrix-replay--2026-05-20
  - docs/validation/reports/2026-05-19-demo-123-live-lifecycle.md#non-static-matrix-replay-task-notes-api---2026-05-20
  - docs/validation/reports/2026-05-19-demo-123-live-lifecycle.md#api-rerun-after-scheduler-skip-task-notes-api---2026-05-20
  - docs/validation/reports/2026-05-19-demo-123-live-lifecycle.md#api-rerun-after-canonical-bootstrap-guidance-task-notes-api---2026-05-20
  - docs/validation/reports/2026-05-19-demo-123-live-lifecycle.md#api-rerun-after-module-artifact-cleanup-task-notes-api---2026-05-20
  - docs/validation/reports/2026-05-19-demo-123-live-lifecycle.md#api-rerun-after-generated-artifact-cleanup-hints-task-notes-api---2026-05-20
  - docs/validation/reports/2026-05-19-demo-123-live-lifecycle.md#api-rerun-after-managed-background-validation-task-notes-api---2026-05-20
  - docs/validation/reports/2026-05-19-demo-123-live-lifecycle.md#api-rerun-after-build-output-prevention-task-notes-api---2026-05-20
verified_by: "partial: go test ./internal/qualityscore -run TestExportRendersFactoryPaceFromTraceSummaries; go test ./internal/scheduler -run TestScheduler_skipsWhenRepoRoleAlreadyActive; go test ./internal/queue -run TestQueue_activeJobForRepoRole; go test ./internal/tools -run 'TestShellExec(AllowsUntrackedRootBuildArtifactCleanup|AllowsUntrackedGoModuleBuildArtifactCleanup|StillBlocksRemovalOfOrdinaryFiles|StillBlocksGoModuleNamedTextFileRemoval)'; go test ./internal/tools -run 'TestShellExec(RejectsShellCommandBackgroundOperator|AllowsShellCommandNonBackgroundAmpersands|RejectsBarePortCommands|BackgroundReportsEarlyExit|BackgroundReturnsPIDForLongRunningProcess)'; go test ./internal/tools -run 'TestShellExec(BlocksGoBuildOutputInsideRepoBeforeArtifact|AllowsGoBuildOutputOutsideRepo|MalformedArgsNotMaskedByDirtyArtifact)'; run12, demo-api-run1 through demo-api-run8 scores exports and trace summaries captured live Factory Pace baselines"
owner: "Codex"
last_attempt: "2026-05-20: first slice adds Factory Pace rows to QUALITY_SCORE.md by joining scoring outcomes to trace summaries; run12 export shows Engineer 92 turns/45 tools and Dogfood 66 turns/32 tools. Non-static demo-api-run1 replay shows Engineer max_turns at 102 trace turns/50 tools and a duplicate scheduled Engineer queued while the first Engineer was still active; scheduler now skips same-repo same-role active work. demo-api-run2 then shows Engineer circle_detected at 89 trace turns/43 tools after repo-local Go build artifact cleanup was blocked; shell_exec permits narrow cleanup of untracked root binaries named after the repo. demo-api-run3 then showed CEO/COO duplicate F-001 feature-contract path and duplicate starter-scenario drift before CTO ticketing; generated bootstrap guidance made canonical feature-contract reuse explicit. demo-api-run4 confirmed CEO/COO/CTO reach Engineer, then exposed module-named Go build artifact cleanup as the next blocker. demo-api-run5 confirmed the exception exists but the guardrail error needs to name the exact cleanup command so Engineer discovers it. demo-api-run6 confirmed the cleanup hint works, then exposed foreground service validation and shell-background process management as the next generic turn sink. demo-api-run7 confirmed managed background validation works and exposed repo-local `go build -o` outputs as the next generic artifact-prevention gap. demo-api-run8 confirmed build-output prevention works and exposed bare `:8080` commands as the next malformed validation loop."
blocker: "none"
blocked_by: []
trace_id: "TBD"
next_action: "Rerun the non-static API canary after bare-port command rejection, confirm Engineer no longer loops on :8080 after validation errors, then target the next largest generic turn sink."
kind: intervention-debt
dedupe_key: "public-example"
metadata:
  category: "factory_pace"
  confidence: "high"
  evidence: "User reported on 2026-05-19 that agents take many turns before solid outcomes, the upper turn limit is too low, and factory pace is not measured."
  origin: "user request 2026-05-19"
  repo: "mars-harness"
  role: "all factory agents"
  severity: "high"
  target: "foundation and deployed harnesses"
source: user request 2026-05-19
created: 2026-05-19
depends_on: []
---

# T-011: Measure and optimize factory pace

## Context

Factory pace is currently an implicit concern rather than a measured performance signal. The user reported that different agents take many turns before reaching a solid outcome while the upper turn limit is also too low for hard work. That combination creates two separate risks: wasted turns when agents drift, and premature failure when a valid task needs a longer but still productive run.

The factory needs a durable pace metric that shows how quickly roles reach useful outcomes, how many turns and tool calls are spent on avoidable loops, and whether runtime limits are calibrated by evidence instead of guesswork.

## Requirements

- Build a clear baseline view of current factory pace from available traces, queue/job outcomes, scoring evidence, and dogfood reports.
- Define pace metrics that operators and future automation can reason about, including turns-to-solid-outcome, tool calls per successful outcome, wall-clock duration, retry count, max-turn failure rate, and avoidable-loop indicators.
- Decide where pace belongs in the operating model: scoring, quality score export, dashboard/CLI visibility, trace summaries, telemetry triage, or some combination.
- Plan and implement targeted speed improvements only after the baseline identifies the largest turn sinks. Candidate fixes may include role prompt tightening, terminal-handoff enforcement, context routing changes, deterministic preflight/remediation shortcuts, model/profile routing, or max-turn calibration by role.
- Calibrate upper limits so productive work can continue while unproductive loops still stop quickly and produce actionable intervention debt.
- Add tests and repeatable evidence that pace improved or that the new metric exposes the bottleneck honestly when improvement is not yet proven.

## Affected Files

- internal/agent/
- internal/serve/
- internal/scoring/
- internal/qualityscore/
- internal/trace/
- internal/tools/
- internal/scanner/
- internal/ui/
- internal/dashboard/
- docs/features/F-005-agent-execution-runtime.md
- docs/features/F-008-scoring-trust-quality.md
- docs/design-docs/scoring-system.md
- docs/design-docs/dogfood-and-decisions.md
- docs/QUALITY_SCORE.md

## BDD Evidence

- Scenario IDs: F-008-S008, F-006-S015, F-006-S016, F-007-S010, F-007-S011, and F-005-S015.
- Evidence links: `go test ./internal/qualityscore -run TestExportRendersFactoryPaceFromTraceSummaries` covers the first quality-export pace slice.
- Evidence links: `go test ./internal/scheduler -run TestScheduler_skipsWhenRepoRoleAlreadyActive` and `go test ./internal/queue -run TestQueue_activeJobForRepoRole` cover the scheduled duplicate-work fix.
- Evidence links: `go test ./internal/tools -run 'TestShellExec(AllowsUntrackedRootBuildArtifactCleanup|AllowsUntrackedGoModuleBuildArtifactCleanup|StillBlocksRemovalOfOrdinaryFiles|StillBlocksGoModuleNamedTextFileRemoval)'` covers the bounded build-artifact cleanup exception.
- Evidence links: `go test ./internal/tools -run 'TestShellExec(RejectsShellCommandBackgroundOperator|AllowsShellCommandNonBackgroundAmpersands|BackgroundReportsEarlyExit|BackgroundReturnsPIDForLongRunningProcess)'` covers managed long-running server validation tool behavior.
- Evidence links: `go test ./internal/tools -run 'TestShellExec(BlocksGoBuildOutputInsideRepoBeforeArtifact|AllowsGoBuildOutputOutsideRepo|MalformedArgsNotMaskedByDirtyArtifact)'` covers repo-local validation binary prevention and dirty-artifact validation ordering.
- Evidence links: `go test ./internal/scanner -run TestInit_success` covers generated CEO/COO canonical feature-contract reuse.
- Verified by: partial; live baseline export and before/after replay evidence remain open.

## Acceptance Criteria

### Functional (happy path)
- [ ] Current factory pace is measured from durable evidence, with a dated baseline documented before optimization work starts.
- [ ] Pace is represented as a first-class metric with role/repo/job attribution rather than as ad hoc notes in chat.
- [ ] The implementation surfaces pace where operators already inspect factory health, such as scores export, trace summaries, dashboard, or CLI output.
- [ ] At least one evidence-backed resolution reduces avoidable turns or improves successful completion before max-turn failure.
- [ ] Max-turn and budget behavior is calibrated by role or work type so higher limits are available for productive work without allowing silent loops.

### Edge cases and negative paths
- [ ] Sparse or missing trace data reports insufficient pace evidence instead of a healthy default.
- [ ] Long but productive runs are distinguished from circular runs using terminal outcomes, tool diversity, diff/evidence progress, or other auditable signals.
- [ ] Pace scoring does not reward rushed incomplete work or penalize necessary validation on complex tickets.
- [ ] Max-turn failures create actionable evidence or intervention debt with the observed turn sink.

### Non-goals
- Blindly raising every role max-turn limit without pace evidence.
- Treating wall-clock speed as more important than correct, tested, documented outcomes.
- Replacing accuracy/value scores; pace should complement them.

### Observability, docs, and regressions
- [ ] Feature contracts document any new pace metric, score effect, runtime limit rule, or user-visible output.
- [ ] Design docs explain the metric formula, thresholds, and how it avoids gaming.
- [ ] Tests cover metric computation, empty evidence, max-turn classification, and any CLI/dashboard/export output changes.
- [ ] A final evidence note records baseline pace, changes made, post-change pace, residual bottlenecks, and follow-up tickets if optimum is not reached.

## Notes

Treat optimum as an evidence-backed operating target, not a fixed magic number. The first implementation slice should make pace visible and identify the biggest bottleneck before changing runtime limits.

## 2026-05-20 Progress

The first implementation slice makes pace visible without changing runtime
limits. `mars-harness scores export` now joins terminal scoring outcomes to
trace summaries by `job_id` and renders `docs/QUALITY_SCORE.md` Factory Pace
rows grouped by repo and role. The rows include job count, average turns,
average tool invocations, average LLM calls, average wall time, limit-stop
count, and a pace signal. Missing trace rows render as missing pace evidence
instead of implying healthy speed.

Remaining work:

- Run quality export against a live repo DB with recent traces and record the
  dated baseline in the active plan or validation report. Done for
  `demo-123-run12` and the non-static `demo-api-run1` API canary.
- The non-static canary identified same-repo same-role scheduled duplication as
  the first generic optimization target. Scheduler fire now checks the queue for
  active same-role work before enqueueing another scheduled job.
- The scheduler-skip rerun identified repo-local compiled binaries as the next
  generic containment trap. `go build .` can create an untracked root binary
  named after the repo; shell cleanup now permits removing that exact generated
  artifact while preserving ordinary deletion blocks.
- The first rerun after the cleanup fix identified a prior planning ambiguity:
  CEO and COO still tried duplicate `F-001` paths or duplicate starter scenario
  IDs before CTO ticketing. Generated bootstrap guidance now instructs CEO to
  hand off the existing feature-contract path and COO to rewrite the starter
  contract in place.
- The rerun after canonical planning guidance confirmed CEO, COO, and CTO now
  reach Engineer on the Task Notes API path, but Engineer generated a root
  binary named after the Go module (`task-notes-api`) rather than the repo
  directory. The cleanup exception now also allows untracked, binary-looking
  root artifacts named after the root `go.mod` module basename.
- The rerun after module-named cleanup confirmed the exception is not
  discoverable enough from generic blast-radius wording. Engineer never tried
  `rm task-notes-api`, so blast-radius errors now append the exact cleanup
  command when the oversized file is an untracked, root-level, binary-looking
  repo/module artifact.
- The rerun after cleanup hints confirmed Engineer can discover and run
  `rm task-notes-api`, then exposed long-running API validation as the next
  generic turn sink. Foreground `go run` spent a timeout, and shell `&`
  process management left port `8080` occupied. `shell_exec` now rejects shell
  background operators, treats early `background:true` exits as boot failures,
  and generated Engineer guidance forbids `cmd & PID=$!` snippets.
- The rerun after managed-background hardening confirmed Engineer can validate
  the API with `background:true`, probe `/health`, and kill the managed PID.
  The next turn sink is now validation build-output prevention: `go build -o`
  inside the target repo created a root binary trap. `shell_exec` blocks that
  output path before execution and keeps malformed shell payload errors visible
  even when an existing artifact has already made the repo dirty.
- The rerun after build-output prevention confirmed no repo-root binary is
  created and the target can stay clean with committed product progress. The
  next turn sink is now malformed port-token recovery: bare `:8080` commands
  repeat after validation errors. `shell_exec` now rejects bare port tokens in
  argv and single-token shell_command mode before process execution and points
  roles back to real server commands plus curl probes.
- Define calibrated thresholds after the baseline, not before it.
- Rerun the API canary to confirm bare-port command rejection keeps Engineer out
  of the `:8080` loop before claiming the optimization improved the live
  factory.
- Continue the representative validation matrix before making broad optimization
  claims.
