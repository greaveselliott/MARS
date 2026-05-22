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

Failure ownership classification is a universal operating-model gate. Before a
live-loop, dogfood, telemetry, review, or operator finding becomes a ticket or
code change, agents classify it as foundation-owned, deployed-owned, or
mixed/unclear. Foundation-owned failures include runtime, orchestration,
generated-default, role-guidance, tool-policy, model/provider, release/update,
telemetry, or mirrored-doctrine defects. Their fixes belong in the
`mars-harness` source harness, runtime substrate, generated target defaults, or
foundation docs and should improve the affected class of users, not just the
current demo. Deployed-owned failures include target product behavior, target
architecture, local package/build/test setup, target docs, target skills, or
project policy. Their fixes belong in the target repo and should improve that
deployed harness or product. Mixed findings may use a small target unblock to
finish evidence, but the reusable defect still gets a foundation follow-up.
Unclear failures stay observations, telemetry, or investigation notes until
ownership is clear; they must not automatically become target backlog noise.

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
   generated directories, large generated diffs, host OS metadata noise, and
   forbidden deletions.
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
7. Host OS metadata such as `.DS_Store`, `Thumbs.db`, and `Desktop.ini` is
   deployed-harness hygiene noise. Generated targets ignore it from day one,
   and lifecycle/disposition gates do not treat it as product work that must
   reopen a completed ticket. Agents must not commit those files; if no product
   changes remain, they should record the disposition instead.

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

`shell_exec` now rejects `go build` before process execution when the output
would be implicit inside the target repository or when an explicit
`go build -o <path>` output path resolves inside the target repository. The
error names the risky output and suggests an external temp output. Build
validation that only needs compile evidence should use `go test ./...`;
validation that needs a runnable binary should write it outside the target repo,
then run or discard it there.

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

## AD-147: Scratch Validation Must Stay Out Of The Target Root

**Status:** Accepted
**Date:** 2026-05-20
**Owner:** Mars Harness maintainers

### Context

The `demo-api-run9` replay confirmed the previous `:8080` fix: Engineer did
not repeat the bare-port command loop, followed the build-output hint, used an
external validation binary, and the lifecycle advanced through QA, Security,
Dogfood, and Release Manager with zero target intervention-debt tickets. The
run also exposed the next generic validation sink.

Engineer created a repo-root `validate.sh` as scratch validation, then tried to
remove it with forbidden `rm`. Because ordinary deletion is intentionally
blocked, the script was committed when the ticket moved to done. Dogfood later
proved the product with direct `go test`, external build output, background
server, and `curl`, but it also ran the committed `validate.sh` and found that
the script depended on the non-portable `timeout` command. QA and Security did
not reject the script because it looked like a harmless support file.

This is not specific to Task Notes. Generic factory validation should avoid
turning temporary proof scripts into product surface, especially when the
script depends on platform utilities that the harness already owns as tool
fields.

### Decision

`file_write` now blocks creation of new root-level validation shell scripts
such as `validate.sh`, `validation.sh`, `verify.sh`, `smoke.sh`, and
`smoke-test.sh`. Existing project-owned scripts can still be edited, but new
scratch validation belongs in existing tests, direct `shell_exec` build/run/curl
evidence, or intentional durable validation code under a tests directory when
the ticket scope calls for it.

`shell_exec` also rejects external `timeout` and `gtimeout` executables before
process execution. Roles should use the tool's `timeout_seconds` field for
bounded foreground commands, or `background:true` plus separate probes for
long-running servers. Generated Engineer and Dogfood guidance mirrors this
rule.

### Consequences

- Validation remains source-only: scratch scripts no longer sneak into target
  commits as accidental product artifacts.
- Blast-radius containment stays strict. The fix does not broaden `rm`; it
  prevents the most common scratch-file trap before the file exists.
- API, CLI, web, and static targets share the same portable validation
  contract: existing tests, external build artifacts, managed background
  processes, and direct probes.
- The next API canary should confirm Engineer no longer creates root
  `validate.sh`, and that Dogfood can validate using direct commands without
  spending turns on script portability.

## AD-148: Background Cleanup Kills Wrapper Descendants

**Status:** Accepted
**Date:** 2026-05-20
**Owner:** Mars Harness maintainers

### Context

The first `demo-api-run10` replay after `v0.42.9` found that the prior run had
left a server child process on port `8080`. Dogfood had started `go run
main.go` with `background:true`; when the job ended, the harness killed the
tracked `go run` wrapper, but the compiled child process survived and kept the
port bound. The next Engineer hit `listen tcp :8080: bind: address already in
use`, repeatedly tried malformed `:8080` commands, and eventually stopped with
`circle_detected`.

The existing tool guards were working as containment: bare ports were rejected,
foundation-owned guardrail telemetry stayed out of the target backlog, and the
runtime failure did not dispatch into Orchestrator. But the source cause was
below model behavior. A software factory should not leak child servers between
target replays or between jobs in the same target.

### Decision

`shell_exec` background cleanup now snapshots tracked background processes,
discovers known descendant PIDs from the local process table, kills descendants
from leaf to root, then kills the tracked process group and process. This keeps
the existing process-group cleanup while covering wrapper commands such as
`go run` that can spawn a server child outside the wrapper's group.

The same process-tree cleanup also applies when an agent calls
`shell_exec` `kill <tracked-background-pid>` during a job. That makes targeted
cleanup of a managed background server equivalent to job-boundary cleanup,
instead of killing only the wrapper process. Agents still start long-running
servers with `background:true`, probe with separate commands, and avoid shell
`&` process management.

### Consequences

- Live target canaries should not inherit stale dev servers from previous
  background validation jobs.
- Same-job validation should not leave compiled `go run` child servers behind
  after a role kills the tracked background PID.
- Port-conflict handling becomes genuine target evidence again instead of a
  side effect of harness cleanup leakage.
- The next API canary should start from a clean port state after harness-owned
  cleanup and verify the scratch-validation guard without manual port cleanup.

## AD-149: No-Op Shell Calls Return Completion Guidance

**Status:** Accepted
**Date:** 2026-05-20
**Owner:** Mars Harness maintainers

### Context

The `demo-api-run13` replay confirmed the tracked-background cleanup fix:
Engineer started `go run src/main.go` with `background:true`, probed
`GET /health`, and `shell_exec` intercepted `kill -9 <tracked-pid>` to clean
the process tree. It then built `<validation-root>`, started that
external binary, verified the health route and negative HTTP cases, and updated
ticket evidence.

The job still failed with `circle_detected` after product validation had passed.
The repeated tool calls were no-op shapes: empty `argv` and single `:` commands
after a managed background process was running. Those calls did not represent
target product defects, but the previous behavior treated them as guardrail
errors. That created eight Engineer guardrail-block telemetry events and ended
the job before it could stop the tracked PID, move the ticket to done, commit,
and record a disposition.

### Decision

`shell_exec` now treats empty argv, blank argv, and single `:` calls as no-op
tool calls that do not execute a process. Instead of raising a guardrail error,
the tool returns explicit completion guidance: no-op calls do not wait for
background processes or finish ticket work; if background processes are active,
the output names the tracked PID(s); after probes, the role should stop the
tracked PID, update ticket evidence, move the ticket to done when appropriate,
commit, push, and call `job_disposition_record`.

Managed background startup output also includes the same discipline: probe the
server, stop the tracked PID when cleanup is needed, and do not use empty argv
or `:` as wait commands. Generated Engineer guidance mirrors that rule so the
target harness teaches roles the exit path before the tool has to recover.

### Consequences

- Harmless malformed no-op tool calls become a recovery hint instead of target
  backlog work, guardrail-loop noise, or an avoidable terminal failure.
- Active managed background PIDs remain visible to the role after a no-op call,
  which makes cleanup discoverable without broad `ps` or `lsof` exploration.
- Shell syntax in argv mode remains blocked when it would mutate or depend on
  shell parsing, such as `: > file`; only the empty/single-`:` no-op shapes are
  softened.
- The next API canary should confirm Engineer stops the validation binary and
  completes the ticket lifecycle rather than looping on no-op calls.

## AD-150: DocSync Findings Block Successful Handoffs

**Status:** Accepted
**Date:** 2026-05-20
**Owner:** Mars Harness maintainers

### Context

The `demo-api-run14` replay confirmed that no-op shell guidance improved the
live lifecycle: Engineer started, probed, and killed managed background
servers, moved `T-001` to done, and recorded a terminal disposition. The same
run exposed a quality escape. Engineer added malformed one-line
`MarsDocSync` metadata pointing at `docs/features/F-001-S002.md`, QA and
Security both ran `docsync_audit`, saw `FAIL:` findings, and still approved.
Dogfood then approved release readiness, and Release Manager generated local
release notes before the no-remote publication blocker stopped the chain.

The run also showed that a policy-blocked external `timeout` command was
classified as `tool_timeout` because the telemetry classifier matched
"timeout" before "tool policy blocked". That made a deterministic policy
rejection look retryable and enqueued a duplicate Dogfood job.

### Decision

Successful dispositions from implementation, review, validation, and release
roles now run the DocSync audit mechanically before acceptance. Engineer,
Pipeline Fixer, QA, Security, Dogfood, Release Manager, and Dependency Manager
cannot record `completed`, `approved`, or `in_review` while `docsync_audit`
has findings. The policy error names the failing files and tells the role to
fix metadata/docs or record `changes_requested`/`blocked` feedback instead.

`internal/docsync` now distinguishes foundation and deployed repos. The
foundation source checkout still enforces expected-doc mappings from prefix
rules. Deployed target repos still require valid top-of-file `MarsDocSync`
metadata and existing documentation references, but they are not forced to
reference foundation-only source docs. Generated Engineer, QA, Security,
Dogfood, and Release Manager prompts now spell out that scenario IDs are not
feature-contract paths, structured `MarsDocSync` blocks are required, and
DocSync `FAIL:` output blocks approval/release readiness.

Telemetry classification now handles guardrail/tool-policy strings before
generic timeout matching, so a rejected external `timeout` wrapper records as
a guardrail block rather than retryable `tool_timeout`.

### Consequences

- DocSync failures become hard lifecycle gates instead of optional reviewer
  judgement calls.
- Target Go apps under `cmd/` can satisfy DocSync with their own feature
  contracts rather than Mars Harness source documentation.
- Policy-blocked validation wrappers no longer enqueue duplicate Dogfood work
  through retry remediation.
- The next API canary should stop at Engineer rework if malformed DocSync
  metadata appears, or proceed through QA/Security/Dogfood/Release only when
  the target source passes DocSync.

## AD-151: No-Op Shell Calls And Missing DocSync Metadata Stop At Tool Boundaries

**Status:** Accepted
**Date:** 2026-05-20
**Owner:** Mars Harness maintainers

### Context

The `demo-api-run15` replay validated part of AD-150: CEO, COO, and CTO again
created product-specific planning and a single ordinary product ticket, runtime
guardrail signals stayed out of target intervention debt, and Engineer began
real Task Notes API implementation. The replay then exposed two earlier-boundary
gaps:

- Engineer wrote `src/main.go` and `src/main_test.go` without any
  `MarsDocSync` metadata. The successful-disposition gate would have blocked
  approval later, but the missing metadata was allowed into the worktree first.
- After `go test ./...` and an external build passed, Engineer repeatedly
  called `shell_exec` with empty `argv`. Soft no-op guidance was treated as a
  successful tool result and the run ended with `circle_detected`.

### Decision

No-op `shell_exec` calls remain non-executing, but they now return a tool error
alongside the same recovery guidance. The model sees that an empty `argv`,
blank `argv`, or single `:` command did not advance work, while still receiving
the tracked background PID list and the next completion steps: stop validation
processes, update ticket evidence, move the ticket to done when appropriate,
commit, push, and record `job_disposition_record`.

`file_write` now rejects source and test files under audited source roots, plus
root-level source files such as `main.go` or `index.html`, unless the first
write includes valid top-of-file `MarsDocSync` metadata pointing at existing
documentation. This catches missing metadata and scenario-ID doc paths such as
`docs/features/F-001-S002.md` before source files are created. The audit engine
also includes root-level source files in its source-file set so direct root
apps receive the same documentation-sync coverage as `src/` and `cmd/` apps.

### Consequences

- No-op shell drift is visible to the local model as a failed action, not a
  harmless success that can be repeated until circle detection.
- Deployed target agents must put documentation routing into the first source
  write, which keeps DocSync failures closer to the cause.
- The successful-disposition DocSync gate remains the backstop for source
  created by other mechanisms or existing files edited through shell commands.
- The next API canary should confirm Engineer writes valid metadata before
  source creation and exits toward ticket completion instead of empty shell
  calls.

## AD-152: Tool Policy Uses The Same Normalized Shell Args As Execution

**Status:** Accepted
**Date:** 2026-05-20
**Owner:** Mars Harness maintainers

### Context

The `demo-api-run16` replay confirmed the new DocSync preflight behavior:
Engineer's first attempted source write already contained a structured
`MarsDocSync` block pointing at the canonical feature contract. The run then
found a policy/execution mismatch. The local model emitted the required ticket
claim command as a JSON-encoded `argv` string:

```json
{"argv":"[\"git\", \"mv\", \"docs/tickets/backlog/T-001-implement-get-health-endpoint-for-task-notes-api.md\", \"docs/tickets/in-progress/\"]"}
```

`shell_exec` execution can normalize that shape, but the Engineer claim
exception still decoded `argv` directly as `[]string`. The policy therefore
blocked the exact command it had suggested, trapping Engineer behind the claim
gate.

### Decision

Shell tool policy paths now decode `shell_exec` arguments through the same
normalizing parser used by execution before evaluating the backlog-to-in-
progress ticket-claim exception. Simple malformed argv shapes that execution
would repair are therefore also understood by ownership and guardrail policy.

### Consequences

- The claim gate remains strict for unclaimed product mutations, but no longer
  blocks the exact `git mv` claim command solely because the model encoded
  `argv` as a JSON string.
- Tool policy and tool execution share one interpretation of normalized shell
  arguments, reducing local-model format drift.
- The next API canary should confirm Engineer can claim `T-001`, write
  DocSync-compliant source, and proceed to validation.

## AD-153: Engineer Shell Work Starts After Visible Ticket Claim

**Status:** Accepted
**Date:** 2026-05-20
**Owner:** Mars Harness maintainers

### Context

The `demo-api-run17` replay confirmed the previous claim normalization fix
allowed the product lifecycle to reach a committed CTO ticket and an Engineer
handoff with zero target intervention-debt tickets. The run then exposed the
next implementation-boundary gap. Engineer read the product ticket and feature
contract, but used shell discovery before claiming the backlog ticket. A broad
`find .` command was correctly blocked, but the model then repeated empty
`shell_exec` calls until `circle_detected` stopped the job. No source was
changed, and `T-001` remained in backlog.

The same replay showed a smaller but recurring local-model formatting issue:
COO attempted to record `job_disposition_record.evidence_links` as a string
containing a list literal before recovering with a strict array. The strict
schema was correct, but the extra retries spent turns without changing the
substance of the handoff.

### Decision

Engineer `shell_exec` calls now have a claim-first preflight. When an ordinary
product ticket exists in `docs/tickets/backlog/` and no ticket is already in
`docs/tickets/in-progress/`, the only allowed Engineer shell command is the
backlog-to-in-progress `git mv` claim. Read-only shell discovery, broad
traversal, validation, and no-op placeholder calls are rejected with the exact
claim command and commit message shape. Purpose-built read tools remain
available, but shell execution cannot preempt visible ticket ownership.

Dispatch disposition decoding now accepts strict arrays plus simple
list-as-string shapes for `evidence_links`, `work_product_ids`, `blocked_by`,
handoff constraints, handoff success evidence, and feedback evidence links.
The recorder normalizes JSON-encoded list strings, Python-style quoted list
strings, and single path strings before validation.

### Consequences

- The Engineer implementation path should move from CTO ticket handoff to
  visible ticket ownership before spending shell turns on discovery or
  validation.
- No-op shell calls before claim are redirected to the claim step instead of
  turning into repeated runtime failures.
- Disposition evidence remains structured, but harmless list formatting drift
  does not consume multiple role turns.
- The next API canary should confirm Engineer claims `T-001` before any shell
  discovery, then proceeds to source implementation with DocSync metadata.

## AD-154: Server Validation Uses Managed Background Execution

**Status:** Accepted
**Date:** 2026-05-20
**Owner:** Mars Harness maintainers

### Context

The `demo-api-run18` replay confirmed AD-153 in the live path. Engineer first
tried a read-only `ls` shell command before claim, policy rejected it with the
exact `git mv` claim command, and Engineer immediately claimed and committed
`T-001`. Engineer then recovered from the source-write DocSync preflight,
implemented `main.go`, added `main_test.go`, ran `docsync_audit`, and passed
`go test ./...`. This was material product progress.

The same run exposed the next process-control gap. After tests and an external
validation build passed, Engineer ran `go run main.go` in the foreground. The
tool timed out after 30 seconds with output showing the HTTP server was
running, then Engineer repeated empty `shell_exec` calls until
`circle_detected`. The failure was not product logic; it was a server
validation command that should have used managed background mode and a separate
readiness probe.

### Decision

`shell_exec` now blocks likely long-running server or watcher commands before
execution unless `background:true` is set. The first enforced shape is a Go
server entrypoint: `go run main.go`, `go run .`, or a package target is treated
as server-like when the referenced source contains common HTTP server markers
such as `ListenAndServe`, `http.Handle`, or a known router constructor. The
same preflight covers common dev-server commands such as `npm start`,
`npm run dev`, `pnpm dev`, `python -m http.server`, `uvicorn`, `gunicorn`,
`rails`, `vite`, and `next`.

The policy error tells the role to rerun the command with `background:true`,
probe readiness with a separate `curl` or equivalent command, and stop the
tracked PID after validation.

### Consequences

- Long-running validation moves from timeout recovery into a deterministic
  preflight.
- CLI-style `go run` programs without server markers can still run in the
  foreground.
- The next API canary should confirm Engineer starts the HTTP service with
  managed background execution, probes `/health`, stops the tracked PID, and
  moves `T-001` to done with evidence.

## AD-155: Security Review Uses Bounded Terminal Evidence

**Status:** Accepted
**Date:** 2026-05-20
**Owner:** Mars Harness maintainers

### Context

The `demo-api-run19` replay confirmed the run18 stabilization in the live
target path. CEO, COO, CTO, Engineer, and QA completed; Engineer claimed
`T-001`, wrote valid DocSync metadata, ran tests, built outside the repo,
started the external validation binary with `background:true`, probed
`/health`, stopped the tracked PID, moved `T-001` to done, and handed off to
QA. The target quality export recorded grade `C`, one done ticket, and zero
open intervention-debt tickets.

The next bottleneck moved downstream to Security. Security successfully
inspected the recent commits, ran docsync, ran tests, performed a managed
runtime smoke, and confirmed `GET /health`, but it also repeated equivalent
validation, killed one managed process before probing it, ran `ping` as a
liveness substitute, and hit `max_turns` before writing a report or recording
`job_disposition_record`. That failure was correctly quarantined as
foundation-owned telemetry and did not create target backlog work.

### Decision

Generated Security guidance now has a bounded review budget for feature work
that has already passed Engineer and QA:

- inspect recent diffs, scan for secrets, read the changed code and done
  ticket, run `docsync_audit`, and run the smallest relevant test command;
- treat `go test ./...` as enough compile evidence for ordinary Go security
  review unless the ticket explicitly needs runtime smoke evidence;
- when runtime smoke is needed, build outside the repo, start the exact
  external binary with `background:true`, probe while the process is still
  running, and then stop the tracked PID;
- after one successful smoke probe and cleanup, write the security report,
  commit it, push if a remote exists, and record `job_disposition_record`
  instead of running more liveness checks.

### Consequences

- Security review should spend fewer turns on repeated build/start/curl
  cycles after the product has already passed implementation and QA evidence.
- Runtime smoke remains available for security, but the role has an explicit
  stop condition after one successful probe.
- The next API canary should confirm Security writes
  `docs/reports/security/security-audit-<date>.md`, commits it, and records an
  approved or changes-requested disposition before hitting the turn limit.

## AD-157: Security Review Reports Remediation Instead Of Patching Product Code

**Status:** Accepted
**Date:** 2026-05-20
**Owner:** Mars Harness maintainers

### Context

The first CLI target in the representative live matrix,
`demo-cli-run1`, proved that product-first planning and implementation were not
specific to the Space Invaders or HTTP API canaries. CEO, COO, CTO, Engineer,
and QA reached a product-specific Note Stats CLI plan, feature contract,
ordinary product ticket, implementation, and review. The run also exposed two
generic lifecycle gaps:

- Engineer created a repo-root `debug.go` scratch probe. It was given valid
  `MarsDocSync` metadata, so the source metadata gate no longer blocked it,
  and it was committed with ticket lifecycle work despite being validation
  noise rather than product code.
- Security correctly ran `go test ./cmd/note-stats` and found a failing line
  count test, but then patched `cmd/note-stats/main.go` inside Security review
  and hit `max_turns` before writing a security report, committing the fix, or
  recording a disposition. The target was left dirty with a passing test fix in
  the worktree.

The runtime contained the failure properly: `max_turns` stayed foundation-owned
telemetry, no intervention-debt ticket was created, and the dirty target survey
paused instead of dispatching more autonomous work into the uncommitted state.
The remaining problem was role authority. Security is a reviewer. A functional
regression discovered during Security review should be handed back to Engineer
with exact evidence, not fixed opportunistically at the end of a review run.

### Decision

Security `file_write` is now limited to
`docs/reports/security/security-audit-<date>.md`. Attempts to write product
code, tests, tickets, feature contracts, or other repo files are rejected before
the worktree changes. The policy error tells Security to write the audit and
record `changes_requested` for Engineer when tests, implementation evidence,
docs, or code need remediation.

Generated Security guidance mirrors that boundary. If a relevant test fails,
Security writes a NEEDS_REMEDIATION audit and records
`job_disposition_record` with `status: changes_requested`,
`feedback.for_role: engineer`, the exact failing command, affected file, and
requested fix. Security still runs bounded evidence collection, but it does not
become an implementation role.

The scratch validation block also generalizes from root shell scripts to
new root-level scratch probes such as `debug.go`, `probe.go`, `scratch.py`, or
`validate.js`. Existing files can still be updated deliberately, and durable
validation belongs in tests or scoped validation code under an owned directory.

### Consequences

- Late reviewer-discovered functional defects route through the normal
  Engineer rework path with explicit evidence instead of leaving dirty product
  fixes behind a failed Security job.
- Root scratch probes can no longer bypass hygiene by adding valid DocSync
  metadata.
- The next CLI canary should confirm Security reports failing tests as
  `changes_requested`, or approves only after tests and DocSync pass, without
  mutating product files.

## AD-158: Review Rework Is Bounded By The Requested Evidence

**Status:** Accepted
**Date:** 2026-05-20
**Owner:** Mars Harness maintainers

### Context

The second CLI matrix canary, `demo-cli-run2`, confirmed the AD-157 authority
boundary: Security wrote only `docs/reports/security/`, committed its audit,
and no target intervention-debt tickets were created. The product ticket reached
done, `go test ./cmd/note-stats` passed, and `scores export` produced a target
quality grade of `B` with one done ticket and zero open intervention debt.

The same run exposed the next lifecycle inefficiency. Security marked a
`NEEDS_REMEDIATION` finding for `--text` flag behavior even after observing the
CLI already failed safely for missing or empty input. Orchestrator correctly
routed the `changes_requested` disposition to Engineer, but Engineer treated it
as broad rework: it inspected many paths, repeated guarded discovery, patched
the same surface, rebuilt and smoked the binary, then added extra newline probes
until `max_turns`. The runtime again contained the failure as foundation
telemetry and did not dispatch an autonomous loop, but product progress had
already been achieved and the remaining work was review accuracy and bounded
handoff discipline.

### Decision

Generated Security guidance now classifies findings from current evidence
instead of speculation. `NEEDS_REMEDIATION` is reserved for failing tests or
DocSync, current exploitable code paths, invalid input that succeeds unsafely,
secrets, credentials, or actionable dependency/configuration risk. Commands
that already exit non-zero safely and future-extension hardening notes are
recorded as PASS notes or low-severity observations, not Engineer rework.
Security report findings now include an evidence line and an explicit
`Required before release` field.

Generated Engineer guidance now has a review-rework fast path. When a
`changes_requested` handoff names a ticket that is already done or in review,
Engineer reads the exact feedback, reproduces the exact failing command or path,
and either records that no code change is needed or makes the smallest patch.
After the requested evidence passes, Engineer runs only the relevant test and
DocSync checks, commits any changed work, pushes, and records
`job_disposition_record`. Extra exploratory edge cases and broad smoke probes
belong in follow-up tickets or Dogfood evidence, not in the same rework job.

### Consequences

- Security should stop converting speculative hardening into product-blocking
  rework.
- Engineer rework should converge after the requested evidence passes instead
  of expanding validation until the turn budget is exhausted.
- The next CLI canary should confirm a Security false positive becomes an
  approved report or a bounded changes-requested handoff that Engineer closes
  with a terminal disposition.

## AD-159: Engineer Implementation Is Bounded By The Ticket Contract

**Status:** Accepted
**Date:** 2026-05-20
**Owner:** Mars Harness maintainers

### Context

The next CLI canary, `demo-cli-run3`, validated that the patched target
generated the new Security and Engineer review-rework guidance, but it exposed
an earlier failure class before Security could run. CEO, COO, and CTO produced
a product-specific plan, feature contract, and implementation ticket. Engineer
claimed `T-001` correctly and did not create root scratch probes, but then
spent 50 turns on open-ended CLI implementation and exploratory edge cases.

The target ticket already said empty `--text ""` should produce zero words,
zero characters, and zero lines. Engineer drifted between alternative empty
text semantics, rewrote tests to follow the implementation rather than the
ticket, investigated character counts with ad hoc commands, and hit
`max_turns` with uncommitted product files. Runtime containment again worked:
the failure stayed foundation-owned telemetry, did not route through
Orchestrator, and created no target intervention-debt ticket. The remaining
gap is that initial implementation needs the same bounded shape as review
rework: the selected ticket and BDD scenario are the contract for the run.

### Decision

Generated Engineer guidance now has a contract-first implementation rule.
Before product writes, Engineer must treat the selected ticket acceptance
criteria and feature contract as the product contract. Tests and code should
be aligned to that contract, not rewritten to justify exploratory behavior.
If an edge case is genuinely ambiguous, Engineer should update the feature
contract or record a blocked disposition rather than inventing semantics. If
an exploratory edge case is useful but outside the selected ticket, it becomes
follow-up evidence rather than same-run validation.

After the required acceptance criteria and one relevant test suite pass,
Engineer should update ticket evidence, move the ticket to done, commit, push,
and record `job_disposition_record` instead of continuing to add unrequested
edge probes.

### Consequences

- Feature delivery should converge on the selected product slice instead of
  expanding until `max_turns`.
- QA and Dogfood remain available for broader edge-case validation after the
  initial ticket is complete.
- The next CLI canary should confirm Engineer commits a passing Note Stats
  implementation and records disposition before Security review.

## AD-160: Ticket Closure Precedes Packaging Exploration

**Status:** Accepted
**Date:** 2026-05-20
**Owner:** Mars Harness maintainers

### Context

The follow-up CLI canary, `demo-cli-run4`, confirmed the AD-159 contract fix
improved the live path. Engineer claimed `T-001`, implemented the Note Stats
CLI, honored the explicit empty-text contract by producing zero words, zero
characters, and zero lines for `--text ""`, proved the hello-world and
multiline cases with an external validation binary, ran
`go test ./cmd/note-stats`, updated ticket evidence, and committed product
code.

The job still hit `max_turns` before QA. After the selected ticket evidence
had passed and product code was committed, Engineer continued exploring
packaging/build-output work. It tried `go build -o bin/note-stats`, hit the
argv guardrail, then wrapped the same repo-local build in
`shell_command: "mkdir -p bin && go build -o bin/note-stats ..."`. That
created an untracked binary before blast-radius validation stopped the tool.
The runtime correctly kept the failure as foundation telemetry and created no
target intervention-debt ticket, but the product slice was ready and the
remaining turn sink was post-success closure discipline.

