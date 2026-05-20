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

Remote trunk freshness is a foundation operating-model gate. For any repository
with `origin/main`, agents start non-trivial work by fetching `origin main` and
ensuring local `main` is at or fast-forwarded to `origin/main` before editing.
Dirty local state, diverged history, rejected pushes, or unavailable remotes are
blockers that must be recorded with a next action unless the user explicitly
requests offline or local-only work. Once a semantic commit, release-note
commit, or required release tag is validated, it is pushed to `origin main` or
the tag remote immediately before unrelated work begins. Work should not sit in
local-only commits when the remote can accept it.

Source-harness live-experience verification is a foundation operating-model
gate. When a Mars Harness source change claims to improve first-run lifecycle,
orchestration, intervention-debt routing, generated target scaffolding, model
or provider behavior, dashboard/control-plane behavior, scoring, update/release,
or safety/guardrail behavior, completion evidence must include a representative
live target run such as `demo-123` or a clearly recorded blocker explaining why
that run could not be performed. Unit, docs, and fake-LLM tests can prove the
deterministic helpers, but they do not replace checking that the installed
harness actually behaves better against a fresh or known-problem target. The
live check starts the continuous improvement loop: run a clean representative
target, review the findings, select one or two bounded actions tied to those
findings, implement and test them, rerun a clean representative target, and
claim improvement only after rerun evidence confirms the fix and the work is
merged or fast-forwarded to trunk and pushed to the remote. The evidence should
record the exact target repo, command, branch/ref or binary used, database/log
paths, observed lifecycle events, product progress, chosen actions, rerun
result, remote ref, and any remaining operator action.

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

Workspace hygiene is a universal operating-model gate. Agents must not expose
generated dependency or build churn to the model as ordinary source diff, and
they must not run package-manager install/fetch commands through raw
`shell_exec`. The deterministic path is:

1. `workspace_hygiene` audits `.gitignore`, tracked generated paths, dirty
   generated directories, large generated diffs, and forbidden deletions.
2. Before model loading, `serve` may make one safe policy repair: append
   missing generated-directory entries such as `node_modules/` to `.gitignore`
   and commit only `.gitignore`. This repair is allowed only when generated
   paths are untracked and `.gitignore` has no user changes; it never deletes
   generated files, stages implementation files, or rewrites package state.
3. Blast-radius checks classify generated dependency/build paths separately
   from implementation files. Generated churn is handled by workspace hygiene;
   implementation changes remain subject to file, line, deletion, and secret
   limits. Ticket lifecycle moves are a narrow exception to deletion counting
   when the same ticket ID moves between lifecycle directories in one diff.
4. `dependency_sync` performs the same safe ignore-policy repair before package
   install/fetch, then runs hygiene preflight using frozen lockfile-respecting
   commands when lockfiles exist.
5. Postflight hygiene blocks when generated artifacts dirty the worktree and
   returns a recipe ID plus exact next action. It does not automatically clean
   or unstage user work.
6. `serve` runs a pre-job hygiene gate before model loading so dirty
   `node_modules/`, build output, generated diffs, or deletion state become
   deterministic blockers instead of repeated LLM, guardrail, and Orchestrator
   loops.

Raw dependency mutation commands such as `npm install`, `npm ci`,
`pnpm install`, `yarn install`, `bun install`, `go mod download`,
`cargo fetch`, `pip install`, `bundle install`, and `composer install` are
blocked by tool policy with guidance to use `dependency_sync`. Target repos own
their hygiene policy: missing ignores, tracked generated dependency trees, or
dirty generated output become target workspace-hygiene intervention debt unless
the project deliberately documents and tests an exception.

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
goals are clear, CEO hands off `exec_plan` to COO; dispatch may route that
directly or use Orchestrator when synthesis is needed.

The COO updates the single active exec plan first, then creates or updates the
feature contracts and scenario schedule named by that plan. COO hands off
`ticket_breakdown` or `architecture_review` to CTO through dispatch.

The CTO validates the plan hypothesis, architecture fit, and whether the
walking skeleton is a real end-to-end path rather than scaffold-only work. CTO
creates implementation tickets only from the current failing scenario or
scenario group.

The Engineer implements one ticket and provides scenario evidence before done.

QA reviews behavior against BDD scenarios and ticket evidence.

Dogfood validates the real build, run, user, or agent path.

The Janitor detects stale goals, stale scenarios, false done, missing evidence,
duplicate work, and misleading in-progress state.

The Release Manager runs after successful product validation as well as on its
schedule. It separates shipped scenarios from enablers in patch notes and
release notes, verifies historical backfill compliance, ensures the GitHub
Release object exists when release credentials are configured, and records
release asset blockers before the lifecycle stops.

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
| Agents build on stale trunk or strand ready work locally | Work starts before fetching `origin/main`, or validated commits are not pushed promptly | Treat remote trunk freshness and immediate push as operating-model gates; fetch before editing, fast-forward or record blockers, push ready commits and tags before unrelated work. |
| Unit tests give false confidence | Operating-model bugs appear only in full loops | BDD E2E/integration is default acceptance; unit and docs tests support deterministic helpers. |
| Source stabilization passes tests but still fails in practice | The changed behavior only appears when the installed harness runs against a real target lifecycle | Treat live-experience verification against `demo-123` or another representative target as a source-harness gate, then rerun after bounded source changes and push confirmed work to remote trunk before claiming improvement. |
| Live demos become endless observation | Findings are gathered without forcing a small action and a rerun | Use the run, review, act, rerun loop: one clean run, one evidence review, one or two bounded fixes, one clean replay, and a recorded next blocker if the replay still stalls. |
| Release discipline is documented but not executed | Product validation stops at Dogfood while semantic target commits remain unreleased | Route approved or completed Dogfood dispositions to Release Manager when configured, and require release backfill compliance before the release-note commit. |
| GitHub Releases page stays stale after tags | The tag workflow cannot start or fails before creating the release object | Treat `gh release view vX.Y.Z` as a required release-object gate. If the workflow fails but the GitHub API is available, create a notes-only release from the generated changelog entry, then keep asset verification as the remaining blocker. |
| Lean becomes endless learning | Hypotheses never close | Exec plans require success and falsification evidence; inconclusive plans are revised, superseded, or split. |
| Operating-model additions create handoff gaps | New rules are added without updating the adjacent artifacts, roles, tools, or evidence path | Treat operating-model changes as system changes: update the whole affected workflow in one task or record the blocker before merging. |

## Mirrored Application

This operating model applies to Mars Harness source and generated target
harnesses unless a rule is explicitly marked source-only. Source changes update
`AGENTS.md`, design docs, product specs, goals, features, exec plans, ticket
docs, quality score, scanner defaults, role prompts, knowledge routes, and
docs-consistency tests in the same task.

The live-experience verification gate is source-only. Generated target repos
still need real build, run, dogfood, and user-path evidence for their product
features, but the named `demo-123` replay rule applies to Mars Harness source
changes because the product being validated is the harness lifecycle itself.
Generated target repos inherit the generic evidence loop: observe a real
product path, review findings, make bounded target-owned changes, rerun the
same path, and claim improvement only after rerun evidence is confirmed,
merged or fast-forwarded to the target's trunk policy, and pushed to the
remote.

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

## AD-108: Remote Trunk Freshness And Immediate Publishing

**Status:** Accepted
**Date:** 2026-05-05
**Owner:** Mars Harness maintainers

### Context

Strict trunk loses its safety properties when agents begin from a stale local
checkout or leave validated work only on disk. Recent local worktree drift made
the installed harness and the source checkout disagree until the changes were
manually replayed onto current `main`. The operating model needs to make remote
trunk freshness and prompt publishing explicit, not implied by "strict trunk".

### Decision

For any foundation or deployed harness repository with an `origin/main` remote,
non-trivial work begins by fetching `origin main` and proving local `main` is at
or fast-forwarded to `origin/main` before edits. Dirty worktrees, unpushed local
commits, diverged histories, missing remotes, network failures, or push
rejections are blockers. Agents record the blocker and next action unless the
user explicitly requests offline or local-only work.

Validated semantic commits are pushed to `origin main` as soon as they are
ready. Source harness release-note commits and release tags are pushed as soon
as their generated files and local checks pass. If a push is rejected, the agent
fetches, rebases or resolves deliberately, reruns relevant checks, and pushes
before starting unrelated work. Force-push and shared-history rewrites remain
outside normal operating policy.

### Consequences

