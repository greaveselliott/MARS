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
| `ceo` | Planner | `strategy` | Owns goals, feature contracts, and scenario priority. |
| `cto-weekly` | Planner | `architecture-planning` | Validates architecture fit and walking-skeleton shape. |
| `coo` | Planner | `ticket-breakdown` | Converts the active scenario group into deduped tickets. |
| `engineer` | Engineer | `ticket-delivery` | Implements one ticket and records evidence. |
| `pipeline-fixer` | Engineer | `pipeline-repair` | Repairs failing build or check paths through bounded commits. |
| `qa` | Reviewer | `quality-review` | Reviews behavior and evidence against ticket and BDD contracts. |
| `security` | Reviewer | `security-review` | Reviews security posture and allowed remediation. |
| `dependency-manager` | Maintainer | `dependency-maintenance` | Keeps dependency and package-maintenance work bounded. |
| `release-manager` | Maintainer | `release-management` | Runs version, changelog, tag, and release evidence flow. |
| `dogfood` | End-to-End Tester | `dogfood-validation` | Exercises real target setup, build, run, and user/agent paths. |
| `janitor` | Orchestrator | `ticket-hygiene` | Drains stale state, misleading in-progress work, and backlog entropy. |

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

## Trigger Routing

Trigger orchestration routes to explicit role keys first, then uses domain and
mode as metadata for selection, reporting, and future payload routing. This
keeps current bundles deterministic while making the intended memory boundary
visible to agents and tools.

The default generated topology remains strict trunk:

- Schedule and chain triggers activate stable role keys.
- Event integrations are optional telemetry or repair inputs, not the default
  delivery path.
- Follow-up work that needs a different responsibility should chain to a role
  whose domain and mode match the next action.

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

Follow-up work remains:

- `MH-043` should produce a checked role registry that reports manifest roles,
  domain, mode, trigger, tools, guardrails, trust, and scoring signals.
- `MH-047` should add native payload-mode routing to jobs and traces where the
  runtime needs more than static manifest metadata.

## Failure Modes And Mitigations

| Failure mode | Mitigation |
| --- | --- |
| Domain names become decorative prose | Generated manifests carry `domain` and `mode`; the role registry follow-up will check them. |
| Role sprawl returns under new names | Prefer new modes inside existing domains unless a new explicit role needs different prompts, tools, schedules, or trust. |
| A mode hides unsafe authority | Tool, trust, scoring, and guardrail policy remain attached to explicit role execution. |
| Existing bundles break | Domain and mode fields are optional and ignored by older manifests. |
| Branch-centric delivery returns through compatibility work | Strict trunk remains the generated default; compatibility integrations must be explicit and cannot replace direct commits to `main`. |