### Decision

Generated Engineer guidance now treats successful acceptance evidence as a
closure trigger. Once the selected ticket's acceptance criteria and one
relevant test suite have passed, Engineer must update ticket evidence, move the
ticket to done, commit, push, and record disposition before packaging,
install, distribution, or extra build-output exploration. Repo-local binaries,
release packages, installer artifacts, and other distributable outputs are not
part of an ordinary feature ticket unless the selected ticket explicitly asks
for them.

`shell_exec` build-output policy also scans shell command segments, not only
commands that start with `go build`. Shell commands such as
`mkdir -p bin && go build -o bin/app ...` are rejected before execution when
the Go build output would land inside the target repository.

### Consequences

- A working feature slice should close and move to QA instead of drifting into
  packaging work.
- Go CLI and service validation can still use external temp binaries as proof,
  but target repos stay source-only unless packaging is explicit scope.
- The next CLI canary should confirm Engineer moves the Note Stats ticket to
  done after the required tests and external validation pass.

## AD-161: Dogfood Findings Commit Before Further Validation

**Status:** Accepted
**Date:** 2026-05-20
**Owner:** Mars Harness maintainers

### Context

The next CLI canary, `demo-cli-run5`, confirmed AD-160 materially improved the
live lifecycle. Engineer moved `T-001` to done before any repo-local packaging
artifact was created, QA approved the finished ticket, and Security completed a
bounded audit report. Dogfood then found a legitimate target-owned product gap:
running the Note Stats CLI with no `--text` argument produced zero-count JSON
instead of a required-argument error.

That finding belonged in the target backlog, but Dogfood created two duplicate
uncommitted tickets for the same issue and continued validation until
`max_turns`. Runtime containment held and no foundation/runtime failures were
materialized as intervention debt, but the useful product finding was left as
dirty target state instead of a clean handoff Engineer could claim.

### Decision

Dogfood-created target findings are now a hard handoff point. After
`ticket_create` produces an uncommitted ticket under the ticket lifecycle
directories, Dogfood may inspect status, diff, read files, commit, push, or
record disposition. Further shell validation and additional `ticket_create`
calls are blocked until the finding ticket is committed and the run records a
terminal disposition.

Generated Dogfood guidance now says the same thing in workflow language: once a
target-owned finding exists, stop additional validation, commit the finding,
attempt push when a remote exists, and record `changes_requested` or another
terminal disposition before continuing.

### Consequences

- Dogfood remains able to create product-defect tickets, but cannot flood the
  backlog with semantically duplicate findings in the same dirty state.
- Target-owned findings become clean, claimable evidence rather than untracked
  files that later roles cannot move or claim.
- Runtime and harness failures remain foundation telemetry; this rule is about
  target-owned finding handoff, not turning foundation failures into backlog
  work.
- The next CLI canary should confirm Dogfood commits exactly one missing-arg
  finding and records a terminal handoff instead of hitting `max_turns`.

## AD-162: Review Approval Requires Live Validation Evidence

**Status:** Accepted
**Date:** 2026-05-20
**Owner:** Mars Harness maintainers

### Context

The `demo-cli-run6` replay validated the AD-161 handoff direction and exposed a
more precise quality escape. The run reached product-specific planning,
ticketing, implementation, QA, Security, Dogfood, Orchestrator, and Engineer
rework with zero intervention-debt tickets. Dogfood created exactly one
target-owned finding for a failing Note Stats test expectation, committed it,
and Orchestrator routed the rework back to Engineer.

Two generic lifecycle problems remained:

- QA approved `T-001` without running the authoritative test suite even though
  `cmd/note-stats/main_test.go` existed.
- Security later ran `go test ./...`, saw a failing test, and still recorded an
  approved disposition. Dogfood caught the same failure and created `T-002`.

The same run showed two smaller tool-policy issues. Literal newline arguments
in `shell_exec` argv mode were rejected even though argv does not use shell
parsing, causing avoidable guardrail tax for multiline validation. The Engineer
then fixed `T-002` and got tests passing, but closing the Dogfood-created
enabler ticket fought the done-move evidence policy because an enabler with
`end_to_end_evidence: required` was treated as if it had to become a feature
ticket.

### Decision

The tool executor now records validation command outcomes in the job session.
`shell_exec` commands recognized as tests or builds increment validation
success/failure counters based on the actual exit code. QA and Security cannot
record `approved`, `completed`, or `in_review` for a named ticket unless the
current job has run an authoritative validation command successfully. If test
files are present, a successful test command is required. Any failing
validation command in the same review job blocks approval and instructs the
role to record `changes_requested` with the exact failing command and Engineer
next action.

Dogfood finding handoff is also strengthened from "while the ticket is
uncommitted" to "after this run creates a Dogfood finding." Once Dogfood calls
`ticket_create`, further validation or another ticket is blocked for the rest
of that run; the allowed path is status/diff/read, commit, push if possible,
and `job_disposition_record`.

`shell_exec` argv validation now allows literal newline characters inside an
argument. It still rejects shell control tokens, redirection, command
substitution, backticks, shell builtins, and other syntax that requires
`shell_command`.

The ticket done-move policy now distinguishes feature tickets from enabler or
remediation tickets. Feature tickets still require BDD scenarios,
`end_to_end_evidence: required`, evidence links, and verifier metadata before
moving to `done/`. Enabler tickets that explicitly require evidence can close
with evidence links and verifier metadata without pretending to be a feature.

### Consequences

- QA and Security approval now has a mechanical "tests/builds actually passed
  in this job" floor instead of relying on prompt compliance.
- Dogfood findings become a single clean handoff rather than a committed ticket
  followed by more same-run exploration.
- Multiline CLI/app validation can use structured argv without falling back to
  shell strings.
- Engineer can close Dogfood-created remediation work once evidence is present
  instead of fighting feature-ticket metadata.
- The next canary should confirm QA/Security approval is blocked until tests
  pass, Dogfood stops immediately after a finding handoff, and Engineer closes
  remediation tickets without `max_turns`.

## AD-163: Feature Scenario IDs Must Match Their Contract Path

**Status:** Accepted
**Date:** 2026-05-21
**Owner:** Mars Harness maintainers

### Context

The `demo-cli-run7` replay used a fresh Note Stats CLI target to avoid
overfitting the live loop to the earlier Space Invaders project. CEO and COO
made product-specific progress: CEO defined the first slice from `README.md`,
COO updated the active plan and `docs/features/F-001-product-walking-skeleton.md`,
and both committed their work. CTO then tried to create the first implementation
ticket but repeatedly hit the ticket planning-order guardrail.

The root cause was a malformed feature contract. COO appended scenario headings
`F-002-S001`, `F-002-S002`, and `F-002-S003` inside the existing `F-001`
contract. CTO reasonably copied those IDs into `ticket_create`. Tool policy
then looked for `docs/features/F-002*.md`, found none, and blocked the ticket.
The role did not recover cleanly and spent turns rereading plan ranges. This is
the same product-first lifecycle class as earlier loops: the harness was close
to useful delivery, but a small doctrine mismatch stalled the first product
ticket.

### Decision

Feature contract writes now enforce scenario/file alignment mechanically. A
scenario heading inside `docs/features/F-001*.md` must use an `F-001-SNNN` ID.
If a role writes `F-002-SNNN` inside that file, `file_write` blocks the update,
names the mismatched heading line, and tells the role to rename the heading or
create/update the matching `docs/features/F-002*.md` contract before ticket
creation.

Generated COO guidance now states that scenario heading IDs must match the
feature contract path, and generated CTO guidance says to route feedback back
to COO instead of creating tickets when the plan or contract shows mismatched
scenario IDs.

### Consequences

- The bad `F-002-SNNN`-inside-`F-001` shape is stopped at planning time rather
  than discovered later by `ticket_create`.
- CTO should see fewer ambiguous ticket policy blocks during fresh bootstrap.
- The next clean canary should reach an ordinary product ticket using scenario
  IDs from the same canonical feature contract path.

## AD-164: Post-Validation Engineer Runs Must Converge Before Context Overflow

**Status:** Accepted
**Date:** 2026-05-21
**Owner:** Mars Harness maintainers

### Context

The `demo-cli-run8` replay confirmed AD-163 on a fresh Note Stats CLI target.
CEO and COO produced product-specific planning, CTO created `T-001` from the
canonical `F-001` contract, and Engineer claimed the ticket, implemented the
CLI, initialized the Go module, and validated the requested behavior:

- `go run ./cmd/note-stats --text "hello world"` returned two words, eleven
  characters, and one line.
- empty `--text` returned zero counts.
- multiline text returned three lines.
- missing `--text` exited non-zero with an actionable error.
- `go test ./...` and `go build -o <validation-root> ./cmd/note-stats`
  passed.

After the implementation commit, Engineer did not move the evidenced ticket to
`docs/tickets/done/` or record `job_disposition_record`. Instead it spent more
turns on broad inspection, `find .`, malformed `shell_exec`, `ls /tmp`, and
other non-product probes. The run ended with `context_overflow` at roughly 41k
prompt tokens. This was still an improvement over earlier loops: the runtime
quarantined the overflow as foundation telemetry, created no intervention-debt
ticket, and did not dispatch Orchestrator into a containment loop. The remaining
defect was convergence after useful product work already existed.

### Decision

Engineer `shell_exec` now has a post-validation completion gate. Once the
current job has successful validation evidence and a successful implementation
commit while an ordinary product ticket remains in `docs/tickets/in-progress/`,
additional exploratory shell commands are blocked. The allowed shell path is
the lifecycle move:

```text
git mv docs/tickets/in-progress/T-NNN-*.md docs/tickets/done/
```

The policy error instructs Engineer to update ticket evidence if needed, move
the ticket to `done/`, commit the lifecycle move, and record
`job_disposition_record` with `next_need: qa_review`.

The agent loop context pruner now also prunes old assistant tool-call arguments
and older prose after pruning old tool results. This matters because large
historical `file_write` payloads and shell command outputs can keep the prompt
over provider limits even when tool result bodies were already replaced.

### Consequences

- Validated implementation runs should stop at ticket lifecycle completion
  instead of spending turns on broad shell exploration.
- Context overflow is less likely in long-running jobs with large historical
  tool calls.
- The next canary should confirm Engineer moves `T-001` to `done/`, records a
  QA handoff, and reaches review without intervention-debt amplification.

## AD-165: Review Rework Must Reopen The Ticket Before Product Mutation

**Status:** Accepted
**Date:** 2026-05-21
**Owner:** Mars Harness maintainers

### Context

The `demo-cli-run9` replay confirmed the post-validation convergence gate
shifted the lifecycle forward: Engineer moved `T-001` to
`docs/tickets/done/`, recorded a QA handoff, and QA began review instead of the
run ending in context overflow. QA then correctly blocked approval because the
Engineer had not run the authoritative test suite after implementation. QA
recorded `changes_requested`, Orchestrator routed rework to Engineer, and the
second Engineer pass ran `go test ./...`, found failing tests, and began
repairing them.

The defect was ticket state. The ticket stayed in `docs/tickets/done/` while
Engineer performed review rework. Because there was no backlog or in-progress
ticket, the product-mutation claim gate did not apply; Engineer could edit
`main_test.go` and continue shell validation while the repo-visible lifecycle
still said the ticket was done. That weakens review truth, hides rework from
ticket ordering, and lets QA rejection become an implicit side channel instead
of an ordinary product workflow.

The same run also showed the initial AD-164 gate was slightly too eager. It
counted an earlier ticket-claim commit plus a later build success as enough to
block additional validation while implementation files were still dirty. QA
caught the missing test evidence, but the policy needed to distinguish "a clean
committed implementation is ready for lifecycle close" from "a job has some
prior commit and has started validation."

The follow-up `demo-cli-run10` replay confirmed the rework guard but exposed a
completion-path edge case. Engineer implemented the selected CLI slice, added
tests, passed `go test ./cmd/note-stats`, passed `docsync_audit`, and moved
`T-001` from `docs/tickets/in-progress/` to `docs/tickets/done/`. The first
version of this rule then interpreted the pending done ticket as rework and
blocked the final implementation commit until the job hit `max_turns`. The
policy therefore needs to recognize an in-progress-to-done ticket move as
ordinary completion, not hidden rework.

### Decision

Engineer product mutation now requires an in-progress ordinary product ticket
not only for backlog work, but also for review rework. If no in-progress or
backlog product ticket exists and a product ticket lives in
`docs/tickets/in-review/` or `docs/tickets/done/`, product file writes and
mutating shell work are blocked until Engineer reopens the ticket with:

```text
git mv docs/tickets/done/T-NNN-*.md docs/tickets/in-progress/
```

or the equivalent move from `docs/tickets/in-review/`. That move is allowed as
a ticket-only mutation, and the policy error tells Engineer to commit the
rework claim before running validation or changing product files.

The normal completion path remains legal. When the worktree already contains a
ticket move from `docs/tickets/in-progress/` to `docs/tickets/done/`, the
implementation commit may include product files, tests, and that lifecycle
move. The rework guard only applies to product mutation after the ticket is
already done or in review with no active completion move.

The post-validation shell convergence gate now checks the worktree before
blocking. If product files or ticket files are still dirty, validation shell
commands remain available. The gate only blocks exploratory shell calls after
successful validation, a successful commit in the job, a remaining in-progress
ticket, and a clean implementation tree.

Generated Engineer guidance now says that a `changes_requested` handoff for a
done or in-review ticket must reopen that ticket before shell validation or
product/source edits when rework is required.

### Consequences

- QA/Security rejection becomes visible repo state again: the ticket is no
  longer silently "done" while implementation rework happens.
- Engineer can still read the done ticket and review feedback before reopening,
  but cannot mutate product code or run validation shell work until the ticket
  is back in the active queue.
- The post-validation stop rule should no longer block legitimate tests while
  implementation files are still uncommitted.
- The next canary should confirm the rework path reopens `T-001`, reruns
  tests, commits the fix, moves the ticket back to done, and returns to QA
  without hidden ticket-state drift.

## AD-166: Repeated No-Op Shell Calls After Validation Are A Loop Boundary

**Status:** Accepted
**Date:** 2026-05-21
**Owner:** Mars Harness maintainers

### Context

The `demo-cli-run11` replay confirmed `demo-cli-run10`'s completion-commit
patch was ready for a live check, but Engineer failed earlier in the lifecycle.
After claiming `T-001`, writing the Note Stats CLI, building it to
`<validation-root>`, and running successful CLI probes, Engineer
called `shell_exec` with empty `argv` repeatedly instead of committing dirty
implementation files or closing the ticket. The no-op tool already returned a
hard error with completion guidance, but the model retried empty calls until
the agent loop stopped with `circle_detected`.

The failure was useful: no target intervention-debt tickets were created, and
the runtime stopped as foundation telemetry. But relying on the generic
circle detector wastes turns and leaves the dirty product work uncommitted.
The no-op recovery path needs to become an explicit convergence boundary when
the job has already produced validation evidence and dirty ticket work.

### Decision

`shell_exec` records no-op failures in the session. If Engineer has already
had a no-op shell failure, has successful validation evidence, has dirty
disposition-blocking files, and still has an ordinary product ticket in
`docs/tickets/in-progress/`, a repeated no-op is blocked before execution with
specific lifecycle guidance: stop shell placeholders, run `git_status`, update
ticket evidence if needed, `git_commit` the implementation and ticket files,
move the ticket to `docs/tickets/done/` when acceptance evidence is present,
commit that lifecycle move, and record `job_disposition_record` with
`next_need: qa_review`.

Generated Engineer guidance and the generic no-op result now also say not to
retry empty `argv` or single `:` calls after validation or dirty work.

### Consequences

- Local-model placeholder calls become a faster, clearer boundary before the
  broader circle detector fires.
- Engineer keeps validation freedom while work is changing, but repeated
  no-op calls after evidence now point at commit and ticket completion.
- The next CLI canary should confirm Engineer commits dirty CLI work instead
  of repeating no-op shell calls, then reaches the ticket completion commit
  and QA handoff check.

## AD-167: Review Validation Uses Fresh Session-Built Artifacts

**Status:** Accepted
**Date:** 2026-05-21
**Owner:** Mars Harness maintainers

### Context

The `demo-cli-run12` replay confirmed the no-op and completion-commit fixes
moved the lifecycle materially forward. CEO, COO, and CTO created a
product-specific Note Stats plan, BDD contract, and implementation ticket with
no target intervention-debt tickets. Engineer claimed `T-001`, implemented the
CLI, ran useful probes, moved the ticket to `docs/tickets/done/`, committed
the implementation plus ticket lifecycle move, and handed off to QA.

The next failure was review validation. QA correctly tried to enforce the
in-job validation policy, but the default generated QA manifest did not include
`shell_exec`, so QA could not run the required authoritative command. QA
therefore requested changes instead of approving. The follow-up Engineer
rework then exposed two shell-evidence problems:

- Engineer ran `<validation-root>` before proving that binary had
  been built in the current role session. A stale binary from an older dogfood
  run can make evidence look current when it is not.
- After Engineer reopened the ticket and rebuilt with
  `go build -o <validation-root>`, the post-validation convergence
  gate treated the fresh binary execution as exploratory shell work and blocked
  the exact proof needed to answer QA.

### Decision

Generated QA now has `shell_exec`, but only as a bounded validation surface.
The QA prompt says to use it for authoritative tests, builds, and direct
runtime probes named by the ticket or BDD scenario, not for product mutation,
broad discovery, package-manager setup, or cleanup. QA remains read-only for
repo writes by default.

`shell_exec` session tracking now records successful external validation
artifacts built with commands such as:

```text
go build -o /tmp/<project>-validation <entrypoint>
```

Executing a `<validation-root>` binary is trusted only if that path was
built successfully earlier in the same role session. Otherwise policy blocks
the command and instructs the role to rebuild the binary first. When the path
is fresh, running it counts as validation evidence and is allowed through the
Engineer post-validation convergence gate.

### Consequences

- QA can satisfy the approval policy with real in-job command evidence instead
  of routing unnecessary rework only because its tool surface was incomplete.
- External validation binaries remain useful, but stale `/tmp` artifacts cannot
  silently prove a current target change.
- Engineer can answer QA rework with the intended build-then-run evidence path
  without reopening a loop after the fresh build succeeds.
- The next CLI canary should confirm QA runs validation itself, approves with
  evidence, and routes to Security or the next release/dogfood step without an
  unnecessary Engineer rework lap.

## AD-168: Direct Runtime Probes Count As Validation Evidence

**Status:** Accepted
**Date:** 2026-05-21
**Owner:** Mars Harness maintainers

### Context

The `demo-cli-run13` replay confirmed the first-run lifecycle now reaches
product implementation without target intervention-debt tickets. CEO, COO, and
CTO produced a product-specific Note Stats plan, feature contract, and
ordinary implementation ticket. Engineer claimed `T-001`, implemented the CLI,
and successfully ran:

```text
go run cmd/note-stats/main.go --text "hello world"
```

The command exercised the product behavior and returned the expected JSON.
However, the role session only classified build and test commands as
validation. Because the direct runtime probe was invisible to the convergence
gate, Engineer's later empty `shell_exec` calls received only generic no-op
guidance. The job stopped with `circle_detected` while the target had useful
product code committed but `T-001` still in `docs/tickets/in-progress/`.

Containment behaved correctly: the loop was recorded as foundation telemetry,
no target intervention-debt ticket was created, and no Orchestrator recovery
loop was dispatched.

### Decision

Successful direct runtime commands now count as validation evidence when they
execute product behavior and exit successfully. Recognized evidence includes
language run commands such as `go run`, `cargo run`, and `dotnet run`, common
interpreter entrypoints, package start scripts, and bounded smoke probes.

Generated Engineer guidance now names this rule explicitly: once a direct
runtime probe passes and the implementation is committed, the next work is to
update ticket evidence, move the ticket to `docs/tickets/done/`, commit the
lifecycle move, and record `job_disposition_record`, not to issue placeholder
shell waits.

### Consequences

- CLI and small-tool targets can prove behavior with direct execution even
  before a fuller test suite exists.
- The existing post-validation convergence gate now catches the live
  `go run` success path and redirects no-op/exploratory shell calls toward
  ticket closure.
- Review roles still require stronger evidence when test files exist; runtime
  probes are a validation signal, not a license to skip authoritative tests.
- The next CLI canary should confirm Engineer turns a successful direct
  runtime probe into ticket completion and QA handoff instead of stopping with
  `circle_detected`.

## AD-169: Expected Runtime Error Probes Do Not Poison Review Approval

**Status:** Accepted
**Date:** 2026-05-21
**Owner:** Mars Harness maintainers

### Context

The `demo-cli-run14` replay confirmed AD-168 fixed the Engineer-side
convergence path. CEO, COO, and CTO produced product-specific Note Stats
planning, a feature contract, and an ordinary implementation ticket with zero
target intervention-debt tickets. Engineer claimed `T-001`, implemented the
CLI, ran direct runtime and test evidence, committed product code, moved the
ticket to `docs/tickets/done/`, committed the lifecycle move, and handed off
to QA without the prior no-op `circle_detected` failure.

QA then used its bounded validation shell surface as intended. It built a
fresh `<validation-root>` binary, ran positive CLI probes, ran an
expected negative probe for missing `--text`, and ran `go test`. The negative
runtime probe exited non-zero because the product correctly rejected invalid
input. However, review approval policy treated every failing validation command
the same way. `job_disposition_record` therefore blocked QA approval even
though the failed runtime command was the expected evidence for the invalid
input path and the authoritative tests passed.

Containment remained correct: the failure stayed foundation telemetry, no
target intervention-debt ticket was created, and the target product work was
cleanly committed. The remaining issue was evidence classification. A failed
build or failed test means the reviewed artifact is not ready. A documented
non-zero runtime probe can be the desired proof for a negative-path scenario.

### Decision

Validation outcome tracking now separates build/test failures from broader
runtime probe failures. QA and Security approval still requires successful
in-job validation evidence, and if test files exist it still requires a
successful test command. Approval is blocked after any failing build or test
command in the same review job.

Non-zero direct runtime probes no longer poison review approval by themselves.
They may be used as evidence for expected error behavior when the role
documents the command and observed result, pairs it with positive runtime
evidence where relevant, and follows it with passing authoritative tests when
tests exist. If the runtime error is unexpected or product-owned, the review
role should still record `changes_requested` with the exact failing command
and requested Engineer action.

### Consequences

- QA can verify required error handling, permission failures, invalid input,
  and other negative-path behavior without being forced into false
  changes-requested loops.
- Failed builds and tests remain hard release blockers for approval.
- Runtime probes remain evidence, but reviewers must distinguish "expected
  product rejection" from "unexpected validation failure" in the disposition.
- The next CLI canary should confirm QA approves the Note Stats ticket after
  positive probes, the expected missing-argument rejection, and passing tests,
  then routes to Security without target intervention-debt amplification.

## AD-170: Post-Validation Completion Guidance Leaves The Shell Path

**Status:** Accepted
**Date:** 2026-05-21
**Owner:** Mars Harness maintainers

### Context

The `demo-cli-run15` replay was intended to validate AD-169 in QA, but it
failed earlier in the Engineer lifecycle. CEO, COO, and CTO again produced
product-specific Note Stats planning, feature-contract updates, and an
ordinary implementation ticket with no target intervention-debt tickets.
Engineer claimed `T-001`, implemented the Go CLI, committed source, recovered
from an unused-import build failure, built a fresh external validation binary,
and ran a successful runtime probe.

The post-validation convergence gate did fire: after the clean implementation
commit and successful validation, additional `shell_exec` calls were blocked
because `T-001` still lived in `docs/tickets/in-progress/`. The model still
repeated empty `shell_exec` calls until `circle_detected`. The target state was
clean and useful product code was committed, but ticket evidence and the
`docs/tickets/done/` move were missing.

The previous error text correctly mentioned "update ticket evidence" and
`git mv ... docs/tickets/done/`, but it did not name the concrete non-shell
tool sequence strongly enough. In practice, the local model kept reaching for
`shell_exec` because the recovery path still included a shell command later in
the sequence.

### Decision

The post-validation completion policy now makes the next allowed action
unambiguous. When Engineer already has successful validation, a clean
implementation commit, and an ordinary product ticket still in progress, any
non-lifecycle `shell_exec` error says:

- do not call `shell_exec` again except the exact `git mv` lifecycle move;
- next use `file_read` on the in-progress ticket;
- then use `file_write` on that same ticket to populate `evidence_links` and
  `verified_by`;
- only after evidence is updated, run the exact `git mv` to
  `docs/tickets/done/`, commit the lifecycle move, and record
  `job_disposition_record`.

Generated Engineer persona and target guidance mirror the same wording so the
model sees the non-shell recovery path before it encounters the policy error.

### Consequences

- A completed, validated implementation should leave the shell loop faster and
  perform the ticket evidence update the policy requires.
- The lifecycle still does not auto-close tickets; the repo-visible evidence
  remains an agent action, not hidden runtime mutation.
- The next CLI canary should confirm Engineer updates `T-001`, moves it to
  `docs/tickets/done/`, records QA handoff, and finally reaches the AD-169 QA
  review check.

## AD-171: Review Approval Distinguishes Expected And Unexpected Runtime Failures

**Status:** Accepted
**Date:** 2026-05-21
**Owner:** Mars Harness maintainers

### Context

The `demo-cli-run16` replay validated the AD-170 Engineer recovery path:
Engineer updated the in-progress ticket evidence, moved `T-001` to
`docs/tickets/done/`, committed the lifecycle move, and handed off to QA.
The same run exposed a sharper review-quality problem. The target brief
required `--text ""` to return zero counts, but the implementation treated an
empty string as if `--text` was missing. QA observed that runtime probe fail:

```text
<validation-root> --text ""
stderr: error: --text flag is required
exit_code: 1
```

The generated test suite nevertheless encoded empty text as an expected error,
so `go test -v` passed. Because AD-169 allowed non-zero runtime probes to avoid
poisoning expected negative-path approval, QA approved the ticket even though
the failing probe contradicted the product brief and the done ticket claimed
empty text was complete.

AD-169's distinction was directionally right: expected invalid-input probes
should not force false rework. The missing piece was a durable way for a role
to state that a non-zero exit was intentionally expected, so policy could block
all other validation failures.

### Decision

`shell_exec` now accepts an optional `expected_exit_code` field. Reviewers use
it only when intentionally proving an error path or invalid-input rejection.
Session outcome tracking records a matching non-zero runtime command as
expected negative-path evidence instead of a validation failure.

QA and Security approval is now blocked after any unexpected failing validation
command in the same job, even when tests pass. Expected non-zero runtime probes
remain acceptable only when they set `expected_exit_code`, match that code, and
are paired with passing tests or positive validation evidence where relevant.

External temp validation binaries such as `<validation-root>` may also
be removed through `shell_exec` cleanup after ticket completion without forcing
Engineer to reopen a finished ticket. That cleanup is outside the target repo
and should not be treated as product rework.

### Consequences

- QA cannot approve a ticket after an observed runtime failure unless that
  failure was explicitly marked and matched as an expected error-path probe.
- Tests that accidentally encode the wrong acceptance criterion no longer mask
  contradictory runtime evidence.
- Expected missing-argument, permission-denied, or invalid-input probes still
  work, but they must be explicit in the tool call.
- Engineer post-completion cleanup of `<validation-root>` binaries no longer
  creates rework-loop pressure.
- The next CLI canary should confirm QA requests Engineer rework for the
  empty-text bug, Engineer reopens `T-001`, fixes the implementation/test
  mismatch, and QA then approves with the missing-argument probe marked by
  `expected_exit_code`.

## AD-172: Review Validation Failures Exit Through Structured Handoff

**Status:** Accepted
**Date:** 2026-05-21
**Owner:** Mars Harness maintainers

### Context

The `demo-cli-run17` replay validated the core AD-171 behavior. QA no longer
approved contradictory runtime evidence, and the target implementation was
eventually corrected so `--text ""` returned zero counts:

```text
<validation-root> --text ""
{"word_count":0,"character_count":0,"line_count":0}
```

The run still exposed two lifecycle weaknesses. First, QA ran the
authoritative `go test ./...` command on its final model turn; the tests failed
on a multi-line counting expectation, but the role reached `max_turns` before
recording `changes_requested`. Second, Engineer built and reused `<validation-root>`
instead of the documented `<validation-root>` path, so same-session
freshness tracking did not protect that executable name.

Both failures are generic factory issues, not Note Stats-specific behavior.
Review roles must turn failed validation into a structured handoff immediately,
and external runnable validation artifacts must use the tracked naming
convention so stale binaries cannot masquerade as current evidence.

### Decision

QA and Security now stop shell validation after any current-job failing build,
test, or unexpected runtime validation command. Further `shell_exec` calls are
blocked with a direct instruction to record `job_disposition_record` using
`status: changes_requested`, `next_need: implementation_rework`,
`feedback.for_role: engineer`, and the exact failing command/output.

Dispatch-mode agent loops now grant one final terminal-tool reminder at the
model turn-budget boundary. If a role has reached the turn budget without the
required terminal tool, the loop appends one last instruction to call
`job_disposition_record` and forbids further inspection or validation. This is
not extra work budget; it is a structured-exit opportunity so findings do not
disappear into raw `max_turns` telemetry.

Go validation builds outside the target repo must use tracked
`/tmp/<project>-validation` style paths. `go build -o /tmp/<project>` is now
blocked before execution and redirected to `/tmp/<project>-validation`, aligning
all external validation binaries with the freshness guard.

Generated QA/Security guidance now tells reviewers to run authoritative tests
early once the implementation files are known, stop immediately on failure, and
record the structured handoff instead of spending more turns on additional
shell validation.

### Consequences

- QA failures should become Engineer rework handoffs rather than `max_turns`
  dead ends.
- Review approval remains strict: failing builds/tests and unexpected runtime
  failures cannot be approved away.
- Expected negative-path runtime probes still work when the role uses
  `expected_exit_code`.
- External temp binaries become uniformly freshness-tracked through the
  `-validation` suffix.
- The next CLI canary should confirm QA either approves after passing tests or
  records `changes_requested` immediately after the first failing validation,
  with no target intervention-debt ticket materialization.

## AD-173: Expected Negative Probes Can Be Corrected Once

**Status:** Accepted
**Date:** 2026-05-21
**Owner:** Mars Harness maintainers

### Context

The `demo-cli-run18` replay confirmed AD-172 changed the live experience:
QA found a product-validation failure, received one terminal-tool grace turn,
and recorded `changes_requested` instead of ending as raw `max_turns`.
Orchestrator routed the rework back to Engineer, and Engineer reopened the done
ticket before product mutation.

The next weakness was the review procedure around expected negative probes.
QA intentionally checked the missing-argument path, but first ran
`<validation-root>` without `expected_exit_code`. The command correctly
exited non-zero, but policy classified it as an unexpected runtime failure.
When QA tried to approve, approval was blocked; when QA tried to rerun the same
command with `expected_exit_code: 1`, the AD-172 shell-stop rule blocked the
correction and forced another rework loop.

That is too brittle for generic product validation. A reviewer should use
`expected_exit_code` on the first attempt, but the harness should allow one
immediate exact-command correction when the only failure is an expected-negative
runtime probe.

### Decision

QA and Security still stop shell validation after failing builds, failing tests,
or uncorrected unexpected runtime failures. Build and test failures remain
terminal for the review job and require `changes_requested`.

For runtime validation only, the tool session now records the exact command and
exit code for an unexpected non-zero validation result. If the reviewer
immediately reruns that same command with a matching non-zero
`expected_exit_code`, policy allows the rerun once, records it as expected
negative-path evidence, and clears the unexpected runtime-failure blocker for
approval. Other shell commands remain blocked until disposition.

Generated QA and Security guidance now says to set `expected_exit_code` on the
first attempt for invalid-input/error-path probes. If the reviewer forgets, the
only allowed recovery is to rerun the exact same command once with the matching
expected code before any other shell validation.

### Consequences

- Reviewers can correct a validation-procedure mistake without turning a safe
  negative-path check into Engineer rework.
- Approval remains strict for real failures: builds, tests, different runtime
  commands, and uncorrected unexpected runtime probes still block approval.
- The next CLI canary should confirm missing-argument checks can be corrected
  with `expected_exit_code` while true product failures still route back to
  Engineer.

## AD-174: Product Completion Requires Repaired Runtime Validation

**Status:** Accepted
**Date:** 2026-05-21
**Owner:** Mars Harness maintainers

### Context

The `demo-cli-run19` replay confirmed the factory still makes product-first
progress on a non-game target. CEO, COO, and CTO created product-specific
goals, an active plan, a feature contract, and `T-001` for a Note Stats CLI.
Engineer claimed the ticket, implemented code, committed the implementation,
updated ticket evidence, moved `T-001` to done, and committed the lifecycle
move with no intervention-debt tickets in the target backlog.

The failure was evidence integrity. Engineer had already observed
`<validation-root> --text ""` fail with
`error: --text flag is required`, even though the brief and ticket required
empty text to return zero counts. The role then marked the acceptance criteria
as complete, moved the ticket to `docs/tickets/done/`, and used the final
turn-budget grace on another lifecycle command instead of the required
`job_disposition_record`. The job ended as `max_turns`, and a manual replay of
the committed target confirmed the product behavior was still wrong.

This is generic: the harness cannot treat ticket metadata, lifecycle moves, or
commits as completion while the same role session still has an unresolved
runtime validation failure for the ticket behavior.

### Decision

`shell_exec` session tracking now records outstanding unexpected runtime
validation failures by exact command fingerprint. A later successful run of
the same runtime command repairs the outstanding failure. For review roles, an
intentional negative-path probe repairs it only when the exact command is
rerun with a matching non-zero `expected_exit_code`; AD-175 keeps that
retroactive correction out of Engineer implementation jobs.

Engineer cannot complete the product lifecycle while such a failure is
outstanding. Tool policy blocks:

- moving an in-progress product ticket into `docs/tickets/done/`;
- committing a staged in-progress-to-done ticket move;
- writing a product ticket directly under `docs/tickets/done/`;
- recording a successful `job_disposition_record`.

The error tells Engineer to fix the behavior and rerun the exact failing
command successfully, or to rerun the exact intentional negative-path command
with `expected_exit_code`, before updating completion evidence or requesting
QA review.

The agent loop also tightens the terminal-tool grace turn. Once the turn-budget
edge reminder has been issued, the next model response may spend the grace only
on the configured terminal tool. If the model calls any other tool, the loop
ends as `max_turns` without executing that extra side-effecting cleanup.

### Consequences

- Product tickets should not move to done while current-job runtime evidence
  still contradicts the BDD contract.
- Expected negative-path validation remains available, but it must be declared
  with `expected_exit_code` to clear the runtime-failure blocker.
- The terminal grace turn becomes a true structured-exit path instead of an
  extra mutation turn.
- The next clean CLI canary should confirm Engineer either repairs failing
  acceptance behavior before ticket completion or is blocked before the done
  move, with the target backlog still free of foundation intervention debt.

## AD-175: Engineer Cannot Reclassify Failed Acceptance As Expected

**Status:** Accepted
**Date:** 2026-05-21
**Owner:** Mars Harness maintainers

### Context

The `demo-cli-run20` replay confirmed AD-174 blocked the bad lifecycle move.
Engineer observed `<validation-root> --text ""` fail, updated the
ticket evidence, and attempted to move `T-001` to `docs/tickets/done/`. Policy
blocked the move because the current job still had an unresolved runtime
validation failure.

The role then tried to rerun the same failing acceptance command with
`expected_exit_code: 1`. That is valid for a reviewer who accidentally forgot
`expected_exit_code` on a known negative-path probe, but it is unsafe for
Engineer after a positive acceptance command has failed. In an implementation
job, a failed acceptance path should be repaired and rerun successfully, not
reclassified as expected failure after the fact.

### Decision

Only QA and Security may use the one-time exact-command
`expected_exit_code` correction for an unexpected runtime validation failure.
Engineer may still use `expected_exit_code` when an intentional error-path
probe is declared on the first run, but once Engineer has observed an
unexpected runtime failure, the outstanding blocker is cleared only by a later
successful run of that exact command.

### Consequences

- Engineer cannot bypass a failed positive acceptance path by retroactively
  adding `expected_exit_code`.
- QA/Security keep the AD-173 correction path for review-procedure mistakes.
- The next clean CLI canary should confirm a failed empty-text acceptance check
  either gets fixed and rerun successfully or blocks ticket completion without
  an expected-exit loop.

## AD-176: Failed Engineer Runtime Validation Requires Rework Before Rerun

**Status:** Accepted
**Date:** 2026-05-21
**Owner:** Mars Harness maintainers

### Context

The `demo-cli-run21` replay confirmed the expected-exit bypass was closed.
Engineer implemented a product-specific Note Stats CLI ticket, ran
`go run ./cmd/note-stats --text "hello world"` successfully, then observed the
required empty-text acceptance probe fail:

```text
go run ./cmd/note-stats --text ""
Error: --text flag is required
```

The harness did not move the ticket to done, did not create target
intervention-debt, and did not dispatch the failure into an autonomous
containment loop. The remaining issue was convergence: Engineer reran runtime
probes, including the same failed command, until `circle_detected` stopped the
job. That is still product-first, but it spends turns proving the same failure
instead of editing the implementation.

### Decision

When Engineer has an outstanding unexpected runtime validation failure,
`shell_exec` now blocks further runtime probes until the implementation has
actually been edited. The policy distinguishes three cases:

- rerunning the same failed runtime command before any post-failure
  `file_write` is blocked with explicit inspect/edit guidance;
- running a different runtime probe while any exact failure is still
  outstanding is blocked until the original failed command is repaired;
- adding `expected_exit_code` after an Engineer runtime failure is blocked
  immediately, preserving AD-175.

After a post-failure edit, Engineer may rerun the exact failed command. A
successful exit clears the runtime-failure blocker and allows normal ticket
evidence and lifecycle completion to continue.

### Consequences

- The live loop should move from "observe failed acceptance, repeat failed
  acceptance" to "observe failed acceptance, edit implementation, rerun exact
  acceptance."
- Negative-path probes remain available when Engineer declares
  `expected_exit_code` on the first intentional run, before any unexpected
  runtime failure exists.
- The next clean CLI canary should confirm the Engineer either edits and fixes
  `--text ""` or receives policy guidance to edit before runtime probes repeat.

## AD-177: Engineer Can Correct Missing-Argument Negative Probes

**Status:** Accepted
**Date:** 2026-05-21
**Owner:** Mars Harness maintainers

### Context

The `demo-cli-run22` replay confirmed AD-176 changed the live behavior.
Engineer repeated the failed empty-text acceptance command once, policy blocked
the unchanged rerun, and Engineer edited the implementation. The exact
`<validation-root> --text ""` command then exited successfully.

The next failure was a boundary between positive acceptance and expected
negative-path validation. Engineer ran `<validation-root>` without
`expected_exit_code` to prove the missing-argument path. The command correctly
failed because the product brief says missing `--text` should fail, but policy
treated it as an unresolved unexpected runtime failure and blocked later runtime
probes. Engineer then looped on unrelated newline probes because the policy
guidance did not leave a valid correction path for the missing-argument check.

### Decision

Engineer may use the one-time exact-command `expected_exit_code` correction
only for obvious missing-argument runtime probes: a validation binary or
language run command invoked without application arguments. This keeps the
generic implementation workflow practical when the role forgets to mark a
negative-path probe up front.

The correction remains blocked for positive acceptance paths with supplied
application input, including `--text ""`. Those failures still require
implementation rework and a later successful exact command.

### Consequences

- Missing-required-input validation can be repaired procedurally without
  forcing fake product behavior or blocking the rest of the acceptance suite.
- Failed positive acceptance remains strict: Engineer cannot turn it into an
  expected failure after observing the failure.
- The next clean CLI canary should confirm Engineer uses
  `expected_exit_code: 1` for `<validation-root>` after an omitted
  expected marker, then continues to evidence/ticket completion.

## AD-178: Runtime Error Stderr Is Failed Evidence

**Status:** Accepted
**Date:** 2026-05-21
**Owner:** Mars Harness maintainers

### Context

The `demo-cli-run23` replay confirmed AD-177 allowed Engineer to recover from
an omitted missing-argument `expected_exit_code`, but exposed a broader runtime
evidence gap. The target validation binary exited zero for:

```text
<validation-root> --text ""
```

while printing `error: --text flag is required` and usage text to stderr. The
harness counted the probe as successful runtime evidence because the exit code
was zero, so Engineer could move toward completion even though the user-visible
CLI behavior contradicted the product brief.

### Decision

Direct runtime validation commands now treat conservative error-shaped stderr
as failed evidence even when the process exits zero. The markers are intentionally
simple and product-agnostic: `error:`, `Usage of`, `panic:`, `Traceback`, and
`exception`.

This applies only to direct runtime probes and same-session validation artifact
executions, not to build or test commands. A later exact rerun with exit code
zero and no error-shaped stderr repairs the blocker.

### Consequences

- Runtime evidence follows the visible product behavior, not only process
  status.
- CLIs, scripts, and small generated apps that print errors but forget to exit
  non-zero no longer close tickets with contradictory evidence.
- The next clean CLI canary should confirm the zero-exit `--text ""` stderr
  error blocks further runtime probes until Engineer edits the implementation
  and the exact command later passes cleanly.

## AD-179: Claimed Engineer No-Op Loops Route To Implementation

**Status:** Accepted
**Date:** 2026-05-21
**Owner:** Mars Harness maintainers

### Context

The `demo-cli-run24` replay confirmed the foundation lifecycle reached
product-specific planning, a product feature contract, and a committed
implementation ticket without intervention-debt amplification. Engineer claimed
`T-001`, committed the claim, then called empty `shell_exec` placeholders before
writing any product code. The generic no-op message primarily described
post-validation completion, so Engineer repeated no-op calls and ended as
`circle_detected` without touching the implementation.

### Decision

After Engineer has an in-progress product ticket and has already received a
no-op `shell_exec` failure, a repeated no-op before successful validation is a
policy boundary. The next action must be implementation-oriented: read the
in-progress ticket and linked feature contract, then use `file_write` to create
or update product files, or record `job_disposition_record` with `status:
blocked` if the ticket cannot be implemented.

The post-validation no-op path remains separate: after successful validation and
dirty implementation/ticket work, repeated no-ops still route to commit,
evidence update, lifecycle move, and QA handoff.

### Consequences

- A claimed ticket cannot stall in shell placeholders before the first product
  edit.
- No-op recovery guidance is phase-aware: pre-implementation points at
  `file_read`/`file_write`; post-validation points at evidence and lifecycle
  completion.
- The next clean CLI canary should confirm Engineer turns the repeated no-op
  block into product file edits instead of another `circle_detected` job.

## AD-180: Runtime Failure Guidance Names Missing-Argument Correction

**Status:** Accepted
**Date:** 2026-05-21
**Owner:** Mars Harness maintainers

### Context

The `demo-cli-run25` replay confirmed AD-179 changed live behavior: Engineer
claimed `T-001`, read the ticket and feature contract, wrote product files, and
ran direct runtime probes instead of looping on no-op shell calls. It then hit a
known negative-path edge. Engineer ran the missing-required-input probe without
`expected_exit_code`, later fixed the positive empty-text path, but kept trying
other runtime probes and completion because the unresolved-runtime blocker did
not explicitly name the allowed correction.

### Decision

Unresolved Engineer runtime-failure policy messages now distinguish positive
acceptance repair from procedural negative-path correction. For an intentional
no-argument or missing-required-input probe, the policy tells Engineer to rerun
the exact earlier command once with `expected_exit_code`, usually `1`. For
positive acceptance failures, Engineer must still edit implementation and make
the exact command pass without `expected_exit_code`.

### Consequences

- The policy already allowed missing-argument correction; the live role now gets
  direct, actionable wording when it is blocked from other probes or completion.
- The stricter positive-acceptance path remains intact.
- The next clean CLI canary should confirm Engineer uses `expected_exit_code`
  for the missing-argument probe and proceeds without repeated completion
  blocks.

## AD-181: Failed Ticket Creation Cannot Become Completed Progress

**Status:** Accepted
**Date:** 2026-05-21
**Owner:** Mars Harness maintainers

### Context

The `demo-cli-run26` replay confirmed AD-180 did not regress the product-first
bootstrap path: CEO recovered from a feature-contract role-boundary block, COO
updated the active plan and canonical feature contract, and CTO reached product
ticket creation without target intervention-debt churn. The next failure was
ticket-materialization integrity. CTO called `ticket_create` with
`bdd_scenarios` encoded as a quoted JSON string, repeated the same malformed
tool call, attempted to hand-write a ticket with `file_write`, then recorded a
successful disposition claiming an implementation ticket existed even though no
ticket had been created.

### Decision

Ticket-creation failures are now tracked as unresolved session state. A failed
`ticket_create`, or a failed `file_write` attempt under `docs/tickets/`, blocks
successful `job_disposition_record` statuses until a later `ticket_create`
succeeds in the same job. Roles can still finish honestly with `status:
blocked`, `failed`, or `changes_requested` and the exact ticket-creation error
as the blocker.

`ticket_create` parse errors now include field-specific repair guidance for
common local-model array drift. If `bdd_scenarios`, `blocked_by`,
`depends_on`, or `evidence_links` is supplied as a quoted list string, the
error names the field and shows the correct JSON array shape.

### Consequences

- CTO and other planning roles cannot claim product progress after ticket
  materialization failed.
- Failed attempts to bypass `ticket_create` by writing ticket markdown directly
  remain guardrail telemetry and also prevent false successful handoff.
- The next clean CLI canary should confirm CTO repairs malformed
  `ticket_create` arguments or records a blocked disposition, rather than
  looping through CTO or handing implementation to Engineer without a backlog
  ticket.

## AD-182: Missing-Argument Runtime Corrections Are Exact And Blocking

**Status:** Accepted
**Date:** 2026-05-21
**Owner:** Mars Harness maintainers

### Context

The `demo-cli-run27` replay confirmed AD-181 changed the live lifecycle:
CTO created a real `T-001` product ticket with valid `bdd_scenarios` array
metadata and Engineer began ticket-backed implementation. The remaining
runtime edge was convergence after a procedural validation mistake. Engineer
ran the missing-required-input validation binary without `expected_exit_code`,
then continued with edits, decisions, commits, and completion attempts before
performing the exact expected-negative correction.

The earlier AD-180 wording told the role what kind of correction was allowed,
but the session did not carry the exact failed command and correction forward
as a hard next action. That let a local model keep exploring after the harness
already knew the correct immediate recovery step.

### Decision

Unexpected runtime validation failures now store the exact unresolved
`shell_exec` command in session state. When the failed command is an obvious
no-argument or missing-required-input probe, the session also stores the exact
correction, including `expected_exit_code: 1`.

While that missing-argument correction is outstanding, Engineer cannot continue
unrelated mutating work such as `file_write`, `git_commit`, `git_push`,
`record_decision`, dependency sync, CLI mutation, tool creation, or persona
creation. The allowed forward paths are:

1. Run the exact stored `shell_exec` correction with matching
   `expected_exit_code`.
2. Record `job_disposition_record` with `status: blocked` and the exact blocker
   if that correction is invalid.

The existing positive-acceptance rule remains strict: a failed acceptance probe
with supplied input is repaired by editing implementation and making the exact
command pass cleanly, not by retroactively adding `expected_exit_code`.

### Consequences

- The harness converts a known procedural negative-path mistake into one clear
  next action instead of letting the role spend turns on adjacent work.
- Missing-argument correction guidance now includes the exact command shape,
  reducing local-model formatting drift.
- Product implementation remains protected from false completion, while
  positive behavior failures still force real code repair.
- The next clean CLI canary should confirm Engineer either runs the exact
  expected-exit correction promptly or records a blocked disposition before
  continuing unrelated mutations.

## AD-183: External Validation Artifacts Must Be Rebuilt After Runtime Edits

**Status:** Accepted
**Date:** 2026-05-21
**Owner:** Mars Harness maintainers

### Context

The `demo-cli-run28` replay confirmed the AD-181 ticket materialization path:
CTO created a valid product ticket and Engineer claimed it. The run did not
reach the AD-182 missing-argument correction because an earlier positive
acceptance path failed first. Engineer built `<validation-root>`,
proved the happy path, then observed `<validation-root> --text ""`
fail even though the brief says empty text should return zero counts. Policy
correctly blocked unchanged reruns and required an implementation edit.

After editing `main.go`, Engineer reran the same `<validation-root>` binary
without rebuilding it. The stale binary still represented the old source, so
the role looped on the same failure and no-op placeholders until
`circle_detected`.

### Decision

The session now records the runtime-edit counter at the moment an external
validation artifact is built. If a runtime validation failure is followed by an
implementation edit, any later execution of a previously built
`<validation-root>` artifact is blocked until the role rebuilds that artifact
with `go build -o <validation-root> ...`.

This is a freshness rule, not a product-specific assertion. It applies to
external validation binaries because they are snapshots of source at build
time. Direct source commands such as `go run` still validate the current source
without a rebuild step.

### Consequences

- Positive acceptance repair cannot be tested against stale binaries.
- Engineer receives an exact rebuild command shape before rerunning runtime
  validation.
- The existing same-session artifact trust rule now covers both first use and
  post-edit freshness.
- The next clean CLI canary should confirm that after an empty-text failure and
  source edit, Engineer rebuilds `<validation-root>` before rerunning the
  acceptance probe.

## AD-184: Ticket Evidence Follows Successful In-Job Validation

**Status:** Accepted
**Date:** 2026-05-21
**Owner:** Mars Harness maintainers

### Context

The `demo-cli-run29` replay confirmed the fresh target lifecycle still reaches
product-specific planning, feature-contract update, ticket creation, and
Engineer implementation work without creating target intervention-debt tickets.
Engineer claimed `T-001`, corrected a missing `MarsDocSync` source write, and
started building the Note Stats CLI.

The run then exposed a smaller evidence-integrity gap. Before any successful
validation command in the Engineer job, the role updated the in-progress ticket
with `evidence_links` and `verified_by`. That metadata made the ticket look
closer to completion than the actual validation transcript supported, while the
job later stalled in repeated no-op placeholders before proving the product
behavior.

### Decision

Engineer cannot populate `evidence_links` or `verified_by` in a ticket under
`docs/tickets/in-progress/` until the same job has recorded at least one
successful validation command. The validation may be a test command, build
command, direct runtime probe, or other shell validation outcome that the
session records as successful.

Empty or placeholder evidence fields remain writable, and non-Engineer roles
retain their existing review and planning boundaries. The rule is intentionally
attached to in-progress Engineer delivery because that is where ticket evidence
can otherwise outrun implementation proof.

### Consequences

- Ticket metadata cannot claim evidence before the current Engineer job has
  actually produced validation evidence.
- The role receives direct guidance to run `go test`, a build, or a runtime
  command that exercises the BDD scenario before updating ticket evidence.
- Completion and ticket-lifecycle gates now have a stronger upstream invariant:
  evidence fields are populated only after at least one successful validation
  signal exists in the same job.
- The next clean CLI canary should confirm Engineer responds by running
  validation before evidence updates instead of repeatedly writing ticket
  metadata or no-op shell placeholders.

## AD-185: Review Validation Artifact Rebuilds Need Exact Corrections

**Status:** Accepted
**Date:** 2026-05-21
**Owner:** Mars Harness maintainers

### Context

The `demo-cli-run30` replay confirmed AD-184. Engineer no longer populated
in-progress ticket evidence before validation. Instead it repaired a failing
test build, built `<validation-root>`, proved the empty-text and
hello-world CLI paths, ran `go test ./...`, updated ticket evidence, moved
`T-001` to done, committed the lifecycle move, and handed off to QA.

QA then hit the same-session artifact freshness guard by trying to execute the
Engineer-built `<validation-root>` binary. The guard correctly blocked
stale cross-session evidence and kept the event as foundation telemetry, but
the role did not quickly recover by rebuilding the binary in its own session.
The blocker text said to rerun the matching build, but did not name the exact
`shell_exec argv` correction.

### Decision

External validation artifact freshness errors now include an exact shell tool
correction. For a root Go CLI, the error names:

```json
["go","build","-o","/tmp/<project>-validation","."]
```

The same helper is used when the binary was never built in the current role
session and when the binary became stale after a post-failure source edit. The
helper prefers the root Go package when `go.mod` and `main.go` are present,
falls back to the first `cmd/*/main.go` package when available, and otherwise
uses `.` as the safest general Go build target.

Generated QA/Security guidance and the persona manuals now say to run the
exact `shell_exec argv` correction from the tool error before rerunning a
`<validation-root>` binary.

### Consequences

- Reviewer recovery after a stale external artifact guard has one explicit next
  command instead of an inferred build target.
- The same-session freshness invariant stays intact: QA and Security cannot
  trust Engineer-built temp binaries from an earlier role session.
- The next clean CLI canary should confirm QA rebuilds `<validation-root>` in
  its own job, reruns the runtime probe, and records a structured disposition
  without a long quiet stall.

## AD-186: Build Guard Corrections Preserve Package Targets

**Status:** Accepted
**Date:** 2026-05-21
**Owner:** Mars Harness maintainers

### Context

The `demo-cli-run31` replay confirmed the product-first lifecycle again. CEO
accepted the Note Stats CLI brief, COO updated the active plan and BDD
contract, CTO created a product ticket, and Engineer claimed, implemented,
validated, evidenced, and moved `T-001` to done without creating target
intervention-debt tickets.

The run then exposed two narrower validation-quality gaps. First, Engineer
treated exit-zero runtime probes as enough evidence even though the explicit
empty-text contract expected `{"words":0,"lines":0,"characters":0}` and the
implementation returned `{"words":0,"lines":1,"characters":0}`. Second, QA
correctly blocked `go build ./cmd/note-stats` because it would create a
repo-local binary, but the recovery message did not name an exact corrected
argv. QA guessed `go build -o <validation-root> .`, which failed for
the `cmd/<name>` layout and forced changes-requested before the intended
review could proceed.

### Decision

Go build-output guardrails now include an exact `shell_exec argv` correction
that preserves the original build package or entrypoint while redirecting the
output to `<validation-root>`. For example:

```json
["go","build","-o","<validation-root>","./cmd/note-stats"]
```

The same correction formatter is used for implicit `go build` output,
repo-local `-o` output, and external temp outputs that lack the tracked
`-validation` suffix. It preserves package patterns such as `./...` and
`./cmd/<name>` instead of normalizing them into invalid path tokens.

The operating model also tightens expected-output evidence. When README,
tickets, or BDD contracts name exact CLI output, API response bodies,
UI-visible state, or persisted data, Engineer should add automated assertions
for those examples and QA should request changes when only exit-code smoke
evidence exists. For Go product code, QA approval is mechanically blocked when
non-test `.go` source files exist but no `_test.go` files are present.

### Consequences

- Reviewers can recover from repo-local Go build guards without guessing the
  package path for root or `cmd/<name>` layouts.
- `go build ./...`, `go build ./cmd/<name>`, and `go build -o <bad-path> ...`
  errors now provide directly runnable `shell_exec argv` guidance.
- Exact expected-output examples become test obligations, reducing false
  confidence from command success alone.
- QA cannot approve Go source changes with no Go test files, so smoke-only
  product delivery routes back to Engineer before Security or release.
- The next clean CLI canary should confirm QA follows the exact build
  correction, and Engineer produces tests that catch the empty-text line-count
  mismatch before moving a ticket to done.

## AD-187: Missing-Input Repro Guards Must Unlock Repair Edits

**Status:** Accepted
**Date:** 2026-05-21
**Owner:** Mars Harness maintainers

### Context

The `demo-cli-run32` replay confirmed that the product-first lifecycle now
reaches a real product ticket with Engineer tests before completion. It also
showed a deterministic containment loop. Engineer ran a runtime probe for a
missing or empty input path, the validation binary panicked, and the harness
correctly required an exact `expected_exit_code` rerun before any unrelated
probe. That rerun still showed the implementation defect, but the same guard
continued to block `file_write` and told Engineer to rerun the same failing
probe again. The result was a runtime-policy loop instead of product repair.

