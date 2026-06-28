# F-013: Board-Driven Integrations

- Feature ID: F-013
- Goals: G-001, G-002, G-003, G-004
- Status: active
- Owner: COO

## Business Logic

MARS supports two operating profiles:

- `ceo-led`: the default existing flow. GitHub webhooks, cron schedules, strict-trunk delivery, generated role tool lists, and local-first behavior remain unchanged when `.harness/integrations.yaml` is absent or names this profile.
- `board-driven`: an optional flow for teams that use a JIRA board as work-order truth. JIRA mirrors tickets into the repo, board state selects the next ready item, `cto-weekly` scopes the ticket, Engineer and QA deliver it, and PR mode can open a human-merged pull request when explicitly enabled.

The board-driven profile is never inferred. Unknown, empty, or missing `flow_profile` fails safe to `ceo-led`.

`.harness/integrations.yaml` stores configuration only. It may name environment variables, field IDs, endpoints, and route settings, but it must not store token values. Generated harnesses receive only `.harness/integrations.example.yaml`; operators copy it to opt in.

Tenant-specific identifiers that operators do not want committed, including JIRA site URLs, Atlassian cloud IDs, board IDs, board URLs, and JIRA custom-field IDs, are configured through env-var name fields such as `base_url_env`, `site_url_env`, `cloud_id_env`, `board_id_env`, `allowed_workspaces_env`, and `*_env` custom-field settings.

JIRA is the system of record for ticket requirements, priority, sprint, rank, workflow status, blockers, and epic context. The local Markdown ticket is a pull-only mirror for the harness to consume. The harness preserves its own lifecycle directory, evidence fields, scoped marker, and agent progress notes.

JIRA ingestion is contained by configuration before any local ticket write. A JIRA issue may mirror only when it matches exactly one `project_repo_map` entry and, when configured, the issue also matches `ingestion.jira.scope.allowed_workspaces` or `ingestion.jira.scope.allowed_workspaces_env` and carries every label in `ingestion.jira.scope.required_labels`. For the Example Target Project rollout, operators can configure the DEMO board/workspace scope through environment variables and the `example-required-label` label through config; these values live in operator-owned configuration, not Go constants.

JIRA polling can use the direct REST provider or the preferred Atlassian MCP provider. Atlassian MCP use is job-scoped: Mars initializes a short-lived MCP session, probes capabilities, calls only approved read/search tools, mirrors accepted issues, closes the session, and terminates any configured local proxy before the sync job completes. A proxy may be a `sidecar` helper while Mars speaks HTTP to the remote endpoint, or a `stdio` helper such as `npx mcp-remote` where Mars speaks MCP JSON-RPC to the subprocess. Mars never exposes the raw external MCP tool surface to roles.

Board text can contain private program details. Durable artifacts must not store raw credentials, tokens, private board dumps, or unnecessary personal data. If raw board content is needed for diagnosis, it stays in local redacted evidence or a controlled fixture with explicit privacy classification.

Failure ownership is classified before ticketing. Connector failures, runtime policy gaps, tool behavior, and generated doctrine gaps are foundation-owned. Incorrect board content, missing business rules, unavailable program decisions, and target-specific process gaps are deployed/program-owned. Mixed or unclear failures remain blocked until evidence separates ownership.

## Step-By-Step Behavior

The scenarios below are the step-by-step BDD contract for the six sequential plans. Each scenario ships independently and preserves the default-off invariant.

## Scenario Schedule

1. F-013-S001 - Optional integrations config loads default-off and exposes a runtime flow profile without changing no-config CEO-led behavior.
2. F-013-S002 - JIRA webhook and poll ingestion mirror issues into tickets with pull-only field-level reconciliation.
3. F-013-S003 - Board-driven selection dispatches the top ready ticket by active sprint, priority, rank, and age with cost guards.
4. F-013-S004 - Frontier model overrides route roles through configured OpenAI-compatible gateways with Bearer tokens from env vars.
5. F-013-S005 - Figma context and PR delivery are gated by config, trust, and branch policy.
6. F-013-S006 - Generated deployed harness context and release notes preserve JIRA relationship and version traceability.

