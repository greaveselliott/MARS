# demo-123 Live Lifecycle Validation — 2026-05-19

## Scope

- Target: `<validation-root>`
- Brief: browser-playable Space Invaders style game with keyboard movement,
  shooting, enemy waves, scoring, lives, and a clear win/lose loop.
- Harness binary: `<validation-root>`, built from the working
  tree after the completed same-role dispatch fix.
- DB: `<validation-root>`
- Log: `<validation-root>`
- Ports: webhook `19095`, dashboard `19094`
- Model stack: local Qwen3-Coder GGUF through managed `llama-server`.

## Baseline Failure

A preceding clean run against `<validation-root>`
proved startup, inference, CEO, and COO were healthy but stopped after COO:

- CEO completed in 6 turns and handed off to COO with `next_need: exec_plan`.
- COO completed in 11 turns with `next_need: feature_contract`.
- Dispatch recorded `same-role next_need has no forward owner`.
- No product ticket was created and no implementation role ran.

The source fix changes completed same-role planning handoffs to route to the
role's default forward owner while preserving the no-work same-role stop.

## Replay Results

The fresh replay reached real product delivery before hitting the next
foundation-level issue:

| Stage | Result | Evidence |
| --- | --- | --- |
| Init/register/start | Passed | Generated harness baseline committed; one CEO bootstrap job seeded. |
| CEO | Passed | Completed in 7 turns, recorded product-slice decision, routed to COO. |
| COO | Passed | Updated active plan and BDD feature contract for Space Invaders, committed `049b798`, routed to CTO. |
| CTO | Passed | Created `T-001` ordinary product ticket, committed `066f984`, routed to Engineer. |
| Engineer | Passed | Claimed `T-001`, wrote `index.html`, `src/input.js`, `src/player.js`, `src/game.js`, `run-tests.js`, and tests; fixed a failing movement test; committed `9e81794` and moved ticket to done with `64d8b6e`. |
| QA | Passed | Read implementation and done ticket, approved `T-001`, routed to Security. |
| Security | Passed | Created `docs/reports/security/security-audit-2026-05-19.md`, committed `2f7691b`, routed to Dogfood. |
| Dogfood | Passed | Served the app with `python3 -m http.server 8080`, verified HTTP 200 and source assets, wrote `docs/reports/dogfood/2026-05-19-e2e-validation.md`, committed `36495d3`, routed to Release Manager. |
| Release Manager | Blocked | `mars_harness_cli` resolved `/path/to/local-redacted` at `0.0.1-dev`; `release` and `tools` commands were unavailable. |

Observed dispatch decisions:

```text
ceo          exec_plan         coo
coo          ticket_breakdown  cto-weekly
cto-weekly   implementation    engineer
engineer     qa_review         qa
qa           security_review   security
security     security_review   dogfood
dogfood      release_review    release-manager
```

Observed job outcomes before the bounded stop:

```text
ceo, coo, cto-weekly, engineer, qa, security, dogfood: passed
release-manager: failed after operator stop while investigating stale CLI
```

## Performance Notes

- Model startup was acceptable for local dogfood: reasoning server reached
  health in about 6 seconds; the coding server reached health in about 7
  seconds.
- Product progress happened before intervention debt: the run created and
  closed an ordinary product ticket before release-governance work.
- Guardrail blocks for broad `find .` commands were recorded as foundation
  telemetry and did not create target backlog churn.
- Optional GitHub/push paths were skipped honestly because the target had no
  `origin` remote.
- The target was left with only `.harness/learnings.yaml` dirty after the
  operator stop interrupted Release Manager auto-commit.

## Follow-Up Tickets

- `T-007`: fix deployed `mars_harness_cli` binary resolution during release
  review so target Release Manager uses a current harness binary or emits
  actionable stale-binary guidance.
- `T-008`: make dashboard stop exit the `start` process cleanly; `POST
  /api/stop` stopped workers and inference but returned `dashboard shutdown:
  context deadline exceeded`, leaving the process to be killed manually.

