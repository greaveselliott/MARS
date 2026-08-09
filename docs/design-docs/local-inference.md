# Local Inference

**Status:** Draft  
**Date:** 2026-04-11  
**Author:** MARS contributors

How the harness serves large language models locally: process boundaries, weight storage, verification, and operational lifecycle (download, health, upgrades).

## Context

MARS targets **plug-and-play** local inference without requiring users to compile C/C++ or manage fragile native bindings in the main binary. Model artifacts are large and must not bloat the repository; integrity and reproducibility still matter for support and evolution.

The agent runtime ([agent-runtime.md](agent-runtime.md)) assumes a **stable HTTP or stdin/stdout contract** to the inference server; this document owns how that server is provisioned and supervised.

## Key Design Decisions

### AD-007: llama.cpp as subprocess, not embedded via CGO

The inference server runs **llama.cpp in a separate process** managed by the harness. The main Go binary stays **CGO-free**, simplifying distribution and cross-compilation. Trade-off: **two processes** to supervise, IPC, and coordinated shutdown—acceptable for clearer packaging and fewer toolchain failures on end-user machines.

The harness is responsible for **argv, env, working directory**, and capturing stderr for diagnostics when jobs fail for infra reasons.

The setup package keeps one typed provenance record for the supported
`llama.cpp` release: immutable source commit, the root license distributed in
the release archives, the source tree's separate `jsonhpp` notice, and exact
archive name, size, and SHA256 per platform. Only entries explicitly marked
enabled may be installed. The Linux arm64/amd64 artifacts are recorded for the
public bootstrap work but remain unavailable until the safe Linux installation
path is delivered; their presence in the record is not an installation claim.

### AD-008: Model weights outside the repo

Weights live under **`~/.mars/models/`** (not committed). Expected hashes are recorded in **`bundle.lock.json`** (e.g. SHA256 per artifact) so installs and upgrades can verify downloads and detect drift.

Corrupt or partial downloads must never be loaded silently; verification runs **before** binding a model to active traffic.

### AD-031: Inference resilience — timeouts, context headroom, and health verification

Three failure modes observed in production pipeline runs (crowd-runner, April 2026):

**1. HTTP client timeout (60s → 5 min).** A 30B Q8 model on Apple Silicon routinely exceeds 60 seconds per completion on complex multi-turn prompts. Local inference has zero per-second cost, so generous timeouts are correct. Default changed from 60s to 5 minutes. Retry count increased from 3 to 5 with backoff ceiling raised from 5s to 15s. Agent loop `chatWithRetries` backoff also increased from 100ms–2s to 2s–15s.

**2. Fast-tier context headroom (8192 → 16384).** Gemma 4 E4B was configured with 8192 context across all hardware profiles. Assembled role prompts with ticket indices exceeded this (8442 tokens observed). Gemma 4 natively supports 128k. Increased all fast-tier profiles to 16384 — sufficient for any role prompt while keeping memory modest. Also added explicit non-retryable error detection for context-exceeded responses (HTTP 400 with "exceed" + "context" in body) so the client doesn't waste retries on prompts that will never fit.

**3. Stale health state (connection refused).** `ServerForRole` returned immediately when a server was previously marked `StateHealthy` without verifying it was still alive. If the server crashed between jobs, the next job got `connection refused`. Added an active `/health` spot-check after `Start` returns. If the server fails the check, it's torn down and restarted before the endpoint is returned. This closes the race window between the supervisor detecting a crash and the next job claiming the "healthy" server.

### AD-032: Zero-config local inference performance profile and llama tuning

Apple Silicon and other unified-memory machines can show low CPU usage while local generation is slow because llama.cpp is doing most work through Metal/GPU kernels and memory bandwidth. The harness must not imply that idle CPU means unused inference capacity.

The default profile is `auto`. Setup/start/serve detect the hardware and choose the model set without operator tuning:

- Apple Silicon / Metal unified-memory machines below 96 GiB RAM use `balanced` automatically, capping high/multi hardware at the medium Q4/Q5 model set.
- Large dedicated GPUs keep `quality` when their VRAM can absorb the Q8 model without squeezing the host.
- Lower hardware profiles already use conservative model sets, so `auto` preserves their detected profile.

The user config remains an escape hatch, not the normal path:

- `performance_profile`: `auto`, `quality`, `balanced`, or `speed`. `quality` uses the detected hardware profile as-is; `balanced` caps high/multi hardware at the medium model set; `speed` uses the low model set when a GPU is present.
- llama-server flags: `llama_parallel`, `llama_threads`, `llama_threads_batch`, `llama_batch_size`, `llama_ubatch_size`, `llama_flash_attention`, and `llama_mlock`.

Default `llama_parallel` is `1` because the strict-trunk default pipeline is one active agent per repo. llama.cpp's auto parallelism can reserve multiple slots and extra KV/cache memory for throughput the default workflow does not use. Operators can set `llama_parallel: 0` to restore llama.cpp auto behavior. When the harness deliberately starts a server with multiple parallel slots, it scales the server `--ctx-size` by the slot count so each request keeps the tier's documented served context window instead of silently receiving a divided window.

Changing the effective `performance_profile` may require additional model files. `mars setup` now verifies the model files required by the active profile before accepting the download marker as complete.

### AD-063: Model Defaults Change Only After Harness-Specific Evaluation

The open-model landscape moves too quickly for hardcoded defaults to remain "best" by assumption. New releases such as Qwen3.6, Laguna XS.2, GLM-5.1, and Mistral Medium 3.5 are candidates, not automatic replacements.

MARS must treat model selection as an evidence loop:

- maintain a current model landscape reference
- expose a `mars models evaluate` command for mechanical benchmark runs
- compare candidates against the current pinned defaults on harness-relevant tasks
- measure tool-call JSON reliability, structured-output reliability, latency, token throughput, memory fit, and ticket-completion behavior
- promote only immutable model artifacts with pinned revisions and SHA256 checksums

The reason is safety and repeatability: a model-card claim or newest-library ranking is useful discovery input, but default registry entries affect autonomous mutating agents. Default changes need local evidence and reproducible artifacts.

### AD-064: Manifest Model Tier Is The Inference Routing Source Of Truth

Target manifests already declare each role's intended model tier with `model: fast`, `model: reasoning`, or `model: coding`. The inference router must honor that manifest tier before falling back to role-name defaults.

This matters because executive/planning roles such as `ceo`, `coo`, and `cto-weekly` should not accidentally route to the heavy coding model just because their role name was omitted from a hardcoded router map. On constrained or newly configured machines, that causes bootstrap jobs to fail with "no local model for tier coding" even though the manifest asked for a reasoning-tier role.

The router normally resolves tiers in this order:

1. manifest `role.model` when it is one of `fast`, `reasoning`, or `coding`
2. built-in role fallback mapping for older/custom callers
3. coding tier for truly unknown roles, preserving the conservative quality default

Validation runners can deliberately force a single server tier for an entire
batch. `validation agent-smoke --single-server` uses that path so role-local
parallel smoke cases share one llama-server process while still preserving
per-case repo, DB, trace, and log isolation. Forced-tier routing is an explicit
validation topology override; ordinary autonomous execution keeps manifest-tier
routing as the source of truth.

Missing local-model errors also name the expected model file path and suggest `mars setup` or remote fallback configuration. Telemetry classifies these as `model_unavailable` instead of `unknown`, routing repeated failures to inference/setup work. The reason is operator recovery: inference failures should tell the user which tier/file is missing and how to repair it.

### AD-299: Lifecycle Start Uses Real Endpoint Overrides, Safe Scoped Cleanup, And Port Reservations

Full lifecycle validation often runs multiple `mars start` processes in
parallel. Those processes must not fight over the same control-plane port,
dashboard port, or llama-server tier port, and a scoped lifecycle start must not
kill live processes owned by another run. `start --model-endpoint <url>` is the
operator escape hatch for a real OpenAI-compatible endpoint: when present, the
server uses that endpoint, skips local model-file preflight, and does not start
local llama-server processes. Fake, stub, mock, canned, or scripted endpoints
are still deterministic test fixtures only and do not raise confidence for live
behavior claims.

Scoped `mars start` performs SQLite sidecar recovery only. It does not
kill processes on the configured webhook/dashboard ports and does not globally
kill `llama-server`. When the default scoped control-plane or dashboard address
is already occupied, the server falls back to an ephemeral local address and logs
the requested and actual listener. Operators that need deterministic ports can
pass `--addr` and `--dashboard-addr`; explicit addresses fail on conflict instead
of being silently remapped.

