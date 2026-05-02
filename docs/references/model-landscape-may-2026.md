# Open Model Landscape — May 2026 Refresh

**Date:** 2026-05-02
**Status:** Current shortlist, not a default-registry decision
**Sources:** [Ollama newest library](https://ollama.com/library?sort=newest), [Qwen3.6](https://ollama.com/library/qwen3.6), [Qwen3-Coder-Next](https://ollama.com/library/qwen3-coder-next), [Laguna XS.2](https://ollama.com/library/laguna-xs.2), [GLM-5.1](https://ollama.com/library/glm-5.1), [Mistral Medium 3.5](https://ollama.com/library/mistral-medium-3.5)

## Why This Exists

The April model snapshot is already stale. Mars Harness currently pins safe defaults, but "best model" is a moving target. The harness should not swap defaults because a model is new or popular; it should evaluate candidates against harness-specific agent work, then promote immutable GGUF revisions with checksums.

This refresh creates the May shortlist and the promotion gate. It does not change default model registry entries.

## Current Harness Defaults

The source registry currently maps coding and reasoning tiers to Qwen3-Coder 30B-A3B variants and fast tier to Gemma 4 E4B variants, selected by hardware profile and performance profile.

Strengths:

- Pinned HuggingFace revisions and SHA256 checksums.
- Known llama.cpp path.
- Hardware-aware `auto`, `quality`, `balanced`, and `speed` profile behavior.

Weaknesses:

- Registry quality is a point-in-time snapshot.
- No repeatable command yet proves a new model is better on this harness's roles.
- Fast tier is probably under-evaluated against newer small tool-capable models.

## May 2026 Candidate Shortlist

| Candidate | Local default candidate? | Why it matters | Main risk |
| --- | --- | --- | --- |
| Qwen3.6 35B-A3B Coding | Yes, quality/balanced candidate | Recent Qwen release with 256K context and explicit agentic coding/thinking-preservation focus. Ollama lists 35B-A3B coding variants around 22GB to 70GB depending quant/runtime. | Needs GGUF/llama.cpp validation, tool-call reliability data, and apples-to-apples comparison with current Qwen3-Coder default. |
| Qwen3.6 27B | Yes, balanced/speed candidate | Smaller Qwen3.6 option listed around 17GB for default variant, likely more practical on Apple Silicon. | May trail 35B coding quality; must beat current defaults on useful work, not just fit memory. |
| Laguna XS.2 | Yes, experimental local candidate | 33B total/3B active model positioned for local long-horizon coding, with 128K context and local-ready memory claims. | Very new; needs harness-specific stability and tool-use evidence. |
| GLM-5.1 | Optional remote/cloud candidate | Strong self-reported agentic engineering, SWE-Bench Pro, NL2Repo, and Terminal-Bench 2.0 results. | Ollama listing is cloud-only; not suitable as a default local/private model. |
| Mistral Medium 3.5 | Optional upper-bound candidate | Recent 128B reasoning/coding model with 256K context. | 80GB Ollama model size makes it unsuitable for normal local defaults. |

## Evaluation Standard

A candidate can only replace a default when it passes all of these:

- It runs through the harness's OpenAI-compatible path or llama.cpp path without custom one-off glue.
- It passes tool-call JSON reliability tests for mutating roles.
- It passes strict structured-output tests for triage and scoring roles.
- It completes representative harness tickets at least as often as the current default.
- It has acceptable tokens/sec and memory use on the target hardware profile.
- It has an immutable source revision and SHA256 before becoming a default.
- It does not require cloud access unless marked as an optional remote profile.

## Ollama Access Policy

Mars Harness should not require a hardcoded shortlist before an operator can try a model. Ollama is the broad catalog and local-provider path: any installed or published Ollama model can be evaluated or selected as an explicit override when the operator accepts that responsibility.

That broad access does not imply default promotion. A model may be:

- evaluated ad hoc by name
- used as an explicit repo, role, or tier override
- promoted to a zero-config default only after benchmark evidence, hardware-fit review, immutable source revision, and SHA256 are recorded

## Benchmark Pack

Initial mechanical benchmark cases now live behind:

```bash
mars-harness models evaluate
mars-harness models evaluate --endpoint <openai-compatible-url> --model <name>
mars-harness models evaluate --endpoint <openai-compatible-url> --model <name> --json
```

The first version checks:

- `tool-call-json`: the model must call a simple repo file inspection tool with parseable JSON arguments.
- `structured-triage-json`: the model must return strict JSON for a harness failure classification.

The next version should add repo-backed tasks:

- brownfield bug diagnosis with expected file references
- small code edit with tests
- ticket-drain decision with multiple in-progress tickets
- guardrail refusal/blocked-tool behavior
- context-glossary retrieval behavior

## Recommendation

Do not change defaults yet.

Run Qwen3.6 35B-A3B, Qwen3.6 27B, Laguna XS.2, and the current Qwen3-Coder default through the same benchmark pack on the primary Apple Silicon machine. Promote only after the results show either higher pass rate at similar speed or similar pass rate with materially better speed/memory behavior.
