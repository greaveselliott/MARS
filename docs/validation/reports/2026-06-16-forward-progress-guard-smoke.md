# Factory Forward Progress Guard Smoke

Date: 2026-06-16
Source ref: local working tree before commit
Binary: installed with `make install` to `/path/to/local-redacted`
Owner: foundation

## Primary Outcome Contract

**Primary Outcome:** Exercise startup and restart safety checks that support the
Factory Forward Progress Guard.

**Primary Pass Gate:** Bounded startup smoke records supporting evidence for
seeded startup, existing ticket routing, explicit reseed, and dirty
ambiguous-state refusal through the installed CLI.

**Primary Status:** `supporting_only`

**Current Primary Blocker:** Full greenfield lifecycle validation still needs
static web, Phaser/browser-game, and Go API targets to reach Engineer
build/smoke evidence.

**Next Primary Action:** Use this smoke evidence only as support, then rely on
the live lifecycle report or a rerun to prove the full primary pass gate.

**Supporting Evidence:** Installed-binary startup/restart behavior, focused
tests, full Go suite, and docsync audit passed for the startup guard.

## Scope

Validated the foundation-owned startup and routing guard added by AD-297:

- startup action is printed through the installed `mars start` surface
- fresh target seeds CEO only when no resumable state exists
- existing in-progress ticket routes to Engineer instead of reseeding CEO
- `--new-lifecycle` explicitly reseeds CEO over existing work
- dirty product workspace without a deterministic ticket/disposition route refuses as ambiguous

This report is a bounded startup smoke. It is not a full model-backed static web,
Phaser, and Go API lifecycle sweep.

## Commands

Unit and integration coverage:

```bash
go test ./cmd/mars ./internal/queue ./internal/serve ./internal/tools
go test ./...
mars tools run docsync_audit --repo . --args-json '{}'
```

Installed-binary smoke setup:

```bash
make install
mktemp -d <validation-root>
mars start --repo <validation-root> --db <validation-root> --log-file <validation-root> --debug --exit-after-seed
mars start --repo <validation-root> --db <validation-root> --log-file <validation-root> --debug --exit-after-seed
mars start --repo <validation-root> --db <validation-root> --log-file <validation-root> --debug --exit-after-seed --new-lifecycle
mktemp -d <validation-root>
mars start --repo <validation-root> --db <validation-root> --log-file <validation-root> --debug --exit-after-seed
mars start --repo <validation-root> --db <validation-root> --log-file <validation-root> --debug --exit-after-seed
```

## Supporting Smoke Results

| Check | Result | Evidence |
| --- | --- | --- |
| Focused tests | Supporting smoke pass only | `cmd/mars`, `internal/queue`, `internal/serve`, and `internal/tools` passed. |
| Full Go suite | Supporting smoke pass only | `go test ./...` passed. |
| Docusync | Supporting smoke pass only | `docsync: checked 303 files, findings 0`. |
| Fresh startup | Supporting smoke pass only | Printed `startup_action=seeded_ceo role=ceo evidence=no active jobs, tickets, or dispositions found`. |
| Existing in-progress ticket | Supporting smoke pass only | Printed `startup_action=routed_existing_ticket role=engineer ... evidence=in-progress ticket T-001`. |
| Explicit reseed | Supporting smoke pass only | Printed `startup_action=seeded_ceo role=ceo evidence=--new-lifecycle requested`. |
| Dirty ambiguous state | Supporting smoke pass only | Exited non-zero and printed `startup_action=refused_ambiguous_state evidence=dirty workspace without deterministic ticket route: index.html`. |

## Live Sweep Follow-Up

The model-backed live validation requested by AD-297 was run later the same
day and is recorded separately:

```text
docs/validation/reports/2026-06-16-forward-progress-guard-live-run.md
```

The live run command shape was:

```bash
mars start --repo <ephemeral-static-web>
mars start --repo <ephemeral-phaser-game>
mars start --repo <ephemeral-go-api>
```

That follow-up partially proved the Go API forward path through Engineer and
successful validation, but static web and Phaser did not reach Engineer. The
live report records the foundation-owned blockers: shared inference-port
contention and CTO ticket-shaping guardrail wedges.
