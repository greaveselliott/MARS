# 2026-06-23 Example Target Project JIRA Live Read Attempt

## Primary Outcome Contract

**Primary Outcome:** Verify MARS can read scoped Example Target Project JIRA issues and
generate local Markdown tickets from live JIRA without using Codex connectors.

**Primary Pass Gate:** A harness-owned JIRA poll, configured with env-var auth,
`project_repo_map`, the DEMO board workspace URL, and the
`example-required-label` label, reads at least one live issue and creates
one or more temp local tickets with `llm_jobs_enqueued:0`.

**Primary Status:** `primary_blocked`

**Current Primary Blocker:** The local JIRA credentials visible to Codex return
HTTP 401 from Atlassian Cloud identity/Agile board endpoints, so live issue
access cannot be proven yet.

**Next Primary Action:** Provide valid Atlassian Cloud credentials to the local
process through env vars, then rerun the live scoped poll against the DEMO board
and label scope.

**Supporting Evidence:** Endpoint probes confirmed the old JIRA search endpoint
is gone and the current JQL route is reachable, so the harness-owned poller was
updated to use `/rest/api/3/search/jql`.

## Evidence

- `JIRA_API_TOKEN` and `JIRA_USERNAME` were visible as environment variable
  names. Values were not printed or written.
- `GET <jira-site-url>/rest/api/3/myself` returned HTTP 401 with
  the current env-backed Basic Auth.
- `GET <jira-site-url>/rest/agile/1.0/board/<board-id>` returned HTTP
  401 with the current env-backed Basic Auth.
- `GET <jira-site-url>/rest/agile/1.0/board/<board-id>/backlog` returned
  HTTP 401 with the current env-backed Basic Auth.
- The legacy JIRA platform search endpoint used by the Plan 2 poller,
  `/rest/api/3/search`, returned HTTP 410.
- A direct endpoint probe against `/rest/api/3/search/jql` reached the current
  JQL search route, which led to the source fix in this slice.

## Source Fix

The harness poller now builds requests against
`/rest/api/3/search/jql` instead of the removed `/rest/api/3/search` endpoint.
Focused JIRA tests and broad source gates passed after the endpoint change.

## Non-Claims

- This attempt did not prove live DEMO issue access.
- This attempt did not prove live ticket generation from Example Target Project JIRA.
- No Codex connector or Atlassian MCP connector was used.
- No JIRA issue was created, updated, or deleted.

## Replay Notes

Set valid Atlassian Cloud credentials without pasting values into chat:

```fish
read -P "JIRA username/email: " jira_user
set -Ux JIRA_USERNAME $jira_user
set -e jira_user

read -s -P "JIRA API token: " jira_token
set -Ux JIRA_API_TOKEN $jira_token
set -e jira_token
```

Then rerun the harness-owned scoped poll using:

- `base_url: <jira-site-url>`
- `auth.email_env: JIRA_USERNAME`
- `auth.api_token_env: JIRA_API_TOKEN`
- `project_repo_map: [{ project: DEMO, repo: <temp repo basename> }]`
- `scope.allowed_workspaces:
  [<jira-site-url>/jira/software/c/projects/DEMO/boards/<board-id>/backlog]`
- `scope.required_labels: [example-required-label]`