The same run exposed a deployed/foundation naming leak: CTO wrote the target
ticket with `cmd/mars-harness/main.go` and Engineer initialized `module
mars-harness` inside a Note Stats CLI target. That was product progress, but
the target implementation shape was still borrowing foundation source names.

### Decision

Missing-input runtime failures now distinguish correction proof from repair
work. The first unexpected no-argument or missing-required-input runtime
failure still blocks mutations until Engineer reruns the exact command with
`expected_exit_code`. If that correction attempt succeeds, the unexpected
failure is cleared. If the correction attempt still fails, the harness records
that the expected-exit repro was attempted and allows implementation
`file_write` so Engineer can fix the product. Completion, commits, ticket
done-moves, and unrelated runtime probes remain blocked until validation is
repaired.

Generated CTO and Engineer guidance now also keeps foundation names out of
deployed targets. CTO tickets must derive affected paths, module names,
command names, and binary names from the target README, repo basename, remote,
or existing local conventions. Engineer must not initialize a fresh target as
`module mars-harness` or create `cmd/mars-harness` unless the target product is
explicitly Mars Harness itself; small fresh Go targets should prefer standard
library tests unless the repo already uses or requires another dependency.

### Consequences

- Missing-input guards still prevent accidental retroactive `expected_exit_code`
  laundering for positive acceptance failures.
- A failed missing-input correction attempt becomes evidence that product code
  needs repair, not a reason to loop on the same command.
- Engineer can edit after proving the negative-path repro, but cannot finish
  the ticket until the exact failing runtime path is repaired and revalidated.
- Fresh deployed targets are less likely to inherit foundation package names,
  reducing generic-software-factory drift toward Mars-specific scaffolds.
- The next clean CLI canary should confirm Engineer fixes a bad missing-input
  implementation after the expected-exit repro instead of looping, and that
  CTO/Engineer derive command/module names from the target product.

## AD-188: Exact Runtime Repairs Clear Repeated Failure Counters

**Status:** Accepted
**Date:** 2026-05-21
**Owner:** Mars Harness maintainers

### Context

The `demo-cli-run33` replay confirmed the AD-187 target-naming fix: CTO created
`T-001` with `cmd/note-stats/main.go`, and Engineer initialized `module
note-stats` instead of leaking foundation `mars-harness` names. It also
confirmed the missing-input repair path: after the `--text ""` positive
acceptance path failed, Engineer was allowed to edit and rebuild.

The run then exposed a counter-level loop. The same exact runtime command
failed more than once while Engineer iterated. When the command later exited
zero, session repair accounting only cleared one outstanding failure count.
The harness still considered an earlier identical failure unresolved and
blocked unrelated probes, causing `circle_detected` even though the exact
command had been rerun successfully.

### Decision

Runtime repair accounting now treats one successful exact rerun as repairing
all outstanding failures for that exact command fingerprint in the current
job. Expected-exit corrections for missing-input probes use the same rule: if
the exact correction succeeds, it clears every unmatched failure for that same
command and expected exit code. The global outstanding-runtime-failure counter
is decremented by the number of repaired matching failures without going below
zero.

### Consequences

- Repeated attempts at the same exact failing command no longer create stale
  outstanding blockers after the command finally succeeds.
- The harness still blocks different runtime probes until the exact failed
  command is repaired.
- Product semantic mismatches may still need QA or automated assertions, but
  the runtime policy no longer traps the role after an exact command becomes
  process-successful.
- The next clean CLI canary should confirm an exact runtime success clears
  repeated same-command failures and lets Engineer continue to tests, evidence,
  or QA handoff.

## AD-189: Review Validation And Ticket Closure Must Preserve Product Progress

**Status:** Accepted
**Date:** 2026-05-21
**Owner:** Mars Harness maintainers

### Context

The `demo-cli-run34` replay confirmed AD-188: Engineer repaired the `--text ""`
positive acceptance failure, reran that exact command successfully, corrected
the omitted-flag negative path with `expected_exit_code: 1`, wrote Go tests,
and reached QA instead of ending in stale `circle_detected`. It also exposed
the next quality and traceability faults.

QA first used shallow `grep` globs that missed `cmd/note-stats-cli/main.go`,
then ran `go mod init` even though the module already existed. Later, QA ran
the intentional no-argument error-path probe without `expected_exit_code`; the
command correctly exited non-zero, but review policy counted it as an
unexpected validation failure and forced `changes_requested`. Orchestrator then
read sample player-movement prose from `docs/tickets/README.md` as if it were
live ticket state. Finally, Engineer had committed product code, tests, and
`go.mod` inside the `chore(tickets): move T-001 to done` lifecycle commit,
weakening traceability even though the product behavior was correct.

### Decision

Reviewer shell execution is now mechanically validation-only. QA and Security
may use shell for read-only inspection, tests, builds, fresh external
`<validation-root>` binaries, runtime probes, and HTTP probes. They may not use
review shell access for package or module initialization such as `go mod init`,
product mutation, broad discovery, cleanup, or placeholder no-op commands.
Generated QA guidance also says intentional negative-path probes should set
`expected_exit_code` on the first run.

Ticket lifecycle moves to `docs/tickets/done/` now require non-ticket product
changes to be committed first. Dirty ticket evidence can move with the
lifecycle commit, but dirty source, tests, docs, package manifests, lockfiles,
config, or validation code block the done move until an implementation/test
commit exists.

Generated Orchestrator guidance now treats `docs/tickets/README.md` as
conventions and examples only. Live routing must use lifecycle directory state,
the source disposition ticket ID, and structured handoff evidence rather than
sample `T-001` prose from the README.

### Consequences

- QA/Security can no longer turn a review into target setup or mutate the repo
  while trying to validate it.
- Expected negative-path probes should not create false rework when the product
  already behaves correctly.
- Product and test commits become visible to downstream reviewers before the
  ticket lifecycle close commit.
- Orchestrator is less likely to route rework from example docs instead of the
  actual completed ticket.
- The next clean CLI canary should confirm Engineer creates a separate
  implementation/test commit before the done move, QA uses
  `expected_exit_code` for the no-argument path on the first run, and
  Orchestrator routes from the real `source_disposition` rather than
  `docs/tickets/README.md` examples.

## AD-190: Review No-Op Loops Get A Terminal Disposition Off-Ramp

**Status:** Accepted
**Date:** 2026-05-21
**Owner:** Mars Harness maintainers

### Context

The `demo-cli-run35` replay confirmed the AD-189 ticket-closure boundary:
Engineer made a separate `feat(cli)` implementation commit for product source,
README, and `go.mod`, then moved `T-001` to done in a lifecycle-only commit.
QA then performed useful validation: it inspected the done ticket and source,
ran `docsync_audit`, built a fresh `<validation-root>` binary, and
validated the happy path and empty-string path.

The review still failed as a harness loop. After successful validation, QA
called `shell_exec` with empty `argv` as a placeholder several times instead of
recording the required `job_disposition_record`. Tool policy correctly blocked
the placeholders and foundation telemetry quarantined the `circle_detected`
failure without creating target intervention debt or dispatching Orchestrator.
The product state was good, but review completion was lost to a protocol loop.

### Decision

Server jobs that require a terminal tool now get one circle-grace turn. If the
model repeats the same tool-call shape enough to trigger circle detection and a
required terminal tool such as `job_disposition_record` is configured, the loop
adds one corrective user message requiring only that terminal tool. If the next
response calls a different tool, the job still ends with `circle_detected`.

Reviewer no-op policy now distinguishes review phases. QA and Security no-op
placeholders after successful validation receive direct disposition guidance:
stop shell validation and record `job_disposition_record` with `status:
approved` or the appropriate quality decision. No-op placeholders after failed
validation route to structured `changes_requested`. Policy-blocked no-op shell
calls are also counted as no-op failures in the session so telemetry and loop
guards can identify review dithering even when the process was never executed.

Generated QA guidance and canonical persona docs now say that after the
required build, test, runtime, and docsync evidence has passed, the next action
is `job_disposition_record`; reviewers must not call empty `argv`, `:`, or wait
commands.

### Consequences

- A reviewer that has already gathered enough passing evidence gets one hard
  off-ramp to the required disposition before the job is declared circular.
- The harness still refuses to auto-approve; the model must call the terminal
  tool with explicit evidence and status.
- Review no-op loops remain foundation-owned telemetry and do not create
  target intervention-debt tickets.
- The next clean CLI canary should confirm QA records an approved disposition
  after validation instead of ending in `circle_detected` from repeated empty
  shell calls.

## AD-191: Unresolved Runtime Failures Freeze Commits And Shell Side Paths

**Status:** Accepted
**Date:** 2026-05-21
**Owner:** Mars Harness maintainers

### Context

The `demo-cli-run36` replay confirmed that product-first bootstrap and
intervention-debt quarantine still hold. CEO, COO, and CTO produced the Note
Stats CLI plan, feature contract, and product ticket. Engineer claimed the
ticket, wrote product source, and ran a validation binary. The happy path
passed, but the contracted empty-string acceptance probe failed with
`Error: --text flag is required`.

The harness correctly blocked ticket completion, successful disposition, and
moving the ticket to `docs/tickets/done/` while that acceptance failure was
unresolved. It still allowed too much adjacent motion: shell-wrapper probes,
unrelated validation such as `go test ./...`, ticket evidence edits, and an
implementation commit happened before the exact failed command passed. The job
ended at `max_turns`; no target intervention-debt ticket or Orchestrator loop
was created, but bad source state was committed as progress.

### Decision

When Engineer has an unresolved positive runtime acceptance failure,
`shell_exec` is constrained to the runtime repair lane. It may rebuild the same
stale `<validation-root>` artifact after a source edit, or rerun the exact
failed runtime command after the source is repaired. Other shell probes, shell
wrappers, tests, placeholders, commits, and ticket moves are blocked until the
failure is repaired.

Engineer product commits are also blocked while an unexpected runtime
validation failure is outstanding. The role must keep failed implementation
state uncommitted, inspect and edit source with repository tools, rebuild a
stale validation artifact if required, and make the exact failed command pass
before committing product work or progressing ticket lifecycle.

### Consequences

- A failed acceptance path cannot be converted into apparent progress through
  a clean commit or a different validation command.
- Stale validation artifact rebuilds remain available, so the repair path does
  not deadlock after legitimate source edits.
- The boundary is intentionally generic: it applies to any target software
  where the role has produced direct runtime evidence, not just CLI canaries.
- The next clean canary should confirm Engineer repairs the failed runtime
  acceptance path before committing implementation work or moving tickets.

## AD-192: Engineer Evidence Blocks Do Not Become Ticket-Creation Debt

**Status:** Accepted
**Date:** 2026-05-21
**Owner:** Mars Harness maintainers

### Context

The `demo-temp-run37` replay deliberately used a different product shape: a
Temperature JSON CLI with required `--celsius 0`, `--celsius 100`, and omitted
flag behavior. The lifecycle reached implementation and validated real product
behavior. Engineer first attempted to populate ticket evidence before
validation, and policy correctly blocked that write. Engineer then ran tests,
corrected the omitted-flag command with exact `expected_exit_code`, committed
product source and tests, updated evidence, passed docsync, and moved the
ticket to done.

The successful disposition was falsely blocked. Session accounting treated the
earlier failed in-progress ticket-evidence write as unresolved ticket-creation
debt. That rule was intended to stop planning roles from bypassing failed
`ticket_create` calls by writing ticket files directly, but it was too broad
for Engineer evidence updates on already-created tickets.

### Decision

Ticket-creation failure accounting now remains scoped to actual ticket
creation failure. Failed `ticket_create` calls still create outstanding
ticket-creation debt until a later successful `ticket_create` clears it.
Failed non-Engineer `file_write` attempts under `docs/tickets/` still count as
ticket-file creation or bypass attempts. Failed Engineer ticket evidence
updates do not increment ticket-creation debt; their own guardrail message is
enough to route Engineer back to validation or evidence repair.

### Consequences

- CTO and other planning roles still cannot record successful handoff after
  malformed ticket creation or direct ticket-file bypass attempts.
- Engineer can recover from pre-validation evidence mistakes and later record
  a valid `qa_review` disposition after successful validation, evidence update,
  commit, and done-ticket lifecycle move.
- Ticket evidence ordering remains enforced, but it no longer poisons a later
  valid implementation handoff.
- The next clean canary should confirm a pre-validation evidence block does
  not prevent Engineer disposition after the ticket is properly completed.

## AD-193: Review No-Op Recovery Becomes Terminal-Only

**Status:** Accepted
**Date:** 2026-05-21
**Owner:** Mars Harness maintainers

### Context

The `demo-temp-run38` replay confirmed that the factory can now move a second
CLI archetype from product-specific CEO planning through CTO ticketing and
Engineer implementation. It also exposed two coordination leaks.

First, COO did not own ticket creation but still tried several alternate paths:
direct `file_write` under `docs/tickets/`, `mars_harness_cli ticket_create`,
and `mars_harness_cli tools run ticket_create`. Policy blocked each attempt,
but the role only escaped by recording a blocked disposition and spending an
extra Orchestrator turn before CTO could create the implementation ticket.

Second, QA read the done ticket, inspected source, built a fresh validation
binary, and ran a passing runtime probe, but then looped on empty or placeholder
`shell_exec` calls instead of recording the terminal disposition. The terminal
tool circle grace helped contain the loop, but because the calls alternated
between empty argv and shell-placeholder shapes, the job still reached
`circle_detected`. QA also failed to turn the missing durable Go tests into a
structured `changes_requested` disposition.

### Decision

Planning handoff is explicit: COO and other non-ticket-owning planning roles
must not create tickets through `ticket_create`, direct ticket `file_write`,
`mars_harness_cli`, or shell commands. They commit planning artifacts and hand
off `next_need: ticket_breakdown` with `suggested_role: cto-weekly`. If such a
planning role already hit a ticket-creation policy block for work it does not
own, the successful `ticket_breakdown` disposition remains available so the
run can move to the proper ticket owner instead of pretending implementation is
ready.

Review no-op recovery is terminal-only. When QA or Security receives a blocked
no-op shell placeholder after successful validation, session state records that
the next tool must be `job_disposition_record`. Further shell validation,
placeholders, or other non-terminal tools are blocked with disposition
guidance. If the target contains Go source but no `_test.go` files, that
guidance tells QA to record `changes_requested` for Engineer tests rather than
approve from runtime smoke evidence alone.

### Consequences

- Planning-role mistakes no longer require an extra blocked disposition plus
  Orchestrator detour before CTO can create a product ticket.
- Review roles cannot burn the rest of their turn budget on alternating
  no-op shell shapes after the policy has already told them to finish.
- Missing durable Go tests become explicit review feedback instead of a hidden
  weakness behind successful build/runtime smoke output.
- The next clean canary should confirm QA exits through a structured
  disposition after the first no-op block, and routes missing-test products to
  Engineer rework.

## AD-194: Failed Test And Build Evidence Freezes Engineer Side Paths

**Status:** Accepted
**Date:** 2026-05-21
**Owner:** Mars Harness maintainers

### Context

The `demo-temp-run39` replay confirmed the AD-193 planning improvements on the
Temperature JSON CLI target. CEO and COO produced product-specific planning,
COO handed directly to CTO without alternate ticket creation, and CTO created
the product ticket. Engineer then implemented source and a `_test.go` file,
which is progress over the missing-test failure in run 38.

The next failure was validation discipline. `go test` failed because the
implementation had duplicate helper definitions and test/runtime entrypoint
drift. Engineer proved some runtime probes with `go run main.go`, but the
authoritative test command still failed. The role then attempted forbidden
`rm` cleanup, updated evidence, and committed product work before tests passed.
No target intervention-debt ticket or Orchestrator loop was created, but the
factory still allowed broken test evidence to be bypassed by runtime side
probes and a product commit.

### Decision

Engineer test and build failures are now tracked as an explicit repair lane.
When Engineer observes a failing test or build command in the current job,
the session records the exact command fingerprint and blocks unrelated
`shell_exec` calls, runtime probes, ticket moves, ticket evidence updates,
successful dispositions, and product commits until the exact failing command
passes after a source or test edit.

The repair lane is generic. It is not tied to Go or to the Temperature JSON
CLI canary. It applies to any recognized test/build command for the target
stack. Source and test `file_write` repairs remain available; ticket evidence
and lifecycle writes are blocked until the failed test/build command is clean.

### Consequences

- Passing runtime probes cannot outrun failing tests or builds.
- Engineers keep a direct repair path through source/test edits and an exact
  validation rerun instead of being pushed into target intervention debt.
- Broken product state is harder to preserve as a semantic implementation
  commit.
- The next clean canary should confirm a failing `go test` forces source/test
  repair and exact passing rerun before ticket evidence, ticket completion, or
  product commit can continue.

## AD-195: CTO Technical Planning Is Not Product Implementation

**Status:** Accepted
**Date:** 2026-05-21
**Owner:** Mars Harness maintainers

### Context

The `demo-temp-run40` replay confirmed the early AD-193 path again: CEO chose
the Temperature JSON CLI slice, COO updated product-specific planning and the
canonical feature contract, committed those artifacts, and routed CTO without
alternate ticket creation.

CTO then created the product ticket but crossed the role boundary. It wrote
`go.mod`, repeatedly attempted product source and test writes, updated README
usage notes, and committed `go.mod` plus the ticket before handing work to
Engineer. Source-file DocSync guardrails blocked some writes, but package and
README product mutations still escaped because CTO was treated as a generic
planner with `file_write`.

### Decision

CTO file writes are now bounded to technical planning artifacts:
`docs/design-docs/`, `docs/reports/strategy/`, and
`docs/goals/observations.md`. Product implementation files, package/module
files, README usage notes, tests, build config, and root product files belong
behind `ticket_create` and ticket-backed Engineer delivery. The generated CTO
prompt, persona docs, role registry, and tools glossary now state this
boundary explicitly.

### Consequences

- CTO can still record architecture rationale and create implementation
  tickets, but cannot start the implementation while shaping that ticket.
- Package/module initialization such as `go.mod` is reserved for Engineer,
  which keeps implementation, validation, and ticket evidence in one role
  transcript.
- The next clean canary should confirm CTO creates and commits only the
  implementation ticket, then hands to Engineer without dirty product files.

## AD-196: Test/Build Repair Lanes Allow Same-Lane Validation, Not Workarounds

**Status:** Accepted
**Date:** 2026-05-21
**Owner:** Mars Harness maintainers

### Context

The `demo-temp-run41` replay confirmed the CTO boundary from AD-195. CEO and
COO produced product-specific Temperature JSON CLI planning, CTO created and
committed only the implementation ticket, and Engineer claimed the ticket
before writing `go.mod`, source, tests, and README usage notes.

The next failure was in AD-194's repair lane strictness. Engineer ran
`go test ./cmd/temperature-json-cli/...`, fixed an unused import, then hit a
subprocess path failure in the test. The guardrail correctly blocked runtime
probes, cleanup, ticket evidence, and product commits while the test failure
was unresolved, but it also rejected reasonable same-lane validation attempts
such as `go test ./cmd/temperature-json-cli` because they were not the exact
original command. That trapped the role into repeated guardrail blocks and
workaround attempts, including an ad hoc root verification script.

### Decision

Engineer test/build repair is lane-based rather than exact-command-only. After
a failing test command, Engineer may edit product source, tests, fixtures, or
package/build config, then run another recognized test command. After a
failing build command, Engineer may make the same bounded repair writes and
run another recognized build command. Runtime probes, unrelated shell
commands, helper scripts, ticket evidence updates, ticket done moves,
successful dispositions, and product commits remain blocked until a same-lane
test/build command passes.

The policy also blocks new root scratch verification files whose names contain
validation, scratch, or verify language, so ad hoc scripts such as
`verify_functionality.sh` do not become product noise during repair.

### Consequences

- The repair lane keeps the quality invariant that failing tests/builds cannot
  be bypassed by runtime probes or commits.
- Engineer can recover from legitimate command-shape drift, package scoping,
  and framework-specific focused test commands without getting stuck on a
  single exact command string.
- Repair writes stay product-relevant: source, tests, fixtures, and build
  config are allowed, while helper scripts, ticket evidence, and unrelated
  docs wait until validation is clean.
- The next clean canary should confirm Engineer can repair a failing test,
  rerun a focused same-lane test command, and continue toward product commit
  and ticket closure without creating workaround files.

## AD-197: Simple CD Validation Shell Commands Count As Same-Lane Repair

**Status:** Accepted
**Date:** 2026-05-21
**Owner:** Mars Harness maintainers

### Context

The `demo-temp-run42` replay confirmed AD-195 again and partially confirmed
AD-196. CTO created only the implementation ticket. Engineer claimed it, wrote
source and tests, ran `go test ./cmd/temperature-json-cli/...`, and was
blocked from runtime probes, build substitution, helper scripts, and product
commits while the failing test lane was unresolved.

The remaining trap was command classification. The local model repeatedly used
the natural shell shape `cd cmd/temperature-json-cli && go test -v .` for
focused package validation. The shell executor can run that command, but the
repair-lane classifier treated any shell control syntax as unclassifiable and
therefore blocked it as an unrelated side path. The result was another
guardrail loop even though the requested command was still a test command in
the correct lane.

### Decision

Validation command recognition now understands the narrow shell pattern
`cd <dir> && <test-or-build command>` for repair-lane classification. The
recognized validation command is the right-hand side after the simple `cd`.
Arbitrary shell control syntax, multiple chained operations, pipes,
redirection, substitutions, and shell wrappers remain unclassified for the
repair lane and are still blocked while a failing test/build lane is
unresolved.

### Consequences

- Engineer can use the common focused validation shape produced by local
  models and Go projects without escaping the test/build repair lane.
- The policy stays narrow: this is not a general shell-wrapper bypass for
  runtime probes, cleanup, ticket moves, or helper scripts.
- The next clean canary should confirm `cd <package> && go test -v .` can
  repair a failed package-pattern test and allow the product lifecycle to move
  toward commit and ticket evidence.

## AD-198: Clean Review Evidence Forces A Terminal Disposition

**Status:** Accepted
**Date:** 2026-05-21
**Owner:** Mars Harness maintainers

### Context

The `demo-temp-run43` replay confirmed the AD-196 and AD-197 direction on a
fresh Temperature JSON CLI target. CEO, COO, and CTO stayed product-specific.
Engineer claimed the ticket, implemented source, tests, README usage notes,
and `go.mod`, corrected an expected missing-input runtime probe with
`expected_exit_code`, repaired a failing `go test`, committed product work,
updated ticket evidence, moved `T-001` to done, and handed off to QA. QA read
the completed ticket and implementation, ran docsync, tests, and happy-path
runtime probes, then approved.

The remaining failure was review completion latency. Security inspected recent
commits, scanned for secrets, ran docsync, ran the relevant test command, read
source, and ran a successful runtime probe. At that point the review had clean
evidence and no failing validation, but the next model turn spent more than
five minutes instead of recording `job_disposition_record`. Stopping the run
cancelled that LLM call and produced a foundation-owned `llm_unreachable`
signal. No target intervention-debt ticket was created, and product progress
was preserved, but the review stage still relied on the model deciding when it
had enough evidence.

### Decision

Review evidence convergence is now mechanical. When QA or Security has both a
successful `file_read` inspection and at least one successful validation
command in the current job, and no test, build, or unexpected runtime
validation failure is outstanding, the agent loop appends a terminal-only
reminder. The next response must call the configured terminal tool,
`job_disposition_record`, and any other tool or prose-only answer ends the job
as a loop boundary instead of executing more inspection.

The same terminal-only grace response is bounded by a short per-call timeout.
After the loop has already told a required-terminal-tool job to finish because
of turn budget, repeated tool-call shape, or sufficient review evidence, the
next LLM completion is capped so a simple disposition cannot consume another
full default inference timeout.

### Consequences

- QA and Security no longer need to stumble into a no-op shell block before the
  runtime tells them to finish.
- Clean review evidence becomes a convergence point: decide, request changes,
  or block, but do not continue broad validation.
- The target backlog remains protected because review dithering and inference
  stalls stay foundation telemetry rather than intervention-debt tickets.
- The next clean canary should confirm Security records a terminal disposition
  immediately after clean read plus validation evidence and then routes forward
  without another long model turn.

## AD-199: Reviewer Command Procedure Failures Stay In Review

**Status:** Accepted
**Date:** 2026-05-21
**Owner:** Mars Harness maintainers

### Context

The `demo-temp-run44` replay confirmed the product-first path on another fresh
Temperature JSON CLI target. CEO, COO, and CTO created product-specific
planning and one implementation ticket. Engineer claimed the ticket, wrote the
Go CLI, proved the two positive Celsius conversions, corrected the
missing-argument negative path with `expected_exit_code`, committed product
work, updated evidence, moved `T-001` to done, and handed to QA. Intervention
signals from guardrail blocks and the operator stop stayed foundation-owned
and did not become target backlog tickets.

QA then hit a review-procedure failure, not a product failure. It ran:

```json
{"argv":["go","build","-o","<validation-root>","cmd/temperature-json-cli"]}
```

Go rejected that package path because local package targets require the
`./cmd/...` form. QA immediately tried corrected build targets, but the review
policy had already counted the first command as a failing build and blocked
further validation, forcing `changes_requested` for implementation rework even
though the target source had previously built and run successfully.

### Decision

Reviewer validation now distinguishes obvious validation-procedure mistakes
from target product failures. For QA and Security, Go test/build commands that
fail because a repo-relative package path omitted `./`, or because the reviewer
builds `.` in a CLI repo whose main package lives under `cmd/*`, are recorded
as `validation:procedure_failure` rather than `validation:command:failure` or
`validation:build:failure`.

That keeps the review lane open for a corrected validation command in the same
job. Real compile errors, failing tests, failing runtime probes, and
uncorrected expected-negative runtime probes still force the existing
structured `changes_requested` path.

### Consequences

- Reviewers can correct their own command-addressing mistakes without sending
  healthy product work back to Engineer.
- The quality gate remains strict for target-owned failures because only
  recognizable validation-procedure failures bypass the failure counters.
- The next clean canary should confirm QA can recover from `cmd/<name>` or root
  `.` build-target mistakes by running the corrected `./cmd/<name>` build,
  then continue to runtime evidence and terminal disposition.

## AD-200: Simple CD Validation Argv Normalizes To Shell Command

**Status:** Accepted
**Date:** 2026-05-21
**Owner:** Mars Harness maintainers

### Context

The `demo-temp-run45` replay started cleanly and confirmed the product-first
bootstrap path again. CEO named the Temperature JSON CLI goal, COO rewrote the
active plan and feature contract, and CTO-weekly created and committed the
implementation ticket without product mutation. Engineer claimed the ticket
and began implementation.

The run then exposed a validation-interface mismatch. Engineer created a
nested Go module under `cmd/temperature-json-cli`, which means the correct
test command is naturally:

```json
{"shell_command":"cd cmd/temperature-json-cli && go test ./..."}
```

The model first attempted the same operation in argv form:

```json
{"argv":["cd","cmd/temperature-json-cli","&&","go","test","./..."]}
```

`shell_exec` correctly rejected that because argv mode does not run shell
builtins or control operators. The model then tried the root command
`go test ./cmd/temperature-json-cli/...`, which Go rejected because the nested
module is outside the root module. That failure opened the Engineer test
repair lane and blocked builds, commits, discovery, and cleanup even though
the original intended validation was a safe focused test command.

### Decision

`shell_exec` now normalizes only this narrow argv mistake into the existing
shell-command path: `["cd","<dir>","&&",<test-or-build-command>...]`. The
right-hand command must classify as a recognized test or build command, and
all tokens must be simple tokens without pipes, redirects, substitutions,
background operators, or arbitrary shell control syntax.

The normalized command then flows through the same shell-command validation
and policy used by AD-197. General shell syntax in argv mode remains rejected.

### Consequences

- Local models can recover from the common `cd ... && go test` argv formatting
  mistake without turning safe validation into a product failure.