## T-008 Stop Replay

After the dashboard stop fix, a fresh stop-focused replay used:

- Target: `<validation-root>`
- Harness binary: `<validation-root>`
- DB: `<validation-root>`
- Log: `<validation-root>`
- Ports: webhook `19115`, dashboard `19114`

The replay initialized a clean Space Invaders target, committed the generated
harness baseline, registered the repo, seeded one CEO bootstrap job, started the
dashboard, and accepted `POST /api/stop` with HTTP `200` and body
`{"ok":true}`. The `start` process then exited without manual kill after
logging `serve: dashboard stop requested, shutting down`, `queue: worker pool
stopped gracefully`, `scheduler: stopped`, `inference router: stopped managed
servers`, `power: sleep prevention released`, and `serve: orchestrator stopped`.

Because the stop was issued while CEO was mid-turn, the CEO job ended as
`llm_unreachable` with `context canceled`. The resulting foundation-owned
runtime signal stayed out of the target backlog, matching the stabilization
rule that operator/runtime stops are telemetry unless explicitly target-owned.
The temporary target was left clean by committing `.harness/learnings.yaml` as
`43338da`; it has no remote to push.

## Run 5 Ticket-Handoff Replay

After the quality-score calibration and `v0.41.27` publication pass, a fresh
replay used:

- Target: `<validation-root>`
- Harness binary: `<validation-root>`
- DB: `<validation-root>`
- Log: `<validation-root>`
- Ports: webhook `19125`, dashboard `19124`
- Model stack: local Qwen3-Coder GGUF through managed `llama-server`

The run improved the original intervention-debt failure class again:

| Stage | Result | Evidence |
| --- | --- | --- |
| Init/register/start | Passed | Generated harness baseline committed, one CEO bootstrap job seeded, reasoning server became healthy after about 32 seconds. |
| CEO | Passed | Read the Space Invaders brief, recorded a product-slice decision, and handed to COO in 8 turns. |
| COO | Passed after guardrail recovery | Tried to finish with uncommitted plan/feature edits, was blocked, committed `8d7205b`, then handed to CTO. The guardrail signal stayed foundation telemetry. |
| CTO | Passed | Created one ordinary product ticket `T-001`, committed `8cd8360`, and handed to Engineer. |
| Engineer | Passed for first slice | Claimed `T-001`, wrote `src/index.html`, `src/styles.css`, `src/main.js`, created `package.json`, served the static app, moved `T-001` to done, and handed to QA. |
| QA | Passed but shallow | Reviewed files and docs, approved `T-001`, but did not run a native/browser smoke itself. |
| Security | Passed | Wrote and committed `docs/reports/security/security-audit-2026-05-19.md`. |
| Dogfood | Found target-owned product gap | Served the app, observed the root route returned a directory listing and the implementation lacked enemy waves, score/lives, and win/lose behavior; created `T-002`. |
| Rework handoff | Blocked | Dogfood recorded `changes_requested` without committing `T-002`; the next Engineer saw the ticket in context but `git mv` could not claim the untracked ticket, causing repeated guardrail blocks and a ticket-gate failure. |

Key observations:

- Product progress happened before process debt: the run reached implementation,
  QA, security, Dogfood, and a target-owned rework ticket.
- Guardrail blocks for broad `find .`, pre-claim product mutation, direct ticket
  file writes, and forbidden `rm` stayed foundation telemetry and did not create
  target intervention-debt churn.
- The next source fix is mechanical ticket handoff consistency: roles that
  create target tickets or evidence must commit those artifacts before terminal
  dispositions such as `changes_requested` can hand work to another role.

## Run 6 Clean-Tree Handoff Replay

After adding the terminal-disposition clean-tree gate, a fresh replay used:

- Target: `<validation-root>`
- Harness binary: `<validation-root>`
- DB: `<validation-root>`
- Log: `<validation-root>`
- Ports: webhook `19135`, dashboard `19134`
- Model stack: local Qwen3-Coder GGUF through managed `llama-server`

