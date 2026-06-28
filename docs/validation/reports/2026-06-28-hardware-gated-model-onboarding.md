# Hardware-Gated Model Onboarding Proof

**Date:** 2026-06-28
**Source Ref:** `1455efb` (`v0.68.3`) with installed `/Users/elliottgreaves/go/bin/mars` reporting `0.68.3`; corrected live-provider proof gathered after this ref
**Validation Type:** Focused unit/golden gates, clean target init replays, installed-binary local setup/download, installed-binary runtime preflight, installed-binary OpenAI route plumbing, installed-binary OpenAI provider-call proof
**Model:** Local route passed after downloading the eligible `local-balanced-q4` bundle. Cloud route plumbing passed with an OpenAI-routed clean target using `OPENAI_API_KEY` by environment variable name only. Real OpenAI usage was proven separately by `mars models evaluate` and a one-turn `mars run orchestrator` on synthetic ephemeral content.

## Primary Outcome Contract

**Primary Outcome:** Earn a real 10/10 confidence claim for hardware-gated setup/init, local eligibility, cloud routing, secret safety, CLI polish, and mirrored Fact-Validated Planning.
**Primary Pass Gate:** Confidence is 10/10 only after every proof gate passes or records a `primary_blocked` report with exact rerun commands for unavailable hardware, weights, or credentials.
**Primary Status:** `passed`
**Current Primary Blocker:** None. The remaining blocker was resolved by a fish universal `OPENAI_API_KEY`, an OpenAI-routed clean target, ignored local env-file write, `mars models evaluate` token-usage proof, and a one-turn installed-binary `mars run orchestrator` proof with `llm_calls=1`.
**Next Primary Action:** Keep this report and release evidence as the durable proof record; rotate the external OpenAI credential if operator policy requires it.
**Supporting Evidence:** Focused package tests, full `go test ./...`, docs/tool sync, secret scan, `make check`, `make install`, clean local/cloud init replays, installed-binary local setup/download, installed-binary local JSON runtime pass, installed-binary cloud JSON credential block, custom OpenAI-compatible endpoint init proof, installed-binary OpenAI route/seed pass, real OpenAI `models evaluate` pass with token usage, real OpenAI `run orchestrator` pass with `llm_calls=1`, raw-key rejection, provider request-capture tests, and generated target doctrine checks passed.

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
/Users/elliottgreaves/go/bin/mars init --repo /private/tmp/mars-custom-endpoint-proof-oMVrjM --model-routing cloud --cloud-provider openai-compatible --cloud-model openai/gpt-4.1-mini --cloud-endpoint https://models.example.test/inference/v1 --api-key-env GITHUB_TOKEN --yes --json
fish -lc 'mars init --repo /private/tmp/mars-confidence10-openai-vltJMo --model-routing cloud --cloud-provider openai --cloud-model gpt-4.1-mini --api-key-env OPENAI_API_KEY --yes --json'
fish -lc 'mars models credentials write-local-env --repo /private/tmp/mars-confidence10-openai-vltJMo --api-key-env OPENAI_API_KEY --yes --json'
fish -lc 'mars start --repo /private/tmp/mars-confidence10-openai-vltJMo --exit-after-seed --yes --json --db /private/tmp/mars-confidence10-openai-vltJMo.mars.db --log-file /private/tmp/mars-confidence10-openai-vltJMo.start.log'
fish -lc 'mars models evaluate --repo /private/tmp/mars-openai-ephemeral-UZMsOV --provider openai --endpoint https://api.openai.com/v1 --model gpt-4.1-mini --api-key-env OPENAI_API_KEY --cloud --json'
fish -lc 'mars run orchestrator --repo /private/tmp/mars-openai-ephemeral-UZMsOV --max-turns 1 --budget 12000 --debug --log-file /private/tmp/mars-openai-ephemeral-UZMsOV.run.log'
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
- Installed binary version: passed with `mars 0.68.3 darwin/arm64`.
- Local setup/download: passed with `{"status":"ok","steps_run":5,"steps_skipped":2}` after downloading `Qwen3-Coder-30B-A3B-Instruct-Q4_K_M.gguf` and `google_gemma-4-E4B-it-Q5_K_M.gguf`.
- Custom OpenAI-compatible init endpoint: passed with JSON-only output and generated `.harness/model-overrides.yaml` storing provider, model, endpoint, and `api_key_env: GITHUB_TOKEN` only.
- OpenAI route/seed proof: passed with JSON-only init, ignored `0600` `.harness/.env.local`, `.harness/.env.example` containing only `OPENAI_API_KEY=`, cloud route using `provider: openai`, `model: gpt-4.1-mini`, and installed-binary `mars start --exit-after-seed --yes --json` returning `status: ok`. This proved route and credential plumbing, not provider usage.
- OpenAI provider-call proof: passed with `mars models evaluate` against a synthetic ephemeral ticket, producing 3/3 passing benchmark cases and token usage from OpenAI responses.
- OpenAI agent-runtime proof: passed as a provider-use proof with one `mars run orchestrator` turn against the synthetic ephemeral target; the command intentionally stopped at `max_turns` but reported `llm_calls=1` and no source repo content was sent.

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

