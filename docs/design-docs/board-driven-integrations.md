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

### AD-303: JIRA Mirroring Is Pull-Only And Scope-Gated

Plan 2 adds JIRA webhook and poll ingestion as a mirror only. JIRA events use a
dedicated `/webhooks/jira` handler, not the GitHub webhook trigger spine. The
handler validates a repo-mapped webhook secret, normalizes the issue, and
creates or reconciles one Markdown ticket by stable `jira_key`; it does not
register `jira_issue.*` triggers and does not enqueue LLM work.

Materialization is contained by configuration before any file write:

- `project_repo_map` must map the JIRA project to exactly one registered repo.
- `scope.allowed_workspaces`, when set, must match the Atlassian host and
  project from the issue URL or configured JIRA base URL. Board/backlog URLs may
  be used as operator-readable workspace scopes.
- `scope.required_labels`, when set, requires every configured label on the
  issue. The Example Target Project rollout can use `example-required-label` in config to
  limit intake to explicitly marked opportunities.

Unmapped, ambiguous, outside-workspace, or missing-label issues drop with a
sanitized operator-visible reason and no fan-out. Reconciliation updates only
JIRA-owned front matter and the JIRA-owned requirements block, preserving the
ticket lifecycle directory, evidence fields, scoped marker, and agent notes.

## Configuration Surface

The v1 config is intentionally vendor-neutral:

- `flow_profile: ceo-led | board-driven`
- `ingestion.jira`: enablement, endpoint, auth env-var names, webhook secret
  env var, poll interval, JQL, project-to-repo map, workspace and required-label
  scope guards, field IDs, ready statuses, priority/rank/age ordering, and
  blocker handling
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
- Workspace and required-label scope guards are config-owned and enforced
  before ticket writes when configured.
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
- 2026-06-23: JIRA mirror intake needs more containment than project mapping
  alone. Board/workspace URLs and labels remain configuration values, with
  `example-required-label` usable as an opt-in Example Target Project label gate.
- 2026-06-23: Live JIRA read verification against Atlassian Cloud showed the
  legacy `/rest/api/3/search` endpoint returns HTTP 410. Poll ingestion uses
  the current `/rest/api/3/search/jql` endpoint for configured JQL reads.