- Agents start from the latest remote truth instead of whichever worktree
  happens to be open.
- Ready work is visible to the remote, CI, release automation, other agents, and
  the user as soon as it is safe to share.
- Local-only commits are treated as incomplete work unless the repo lacks
  `origin/main` or the user deliberately requested offline work.
- Generated target harnesses inherit the same remote-trunk rule, with missing
  or unavailable remotes reported as blockers rather than silently ignored.

## AD-110: Source Improvements Require Live-Experience Verification

**Status:** Accepted
**Date:** 2026-05-19
**Owner:** Mars Harness maintainers

### Context

The `demo-123` lifecycle run showed that Mars Harness can pass focused tests
while the live operator experience still stalls: duplicate bootstrap work,
Orchestrator loops, repo-local runtime artifacts, intervention-debt starvation,
or generic generated target doctrine may only become visible when the installed
tool drives a fresh target from a real brief.

### Decision

For Mars Harness source changes that claim to improve lifecycle behavior,
orchestration, target generation, intervention-debt routing, scoring, safety,
model/provider operation, dashboard/control-plane flows, or update/release
operation, completion evidence must include a representative live-experience
check. `demo-123` is the canonical small replay target for first-run lifecycle
changes: create or reuse a fresh Space Invaders-style target, run the installed
or explicitly built harness against it, and record whether the lifecycle
reaches product-specific planning, feature contracts, ordinary product tickets,
or implementation without falling back into the old failure class.

If the live check is not possible, the agent must record the blocker with exact
date, command intended, missing dependency or credential, and the replay steps
needed. A skipped live check without a blocker is not complete evidence for
this class of source change.

### Consequences

- Source-harness stabilization is judged against the operator experience, not
  only package tests.
- Fake-LLM and unit tests remain required for deterministic behavior, but live
  target runs catch integration drift across CLI, queue, generated docs, tools,
  guardrails, and model/runtime surfaces.
- Future v2 or reset work can use the recorded live checks as evidence of what
  actually improved for users.

## AD-111: Fresh Bootstrap Technical Planning Is Bounded

**Status:** Accepted
**Date:** 2026-05-19
**Owner:** Mars Harness maintainers

### Context

The next `demo-123` live replay improved the original first-run failure class:
CEO, Orchestrator, and COO reached product-specific Space Invaders planning and
feature-contract updates without intervention-debt tickets or repo-local
runtime artifacts. The remaining stall moved one step later. Orchestrator
correctly selected CTO for ticket shaping, but the generated CTO role spent its
opening turns on governance/audit tooling and then expanded the first BDD
scenario into several independent backlog tickets before committing or handing
off to Engineer.

That behavior is better than intervention-debt starvation, but it still delays
visible product progress and recreates backlog entropy during the first
lifecycle.

### Decision

Fresh bootstrap technical planning is bounded. When CTO is dispatched after
CEO/COO planning, during fresh bootstrap, or while the ordinary product backlog
is empty, CTO must create or confirm exactly one independent ordinary feature
ticket for the current failing scenario, then commit, record
`job_disposition_record` with implementation as the next need, and stop.

Broad governance, doctrine, docsync, tool-inventory, dependency, release, and
architecture-audit workflows are deferred until product tickets and first
implementation evidence exist. `ticket_create` mechanically enforces the same
boundary by treating a second independent feature ticket for the same BDD
scenario set as a duplicate unless the new ticket explicitly carries
`depends_on`.

### Consequences

- The first lifecycle should move from product plan to one engineer-ready slice
  instead of turning one scenario into a miniature backlog.
- CTO still owns architecture fit, but only the current scenario's material
  architecture choices are in scope during first bootstrap.
- Deeper decomposition remains possible after implementation evidence exists or
  when dependency metadata makes the relationship explicit.

## AD-112: Successful Dispatch Handoffs Preserve Clean Target State

**Status:** Accepted
**Date:** 2026-05-19
**Owner:** Mars Harness maintainers

### Context

The next clean `demo-123` replay showed the lifecycle now reaches the Engineer
handoff with one ordinary product ticket and no target intervention-debt
tickets. Two new first-run defects appeared at that later stage: the CTO
created a duplicate bare `docs/features/F-001.md` even though
`docs/features/F-001-product-walking-skeleton.md` already existed, and both COO
and CTO recorded successful handoffs while planning and ticket artifacts were
still uncommitted.

Prompt-level commit gates were already present, but live behavior proved that
successful handoff cleanliness must be mechanical.

### Decision

Mutating tool policy now protects canonical planning artifacts and clean
handoffs:

- `file_write` rejects a new `docs/features/F-NNN*.md` contract when any
  different contract with the same `F-NNN` feature ID already exists.
- successful `job_disposition_record` calls from non-Orchestrator roles are
  rejected while repo-visible changes remain uncommitted.
- Orchestrator remains able to record dispositions when it sees dirty state
  left by a prior role, so the recovery path can route or stop deliberately.

### Consequences

- COO, CTO, Engineer, and other producing roles cannot hide uncommitted target
  work behind a successful disposition.
- Generated targets keep one canonical feature-contract path per feature ID
  unless a human intentionally cleans up old files.
- The next live replay should either show committed planning/ticket artifacts
  before Engineer starts or expose a narrower commit-tool/remoting problem.

## AD-113: Dispatch Routing Normalizes Canonical Role Aliases

**Status:** Accepted
**Date:** 2026-05-19
**Owner:** Mars Harness maintainers

### Context

The next `demo-123` replay proved clean handoff policy worked for CEO and COO,
but exposed a narrower routing defect. Orchestrator correctly identified that
technical ticket breakdown was next and recorded `suggested_role: cto`. The
generated target manifest uses `cto-weekly` as the executable role key, so the
dispatcher rejected the otherwise reasonable shorthand and queued Orchestrator
again instead of reaching CTO.

### Decision

Dispatch routing now validates suggested roles against the target manifest
after applying a small canonical alias map. `cto` and `architecture` resolve to
`cto-weekly`, `release` resolves to `release-manager`, and `dependency`
resolves to `dependency-manager` when those executable role keys exist.
Unknown suggested roles still fall back to the existing ambiguous route path
instead of inventing a role.

### Consequences

- Orchestrator can use ordinary domain language without causing a dispatch loop.
- Generated target manifests remain executable by their explicit role keys.
- Future alias additions should stay small and tied to stable generated role
  names rather than becoming a free-form role inference system.

## AD-114: Failed Orchestrator Dispatch Falls Forward Or Stops

**Status:** Accepted
**Date:** 2026-05-19
**Owner:** Mars Harness maintainers

### Context

The next live `demo-123` replay confirmed that duplicate feature-contract
creation and shorthand role aliases were fixed, but exposed another narrower
failure. COO committed a product-specific plan and feature-contract update and
recorded a `ticket_breakdown` handoff. The following Orchestrator job reached
max turns without recording its own disposition. The server then recorded the
Orchestrator failure and fed that failed Orchestrator disposition back through
dispatch, which selected Orchestrator again instead of preserving product
momentum toward CTO ticket shaping.

That failure is the same class as the original intervention-debt starvation:
runtime trouble consumed the delivery loop before the product had an ordinary
ticket or implementation evidence.

### Decision

Dispatch-mode Orchestrator failure recovery is deterministic and non-recursive.
When Orchestrator fails before recording a disposition, the server records the
failure as telemetry, then inspects the dispatch trigger that caused the
Orchestrator run:

- if the trigger carries a non-Orchestrator source disposition with a
  deterministic routing signal, such as `next_need`, `suggested_role`,
  `handoff.target_role`, or `feedback.for_role`, the runtime falls forward to
  the target role selected from that original source handoff
- the fallback trigger preserves the original source role, source job, and
  source disposition so the next role receives the product handoff rather than
  a failed-Orchestrator handoff
- if the trigger is missing, source-owned by Orchestrator, lacks a deterministic
  routing signal, or would select Orchestrator again, dispatch records a stopped
  decision and creates no recursive Orchestrator job

### Consequences

- Orchestrator max-turn, context, or provider failures cannot autonomously
  spawn another Orchestrator loop.
- Product-specific handoffs that already contain enough structure can continue
  to the next delivery owner even when Orchestrator itself times out.
- Ambiguous failures still stop with telemetry and an operator-visible routing
  blocker instead of pretending product progress happened.

## AD-116: Engineer Dispatch Requires An Open Product Ticket

**Status:** Accepted
**Date:** 2026-05-19
**Owner:** Mars Harness maintainers

### Context

