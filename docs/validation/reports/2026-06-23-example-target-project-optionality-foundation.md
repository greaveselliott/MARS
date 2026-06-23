# 2026-06-23 Example Target Project Optionality Foundation Validation

## Primary Outcome Contract

**Primary Outcome:** Prove Plan 1 of F-013: optional board-driven integration
substrate exists while no-config repos keep the 2026-06-23 CEO-led,
GitHub-compatible, strict-trunk behavior.

**Primary Pass Gate:** Targeted Plan 1 tests pass, docsync/docs consistency
passes, `go test ./...` and `make check` pass, and installed-binary
clean-project validation covers generated defaults plus orchestration.

**Primary Status:** `primary_blocked`

**Current Primary Blocker:** Local environment blocks broad and live gates:
missing private release auth, missing local quality-profile model files,
sandboxed writes under `/path/to/local-redacted`, sandboxed localhost
`httptest` binding, sandboxed codeintel DB opens, and background process kill
assertions that cannot observe child termination.

**Next Primary Action:** Run the replay commands below from an environment with
private release auth, writable Go/module/home caches, permitted localhost
listeners, local model weights or explicit model endpoint, and normal process
inspection.

**Supporting Evidence:** Focused Plan 1 unit and docs gates passed; broad gates
were attempted and failed for environment reasons listed below.

## Matrix Selection

- **Selected Matrix:** AD-284 minimum for Plan 1 foundation runtime/generated
  defaults change.
- **Selected Cases Or Archetypes:** Orchestration plus generated defaults:
  optional integrations loader, scheduler rebuild/suppression, executor
  effective-tool hook, init/upgrade example config. Clean-project archetype
  replay was blocked before execution.
- **Source Ref:** Local working tree on `main` at the F-013 Plan 1 implementation
  state on 2026-06-23.
- **Binary:** Source tree tests used `go test`; installed-binary clean-project
  validation was not executed because `make check` and local environment gates
  were blocked first.
- **Model Identity:** No live model validation completed. Broad tests attempted
  local quality profile and failed because required model files were absent.
- **Target/Run Paths:** Source repo
  `/path/to/local-redacted`; temporary
  test targets under macOS `$TMPDIR`.
- **Cleanup Status:** No persistent validation target was created. `make check`
  was interrupted after repeated hard failures and two minutes with no new
  output.

## Commands And Results

| Command | Result | Notes |
| --- | --- | --- |
| `git diff --check` | PASS | No whitespace errors. |
| `GOCACHE=<validation-root> go test -count=1 ./internal/integrations` | PASS | Loader defaults, profile gates, unknown fields, schedule suppression helpers. |
| `GOCACHE=<validation-root> go test -count=1 ./internal/scheduler` | PASS | Schedule replacement swaps atomically and rejects invalid replacement without clobbering current schedules. |
| `GOCACHE=<validation-root> go test -count=1 ./internal/serve -run 'TestEffectiveToolAllowlist\|TestServer_registerCronSchedules'` | PASS | No-config allowlist unchanged; gated registered future tools only; board-driven planning schedules suppressed and stale schedules replaced. |
| `GOCACHE=<validation-root> go test -count=1 ./internal/scanner -run 'TestInit_success\|TestUpgrade_preservesUserConfiguredManifestAndPrompts'` | PASS | Init/upgrade write `.harness/integrations.example.yaml` and do not write `.harness/integrations.yaml`. |
| `GOCACHE=<validation-root> go test -count=1 ./internal/docsync` | PASS | New `internal/integrations` docsync map is covered. |
| `GOCACHE=<validation-root> go test -count=1 ./internal/docsconsistency` | PASS | Active plan, feature catalog, strict-trunk default docs, and docsync audit pass. |
| `GOCACHE=<validation-root> go test ./...` | FAIL | Environment blockers: private release auth missing, model files missing, Go module stat-cache writes denied, `httptest` listener bind denied, codeintel DB open denied, background process kill tests could not observe child termination. |
| `GOCACHE=<validation-root> make check` | BLOCKED | Began `CGO_ENABLED=0 go build` and race/coverage tests, hit the same environment blockers, then was interrupted after two quiet minutes with prior hard failures. |

## Failure Classes

- **Foundation-owned:** None identified from the focused Plan 1 slice.
- **Deployed-owned:** None; no target repo was mutated.
- **Environment/blocked:** Missing private release auth, missing model weights,
  sandboxed home/cache/log writes, sandboxed localhost listener binds, sandboxed
  codeintel DB opens, process-inspection limitations.
- **Mixed/unclear:** Full `make check` did not reach a clean terminal report
  before interruption, so the official gate remains blocked rather than failed
  by Plan 1 code.

## Replay Commands

```bash
mars-harness auth github check
mars-harness setup
GOCACHE=<validation-root> go test ./...
GOCACHE=<validation-root> make check
make install
mars-harness validation agent-smoke --suite fast --role engineer --project-type static-web --keep-runs
mars-harness validation agent-smoke --suite fast --role engineer --project-type go-api --keep-runs
```

If the operator prefers a full lifecycle target instead of agent-smoke, use the
AD-284 replay profile from `docs/design-docs/validation-matrix-gating.md` after
`make install` succeeds.
