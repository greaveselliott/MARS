---
id: T-058
title: Harden the embedded dashboard browser and authenticated control boundary
priority: high
complexity: large
work_type: enabler
bdd_scenarios: ["F-017-S002", "F-010-S024"]
end_to_end_evidence: required
evidence_links:
  - docs/validation/reports/2026-07-12-open-source-dashboard-browser-security.md#result
verified_by: "QA and Security source review PASS; Dogfood installed browser lane BLOCKED"
owner: "engineer"
last_attempt: "2026-07-12"
blocker: "Installed real-browser acceptance is blocked because the environment denies loopback bind, approval credits prevent the required escalation, and the in-app browser has no available surface."
blocked_by: []
trace_id: "docs/validation/reports/2026-07-12-open-source-dashboard-browser-security.md"
next_action: "From a fresh session with approval credits and an in-app browser, run the exact installed static-browser replay in the linked report; keep T-058 and F-010-S024 incomplete until it passes."
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
- Keep direct loopback GET/HEAD page/login shells, embedded static assets, and a minimal redacted status projection available as a legacy observer surface. Require authentication for repos, roles, telemetry, evolution, quality, throughput, orchestration, decisions, and bounded SSE.
- Treat this observer surface as trust for the local browser/operator only, not as protection against arbitrary same-user host code.
- Reject unconfigured Host values and default-deny CORS.
- Permit a remote browser only through an exact configured HTTPS reverse-proxy origin while the MARS listener remains loopback-only. Only a data-free login shell and its minimal assets are anonymous remotely; dashboard pages, APIs, assets, and SSE require authentication.
- Ignore Forwarded and X-Forwarded authority. Reject wildcard, suffix, path, query, fragment, userinfo, scheme downgrade, and missing-secret proxy configuration.

### Control authentication and sessions

- Resolve MARS_DASHBOARD_CONTROL_SECRET only from the environment, require at least 32 bytes, and never accept it through CLI arguments, URLs, YAML, generated files, logs, traces, HTML, errors, or responses.
- Missing secret keeps observation available but disables every HTTP mutation with actionable 503. A malformed configured secret fails startup.
- Login verifies the secret in constant time and creates an opaque CSPRNG in-memory session. There is no default/shared/anonymous mutation fallback.
- Bootstrap the current session CSRF through authenticated same-origin `GET /api/session` so the login-shell redirect works without a second secret entry and without URLs, logs, or browser storage.
- Rotate sessions on login; enforce 30-minute idle and eight-hour absolute expiry, a 128-session cap, logout invalidation, HTTP/terminal warm-restart invalidation, and prompt SSE termination for logout/expiry/restart.
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
- Vendor exact htmx 2.0.4 and Chart.js 4.4.7 assets with immutable source URLs, SHA-256 hashes, versions, htmx Zero-Clause BSD license metadata, and Chart.js MIT license metadata.
- Remove every runtime CDN request. The complete dashboard must work with outbound networking disabled.
- Do not finalize ownership-dependent project copyright or NOTICE wording.

## Interfaces And Scope

Expected code includes internal/dashboard gateway/session/security logic, dashboard templates and static scripts/assets, serve route wrapping, start/serve configuration propagation, CLI-over-environment trusted-origin resolution, mars_cli and scanner-generated guidance, BDD/docs, tests, and MarsDocSync routes.

Update dashboard design/API/operations/observability/CLI/quickstart/troubleshooting/workflow references where behavior changes. Keep GitHub/JIRA webhook behavior, GitHub App setup callback, remote telemetry collector intake, filesystem containment, execution profiles, system-wide redaction/retention, TanStack sidecar work, and public/legal/release actions out of scope.

## Acceptance Criteria

- Loopback-only listeners remain enforced.
- Direct loopback page/login shells and minimal status are bounded/redacted; privileged reads and all mutations are locked without a valid session.
- A correct login creates one bounded rotating session; invalid/short/missing secret behavior fails closed without disclosure.
- Exact Host/Origin, session, CSRF, method, type, body, value, and rate policy passes before exactly one callback.
- Hostile Host/Origin, no-CORS form/fetch, missing/wrong/stale/cross-session CSRF, unsupported method, smuggled/trailing/unknown JSON, oversize/slow input, and floods mutate nothing.
- Trusted HTTPS proxy dashboard pages, reads, assets, and SSE require authentication after a data-free login shell; forwarded-header spoofing and origin tricks fail.
- Hostile HTML/SVG/script/attribute/control-character fixtures render only as text across SSE, roles, repos, telemetry, decisions, errors, and empty states.
- No unsafe DOM sink, inline script/handler, CDN URL, external runtime request, or sensitive raw DTO remains.
- CSP and companion headers apply to every response class.
- Vendored assets match pinned hashes/licenses and the dashboard works with outbound networking disabled.
- Full tests, race, vet, fuzz, vulnerability, DocSync, docs consistency, link/forbidden-content, and diff gates pass.
- Installed AD-284 static-browser clean target plus real browser negative/positive smoke records docs/validation/reports/2026-07-12-open-source-dashboard-browser-security.md.
- F-017-S002 and F-010-S012 remain incomplete after this slice.

## Stop Conditions

Stop if any HTTP mutation is anonymous, any hostile Host/cross-origin request or invalid CSRF reaches a callback, non-loopback binding is re-enabled, remote reads remain anonymous, a dynamic HTML sink executes hostile data, any runtime CDN remains, CSP requires unsafe-inline/eval, a control secret or session cookie reaches logs/URLs/traces/responses, a CSRF value reaches any response other than the authenticated login or authenticated session-bootstrap JSON contracts, dashboard DTOs expose raw credential-bearing content, or scope requires GitHub setup, telemetry intake, filesystem, execution, TanStack, legal, visibility, or public-release work.

