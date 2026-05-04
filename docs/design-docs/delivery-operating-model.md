# AD-074: BDD-Led Goal-Driven Walking-Skeleton Operating Model

**Status:** Accepted
**Date:** 2026-05-02
**Owner:** Mars Harness maintainers

## Context

Mars Harness had the right instincts: tickets, strict trunk, dogfood, telemetry,
quality score, role scoring, and self-improvement. The gap was delivery truth.
Agents could move tickets, add docs, or create enabler code while still leaving
the intended feature half-built. The user-visible complaint was concrete:
dogfood creates tickets, engineers start many of them, and handoff happens
without enough proof that the work is actually complete.

We need an operating model that makes feature completeness observable before
implementation starts, then lets autonomous agents take the smallest useful
path to pass evidence. Lean and walking skeleton are good delivery strategies,
but they become vague without a feature contract and falsification loop. BDD
provides the contract; walking skeleton provides the implementation strategy.

## Decision

Mars Harness uses **BDD-Led Goal-Driven Walking-Skeleton Delivery** as the
canonical operating model for the source harness and generated target
harnesses.

The closed loop is:

1. Goals define outcomes and competing priorities.
2. BDD feature contracts define the full intended capability before
   implementation.
3. Exec plans rank failing BDD scenarios and state the hypothesis for why this
   work advances active goals.
4. Tickets implement the next highest-value failing scenario or scenario group.
5. Walking skeleton delivery makes scenarios pass through the thinnest real
   end-to-end path.
6. Evidence from BDD E2E/integration, dogfood, telemetry, quality score, and
   feedback updates goals and plans.

Core rule:

> BDD defines the full feature. Walking skeleton is the implementation
> strategy. The schedule is the ordered list of failing BDD scenarios.

Business logic is first-class BDD. Every product rule, workflow branch, state
transition, validation, permission check, scoring/trust calculation, queue
routing rule, release classification, or other user-visible outcome belongs in
`docs/features/` as step-by-step behavior before or alongside implementation.
Tickets may reference business logic, and code may comment on it, but neither
is the durable source of truth. Feature contracts must include `Business
Logic`, `Step-By-Step Behavior`, scenario schedule, Given/When/Then scenarios,
and evidence.

No stale documentation is a universal operating-model rule. All durable docs
are live system artifacts, not retrospective notes. Source files and newly
created or materially changed code files must carry a top-of-file
`MarsDocSync` metadata comment block with a `docs:` list of repo-relative docs
that describe, constrain, or explain that code. The same change updates those
docs, or records in the ticket, plan, review, or commit evidence why the listed
docs were checked and did not need content changes. The full docsync
architecture and universal operating model are documented in
[documentation-sync-architecture.md](documentation-sync-architecture.md).

Operating-model changes must be **symbiotic** with the existing system. A new
rule, artifact, role behavior, tool, gate, or automation must fit the closed
loop without creating handoff gaps, duplicate sources of truth, or
inconsistencies with current workflows. If a proposed change alters how work
moves between goals, BDD, plans, tickets, roles, evidence, release, scoring, or
self-improvement, the same change must update the affected artifacts, generated
target defaults, role prompts, knowledge routes, and tests so agents have one
coherent path to follow.

No feature is shipped until its in-scope BDD scenarios pass or the CEO
explicitly descopes, supersedes, or invalidates them. Enabler work may complete
without shipping a feature, but it must be labelled as enabler work and must
not be represented as shipped feature value.

A repeated process promotion to formalized tools is part of the operating model.
When agents or humans use a
multi-step process that is likely to recur, is risky to perform manually, needs
consistent validation, crosses source and deployed harness boundaries, or
requires exact command ordering, the harness should create or improve a
first-class tool for it. The tool must be mirrored when the process applies to
both foundation and deployed harnesses, documented in the tools glossary, added
to generated target defaults when appropriate, covered by tests, and wired into
role allowlists only where the role should use it. Until the tool exists, the
process should be captured as a ticket or decision rather than left only in
chat.

Built-in tool creation must dogfood the meta-tool path. New built-in tools
originate through `tool_create`, one tool at a time, before manual
implementation and any later refactor into shared helper files. Bypassing
`tool_create` is an exception: the agent must first record the reason with
`record_decision`, then add design-doc rationale and tests that preserve the
exception context. Completing the artifact shape without the governed creation
path is not a complete operating-model change.

## Artifact Ownership

Goals live in `docs/goals/`. Active goals may come from user chat, product
requirements, telemetry, quality scores, dogfood, GitHub issues, feedback
forms, or manual docs. Weak/noisy evidence becomes an observation until it is
actionable.