The next `demo-123` replay confirmed the ticket lifecycle deletion exception
in tests, but live routing exposed another product-progress invariant. CEO and
Orchestrator skipped the COO planning update, CTO completed without creating an
ordinary product ticket, and Orchestrator still routed Engineer. Engineer then
began writing product files with no ticket context, reintroducing free-floating
implementation work after the system had spent several iterations removing
intervention-debt and governance noise from the front of the queue.

### Decision

Dispatch now treats Engineer implementation as ticket-backed work. If a
dispatch decision selects `engineer` while the target has no ordinary product
ticket in `docs/tickets/backlog/` or `docs/tickets/in-progress/`, the runtime
rewrites the next role to `cto-weekly` with `next_need: ticket_breakdown`.
Intervention-debt tickets do not satisfy this prerequisite.

If an ordinary product ticket exists in backlog or in-progress, Engineer can be
enqueued normally and the existing ticket gate remains responsible for claim,
movement to done or review, evidence, and clean handoff.

When the source disposition came from Engineer and the source status is
`completed`, the absence of an open product ticket means the ticket has likely
been moved to `docs/tickets/done/`. In that case stale `next_need:
implementation` is corrected to QA review instead of sending the loop back to
Engineer or CTO.

### Consequences

- Implementation cannot start from vibes or stale handoff prose alone.
- CTO ticket-shaping remains the mechanical boundary between product planning
  and Engineer work.
- Completed Engineer work advances to QA even when the model leaves stale
  implementation wording in the handoff.
- This can expose CTO no-op behavior earlier, but that is preferable to hiding
  the missing ticket behind untracked product code.

## AD-117: Ticket-Gate Failure Recovery Is Bounded Engineer Repair

**Status:** Accepted
**Date:** 2026-05-19
**Owner:** Mars Harness maintainers

### Context

The next clean `demo-123` replay reached actual implementation and completed
the first product ticket, but the Engineer moved the ticket to done without
filling `evidence_links`. The ticket gate correctly failed the job. The failure
then went through dispatch-mode Orchestrator recovery, and Orchestrator hit max
turns while trying to reason about a deterministic ticket-evidence repair.

### Decision

Engineer ticket-gate failures are repaired by one bounded Engineer retry. The
server records the failure as telemetry, keeps intervention-debt tickets
quarantined, and enqueues an Engineer `ticket_gate_repair` job carrying the
gate error, source job ID, and a ticket-lifecycle/evidence-only repair scope.
Generated Engineer guidance treats this trigger as a fast path: update ticket
evidence, lifecycle placement, or handoff metadata, commit the correction, and
avoid product-code implementation unless the gate reason explicitly names
invalid code. If that repair job fails a ticket gate again, the server stops
automatic recovery instead of enqueueing another repair or routing through
Orchestrator.

### Consequences

- Missing ticket evidence remains a product-owned fix, not foundation
  intervention debt.
- The repair path has the role and context needed to update the done ticket,
  add evidence, and commit.
- The fast path keeps evidence repair narrow so it does not become a second
  open-ended implementation run.
- Repeated gate failures stop as one operator-visible blocker instead of
  becoming an Orchestrator or Engineer loop.

## AD-118: Dispatch Protocol Failures Stop As Telemetry

**Status:** Accepted
**Date:** 2026-05-19
**Owner:** Mars Harness maintainers

### Context

In a clean `demo-123` replay, QA reached the review stage after Engineer
completed the Space Invaders ticket, but the generated QA prompt encouraged a
review prose response while the default QA manifest was read-only. QA replied
without calling `job_disposition_record`, so the executor failed the job with a
dispatch protocol error. Routing that deterministic protocol miss through
Orchestrator only created more lifecycle work without changing the missing
role instruction.

### Decision

Dispatch protocol failures are foundation-owned telemetry and stop by default.
The server records the failed disposition and telemetry event, but it does not
enqueue Orchestrator for deterministic "role did not call
`job_disposition_record`" failures. Generated QA guidance is read-only by
default and requires exactly one disposition before finishing.

### Consequences

- Protocol misses point directly at prompt/tooling fixes instead of becoming
  target backlog or autonomous routing work.
- QA review remains durable through disposition data even when the role cannot
  write review files.
- Operators get one clear blocker if a role prompt still violates dispatch
  protocol.

## AD-119: Completed Tickets Cancel Stale Ticket-Owner Survey Jobs

**Status:** Accepted
**Date:** 2026-05-19
**Owner:** Mars Harness maintainers

### Context

During the same replay, the orchestrator survey saw T-001 in
`docs/tickets/in-progress/` while a bounded Engineer repair was running and
queued a separate Engineer ticket-owner job. The repair then moved T-001 to
`docs/tickets/done/`, leaving the pending survey job stale.

### Decision

When an Engineer completes in dispatch mode, the server cancels pending
Engineer survey jobs whose referenced tickets are no longer eligible
in-progress work. Live survey jobs stay pending when at least one referenced
ticket still exists in `docs/tickets/in-progress/` and is not blocked.

### Consequences

- Repairs and ordinary delivery can finish a ticket without a queued stale
  Engineer run claiming already-completed work.
- The watchdog still routes genuinely live in-progress tickets.
- This keeps product progress moving toward QA instead of re-entering
  implementation after done-state evidence exists.

## AD-120: Completed Review Handoffs Move Forward

**Status:** Accepted
**Date:** 2026-05-19
**Owner:** Mars Harness maintainers

### Context

A later `demo-123` replay reached the intended product lane through CEO, COO,
CTO, Engineer, bounded ticket-gate repair, QA, and Security with zero
intervention-debt tickets. Security approved the Space Invaders ticket, but the
next Orchestrator handoff suggested QA again for the same ticket. QA then
failed the dispatch protocol on a duplicated read-only review, proving that
the lifecycle still allowed a completed review stage to regress.

### Decision

Dispatch routing now treats approved or completed QA, Security, Dependency
Manager, and Release Manager handoffs as a forward-only review chain. When an
Orchestrator disposition suggests an earlier reviewer for the same completed
review chain, the deterministic router rewrites to the next forward review
owner present in the manifest. If no forward review owner remains, dispatch
stops with a review-chain-complete reason.

### Consequences

- Security approval no longer sends completed work back to QA by default.
- A target can still move backward through explicit changes-requested or
  feedback/rework statuses before a review is approved.
- The runtime protects the delivery lane from review churn even when the
  Orchestrator prompt makes a locally plausible but stale suggestion.

## AD-121: Runtime Learnings Do Not Block Product Handoffs

**Status:** Accepted
**Date:** 2026-05-19
**Owner:** Mars Harness maintainers

### Context

The next clean `demo-123` replay got farther than previous runs: CEO,
Orchestrator, CTO, and Engineer produced a real Space Invaders ticket and
browser implementation files, and the bounded ticket-gate repair was enqueued
without target intervention-debt tickets. The repair then ran into a narrower
foundation/runtime defect. Convention learning updated
`.harness/learnings.yaml`, and the successful-disposition clean-tree gate
blocked `job_disposition_record` even though the only remaining dirty path was
runtime-managed learning metadata.

This turned a product evidence repair into another guardrail block and risked
re-entering the same lifecycle stall class the stabilization is trying to
remove.

### Decision

Successful non-Orchestrator `job_disposition_record` calls still require a clean
target worktree for product, ticket, source, and documentation changes. The
gate now ignores `.harness/learnings.yaml` when that runtime-managed learning
file is the only dirty path, and omits that file from the blocking path list
when additional product paths are dirty.

The exception is deliberately scoped to disposition cleanliness. It does not
disable repo containment, secret scanning, blast-radius checks, or the rule
that produced product work must be committed before a successful handoff.
When the server records runtime job lessons and `.harness/learnings.yaml` is the
only resulting dirty path, the executor stages and commits that file with a
small `chore(learnings)` commit so the target is handed off cleanly.

### Consequences

- Runtime convention detection can no longer strand an otherwise clean product
  handoff.
- Runtime-only learning updates do not leave the target dirty after a completed
  server job.
- Product and documentation dirty paths still block successful dispositions and
  name the path that needs a commit.
- The live-demo validation loop can continue past metadata churn and expose the
  next product-lifecycle bottleneck instead of repeatedly repairing the harness
  around its own learning file.

## AD-122: COO Cannot Implement Before CTO Ticketing

**Status:** Accepted
**Date:** 2026-05-19
**Owner:** Mars Harness maintainers

### Context

