# Mars Parity Supersession Plan

**Status:** Active
**Created:** 2026-05-02
**Updated:** 2026-05-02
**Owner:** Mars Harness
**Source:** User request to compare `../mars` with this repository and plan the work required for Mars Harness to supersede Mars.
**Supporting audit:** [mars-meta-harness-relevance-audit.md](../../references/mars-meta-harness-relevance-audit.md)

## Purpose

Mars Harness should supersede the Mars repository's Cursor-based meta-harness. That does not mean copying Mars exactly. It means preserving the useful operating system Mars evolved through real autonomous delivery, then rebuilding those patterns as first-class, local, strict-trunk Mars Harness capabilities.

The target end state is:

- Mars Harness can manage Mars-class repositories without Cursor automations.
- Mars Harness retains strict trunk-based delivery as the canonical workflow.
- Mars Harness exposes the same or better planning, quality, intervention, dogfood, and orchestration loops that Mars currently encodes in docs, Cursor rules, GitHub Actions, scripts, and automation prompts.
- Mars Harness remains the product and system of record for future autonomous delivery.

## Sources Reviewed

This plan is based on a direct repository comparison between:

- `../mars/AGENTS.md`
- `../mars/template/AGENTS.md`
- `../mars/.cursor/rules/*.mdc`
- `../mars/docs/automations/BOTS.md`
- `../mars/docs/automations/prompts/*.md`
- `../mars/docs/design-docs/index.md`
- `../mars/docs/design-docs/automation-team.md`
- `../mars/docs/design-docs/agent-context-model.md`
- `../mars/docs/design-docs/conversation-as-system-record.md`
- `../mars/docs/design-docs/self-correcting-ci.md`
- `../mars/scripts/*`
- `../mars/.github/workflows/*`
- `AGENTS.md`
- `.cursor/rules/*.mdc`
- `docs/design-docs/index.md`
- `docs/design-docs/tenets.md`
- `docs/design-docs/trigger-orchestration.md`
- `docs/design-docs/self-improvement.md`
- `docs/design-docs/dogfood-and-decisions.md`
- `docs/design-docs/scoring-system.md`
- `docs/design-docs/guardrails.md`
- `docs/design-docs/context-efficiency.md`
- `internal/scanner/init.go`
- `internal/agent`, `internal/serve`, `internal/scoring`, `internal/trust`, `internal/evolution`, `internal/tools`, `internal/guardrails`, `internal/safety`, `internal/scheduler`, `internal/doctor`, `internal/models`, and related packages
- `.github/workflows/*`

## Executive Assessment

Mars Harness is philosophically close to Mars but not yet operationally equivalent.

| Area | Current parity | Assessment |
| --- | ---: | --- |
| Core philosophy | High | The nine tenets align well with Mars's automation beliefs. |
| Repo documentation | Medium | Harness has strong product docs, but Mars has richer operating-model docs. |
| Generated target guidance | Medium-low | Mars template `AGENTS.md` is more specific and more actionable than Harness-generated target scaffolding. |
| Role topology | Medium | Harness has more native roles, but Mars has a cleaner domain-shaped 6-bot model with payload modes. |
| Automation registry | Low | Mars has `docs/automations/BOTS.md`; Harness has manifests but no comparable human-readable role registry. |
| Trigger orchestration | Medium | Harness has queue/scheduler primitives; Mars has more mature event routing, caps, and watchdog behavior. |
| Intervention debt | Low | Mars makes human intervention a first-class automation debt signal; Harness has the design direction but not the full loop. |
| Quality score | Medium-low | Harness has scoring infrastructure; Mars has more repo-visible quality artifacts and regression scripts. |
| Dogfood coverage | Medium-low | Harness has dogfood design and recent ticket-drain fixes; Mars has broader matrix-style dogfood and generated-app checks. |
| Deterministic remediation | Low | Mars uses deterministic CI and maintenance scripts before LLM work; Harness should absorb these as native recipes. |
| Local inference and setup | High | Harness is ahead: local models, hardware detection, setup, doctor, and runtime packaging are product concerns. |
| Safety, guardrails, trust | Medium-high | Harness is ahead architecturally, but all enforcement paths must stay mechanically wired. |
| Release/distribution | Medium | Harness has binary release direction; Mars has more mature product release hygiene. |

