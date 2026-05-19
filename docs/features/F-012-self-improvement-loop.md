# F-012: Self-Improvement Loop

- Feature ID: F-012
- Goals: G-001, G-002, G-003
- Status: partially-passing
- Owner: CEO

## Business Logic

This feature contract is the durable home for business logic in this area. Product rules, workflow branches, state transitions, validations, permissions, scoring or trust decisions, routing rules, release classification, and user-visible outcomes must be documented here before or alongside implementation. Do not rely on ticket text or code comments as the only description of behavior.

## Step-By-Step Behavior

The scenarios below are the step-by-step BDD contract for this feature. Each scenario should describe the starting state, the action or event, and the observable outcome. When implementation changes business logic, update these steps and their evidence before claiming the feature is complete.

## Scenario Schedule

1. F-012-S001 - Telemetry classifies common failure categories and proposes deterministic remediation.
2. F-012-S002 - Repeated telemetry patterns and low scores triage into prompt, skill, process, guardrail, context, inference, manifest, tool-policy, or unknown targets.
3. F-012-S003 - Actionable telemetry creates or updates goals, observations, or intervention-debt tickets with dedupe keys.
4. F-012-S004 - Human interventions are detected and stored for reviewer analysis.
5. F-012-S005 - Bounded evolution proposals are rate-limited, scoped, and blocked from unsafe surfaces.
6. F-012-S006 - Repeated useful procedure becomes a compact skill or formalized tool instead of chat memory.
7. F-012-S007 - Source and generated target harnesses receive the same self-improvement doctrine when applicable.
8. F-012-S008 - Deployed harnesses keep raw telemetry local and export only opt-in anonymous aggregate reports.
9. F-012-S009 - Foundation telemetry collector intake triages repeated anonymous patterns into Mars Harness source work.
10. F-012-S010 - Deterministic remediation recipes are registered with applicability, safety, and operator actions before LLM repair.

## Scenarios

### F-012-S001: Failure Classification And Remediation

Given an agent run fails with context overflow, LLM unreachable, inference crash, model unavailable, tool timeout, loop, max turns, budget, manifest, ticket-gate, or unknown symptoms
When telemetry records the failure
Then it assigns a category and retry/remediation policy, and deterministic repair candidates can be selected by a remediation registry before an LLM repair role is considered

### F-012-S002: Improvement Target Triage

Given telemetry patterns or sparse/low role scores are detected
When triage evaluates them
Then the proposed target is one of prompt, skill, process, guardrail, context, inference, manifest, tool policy, or unknown with evidence

### F-012-S003: Goal Observation And Ticket Creation

Given triage produces actionable or weak evidence
When the harness records the proposal
Then target-owned actionable evidence creates or updates active goals or intervention-debt tickets, weak evidence becomes an observation, foundation-owned harness failures stay local telemetry or anonymous foundation telemetry, and repeated intervention-debt updates are compacted instead of inflating future context

### F-012-S004: Intervention Detection

Given human commits, comment-only changes, or squash merges occur after autonomous work
When intervention detection runs
Then real human correction is recorded as intervention evidence and non-interventions are ignored

### F-012-S005: Bounded Evolution Review

Given a reviewer proposes harness evolution
When rate limits, worsening checks, file scope, protected prompts, and proposal validity are evaluated
Then unsafe or out-of-scope evolution is blocked and valid scoped results are stored

### F-012-S006: Skill Or Tool Formalization

Given a repeated, risky, validation-heavy, or likely-to-recur process appears in human recovery or agent work
When self-improvement triage chooses the improvement surface
Then it prefers a compact skill or formalized tool with tests, glossary updates, and target mirroring when applicable instead of growing role prompts endlessly

### F-012-S007: Mirrored Self-Improvement Doctrine

Given the source harness adds or changes self-improvement rules
When initialized target harnesses should inherit them
Then generated prompts, skills, knowledge routes, docs, and consistency tests are updated in the same task

### F-012-S008: Anonymous Foundation Telemetry Export

Given a deployed harness has local raw telemetry in its repo-specific SQLite database
When anonymous reporting is disabled
Then no network call is made and disabled reporting is treated as healthy

Given anonymous reporting is enabled and a collector endpoint is configured
When the harness previews, exports, or sends a report
Then the payload contains only allowlisted aggregate fields and excludes raw traces, prompts, paths, remotes, ticket text, command output, raw error messages, commit SHAs, usernames, file paths, and source content

### F-012-S009: Foundation Collector Intake And Triage

Given a foundation collector receives anonymous report envelopes
When it validates and stores them
Then local dogfood uses a SQLite intake database and future hosted operation can use a Postgres-compatible backend without changing the deployed-harness protocol

Given repeated anonymous signatures appear across distinct anonymous report keys or harness versions
When foundation triage runs
Then it creates Mars Harness source work rather than intervention-debt tickets in the deployed target repository

### F-012-S010: Deterministic Remediation Registry

Given a failure signal names dirty workspace state, stale in-progress tickets, missing or invalid manifest state, missing generated docs, known doctor remediation, repeated scanner duplicates, missing dependency setup, or model artifact checksum mismatch
When deterministic remediation planning runs
Then the registry returns applicable recipes with stable IDs, candidate commands, safety classification, skipped reasons for operator or approval-required repairs, and next actions that are recorded in failed outcome details with trace IDs before generic telemetry retry is considered
And a ready auto-safe generated-docs recipe executes through the non-shell harness upgrade API, records applied/noop/failed execution evidence, and defers the generic telemetry retry path so deterministic repair runs first
And an auto-safe recipe without a registered executor does not suppress generic telemetry retry
And doctor reports known manifest and generated-docs recipe IDs with concrete fix commands before the same failures reach agent runtime

## Out of Scope

- Unbounded self-modification.
- Editing protected reviewer prompts through the reviewer role itself.
- Treating a noisy one-off failure as an autonomous product goal without review.

## Descoped Scenarios

None.

## Evidence

- F-012-S001: `go test ./internal/telemetry -run 'TestClassify|TestRetryable|TestRemediate'`
- F-012-S002: `go test ./internal/telemetry -run TestTriage`
- F-012-S003: `go test ./internal/telemetry -run TestRecordGoalFromProposal`, `go test ./internal/serve -run TestCreateInterventionDebt`, and `go test ./internal/tools -run TestTicketCreate_interventionDebtDedupeCompactsRepeatedUpdates`
- F-012-S004: `go test ./internal/evolution -run TestDetect`
- F-012-S005: `go test ./internal/evolution -run 'TestCanReview|TestValidateReviewResult|TestRecordEvolution|TestStore'`
- F-012-S006: `go test ./internal/tools -run TestToolCreate` plus docs-consistency checks for skill and tools glossary updates
- F-012-S007: `go test ./internal/scanner -run TestInit_success` and `go test ./internal/docsconsistency`
- F-012-S008: `go test ./internal/telemetry -run TestBuildAnonymousReport` and `go test ./internal/config -run TestLoad_envTelemetryOverrides`
- F-012-S009: `go test ./internal/foundationtelemetry`
- F-012-S010: `go test ./internal/remediation`, `go test ./internal/serve -run 'TestHandleJobFailed(RecordsDeterministicRemediation|ExecutesGeneratedDocs)|TestHandleRemediation(ExecutableReadyRecipe|AutoSafeWithoutExecutor|OperatorRecipe)'`, and `go test ./internal/doctor -run TestCheckDeterministicRemediationHealth`