The next clean `demo-123` replay proved the runtime-learnings exception worked:
CEO completed with `.harness/learnings.yaml` as the only remaining dirty path,
and the lifecycle advanced through Orchestrator to COO without intervention-debt
tickets. COO then expanded the first feature contract and committed a root
`index.html` Space Invaders implementation before CTO created any ordinary
product ticket.

That is product progress, but it bypasses the role ownership and ticket-backed
delivery boundary. The first lifecycle must be product-first and ticket-backed:
COO plans, CTO creates the implementation ticket, and Engineer implements.

### Decision

COO is mechanically constrained to planning artifacts. `file_write` calls from
COO are allowed only for `docs/exec-plans`, `docs/features`, and
`docs/goals/observations.md`. Attempts to create or edit implementation files,
root product files, source, package manifests, tests, or build scripts are
blocked before mutation. Mutating `shell_exec` calls from COO are also blocked,
so existing target manifests cannot bypass the planning boundary with shell
commands. Generated COO manifests no longer include `shell_exec` by default,
and generated COO guidance names the no-implementation boundary explicitly.

### Consequences

- COO can still produce the active plan, BDD feature contract, scenario
  schedule, and CTO handoff.
- Product code cannot appear before an ordinary product ticket exists unless a
  human bypasses the harness outside agent execution.
- The next live replay should either reach CTO ticket creation or expose a
  narrower planning/ticket-shaping blocker, rather than shipping root product
  code from COO.

## AD-123: Completed Engineer Work Must Reach QA Before More Planning

**Status:** Accepted
**Date:** 2026-05-19
**Owner:** Mars Harness maintainers

### Context

The next clean `demo-123` replay proved the COO planning boundary worked: CEO,
COO, CTO, and Engineer stayed on the Space Invaders product lane, CTO created
an ordinary product ticket, and Engineer implemented browser game files under
`src/` with no intervention-debt tickets. After Engineer moved T-001 to
`docs/tickets/done/`, the Orchestrator still selected `cto-weekly` twice before
QA reviewed the delivered slice.

The earlier Engineer ticket prerequisite only rewrote direct Engineer dispatch
when no open product ticket existed. It did not treat a completed Engineer
source disposition as a review boundary when Orchestrator selected CTO planning
instead.

### Decision

When a dispatch trigger comes from a completed Engineer disposition and the
target has no open ordinary product ticket in backlog or in progress, the next
dispatch is QA review. This rule applies before further implementation,
ticket-shaping, or planning handoffs, including Orchestrator-selected
`cto-weekly` routes and deterministic Orchestrator-failure fallback.

### Consequences

- Delivered slices move to review before the harness creates the next ticket.
- CTO can still shape work when no ticket exists before implementation begins.
- The product lane avoids a planning fan-out immediately after an Engineer has
  produced unreviewed product code.

## AD-124: Review Rework Reuses The Existing Product Ticket

**Status:** Accepted
**Date:** 2026-05-19
**Owner:** Mars Harness maintainers

### Context

The next clean `demo-123` replay validated the QA boundary: after Engineer
completed T-001, Orchestrator routed to QA instead of CTO. QA then requested
implementation rework on T-001 because the submitted evidence did not include
the expected runnable test output. Orchestrator correctly selected Engineer for
`implementation_rework`, but the no-open-ticket prerequisite saw T-001 in
`docs/tickets/done/` and rewrote the handoff back to CTO ticket shaping.

That turned review feedback on a concrete delivered ticket into a duplicate
ticket-creation path. It also risked starving the actual product fix behind more
planning work.

### Decision

The Engineer ticket prerequisite now distinguishes fresh implementation from
review rework. Fresh implementation still requires an ordinary product ticket in
backlog or in progress. However, when the source disposition is
`changes_requested`, names an existing ordinary product ticket, and the next
need asks for implementation rework or an Engineer fix, dispatch allows
Engineer to run against that same ticket even if the ticket currently lives in
`docs/tickets/done/` or `docs/tickets/in-review/`.

Intervention-debt tickets do not satisfy this exception. Reviews that ask for
ticket breakdown, architecture, or planning still route to CTO or COO as
appropriate.

### Consequences

- QA feedback repairs the delivered slice instead of creating a second ticket.
- The no-ticket implementation guard remains intact for fresh work.
- The first lifecycle can continue from implementation to QA to rework without
  drifting back into ticket fan-out.

## AD-125: Dispatch Completion Requires Terminal Tool Recovery

**Status:** Accepted
**Date:** 2026-05-19
**Owner:** Mars Harness maintainers

### Context

The next clean `demo-123` replay after ticket lifecycle stabilization reached a
stronger product-first path: CEO, COO, CTO, Engineer, a bounded ticket-gate
repair, Orchestrator, and QA all stayed on the Space Invaders workstream with no
target intervention-debt tickets. QA then produced a prose approval-style answer
without calling `job_disposition_record`. The generated QA prompt already
required exactly one disposition, so the failure showed that prompt guidance
alone is not enough for local-model tool discipline.

### Decision

Dispatch-mode server jobs now pass `job_disposition_record` to the agent loop as
a required terminal tool. If a dispatch role attempts to finish with prose and no
tool call, the loop appends one corrective user turn requiring the next response
to be a tool call. If the role has completed inspection, that tool must be the
terminal disposition tool. If the role still needs evidence, it may call an
allowed non-terminal tool and record the terminal disposition when review is
complete. A successful terminal disposition still stops the loop immediately. If
the role still finishes without a disposition after that correction, the existing
protocol-failure path records foundation telemetry and stops without Orchestrator
recovery or target backlog debt.

### Consequences

- QA and other read-mostly roles get an in-band chance to satisfy the protocol
  before a deterministic failure is recorded, without spending the whole job
  budget on repeated reminders.
- Terminal-tool recovery does not force a premature disposition when the role
  has discovered evidence but still needs to inspect referenced files.
- The Orchestrator still receives structured dispositions rather than prose.
- Repeated refusal or inability to call the tool remains an operator-visible
  foundation blocker, not a target product ticket.

## AD-126: QA Must Inspect The Repo Before Liveness Blocking

**Status:** Accepted
**Date:** 2026-05-19
**Owner:** Mars Harness maintainers

### Context

The next clean `demo-123` replay validated terminal-tool recovery: QA called
`job_disposition_record` instead of ending in prose. The run then exposed a
later review failure. After Engineer and a bounded evidence repair completed
T-001, QA blocked with `next_need: liveness` because implementation source was
not included in the dispatch trigger context, even though QA had read-only repo
tools and the target repo contained `index.html`, `script.js`, `style.css`, the
ticket, and recent commits. Orchestrator then routed the liveness block back to
CTO, turning a QA inspection miss into planning work.

### Decision

QA's generated role prompt now makes repository inspection mandatory before a
liveness or missing-context block. QA must read the completed ticket, recent
commits, and implementation files named by the ticket or evidence links with
`file_read` or `grep`; missing source prose in the trigger is not enough to
block. If files exist but evidence is weak, QA uses `changes_requested` for
Engineer with exact missing commands, file paths, reports, or browser evidence.
`blocked` is reserved for artifacts that truly cannot be read after trying the
available read-only tools.

Dispatch also protects this boundary. If a QA blocked disposition is about
missing trigger-provided source or liveness context and Orchestrator selects
CTO, COO, CEO, or Janitor, the runtime rewrites the handoff back to QA for a
repo-inspection retry instead of turning a review-tooling miss into planning.

### Consequences

- QA cannot avoid review merely because the trigger payload is compact.
- Missing or low-quality evidence stays an Engineer feedback path; unreadable
files remain a true blocker.
- Planning roles are not reintroduced after implementation solely to compensate
  for QA not using its read-only tools.

The later clean `demo-123` replay showed one more form of the same failure. QA
first looked for the ticket at `docs/tickets/T-001-...md`, then found the
feature contract with grep, but the terminal-tool recovery prompt pushed it to
record a blocked disposition before reading the feature contract and source
files. Ticket index entries now include the lifecycle path, generated QA prompts
tell reviewers to search `backlog/`, `in-progress/`, `in-review/`, and `done/`,
and terminal recovery allows continued read-only inspection before the final
disposition.

## AD-127: Inline Tool-Call Tags Are Runtime Tool Calls

**Status:** Accepted
**Date:** 2026-05-19
**Owner:** Mars Harness maintainers

### Context

The next clean `demo-123` replay validated that QA now inspects target files and
can produce useful product feedback. QA requested rework because T-001's first
Space Invaders implementation lacked real life-decrement and game-over logic.
Engineer then committed a product fix for complete life management.