BDD feature contracts live in `docs/features/`. They use Markdown
Given/When/Then in v1 and also carry step-by-step business-logic notes for the
rules behind each scenario. We deliberately do not add a custom Gherkin parser
yet; Go integration/E2E tests and explicit evidence commands execute behavior.

Exec plans live in `docs/exec-plans/` with exactly one active plan. Active and
backlog plans must name goals, BDD feature contracts, hypothesis, success and
falsification evidence, scenario schedule, current failing scenario, walking
skeleton slice, and learning or MVP outcome.

Planning order is strict: active exec plan first, then feature contract, then
tickets, then implementation delivery. A project that has feature docs or
tickets without a current plan has lost the control plane; repair the plan
before widening scope.

Tickets live in `docs/tickets/` and carry `work_type`, `bdd_scenarios`,
`end_to_end_evidence`, `evidence_links`, and `verified_by`. Feature tickets
require scenario evidence before done. Enabler tickets can complete without
feature evidence, but cannot claim shipped feature value.

## Role Responsibilities

Role responsibilities use explicit starter role names here because those roles
remain the executable manifest units. Their canonical domain and mode mapping
lives in [harness-operating-model.md](harness-operating-model.md).

The CEO owns vision, active goals, and final strategy/scope decisions. When the
goals are clear, CEO hands off `exec_plan` to COO through Orchestrator.

The COO updates the single active exec plan first, then creates or updates the
feature contracts and scenario schedule named by that plan. COO hands off
`ticket_breakdown` or `architecture_review` to CTO through Orchestrator.

The CTO validates the plan hypothesis, architecture fit, and whether the
walking skeleton is a real end-to-end path rather than scaffold-only work. CTO
creates implementation tickets only from the current failing scenario or
scenario group.

The Engineer implements one ticket and provides scenario evidence before done.

QA reviews behavior against BDD scenarios and ticket evidence.

Dogfood validates the real build, run, user, or agent path.

The Janitor detects stale goals, stale scenarios, false done, missing evidence,
duplicate work, and misleading in-progress state.

The Release Manager separates shipped scenarios from enablers in patch notes
and release notes.

## Failure Modes And Mitigations

| Failure Mode | Why It Happens | Mitigation |
| --- | --- | --- |
| BDD becomes decorative prose | Scenarios are written but not executed or checked | Every feature needs at least one integration/E2E test or command mapped to scenario IDs. |
| Business logic hides in code or tickets | Agents implement rules that future planners and reviewers cannot see | Require `Business Logic` and `Step-By-Step Behavior` sections in feature contracts and update them whenever behavior changes. |
| Documentation drifts stale from code | Code changes do not reveal which docs own the behavior | Require top-of-file `MarsDocSync` metadata on new or materially changed code files and review the listed docs in the same change. |
| Walking skeleton becomes scaffold theater | Thin slice is mistaken for placeholder architecture | Slice must pass through a real user, CLI, agent, tool, ticket, docs, or evidence path as applicable. |
| Half-features are marked done | Ticket AC passes locally while the feature contract still fails | Feature truth lives in BDD scenario state, not ticket count. |
| Enabler work is misrepresented as shipped value | Release notes infer feature status from commit text | Release notes and quality score classify by `work_type` and scenario evidence. |
| Autonomous goals create thrash | Feedback or telemetry is noisy | Require source, confidence, dedupe key, evidence, and review trigger; weak signals go to observations. |
| CEO ranks by vibes | Goals are ambiguous or incomparable | CEO ranks by value, urgency, confidence, unblock potential, and falsification risk. |
| Competing goals blur strategy | Multiple goals pull in different directions | Goals declare `Competes With`; active plan records tradeoffs and deferred goals. |
| Source and target harness diverge | Init gets new doctrine but old targets keep stale docs | `update check` and `doctor --repo` report drift; `update harness` writes missing defaults only; stale user-owned files become migration tickets. |
| Unit tests give false confidence | Operating-model bugs appear only in full loops | BDD E2E/integration is default acceptance; unit and docs tests support deterministic helpers. |
| Lean becomes endless learning | Hypotheses never close | Exec plans require success and falsification evidence; inconclusive plans are revised, superseded, or split. |
| Operating-model additions create handoff gaps | New rules are added without updating the adjacent artifacts, roles, tools, or evidence path | Treat operating-model changes as system changes: update the whole affected workflow in one task or record the blocker before merging. |

## Mirrored Application

This operating model applies to Mars Harness source and generated target
harnesses unless a rule is explicitly marked source-only. Source changes update
`AGENTS.md`, design docs, product specs, goals, features, exec plans, ticket
docs, quality score, scanner defaults, role prompts, knowledge routes, and
docs-consistency tests in the same task.