Bottom line: Mars Harness has the better foundation for a standalone product, but Mars currently has the better proven autonomous operating model. Supersession requires importing Mars's operating discipline into Harness-native primitives.

## Important Differences

### Mars Is Cursor-Native

Mars's current meta-harness is distributed across:

- Cursor automations
- GitHub Actions
- PR events
- branch gates
- auto-merge workflows
- shell and Node scripts
- repo docs and `.cursor/rules`

Mars Harness must not inherit the branch/PR delivery model. It should translate the concepts into strict trunk:

- PR comments become run comments, ticket comments, trace notes, status checks, or GitHub check-run output.
- PR review loops become reviewer jobs against `main` commits or pending ticket work.
- auto-merge becomes "commit and push `main` after completed step".
- branch hygiene becomes clean-main hygiene.
- stuck PR detection becomes stuck job, stuck ticket, stuck in-progress, and failed-check detection.

### Mars Harness Is Product-Native

Harness already has primitives Mars lacks as a product:

- local inference management
- hardware profiles
- SQLite queue
- trust levels
- scoring database
- trace recording
- tool registry
- guardrail and safety packages
- bundle generation
- dashboard
- doctor checks
- setup flow

The parity work should therefore avoid recreating Mars's Cursor setup and should instead promote Mars's lessons into these native packages.

## Supersession Principles

1. **Strict trunk remains canonical.** No default feature branches, PR creation, PR merge, or no-push-main guidance returns.
2. **Deterministic first, LLM second.** Known failures should be fixed by recipes, checks, and policy before spending model turns.
3. **Repo-visible state wins.** Dashboards and databases are useful, but plans, decisions, tickets, traces, and quality summaries must have durable repo artifacts.
4. **Every intervention creates debt.** Any human rescue, failed job, guardrail block, regression, or handoff without completion should create or update a ticket.
5. **Finish active work before starting new work.** In-progress tickets are not a holding pen. They are priority work unless explicitly blocked.
6. **Modes beat role sprawl.** Keep role memory coherent by using domain-shaped roles with payload modes where practical.
7. **Generated targets are first-class.** `mars-harness init` should produce target repo instructions and docs at least as good as Mars's template guidance.
8. **Optional GitHub is telemetry, not delivery.** Webhooks, checks, comments, and statuses are acceptable; PR delivery is not the default.
9. **Mars-specific product rules stay in Mars.** Harness should import the meta-harness pattern, not hard-code Mars's SaaS generator constraints globally.

## Comparison Findings

### 1. Agent Entrypoint And Generated Guidance

Mars has two strong entrypoint documents:

- root `AGENTS.md` for agents maintaining Mars itself
- `template/AGENTS.md` for agents working inside generated apps

The template file is especially important. It gives generated projects a clear local architecture, constraints, auth/database/email rules, testing discipline, template-sync guidance, and dogfood QA expectations.

Mars Harness has a strong root `AGENTS.md`, but its generated target guidance is thinner. `internal/scanner/init.go` creates useful docs and tickets, yet it does not match the specificity or operating value of the Mars template.

Gap:

- Generated target repos need a richer `AGENTS.md`.
- Harness-generated docs should include architecture constraints, run/test commands, ticket workflow, decision logging, local-first rules, and dogfood expectations.
- The generated guidance must be generic enough for any target repo while still concrete enough to steer agents.

### 2. Cursor Rules And Governance

Mars has mature `.cursor/rules` for:

- documentation discipline
- intervention-to-automation
- knowledge routing
- scaffold testing
- branch cleanliness
- no-push-main workflow

Harness has strict-trunk equivalents for some of these, including documentation discipline and intervention-to-automation. However, the Harness rules are less detailed and are not yet backed by the same mechanical checks.

Gap:

- Harness needs active plan hygiene thresholds.
- Harness needs stronger generated knowledge routes.
- Harness needs docs consistency checks that cover active plans, stale decisions, and generated target instructions.
- Branch-specific rules must remain excluded or translated into clean-main rules.

### 3. Role Model

Mars's modern automation design is the 6-bot Agent Context Model:

- Planner
- Engineer
- Reviewer
- Maintainer
- End-to-End Tester
- Orchestrator