- The policy remains narrow: cleanup, runtime probes, shell wrappers,
  redirection, pipes, substitutions, package-manager mutation, and ticket moves
  are not normalized.
- The next clean canary should confirm the Engineer can run nested-module test
  validation through the normalized argv shape, continue to product commit,
  and reach QA for the AD-199 and AD-198 review checks.

## AD-201: CLI Input-Validation Probes Are Expected Negative-Path Evidence

**Status:** Accepted
**Date:** 2026-05-21
**Owner:** Mars Harness maintainers

### Context

The `demo-temp-run46` replay validated the earlier stabilizations and then
found a narrower loop in Engineer runtime validation. CEO, COO, and
CTO-weekly again stayed product-first for the Temperature JSON CLI target.
Engineer claimed `T-001`, wrote the CLI implementation, created `go.mod` and
tests, and first attempted to build a repo-local binary. AD-196 blocked that
artifact path and gave the exact `<validation-root>`
correction, which Engineer followed. The binary then returned correct JSON for
`--celsius 0` and `--celsius 100`.

Engineer next ran the validation binary without arguments. For this product,
that is a useful negative-path check: the CLI should reject missing required
input and print a clear error. The binary exited non-zero with
`--celsius flag is required`, but because the role had not supplied
`expected_exit_code`, the runtime policy treated the probe as an unexpected
failure. The correction guidance existed, but the local model drifted into
other probes, edits, commits, ticket moves, and dispositions instead of
rerunning that exact command with `expected_exit_code`. The guardrail then did
the right thing by blocking completion, but the job made no further product
progress.

The `demo-temp-run47` replay confirmed that the missing-input probe no longer
poisoned the job. Engineer successfully ran `go run ...` with no arguments,
received clear missing-input usage text, and continued. The next negative path
was an explicit bad input value: `go run ... invalid` returned
`Invalid temperature value 'invalid'. Must be a number.` The product behavior
was correct, but the same runtime repair guardrail treated the invalid-input
check as unexpected because the classifier only recognized missing input.

### Decision

An obvious CLI input-validation runtime probe now counts as expected
negative-path validation immediately when all of these are true:

- The command shape is a direct runtime probe, such as a single validation
  binary, `go run`, `cargo run`, or a language entrypoint.
- The probe is recognizably an input-validation check, such as no required
  input or a deliberately bad argument like `invalid`.
- The command exits non-zero.
- Output or stderr clearly describes required, missing, usage, or similar
  input validation.
- Output does not contain crash markers such as panic, traceback, exception,
  runtime error, or segmentation fault.

Explicit `expected_exit_code` remains supported and still records expected
negative-path evidence. Positive acceptance failures and crash-like missing
input failures still open the runtime repair lane and block completion until
the exact command is repaired.

### Consequences

- Engineer can validate standard CLI missing-input and invalid-input behavior
  without a second corrective tool call.
- The guardrail remains strict for crashes and broken positive behavior.
- The next clean canary should confirm Engineer can commit and close the
  Temperature JSON CLI ticket after positive JSON checks, missing-input
  validation, and invalid-input validation, then reach QA/Security review.

## AD-202: Engineer Command Procedure Failures Stay Out Of Repair Lane

**Status:** Accepted
**Date:** 2026-05-21
**Owner:** Mars Harness maintainers

### Context

The `demo-temp-run48` replay confirmed AD-201 in the live lifecycle. Engineer
implemented the Temperature JSON CLI, built the external validation binary,
proved positive Celsius conversions, proved missing-input behavior, proved
invalid-input behavior, ran `go test ./...`, committed product work, moved
`T-001` to done, and handed off to QA.

QA then requested implementation rework because the delivered Go package had
no `_test.go` file, even though the ticket carried runtime evidence. The
orchestrated Engineer rework exposed another command-procedure trap: Engineer
ran `go build -o <validation-root>
cmd/temperature-json-cli/`. Go rejected the package target because it needed
`./cmd/temperature-json-cli`. The corrected command was obvious, but the
Engineer repair-lane policy treated the first command as a real build failure
and blocked the corrected validation command until a source edit occurred.

### Decision

Validation-procedure failure classification now applies to Engineer as well
as QA and Security. Recognizable Go package-target mistakes, including missing
`./` for repo-relative package paths and root `.` builds in `cmd/*` CLI repos,
record `validation:procedure_failure` rather than test/build failure counters.

Corrected validation commands remain available in the same job. Real compile
errors, failing tests, and target-owned build failures still open the normal
Engineer repair lane.

### Consequences

- Engineer can correct command-addressing mistakes without being forced into a
  meaningless source edit.
- The repair lane remains strict for real target failures.
- The next clean canary should confirm Engineer rework can recover from
  `cmd/...` build target mistakes and continue to validation, commit, and
  review handoff.

## AD-203: Surplus CLI Argument Probes Are Expected Negative-Path Evidence

**Status:** Accepted
**Date:** 2026-05-21
**Owner:** Mars Harness maintainers

### Context

The `demo-temp-run49` replay confirmed AD-201 and AD-202 were moving the
alternate CLI target forward. CEO, COO, and CTO-weekly produced
product-specific planning and a product ticket. Engineer claimed `T-001`,
created the Go CLI with DocSync metadata, corrected a repo-local build-output
guardrail by building `<validation-root>`, proved the
positive `25` Celsius conversion, and proved the missing-input negative path.

Engineer then ran `<validation-root> 25 30`. The CLI
correctly rejected the surplus positional argument with `error: too many
arguments provided`, but the runtime policy treated the non-zero exit as an
unexpected product failure. The guardrail then blocked later runtime probes,
build/test commands, and completion. The classification was too narrow: it
recognized missing input and obviously invalid single values, but not the
common CLI validation case where extra input is rejected.

### Decision

Surplus-argument CLI probes now count as expected negative-path runtime
evidence when all of these are true:

- The command is a direct runtime probe.
- The command includes more than one product argument.
- The command exits non-zero.
- Output names surplus input, such as too many arguments, too many args, too
  many values, at most one, only one, single argument, or exactly one argument.
- Output does not contain crash markers such as panic, traceback, exception,
  runtime error, or segmentation fault.

Missing-input and invalid-input probes keep the AD-201 classification. Positive
acceptance failures and crash-like extra-input failures still open the strict
runtime repair lane.

### Consequences

- Engineer can validate ordinary CLI arity behavior without poisoning product
  completion.
- Multi-argument positive paths are not blanket-exempt; the output must show a
  surplus-input validation message.
- The next clean canary should confirm the Temperature JSON CLI can complete
  positive, missing-input, invalid-input, and surplus-argument validation
  without entering the unresolved runtime failure lane.

## AD-204: Test-Build Repair Can Remove Same-Job Bad Test Files

**Status:** Accepted
**Date:** 2026-05-21
**Owner:** Mars Harness maintainers

### Context

The `demo-temp-run50` replay confirmed the factory could again reach Engineer
implementation on the alternate CLI target. Engineer claimed `T-001`, wrote a
Go CLI with DocSync metadata, built an external validation binary, proved
positive Celsius conversions, and used explicit `expected_exit_code` for
missing-input and invalid-input probes. It then added Go tests, and those
tests failed with a real compile error caused by duplicate test helper/type
definitions.

The test/build repair lane correctly blocked runtime probes, build-command
switching, ticket moves, and unrelated shell commands while the test failure
was unresolved. However, the only available deletion surface for the bad
same-job test files was `shell_exec rm ...`, and the repair lane blocked that
too. The model then created more duplicate test files while trying to route
around the blockage. The guardrail protected product completion, but it also
removed the practical path for deleting broken test artifacts created during
the same repair attempt.

### Decision

During an unresolved Engineer test/build failure, `shell_exec rm` or `unlink`
is allowed only when all of these are true:

- The role is Engineer.
- A test/build validation failure is currently outstanding.
- The removal command has no recursive flags.
- Every removed path is inside the repo.
- Every removed path is test-like, such as a source file whose basename
  contains `test`, or a path under `testdata/` or `fixtures/`.
- Every removed path was written by the same job after the test/build failure
  began.

Unmarked test files, product source files, recursive removal, generated
cleanup outside the existing artifact rules, and ordinary cleanup remain
blocked.

### Consequences

- Engineer can undo bad same-job test repair files without leaving the failing
  test lane.
- The model cannot delete old tests or source files simply to make validation
  pass.
- The next clean canary should confirm Engineer can recover from bad test-file
  generation and then rerun the same-lane test successfully.

## AD-205: Review Terminal Evidence Gets One Missed-Tool Correction

**Status:** Accepted
**Date:** 2026-05-21
**Owner:** Mars Harness maintainers

### Context

The `demo-temp-run51` replay confirmed the product-first path through a full
fresh target lifecycle. CEO and COO produced a product-specific Temperature
JSON CLI plan and feature contract. CTO-weekly created one product ticket.
Engineer claimed it, wrote the Go CLI and tests, built the external validation
binary, proved positive and negative CLI behavior, ran `go test ./...`, passed
DocSync, committed the implementation, moved `T-001` to done, and handed off
to QA.

QA then repeated a familiar review command-procedure mistake. It first ran
`go build -o <validation-root> cmd/temperature-json-cli`,
then corrected the package target to `./cmd/temperature-json-cli`. The
corrected build succeeded, which meant review evidence was sufficient. The
agent loop correctly appended terminal-only guidance requiring
`job_disposition_record`, but the next model response attempted another
non-terminal tool and the job ended as `circle_detected`. Product progress was
preserved and no target intervention debt was created, but the strict boundary
lost a good review after the model had just repaired its command.

### Decision

Review evidence convergence remains mechanical: once QA or Security has clean
read plus validation evidence and no outstanding product failure, the next
useful action is `job_disposition_record`.

The first non-terminal response after that terminal-evidence reminder is now
rejected in-band instead of ending the job immediately. The rejected tool is
not executed. The loop appends one stronger correction naming the attempted
non-terminal tool, repeats the terminal disposition guidance, and allows one
more model response. If that response is still not the required terminal tool,
the job ends with `circle_detected`.

### Consequences

- A single missed terminal-tool instruction no longer loses a completed review.
- Review roles still cannot keep inspecting or validating indefinitely after
  sufficient evidence exists.
- The rejected non-terminal tool does not mutate target state or consume a
  tool execution.
- The next clean canary should confirm QA turns the correction into an
  approved or changes-requested `job_disposition_record` instead of another
  `circle_detected` failure.

## AD-206: Test-Build Repair Writes Stay In The Failed Scope

**Status:** Accepted
**Date:** 2026-05-21
**Owner:** Mars Harness maintainers

### Context

The `demo-temp-run52` replay used the AD-205 binary against a fresh
Temperature JSON CLI target. CEO and CTO-weekly still produced a product
ticket without target intervention-debt churn, and Engineer claimed `T-001`.
Engineer wrote `cmd/temperature-json-cli/main.go`, `cmd/temperature-json-cli/main_test.go`,
and `go.mod`, then ran `go test ./cmd/temperature-json-cli/...`.

That package test failed with ordinary test/build defects. The repair lane
correctly blocked runtime probes, destructive `rm -rf`, commits, and ticket
completion. However, `file_write` still allowed any source file while the test
failure was outstanding. The model responded by creating a parallel root
`main.go` and `main_test.go`, then repeated root `go test ./...` attempts
while the original package tests remained failing. The guardrail prevented
unsafe completion, but the allowed repair surface was too broad.

### Decision

When an Engineer test/build command fails, the runtime records a repair scope
from recognized Go package targets such as `./cmd/temperature-json-cli/...`.
While that failure remains outstanding, source, test, fixture, and testdata
writes are allowed only inside the failed scope. Package/build config files
such as `go.mod`, lockfiles, and test/build configuration remain available
because they may be required to fix the same lane.

If no narrower scope can be derived, such as a repo-wide `go test ./...`, the
repair lane keeps the prior repo-wide source repair behavior. Scope tracking
is cleared when the test/build failure is repaired.

### Consequences

- Engineer can repair the package that failed without spawning alternate
  entrypoints that make validation noisier.
- Same-lane validation remains available after scoped repair edits.
- Broad repo-wide failures still allow broad repair because no narrower
  package scope is known.
- The next clean canary should confirm a failing package test leads to
  `cmd/...` repair or structured blocked disposition, not root-level duplicate
  implementation.

## AD-209: Test-Build Repair Can Remove Same-Job Tests Written Before Failure

**Status:** Accepted
**Date:** 2026-05-21
**Owner:** Mars Harness maintainers

### Context

The `demo-temp-run55` replay used a fresh Temperature JSON CLI target to
validate run metadata and the earlier release/tag fixes. CEO, COO, and
CTO-weekly again produced product-specific planning and one ordinary product
ticket without target intervention-debt pollution. Engineer claimed `T-001`
and wrote real product files under `cmd/temperature-json-cli/`.

The run failed in the Engineer phase. The first failing command was
`go test ./cmd/temperature-json-cli -run TestTemperatureCLI`, and the failure
was caused by duplicate or placeholder generated test files. The test/build
repair lane correctly blocked runtime probes, build switching, ticket evidence,
ticket completion, commits, and a false successful disposition. However, the
only cleanup path for a duplicate test written before the failing command was
`rm -f cmd/temperature-json-cli/main_test.go`. AD-204 allowed removal only for
test files written after the failure began, so that cleanup stayed blocked and
the role spent the remaining turns trying to work around the test failure.

### Decision

Every successful Engineer `file_write` now records the written path in the
job session. During an unresolved test/build repair lane, non-recursive
`rm` or `unlink` may remove a test-like file only when that exact path was
created or rewritten earlier by the same Engineer job. The previous safety
constraints still apply: the path must resolve inside the repo, must be
test-like or fixture/testdata, and source removal remains blocked even if the
source file was written by the same job.

This expands the same-job cleanup exception from "written after the failing
test" to "written earlier in the same job" without allowing deletion of old
project tests.

### Consequences

- Engineer can prune duplicate generated tests that it created before the
  first failing test command.
- Pre-existing project tests remain protected from deletion as a repair
  shortcut.
- The test/build repair lane keeps runtime probes, ticket completion, commits,
  and successful dispositions blocked until the same-lane test/build command
  passes.
- The next clean canary should confirm Engineer removes or rewrites duplicate
  generated tests, reruns `go test`, commits product work, and reaches QA
  instead of exhausting `max_turns`.

## AD-210: Review Terminal Boundary Waits For DocSync Evidence

**Status:** Accepted
**Date:** 2026-05-21
**Owner:** Mars Harness maintainers

### Context

The `demo-temp-run56` replay used a fresh Temperature JSON CLI target after
AD-209. The run validated the same-job test cleanup direction: the first
Engineer completed product work, QA requested ordinary test-coverage rework,
Orchestrator sent the same ticket back to Engineer, and the second Engineer
added `cmd/temperature-json-cli/cli_test.go`, passed
`go test ./cmd/temperature-json-cli/`, committed, pushed locally, moved
`T-001` back to done, and recorded a successful disposition.

The next QA job then read the ticket, README, implementation, and tests, and
ran `go test ./cmd/temperature-json-cli/` successfully. At that point the
agent loop decided review evidence was sufficient and forced a terminal-only
`job_disposition_record` boundary. The QA role guidance still requires
`docsync_audit` evidence before approval, so the model attempted
`docsync_audit`; the runtime rejected it as post-validation churn, sent one
stronger terminal correction, and the job ended with `circle_detected` after a
second missed terminal call.

The failure was not product logic and not target backlog work. It was a
foundation review-boundary contradiction between "docsync is required before
approval" and "validation evidence is already sufficient, no more tools".

### Decision

Review terminal convergence now waits for docsync evidence. For review roles
that require docsync on successful dispositions, clean `file_read` plus
successful validation is no longer enough to trigger the terminal-only
boundary. The job must also have a successful `docsync_audit` tool call.

After `docsync_audit` has run, the existing convergence behavior remains: the
next non-terminal tool after sufficient review evidence is rejected without
execution, the model gets one stronger terminal-only correction, and repeated
misses still end with `circle_detected`.

The generated QA persona now states this ordering explicitly: run
`docsync_audit` before final approval, then call `job_disposition_record`;
successful `job_disposition_record` approvals still run the mechanical
docsync policy as a final guard.

### Consequences

- QA and Security can satisfy no-stale-documentation evidence before the
  runtime closes the review loop.
- The runtime no longer asks models to choose between role doctrine and the
  terminal-tool correction.
- Successful approvals still get mechanical docsync enforcement at
  `job_disposition_record`.
- The next clean canary should confirm the post-rework QA path runs
  `docsync_audit`, records an approved or changes-requested disposition, and
  proceeds to the next lifecycle role instead of ending with
  `circle_detected`.

## AD-211: Review Terminal Boundary Waits For Required Tests

**Status:** Accepted
**Date:** 2026-05-21
**Owner:** Mars Harness maintainers

### Context

The `demo-temp-run57` replay validated the AD-210 ordering change but exposed
the next review-boundary edge. QA read the done ticket, feature contract,
README, implementation file, and ran `docsync_audit` successfully. It then
built the external validation binary with
`go build -o <validation-root> ./cmd/temperature-json-cli`.
Because the terminal convergence heuristic treated any successful validation
command as enough, it forced the `job_disposition_record` boundary after that
build even though the target contained `_test.go` files and QA had not yet run
the authoritative Go test command. The model attempted more shell validation,
received the stronger terminal-only correction, and ended with
`circle_detected`.

The failure stayed foundation-owned telemetry and did not create target
intervention debt or dispatch an Orchestrator loop.

### Decision

Review terminal convergence now mirrors the successful-disposition approval
policy for test-bearing repositories. If the target repo contains test files,
clean review evidence is not considered terminal-sufficient until the review
job has recorded a successful test command. Build-only validation may still be
valuable evidence, but it cannot prematurely cut off QA or Security before the
test suite has run.

The missing-test path remains unchanged: if Go source exists with no
`_test.go` files, the terminal guidance points QA at a `changes_requested`
disposition for Engineer test coverage rather than allowing indefinite
inspection.

### Consequences

- Reviewers can still build external validation binaries, but build-only
  evidence no longer preempts required tests.
- The terminal-only boundary remains available after docsync plus successful
  tests, preserving the protection against long post-evidence review loops.
- The next clean canary should confirm QA runs `go test` after external build
  evidence and records a structured disposition instead of ending with
  `circle_detected`.

## AD-212: Review No-Op Recovery Uses The Same Evidence Gates As Approval

**Status:** Accepted
**Date:** 2026-05-21
**Owner:** Mars Harness maintainers

### Context

The `demo-temp-run58` replay validated that AD-211 prevented the terminal
boundary from firing immediately after build evidence, but exposed a sibling
path. QA built `<validation-root>` and then called
`shell_exec` with an empty argv. The review no-op guard still treated any prior
successful validation as "terminal evidence is sufficient" and told QA to call
`job_disposition_record` with approval. The disposition policy correctly
rejected approval because test files existed and the authoritative test command
had not passed. The agent loop then repeated terminal-only guidance and QA
ended with `circle_detected`.

The failure stayed foundation-owned telemetry and did not dispatch an
Orchestrator loop or create target intervention debt.

### Decision

Review no-op recovery now shares the same evidence gates as successful review
approval. A blocked review `shell_exec` no-op no longer globally marks
`review:terminal_disposition:required` just because some validation succeeded.
The agent loop alone turns review evidence into a terminal-only boundary, using
`ReviewTerminalEvidenceSatisfied` with access to the target repo, test files,
docsync state, read evidence, and validation failures.

The no-op policy now gives concrete missing-evidence guidance:

- If test files exist and no test command has passed, the next action is the
  authoritative test command, such as `go test ./...`, or an honest
  `changes_requested` disposition if tests cannot be run.
- If docsync is required and `docsync_audit` has not passed, the next action is
  `docsync_audit` or an honest `changes_requested` disposition.
- Only after the required evidence gates are met does a review no-op route to
  final `job_disposition_record` approval guidance.

### Consequences

- Build-only or runtime-only evidence cannot trap QA/Security between no-op
  terminal guidance and approval policy.
- The runtime preserves the no-op guardrail without turning it into a false
  terminal boundary.
- The next clean canary should confirm QA can recover from a no-op after build
  evidence by running tests, then recording a structured disposition.

## AD-213: Mars Harness CLI Workflows Use Structured Tool Resolution

**Status:** Accepted
**Date:** 2026-05-21
**Owner:** Mars Harness maintainers

### Context

The `demo-temp-run59` replay validated the AD-212 review fix and reached a
complete product lifecycle: CEO, COO, CTO-weekly, Engineer, QA, Security,
Dogfood, and Release Manager all advanced ordinary product work without
intervention-debt starvation. QA approved after build, runtime, test, and
docsync evidence, and the target received a local `release: notes 0.2.0` commit
and `v0.2.0` tag.

The remaining release rough edge was command resolution. The first
Release Manager pass ran `mars-harness release notes --repo . --bump auto
--dry-run` through `shell_exec`, which resolved an older installed
`mars-harness` binary whose command surface did not include `release`. The
role then read the `mars_harness_cli` reference but repeated the same stale
shell command until loop containment forced a failed liveness disposition.
Orchestrator recovered by dispatching Release Manager again; the second pass
completed local release artifacts and stopped cleanly on the expected missing
remote publication blocker.

### Decision

Mars Harness CLI workflows inside agent jobs now use the structured
`mars_harness_cli` tool rather than direct `shell_exec mars-harness ...`
commands. The tool resolves the active harness executable before PATH, which is
the same binary that started the job, and therefore avoids stale installed
binaries in deployed target repos.

The `shell_exec` policy rejects direct `mars-harness` binary invocations in
argv mode or as the first executable in a shell command. The error names the
equivalent `mars_harness_cli` args so the model can recover without guessing a
binary path.

Generated Release Manager guidance now states this explicitly for release
notes, backfill, and related Mars Harness CLI workflows.

### Consequences

- Release review should no longer fail solely because the target machine has
  an older `mars-harness` earlier on PATH.
- The structured CLI tool remains the single mirrored authority for setup,
  update, release, score, trust, model, and harness commands.
- Direct shell remains available for product validation commands such as
  `go test`, built binaries, curl probes, and Git commands that are already
  covered by dedicated release/tag guardrails.
- The next clean canary should confirm Release Manager reaches local release
  notes directly through `mars_harness_cli` and stops only on real publication
  blockers such as a missing remote.

## AD-215: Engineer Test/Build Rework Guidance Carries Failure Output

**Status:** Accepted
**Date:** 2026-05-21
**Owner:** Mars Harness maintainers

### Context

The `demo-slug-run61` replay showed a healthy product-first path through CEO,
COO, CTO-weekly, Engineer, QA, Orchestrator, and rework Engineer before it hit
a repair-guidance gap. QA correctly requested missing Go tests. Engineer added
tests, and `go test` exposed a contract-shaped product mismatch in the Slugify
CLI word-count behavior. Guardrails blocked unrelated shell work, commits,
ticket completion, and successful disposition while the test/build lane was
unresolved, but the repeated guidance named only the command and not the
assertion failure.

### Decision

The tool session now stores a compact copy of the latest failing test/build
output along with the unresolved command. Subsequent guardrail messages include
that output and explicitly state that when a failing assertion matches the
ticket, README, or BDD contract, Engineer must edit the implementation rather
than deleting or weakening the test.

### Consequences

- Rework roles get the concrete failure text even after intervening policy
  blocks or model retries.
- Contract-aligned tests remain first-class product evidence rather than
  disposable scaffolding.
- Successful same-lane validation clears the stored failure output with the
  rest of the unresolved repair state.
- The `demo-slug-run62` replay validated the behavior in a fresh target:
  product planning, ordinary ticketing, implementation with tests, QA,
  Security, Dogfood, local release notes, and tag creation completed without
  re-entering unresolved test/build churn or creating target intervention-debt
  tickets. The remaining blocker was expected missing remote publication in the
  temporary target.

## AD-216: Missing Module Bootstrap Is A Test/Build Repair Action

**Status:** Accepted
**Date:** 2026-05-21
**Owner:** Mars Harness maintainers

### Context

The `demo-notes-api-run63` replay broadened the validation matrix from compact
CLIs to a small HTTP JSON API with a local bare `origin`. The run confirmed the
product-first path again: retry-after-bind used the same CEO bootstrap job,
CEO/COO/CTO-weekly created product-specific planning and an ordinary Notes API
ticket, Engineer claimed and pushed that ticket, and a ticket-gate guardrail
remained foundation telemetry instead of target intervention debt.

Engineer then implemented Go source before initializing the module. The first
`go test ./internal/note` failed with Go's missing-module guidance:
`cannot find main module ... go mod init`. The unresolved test/build repair
lane correctly blocked runtime probes, commits, ticket moves, and unrelated
shell work, but it also blocked the direct remediation command
`go mod init demo-notes-api`. The role then tried destructive or low-value
workarounds such as deleting tests and creating placeholders.

### Decision

Engineer test/build repair now treats missing package/module bootstrap as a
bounded repair action. When the latest failing test/build output explicitly
shows Go's missing-module failure and `go.mod` is absent, Engineer may run
`go mod init <module>` even while the test/build lane is unresolved. The
command remains blocked when `go.mod` already exists or when the failing output
does not prove a missing module.

All other repair-lane gates remain intact: runtime probes, helper scripts,
ticket evidence, ticket completion, successful disposition, product commits,
and unrelated shell commands stay blocked until same-lane validation passes.

### Consequences

- Fresh Go targets can recover from an honest bootstrap ordering mistake
  without forcing the model into placeholder files or test deletion.
- The exception is evidence-backed and narrow: it requires missing-module
  output and an absent module file.
- The next API canary should confirm Engineer runs `go mod init`, reruns
  same-lane tests successfully, commits product work, and continues to QA
  without intervention-debt churn.

## AD-217: Dependency Mutation And Test Cleanup Stay Evidence-Preserving

**Status:** Accepted
**Date:** 2026-05-21
**Owner:** Mars Harness maintainers

### Context

The patched `demo-notes-api-run64` replay avoided the missing-module trap by
writing `go.mod` before validation, but exposed two adjacent generic policy
gaps on an HTTP service target.

First, Engineer used raw `go get github.com/stretchr/testify`, which mutated
dependency state outside `dependency_sync` and contradicted the target brief's
standard-library preference. The raw dependency guard already blocked
`go mod download` but missed `go get`.

Second, after adding tests and seeing an assertion-shaped store failure,
Engineer deleted a same-job test file. Same-job test cleanup existed to remove
duplicate/generated tests during repair, but Run 64 showed that the exception
was too broad when the failure was ordinary product assertion evidence.

### Decision

Raw `go get` is now classified as dependency mutation and blocked with the same
`dependency_sync` guidance as raw package-manager install/fetch commands.

Same-job test cleanup during unresolved test/build repair is now allowed only
when the latest failing output looks like duplicate/generated-test conflict:
markers such as redeclarations, already-declared symbols, duplicate definitions,
mixed packages, or parse/declaration errors. Assertion failures and contract
mismatches must be repaired by source/test edits and same-lane validation, not
by removing the evidence.

### Consequences

- Engineers cannot silently introduce external dependencies through raw
  `go get` while bypassing workspace hygiene and dependency sync policy.
- The duplicate-test cleanup escape hatch remains available for generated test
  collisions but no longer weakens newly written assertion evidence.
- The next API canary should confirm raw dependency mutation is blocked before
  it changes `go.mod`, assertion-failure test deletion is blocked, and Engineer
  repairs the implementation or test logic instead.

## AD-218: Runtime Validation No-Ops Converge Immediately

**Status:** Accepted
**Date:** 2026-05-21
**Owner:** Mars Harness maintainers

### Context

The `demo-inventory-api-run65` canary broadened live validation to a new HTTP
JSON Inventory API target with only a README brief and a local bare `origin`.
The run confirmed the product-first lifecycle after the v0.42.18 guardrail
changes: CEO, COO, CTO-weekly, Engineer, QA, Security, Dogfood, and
Orchestrator all completed real product work before the second Engineer rework
job failed.