The follow-up QA job failed a different dispatch-protocol path. The trace showed
that the fast local model attempted tool calls as inline text:
`<tool_call>file_read{path:<|"|>...<|"|>}</tool_call>` and later
`<tool_call>job_disposition_record{...}</tool_call>`. The runtime already
supported structured `tool_calls`, JSON/fenced JSON, and `<function=name>`
blocks, but not this inline `<tool_call>` syntax. As a result, a legitimate tool
attempt was classified as prose, the one terminal-tool reminder fired, and the
job ended without a recorded disposition.

### Decision

The agent parser now treats inline `<tool_call>name{...}</tool_call>` blocks as
tool calls. It converts unquoted-key arguments, `<|"|>` sentinel strings,
string arrays, and nested objects such as review feedback into JSON arguments
before execution. This parser path runs after the older `<function=name>`
normalizer and before generic JSON extraction.

### Consequences

- Fast local models that choose inline tool-call tags can still inspect files
  and record terminal dispositions.
- Required terminal-tool recovery now catches true prose misses, while attempted
  inline tool calls execute through the normal allowlist and tool-policy path.
- Tool-call compatibility stays in the runtime rather than duplicating more
  syntax-specific instructions into every generated role prompt.

## AD-128: Product Approval Routes To Dogfood Before Governance

**Status:** Accepted
**Date:** 2026-05-19
**Owner:** Mars Harness maintainers

### Context

The next clean `demo-123` replay validated inline tool-call parsing in the live
lifecycle. Mars Harness created T-001, Engineer committed a Space Invaders
walking skeleton, moved the ticket to done, QA approved the work, and Security
completed with no critical or high severity findings. No target
intervention-debt tickets were created.

After that product approval, dispatch drifted into governance: Security's
completed disposition was forwarded to Dependency Manager by the automatic
review chain, Dependency Manager hit `max_turns`, and later Orchestrator/CTO
attempted repeated ticket shaping without a ticket-state change until the loop
guard stopped the churn. For a fresh target product slice with no dependency
change, automatic Dependency Manager and Release Manager routing is too heavy;
the next product-first check is whether a user can run and inspect the thing
that was just built.

### Decision

Automatic review progression now follows QA → Security → Dogfood, then stops
when no forward product validation owner remains. Dependency Manager and
Release Manager stay available through explicit `next_need`, `suggested_role`,
handoff, or feedback requests, but they are no longer the default continuation
after Security for a completed product ticket.

### Consequences

- Fresh target bootstraps prioritize runnable product validation over
  dependency/release governance after the first approved slice.
- Dependency and release work can still run when there is a concrete dependency
  or release need.
- Post-approval lifecycle churn is contained earlier, and the expected next
  evidence for `demo-123` is a Dogfood run or a clean stop, not automatic
  Dependency Manager max-turn failure.

## AD-129: Runtime Learnings Are Auto-Committed When Sole Dirty Path

**Status:** Accepted
**Date:** 2026-05-19
**Owner:** Mars Harness maintainers

### Context

The same live `demo-123` replay that validated inline tool-call parsing and
post-approval Dogfood routing left one residual operational defect:
`.harness/learnings.yaml` stayed modified after runtime convention learning.
The earlier disposition exception let jobs finish, but the target repo still
looked dirty to the next operator or agent even when no product, ticket, source,
or documentation work was pending.

That is still part of the intervention-debt failure class. Runtime-owned
metadata should become durable evidence, but it should not become a hidden
handoff tax for ordinary product delivery.

### Decision

After each server job records runtime lessons, the executor checks the target
worktree. If `.harness/learnings.yaml` is the only dirty path, the executor
stages and commits it with a `chore(learnings): update runtime learnings for
<role>` message. If any other path is dirty, the executor leaves the worktree
untouched so product, ticket, documentation, source, or mixed changes remain
visible to the responsible role and normal guardrails.

### Consequences

- Runtime convention learning becomes repo-visible history instead of an
  uncommitted target-state leak.
- Product work is never silently swept into an automatic learning commit.
- Clean demo validation can distinguish real product or governance drift from
  harmless runtime metadata churn.

## AD-130: Deterministic Product Handoffs Bypass Orchestrator

**Status:** Accepted
**Date:** 2026-05-19
**Owner:** Mars Harness maintainers

### Context

The clean `demo-123` replay after intervention-debt quarantine finally reached
the intended product loop: CEO and COO planned a Space Invaders slice, CTO
created T-001, Engineer implemented player movement and game-loop feedback, QA
approved it, and Dogfood validated it. No target intervention-debt ticket flood
occurred, and the target repo ended clean.

The run still spent too much time on deterministic lifecycle handoffs. Each
clear terminal disposition detoured through Orchestrator before the next role
ran, even when the source role already recorded `suggested_role`, `next_need`,
or an unambiguous review-chain continuation. One ticket-gate false negative also
created an unnecessary Engineer repair because the gate understood inline YAML
lists but not normal multiline YAML lists.

### Decision

Dispatch now routes completed, approved, in-review, and no-work
non-Orchestrator dispositions directly when the target can be selected from
`suggested_role`, `handoff.target_role`, `feedback.for_role`, `next_need`, or
the default product validation spine. A no-work disposition with no named need
still stops. Orchestrator remains the fallback for ambiguous, blocked, failed,
conflicting, or governance-heavy handoffs.

Direct dispatch does not enqueue a same-role handoff loop when `next_need` maps
back to the source role. If a non-Orchestrator role records completed and the
role has a default forward owner, dispatch treats the completed status as
evidence that local work is done and routes to that forward owner. If the role
records no-work and the only route is a same-role `next_need`, dispatch stops
with a same-role reason instead of pretending no-work is progress. Roles must
continue their own work, record a blocker, or provide an explicit structured
handoff when neither completed-forward routing nor a clear target applies.

The direct path still uses the same role validation, ticket prerequisites,
review-chain progression, repeated-route loop guard, durable decision record,
and typed dispatch trigger as Orchestrator-selected routes. The completed-ticket
evidence gate now accepts multiline YAML lists for fields such as
`bdd_scenarios` and `evidence_links`, matching the shape produced by live
target agents.

### Consequences

- Fresh product delivery no longer pays an Orchestrator LLM job for obvious
  CEO → COO, COO → CTO, CTO → Engineer, Engineer → QA, and QA → Dogfood
  handoffs.
- Orchestrator remains available when synthesis, blocker triage, or governance
  review is actually needed.
- Normal YAML ticket evidence no longer causes a false repair loop after a
  successful feature implementation.

## AD-131: CEO Strategy Writes Stop At Goal Boundaries

**Status:** Accepted
**Date:** 2026-05-19
**Owner:** Mars Harness maintainers

### Context

The next clean `demo-123` replay showed a narrower version of the same
role-boundary failure that previously let COO implement before CTO ticketing.
CEO correctly read the project brief and wrote product-facing planning
artifacts, but it crossed its ownership boundary by writing the active exec plan
and BDD feature contract directly. It then recorded `status: no_work` with
`next_need: strategy_advice`, which stopped the product chain before COO, CTO,
or Engineer could create and execute the first ticket.

The model prompt already said CEO does not own exec plans, feature contracts, or
tickets. The live run proved that prompt-only ownership is not strong enough for
bootstrap.

### Decision

`file_write` policy now constrains CEO to strategy artifacts:
`docs/goals/active.md`, `docs/goals/observations.md`,
`docs/product-specs/vision.md`, and `docs/reports/strategy/*.md`. Attempts to
write `docs/exec-plans/*`, `docs/features/*`, implementation files, or other
planning/delivery artifacts are blocked with guidance to hand off to COO or CTO.

The dispatch router also honors `no_work` dispositions that name a deterministic
`next_need`, so a role can truthfully say it has no local work while still
routing the next owner. Generated CEO guidance now says fresh bootstrap should
prefer `next_need: exec_plan` over `strategy_advice` when README and active
goals already define a visible first product slice.

### Consequences

- CEO can still make strategy and goal decisions, but cannot silently take COO's
  plan/feature-contract work.
- Fresh bootstrap is pushed back toward CEO → COO → CTO → Engineer instead of
  CEO-owned planning or advisory detours.
- `no_work` no longer means "stop" when the disposition also names an explicit
  next need.

## AD-132: QA Approval Does Not Reapply The Engineer Pre-Review Guard