The key insight is not the number six. The key insight is domain memory plus payload-routed modes. For example, Engineer handles next-ticket work, pipeline fixes, and review-comment fixes without splitting into many unrelated memories.

Harness currently defaults to more explicit roles:

- CEO
- CTO
- COO
- Engineer
- QA
- Security
- Dependency Manager
- Release Manager
- Dogfood
- Pipeline Fixer
- Janitor

This is understandable for a product manifest, but it risks role sprawl and fragmented memory.

Gap:

- Harness needs a canonical operating-model doc equivalent to Mars's automation-team and agent-context-model docs.
- Harness should either adopt the 6-domain model directly or map its 11 roles to 6 canonical domains with modes.
- Role modes should become explicit payload/config concepts in manifests, traces, scoring, and generated prompts.

### 4. Automation Registry

Mars has `docs/automations/BOTS.md`, a human-readable source of truth for deployed automations, including:

- bot name
- domain
- triggers
- mode payloads
- model
- tools/integrations
- secrets/environment
- status

Harness has `.harness/manifest.yaml` generation and design docs, but no equivalent repo-visible registry.

Gap:

- Harness needs a role registry artifact, generated and checked against manifests.
- The registry should list role modes, schedules, event triggers, trust level, tool capabilities, guardrails, model routing, and scoring signals.
- The registry should be present both in Harness docs and generated target bundles.

### 5. Trigger Orchestration

Mars has extensive GitHub Actions orchestration:

- schedule-driven planner and maintenance jobs
- path-gated reviewers
- dependency bot flows
- pipeline fixer dispatch
- end-to-end tester dispatch
- orchestrator watchdog
- daily caps and concurrency groups
- PR comment and review-comment routing

Harness has a native scheduler and queue, which is the right long-term location, but the behavior is not yet as mature.

Gap:

- Harness needs payload-mode routing.
- Harness needs native caps, concurrency groups, stuck-job detection, and priority rules.
- Harness needs event triggers for failed checks, dogfood failures, stale in-progress tickets, intervention debt, and quality regression.
- Harness needs an Orchestrator process or role that continuously surveys the system for unattended failure states.

### 6. Intervention Debt And Self-Improvement

Mars treats any human intervention as automation debt. The loop is explicit:

- record the intervention
- create a ticket
- route it to Planner/Engineer
- update automation so the same human step is less likely next time

Harness has `self-improvement.md`, `learnings`, `evolution`, `record_decision`, trust, and scoring primitives. It is conceptually ahead but not fully closed mechanically.

Gap:

- Harness needs a first-class `intervention-debt` ticket type.
- Failed terminal reasons, guardrail blocks, human follow-up commits, reverted commits, repeated ticket-gate failures, and manual run interruptions should create or update intervention-debt tickets.
- Planner should treat intervention debt as high-priority work.
- Evolution should be bound to those tickets and trace IDs.

### 7. Tickets And Active Work

Mars's docs and scripts enforce more mature plan hygiene. Harness has recently added in-progress draining and dogfood ticket dedupe decisions, but the user-observed failure remains central: the dogfood tester can create many tickets, and the engineer can accumulate many in-progress files without finishing them.

Gap:

- In-progress tickets must be the highest default priority.
- Tickets should have explicit owner, blocker, last-attempt, and failure-summary metadata.
- A ticket moved to in-progress must either complete, be returned to backlog with a blocker note, or create a dependency ticket.
- Dogfood-generated tickets should be grouped, deduped, severity-ranked, and capped per run.
- The engineer should not hand off unfinished work without recording the blocker and scheduling/fixing the blocker.

### 8. Quality Score

Mars has repo-visible quality artifacts and quality regression scripts. Harness has scoring packages and commands underway, but the score is not yet as visible in the repo.

Gap:

- Harness needs a generated `docs/QUALITY_SCORE.md` or equivalent.
- Scoring should summarize role health, stuck tickets, failed dogfood, guardrail blocks, intervention debt, check pass/fail rate, no-op runs, and human follow-up rate.
- Score regression should trigger Orchestrator/Planner work.

### 9. Dogfood Matrix

Mars has strong dogfood coverage for generated apps and scaffold combinations. Harness has dogfood design and dry-run workflows, but needs a more explicit product matrix.

