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

## Run 7 Factory Pace Replay — 2026-05-20

The next continuous-improvement replay used:

- Target: `<validation-root>`
- Harness binary: `<validation-root>`
- DB: `<validation-root>`
- Log: `<validation-root>`
- Ports: webhook `19145`, dashboard `19144`
- Model stack: local Qwen3-Coder and Gemma GGUF models through managed
  `llama-server`

The replay validated the live loop after the dirty-target survey fix and
produced a clearer pace baseline:

```text
role             status      turns  tools  wall time
ceo              completed       7      7  28.815s
coo              completed      17     17  149.301s
cto-weekly       completed      18     18  104.133s
engineer         completed      41     41  180.538s
qa               completed      12     12  37.293s
security         completed      13     13  47.683s
dogfood          completed      21     21  112.306s
release-manager  completed      28     28  96.791s
janitor          failed          2      1  25.587s
```

Observed target outcome:

- CEO, COO, CTO, Engineer, QA, Security, Dogfood, and Release Manager all
  reached terminal successful dispositions.
- The target advanced through product-specific plan, feature contract,
  ordinary product ticket, player-movement implementation, QA approval,
  security audit, Dogfood HTTP validation, and local release notes.
- Intervention-debt count remained `0`.
- Dogfood completed instead of hitting `max_turns`, and no dirty watchdog
  routing occurred after Dogfood.
- The dashboard stop endpoint returned `{"ok":true}` and the `start` process
  exited cleanly.

Observed telemetry:

```text
cto-weekly|guardrail_block|1
engineer|guardrail_block|4
release-manager|guardrail_block|1
janitor|unknown|1
```

The replay exposed the next bounded factory-pace sinks:

- Engineer spent avoidable turns on broad `find .`, build/script discovery,
  mutation before ticket claim, and a stray `:q` command before completing a
  small player-movement ticket.
- Dogfood improved materially versus run 6 but still spent turns on malformed
  `shell_exec` argv, `.harness` inspection, and a Podman probe before the
  static HTTP smoke path.
- Release Manager handled local release notes but still attempted GitHub
  release commands in a target with no remote.
- Janitor failed after passing `ls -F docs/tickets/backlog/` as a single argv
  element and then producing an empty response.

The follow-up source fix hardens `shell_exec` to normalize harmless malformed
argv shapes and tightens generated Dogfood guidance so static package manifests
with `python3 -m http.server` skip Podman, dependency sync, and build detours.

## Run 8 Patched Replay — 2026-05-20

The next replay used the patched binary from the run 7 fix:

- Target: `<validation-root>`
- Harness binary: `<validation-root>`
- DB: `<validation-root>`
- Log: `<validation-root>`
- Ports: webhook `19155`, dashboard `19154`

Observed lifecycle:

```text
role             status      turns  tools  wall time
ceo              completed      11     11  57.007s
coo              completed      14     14  116.514s
cto-weekly       completed      15     15  97.596s
engineer         completed      41     41  265.196s
qa               completed      13     13  47.216s
security         completed      18     18  181.812s
dogfood          completed      36     36  159.548s
release-manager  completed      20     20  68.194s
```

Observed target outcome:

- The harness produced product-specific goals, an active plan, feature contract,
  ordinary feature ticket, player movement implementation, QA approval, security
  audit, Dogfood validation report, and local release notes.
- Every claimed role through Release Manager completed. Dispatch then stopped
  with `same-role next_need has no forward owner`, so Janitor did not run.
- Intervention-debt count remained `0`.
- The target working tree was clean at stop time.

Observed telemetry:

```text
ceo|guardrail_block|1
coo|guardrail_block|1
engineer|guardrail_block|2
dogfood|guardrail_block|2
```

Confirmed improvement:

- The malformed `shell_exec` argv from run 7 no longer blocked `ls`-style calls.
- The lifecycle reached a product release-note commit without dirty watchdog
  routing or target intervention-debt amplification.

Remaining pace and quality sinks:

- CTO still encoded `git_commit.paths` and `job_disposition_record.evidence_links`
  as JSON strings before recovering.
- Engineer created a throwaway root validation script, then `rm` guardrails
  blocked cleanup; the later ticket commit swept `test-ship-movement.js` into
  the target history.
- Security repeatedly emitted a malformed `file_write` payload with
  `<parameter=content>` embedded in the path before falling back to a heredoc.
- Dogfood still probed Podman, used one broad `find .` shape, attempted a
  shell-loop readiness check in argv mode, and spent 36 turns validating a small
  static page.
- Release Manager checked GitHub release state even though the target had no
  remote, then correctly treated publication as blocked.

The follow-up source fix expands the run 7 tool normalization to `file_write`
parameter-marker drift and tightens generated Dogfood guidance for static
projects without package manifests: classify static projects from bounded file
name listings, skip Podman before probing it when no container path is selected,
avoid shell-loop readiness scripts, and avoid throwaway root validation files.

## Run 9 Static Guidance Replay — 2026-05-20

The next replay used:

- Target: `<validation-root>`
- Harness binary: `<validation-root>`
- DB: `<validation-root>`
- Log: `<validation-root>`
- Ports: webhook `19165`, dashboard `19164`

Observed lifecycle:

```text
role        status     turns  tools  wall time
ceo         completed      6      6  23.679s
coo         completed     13     13  80.835s
cto-weekly  completed     16     16  76.586s
engineer    failed        50     50  387.450s
```

Observed target outcome:

- CEO, COO, and CTO stayed product-first and reached a committed product ticket.
- Engineer implemented substantial Space Invaders behavior and committed source
  changes, but failed with `max_turns` before moving `T-001` to done or recording
  a terminal disposition.
