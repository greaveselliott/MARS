# AD-084: Canonical Harness Operating Domains

**Status:** Accepted
**Date:** 2026-05-03
**Owner:** Mars Harness maintainers

## Context

Mars Harness already ships explicit starter roles and a strict-trunk delivery
loop. The remaining role-model gap is vocabulary: Mars proved that autonomous
work stays easier to route when role memory is grouped by stable domains and
specific run modes, while Mars Harness manifests currently expose only role
keys such as `engineer`, `qa`, and `pipeline-fixer`.

The product should not break existing bundles or collapse useful starter roles
into one overloaded prompt. It needs a canonical model that can guide trigger
routing, trace review, scoring, trust, and generated target guidance while
preserving explicit manifest role entries.

## Decision

Mars Harness defines six canonical operating domains:

| Domain | Responsibility | Boundary |
| --- | --- | --- |
| Planner | Chooses goals, scenarios, architecture direction, and ticket shape. | Does not implement product code or mark delivery complete. |
| Engineer | Changes source, tests, docs, and deterministic repair code for one bounded ticket or repair. | Must provide evidence and respect ticket, guardrail, trust, and release gates. |
| Reviewer | Checks behavior, design fit, security posture, tool exposure, evidence, and completion claims. | Reports findings or bounded fixes only when explicitly allowed. |
| Maintainer | Keeps dependencies, release state, docs hygiene, scores, and routine upkeep current. | Does not redefine product direction without Planner input. |
| End-to-End Tester | Exercises real build, run, user, or agent paths and records reproducible evidence. | Does not paper over failures by changing acceptance criteria. |
| Orchestrator | Keeps the autonomous loop healthy: queue state, stuck work, recovery, routing, and ticket hygiene. | Does not hide blocked work or start new backlog while active work is misleading. |

The manifest keeps explicit roles as the stable invocation units. Each role may
declare optional `domain` and `mode` metadata:

- `domain` is one of `planner`, `engineer`, `reviewer`, `maintainer`,
  `end-to-end-tester`, or `orchestrator`.
- `mode` is a short lower-kebab-case purpose within the domain, such as
  `ticket-delivery`, `quality-review`, or `pipeline-repair`.

This is an additive contract. Existing manifests without `domain` or `mode`
remain valid. New generated manifests include the metadata so future registry,
trace, scoring, and trigger tooling can reason about role purpose without
renaming role keys or overwriting user-owned target manifests.

Legacy compatibility note: PR or branch-centric delivery is rejected as the
generated default. Compatibility event handlers may exist only as explicit
integrations; normal roles make semantic commits to `main` and push directly.

## Default Role Mapping

| Manifest role | Domain | Mode | Notes |
| --- | --- | --- | --- |
| `ceo` | Planner | `strategy` | Owns vision, active goals, and final strategy/scope decisions. |
| `head-of-strategy` | Planner | `strategy-advisory` | Optional dispatch/manual advisor for strategy memos, tradeoffs, executive narrative, and goal conflicts; CEO owns the final decision. |
| `coo` | Planner | `execution-planning` | Owns active exec plans, BDD feature contracts, scenario schedule, and current failing scenario; planning-only, with implementation routed behind CTO tickets and Engineer delivery. |
| `cto-weekly` | Planner | `technical-planning` | Owns architecture fit, bounded technical decomposition, and implementation tickets. |
| `engineer` | Engineer | `ticket-delivery` | Implements one ticket and records evidence. |
| `pipeline-fixer` | Engineer | `pipeline-repair` | Repairs failing build or check paths through bounded commits. |
| `qa` | Reviewer | `quality-review` | Reviews behavior and evidence against ticket and BDD contracts. |
| `security` | Reviewer | `security-review` | Reviews security posture and allowed remediation. |
| `dependency-manager` | Maintainer | `dependency-maintenance` | Keeps dependency and package-maintenance work bounded. |
| `release-manager` | Maintainer | `release-management` | Runs version, changelog, tag, and release evidence flow. |
| `dogfood` | End-to-End Tester | `dogfood-validation` | Exercises real target setup, build, run, and user/agent paths. |
| `janitor` | Orchestrator | `ticket-hygiene` | Drains stale state, misleading in-progress work, and backlog entropy. |

For Mars Harness source stabilization, the End-to-End Tester domain also owns
the live demo improvement loop against representative target repos. The
canonical foundation rules live in
[foundation-operating-model.md](foundation-operating-model.md) (AD-291, AD-292);
the validation matrix defines the replay set; no single demo subject becomes
foundation doctrine. A replay should show whether a source change actually
improves the operator path from brief to product plan, feature contract,
product ticket, implementation, review, or dogfood evidence without
intervention-debt starvation. When the live check cannot run, the owning role
records the exact blocker and replay steps instead of treating deterministic
tests as sufficient.

## Source-Only Foundation Role Mapping

The source repo may define foundation-only roles that help external AI clients
consume the foundation operating model. These roles are not generated into
target harnesses and do not make `mars-harness` a normal target of its own
agent runs.

