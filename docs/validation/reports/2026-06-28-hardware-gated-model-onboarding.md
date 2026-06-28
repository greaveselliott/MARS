# Hardware-Gated Model Onboarding Proof

**Date:** 2026-06-28
**Source Ref:** `0c0636a` plus local runtime continuation evidence gathered with installed `/Users/elliottgreaves/go/bin/mars` reporting `0.67.0`
**Validation Type:** Focused unit/golden gates, clean target init replays, installed-binary local setup/download, installed-binary runtime preflight
**Model:** Local route passed after downloading the eligible `local-balanced-q4` bundle. Cloud route remains blocked before provider use because no supported provider credential environment variable is set.

## Primary Outcome Contract

**Primary Outcome:** Earn a real 10/10 confidence claim for hardware-gated setup/init, local eligibility, cloud routing, secret safety, CLI polish, and mirrored Fact-Validated Planning.
**Primary Pass Gate:** Confidence is 10/10 only after every proof gate passes or records a `primary_blocked` report with exact rerun commands for unavailable hardware, weights, or credentials.
**Primary Status:** `primary_blocked`
**Current Primary Blocker:** Local runtime proof now passes. Cloud runtime proof cannot reach a live provider because `ANTHROPIC_API_KEY`, `OPENAI_API_KEY`, `GEMINI_API_KEY`, `MISTRAL_API_KEY`, `XAI_API_KEY`, `DEEPSEEK_API_KEY`, `GROQ_API_KEY`, and `COHERE_API_KEY` are all absent from the process environment.
**Next Primary Action:** Export a supported cloud provider credential, write it into the target's ignored local env file, then rerun the cloud start proof in "Continuation Commands".
**Supporting Evidence:** Focused package tests, full `go test ./...`, docs/tool sync, secret scan, `make check`, `make install`, clean local/cloud init replays, installed-binary local setup/download, installed-binary local JSON runtime pass, installed-binary cloud JSON credential block, raw-key rejection, provider request-capture tests, and generated target doctrine checks passed.

## Proof Commands

```bash
go test ./internal/hardware ./internal/models ./internal/setup ./internal/inference ./internal/serve ./internal/safety ./internal/guardrails ./internal/scanner ./internal/tools ./cmd/mars
go test ./internal/tools -run TestMarsCLI
go test ./cmd/mars -run 'Test.*Command|Test.*CLI|Test.*Setup|Test.*Init'
/Users/elliottgreaves/go/bin/mars guardrails secret-scan --repo . --json
/Users/elliottgreaves/go/bin/mars docsync audit --repo . --json
go test ./...
make check
make install
/Users/elliottgreaves/go/bin/mars version
/Users/elliottgreaves/go/bin/mars setup --inference local --local-bundle auto --download --yes --json
```

## Gate Results

- Focused tests: passed.
- CLI reference tests: passed.
- Command-shape tests: passed.
- Secret scan: passed with `{"status":"ok","findings":null}`.
- DocSync audit: passed with `findings: null`.
- Full `go test ./...`: passed.
- `make check`: passed after rerunning with normal Go cache access outside the sandbox; coverage ratchet passed with `internal/serve` at 66.2%, vulnerability scan found no called vulnerabilities, and fuzz smoke passed.
- `make install`: passed and installed `/Users/elliottgreaves/go/bin/mars`.
- Installed binary version: passed with `mars 0.67.0 darwin/arm64`.
- Local setup/download: passed with `{"status":"ok","steps_run":5,"steps_skipped":2}` after downloading `Qwen3-Coder-30B-A3B-Instruct-Q4_K_M.gguf` and `google_gemma-4-E4B-it-Q5_K_M.gguf`.

## Clean Local Target Replay

**Target repo:** `/private/tmp/mars-confidence10-local-2ah3cK`

```bash
/Users/elliottgreaves/go/bin/mars init \
  --repo /private/tmp/mars-confidence10-local-2ah3cK \
  --model-routing local \
  --local-bundle auto \
  --yes \
  --json
```

**Result:** Passed. Output was JSON only, the generated harness baseline was committed, `.harness/model-overrides.yaml` contained `routing: local` and `local_bundle: auto`, `.gitignore` contained `.harness/.env.local`, and generated `AGENTS.md` contained Fact-Validated Planning.

```bash
/Users/elliottgreaves/go/bin/mars start \
  --repo /private/tmp/mars-confidence10-local-2ah3cK \
  --exit-after-seed \
  --yes \
  --json \
  --db /private/tmp/mars-confidence10-local-2ah3cK.mars.db \
  --log-file /private/tmp/mars-confidence10-local-2ah3cK.start.log
```

**Result:** Passed. Hardware eligibility selected `local-balanced-q4`, local model files were present after setup/download, the generated harness baseline was committed, and the seed path returned JSON-only success:

```json
{
  "repo": "/private/tmp/mars-confidence10-local-2ah3cK",
  "seeded": true,
  "status": "ok"
}
```

## Clean Cloud Target Replay

**Target repo:** `/private/tmp/mars-confidence10-cloud-iGtSnZ`

