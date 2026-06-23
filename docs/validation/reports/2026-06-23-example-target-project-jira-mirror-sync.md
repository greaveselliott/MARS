# 2026-06-23 Example Target Project JIRA Mirror And Sync Validation

## Primary Outcome Contract

**Primary Outcome:** Prove Plan 2 of F-013: board-driven JIRA webhook and poll
ingestion can mirror scoped issues into local tickets while JIRA remains the
source of record and no JIRA event enqueues LLM work directly.

**Primary Pass Gate:** Missing integrations config keeps the JIRA route
disabled. Board-driven JIRA config requires explicit project-to-repo mapping
and optional workspace/label scope before any local ticket write. First sighting
creates one backlog ticket by `jira_key`; later sightings reconcile JIRA-owned
fields without clobbering harness-owned lifecycle/evidence/notes. Broad source
gates and installed-binary smoke pass.

**Primary Status:** `primary_passed`

**Current Primary Blocker:** None.

**Next Primary Action:** Start Plan 3 from `T-046` only; do not add Figma, PR
delivery, or frontier model-routing implementation code in the prioritisation
and cost-guard slice.

**Supporting Evidence:** Focused JIRA mirror/webhook/poll tests, serve route
tests, docs consistency, broad `go test ./...`, `make check`, and
installed-binary smoke passed for the Plan 2 closure scope.

## Matrix Selection

- **Selected Matrix:** AD-284 minimum adapted for a foundation runtime
  ingestion change.
- **Selected Cases Or Archetypes:** Default-off route behavior, board-driven
  scoped JIRA webhook ingestion, generated defaults, SQLite queue containment,
  and installed-binary local control-plane smoke.
- **Source Ref:** Local working tree on `main` during Plan 2 validation on
  2026-06-23.
- **Binary:** Installed `/path/to/local-redacted` from the
  current source tree with `make install`.
- **Model Identity:** Installed-binary JIRA smoke used
  `--model-endpoint http://127.0.0.1:9999/v1` because no JIRA event should
  create model work. No OpenAI or Atlassian network call was required.
- **Target/Run Paths:** Source repo
  `/path/to/local-redacted`; installed
  smoke root `<validation-root>`.
- **Cleanup Status:** The temporary harness processes were stopped after the
  webhook evidence was captured. The smoke targets remain under `/tmp`
  for inspection.

## Commands And Results

| Command | Result | Notes |
| --- | --- | --- |
| `git diff --check` | PASS | No whitespace errors. |
| `go test -count=1 ./internal/jira ./internal/integrations ./internal/serve -run 'TestMirror|TestWebhook|TestPoll|TestServerJIRA|TestLoad_boardDrivenConfig'` | PASS | Focused Plan 2 mirror, webhook, poll, config, and serve route coverage. |
| `go test -count=1 ./internal/scanner ./internal/docsync ./internal/docsconsistency` | PASS | Generated defaults and docsync/docs consistency remained green. |
| `go test -cover ./internal/jira` | PASS | `internal/jira` coverage was 75.1%, above the 70% ratchet floor. |
| `GOCACHE=<validation-root> go test ./...` | PASS | Broad source gate passed. |
| `GOCACHE=<validation-root> make check` | PASS | Race/coverage, coverage ratchet, fuzz smoke, and vet path passed. A race found earlier in `record_decision` role attribution was fixed before this pass. |
| `GOCACHE=<validation-root> go vet ./...` | PASS | Explicit vet fallback confirmation after `make check`. |
| `make install` | PASS | Installed current binary to `/path/to/local-redacted`; fish PATH already configured. |
| Installed binary no-config smoke | PASS | `POST /webhooks/jira` returned `404`; `.harness/integrations.example.yaml` existed; `.harness/integrations.yaml` was not written. |
| Installed binary scoped board-driven smoke | PASS | Missing-label DEMO issue returned `status=dropped`, `reason=scope_required_label_missing`, and `llm_jobs_enqueued=0`; no ticket was written. |
| Installed binary scoped create smoke | PASS | DEMO issue with the configured workspace and `example-required-label` label returned `status=created`, wrote `docs/tickets/backlog/T-001-scoped-example-required-label.md`, and `llm_jobs_enqueued=0`. |
| SQLite queue containment smoke | PASS | Queue count stayed `1` before and after the JIRA webhooks; the existing count was the normal `start` seed job, not JIRA work. |

## Config Containment Evidence

The installed smoke used this board-driven scope shape in the target repo's
`.harness/integrations.yaml`:

```yaml
flow_profile: board-driven
ingestion:
  jira:
    enabled: true
    base_url: <jira-site-url>
    webhook_secret_env: MARS_VALIDATION_JIRA_WEBHOOK_SECRET
    project_repo_map:
      - project: DEMO
        repo: board-driven-repo
    scope:
      allowed_workspaces:
        - <jira-site-url>/jira/software/c/projects/DEMO/boards/<board-id>/backlog
      required_labels:
        - example-required-label
```

The workspace URL and label are configuration values only. The source code keeps
the scope logic generic and does not hardcode Example Target Project project keys, board IDs, or
labels as constants.

## Findings Closed

- **Default-off JIRA route:** Missing config returns disabled integrations, and
  the installed no-config route returned 404.
- **Config-contained blast radius:** Explicit `project_repo_map`,
  `scope.allowed_workspaces`, and `scope.required_labels` gate ticket writes.
- **No fan-out:** Unmapped or ambiguous projects drop instead of writing to all
  repos.
- **No direct LLM work:** Webhook and poll results report
  `llm_jobs_enqueued:0`, and installed smoke confirmed queue count did not
  change across JIRA webhook delivery.
- **Pull-only reconciliation:** Tests preserve lifecycle directory, evidence
  fields, scoped marker, and agent notes byte-for-byte while updating JIRA-owned
  fields/body.
- **Secret handling:** Config stores env-var names only. Signature and poll
  auth tests verify missing env vars name the variable, not the value, and
  secret-like payload text is redacted from mirrored tickets.

## Residual Observations

- Plan 2 intentionally does not select ready board work or dispatch
  `cto-weekly`. That begins in Plan 3 with cost guards.
- Plan 2 intentionally does not write back to JIRA.
- The installed smoke uses an inert local model endpoint because direct model
  execution is a non-goal for JIRA ingestion events.

## Replay Commands

```bash
GOCACHE=<validation-root> go test ./...
GOCACHE=<validation-root> make check
GOCACHE=<validation-root> go vet ./...
make install
# Re-run the installed smoke by replaying the retained shell transcript logic
# against a new <validation-root> root.
```