Gap:

- Harness dogfood should cover setup, init, register, run, start, serve, scan, doctor, trust, scores, dashboard, local inference, optional GitHub helpers, and generated target operation.
- Harness should dogfood against `../mars` as a target repo before claiming supersession.
- Fake-LLM deterministic integration tests should cover the full loop in CI.
- Optional live local-model dogfood can run separately from deterministic CI.

### 10. Deterministic Remediation

Mars learned to avoid asking an LLM to solve problems that deterministic scripts can solve. Examples include changeset guarantee checks, auto-merge maintenance, rebase helpers, stale plan checks, and dependency maintenance helpers.

Harness has some deterministic checks, but it should absorb this pattern more broadly.

Gap:

- Add native remediation recipes for known failure classes.
- Run recipes before assigning judgment work to Engineer or Pipeline Fixer.
- Record recipe attempts in traces and scoring.
- Promote successful repeated recipes into permanent doctor checks or setup behavior.

### 11. Generated And Fresh Docs

Mars has stronger doc freshness conventions and generated references. Harness has generated docs directory scaffolding but fewer generated artifacts.

Gap:

- Harness should generate and check role registry, quality score, package map, tool inventory, model inventory, and optional target repo knowledge routes.
- `doctor` should flag stale generated artifacts.

### 12. Setup And Zero Config

Harness is ahead here. Mars does not provide the same local inference and hardware product surface. Recent Harness work on automatic performance tuning also pushes this in the right direction.

Gap:

- Setup and doctor should prove the selected inference config is effective, not just present.
- Model and llama artifacts must stay pinned and checksum-verified.
- GitHub setup must remain optional and must not be described as complete unless validated.

## Workstreams

### A. Canonical Harness Operating Model

Create the Harness-native equivalent of Mars's automation-team and agent-context-model docs.

Tasks:

- [ ] Add `docs/design-docs/harness-operating-model.md`.
- [ ] Define the canonical domain roles: Planner, Engineer, Reviewer, Maintainer, End-to-End Tester, and Orchestrator.
- [ ] Map current roles to canonical domains and modes:
  - CEO -> Planner strategy mode
  - CTO -> Planner architecture mode or Reviewer architecture mode
  - COO -> Planner execution mode
  - Engineer -> Engineer next-ticket mode
  - QA -> Reviewer functional mode
  - Security -> Reviewer security mode
  - Dependency Manager -> Maintainer dependencies mode
  - Release Manager -> Maintainer release mode
  - Dogfood -> End-to-End Tester dogfood mode
  - Pipeline Fixer -> Engineer pipeline-fix mode
  - Janitor -> Maintainer hygiene mode
- [ ] Decide whether manifests expose the 6 roles plus modes, or retain explicit roles while declaring the 6-domain model as canonical.
- [ ] Update `docs/design-docs/trigger-orchestration.md` to reflect the canonical model.
- [ ] Index the new design doc in `docs/design-docs/index.md`.

Acceptance:

- The role model is explicit, strict-trunk native, and no longer spread across scattered docs.
- Current 11-role manifests have a migration path that does not break existing bundles.

### B. Role Registry And Automation Inventory

Build a Harness equivalent of `docs/automations/BOTS.md`.

Tasks:

- [ ] Add `docs/roles/ROLES.md` or `docs/automations/ROLES.md` as the repo-visible registry.
- [ ] Include role, modes, trigger sources, schedules, tools, trust level, guardrails, model routing, score signals, and escalation behavior.
- [ ] Generate a target repo registry during `mars-harness init`.
- [ ] Add a consistency check that generated manifests and the registry agree.
- [ ] Surface registry health in `mars-harness doctor`.

Acceptance:

- A human or agent can answer "what autonomous roles exist and what can they do?" from a single checked-in file.
- The registry is mechanically checked and not merely aspirational.

### C. Conversation And Decision Record

Import Mars's "conversation as system record" discipline into Harness.

Tasks:

- [ ] Add `docs/design-docs/conversation-as-system-record.md`, adapted for strict trunk.
- [ ] Update root and generated `AGENTS.md` guidance to require persistent artifacts for plans, decisions, investigations, quality findings, and completed work.
- [ ] Update `.cursor/rules/documentation-discipline.mdc` with active-plan hygiene thresholds.
- [ ] Add an active-plan hygiene checker in Go or shell with no new runtime dependency.
- [ ] Wire the checker into CI and `mars-harness doctor`.

