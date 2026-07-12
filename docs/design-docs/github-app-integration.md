# GitHub App And Webhook Ingress

**Status:** Accepted
**Owner:** Security
**Related:** F-006, F-010, F-011, F-017, T-057

## Context

GitHub events are optional inputs to a local-first strict-trunk system. They can
cause autonomous work, so transport authenticity alone is insufficient: MARS
must also authorize the principal, exact registered repository, branch, event
shape, and same-repository provenance before trigger matching or queue writes.
Local health must remain available when this optional policy is absent.

## AD-309: Loopback And Fail-Closed GitHub Ingress

Control and dashboard listeners default to explicit `127.0.0.1` addresses.
Literal `localhost`, IPv4 loopback, and IPv6 `::1` are accepted; wildcard, LAN,
DNS hostname, and other non-loopback binds are rejected before listening. A
remote operator must place an authenticated gateway or reverse proxy in front
of the loopback service. Dashboard authentication and browser protections are a
later F-017 slice and do not weaken this boundary.

GitHub webhook ingress is disabled with HTTP 503 unless all three policy inputs
exist: a webhook secret of at least 32 bytes, at least one
trusted numeric GitHub actor ID, and at least one registered non-empty
`owner/repo` remote with an exact branch. Actor IDs resolve with CLI over env
over YAML precedence: repeatable `--webhook-actor-id`, then comma-separated
`MARS_WEBHOOK_ALLOWED_ACTOR_IDS`, then `webhook_allowed_actor_ids`. Invalid,
zero, negative, or over-limit policy fails closed. Login names are display-only.

`MARS_WEBHOOK_SECRET` has strict precedence. When it is absent, MARS may load
the GitHub-generated webhook secret from the local GitHub App credentials file,
but only when that path is a regular owner-only `0600` file and the bounded JSON
decodes successfully. The fallback is never copied into YAML, exposed through a
CLI flag, logged, returned over HTTP, added to traces, or returned by `RunSetup`
after either successful persistence or a write failure. Missing fallback state
keeps local operation healthy and webhook ingress disabled; unsafe or malformed
fallback state fails with an actionable local error.
Setup credential persistence never opens or truncates the destination. It
writes a random owner-only `O_EXCL` temp file inside the verified real `0700`
parent, verifies the descriptor is regular `0600`, syncs and closes it, then
atomically renames it over the destination. A destination symlink is replaced
without following its target; unsafe parent types are rejected and failed temp
files are removed. Loading performs `lstat`, then open and `fstat`, and requires
`SameFile` plus regular `0600` mode before reading, so a path swap or symlink
cannot redirect the bounded credentials read.

After HMAC-SHA256 verification over the exact bounded body, supported events
must prove their own authority boundary:

- `push`: `sender.id`, exact repository, and `refs/heads/<registered-branch>`;
- `workflow_run`: completed action, `workflow_run.actor.id`, exact head
  repository, and exact head branch;
- `pull_request` legacy compatibility event: trusted sender, exact base repository/branch, and a
  same-repository non-fork head;
- `check_suite`: trusted sender, exact top-level repository, and exact head branch;
- `merge_group`: trusted sender, exact repository, and exact base branch.

Unauthorized signed requests receive 202 without trigger matching or replay
state. `issue_comment` is recognized but never dispatches, and new App
manifests do not subscribe to it. Unsupported signed events also receive 202.
Malformed required metadata returns 400, bad HMAC returns 401, oversized bodies
return 413, and only POST is accepted on `/webhook`. `/healthz` accepts GET and
HEAD only.

Workflow-run, pull-request, check-suite, and merge-group actions must be
non-empty bounded tokens without whitespace or controls before authorization.
A legitimate bounded workflow-run action other than `completed` is recognized
but unauthorized for autonomous work and returns 202. Registered and event
branch names also reject Git-invalid leading dot/dash names and dot-prefixed
path components without invoking a shell command.

The webhook body limit is 2 MiB: large enough for the supported event metadata
and small trigger predicates while bounding memory before parsing. Replay
identity binds both bounded delivery ID and SHA-256 body digest. An
in-memory concurrent reservation store is capped at 10,000 entries with a
24-hour TTL and rolls back callback failures. SQLite atomically records the
delivery/body receipt and all derived jobs. The receipt outlives job completion,
failure, and process restart for the TTL, so neither a repeated delivery ID nor
the same signed body under a changed delivery ID recreates work. Queue failure
rolls back the receipt and returns 500 so GitHub can retry.

Registered remotes are case-normalized, branches remain case-sensitive, and an
empty remote never becomes a wildcard. `mars start --remote owner/repo
--branch <branch>` establishes the boundary; later starts with empty values
preserve the existing registration. The JIRA webhook route, signature policy,
and mirror-only behavior are separate and unchanged.

## Security Properties

- Secrets and full payloads are not logged, traced, or stored in job payloads.
- Logged/stored delivery, event, repository, branch, action, and display fields
  are bounded.
- HTTP servers cap headers at 64 KiB and use 15-second read/header timeouts plus
  a 60-second idle timeout.
- Repository, actor, branch, fork, and replay rejection occurs before autonomous
  queue mutation.
- Optional GitHub absence never prevents loopback health or local workflows.

## Deferred Work

Dashboard authentication, CSRF/Origin/Host/CSP/XSS controls, GitHub App setup
state/nonce, telemetry authentication, and non-loopback gateway support remain
separate F-017 tickets.