When local inference is used, the router reserves tier ports before launching a
server. A live locked port with a healthy `/health` endpoint may be reused;
otherwise the router allocates the next bounded port in that tier range. If no
allowed port is safe, startup returns and telemetry classifies the terminal
class `inference_port_conflict` with the tier, role, port, owning PID when
known, and the remediation to stop the owner or rerun with `--model-endpoint`.
Fresh lock files with incomplete metadata are treated as active startup locks
for a short grace window rather than being deleted as stale. This prevents two
parallel `start` processes from racing between exclusive lock creation and
metadata write completion, which otherwise can launch duplicate llama-server
processes on the same port.

### AD-066: Ollama Is A Catalog And Swap Provider, Not Automatic Default Promotion

MARS should make it easy to evaluate and explicitly run any model available through Ollama. The model registry should not be the only way to try a model. Operators should be able to list local Ollama models, reference published Ollama model names as evaluation candidates, and swap a tier or role to an Ollama model without editing several files by hand.

This is a provider/candidate path, not a shortcut around default safety. There are three distinct states:

- **Ad-hoc candidate:** a model name supplied to evaluation or a one-off run.
- **Explicit override:** a repo, role, or tier deliberately configured to use an Ollama model.
- **Default registry entry:** a harness-managed local default with immutable source revision, SHA256, benchmark evidence, and hardware-fit rationale.

Only the third state is allowed to change zero-config defaults. Ollama access makes exploration broad and simple; benchmark-backed promotion keeps autonomous default behavior reproducible and supportable.

The intended operator experience is:

- discover available local Ollama models
- evaluate any model by name through the standard benchmark command
- switch `fast`, `reasoning`, or `coding` tier for a repo or role with one clear command or manifest edit
- see doctor/setup warnings when an explicit Ollama override is unavailable locally
- keep default promotion blocked until pinned artifacts and benchmark reports exist

Implemented command shape:

```bash
mars models list --provider ollama
mars models evaluate --provider ollama --model qwen3.6:27b
mars models override --repo /path/to/repo --tier coding --provider ollama --model qwen3.6:27b
mars models override --repo /path/to/repo --role engineer --provider openai-compatible --endpoint http://127.0.0.1:8088/v1 --model repo-coder
```

Overrides are stored in `.harness/model-overrides.yaml`. The runtime checks a
role override first, then a tier override, then the existing pinned local
registry route. Ollama overrides default to `http://127.0.0.1:11434/v1`.
Evaluation reports include a promotion section; ad-hoc Ollama candidates and
cloud-only candidates are blocked from default promotion unless pinned artifact
revision, SHA256, benchmark evidence, and docs rationale are present.

### AD-306: Hardware-Gated Model Routing And Secret-Safe Cloud Credentials

Model selection must pass through one shared resolver before setup downloads
weights, init writes routing config, overrides are accepted, or run/start/serve
create an LLM client. The resolver owns local bundle eligibility, `auto`
selection, cloud provider routing, and credential environment validation.
Duplicating bundle checks in individual commands is not allowed because it
lets unsupported local models reach runtime through config or override paths.

Local bundle metadata records the tier-to-GGUF mapping plus required RAM,
dedicated VRAM, unified memory, disk estimate, OS, arch, backend, and profile.
`auto` means the highest-ranked eligible bundle for the detected machine.
Unknown hardware resources disable risky bundles and return a remediation that
offers local-auto, cloud, or deferred setup instead of guessing.

Committed target configuration stores only provider routing metadata and
`api_key_env`. Raw cloud provider keys are never accepted by flags, generated
config, logs, telemetry, traces, JSON output, or errors. `mars models
credentials write-local-env` may read a secret from the named process
environment variable and write it to ignored `.harness/.env.local` with `0600`
permissions; `.harness/.env.example` records only the variable name.

Cloud providers are not considered selectable merely because a name appears in
the registry. Each selectable provider must have official-documentation fixture
evidence and request-capturing tests that prove the request shape, auth header,
and redaction behavior. Providers without that evidence are listed as
unavailable with a reason.

Supported cloud provider routes after the 2026-06-28 validation pass:

| Provider | Default endpoint | Credential env | Support status |
| --- | --- | --- | --- |
| OpenAI | `https://api.openai.com/v1` | `OPENAI_API_KEY` | Supported with live generation and one-turn runtime proof. |
| Anthropic | `https://api.anthropic.com/v1` | `ANTHROPIC_API_KEY` | Supported with live generation and one-turn runtime proof. |
| Gemini | `https://generativelanguage.googleapis.com/v1beta/openai` | `GEMINI_API_KEY` | Supported with live generation and one-turn runtime proof through Google's OpenAI-compatible endpoint. |
| Mistral | `https://api.mistral.ai/v1` | `MISTRAL_API_KEY` | Supported with live generation and one-turn runtime proof. |
| DeepSeek | `https://api.deepseek.com/v1` | `DEEPSEEK_API_KEY` | Supported. |
| xAI | `https://api.x.ai/v1` | `XAI_API_KEY` | Supported. |

TTY setup/init/model commands may use aligned sections, concise tables, muted
color, disabled choices, and progress state. Non-TTY output is plain, `NO_COLOR`
and `TERM=dumb` disable styling, `--plain` forces plain text, and `--json`
emits JSON only. `--yes` never prompts; missing required choices produce a
machine-readable remediation.

### Open topics (M2 and beyond)

- **Hardware detection:** CPU vs GPU paths, memory ceilings, and safe default model bundles; degrade gracefully when VRAM is insufficient.
- **Model registry and provider catalog:** naming, versioning, Ollama/local-provider discovery, compatibility with server flags, deprecation notices in CLI output, simple tier/role swaps, and benchmark-backed promotion.
- **Override diagnostics:** doctor/setup should warn when `.harness/model-overrides.yaml` references an unavailable local Ollama model or an unreachable OpenAI-compatible endpoint.
- **Download with resume:** partial files, checksum retry, bandwidth-friendly defaults; mirror URLs as optional fallback.
- **Server lifecycle:** start/stop, backoff on crash, upgrade without orphan processes; pidfile or equivalent for operator tooling.
- **Multi-model serving:** concurrent endpoints vs serial reuse; resource isolation when two roles need different sizes.

## Discoveries

- **Local inference timeout math:** On Apple M1 Max (64GB), Qwen3-Coder-30B-A3B Q8_0 with 32k context can take 2–4 minutes per completion when generating long multi-tool responses. The 60s default was set assuming cloud API speeds. Any timeout below 3 minutes will produce false-positive timeouts on complex engineer turns.
- **Fast-tier context floor:** Role prompts with ticket indices typically assemble to 5000–9000 tokens. Any context window below 12k risks overflow on mature projects with many tickets. 16k provides comfortable headroom.
- **Health check race window:** The supervisor restart loop (exponential backoff 1s→30s) can leave a 1–30 second gap where `State()` returns `StateHealthy` but the process is dead. Active verification on every `ServerForRole` call is cheap (2s timeout HTTP GET) and closes this gap completely.
- **Apple Silicon performance diagnosis:** Low CPU with high RAM during Qwen Q8 generation is expected when Metal is active. The limiting resource is usually memory bandwidth and model size, not CPU thread count. Reducing quantization/profile size and limiting parallel slots are the first knobs to try.
- **Manifest/router mismatch:** A sample-target bootstrap CEO job failed because `ceo` was missing from the static router map and unknown roles defaulted to coding, even though the generated manifest declared `model: reasoning`. Routing by manifest tier fixes the mismatch and makes bootstrap failures less likely on partial local-model installs.
- **RAM pressure silently degrades inference and confounds pace data (2026-06-12):** During the 2026-06-11 demo-11 baseline, the quality-profile `Qwen3-Coder-30B-A3B-Instruct-Q8_0` weights (32.5 GB per tier server, both resident) maxed a 64 GiB unified-memory machine and drastically degraded inference: cto-weekly wedged at 12.1 minutes versus 205s for the same stage on the balanced model — same harness, target, and prompts. This motivated the operator's swap to `performance_profile: balanced` for all subsequent runs. The convergence/pace data captured under RAM pressure is confounded by the degradation and was reclassified evidence-only (see the AD-285 model-identity amendment in validation-matrix-gating.md). The degradation was caught only by human observation; T-034 proposes the mechanical guard (pace-anomaly telemetry rows and/or a doctor check flagging configured model footprint approaching physical RAM) so degraded inference becomes a visible finding.
- **Cloud provider support classification (2026-06-28):** Provider support is backed by official-doc-backed adapters, request-capture tests, env-var credential paths, and live validation evidence. OpenAI, Anthropic, Gemini, Mistral, DeepSeek, and xAI are documented as supported cloud routes.