```bash
/Users/elliottgreaves/go/bin/mars init \
  --repo /private/tmp/mars-confidence10-cloud-iGtSnZ \
  --model-routing cloud \
  --cloud-provider anthropic \
  --cloud-model claude-test \
  --api-key-env ANTHROPIC_API_KEY \
  --yes \
  --json
```

**Result:** Passed. Output was JSON only, the generated harness baseline was committed, `.harness/model-overrides.yaml` stored `api_key_env: ANTHROPIC_API_KEY`, `.harness/.env.example` contained only `ANTHROPIC_API_KEY=`, `.gitignore` contained `.harness/.env.local`, and generated `AGENTS.md` contained Fact-Validated Planning.

```bash
/Users/elliottgreaves/go/bin/mars models credentials write-local-env \
  --repo /private/tmp/mars-confidence10-cloud-iGtSnZ \
  --api-key-env ANTHROPIC_API_KEY \
  --yes \
  --json
```

**Result:** `primary_blocked`. `ANTHROPIC_API_KEY` was not present in the process environment, so no `.harness/.env.local` secret file was written.

```bash
/Users/elliottgreaves/go/bin/mars start \
  --repo /private/tmp/mars-confidence10-cloud-iGtSnZ \
  --exit-after-seed \
  --yes \
  --json \
  --db /private/tmp/mars-confidence10-cloud-iGtSnZ.mars.db \
  --log-file /private/tmp/mars-confidence10-cloud-iGtSnZ.start.log
```

**Result:** `primary_blocked`. Runtime blocked before provider use because `ANTHROPIC_API_KEY` was not set. No raw key value was accepted or printed.

Credential presence probe, by variable name only, found no usable cloud provider credential:

```text
ANTHROPIC_API_KEY=missing
OPENAI_API_KEY=missing
GEMINI_API_KEY=missing
MISTRAL_API_KEY=missing
XAI_API_KEY=missing
DEEPSEEK_API_KEY=missing
GROQ_API_KEY=missing
COHERE_API_KEY=missing
```

## Secret And JSON Proof

- Raw `--api-key` is rejected with JSON-only output and remediation to use `--api-key-env <ENV_NAME>`.
- `--json` setup/init/start/model inspection paths suppress logs, TTY display events, styling, and animations.
- Secret scan skips ignored `.harness/.env.local`, redacts matches, and returns JSON-only output in automation mode.
- `.harness/.env.local` writer is tested for `0600`; `.harness/.env.example` is tested for env names only.

## Provider Evidence

Provider registry tests require every selectable provider to carry official-doc evidence and request-capture coverage. OpenAI-compatible providers are exercised through table-driven request-capture tests; Anthropic is exercised through a native Messages API request-capture test. Cohere remains unavailable until a native request-capture adapter is implemented.

Official documentation URLs recorded in the provider registry:

- OpenAI Chat Completions: `https://platform.openai.com/docs/api-reference/chat/create`
- Anthropic Messages: `https://docs.anthropic.com/en/api/messages`
- Gemini OpenAI compatibility: `https://ai.google.dev/gemini-api/docs/openai`
- Mistral chat completion: `https://docs.mistral.ai/capabilities/completion/`
- xAI API reference: `https://docs.x.ai/docs/api-reference`
- DeepSeek chat completion: `https://api-docs.deepseek.com/api/create-chat-completion`
- Groq API reference: `https://console.groq.com/docs/api-reference`
- Ollama OpenAI compatibility: `https://docs.ollama.com/api/openai-compatibility`
- Cohere chat reference: `https://docs.cohere.com/reference/chat`

## Continuation Commands

Local runtime proof has passed. Rerun command if the local model directory changes:

```bash
mars start --repo /private/tmp/mars-confidence10-local-2ah3cK --exit-after-seed --yes --json --db /private/tmp/mars-confidence10-local-2ah3cK.mars.db --log-file /private/tmp/mars-confidence10-local-2ah3cK.start.log
```

Cloud runtime proof after credentials are available:

```bash
export ANTHROPIC_API_KEY=<secret>
make install
mars models credentials write-local-env --repo /private/tmp/mars-confidence10-cloud-iGtSnZ --api-key-env ANTHROPIC_API_KEY --yes --json
mars start --repo /private/tmp/mars-confidence10-cloud-iGtSnZ --exit-after-seed --yes --json --db /private/tmp/mars-confidence10-cloud-iGtSnZ.mars.db --log-file /private/tmp/mars-confidence10-cloud-iGtSnZ.start.log
```

## Classification

- Hardware gating, local bundle selection, setup/init routing, runtime preflight, provider routing, JSON output, generated doctrine, and secret handling: foundation-owned, implemented and covered by supporting evidence.
- Missing local weights: resolved by installed-binary `mars setup --inference local --local-bundle auto --download --yes --json`.
- Missing cloud provider credential: environment blocker, recorded as `primary_blocked`.
- Final 10/10 confidence claim: not made in this report; the remaining missing evidence is one live cloud runtime proof using a real provider credential.