The first sandboxed start attempt initialized the target and seeded CEO but
could not bind the local webhook port. Rerunning the same target outside the
sandbox started successfully. The replay showed bootstrap recovery, product
progress, and a remaining watchdog gap:

| Stage | Result | Evidence |
| --- | --- | --- |
| Init/register/start | Passed with retry | Target scaffold committed, start retry succeeded, and the run continued from a single active CEO lifecycle. |
| CEO | Passed after clean-tree recovery | CEO wrote `docs/product-specs/vision.md`, was blocked from disposition while it was uncommitted, committed `e8473a9`, then handed to COO. |
| COO | Passed after scope recovery | COO wrote product-specific plan and `F-002`, attempted implementation and a second active plan, recovered, committed `fecfaa0`, then handed to CTO. |
| CTO | Passed | Created and committed ordinary product ticket `T-001` as `b1e9f6f` before handing to Engineer. |
| Engineer | Passed but tool-noisy | Claimed `T-001`, implemented static `src/` game files, served and curled the app, moved `T-001` to done, and handed to QA. It also used broad process inspection/kills during local server cleanup. |
| QA | Passed but shallow | Read files and docs, approved without an independent browser/native smoke. |
| Security | Passed | Wrote and committed `docs/reports/security/security-audit-2026-05-20.md` as `2b5c74e`. |
| Dogfood | Found product gap but hit max turns | Dogfood eventually served `src/` successfully, created target-owned `T-002`, then hit `max_turns` before committing or recording a terminal disposition. |
| Watchdog | Regressed | Runtime-failure dispatch was quarantined, but the later orchestrator survey routed an Engineer job for `dogfood_failure` while `T-002` was still uncommitted. |

Observed job outcomes at stop:

```text
ceo|completed|1
coo|completed|1
cto-weekly|completed|1
engineer|completed|1
qa|completed|1
security|completed|1
dogfood|failed|1
engineer|failed|1   # operator stop while investigating the watchdog-routed job
```

Observed telemetry:

```text
ceo|guardrail_block|1
coo|guardrail_block|2
engineer|guardrail_block|1
dogfood|guardrail_block|3
dogfood|max_turns|1
engineer|llm_unreachable|1   # operator stop
```

Intervention-debt count remained `0`. The remaining blocker is no longer
terminal disposition handoff alone; it is failed-role cleanup after a role
creates target-owned artifacts on the final turn. The follow-up source fix is
to make the orchestrator survey pause autonomous routing for a repo with
uncommitted non-runtime target changes. Runtime-only `.harness/learnings.yaml`
remains non-blocking.

The patched short replay used `<validation-root> serve`
against the same dirty target and DB. Startup survey logged
`orchestrator survey paused for dirty target workspace` with
`docs/tickets/backlog/T-002-dogfood-pre-flight-missing-core-game-mechanics-for-space-inv.md`
as the reason, and the DB still had zero pending Engineer jobs afterward. The
temporary target has no remote, so the Dogfood finding was preserved locally as
commit `ebd22a1` (`dogfood: preserve T-002 finding evidence`) and left clean.

## Assessment

The lifecycle is materially healthier than the older intervention-debt-heavy
runs. The harness now reaches product-specific planning, product ticketing,
implementation, QA, security, dogfood validation, and release review on a fresh
Space Invaders target. The next limiting factor is no longer early planning or
intervention-debt amplification. The stale deployed CLI boundary was fixed in
`T-007`, and the dashboard stop cleanup issue was fixed in `T-008`; the
next limiting factors found by runs 5 and 6 are uncommitted target-ticket
handoff after Dogfood creates a target-owned finding and watchdog recovery
routing that can consume those uncommitted artifacts after a max-turn failure.
The dirty-target survey pause is now confirmed by a patched replay. The
remaining live-loop work is to reduce Dogfood turn/tool waste, make static app
serving evidence more deterministic, and keep broadening product and
observer-mode dogfood evidence.
