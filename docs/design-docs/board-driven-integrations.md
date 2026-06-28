# Board-Driven Integrations

**Status:** Draft
**Date:** 2026-06-23
**Owner:** MARS maintainers
**Feature Contract:** [F-013-board-driven-integrations.md](../features/F-013-board-driven-integrations.md)
**Operator Runbook:** [atlassian-mcp-jira-intake.md](../runbooks/atlassian-mcp-jira-intake.md)

## Context

MARS currently defaults to a CEO-led, GitHub-compatible, strict-trunk
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

MARS uses a small integrations config loaded from
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

Polling supports provider selection. The default `rest` provider keeps the
direct Atlassian REST search path. The preferred Example Target Project read path is
`provider: atlassian_mcp`, which opens a short-lived session to Atlassian's
official MCP endpoint, probes read capabilities, calls only approved JIRA
read/search tools, and closes the session when the sync job completes. If an
operator configures a local MCP proxy, Mars starts it for that sync job and
terminates it before the job claims success. Proxy transport is explicit:
`sidecar` starts a helper while Mars still speaks HTTP to `endpoint_url`;
`stdio` speaks MCP JSON-RPC directly to the helper subprocess, which supports
OAuth-capable helpers such as `npx mcp-remote`. No JIRA MCP sidecar is part of
`serve`, `start`, setup, or no-config repos.

Materialization is contained by configuration before any file write:

- `project_repo_map` must map the JIRA project to exactly one registered repo.
- `scope.allowed_workspaces`, when set, must match the Atlassian host and
  project from the issue URL or configured JIRA base URL. Board/backlog URLs may
  be used as operator-readable workspace scopes.
- `scope.required_labels`, when set, requires every configured label on the
  issue. The Example Target Project rollout can use `example-required-label` in config to
  limit intake to explicitly marked opportunities.
- `scope.board_id`, when set, is enforced only when the MCP provider advertises
  a board-aware read tool. If the provider lacks such a tool, Mars records a
  `board_scope_not_enforced_by_provider` warning and continues to rely on the
  hard project, workspace, label, and JQL containment gates rather than widening
  scope silently.

Unmapped, ambiguous, outside-workspace, or missing-label issues drop with a
sanitized operator-visible reason and no fan-out. Reconciliation updates only
JIRA-owned front matter and the JIRA-owned requirements block, preserving the
ticket lifecycle directory, evidence fields, scoped marker, and agent notes.

## Configuration Surface

The v1 config is intentionally vendor-neutral:

- `flow_profile: ceo-led | board-driven`
- `ingestion.jira`: enablement, provider (`rest` or `atlassian_mcp`), endpoint,
  auth env-var names, optional MCP endpoint/cloud/site/proxy transport settings,
  webhook secret env var, poll interval, JQL, project-to-repo map, workspace,
  board, and required-label scope guards, env-var indirection for
  tenant-specific values such as base URL, MCP site URL, cloud ID, board ID,
  ID-bearing workspace URLs, and custom field IDs,
  ready statuses, priority/rank/age ordering, and blocker handling
- `design_sources.figma`: enablement, token env-var name, and base URL
- `delivery`: `trunk | pull_request`, branch pattern, and minimum trust

Config stores names of environment variables, never secret values.
Tenant-specific values that operators do not want committed, including JIRA
site URLs, Atlassian cloud IDs, board IDs, ID-bearing workspace URLs, and JIRA
custom-field IDs, should use env-var indirection. Example Target Project field IDs may appear
only as local environment values or operator-owned local config, never as Go
constants.

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
- 2026-06-24: The operator selected Atlassian's official MCP server as the
  preferred Example Target Project JIRA read provider. Mars treats it as a job-scoped external
  integration runtime: the Go binary owns provider selection, read-tool
  allowlisting, session/proxy cleanup, containment, and local ticket mirroring.
- 2026-06-24: Live Atlassian MCP probing with the current API-token credential
  reached the MCP server but advertised only Teamwork Graph tools. Atlassian's
  support docs list `searchJiraIssuesUsingJql` under the `search_jira`
  permission group and note API-token tool availability depends on token/admin
  scopes, so T-047 is blocked on credential/tool availability rather than on
  Mars MCP transport.
- 2026-06-24: Live OAuth probing through the official `mcp-remote` stdio proxy
  completed SSO and exposed `searchJiraIssuesUsingJql`, `getJiraIssue`, and the
  wider JIRA tool surface. A scoped JQL read for project `DEMO` with label
  `example-required-label` returned issues. The remaining Mars-owned gap
  was support for stdio MCP proxy transport and Atlassian's array-shaped
  `fields` argument, not JIRA MCP availability.
