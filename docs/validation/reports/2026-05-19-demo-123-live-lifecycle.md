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

## Assessment

The lifecycle is materially healthier than the older intervention-debt-heavy
runs. The harness now reaches product-specific planning, product ticketing,
implementation, QA, security, dogfood validation, and release review on a fresh
Space Invaders target. The next limiting factor is no longer early planning or
intervention-debt amplification; it is deployed CLI binary resolution at the
release boundary, plus shutdown cleanup ergonomics.