- Runtime containment worked correctly: `max_turns` was recorded as foundation
  telemetry, no intervention-debt ticket was created, and no Orchestrator loop
  was dispatched.

Observed telemetry:

```text
coo|guardrail_block|1
engineer|guardrail_block|3
engineer|max_turns|1
```

New failure class:

- Engineer still treated a no-manifest static project as package-managed and ran
  `npm run build`, then spent turns on broad `find .`, empty `shell_exec` calls,
  repeated listing/evidence commands, and source rewrites after implementation
  evidence had already passed.
- The next fix must move static-project validation doctrine into the Engineer
  role, not only Dogfood: no npm commands without `package.json`, use the static
  HTTP smoke path as the full verification suite for no-manifest static projects,
  and stop editing once source is committed and evidence is recorded.

## Run 10 Engineer Terminal Replay — 2026-05-20

The next replay used:

- Target: `<validation-root>`
- Harness binary: `<validation-root>`
- DB: `<validation-root>`
- Log: `<validation-root>`
- Ports: webhook `19175`, dashboard `19174`

Observed lifecycle before manual stop:

```text
role             status      turns  tools  wall time
ceo              completed       6      6  22.190s
coo              completed      15     15  115.993s
cto-weekly       completed      21     21  94.972s
engineer         completed      31     31  189.343s
qa               completed      10     10  37.212s
security         completed       9      9  50.977s
dogfood          completed      25     25  122.552s
release-manager  completed      24     24  70.239s
orchestrator     completed      16     16  68.190s
dogfood          running         5      5  stopped manually
```

Observed target outcome:

- Engineer no longer hit `max_turns`; it claimed the ticket, implemented a
  static Space Invaders movement slice, ran a static HTTP smoke, moved `T-001`
  to done, and handed off to QA.
- Dogfood skipped Podman entirely when no package/container manifest existed and
  completed with committed evidence.
- Intervention-debt count remained `0`.
- Release Manager generated local release notes, but then guessed and added a
  fake GitHub remote in the throwaway target before recording release blocked.
- Orchestrator treated the release-blocked disposition as a reason to route back
  to Dogfood, creating an unnecessary loop edge. The run was stopped manually at
  the second Dogfood pass.

Observed telemetry:

```text
cto-weekly|guardrail_block|1
engineer|guardrail_block|2
dogfood|guardrail_block|1
```

Confirmed improvement:

- The Engineer static-validation prompt fix turned the run 9 `max_turns` failure
  into a completed Engineer handoff.
- The Dogfood static prompt fix removed the Podman probe from the main Dogfood
  pass.

Remaining pace and routing sinks:

- CTO still searches lifecycle tickets through shallow `docs/tickets/*` globs
  and spends turns recovering from missing directory reads.
- Engineer still attempts one broad `find .` and one raw `mv` before using
  `git mv`.
- Dogfood still attempts one broad `find .` and emits several empty
  `shell_exec` calls.
- Release Manager must never mutate remotes in generated targets. No-remote
  targets should stop after local release notes/tag evidence and record a
  release-blocked disposition.
- Orchestrator should not route release-blocked publication failures back to
  Dogfood unless a product validation artifact explicitly changed.

The follow-up source fix blocks `git remote add`, `git remote set-url`, and
related remote mutation commands through `shell_exec`, and strengthens generated
Release Manager guidance so no-remote targets record publication blockers
instead of inventing remotes.

## Run 11 Release-Blocked Terminal Replay — 2026-05-20

The next replay used:

- Target: `<validation-root>`
- Harness binary: `<validation-root>`
- DB: `<validation-root>`
- Log: `<validation-root>`
- Ports: webhook `19185`, dashboard `19184`

Observed lifecycle before dashboard stop:

```text
role             status      turns  tools  wall time
ceo              completed       7      7   28.421s
coo              completed       9      9   32.757s
cto-weekly       completed      15     15   66.122s
engineer         completed      41     41  265.055s
qa               completed      12     12   36.957s
security         completed      12     12   56.007s
dogfood          completed      37     37  201.970s
release-manager  completed      26     26  166.329s
```

Observed dispatch decisions:

```text
ceo|exec_plan|coo|deterministic||using role suggested_role without Orchestrator detour
coo|feature_contract|cto-weekly|deterministic||completed same-role next_need already belongs to source role; routing to default forward owner without Orchestrator detour
cto-weekly|implementation|engineer|deterministic||using role suggested_role without Orchestrator detour
engineer|qa_review|qa|deterministic||routing completed work by next_need without Orchestrator detour
qa|security_review|security|deterministic||routing completed work by next_need without Orchestrator detour
security|security_review|dogfood|deterministic||current review next_need already belongs to source role; routing to next review owner without Orchestrator detour
dogfood|release_review|release-manager|deterministic||routing completed work by next_need without Orchestrator detour
release-manager|release_blocked||deterministic|release publication blocked|release publication blocker is operator-visible; stopping dispatch without Orchestrator detour
```

Confirmed improvement:

- The target reached product planning, ticketing, implementation, QA, Security,
  Dogfood, local release notes, and local tag creation.
- Pending/running queue count was `0` after Release Manager.
- The target had no git remote after the run, and no fake remote was added.
- No Orchestrator or Dogfood loop followed the `release_blocked` disposition.
- No intervention-debt ticket files were created.

Remaining live-loop findings:

- `mars_harness_cli` rejected list-shaped arguments emitted as a string, e.g.
  `{"args":"['release', 'notes', '--repo', '.', '--bump', 'auto']"}`.
- `shell_exec` rejected Python/single-quoted list strings for `argv`, causing a
  shell-command fallback.
- Engineer recovered from a duplicate `F-001` creation guardrail, but static
  source files kept `MarsDocSync` pointers to a non-existent generated feature
  path.