| Source role | Domain | Mode | Notes |
| --- | --- | --- | --- |
| `foundation-maintainer` | Maintainer | `foundation-build` | Manual/operator-invoked source role for maintaining the software factory, classifying foundation/deployed ownership, keeping vendor adapters thin, and preserving release/docsync/live-validation discipline. |

## Mode Boundaries

Modes classify why a role is running; they do not loosen policy.

- Tool allowlists remain per explicit role.
- Guardrails apply to the running role and the affected files, not merely the
  domain label.
- Trust is earned and revoked per role, with optional domain aggregation only
  for reporting.
- Scores must record role, domain, and mode when available, but completion
  evidence still comes from tickets, tests, traces, and release artifacts.
- A mode cannot turn a read-only reviewer into a mutating engineer unless the
  manifest role, trust level, and guardrails allow that behavior.

## Personal Guides

Role prompts include a generated Personal Guide rendered from canonical Go
persona definitions. A guide states the role's modus operandi, priorities,
ownership boundary, non-ownership boundary, preferred feedback format,
feedback it needs, feedback it gives, stop conditions, and orchestrator
handoff expectations so other agents can brief it explicitly instead of
relying on implicit expectations.

The guide does not grant new authority: final decisions, tools, schedules,
trust, and guardrails still come from the manifest, role registry, and owning
role contracts.

## AD-105: Foundation Agents Use Canonical Persona Manuals

**Status:** Accepted
**Date:** 2026-05-04
**Owner:** Mars Harness maintainers

Foundation-agent personas are canonical Go structs in `internal/personas`.
Generated docs under `docs/roles/personas/` and generated prompt Personal Guide
sections are checked surfaces, not separate sources of truth. Tests validate
that every default persona has ownership, non-ownership, feedback, stop
condition, and orchestrator handoff sections, and that checked manuals match the
canonical Go definitions.

The default ownership spine is:

`CEO -> COO -> CTO -> Engineer -> QA -> Security -> Dependency Manager -> Release Manager`

The Orchestrator sits between every active role. It reads dispositions,
structured handoff/feedback objects, persona manuals, manifest validity, and
loop guards before choosing the next best role. `head-of-strategy`, `dogfood`,
`pipeline-fixer`, and `janitor` remain support, advisory, or recovery roles;
they do not replace the mandatory delivery owners.

Routing ownership is intentionally hard-cut:

- `goal`, `goals`, `goal_decision`, `vision`, and `scope_decision` route to CEO.
- `strategy_advice`, `executive_narrative`, `tradeoff_analysis`, and
  `goal_conflict` route to Head of Strategy when configured, otherwise CEO.
- `exec_plan`, `planning`, `feature_contract`, `scenario_schedule`, and
  `current_failing_scenario` route to COO.
- `ticket`, `ticket_shaping`, `ticket_breakdown`, `technical_ticket`,
  `implementation_ticket`, and `architecture_review` route to CTO.
- `implementation`, `implementation_rework`, `engineering`, and `fix` route to
  Engineer.
- `qa`, `qa_review`, `evidence_review`, and `review` route to QA.

COO no longer receives `ticket_create`; COO owns execution planning and BDD
contracts. CTO receives `ticket_create`; CTO owns technical decomposition and
implementation tickets. During fresh bootstrap or an empty product backlog, CTO
must keep technical planning product-first: no broad governance, docsync,
tool-inventory, dependency, release, or architecture-audit passes run before a
current-scenario ticket exists. CTO creates or confirms at most one independent
ordinary feature ticket for the current BDD scenario, records implementation as
the next need, and stops. Further decomposition of the same scenario must either
wait for first implementation evidence or be explicitly modeled with
`depends_on`. New generated target harnesses inherit this split, while existing
target upgrades preserve user-owned prompts and manifests.

`job_disposition_record` accepts optional `handoff` and `feedback` objects so
agents can state exactly what they expect from the next role or what correction
the prior role must make. The Orchestrator uses those fields with persona
manuals to avoid implicit feedback, role ping-pong, and intervention-debt
floods.

Successful non-Orchestrator dispositions require a clean target worktree. A
role that produced plans, contracts, tickets, code, or learning files must
commit those changes before `job_disposition_record` can complete. Orchestrator
is allowed to route dirty state left by a prior role so recovery remains
possible, but ordinary forward handoffs cannot hide uncommitted work.

## AD-106: Structured Disposition Packets Travel Through Orchestrator

**Status:** Accepted
**Date:** 2026-05-04
**Owner:** Mars Harness maintainers

Dispatch-mode routing treats `handoff` and `feedback` as runtime data, not
decorative transcript text. When a non-Orchestrator role completes, the server
enqueues Orchestrator with a typed dispatch trigger containing a routing-safe
`source_disposition`: status, next need, ticket ID, reason, evidence links,
trace ID, handoff, and feedback. Orchestrator reads that packet first, chooses
one next owner, and records its own cleaned disposition for the target role.

For Orchestrator-owned dispositions, routing precedence is:
`suggested_role`, then `handoff.target_role`, then `feedback.for_role`, then
`next_need`, then the default completion route. Structured target fields must
agree when more than one is supplied; conflicting owners fail validation with an
actionable error instead of being guessed.

