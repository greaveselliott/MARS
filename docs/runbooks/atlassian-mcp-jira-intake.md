# Atlassian MCP JIRA Intake Runbook

**Status:** Draft
**Updated:** 2026-06-24
**Owner:** foundation-maintainer
**Related:** [F-013-board-driven-integrations.md](../features/F-013-board-driven-integrations.md), [board-driven-integrations.md](../design-docs/board-driven-integrations.md)

Use this runbook when a deployed harness should mirror scoped JIRA issues
through Atlassian's official MCP server. It is operator-facing procedure, not a
replacement for the F-013 feature contract.

## Safety Rules

- Keep JIRA intake read-only: no comments, transitions, issue updates, or issue
  creation.
- Do not expose raw Atlassian MCP tools to agent roles.
- Do not commit secret values, OAuth cache files, account IDs, cloud IDs, raw
  issue IDs, or private board dumps.
- Prefer env-var indirection for tenant-specific values. Mars supports env-var
  names for `base_url`, `mcp.site_url`, `mcp.cloud_id`, workspace scope URLs,
  board IDs, and instance-specific custom field IDs.
- Prefer `mcp.site_url` over `mcp.cloud_id` when the Atlassian MCP tool accepts
  the site URL. Mars falls back from `cloud_id` to `site_url` and then
  `base_url` when calling Atlassian MCP; committed configs should usually use
  `site_url_env` and `base_url_env` instead of direct tenant URLs.
- Commit only scope values that the operator explicitly allows in that repo.
  If board IDs or board URLs are considered internal identifiers, keep
  `.harness/integrations.yaml` local/untracked or use project, label, and JQL
  containment until a secret-backed config path exists.

## Prerequisites

- A Mars Harness build that includes T-047.
- Node.js and `npx` available on the machine that runs the harness.
- Browser/SSO access for the Atlassian user that will authorize MCP.
- Operator-approved values for:
  - Atlassian site URL, for example `<site>.atlassian.net`.
  - JIRA project key, for example `<project-key>`.
  - Required intake label, for example `<required-label>`.
  - Target repo name or repo path mapping.
  - Optional board backlog URL or board ID when these are allowed in repo config.

## Configuration

Copy `.harness/integrations.example.yaml` to `.harness/integrations.yaml` only
in the target repo that should ingest JIRA issues. Keep values as placeholders
until the operator has approved what may be committed.

```yaml
version: 1
flow_profile: board-driven

ingestion:
  jira:
    enabled: true
    provider: atlassian_mcp
    base_url: ""
    base_url_env: JIRA_SITE_URL
    auth:
      email_env: JIRA_EMAIL
      api_token_env: JIRA_API_TOKEN
      bearer_token_env: ""
    mcp:
      endpoint_url: https://mcp.atlassian.com/v1/mcp/authv2
      cloud_id: ""
      cloud_id_env: "" # set to ATLASSIAN_CLOUD_ID only when site_url is not accepted
      site_url: ""
      site_url_env: JIRA_SITE_URL
      timeout: 30s
      proxy:
        enabled: true
        transport: stdio
        command: npx
        args: ["-y", "mcp-remote", "https://mcp.atlassian.com/v1/mcp/authv2"]
        env_passthrough: []
    poll_interval: 60s
    jql: 'project = "<project-key>" AND labels = "<required-label>"'
    project_repo_map:
      - { project: "<project-key>", repo: "<repo-name-or-path>" }
    scope:
      allowed_workspaces: []
      allowed_workspaces_env: JIRA_ALLOWED_WORKSPACES
      required_labels:
        - <required-label>
      board_id: ""
      board_id_env: JIRA_BOARD_ID
    fields:
      sprint: ""
      sprint_env: JIRA_FIELD_SPRINT
      rank: ""
      rank_env: JIRA_FIELD_RANK
      epic_link: ""
      epic_link_env: JIRA_FIELD_EPIC
      story_points: ""
      story_points_env: JIRA_FIELD_STORY_POINTS
```

