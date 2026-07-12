# Open-Source Webhook Ingress Installed Validation

## Primary Outcome Contract

**Primary Outcome:** publish MARS as a supported open-source project without
  exposing confidential material, weakening controls, or distributing unsafe
  or unverifiable binaries.
**Primary Pass Gate:** logged-out users can clone, build, install, update,
  report vulnerabilities, and submit safe fork PRs; history is approved;
  runtime P0s are closed; and public artifacts are licensed, signed, and
  traceable.
**Primary Status:** `primary_blocked`

**Current Primary Blocker:** legal ownership/licensing clearance and the
  remaining F-017 runtime, release, public-access, community, audit, and cutover
  gates are incomplete.
**Next Primary Action:** integrate T-057 as the completed webhook-ingress
  slice, keep F-017-S002 incomplete, and deliver the next bounded runtime P0
  ticket without changing visibility.
**Supporting Evidence:** the installed T-057 candidate passed the loopback,
  disabled-policy, authorization, queue-mutation, and durable-replay checks in
  this report on clean static-browser and API/service targets.

## T-057 Dogfood Result

**Result:** PASS for the bounded T-057 installed-binary ingress contract.

The candidate bound both HTTP surfaces only to explicit IPv4 loopback, kept
health available with optional GitHub policy absent, rejected non-loopback
configuration before bind, left the real SQLite queue unchanged for rejected
traffic, enqueued one authorized event exactly once, and did not recreate that
work after terminal job state and process restart. No GitHub or other remote
state was used or mutated.

## Matrix Selection And Candidate Identity

- **Source-change class:** orchestration/control-plane ingress; AD-284 minimum
  is the static-browser archetype plus API/service or CLI/tooling.
- **Selected archetypes:** clean static-browser and clean Go API/service.
- **Source ref:** `d1653e79d9abd9669d405da9f6941c172d359660`
  plus the uncommitted T-057 candidate diff.
- **Pre-report candidate diff SHA-256:**
  `5493410ff32b09a17ab45c68f6780a675cc3f5a00d7adf8060463b0b49729cac`.
- **Installed command:** `$HOME/go/bin/mars`, installed with
  `make install`; reported `mars 0.68.45 darwin/arm64 commit=unknown
  built=unknown`.
- **Installed binary SHA-256:**
  `8ad1b1cb1cdf4ae759af81768e49f5099f59869b1ef4e847708726593e12b271`.
- **Target paths:**
  `/private/tmp/mars-t057-dogfood/static-browser` and
  `/private/tmp/mars-t057-dogfood/api-service`.
- **Target refs after validation:** static
  `631f7201cc0f5d918ed9f99e937f1ffb2ef8020c`; API
  `98c52774bd302c9944a854af3015258667970889`.
- **Database paths:** `/private/tmp/mars-t057-dogfood/static.db` and
  `/private/tmp/mars-t057-dogfood/api.db`.
- **Log paths:** the run used isolated `static-*.log`, `api-*.log`, and
  `reject-*.log` files below `/private/tmp/mars-t057-dogfood/`; all were
  removed during cleanup after the summarized evidence was recorded.
- **Trace evidence:** one API pipeline-fixer trace existed in the isolated API
  database before cleanup. The static HTTP/queue smoke invoked no model.

## Commands

The exact installed server commands were:

```bash
make install
MARS_DASHBOARD_PORT=19390 mars serve --addr 127.0.0.1:19391 --db /private/tmp/mars-t057-dogfood/static.db --debug --log-file /private/tmp/mars-t057-dogfood/static-no-policy.log --code-intel=false
MARS_DASHBOARD_PORT=19500 mars serve --addr 127.0.0.1:19501 --db /private/tmp/mars-t057-dogfood/api.db --debug --log-file /private/tmp/mars-t057-dogfood/api-no-policy.log --code-intel=false
MARS_DASHBOARD_PORT=19400 MARS_WEBHOOK_ALLOWED_ACTOR_IDS=42 MARS_WEBHOOK_SECRET="<owner-only temporary secret>" mars serve --addr 127.0.0.1:19401 --db /private/tmp/mars-t057-dogfood/static.db --debug --log-file /private/tmp/mars-t057-dogfood/static-auth.log --code-intel=false
MARS_DASHBOARD_PORT=19502 MARS_WEBHOOK_ALLOWED_ACTOR_IDS=42 MARS_WEBHOOK_SECRET="<owner-only temporary secret>" mars serve --addr 127.0.0.1:19503 --db /private/tmp/mars-t057-dogfood/api.db --debug --log-file /private/tmp/mars-t057-dogfood/api-auth-live.log --code-intel=false --concurrency 1
```

The three direct control-address probes used `0.0.0.0`, `192.0.2.10`, and an
arbitrary hostname. A scoped `mars start` probe used a loopback control address
and `0.0.0.0` for `--dashboard-addr`. Each command exited nonzero before bind
with the documented loopback/reverse-proxy remediation.

