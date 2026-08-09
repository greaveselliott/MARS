---
id: T-074
title: Close GitHub App setup and telemetry listener boundaries
priority: high
complexity: medium
work_type: enabler
bdd_scenarios: ["F-017-S002"]
end_to_end_evidence: required
evidence_links: ["commit:596524ebaa0fd08718fe2630861a0b0c241a674a", "binary-sha256:b66394b7a44c2784e0eb662f91d68352844c92cb6bc8a4a51bc4da81e863ded7"]
verified_by: "Checkpoint A: QA, Security, Dogfood, Release Manager, and Orchestrator; Checkpoint B pending"
owner: "foundation-maintainer"
last_attempt: "2026-08-09: Checkpoint A passed at 596524e; Checkpoint B is next"
blocker: "none"
blocked_by: []
trace_id: "launch-network-boundaries:2026-08-09"
next_action: "Implement and validate only the source-only GitHub manifest callback checkpoint, then record T-074 closure evidence."
dedupe_key: "open-source:network-entry-point-hardening"
metadata:
  classification: "foundation-owned"
  mutation_authority: "repository-source-tests-docs-only"
  primary_status: "primary_blocked"
  publication_authority: "denied"
  t073_machine_evidence: "5068334"
source: MARS Launch-Complete Open-Source Delivery Plan — T-074
created: 2026-08-09
depends_on: [T-072]
---

# T-074: Close GitHub App setup and telemetry listener boundaries

## Context

T-073 machine-verifiable work is complete and parked on owner/legal input. F-017-S002 is now current. Two HTTP entry points remain outside the launch contract: the directly reachable telemetry collector defaults to a wildcard listener, and the source-only GitHub App manifest flow lacks loopback/state/single-use request admission. The repository remains private and the launch version freeze remains in force.

## Outcome

Make telemetry collection a literal-loopback-only local service and make the existing internal GitHub manifest flow safe before any future CLI wiring. Do not add remote telemetry, authentication infrastructure, a generalized HTTP framework, or a new GitHub setup command.

## Checkpoint A — Reachable Telemetry Collector

- Default and blank fallback to exactly 127.0.0.1:9092.
- Use or add a narrow literal-loopback-IP validation seam; do not use the current DNS-accepting validator unmodified. Reject wildcard, LAN, public, and DNS binds before opening or creating the SQLite database.
- Require the exact bound Host, POST, application/json, and either no Origin or the exact local Origin; forwarded headers grant no authority.
- Enforce an exact 2 MiB body limit, one JSON value plus EOF, and fixed redacted status responses. Method, Host, Origin, media-type, oversized, trailing, nil-store, and already-canceled-context failures make zero store calls. An admitted request whose store later fails returns a fixed 500, is never reported accepted, leaks no store detail, and remains safe to retry through existing deduplication.
- Add conventional header/read/write/idle timeouts and bounded graceful shutdown.
- Preserve the existing intake schema and opt-in outbound reporting. Remote collection remains unavailable.

Checkpoint A passed at exact commit `596524ebaa0fd08718fe2630861a0b0c241a674a`.
Focused normal/race tests, vet, docs-consistency, DocSync, mirrored CLI checks,
and four CGO-disabled Darwin/Linux builds passed. A clean commit-bound
Darwin/arm64 binary built with Go 1.26.5, `vcs.modified=false`, and SHA-256
`b66394b7a44c2784e0eb662f91d68352844c92cb6bc8a4a51bc4da81e863ded7`
accepted one synthetic loopback report and persisted exactly one report and one
pattern. The same installed candidate rejected a wildcard bind before creating
the requested database directory. QA, Security, Dogfood, Release Manager, and
the Orchestrator approved this checkpoint.

## Checkpoint B — Source-Only GitHub App Manifest Flow

- Default to 127.0.0.1:9999 and reject non-literal-loopback binds before listening.
- Generate at least 32 random bytes per run and carry the URL-safe state through the supported manifest redirect flow.
- Require exact GET method, Host, path, one bounded code, matching state, and an empty bounded body.
- Atomically consume matching state before exchange so only one concurrent callback wins; replay and post-terminal requests cannot exchange or persist credentials.
- End the flow after the first admitted callback, including exchange failure, and on timeout/cancellation.
- Bound server headers and read/write/idle timeouts plus the exchange context and response body.
- Return only fixed redacted error classes. The setup form and GitHub redirect are the sole permitted exposure of state; state and code never enter CLI output, logs, traces, function results, or success/error messages. Provider response bodies, PEM, client secrets, and webhook secrets never enter any of those surfaces.
- Preserve the existing owner-only atomic credential persistence. Do not wire RunSetup into the CLI in this ticket.

## Affected Interfaces And DocSync

Direct code is limited to internal/github/setup.go and focused tests, internal/foundationtelemetry HTTP intake and tests, telemetry collect command construction/tests, and the existing loopback helper only if its literal-IP contract needs a narrow reuse seam. Sync the directly owning GitHub integration, telemetry, product-surface, F-011/F-012/F-017, CLI-reference, observability, safety, and mirrored mars_cli guidance surfaces. Do not touch dashboard/webhook behavior, repository filesystem, execution profiles, trace retention, auth/bootstrap, release producer, versions, tags, Releases, GitHub settings, visibility, or announcement.

## Validation

Run focused normal and race tests for internal/github, internal/foundationtelemetry, internal/network, affected cmd/mars tests, and mirrored CLI tests; DocSync/docs-consistency checks; four CGO-disabled cross-builds; and an installed-candidate clean-HOME telemetry smoke proving a valid default-loopback intake and a rejected non-loopback bind with no database creation. Use a fake conversion service for GitHub tests; do not create a live GitHub App.

## Acceptance

- Telemetry defaults to 127.0.0.1:9092, every non-loopback bind fails before DB/listener state, valid local intake still works, every pre-admission rejection makes zero store calls, and an admitted storage failure returns only a fixed non-success response without leaking values.
- GitHub setup binds only to literal loopback, accepts exactly one state-bound callback, rejects wrong/missing/replayed/concurrent requests without additional exchange or persistence, and shuts down on terminal callback or timeout.
- Fixed errors and logs contain no sentinel code/state/body/path/token/credential values.
- QA and Security approve both frozen source diffs; Dogfood passes installed telemetry smoke; Release Manager verifies VERSION 0.68.49, private visibility, and zero release/publication mutation.
- T-074 closes only this network-entry-point slice. F-017-S002 remains incomplete pending T-075, T-076, and resumed T-058; Primary Status remains primary_blocked.

## No-Go

Any remote listener, callback replay, multiple exchange, false-clean oversized/trailing request, sensitive value in output, database creation after rejected bind, new GitHub App CLI reachability, remote telemetry design, unrelated runtime scope, version/tag/Release/settings/visibility/announcement mutation, or false claim that F-017-S002 is complete.
