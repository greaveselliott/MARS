---
id: T-057
title: Default control listeners to loopback and fail closed on untrusted GitHub webhooks
priority: high
complexity: large
work_type: enabler
bdd_scenarios: ["F-017-S002", "F-011-S003", "F-011-S006", "F-006-S004", "F-010-S023"]
end_to_end_evidence: required
evidence_links: []
verified_by: "TBD"
owner: "engineer"
last_attempt: "TBD"
blocker: "none"
blocked_by: []
trace_id: "TBD"
next_action: "Implement loopback-only listeners and authenticated, numerically authorized, repository/branch/fork-aware, replay-safe GitHub webhook ingress."
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

- MARS_WEBHOOK_SECRET remains environment-only and is never logged, persisted, traced, returned, or exposed through a CLI flag.
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
- check_suite requires trusted principal plus exact head repository/branch.
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
