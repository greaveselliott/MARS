# Reviewer Validation Capability Guard Replay

**Date:** 2026-06-26
**Source Ref:** working tree after reviewer-validation capability guard patch
**Validation Type:** Clean target scoped lifecycle replay
**Model:** OpenAI-compatible endpoint via local proxy, `gpt-4.1-mini`

## Primary Outcome Contract

**Primary Outcome:** Prove the COO product-capability guard no longer treats reviewer/manual validation instructions as missing product capabilities.
**Primary Pass Gate:** A clean Space Invaders target containing the phrase "Include enough validation instructions for a reviewer to confirm the game works" reaches a successful COO `job_disposition_record` and dispatches to `cto-weekly` without `guardrail_block` or `guardrail_loop` telemetry for that phrase.
**Primary Status:** `primary_passed`
**Current Primary Blocker:** None for the reviewer-validation false negative. The replay was stopped deliberately after the pass gate; the resulting `llm_unreachable` event is from operator cancellation during the next `cto-weekly` job.
**Next Primary Action:** Use the normal lifecycle replay if validating full product delivery beyond the fixed guardrail.
**Supporting Evidence:** Focused unit regression passed; broad `go test ./...` passed; `make check` passed with race coverage and fuzz smoke; clean target replay reached COO completion and dispatched to `cto-weekly` with zero guardrail-block events.

## Setup

- Source root: `<source-root>`
- Target repo: `<validation-root>/target`
- Database: `<validation-root>/mars.db`
- Logs: `<validation-root>/logs/start.log`
- Proxy: local OpenAI-compatible proxy with API key sourced from operator environment; no secret values recorded.
- Target brief: browser-playable Space Invaders game with keyboard movement, firing, alien movement, score/lives/win/lose/restart, and reviewer validation instructions.

## Command

```bash
OPENAI_VALIDATION_MODEL=gpt-4.1-mini OPENAI_PROXY_PORT=18654 fish -lc 'exec python3 <proxy-script>'

HOME=<validation-root>/home \
TMPDIR=<validation-root>/tmp \
GOCACHE=<validation-root>/go-cache \
<source-root>/build/mars-harness start \
  --repo <validation-root>/target \
  --db <validation-root>/mars.db \
  --log-file <validation-root>/logs/start.log \
  --concurrency 1 \
  --debug \
  --new-lifecycle \
  --model-endpoint http://127.0.0.1:18654/v1 \
  --addr 127.0.0.1:19191 \
  --dashboard-addr 127.0.0.1:19190
```

## Result

- CEO completed and handed off through Orchestrator.
- COO rewrote the active plan and feature contract for the Space Invaders slice.
- COO committed the planning update locally in the target replay.
- COO recorded `job_disposition_record` successfully with `next_need: ticket_breakdown` and `suggested_role: cto-weekly`.
- Server dispatched the next job to `cto-weekly`.
- `guardrail_block` count: 0.
- `guardrail_loop` count: 0.
- OpenAI proxy chat completions succeeded throughout the proof point.

## Stop Reason

The replay was stopped immediately after the fixed pass gate was reached. The active `cto-weekly` job then reported `llm_unreachable` because the operator cancelled the server context. That cancellation is not evidence of an inference outage or product guardrail failure.

## Classification

- False-negative guardrail: foundation-owned, fixed in `internal/tools`.
- Target product implementation: not attempted in this scoped replay.
- OpenAI proxy/key handling: evidence-only; the source fix does not depend on provider-specific behavior.