- Target `docsync_audit` reported `checked 0 files`, so stale static asset
  metadata did not become review evidence.
- Dogfood still spent turns discovering the static app entry directory before
  serving `src/`.

## Run 12 Tool-Argument And Matrix Replay — 2026-05-20

The next replay used:

- Target: `<validation-root>`
- Harness binary: `<validation-root>`
- DB: `<validation-root>`
- Log: `<validation-root>`
- Ports: webhook `19195`, dashboard `19194`

Observed lifecycle before dashboard stop:

```text
role          status      count
ceo           completed       1
coo           completed       1
cto-weekly    completed       1
engineer      completed       1
qa            completed       1
security      completed       1
dogfood       completed       1
orchestrator  completed       1
cto-weekly    failed          1  # operator stop during rework planning
```

Confirmed improvement:

- Static source files used the canonical
  `docs/features/F-001-product-walking-skeleton.md` path in `MarsDocSync`
  metadata instead of inventing a second `F-001` feature contract.
- No target intervention-debt tickets were created; guardrail and operator-stop
  runtime signals stayed in foundation telemetry.
- Product progress still reached planning, ticketing, implementation, QA,
  Security, Dogfood, and a target-owned rework ticket.

New evidence:

- The string-list payload issue is generic. Orchestrator emitted
  `workspace_hygiene` with `paths` as a JSON string, which failed for the same
  reason `mars_harness_cli.args` failed in run 11.
- Prompt-only static guidance is insufficient. Engineer and Dogfood still
  started a repo-root static server and recovered to `src/` after seeing a
  directory listing.
- `docsync_audit` still reported `checked 0 files` even though static assets
  carried `MarsDocSync` comments, so static asset metadata needs mechanical
  detection before it can be relied on as review evidence.
- Using only the Space Invaders static app risks overfitting. Future broad
  lifecycle claims should run a representative matrix across at least one
  additional project archetype.

Follow-up verification for T-015:

```text
go run ./cmd/mars-harness tools run docsync_audit --repo <validation-root> --args-json '{}'

# docsync_audit
docsync: checked 3 files, findings 1
FAIL: src/index.html: missing MarsDocSync docs metadata
```

The same run-12 target now audits static `src/` files instead of reporting
`checked 0 files`. The remaining finding is product evidence: `src/index.html`
needs metadata, while the inline CSS/JavaScript metadata is detected.

Factory pace baseline for T-011:

```text
go run ./cmd/mars-harness scores export --repo <validation-root> --db <validation-root>

Exported quality score to <validation-root>
Overall grade: A
```

The exported Factory Pace section captured eight traced jobs. The slowest
averages were Engineer at 92 turns / 45 tool invocations / 269.0s, Dogfood at
66 turns / 32 tool invocations / 192.2s, CTO at 36 turns / 17 tool invocations
/ 89.8s, Orchestrator at 36 turns / 17 tool invocations / 91.0s, and COO at
32 turns / 14 tool invocations / 79.7s. The demo target committed the local
quality export as `438ff4b chore: export quality pace baseline`; the target has
no remote.

## Non-Static Matrix Replay: Task Notes API - 2026-05-20

Purpose: avoid overfitting the factory loop to the Space Invaders static app by
running a second fresh target archetype.

- Target: `<validation-root>`
- Brief: build a tiny local HTTP JSON API for task notes with create, list,
  complete, and health endpoints.
- DB: `<validation-root>`
- Binary: `<validation-root>`
- Target local commits: product planning reached `9b4897b`, ticket shaping
  reached `dc7c0a0`, first Engineer implementation reached `5f57ecf`, and the
  interrupted duplicate Engineer evidence plus quality export were committed as
  `8fc6909`. The target has no remote.

Job state at operator stop:

```text
ceo|completed|1
coo|completed|1
cto-weekly|completed|1
engineer|failed|1
engineer|running|1
```

Trace-derived pace:

```text
ceo|16 turns|7 tools|7 LLM calls|29.0s|completed
coo|42 turns|20 tools|20 LLM calls|210.3s|completed
cto-weekly|34 turns|16 tools|16 LLM calls|86.5s|completed
engineer|102 turns|50 tools|50 LLM calls|210.7s|max_turns
```

Telemetry:

```text
coo|guardrail_block|2
engineer|guardrail_block|2
engineer|max_turns|1
```

The good news is that the non-static replay reached product-specific planning,
updated the canonical `F-001` feature contract, created an ordinary product
ticket, claimed it, and produced a partial Go API implementation without
creating target intervention-debt tickets for runtime failures. This confirms
the stabilization is generic enough to start actual product work outside the
static game path.

The new bottleneck is scheduled duplicate work. While the first Engineer was
still running, the minute scheduler enqueued a second Engineer for the same
repo. After the first Engineer hit `max_turns`, the duplicate was claimed and
started redoing implementation work. The queue serialized execution, but it did
not prevent same-repo same-role scheduled work from stacking behind an active
job. This became the first generic optimization target for `T-011`.

## API Rerun After Scheduler Skip: Task Notes API - 2026-05-20

Purpose: rerun the non-static API canary after the scheduler active-role skip
fix, while continuing to collect the next generic factory bottleneck.

- Target: `<validation-root>`
- DB: `<validation-root>`
- Binary: `<validation-root>`
- Source version: `v0.42.1` plus the pushed scheduler fix
  `e9187f2 fix(scheduler): skip active same-role scheduled work`
- Target local evidence commit: `86a298a chore: capture build artifact containment evidence`
- Target remote: none

Job state:

```text
ceo|completed|1
coo|completed|1
cto-weekly|completed|1
engineer|failed|1
```

Trace-derived pace:

