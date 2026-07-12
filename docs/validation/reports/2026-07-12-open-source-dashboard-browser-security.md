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
gates are incomplete. T-058's installed real-browser acceptance is also blocked
by the local browser/bind environment described below.
**Next Primary Action:** Rerun this installed dashboard lane when the in-app
browser and approved loopback binding are available; keep F-017-S002 and T-058
incomplete until the rerun passes.
**Supporting Evidence:** The exact installed uncommitted candidate passes the
fail-closed wildcard-listener and malformed-control-secret startup probes. Unit,
race, QA, and Security evidence cover the implementation, but do not substitute
for the blocked installed browser lane.

## Result

**T-058 Dogfood result: BLOCKED.**

The installed candidate was built successfully, but the environment denied the
required localhost bind with `operation not permitted`. The mandatory in-app
browser runtime also reported no available browser surfaces. An escalation to
allow the installed candidate to bind only to localhost was attempted and
rejected because the environment's approval-credit limit had been reached.
Creating the clean AD-284 static-browser seed through the approved file-edit
path was rejected for the same reason. No alternative browser backend or
permission workaround was used.

Consequently, this report does **not** claim installed success for session
bootstrap, Host/Origin/CSRF/body/rate negatives, zero callback/queue mutation,
logout/restart stream invalidation, offline asset loading, or hostile-string
browser rendering. Those claims remain unconfirmed pending the exact rerun.

## Matrix Selection And Candidate Identity

- **Source-change class:** orchestration/control-plane plus generated doctrine;
  AD-284 requires a clean static-browser archetype and API/service or CLI/tooling
  union. This attempt selected the clean static-browser dashboard lane.
- **Source ref:** `f06fdc96044e34d83c80dcb653853230e4b1a36d` plus the
  uncommitted T-058 candidate diff.
- **Pre-report candidate diff SHA-256:**
  `187f05783d03eb1d85bf81ed4ec140c0e4d210797f5f7ff2c6d4b2f0854d1469`.
- **Installed command:** `$HOME/go/bin/mars`, installed with `make install`;
  reported `mars 0.68.47 darwin/arm64 commit=unknown built=unknown`.
- **Installed binary SHA-256:**
  `49c1f9c46478f441e20fe48c198b9e31796e401801f991c5d8b2504e396c86e7`.
- **Selected target path:** `/private/tmp/mars-t058-dogfood/static-browser`.
  The directory was created, but seed file creation was blocked before a target
  repo or harness existed.
- **Isolated runtime paths:** database and logs below
  `/private/tmp/mars-t058-dogfood/`; no project or shared MARS database was used.
- **Model identity:** none; the server did not bind and no agent/model job ran.
- **Secret handling:** one credential was generated only inside an ephemeral
  shell environment, never printed or written to the repo, then unset when the
  shell exited. No payload fixture was created.
- **Cleanup:** the server exited; no listener, browser session, queue job, or
  model process remained. Isolated failed-attempt files may remain under the
  named temporary root for operator inspection.

## Per-Case Evidence

| Case | Result | Durable interpretation |
| --- | --- | --- |
| Exact candidate install and identity | PASS | Installed binary/version/hash recorded above. |
| Wildcard control listener | PASS (rejected) | `0.0.0.0:19691` failed before bind with actionable loopback/reverse-proxy guidance. |
| Malformed configured control secret | PASS (rejected) | A short non-secret fixture failed startup with the documented 32-byte remediation. |
| Loopback dashboard/control bind | BLOCKED | Installed server returned `listen tcp 127.0.0.1:19691: bind: operation not permitted`. |
| Clean AD-284 static-browser seed | BLOCKED | Approved file-edit operation was rejected by the environment approval-credit limit. |
| In-app real browser | BLOCKED | Browser runtime discovery returned no available browser surfaces after required troubleshooting. |
| Anonymous shells/assets/minimal status | UNCONFIRMED | Requires the blocked listener/browser rerun. |
| Locked privileged reads/SSE/mutations | UNCONFIRMED | Requires the blocked listener/browser rerun. |
| Login, session bootstrap, cookie, CSRF | UNCONFIRMED | Requires the blocked listener/browser rerun. |
| Host/Origin/method/body/rate negatives with zero mutation | UNCONFIRMED | Requires the blocked listener/browser rerun and isolated DB inspection. |
| Logout/direct restart invalidation | UNCONFIRMED | Requires the blocked listener/browser rerun. |
| CSP, offline vendored assets, no external requests | UNCONFIRMED | Requires the blocked real-browser network inspection. |
| Hostile runtime strings render only as text | UNCONFIRMED | Requires the blocked clean target and real browser. |