Acceptance:

- Significant agent conversations create repo artifacts.
- Stale active plans, trailing verification notes, and unresolved TBDs are detected automatically.

### D. Intervention Debt Loop

Close the self-improvement loop Mars already practices.

Tasks:

- [ ] Define an `intervention-debt` ticket schema.
- [ ] Extend ticket creation and scanner dedupe to handle intervention-debt tickets.
- [ ] Create or update intervention-debt tickets on:
  - non-success terminal agent result
  - guardrail block
  - repeated tool loop
  - human follow-up commit touching the same files
  - revert of an agent commit
  - ticket moved to in-progress without completion after a threshold
  - manual stop or timeout
- [ ] Prioritize intervention-debt tickets above ordinary backlog work unless an explicit severity override exists.
- [ ] Link intervention-debt tickets to traces, commits, role, repo, and score events.
- [ ] Feed eligible intervention-debt tickets into bounded evolution.

Acceptance:

- Human rescue work automatically becomes visible automation debt.
- Evolution work is trace-linked and grounded in real failures.

### E. Active Ticket Drain And Blocker Repair

Make the user-observed "lots of in-progress tickets" failure impossible to ignore.

Tasks:

- [ ] Add ticket metadata for owner, last attempt, blocker, blocked-by, trace ID, and next action.
- [ ] Enforce a queue policy: in-progress tickets are considered before backlog tickets.
- [ ] Prevent an Engineer run from opening a new backlog ticket while eligible in-progress tickets remain.
- [ ] Require every unfinished in-progress ticket to end with one of:
  - completion and move to done
  - blocker note and return to backlog
  - dependency ticket creation and explicit blocked state
  - guardrail-blocked intervention-debt ticket
- [ ] Add a stale in-progress scanner and Orchestrator trigger.
- [ ] Cap dogfood ticket creation per run by severity, grouping, and dedupe key.

Acceptance:

- In-progress cannot grow unbounded from normal autonomous operation.
- Blockers are actively repaired rather than becoming silent handoffs.

### F. Quality Score As Repo Artifact

Make Harness scoring as visible as Mars's quality documentation.

Tasks:

- [ ] Add `mars-harness scores export --repo <path>` or equivalent.
- [ ] Generate `docs/QUALITY_SCORE.md`.
- [ ] Include role score, recent outcomes, stuck tickets, failed dogfood, guardrail blocks, intervention debt, check results, no-op runs, and human follow-up rate.
- [ ] Add a quality-regression detector.
- [ ] Trigger Planner/Orchestrator when quality regresses.
- [ ] Add dashboard links to the same source data without making the dashboard the source of truth.

Acceptance:

- Repo readers can see current autonomous health without opening SQLite or the dashboard.
- Score regressions create work automatically.

### G. Native Orchestrator And Event Router

Replace GitHub Actions orchestration patterns with native Harness orchestration.

Tasks:

- [ ] Add an Orchestrator loop that surveys queue, tickets, scores, guardrails, traces, and checks.
- [ ] Add event routing for failed checks, dogfood failures, stale in-progress tickets, quality regression, intervention debt, dependency alerts, and release readiness.
- [ ] Add concurrency groups and daily caps to queue scheduling.
- [ ] Add payload-mode support to jobs and role prompts.
- [ ] Add watchdog detection for stuck jobs and silent no-op runs.

Acceptance:

- Harness can reproduce Mars's automation routing without Cursor or GitHub Actions as the primary scheduler.
- Failure states are converted into prioritized jobs.

### H. Deterministic Remediation Recipes

Promote Mars's script-first lesson into Harness-native recipes.

Tasks:

- [ ] Add a remediation package for deterministic fixes.
- [ ] Start with recipes for:
  - dirty working tree before run
  - stale in-progress tickets
  - missing or invalid manifest
  - missing generated docs
  - failed doctor checks with known remediation
  - repeated scanner duplicate tickets
  - failed tests caused by missing dependency setup
  - model artifact checksum mismatch
