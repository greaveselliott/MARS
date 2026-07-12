# Dashboard Control Plane

**Status:** Planned
**Updated:** 2026-07-12
**Owner:** MARS maintainers
**Sources:** [product surface](product-surface.md), [F-010 dashboard feature contract](../features/F-010-dashboard-control-plane.md), [dashboard design doc](../design-docs/dashboard.md), [Node.js release schedule](https://nodejs.org/en/about/previous-releases), [pnpm releases](https://github.com/pnpm/pnpm/releases)

## Product Promise

The next MARS dashboard is a local authenticated control plane for
operators running `mars start` or `mars serve`. It must show
what the autonomous organization is doing, which agents and models are involved,
what work is active, how much execution budget is being consumed, what quality
and delivery signals exist, and what action an operator can safely take.

The existing embedded htmx and Chart.js dashboard remains the legacy/current
implementation until the TanStack dashboard slices ship. This spec governs the
planned replacement and supersedes narrow restyle work as the product contract
for the next dashboard generation.

The legacy/current implementation is nevertheless security-supported under
F-010-S024 and AD-310: its listener remains loopback-only, local page/login
shells, assets, and minimal status are bounded redacted observer surfaces,
privileged reads and SSE require login, and browser mutations require an
environment-only control secret plus an in-memory session, exact Host/Origin,
and session CSRF, and any explicitly configured HTTPS reverse-proxy origin
requires authentication for reads as well. Its htmx 2.0.4 and Chart.js 4.4.7
files are pinned and embedded for offline use under a strict CSP. This does not
claim that the future TanStack local-admin persistence or gateway in
F-010-S012/MH-053 has shipped.

## Runtime Contract

- The frontend uses TanStack Start as a sidecar application, with TanStack
  Router, Query, Table, and Form for route, data, table, and form state.
- UI primitives use shadcn/ui, Tailwind, and lucide icons with an operations
  dashboard visual language: dense, polished, readable, and built for repeated
  work instead of a landing-page composition.
- Node.js `24.x` is an external prerequisite for the TanStack sidecar. Mars
  Harness must not bundle Node, download Node, install Node, or run a Node
  installer on the user's behalf.
- `pnpm@11.1.1` is the required package manager version. The future
  `web/dashboard/package.json` must pin it through `packageManager`.
- If Node `24.x` or `pnpm@11.1.1` is missing, the dashboard setup or launch path
  must fail with actionable remediation text and preserve the core orchestrator.
- The Go server remains the authoritative runtime, API gateway, auth enforcer,
  event broker, and source of dashboard data. The TanStack sidecar must not own
  queue, scoring, trust, safety, or orchestration state.

## Authentication Contract

The dashboard is local-admin only. There is no anonymous read-only dashboard mode
for the TanStack control plane.

- All dashboard routes, dashboard API routes, event streams, command endpoints,
  feedback endpoints, roster mutation proposals, and model mutation proposals
  require a valid local-admin session.
- The only unauthenticated routes are static assets required to render login,
  the login submit route, and non-sensitive health checks already exposed by the
  Go server.
- Local-admin credentials are stored locally with a password hash, creation
  timestamp, and disabled timestamp. The implementation ticket must choose the
  exact storage package and migration shape before code lands.
- Sessions use HttpOnly cookies, SameSite strict behavior, expiration, and
  server-side invalidation.
- Mutating dashboard requests require CSRF protection tied to the session.
- The Go gateway enforces auth before proxying to the TanStack sidecar or
  serving dashboard data. The sidecar must treat gateway identity as the only
  trusted identity boundary.

## Nonblocking API Contract

Dashboard APIs must not be blocked by the processing orchestrator.

- Read endpoints use independently cancellable request contexts, short timeouts,
  and data snapshots from SQLite, trace stores, telemetry stores, runtime status
  caches, or event streams.
- Dashboard reads must not wait on long-running agent execution, model
  generation, repo scanning, release verification, or queue worker critical
  sections.
- Control actions return an asynchronous command id and status instead of
  holding the HTTP request open while work completes.
- Server-sent events or an equivalent authenticated stream publishes command
  status, run state, feedback acknowledgement, and live metric changes.
- Any stale, partial, missing, or permission-blocked data source is represented
  as a typed unavailable state rather than an empty success response.

## Views

### Overview

The Overview is the first authenticated screen. It shows all active agents and
answers who is working, on what, for how long, with which model, and with what
signals.

Required sections:

- active agents, roles, canonical domain/mode, current ticket or task, repo, and
  owner
- run duration, queue age, last event, turn count, tool-call count, and command
  status
- active issues, blockers, guardrail blocks, failed checks, telemetry warnings,
  and unavailable states
- active work items with lifecycle status and linked feature scenario ids
- token usage by active run, role, model, and recent window
- models being used by role and tier, with provider, endpoint kind, and health
- quality score and score movement from repo-visible quality artifacts
- GitHub-derived DORA metrics when configured and available

### Active Work

Active Work renders the current ticket or task detail as an operator-friendly
work surface. It must show title, context, BDD scenario ids, acceptance criteria,
blockers, last evidence, related files, current owner, and next action. Markdown
rendering must be readable without making the ticket file less authoritative.

The dashboard may link to files and traces, but the repo artifact remains the
source of truth.

### Active Work Preview

Preview is adaptive. It starts with frontend projects but must not assume every
target is a single local web app.

Preview provider types:

- web app URL preview for local dev servers, static servers, and deployed review
  environments
- screenshot or video artifact preview when a live server is unavailable
- mobile app preview metadata such as simulator target, deep link, latest build
  artifact, and screen capture
- API preview with base URL, OpenAPI or route inventory, sample request, sample
  response, logs, and trace spans
- cloud or distributed-system preview with environment links, service health,
  deployment plan, infrastructure diff, and observability links
- library or CLI preview with command transcript, generated artifact, examples,
  and test evidence

Annotations attach to the most stable available anchor: route, selector, file,
line, log span, trace id, screenshot coordinate, API route, or environment link.
When no stable anchor exists, the feedback is attached to the active work item
and current run.

### Feedback

The feedback box writes to a next-turn feedback mailbox for the active work
item. Feedback is visible immediately in the dashboard and is added to the next
safe agent context boundary: next turn, next tool-recovery prompt, next handoff,
or next Orchestrator routing decision.

Feedback does not directly rewrite prompts, role files, model overrides, or
ticket state. Any durable change still flows through the normal repo artifact,
ticket, or code-host proposal path.

### Agent Roster

The Agent Roster lists every configured agent with role key, display name,
canonical domain, mode, prompt source, tools, guardrails, triggers, trust,
scores, recent outcomes, token usage, model tier, model override, and current
availability.

The dashboard can propose changes to prompts, role metadata, model routing,
tool allowlists, guardrails, and schedules. Proposed changes are generated as a
draft code-host proposal or a local patch preview with explicit files, reason,
validation plan, and rollback notes. The dashboard must not silently mutate
role source files.

### Available Models

The Models view lists offline and cloud-hosted models currently available to
MARS.

Model sources:

- pinned registry models and download status
- locally cached model files with checksum state
- reachable llama.cpp or OpenAI-compatible endpoints configured for the harness
- Ollama models discoverable through the explicit Ollama catalog path
- configured cloud or remote OpenAI-compatible providers when the user has
  opted in with credentials

For each model, the dashboard shows provider, model name, local/cloud kind,
endpoint, health, context window when known, role/tier eligibility, benchmark
evidence when available, current usage, and unavailable reason. Adding a model
creates an explicit model override or registry-change proposal with validation
requirements; it does not promote a model as a default solely because the model
exists.

## GitHub-Derived DORA

Dashboard DORA metrics are optional GitHub-derived delivery signals. They are
not inferred from ticket movement alone and are not shown as healthy when the
required GitHub evidence is missing.

The repository owns DORA configuration in `.harness/dashboard.yaml`:

```yaml
dashboard:
  dora:
    window_days: 30
    deployment_workflows:
      - Release
      - Deploy
    branch: main
```

Metric definitions:

- Deployment Frequency: count successful terminal GitHub workflow runs whose
  workflow name and branch match the configured deployment set inside the
  selected window.
- Lead Time For Changes: for each successful deployment run, compare the prior
  successful deployment SHA to the current successful deployment SHA, find the
  earliest commit timestamp in that range, and measure time until workflow
  completion.
- Change Failure Rate: failed, cancelled, or timed-out configured deployment
  workflow runs divided by all terminal configured deployment workflow runs in
  the selected window.
- Mean Time To Restore: elapsed time from a failed configured deployment run to
  the next successful configured deployment run for the same workflow and
  branch.

Merged code-host proposals do not count as deployments. Tags do not count as
deployments unless the configured deployment workflow succeeds. When history is
insufficient, DORA cards show "insufficient history" with the missing evidence.

## Unavailable States

Unavailable states are product behavior, not incidental errors.

- Node missing: "Node.js 24.x is required for the TanStack dashboard" plus a
  platform-specific install hint.
- Wrong Node version: show installed version, required version, and a command
  to re-run the prerequisite check.
- pnpm missing or wrong version: show installed version, required version, and a
  package-manager remediation.
- Auth missing: show the local-admin setup path and keep dashboard data locked.
- GitHub auth missing: hide DORA values behind "GitHub auth unavailable" while
  keeping local dashboard views usable.
- No remote configured: DORA and code-host proposal actions explain that a
  remote is required.
- Missing DORA config: show the expected `.harness/dashboard.yaml` fields.
- No matching deployment runs: show the configured workflow names and selected
  window.
- Rate limit or permission error: show the affected metric and retry guidance.
- Preview unavailable: show which provider was attempted and what evidence can
  still be shown instead.

## Safety And Source Of Truth

- Repo artifacts remain the durable source of truth for tickets, feature
  contracts, exec plans, design docs, product specs, quality score, role files,
  and model overrides.
- The dashboard can display, annotate, propose, and enqueue. It cannot silently
  bypass guardrails, trust policy, ticket gates, local-admin auth, or source
  docs.
- Secret values, auth tokens, model provider keys, session cookies, CSRF tokens,
  and webhook secrets are never rendered in the dashboard.
- Mutating actions must be attributable to the authenticated local admin and
  recorded with command status, run id, trace id, or proposal id.

## Out Of Scope

- Hosted SaaS dashboard operation.
- Bundling Node inside MARS.
- Auto-installing Node or pnpm.
- Replacing the repo-visible quality score with a dashboard-only score.
- Treating every reachable model as safe for automatic default promotion.
- Treating GitHub-derived DORA as available without configured workflows and
  validated access.

## Validation Contract

The epic is not complete until implementation tickets provide:

- prerequisite tests for Node `24.x` and `pnpm@11.1.1` detection and remediation
- auth, session, CSRF, and protected-route tests
- nonblocking API tests that run while orchestrator work is active
- asynchronous command-status tests
- Overview, Active Work, Preview, Feedback, Agent Roster, Models, and DORA
  route tests
- browser visual verification for desktop and mobile layouts
- accessibility checks for keyboard navigation, labels, focus, contrast, and
  reduced-motion behavior
- migration evidence proving legacy dashboard routes either redirect, remain as
  a deliberate fallback, or are removed with documented operator impact