## Exact Commands And Observed Blocker

Successful preparation and fail-closed probes:

```bash
make install
$HOME/go/bin/mars version
MARS_DASHBOARD_PORT=19690 $HOME/go/bin/mars serve --addr 0.0.0.0:19691 --db /private/tmp/mars-t058-dogfood/reject.db --debug --log-file /private/tmp/mars-t058-dogfood/reject.log --code-intel=false
MARS_DASHBOARD_CONTROL_SECRET=short MARS_DASHBOARD_PORT=19690 $HOME/go/bin/mars serve --addr 127.0.0.1:19691 --db /private/tmp/mars-t058-dogfood/short-secret.db --debug --log-file /private/tmp/mars-t058-dogfood/short-secret.log --code-intel=false
```

The real installed server command used an ephemeral in-memory 32-byte-or-longer
credential (value deliberately omitted) and failed with:

```text
serve: failed to bind 127.0.0.1:19691 ... bind: operation not permitted
```

The required escalation for a localhost-only bind was rejected with the
environment approval-credit limit. This is an environment-owned validation
blocker, not a product pass or product failure.

## Exact Rerun

From a fresh operator-authorized session with the in-app browser available:

```bash
make install
mkdir -p /private/tmp/mars-t058-dogfood/static-browser
git -C /private/tmp/mars-t058-dogfood/static-browser init
```

Using an approved editor, add this exact minimal static-browser seed before
initializing the harness:

```text
README.md: # Clean Static Browser Target
index.html: <!doctype html><html lang="en"><head><meta charset="utf-8"><title>Static Browser Seed</title></head><body><main id="app">Static browser seed</main><script src="app.js"></script></body></html>
app.js: document.getElementById("app").textContent = "Static browser seed ready";
```

Then continue:

```bash
git -C /private/tmp/mars-t058-dogfood/static-browser add README.md index.html app.js
git -C /private/tmp/mars-t058-dogfood/static-browser commit -m 'chore: seed static browser validation target'
$HOME/go/bin/mars init --repo /private/tmp/mars-t058-dogfood/static-browser --model-routing defer --yes
MARS_DASHBOARD_CONTROL_SECRET='<owner-only ephemeral 32+-byte value>' MARS_DASHBOARD_PORT=19690 \
  $HOME/go/bin/mars serve --addr 127.0.0.1:19691 \
  --db /private/tmp/mars-t058-dogfood/dashboard.db --debug \
  --log-file /private/tmp/mars-t058-dogfood/dashboard.log --code-intel=false
```

Then use the in-app browser only to run the ticket's anonymous/authenticated,
negative mutation, restart/logout, offline-network, hostile-rendering, and
security-header matrix. Keep the credential and hostile payload fixtures
ephemeral and record only statuses, redacted DOM observations, listener
addresses, and zero-mutation counts here.

## Failure Ownership And Disposition

- **Environment/evidence-only:** localhost binding, in-app browser discovery,
  and clean-seed creation were blocked by the current sandbox/approval-credit
  state.
- **Foundation-owned product finding:** none established by this blocked run.
- **Deployed-owned finding:** none; no target was created or run.
- **Disposition:** keep T-058 in progress and its installed-browser acceptance
  unconfirmed. Unit/race/reviewer evidence remains supporting-only.
