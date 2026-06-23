# 2026-06-23 Example Target Project Optionality Foundation Validation

## Primary Outcome Contract

**Primary Outcome:** Prove Plan 1 of F-013: optional board-driven integration
substrate exists while no-config repos keep the 2026-06-23 CEO-led,
GitHub-compatible, strict-trunk behavior.

**Primary Pass Gate:** Targeted Plan 1 tests pass, docsync/docs consistency
passes, `go test ./...` and `make check` pass, release backfill passes, and
installed-binary clean-project validation covers generated defaults plus
orchestration without local GGUF models.

**Primary Status:** `primary_passed`

**Current Primary Blocker:** None. Plan 1 validation blockers are closed.

**Next Primary Action:** Start Plan 2 from `T-045` only; do not add
prioritisation, Figma, PR delivery, or frontier model routing in the JIRA mirror
slice.

**Supporting Evidence:** Focused tests, broad source gates, release backfill,
installed-binary agent-smoke, and full `start` clean-project replays passed for
the Plan 1 closure scope.

**Closure Summary:** The blocker set from the previous report is closed. Broad
source gates no longer require local offline model files for tests that do not
exercise local model preflight; deterministic drift in codeintel, scoring, and
persona docs is repaired; release backfill markers now resolve; and
installed-binary validation with an OpenAI-compatible endpoint completes the
static-browser and Go API Engineer handoffs without reproducing the previous
package-manager or Go source-repair loops.

## Matrix Selection

- **Selected Matrix:** AD-284 minimum for Plan 1 foundation runtime/generated
  defaults change.
- **Selected Cases Or Archetypes:** Orchestration plus generated defaults:
  optional integrations loader, scheduler rebuild/suppression, executor
  effective-tool hook, init/upgrade example config, static browser clean target,
  and Go API clean target.
- **Source Ref:** Local working tree on `main` during Plan 1 validation closure
  on 2026-06-23.
- **Binary:** Installed `/path/to/local-redacted` from the
  current source tree with `make install`.
- **Model Identity:** Clean-project replays used a local auth-injecting
  OpenAI-compatible proxy on `http://127.0.0.1:18654/v1`, backed by
  `gpt-4.1-mini`. The API key came from `launchctl getenv OPENAI_API_KEY`; no
  key value was written to repo artifacts, command output, traces, or logs.
- **Target/Run Paths:** Source repo
  `/path/to/local-redacted`; agent
  smoke root `<validation-root>`;
  full start replay root
  `<validation-root>`.
- **Cleanup Status:** The OpenAI proxy was stopped after validation. Full
  `start` replays were stopped after the required Engineer terminal handoff to
  avoid spending into downstream QA cycles; retained temp targets remain under
  `/tmp` for inspection.

## Commands And Results

| Command | Result | Notes |
| --- | --- | --- |
| `git diff --check` | PASS | No whitespace errors. |
| `go test -count=1 ./cmd/mars-harness -run 'TestStartCommand'` | PASS | Start command tests inject a hosted endpoint and no longer trip unrelated local model preflight. |
| `go test -count=1 ./internal/serve` | PASS | Includes focused missing-local-model preflight coverage. |
| `go test -count=1 ./internal/codeintel ./internal/scoring ./internal/personas` | PASS | Deterministic DB, pinned scoring clock, and canonical persona doc newline drift repaired. |
| `go test -count=1 ./internal/integrations ./internal/scheduler ./internal/docsync ./internal/docsconsistency` | PASS | Plan 1 focused packages remain green. |
| `GOCACHE=<validation-root> go test ./...` | PASS | Broad source gate passes on this laptop without local GGUF model validation. |
| `GOCACHE=<validation-root> make check` | PASS | Race/coverage, coverage ratchet, fuzz smoke, and lint/vet path pass; `govulncheck` is not installed and was skipped by the Makefile. |
| `GOCACHE=<validation-root> go run ./cmd/mars-harness release backfill-notes --repo . --check` | PASS | Checked 244 entries, changed 0 after marker repair. |
| `make install` | PASS | Installed current binary to `/path/to/local-redacted`; fish PATH already configured. |
| `/path/to/local-redacted auth github check` | PASS | Private release auth resolves via `env-github-token`; no token value logged. |
| `mars-harness validation agent-smoke --case static-web-ticket --model-endpoint http://127.0.0.1:18654/v1` | PASS | Installed-binary live Engineer case completed with `next_need=qa_review` and suggested `qa`; retained report at `<validation-root>`. |
| `mars-harness validation agent-smoke --suite default --case go-api-ticket --model-endpoint http://127.0.0.1:18654/v1` | PASS | Installed-binary live Engineer case completed with `next_need=qa_review` and suggested `qa`; retained report at `<validation-root>`. |
| `mars-harness start --repo <validation-root> ... --model-endpoint http://127.0.0.1:18654/v1` | PASS | Generated defaults, wrote `.harness/integrations.example.yaml`, did not write `.harness/integrations.yaml`, loaded `flow_profile="ceo-led"`, replaced 8 schedules, completed CEO -> COO -> CTO -> Engineer, and Engineer delivered plain `index.html`, `style.css`, and `app.js` with `node --check`, Python static server, curl smoke, and no package manager. |
| `mars-harness start --repo <validation-root> ... --model-endpoint http://127.0.0.1:18654/v1` | PASS | Generated defaults, wrote `.harness/integrations.example.yaml`, did not write `.harness/integrations.yaml`, loaded `flow_profile="ceo-led"`, replaced 8 schedules, completed CEO -> COO -> CTO -> Engineer, and Engineer delivered Go API source/tests with `go test`, `go build`, `go run`, and HTTP CRUD curl smoke. |