```text
ceo|16 turns|7 tools|7 LLM calls|24.8s|completed
coo|32 turns|15 tools|15 LLM calls|95.0s|completed
cto-weekly|38 turns|18 tools|18 LLM calls|77.6s|completed
engineer|89 turns|43 tools|44 LLM calls|354.8s|circle_detected
```

Telemetry:

```text
coo|guardrail_block|1
cto-weekly|guardrail_block|1
engineer|circle_detected|1
engineer|guardrail_block|21
```

The scheduler duplicate-work fix did not regress product progress: the clean
API canary again reached product-specific planning, ticket creation, and
ticket-backed implementation. No target intervention-debt tickets were created
for runtime or guardrail failures. This run did not coincide with the default
Engineer cron boundary, so it is lifecycle health evidence plus unit-tested
scheduler coverage rather than a live cron-boundary proof.

The next generic bottleneck is repo-local compiled build artifacts. Engineer
ran `go build .`, which produced an untracked binary named `demo-api-run2` in
the target root. Blast-radius line counting treated that binary as a huge file,
then blocked later file writes, tests, ticket lifecycle moves, and commits.
Engineer correctly attempted cleanup with `rm demo-api-run2`, but destructive
shell policy blocked all `rm` commands. The run ended with `circle_detected`
after repeated terminal disposition attempts against an in-progress ticket that
could not be moved because the generated binary kept the repo outside safety
limits.

## API Rerun After Build-Artifact Cleanup: Task Notes API - 2026-05-20

Purpose: rerun the non-static API canary after the bounded root build-artifact
cleanup exception, and avoid treating the Space Invaders static target as the
only proof of generic factory health.

- Target: `<validation-root>`
- DB: `<validation-root>`
- Binary: `<validation-root>`
- Source version: `v0.42.2` plus the pushed build-artifact cleanup fix
  `1e42e21 fix(tools): allow cleanup of root build artifacts`
- Target local evidence commit:
  `b42890b chore: capture feature-contract guidance evidence`
- Target remote: none

Job state at operator stop:

```text
ceo|completed|1
coo|running|1
```

Trace-derived pace:

```text
ceo|32 turns|15 tools|15 LLM calls|83.2s|completed
```

Telemetry:

```text
ceo|guardrail_block|2
coo|guardrail_block|2
```

The run exposed a planning guidance bottleneck before it could exercise the
build-artifact cleanup path. CEO correctly discovered the generated canonical
`docs/features/F-001-product-walking-skeleton.md` contract, but also attempted
to create a second product-specific `docs/features/F-001-task-notes-api.md`
contract. COO then attempted the same duplicate path, and its later canonical
contract update tried to append duplicate scenario headings. The existing
guardrails prevented duplicate feature contracts and duplicate scenario IDs, so
the target stayed protected and no intervention-debt tickets were created, but
the lifecycle stopped before CTO ticketing or Engineer implementation.

This is a generic fresh-bootstrap issue. The source fix now makes canonical
feature-contract reuse explicit in generated CEO and COO guidance: CEO must not
write `docs/features/`, and COO must search `docs/features/F-NNN*.md`, edit the
existing path when present, and rewrite starter scenarios in place with unique
IDs. The next API canary should confirm bootstrap planning reaches CTO and then
exercise the build-artifact cleanup improvement.

## API Rerun After Canonical Bootstrap Guidance: Task Notes API - 2026-05-20

Purpose: rerun the non-static API canary after canonical feature-contract
guidance, confirm the fresh lifecycle reaches CTO and Engineer, and collect the
next generic factory bottleneck without overfitting to the Space Invaders demo.

- Target: `<validation-root>`
- DB: `<validation-root>`
- Binary: `<validation-root>`
- Source version: `v0.42.3` plus the pushed canonical bootstrap guidance fix
  `decb5f3 fix(scanner): reuse canonical bootstrap feature contracts`
- Target local evidence commit:
  `d9b64ce chore: capture run4 module artifact evidence`
- Target remote: none

Job state:

```text
ceo|completed|1
coo|completed|1
cto-weekly|completed|1
engineer|failed|1
```

Trace-derived pace:

```text
ceo|28 turns|13 tools|13 LLM calls|53.7s|completed
coo|36 turns|17 tools|17 LLM calls|111.2s|completed
cto-weekly|32 turns|15 tools|15 LLM calls|70.7s|completed
engineer|102 turns|50 tools|50 LLM calls|357.0s|max_turns
```

Telemetry:

```text
coo|guardrail_block|1
cto-weekly|guardrail_block|1
engineer|guardrail_block|25
engineer|max_turns|1
```

The canonical bootstrap guidance improved the live lifecycle. CEO, COO, and
CTO completed exactly once, COO updated the canonical
`docs/features/F-001-product-walking-skeleton.md` path instead of creating a
duplicate product-specific `F-001` contract, CTO created an ordinary product
ticket, and Engineer began implementing the Task Notes API. No target
intervention-debt tickets were created for the runtime or guardrail failures,
and the max-turn failure stayed foundation-owned telemetry rather than
dispatching an Orchestrator recovery loop.

The next generic bottleneck is still generated build-artifact containment, but
with a broader artifact name. Engineer generated a root executable named
`task-notes-api`, matching the Go module basename instead of the target repo
directory `demo-api-run4`. The existing cleanup exception allowed only the repo
basename, so blast-radius checks repeatedly blocked file writes,
`record_decision`, and terminal disposition because the untracked binary looked
like a 33,970-line source change. The source fix now extends the same bounded
cleanup rule to untracked, root-level, binary-looking artifacts named after the
root `go.mod` module basename. Ordinary deletion, tracked files, nested paths,
and recursive removal remain blocked.

## API Rerun After Module Artifact Cleanup: Task Notes API - 2026-05-20