**Status:** Accepted
**Date:** 2026-05-19
**Owner:** Mars Harness maintainers

### Context

The next clean `demo-123` replay reached the intended first-slice lifecycle:
CEO, COO, CTO, Engineer, and QA all ran; Engineer implemented the Space
Invaders player-movement ticket; QA approved it; and no target intervention
debt tickets were created. The remaining defect appeared after QA approval. The
dispatch trigger still carried the original completed Engineer disposition for
T-001, so the completed-Engineer/no-open-ticket prerequisite re-fired and
overrode QA's current `next_need: dogfood_validation` handoff with another QA
job.

The pre-review guard is valuable: it stops the system from planning or
implementing a second slice before the completed product ticket reaches QA. But
once QA has actually reviewed and approved that work, reapplying the same guard
turns a safety invariant into a validation loop.

### Decision

The completed-Engineer/no-open-ticket prerequisite now applies only to
Orchestrator fallback decisions before QA review. It can still rewrite
ambiguous or stale implementation/planning handoffs to QA when the current
decision is coming from Orchestrator and the latest source disposition is a
completed Engineer ticket.

After QA records `approved` or `completed`, dispatch honors QA's current
disposition and routes forward through the product-validation chain such as
Dogfood validation, or stops when no forward owner exists. The stale Engineer
source disposition may remain in the trigger for evidence, but it no longer
overrides the current reviewer outcome.

### Consequences

- The QA-before-more-planning invariant remains intact before review.
- QA approval now advances validation instead of enqueueing repeated QA work for
  the same completed ticket.
- Live target validation remains the operating-model feedback loop: each replay
  can expose the next narrow lifecycle boundary without turning foundation
  safety checks into product backlog churn.

## AD-133: QA Review Must Start With Tools, Not Prose

**Status:** Accepted
**Date:** 2026-05-19
**Owner:** Mars Harness maintainers

### Context

The next clean `demo-123` replay validated the product-first lifecycle through
CEO, COO, CTO, Engineer, and bounded ticket-gate repair. Engineer produced a
real Space Invaders `src/index.html`, moved T-001 to done, repaired BDD evidence
metadata after the gate caught missing fields, and routed to QA with zero target
intervention-debt tickets.

QA then exposed a different liveness problem. The role completed in two prose
messages, used zero tools, and never called `job_disposition_record`. The
runtime correctly classified this as a dispatch protocol failure and, per the
deterministic containment rule, did not route the miss through Orchestrator. The
bad part was upstream: generated QA guidance named shell-style review commands
even though default QA is read-only and lacked read-only git inspection tools;
QA also ran on the `fast` tier, which proved unreliable for tool-following in
this review path.

### Decision

Generated QA now runs on the `reasoning` tier and gets only read-only git
inspection tools (`git_status`, `git_diff`) in addition to the existing
read-only review tools and `job_disposition_record`. The generated QA prompt
starts with an explicit tool-first rule: the first assistant response should be
an allowed read-only tool call, not a narrative preamble.

The prompt no longer asks default QA to narrate unavailable shell commands such
as `git log`, `git diff`, `npm`, or browser checks as if it could run them. If
runnable or browser evidence is missing, QA must request Engineer rework or
Dogfood validation through `job_disposition_record` instead of prose-approving
the work.

### Consequences

- QA remains read-only, but now has enough repo-inspection surface to review the
  implementation evidence it is asked to approve.
- Protocol failures still stop as telemetry rather than becoming Orchestrator
  loops or target backlog churn.
- Review uses a more reliable model tier, trading a little speed for lifecycle
  progress and durable dispositions.

## AD-134: Done-Ticket Evidence Is a Tool Preflight

**Status:** Accepted
**Date:** 2026-05-19
**Owner:** Mars Harness maintainers

### Context

A later clean `demo-123` replay showed the lifecycle now reaches real product
implementation without intervention-debt amplification: CEO, COO, CTO, and
Engineer produced a browser Space Invaders player-movement slice, ordinary
product ticket, tests, commits, and zero target intervention-debt tickets.

The remaining failure was timing. Engineer moved T-001 to `docs/tickets/done/`
with `evidence_links: []` and `verified_by: TBD`; the post-run ticket gate
correctly rejected the completed job, then the bounded ticket-gate path queued a
repair Engineer. That repair is bounded, but it still spends another model run
on a metadata issue the original Engineer can fix before leaving the tool call.

### Decision

Ticket evidence is now enforced before the lifecycle move, not only after the
job completes. Tool policy blocks `git mv` or `mv` into
`docs/tickets/done/` when a feature ticket still lacks required BDD evidence
fields, and it blocks `file_write` updates that would save a done feature ticket
with empty evidence. The error names the missing fields and tells the role to
update `evidence_links` and `verified_by` before moving or saving the ticket as
done.

Generated target guidance now repeats the same rule in the ticket lifecycle and
Engineer commit gate: code work may finish only after the feature ticket has
concrete evidence metadata, then the ticket can move to done and dispatch can
route review normally.

### Consequences

- The first Engineer run can repair missing evidence in-place instead of
  failing late and requiring a bounded repair job.
- Ticket-gate repair remains available for stale or externally introduced
  false-done tickets, but it is no longer the default path for ordinary
  Engineer completion.
- Live demo replay remains the foundation operating model for discovering the
  next lifecycle bottleneck: run, observe, patch the smallest mechanical guard,
  and replay cleanly.

## AD-135: Runtime Failures Stop Dispatch And Ticket Moves Stay Atomic

**Status:** Accepted
**Date:** 2026-05-19
**Owner:** Mars Harness maintainers

### Context

A clean `demo-123` replay after AD-134 reached product planning, ticketing, and
real Space Invaders implementation without target intervention-debt tickets.
The next failure was no longer lack of product progress; it was cleanup churn.
Engineer committed implementation and ticket evidence, but copied T-001 into
`docs/tickets/done/` while leaving the backlog copy in place, then hit
`max_turns`. Dispatch failure handling treated that runtime miss as work for
Orchestrator, and Orchestrator selected CTO ticket shaping even though the
product slice already existed.

The same run also showed two content-quality drifts: COO appended duplicate
`F-001-S001` scenario headings to the starter feature contract, and CTO's
ticket body repeated the canonical ticket title heading.

### Decision

Non-Orchestrator runtime failures such as `max_turns`, context overflow, model
or inference failures, tool timeouts, guardrail blocks, manifest errors, manual
stops, and unknown terminal failures stop as foundation telemetry. They do not
route through Orchestrator by default. Ticket-gate failures keep their bounded
single repair path, and Orchestrator failures can still use deterministic
source handoff fallback when the trigger contains a safe non-Orchestrator
source disposition.

Ticket lifecycle completion must be atomic. Tool policy blocks copying a
feature ticket into `docs/tickets/done/`, blocks saving a done ticket while the
same ticket ID still exists in backlog/in-progress/in-review, and accepts
multiline `evidence_links` as valid preflight evidence. Feature contracts also
reject duplicate scenario heading IDs, and `ticket_create` strips duplicate
leading ticket-title headings from model-provided bodies.

### Consequences

- A productive Engineer run that overruns no longer becomes a planning loop.
- The operator sees one foundation telemetry failure to investigate rather than
  a new autonomous Orchestrator/CTO cycle.
- Feature and ticket artifacts stay cleaner during bootstrap, which makes the
  next QA or replay easier to reason about.

## AD-136: Dogfood Validation Is Observation-First

**Status:** Accepted
**Date:** 2026-05-19
**Owner:** Mars Harness maintainers

### Context

The next clean `demo-123` replay finally showed the intended product-first
shape. CEO, COO, CTO, Engineer, bounded Engineer repair, QA, and Security all
stayed on the Space Invaders lane. The target produced a product plan, BDD
feature contract, ordinary T-001 product ticket, browser-game implementation,
ticket completion, QA approval, and security review with zero target
intervention-debt tickets. Runtime guardrail and ticket-gate signals remained
foundation telemetry.

Dogfood then exposed the next boundary. It reached the right validation phase,
but spent 40 turns on shell-heavy validation, hit `max_turns`, and left
`package.json` plus `package-lock.json` dirty after trying to add start/dev
support itself. The runtime correctly stopped the failure as foundation
telemetry and did not route Orchestrator or create target intervention-debt,
but Dogfood had crossed from validation into product mutation.

### Decision