## Scenarios

### F-013-S001: Optionality Foundation

Given a repo has no `.harness/integrations.yaml`
When `serve`, `start`, `run`, init-generated role context, scheduler registration, and strict-trunk policy are used
Then behavior matches the current CEO-led/GitHub/trunk flow
And no JIRA webhook route is mounted
And no JIRA poller starts
And no board-driven backlog dispatch path starts work
And role effective tool allowlists remain byte-identical to the manifest
And generated manifests do not include future integration tools statically

Given `.harness/integrations.yaml` is present with an empty or unknown `flow_profile`
When integrations config is loaded
Then the effective profile is `ceo-led`
And the loader does not enable board-driven behavior implicitly

Given `.harness/integrations.yaml` sets `flow_profile: board-driven`
When the server starts or warm-restarts
Then schedules for `ceo`, `coo`, `head-of-strategy`, and `cto-weekly` are suppressed
And those roles remain available for dispatch or manual invocation
And delivery, review, security, dogfood, release, janitor, and orchestrator schedules keep their existing behavior
And startup/restart logs and status APIs expose the active profile

Given an operator runs `mars init` or `mars upgrade`
When generated harness defaults are written
Then `.harness/integrations.example.yaml` is present when missing
And `.harness/integrations.yaml` is never written by default
And existing user-owned harness files are preserved.

### F-013-S002: JIRA Mirror And Sync Model

Given JIRA ingestion is enabled and a JIRA issue maps to a registered repo
When a signed webhook or poll result is received
Then the issue is normalized without entering the GitHub webhook trigger spine
And the first sighting creates exactly one backlog ticket carrying `jira_key`, `jira_updated`, `jira_created`, `priority`, `sprint`, `sprint_active`, `rank`, `jira_status`, `blocked_by`, and `epic` front matter
And no LLM job is enqueued by the ingestion event.

Given JIRA ingestion config includes `scope.allowed_workspaces` and `scope.required_labels`
When an issue is outside the configured workspace or lacks a required label
Then the ingestion event is dropped with an operator-visible reason
And no local ticket is created or updated
And no LLM job is enqueued.

Given JIRA ingestion config sets `provider: atlassian_mcp`
When a poll sync runs
Then Mars initializes a short-lived Atlassian MCP session
And it calls only approved JIRA read/search tools
And it closes the MCP session after the sync
And any configured local proxy is terminated even when capability probing fails
And advertised write tools are never called.

Given JIRA ingestion config sets `provider: atlassian_mcp`
And config enables `mcp.proxy.transport: stdio`
And the proxy command is `npx mcp-remote https://mcp.atlassian.com/v1/mcp/authv2`
When a poll sync runs
Then Mars starts the proxy only for that sync job
And Mars speaks MCP JSON-RPC over the proxy stdin/stdout
And OAuth/session state remains owned by the external helper or environment
And Mars still applies project, workspace, board, and required-label containment before writing local tickets
And Mars terminates the proxy before the sync job completes.

Given JIRA ingestion config includes `scope.board_id`
When the Atlassian MCP provider exposes a board-aware read tool
Then Mars uses the configured board id for the read
And no board-scope warning is recorded.

Given JIRA ingestion config includes `scope.board_id`
When the Atlassian MCP provider does not expose a board-aware read tool
Then Mars records `board_scope_not_enforced_by_provider`
And it continues only under project, workspace, required-label, and JQL containment.

Given JIRA ingestion config uses env-var fields for tenant-specific IDs
When the configured env vars contain the site URL, cloud ID, board ID, allowed workspace URLs, or custom field IDs
Then Mars resolves those values at sync time before capability probing or ticket mirroring
And if any configured ID env var is missing, ingestion fails closed without widening scope or writing tickets.

Given a previously mirrored issue changes in JIRA
When the next poll or webhook refreshes the mirror
Then JIRA-owned front matter and the requirements body refresh in place
And the ticket lifecycle directory, harness-owned evidence fields, scoped marker, and appended agent notes are preserved.

