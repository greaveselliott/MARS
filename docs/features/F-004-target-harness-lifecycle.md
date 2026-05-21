# F-004: Target Harness Lifecycle

- Feature ID: F-004
- Goals: G-001, G-002
- Status: partially-passing
- Owner: CEO

## Business Logic

This feature contract is the durable home for business logic in this area. Product rules, workflow branches, state transitions, validations, permissions, scoring or trust decisions, routing rules, release classification, and user-visible outcomes must be documented here before or alongside implementation. Do not rely on ticket text or code comments as the only description of behavior.

## Step-By-Step Behavior

The scenarios below are the step-by-step BDD contract for this feature. Each scenario should describe the starting state, the action or event, and the observable outcome. When implementation changes business logic, update these steps and their evidence before claiming the feature is complete.

## Scenario Schedule

1. F-004-S001 - `mars-harness init` creates a complete target harness in a git checkout.
2. F-004-S002 - Init writes mirrored operating-model docs, roles, guardrails, knowledge routes, tickets, and release defaults.
3. F-004-S003 - `mars-harness upgrade` fills missing defaults without overwriting user-owned configuration.
4. F-004-S004 - `mars-harness update check` reports tool, metadata, and operating-model drift without mutating state.
5. F-004-S005 - `doctor --repo` reports target harness health and active-plan hygiene.
6. F-004-S006 - Auto-harness scaffolding runs before register, scan, run, or start when the manifest is missing.
7. F-004-S007 - Generated target doctrine stays mirrored when source operating rules change.
8. F-004-S008 - `mars-harness eject` removes generated harness artifacts and the associated per-repo database only after explicit confirmation.

## Scenarios

### F-004-S001: Init Creates Harness

Given a user points Mars Harness at a git checkout without `.harness/manifest.yaml`
When `mars-harness init --repo <path>` runs
Then the repo receives a usable target harness with manifest, metadata, role prompts, guardrails, knowledge routes, docs, ticket directories, quality score, dogfood evidence report directory, version, changelog, and root ignore policy for host OS metadata such as `.DS_Store`

### F-004-S002: Init Mirrors Doctrine

Given a target harness is generated
When its docs and prompts are inspected
Then they include compact AGENTS guidance, goals, feature contracts, ticket metadata rules, exec-plan lifecycle, design-doc index, context glossary, quality score, release guidance, and self-improvement routing

### F-004-S003: Upgrade Is Non-Destructive

Given a target repo already has user-edited manifest, roles, guardrails, tickets, docs, or AGENTS guidance
When `mars-harness upgrade --repo <path>` runs
Then missing default files and safe host OS metadata ignore entries are added while existing user-owned files are preserved

### F-004-S004: Update Check Reports Drift

Given the installed tool or generated target metadata is stale
When `mars-harness update check --repo <path>` runs
Then the command reports current, latest, and recommended action in text or JSON without changing target files

### F-004-S005: Doctor Reports Target Health

Given a target repo has missing metadata, operating-model drift, or active-plan hygiene issues
When `mars-harness doctor --repo <path>` runs
Then the diagnosis names the failing area and gives actionable remediation

### F-004-S006: Auto-Scaffold Before Operations

Given register, scan, run, or start is invoked against a repo without a manifest
When the repo is a git checkout
Then the command runs the same scaffold path as init before continuing

### F-004-S007: Mirrored Rule Evolution

Given a source operating rule changes
When the change applies to initialized target harnesses
Then source docs, generated defaults, role guidance, knowledge routes, and tests are updated in the same task or the blocker is recorded

Given generated role guidance describes BDD scenarios, DocSync metadata, validation, review, dogfood, or release readiness
When a fresh target is initialized
Then Engineer guidance distinguishes scenario IDs from feature-contract paths, shows the structured `MarsDocSync` block form, QA and Security guidance names terminal-only review convergence after clean read plus validation evidence, and QA, Security, Dogfood, and Release Manager guidance treat `docsync_audit` `FAIL:` output as a blocker instead of approving stale source documentation

### F-004-S008: Target Harness Eject Kill Switch

Given a target repo has generated Mars Harness artifacts and a repo-scoped SQLite database
When `mars-harness eject --repo <path>` runs without `--apply`
Then the command reports the files and database it would remove without mutating the target

Given the same target repo
When `mars-harness eject --repo <path> --apply --confirm repo` runs
Then `.harness/`, generated harness docs, generated ticket/feature/release/report defaults including dogfood reports, root generated guidance/version files, and the associated per-repo database are removed without rewriting git history

## Out of Scope

- Silently overwriting target-owner prompt or manifest edits.
- Migrating every historical target file automatically without review.
- Making the harness the target of its own agents during a run.

## Descoped Scenarios

None.

## Evidence

- F-004-S001: `go test ./internal/scanner -run TestInit_success`
- F-004-S002: `go test ./internal/scanner -run TestInit_success`
- F-004-S003: `go test ./internal/scanner -run TestUpgrade_preservesUserConfiguredManifestAndPrompts`
- F-004-S004: `go test ./internal/updatecheck`
- F-004-S005: `go test ./internal/doctor -run TestCheck`
- F-004-S006: `go test ./internal/scanner -run TestEnsureHarness`
- F-004-S007: `go test ./internal/scanner -run TestInit_success` and `go test ./internal/docsconsistency`
- F-004-S008: `go test ./internal/scanner -run TestEject`