Set the tenant-specific and ID-bearing values in the process environment instead
of committing them:

```bash
set -gx JIRA_SITE_URL "https://<site>.atlassian.net"
set -gx JIRA_ALLOWED_WORKSPACES "https://<site>.atlassian.net/jira/software/c/projects/<project-key>/boards/<board-id>/backlog"
set -gx JIRA_BOARD_ID "<board-id>"
set -gx ATLASSIAN_CLOUD_ID "<cloud-id>" # only when cloud_id_env is configured
set -gx JIRA_FIELD_SPRINT "<sprint-custom-field-id>"
set -gx JIRA_FIELD_RANK "<rank-custom-field-id>"
set -gx JIRA_FIELD_EPIC "<epic-custom-field-id>"
set -gx JIRA_FIELD_STORY_POINTS "<story-points-custom-field-id>"
```

If board IDs or board URLs must not be committed and cannot be supplied through
environment variables, leave `board_id_env` and `allowed_workspaces_env` empty.
Keep project, required-label, and JQL containment in place. The MCP provider
records `board_scope_not_enforced_by_provider` when no board-aware read tool is
available, so this trade-off remains visible.

## First-Run Auth

The first stdio-proxy sync may open a browser for Atlassian OAuth/SSO:

```bash
npx -y mcp-remote https://mcp.atlassian.com/v1/mcp/authv2
```

Complete the browser flow as the intended Atlassian user. The external helper
owns its OAuth/session cache. Mars does not store token values in repo files,
tickets, traces, or generated docs.

## Verification

For source validation, run the live smoke with repo-safe placeholders. Prefer
omitting `MARS_JIRA_CLOUD_ID` when `MARS_JIRA_SITE_URL` works for the site.

```bash
MARS_JIRA_LIVE=1 \
MARS_JIRA_MCP_STDIO_PROXY=1 \
MARS_JIRA_MCP_ENDPOINT=https://mcp.atlassian.com/v1/mcp/authv2 \
MARS_JIRA_SITE_URL=https://<site>.atlassian.net \
GOCACHE=<validation-root> \
go test -count=1 ./internal/jira -run TestLiveAtlassianMCPJiraIntake -v
```

Expected result:

- `tools/list` advertises `searchJiraIssuesUsingJql`.
- Mars calls only approved read/search tools.
- Matching JIRA issues are mirrored into local Markdown tickets by `jira_key`.
- `llm_jobs_enqueued` remains `0` for ingestion.
- The stdio proxy is terminated before the sync completes.

## Common Failures

| Symptom | Meaning | Operator action |
| --- | --- | --- |
| Only Teamwork Graph tools are listed | The direct HTTP/API-token path is active or the credential lacks the JIRA MCP surface. | Use `mcp.proxy.transport: stdio` with `mcp-remote` and complete OAuth/SSO. |
| `scope_workspace_mismatch` | The issue URL/site/project does not match configured workspace containment. | Check `base_url`, `site_url`, `project_repo_map`, and any allowed workspace URL. |
| `scope_required_label_missing` | The issue does not carry every configured required label. | Add the approved intake label in JIRA or change config deliberately. |
| No tickets are created | JQL returned no eligible issues or all issues were dropped by containment. | Run a read-only JQL check and inspect dropped reasons. |
| Browser auth timeout | OAuth was not completed before the helper timed out. | Complete SSO, then rerun the sync. |
| `npx` not found | The external stdio helper runtime is unavailable. | Install Node.js or configure another approved stdio MCP helper. |

## Completion Evidence

Record the validation result in `docs/validation/reports/` with:

- Exact command, with internal IDs redacted or replaced by placeholders.
- Whether direct HTTP/API-token auth failed closed.
- Whether stdio OAuth exposed `searchJiraIssuesUsingJql`.
- Count of mirrored tickets, not raw issue dumps.
- Confirmation that no JIRA write tools were called.
- Confirmation that no LLM jobs were enqueued by ingestion.