## Anthropic Cloud Target Blocker Replay

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

**Result:** Historical `primary_blocked` replay. Runtime blocked before provider use because `ANTHROPIC_API_KEY` was not set. No raw key value was accepted or printed.

Historical credential presence probe, by variable name only, found no usable cloud provider credential:

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

## Clean OpenAI Cloud Target Replay

**Target repo:** `/private/tmp/mars-confidence10-openai-vltJMo`

```bash
fish -lc 'mars init \
  --repo /private/tmp/mars-confidence10-openai-vltJMo \
  --model-routing cloud \
  --cloud-provider openai \
  --cloud-model gpt-4.1-mini \
  --api-key-env OPENAI_API_KEY \
  --yes \
  --json'
```

**Result:** Passed. Output was JSON only and the generated harness baseline was committed:

```json
{
  "committed": true,
  "model_routing_path": "/private/tmp/mars-confidence10-openai-vltJMo/.harness/model-overrides.yaml",
  "repo": "/private/tmp/mars-confidence10-openai-vltJMo",
  "status": "ok"
}
```

The generated model routing stored provider metadata and the credential environment variable name only:

```yaml
version: 2
default:
    routing: cloud
    provider: openai
    model: gpt-4.1-mini
    endpoint: https://api.openai.com/v1
    api_key_env: OPENAI_API_KEY
    reason: configured by mars init
```

```bash
fish -lc 'mars models credentials write-local-env \
  --repo /private/tmp/mars-confidence10-openai-vltJMo \
  --api-key-env OPENAI_API_KEY \
  --yes \
  --json'
```

**Result:** Passed. `.harness/.env.local` was written with mode `0600`, `.gitignore` ignored `.harness/.env.local`, and `.harness/.env.example` contained only `OPENAI_API_KEY=`. No raw key value was printed.

```json
{
  "example": "/private/tmp/mars-confidence10-openai-vltJMo/.harness/.env.example",
  "local_env": "/private/tmp/mars-confidence10-openai-vltJMo/.harness/.env.local",
  "status": "ok"
}
```

```bash
fish -lc 'mars start \
  --repo /private/tmp/mars-confidence10-openai-vltJMo \
  --exit-after-seed \
  --yes \
  --json \
  --db /private/tmp/mars-confidence10-openai-vltJMo.mars.db \
  --log-file /private/tmp/mars-confidence10-openai-vltJMo.start.log'
```

**Result:** Passed. The installed binary accepted the cloud route and credential-env path and returned JSON-only success:

```json
{
  "repo": "/private/tmp/mars-confidence10-openai-vltJMo",
  "repo_id": "86c745d0-dd79-4d22-851b-689a436ef898",
  "seeded": true,
  "status": "ok"
}
```

This command exits after seeding and validates cloud route/credential plumbing. It does not by itself prove the provider accepted a request.

## Real OpenAI Provider Usage Proof

**Target repo:** `/private/tmp/mars-openai-ephemeral-UZMsOV`

This target contains generated harness content and one synthetic ticket:

```text
docs/tickets/backlog/MH-EPHEMERAL-001.md
```

The synthetic ticket avoids sending source-repo ticket text to OpenAI while still exercising the repo-backed benchmark case.

```bash
fish -lc 'mars models evaluate \
  --repo /private/tmp/mars-openai-ephemeral-UZMsOV \
  --provider openai \
  --endpoint https://api.openai.com/v1 \
  --model gpt-4.1-mini \
  --api-key-env OPENAI_API_KEY \
  --cloud \
  --json'
```

**Result:** Passed as a real provider-call proof. The command calls `llm.Client.ChatCompletion` for benchmark cases and OpenAI returned token usage:

```json
{
  "model": "gpt-4.1-mini",
  "provider": "openai",
  "endpoint": "https://api.openai.com/v1",
  "repo_root": "/private/tmp/mars-openai-ephemeral-UZMsOV",
  "cases": [
    {
      "name": "tool-call-json",
      "passed": true,
      "prompt_tokens": 73,
      "completion_tokens": 21,
      "total_tokens": 94
    },
    {
      "name": "structured-triage-json",
      "passed": true,
      "prompt_tokens": 61,
      "completion_tokens": 25,
      "total_tokens": 86
    },
    {
      "name": "repo-ticket-completion-json",
      "passed": true,
      "prompt_tokens": 176,
      "completion_tokens": 39,
      "total_tokens": 215
    }
  ],
  "summary": {
    "passed": 3,
    "failed": 0,
    "total": 3
  }
}
```

The persisted evaluation report was written under the ephemeral target:

```text
/private/tmp/mars-openai-ephemeral-UZMsOV/docs/generated/model-evaluations/20260628T184347Z-openai-gpt-4.1-mini.json
```