- [ ] Run recipes before LLM repair jobs where safe.
- [ ] Record recipe attempts in traces, scores, and quality export.
- [ ] Promote repeated successful recipes into doctor/setup checks.

Acceptance:

- Known failure classes are handled deterministically before model tokens are spent.
- LLM agents focus on judgment, not mechanical cleanup.

### I. Dogfood Matrix And Supersession Benchmark

Define what it means for Harness to supersede Mars.

Tasks:

- [ ] Add `docs/design-docs/dogfood-matrix.md`.
- [ ] Define a Harness product dogfood matrix covering setup, init, register, run, start, serve, scan, doctor, trust, scores, dashboard, local inference, optional GitHub, and upgrade.
- [ ] Add deterministic fake-LLM integration tests for the full loop.
- [ ] Add a dogfood target profile for `../mars`.
- [ ] Run Harness against Mars as a target repo in observer mode first.
- [ ] Graduate to contributor mode once guardrails, ticket policy, and quality score are trusted.
- [ ] Document supersession results in a completed execution plan.

Acceptance:

- Mars Harness can operate on the Mars repo without depending on Cursor automations.
- Supersession is supported by repeatable tests and recorded dogfood evidence.

### J. Generated Target Parity

Make `mars-harness init` create target repo guidance as useful as Mars's template.

Tasks:

- [ ] Expand generated `AGENTS.md` with architecture, workflow, run/test, ticket, decision, guardrail, and dogfood sections.
- [ ] Generate target `docs/design-docs/index.md` with explicit decision-record instructions.
- [ ] Generate target `docs/exec-plans/README.md` with active-plan hygiene.
- [ ] Generate target `docs/tickets/README.md` with direct `main` completion workflow and in-progress drain rules.
- [ ] Generate target knowledge routes from repo scan.
- [ ] Add golden tests for generated bundle contents.
- [ ] Update `mars-harness upgrade` to safely refresh generated harness plumbing with dry-run and backup behavior.

Acceptance:

- A newly initialized target repo is immediately understandable to Codex, Cursor, Mars Harness agents, and humans.
- The generated guidance is strict-trunk native and does not mention PR delivery as the default.

### K. Optional GitHub Integration

Keep GitHub as optional infrastructure, not the delivery model.

Tasks:

- [ ] Implement optional webhook ingestion for checks, statuses, and comments.
- [ ] Implement optional check-run/status/comment publishing.
- [ ] Enrich webhook payloads with run ID, role, mode, repo, branch, commit, failing jobs, and trace ID.
- [ ] Exclude default PR creation and merge tools from the registry.
- [ ] Add fake GitHub server integration tests.

Acceptance:

- GitHub improves observability and event intake without becoming the default workflow.
- No generated default instructs agents to open or merge PRs.

### L. Setup, Doctor, Models, And Release Hardening

Preserve Harness's product advantage over Mars.

Tasks:

- [ ] Ensure setup chooses the best local inference configuration automatically.
- [ ] Ensure doctor explains the effective inference config and measured throughput.
- [ ] Pin model artifacts by immutable revision and SHA256.
- [ ] Pin llama.cpp artifacts and verify checksums.
- [ ] Keep `doctor --json` parseable on stdout with logs only on stderr.
- [ ] Add doctor checks for docs hygiene, role registry, scoring/trust DB, guardrail loading, model checksum state, ticket health, and optional GitHub config.
- [ ] Strengthen release workflow with dogfood, coverage, docs consistency, and active-plan hygiene.

Acceptance:

- Zero-config local setup remains true.
- Doctor can prove readiness rather than merely list configuration.

## Milestones

### M0 - Plan Recorded

- [x] Commit this plan to `main`.
- [x] Push `main`.

### M1 - Documentation And Role Parity

- [ ] Complete workstreams A, B, and C.
- [ ] Generated target docs are upgraded.
- [ ] CI and doctor check documentation hygiene.

### M2 - Intervention And Ticket Closure

- [ ] Complete workstreams D and E.
- [ ] In-progress ticket accumulation is mechanically controlled.
- [ ] Human interventions become trace-linked tickets.

### M3 - Quality And Orchestration

- [ ] Complete workstreams F and G.
- [ ] Quality score is repo-visible.
- [ ] Orchestrator detects unattended failure states.

