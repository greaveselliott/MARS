# Hardware-Gated Model Onboarding Proof

**Date:** 2026-06-28
**Source Ref:** working tree after hardware-gated model onboarding implementation
**Validation Type:** Focused unit/golden gates, clean target init replays, installed-style binary runtime preflight
**Model:** Local route blocked before model launch because GGUF weights are not installed; cloud route blocked before model launch because `ANTHROPIC_API_KEY` is not set.

## Primary Outcome Contract

**Primary Outcome:** Earn a real 10/10 confidence claim for hardware-gated setup/init, local eligibility, cloud routing, secret safety, CLI polish, and mirrored Fact-Validated Planning.
**Primary Pass Gate:** Confidence is 10/10 only after every proof gate passes or records a `primary_blocked` report with exact rerun commands for unavailable hardware, weights, or credentials.
**Primary Status:** `primary_blocked`
**Current Primary Blocker:** This host is eligible for `local-balanced-q4`, but the required local GGUF weights are missing. `ANTHROPIC_API_KEY` is also not set, so the cloud runtime proof cannot reach a live provider.
**Next Primary Action:** Install local weights and/or export cloud credentials, then rerun the commands in "Continuation Commands".
**Supporting Evidence:** Focused package tests, full `go test ./...`, docs/tool sync, secret scan, `make check`, `make install`, clean local/cloud init replays, installed-binary JSON runtime preflights, raw-key rejection, provider request-capture tests, and generated target doctrine checks passed.

## Proof Commands

```bash
go test ./internal/hardware ./internal/models ./internal/setup ./internal/inference ./internal/serve ./internal/safety ./internal/guardrails ./internal/scanner ./internal/tools ./cmd/mars
go test ./internal/tools -run TestMarsCLI
go test ./cmd/mars -run 'Test.*Command|Test.*CLI|Test.*Setup|Test.*Init'
/private/tmp/mars-proof-bin/mars guardrails secret-scan --repo . --json
/private/tmp/mars-proof-bin/mars docsync audit --repo . --json
go test ./...
make check
make install
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

## Clean Local Target Replay

**Target repo:** `/private/tmp/mars-validation-local-json-ihGJFd`

```bash
/private/tmp/mars-proof-bin/mars init \
  --repo /private/tmp/mars-validation-local-json-ihGJFd \
  --model-routing local \
  --local-bundle auto \
  --yes \
  --json
```

**Result:** Passed. Output was JSON only, the generated harness baseline was committed, `.harness/model-overrides.yaml` contained `routing: local` and `local_bundle: auto`, `.gitignore` contained `.harness/.env.local`, and generated `AGENTS.md` contained Fact-Validated Planning.

```bash
/Users/elliottgreaves/go/bin/mars start \
  --repo /private/tmp/mars-validation-local-json-ihGJFd \
  --exit-after-seed \
  --yes \
  --json \
  --db /private/tmp/mars-validation-local-json-ihGJFd.installed.mars.db \
  --log-file /private/tmp/mars-validation-local-json-ihGJFd.installed.start.log
```

**Result:** `primary_blocked`. Hardware eligibility passed and selected `local-balanced-q4`, but runtime preflight blocked before launch because the required weights are missing:

```json
{
  "status": "error",
  "remediation": "run mars setup --inference local --local-bundle auto --download --yes --json, then rerun the command"
}
```

## Clean Cloud Target Replay

**Target repo:** `/private/tmp/mars-validation-cloud-json-5huKDM`

```bash
/private/tmp/mars-proof-bin/mars init \
  --repo /private/tmp/mars-validation-cloud-json-5huKDM \
  --model-routing cloud \
  --cloud-provider anthropic \
  --cloud-model claude-test \
  --api-key-env ANTHROPIC_API_KEY \
  --yes \
  --json
```

**Result:** Passed. Output was JSON only, the generated harness baseline was committed, `.harness/model-overrides.yaml` stored `api_key_env: ANTHROPIC_API_KEY`, `.harness/.env.example` contained only `ANTHROPIC_API_KEY=`, `.gitignore` contained `.harness/.env.local`, and generated `AGENTS.md` contained Fact-Validated Planning.

```bash
/private/tmp/mars-proof-bin/mars models credentials write-local-env \
  --repo /private/tmp/mars-validation-cloud-json-5huKDM \
  --api-key-env ANTHROPIC_API_KEY \
  --yes \
  --json
```

**Result:** `primary_blocked`. `ANTHROPIC_API_KEY` was not present in the process environment, so no `.harness/.env.local` secret file was written.

```bash
/Users/elliottgreaves/go/bin/mars start \
  --repo /private/tmp/mars-validation-cloud-json-5huKDM \
  --exit-after-seed \
  --yes \
  --json \
  --db /private/tmp/mars-validation-cloud-json-5huKDM.installed.mars.db \
  --log-file /private/tmp/mars-validation-cloud-json-5huKDM.installed.start.log
```

**Result:** `primary_blocked`. Runtime blocked before provider use because `ANTHROPIC_API_KEY` was not set. No raw key value was accepted or printed.

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

Local runtime proof after weights are available:

```bash
make install
mars setup --inference local --local-bundle auto --download --yes --json
mars start --repo /private/tmp/mars-validation-local-json-ihGJFd --exit-after-seed --yes --json --db /private/tmp/mars-validation-local-json-ihGJFd.mars.db --log-file /private/tmp/mars-validation-local-json-ihGJFd.start.log
```

Cloud runtime proof after credentials are available:

```bash
export ANTHROPIC_API_KEY=<secret>
make install
mars models credentials write-local-env --repo /private/tmp/mars-validation-cloud-json-5huKDM --api-key-env ANTHROPIC_API_KEY --yes --json
mars start --repo /private/tmp/mars-validation-cloud-json-5huKDM --exit-after-seed --yes --json --db /private/tmp/mars-validation-cloud-json-5huKDM.mars.db --log-file /private/tmp/mars-validation-cloud-json-5huKDM.start.log
```

## Classification

- Hardware gating, local bundle selection, setup/init routing, runtime preflight, provider routing, JSON output, generated doctrine, and secret handling: foundation-owned, implemented and covered by supporting evidence.
- Missing local weights: environment blocker, recorded as `primary_blocked`.
- Missing Anthropic credential: environment blocker, recorded as `primary_blocked`.
- Final 10/10 confidence claim: not made in this report; claim is allowed only after one local runtime proof and one cloud runtime proof pass, or after the relevant unavailable resources remain recorded as blockers.
