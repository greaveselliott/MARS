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
<source-root>/build/mars start \
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

## Continuation Replay

**Continuation Command:** resumed the same validation target without
`--new-lifecycle`, using the same local OpenAI-compatible proxy and isolated
`HOME`, `TMPDIR`, `GOCACHE`, DB, log, API, and dashboard paths under
`<validation-root>`.

**Observed Progress:**

- Startup resumed the lifecycle at `cto-weekly` from the prior COO handoff.
- CTO created a first-slice ticket for project-brief visibility.
- Engineer claimed the ticket, added `package.json`, and completed it after
  recovering from an invalid JavaScript-target `go test ./...` review command.
- QA and Security approved the no-op `npm run build` / `npm run test`
  evidence.
- Dogfood detected the product gap: no `index.html` and no runnable browser
  application entrypoint.
- The next CTO pass tried to create the real playable Space Invaders ticket,
  but policy blocked both ticket creation and implementation handoff.

**Continuation Status:** `primary_passed_supporting_blocked`

The original reviewer-validation false negative remains fixed. The full build
continuation found a separate foundation-owned first-slice ticketing deadlock:
a weak done ticket could claim `F-001-S001` without build/smoke proof, leaving
CTO unable to create the real executable first-slice ticket and unable to hand
off to Engineer.

**Source Follow-Up:**

- `ticket_create` now rejects verification-only first-slice tickets before
  first proof.
- If weak done evidence already claimed the first-slice scenario, CTO may
  create a new exact first-slice implementation ticket when no Engineer-ready
  first-slice ticket is waiting.
- Focused regression command:
  `GOCACHE=<validation-root>/go-cache go test ./internal/tools -run 'TestCTOTicketCreate|TestCTOCompletion'`
  passed.

## Fresh Build Replay After Follow-Up Fixes

**Fresh Replay Source Ref:** working tree after the first-slice recovery patch
and the OpenAI-compatible tool-call batch patch.

**Fresh Replay Command:** new isolated validation target under
`<validation-root>`, isolated `HOME`, `TMPDIR`, `GOCACHE`, DB, API address, and
dashboard address, using the same local OpenAI-compatible proxy and
`gpt-4.1-mini`.

```bash
HOME=<validation-root>/home \
TMPDIR=<validation-root>/tmp \
GOCACHE=<validation-root>/go-cache \
<source-root>/build/mars start \
  --repo <validation-root>/target \
  --db <validation-root>/mars.db \
  --log-file <validation-root>/logs/start.log \
  --concurrency 1 \
  --debug \
  --new-lifecycle \
  --model-endpoint http://127.0.0.1:<proxy-port>/v1 \
  --addr 127.0.0.1:<api-port> \
  --dashboard-addr 127.0.0.1:<dashboard-port>
```

**Additional Foundation-Owned Failure Found:** The fresh replay progressed past
the weak-ticket blocker, then COO emitted a multi-tool OpenAI-compatible
assistant message. The runtime executed one model tool call and injected a
synthetic `code_index` assistant/tool pair before returning tool responses for
the rest of the original model tool calls, which caused the provider to reject
the next request. This was a foundation-owned agent-runtime protocol bug.

**Additional Source Follow-Up:**

- The agent loop now completes all model-provided tool responses in a batch
  before appending synthetic code-graph refreshes or review terminal-evidence
  reminders.
- If a terminal tool stops a job mid-batch, the remaining model-provided tool
  calls receive skipped tool responses so the transcript remains valid.
- Focused regression command:
  `GOCACHE=<validation-root>/go-cache go test ./internal/agent -run 'TestRun_refreshesCodeGraphAfterMutation|TestRunRefreshesCodeGraphAfterCompletingModelToolCallBatch|TestRun_threeToolCallsHappyPath'`
  passed.

**Fresh Replay Result:** `primary_passed_build_validation`

- CEO completed the target strategy handoff.
- COO completed the plan and feature-contract update without triggering the
  original reviewer-validation false negative or the OpenAI tool-call
  adjacency failure.
- CTO created one executable first-slice implementation ticket for the earliest
  product scenario: player ship left/right keyboard movement.
- Engineer implemented the slice with browser files, source modules,
  JavaScript tests, and package scripts, then committed the work and moved the
  ticket to done with evidence.
- QA approved after `npm run test`, `npm run build`, DocSync audit, a local
  static server, and an HTTP smoke probe of the rendered `index.html`.
- Security approved after secret scanning, `npm run test`, `npm run build`, a
  local static server, and an HTTP smoke probe of the rendered `index.html`.
- Dogfood approved after build/test/smoke evidence and committed the generated
  lockfile produced by dependency sync.
- Target repo ended clean on `main`.

**Stop Reason:** The server was stopped deliberately after the first-slice
build, QA, Security, and Dogfood pass gates completed. A later redundant
Dogfood continuation had begun broad discovery and hit a BSD-incompatible
`ls --ignore` validation command before operator stop; the final
context-cancelled event is stop evidence, not a failure of the first-slice
build validation.

**Validation Classification:**

- Reviewer-validation false negative: foundation-owned, fixed.
- Weak first-slice ticket recovery: foundation-owned, fixed.
- OpenAI-compatible tool-call batch adjacency: foundation-owned, fixed.
- Generated target product work: validation evidence only; no product-specific
  Space Invaders implementation was copied into foundation defaults.
