---
id: T-058
title: Harden the embedded dashboard browser and authenticated control boundary
priority: high
complexity: large
work_type: enabler
bdd_scenarios: ["F-017-S002", "F-010-S024"]
end_to_end_evidence: required
evidence_links: []
verified_by: "TBD"
owner: "engineer"
last_attempt: "TBD"
blocker: "none"
blocked_by: []
trace_id: "TBD"
next_action: "Implement the bounded embedded-dashboard session, Host/Origin/CSRF, request, DOM, CSP, redaction, and offline asset contract without relaxing loopback listeners."
dedupe_key: "open-source:dashboard-browser-control-security"
metadata:
  classification: "foundation-owned,mirrored-doctrine"
  primary_status: "primary_blocked"
  technical_lane: "authorized"
source: docs/exec-plans/active/current-operating-plan.md — F-017-S002 dashboard/browser slice
created: 2026-07-12
depends_on: [T-057]
---

# T-058: Harden the embedded dashboard browser and authenticated control boundary

## Context

T-057 closed the listener and GitHub webhook ingress boundary in private v0.68.46, but the shipped embedded dashboard still accepts unauthenticated browser mutations, trusts arbitrary Host and cross-origin requests, renders hostile runtime data through HTML sinks, loads htmx and Chart.js from public CDNs, lacks a strict CSP, and exposes insufficiently bounded browser surfaces.

This ticket hardens the current embedded Go dashboard only. It implements F-010-S024 and one bounded F-017-S002 slice. It does not complete the future TanStack-specific F-010-S012/MH-053 contract.

## Required Behavior

### Listener and trust boundary

- Preserve T-057 loopback-only dashboard and control listeners. Do not enable wildcard, LAN, hostname, or direct remote binding.
- Keep direct loopback GET/HEAD pages, bounded redacted read APIs, embedded static assets, and bounded redacted SSE available as a legacy observer surface.
- Treat this observer surface as trust for the local browser/operator only, not as protection against arbitrary same-user host code.
- Reject unconfigured Host values and default-deny CORS.
- Permit a remote browser only through an exact configured HTTPS reverse-proxy origin while the MARS listener remains loopback-only. Remote pages, APIs, assets, and SSE require authentication.
- Ignore Forwarded and X-Forwarded authority. Reject wildcard, suffix, path, query, fragment, userinfo, scheme downgrade, and missing-secret proxy configuration.

### Control authentication and sessions

- Resolve MARS_DASHBOARD_CONTROL_SECRET only from the environment, require at least 32 bytes, and never accept it through CLI arguments, URLs, YAML, generated files, logs, traces, HTML, errors, or responses.
- Missing secret keeps observation available but disables every HTTP mutation with actionable 503. A malformed configured secret fails startup.
- Login verifies the secret in constant time and creates an opaque CSPRNG in-memory session. There is no default/shared/anonymous mutation fallback.
- Rotate sessions on login; enforce 30-minute idle and eight-hour absolute expiry, a 128-session cap, logout invalidation, and restart invalidation.
- Cookies are host-only, Path=/, HttpOnly, SameSite=Strict, and Secure for configured HTTPS proxy origins.
- Treat authentication as a browser/network boundary, not containment against code already running as the same OS user.

### Request authorization and bounds

- Wrap the full dashboard mux at the gateway so serve-attached telemetry, evolution, roles, quality, throughput, and orchestration routes inherit the same policy.
- Pages/static use GET/HEAD, SSE GET, read APIs GET/HEAD, and login/control/logout POST. Reject unsupported methods and OPTIONS with 405 plus Allow.
- Require exact Host, authenticated session, exact same-origin Origin, and a session-bound CSRF token before pause, resume, restart, stop, scan, run-role, emergency-stop, or logout callbacks.
- Reject absent Origin for browser mutations. Use constant-time CSRF comparison and prevent cross-session/stale token reuse.
- Require appropriate content type, a 4 KiB login limit, a 64 KiB control JSON limit, strict unknown-field rejection, a single JSON value, bounded repo/role values, and no body for bodyless mutations.
- Bound and rate-limit login, mutation, and SSE connection/reconnect pressure. Return 429 with Retry-After. Rejections run no callback and create no queue, repo, runtime, or filesystem mutation.
- Preserve authenticated stop return-before-shutdown behavior and T-057 server header/read/idle timeout limits.

