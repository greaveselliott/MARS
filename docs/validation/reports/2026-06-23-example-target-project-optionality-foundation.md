# 2026-06-23 Example Target Project Optionality Foundation Validation

## Primary Outcome Contract

**Primary Outcome:** Prove Plan 1 of F-013: optional board-driven integration
substrate exists while no-config repos keep the 2026-06-23 CEO-led,
GitHub-compatible, strict-trunk behavior.

**Primary Pass Gate:** Targeted Plan 1 tests pass, docsync/docs consistency
passes, `go test ./...` and `make check` pass, and installed-binary
clean-project validation covers generated defaults plus orchestration.

**Primary Status:** `primary_blocked`

**Current Primary Blocker:** Plan 1 code remains implemented and released as
`v0.63.0`, but validation closure is still blocked. Focused Plan 1 gates pass,
and OpenAI-backed installed-binary clean-project replays now prove generated
defaults, `ceo-led` profile loading, schedule registration, and CEO -> COO ->
CTO orchestration on two clean targets. The remaining blockers are source broad
gates that still require unavailable local Gemma weights or unrelated existing
test repairs, clean-project Engineer/guardrail loops found by the live replays,
and the unavailable historical release marker `d8e8c6fcc990`.

**Next Primary Action:** Resolve or explicitly accept the recorded blockers
before promoting Plan 2. The smallest likely next source slice is to make broad
source tests and clean-project validation usable on a hosted
OpenAI-compatible endpoint without requiring local Gemma weights, then address
the live Engineer/guardrail loops as foundation-owned follow-up evidence.

**Supporting Evidence:** Focused Plan 1 unit and docs gates passed; release
assets and GitHub mirror remain valid; OpenAI-backed clean-project validation
advanced through real orchestration before hitting diagnosed non-optionality
blockers.

## Matrix Selection

- **Selected Matrix:** AD-284 minimum for Plan 1 foundation runtime/generated
  defaults change.
- **Selected Cases Or Archetypes:** Orchestration plus generated defaults:
  optional integrations loader, scheduler rebuild/suppression, executor
  effective-tool hook, init/upgrade example config, static browser clean target,
  and Go API clean target.
- **Source Ref:** Local working tree on `main` at the F-013 Plan 1 implementation
  state on 2026-06-23.
- **Binary:** Installed `/path/to/local-redacted`, reporting
  `mars-harness 0.63.0 darwin/arm64 commit=unknown built=unknown`.
- **Model Identity:** Clean-project replays used a local auth-injecting
  OpenAI-compatible proxy on `http://127.0.0.1:18654/v1`, backed by
  `gpt-4.1-mini-2025-04-14`. A one-token probe returned `OK`. No API key value
  was written to repo artifacts, command text, or logs.
- **Target/Run Paths:** Source repo
  `/path/to/local-redacted`;
  clean-project replay root
  `<validation-root>`.
