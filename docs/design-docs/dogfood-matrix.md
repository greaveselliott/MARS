# AD-088: Dogfood Matrix And Mars Supersession Benchmark

**Status:** Accepted
**Date:** 2026-05-03
**Owner:** Mars Harness maintainers

## Context

Mars Harness should not claim autonomous-delivery readiness from design intent,
dry-run prompts, or unit tests alone. The product promise spans setup,
generated harnesses, orchestration, local inference, tools, safety, scoring,
quality export, and release discipline. Those surfaces need one repo-visible
dogfood matrix so agents can tell which paths are proven, which are skipped,
and which failures must become intervention debt.

## Decision

Mars Harness uses this matrix as the source dogfood benchmark. A dogfood run is
complete only when each required surface has a reproducible evidence command,
expected artifact, and failure route.

Durable validation profiles, evidence contracts, and run reports live under
`docs/validation/`. The term `dogfood` remains the runtime role and mode name;
the documentation directory uses validation language so evidence artifacts are
not tied to one role label.

Evidence is stored in three places:

- `docs/validation/profiles/` defines reusable target profiles and guardrails.
- `docs/validation/reports/` stores completed cross-target validation reports
  when a trial has operator-visible results.
- Target-local `docs/reports/dogfood/` stores role-produced run evidence that
  belongs to that target. Target-owned findings become deduped tickets through
  `ticket_create`; foundation/runtime failures stay foundation telemetry or
  source-repo work.

| Surface | Evidence command | Expected artifact | Failure route |
| --- | --- | --- | --- |
| setup/path | `mars-harness setup --test-mode` and `mars-harness path setup --install-dir <tmp-bin>` | Config and shell-path result without duplicate profile snippets | Intervention-debt ticket for setup or shellpath |
| init/upgrade | `mars-harness init --repo <temp repo>` then `mars-harness update harness --repo <temp repo>` | `.harness/`, goals, BDD docs, role registry, quality score, release docs | Target-harness drift ticket |
| register/start | `mars-harness start --repo <temp repo> --db <temp db outside repo>` with deterministic shutdown | Per-repo DB, registered repo, one idempotent CEO bootstrap job | Queue/orchestrator telemetry |
| serve/control plane | `mars-harness serve --db <temp db> --addr :0` plus API control calls | Health, pause/resume/restart/scan/run-role endpoints respond | Control-plane ticket |
| run/dry-run | `mars-harness run engineer --repo <temp repo> --dry-run --trace`; for uninitialized observer targets use `--dry-run --no-init` | Assembled prompt includes role, tools, guardrails, tickets, routes, or the command explicitly reports the no-init missing-harness boundary without writing | Context or bundle ticket |
| scan/tickets | `mars-harness scan --repo <temp repo> --tickets` | Deduped backlog tickets | Scanner ticket |
| doctor/update | `mars-harness doctor --repo <temp repo> --json` and `mars-harness update check --repo <temp repo> --skip-remote --json` | Actionable OK/warn/fail output | Doctor/update ticket |
| scores/trust/quality | `mars-harness scores --repo <temp repo>`, `mars-harness trust --repo <temp repo>`, `mars-harness scores export --repo <temp repo>` | Empty or live evidence is actionable; quality score refreshes | Scoring/trust ticket |
| dashboard | HTTP handler tests for quality, repos, roles, status, controls | Server-rendered pages and JSON APIs do not crash on empty modules | Dashboard ticket |
| local inference | `mars-harness models list` and router missing-model tests | Pinned models and actionable missing-model remediation | Inference ticket |
| optional GitHub | `mars-harness release verify-assets --version vX.Y.Z` when credentials exist | Release assets verified or explicit blocker recorded | Release blocker ticket |
| upgrade safety | `mars-harness update harness --repo <target with edits>` | Missing defaults written, user-owned files preserved | Migration ticket |

Optional GitHub paths are skipped honestly when credentials, remote visibility,
or release assets are unavailable. A skipped optional path is not a pass; it is
recorded as blocked or not applicable with the reason.

## Fake-LLM Loop

CI must include a deterministic fake-LLM dogfood path before contributor-mode
autonomy is trusted. The loop should cover ticket creation, file edit, test
execution, direct-main commit attempt or guarded substitute, score recording,
trust visibility, and quality export without requiring network, real models,
or GitHub credentials.

The fast foundation acceptance gate is release-blocking and runs under the
normal `go test ./...` path. It uses an OpenAI-compatible fake LLM endpoint
through the real executor, router fallback, tool registry, trust policy,
telemetry, scoring, ticket gate, and intervention-debt creation paths. The gate
must prove:

- a clean initialized target baseline can run a generated starter role with
  contributor trust and controlled mutation
- `init` and auto-init commands commit only the generated harness scaffold
  before a bootstrap role can run, so blank-repo initialization is not
  misclassified as dirty target state
- destructive shell commands are blocked before execution
- dirty worktrees that already exceed blast radius are contained before LLM
  invocation
- read-only shell inspection does not create blast-radius noise in an already
  dirty repo
- repeated intervention-debt updates stay bounded so ticket context does not
  become its own context-overflow source
- the broader fake-LLM dogfood loop can create a deduped finding through
  `ticket_create`, run a bounded test command, commit evidence on `main`,
  attempt `git_push` without failing when no remote exists, record a scoring
  outcome, and invoke the `scores export` quality hook through
  `mars_harness_cli`

Broader scheduled dogfood remains useful for product coverage, but it is not a
substitute for this fast foundation containment gate.

## Mars Observer Profile

`../mars` is the supersession target. The first trial must run observer mode:

- no file writes, commits, pushes, or destructive shell commands
- target-specific guardrails loaded before any role runs
- findings recorded as a completed validation report or intervention-debt tickets
- contributor mode allowed only after observer evidence shows no unsafe writes,
  no false-done claims, and actionable failure routing

The concrete observer contract lives in
[docs/validation/profiles/mars-observer.md](../validation/profiles/mars-observer.md).

Observer mode is mechanical, not only procedural. A Mars observer validation
run must use observer trust so mutating tools such as `file_write`,
`ticket_create`, `shell_exec` mutations, `git_commit`, and `git_push` are
blocked before they can touch the target. The role may still record a blocked
disposition and source-side evidence, because the absence of contributor-mode
authority is itself a valid validation result.

## Consequences

Dry-run role checks remain useful smoke tests, but they do not prove autonomous
delivery. Claims about Mars supersession require matrix evidence, fake-LLM
integration, and observer-mode target results.