### Browser injection and disclosure

- Treat event type/data, repo, role, job, model, telemetry, decision, error, and quality data as hostile.
- Remove dynamic innerHTML, outerHTML, insertAdjacentHTML, document.write, inline scripts, inline handlers, and dynamic markup construction.
- Use createElement, textContent, replaceChildren, validated fixed classes, and bounded redacted DTOs.
- Do not return raw secret-bearing job errors, telemetry messages, traces, tool content, credentials, or unnecessary absolute personal paths.
- Bound SSE clients, per-session clients, event size, replay count/bytes, and reconnect behavior; do not reconnect indefinitely after 401 or 403.

### CSP and offline assets

- Apply security headers to success, redirect, error, static, API, and SSE responses: self-only CSP without unsafe-inline/eval, nosniff, no-referrer, frame denial, same-origin resource policy, restrictive permissions policy, and no-store for sensitive responses.
- Move template scripts and event handlers into embedded static JavaScript.
- Vendor exact htmx 2.0.4 and Chart.js 4.4.7 assets with immutable source URLs, SHA-256 hashes, versions, and MIT license metadata.
- Remove every runtime CDN request. The complete dashboard must work with outbound networking disabled.
- Do not finalize ownership-dependent project copyright or NOTICE wording.

## Interfaces And Scope

Expected code includes internal/dashboard gateway/session/security logic, dashboard templates and static scripts/assets, serve route wrapping, start/serve configuration propagation, CLI-over-environment trusted-origin resolution, mars_cli and scanner-generated guidance, BDD/docs, tests, and MarsDocSync routes.

Update dashboard design/API/operations/observability/CLI/quickstart/troubleshooting/workflow references where behavior changes. Keep GitHub/JIRA webhook behavior, GitHub App setup callback, remote telemetry collector intake, filesystem containment, execution profiles, system-wide redaction/retention, TanStack sidecar work, and public/legal/release actions out of scope.

## Acceptance Criteria

- Loopback-only listeners remain enforced.
- Direct loopback reads are bounded/redacted; all mutations are disabled without a valid configured control secret.
- A correct login creates one bounded rotating session; invalid/short/missing secret behavior fails closed without disclosure.
- Exact Host/Origin, session, CSRF, method, type, body, value, and rate policy passes before exactly one callback.
- Hostile Host/Origin, no-CORS form/fetch, missing/wrong/stale/cross-session CSRF, unsupported method, smuggled/trailing/unknown JSON, oversize/slow input, and floods mutate nothing.
- Trusted HTTPS proxy reads and SSE require authentication; forwarded-header spoofing and origin tricks fail.
- Hostile HTML/SVG/script/attribute/control-character fixtures render only as text across SSE, roles, repos, telemetry, decisions, errors, and empty states.
- No unsafe DOM sink, inline script/handler, CDN URL, external runtime request, or sensitive raw DTO remains.
- CSP and companion headers apply to every response class.
- Vendored assets match pinned hashes/licenses and the dashboard works with outbound networking disabled.
- Full tests, race, vet, fuzz, vulnerability, DocSync, docs consistency, link/forbidden-content, and diff gates pass.
- Installed AD-284 static-browser clean target plus real browser negative/positive smoke records docs/validation/reports/2026-07-12-open-source-dashboard-browser-security.md.
- F-017-S002 and F-010-S012 remain incomplete after this slice.

## Stop Conditions

Stop if any HTTP mutation is anonymous, any hostile Host/cross-origin request or invalid CSRF reaches a callback, non-loopback binding is re-enabled, remote reads remain anonymous, a dynamic HTML sink executes hostile data, any runtime CDN remains, CSP requires unsafe-inline/eval, a secret/session/CSRF value reaches logs/URLs/traces/responses, dashboard DTOs expose raw credential-bearing content, or scope requires GitHub setup, telemetry intake, filesystem, execution, TanStack, legal, visibility, or public-release work.