- **Cleanup Status:** Static and API `start` processes were stopped after
  diagnosed wedges, and the temporary OpenAI proxy was stopped after evidence
  capture. Target worktrees remain under `/tmp` with uncommitted
  generated product files for inspection.

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
| `/path/to/local-redacted auth github check` | PASS | Private release auth resolves via `env-github-token`; no token value logged. |
| `GOCACHE=<validation-root> make install` | PASS | Installed binary to `/path/to/local-redacted` and updated fish PATH config. |
| `/path/to/local-redacted doctor --json` | FAIL | Local setup was stopped after operator clarified this 16 GB laptop should use OpenAI-hosted validation; doctor still requires missing local `google_gemma-4-E4B-it-Q5_K_M.gguf` for the balanced profile. |
| OpenAI proxy probe through `http://127.0.0.1:18654/v1/chat/completions` | PASS | Real upstream model `gpt-4.1-mini-2025-04-14` returned `OK`; key supplied via `launchctl getenv OPENAI_API_KEY`. |
| `GOCACHE=<validation-root> GOMODCACHE=<validation-root> go test ./...` | FAIL | Ran with normal localhost/process permissions. Fails include missing local `google_gemma-4-E4B-it-Q5_K_M.gguf` in serve/start tests that do not take `--model-endpoint`, `cmd/mars-harness` release/auth test state, `internal/personas` trailing-newline fixture drift, `internal/scoring` legacy fixture drift, and `internal/codeintel` refresh expectation. |
| `GOCACHE=<validation-root> GOMODCACHE=<validation-root> make check` | FAIL | `CGO_ENABLED=0 go build ./cmd/mars-harness` passed. Race/coverage suite then failed on the same broad-test blockers; coverage gate also reported missing/stale floor entries and `internal/config`/`internal/trace` floor misses. |
| `GOCACHE=<validation-root> go run ./cmd/mars-harness release notes --repo . --bump auto` | PASS | Generated `0.63.0` release files from commit `f0cb84b`. |
| `git fetch origin '+refs/heads/*:refs/remotes/origin/*' '+refs/tags/*:refs/tags/*'` | PASS | Full remote heads/tags fetched. Repository is not shallow. |
| `git fetch origin d8e8c6fcc990` | FAIL | Remote does not advertise the historical marker ref. |
| `GOCACHE=<validation-root> GOMODCACHE=<validation-root> /path/to/local-redacted release backfill-notes --repo . --check` | BLOCKED | Legacy marker `d8e8c6fcc990` remains unavailable after full fetch, so the required backfill check cannot complete. |
| `git push origin main` | PASS | Pushed feature commit `f0cb84b` and release-note commit `968250c` to `origin/main`. |
| `git tag -f v0.63.0 968250c` | PASS | Tagged the release-note commit locally. |
| `git push origin v0.63.0` | PASS | Pushed tag `v0.63.0`. |
| `GOCACHE=<validation-root> go run ./cmd/mars-harness release publish-assets --repo . --version v0.63.0 --upload auto` | PASS | Built local assets and uploaded the GitHub mirror. |
| `/path/to/local-redacted release verify-assets --dist dist/releases --version v0.63.0` | PASS | Verified local dist assets and checksums. |
| `gh release view v0.63.0 --repo greaveselliott/mars-harness --json tagName,name,url,isDraft,isPrerelease` | PASS | Verified GitHub release mirror at `https://github.com/greaveselliott/mars-harness/releases/tag/v0.63.0`. |
| `mars-harness start --repo <validation-root> ... --model-endpoint http://127.0.0.1:18654/v1` | PARTIAL PASS / BLOCKED | Generated defaults, wrote `.harness/integrations.example.yaml`, did not write `.harness/integrations.yaml`, loaded `flow_profile="ceo-led"`, replaced 8 schedules, and completed CEO -> COO -> CTO through the real OpenAI endpoint. Engineer then wedged after writing React/Vitest-style files for a no-package-manager static target and entering unresolved `npm` guardrail loops. Log: `<validation-root>`. |
| `mars-harness start --repo <validation-root> ... --model-endpoint http://127.0.0.1:18654/v1` | PARTIAL PASS / BLOCKED | Generated defaults, wrote `.harness/integrations.example.yaml`, did not write `.harness/integrations.yaml`, loaded `flow_profile="ceo-led"`, replaced 8 schedules, and completed CEO -> COO -> CTO through the real OpenAI endpoint. Engineer created Go API files and tests but wedged after unresolved `AddTaskNote`/test repair guardrail blocks. Log: `<validation-root>`. |

## Failure Classes

- **Foundation-owned:** Clean-project live replay blockers are foundation-owned
  follow-up evidence: hosted endpoint validation currently needs a local proxy
  because `start --model-endpoint` does not pass an API key into executor
  clients yet; source broad tests still require local model files even when the
  operator wants hosted validation; Engineer/guardrail loops were reproduced in
  both static and API archetypes. These are not optionality-regression evidence,
  but they block Plan 1 validation closure.
- **Deployed-owned:** None assigned. Temporary validation target mutations are
  evidence-only and remain under `/tmp`; they are not product backlog
  work for an operator-owned target repo.
- **Environment/blocked:** Local Gemma weight remains incomplete by operator
  choice because this 16 GB laptop should validate through OpenAI-hosted models,
  not local balanced-profile weights. Source tests without `--model-endpoint`
  still fail on that local preflight.
- **Release/partial:** `0.63.0` notes, push, tag, local assets, local
  verification, and GitHub mirror verification passed. The required backfill
  check remains blocked because legacy marker commit `d8e8c6fcc990` is
  unavailable locally and from the remote.
- **Mixed/unclear:** Broad source failures include unrelated existing fixture
  and coverage drift (`internal/personas`, `internal/scoring`,
  `internal/codeintel`, coverage floors). They prevent closure but are not
  evidence that F-013-S001 optionality behavior regressed.

## Replay Commands

```bash
mars-harness auth github check
launchctl setenv OPENAI_API_KEY <set outside Codex; do not print the value>
OPENAI_API_KEY="$(launchctl getenv OPENAI_API_KEY)" OPENAI_PROXY_PORT=18654 python3 <validation-root>
GOCACHE=<validation-root> GOMODCACHE=<validation-root> go test ./...
GOCACHE=<validation-root> GOMODCACHE=<validation-root> make check
GOCACHE=<validation-root> GOMODCACHE=<validation-root> /path/to/local-redacted release backfill-notes --repo . --check
make install
/path/to/local-redacted start --repo <validation-root> --db <validation-root> --log-file <validation-root> --concurrency 1 --debug --addr 127.0.0.1:19101 --dashboard-addr 127.0.0.1:19201 --model-endpoint http://127.0.0.1:18654/v1
/path/to/local-redacted start --repo <validation-root> --db <validation-root> --log-file <validation-root> --concurrency 1 --debug --addr 127.0.0.1:19111 --dashboard-addr 127.0.0.1:19211 --model-endpoint http://127.0.0.1:18654/v1
```

If the operator prefers a full lifecycle target instead of agent-smoke, use the
AD-284 replay profile from `docs/design-docs/validation-matrix-gating.md` after
`make install` succeeds.
