# Mars Observer Validation — 2026-05-19

## Scope

- Profile: [mars-observer.md](../profiles/mars-observer.md)
- Requested profile target: `../mars`
- Resolved target: `/path/to/local-redacted`
- Target commit: `aa79b0039e7a2fb75c539fa427c02160ff2a33b9`
- Target branch state: `main...origin/main`
- Trust mode: `observer`
- Harness version: `0.41.19`

The relative `../mars` profile target does not exist next to this Codex
worktree. The trial used the canonical local Mars checkout under
`/path/to/local-redacted` and verified the real target worktree
stayed clean.

## Commands

| Command | Result | Notes |
| --- | --- | --- |
| `git -C /path/to/local-redacted status --short --branch` | Passed | Target was clean before and after the observer trial. |
| `go run ./cmd/mars-harness doctor --repo /path/to/local-redacted --json` | Warned | Core host checks passed; target-specific warnings showed missing `.harness`, missing per-repo DB directory, operating-model drift, role registry unavailable, deterministic remediation recommending `init`, and active-plan hygiene issues in Mars. |
| `go run ./cmd/mars-harness update check --repo /path/to/local-redacted --skip-remote --json` | Warned | Tool remote check was skipped; harness version was current at `0.41.19`; the recommended action was `mars-harness init --repo /path/to/local-redacted`. |
| `go run ./cmd/mars-harness tools run git_status --repo /path/to/local-redacted --trust observer --json` | Passed | Observer-trust read-only tool returned exit code 0 with empty output. |
| `go run ./cmd/mars-harness tools run file_write --repo /path/to/local-redacted --trust observer ...` | Blocked | Policy rejected `file_write` with `trust level observer cannot run mutating tool "file_write"` before writing. `observer-proof.txt` was not created. |
| `go run ./cmd/mars-harness run engineer --repo <validation-root> --dry-run --trace` | Passed on temp clone only | The real target was not used because `run --dry-run` auto-initializes missing `.harness/`. The temp clone proved context assembly, but also confirmed the command is not observer-safe for uninitialized real targets. |

## Findings

- Mars Harness can inspect Mars in observer mode without modifying the real
  checkout for doctor, update-check, and read-only tool execution.
- Observer trust blocks mutating built-in tools before they touch Mars.
- Mars is not ready for contributor-mode dogfood. It lacks `.harness/`, has
  operating-model drift, cannot load a role registry, and has active-plan
  hygiene warnings from its existing docs.
- The current profile command `run engineer --dry-run --trace` is unsafe against
  an uninitialized observer target because normal `run` auto-initializes the
  target before assembling the prompt.

## Follow-Up

- `T-009`: add a non-mutating observer dry-run or context-preview path for
  uninitialized targets so future observer reports do not need a temporary
  clone.
- A source-side maintainer must explicitly accept this observer report before
  any contributor-mode Mars trial.

## Assessment

The first observer trial is safe but not a graduation pass. It proves the
read-only inspection and trust-enforcement boundary, and it leaves the real
Mars checkout clean. It also identifies the next product gap: dry-run context
preview needs an observer-safe no-init mode before Mars Harness can claim a
clean observer validation path for uninitialized legacy targets.