The failure happened after Engineer repaired a Dogfood-discovered route
registration bug, passed build/tests, started the API in managed background
mode, and successfully probed `/health`. Instead of stopping the tracked
background PID, committing the dirty implementation, updating ticket evidence,
moving the ticket to done, and handing to QA, Engineer called no-op
`shell_exec` placeholders twice. The second placeholder was blocked, but the
role hit `circle_detected` before recording completion.

### Decision

Engineer no-op shell placeholders after successful validation and dirty
implementation or ticket work now fail in pre-tool policy on the first attempt.
The guardrail names the active ticket, dirty files, tracked background PID kill
commands when present, evidence update, implementation commit, ticket
lifecycle move, push, and `job_disposition_record` handoff.

Concrete validation commands remain allowed while implementation work is dirty,
so this does not prevent the useful `go test`, external build, `go run
background:true`, and `curl` sequence that proved the repair.

### Consequences

- Post-runtime-validation roles no longer need to spend a generic no-op failure
  before receiving terminal convergence instructions.
- Managed background validation cleanup is part of the same convergence path as
  ticket evidence and lifecycle completion.
- The next clean Inventory/API-style canary should confirm Engineer commits the
  route repair and closes the Dogfood ticket instead of ending in
  `circle_detected`.

## AD-219: Managed Background Capture Is Race-Safe

**Status:** Accepted
**Date:** 2026-05-21
**Owner:** Mars Harness maintainers

### Context

After GitHub Actions billing capacity was restored, the previously opaque
workflow failures became actionable CI evidence. The `v0.42.19` release
workflow rerun passed and `mars-harness release verify-assets --version
v0.42.19` found all required binaries and `checksums.txt`, but the current-main
CI run exposed a race in `internal/tools` under `go test ./... -race -count=1
-coverprofile=coverage.out -covermode=atomic`.

The race came from managed `shell_exec` background validation. While a
long-running target server stayed alive beyond the startup capture window,
stdout and stderr goroutines continued writing to `bytes.Buffer` values while
the tool assembled the initial output snapshot returned to the role.

### Decision

The background startup capture buffers are synchronized. The tool still returns
after the same startup window, still reports early exits as boot failures, and
still keeps the tracked process available for readiness probes and cleanup, but
snapshot assembly now reads stdout and stderr through the same lock used by the
capture writers.

### Consequences

- Race-detected CI can exercise managed background validation without failing on
  the output snapshot path.
- The live factory loop keeps the managed-background lifecycle that proved useful
  in API canaries without weakening process cleanup or probe guidance.
- Release workflow status must be checked again after infrastructure blockers
  clear because budget failures can hide real product defects behind missing
  logs.

## AD-220: Ticket Evidence Guard Preserves Path Case

**Status:** Accepted
**Date:** 2026-05-21
**Owner:** Mars Harness maintainers

### Context

The `v0.42.20` Release workflow published all binary assets, but the paired
main-branch CI run failed two `internal/tools` policy tests on Linux:
feature and enabler tickets could appear to move into `docs/tickets/done/`
without required evidence. The same tests passed locally on macOS.

The root cause was case sensitivity. The shared `shellFields` helper lowercased
shell commands so destructive-command checks could compare command tokens
simply. `checkShellTicketDoneEvidencePolicy` reused those lowercased tokens for
ticket file paths. On macOS, `T-001-ship.md` still resolved after becoming
`t-001-ship.md`; on Linux, the read failed and the policy skipped evidence
validation.

### Decision

Path-sensitive ticket lifecycle parsing now uses a case-preserving shell token
helper. Command-name matching for `git`, `mv`, and `cp` remains
case-insensitive, but ticket source paths keep their original case before
`Root.ResolvePath` and `os.ReadFile` inspect frontmatter.

### Consequences

- Linux CI and macOS local runs enforce the same ticket evidence requirement.
- Uppercase ticket IDs remain safe in shell-command lifecycle moves.
- Command safety checks can continue using normalized shell tokens without being
  reused for path-sensitive policy decisions.

## AD-221: Workspace Noise Does Not Reopen Product Tickets

**Status:** Accepted
**Date:** 2026-05-21
**Owner:** Mars Harness maintainers

### Context

A live `demo-6` deployed harness run made real Tetris product progress, moved
`T-001` to `docs/tickets/done/`, and committed the lifecycle move, but the
Engineer could not record its terminal disposition. Finder-created `.DS_Store`
metadata remained dirty after the ticket was done. The disposition gate treated
that metadata as ordinary uncommitted work, then the Engineer tried to commit
`.DS_Store`; rework policy blocked the commit because product mutations after a
done ticket require reopening the ticket. The role spent the remaining turn
budget in that contradiction and failed with `max_turns`.

### Decision

Generated target harnesses now add root `.gitignore` entries for `.DS_Store`,
`Thumbs.db`, and `Desktop.ini` during `init` and `upgrade`. Tool policy also
classifies those host OS metadata files as workspace noise for lifecycle and
disposition gates: dirty workspace noise does not block ticket moves into
`docs/tickets/done/` or terminal disposition, and `git_commit` rejects attempts
to commit only workspace noise with guidance to record the disposition when no
product work remains.

### Consequences

- Clean demo runs should not turn Finder or desktop metadata into product work.
- Completed product tickets no longer need to be reopened just to satisfy an OS
  metadata dirty-tree blocker.
- Agents still cannot commit `.DS_Store` or similar metadata; the files must be
  ignored, removed, or left outside product completion accounting.

## AD-222: Browser JavaScript Tickets Must Keep Target Shape

**Status:** Accepted
**Date:** 2026-05-22
**Owner:** Mars Harness maintainers

### Context

The `demo-tetris-12` live replay showed a planning-level failure before product
validation could help. The target brief said "Create Tetris using Phaser JS",
but CTO produced an ordinary feature ticket whose affected files and design
guidance described a fresh Go CLI with `cmd/phaser-tetris-demo/main.go` and
`go.mod`. Engineer then followed that ticket literally, creating a Go static
file server, a CDN-only Phaser HTML page, and a Phaser-related Go module
dependency. The problem was not ticket volume or intervention debt; the first
authoritative implementation contract had inherited the foundation repo's Go
shape instead of the deployed target's browser framework shape.

### Decision

CTO ticket creation is now target-shape aware for browser JavaScript briefs.
When README, the vision, the feature contract, or source HTML references Phaser
and the README does not explicitly name a Go backend, feature tickets cannot
prescribe Go CLI paths, `go.mod`, `cmd/*`, or Go module setup. Generated CTO
guidance instead points browser JavaScript targets toward `package.json`,
`index.html`, `src/*.js` or `src/*.ts`, tests, build config, `npm run build`,
and browser/runtime smoke evidence.

Engineer file writes also block Go module or `cmd/*.go` scaffolding for
Phaser-only briefs. Browser-framework completion detection now derives Phaser
from README/vision and HTML script tags, not only from an existing
`package.json`; completion requires a package manifest with local `phaser`
dependency, a deterministic build script, same-job build evidence, and no
obvious Phaser lifecycle defects.

### Consequences

- Fresh browser-game targets should start from a browser JavaScript package
  surface instead of a Go wrapper unless the brief asks for a Go backend.
- Bad technical ticket shape is caught at ticket creation before Engineer
  spends implementation cycles on the wrong architecture.
- CDN-only Phaser sketches stay visible as incomplete validation surfaces rather
  than passing static HTTP evidence.
- Legitimate Go-backed browser targets remain possible when the README states
  that backend requirement explicitly.

## AD-223: Browser Game Evidence Must Prove Product Behavior

**Status:** Accepted
**Date:** 2026-05-22
**Owner:** Mars Harness maintainers

### Context

The `demo-tetris-13` replay improved target shape: CTO created a Phaser/JS
ticket and Engineer produced `package.json`, `index.html`, and `src/main.js`.
However, the build script was only `echo 'Building Phaser Tetris Demo...'`, QA
accepted build and syntax evidence, and Dogfood reported success from HTTP 200
reachability plus file inspection. The target still contained only a movable
sample block, while the project brief requested complete playable Tetris.

### Decision

Browser-framework evidence now distinguishes real validation from file
delivery. A package build script that is only `echo`, `true`, `exit 0`, or an
equivalent no-op is treated as missing build evidence. QA, Security, and Dogfood
cannot approve browser-framework work from HTTP/build evidence alone; they must
also run a browser-product smoke or equivalent source/runtime assertion that
checks mounted UI state such as Phaser game/canvas behavior. Generated Engineer,
QA, and Dogfood guidance now states that HTTP 200 is file delivery, not
JavaScript correctness.

### Consequences

- Browser projects should stop passing review with placeholder build scripts.
- Dogfood reports should describe product behavior evidence, not only server
  liveness.
- The policy is generic to browser frameworks, while Phaser guidance remains
  specific enough to catch the live game failure class.

## AD-224: Release Waits For Generated Feature Scenario Coverage

**Status:** Accepted
**Date:** 2026-05-22
**Owner:** Mars Harness maintainers

### Context

The same `demo-tetris-13` replay completed one walking-skeleton ticket and
immediately reached release review. That was lifecycle progress, but it was not
project completion. The generated feature contract still contained uncovered
scenarios, and the active project brief still named a larger product. Without a
mechanical release gate, a target could version a thin scaffold as if the build
were finished.

### Decision

Dispatch now checks release-bound decisions against visible target work. If
ordinary product tickets remain open, release-bound dispatch routes Engineer
before Release Manager. If the generated target feature contract
`docs/features/F-001-product-walking-skeleton.md` still contains scenarios that
are not referenced by any done product ticket, release-bound dispatch routes CTO
for ticket shaping before version publication.

### Consequences

- The lifecycle continues draining product work instead of releasing after the
  first slice.
- Scenario coverage becomes the mechanical bridge between BDD planning and
  release readiness.
- Temporary targets without a remote can still record release blockers, but only
  after the generated product scenario schedule is covered.

## AD-225: Product Brief Capabilities Must Become Scenarios Before Ticketing

**Status:** Accepted
**Date:** 2026-05-22
**Owner:** Mars Harness maintainers

### Context

The `demo-tetris-14` replay exposed a planning gap after target-shape and
browser-evidence fixes. COO copied the explicit Tetris requirements into
Business Logic, but left the Scenario Schedule as generic starter headings:
"project brief becomes a visible product slice", "user can run or inspect the
first product behavior", and "product evidence is captured". CTO then created
one narrow grid ticket, explicitly out-of-scoping movement, rotation, line
clearing, scoring, game over, and restart. Engineer spent 36 turns on that thin
slice and eventually hit local-model context overflow. The lifecycle had
product-first motion, but it still lacked a mechanical path from complete
product brief to complete product build.

### Decision

Planning and ticket creation now check product capability coverage. When README,
active goals, or the product brief says the target product should include or
support explicit capabilities, the generated feature scenario surface must
represent those capabilities as concrete scenarios or list them under Descoped
Scenarios with reasons. COO cannot record completed planning while the scenario
schedule remains generic. CTO cannot create feature tickets from a
non-decomposed contract; it must route feedback to COO instead of turning the
first visible fragment into the whole implementation backlog.

Generated COO and CTO guidance now names this rule: BDD defines the whole
feature, walking skeleton is only the delivery strategy, and product-capability
lists from the brief must become scenario coverage before implementation
tickets are created.

### Consequences

- A "complete app" brief should become a scenario schedule for the app's real
  capabilities rather than a single scaffold or grid ticket.
- CTO remains free to create one ticket at a time, but only after the full
  feature contract has concrete future scenarios for the remaining product
  behavior.
- The policy is generic to explicit "include/support/features" capability
  language, so Tetris is one validation target rather than the only product
  shape the factory optimizes for.

## AD-226: Browser Framework Completion Must Fail Before QA On Broken Module Graphs

**Status:** Accepted
**Date:** 2026-05-22
**Owner:** Mars Harness maintainers

### Context

The `demo-tetris-15` replay confirmed the planning fixes: CEO, COO, and CTO
created a product-specific Phaser Tetris feature and one ordinary product
ticket without intervention-debt flooding. QA also caught the first broken
Phaser implementation and routed rework. The rework still failed, though:
Engineer marked the ticket done after `node --check`, `npm run build`, and grep
evidence even though `src/main.js` imported `TetrisGame` from `src/game.js`,
`src/game.js` did not export it, and the Phaser scene callbacks were bound to a
wrapper object that used scene APIs through the wrong `this`. QA policy then
blocked approval for missing browser-product smoke, but the reviewer spent the
remaining turn budget trying to start the app and ended in `max_turns`.

### Decision

Browser-framework completion now shifts that failure left into Engineer policy.
For package-managed browser framework targets, syntax-only build scripts such
as `node --check ...` are treated like no-op build scripts because they do not
prove bundling, module resolution, or browser lifecycle correctness. Engineer
cannot populate ticket evidence, move a browser-framework ticket to done, or
record successful disposition until the same job has successful build evidence
and one browser-product smoke or equivalent source/runtime assertion. The
source policy also detects local named imports whose target module does not
export the imported symbol, classic script tags that load ES module syntax, and
Phaser wrapper patterns that bind scene callbacks to the wrong context.

### Consequences

- QA receives fewer already-broken browser-game tickets and can spend review
  turns on product behavior instead of first-pass module graph failures.
- `node --check` remains useful for intentionally static JavaScript, but it no
  longer satisfies package-managed browser-framework build evidence by itself.
- The rule stays generic to browser frameworks while recording concrete Phaser
  lifecycle checks discovered in the live Tetris build loop.

## AD-227: Product Progress Max-Turns Continue The Active Ticket Once

**Status:** Accepted
**Date:** 2026-05-22
**Owner:** Mars Harness maintainers

### Context

The `demo-tetris-16` replay made visible product progress after the
browser-framework evidence shifts: CEO, COO, and CTO created product-specific
Tetris planning and an ordinary Phaser-shaped product ticket; Engineer claimed
the ticket, wrote source files, committed implementation work, and pushed. The
role then exhausted its turn budget before ticket evidence, ticket closure, and
`job_disposition_record`, leaving the lifecycle blocked even though there was a
clear active product ticket to continue. This is different from
intervention-debt flooding: the factory was building the product, but the
runtime treated partial product progress plus max turns as a dead end.

The same replay also showed that browser-framework guards need to catch module
graph failures directly. The implementation used `import`/`export` syntax from
`src/main.js` while `index.html` loaded it through a classic script tag, and
`src/game.js` referenced `createRandomTetromino` and `rotateTetromino` from
another local module without importing them. Those defects should be found
before a role spends review budget trying to open a blank browser app.

### Decision

Dispatch-mode failure handling now treats Engineer `max_turns` with an
ordinary in-progress product ticket as a bounded continuation case. When the
failed job is not already a ticket-gate repair or product-continuation job, the
harness enqueues one same-role `product_continuation` job. The continuation
prompt tells Engineer to inspect the current ticket, latest commits, and dirty
files; fix only remaining product/build/validation/lifecycle gaps; update
evidence; move the ticket to done; commit; push when possible; and record
`job_disposition_record`. Recursive continuations are blocked, preserving the
rule that runtime failures do not become unbounded autonomous loops.

Browser-framework source inspection now also blocks local-module symbols that
are exported by one local file but used by another module without an import.
Together with the existing classic-script module-loading and missing named
export checks, this catches the common "blank app after HTTP 200" failure class
before QA or Dogfood are asked to approve it.

### Consequences

- Product progress is no longer discarded just because the first Engineer run
  ran out of turns after useful commits.
- Continuation remains bounded and product-scoped rather than routing runtime
  failures through Orchestrator or target intervention-debt tickets.
- Browser-app completion checks stay generic to module graph health, while the
  Phaser Tetris replay remains the concrete evidence that motivated them.

## AD-228: Browser Validation Procedure Mistakes Must Not Freeze Product Repair

**Status:** Accepted
**Date:** 2026-05-22
**Owner:** Mars Harness maintainers

### Context

The `demo-tetris-17` replay verified that AD-227 worked mechanically: the
first Engineer job reached `max_turns` with an active product ticket, and the
harness immediately enqueued a bounded product-continuation Engineer job
instead of stopping. The continuation then showed why the original role burned
so many turns. Engineer had run `node --check index.html`, which is a
validation-procedure mistake because Node's syntax checker is for JavaScript,
not HTML. The runtime treated the failing HTML syntax probe as an unexpected
product runtime failure and froze commits, shell commands, and disposition
until that exact impossible command passed. Repeated policy feedback inflated
the local-model prompt until it exceeded the 32k context window.

The same run showed that Phaser target-shape guidance still arrived too late.
Engineer wrote a CDN-only `index.html`, a `package.json` with a local Phaser
dependency but no deterministic build script, and repo-root scratch validation
files such as `test-phaser.js` and `test-phaser.html`. Completion policy would
eventually block those artifacts, but the role spent the turn budget learning
the constraints through failure.

### Decision

`node --check` against `.html` or `.htm` is now blocked before execution and is
also classified as a validation-procedure failure if an older path records the
result. It no longer creates an unresolved runtime validation failure that must
be repaired by rerunning an impossible command.

For Phaser browser targets, Engineer file writes now block invalid package and
entrypoint shapes earlier: `package.json` must declare the local `phaser` npm
dependency and include a deterministic build script that can fail on broken
source, and `index.html` cannot load Phaser from a CDN-only script tag. Root
scratch validation files with names such as `test-phaser.js` or
`test-phaser.html` are blocked so validation evidence does not become product
noise.

### Consequences

- A validation-procedure mistake no longer poisons the rest of the product
  delivery job or forces the local model into a context-overflow loop.
- Browser-framework constraints fail at the first package/entrypoint write
  instead of only at ticket evidence or QA approval.
- Static projects can still use `node --check` for JavaScript files; browser
  entrypoints must be proven through build/static/browser smoke rather than
  treating HTML as JavaScript.

## AD-229: Product-Specific BDD Vocabulary Must Not Trip Starter Guards

**Status:** Accepted
**Date:** 2026-05-22
**Owner:** Mars Harness maintainers

### Context

The `demo-tetris-18` replay advanced beyond CEO and into COO with the patched
browser validation rules. COO rewrote the generated
`docs/features/F-001-product-walking-skeleton.md` into a Tetris-specific
contract with scenarios for visible playfield, falling tetrominoes, movement,
rotation, line clearing, scoring, game over, and restart. The runtime still
blocked the planning disposition because the Business Logic section reused
standard BDD vocabulary from the generated scaffold: "Product rules, workflow
branches, state transitions..." In this case the phrase was no longer starter
text; it was part of a real product-specific paragraph.

### Decision

The COO starter-placeholder guard now rejects actual scaffold markers such as
"starter contract is seeded" and "replace placeholder nouns", but it does not
treat durable BDD vocabulary as placeholder text by itself. Product-specific
feature contracts may still explain product rules, workflow branches, state
transitions, scoring decisions, routing rules, and user-visible outcomes. The
separate product-capability coverage guard remains responsible for proving the
scenario schedule covers explicit README/goal capabilities before CTO ticketing.

### Consequences

- COO can use the foundation glossary's BDD language without being trapped in a
  false planning loop.
- The starter guard remains strict against real generated scaffold text.
- Product capability coverage stays mechanical, so relaxing the phrase match
  does not permit generic starter scenarios to reach ticket creation.

## AD-230: Browser Build Repair Must Stay Compact And Config-Safe

**Status:** Accepted
**Date:** 2026-05-22
**Owner:** Mars Harness maintainers

### Context

The `demo-tetris-19` replay reached the desired product-delivery lane: CEO and
COO completed, CTO created an ordinary Tetris ticket, and Engineer claimed it
and wrote a Phaser/Vite app skeleton. The next failure was more subtle.
Engineer introduced a `vite.config.js` shape that caused `npm run build` to
evaluate Phaser/browser runtime code in Node, producing `ReferenceError:
window is not defined`. The unresolved-build guard correctly blocked unrelated
shell commands, but every blocked retry repeated a long failing-output excerpt.
That loop consumed the local model context and the Engineer job ended with
`llm_unreachable`/context overflow before the product build was completed.

### Decision

For Phaser briefs, `file_write` now rejects `vite.config.js` and
`vite.config.ts` content that imports Phaser, browser runtime code, or local
`src/*` game modules. Vite config must stay limited to Vite/plugin
configuration; Phaser and game modules belong in browser entrypoints.

Unresolved test/build guardrail guidance now includes only a bounded compact
failing-output excerpt. The full failure remains visible in telemetry/tool
results, but repeated policy blocks no longer replay long build output into the
agent prompt.

### Consequences

- The common Phaser/Vite `window is not defined` build trap is blocked at
  config-write time instead of after a failed build loop.
- Repeated repair-lane guardrails preserve the invariant that failing builds
  must be repaired before side paths continue, while reducing context pressure.
- The rule remains product-shape specific: ordinary Vite plugin imports are
  allowed, and browser runtime code still belongs in `src/*` entrypoints.

## AD-231: Test Repair Lanes Must Allow Real Test Files

**Status:** Accepted
**Date:** 2026-05-22
**Owner:** Mars Harness maintainers

### Context

The `demo-tetris-20` replay advanced through CEO, COO, and CTO into a
product-specific Phaser Tetris implementation. Engineer wrote a Vite/Phaser app
and hit a legitimate integration-test failure: Jest could not find the
`jest-environment-jsdom` package. The unresolved test/build guard correctly
blocked unrelated shell probes and switching to build validation, but it also
blocked a focused write to `tests/integration/playfield.test.js`. That
contradicted the repair-lane guidance, which explicitly allows source, tests,
fixtures, and package/build config edits before rerunning same-lane validation.

### Decision

Engineer test/build repair writes now recognize real test files as first-class
repair targets. Files under `test/` or `tests/`, Go `_test.go` files, and
conventional JavaScript/TypeScript `*.test.*` or `*.spec.*` files are eligible
same-lane repair writes. Scope limits still apply when the failing validation
command recorded a narrow package or directory scope, and root scratch
validation probes remain blocked by the separate scratch-file policy.

### Consequences

- The harness can repair failing browser integration tests without forcing the
  agent into ticket churn, helper scripts, or unrelated validation commands.
- The policy matches its user-facing guidance: real tests are valid repair
  surface, while docs, ticket evidence, commits, and side probes still wait
  for successful validation.
- The allowance stays bounded to durable test locations and conventional test
  filenames, preserving protection against ad hoc root scratch files.

## AD-232: Browser Game Builds Must Be Real Bundles With Clean Output

**Status:** Accepted
**Date:** 2026-05-22
**Owner:** Mars Harness maintainers

### Context

The `demo-tetris-21` replay confirmed the lifecycle now reaches product code
quickly: CEO, COO, and CTO completed in under four minutes, CTO created an
ordinary visible-playfield ticket, and Engineer wrote a Phaser implementation.
The job still failed to complete. Engineer first tried a copy-only build script
(`mkdir -p dist && cp src/index.html dist/index.html && echo ...`) and later
used `live-server`, syntax checks, and static source reads instead of a real
browser bundle plus product smoke. The code also left `dist/index.html`
unignored, causing the automatic continuation job to fail workspace hygiene
before it could repair the ticket. Source inspection caught some module-entry
issues, but it did not catch `GameScene.js` extending `Phaser.Scene` without
importing Phaser in that module.

### Decision

Phaser/browser-framework package writes and completion checks now treat
copy-only build scripts as no-op validation. Build scripts made only of
`mkdir`, `cp`, `copy`, `rsync`, `touch`, static-server commands, `echo`, or
syntax-only checks do not count as deterministic browser build evidence.
Generated target `.gitignore` now includes common dependency and build output
directories (`node_modules/`, `dist/`, `build/`, `coverage/`, `.vite/`) from
initialization so first-run browser builds do not poison continuation hygiene.
Browser-framework source inspection also blocks modules that reference
`Phaser.*` or `extends Phaser.Scene` without importing or defining Phaser in
that same module.

### Consequences

- Phaser targets are pushed toward Vite or an equivalent bundler instead of
  file-copy sketches that hide module graph failures.
- Continuation jobs can start from a clean ignored-output policy after a build
  creates `dist/`.
- Missing Phaser imports are detected before ticket evidence, done moves, QA
  handoff, or review approval.
- The rules remain general to browser-framework delivery while using the
  Tetris replay as concrete evidence for the failure mode.

## AD-233: Browser Smoke And Planning Scope Need Executable Off-Ramps

**Status:** Accepted
**Date:** 2026-05-22
**Owner:** Mars Harness maintainers

### Context

The `demo-tetris-22` replay improved the previous browser-game failure: the
generated target ignored `dist/` and `node_modules/`, Engineer used a real
Phaser/Vite package with `vite build`, and Phaser imports were present in the
scene module. The run still failed to finish the first ticket. Engineer spent
two max-turn jobs trying to satisfy browser-product smoke, but the guardrail
only said to use Playwright/Puppeteer or an equivalent source/runtime
assertion. The model tried repo-root scratch files and a `node -e` eval body in
argv mode; scratch files were correctly blocked, but argv validation treated
JavaScript semicolons as shell syntax.

The same replay exposed a planning regression. The project brief explicitly
required line clearing, scoring, game over, and restart behavior, but COO's
feature contract put those capabilities under `Out of Scope` while claiming
`Descoped Scenarios: None`. That let CTO create only a visible-playfield ticket
and made the lifecycle look productive while drifting away from the complete
Tetris build.

### Decision

`shell_exec` argv validation now treats language eval payloads after flags such
as `node -e`, `node --eval`, and `python -c` as code arguments rather than shell
syntax. Browser-framework completion guidance names a concrete fallback:
when Playwright/Puppeteer is unavailable, Engineer can run a bounded `node -e`
source/runtime assertion that checks module entry, `new Phaser.Game`, mount
container/canvas evidence, and Phaser imports, then prints
`browser smoke: Phaser canvas #game new Phaser.Game` so the session records
browser-product smoke.

Product capability coverage now also rejects feature contracts that put
explicit brief requirements under generic `Out of Scope` text unless those
requirements are represented under `Descoped Scenarios` with rationale.
Required product behavior must either become in-scope scenario coverage or be
deliberately descoped; it cannot silently disappear before ticketing.

### Consequences

- Browser game jobs get a concrete smoke-validation off-ramp without lowering
  the quality bar to `node --check`, grep-only evidence, or HTTP 200.
- Repo-root scratch validation files remain blocked, but `node -e` source
  assertions no longer fail because JavaScript code contains semicolons.
- The first-run plan stays aligned to explicit product requirements, reducing
  the risk that a complete-app brief becomes a one-ticket scaffold build.
- The rule is generic to product briefs: Tetris exposed it, but any deployed
  target with explicit required capabilities receives the same protection.

## AD-234: Browser Bundles Must Not Externalize Runtime Frameworks

**Status:** Accepted
**Date:** 2026-05-22
**Owner:** Mars Harness maintainers

### Context

The `demo-tetris-23` replay validated the planning-scope fix: COO produced an
eight-scenario Tetris feature contract covering visible playfield, falling
tetrominoes, movement, rotation, line clearing, scoring, game over, and
restart. Engineer then created and committed a real Phaser/Vite package for
the first playfield scenario. Two remaining issues appeared. First, Engineer
still did not run the smoke assertion after build success; it repeatedly tried
ticket evidence, done moves, and successful disposition while the policy asked
for browser-product smoke. The guardrail named `node -e`, but it did not show a
literal tool call. Second, the generated `vite.config.js` added
`rollupOptions.external: ['phaser']`. That can let `vite build` complete while
leaving a bare `phaser` import for the browser, so build evidence no longer
proves the deployed bundle can load the local dependency.

### Decision

Browser-product smoke guardrails for Phaser now include a literal
`shell_exec argv ["node","-e", ...]` command. The command reads `index.html`
and `src/main.js`, checks the mount surface, verifies `new Phaser.Game`, checks
for Phaser import usage, and prints
`browser smoke: Phaser canvas #game new Phaser.Game` so the runtime records the
same-job browser smoke success.

Phaser Vite config writes and existing-source completion checks now reject
config that externalizes `phaser` through `rollupOptions.external` or an
equivalent external entry. Phaser must stay in the production browser bundle so
`npm run build` validates the module graph the browser will actually load.

### Consequences

- The model receives a copy-pastable next action instead of another abstract
  browser-smoke reminder.
