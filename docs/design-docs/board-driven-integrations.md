# Board-Driven Integrations

**Status:** Draft
**Date:** 2026-06-23
**Owner:** Mars Harness maintainers
**Feature Contract:** [F-013-board-driven-integrations.md](../features/F-013-board-driven-integrations.md)

## Context

Mars Harness currently defaults to a CEO-led, GitHub-compatible, strict-trunk
operating flow. Some teams run delivery from a board such as JIRA and need a
workflow that mirrors board tickets, scopes them through planning roles,
delivers through Engineer and QA, and can open human-merged pull requests.

The integration path must be opt-in. A target repo without
`.harness/integrations.yaml` must keep the existing route graph, scheduler,
effective tool allowlists, strict-trunk delivery policy, and CEO-led behavior.
Generated targets receive an example config only; operators copy it when they
want the board-driven profile.

## Decisions

### AD-302: Board-Driven Integrations Are Optional Profile Gates

Mars Harness uses a small integrations config loaded from
`.harness/integrations.yaml`. Missing config, empty `flow_profile`, or unknown
profile names normalize to `ceo-led`. The `board-driven` profile is never
inferred from partial JIRA, Figma, delivery, or model settings.

Plan 1 creates only the substrate:

- `internal/integrations` loads config with version `1` defaults and tolerant
  YAML parsing.
- `init` and `upgrade` write `.harness/integrations.example.yaml` when missing
  and never write `.harness/integrations.yaml` by default.
- `serve` and warm restart reload the config and expose `flow_profile` in
  status responses and logs.
- Board-driven scheduling suppresses only `ceo`, `coo`,
  `head-of-strategy`, and `cto-weekly` cron registrations. The roles remain
  available for explicit dispatch.
- Scheduler restart replaces the registered schedule set instead of appending,
  so profile changes cannot leave stale CEO-led cron entries behind.
- Executor tool allowlists are computed late. The no-config and `ceo-led`
  result is a copy of the manifest list. Future integration tools are appended
  only when the board-driven profile, section gate, and registry all agree.

No JIRA route, JIRA poller, board selector, Figma fetcher, pull-request tool, or
frontier-model gateway behavior is enabled by this decision.

## Configuration Surface

The v1 config is intentionally vendor-neutral:

- `flow_profile: ceo-led | board-driven`
- `ingestion.jira`: enablement, endpoint, auth env-var names, webhook secret
  env var, poll interval, JQL, project-to-repo map, field IDs, ready statuses,
  priority/rank/age ordering, and blocker handling
- `design_sources.figma`: enablement, token env-var name, and base URL
- `delivery`: `trunk | pull_request`, branch pattern, and minimum trust

Config stores names of environment variables, never secret values. Example Target Project
field IDs may appear only as commented examples in the generated example file,
not as Go constants.

Model routing remains in `.harness/model-overrides.yaml`.

## Plan Boundaries

Plan 2 will add JIRA mirror and sync. JIRA remains the source of record and
events do not enqueue LLM work directly.

Plan 3 will add board selection and paid-model cost guards together, so board
dispatch cannot run unbounded.

Plan 4 will wire frontier model overrides through env-var backed API keys.

Plan 5 will add bounded Figma context and gated pull-request delivery.

Plan 6 will mirror deployed-harness operating context and release traceability.

## Security And Blast Radius

- Config parse failures fail closed to `ceo-led` at runtime callers that must
  keep the orchestrator available, while the loader still returns the parse
  error for direct validation.
- No-config repos do not mount JIRA routes or start pollers.
- Project-to-repo mapping is required before future JIRA ingestion can fan out
  work.
- Secrets remain in environment variables and are never stored in generated
  config, traces, or logs.
- Pull-request delivery will be gated by explicit delivery mode, sanitized
  branch pattern, no force-push, base `main`, one PR per JIRA key, and trust.

## Discoveries

- 2026-06-23: Warm restart previously reused schedule registration semantics
  that append entries. Board-driven profile toggles need schedule replacement
  so removed or suppressed schedules cannot survive restart as stale cron
  entries.
- 2026-06-23: Future integration tools must be injected after registry
  construction and profile/section checks. Static generated manifest tool lists
  would violate the no-config invariant.