Purpose: rerun the non-static API canary after module-named build-artifact
cleanup, confirm Engineer can continue through a generated Go binary, and
collect the next generic factory bottleneck.

- Target: `<validation-root>`
- DB: `<validation-root>`
- Binary: `<validation-root>`
- Source version: `v0.42.4` plus the pushed module-named cleanup fix
  `3e57ba1 fix(tools): allow module-named build artifact cleanup`
- Target local evidence commit:
  `d3ff879 chore: capture run5 cleanup discoverability evidence`
- Target remote: none

Job state:

```text
ceo|completed|1
coo|completed|1
cto-weekly|completed|1
engineer|failed|1
```

Trace-derived pace:

```text
ceo|26 turns|12 tools|12 LLM calls|56.9s|completed
coo|26 turns|12 tools|12 LLM calls|78.0s|completed
cto-weekly|38 turns|18 tools|18 LLM calls|84.9s|completed
engineer|102 turns|50 tools|50 LLM calls|232.4s|max_turns
```

Telemetry:

```text
ceo|guardrail_block|1
engineer|guardrail_block|16
engineer|max_turns|1
```

The module-name cleanup patch did not regress early lifecycle behavior. CEO,
COO, and CTO completed once; COO again reused the canonical `F-001` feature
contract; CTO created an ordinary product ticket; and Engineer claimed the
ticket, committed product code, added a Go module, and reached validation.
Runtime failures still stayed foundation-owned telemetry with zero target
intervention-debt ticket amplification.

The run exposed the missing ergonomics around cleanup discovery. Engineer ran
`go build -o task-notes-api src/main.go src/api.go`, which produced an
untracked binary that the source policy would allow to be removed with
`rm task-notes-api`. However, the blast-radius error still only said to split
changes or raise `MaxLinesPerFile`; it did not say that this specific
generated artifact had a safe cleanup path. Engineer therefore kept retrying
builds, writes, `git add`, and ticket movement against the dirty repo rather
than trying `rm task-notes-api`, and eventually hit `max_turns`. The next fix
keeps the removal exception narrow but makes the error actionable by appending
the exact cleanup command when the oversized file is an untracked, root-level,
binary-looking repo/module artifact.

## API Rerun After Generated Artifact Cleanup Hints: Task Notes API - 2026-05-20

Purpose: rerun the non-static API canary after blast-radius errors started
naming the exact generated-artifact cleanup command, confirm Engineer can
recover from a module-named Go binary, and collect the next generic factory
bottleneck.

- Target: `<validation-root>`
- DB: `<validation-root>`
- Binary: `<validation-root>`
- Source version: `v0.42.5` plus the pushed cleanup-hint fix
  `1025fe7 fix(tools): hint generated artifact cleanup`
- Target local evidence commit:
  `ccd5367 chore: capture run6 server validation evidence`
- Target remote: none

Job state:

```text
ceo|completed|1
coo|completed|1
cto-weekly|completed|2
engineer|failed|1
engineer|pending|1
```

Trace-derived pace:

```text
ceo|20 turns|9 tools|9 LLM calls|31.7s|completed
coo|28 turns|13 tools|13 LLM calls|79.6s|completed
cto-weekly|20 turns|9 tools|9 LLM calls|35.9s|completed
cto-weekly|30 turns|14 tools|14 LLM calls|62.8s|completed
engineer|78 turns|38 tools|38 LLM calls|282.9s|llm_unreachable after manual stop
```

Telemetry:

```text
cto-weekly|guardrail_block|1
engineer|guardrail_block|5
engineer|tool_timeout|1
engineer|llm_unreachable|1
```

The cleanup-hint fix improved the real lifecycle. Engineer built the
module-named `task-notes-api` binary, saw the blast-radius message naming
`rm task-notes-api`, removed it, and continued instead of exhausting the run on
artifact cleanup. Product progress remained intact: CEO and COO planned from
the Task Notes API brief, CTO created `T-001`, Engineer claimed the ticket,
committed a Go `GET /health` implementation, and preserved local quality
evidence. No target intervention-debt tickets were created for runtime or
guardrail failures.

The next generic bottleneck is server validation. Engineer first ran
`go run src/main.go` as a foreground process, which spent a 30-second timeout
before returning a "use background:true" hint. It then tried to emulate a
background server through shell syntax inside `shell_command`, which left port
`8080` occupied by the compiled server process. A later tool-managed
`background:true` attempt reported "address already in use" but still looked
like a started background process, and the role spent more turns on malformed
`:8080` shell calls and process inspection. The source fix now rejects shell
background `&` inside `shell_command`, reports `background:true` startup exits
as tool errors with initial output, and mirrors the managed-background rule
into generated Engineer guidance.

## API Rerun After Managed Background Validation: Task Notes API - 2026-05-20

Purpose: rerun the non-static API canary after managed background validation
hardening, confirm service probing no longer leaks port `8080`, and collect the
next generic factory bottleneck.

- Target: `<validation-root>`
- DB: `<validation-root>`
- Binary: `<validation-root>`
- Source version: `v0.42.6` plus the pushed managed-background fix
  `fe2fb41 fix(tools): harden managed server validation`
- Target local evidence commit: none; the failed run intentionally retained the
  dirty generated binary state as evidence
- Target remote: none

Job state:

```text
ceo|completed|1
coo|completed|1
cto-weekly|completed|1
engineer|failed|1
```

Trace-derived pace:

```text
ceo|30 turns|14 tools|14 LLM calls|61.6s|completed
coo|32 turns|15 tools|15 LLM calls|102.6s|completed
cto-weekly|36 turns|17 tools|17 LLM calls|78.3s|completed
engineer|71 turns|34 tools|35 LLM calls|154.2s|circle_detected
```

Telemetry:

```text
ceo|guardrail_block|1
coo|guardrail_block|1
cto-weekly|guardrail_block|1
engineer|guardrail_block|5
engineer|circle_detected|1
```

The managed-background fix improved the live service path. Engineer started
`go run main.go` with `background:true`, received a successful startup result,
probed `http://localhost:8080/health`, and killed PID `43395` cleanly. The
previous foreground timeout, shell `&` leak, port-conflicted false success, and
malformed `:8080` cleanup loop did not recur. No target intervention-debt
tickets were created.

The next generic bottleneck is prevention of validation build artifacts.
Engineer ran `go build -o task-notes-api main.go`, which created an untracked
root binary inside the target repo. Blast-radius validation correctly blocked
the 34,014-line binary-shaped diff and kept the failure as foundation telemetry,
but the repo was already dirty. The model then emitted malformed empty
`shell_exec` calls; those calls were masked by the dirty blast-radius precheck
until `circle_detected`. The source fix now rejects `go build -o <path>` when
`<path>` resolves inside the target repo, tells roles to write runnable
validation binaries to an external temp path, and validates malformed
`shell_exec` payloads before dirty-diff containment can obscure the true tool
shape error.

## API Rerun After Build Output Prevention: Task Notes API - 2026-05-20

Purpose: rerun the non-static API canary after repo-local validation binaries
were blocked before execution, confirm no root binary is created, and collect
the next generic factory bottleneck.

- Target: `<validation-root>`
- DB: `<validation-root>`
- Binary: `<validation-root>`
- Source version: `v0.42.7` plus the pushed build-output prevention fix
  `e9dd4a9 fix(tools): prevent repo-local validation binaries`
- Target local evidence commit:
  `1d2f256 chore: capture run8 quality evidence`
- Target remote: none

Job state:

```text
ceo|completed|1
coo|completed|1
cto-weekly|completed|1
engineer|failed|1
```

Trace-derived pace:

```text
ceo|22 turns|10 tools|10 LLM calls|38.0s|completed
coo|24 turns|11 tools|11 LLM calls|75.3s|completed
cto-weekly|38 turns|18 tools|18 LLM calls|100.6s|completed
engineer|53 turns|25 tools|26 LLM calls|69.9s|circle_detected
```

Telemetry:

```text
cto-weekly|guardrail_block|1
engineer|guardrail_block|6
engineer|circle_detected|1
```

The build-output prevention fix worked in the live service canary. Engineer
claimed `T-001`, initialized the Go module, wrote `src/main.go`, committed the
`GET /health` implementation, and then attempted
`go build -o task-notes-api src/main.go`. `shell_exec` blocked that command
before execution because `task-notes-api` resolved inside the target repo. A
direct filesystem check confirmed `<validation-root>`
does not exist. The target ended clean after committing exported quality
evidence, and the quality score recorded zero open intervention-debt tickets.

The next generic bottleneck is malformed port-token recovery. After the
build-output block, Engineer called `shell_exec` with `argv:[":8080"]` twice.
The runtime treated it as an executable lookup and returned "executable file not
found"; the repeated malformed command triggered `circle_detected`. The source
fix now rejects bare port tokens such as `:8080` in `argv` or single-token
`shell_command` mode before process execution, states that ports are not
commands, and gives the `background:true` plus `curl http://localhost:8080/health`
validation shape.

## API Rerun After Bare-Port Rejection: Task Notes API - 2026-05-20

Purpose: rerun the non-static API canary after bare-port command rejection,
confirm Engineer recovers into product validation, and collect the next generic
turn sink without overfitting to the Space Invaders static game path.

- Target: `<validation-root>`
- DB: `<validation-root>`
- Binary: `<validation-root>`
- Source version: `v0.42.8` plus
  `1aa405f fix(tools): reject bare port validation commands`
- Target local evidence commits:
  `2dd2597 release: notes 0.2.0` and
  `cfcefd6 chore: capture run9 quality evidence`
- Target remote: none

Job state:

```text
ceo|completed|1
coo|completed|1
cto-weekly|completed|1
dogfood|completed|1
engineer|completed|1
qa|completed|1
release-manager|completed|1
security|completed|1
```

Telemetry:

```text
ceo|guardrail_block|2
coo|guardrail_block|1
cto-weekly|guardrail_block|1
dogfood|guardrail_block|2
engineer|guardrail_block|5
```

The bare-port fix worked in the live service canary. Engineer did not repeat
the `:8080` loop. After `go build -o task-notes-api` was blocked, it followed
the hint and built `<validation-root>`. Dogfood later validated
the shipped API with `go test ./...`, external build output, a managed
`background:true` server, `curl /health` returning HTTP 200 with JSON, and POST
`/health` returning HTTP 405. Release Manager generated local `0.2.0` release
notes and stopped on the expected no-remote publication blocker. `scores export`
reported overall grade `A`, zero open intervention-debt tickets, and Factory
Pace rows for all eight roles.

The next generic bottleneck is scratch validation pollution. Engineer created a
repo-root `validate.sh`, then tried forbidden `rm` cleanup twice before the
script was accidentally committed with the done-ticket move. Dogfood proved the
script was not portable because it called the host `timeout` command, which was
unavailable. QA and Security approved without rejecting the accidental script.
The source fix now blocks new repo-root validation shell scripts such as
`validate.sh` and rejects external `timeout`/`gtimeout` commands, steering
agents to existing tests, direct build/run/curl evidence, tool
`timeout_seconds`, and managed `background:true` probes.

## API Rerun After Scratch Validation Prevention: Task Notes API - 2026-05-20

Purpose: rerun the non-static API canary after scratch validation script
prevention and external timeout rejection, then validate that portable
background/server cleanup remains healthy.