- `vite build` can no longer be satisfied by a configuration that removes the
  main runtime framework from the browser bundle.
- Source inspection catches already-written Vite externalization before ticket
  evidence, lifecycle closure, or review approval.
- The rule generalizes from Tetris: package-managed browser frameworks should
  prove bundled runtime dependencies, not merely compile a shell around them.

## AD-235: Product Tickets Must Start At The Earliest Uncovered Scenario

**Status:** Accepted
**Date:** 2026-05-22
**Owner:** Mars Harness maintainers

### Context

The `demo-tetris-24` replay exercised the literal smoke guidance and
Vite-externalization guardrails against a fresh Phaser Tetris target. COO
produced a product-specific feature contract, but it grouped the complete
Tetris brief into four scenarios and set the active plan's current failing
scenario to keyboard controls before any playfield ticket existed. CTO then
created `T-001` for `F-001-S002`, skipping the earliest uncovered scenario.

Engineer made real product files, but the implementation still did not finish:
`package.json` used `python3 -m http.server 18081` as its app server, colliding
with Mars Harness inference/runtime ports; `src/index.html` loaded Phaser from
a CDN script under a nested HTML path; and `src/main.js` referenced global
`Phaser` without importing it and constructed `new Phaser.Game` inside a scene
callback. The job exhausted 48 turns and ended with context overflow rather
than a verified done ticket.

### Decision

Feature `ticket_create` now enforces scenario order mechanically. A feature
ticket must include the earliest uncovered scenario from the feature contract;
later scenarios may be batched only when the same ticket also includes that
earliest scenario. This prevents a stale or over-eager active-plan cursor from
skipping the first unproven product slice.

Phaser package and source policies now block several live-demo failure modes at
write time: package runtime scripts cannot use Mars Harness reserved ports
`18080`-`18089`, Phaser package scripts cannot use static source servers such
as `python3 -m http.server` in place of Vite dev/preview, nested HTML entry
points cannot load Phaser from CDN script tags, Phaser source that references
`Phaser.*` must import or define Phaser in that module, and `new Phaser.Game`
must be created exactly once from the browser entrypoint rather than inside
scene callbacks.

### Consequences

- Fresh product delivery starts with the first uncovered BDD scenario even when
  the active plan cursor drifts.
- Target dev servers avoid local inference/runtime port collisions.
- Invalid Phaser/browser app shapes are rejected before they accumulate into a
  long Engineer context-overflow loop.
- Static HTTP evidence remains available for intentionally static projects,
  but package-managed Phaser apps are steered toward Vite build/dev/preview
  paths that actually validate npm module loading.

## AD-236: Wrapped Briefs And Browser Evidence Keep Planning And Closure Honest

**Status:** Accepted
**Date:** 2026-05-22
**Owner:** Mars Harness maintainers

### Context

The `demo-tetris-25` replay confirmed the earliest-scenario ticket gate: CTO
first tried to create a keyboard-controls ticket, the runtime rejected it, and
the next CTO attempt created `T-001` for `F-001-S001` instead. Engineer then
claimed the visible-playfield ticket and produced a much better Phaser/Vite
shape with local `phaser`, Vite scripts on app ports, source under `src/`, and
one top-level `new Phaser.Game`.

Two issues remained. First, the target README wrapped one required capability
sentence across two physical lines: "visible playfield, falling tetrominoes,"
then "keyboard movement and rotation, line clearing, scoring, game over, and
restart". The brief parser treated the single newline like a sentence boundary,
so COO could still leave later wrapped capabilities under `Out of Scope`.
Second, after Engineer committed implementation files, the post-validation
shell convergence gate blocked follow-up `npm run build` and browser-product
smoke commands even though browser-framework completion explicitly requires
that evidence before ticket closure.

### Decision

Brief capability extraction now treats single newlines as wrapped prose and
only treats blank-line runs as sentence boundaries. Product capability checks
therefore see the whole requirement sentence before COO handoff or CTO
ticketing, including wrapped lists such as Tetris line clearing, scoring, game
over, and restart behavior.

Engineer post-validation convergence still blocks exploratory shell churn after
a clean implementation commit, but it permits the two missing browser-framework
validation steps needed to close a product ticket: the package build command
when same-job build evidence is absent, and the browser-product smoke/source
runtime assertion when build evidence exists but mounted product-state evidence
is absent.

### Consequences

- Wrapped README briefs are evaluated as operator intent, not accidental
  paragraph fragments.
- Browser-framework tickets can converge after an implementation commit instead
  of looping on a guardrail that demands evidence while blocking the evidence
  commands.
- The exception is narrow: it applies only to missing required browser build or
  product-smoke evidence, not broad discovery or unrelated post-validation
  shell activity.

## AD-237: Product Capability Guards Ignore Validation-Evidence Clauses

**Status:** Accepted
**Date:** 2026-05-22
**Owner:** Mars Harness maintainers

### Context

The `demo-tetris-26` replay proved AD-236 fixed the wrapped-line issue: COO kept
visible playfield, falling tetrominoes, keyboard movement and rotation, line
clearing, scoring, game over, and restart in the feature schedule. The next
planning loop came from a different sentence in the README: "include enough
build or smoke evidence to prove the game mounts and plays." The capability
guard treated that validation/evidence clause as another product capability,
so COO repeatedly added or rewrote a build/smoke scenario and still could not
complete because the matcher held onto tail fragments such as "plays".

The same replay showed a normal wording mismatch. A feature scenario that said
"keyboard controls" with left/right/down/rotate keys was a reasonable coverage
for the brief's "keyboard movement and rotation", but the keyword matcher
looked for the literal movement stem.

### Decision

Product capability extraction now ignores brief fragments that are plainly
about validation evidence rather than user-visible product behavior: evidence,
smoke test, validation evidence, proof/prove/proving, verified-by wording, and
build artifacts. The splitter also drops short validation tails such as "plays"
or "mounts" after one of those evidence fragments has been skipped.

Capability coverage also treats keyboard controls, left, right, or down input
language as movement coverage. Rotation remains separately required when the
brief names rotation.

### Consequences

- Operators can ask for build or smoke evidence in the project brief without
  accidentally expanding product scope into a validation-only feature.
- Planning remains strict about real product capabilities while avoiding
  repeated guardrail loops over evidence phrasing.
- The matcher gets a small, domain-agnostic control synonym that helps games,
  editors, and other keyboard-driven software without making unrelated product
  behavior pass silently.

## AD-238: Out-Of-Scope Capability Checks Must Respect Advanced-Only Qualifiers

**Status:** Accepted
**Date:** 2026-05-22
**Owner:** Mars Harness maintainers

### Context

The next `demo-tetris-27` replay cleared the validation-evidence capability
loop and moved the active plan back to `F-001-S001`. COO produced a
product-specific scenario schedule and repeatedly repaired genuine out-of-scope
mistakes. It then got stuck on a false positive: the contract already included
line clearing and scoring scenarios, but its Out of Scope section said
"Advanced scoring or game modes beyond basic line clearing." The guard matched
the words "scoring" and "line clearing" inside that advanced-only qualifier and
treated the basic capabilities as descoped.

### Decision

Out-of-scope capability checks are now line-aware. A required capability only
counts as out of scope when the matching out-of-scope line actually describes
that base capability as excluded. Lines that use an explicit qualifier such as
"advanced ... beyond basic ..." are treated as excluding only the advanced
extension, not the basic behavior already covered by the scenario schedule.

### Consequences

- Agents can keep advanced variants, modes, polish, or stretch mechanics out of
  scope without accidentally descoping the core capability.
- The guard still blocks direct descoping of required product behavior such as
  restart when it is listed in Out of Scope without a descoped scenario reason.
- Planning loops lose another false-positive source while preserving the
  product-brief coverage invariant.

## AD-239: Circle-Detected Product Progress Gets A Bounded Continuation

**Status:** Accepted
**Date:** 2026-05-22
**Owner:** Mars Harness maintainers

### Context

The `demo-tetris-28` replay progressed past planning: CEO, COO, and CTO
completed, CTO created exactly one ordinary `F-001-S001` product ticket, and
Engineer claimed it. Engineer then wrote real Phaser/Vite package and source
files, while write-time guards correctly blocked invalid Vite imports, missing
DocSync metadata, and a broken Phaser lifecycle helper. The job ended as
`circle_detected` after repeated guardrail blocks, leaving an active product
ticket and uncommitted product files.

The runtime correctly kept the failure out of target intervention debt and did
not route it through Orchestrator. But it also stopped without a bounded
follow-up job, even though the existing product-continuation lane already
handles the same situation for `max_turns` after useful product progress.

### Decision

Engineer `circle_detected` with an ordinary in-progress product ticket now
enqueues one bounded `product_continuation` Engineer job, using the same
non-recursive continuation guard as `max_turns`. The continuation prompt tells
Engineer to inspect the active ticket, latest commits, and dirty files, repair
only remaining product/build/validation/lifecycle gaps, update evidence, move
the ticket to done when acceptance is met, commit, push when possible, and
record `job_disposition_record`.

### Consequences

- Repeated guardrail loops during useful product implementation no longer end
  the lifecycle with dirty partial product work and no next job.
- Runtime failures still do not become intervention-debt tickets or Orchestrator
  loops.
- Recursive continuations remain blocked, so a second loop boundary becomes
  operator-visible evidence for the next source fix.

## AD-240: Bootstrap Ticketing And Browser Smoke Evidence Must Keep The Factory Moving

**Status:** Accepted
**Date:** 2026-05-22
**Owner:** Mars Harness maintainers

### Context

The `demo-tetris-29` replay got further than prior runs. CEO and COO completed
product-specific planning, CTO created an ordinary product ticket, Engineer
implemented a real Phaser/Vite playfield slice, build and smoke validation ran,
and the ticket reached `done`. Two new sources of waste appeared after that
progress:

- CTO still seeded only the first scenario ticket, so the factory had no ready
  backlog for the remaining falling-piece, controls, scoring, game-over, and
  restart scenarios.
- QA copied or interpreted browser-smoke evidence as shell syntax, hit a
  helper/escaping failure, and routed "the implementation is correct but the
  smoke test is wrong" back to Engineer as implementation rework.

### Decision

Bootstrap technical planning now supports a small backlog batch. Once a feature
scenario is already covered by any ordinary ticket in backlog, in-progress,
in-review, or done, `ticket_create` may advance to the next uncovered scenario.
This lets CTO create one to three ordered product tickets during fresh
bootstrap instead of leaving the factory with a single active slice.

Phaser browser-smoke guidance now uses string checks rather than JSON-escaped
regular expressions, so evidence copied into tickets remains easier for QA to
rerun and reason about. QA changes-requested dispositions are also blocked when
the feedback says the implementation is correct and the problem is the smoke
helper or validation wording while browser-framework source inspection is
clean. That situation is a foundation/dogfood validation finding or an
approval-with-corrected-evidence path, not target Engineer rework.

### Consequences

- Fresh projects get enough product backlog to keep building after the first
  visible slice without broad overplanning.
- Browser validation evidence is less likely to become a quoting trap that
  causes false rework.
- QA can still request changes for real source, build, scenario, or acceptance
  failures; only "the test is wrong but the implementation is correct" loops
  are intercepted.

## AD-241: QA Validation Setup And CTO Batch Handoffs Must Not Stall Product Builds

**Status:** Accepted
**Date:** 2026-05-22
**Owner:** Mars Harness maintainers

### Context

The `demo-tetris-30` replay showed material improvement but exposed the next
factory bottlenecks. CEO and COO produced product-specific planning, CTO
created a real Phaser ticket, and Engineer shipped a Vite/Phaser skeleton after
one bounded continuation. The run still wasted turns because CTO handed off
after covering only `F-001-S001`, leaving the rest of Tetris unticketed, and QA
requested Engineer rework after its own smoke setup failed against a stopped
`localhost:5173` server even though `npm run build` passed and source inspection
was clean.

### Decision

CTO implementation handoff is now mechanically blocked when a feature has
multiple early scheduled scenarios but ordinary tickets cover fewer than the
first two or three. CTO may satisfy this with separate ordered tickets or one
bounded grouped ticket when adjacent early scenarios are naturally the same
walking skeleton slice.

QA changes-requested routing is also narrowed for browser-framework projects.
When build evidence exists and source inspection has no framework lifecycle
findings, a dev-server setup failure, stopped localhost probe, or malformed
smoke helper cannot be routed to target Engineer as `implementation_rework`.
QA must rerun the managed smoke/source assertion, approve with corrected
evidence, or record a foundation/dogfood validation finding.

### Consequences

- Fresh project builds have enough early product backlog to continue after the
  first visible slice.
- QA can still block real browser-framework defects, missing build evidence, or
  missing product-smoke evidence.
- QA-owned validation setup failures become foundation evidence rather than
  target backlog churn.

## AD-242: Capability Category Prefixes Are Not Product Requirements

**Status:** Accepted
**Date:** 2026-05-22
**Owner:** Mars Harness maintainers

### Context

The `demo-tetris-31` replay improved planning quality: COO produced a full
seven-scenario Tetris feature contract. The run then stalled because the
capability extractor read strategy prose from the generated product vision as
the literal requirement "all core Tetris mechanics: visible playfield grid".
The feature schedule already covered "visible playfield grid"; the false
requirement came from preserving the category prefix before the colon.

### Decision

Brief capability extraction now strips category prefixes before colon-delimited
lists when the prefix is a generic grouping label such as mechanics,
capabilities, features, behaviors, or behaviours. The individual list items
remain first-class product capabilities; the category label itself does not
need a scenario.

### Consequences

- COO and CTO can use normal strategy prose such as "all core mechanics:
  grid, falling pieces, controls" without creating phantom requirements.
- Capability coverage still requires each listed behavior to appear in the
  scenario schedule or be explicitly descoped.

## AD-243: Engineer Jobs Close One Product Ticket Before Claiming The Next

**Status:** Accepted
**Date:** 2026-05-22
**Owner:** Mars Harness maintainers

### Context

The `demo-tetris-32` replay got further than earlier runs: CEO, COO, and CTO
produced product-specific planning, CTO seeded a small Tetris ticket batch, and
Engineer completed real Phaser/Vite work for `T-001` and `T-002`. The run then
showed two adjacent lifecycle defects. First, the CTO batch gate tempted the
model into creating a grouped `T-003` that re-covered `F-001-S001` and
`F-001-S002` after those scenarios already had tickets, instead of creating
only the next uncovered scenario. Second, after moving `T-002` to
`docs/tickets/done/`, Engineer tried to claim `T-003` before committing the
`T-002` lifecycle move and recording a QA handoff. The guardrails blocked the
new claim repeatedly, and the job ended at `max_turns` despite real product
progress.

### Decision

Feature `ticket_create` now rejects already-covered BDD scenario IDs in a new
ordinary feature ticket. Once an earlier ordinary ticket in backlog,
in-progress, in-review, or done covers `F-001-S001`, a later ticket cannot
include that scenario again unless it is linked as explicit dependent work by a
separate mechanism. The policy feedback names the covered scenarios and directs
CTO to create the next uncovered scenario ticket only, or group it with later
uncovered adjacent scenarios.

Engineer ticket completion is also single-ticket per job. A successful
in-progress-to-done ticket move is recorded in the role session. The final
lifecycle commit is allowed even when more backlog tickets exist, but further
product mutation or `git mv` claims are blocked until Engineer commits the
lifecycle move, pushes when a remote exists, and records
`job_disposition_record` with `next_need: qa_review`.

### Consequences

- CTO can still seed a useful early backlog batch, but cannot satisfy the batch
  gate by duplicating scenarios already covered by earlier tickets.
- Engineer jobs stop at a clear review boundary after one ticket instead of
  chaining into the next product ticket and losing the completed work to turn
  budget exhaustion.
- The factory keeps product progress while preserving review evidence between
  slices.

## AD-244: Browser Validation Helpers Are Not Product Framework Source

**Status:** Accepted
**Date:** 2026-05-22
**Owner:** Mars Harness maintainers

### Context

The `demo-tetris-33` replay confirmed the previous fixes: CTO created a clean
two-ticket early batch without duplicate scenario coverage, and Engineer
started a real Phaser/Vite implementation for `T-001`. The next loop appeared
after the browser product-smoke assertion failed. Engineer attempted to create
a root helper file named `validate-phaser.js` that inspected `src/main.js` for
`new Phaser.Game`, but the Phaser source-shape guard treated the helper itself
as product app source because the helper contained the probe string. That left
the job blocked by validation-helper policy rather than product code.

### Decision

Browser-framework source inspection now distinguishes validation/probe helper
paths from product source. Root helpers such as `validate-phaser.js`, files
with smoke or probe names, `scripts/*` helpers, and conventional test/spec
paths are allowed to contain inspection strings such as `new Phaser.Game`
without triggering Phaser lifecycle checks. Product source under `src/`,
entrypoints, package scripts, and Vite config still receive the same lifecycle,
import, bundle, and module-graph checks.

AD-260 later narrows new root helper creation because the live loop showed
they become ticket-closeout clutter. The source-inspection exemption remains
for existing helpers and intentional durable helpers under `scripts/` or
`tests/`.

### Consequences

- Engineers can repair or explain browser-smoke failures with small helper
  files when inline `node -e` becomes unwieldy.
- Product-source guardrails remain strict for shipped app files.
- Browser validation helper mistakes stay foundation/runtime evidence instead
  of becoming false product implementation defects.

## AD-245: Product Capabilities Must Be Visible In The Scenario Outline

**Status:** Accepted
**Date:** 2026-05-22
**Owner:** Mars Harness maintainers

### Context

The `demo-tetris-34` replay regressed from the fuller `demo-tetris-31` feature
contract shape. COO represented visible playfield, falling tetrominoes,
keyboard movement and rotation, line clearing, scoring, game over, and restart
inside one broad "user can run or inspect the first product behavior"
scenario. Although the scenario body mentioned the requested mechanics, the
scenario schedule and headings gave CTO no clean product slices to turn into an
ordered backlog.

### Decision

Planning completion now requires explicit product brief capabilities to appear
in the scenario schedule entries or scenario headings, or to be listed under
Descoped Scenarios with reasons. A feature contract cannot hide multiple
requested capabilities inside a single broad runnable/inspectable scenario body
and then hand off to CTO as if the product backlog were decomposed.

### Consequences

- COO must expose product mechanics as ticketable BDD slices before CTO starts
  ticketing.
- CTO receives a clearer scenario outline and can create a small ordered
  backlog instead of one oversized product ticket.
- Validation/evidence prose can still live in scenario bodies and evidence
  sections, but it cannot replace product decomposition.

## AD-246: CTO Handoff Counts Product Scenarios, Not Process-Only Scenarios

**Status:** Accepted
**Date:** 2026-05-22
**Owner:** Mars Harness maintainers

### Context

In `demo-tetris-34`, after CTO created the first ordinary product ticket, the
handoff gate kept demanding `F-001-S003`. That scenario was not another Tetris
mechanic; it described keeping product evidence ahead of governance expansion
and intervention debt. The gate therefore created a deterministic loop: CTO
tried to satisfy a process-only scenario as if it were product implementation,
then hit ticket-shape and disposition blocks.

### Decision

The CTO implementation handoff gate now derives its required early batch from
scenarios that cover product brief capabilities. Evidence-only, governance,
telemetry, intervention-debt, or wider-automation ordering scenarios do not
force target implementation tickets once early product scenarios are ticketed.
When no explicit capability list exists, the gate falls back to the first two
or three scheduled scenarios so generic projects still get a small backlog.

### Consequences

- Product implementation can begin after real product scenarios are ticketed.
- Evidence and governance scenarios stay documented, but they do not become
  fake target backlog work.
- The factory still protects against a single planning-only ticket handoff
  because product-capability scenarios remain required before Engineer runs.

## AD-247: CTO Ticket Creation Recovers Pending Scenario Batches

**Status:** Accepted
**Date:** 2026-05-22
**Owner:** Mars Harness maintainers

### Context

The next `demo-tetris-35` loop confirmed that COO could now produce a good
product scenario outline and that CTO could create the first product ticket.
The handoff gate correctly asked CTO to add `F-001-S002` and `F-001-S003`
before Engineer handoff. CTO then repeatedly retried `ticket_create`, but the
tool arguments omitted the `bdd_scenarios` array or re-covered `F-001-S001`,
so the role burned turns against policy instead of creating the next product
ticket.

### Decision

When the CTO handoff gate blocks implementation because specific product
scenarios are still missing, that pending scenario batch is stored in the tool
session. A later CTO `ticket_create` with no usable `bdd_scenarios` can infer
the missing BDD scenario IDs from either the ticket title/body/metadata or the
pending handoff state before the ticket is created. Policy feedback still tells
CTO to use an explicit JSON array, but the tool can now recover from local-model
argument drift during the same job.

### Consequences

- CTO can recover from a missing `bdd_scenarios` array after the gate has
  already named the exact next product scenarios.
- Created tickets still contain durable BDD scenario IDs, preserving evidence
  and ticket ordering.
- The policy continues to reject duplicate already-covered scenario IDs, so
  automatic recovery does not reopen the earlier duplicate-ticket loop.

## AD-248: Planner Roles Cannot Mutate Targets Through Shell

**Status:** Accepted
**Date:** 2026-05-22
**Owner:** Mars Harness maintainers

### Context

The `demo-tetris-36` replay found a fresh-bootstrap leak before the product
ticket spine could form. CEO read the Phaser Tetris brief and hit several
correct policy blocks, but shell execution still attempted product and
dependency-shaped mutations such as root package setup before COO, CTO, or
Engineer owned the work. That left untracked product-shaped files in the target
and spent turns on guardrails rather than the next product planning handoff.

### Decision

Planner roles cannot use `shell_exec` for mutating commands. CEO, Head of
Strategy, COO, CTO, and CTO-weekly may still perform read-only inspection when
otherwise policy-safe, but shell-based file, package, dependency, cleanup, or
ticket mutations are rejected. The role-specific write path remains explicit:
CEO writes strategy artifacts, COO writes planning artifacts, CTO creates
tickets and bounded technical rationale, and implementation or dependency
mutation belongs to ticket-backed Engineer or dependency tools.

### Consequences

- Fresh bootstrap stays in the product-first handoff path instead of creating
  dependency or product files before a ticket exists.
- Existing target manifests that still expose `shell_exec` to planner roles are
  contained by policy instead of relying on prompts.
- Read-only planning inspection remains available, so the policy does not blind
  planners that need to inspect target state before handoff.

## AD-249: Capability Extraction Separates Product Scope From Non-Goals And Operations

**Status:** Accepted
**Date:** 2026-05-22
**Owner:** Mars Harness maintainers

### Context

The `demo-tetris-37` replay moved cleanly through CEO and into COO after the
planner shell containment fix. COO created a product-specific active plan and
feature contract, but the completion gate repeatedly rejected handoff because
capability extraction treated the active-goal Scope and Non-Goals block as one
long sentence. Optional mechanics listed as non-goals, such as hold piece, and
operational constraints, such as npm install/build/local-validation scripts,
were therefore promoted into required product scenario coverage.

### Decision

Product capability extraction now treats Markdown bullets, numbered list
entries, headings, and blank lines as capability statement boundaries. It
recognizes explicit product-action phrasing such as must implement, must
detect, must allow, and should let, while leaving generic access phrasing,
operational build/install constraints, and non-goal prose outside the required
product scenario set. Capability matching also normalizes small gameplay
wording differences such as tracking score to score behavior and reaching game
over to game-over behavior, plus line-clearing and restart wording such as
full lines or another round.

### Consequences

- COO can descoped optional mechanics without being forced to make them target
  implementation tickets.
- Build scripts, install steps, and validation constraints can remain in active
  goals as delivery constraints without becoming BDD product scenarios.
- The scenario coverage gate still requires real product behaviors such as
  playfield, controls, scoring, game over, and restart to appear in the
  Scenario Schedule or scenario headings.

## AD-250: Capability Matching Accepts Natural Scenario Titles

**Status:** Accepted
**Date:** 2026-05-22
**Owner:** Mars Harness maintainers

### Context

The `demo-tetris-41` replay reached a stronger COO state: the active plan and
feature contract were product-specific, and the contract grouped related
Tetris capabilities into five runnable scenarios. The gate still rejected the
handoff because the README phrase "reach game over when the stack fills" was
matched too literally against the scenario title "Game ends when stack fills
and user can restart." The scenario body named "game over", but the outline
guard deliberately reads only the schedule and headings to make sure
capabilities are visible before ticketing.

The follow-on `demo-tetris-42` replay showed the same tightness in two more
natural planning phrases. A schedule entry saying "Playfield is visible and
keyboard controls work" should cover "see a Tetris playfield" without requiring
the product name in every title, and "Tetrominoes move and rotate with
keyboard" should cover "rotate falling tetrominoes with the keyboard" without
requiring the motion modifier "falling" inside the rotation scenario title.

### Decision

Capability matching now treats game-over wording and game-ending wording as
the same product capability. Product-name tokens such as Tetris and modifier
tokens such as falling are not required outline keywords when the actual
product behavior, such as playfield or rotate tetromino with keyboard, is
visible. Scenario outlines may say "game over" or "game ends" when covering an
explicit game-over brief requirement, while still requiring restart, playfield,
controls, scoring, and line-clearing behavior to be visibly represented in
schedule entries or scenario headings.

### Consequences

- COO can write natural scenario titles without being forced to mirror README
  prose exactly.
- The outline guard still rejects collapsed product contracts that hide
  multiple capabilities inside one broad runnable/inspectable scenario.
- Fresh browser-game bootstraps can advance from planning to CTO ticketing
  once the schedule clearly exposes the requested product behaviors.

## AD-251: Advanced Score Persistence Does Not Descope Basic Scoring

**Status:** Accepted
**Date:** 2026-05-22
**Owner:** Mars Harness maintainers

### Context

The `demo-tetris-43` and `demo-tetris-44` replays produced strong product
contracts with separate scenarios for browser access, playfield, movement,
rotation, line clearing with score points, game over, and restart. The
completion gate still blocked COO because the Out of Scope section said "High
score tracking or persistence," "animations beyond basic movement," and "UI
beyond the game grid and score display." Those lines exclude advanced
extensions, not the basic score and movement capabilities that the scenario
schedule already kept in scope.

### Decision

Out-of-scope capability checks now treat high-score tracking, persistence, and
`beyond ...` qualifiers as advanced extensions. Such lines do not count as
descoping the basic movement or score-tracking capability when that behavior
remains visible in the scenario schedule or scenario headings. Directly listing
"movement", "scoring system", or equivalent basic behavior under Out of Scope
still requires a Descoped Scenarios rationale.

### Consequences

- COO can keep first-slice scope small by excluding high-score persistence
  without blocking ticket handoff for ordinary score behavior.
- The gate still protects against accidentally moving required product behavior
  out of scope.
- Browser-game planning can express a minimal viable scoring rule separately
  from later persistence or leaderboard work.

## AD-252: Failure Ownership Classification Is Universal Doctrine

**Status:** Accepted
**Date:** 2026-05-22
**Owner:** Mars Harness maintainers

### Context

The Tetris/Phaser live loop improved first-run product delivery, but it also
showed the danger of overfitting. Some failures were general factory defects:
planning capability matching, QA validation-procedure recovery, and wrong
ticket routing. Others could have been real target product bugs if source
inspection or runtime evidence had proven broken game behavior. Treating every
symptom as either a target ticket or a source fix would make the harness worse:
target repos get polluted with foundation debt, while foundation changes become
too specific to one stack or product.

### Decision

Every live demo, dogfood, telemetry, review, and operator finding is classified
before action:

- **Foundation-owned:** runtime, orchestration, generated defaults, role
  guidance, tool policy, model/provider behavior, telemetry, release/update, or
  mirrored doctrine. Fix in `mars-harness` source, tests, docs, role prompts,
  generated target defaults, tools, or skills. The improvement must benefit all
  applicable users or a clearly named project class.
- **Deployed-owned:** target product behavior, target architecture, local
  package/build/test setup, target docs, target skills, or project policy. Fix
  in the target repo and preserve target evidence.