### M4 - Deterministic Repair And Dogfood

- [ ] Complete workstreams H and I.
- [ ] Full fake-LLM loop passes in CI.
- [ ] Harness can run against `../mars` in observer mode.

### M5 - Generated Target Supremacy

- [ ] Complete workstream J.
- [ ] New target repos have Mars-template-level guidance from day one.

### M6 - Optional Integration And Release Hardening

- [ ] Complete workstreams K and L.
- [ ] Optional GitHub integrations are status/check/comment oriented.
- [ ] Release process proves dogfood and parity checks.

### M7 - Mars Supersession Trial

- [ ] Register `../mars` as a target repo.
- [ ] Run Planner, Engineer, Reviewer, End-to-End Tester, Maintainer, and Orchestrator equivalents through Harness.
- [ ] Record gaps found during the trial.
- [ ] Fix or explicitly defer all P0/P1 gaps.
- [ ] Publish a completed supersession report.

## First Ten Implementation Tickets

1. Create the Harness operating model design doc and role-mode mapping.
2. Add a checked role registry equivalent to Mars's `BOTS.md`.
3. Add conversation-as-system-record design doc and generated guidance.
4. Implement active-plan hygiene checks in doctor and CI.
5. Add intervention-debt ticket type, creation rules, and dedupe.
6. Enforce in-progress ticket priority and blocked-ticket outcomes.
7. Add `docs/QUALITY_SCORE.md` export from scoring data.
8. Add native Orchestrator survey loop for stuck jobs, stale tickets, and score regression.
9. Add deterministic remediation recipe framework.
10. Define and test the Harness dogfood matrix, including observer-mode operation on `../mars`.

## Acceptance Criteria For Supersession

Mars Harness can be considered ready to supersede Mars's meta-harness when all of the following are true:

- `mars-harness init` emits target repo guidance at least as useful as Mars's `template/AGENTS.md`.
- A role registry equivalent to Mars's `docs/automations/BOTS.md` exists, is generated where appropriate, and is mechanically checked.
- The six Mars automation domains are supported directly or through explicit role-mode mapping.
- In-progress tickets are drained before new backlog work by default.
- Human interventions, failed terminal runs, guardrail blocks, score regressions, and dogfood failures automatically create or update tickets.
- `docs/QUALITY_SCORE.md` or equivalent is generated from scoring data.
- Native Harness orchestration covers Mars's Planner, Engineer, Reviewer, Maintainer, End-to-End Tester, and Orchestrator loops.
- Deterministic remediation recipes run before LLM repair where applicable.
- Fake-LLM integration tests cover create ticket -> edit -> test -> commit to `main` -> push `main` -> score -> quality export.
- Harness can register and operate on `../mars` without requiring Cursor automations.
- No default docs, generated bundles, prompts, trust capabilities, or tools reintroduce branch/PR delivery.

## Risks

- Local models may not match Cursor-hosted model quality for broad planning and judgment tasks.
- Strict trunk raises safety requirements because there is no PR buffer.
- Richer generated docs can become context bloat if knowledge routing is not careful.
- Harness may accidentally import Mars-specific SaaS generator constraints into generic target repos.
- Running Harness against Mars can create confusing dual-system interactions while Cursor automations still exist.
- Dogfood ticket generation can overwhelm engineering unless grouping, dedupe, and caps are enforced.
- Quality score can become vanity telemetry unless tied to queue priority and intervention debt.

## Open Decisions

- Should Harness expose six canonical roles with modes, or keep 11 default roles while declaring six canonical domains?
- Should the role registry live under `docs/roles/`, `docs/automations/`, or generated `.harness/` docs?
- Should `docs/QUALITY_SCORE.md` be committed after every score change or only after scheduled/exported quality runs?
- What is the minimum observer-mode success window before allowing Harness to operate on `../mars` in contributor mode?
- Which Mars scripts should become native Go checks, and which should remain optional compatibility scripts?

## Notes

Mars Harness should treat Mars as the successful prototype, not the final architecture. The strongest path is to preserve Mars's feedback loops while replacing Cursor and PR-shaped delivery with local inference, strict trunk, native queue orchestration, guardrails, trust, scoring, and repo-visible state.
