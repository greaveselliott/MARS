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

| Surface | Evidence command | Expected artifact | Failure route |
| --- | --- | --- | --- |
| setup/path | `mars-harness setup --test-mode` and `mars-harness path setup --install-dir <tmp-bin>` | Config and shell-path result without duplicate profile snippets | Intervention-debt ticket for setup or shellpath |
| init/upgrade | `mars-harness init --repo <temp repo>` then `mars-harness update harness --repo <temp repo>` | `.harness/`, goals, BDD docs, role registry, quality score, release docs | Target-harness drift ticket |
| register/start | `mars-harness start --repo <temp repo> --db <temp db>` with deterministic shutdown | Per-repo DB, registered repo, seeded CEO job | Queue/orchestrator intervention debt |
| serve/control plane | `mars-harness serve --db <temp db> --addr :0` plus API control calls | Health, pause/resume/restart/scan/run-role endpoints respond | Control-plane ticket |
| run/dry-run | `mars-harness run engineer --repo <temp repo> --dry-run --trace` | Assembled prompt includes role, tools, guardrails, tickets, routes | Context or bundle ticket |
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

## Mars Observer Profile

`../mars` is the supersession target. The first trial must run observer mode:

- no file writes, commits, pushes, or destructive shell commands
- target-specific guardrails loaded before any role runs
- findings recorded as a completed validation report or intervention-debt tickets
- contributor mode allowed only after observer evidence shows no unsafe writes,
  no false-done claims, and actionable failure routing

The concrete observer contract lives in
[docs/validation/profiles/mars-observer.md](../validation/profiles/mars-observer.md).

## Consequences

Dry-run role checks remain useful smoke tests, but they do not prove autonomous
delivery. Claims about Mars supersession require matrix evidence, fake-LLM
integration, and observer-mode target results.