Signed-request commands used `curl --data-binary`, HMAC-SHA256 from an
owner-only generated temporary secret, and ephemeral realistic nested GitHub
fixtures. Exact secret bytes and payload bodies are deliberately excluded from
the repository, terminal summary, logs, and this report. This is the required
security redaction exception to AD-285's exact-command rule; event type,
expected policy boundary, HTTP status, queue delta, and replay result are
recorded below.

## Per-Case Evidence

| Archetype | Case | Result | Queue evidence |
| --- | --- | --- | --- |
| Static browser | Actual listener inspection | control `127.0.0.1:19391`, dashboard `127.0.0.1:19390` | not applicable |
| API/service | Actual listener inspection | control `127.0.0.1:19501`, dashboard `127.0.0.1:19500` | not applicable |
| Both | GET `/healthz` without webhook policy | HTTP 200 | zero jobs, zero receipts |
| Both | POST `/webhook` without webhook policy | HTTP 503 | zero jobs, zero receipts |
| Static browser | invalid HMAC | HTTP 401 | zero jobs, zero receipts |
| Static browser | signed untrusted actor | HTTP 202 | zero jobs, zero receipts |
| Static browser | signed wrong repository | HTTP 202 | zero jobs, zero receipts |
| Static browser | signed wrong branch | HTTP 202 | zero jobs, zero receipts |
| Static browser | signed fork pull request | HTTP 202 | zero jobs, zero receipts |
| Static browser | signed issue comment | HTTP 202 | zero jobs, zero receipts |
| Static browser | signed trusted exact workflow event | HTTP 200 | one pipeline-fixer job, one receipt |
| Static browser | duplicate delivery and same body under a new delivery ID | HTTP 200 | remained one job and one receipt before and after restart |
| API/service | signed trusted exact workflow event | HTTP 200 | one webhook-created pipeline-fixer job, one receipt |
| API/service | same delivery and same body under a new delivery ID after completion/restart | HTTP 200 | remained one webhook-created job and one receipt |
| Both | wildcard, LAN, hostname, and wildcard dashboard configuration | rejected before bind | no listener or queue mutation |

The static worker pool was paused before ingress. Clean shutdown transitioned
the single pending job to `cancelled`; durable replay still remained single-shot
after restart. The API run supplied stronger completion evidence: the
webhook-created pipeline-fixer job reached `completed`, then two post-restart
replays did not create a second webhook job or receipt.

## Real-Model Lifecycle Evidence

The API/service webhook job ran through the real installed server worker and
the configured local inference router. No fake, stub, canned, mock, or scripted
endpoint was used.

- **Coding tier:** `Qwen3-Coder-30B-A3B-Instruct`, `Q4_K_M`, immutable revision
  `a000510ef6de0a66dafa731c2d8d712a96fa7009`, pinned SHA-256
  `79ad15a5ee3caddc3f4ff0db33a14454a5a3eb503d7fa1c1e35feafc579de486`,
  context `32768`, balanced performance profile.
- **Webhook job:** pipeline-fixer, 18 turns, 17 tool invocations, about 14,845
  tokens, 60.7 seconds; harness job status `completed` with a recorded failed
  disposition because the deliberately minimal target had no CI failure
  artifact or tests to repair.
- **Product progress:** `go test ./...` passed; no product file changed. MARS
  committed only its bounded runtime learning artifact, leaving the target
  worktree clean at `98c52774`.
- **Follow-on reasoning tier:** the same Qwen model at context `131072` began
  server startup for an automatically routed Orchestrator job. Dogfood stopped
  the run after the webhook-owned job completed, so this follow-on recorded an
  operator-induced inference cancellation and is excluded from lifecycle pass
  claims.
- **Target intervention debt:** zero for both targets. Guardrail blocks from
  model-generated unsafe or invalid local commands stayed foundation telemetry
  and did not become target backlog.

## Failure Ownership And Stop Reason

- **T-057 boundary result:** `foundation-owned`, pass. The claimed listener,
  policy, authorization, zero-mutation, and replay failure signatures did not
  recur.
- **Minimal API target lacked a real CI failure artifact:** `deployed-owned
  validation-fixture limitation`; the pipeline-fixer accurately recorded a
  failed disposition after bounded inspection. This does not affect ingress
  authorization or durable queue evidence.
- **Follow-on Orchestrator inference cancellation:** `evidence-only,
  operator-induced`; Dogfood stopped after the webhook-derived job completed to
  prevent unrelated autonomous work. It is not counted as a T-057 failure.
- **Static full model lifecycle:** not required for the bounded ingress result;
  the static case supplied independent installed listener, no-policy,
  rejection, authorization, and restart evidence, while the API case supplied
  real-model completion evidence.

## Cleanup

All installed-candidate control, dashboard, and inference listeners were
closed. Final socket inspection found no listener on the test ports or local
model ports. Both target worktrees were clean, and all isolated target, DB,
log, trace, secret, and fixture files under
`/private/tmp/mars-t057-dogfood/` were removed after evidence extraction. The
source repository remained limited to the pre-existing T-057 candidate plus
this report and its ticket evidence.
