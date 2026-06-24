---
id: T-047
title: Add ephemeral Atlassian MCP Jira intake
priority: high
complexity: medium
work_type: enabler
bdd_scenarios: []
end_to_end_evidence: required
evidence_links: ["docs/validation/reports/2026-06-24-example-target-project-atlassian-mcp-jira-intake.md"]
verified_by: "focused tests; live OAuth MCP smoke; go test ./...; make check"
owner: "foundation-maintainer"
last_attempt: "2026-06-24"
blocker: "none"
blocked_by: ["T-045"]
trace_id: "TBD"
next_action: "Plan 2 provider revision is complete and released as v0.65.0; resume Plan 3 from T-046."
dedupe_key: "public-example"
source: operator request 2026-06-24 — Plan 2 Revision: Ephemeral Atlassian MCP Jira Intake
created: 2026-06-24
depends_on: [T-045]
---

# T-047: Add ephemeral Atlassian MCP Jira intake

## Context

The initial Plan 2 JIRA poll path was direct REST and live verification exposed credential/scope and endpoint uncertainty. The operator selected Atlassian official MCP as the preferred JIRA read provider, used ephemerally per sync job. This is a Plan 2 revision/enabler, not a new F-013 scenario.

## Requirements

- Add provider config for `atlassian_mcp` under `.harness/integrations.yaml`.
- Use env-var names only for auth and never store token values.
- Use env-var indirection for tenant-specific values that should not be committed, including JIRA site URLs, Atlassian cloud IDs, board IDs, ID-bearing workspace URLs, and JIRA custom-field IDs.
- Implement Go MCP clients for short-lived HTTP and stdio initialize, capability probe, read tool call, and close.
- Keep optional proxy lifecycle job-scoped and terminate it even on errors.
- Support an ephemeral stdio OAuth proxy such as `npx mcp-remote` without embedding Node or Python in the Mars binary.
- Enforce Mars-owned project, workspace, board capability, and required-label containment before ticket writes.
- Keep JIRA intake read-only and do not expose raw MCP tools to agents.
- Mirror accepted issues by `jira_key` through the existing pull-only ticket reconcile path.

## Acceptance Criteria

- [x] Focused MCP/JIRA/config tests pass.
- [x] Advertised write tools are never called.
- [x] Missing required MCP read capability fails closed.
- [x] Board id is used when a board read tool is available; otherwise a warning records that provider board scope is unavailable.
- [x] ID-bearing config values can be supplied from environment variables and fail closed when configured env vars are missing.
- [x] Live Example Target Project read smoke passes through the OAuth-backed stdio MCP proxy.

## Evidence

- PASS: `GOCACHE=<validation-root> go test -count=1 ./internal/mcpclient ./internal/jira ./internal/integrations ./internal/scanner ./internal/docsconsistency ./internal/docsync`
- PASS: `git diff --check`
- PASS: `GOCACHE=<validation-root> go test ./...`
- PASS: `GOCACHE=<validation-root> make check`
- PASS: `mars-harness release backfill-notes --repo . --check`
- PASS: `mars-harness release publish-assets --repo . --version v0.65.0 --upload auto`
- PASS: `mars-harness release verify-assets --dist dist/releases --version v0.65.0`
- PASS: GitHub release mirror published for `v0.65.0`.
- BLOCKED: `MARS_JIRA_LIVE=1 MARS_JIRA_CLOUD_ID=<cloud-id-or-empty-when-site-url-works> MARS_JIRA_SITE_URL=<jira-site-url> GOCACHE=<validation-root> go test -count=1 ./internal/jira -run TestLiveAtlassianMCPJiraIntake -v`
- BLOCKED: `MARS_JIRA_LIVE=1 MARS_JIRA_MCP_ENDPOINT=https://mcp.atlassian.com/v1/mcp/authv2 MARS_JIRA_CLOUD_ID=<cloud-id-or-empty-when-site-url-works> MARS_JIRA_SITE_URL=<jira-site-url> GOCACHE=<validation-root> go test -count=1 ./internal/jira -run TestLiveAtlassianMCPJiraIntake -v`
- PASS: manual `npx -y mcp-remote https://mcp.atlassian.com/v1/mcp/authv2` diagnostic with OAuth/SSO exposed `searchJiraIssuesUsingJql` and returned scoped DEMO issues carrying `example-required-label`.
- PASS: `MARS_JIRA_LIVE=1 MARS_JIRA_MCP_STDIO_PROXY=1 MARS_JIRA_MCP_ENDPOINT=https://mcp.atlassian.com/v1/mcp/authv2 MARS_JIRA_CLOUD_ID=<cloud-id-or-empty-when-site-url-works> MARS_JIRA_SITE_URL=<jira-site-url> GOCACHE=<validation-root> go test -count=1 ./internal/jira -run TestLiveAtlassianMCPJiraIntake -v`
- PASS: `env -u MARS_JIRA_CLOUD_ID MARS_JIRA_LIVE=1 MARS_JIRA_MCP_STDIO_PROXY=1 MARS_JIRA_MCP_ENDPOINT=https://mcp.atlassian.com/v1/mcp/authv2 MARS_JIRA_SITE_URL=<jira-site-url> GOCACHE=<validation-root> go test -count=1 ./internal/jira -run TestLiveAtlassianMCPJiraIntake -v`

Live result: direct HTTP/API-token auth still exposes only Teamwork Graph tools
and fails closed. OAuth via a job-scoped stdio `mcp-remote` proxy exposes the
JIRA read/search tools, returns DEMO issues with the required label, and Mars
mirrors a local temp ticket after applying project, workspace, and label
containment. The passing live smoke does not require checking a concrete cloud
ID into the repo because `mcp.site_url` can be used as the cloud identifier. No
JIRA write was attempted.
