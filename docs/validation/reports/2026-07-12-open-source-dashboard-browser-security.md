# Validation Report: T-058 Open-Source Dashboard Browser Security

**Date:** 2026-07-12
**Author:** Dogfood role packet; foundation-maintainer remains accountable

## Primary Outcome Contract

**Primary Outcome:** Publish MARS as a supported open-source project without
exposing confidential material, weakening controls, or distributing unsafe or
unverifiable binaries.
**Primary Pass Gate:** Logged-out users can clone, build, install, update,
report vulnerabilities, and submit safe fork PRs; history is approved; runtime
P0s are closed; public artifacts are licensed, signed, and traceable.
**Primary Status:** `primary_blocked`
**Current Primary Blocker:** Legal ownership/licensing clearance and the
remaining F-017 runtime, release, public-access, community, audit, and cutover
gates are incomplete. T-058's installed HTTP/SQLite boundary passes, but the
required in-app browser surface is unavailable, so real DOM hostile-string and
offline browser-network acceptance remain unconfirmed.
**Next Primary Action:** Open the in-app browser and rerun the final hostile-DOM
and no-external-request matrix against installed private v0.68.48; keep T-058
and F-010-S024 incomplete until that browser evidence passes. F-017-S002 remains
independently incomplete across later runtime slices.
**Supporting Evidence:** Private v0.68.48 is published with nine local and nine
remote assets verified and a clean rolling release audit. Its exact installed
binary passed the clean-target loopback, anonymous/authenticated HTTP, request
negative, zero-job mutation, session invalidation, security-header, and
vendored-asset integrity matrix below. QA and Security independently passed the
source implementation.

## Result

**T-058 Dogfood result: BLOCKED (installed HTTP/SQLite lane PASS; mandatory
real-browser lane BLOCKED).**

The earlier loopback-bind and clean-seed blockers are cleared. Installed
private v0.68.48 bound only to two explicit loopback sockets and ran against a
fresh AD-284 static-browser target with an isolated database and log. The
authenticated HTTP matrix reached exactly one accepted pause, rejected hostile
Host/Origin/session/CSRF/method/type/body/value/rate requests, and left the job
count at zero. HTTP warm restart closed the active SSE stream and invalidated
the session; logout invalidated a replacement session.

The required in-app browser connection was initialized and troubleshot through
the mandated browser surface, but browser discovery returned an empty
inventory. No unrelated browser backend or source-only simulation was used.
Consequently, this report does **not** claim that hostile runtime strings were
observed as inert text in a real DOM, that the vendored assets rendered with
outbound networking disabled, or that a real browser emitted zero external
requests. Those are the sole remaining T-058 Dogfood blockers.

## Resumed Installed Run — 2026-07-12 21:31 BST

### Matrix Selection And Candidate Identity

- **Selected matrix/archetype:** AD-284 clean static-browser target, the
  ticket's installed dashboard/browser lane. T-057 already supplied the
  adjacent clean API/service evidence for the wider orchestration union.
- **Source ref:** `db35cc733c7480cd2ffb01bf6fd0e5da9f21dd00`
  (`v0.68.48`); behavior commit
  `0bd72b1 fix(dashboard): harden authenticated browser controls (T-058)`.
- **Installed command:** `$HOME/go/bin/mars`, freshly installed from the tagged
  checkout with `make install`; reported
  `mars 0.68.48 darwin/arm64 commit=unknown built=unknown`.
- **Installed binary SHA-256:**
  `75ca2b808e652c52da3dd2d071baa689580ce7a8dad42f9812a3b8beca378930`.
- **Target path:**
  `/private/tmp/mars-t058-dogfood-resume/static-browser`.
- **Target commits:** `c442c59` seeded the three-file static-browser app;
  `3656edb` is the fresh generated MARS harness baseline.
- **Runtime paths:** isolated database
  `/private/tmp/mars-t058-dogfood-resume/dashboard.db` and owner-local log
  `/private/tmp/mars-t058-dogfood-resume/dashboard.log`.
- **Addresses:** dashboard `127.0.0.1:19790`; control/webhook
  `127.0.0.1:19791`.
- **Model identity:** none. No inference tier, model server, or agent job ran.
- **Job sequence:** register one clean target; start installed server; exercise
  observer/auth/request/session lanes; stop server. Jobs, traces, telemetry,
  decisions, and intervention debt remained `0`.
- **Product progress:** installed dashboard security boundary exercised; no
  target product source or ticket was changed after harness initialization.
- **Secret/payload handling:** the control credential was generated in memory
  inside the bounded run, never printed, persisted, or committed, and was unset
  during cleanup. No hostile payload was created because the mandatory browser
  was unavailable.
- **Cleanup:** server and SSE clients exited; neither validation port remained
  listening; ephemeral cookie, login, session, header, page, and asset-response
  fixtures were deleted. The isolated target, database, log, and redacted
  runner remain outside the repo for owner inspection.

### Per-Case Evidence