Dogfood is now observation-first by default. Generated Dogfood guidance forbids
editing product source, package manifests, lockfiles, config, or harness
scaffold to make validation pass. Missing scripts, missing dependencies, or
bootability gaps are target-owned findings created through `ticket_create`, not
Dogfood edits. Failed pre-flight checks stop validation after tickets and a
structured disposition. Foundation/runtime failures such as tool loops,
timeouts, guardrail blocks, model failures, and `max_turns` remain telemetry or
blocked dispositions unless an operator explicitly opts into ticket
materialization.

Tool policy backs that rule by blocking Dogfood `file_write` outside
`docs/reports/dogfood/*.md`. Dogfood can still write bounded evidence reports,
create target-owned tickets, record decisions, run bounded validation commands,
and commit finding artifacts. It cannot silently mutate package or product files
while claiming to be an end-to-end tester.

`git_push` also treats local demo repositories with no configured remote as a
clean skip. Roles still call the tool after commits, but a no-remote demo no
longer burns turns on push retries.

### Consequences

- Live replay remains the foundation operating model: run a clean demo, observe
  the next lifecycle bottleneck, patch the smallest durable guard, and replay.
- Dogfood validates the product without becoming an unscheduled Engineer.
- No-remote demo repositories keep local commits without noisy push failures.
- A failed Dogfood run now leaves one foundation signal and durable evidence,
  not product churn or intervention-debt starvation.

## AD-137: Engineer Product Mutation Requires A Ticket Claim

**Status:** Accepted
**Date:** 2026-05-19
**Owner:** Mars Harness maintainers

### Context

The next clean `demo-123` replay improved the bootstrap path: CEO, COO, and CTO
created a Space Invaders plan, feature contract, and ordinary product ticket,
and Engineer committed real browser-game files with no intervention-debt
tickets. The remaining failure was ticket truth. Engineer skipped the backlog to
in-progress claim, committed implementation against T-001 while the ticket still
lived in backlog, left package files dirty, and eventually ended with
`circle_detected`. The run made product progress, but the visible ticket
lifecycle did not describe that progress.

### Decision

Engineer product mutation is now mechanically claim-backed. When ordinary
product backlog tickets exist and no ticket is in `docs/tickets/in-progress/`,
Engineer cannot use product-mutating `file_write`, mutating `shell_exec`,
`dependency_sync`, `mars_harness_cli`, or `git_commit`. Ticket-only edits and
the `git mv docs/tickets/backlog/T-NNN-*.md docs/tickets/in-progress/` claim
move remain allowed so the model can recover in the same run. Generated
Engineer guidance names the same rule.

Successful Engineer dispositions are also tied to ticket state. A completed,
approved, or in-review disposition with `ticket_id` is rejected unless that
ticket lives in `docs/tickets/done/`; `no_work` can still finish without a
ticket ID.

### Consequences

- Fresh product work cannot become source commits while the ticket remains in
  backlog.
- The first visible Engineer action should be a ticket claim, making later QA
  and replay analysis easier.
- A missed claim becomes one actionable policy block instead of a late
  circle-detected failure with contradictory ticket state.

## AD-138: Live Demo Improvement Loop

**Status:** Accepted
**Date:** 2026-05-19
**Owner:** Mars Harness maintainers

### Context

The recent `demo-123` stabilization cycle found issues that unit tests and
fake-LLM harness checks could not expose in isolation: first-run planning could
stall behind intervention debt, dispatch could loop, Engineer could make real
product changes without a visible ticket claim, Dogfood could stop after
observing instead of advancing evidence, and job status could look noisier than
the actual product progress. The useful pattern was not one large redesign. It
was an iterative operating loop: run the harness against a clean representative
target, inspect concrete evidence, choose the next smallest source fix, test it,
and run the target again.

### Decision

Source lifecycle stabilization uses a continuous live demo improvement loop:

1. Start from remote trunk or record the trunk blocker.
2. Run a clean representative target. For first-run lifecycle work, `demo-123`
   with a small Space Invaders brief is one canonical replay, but it is not the
   whole acceptance suite.
3. Record evidence: exact command, target path, source ref or binary,
   database/log paths, job sequence, target commits/tickets/docs, telemetry,
   product progress, intervention-debt count, runtime artifacts, and stop
   reason.
4. Review findings into concrete categories: product-progress blocker,
   foundation/runtime defect, generated-target guidance gap, role/tool
   behavior gap, performance/noise issue, or target-owned product finding.
5. Select one or two bounded source actions tied to that evidence. Broad
   redesign, speculative governance expansion, or unrelated refactors are
   deferred.
6. Implement the bounded actions with deterministic tests and documentation.
7. Rerun a clean representative target or record the exact blocker and replay
   command.
8. Claim improvement only from rerun evidence: better product progress, fewer
   autonomous loops, fewer target intervention-debt tickets, clearer operator
   blockers, or a smaller remaining failure class.
9. Merge or fast-forward the confirmed fix according to the repo's trunk policy
   and push it to the remote before closing the loop. Branch-only or local-only
   confirmed fixes are incomplete.

This loop can repeat indefinitely. Each cycle should leave the repo with a
durable evidence trail and one narrower next problem, not a widening backlog of
process work.

The loop uses a small validation portfolio so source improvements do not
overfit to one project shape:

| Archetype | Example Brief | Catches |
| --- | --- | --- |
| Static browser app | `demo-123`: small browser game or utility with no package manifest. | Bootstrap, planning, product ticketing, static file creation, static smoke evidence, no-remote release handling. |
| Package-managed frontend | A Vite/React or Next.js app with `package.json`, build/dev scripts, and dependency hygiene. | Dependency sync, build commands, dev-server lifecycle, generated output ignores, UI smoke validation. |
| API or service | A small Go or Python HTTP service with tests and a local health endpoint. | Non-frontend project planning, service start checks, test selection, port readiness, package-free or compiled flows. |
| CLI/tooling project | A command-line utility with flags, help text, tests, and release notes. | CLI contract updates, docsync, command-level validation, release/version workflows without browser assumptions. |
| Existing repo maintenance | A seeded repo with pre-existing files, tickets, and one known bug or feature gap. | Resume behavior, ticket selection, dirty-worktree safety, no over-scaffolding, backward-compatible edits. |

Use the smallest matrix subset that matches the source change. A narrow static
prompt fix may replay only the static browser app. A generic tool, dispatch,
release, docsync, guardrail, or operating-model claim should run at least two
different archetypes or record why the second replay is blocked. Broad
lifecycle claims should run the static canary plus one package-managed or
service archetype before being treated as generally improved.

### Consequences

- Live target runs are not a one-time proof; they are the foundation operating
  model for stabilizing the harness lifecycle.
- Confirmation includes remote publication. A local green replay does not count
  as complete until the work is on the remote trunk or the remote blocker is
  recorded with replay and push steps.
- `demo-123` remains source-only shorthand for one canonical first-run replay.
  Generated targets inherit the generic product evidence loop and validation
  matrix idea, not the specific demo name or game domain.
- Generic foundation fixes should be phrased and tested against tool/runtime
  behavior across archetypes. Project-shape-specific guidance is still useful,
  but it cannot be the only evidence for a general lifecycle claim.
- Stabilization work stays product-first because the primary metric is whether
  the target advances from brief to product plan, feature contract, ticket,
  implementation, review, or dogfood evidence before governance or
  intervention-debt work takes over.
- Performance and evidence-cost findings are valid loop inputs once product
  progress is healthy. For intentionally static HTML/CSS/JS targets, generated
  role guidance should prefer bounded static HTTP smoke evidence, one full-file
  ticket evidence update, and observation-first Dogfood validation over
  package-manager, container, or repeated shell-edit churn that does not match
  the target's shape.
- Target-owned findings produced during validation must be committed before
  handoff. The `demo-123-run5` replay showed that an uncommitted Dogfood
  `ticket_create` result can strand Engineer behind claim guardrails because
  `git mv` cannot move an untracked ticket. Terminal dispositions that approve,
  request changes, block, or fail now require a clean tree for non-Orchestrator
  roles, while runtime-only `.harness/learnings.yaml` updates remain
  non-blocking by themselves.
- Failed validation jobs can still leave target-owned artifacts behind if the
  final tool call creates a ticket and the role then hits `max_turns`. The
  `demo-123-run6` replay showed the direct runtime-failure dispatch was
  quarantined correctly, but the later orchestrator survey routed Engineer for
  `dogfood_failure` while `T-002` was uncommitted. Survey routing now pauses
  for dirty target workspaces, excluding runtime-only `.harness/learnings.yaml`,
  so failed-role cleanup ends as an operator-visible blocker instead of an
  autonomous dirty-tree handoff.
