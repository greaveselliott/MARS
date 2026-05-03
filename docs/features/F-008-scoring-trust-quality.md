# F-008: Scoring Trust And Quality

- Feature ID: F-008
- Goals: G-001, G-003
- Status: partially-passing
- Owner: QA

## Scenario Schedule

1. F-008-S001 - Role outcomes are recorded and rolled into accuracy/value scores.
2. F-008-S002 - Trust levels change capabilities according to observed scores and manual overrides.
3. F-008-S003 - `scores` and `trust` commands expose repo and role state to operators.
4. F-008-S004 - `scores export` refreshes `docs/QUALITY_SCORE.md` from database evidence while preserving manual notes.
5. F-008-S005 - Low scores or telemetry patterns create deduped intervention-debt tickets.
6. F-008-S006 - Release notes and quality score distinguish shipped feature scenarios from enabler work.
7. F-008-S007 - Missing or sparse evidence is represented as insufficient evidence rather than healthy state.

## Scenarios

### F-008-S001: Outcome-Based Scores

Given role outcomes such as merged work, failed checks, timeouts, noops, or reverts are recorded
When score computation runs
Then rolling scores reflect real outcomes with minimum sample handling

### F-008-S002: Capability-Gated Trust

Given a role has observer, contributor, or autonomous trust
When scores improve, drop, or an override is set
Then capabilities change according to the trust contract and audit reason

### F-008-S003: Operator Score And Trust Visibility

Given stored outcomes or trust levels exist
When the user runs `mars-harness scores` or `mars-harness trust`
Then the command reports current role state for the requested repo

### F-008-S004: Repo-Visible Quality Score

Given a repo has score, telemetry, ticket, dogfood, guardrail, check, no-op, or human follow-up evidence
When `mars-harness scores export --repo <path>` runs
Then `docs/QUALITY_SCORE.md` is refreshed as the repo-visible quality artifact while manual notes are preserved

### F-008-S005: Intervention-Debt Creation

Given repeated telemetry failures or low score snapshots are detected
When triage runs
Then a deduped `kind: intervention-debt` ticket is created or updated through the canonical ticket path

### F-008-S006: Feature Evidence Classification

Given done tickets include feature or enabler work
When quality or release notes are generated
Then shipped scenario IDs are listed only for feature work with evidence and enabler work remains separate

### F-008-S007: Insufficient Evidence

Given database or evidence inputs are missing or sparse
When quality score export or score evaluation runs
Then the output says evidence is insufficient instead of implying the system is healthy

## Out of Scope

- Treating scores as vanity dashboard numbers with no behavioral effect.
- Fully autonomous business-priority scoring without human-readable evidence.
- Hiding manual trust overrides.

## Descoped Scenarios

None.

## Evidence

- F-008-S001: `go test ./internal/scoring`
- F-008-S002: `go test ./internal/trust`
- F-008-S003: command-level coverage is present through underlying scoring and trust stores; broader CLI E2E evidence is still pending
- F-008-S004: `go test ./internal/qualityscore`
- F-008-S005: `go test ./internal/serve -run TestCreateInterventionDebt` and `go test ./internal/telemetry -run TestTriage`
- F-008-S006: `go test ./internal/release -run TestPrepareClassifiesDeliveryEvidenceFromDoneTickets`
- F-008-S007: `go test ./internal/qualityscore -run TestExportMissingDatabasePreservesManualNotes` and `go test ./internal/scoring -run TestComputeScore_minimumSampleSize`