Generated targets receive goal docs, feature-contract docs, this design
decision seed, updated exec-plan and ticket templates, updated role prompts,
updated knowledge routes, and target drift checks. Existing target adoption is
non-destructive: missing defaults are written, stale user-owned files are
reported, and conflicts become intervention-debt migration tickets instead of
silent overwrites.

## Consequences

This makes the harness more demanding. A ticket can be done only when the
evidence is visible. That is intentional: the system should optimise for
truthful completion, not autonomous motion. The tradeoff is more upfront
structure in goals and feature contracts, but less downstream ambiguity, fewer
half-shipped features, and clearer evidence for self-improvement.

---

## AD-097: Business Logic Is First-Class BDD

**Status:** Accepted
**Date:** 2026-05-04
**Owner:** Mars Harness maintainers

### Context

The BDD-led operating model already required feature contracts and scenario
evidence, but business rules could still be scattered across ticket bodies,
code comments, role prompts, or release notes. That makes future agents guess
why behavior exists and tempts implementation to outrun product truth.

### Decision

All business logic is documented step by step under `docs/features/`. A feature
contract must carry the business behavior, not merely a scenario title list.
The required sections are `Business Logic`, `Step-By-Step Behavior`, `Scenario
Schedule`, Markdown Given/When/Then scenarios, and `Evidence`.

Business logic includes product rules, workflow branches, state transitions,
validations, permissions, scoring and trust behavior, queue or orchestration
routing, release classification, and user-visible outcomes. Tickets and code
may reference or implement this behavior, but they are not the durable source
of truth.

### Consequences

- Planner roles document behavior before or alongside implementation.
- Engineer and reviewer roles can detect when code changes outrun the feature
  contract.
- Generated target harnesses inherit the same first-class BDD rule.
- Docs-consistency tests enforce the required feature-contract sections.

---

## AD-098: No Stale Documentation

**Status:** Accepted
**Date:** 2026-05-04
**Owner:** Mars Harness maintainers

### Context

The repo is the system of record, but docs can still lag behind implementation
when agents change code without a durable pointer to the docs that describe the
affected behavior. Reviewers then have to rediscover which feature contract,
design doc, product spec, ticket guide, README, or generated target surface
needs attention.

### Decision

All documentation is kept current as the system changes. Every source file and
every newly created or materially changed code file must include a top-of-file
metadata comment block named `MarsDocSync` that lists repo-relative
documentation paths associated with that code. The source-to-documentation map
lives in [code-documentation-map.md](code-documentation-map.md), the architecture
and universal operating model live in
[documentation-sync-architecture.md](documentation-sync-architecture.md), and the
gate is implemented by `internal/docsync` and checked by
`mars-harness docsync audit --repo .` or the mirrored `docsync_audit` tool.

The canonical shape is:

```text
/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/delivery-operating-model.md
- docs/features/F-001-delivery-operating-model.md
*/
```

The listed docs are not decorative. They are the review checklist for the
change. If behavior, public surface, workflow, architecture, generated output,
or operating doctrine changes, the same commit updates the relevant docs. If
the docs are still correct, the ticket, plan, review, or commit evidence should
say they were checked and remain current.

Generated files, language-specific license headers, or framework-mandated file
headers may keep their required first line, but the `MarsDocSync` block must be
near the top of the file before implementation declarations.

### Consequences

- Code review can identify associated docs without guessing.
- Automation has a stable marker for stale-doc checks and fails when source
  files lack metadata, reference missing docs, or drift from the code map.
- Generated target harnesses inherit the same documentation-sync doctrine.
- Agents treat docs updates as part of implementation, not as optional cleanup.

## AD-103: CLI Tool And Skill Synchronization

**Status:** Accepted
**Date:** 2026-05-04
**Owner:** Mars Harness maintainers

### Context

The CLI is a foundation control plane, but agents usually discover it through
mirrored tools, generated target doctrine, role allowlists, and compact skills.
If the CLI changes without synchronizing those mirrors, agents can keep using
stale command references, miss new repo-aware commands, or follow old workflow
steps even though the source command tree changed.

### Decision

Whenever `cmd/mars-harness` changes a command, flag, output contract, repo
behavior, mutability expectation, or recurring workflow, the same change updates
the `mars_harness_cli` reference, the `mars_harness_cli` repo shortcut map,
tool-selection guidance, generated target doctrine, and any skills that name the
affected CLI workflow. The full architecture and evidence model live in
[cli-tool-skill-sync.md](cli-tool-skill-sync.md).

### Consequences

- CLI work is not complete until agent-facing tool and skill surfaces are
  current.
- Tests compare the Cobra command tree with the mirrored CLI reference and repo
  shortcut map.
- Generated targets inherit the same rule so deployed harnesses do not drift
  from foundation CLI behavior.
- CLI release notes explain both operator impact and agent/tool synchronization
  impact.
