# Factory Forward Progress Guard Smoke

Date: 2026-06-16
Source ref: local working tree before commit
Binary: installed with `make install` to `/path/to/local-redacted`
Owner: foundation

## Scope

Validated the foundation-owned startup and routing guard added by AD-297:

- startup action is printed through the installed `mars-harness start` surface
- fresh target seeds CEO only when no resumable state exists
- existing in-progress ticket routes to Engineer instead of reseeding CEO
- `--new-lifecycle` explicitly reseeds CEO over existing work
- dirty product workspace without a deterministic ticket/disposition route refuses as ambiguous

This report is a bounded startup smoke. It is not a full model-backed static web,
Phaser, and Go API lifecycle sweep.

## Commands

Unit and integration coverage:

```bash
go test ./cmd/mars-harness ./internal/queue ./internal/serve ./internal/tools
go test ./...
mars-harness tools run docsync_audit --repo . --args-json '{}'
```

Installed-binary smoke setup:

```bash
make install
mktemp -d <validation-root>
mars-harness start --repo <validation-root> --db <validation-root> --log-file <validation-root> --debug --exit-after-seed
mars-harness start --repo <validation-root> --db <validation-root> --log-file <validation-root> --debug --exit-after-seed
mars-harness start --repo <validation-root> --db <validation-root> --log-file <validation-root> --debug --exit-after-seed --new-lifecycle
mktemp -d <validation-root>
mars-harness start --repo <validation-root> --db <validation-root> --log-file <validation-root> --debug --exit-after-seed
mars-harness start --repo <validation-root> --db <validation-root> --log-file <validation-root> --debug --exit-after-seed
```

## Results

| Check | Result | Evidence |
| --- | --- | --- |
| Focused tests | Pass | `cmd/mars-harness`, `internal/queue`, `internal/serve`, and `internal/tools` passed. |
| Full Go suite | Pass | `go test ./...` passed. |
| Docusync | Pass | `docsync: checked 303 files, findings 0`. |
| Fresh startup | Pass | Printed `startup_action=seeded_ceo role=ceo evidence=no active jobs, tickets, or dispositions found`. |
| Existing in-progress ticket | Pass | Printed `startup_action=routed_existing_ticket role=engineer ... evidence=in-progress ticket T-001`. |
| Explicit reseed | Pass | Printed `startup_action=seeded_ceo role=ceo evidence=--new-lifecycle requested`. |
| Dirty ambiguous state | Pass | Exited non-zero and printed `startup_action=refused_ambiguous_state evidence=dirty workspace without deterministic ticket route: index.html`. |

## Remaining Live Sweep

The model-backed live validation requested by AD-297 remains unrun here:

```bash
mars-harness start --repo <ephemeral-static-web>
mars-harness start --repo <ephemeral-phaser-game>
mars-harness start --repo <ephemeral-go-api>
```

That sweep is intentionally recorded as remaining evidence because previous
MacBook Pro M1 Max full sweeps have taken around 20 hours. The startup guard
has installed-binary smoke evidence and full deterministic tests, but claims
about time-to-first-Engineer, time-to-first-build, no planner oscillation in a
real model loop, and first successful smoke still require the full lifecycle
runner/hardware lane.
