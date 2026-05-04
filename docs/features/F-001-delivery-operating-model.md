# F-001: Delivery Operating Model

- Feature ID: F-001
- Goals: G-001, G-002, G-003
- Status: passing
- Owner: CEO

## Business Logic

This feature contract is the durable home for business logic in this area. Product rules, workflow branches, state transitions, validations, permissions, scoring or trust decisions, routing rules, release classification, and user-visible outcomes must be documented here before or alongside implementation. Do not rely on ticket text or code comments as the only description of behavior.

## Step-By-Step Behavior

The scenarios below are the step-by-step BDD contract for this feature. Each scenario should describe the starting state, the action or event, and the observable outcome. When implementation changes business logic, update these steps and their evidence before claiming the feature is complete.

## Scenario Schedule

1. F-001-S001 - Goal, BDD feature, and active plan are linked.
2. F-001-S002 - Feature ticket completion requires BDD scenario evidence.
3. F-001-S003 - Generated target harness mirrors goals, BDD, plan, ticket, role, and knowledge guidance.
4. F-001-S004 - Existing target drift is reported without overwriting user-owned files.
5. F-001-S005 - Release notes and quality score distinguish shipped scenarios from enabler work.
6. F-001-S006 - Telemetry can create or update goals/observations from structured evidence.
7. F-001-S007 - Business logic is documented step by step in feature contracts.
8. F-001-S008 - Code changes declare associated documentation and keep it current.
9. F-001-S009 - Source-wide docsync audit maps code to architecture and feature documentation.
10. F-001-S010 - Documentation sync has a universal operating model for source and generated targets.
11. F-001-S011 - CLI changes synchronize mirrored tools, repo shortcuts, generated doctrine, and skills.

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

### F-001-S007: Business Logic Is First-Class BDD

Given business logic changes through a product rule, workflow branch, state transition, validation, permission, scoring rule, routing rule, or user-visible outcome
When a planner, engineer, reviewer, or maintainer records or implements that behavior
Then the matching `docs/features/F-NNN-*.md` contract documents the behavior step by step with Business Logic, Step-By-Step Behavior, Given/When/Then scenarios, and evidence before the feature is claimed complete

### F-001-S008: No Stale Documentation

Given code is created or materially changed
When an agent prepares the change for review or commit
Then the changed code carries a top-of-file `MarsDocSync` metadata block with a `docs:` array listing associated documentation, and those docs are updated in the same change or explicitly checked as still current

### F-001-S009: Source-Wide Docsync Audit

Given a foundation or deployed harness source tree is audited
When `mars-harness docsync audit --repo .` or the mirrored `docsync_audit` tool runs
Then every source file under the audited source roots declares a top-of-file `MarsDocSync` block with a `docs:` array, every referenced doc exists, and every file includes the documentation required by the canonical code map

### F-001-S010: Universal Documentation Sync Operating Model

Given an agent changes source, tools, generated target defaults, role behavior, CLI behavior, architecture, or business logic
When the agent prepares completion evidence
Then it follows the documented documentation-sync operating model: read changed-file `MarsDocSync` metadata, classify the documentation impact, update or verify the listed docs, repair metadata or the canonical map when ownership changes, run docsync evidence, and record which docs changed or remained current

### F-001-S011: CLI Tool And Skill Synchronization

Given a `mars-harness` CLI command, flag, output contract, repo behavior, mutability expectation, or recurring workflow changes
When the change is prepared for completion
Then the `mars_harness_cli` reference, repo shortcut map, generated target doctrine, and any skills that name the affected workflow are updated or explicitly checked as current, and CLI sync evidence is recorded

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
- F-001-S007: `go test ./internal/docsconsistency -run TestFeatureContractsDeclareRequiredFields`
- F-001-S008: `go test ./internal/docsconsistency -run TestOperatingModelCodeFilesDeclareDocSyncMetadata`
- F-001-S009: `go test ./internal/docsync ./internal/docsconsistency -run 'TestDocSync|TestOperatingModelCodeFilesDeclareDocSyncMetadata'` and `mars-harness docsync audit --repo .`
- F-001-S010: `go test ./internal/docsconsistency -run TestAD074OperatingModelArtifactsExist` verifies the architecture and universal operating model are documented.
- F-001-S011: `go test ./cmd/mars-harness -run TestMarsHarnessCLI` verifies the live Cobra command tree, `mars_harness_cli` reference, and repo shortcut map stay synchronized.