The actual agent runtime was also exercised against the same synthetic target:

```bash
fish -lc 'mars run orchestrator \
  --repo /private/tmp/mars-openai-ephemeral-UZMsOV \
  --max-turns 1 \
  --budget 12000 \
  --debug \
  --log-file /private/tmp/mars-openai-ephemeral-UZMsOV.run.log'
```

**Result:** Provider-use proof passed. The command returned non-zero because the run was deliberately capped at one turn, but it proved the `mars run` agent loop used the OpenAI route:

```text
model override selected role="orchestrator" provider="openai" model="gpt-4.1-mini" endpoint="https://api.openai.com/v1"
agent loop finished ... end_reason="max_turns" llm_calls="1" tool_invocations="2"
```

## Custom Endpoint Cloud Route Proof

**Target repo:** `/private/tmp/mars-custom-endpoint-proof-oMVrjM`

```bash
/Users/elliottgreaves/go/bin/mars init \
  --repo /private/tmp/mars-custom-endpoint-proof-oMVrjM \
  --model-routing cloud \
  --cloud-provider openai-compatible \
  --cloud-model openai/gpt-4.1-mini \
  --cloud-endpoint https://models.example.test/inference/v1 \
  --api-key-env GITHUB_TOKEN \
  --yes \
  --json
```

**Result:** Passed. The generated route committed a custom endpoint and the credential environment variable name only:

```yaml
provider: openai-compatible
model: openai/gpt-4.1-mini
endpoint: https://models.example.test/inference/v1
api_key_env: GITHUB_TOKEN
```

This is supporting evidence for custom cloud/frontier routing. The primary cloud proof above uses OpenAI because an `OPENAI_API_KEY` was available to the fish environment without printing or committing the raw value.

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

## Rerun Commands

Local runtime proof:

```bash
mars start --repo /private/tmp/mars-confidence10-local-2ah3cK --exit-after-seed --yes --json --db /private/tmp/mars-confidence10-local-2ah3cK.mars.db --log-file /private/tmp/mars-confidence10-local-2ah3cK.start.log
```

OpenAI route/seed proof:

```bash
fish -lc 'mars models credentials write-local-env --repo /private/tmp/mars-confidence10-openai-vltJMo --api-key-env OPENAI_API_KEY --yes --json'
fish -lc 'mars start --repo /private/tmp/mars-confidence10-openai-vltJMo --exit-after-seed --yes --json --db /private/tmp/mars-confidence10-openai-vltJMo.mars.db --log-file /private/tmp/mars-confidence10-openai-vltJMo.start.log'
```

OpenAI real provider-use proof:

```bash
fish -lc 'mars models evaluate --repo /private/tmp/mars-openai-ephemeral-UZMsOV --provider openai --endpoint https://api.openai.com/v1 --model gpt-4.1-mini --api-key-env OPENAI_API_KEY --cloud --json'
fish -lc 'mars run orchestrator --repo /private/tmp/mars-openai-ephemeral-UZMsOV --max-turns 1 --budget 12000 --debug --log-file /private/tmp/mars-openai-ephemeral-UZMsOV.run.log'
```

Custom OpenAI-compatible endpoint proof, if GitHub Models or another OpenAI-compatible endpoint is explicitly approved:

```bash
export GITHUB_TOKEN=<token-with-model-scope>
mars models credentials write-local-env --repo /private/tmp/mars-custom-endpoint-proof-oMVrjM --api-key-env GITHUB_TOKEN --yes --json
mars start --repo /private/tmp/mars-custom-endpoint-proof-oMVrjM --exit-after-seed --yes --json --db /private/tmp/mars-custom-endpoint-proof-oMVrjM.mars.db --log-file /private/tmp/mars-custom-endpoint-proof-oMVrjM.start.log
```

## Classification

- Hardware gating, local bundle selection, setup/init routing, runtime preflight, provider routing, JSON output, generated doctrine, and secret handling: foundation-owned, implemented and covered by supporting evidence.
- Missing local weights: resolved by installed-binary `mars setup --inference local --local-bundle auto --download --yes --json`.
- Missing cloud provider credential: resolved for the OpenAI proof by fish universal `OPENAI_API_KEY`; the raw value was not printed, committed, logged, or included in this report.
- GitHub Models token use: no longer required for the primary proof; it still requires explicit approval and a token with the required model scope if used as an additional custom OpenAI-compatible proof.
- Correction to earlier evidence: `mars start --exit-after-seed` only proved cloud route/credential plumbing; real provider usage was not proven until `mars models evaluate` returned OpenAI token usage and `mars run orchestrator` reported `llm_calls=1`.
- Final 10/10 confidence claim: earned for the requested proof plan because every required gate has either passed or has durable supporting/blocker evidence, and the last primary blocker, real cloud provider usage through the MARS CLI and agent runtime, passed with a real OpenAI credential path.