- Target: `<validation-root>`
- DB: `<validation-root>`
- Binary: `<validation-root>`
- Source version: `v0.42.9` plus
  `053a6e2 fix(tools): block scratch validation scripts`
- Target remote: none

Job state at stop:

```text
ceo|completed|1
coo|completed|1
cto-weekly|completed|1
engineer|failed|1
```

Positive evidence:

- CEO, COO, and CTO reached product-specific planning and a single ordinary
  product ticket.
- Engineer claimed the ticket, wrote Go source and tests, initialized the Go
  module, and passed `go test ./...`.
- Bare `:8080` commands were rejected as tool-shape errors and did not create
  intervention-debt tickets.

Residual finding:

- The run inherited a stale server child process on port `8080` from the
  previous replay. The previous job cleanup killed the tracked `go run` wrapper
  but not the compiled child server. Engineer hit `listen tcp :8080: bind:
  address already in use`, repeated `:8080` cleanup attempts despite the guard,
  and stopped with `circle_detected`.

The source fix now makes `shell_exec` background cleanup discover and kill known
descendants before killing the tracked process group and process. The next API
canary should confirm no stale `go run` child server leaks into the target
validation path, then continue checking for root `validate.sh` and external
`timeout` use.

## API Rerun After Background Descendant Cleanup: Task Notes API - 2026-05-20

Purpose: rerun the non-static API canary after background descendant cleanup and
confirm stale server children no longer block product delivery.

- Target: `<validation-root>`
- DB: `<validation-root>`
- Binary: `<validation-root>`
- Source version: `v0.42.10` plus
  `aafa166 fix(tools): clean background process descendants`
- Target local evidence commit: `187725f chore: capture run11 quality evidence`
- Target remote: none

Job state at stop:

```text
ceo|completed|1
coo|completed|1
cto-weekly|completed|1
dogfood|failed|1
engineer|completed|1
qa|completed|1
security|completed|1
```

Telemetry:

```text
ceo|guardrail_block|3
dogfood|guardrail_block|7
dogfood|max_turns|1
engineer|guardrail_block|6
```

Positive evidence:

- CEO, COO, CTO, Engineer, QA, and Security completed product-specific work with
  one ordinary product ticket and zero target intervention-debt tickets.
- Engineer implemented the Go `/health` endpoint, initialized the module,
  passed `go test .`, recovered from a repo-local build-output block by using
  `<validation-root>`, validated the live endpoint with `curl`, and moved
  `T-001` to done.
- Dogfood found and killed a leaked `go run` child server with `lsof`, validated
  `GET /health` with HTTP 200 JSON evidence, confirmed a missing route returned
  404, ran `go test -v ./...`, and committed an E2E report.
- The Dogfood `max_turns` failure stayed in foundation telemetry and did not
  create target intervention-debt or dispatch another autonomous recovery loop.

Residual findings:

- Dogfood ran `go build ./...` without `-o`; the command created the default
  repo-root `task-notes-api` binary before post-command blast-radius validation
  rejected it. Dogfood recovered with the allowed generated-artifact cleanup,
  but the artifact should have been blocked before process execution.
- Manual `kill -9 <go-run-wrapper-pid>` can still leave the compiled `go run`
  child process alive during the same job. Dogfood recovered by killing the
  child PID directly, and the post-run operator check confirmed port `8080` was
  clear after cleanup.

The source fix now blocks `go build` without `-o` before execution and points
roles to `go test ./...` for compile validation or `/tmp/<name>-validation` for
runnable validation binaries. The next API canary should confirm Dogfood no
longer creates the implicit `task-notes-api` artifact and measure whether the
turn budget is now sufficient for a terminal Dogfood disposition.

## API Rerun After Default Go Build Preflight: Task Notes API - 2026-05-20

Purpose: rerun the non-static API canary after default `go build` output
preflight and confirm the validation path no longer creates repo-local build
artifacts before blast-radius checks.

- Target: `<validation-root>`
- DB: `<validation-root>`
- Binary: `<validation-root>`
- Source version: `v0.42.11` plus
  `f7b5f48 fix(tools): block implicit go build artifacts`
- Target local evidence commits:
  `c1a5170 chore: preserve run12 failed engineer workspace` and
  `dc4a0e2 chore: capture run12 quality evidence`
- Target remote: none

Job state at stop:

```text
ceo|completed|1
coo|completed|1
cto-weekly|completed|1
engineer|failed|1
```

Telemetry:

```text
coo|guardrail_block|1
engineer|circle_detected|1
engineer|guardrail_block|5
```

Positive evidence:

- CEO, COO, and CTO reached product-specific planning and a single ordinary
  product ticket with zero target intervention-debt tickets.
- COO hit the commit-before-disposition guardrail, corrected by committing the
  plan and feature-contract changes, and then completed.
- Engineer claimed `T-001`, implemented the Go `/health` endpoint, recovered
  from a foreground server timeout by using `background:true`, and validated
  `/health`, POST method rejection, and missing-route behavior.
- Explicit repo-local `go build -o task-notes-api` was blocked before artifact
  creation, and Engineer recovered with `<validation-root>`.
- The runtime failure stayed in foundation telemetry and did not create target
  intervention-debt or dispatch an autonomous recovery loop.

Residual finding:

- Same-job cleanup still leaked the compiled `go run` child. Engineer called
  `kill -TERM <tracked-wrapper-pid>` after the first live validation. The
  wrapper died, but its child server kept port `8080` bound. The subsequent
  `<validation-root>` binary exited during startup, and Engineer
  repeated bare `:8080` commands until `circle_detected`.

The source fix now intercepts `shell_exec` `kill <tracked-background-pid>` for
managed background processes and applies the same descendant process-tree
cleanup used at job end. The next API canary should confirm the `/tmp`
validation binary can start after a targeted kill without manual `lsof` cleanup
or bare-port loops.