Orchestrator routing validates against executable manifest role keys. Common
domain shorthands are normalized when the matching generated role exists:
`cto` and `architecture` route to `cto-weekly`, `release` routes to
`release-manager`, and `dependency` routes to `dependency-manager`. This keeps
reasonable role-language from causing another Orchestrator pass while still
rejecting unknown roles that have no manifest owner.

If Orchestrator itself fails before recording a disposition, the runtime does
not turn that failed Orchestrator result into another Orchestrator job. It may
fall forward from the original non-Orchestrator source disposition when the
trigger still carries deterministic routing signal, and otherwise records a
stopped decision so recovery is visible without becoming recursive work.

If any dispatch-mode role fails the protocol by completing without
`job_disposition_record`, the runtime records telemetry and stops rather than
asking Orchestrator to reason about a missing tool call. The fix belongs in
role guidance, tooling, or operator retry conditions.

## Trigger Routing

Trigger orchestration routes to explicit role keys first, then uses domain and
mode as metadata for selection, reporting, and future payload routing. This
keeps current bundles deterministic while making the intended memory boundary
visible to agents and tools.

The default generated topology remains strict trunk with dispatch routing:

- Schedules, events, surveys, and Orchestrator dispatch activate stable role keys.
- Event integrations are optional telemetry or repair inputs, not the default
  delivery path.
- Follow-up work that needs a different responsibility returns a disposition to
  Orchestrator, which chooses a role whose domain and mode match the next
  action.

## Migration Path

Existing 11-role bundles continue to load because `domain` and `mode` are
optional fields. Operators can adopt the model incrementally:

1. Keep current role names and prompts.
2. Add `domain` and `mode` metadata to each manifest role when changing the
   manifest deliberately.
3. Use the default mapping above for generated starter roles.
4. Let future role-registry checks report missing or unknown metadata without
   overwriting user-owned manifests.

This intentionally retains explicit roles instead of exposing only six
canonical roles. The six domains are canonical for reasoning and routing; the
manifest roles remain the executable units because they carry schedules, tool
allowlists, prompts, chains, trust, and guardrails.

## Generated Targets And Follow-Up

This slice updates generated target defaults with optional domain/mode metadata
and a seed version of this design decision. Existing targets are not rewritten;
`upgrade` still fills only missing defaults.

## AD-085: Checked Role Registry

**Status:** Accepted
**Date:** 2026-05-03
**Owner:** Mars Harness maintainers

Mars Harness keeps `docs/roles/ROLES.md` as the checked role inventory for the
foundation harness and generated target harnesses. The registry records each
manifest role's origin, canonical domain, mode, trigger sources, schedule,
tools, trust level, guardrails, model routing, scoring signals, and escalation
behavior.

The registry links back to this operating-model decision and the tools glossary
instead of duplicating long architecture sections. Runtime truth remains in
`.harness/manifest.yaml`; the registry is the human and agent inventory that
must stay consistent with that runtime surface.

`mars-harness init` emits the target registry. `mars-harness doctor --repo`
checks the registry against the target manifest and reports actionable drift,
including custom target roles that need `Origin` set to `custom`. Optional
GitHub webhook triggers must be marked optional so compatibility repair inputs
do not replace the schedule-and-chain strict-trunk delivery model.

Follow-up work remains:

- `MH-047` should add native payload-mode routing to jobs and traces where the
  runtime needs more than static manifest metadata.

## AD-274: Foundation Role And Vendor-Neutral Client Adapters

**Status:** Accepted
**Date:** 2026-05-23
**Owner:** Mars Harness maintainers

The foundation operating model must be consumable by all major AI coding
clients, not only one vendor runtime. Mars Harness therefore defines
`foundation-maintainer` as a source-only role profile for maintaining the
software factory and keeps vendor-specific instruction files as thin adapters.

`AGENTS.md` and `docs/roles/personas/foundation-maintainer.md` are the
canonical foundation instruction surfaces. `CLAUDE.md`, `GEMINI.md`,
`.github/copilot-instructions.md`, and `.cursor/rules/*.mdc` point at those
surfaces instead of carrying independent doctrine. Clients that natively read
`AGENTS.md` use it directly.

The runtime supports `mars-harness run foundation-maintainer --repo . --dry-run
--no-init` against the source repository without scaffolding a source
`.harness/manifest.yaml`. The role is rejected for non-source repositories and
is not generated into deployed target manifests.

## Failure Modes And Mitigations

| Failure mode | Mitigation |
| --- | --- |
| Domain names become decorative prose | Generated manifests carry `domain` and `mode`; `docs/roles/ROLES.md` and doctor role-registry checks verify them. |
| Role sprawl returns under new names | Prefer new modes inside existing domains unless a new explicit role needs different prompts, tools, schedules, or trust. |
| A mode hides unsafe authority | Tool, trust, scoring, and guardrail policy remain attached to explicit role execution. |
| Existing bundles break | Domain and mode fields are optional and ignored by older manifests. |
| Branch-centric delivery returns through compatibility work | Strict trunk remains the generated default; compatibility integrations must be explicit and cannot replace direct commits to `main`. |
