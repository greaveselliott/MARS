---
id: T-057
title: Default control listeners to loopback and fail closed on untrusted GitHub webhooks
priority: high
complexity: large
work_type: enabler
bdd_scenarios: ["F-017-S002", "F-011-S003", "F-011-S006", "F-006-S004", "F-010-S023"]
end_to_end_evidence: required
evidence_links:
  - docs/validation/reports/2026-07-12-open-source-webhook-ingress.md#t-057-dogfood-result
verified_by: "QA, Security, Dogfood, and foundation-maintainer Orchestrator"
owner: "engineer"
last_attempt: "2026-07-12"
blocker: "none"
blocked_by: []
trace_id: "docs/validation/reports/2026-07-12-open-source-webhook-ingress.md"
next_action: "No T-057 action remains; private v0.68.46 is verified and the next bounded F-017-S002 dashboard/browser slice is planned separately."
dedupe_key: "open-source:loopback-webhook-ingress"
metadata:
  classification: "foundation-owned,mirrored-doctrine"
  primary_status: "primary_blocked"
  technical_lane: "authorized"
source: docs/exec-plans/active/current-operating-plan.md — F-017-S002 OSS-02 webhook ingress
created: 2026-07-12
depends_on: [T-055, T-056]
---

# T-057: Default control listeners to loopback and fail closed on untrusted GitHub webhooks

## Context

MARS currently defaults control and dashboard listeners to wildcard addresses. GitHub webhook signatures are optional, start omits the secret, empty registered remotes match every repository, actor/branch/fork provenance is not authorized, issue comments dispatch by default, and replay protection is in-memory and lost on restart.

This ticket closes the listener and GitHub ingress P0 boundary only. Dashboard authentication/browser hardening, GitHub setup nonce, remote telemetry authentication, filesystem containment, and execution profiles remain separate tickets.

## Required Behavior

### Listener boundary

- start, serve, direct server construction, and dashboard construction default to 127.0.0.1 ports; scoped conflict fallback remains 127.0.0.1:0.
- Accept literal localhost, IPv4 loopback, and IPv6 ::1. Reject wildcard, LAN, arbitrary hostname, and other non-loopback control/dashboard binds with actionable reverse-proxy or authenticated-gateway guidance.
- Health remains available on loopback. No dashboard authentication, CSP, or frontend work enters this ticket.

### Disabled and authenticated ingress

- MARS_WEBHOOK_SECRET has precedence and is never logged, persisted by runtime, traced, returned, or exposed through a CLI flag. When absent, the GitHub-generated setup secret may be loaded only from the bounded regular owner-only 0600 GitHub App credentials file; setup persists it there but never returns it after success or write failure.
- Missing secret, trusted actor policy, or registered repository keeps local start/serve healthy but makes POST /webhook return 503 and dispatch nothing.
- Require a secret of at least 32 bytes and valid HMAC-SHA256 over the exact bounded request body.
- Missing or invalid HMAC returns 401; missing required headers/malformed required metadata returns 400; oversized input returns 413 with the configured limit; unsupported signed events return 202 with no callback.
- /healthz supports GET/HEAD only and /webhook POST only.
- Set MaxHeaderBytes 64 KiB, body read timeout 15 seconds, idle timeout 60 seconds, and bound all logged/stored delivery/event/repository/branch fields.

### Principal, repository, branch, and event authorization

- Use trusted numeric GitHub actor IDs as authority; login names are display-only.
- Resolve a bounded deduplicated numeric allowlist from YAML webhook_allowed_actor_ids, MARS_WEBHOOK_ALLOWED_ACTOR_IDS, and repeatable CLI --webhook-actor-id with CLI > env > YAML precedence. Missing/malformed/zero/negative IDs fail closed with actionable errors.
- Registered nonempty owner/repo remotes are the only repository allowlist. Validate/normalize remotes; empty remotes never match.
- start gains --remote owner/repo and preserves an existing registered remote/branch when a later start supplies empty values.
- Require exact case-normalized repository and exact case-sensitive registered branch.
- push requires sender.id and refs/heads/<registered branch>.
- workflow_run requires completed action, workflow_run.actor.id, exact head_repository, and exact head_branch.
- pull_request requires trusted action actor, exact base repository/branch, and same-repository non-fork head.
- check_suite requires trusted principal plus exact top-level repository and head branch.
- merge_group requires trusted principal and exact base branch.
- Malformed/incomplete event-specific metadata fails closed.
- Fork-derived events never dispatch.
- issue_comment is recognized but always returns 202 and never dispatches in this ticket. Remove issue_comment from new App manifests.
- Unauthorized actor/repository/branch/fork requests return 202, log only bounded reason metadata, consume no replay capacity, and enqueue nothing.