## Findings Closed

- **Model preflight leakage:** `serve.New` now requires local model preflight
  only when explicitly configured by real serve/start paths without
  `--model-endpoint`. Tests that do not exercise local inference inject an
  endpoint; one focused test preserves the missing-local-model failure.
- **Deterministic drift:** Codeintel uses an isolated DB in the drifting test,
  scoring legacy fixture uses a pinned clock, and persona docs are rendered with
  a single normal trailing newline.
- **Release backfill:** Unreachable historical changelog markers were replaced
  with reachable commits; the backfill check now passes.
- **Static live loop:** CTO/Engineer now respect explicit no-package-manager
  static targets. The full replay produced no `package.json`, JSX, Vite, or
  npm path.
- **Go API live loop:** Engineer was able to write product source/tests,
  validate, commit, move the ticket to done, and record terminal handoff.
- **Coverage ratchet:** Added focused tests for config, trace, and serve helper
  paths rather than lowering floors.

## Residual Observations

- The full static `start` replay exposed a downstream QA smoke-order issue
  after Engineer had already completed; this was outside the Plan 1 pass gate
  and should become a future QA/dogfood improvement ticket only if it recurs.
- The full `start` replays were manually stopped after Engineer handoff, so the
  retained logs include context-cancelled QA/rework jobs caused by operator stop.
  These are not treated as product or Plan 1 failures.
- Local offline model validation remains intentionally out of scope for this
  16 GB laptop; hosted OpenAI-compatible validation is the accepted local route.

## Replay Commands

```bash
mars-harness auth github check
launchctl setenv OPENAI_API_KEY <set outside Codex; do not print the value>
OPENAI_API_KEY="$(launchctl getenv OPENAI_API_KEY)" OPENAI_PROXY_PORT=18654 OPENAI_VALIDATION_MODEL=gpt-4.1-mini python3 <validation-root>
GOCACHE=<validation-root> go test ./...
GOCACHE=<validation-root> make check
GOCACHE=<validation-root> go run ./cmd/mars-harness release backfill-notes --repo . --check
make install
/path/to/local-redacted validation agent-smoke --case static-web-ticket --root <validation-root> --model-endpoint http://127.0.0.1:18654/v1 --max-turns 16 --timeout 10m --keep-runs --report <validation-root> --json
/path/to/local-redacted validation agent-smoke --suite default --case go-api-ticket --root <validation-root> --model-endpoint http://127.0.0.1:18654/v1 --max-turns 20 --timeout 10m --keep-runs --report <validation-root> --json
/path/to/local-redacted start --repo <validation-root> --db <validation-root> --log-file <validation-root> --concurrency 1 --debug --addr 127.0.0.1:19131 --dashboard-addr 127.0.0.1:19231 --new-lifecycle --model-endpoint http://127.0.0.1:18654/v1
/path/to/local-redacted start --repo <validation-root> --db <validation-root> --log-file <validation-root> --concurrency 1 --debug --addr 127.0.0.1:19141 --dashboard-addr 127.0.0.1:19241 --new-lifecycle --model-endpoint http://127.0.0.1:18654/v1
```