- **Mixed/unclear:** use the smallest local unblock only when needed to finish
  target evidence, then record a foundation follow-up for the reusable defect;
  otherwise keep the finding as observation, telemetry, or investigation until
  ownership is clear.

Live improvement batches are grouped by ownership and generality, not by the
demo stack that exposed them. A Phaser or Tetris observation can justify a
foundation change only when the failing mechanism is generic to browser
frameworks, review routing, ticket lifecycle, tool policy, or generated
doctrine. A genuine product bug stays deployed-owned.

### Consequences

- The continuous improvement loop remains product-finishing oriented without
  turning one representative demo into the whole factory specification.
- Target backlog stays focused on target value; foundation defects remain
  foundation evidence or source tickets until fixed for all applicable users.
- Review and dogfood reports must name the failure owner, fix level, and
  rerun evidence before claiming improvement.
- Generated target harnesses inherit the same classification rule, while
  source-only mechanics such as `demo-123` remain foundation shorthand.

## AD-253: Review Procedure Failures And Rework Routing Stay At The Right Layer

**Status:** Accepted
**Date:** 2026-05-22
**Owner:** Mars Harness maintainers

### Context

The first classified post-AD-252 finding came from `demo-tetris-45`. QA ran
`curl -f http://localhost:5173/` before starting a dev server. That was a
review procedure failure, not proof that the target product implementation was
broken. The same run then exposed a second foundation-owned defect: after QA
requested rework on `T-003`, Engineer's rework preflight selected the oldest
completed ticket, `T-001`, because it scanned all done tickets instead of the
dispatch source disposition.

### Decision

Review HTTP probes that fail because no server is listening are tracked as
validation-procedure failures. They do not increment product validation failure
counters, and QA/Security may recover by starting the appropriate dev/static
server with `background:true`, running a separate readiness/product probe, and
then approving or requesting changes from real evidence. Failing builds,
failing tests, and runtime probes that execute a product command still remain
target-owned validation failures unless a procedure classifier explicitly
matches them.

Engineer rework selection now honors the dispatch trigger's
`source_disposition.ticket_id` for `changes_requested` and
`implementation_rework` handoffs. When the Orchestrator routes Engineer back to
a specific ticket, the claim/reopen policy must require that ticket, not an
arbitrary older done or in-review product ticket.

### Consequences

- QA validation setup mistakes no longer create false target rework or freeze
  review jobs before the correct server/probe sequence can run.
- Engineer rework stays attached to the ticket that was actually reviewed.
- Both fixes are foundation-owned and generic: they apply to any web target or
  review handoff, not to Phaser or Tetris specifically.

## AD-254: CTO Duplicate Ticket Failures Do Not Poison Covered Batches

**Status:** Accepted
**Date:** 2026-05-22
**Owner:** Mars Harness maintainers

### Context

The `demo-tetris-47` clean rerun validated the product-first path through CEO
and COO, then exposed a CTO recovery loop. CTO created a valid first product
ticket, hit the early-scenario handoff gate, created additional ordinary product
tickets until the first three early scenarios were covered, and committed the
batch. After that it attempted a broad duplicate `ticket_create` for scenarios
that were already covered. The duplicate ticket guard correctly blocked the
extra ticket, but the successful-disposition guard then rejected Engineer
handoff because the last ticket creation attempt failed and no later successful
ticket creation followed. The repo already contained the required early product
ticket batch, so the role was trapped in policy recovery instead of handing off
to implementation.

### Decision

Successful CTO implementation handoff is now coverage-aware after a failed
ticket creation attempt. A failed duplicate `ticket_create` still blocks the
duplicate ticket, but it does not poison disposition when the ticket lifecycle
already covers the required early product scenarios for the feature contract.
The existing clean-tree, ticket lifecycle, and early-scenario coverage gates
still apply; this only removes the ordering assumption that a successful
`ticket_create` must occur after the latest failed duplicate attempt.

### Consequences

- CTO can recover from redundant or duplicate ticket attempts once the target
  repo already has the valid batch needed for Engineer.
- The duplicate-ticket policy still protects target backlog quality.
- The fix is foundation-owned and generic to BDD ticket lifecycle handoff; it
  does not encode a Tetris or Phaser-specific exception.

## AD-255: Capability Matching Treats Product Pieces As Specific Game Pieces

**Status:** Accepted
**Date:** 2026-05-22
**Owner:** Mars Harness maintainers

### Context

The `demo-tetris-48` rerun was intended to validate AD-254, but it surfaced an
earlier COO gate before CTO ticketing. The README said pieces should "lock into
the stack," while the refined feature contract used Tetris-specific language:
"Tetrominoes Lock Into Stack On Contact" and "the tetromino locks into the
stack." The contract was product-specific and covered the behavior, but the
capability matcher treated `pieces` as a separate required keyword instead of a
generic form of the target's concrete game piece.

### Decision

Capability matching now normalizes `piece` and `pieces` to the same product
capability keyword as `tetromino`. This preserves the requirement that locking
behavior be visible in the scenario schedule or headings, while allowing the
scenario contract to use domain-specific nouns.

### Consequences

- COO can use precise product vocabulary without being forced to repeat README
  wording verbatim.
- The planning gate remains generic: it accepts specific nouns that represent
  the same product object, not a Tetris-only exception to scenario coverage.

## AD-256: Duplicate Feature Path Guidance Respects Planner Ownership

**Status:** Accepted
**Date:** 2026-05-22
**Owner:** Mars Harness maintainers

### Context

The `demo-tetris-49` rerun exposed a foundation-owned policy contradiction in
fresh bootstrap. CEO attempted to create `docs/features/F-001.md` after finding
that the generated target already had the canonical
`docs/features/F-001-product-walking-skeleton.md`. The duplicate feature path
guard correctly rejected the duplicate, but its recovery text told the active
role to update the canonical contract. CEO then followed that instruction and
hit the planner write boundary, because feature contracts belong to COO
planning, not CEO strategy. The guardrails were individually correct, but the
combined feedback sent the role into avoidable recovery turns.

### Decision

Duplicate feature contract path guidance is now role-aware. Roles that are not
allowed to write feature contracts, including CEO, are told to record strategy
in their allowed strategy artifacts or hand off to COO to update the canonical
feature contract. Roles that may own feature contracts still receive the normal
instruction to update the canonical contract instead of creating a duplicate
path.

### Consequences

- Fresh bootstrap preserves the CEO -> COO ownership boundary instead of
  encouraging CEO to write planning artifacts.
- Duplicate feature ID protection remains intact for every role.
- The fix is foundation-owned and generic to generated target planning policy,
  not specific to the Tetris demo.

## AD-257: Browser-Framework Entry Validation Must Not Trap Engineer

**Status:** Accepted
**Date:** 2026-05-22
**Owner:** Mars Harness maintainers

### Context

The `demo-tetris-50` rerun validated the CEO/COO bootstrap path and reached
Engineer with ordinary product tickets. Engineer then hit two foundation-owned
entry-loop defects before source could be written. First, `node --check
main.js` failed because the file did not exist yet; the runtime policy treated
that procedure mistake as an unresolved product validation failure, later
blocking the correct `node --check src/main.js` command. Second, the Phaser
source guard misread a valid top-level `new Phaser.Game(config)` placed after
scene callback declarations as if that construction occurred inside each
callback function. Engineer repeated the same blocked write instead of reaching
the first product implementation.

### Decision

`node --check` against a missing file is now validation-procedure failure, not
an unresolved product runtime failure. It records procedure evidence and leaves
Engineer free to create or inspect the correct implementation path.

Phaser source lifecycle checks now inspect the actual JavaScript function body
for `preload`, `create`, and `update` before flagging recursive game
construction. A single top-level `new Phaser.Game(...)` after callback
declarations is allowed; constructing `new Phaser.Game(...)` inside a scene
callback remains blocked.

### Consequences

- Browser-framework tickets can recover from an early wrong-path syntax probe
  without poisoning later validation.
- The Phaser lifecycle guard still protects real recursive game construction
  while no longer rejecting normal module startup shape.
- The fix is foundation-owned and generic to browser-framework validation and
  Phaser entrypoint policy; it does not encode target-specific Tetris behavior.

## AD-258: Product Capability Coverage Follows Active Feature Contracts

**Status:** Accepted
**Date:** 2026-05-22
**Owner:** Mars Harness maintainers

### Context

The `demo-tetris-51` rerun validated the AD-257 code path far enough to expose
a planning-level contradiction. COO created an active product-specific
`F-002` feature contract for core Tetris gameplay and marked the generated
`F-001` walking skeleton as superseded. The handoff guard still evaluated
`F-001` as the only contract that could satisfy README capabilities, so COO
cycled between editing `F-001`, editing `F-002`, and retrying disposition
instead of handing product ticket breakdown to CTO.

### Decision

Product capability coverage now aggregates active feature contracts under
`docs/features/F-*.md`. Superseded feature contracts are ignored for capability
coverage and for the feature ID list used by CTO handoff gates. If no active
feature contract exists, the generated `F-001-product-walking-skeleton.md`
remains the fallback contract.

### Consequences

- COO can either refine `F-001` or deliberately supersede it with a more
  product-specific active feature contract without creating a policy loop.
- CTO ticket sequencing starts from active product contracts instead of stale
  superseded walking-skeleton scenarios.
- Generated `F-001` remains useful as a bootstrap fallback, but it no longer
  monopolizes product capability coverage after the project has a better
  active feature contract.

## AD-259: Browser-Framework Node Eval Failures Are Procedure Mistakes

**Status:** Accepted
**Date:** 2026-05-22
**Owner:** Mars Harness maintainers

### Context

The `demo-tetris-52` rerun validated the AD-258 planning path: CEO, COO, and
CTO reached an ordinary product ticket, and Engineer created a package-managed
browser app, installed dependencies, started Vite, and confirmed the HTTP
entrypoint responded. Engineer then attempted a plain Node smoke probe that
`require()`d a Phaser scene module. Because Phaser initializes browser-only
globals at module load, the probe failed with `ReferenceError: window is not
defined` from `node_modules/phaser`. The runtime recorded that as an
unexpected product runtime failure and blocked subsequent `node --check` and
`npm run build` validation, even though the failure was caused by the probe
choice rather than by executing the product in its browser environment.

### Decision

Plain Node eval commands that import or require project browser modules and
fail with missing browser globals from a declared browser-framework package are
classified as validation-procedure failures. They record procedure evidence
but do not create an unresolved product runtime blocker. Engineer can continue
with the correct validation surface: package build, managed dev or preview
server, browser-product smoke, or a source/runtime assertion that does not load
browser-only framework startup under plain Node.

### Consequences

- Browser-framework delivery no longer gets stuck after an over-eager Node
  module probe of code that is meant to run in a browser bundle.
- Real browser-framework defects remain guarded by build evidence, source
  inspection, and product-smoke requirements.
- The rule is foundation-owned and general to browser-framework targets; the
  Tetris replay is evidence, not the product-specific rule.

## AD-260: Root Browser-Smoke Helpers Stay Out Of Product Repos

**Status:** Accepted
**Date:** 2026-05-22
**Owner:** Mars Harness maintainers

### Context

The `demo-tetris-53` replay confirmed that the AD-259 change removed the
browser-framework Node eval trap. Engineer reached a real product slice,
created a Vite/Phaser package, passed `npm run build`, ran a managed Vite
server, confirmed the HTTP entrypoint, and then ran the documented direct
`node -e` browser-smoke source assertion successfully. The job still ended at
`max_turns` because Engineer first wrote a repo-root `validate-game.js`
scratch helper, committed it with the product implementation and ticket
evidence, then spent the final turns trying to remove or inspect that helper
instead of moving the ticket to `done`.

### Decision

New repo-root JavaScript/TypeScript/Ruby/Python/Go/shell helper files with
validation, validate, smoke, probe, scratch, verify, or test-like names are
blocked as scratch validation noise. Browser-framework helper code remains
allowed only when it is intentionally durable under `scripts/` or `tests/`.
For one-off product-smoke proof, roles should use direct `shell_exec` commands
such as the documented `node -e` assertion, package build, managed server
probes, or browser automation.

### Consequences

- One-off browser-smoke helpers no longer become accidental product files that
  derail ticket closure or require cleanup commits.
- Durable validation remains possible under explicit validation directories.
- The rule generalizes beyond Phaser/Tetris: root scratch validation belongs
  outside the target product surface for any deployed harness.

## AD-261: Dependency Sync Repairs Missing Build Dependencies

**Status:** Accepted
**Date:** 2026-05-22
**Owner:** Mars Harness maintainers

### Context

The `demo-tetris-54` replay validated the root scratch-helper guardrail: the
Engineer used a direct `node -e` browser-smoke assertion and avoided creating a
repo-root helper file. The next lifecycle blocker happened after `npm run
build` failed with `vite: command not found`. Engineer correctly called
`dependency_sync`, which installed the package dependencies, but the unresolved
test/build repair lane still treated the build failure as unchanged and blocked
the natural `npm run build` rerun until another source edit occurred. That
pushed the role toward unrelated probes and unnecessary product mutation.

### Decision

Successful `dependency_sync` by Engineer now counts as a repair action for an
outstanding test/build failure in the same job. The repair lane still requires a
later successful same-lane test or build command before commits, ticket
completion, runtime probes, or successful disposition can proceed; dependency
sync only opens the rerun path when package-manager setup was the bounded
repair action.

### Consequences

- Missing local build tools and package dependencies can be repaired without
  forcing an artificial source edit.
- The validation gate still prevents blind completion after dependency install;
  same-lane validation must pass before delivery can close.
- The rule is foundation-owned and generic to package-managed target projects,
  not specific to Tetris or Phaser.

## AD-262: Browser Framework Tickets And Review Smoke Stay In One Evidence Path

**Status:** Accepted
**Date:** 2026-05-22
**Owner:** Mars Harness maintainers

### Context

The `demo-tetris-55` replay validated the dependency-sync repair path and
reached a full package-managed Phaser implementation. The next lifecycle drag
was not implementation capability: CTO created a ticket whose acceptance
criteria still said the Phaser library should load from a CDN, contradicting
the foundation policy that requires local npm dependencies and package build
evidence. QA then correctly blocked HTTP/build-only approval, but the approval
error did not include the canonical smoke command already known by the
foundation policy. QA also tried to stop its managed Vite server with
`kill <pid>` and hit the generic validation-only shell boundary before the
tracked background cleanup path could run.

### Decision

Phaser/browser JavaScript feature tickets cannot prescribe CDN-only runtime
loading or CDN acceptance criteria; they must require a local `phaser` package,
deterministic package build evidence, and browser-product smoke evidence. The
review approval gate now prints the same canonical browser-smoke command that
Engineer completion uses when product smoke is missing. QA and Security may
stop tracked background PIDs created by `shell_exec background:true`; arbitrary
cleanup and untracked process kills remain blocked.

### Consequences

- Planning, implementation, and review now point at the same browser-framework
  evidence path instead of creating contradictory ticket text.
- QA can recover from the normal managed-dev-server validation flow without
  turning cleanup into a target rework request.
- The rule is foundation-owned and general to browser-framework targets; the
  replay supplied evidence, but the fix benefits any deployed harness using a
  browser framework.

## AD-263: Generic Gameplay Summary Labels Are Not Standalone Capabilities

**Status:** Accepted
**Date:** 2026-05-22
**Owner:** Mars Harness maintainers

### Context

The `demo-tetris-56` replay validated the browser-framework ticket and review
guidance changes far enough to reach COO planning again. COO created and
repaired a product-specific feature contract with scenarios for opening the
browser game, playfield visibility, movement and rotation, line clearing,
scoring, game over, and restart. The capability gate still rejected handoff
because CEO had created a goal titled "Implement core Tetris gameplay
mechanics", and capability extraction treated the generic summary phrase "core
Tetris gameplay mechanics" as a separate requirement even though the concrete
mechanics were already covered.

### Decision

Capability extraction no longer treats generic summary terms such as `core`,
`gameplay`, `mechanic`, or `mechanics` as standalone capability keywords.
Concrete behavior words still drive coverage: playfield, move, rotate, line
clearing, score, game over, restart, browser access, collision, locking, and
similar product actions remain required when the brief or goals name them.

### Consequences

- Goal headings can summarize a product area without creating an impossible
  duplicate coverage requirement.
- COO still has to cover the actual product behaviors before CTO ticketing.
- The fix is generic to games and application domains that use summary labels
  such as core workflow, gameplay mechanics, or product mechanics.

## AD-264: Alternate Input Scope Does Not Descope Basic Movement

**Status:** Accepted
**Date:** 2026-05-22
**Owner:** Mars Harness maintainers

### Context

The `demo-tetris-57` replay confirmed the AD-263 generic-label fix: CEO
completed quickly and COO no longer looped on "core gameplay mechanics".
COO then produced a product-specific plan and feature contract, but the
capability guard rejected handoff because the Out of Scope section listed
"Mobile touch controls". The matcher treated the word `controls` as movement
coverage, so an excluded alternate input modality looked like the required
keyboard movement behavior had been descoped.

### Decision

Movement coverage may still be satisfied by concrete directional language or
keyboard controls, but generic `controls` alone no longer covers the `move`
capability. This keeps "keyboard controls work" as valid coverage while
preventing "mobile touch controls" or other alternate-input exclusions from
blocking a feature contract that already covers keyboard movement.

### Consequences

- Product contracts can explicitly exclude alternate platforms or input
  modalities without accidentally descoping the required primary behavior.
- The capability matcher remains strict about actual movement scenarios.
- The fix generalizes beyond Tetris to browser apps, games, and tools with
  optional mobile, touch, mouse, or alternate-device input modes.

## AD-265: Browser Evidence Completion Stops Further Shell Exploration

**Status:** Accepted
**Date:** 2026-05-22
**Owner:** Mars Harness maintainers

### Context

The `demo-tetris-58` replay validated the latest planning and
browser-framework fixes through real implementation: CEO and COO produced a
product-specific plan, CTO created a local-dependency Phaser ticket, and
Engineer built a Vite/Phaser slice. Engineer then passed the canonical
browser-product smoke and `npm run build`, but continued shell exploration into
`dist/assets` instead of committing the remaining dirty files, updating ticket
evidence, moving the ticket to `done`, and recording disposition. That extra
bundle inspection pushed the next prompt over the local model context window.

### Decision

For browser-framework Engineer jobs, once the same job has passed all required
completion evidence, including deterministic package build and browser-product
smoke, further `shell_exec` exploration is blocked while implementation or
ticket files remain dirty. The only shell exception is tracked background PID
cleanup. The policy message sends Engineer to `git_status`, `git_commit`,
ticket evidence update, `git mv ... docs/tickets/done/`, lifecycle commit,
push when configured, and `job_disposition_record`.

### Consequences

- Successful browser-framework validation becomes a convergence point instead
  of an invitation to inspect generated bundles or repeat probes.
- Context-overflow failures after already-sufficient build/smoke evidence are
  reduced without weakening the build and smoke gates.
- The rule is foundation-owned and generic to package-managed browser targets,
  not a Tetris-specific optimization.

## AD-266: Active Scenario Deduplication And Review Evidence Shortcuts

**Status:** Accepted
**Date:** 2026-05-22
**Owner:** Mars Harness maintainers

### Context

The `demo-tetris-59` replay validated AD-265 through the first three product
tickets. CEO, COO, and CTO produced product-specific planning and multiple
ordinary product tickets; Engineer completed T-001, T-002, and evidence-only
T-003 without intervention-debt flooding. The remaining drag came from
repeatable foundation behavior: Dogfood created a product finding for
`F-001-S003` even though active product tickets already covered that scenario,
QA and Security first tried validation shapes that policy later corrected, and
Orchestrator wasted turns looking for `docs/tickets/T-NNN...` paths that do not
exist.

### Decision

`ticket_create` now treats active feature-ticket scenario overlap as a duplicate
even when the proposed ticket's scenario list is not exactly equal to the
existing ticket's list. This applies to backlog, in-progress, and in-review
tickets while preserving explicit dependent decompositions through
`depends_on`. Review role guidance now tells QA and Security to run the
canonical browser-product smoke immediately after package build evidence for
browser-framework tickets, and Security no longer starts from broad recursive
secret scans through `shell_exec`. Orchestrator guidance now uses lifecycle
ticket paths from disposition context or the ticket index and forbids assuming
flat `docs/tickets/T-NNN...` paths.

### Consequences

- Dogfood can still create target-owned findings, but not duplicate scenarios
  that active product tickets already cover.
- Review roles spend fewer turns rediscovering the browser-framework evidence
  gates that policy already enforces.
- Security remains bounded to the changed surface and avoids predictable
  shell-policy failures for broad recursive scans.
- Orchestrator handoffs preserve ticket IDs and paths without burning routing
  turns on impossible ticket locations.

## AD-267: Post-Build Browser Smoke Gate Blocks Substitute Probes

**Status:** Accepted
**Date:** 2026-05-22
**Owner:** Mars Harness maintainers

### Context

The `demo-tetris-60` rerun confirmed the product-first bootstrap and active
scenario dedupe path: one CEO bootstrap job was seeded, CEO and COO created
Tetris-specific planning artifacts, CTO created a three-ticket product backlog,
and Engineer claimed the first ticket and reached a successful Vite/Phaser
package build. The next failure was validation drift before ticket closure.
After `npm run build` passed but before browser-product smoke passed, Engineer
inspected `dist/assets`, attempted `require('phaser')` in plain Node, and tried
to require browser bundles from Node. Those checks predictably produced
browser-global failures such as missing `window` or `document`, then sent the
job toward source churn instead of mounted product evidence.

### Decision

For browser-framework Engineer jobs, once the same job has successful package
build evidence and the target repo still has dirty implementation or ticket
work, `shell_exec` is limited to build reruns, canonical browser-product smoke
or equivalent source/runtime assertion, and tracked background PID cleanup until
browser-product smoke passes. Runtime policy blocks substitute shell validation
such as generated-bundle inspection, plain Node `require('phaser')`, requiring
Vite browser bundles from Node, `node --check` on HTML, or trivial environment
probes. The prompt and generated target guidance name the same boundary so the
role chooses the product-smoke lane directly.

### Consequences

- Build success now routes toward mounted UI evidence instead of local-model
  experimentation with browser-only bundles under Node.
- The rule remains foundation-owned and generic to package-managed browser apps,
  not a Tetris-only behavior.
- Engineer can still rerun the real build after repairs and can still run the
  canonical smoke command needed to close the ticket.

## AD-268: Out-Of-Scope Explanations Do Not Descope Basic Product Capabilities

**Status:** Accepted
**Date:** 2026-05-22
**Owner:** Mars Harness maintainers

### Context

The `demo-tetris-61` rerun confirmed the post-build browser-smoke policy in the
fresh generated target guidance, but surfaced an earlier planning loop. COO
created a valid product-specific scenario schedule covering playfield, falling
pieces, movement, rotation, locking, line clearing, score, game over, and
restart. It also listed advanced-only exclusions such as high-score persistence,
combos/back-to-back scoring systems, previews, multiplayer, mobile touch
controls, sounds, and animation polish under Out of Scope. The capability guard
then treated explanatory text like "clear reasons" as a descoping of line
clearing, and treated "advanced scoring systems" as a descoping of basic score
tracking. COO repeatedly rewrote the feature contract and could not hand off to
CTO, even though the required scenarios were present.

### Decision

Out-of-scope capability checks now ignore section-introduction and rationale
prose such as "the following capabilities", "clear reasons", and "explicit
rationale". They also treat advanced scoring systems, combos, back-to-back
scoring, high-score persistence, and other advanced-only extensions as leaving
basic score tracking and line clearing in scope when those basics are already
covered by the Scenario Schedule and scenario headings. COO and generated target
guidance now state that Out of Scope may list advanced-only extensions, but must
not imply that basic in-scope capabilities are excluded.

### Consequences

- Valid feature contracts can exclude future polish or advanced mechanics
  without blocking the first complete product build.
- The guard still rejects actual required behavior listed under Out of Scope
  without a Descoped Scenarios rationale.
- Planning policy remains generic to deployed software projects where advanced
  extensions share words with the base product behavior.

## AD-269: Capability Matching Ignores Glue Words Around Real Behaviors

**Status:** Accepted
**Date:** 2026-05-22
**Owner:** Mars Harness maintainers

### Context

The `demo-tetris-62` replay confirmed that AD-268 removed the false
Out-of-Scope blocker. COO then repaired the feature contract into separate
scenario entries for visible grid, falling tetrominoes, movement/rotation,
locking, line clearing, scoring, game over, restart, and runnable gameplay. The
next blocker came from capability extraction noise: active goal and README
phrases such as "core Tetris gameplay including visible grid" and "game over
detection" produced standalone keywords for "including" and "detection". Those
are connective words around the real product behaviors, not separate
capabilities, so the guard kept rejecting a valid schedule.

### Decision

Capability keyword matching now treats include/includes/including,
show/shows/showing, display/displays/displayed, and detect/detected/detection
as stop words. The concrete behavior words around them still matter: visible
grid, line clearing, score tracking, game over, restart, and similar product
terms must be covered by scenario schedule entries or scenario headings. COO
and generated target guidance now call these "generic glue words" and tell
planners to break out the actual behaviors they introduce.

### Consequences

- Product capability checks remain strict about concrete behavior while avoiding
  false missing-scenario loops for ordinary prose.
- Active goal wording can contain natural phrases such as "including X" or
  "game over detection" without forcing unnatural scenario titles.
- The fix generalizes to non-game projects where "including" and "detection"
  are connective planning language rather than product outcomes by themselves.

## AD-270: Enhancement-Only Exclusions Do Not Descope Covered Gameplay

**Status:** Accepted
**Date:** 2026-05-22
**Owner:** Mars Harness maintainers

### Context

The `demo-tetris-64` rerun confirmed the bootstrap idempotency and
product-specific planning improvements on the released `v0.42.26` binary, but
found one remaining Out-of-Scope loop. COO covered falling tetromino pieces and
line clearing in the Scenario Schedule and scenarios. Its Out of Scope section
then excluded "Animations for piece movement or line clearing", which is a
polish/enhancement exclusion. The capability guard treated the line as if basic
piece movement and line clearing were themselves descoped, blocking CTO handoff
even though the basic behaviors were required and covered.

### Decision

Out-of-scope capability checks now treat enhancement-only lines such as
animation polish, animations for piece movement, animations for line clearing,
visual polish, visual effects, optional previews, sound/audio, multiplayer,
mobile-touch controls, hold-piece, and hard-drop variants as leaving the
covered basic behavior in scope. The rule is line-aware: a direct Out of Scope
line for required behavior such as movement, line clearing, scoring, game over,
or restart still requires a Descoped Scenarios rationale before planning can
hand off.

### Consequences

- First-slice planning can exclude polish without accidentally excluding the
  product behavior that polish would enhance.
- The fix is foundation-owned and applies to any deployed project where
  advanced presentation or animation work refers to core behavior.
- Runtime telemetry continues to quarantine these policy false positives as
  foundation signals rather than creating deployed intervention-debt tickets.

## AD-271: COO Current-Failing-Scenario Recovery Updates The Single Active Plan

**Status:** Accepted
**Date:** 2026-05-22
**Owner:** Mars Harness maintainers

### Context

The `demo-tetris-65` replay verified that the enhancement-only Out-of-Scope
fix removed the previous planning loop. COO produced a product-specific active
plan and BDD feature contract, then attempted to recover by creating
`docs/exec-plans/active/current-failing-scenario.md`. Generated deployed
guidance already says there must be exactly one active exec plan, but the
runtime policy's generic implementation-boundary error did not tell COO how to
repair the attempt.

### Decision

The current failing scenario remains a section of
`docs/exec-plans/active/current-operating-plan.md`. COO may update that active
plan and the matching feature contract, but attempts to create another Markdown
file under `docs/exec-plans/active/` are blocked with specific guidance to keep
one active plan and update `current-operating-plan.md` instead. The generated
COO prompt and persona stop conditions now name the same rule directly.

### Consequences

- Planning recovery preserves one source of truth for deployed active work.
- COO receives actionable recovery guidance instead of bouncing through a
  generic policy error.
- Waiting plans and reports still belong in backlog/report locations; the
  active directory stays a single current operating plan.