### Replay and queue idempotency

- Replay identity binds both delivery ID and SHA-256 body digest.
- Only fully authenticated and authorized deliveries enter replay state.
- Cap in-memory entries at 10,000 with 24-hour TTL; concurrent duplicates receive a successful acknowledgement and no second callback.
- Make webhook job idempotency durable across completion and restart so the same authorized delivery/body identity cannot recreate completed or failed work.
- Queue/callback failure must not be represented as a successfully processed mutation.

## Interfaces And Scope

Expected changes include github.Event event-specific identity/provenance fields, WebhookConfig actor/repository/branch authorization and bounded replay, serve.Config policy, RepoRegistry normalized remote lookup/preservation, start --remote and actor-ID policy resolution, trigger routing, queue idempotency, App manifest events, direct listener defaults, generated mars_cli/scanner guidance, BDD/docs, tests, and MarsDocSync routes.

Do not add secret YAML/CLI fields, login-based authority, fork overrides, issue-comment commands, non-loopback gateway support, dashboard auth/CSP/XSS, GitHub setup nonce, telemetry auth, release/CI/legal/public actions, or arbitrary webhook payload storage.

## Acceptance Criteria

- Default and direct listeners bind only loopback; every non-loopback control/dashboard address is rejected before bind.
- Local-first operation and health work without GitHub policy; /webhook returns 503 and dispatches zero.
- Short/missing secret, missing/malformed actors, invalid signature, wrong repo/branch/actor, fork origin, issue comment, malformed metadata, oversized body/header, and unsupported events enqueue zero.
- Realistic nested GitHub fixtures prove authorized push/workflow_run/pull_request/check_suite/merge_group routing.
- Empty remote never matches; start preserves existing registration; start --remote validates owner/repo.
- Rejected traffic does not enter replay state.
- Same delivery ID/body, changed delivery ID with identical signed body, concurrent duplicates, and replay after completion/restart enqueue exactly one job.
- Replay memory remains capped.
- No secret or full payload enters logs/traces/evidence.
- JIRA webhook behavior is unchanged.
- CLI/tool/generated guidance and all MarsDocSync docs update together.
- Focused integration tests use the real SQLite queue and prove rejected traffic leaves queue count unchanged.
- Full tests, race, vet, vulnerability, fuzz, DocSync, docs consistency, and diff gates pass.
- Installed candidate runs AD-284 static-browser plus API/service clean targets and records docs/validation/reports/2026-07-12-open-source-webhook-ingress.md, or records the exact blocker and replay command.
- F-017-S002 remains incomplete after this slice.

## Stop Conditions

Stop if any listener can bind non-loopback, start omits policy, missing configuration dispatches, empty remote is wildcard, authority uses login alone, branch/fork provenance is inferred, issue comments dispatch, replay after completion/restart recreates work, replay memory is unbounded, optional GitHub absence breaks local operation, dashboard/browser/setup/telemetry scope is required, a secret/payload is exposed, or any GitHub/public state would be mutated.

## Engineer Evidence

- Added shared literal-loopback validation; direct server/dashboard defaults use
  `127.0.0.1`, non-loopback binds fail before listen, and scoped conflict
  fallback remains `127.0.0.1:0`.
- Added CLI-over-env-over-YAML trusted numeric actor policy, env-first
  owner-only GitHub App secret resolution for both `start` and `serve`, exact normalized remote and
  case-sensitive branch registration/preservation, and synchronized
  `mars_cli` plus generated target guidance.
- Added realistic nested fixtures for authorized push, workflow-run,
  pull-request, check-suite, and merge-group inputs, alongside disabled,
  signature, actor, repository, branch, fork, issue-comment, method, body-limit,
  malformed metadata, concurrent replay, cap/TTL, and callback rollback tests.
- Added transactional SQLite webhook receipts that bind delivery ID and body
  SHA with derived queue jobs. Integration evidence proves rejected traffic
  leaves the real queue empty and authorized work remains single-shot after job
  completion, process restart, and a changed delivery ID carrying the same
  signed body.
