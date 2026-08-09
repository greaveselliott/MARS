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
verified_by: "QA and Security source review PASS; Dogfood installed HTTP/SQLite and real-browser lanes PASS; Security and Orchestrator GO"
owner: "engineer"
last_attempt: "2026-08-09: exact current candidate passed the corrected installed in-app-browser replay"
blocker: "none"
blocked_by: []
trace_id: "docs/validation/reports/2026-07-12-open-source-dashboard-browser-security.md"
next_action: "Complete; create T-077 through ticket_create while the repository remains private."
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

At ticket creation, T-057 had closed the listener and GitHub webhook ingress
boundary in private v0.68.46, but the shipped embedded dashboard still accepted
unauthenticated browser mutations, trusted arbitrary Host and cross-origin
requests, rendered hostile runtime data through HTML sinks, loaded htmx and
Chart.js from public CDNs, lacked a strict CSP, and exposed insufficiently
bounded browser surfaces.

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
- F-010-S024 and F-017-S002 complete only after the installed real-browser lane
  and the later runtime slices pass; F-010-S012 remains the separate future
  TanStack contract.

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
- Root supporting gates pass for uncached full tests, the exact all-package
  race suite, vet, fuzz smoke with a writable isolated Go cache, DocSync,
  docs consistency, JavaScript syntax, forbidden application-owned sinks/CDNs,
  and diff validation. The pinned fail-closed vulnerability gate also passes
  with zero called vulnerabilities; its one reported module vulnerability is
  not reached by MARS.

## Dogfood Evidence — 2026-07-12

- **BLOCKED only on the mandatory real-browser cases:** the in-app browser
  backend completed required initialization and troubleshooting but returned an
  empty inventory. Hostile-string real-DOM rendering and browser network
  observation with outbound access disabled remain unconfirmed. No unrelated
  browser backend or source-only simulation was used.
- PASS installed supporting evidence: `make install` produced exact private
  `v0.68.48` (`mars 0.68.48 darwin/arm64`, binary SHA-256 recorded in the
  report), and a fresh AD-284 static-browser target ran on isolated loopback
  dashboard/control sockets, database, and log.
- PASS observer/auth boundary: anonymous shells, embedded assets, and the
  four-field path-free status projection remained available; privileged reads,
  SSE, and mutations returned `401`. Login, host-only HttpOnly SameSite=Strict
  cookie, and `/api/session` CSRF bootstrap passed without recording values.
- PASS zero-mutation matrix: bad Host/Origin/session/CSRF/method/content type,
  malformed/unknown/oversized bodies, and repeated invalid scan requests were
  rejected; rate limiting reached `429`, jobs stayed `0 -> 0`, and the single
  authorized pause was the only observed state change.
- PASS session lifecycle: HTTP warm restart returned `200`, terminated the
  active SSE client, and invalidated the old session; logout invalidated a
  replacement session. Direct non-HTTP restart remains supporting integration
  evidence because the installed run was non-interactive.
- PASS response boundary: strict CSP and companion headers were present without
  `unsafe-inline`/`unsafe-eval`; installed HTTP htmx 2.0.4 and Chart.js 4.4.7
  hashes matched the pinned values. Real-browser offline rendering and zero
  external-request observation remain blocked.
- No model ran, no target product source changed after initialization, no job,
  trace, telemetry, decision, or intervention debt was created, and no control
  credential, cookie, CSRF, or hostile payload entered the repo or report.
- Exact candidate identity, target commits, paths, statuses, aggregate SQLite
  counts, cleanup state, failure ownership, and remaining browser rerun are in
  `docs/validation/reports/2026-07-12-open-source-dashboard-browser-security.md`.

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

## Historical Orchestrator Disposition — 2026-07-12

At that checkpoint T-058 remained in backlog with its source implementation and reviewer gates
complete; the installed static-browser HTTP/SQLite security matrix also passes
as historical supporting evidence.
F-010-S024 cannot pass until an available in-app browser confirms hostile
strings remain inert in the real DOM and the vendored dashboard renders with
outbound networking disabled and zero external request attempts.
Resume only after T-076 against the then-current installed candidate.
`primary_blocked`, F-010-S024, and F-017-S002 remained unchanged until that
replay passed. The final installed evidence below supersedes this historical
disposition.

## Final Installed Browser Evidence — 2026-08-09

- Exact pushed source commit
  `57d7851d9c82975256761b0134d20e91382e9bcd`, Go 1.26.5, and
  installed-candidate SHA-256
  `ef5258e3b135c1e03a53635655b565c0e069fd16a3ca67af7631c76f9fa9e2bc`
  identify the corrected replay candidate.
- The first real-browser replay found one bounded foundation defect: successful
  logout removed server authority but left previously rendered privileged job
  rows in the DOM. Commit `57d7851` now reloads the current same-origin page
  after successful logout; `go test ./internal/dashboard -count=1` and the
  adjacent logout/static regression pass, and Security returned GO.
- Against the corrected installed candidate, anonymous DOMs showed no target
  name/path or privileged controls; authenticated login worked and cleared the
  submitted secret field; hostile runtime data appeared only as text with no
  executing marker, handler node, or nested SVG.
- The real-browser asset inventory observed eight assets, all from the exact
  loopback page origin, with zero external origins and zero browser console
  warnings/errors. Logout reloaded to the anonymous page with no privileged job
  cells, hostile marker, or target data retained.
- Observer startup created two expected deferred-model failures and two
  aggregate telemetry rows in the isolated database, zero traces, and no target
  worktree change. Both listeners stopped and all ephemeral files were removed.
  No credential, cookie, CSRF token, or hostile fixture is retained.

T-058 and F-010-S024 pass. Together with T-074 through T-076, this closes
F-017-S002 under the owner governor. The repository remains private at
`VERSION=0.68.49` and Primary Status `primary_blocked`; T-073 and T-077 through
T-081 remain launch blockers, and this ticket grants no release, settings,
visibility, signing, publication, or announcement authority.