## Engineer Evidence — 2026-07-12

- Implemented one fail-closed gateway around the complete dashboard mux,
  including serve-attached routes. Direct loopback GET/HEAD observation remains
  available; mutation requires the environment-only 32-byte control secret,
  constant-time login, rotating bounded in-memory session, exact Host/Origin,
  session CSRF, method/content/body/value/rate policy, and reaches each callback
  only after authorization.
- Added exact HTTPS proxy-origin validation with authenticated remote reads,
  Secure proxy cookies, forwarded-header non-authority, idle/absolute expiry,
  session cap, logout/restart invalidation, and bounded SSE replay/client/rate
  behavior while preserving T-057 loopback-only binds and server timeouts.
- Added a data-free unauthenticated login shell and authenticated
  `/api/session` CSRF bootstrap, with no control secret, cookie, or CSRF value
  stored in browser storage. Tightened anonymous loopback observation to the
  minimal path-free status projection; repository, role, telemetry, decision,
  orchestration, and SSE data all require an active session.
- Wired session and SSE invalidation into the authoritative server restart path
  so terminal, direct, and HTTP restarts share the same security boundary.
- Hashes both configured and submitted control secrets before constant-time
  comparison, preserves SSE `no-store`, and redacts complete authorization
  credentials and personal paths from bounded dashboard projections.
- Replaced application-owned HTML parsing sinks and inline handlers/scripts with
  fixed-class `createElement`, `textContent`, and `replaceChildren` rendering;
  added bounded redacted read/SSE DTOs and generic browser errors.
- Added strict self-only CSP and companion headers to the gateway. Vendored and
  embedded htmx 2.0.4 under Zero-Clause BSD and Chart.js 4.4.7 under MIT, with immutable
  sources, and independently checked SHA-256 values in
  `internal/dashboard/static/vendor/ASSETS.md`; htmx eval/script/inline-style
  behavior is disabled by template configuration and runtime CDN URLs are gone.
- PASS: `go test ./internal/dashboard ./internal/serve ./cmd/mars ./internal/tools ./internal/scanner -count=1`.
- PASS: `go test ./internal/dashboard ./internal/serve ./cmd/mars -race -count=1`.
- PASS: `go test ./internal/docsconsistency -count=1`.
- PASS: `go run ./cmd/mars docsync audit --repo .` (`341` checked, `0` findings).
- PASS: `git diff --check`.
- PASS asset hashes: htmx
  `e209dda5c8235479f3166defc7750e1dbcd5a5c1808b7792fc2e6733768fb447`;
  Chart.js
  `2812cb8825fdc57469eb2f7bb055e9429244e599920511ee477e828499b632cb`.
- Root supporting gates pass for uncached full tests, vet, fuzz smoke with a
  writable isolated Go cache, DocSync, docs consistency, JavaScript syntax,
  forbidden application-owned sinks/CDNs, and diff validation. The exact full
  race rerun is environment-blocked because required unsandboxed execution was
  rejected after approval credits were exhausted; the focused dashboard,
  serve, and CLI race suite passed independently. The fail-closed vulnerability
  gate cannot resolve `vuln.go.dev`; this ticket changes no dependency or
  toolchain input from T-055's pinned clean scan.

## Dogfood Evidence — 2026-07-12

- **BLOCKED:** installed real-browser acceptance remains unconfirmed; see
  `docs/validation/reports/2026-07-12-open-source-dashboard-browser-security.md`.
- PASS supporting evidence: `make install` produced the exact uncommitted
  candidate (`mars 0.68.47 darwin/arm64`, installed binary SHA-256 recorded in
  the report); wildcard control binding and a malformed configured dashboard
  secret were rejected before bind with actionable remediation.
- The installed localhost run failed with `bind: operation not permitted`.
  The required localhost-only escalation, clean-target file creation, and
  in-app browser availability were blocked by the environment approval-credit
  limit/no-browser state. No unrelated browser backend or permission workaround
  was used.
- No model ran, no target/queue work was created, no secret or hostile payload
  entered the repo or report, and the ephemeral shell credential was unset on
  exit.
- Exact replay command and the still-unconfirmed Host/Origin/CSRF/body/rate,
  zero-mutation, session invalidation, CSP/offline-asset, and hostile-rendering
  cases are durable in the report. Keep this ticket in progress until that
  installed browser rerun passes.

## QA And Security Evidence — 2026-07-12

- QA PASS after two correction loops. Accepted findings fixed remote login
  bootstrap, direct/HTTP restart and SSE invalidation, the complete method
  matrix, anonymous status projection, DTO bounds, fixed-digest secret
  comparison, SSE headers, htmx license truth, and Authorization redaction for
  Bearer, Basic, Digest, and JSON-shaped values.
- Security PASS after an independent corrected-diff re-read and focused normal
  and race tests. The outer mux gateway, tightened observer surface, proxy
  authority, session/CSRF lifecycle, request bounds, safe DOM/CSP, redaction,
  and vendored asset provenance have no remaining source-review blocker.
- QA and Security success is supporting source evidence only and does not
  substitute for the blocked installed real-browser lane.

## Orchestrator Disposition — 2026-07-12

T-058 remains in progress. The source implementation and reviewer gates pass,
but F-010-S024 cannot pass until installed static-browser, offline network,
hostile DOM, and zero-mutation evidence is rerun in an environment that permits
loopback binding and exposes the in-app browser. `primary_blocked` remains
unchanged, and no next implementation ticket may start while T-058 is current.