| Case | Result | Installed evidence |
| --- | --- | --- |
| Exact tagged candidate | PASS | Installed version and binary hash match the identity above. |
| Loopback boundary | PASS | Process listeners were exactly `127.0.0.1:19790` and `127.0.0.1:19791`; no wildcard/LAN socket existed. |
| Clean AD-284 target | PASS | Fresh static-browser seed and generated harness commits ran against the isolated database. |
| Anonymous shells/assets/status | PASS (HTTP) | `/`, `/login`, `/pipeline`, app JS, htmx, and Chart.js returned `200`/the expected root redirect. Anonymous `/api/status` exposed only `active_jobs`, `healthy`, `paused`, and `uptime_secs`; it contained no repos. |
| Privileged reads/SSE locked | PASS (HTTP) | Repos, repo roles, telemetry, evolution, roles, quality, throughput, orchestration, decisions, and SSE all returned `401` without a session. Anonymous pause returned `401`. |
| Login and session bootstrap | PASS (HTTP) | Correct login returned `200`; the cookie was host-only, root-path, HttpOnly, and SameSite=Strict. Authenticated `/api/session` returned the current matching CSRF without a second secret entry. Values were not recorded. |
| Authenticated bounded reads | PASS (HTTP) | Repos returned `200`; projected paths contained only `static-browser`, with no absolute path. |
| Host/Origin/session/CSRF/method | PASS (HTTP/DB) | Bad Host `400`; bad or absent Origin `403`; missing session `401`; wrong CSRF `403`; unsupported method `405`. No rejected request changed state. |
| Content type and body bounds | PASS (HTTP/DB) | Body on a bodyless mutation `400`; wrong type `415`; malformed, unknown-field, and oversized JSON `400`. No job was created. |
| One authorized mutation | PASS | Exactly one authenticated pause returned `200`, and status changed to `paused=true`. |
| Rate limiting with zero mutation | PASS (HTTP/SQLite) | Repeated invalid scan bodies reached `429`; jobs remained `0 -> 0`, and paused state remained true. |
| HTTP restart and SSE/session invalidation | PASS | Authenticated restart returned `200`, the open SSE client exited promptly, and the old session then received `401`. |
| Logout invalidation | PASS | Authenticated logout returned `200`; the former session then received `401`. |
| Direct non-HTTP restart invalidation | SUPPORTING ONLY | Source integration coverage passed independently; the installed non-TTY run cannot invoke the terminal/direct method. |
| Security headers | PASS (HTTP) | CSP, nosniff, no-referrer, frame denial, same-origin resource policy, restrictive permissions policy, and no-store were present; CSP contained neither `unsafe-inline` nor `unsafe-eval`. |
| Vendored asset delivery/integrity | PASS (HTTP) | htmx response hash `e209dda5c8235479f3166defc7750e1dbcd5a5c1808b7792fc2e6733768fb447`; Chart.js response hash `2812cb8825fdc57469eb2f7bb055e9429244e599920511ee477e828499b632cb`. Two URL literals are inert provenance/license metadata inside the vendored Chart.js bundle, not application-owned runtime endpoints. |
| Real browser offline/no external requests | BLOCKED | The mandatory in-app browser inventory was empty; HTTP delivery and source evidence do not substitute for browser network observation. |
| Hostile strings render only as text | BLOCKED | Requires the unavailable real DOM; no payload value was created or recorded. |

### Exact Execution Shape

Preparation used the installed tagged source and clean target:

```bash
make install
git -C /private/tmp/mars-t058-dogfood-resume/static-browser init
$HOME/go/bin/mars init \
  --repo /private/tmp/mars-t058-dogfood-resume/static-browser \
  --model-routing defer --yes
$HOME/go/bin/mars register \
  --repo /private/tmp/mars-t058-dogfood-resume/static-browser \
  --remote local/static-browser --branch main \
  --db /private/tmp/mars-t058-dogfood-resume/dashboard.db
```

The bounded validation runner generated the 32-byte-or-longer credential only
in memory, exported it to the installed server, issued the statuses recorded
above, queried only aggregate SQLite counts, and cleaned up. The exact server
shape was:

```bash
MARS_DASHBOARD_PORT=19790 \
  $HOME/go/bin/mars serve --addr 127.0.0.1:19791 \
  --db /private/tmp/mars-t058-dogfood-resume/dashboard.db --debug \
  --log-file /private/tmp/mars-t058-dogfood-resume/dashboard.log \
  --code-intel=false
```

The credential value is deliberately omitted. The raw cookie and CSRF values
were neither printed nor retained.

### Exact Remaining Browser Rerun

With an in-app browser surface open, start the same installed tagged binary and
isolated target using a new ephemeral credential. In the in-app browser only:

1. Open the observer pipeline/login shells and confirm the DOM contains no
   target path or privileged data.
2. Log in, bootstrap `/api/session`, and confirm the authenticated page and SSE
   operate without local/browser storage of the secret or CSRF.
3. Seed hostile repository/runtime strings outside the repo evidence path and
   confirm each appears only as text across repos, roles, telemetry, decisions,
   errors, empty states, and SSE; confirm no script, SVG, or attribute executes.
4. Disable outbound browser networking, reload every page/chart, and inspect
   the browser network ledger. Require all resources to resolve from
   `127.0.0.1:19790` with zero external request attempts.
5. Log out and repeat restart invalidation while observing the real SSE client.
6. Record only response statuses, redacted DOM observations, and request
   origins; do not record the credential, cookie, CSRF, or hostile fixture.

## Initial Blocked Attempt — 2026-07-12

The initial pre-release attempt installed the uncommitted 0.68.47 candidate
but the environment denied localhost binding and clean-seed creation, while
browser discovery returned no surface. It still proved fail-closed rejection
of wildcard binding and a malformed short control-secret fixture. The resumed
run above supersedes the bind/seed blockers; the original browser-unavailable
classification remains current.

## Failure Ownership And Disposition

- **Environment/evidence-only:** the in-app browser backend is unavailable
  after mandated initialization, troubleshooting, and one inventory read.
- **Foundation-owned product finding:** none established. The installed
  HTTP/SQLite security matrix passed without a new failure class.
- **Deployed-owned finding:** none. The clean target was not mutated after
  initialization and produced no job, trace, telemetry, decision, or
  intervention debt.
- **Disposition:** keep T-058 in progress and F-010-S024 unconfirmed only for
  the remaining real-browser DOM and network cases. The installed HTTP/SQLite
  result is durable supporting evidence, not a waiver for those cases.