## API Rerun After Tracked Background Kill: Task Notes API - 2026-05-20

Purpose: rerun the non-static API canary after same-job tracked-background kill
interception and confirm the external validation binary can start without
manual port cleanup.

- Target: `<validation-root>`
- DB: `<validation-root>`
- Binary: `<validation-root>`
- Source version: `v0.42.12` plus
  `7b9d8e7 fix(tools): kill tracked background process trees`
- Target local evidence commits:
  `8f60e56 chore: preserve run13 failed engineer workspace` and
  `06a2755 chore: capture run13 quality evidence`
- Target remote: none

Job state at stop:

```text
ceo|completed|1
coo|completed|1
cto-weekly|completed|1
engineer|failed|1
```

Telemetry:

```text
ceo|guardrail_block|2
coo|guardrail_block|1
cto-weekly|guardrail_block|1
engineer|circle_detected|1
engineer|guardrail_block|8
```

Positive evidence:

- CEO, COO, and CTO again produced product-specific planning, a feature
  contract, and one ordinary product ticket with zero target intervention-debt
  tickets.
- COO's attempted implementation write to `main.go` was blocked by role
  ownership policy, then COO recovered by committing only planning artifacts.
- CTO created `T-001` and committed it before recording the implementation
  handoff.
- Engineer claimed `T-001`, wrote Go product code and tests, and got
  `go test ./src` passing.
- Repo-local `go build -o task-notes-api` was blocked before artifact creation,
  and Engineer recovered with `<validation-root>`.
- Same-job tracked-background cleanup worked: Engineer started
  `go run src/main.go` with `background:true`, curled `/health`, and
  `shell_exec` intercepted `kill -9 <tracked-pid>` with "Killed background
  process tree", allowing the external validation binary to start on port
  `8080`.
- The external validation binary returned valid `/health` JSON, rejected
  `POST /health`, and returned Not found for `/invalid`.
- The terminal runtime failure stayed foundation-owned: no target
  intervention-debt tickets were created and no Orchestrator recovery loop was
  dispatched.

Residual finding:

- After product validation and ticket evidence updates, Engineer started the
  external validation binary a second time and then called `shell_exec` with
  empty `argv` and repeated single `:` commands. Those no-op calls became
  guardrail blocks and triggered `circle_detected` before the role could stop
  the tracked PID, move `T-001` to done, commit, and record a disposition.

The source fix now treats empty `argv`, blank `argv`, and single `:` calls as
no-op recovery hints rather than process-execution or guardrail failures. The
tool names active tracked background PIDs and tells the role to stop the PID,
update ticket evidence, commit, push, and record `job_disposition_record`.
Generated Engineer guidance also forbids empty/no-op shell calls as wait
commands. The next API canary should confirm Engineer completes the ticket
lifecycle after validation instead of looping on no-op calls.

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
run 7 and run 8 replays confirm Dogfood can now reach a terminal disposition
without dirty watchdog routing, and Release Manager can generate local release
notes for a clean target. Run 9 confirms max-turn containment works without
intervention-debt amplification, run 10 confirms the Engineer static-path prompt
fix turns that max-turn failure into a completed QA handoff, and run 11 confirms
no-remote release publication blockers now stop dispatch without remote mutation
or a Dogfood loop. Run 12 confirms structured-array payload drift is broader
than release tooling and validates the need for a representative live validation
matrix beyond one static Space Invaders canary. The Task Notes API replay then
confirmed the lifecycle reaches non-static product planning and implementation,
but exposed scheduled same-role duplication while an Engineer was already
running. The follow-up API rerun kept product progress intact and exposed the
next generic failure: repo-local compiled binaries can trap the agent behind
blast-radius and destructive-command policy. The next API rerun showed that
first-run generated planning guidance also needs to steer CEO and COO away from
duplicate `F-001` feature paths and duplicate starter scenarios before the
implementation path can be assessed. Factory pace is still dominated by
avoidable tool-use recovery, shallow ticket/file discovery, long-running process
mistakes, build artifact cleanup gaps, feature-contract path ambiguity, and
representative matrix coverage. The run4 API rerun confirms canonical planning
guidance reaches CTO and Engineer, but shows generated build artifacts must
cover module-named binaries as well as repo-named binaries. Run5 confirms the
module-named cleanup exception itself is not enough unless the guardrail error
also names the cleanup command. Run6 confirms that cleanup hints are effective
in real implementation and shifts the next generic bottleneck to managed
long-running server validation. Run7 confirms managed background validation
works in the service canary and shifts the next generic bottleneck to preventing
repo-local validation binaries before they dirty the target. Run8 confirms the
build-output prevention works and shifts the next generic bottleneck to
malformed bare-port validation commands. Run9 confirms bare-port rejection and a
full non-static lifecycle through local release notes; run10 exposes stale
`go run` child cleanup; run11 confirms the lifecycle reaches product
implementation, QA, Security, and Dogfood without intervention-debt
amplification, but shows implicit `go build` outputs and wrapper-child process
ownership still consume Dogfood turns. Run12 confirms product planning and
ticketing remain healthy and validates the external-build recovery path, but
same-job `kill <tracked-wrapper-pid>` still leaves a compiled `go run` child
server behind. Run13 confirms tracked-background kill interception works and the
`/tmp` validation binary can start without manual `lsof` cleanup or bare-port
loops; the remaining turn sink is empty/`:` no-op shell calls after successful
validation. The remaining live-loop work is to rerun the API canary after
no-op shell guidance, confirm Engineer completes the ticket lifecycle, and keep
validating against multiple software archetypes before making generic lifecycle
claims.