- The next loop targets are allowed to stay explicit. After the current
  improvement, remaining findings such as Engineer tool/turn bloat or Dogfood
  continuation behavior should be addressed by another bounded run-review-act
  rerun cycle.

## AD-143: Bootstrap Planning Reuses Canonical Feature Contracts

**Status:** Accepted
**Date:** 2026-05-20
**Owner:** Mars Harness maintainers

### Context

The non-static `demo-api-run3` replay used a fresh Task Notes API target after
the scheduler and build-artifact fixes. CEO correctly found the generated
canonical `docs/features/F-001-product-walking-skeleton.md` contract, but then
also tried to create a more product-specific `docs/features/F-001-task-notes-api.md`
path. Tool policy blocked that duplicate path. COO then followed the same
ambiguous guidance: it first tried the duplicate path, then tried to update the
canonical file by appending scenario headings that duplicated starter IDs.
Those guardrails protected the target, but the bootstrap stalled before CTO
ticketing or Engineer validation.

This is a generic first-run issue, not an API-specific one. Any fresh project
can suggest a better slug than the generated starter contract, and any planner
can be tempted to append scenarios rather than rewrite the starter file.

### Decision

Generated bootstrap planning guidance now treats feature-contract path
canonicalization as an explicit role duty:

- CEO does not write `docs/features/` and does not invent new `F-001` feature
  paths during bootstrap. When CEO inspects feature contracts, it names the
  existing canonical path in the COO handoff.
- COO resolves `docs/features/F-NNN*.md` before writing. If any match exists,
  COO edits exactly that existing path, including the generated
  `docs/features/F-001-product-walking-skeleton.md` starter.
- COO may rename the product behavior inside the contract, but it does that by
  rewriting the existing contract in place with one unique scenario set rather
  than creating a second path or appending duplicate scenario headings.

The existing tool-policy duplicate-path and duplicate-scenario checks remain as
hard stops. The prompt change is the first layer: make the happy path obvious
so planners avoid the guardrail instead of repeatedly discovering it.

### Consequences

- Fresh targets keep one canonical feature-contract path per feature ID while
  still allowing product-specific behavior and scenario schedules.
- First-run planning should move from CEO to COO to CTO without burning turns on
  avoidable guardrail blocks.
- The next live API canary can test the build-artifact cleanup fix because
  bootstrap planning should no longer stall on duplicate `F-001` contract work.

## AD-144: Long-Running Validation Uses Managed Background Processes

**Status:** Accepted
**Date:** 2026-05-20
**Owner:** Mars Harness maintainers

### Context

The `demo-api-run6` Task Notes API replay confirmed that generated artifact
cleanup hints work in a real service target: Engineer built `task-notes-api`,
received the exact `rm task-notes-api` remediation, cleaned the binary, and
continued. The next bottleneck was not product planning or artifact cleanup. It
was server validation.

Engineer first ran `go run src/main.go` as a foreground command. The command
timed out after 30 seconds with startup logs, and a follow-up `curl` failed
because the foreground timeout had stopped the server. Engineer then tried to
emulate background execution inside `shell_command` with shell `&` and PID
management. That pattern left the compiled server process listening on
`8080`, caused a later tool-managed `background:true` start to fail with
"address already in use", and consumed turns on malformed `:8080` commands and
manual process cleanup.

This is a generic service and web-app issue, not a Task Notes API issue. Any
target that needs a local dev server, static server, API process, watcher, or
health probe can hit the same failure if agents turn shell background syntax
into their own process manager.

### Decision

Long-running validation is owned by the `shell_exec` tool boundary:

- `shell_exec` rejects the shell background operator `&` inside
  `shell_command`. Roles must use `background:true` for long-running servers
  and watchers, then run readiness probes as separate tool calls.
- `background:true` startup is no longer an unconditional success. If the
  process exits during the startup capture window, the tool returns an error
  with the initial output and exit code so roles treat port conflicts, crashes,
  and missing commands as boot failures.
- Foreground commands now set a short wait delay after timeout cancellation so
  leaked pipes cannot strand the tool until the outer executor TTL.
- Generated Engineer guidance explicitly forbids `cmd & PID=$!` style
  backgrounding and tells the role to probe and clean up the managed process
  deliberately.

### Consequences

- Web, API, static, and service targets share one validation rule instead of
  relying on per-language shell snippets.
- Agents should spend fewer turns on foreground server timeouts, broad `ps`
  inspection, port-conflict recovery, and malformed shell attempts.
- A crashed or port-conflicted dev server becomes immediate evidence to fix,
  not a false success that lets ticket closure proceed.
- The next API canary should confirm Engineer can validate a service without
  leaking port `8080` or enqueueing timeout recovery work after manual stop.

## AD-145: Validation Build Outputs Stay Outside Target Repositories

**Status:** Accepted
**Date:** 2026-05-20
**Owner:** Mars Harness maintainers

### Context

The `demo-api-run7` Task Notes API replay confirmed the managed-background
process fix in live conditions. Engineer started `go run main.go` with
`background:true`, probed `GET /health`, and killed the managed PID cleanly. The
next bottleneck was not service validation; it was validation build output.

Engineer ran `go build -o task-notes-api main.go`, which created an untracked
root binary inside the target repo. Blast-radius validation correctly rejected
the 34k-line binary-shaped diff and kept the failure as foundation telemetry,
but the repo was already dirty. The model then emitted malformed empty
`shell_exec` calls while the binary remained present, so pre-tool dirty-worktree
containment masked the malformed-call error with the blast-radius blocker and
the role ended with `circle_detected`.

Cleanup exceptions are necessary, but this replay showed they are recovery, not
prevention. A generic factory should not let validation commands create
repo-local compiled artifacts in the first place when an external temp output
path is sufficient.

### Decision

`shell_exec` now rejects `go build -o <path>` before process execution when the
explicit output path resolves inside the target repository. The error names the
bad output path and suggests an external temp output. Build validation that only
needs package coverage should use `go build ./...`; validation that needs a
runnable binary should write it outside the target repo, then run or discard it
there.

Malformed `shell_exec` invocations are also validated before the dirty-worktree
blast-radius precheck. That keeps a bad tool payload visible as a tool-call
shape error instead of obscuring it behind an existing generated-artifact
blocker.

### Consequences

- The preferred path is prevention: generated binaries should not enter target
  worktrees during validation.
- The bounded cleanup exception remains for older runs, user-created artifacts,
  and accidental binaries that already exist.
- Generic Go, API, CLI, and service targets get the same rule; this is not a
  Task Notes API special case.
- The next API canary should confirm Engineer reaches validation without
  creating a repo-local binary trap, then either commits completed ticket work
  or exposes the next generic lifecycle bottleneck.

## AD-146: Bare Port Tokens Are Invalid Validation Commands

**Status:** Accepted
**Date:** 2026-05-20
**Owner:** Mars Harness maintainers

### Context

The `demo-api-run8` replay confirmed the v0.42.7 build-output guardrail:
Engineer tried `go build -o task-notes-api src/main.go`, and `shell_exec`
blocked it before the `task-notes-api` binary appeared in the repo. The target
worktree stayed clean after the failed job, and the product implementation
commit was preserved.

The remaining loop was a malformed validation command. After the build-output
block, Engineer called `shell_exec` with `argv: [":8080"]` twice. That token is
a port, not a process to execute. The runtime returned a normal exec-not-found
error both times, and circle detection stopped the job. This is the same family
as the earlier run6 `:8080` cleanup attempts: local models sometimes collapse
"server on port 8080" into a command-shaped port token.

### Decision

`shell_exec` now rejects bare port tokens such as `:8080` in both `argv` and
single-token `shell_command` mode before process execution. The error states
that ports are not executable commands, instructs the role to start the app with
the real server command using `background:true`, and gives the corresponding
`curl http://localhost:8080/health` probe shape.

The repo-local build-output blocker also gives a more direct recovery action:
rerun the same build with `-o /tmp/<artifact>-validation` or another external
temp path, then run or delete that external binary. The point is to keep agents
on the validation path without turning generated artifacts or malformed port
tokens into loops.

### Consequences

- API, web, static-server, CLI, and service targets get a generic recovery hint
  when a port leaks into command position.
- Validation errors stay tool-shape errors instead of expensive subprocess
  failures.
- The next API canary should confirm Engineer reacts to the repo-local build
  block by using an external validation binary or managed server validation,
  rather than repeating `:8080`.
