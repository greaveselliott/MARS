# Open Model Landscape — April 2026

Research findings informing MARS's model selection, hardware profiles, and quality expectations.

**Date:** 2026-04-11
**Sources:** SWE-bench Verified leaderboard, vLLM GitHub issues, llama.cpp PRs, independent tool-calling benchmarks, Google DeepMind Gemma 4 announcement, Qwen3-Coder-Next technical reports.

## SWE-bench Verified (the benchmark that matters)

SWE-bench Verified is the standard benchmark for evaluating AI agents on real-world software engineering tasks: navigate codebases, diagnose issues, make multi-file patches that pass actual unit tests. 500 human-validated problems from popular open-source Python repositories.

This is the closest proxy for what the Engineer and Pipeline Fixer roles actually do.

### Scores (April 2026)

| Model | SWE-bench Verified | Type | VRAM (Q4) | License |
|---|---|---|---|---|
| Claude Opus 4.6 | 83.7% | Proprietary API | N/A | Commercial |
| Qwen3-Coder-Next | 70.6% | Open source | ~46 GB | Apache 2.0 |
| DeepSeek-V3.2 | 70.2% | Open source | ~48 GB | Permissive |
| Gemma 4 31B | 42.1% | Open source | 18-20 GB | Apache 2.0 |
| Gemma 4 26B A4B | ~38% (est.) | Open source | 15-17 GB | Apache 2.0 |

### Key finding

Best local model (Qwen3-Coder-Next) achieves **84% of Opus 4.6 quality** on real SWE tasks. Gemma 4 31B achieves only 50%. For code-generating roles, Qwen3-Coder-Next is the correct choice; Gemma 4 is adequate for planning and review but not competitive for code generation.

### Misleading benchmarks

LiveCodeBench and tau2-bench scores cited in marketing overstate Gemma 4's capabilities for this use case. They measure competitive coding puzzles and synthetic agentic tasks, not real multi-file software engineering. Do not use them for model selection decisions.

## Gemma 4 Model Family

Released April 2026 by Google DeepMind. Built from Gemini 3 research.

| Variant | Parameters | Active params | VRAM (Q4) | Context | Strengths |
|---|---|---|---|---|---|
| 31B IT Thinking | 31B (dense) | 31B | 18-20 GB | 256K | Reasoning, planning, review. 86.4% tau2-bench (agentic) |
| 26B A4B IT Thinking | 26B (MoE) | 3.8B | 15-17 GB | 256K | 2-2.5x faster than 31B. Adequate for review tasks |
| E4B | 4B | 4B | ~3 GB | 256K | Edge devices, CPU inference. Too small for judgment tasks |
| E2B | 2B | 2B | ~1.5 GB | 256K | IoT, Raspberry Pi |

Native function calling across all sizes. Apache 2.0 licensed.

## Qwen3-Coder-Next

Released February 2026 by Alibaba. 80B total / 3B active parameters (MoE). Trained on 800,000 real GitHub PRs with test execution feedback. Hybrid attention architecture handles 256K context natively. Apache 2.0.

VRAM: ~46 GB at Q4_K_XL. Runs on a single RTX A6000 (48 GB) or dual RTX 4090s.

## Tool Calling Maturity

### vLLM + Gemma 4 (NOT production-ready, April 2026)

Open bugs on vLLM's Gemma 4 tool parser:

- **#39089** (OPEN): Boolean value corruption during streaming mode
- **#39069** (OPEN): String truncation when arguments contain internal quotes — critical for coding agents where tool arguments are code snippets
- **#38946** (fixed): Invalid JSON during streaming
- **#38837** (fixed): Constructor crash (`Gemma4ToolParser.__init__()` missing `tools` parameter)

### vLLM + Qwen3-Coder-Next

Different parser path. Potentially more stable. Needs testing.

### llama.cpp + Gemma 4

Specialized parser merged in PR #21418 with interleaved thinking support. Newer than vLLM's implementation, less validated at scale, but designed specifically for Gemma 4's tool format.

### Ollama (NOT viable for production)

Three documented issues with Gemma 4:
- Thinking mode parameter (`think: false`) ignored
- Multi-turn tool calls fail when streaming is enabled
- Parser corrupts quoted arguments

Do not use Ollama for production tool calling.

### Independent tool calling benchmark (13 local models)

- Qwen3.5 4B: 97.5% pass rate (format compliance)
- Nemotron Nano 4B: 95.0%
- Qwen3 8B: 85.0%

Important: format compliance (does the JSON parse correctly?) is different from decision quality (did the model choose the right tool with the right arguments?). No benchmark reliably measures the latter.

## Hardware Profiles

| Profile | GPU VRAM | Primary model | Secondary model | Use case |
|---|---|---|---|---|
| `cpu` | None | Gemma 4 E4B (CPU) | — | Review only; code gen needs cloud fallback |
| `light` | 8-16 GB | Gemma 4 26B A4B Q3 | — | All roles, slower code gen |
| `standard` | 24 GB | Gemma 4 31B Q4 | — | All roles, adequate quality |
| `full` | 48+ GB | Qwen3-Coder-Next Q4 | Gemma 4 26B A4B | Best local quality, concurrent |
| `multi` | 2x 24+ GB | Qwen3-Coder-Next (GPU 0) | Gemma 4 31B (GPU 1) | Full quality, full concurrency |

## Economics

### Self-hosted

- GPU server hardware: 2x RTX 4090 + chassis = ~$5,000-7,000 one-time
- Amortised 24 months: ~$250-300/month
- Electricity: ~$45-60/month at sustained load
- Total: ~$300-360/month

### vs Cloud API

- Opus 4.6: $15 input / $75 output per MTok
- Sonnet 4.6: $3 input / $15 output per MTok
- Plus Cursor token fees on Teams/Enterprise
- Estimated monthly: $500-2,000+

### Break-even: 4-8 months on-premise, then marginal inference cost is zero.

Cloud GPU instances (Lambda, RunPod) narrow or eliminate the cost advantage. Only viable for burst capacity, not steady-state.

## Serving Infrastructure Recommendation

**Primary:** llama.cpp managed as subprocess by the harness binary. Downloaded during `mars setup`. User never interacts with it directly.

**Alternative:** If user already has vLLM running, the manifest can point to it. But vLLM is not a requirement.

**Why not vLLM as default:** Requires Python, CUDA toolkit, pip packages — a day of debugging for most people. llama.cpp is a single binary with no runtime dependencies beyond GPU drivers.

## Model Landscape Trajectory

The landscape is moving fast. Key signals to watch:

- SWE-bench Verified scores for open models crossing 75% (enables local Engineer role at near-parity)
- vLLM Gemma 4 tool calling bugs closing (#39089, #39069)
- New open models from Qwen, DeepSeek, or others exceeding Qwen3-Coder-Next

The bundle manifest's per-role model assignment means swapping models is a config change, not a code change. Design for this.