### F-013-S003: Board Prioritisation And Cost Guards

Given board-driven mode is enabled and mirrored backlog tickets exist
When the orchestrator survey runs while no product ticket is in progress
Then it selects only active-sprint tickets whose `jira_status` is in the configured ready set and whose blockers are resolved
And it orders ready tickets by configured priority order, LexoRank, and `jira_created`
And it dispatches exactly one `cto-weekly` scoping job for the winner.

Given board-driven dispatch can spend paid gateway tokens
When DailyCap or the rolling spend/turn threshold is reached
Then board-driven dispatch pauses with dashboard and log signals
And no silent loop continues.

### F-013-S004: Frontier Model Parity

Given `model-overrides.yaml` configures an OpenAI-compatible provider for a role or tier
When the role runs
Then executor uses the configured endpoint and model
And it reads the Bearer token from the configured env-var name
And missing required token values fail loudly without logging token contents.

### F-013-S005: Figma And PR Delivery

Given a board-driven ticket references a Figma file and Figma integration is enabled
When a scoped role calls `figma_fetch`
Then the tool returns a bounded summary of relevant frames, text, and components
And raw file JSON, token values, and signed URLs are not written to traces.

Given `delivery.mode: pull_request` is enabled
When delivery is ready for review
Then the harness may push only a sanitized branch matching `delivery.branch_pattern`
And `github_pr_open` opens or reuses one PR for the ticket
And it never force-pushes, auto-merges, changes base away from `main`, or bypasses trust gates.

### F-013-S006: Context Mirroring And Traceability

Given a target harness is initialized or upgraded
When board-driven integration defaults are generated
Then target context includes vendor-neutral JIRA sync operating-model guidance, a knowledge route, and AGENTS wording that JIRA is the requirements source of record when enabled.

Given a shipped ticket carries `jira_key`
When release notes are generated
Then CHANGELOG entries surface the originating JIRA key alongside the Mars ticket ID.

## Out of Scope

- JIRA write-back from harness to JIRA.
- Centralized egress proxy.
- Auto-merging PRs.
- Hardcoded Example Target Project project keys, board IDs, model names, endpoints, or token values.
- Storing raw credentials, tokens, private board dumps, or unnecessary personal data in durable repo artifacts.

## Descoped Scenarios

None.

## Evidence

- F-013-S001: complete. Focused tests, `go test ./...`, `make check`, release backfill, and OpenAI-backed installed-binary clean-project validation pass. Static browser and Go API targets prove generated defaults, default `ceo-led` profile, schedule replacement, no generated `.harness/integrations.yaml`, and CEO -> COO -> CTO -> Engineer handoff with product evidence; see `docs/validation/reports/2026-06-23-example-target-project-optionality-foundation.md`.
- F-013-S002: complete for the initial webhook/REST mirror and the 2026-06-24 T-047 Atlassian MCP OAuth/stdio provider revision. Focused JIRA mirror, webhook, poll, config, and serve route tests pass; broad `go test ./...`, `make check`, explicit `go vet ./...`, and installed-binary validation passed for the initial mirror. The installed smoke proves no-config `/webhooks/jira` returns 404, missing-label DEMO issues are dropped with `scope_required_label_missing`, scoped DEMO issues with `example-required-label` create one local backlog ticket, and JIRA webhooks do not change the SQLite queue count; see `docs/validation/reports/2026-06-23-example-target-project-jira-mirror-sync.md`. The T-047 revision adds job-scoped Atlassian MCP read support, records the live Basic-token limitation plus the successful OAuth `mcp-remote` read in `docs/validation/reports/2026-06-24-example-target-project-atlassian-mcp-jira-intake.md`, and is released as `v0.65.0`.
- F-013-S003: planned selector, survey dispatch, DailyCap, circuit breaker, and dashboard/log signal tests.
- F-013-S004: planned request-capturing gateway tests for endpoint/model/Bearer behavior.
- F-013-S005: planned Figma bounded-output tests, PR policy/trust tests, and disposable repo PR validation.
- F-013-S006: planned init/upgrade golden tests and release-note `jira_key` tests.