- Accepted QA/Security corrections now use the top-level repository for
  check-suite authorization; require complete bounded action/base/head metadata;
  reject unsafe remote/branch characters and invalid direct actor policies at
  both handler and server boundaries; subscribe new Apps to merge-group events;
  and prove durable TTL expiry plus failed-job replay behavior.
- The accepted setup-secret design gives `MARS_WEBHOOK_SECRET` precedence and
  otherwise reads only bounded regular owner-only 0600 GitHub App credentials.
  Setup clears the returned webhook secret after both successful persistence
  and persistence failure; mode, malformed, oversized, missing, precedence,
  and non-exposure tests pass.
- Final review corrections validate every required event action as a bounded
  token before policy, retain 202 for legitimate non-completed workflow actions,
  reject Git-invalid leading-dot/dash and dot-component branches, and make
  setup credential writes atomic and symlink-safe with O_EXCL temp, fstat,
  fsync/close, rename, and cleanup. The loader proves lstat/open/fstat SameFile
  identity before any bounded read; destination-symlink, parent-symlink,
  open-swap, non-regular, and temp-cleanup tests pass.
- A live no-policy server test proves GET health remains available while POST
  webhook returns 503 and the real SQLite queue remains empty. The supported
  webhook body ceiling is documented as 2 MiB.
- Preserved the separate JIRA route and its existing tests without changing its
  behavior.
- PASS: `go test ./internal/network ./internal/github ./internal/queue
  ./internal/config ./internal/dashboard ./internal/serve ./cmd/mars`.
- PASS: uncached `go test ./... -count=1`; focused race
  `go test -race ./internal/github ./internal/queue ./internal/serve -count=1`;
  `go vet ./...`; `git diff --check`; and source `docsync audit` (337 files,
  zero findings).
- PASS: independent QA and Security final review found no blocking issue after
  the accepted credential-file, action-token, branch, and authentic
  check-suite fixture corrections.
- PASS: exact full repository race suite, two-second fuzz smoke, full uncached
  tests, vet, docs consistency, DocSync, and diff validation. The pinned
  vulnerability scan could not reach `vuln.go.dev` from the restricted
  sandbox; T-055's clean pinned v1.6.0 scan remains applicable because this
  ticket changed no dependency or toolchain input.

## Dogfood Evidence

- PASS: the installed candidate ran on clean static-browser and Go API/service
  targets with isolated DB/log paths. Actual socket inspection proved both
  control and dashboard listeners bound only to explicit `127.0.0.1` addresses;
  wildcard, LAN, hostname, and wildcard-dashboard configuration failed before
  bind with actionable remediation.
- PASS: both archetypes kept GET health at 200 while absent webhook policy made
  POST webhook return 503 with zero jobs and zero replay receipts.
- PASS: bad HMAC returned 401. Signed untrusted actor, wrong repository, wrong
  branch, fork-derived pull request, and issue-comment cases returned 202 and
  left the real SQLite queue and receipt table empty.
- PASS: one trusted exact workflow event created one job and one receipt.
  Duplicate delivery plus identical body under a changed delivery ID remained
  single-shot across process restart. On the API target the webhook-derived job
  completed through the real local Qwen3-Coder model before restart; the same
  replays still created no additional webhook job.
- PASS: no fake model endpoint, GitHub mutation, secret value, or payload body
  was used as committed evidence. Target intervention debt remained zero and
  both target worktrees were clean before cleanup.
- Report: [2026-07-12 open-source webhook ingress installed validation](../../validation/reports/2026-07-12-open-source-webhook-ingress.md#t-057-dogfood-result).

## QA And Security Evidence

- QA PASS: realistic GitHub payload shape, policy ordering, queue-zero negative
  cases, replay identity, generated guidance, and regression gates match the
  ticket contract.
- Security PASS: owner-only atomic credential persistence and bounded loading,
  direct-constructor policy validation, authenticated event authorization,
  fork/branch/repository containment, and durable replay behavior fail closed.
- Accepted review findings were corrected in the same ticket and rerun before
  both final dispositions.

## Orchestrator Disposition

T-057 is complete for its bounded OSS-02 contract. The implementation,
independent review, exact full race suite, and installed clean-project matrix
all pass. F-017-S002 remains incomplete because dashboard/browser, filesystem,
execution, telemetry, and other runtime P0 slices are still scheduled. The
primary outcome remains `primary_blocked`; this result does not authorize a
visibility change or public release.
