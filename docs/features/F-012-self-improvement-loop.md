# F-012: Self-Improvement Loop

- Feature ID: F-012
- Goals: G-001, G-002, G-003
- Status: partially-passing
- Owner: CEO

## Scenario Schedule

1. F-012-S001 - Telemetry classifies common failure categories and proposes deterministic remediation.
2. F-012-S002 - Repeated telemetry patterns and low scores triage into prompt, skill, process, guardrail, context, inference, manifest, tool-policy, or unknown targets.
3. F-012-S003 - Actionable telemetry creates or updates goals, observations, or intervention-debt tickets with dedupe keys.
4. F-012-S004 - Human interventions are detected and stored for reviewer analysis.
5. F-012-S005 - Bounded evolution proposals are rate-limited, scoped, and blocked from unsafe surfaces.
6. F-012-S006 - Repeated useful procedure becomes a compact skill or formalized tool instead of chat memory.
7. F-012-S007 - Source and generated target harnesses receive the same self-improvement doctrine when applicable.

## Scenarios

### F-012-S001: Failure Classification And Remediation

Given an agent run fails with context overflow, LLM unreachable, inference crash, model unavailable, tool timeout, loop, max turns, budget, manifest, ticket-gate, or unknown symptoms
When telemetry records the failure
Then it assigns a category and retry/remediation policy

### F-012-S002: Improvement Target Triage

Given telemetry patterns or sparse/low role scores are detected
When triage evaluates them
Then the proposed target is one of prompt, skill, process, guardrail, context, inference, manifest, tool policy, or unknown with evidence

### F-012-S003: Goal Observation And Ticket Creation

Given triage produces actionable or weak evidence
When the harness records the proposal
Then actionable evidence creates or updates active goals or intervention-debt tickets, while weak evidence becomes an observation

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

## Out of Scope

- Unbounded self-modification.
- Editing protected reviewer prompts through the reviewer role itself.
- Treating a noisy one-off failure as an autonomous product goal without review.

## Descoped Scenarios

None.

## Evidence

- F-012-S001: `go test ./internal/telemetry -run 'TestClassify|TestRetryable|TestRemediate'`
- F-012-S002: `go test ./internal/telemetry -run TestTriage`
- F-012-S003: `go test ./internal/telemetry -run TestRecordGoalFromProposal` and `go test ./internal/serve -run TestCreateInterventionDebt`
- F-012-S004: `go test ./internal/evolution -run TestDetect`
- F-012-S005: `go test ./internal/evolution -run 'TestCanReview|TestValidateReviewResult|TestRecordEvolution|TestStore'`
- F-012-S006: `go test ./internal/tools -run TestToolCreate` plus docs-consistency checks for skill and tools glossary updates
- F-012-S007: `go test ./internal/scanner -run TestInit_success` and `go test ./internal/docsconsistency`
