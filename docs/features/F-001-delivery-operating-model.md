# F-001: Delivery Operating Model

- Feature ID: F-001
- Goals: G-001, G-002, G-003
- Status: passing
- Owner: CEO

## Scenario Schedule

1. F-001-S001 - Goal, BDD feature, and active plan are linked.
2. F-001-S002 - Feature ticket completion requires BDD scenario evidence.
3. F-001-S003 - Generated target harness mirrors goals, BDD, plan, ticket, role, and knowledge guidance.
4. F-001-S004 - Existing target drift is reported without overwriting user-owned files.
5. F-001-S005 - Release notes and quality score distinguish shipped scenarios from enabler work.
6. F-001-S006 - Telemetry can create or update goals/observations from structured evidence.

## Scenarios

### F-001-S001: Goal, Feature, And Plan Alignment

Given at least one active goal exists
When the CEO updates the single active exec plan
Then the plan references an active goal, a BDD feature contract, a hypothesis, success evidence, falsification evidence, scenario schedule, current failing scenario, walking skeleton slice, and learning or MVP outcome

### F-001-S002: Feature Ticket Evidence Gate

Given a feature ticket maps to a BDD scenario
When the engineer attempts to move it to done
Then the ticket must include non-empty `bdd_scenarios`, `end_to_end_evidence: required`, non-empty `evidence_links`, and `verified_by`

### F-001-S003: Generated Target Mirror

Given `mars-harness init` runs on a target repo
When the scaffold is inspected
Then goals, feature contracts, AD-074, role prompts, ticket metadata, exec-plan metadata, quality guidance, and knowledge routes mirror the source doctrine

### F-001-S004: Non-Destructive Target Drift

Given an existing target has old operating-model docs
When `mars-harness update check` or `doctor --repo` runs
Then stale or missing operating-model artifacts are reported without overwriting user-owned files

### F-001-S005: Enabler Classification

Given an enabler ticket completes
When release notes or quality score are updated
Then they classify it as enabler work and do not claim shipped feature scenarios

### F-001-S006: Evidence-Derived Goals

Given telemetry, dogfood, quality, or feedback contains structured actionable evidence
When triage runs
Then it creates or updates an active goal or observation with source, confidence, dedupe key, and review trigger

## Out of Scope

- Custom Gherkin parsing.
- Fully autonomous goal scoring and weighting.
- Overwriting user-owned target docs during update.

## Descoped Scenarios

None.

## Evidence

- F-001-S001: `go test ./internal/docsconsistency -run TestActivePlanReferencesActiveGoalAndFeatureContract`
- F-001-S002: `go test ./internal/serve -run TestValidateEngineerTicketGate`
- F-001-S003: `go test ./internal/scanner -run TestInit_success`
- F-001-S004: `go test ./internal/updatecheck -run TestRun_reportsOperatingModelDrift`
- F-001-S005: docs-consistency checks for `docs/QUALITY_SCORE.md` and `docs/tickets/README.md`
- F-001-S006: `go test ./internal/telemetry -run TestRecordGoalFromProposal`
