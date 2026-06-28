# 2026-06-24 Example Target Project Atlassian MCP JIRA Intake Validation

## Primary Outcome Contract

**Primary Outcome:** Prove the Plan 2 provider revision: board-driven JIRA
polling can use Atlassian's official MCP server through a job-scoped read
interaction while MARS remains default-off, read-only, and responsible
for project/workspace/board/label containment.

**Primary Pass Gate:** Focused MCP/JIRA/config tests pass. The MCP path
initializes a short-lived session, probes tools, calls only approved read/search
tools, closes the session, terminates any configured local proxy on error, and
mirrors only scoped issues by `jira_key`. Live Example Target Project read verification either
creates scoped local tickets for `DEMO` + `example-required-label` or
records a precise auth/scope/capability blocker.

**Primary Status:** `primary_passed`

**Current Primary Blocker:** None for Atlassian MCP read capability. The
remaining work is the normal semantic commit, release notes, backfill, tag,
asset publication, and release verification for the source change.

**Next Primary Action:** Commit the validated T-047 source/docs changes, run the
release-note and release-asset workflow, then resume Plan 3 from T-046.

**Supporting Evidence:** Focused MCP client, JIRA, integrations, scanner,
docsync, docs consistency, broad Go tests, `make check`, and live Example Target Project
OAuth-backed MCP smoke pass for T-047. The provider remains read-only for Plan
2: no comments, transitions, issue updates, raw external MCP tool exposure, or
direct LLM job enqueue.

## Evidence

| Check | Result | Notes |
| --- | --- | --- |
| `GOCACHE=<validation-root> go test -count=1 ./internal/mcpclient` | PASS | HTTP and stdio MCP clients cover initialize, tools/list, tools/call, session/header behavior, SSE parsing, RPC errors, malformed stdio output, subprocess close, and response-id matching. Run outside sandbox because `httptest` listeners and subprocess helpers are blocked or flaky inside the Codex sandbox. |
| `GOCACHE=<validation-root> go test -count=1 ./internal/jira ./internal/integrations` | PASS | JIRA MCP tests cover scoped DEMO ticket mirroring, advertised write tool not called, missing label drop, stdio proxy without static auth, env-backed tenant URL/ID resolution and fail-closed behavior, board read tool selection with `boardId`, board warning when unsupported, missing read capability fail-closed, API-gateway URL containment, and proxy cleanup on error. Run outside sandbox because `httptest` listeners are blocked inside the Codex sandbox. |
| `GOCACHE=<validation-root> go test -count=1 ./internal/mcpclient ./internal/jira ./internal/integrations ./internal/scanner ./internal/docsconsistency ./internal/docsync` | PASS | Focused implementation, generated example, docsync, docs consistency, and coverage-floor gates pass. |
| `git diff --check` | PASS | No whitespace errors. |
| `GOCACHE=<validation-root> go test ./...` | PASS | Broad Go test suite passed outside the sandbox. |
| `GOCACHE=<validation-root> make check` | PASS | Build, race/coverage tests, coverage ratchet, fuzz smokes, and `go vet ./...` fallback passed. `govulncheck` and `golangci-lint` are not installed locally, so make used its documented skip/fallback behavior. |
| `MARS_JIRA_LIVE=1 MARS_JIRA_CLOUD_ID=<cloud-id-or-empty-when-site-url-works> MARS_JIRA_SITE_URL=<jira-site-url> GOCACHE=<validation-root> go test -count=1 ./internal/jira -run TestLiveAtlassianMCPJiraIntake -v` | BLOCKED | The MCP session reached Atlassian and `tools/list` succeeded, but available tools were only `addTeamworkGraphContext`, `getTeamworkGraphContext`, and `getTeamworkGraphObject`. `searchJiraIssuesUsingJql` was not available, so no local tickets were generated. |
| `MARS_JIRA_LIVE=1 MARS_JIRA_MCP_ENDPOINT=https://mcp.atlassian.com/v1/mcp/authv2 MARS_JIRA_CLOUD_ID=<cloud-id-or-empty-when-site-url-works> MARS_JIRA_SITE_URL=<jira-site-url> GOCACHE=<validation-root> go test -count=1 ./internal/jira -run TestLiveAtlassianMCPJiraIntake -v` | BLOCKED | The documented `authv2` endpoint produced the same limited Teamwork Graph tool surface, so the remaining blocker is credential/permission-group availability rather than endpoint selection. |
| Manual diagnostic: `npx -y mcp-remote https://mcp.atlassian.com/v1/mcp/authv2` plus MCP `initialize`, `tools/list`, and scoped `searchJiraIssuesUsingJql` call | PASS | The OAuth flow completed through the operator's SSO browser session, exposed the JIRA read/search tools, and returned DEMO issues carrying `example-required-label`. This proved the blocker was the direct API-token HTTP path, not Atlassian MCP read availability. Raw issue content is intentionally not stored in this report. |
| `MARS_JIRA_LIVE=1 MARS_JIRA_MCP_STDIO_PROXY=1 MARS_JIRA_MCP_ENDPOINT=https://mcp.atlassian.com/v1/mcp/authv2 MARS_JIRA_CLOUD_ID=<cloud-id-or-empty-when-site-url-works> MARS_JIRA_SITE_URL=<jira-site-url> GOCACHE=<validation-root> go test -count=1 ./internal/jira -run TestLiveAtlassianMCPJiraIntake -v` | PASS | Mars spawned `npx mcp-remote` as a job-scoped stdio MCP proxy, reused the OAuth session, probed tools, called only the approved JIRA read/search tool, applied DEMO workspace/project/label containment, mirrored a local temp ticket, and closed the session. |
| `env -u MARS_JIRA_CLOUD_ID MARS_JIRA_LIVE=1 MARS_JIRA_MCP_STDIO_PROXY=1 MARS_JIRA_MCP_ENDPOINT=https://mcp.atlassian.com/v1/mcp/authv2 MARS_JIRA_SITE_URL=<jira-site-url> GOCACHE=<validation-root> go test -count=1 ./internal/jira -run TestLiveAtlassianMCPJiraIntake -v` | PASS | The live smoke also passed with `mcp.cloud_id` unset, proving the repo does not need to store the concrete Atlassian cloud ID when `site_url` is configured. |

## Live Read Status

Passed through the OAuth-backed stdio proxy route. Direct HTTP with the
operator's API-token credential still exposes only Teamwork Graph tools and
therefore fails closed before search or ticket writes. The supported route for
this workspace is a job-scoped stdio proxy such as `npx mcp-remote` against
`https://mcp.atlassian.com/v1/mcp/authv2`, with Mars owning tool allowlisting,
containment, local mirroring, and cleanup. The validated config can leave
`mcp.cloud_id` blank and use env-backed `mcp.site_url` instead, avoiding a
checked-in cloud ID or tenant URL.

No JIRA write path is implemented or exercised by this revision.
