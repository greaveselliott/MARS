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
| Release Manager | Blocked | `mars_cli` resolved `/path/to/local-redacted` at `0.0.1-dev`; `release` and `tools` commands were unavailable. |

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

- `T-007`: fix deployed `mars_cli` binary resolution during release
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

- `mars_cli` rejected list-shaped arguments emitted as a string, e.g.
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
  reason `mars_cli.args` failed in run 11.
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
go run ./cmd/mars tools run docsync_audit --repo <validation-root> --args-json '{}'

# docsync_audit
docsync: checked 3 files, findings 1
FAIL: src/index.html: missing MarsDocSync docs metadata
```

The same run-12 target now audits static `src/` files instead of reporting
`checked 0 files`. The remaining finding is product evidence: `src/index.html`
needs metadata, while the inline CSS/JavaScript metadata is detected.

Factory pace baseline for T-011:

```text
go run ./cmd/mars scores export --repo <validation-root> --db <validation-root>

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

## API Rerun After No-Op Shell Guidance: Task Notes API - 2026-05-20

Fresh target:
`<validation-root>`

Harness DB:
`<validation-root>`

Target local evidence commit:
`c423d9f chore: capture run14 quality evidence`

Job state at stop:

```text
ceo|completed|1
coo|completed|1
cto-weekly|completed|1
dogfood|completed|1
dogfood|failed|1
engineer|completed|1
qa|completed|1
release-manager|completed|1
security|completed|1
```

Telemetry:

```text
cto-weekly|guardrail_block|1
dogfood|guardrail_block|1
dogfood|llm_unreachable|1
dogfood|tool_timeout|1
engineer|guardrail_block|2
```

Positive evidence:

- CEO, COO, and CTO produced product-specific planning and a single ordinary
  product ticket with zero target intervention-debt tickets.
- Engineer claimed `T-001`, implemented `cmd/task-notes-api/main.go`, added
  `cmd/task-notes-api/main_test.go`, initialized `go.mod`, passed
  `go test ./...`, validated live `/health` responses, stopped tracked
  background PIDs, moved the ticket to done, and recorded `qa_review`.
- The v0.42.13 no-op shell fix worked in the live path: no empty `argv` or
  single `:` loop recurred after validation.
- Release Manager generated local `0.2.0` notes and stopped with an explicit
  no-remote publication blocker rather than adding or guessing a remote.
- No target intervention-debt tickets were created.

Residual findings:

- Engineer treated `F-001-S002` as if it were a feature-contract file and added
  malformed one-line `// MarsDocSync: docs/features/F-001-S002.md` metadata.
  `docsync_audit` correctly reported missing metadata for the implementation
  and test files.
- QA and Security both observed the DocSync failures and still approved, and
  Dogfood approved release readiness. This proved the DocSync rule needed a
  mechanical successful-disposition gate rather than relying on reviewer
  judgement.
- Dogfood used an external `timeout` wrapper. The shell policy rejected it, but
  telemetry classified the event as retryable `tool_timeout` and enqueued a
  duplicate Dogfood job.
- `scores export` produced an overall A despite the DocSync escape, so scoring
  still needs a later quality-signal correction after the handoff gate is
  fixed.

The source fix now blocks successful Engineer, QA, Security, Dogfood, Release
Manager, Dependency Manager, and Pipeline Fixer dispositions while
`docsync_audit` has findings. Deployed target repos still require valid
metadata, but they are no longer forced to cite foundation-only source docs.
Policy-blocked external `timeout` commands now classify as guardrail blocks so
they do not trigger deterministic retry work. The next API canary should either
produce valid DocSync metadata and proceed, or stop at implementation rework
instead of approving stale documentation.

### API Rerun After DocSync Disposition Gate (Task Notes API) - 2026-05-20

Target: `<validation-root>`

Binary: `<validation-root>`

Command:

```bash
<validation-root> start \
  --repo <validation-root> \
  --db <validation-root> \
  --log-file <validation-root> \
  --debug
```

Job outcomes:

```text
ceo|completed|1
coo|completed|1
cto-weekly|completed|1
engineer|failed|1
```

Telemetry:

```text
coo|guardrail_block|1
cto-weekly|guardrail_block|1
engineer|circle_detected|1
engineer|guardrail_block|2
```

Positive evidence:

- CEO, COO, and CTO reached product-specific planning and one ordinary product
  ticket with zero target intervention-debt tickets.
- COO and CTO recovered from clean-handoff guardrails by committing planning
  and ticket artifacts before disposition.
- Engineer claimed `T-001` before source mutation, implemented a Go
  `GET /health` endpoint, ran `go mod tidy`, passed `go test ./...`, and
  recovered from the repo-local build-output guardrail by building to
  `<validation-root>`.
- The failed Engineer job did not dispatch through Orchestrator and created no
  target intervention-debt tickets.
- `scores export` wrote `docs/QUALITY_SCORE.md` with overall grade
  `Insufficient evidence`, which is appropriately conservative for a failed
  implementation lifecycle.

Residual findings:

- Engineer wrote `src/main.go` and `src/main_test.go` with no `MarsDocSync`
  metadata. Manual audit output:

  ```text
  docsync: checked 2 files, findings 2
  FAIL: src/main.go: missing MarsDocSync docs metadata
  FAIL: src/main_test.go: missing MarsDocSync docs metadata
  ```

- After validation passed, Engineer repeatedly called `shell_exec` with empty
  `argv`. Soft no-op guidance returned as a successful tool result, so the
  model spent turns without moving the ticket to done or recording a blocked
  disposition and eventually hit `circle_detected`.
- CTO wasted discovery turns by grepping `docs/tickets/*.md`, which misses the
  lifecycle subdirectories, but still produced the correct single product
  ticket. This remains a lower-priority discovery-efficiency follow-up.

Source action:

- `shell_exec` no-op calls now return a tool error with the same completion
  guidance, making empty `argv` and single `:` calls visibly non-progressing.
- `file_write` now rejects source/test writes under source roots, plus
  root-level source files such as `main.go` or `index.html`, unless content
  includes valid top-of-file `MarsDocSync` docs metadata pointing at existing
  documentation.
- `internal/docsync` now audits root-level source files so direct-root app
  layouts receive the same metadata coverage as `src/` and `cmd/`.

The next API canary should confirm Engineer writes valid DocSync metadata
before source creation, avoids empty-shell completion loops, and reaches ticket
completion or a clear blocked disposition.

### API Rerun After No-Op Hard Error And DocSync Write Preflight (Task Notes API) - 2026-05-20

Target: `<validation-root>`

Binary: `<validation-root>`

Command:

```bash
<validation-root> start \
  --repo <validation-root> \
  --db <validation-root> \
  --log-file <validation-root> \
  --debug
```

Job outcomes:

```text
ceo|completed|1
coo|completed|1
cto-weekly|completed|1
engineer|failed|1
```

Telemetry at stop:

```text
ceo|guardrail_block|4
coo|guardrail_block|1
cto-weekly|guardrail_block|1
engineer|guardrail_block|5
engineer|llm_unreachable|1
```

Positive evidence:

- Product-specific planning and ticketing still reached Engineer with zero
  target intervention-debt tickets.
- `file_write` DocSync preflight did not prevent useful implementation: the
  first attempted source write included a structured `MarsDocSync` block
  pointing at the canonical feature contract.
- Runtime failures and guardrail blocks stayed foundation-owned and did not
  create target intervention-debt tickets.

Residual findings:

- CEO and COO spent extra turns on duplicate feature-contract creation,
  broad discovery, and attempted ticket creation through the CLI/tool boundary.
  They recovered and completed, but this is still factory-pace drag.
- Engineer attempted the correct `git mv` claim command, but emitted `argv` as
  a JSON-encoded array string. `shell_exec` execution can normalize that shape,
  while the claim exception in policy could not, so policy blocked the exact
  command it told Engineer to run.
- The run was stopped after this evidence was captured; the final
  `llm_unreachable` telemetry came from operator shutdown, not from the original
  claim-policy failure.

Source action:

- The Engineer claim exception now decodes `shell_exec` arguments through the
  same normalizing parser as execution, so JSON-string argv drift no longer
  blocks backlog-to-in-progress ticket moves.

The next API canary should confirm Engineer can claim `T-001`, keep the valid
DocSync metadata, and continue to build/test/ticket completion.

### API Rerun After Claim Argv Normalization (Task Notes API) - 2026-05-20

Target: `<validation-root>`

Binary: `<validation-root>`

Command:

```bash
<validation-root> start \
  --repo <validation-root> \
  --db <validation-root> \
  --log-file <validation-root> \
  --debug
```

Job outcomes:

```text
ceo|completed|1
coo|completed|1
cto-weekly|completed|1
engineer|failed|1
```

Telemetry at stop:

```text
ceo|guardrail_block|1
cto-weekly|guardrail_block|1
engineer|circle_detected|1
engineer|guardrail_block|1
```

Positive evidence:

- Product-specific planning and ticketing still reached Engineer with zero
  target intervention-debt tickets.
- CTO created and committed `T-001` for the Task Notes API health endpoint,
  confirming the previous JSON-string argv claim-policy issue was no longer
  blocking the planning-to-implementation handoff.
- Runtime failures stayed foundation-owned; Engineer failure did not dispatch
  through Orchestrator and did not create target intervention-debt tickets.
- `scores export` wrote `docs/QUALITY_SCORE.md` with overall grade
  `Insufficient evidence`, preserving a conservative evidence claim.

Residual findings:

- COO initially emitted `job_disposition_record.evidence_links` as a string
  containing list syntax. The role recovered, but the strict decoder spent two
  avoidable turns.
- Engineer read the backlog ticket and feature contract but did not claim
  `T-001` before shell discovery. A broad `find .` command was blocked, then
  two empty `shell_exec` calls ended the job with `circle_detected`.
- The target had no source implementation; `T-001` remained in backlog.

Source action:

- Engineer `shell_exec` is now claim-first while ordinary product backlog work
  exists and no in-progress ticket is present. The only allowed pre-claim shell
  command is the backlog-to-in-progress `git mv` claim; read-only discovery,
  validation, and no-op shell placeholders receive exact claim guidance.
- `job_disposition_record` now normalizes strict arrays, JSON-string lists,
  Python-style list strings, and single strings for evidence and handoff list
  fields before validation.

The next API canary should confirm Engineer claims `T-001` before shell
discovery, then proceeds to source implementation with DocSync metadata.

### API Rerun After Claim-First Shell Policy (Task Notes API) - 2026-05-20

Target:
`<validation-root>`

Binary:
`<validation-root>`

Command:

```bash
<validation-root> start --repo <validation-root> --db <validation-root> --log-file <validation-root> --debug
```

The first unprivileged start attempt initialized and seeded the target before
the local sandbox rejected the dashboard bind. The escalated rerun processed a
fresh CEO bootstrap job and produced one normal lifecycle path; the bind
restriction is local to the validation environment and is not counted as a
target lifecycle failure.

Job outcomes at stop:

```text
ceo|completed|1
coo|completed|1
cto-weekly|completed|1
engineer|failed|1
```

Telemetry at stop:

```text
cto-weekly|guardrail_block|1
engineer|circle_detected|1
engineer|guardrail_block|5
```

Target commits of interest:

```text
3b187e7 chore(learnings): update runtime learnings for ceo
73334e0 plan: update feature contract and exec plan for F-001-S001 health endpoint
701f2c5 tickets: create implementation ticket T-001 for GET /health endpoint implementation
a814c0a chore(tickets): claim T-001
ceffc12 feat: implement GET /health endpoint with service status, name, and timestamp (T-001 step 1)
45973ec test: add tests for health endpoint (T-001 step 2)
611bf4a chore: capture run18 quality evidence
```

Positive evidence:

- AD-153 is validated in the live path. Engineer first tried `shell_exec`
  `ls -la` before claiming the product ticket; policy blocked it with the
  exact `git mv` claim command, and Engineer immediately claimed and committed
  `T-001`.
- CEO, COO, and CTO reached product-specific planning and a product ticket
  without duplicate bootstrap jobs or intervention-debt ticket amplification.
- Source-write DocSync preflight worked as intended. Engineer recovered by
  writing valid `MarsDocSync` metadata on `main.go` and `main_test.go`.
- `docsync_audit` reported zero findings, `go test ./...` passed, and the
  external validation build `go build -o <validation-root>`
  succeeded.
- Runtime failures stayed foundation-owned. No target intervention-debt ticket
  was created, and the Engineer failure did not dispatch Orchestrator.

Residual findings:

- After passing tests and the external build, Engineer ran `go run main.go` in
  the foreground. The tool timed out after 30 seconds while the server printed
  `Starting server on :8080`.
- Engineer then repeated empty `shell_exec` calls until `circle_detected`.
- The role tried to remove `<validation-root>` with `rm`; the
  destructive shell policy blocked it. This is a minor cleanup-friction finding
  because the artifact was outside the target repo.

Source action:

- `shell_exec` now rejects likely server and watcher commands when run in the
  foreground. HTTP-shaped `go run` commands and common dev-server commands
  must use `background:true`, a separate readiness probe, and tracked PID
  cleanup.

The next API canary should confirm Engineer reaches implementation again,
starts the service in managed background mode, probes `/health`, stops the
tracked PID, and moves `T-001` to done with evidence.

### API Rerun After Foreground Server Preflight (Task Notes API) - 2026-05-20

Target:
`<validation-root>`

Binary:
`<validation-root>`

Command:

```bash
<validation-root> start --repo <validation-root> --db <validation-root> --log-file <validation-root> --debug
```

Job outcomes at stop:

```text
ceo|completed|1
coo|completed|1
cto-weekly|completed|1
engineer|completed|1
qa|completed|1
security|failed|1
```

Telemetry at stop:

```text
coo|guardrail_block|1
cto-weekly|guardrail_block|1
engineer|guardrail_block|4
security|guardrail_block|2
security|max_turns|1
```

Target commits of interest:

```text
f7a08ec chore(learnings): update runtime learnings for ceo
b3b9c8f plan: update active scenario schedule and feature contract for Task Notes API walking skeleton
e7d3ae9 tickets: create implementation ticket T-001 for GET /health endpoint
09c96a3 chore(tickets): claim T-001
2f2f76f feat: implement GET /health endpoint for Task Notes API (T-001)
54384cd chore: update ticket evidence and metadata (T-001)
bd3732c chore(tickets): move T-001 to done
6eebbc2 chore(learnings): update runtime learnings for qa
64014bd chore: capture run19 quality evidence
```

Positive evidence:

- The run reached real product completion for the non-static API canary:
  `T-001` moved to `docs/tickets/done/`, QA approved it, and the target
  worktree was clean after quality-score evidence was committed.
- Engineer again obeyed the claim-first shell guard after an initial `ls`
  attempt, then claimed and committed `T-001`.
- Source and test writes included valid `MarsDocSync` metadata on first write,
  `docsync_audit` reported zero findings, `go test ./...` and `go test -v`
  passed, and the external `<validation-root>` build succeeded.
- Runtime validation used managed background execution: the external binary was
  started with `background:true`, `curl -s http://localhost:8080/health`
  returned JSON with `service`, `status`, and `timestamp`, and the tracked PID
  was killed.
- `scores export` wrote `docs/QUALITY_SCORE.md` with overall grade `C`, one
  done ticket, zero open intervention-debt tickets, and factory-pace rows for
  CEO, COO, CTO, Engineer, QA, and Security.
- The failed Security job stayed foundation-owned: no Orchestrator recovery loop
  and no target intervention-debt ticket.

Residual findings:

- COO still spent turns trying `mars_cli ticket_create` and direct
  ticket `file_write` before recovering to a clean planning handoff. This is
  pace drag, not product starvation.
- Engineer still spent discovery turns on broad `find`, extra `ls`, and an
  initial attempt to run `./task-notes-api` after building the validation binary
  outside the repo. It recovered and completed, so this is now an optimization
  target rather than a blocker.
- Security successfully reached test, docsync, external build, managed
  background start, curl, and kill evidence, but then repeated validation and
  hit `max_turns` before writing a security report or terminal disposition.

Source action:

- Generated Security guidance now has a bounded terminal evidence path:
  inspect changed diff, scan for secrets, read changed code and done ticket,
  run `docsync_audit`, run the smallest relevant test, perform at most one
  managed smoke probe when needed, then write the audit, commit, push if
  possible, and record `job_disposition_record`.

The next API canary should confirm Security writes and commits
`docs/reports/security/security-audit-<date>.md`, records an approved or
changes-requested disposition before `max_turns`, and keeps the target backlog
free of foundation-owned intervention debt.

### API Rerun After Bounded Security Review (Task Notes API) - 2026-05-20

Target:
`<validation-root>`

Binary:
`<validation-root>`

Command:

```bash
<validation-root> start --repo <validation-root> --db <validation-root> --log-file <validation-root> --debug
```

Job outcomes at stop:

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

Telemetry at stop:

```text
cto-weekly|guardrail_block|1
dogfood|guardrail_block|1
engineer|guardrail_block|7
```

Target commits of interest:

```text
d3b14dd chore(learnings): update runtime learnings for ceo
4091625 plan: update active scenario schedule and feature contract for GET /health endpoint
547cf6e tickets: create implementation ticket for current scenario F-001-S002
8e5f386 chore(tickets): claim T-001
c971487 feat: implement GET /health endpoint with service name, status, and timestamp (T-001)
756495f chore(tickets): update T-001 evidence and acceptance criteria
a688616 chore(tickets): move T-001 to done
f01fc55 chore(learnings): update runtime learnings for qa
0bbc493 security: audit 2026-05-20
f0cda20 dogfood: E2E validation findings 2026-05-20
7441faf release: notes 0.2.0
c28174f chore: capture run20 quality evidence
```

Positive evidence:

- The full non-static API lifecycle completed through product planning,
  product ticketing, Engineer implementation, QA, Security, Dogfood, and
  Release Manager without intervention-debt ticket amplification.
- Security validated the run19 source action. It completed in 14 turns: read
  recent commits, scanned for secrets, ran `docsync_audit`, ran
  `go test ./...`, built the validation binary outside the repo, started it
  with `background:true`, curled `/health`, killed the tracked PID, wrote and
  committed `docs/reports/security/security-audit-2026-05-20.md`, and recorded
  terminal disposition.
- Dogfood completed after rechecking test/build/runtime behavior, wrote and
  committed `docs/reports/dogfood/dogfood-validation-2026-05-20.md`, and
  recorded terminal disposition.
- Release Manager generated release notes, committed `release: notes 0.2.0`,
  and created local tag `v0.2.0`. It stopped cleanly with a release publication
  blocker because the disposable target had no `origin` remote.
- `scores export` wrote `docs/QUALITY_SCORE.md` with overall grade `A`, one
  done product ticket, zero open intervention-debt tickets, and Factory Pace
  rows through Release Manager.

Residual findings:

- Engineer still spent seven guardrail blocks before completion. The largest
  repeated sources were pre-claim discovery, broad `find .`, source DocSync
  preflight recovery, in-repo build-output prevention, and malformed shell argv
  recovery while stopping a validation process.
- Dogfood still tried a default `go build ./...` before recovering to
  `go build -o <validation-root> ./...`.
- The static-game and Task Notes API canaries are now productive, but they are
  not enough to claim generic software-factory performance. The next loop
  should add distinct target archetypes and treat repeated cross-target
  guardrail tax as the next optimization input.

Source action:

- No new blocking source change is required from run20. The run validates the
  bounded Security guidance added after run19.
- Record the confirmed improvement and shift the active plan toward a
  representative validation matrix rather than further tuning only the
  Space Invaders or Task Notes API canaries.

## CLI Matrix Replay: Note Stats CLI — 2026-05-20

Purpose: broaden live validation beyond Space Invaders/static web and the Task
Notes HTTP API by testing a small CLI project:
`<validation-root>`.

Brief:

> Build a tiny command-line tool for note analysis. Start with a product
> walking skeleton: a `note-stats` CLI that accepts `--text "some words"` and
> prints JSON with `word_count`, `character_count`, and `line_count`.

Source binary:
`<validation-root>`

Run command:

```bash
<validation-root> start --repo <validation-root> --db <validation-root> --log-file <validation-root> --debug
```

Observed queue state after stop:

```text
ceo|completed|1
coo|completed|1
cto-weekly|completed|1
engineer|completed|1
qa|completed|1
security|failed|1
```

Telemetry:

```text
coo|guardrail_block|1
engineer|guardrail_block|4
security|guardrail_block|1
security|max_turns|1
```

Target commits:

```text
f69f806 chore(learnings): update runtime learnings for qa
dc281ab chore(tickets): move T-001 to done
39281a4 feat: implement note-stats CLI with --text flag and JSON output (T-001)
d36e740 chore(tickets): claim T-001
72678b6 tickets: create implementation ticket for Note Stats CLI walking skeleton [2026-05-19]
798e37e plan: update active scenario schedule and feature contract for Note Stats CLI
a910cbd chore(learnings): update runtime learnings for ceo
7883006 chore(harness): initialize MARS
99b37be chore: seed cli brief
```

Positive findings:

- Product-first flow generalized to a CLI target: planning, feature contract,
  ticketing, implementation, and QA happened without intervention-debt ticket
  amplification.
- Claim-first shell policy blocked pre-claim discovery and guided Engineer to
  `git mv` the backlog ticket into progress before implementation.
- In-repo `go build -o note-stats` was blocked before producing a target-root
  binary, and the agent recovered to `<validation-root>`.
- After the run, `go test ./cmd/note-stats` passed in the target, and
  `scores export` produced `docs/QUALITY_SCORE.md` with overall grade `C` and
  zero open intervention-debt tickets.

Failure findings:

- Engineer created and committed a root `debug.go` scratch probe with valid
  `MarsDocSync` metadata. The file was not part of the product commit, but it
  was later captured in the ticket-lifecycle commit.
- QA approved through read-only inspection, but Security later ran
  `go test ./cmd/note-stats` and found a failing edge case for line counting.
- Security patched `cmd/note-stats/main.go` directly, reran the test
  successfully, then hit `max_turns` before committing, writing
  `docs/reports/security/security-audit-2026-05-20.md`, or recording
  `job_disposition_record`.
- The target ended dirty with:

```text
 M cmd/note-stats/main.go
 M docs/QUALITY_SCORE.md
```

Assessment:

The CLI target confirms the factory is no longer overfitted to a single game or
HTTP service. The product path is generic enough to plan and implement a small
command-line application. The next foundation fix is also generic: Security
must report functional/product remediation back to Engineer instead of
opportunistically patching product files during review, and root scratch probes
must be blocked even when they contain valid DocSync metadata.

## CLI Matrix Replay: Note Stats CLI Rework Bounds — 2026-05-20

Command:

```bash
<validation-root> start --repo <validation-root> --db <validation-root> --log-file <validation-root> --debug
```

Observed queue state after stop:

```text
ceo|completed|1
coo|completed|1
cto-weekly|completed|1
engineer|completed|2
engineer|failed|1
orchestrator|completed|2
qa|completed|2
security|completed|2
```

Telemetry:

```text
cto-weekly|guardrail_block|1
engineer|guardrail_block|12
engineer|max_turns|1
security|guardrail_block|5
```

Target commits:

```text
b447222 feat(cmd/note-stats): improve flag validation to properly detect when --text flag is provided without value (T-001 step 1)
7834190 security: audit 2026-05-20
8a4463e feat(cmd/note-stats): improve flag validation to require non-empty text value (T-001)
2aff059 security: audit 2026-05-20
5cff99b chore(learnings): update runtime learnings for qa
7ad9139 chore(tickets): move T-001 to done
e28d70f chore(tickets): claim T-001
23d5958 tickets: create implementation ticket T-001 for current scenario F-001-S002
3f7b9cf chore(learnings): update runtime learnings for coo
de5266b plan: update active scenario schedule and feature contract for note-stats CLI
e3927b6 chore(learnings): update runtime learnings for ceo
35788dc chore(harness): initialize MARS
a8ed8a1 chore: seed cli brief
```

Positive findings:

- The AD-157 Security authority boundary worked. Security wrote and committed
  only `docs/reports/security/security-audit-2026-05-20.md`; it did not patch
  product, test, ticket, or feature files.
- The generalized root scratch-probe block worked. No `debug.go` or equivalent
  root probe was added to the target.
- Product progress remained intact. The target had one done product ticket,
  `go test ./cmd/note-stats` passed, and the worktree was clean before score
  export.
- Runtime containment held. Engineer's final `max_turns` failure stayed in
  foundation telemetry, did not dispatch Orchestrator, and created no target
  intervention-debt ticket.
- `scores export` produced overall grade `B`, one done product ticket, and
  zero open intervention-debt tickets.

Failure findings:

- Security still converted a speculative or already-safe CLI flag concern into
  `NEEDS_REMEDIATION`. It had observed missing and empty `--text` inputs
  failing safely before asking Engineer for rework.
- Engineer handled the review handoff as broad implementation rework. It made a
  small flag-validation patch and committed it, but then kept running extra
  smoke and newline probes until `max_turns` before recording a disposition.
- CLI convention detection remains web-biased: the run still detected
  `start_command: go run ./cmd/...` and `dev_port: 8080` for a non-server CLI.

Assessment:

The CLI replay confirms the previous source fix improved the live experience,
but also shows that reviewer accuracy and review-rework bounds now dominate
factory pace. The next generic source fix should make Security block only on
current failing or exploitable evidence, and make Engineer close
`changes_requested` handoffs after the exact requested evidence is proven
instead of expanding into unrelated validation.

## CLI Matrix Replay: Note Stats CLI Contract Bounds — 2026-05-20

Command:

```bash
<validation-root> start --repo <validation-root> --db <validation-root> --log-file <validation-root> --debug
```

Observed queue state after stop:

```text
ceo|completed|1
coo|completed|1
cto-weekly|completed|1
engineer|failed|1
```

Telemetry:

```text
ceo|guardrail_block|1
coo|guardrail_block|1
engineer|guardrail_block|7
engineer|max_turns|1
```

Target commits:

```text
bf53fc8 chore(tickets): claim T-001
3c68047 tickets: create implementation ticket T-001 for note-stats CLI tool implementation
dfa1cfa plan: update active scenario schedule and feature contract for note-stats CLI
75b2c32 chore(learnings): update runtime learnings for ceo
ed38528 chore(harness): initialize MARS
a335959 chore: seed cli brief
```

Positive findings:

- The generated target included the new Security evidence-grounding guidance
  and Engineer bounded review-rework guidance from AD-158.
- CEO, COO, and CTO reached product-specific planning, feature-contract
  updates, and ordinary product ticketing for the CLI target.
- Engineer respected the claim-first shell boundary and claimed `T-001` before
  product mutation.
- Root scratch-probe prevention held again: no `debug.go`, `probe.go`, or
  equivalent root scratch file was created.
- Runtime containment held. The Engineer `max_turns` failure stayed
  foundation-owned telemetry, did not route through Orchestrator, and created
  zero target intervention-debt tickets.

Failure findings:

- Engineer never reached QA or Security. It spent 50 turns in initial
  implementation and hit `max_turns` with uncommitted `cmd/` and `go.mod`
  files.
- The ticket explicitly required empty `--text ""` to produce zero words,
  zero characters, and zero lines. Engineer drifted to a one-line empty-text
  interpretation and rewrote tests around implementation behavior instead of
  the ticket and BDD contract.
- Manual verification after the stop showed `go test ./cmd/note-stats` failed
  with `Expected {2 17 2}, got {2 18 2}`.

Assessment:

The third CLI replay validates the Security authority and scratch-probe fixes
but moves the bottleneck earlier in the lifecycle: initial implementation can
still expand into exploratory edge-case interpretation before the selected
ticket is complete. The next generic source fix is contract-first Engineer
implementation. The selected ticket acceptance criteria and BDD feature
contract must remain the product contract for the run; useful edge cases beyond
that contract become follow-up evidence instead of same-run rewrites.

## CLI Matrix Replay: Note Stats CLI Closure Bounds — 2026-05-20

Command:

```bash
<validation-root> start --repo <validation-root> --db <validation-root> --log-file <validation-root> --debug
```

Observed queue state after stop:

```text
ceo|completed|1
coo|completed|1
cto-weekly|completed|1
engineer|failed|1
```

Telemetry:

```text
ceo|guardrail_block|4
coo|guardrail_block|1
cto-weekly|guardrail_block|1
engineer|guardrail_block|12
engineer|max_turns|1
```

Target commits:

```text
8c11a5f feat: implement note-stats CLI tool with --text flag that outputs JSON counts (T-001)
2c2764c chore(tickets): claim T-001
77b1be8 tickets: create implementation ticket for current scenario F-001-S001
8a237e5 plan: update active scenario schedule and feature contract for note-stats CLI tool
bf00c80 chore(learnings): update runtime learnings for ceo
eb57d96 CEO: Updated active goals for Demo CLI Run 4 project
f66f4bb chore(harness): initialize MARS
223335c chore: seed cli brief
```

Positive findings:

- The contract-first fix improved the exact `demo-cli-run3` failure. Engineer
  honored the README/ticket requirement for empty text:
  `--text ""` returned `{"word_count":0,"character_count":0,"line_count":0}`.
- Engineer also proved `hello world`, multiline text, and
  `go test ./cmd/note-stats`, then updated the ticket evidence and committed
  the product implementation.
- Root scratch-probe prevention still held, and no target intervention-debt
  tickets were created.
- Runtime containment held again: the Engineer `max_turns` failure did not
  route through Orchestrator.

Failure findings:

- Engineer did not move `T-001` to done or record terminal disposition after
  required acceptance evidence passed and the product commit existed.
- It continued into packaging/build-output work that was not required by the
  feature ticket. An argv `go build -o bin/note-stats` was blocked before
  creation, but the shell-wrapped command
  `mkdir -p bin && go build -o bin/note-stats ...` created an untracked
  `bin/` artifact before blast-radius validation stopped the tool.
- The target ended with an untracked `bin/` directory and no QA/Security
  handoff.

Assessment:

The contract-first implementation rule is effective for the selected product
semantics, but closure still needs to be stronger after success. The next
source fix should make ticket closure precede packaging/distribution
exploration, and the `shell_exec` build-output policy should inspect shell
command segments so repo-local build artifacts are blocked before they enter
the target.

### CLI Matrix Replay: Dogfood Finding Handoff Bounds — 2026-05-20

Purpose: validate the closure-before-packaging fix on a fresh CLI target and
observe whether the lifecycle reaches product validation without the
post-success build-output trap.

Target:

- Repo: `<validation-root>`
- Binary: `<validation-root>`
- Command:
  `<validation-root> start --repo <validation-root> --db <validation-root> --log-file <validation-root> --debug`
- Brief: Note Stats CLI where `--text ""` returns zero words, characters, and
  lines; the first slice explicitly does not require packaged binaries or
  install artifacts.

Observed queue state:

```text
security|completed|1
coo|completed|1
cto-weekly|completed|1
dogfood|failed|1
qa|completed|1
engineer|completed|1
ceo|completed|1
```

Telemetry:

```text
coo|guardrail_block|1
dogfood|guardrail_block|5
dogfood|max_turns|1
engineer|guardrail_block|5
security|guardrail_block|1
```

Target commits before Dogfood failure:

```text
e9a41d2 security: audit 2026-05-20
28d24cf chore(learnings): update runtime learnings for qa
4fdd999 chore(tickets): move T-001 to done
f6ea6b1 chore(tickets): claim T-001
363d374 tickets: create implementation ticket T-001 for F-001-S001 scenario
93d98fd chore(learnings): update runtime learnings for coo
4bd222b plan: update active scenario schedule and feature contract for note-stats CLI
8f7d95a chore(learnings): update runtime learnings for ceo
```

Positive findings:

- The closure-before-packaging fix held in the live path. Engineer moved
  `T-001` to done, recorded disposition, and did not create repo-local
  packaging output before QA.
- QA approved the completed ticket, and Security wrote a bounded audit report.
- Dogfood found a valid target-owned product gap rather than a harness/runtime
  issue: running the CLI without `--text` returns zero-count JSON instead of a
  required-argument error.
- Runtime containment still held: Dogfood `max_turns` remained foundation
  telemetry and did not create target intervention-debt tickets.

Failure findings:

- Dogfood created two uncommitted tickets for the same missing-argument issue:
  `T-002-dogfood-pre-flight-cli-tool-missing-proper-argument-validati.md` and
  `T-003-dogfood-cli-tool-does-not-properly-validate-required-text-ar.md`.
- Dogfood continued shell validation after creating the first target-owned
  finding instead of committing the ticket and recording
  `changes_requested`.
- The target ended dirty with untracked backlog tickets and no terminal
  Dogfood handoff for Engineer.

Assessment:

The source lifecycle has moved past planning, implementation, QA, Security, and
post-success packaging drift for this CLI archetype. The remaining failure is a
Dogfood handoff boundary: target-owned findings must become committed,
claimable tickets before validation continues. The next source fix should block
Dogfood shell validation and additional `ticket_create` calls while a finding
ticket is uncommitted, then rerun the CLI canary to confirm one clean finding
handoff or a release-ready pass.

### CLI Matrix Replay: Review Validation Gates And Remediation Closure

The `demo-cli-run6` replay used
`<validation-root>` with the same Note Stats
CLI brief after the Dogfood handoff fix.

Queue summary:

```text
ceo|completed|1
coo|completed|1
cto-weekly|completed|1
engineer|completed|1
qa|completed|1
security|completed|1
dogfood|completed|1
orchestrator|completed|1
engineer|failed|1
```

Telemetry summary:

```text
ceo|guardrail_block|2
security|guardrail_block|3
dogfood|guardrail_block|7
engineer|guardrail_block|17
engineer|max_turns|1
```

Target commits:

```text
125d72a chore(tickets): close T-002 after max-turn evidence
fba6deb fix(tests): correct test expectation for tab-separated text in note-stats CLI tool (T-002)
7b3b2f8 chore(tickets): claim T-002
71d3138 dogfood: E2E validation findings [2024-05-20]
b32fced security: audit 2026-05-20
85ddf2c chore(learnings): update runtime learnings for qa
d9e0379 chore(tickets): move T-001 to done
5b9832b chore(tickets): claim T-001
```

Positive findings:

- The bootstrap idempotency key prevented a duplicate CEO job after the first
  bind failure retry.
- The lifecycle reached product-specific planning, ticketing, implementation,
  QA, Security, Dogfood, Orchestrator, and Engineer rework with zero target
  intervention-debt tickets.
- Dogfood created exactly one target-owned finding for the failing Note Stats
  test expectation. The new uncommitted-finding guard blocked further
  validation until the ticket was committed.
- Orchestrator routed Dogfood `changes_requested` back to Engineer. Engineer
  claimed the finding, fixed the test expectation, and `go test ./...` passed.
- The final Engineer `max_turns` failure stayed foundation telemetry and did
  not route through Orchestrator or create target debt.

Failure findings:

- QA approved `T-001` without running `go test ./...` even though
  `cmd/note-stats/main_test.go` existed.
- Security ran `go test ./...`, observed a failing unit test, and still
  recorded an approved disposition. Dogfood caught the same defect later.
- After Dogfood committed the finding, it resumed validation before terminal
  disposition; the guard only covered the uncommitted state.
- `shell_exec` argv mode rejected a literal newline argument, forcing shell
  fallback and extra guardrail turns for multiline CLI validation.
- Engineer fixed `T-002` but hit `max_turns` while closing the remediation
  ticket because the done-move policy treated an evidenced enabler ticket as if
  it had to become a feature ticket with BDD metadata.

Assessment:

This run is a good factory-loop outcome: the product progressed, Dogfood found
a real defect, Orchestrator routed rework, Engineer fixed it, tests passed, and
runtime failure containment held. The remaining source work is now narrow and
mechanical: make QA/Security approval depend on live validation evidence,
strengthen Dogfood finding creation into a same-run terminal handoff boundary,
allow literal newline argv arguments, and let evidenced enabler/remediation
tickets close without inventing feature metadata.

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
validation. Run14 confirms no-op shell guidance lets Engineer complete the
ticket lifecycle and shifts the next generic bottleneck to mechanical DocSync
approval gates plus non-retryable classification for policy-blocked timeout
wrappers. Run15 confirms runtime failures remain quarantined without
intervention-debt tickets, but shows source DocSync metadata must be enforced
at write time and no-op shell guidance must fail the tool call to prevent
circle detection. Run16 confirms valid DocSync metadata can appear on the first
source write, but shows shell policy must use the same argv normalization as
execution so JSON-string `git mv` ticket-claim commands satisfy the claim gate.
Run17 confirms the claim-argv normalization path reaches Engineer, but shows
Engineer shell work still needs a mechanical claim-first boundary and
disposition evidence-list normalization. Run18 confirms the claim-first shell
boundary works in the live path, source-write DocSync preflight can recover
without human intervention, and API implementation/testing can make material
product progress without intervention-debt tickets; the next generic
bottleneck is foreground server validation after tests/builds pass. The
Run19 canary confirms managed background validation and ticket completion for
the non-static API path, plus QA approval and a target quality-score export.
The next generic bottleneck is bounded Security terminal evidence: Security can
validate the work, but still needs to stop after one sufficient proof, write
the audit, and record disposition before max turns. Run20 confirms bounded
Security terminal evidence, Dogfood validation, and local target release notes
all complete in the non-static API path with overall quality grade `A` and
zero target intervention-debt tickets. The Note Stats CLI matrix replay confirms
the planning, ticketing, implementation, and QA path is not specific to static
web or HTTP services, while exposing Security product-mutation and root scratch
probe gaps. The second CLI replay validates those authority and scratch-probe
fixes, while exposing the next generic bottleneck: Security false positives and
unbounded Engineer review rework can still turn a working product slice into a
max-turn failure. The third CLI replay validates the patched target guidance
and root scratch-probe block, while exposing a more basic Engineer boundary:
initial implementation must obey the selected ticket and BDD contract instead
of rewriting tests around exploratory semantics. The fourth CLI replay confirms
that contract-first implementation works for the selected semantics and moves
the bottleneck to post-success closure: Engineer must stop after acceptance
evidence, move the ticket to done, and avoid repo-local packaging exploration
unless it is explicitly in scope. The fifth CLI replay confirms that closure
before packaging now reaches QA, bounded Security, and Dogfood, then moves the
bottleneck to Dogfood finding handoff: target-owned findings must be committed
and dispositioned before further validation or duplicate ticket creation. The
sixth CLI replay confirms that Dogfood handoff now creates one committed
finding and routes rework, but exposes mechanical review-approval and
remediation-closure gaps: QA/Security must be unable to approve without passing
validation evidence, Dogfood must stop after a finding until disposition, argv
must allow literal multiline arguments, and evidenced enabler tickets must move
to done without feature metadata churn. The seventh CLI replay deliberately
restarted from a fresh generic Note Stats target and confirmed CEO/COO planning
remained product-specific with zero target intervention-debt tickets, but CTO
stalled because COO wrote `F-002-SNNN` scenario headings inside the `F-001`
feature contract. The eighth CLI replay confirmed scenario/file alignment and
post-validation convergence enough to reach QA review and real test failure
detection, then exposed rework ticket-state drift: QA rejection routed Engineer
back to a ticket still marked done. The remaining live-loop work is now to
rerun the CLI canary after rework-ticket reopening is enforced, validate
multiple software archetypes before making generic lifecycle claims, and
optimize repeated guardrail tax only when it appears across that representative
matrix.

## Note Stats CLI Run 8: Scenario Alignment Fixed, Post-Commit Convergence Failed

**Target:** `<validation-root>`
**Database:** `<validation-root>`
**Binary:** `<validation-root>`
**Date:** 2026-05-21

### Outcome

Run 8 confirmed the run 7 source fix. CEO and COO stayed product-specific,
COO updated `docs/features/F-001-product-walking-skeleton.md` with aligned
`F-001` scenario IDs, and CTO created `T-001` instead of stalling on a missing
`F-002` contract. Engineer claimed the ticket, implemented the Go CLI, and
validated the requested behavior:

- `go run ./cmd/note-stats --text "hello world"` returned
  `{"word_count":2,"character_count":11,"line_count":1}`.
- empty `--text` returned zero counts.
- multiline input returned three lines.
- missing `--text` exited non-zero with `error: --text flag is required`.
- `go test ./...` and
  `go build -o <validation-root> ./cmd/note-stats` passed.

The run still failed after useful product work existed. Engineer committed the
implementation but did not move `T-001` to `docs/tickets/done/` or call
`job_disposition_record`. It spent additional turns on `ls`, `find .`,
malformed `shell_exec`, `/tmp` listing, and extra validation probes until the
model request exceeded the 32k context window at about 41k prompt tokens.

### Runtime Signals

- Jobs: CEO completed, COO completed, CTO completed, Engineer failed.
- Telemetry: one `context_overflow`; ten `guardrail_block` events.
- Target intervention-debt tickets: zero.
- Containment behavior: correct. The context overflow was foundation telemetry,
  and the server did not route the runtime failure through Orchestrator or
  create a target backlog ticket.

### Decision From Run 8

AD-164 adds a post-validation Engineer convergence gate. After successful
validation and an implementation commit, if an ordinary product ticket remains
in `docs/tickets/in-progress/`, further exploratory `shell_exec` calls are
blocked. The allowed shell path is the ticket lifecycle move to `done/`, then a
commit and `job_disposition_record` handoff to QA.

The agent loop context pruner also now removes old assistant tool-call
arguments and older prose, not just old tool results, so historical `file_write`
payloads and shell calls cannot keep the prompt above provider limits after
the output bodies were pruned.

### Next Check

Run the CLI canary again. Expected improvement: Engineer moves `T-001` to
`done/`, records a QA handoff, and the lifecycle reaches review without context
overflow or target intervention-debt amplification.

## Note Stats CLI Run 9: QA Review Caught Missing Tests, Rework Needed Ticket Reopen

**Target:** `<validation-root>`
**Database:** `<validation-root>`
**Binary:** `<validation-root>`
**Date:** 2026-05-21

### Outcome

Run 9 confirmed the run 8 convergence fix moved the lifecycle forward. CEO,
COO, and CTO completed, Engineer claimed `T-001`, implemented the Note Stats
CLI, moved the ticket to `docs/tickets/done/`, and recorded a QA handoff.

QA then exposed a real evidence gap. It attempted to approve after inspection,
but `job_disposition_record` blocked approval because test files existed and
QA had not run an authoritative test command. QA recovered by recording
`changes_requested` with `next_need: implementation_rework`, and Orchestrator
routed the same ticket back to Engineer. No target intervention-debt ticket was
created; policy blocks remained foundation telemetry.

The rework Engineer pass ran `go test ./...`, found failing test expectations,
patched `main_test.go`, reran `go test ./...` and `go test -v` to passing, and
verified CLI behavior with `go run main.go --text ...`. The operator stopped
the run at that point to patch the source lifecycle rules.

### Runtime Signals

- Jobs: CEO completed, COO completed, CTO completed, Engineer completed, QA
  completed with changes requested, Orchestrator completed, second Engineer
  was manually stopped after passing rework tests.
- Target commits before stop included:
  `1bb96f9 feat: implement note-stats CLI tool with --text flag and JSON output (T-001)`,
  `30ce87d chore(tickets): move T-001 to done`, and
  `41af758 chore(learnings): update runtime learnings for qa`.
- Telemetry: 15 guardrail blocks and one manual-stop `llm_unreachable` event.
- Target intervention-debt tickets: zero.

### Finding

The first version of the AD-164 post-validation shell gate fired too early
because it counted the ticket-claim commit plus a later build success as enough
to stop additional validation while product files were still dirty. QA caught
the missing tests, but Engineer should be allowed to keep validating while the
implementation tree is dirty.

QA rework also revealed ticket-state drift. The rejected ticket remained in
`docs/tickets/done/`, so the second Engineer run could mutate tests and run
validation while the lifecycle still claimed the ticket was complete. That
state is misleading for reviewers, queue ordering, and future autonomy.

### Decision From Run 9

AD-165 makes review rework reopen done or in-review product tickets before
Engineer can mutate product files or run validation shell commands. The
post-validation shell gate now only blocks exploratory shell calls when the
worktree is clean after successful validation and an implementation commit.

### Next Check

Run the CLI canary again. Expected improvement: if QA requests rework after a
done ticket, Engineer first moves the ticket back to
`docs/tickets/in-progress/`, commits the rework claim, performs the bounded
fix, reruns tests, moves the ticket back to done with evidence, and hands off
to QA without hidden lifecycle drift.

## Note Stats CLI Run 10: Completion Commit Needed A Rework-Guard Exception

**Target:** `<validation-root>`
**Database:** `<validation-root>`
**Binary:** `<validation-root>`
**Date:** 2026-05-21

### Outcome

Run 10 confirmed the product-first lifecycle remained healthy after the review
rework policy changes. CEO completed product-specific planning for the Note
Stats brief. COO updated the active plan and feature contract. CTO created
ordinary product ticket `T-001` through `ticket_create`, hit the clean-tree
handoff guard, recovered by committing the ticket, and handed to Engineer.

Engineer followed the visible ticket lifecycle: it was blocked from shell work
before claiming `T-001`, moved the ticket into `docs/tickets/in-progress/`,
committed the claim, implemented the Go CLI, wrote tests, passed
`go test ./cmd/note-stats`, ran CLI evidence, and passed `docsync_audit`.
The run created zero target intervention-debt tickets.

### Finding

The first review-rework guard over-applied during ordinary completion. After
Engineer moved `T-001` from `docs/tickets/in-progress/` to
`docs/tickets/done/`, `git_commit` saw a done ticket plus product changes and
required reopening the ticket for rework. Engineer bounced between moving the
ticket to done and reopening it until `max_turns`:

```text
ceo|completed|1
coo|completed|1
cto-weekly|completed|1
engineer|failed|1
```

Telemetry stayed quarantined as foundation evidence:

```text
guardrail_block|11
max_turns|1
```

### Decision From Run 10

The rework policy now treats a pending ticket move from
`docs/tickets/in-progress/` to `docs/tickets/done/` as normal completion.
`git_commit` remains blocked for product mutation against tickets that already
live in `docs/tickets/done/` or `docs/tickets/in-review/` with no active
completion move.

### Next Check

Run the CLI canary again. Expected improvement: Engineer completes the same
Note Stats slice, commits product files plus the move to `docs/tickets/done/`,
records a QA handoff, and reaches QA without a `max_turns` loop or target
intervention-debt amplification.

## Note Stats CLI Run 11: Repeated No-Op Shell Calls Stopped Product Commit

**Target:** `<validation-root>`
**Database:** `<validation-root>`
**Binary:** `<validation-root>`
**Date:** 2026-05-21

### Outcome

Run 11 stayed on the product path but found the next convergence gap. CEO and
COO completed product-specific planning. CTO took one extra same-role
`ticket_shaping` pass, then created `T-001`, committed it, and handed to
Engineer. Engineer claimed `T-001`, wrote `main.go` and `go.mod`, built the
binary to `<validation-root>`, and ran successful CLI probes.

The job failed before commit and ticket completion. After validation evidence
existed and the target worktree was dirty, Engineer repeatedly called
`shell_exec` with empty `argv` placeholders. The tool returned no-op completion
guidance, but the model retried until the agent loop stopped with
`circle_detected`.

### Runtime Signals

```text
ceo|completed|1
coo|completed|1
cto-weekly|completed|2
engineer|failed|1
```

```text
circle_detected|1
guardrail_block|4
```

Target state after the bounded stop:

```text
 M README.md
?? go.mod
?? main.go
```

No target intervention-debt tickets were created.

### Decision From Run 11

Repeated no-op `shell_exec` calls after successful validation and dirty
in-progress ticket work are now a policy boundary. The first no-op still
returns generic completion guidance; a repeated Engineer no-op in this state
is blocked before execution with direct instructions to inspect status, commit
dirty work, update evidence, complete the ticket lifecycle, and record terminal
disposition.

### Next Check

Run the CLI canary again. Expected improvement: after the first no-op guidance
or repeated-no-op policy block, Engineer commits the dirty CLI implementation,
updates evidence, moves `T-001` to `docs/tickets/done/`, commits the lifecycle
move, and reaches QA handoff.

## Note Stats CLI Run 12: QA Validation Surface And Fresh Artifact Proof

**Target:** `<validation-root>`
**Database:** `<validation-root>`
**Binary:** `<validation-root>`
**Date:** 2026-05-21

### Outcome

Run 12 confirmed the previous convergence fixes moved the lifecycle past
Engineer and into review. CEO, COO, and CTO completed the product-specific
plan, BDD contract, and ordinary product ticket. Engineer claimed `T-001`,
implemented the Note Stats CLI, ran validation, moved the ticket to
`docs/tickets/done/`, committed the lifecycle, and recorded a QA handoff.
Target intervention-debt tickets remained at zero.

Runtime state at the bounded stop:

```text
ceo|completed|1
coo|completed|1
cto-weekly|completed|1
engineer|completed|1
engineer|failed|1
orchestrator|completed|1
qa|completed|1
```

```text
guardrail_block|7
llm_unreachable|1
```

The stopped second Engineer job is the manual stop after observing enough
evidence for the next patch, not an autonomous product failure class.

### Findings

QA could not approve because the generated QA manifest required successful
in-job validation evidence but did not include `shell_exec`. QA therefore
requested changes instead of fabricating approval. That was the correct local
behavior, but the role/tool surface contradicted the review policy.

The rework pass also showed that `<validation-root>` binaries need freshness
proof. Engineer executed `<validation-root>` before rebuilding it in
the current role session, which could have reused a stale binary from an older
canary. After Engineer did rebuild with
`go build -o <validation-root>`, the post-validation convergence
gate blocked running the freshly built binary because it looked like extra
shell exploration.

### Decision From Run 12

AD-167 gives QA bounded `shell_exec` access for validation commands, while
keeping QA read-only for repo writes by default. `shell_exec` now records
external validation binaries produced by successful same-session
`go build -o <validation-root> ...` commands. Running a `<validation-root>`
binary is blocked unless the same role session built it first; once fresh, it
counts as validation evidence and is allowed through Engineer's convergence
gate.

### Next Check

Run the CLI canary again. Expected improvement: QA runs its own authoritative
validation command, approves with evidence, and routes to Security or the next
release/dogfood step without unnecessary Engineer rework or stale `/tmp`
artifact evidence.

## Note Stats CLI Run 13: Direct Runtime Probe Classification

**Target:** `<validation-root>`
**Database:** `<validation-root>`
**Binary:** `<validation-root>`
**Date:** 2026-05-21

### Outcome

Run 13 preserved the product-first path and intervention-debt quarantine. CEO,
COO, and CTO produced product-specific Note Stats planning, a feature contract,
and `T-001` without materializing target intervention-debt tickets. Engineer
claimed the ticket, wrote `cmd/note-stats/main.go` plus `go.mod`, ran a direct
runtime proof, and committed product code.

Runtime state at the bounded stop:

```text
ceo|completed|1
coo|completed|1
cto-weekly|completed|1
engineer|failed|1
```

```text
circle_detected|1
guardrail_block|4
```

Target commits reached:

```text
ca79d5d feat: implement note-stats CLI tool with --text flag (T-001)
c8b0407 chore(tickets): claim T-001
08741d0 tickets: create implementation ticket for current scenario F-001-S002
18f8449 plan: update active scenario schedule and feature contract for Note Stats CLI tool
```

No target intervention-debt tickets were created. The product code was useful,
but `T-001` remained in `docs/tickets/in-progress/`.

### Finding

Engineer successfully ran:

```text
go run cmd/note-stats/main.go --text "hello world"
```

and received:

```text
{"word_count":2,"character_count":11,"line_count":1}
```

That is valid end-to-end product behavior for the CLI slice, but the runtime
only classified tests, builds, and same-session `<validation-root>` binaries
as validation. The successful direct runtime probe therefore did not increment
the validation evidence counters. After the implementation commit, later
empty `shell_exec` calls received generic no-op guidance instead of the
post-validation completion gate, and the role stopped with `circle_detected`.

### Decision From Run 13

AD-168 classifies successful direct runtime commands as validation evidence
when they execute ticket behavior. The generated Engineer guidance now states
that successful runtime probes should lead to ticket evidence, a
`docs/tickets/done/` lifecycle move, and `job_disposition_record` instead of
placeholder shell waits.

### Next Check

Run the CLI canary again. Expected improvement: after the `go run` proof and
implementation commit, Engineer updates ticket evidence, moves `T-001` to
done, commits the lifecycle move, and hands off to QA without a no-op
`circle_detected` failure.

## Note Stats CLI Run 14: Expected Runtime Error Probes Should Not Poison QA Approval

**Target:** `<validation-root>`
**Database:** `<validation-root>`
**Binary:** `<validation-root>`
**Date:** 2026-05-21

### Outcome

Run 14 confirmed the direct-runtime evidence fix moved the CLI lifecycle
forward again. CEO, COO, and CTO completed product-specific Note Stats
planning, feature-contract work, and ordinary product ticket creation with no
target intervention-debt tickets. Engineer claimed `T-001`, implemented the
Go CLI and tests, handled the repo-local build-output guard by building an
external `<validation-root>` binary, ran positive and negative CLI
probes, ran `go test`, committed product code, moved the ticket to
`docs/tickets/done/`, committed the lifecycle move, and recorded a QA handoff.

Runtime state at the bounded stop:

```text
ceo|completed|1
coo|completed|1
cto-weekly|completed|1
engineer|completed|1
qa|failed|1
```

```text
guardrail_block|8
max_turns|1
```

Target commits included:

```text
df34476 chore(tickets): move T-001 to done
69f85d5 feat: implement note-stats CLI tool with --text flag (T-001)
3d95915 chore(tickets): claim T-001
2aa7ae8 tickets: create implementation ticket for note-stats CLI tool
8c97329 plan: update active scenario schedule and feature contract for note-stats CLI tool
8389973 CEO: Define first product slice for note-stats and update goals/observations
```

The target worktree was clean after the bounded stop, and the ticket lived in
`docs/tickets/done/`.

### Finding

QA used the new bounded shell validation surface but hit a classification
edge. It attempted `go build ./cmd/note-stats`, received the expected
repo-local build-output guardrail, recovered with:

```text
go build -o <validation-root> ./cmd/note-stats
```

It then ran positive CLI probes, an expected missing-argument probe, and
`go test`. The missing-argument probe exited non-zero because the CLI correctly
rejected invalid input. That was useful negative-path evidence, not a product
failure. Review approval policy still saw the non-zero runtime probe as a
failing validation command and blocked `approved` dispositions twice. QA then
hit `max_turns`.

### Decision From Run 14

AD-169 separates failing build/test commands from expected runtime error
probes. QA and Security approval remains blocked after failed builds or tests,
and tests are still required when test files exist. A documented non-zero
runtime probe can be evidence for an expected error path when it is paired
with positive runtime evidence and passing authoritative tests.

### Next Check

Run the CLI canary again. Expected improvement: QA approves after positive
probes, the expected missing-argument rejection, and passing tests, then routes
to Security without creating target intervention-debt tickets or false rework.

## Note Stats CLI Run 15: Post-Validation Gate Needed A Non-Shell Next Step

**Target:** `<validation-root>`
**Database:** `<validation-root>`
**Binary:** `<validation-root>`
**Date:** 2026-05-21

### Outcome

Run 15 did not reach QA, so it did not yet live-validate AD-169. It did confirm
the product-first lifecycle remained intact before the next bottleneck:

```text
ceo|completed|1
coo|completed|1
cto-weekly|completed|1
engineer|failed|1
```

Telemetry remained foundation-owned:

```text
circle_detected|1
guardrail_block|11
```

There were still zero target intervention-debt tickets. The target worktree was
clean at stop. The ticket remained in progress:

```text
docs/tickets/in-progress/T-001-implement-first-visible-product-slice-for-note-stats-cli-too.md
```

Target commits included:

```text
a9662f5 fix(cli): remove unused import in main.go (T-001)
abde046 feat(cli): implement first visible product slice for Note Stats CLI tool (T-001)
d5787f4 chore(tickets): claim T-001
8ec5dcb tickets: create implementation ticket T-001 for current failing scenario F-001-S002
```

### Finding

Engineer implemented useful product code, then validated it:

- repo-local build output was blocked correctly;
- `<validation-root>` build first failed on an unused import;
- Engineer fixed and committed the import cleanup;
- the external build then passed;
- `<validation-root> --text "Hello world"` returned the expected JSON.

After that, the post-validation completion gate correctly blocked further
exploratory `shell_exec` because the implementation was committed, validation
had passed, and `T-001` was still in `docs/tickets/in-progress/`. The model
nevertheless retried empty `shell_exec` calls and hit `circle_detected` instead
of reading/updating the ticket evidence and moving it to done.

### Decision From Run 15

AD-170 tightens the post-validation completion message and generated Engineer
guidance. The recovery path now explicitly says the next tool must be
`file_read` on the in-progress ticket followed by `file_write` on the same
ticket to populate `evidence_links` and `verified_by`. `shell_exec` is named as
unavailable except for the exact final `git mv` into `docs/tickets/done/`.

### Next Check

Run the CLI canary again. Expected improvement: after product validation,
Engineer updates the ticket evidence, moves `T-001` to done, records a QA
handoff, and reaches the QA review path where AD-169 can be validated.

## Note Stats CLI Run 16: Unexpected Runtime Failure Slipped Past QA

**Target:** `<validation-root>`
**Database:** `<validation-root>`
**Binary:** `<validation-root>`
**Date:** 2026-05-21

### Outcome

Run 16 validated the product-first lifecycle through QA and Security before the
operator stopped Dogfood to patch the next source issue:

```text
ceo|completed|1
coo|completed|1
cto-weekly|completed|1
dogfood|failed|1
engineer|completed|1
qa|completed|1
security|completed|1
```

The Dogfood failure was operator-induced shutdown during validation, not the
primary finding. The target worktree was clean. Target commits included:

```text
85089e0 security: audit 2026-05-21
2809bcc chore(learnings): update runtime learnings for qa
39368ee chore(tickets): move T-001 to done
b0f0da7 chore(tickets): claim T-001
401a312 tickets: create implementation tickets for current scenario
5e199cb plan: update active scenario schedule and feature contract for F-001
8413800 chore(learnings): update runtime learnings for ceo
```

### Findings

The previous loop fix worked: after validation and a clean implementation
commit, Engineer used `file_write` on the in-progress ticket, moved it to
`docs/tickets/done/`, committed that lifecycle move, and recorded the QA
handoff. That resolves the AD-170 shell-loop failure.

The next issue was a quality escape. The target brief required empty text to
produce zero counts, but the implementation treated empty `--text` as missing:

```text
go run . --text ''
error: --text flag is required
exit status 1
```

QA saw the same failure through `<validation-root> --text ""`, then
ran a generated test suite that incorrectly expected empty text to fail. Tests
passed and QA approved. Security then approved and Dogfood continued generic
validation without catching the product contradiction before shutdown.

### Decision From Run 16

AD-171 adds explicit `shell_exec expected_exit_code` support and changes review
approval policy so QA/Security cannot approve after an unexpected failing
validation command. Expected negative-path probes still pass when the role
declares the expected non-zero exit code, but an undeclared runtime failure now
requires `changes_requested`.

The same source change allows `rm <validation-root>` cleanup after ticket
completion without treating it as product rework, addressing the noisy cleanup
block seen after Engineer moved `T-001` to done.

### Next Check

Run the CLI canary again. Expected improvement: QA should request Engineer
rework for the empty-text bug, Engineer should reopen and fix `T-001`, and the
subsequent QA pass should approve only after the missing-argument probe uses
`expected_exit_code`.

## Note Stats CLI Run 17: QA Found A Failing Test But Hit The Turn Budget

**Target:** `<validation-root>`
**Database:** `<validation-root>`
**Binary:** `<validation-root>`
**Date:** 2026-05-21

### Outcome

Run 17 validated the AD-171 distinction between expected and unexpected
runtime failures, but exposed a structured-handoff gap at the review boundary:

```text
ceo|completed|1
coo|completed|1
cto-weekly|completed|1
engineer|completed|1
qa|failed|1
```

Telemetry stayed in foundation space:

```text
guardrail_block|10
max_turns|1
```

No target intervention-debt tickets were created. The target worktree was clean
after runtime learnings were committed. Target commits included:

```text
4765f27 chore(learnings): update runtime learnings for qa
89d87e4 chore(tickets): move T-001 to done
10d04fb feat(cli): implement core note stats CLI with --text flag and JSON output (T-001)
a0953b5 chore(tickets): claim T-001
166a02b tickets: create implementation ticket for current scenario F-001-S001
```

### Findings

The previous review fix mostly worked. The live product behavior for the
empty-text case was corrected before QA reviewed it:

```text
<validation-root> --text ""
{"word_count":0,"character_count":0,"line_count":0}
```

QA also verified the missing-argument error path intentionally:

```text
<validation-root>
exit_code: 1
```

The remaining failure happened on the authoritative test command. QA ran
`go test ./...` on its final turn, and the target test suite failed on a
multi-line counting expectation:

```text
--- FAIL: TestCountText
note_stats_test.go:52: Multi-line text: expected ...
```

Because the failing test happened at the turn-budget edge, QA did not get a
chance to record `changes_requested`. This is a factory handoff failure: a
review role found a product-owned defect, but the system recorded only
`max_turns`.

The same run showed a freshness-policy gap. Engineer built and reused
`<validation-root>` instead of the documented `<validation-root>` path.
That path did not participate in same-session validation artifact tracking.

### Decision From Run 17

AD-172 adds three stabilizers:

- QA/Security stop shell validation after any failing build, test, or
  unexpected runtime validation command and must record structured
  `changes_requested`.
- Dispatch jobs get one final terminal-tool prompt at the turn-budget boundary,
  so a reviewer that just found a failure can still emit `job_disposition_record`
  instead of losing the finding to `max_turns`.
- External Go validation builds must use `/tmp/<project>-validation`; temp
  outputs without the `-validation` suffix are blocked before execution.

### Next Check

Run a fresh CLI canary. Expected improvement: if QA sees a failing test, the
next event should be a `changes_requested` disposition and Engineer rework, not
raw `max_turns`. If tests pass, QA should approve and route forward without
target intervention-debt creation.

## Note Stats CLI Run 18: Structured QA Rework Works, Expected-Negative Correction Is Too Strict

### Setup

- Target: `<validation-root>`
- DB: `<validation-root>`
- Binary: `<validation-root>`
- Brief: Note Stats CLI with `--text`, JSON counts, empty text as zero counts,
  and missing `--text` as an actionable error.

### Observed Lifecycle

Run 18 stayed product-first:

```text
ceo|completed|1
coo|completed|1
cto-weekly|completed|1
engineer|completed|2
qa|completed|2
orchestrator|completed|1
orchestrator|failed|1
```

The final Orchestrator failure was operator-induced shutdown while stopping the
canary for source work. Foundation telemetry stayed out of the target backlog:

```text
guardrail_block|12
llm_unreachable|1
```

### Findings

The AD-172 behavior was confirmed. QA found product-validation failures and
recorded `changes_requested` rather than ending as `max_turns`. Orchestrator
then routed rework to Engineer, and Engineer reopened `T-001` from done to
in-progress before mutating product files.

The validation-artifact freshness guard also worked across jobs. When the
rework Engineer tried to execute `<validation-root>` before rebuilding
it in that same role session, policy blocked the command and required a fresh
`go build -o <validation-root> .`.

The next generic issue is expected-negative probe correction. QA intentionally
tested the missing-argument path but first ran:

```text
<validation-root>
stderr: Error: --text flag is required
exit_code: 1
```

Because that call omitted `expected_exit_code`, policy treated it as an
unexpected runtime failure. QA then tried to rerun the same command with
`expected_exit_code: 1`, but the shell-stop rule blocked the corrective rerun.
That turned a validation-procedure mistake into another Engineer rework loop.

### Decision From Run 18

AD-173 allows exactly one correction for this class: if the only failure is an
unexpected non-zero runtime probe, QA/Security may immediately rerun the exact
same command with a matching non-zero `expected_exit_code`. Build failures,
test failures, different runtime commands, and uncorrected runtime failures
still require structured `changes_requested`.

### Next Check

Run a fresh CLI canary and confirm the missing-argument probe can be corrected
with `expected_exit_code` while real product failures still route to Engineer.

## Note Stats CLI Run 19: Completion Must Respect Unresolved Runtime Evidence

### Setup

- Target: `<validation-root>`
- DB: `<validation-root>`
- Binary: `<validation-root>`
- Brief: Note Stats CLI with `--text`, JSON counts, empty text as zero counts,
  and missing `--text` as an actionable error.

### Observed Lifecycle

Run 19 stayed product-first and did not create target intervention-debt tickets:

```text
ceo|completed|1
coo|completed|1
cto-weekly|completed|1
engineer|failed|1
```

The target repo was clean after the run and contained product commits through
the ticket lifecycle move:

```text
86d274e feat(cli): implement note stats CLI with --text flag and JSON output (T-001)
978d9ec chore(tickets): move T-001 to done
```

### Findings

The committed product still failed an acceptance path:

```text
go run cmd/note-stats/main.go --text ""
error: --text flag is required
exit status 1
```

Engineer had already observed the same failure during the role session with
`<validation-root> --text ""`, but later marked the empty-text
criterion complete, moved `T-001` to done, and committed the lifecycle move.
That means ticket metadata and lifecycle state outran the validation evidence.

The final turn also showed the terminal-grace boundary was too permissive.
After the turn-budget reminder, Engineer used the grace turn on
`git mv docs/tickets/in-progress/... docs/tickets/done/` instead of
`job_disposition_record`. The command executed, but the job still ended as
`max_turns`.

### Decision From Run 19

AD-174 records outstanding unexpected runtime validation failures by exact
runtime-command fingerprint. Engineer cannot complete the product lifecycle
while any such failure is unresolved. Completion can resume only after the
exact failing command succeeds or, for intentional negative-path checks, after
the exact command is rerun with matching `expected_exit_code`.

AD-174 also constrains the budget-edge terminal grace: once the grace prompt is
issued, only the configured terminal disposition tool may execute. Any other
tool call ends the job without side effects.

### Next Check

Run a fresh CLI canary. Expected improvement: if Engineer sees an acceptance
runtime failure, the ticket cannot move to done until that exact path is
repaired and revalidated. The run should still remain product-first and avoid
target intervention-debt tickets for foundation/runtime issues.

## Note Stats CLI Run 20: Done Move Blocked, Engineer Expected-Exit Bypass Found

### Setup

- Target: `<validation-root>`
- DB: `<validation-root>`
- Binary: `<validation-root>`
- Brief: Note Stats CLI with `--text`, JSON counts, empty text as zero counts,
  and missing `--text` as an actionable error.

### Observed Lifecycle

Run 20 confirmed the bootstrap path remains product-first:

```text
CEO -> COO -> CTO -> Engineer
```

Engineer observed the same empty-text failure:

```text
<validation-root> --text ""
stderr: error: --text flag is required
exit_code: 1
```

When Engineer later attempted the ticket lifecycle move, policy blocked it:

```text
policy: engineer cannot move a product ticket to docs/tickets/done while an unexpected runtime validation failure is unresolved in this job
```

The target stayed with `T-001` in `docs/tickets/in-progress/` and no target
intervention-debt tickets were created.

### Finding

After the correct block, Engineer tried to rerun the same failed acceptance
command with `expected_exit_code: 1` and repeated it until `circle_detected`.
That revealed a missing role boundary. The AD-173 expected-exit correction was
intended for QA/Security review-procedure mistakes, not for Engineer to
retroactively redefine a positive acceptance failure as expected.

### Decision From Run 20

AD-175 limits the one-time exact-command expected-exit correction to QA and
Security. Engineer may still declare expected non-zero probes up front for
intentional negative paths, but once an unexpected runtime validation failure
has been observed in an Engineer job, only a later successful run of that exact
command repairs the blocker.

### Next Check

Run a clean CLI canary and confirm Engineer repairs `--text ""` before ticket
completion, or remains blocked without entering an expected-exit loop.

## Run 21: Expected-Exit Bypass Closed, Repeat-Failure Loop Found

### Setup

Run 21 used a clean Note Stats CLI target at
`<validation-root>` with binary
`<validation-root>`.

### What Improved

CEO, COO, and CTO reached product-specific planning and ticketing. CTO created
`T-001` for the Note Stats CLI behavior instead of source-harness doctrine
work. Engineer claimed the product ticket, wrote `cmd/note-stats/main.go`,
initialized `go.mod`, ran `docsync_audit`, and proved:

```text
go run ./cmd/note-stats --text "hello world"
{"word_count":2,"character_count":11,"line_count":1}
```

When runtime guardrails or protocol blocks fired, the harness kept those
signals as foundation telemetry and did not create target intervention-debt
tickets.

### Finding

Engineer then observed the empty-text acceptance check fail:

```text
go run ./cmd/note-stats --text ""
Error: --text flag is required
```

The harness did not move `T-001` to done and did not dispatch the runtime
failure back through Orchestrator. However, Engineer repeated runtime probes
instead of editing the implementation, and the job ended as `circle_detected`.

### Decision From Run 21

AD-176 adds an edit-before-rerun boundary for Engineer runtime validation
failures. After an unexpected runtime failure, Engineer cannot keep running
runtime probes or add `expected_exit_code`; it must inspect/edit the
implementation, then rerun the exact failed command successfully before ticket
completion can proceed.

### Next Check

Run a clean CLI canary and confirm the failed empty-text acceptance check
causes Engineer to edit the implementation before rerunning that exact command.

## Run 22: Edit-Before-Rerun Works, Missing-Argument Correction Needed

### Setup

Run 22 used a clean Note Stats CLI target at
`<validation-root>` with binary
`<validation-root>`.

### What Improved

Engineer initially repeated the failed empty-text command:

```text
<validation-root> --text ""
error: missing text value for --text flag
```

AD-176 blocked the unchanged repeat and told Engineer to inspect/edit the
implementation. Engineer then used `file_write`, rebuilt the validation binary,
and reran the exact failed command successfully:

```text
<validation-root> --text ""
{"word_count":0,"character_count":0,"line_count":1}
```

The run again kept guardrail and loop findings as foundation telemetry rather
than target intervention-debt tickets.

### Finding

Engineer later ran the missing-argument check without `expected_exit_code`:

```text
<validation-root>
error: missing --text flag
```

That is an expected negative-path probe for this product brief, but policy
treated it as an unresolved unexpected runtime failure. Engineer then tried
other runtime probes, which were correctly blocked while the exact missing-arg
failure remained unresolved, and the job ended as `circle_detected`.

### Decision From Run 22

AD-177 permits Engineer to correct obvious missing-argument runtime probes by
rerunning the exact command once with matching `expected_exit_code`. It does
not permit retroactive correction for positive acceptance paths with supplied
input, such as `--text ""`.

### Next Check

Run a clean CLI canary and confirm Engineer can correct the missing-argument
probe with `expected_exit_code: 1`, then continue toward ticket evidence and
QA handoff.

## Run 23: Zero-Exit Error Stderr Counted As Success

### Setup

Run 23 used a clean Note Stats CLI target at
`<validation-root>` with binary
`<validation-root>`.

### What Improved

CEO, COO, and CTO again produced product-specific planning and tickets rather
than source-harness doctrine work. Engineer reached a concrete CLI
implementation and AD-177 let the missing-argument negative-path probe be
handled without creating target intervention-debt or an Orchestrator loop.

### Finding

The target binary printed an error for the empty-text acceptance case but still
returned exit code zero:

```text
<validation-root> --text ""
stderr: error: --text flag is required
Usage of <validation-root>
```

Because runtime validation previously looked only at exit code, the harness
counted that command as successful evidence. The role then moved toward
completion with a transcript that contradicted the user-visible behavior.

### Decision From Run 23

AD-178 treats conservative error-shaped stderr from direct runtime validation
as failed evidence even when the command exits zero. A clean exact rerun repairs
the blocker only when the command exits zero without error-shaped stderr.

### Next Check

Run a clean CLI canary and confirm the zero-exit stderr failure blocks further
runtime probes until Engineer edits the implementation, rebuilds, and reruns
the exact empty-text command cleanly.

## Run 24: Claimed Engineer No-Op Loop Before Implementation

### Setup

Run 24 used a clean Note Stats CLI target at
`<validation-root>` with binary
`<validation-root>`.

### What Improved

The first three roles stayed product-first. CEO recorded the Note Stats CLI
slice. COO updated the active plan and `F-001` feature contract. CTO created
and committed `T-001` for the implementation work. Guardrail findings for an
uncommitted ticket and broad shell discovery were retained as foundation
telemetry and did not create target intervention-debt tickets.

### Finding

Engineer claimed `T-001` and committed the claim, then called empty
`shell_exec` placeholders before writing product code. The no-op tool output
mostly described post-validation completion, so the model repeated no-op calls
and the job ended as `circle_detected` without any implementation.

### Decision From Run 24

AD-179 makes repeated no-op shell calls phase-aware. With a claimed ticket and
no successful validation yet, a repeated no-op is blocked with guidance to read
the in-progress ticket and feature contract, then use `file_write` for product
implementation or record a blocked disposition.

### Next Check

Run a clean CLI canary and confirm Engineer recovers from the repeated no-op
block by editing product files instead of ending as `circle_detected`.

## Run 25: Missing-Argument Correction Needs Explicit Blocker Text

### Setup

Run 25 used a clean Note Stats CLI target at
`<validation-root>` with binary
`<validation-root>`.

### What Improved

Engineer did not repeat the run24 no-op loop. It claimed `T-001`, committed the
claim, read the ticket and feature contract, wrote Go product files, and ran
direct runtime probes. The first empty-text probe failed, Engineer edited, and
the exact empty-text command later passed.

### Finding

Engineer then ran the missing-argument probe without `expected_exit_code`. That
is an expected negative-path check for this brief, but later policy blocks only
said to fix the behavior and rerun the exact failing command. The role kept
trying other positive probes and completion instead of rerunning the exact
missing-argument command with `expected_exit_code: 1`.

### Decision From Run 25

AD-180 updates unresolved runtime-failure guidance to name the missing-required
input correction explicitly: rerun the exact no-argument command once with
`expected_exit_code`, usually `1`. Positive acceptance failures still require
implementation repair and a clean exact rerun.

### Next Check

Run a clean CLI canary and confirm Engineer corrects the missing-argument probe
with `expected_exit_code`, then continues toward ticket completion without
repeating completion blocks.

## Run 26: Ticket Creation Failure Became False Progress

### Setup

Run 26 used a clean Note Stats CLI target at
`<validation-root>` with binary
`<validation-root>`.

### What Improved

CEO recovered from a role-boundary block after attempting to write the feature
contract and handed off instead of spinning. COO then updated and committed the
active plan plus canonical `F-001` feature contract. The run stayed
product-specific and did not create target intervention-debt tickets for the
guardrail blocks.

### Finding

CTO reached the right lifecycle stage but failed to materialize the product
ticket. It called `ticket_create` with `bdd_scenarios` as a quoted JSON string,
repeated that malformed call, attempted to hand-write
`docs/tickets/backlog/T-001-...md`, then called `job_disposition_record` with
`status: completed` and claimed an implementation ticket existed. No ticket had
actually been created, and the dispatcher re-entered CTO rather than moving to
Engineer with ordinary product work.

### Decision From Run 26

AD-181 tracks failed `ticket_create` and failed ticket-file `file_write`
attempts as unresolved ticket-creation state. Successful dispositions are
blocked until a later `ticket_create` succeeds; blocked or failed dispositions
remain available for honest handoff. `ticket_create` parse errors now name
common array-shape repairs, including `bdd_scenarios:["F-001-S002"]`.

### Next Check

Run a clean CLI canary and confirm CTO either repairs malformed
`ticket_create` arguments and creates a real backlog ticket, or records an
honest blocked disposition without re-entering a false-completed CTO loop.

## Run 27: Missing-Argument Correction Needed To Become The Only Next Action

### Setup

Run 27 used a clean Note Stats CLI target at
`<validation-root>` with binary
`<validation-root>`.

### What Improved

AD-181 was confirmed against the live lifecycle. CEO and COO kept the run
product-specific, CTO created and committed a real `T-001` product ticket with
`bdd_scenarios` as a JSON array, and Engineer claimed the ticket and began
implementation instead of receiving a false completed handoff.

### Finding

Engineer later ran the missing-argument runtime validation binary without
`expected_exit_code`. That mistake is recoverable for this product brief, but
the blocker did not force the exact correction as the immediate next action.
Engineer continued through adjacent edits, decisions, commits, and completion
attempts before resolving the procedural validation gap.

Two non-blocking follow-up findings were also recorded for the next loop:
target naming still drifts toward source-harness names such as
`cmd/mars`/`module mars`, and runtime validation still checks
process success more strongly than semantic JSON values such as empty-text line
count.

### Decision From Run 27

AD-182 stores the exact unresolved runtime validation command and, for obvious
missing-argument probes, the exact `expected_exit_code` correction. Engineer is
blocked from unrelated mutating work until that correction runs or the role
records an honest blocked disposition.

### Next Check

Run a clean CLI canary and confirm Engineer either runs the exact
missing-argument `expected_exit_code` correction before unrelated work, or
records a blocked disposition that names why the correction cannot be applied.

## Run 28: External Validation Binary Was Stale After Source Edit

### Setup

Run 28 used a clean Note Stats CLI target at
`<validation-root>` with binary
`<validation-root>`.

### What Improved

The run again reached ordinary product work. CTO created `T-001` with valid
ticket metadata, committed it, and handed to Engineer. Engineer claimed the
ticket, used a product-specific module name (`note-stats-cli`), wrote source,
and built an external validation binary rather than leaving a repo-local build
artifact.

### Finding

The first empty-text acceptance probe failed:

```text
<validation-root> --text ""
stderr: Error: --text flag is required
exit_code: 1
```

Policy correctly blocked an unchanged rerun and required implementation work.
Engineer then edited `main.go`, but reran the same `<validation-root>`
binary without rebuilding it. The artifact still represented the pre-edit
source, so the same failure repeated and the job ended as `circle_detected`.

### Decision From Run 28

AD-183 records the runtime-edit watermark when `<validation-root>` artifacts
are built. After a runtime failure and implementation edit, executing the old
artifact is blocked until the role rebuilds it with `go build -o
<validation-root> ...`.

### Next Check

Run a clean CLI canary and confirm Engineer rebuilds the external validation
artifact after editing source, then reruns the empty-text acceptance probe
against the rebuilt binary.

## Run 29: Ticket Evidence Was Written Before Validation

### Setup

Run 29 used a clean Note Stats CLI target at
`<validation-root>` with binary
`<validation-root>`.

### What Improved

The run stayed product-first. CEO, COO, and CTO produced a product-specific
plan, feature contract, and ordinary product ticket. CTO committed the ticket,
Engineer claimed it, and source-file policy caught a missing `MarsDocSync`
block before product code entered the target.

### Finding

Engineer later updated the in-progress ticket with `evidence_links` and
`verified_by` before any successful validation command had run in the same job.
That made ticket metadata look ahead of the actual validation transcript. The
role then repeated no-op shell placeholders and ended as `circle_detected`
before proving the Note Stats CLI behavior.

### Decision From Run 29

AD-184 blocks Engineer `file_write` updates that populate in-progress ticket
`evidence_links` or `verified_by` until the same job records successful
validation. The validation can be a test, build, or direct runtime command that
exercises the BDD scenario.

### Next Check

Run a clean CLI canary and confirm Engineer responds to the guardrail by
running validation before ticket evidence updates, then continues toward ticket
completion without no-op placeholder loops.

## Run 30: QA Stalled After Stale Artifact Guard

### Setup

Run 30 used a clean Note Stats CLI target at
`<validation-root>` with binary
`<validation-root>`. The first start
attempt initialized the target and seeded CEO but could not bind the local
orchestrator port inside the sandbox. The same DB/target was then restarted
outside the sandbox, which also made the failed-start path part of the
observation.

### What Improved

CEO, COO, and CTO stayed product-specific. CTO created a real `T-001` product
ticket and Engineer claimed it. Engineer did not write ticket evidence before
validation. It repaired a failing unused-import test build, built
`<validation-root>`, proved the empty-text and hello-world CLI paths,
ran `go test ./...`, passed docsync, wrote ticket evidence, moved `T-001` to
done, committed the lifecycle move, and handed off to QA.

### Finding

QA correctly rejected the Engineer-built `<validation-root>` binary as
stale cross-session evidence:

```text
policy: external validation binary "<validation-root>" must be built
in this role session before it can be trusted
```

The role then went quiet for several minutes instead of rebuilding the binary
in its own session or recording a structured review disposition. No
target-owned intervention-debt ticket was created; the guardrail and later
manual stop stayed foundation telemetry.

### Decision From Run 30

AD-185 makes external validation artifact freshness errors exact-action
oriented. The policy now names the `shell_exec argv` rebuild command, for
example `["go","build","-o","<validation-root>","."]` for a root Go
CLI, and generated QA/Security guidance says to run the exact correction before
rerunning the binary.

### Next Check

Run a clean CLI canary and confirm QA rebuilds `<validation-root>` in its own
role session, reruns the runtime probe, and records an approval or
changes-requested disposition without a long quiet stall.

## Run 31: Build Correction Guessed The Wrong Package Target

### Setup

Run 31 used a clean Note Stats CLI target at
`<validation-root>` with binary
`<validation-root>`.

### What Improved

The lifecycle stayed product-first through implementation. CEO accepted the
README brief, COO updated the active plan and F-001 feature contract, CTO
created a real `T-001` product ticket, and Engineer claimed the ticket before
writing source. Intervention signals from claim-first, dirty disposition, build
artifact, and runtime-probe guardrails stayed foundation-owned telemetry and
did not materialize target intervention-debt tickets.

Engineer also followed the exact missing-argument correction when prompted:
after running the omitted-`--text` probe without `expected_exit_code`, the
policy blocked unrelated mutation and Engineer reran the exact command with
`expected_exit_code: 1`.

### Findings

Two gaps remained. First, Engineer treated successful exit as enough evidence
even when the empty-text probe printed the wrong JSON:

```text
{"words":0,"lines":1,"characters":0}
```

The product brief and feature contract expected:

```text
{"words":0,"lines":0,"characters":0}
```

Second, QA correctly blocked `go build ./cmd/note-stats` because the command
would create a repo-local `note-stats` binary, but the recovery text did not
name an exact corrected argv. QA guessed `go build -o <validation-root> .`,
which failed because the target has no Go files at the repository root. The
review then had to record `changes_requested` from a harness-guidance failure
rather than from the intended product validation.

### Decision From Run 31

AD-186 adds exact build correction output for Go build guardrails. The error
now preserves the original package target, for example:

```json
["go","build","-o","<validation-root>","./cmd/note-stats"]
```

Generated Engineer and QA guidance now also says exact expected-output
examples need automated assertions. Exit-code-only runtime smoke evidence is
not enough when the README, ticket, or BDD contract names exact CLI output, API
body, UI state, or persisted data. QA approval is also blocked for Go source
changes when no `_test.go` files exist, so smoke-only Go delivery routes back
to Engineer before Security or release.

### Next Check

Run a clean CLI canary and confirm the build guard emits the exact
`./cmd/<name>` correction, QA follows it, and the implementation path includes
tests that catch expected-output mismatches before ticket completion.

## Run 32: Missing-Input Correction Repro Looped Before Repair

### Setup

Run 32 used a clean Note Stats CLI target at
`<validation-root>` with binary
`<validation-root>`.

Target brief:

```text
# Note Stats CLI

Build a small Go CLI that accepts `--text` and prints JSON counts for words,
lines, and characters.

Acceptance examples:
- `--text "hello world"` prints `{"words":2,"lines":1,"characters":11}`
- `--text ""` prints `{"words":0,"lines":0,"characters":0}`
- omitting `--text` exits non-zero with a clear error
```

Command:

```bash
<validation-root> start \
  --repo <validation-root> \
  --db <validation-root> \
  --log-file <validation-root> \
  --debug
```

### Observations

- CEO accepted the README-derived product goal and COO updated the active plan
  plus BDD contract without creating target intervention-debt tickets.
- CTO created `T-001` for product implementation and committed it, confirming
  the product-first path still beats intervention-debt amplification.
- Engineer claimed the ticket before mutation, wrote product code and a
  `_test.go` file, and the tests caught an initial counting bug before runtime
  smoke evidence.
- The build-output guard emitted an exact correction for a repo-local Go build
  and preserved the package target:

```json
["go","build","-o","<validation-root>","./cmd/mars"]
```

- Runtime validation then exposed two generic gaps. First, the target ticket
  and implementation used foundation names (`cmd/mars` and `module
  mars`) even though the product was Note Stats CLI. Second, after the
  missing-input runtime probe panicked, the harness required an exact
  `expected_exit_code` repro but continued blocking `file_write` when that repro
  still failed. Engineer was sent back to rerun the same failing command rather
  than allowed to repair the implementation.
- Guardrail and shutdown/inference signals were kept as foundation telemetry
  rather than target backlog tickets.

### Decision From Run 32

AD-187 changes missing-input runtime containment to unlock implementation edits
after Engineer has attempted the exact expected-exit correction and it still
fails. Completion, commits, ticket done-moves, and unrelated runtime probes
remain blocked until the exact runtime path is repaired. Generated CTO and
Engineer guidance now also says target tickets and fresh Go modules must derive
command, module, and binary names from the target project, not from foundation
`mars` defaults.

### Next Check

Run another clean CLI canary and confirm that a failed missing-input correction
attempt permits product repair rather than a repro loop, while completion stays
blocked until the exact runtime command passes. Also confirm CTO/Engineer choose
target-derived names such as `cmd/note-stats` and `module note-stats` for the
Note Stats target.

## Run 33: Target Naming Fixed, Repeated Runtime Failures Left Stale Counters

### Setup

Run 33 used a clean Note Stats CLI target at
`<validation-root>` with binary
`<validation-root>`.

### Observations

- CEO and COO reached product planning and BDD update without target
  intervention-debt ticket creation.
- CTO created `T-001` with target-derived affected files:
  `cmd/note-stats/main.go`, `go.mod`, and optional target docs. The previous
  `cmd/mars` naming leak did not repeat.
- Engineer initialized `module note-stats`, confirming generated target
  guidance no longer steers fresh targets toward foundation module names.
- The missing-input/runtime repair path allowed Engineer to edit after the
  positive `--text ""` acceptance path failed, rather than trapping the job
  before source repair.
- The same `--text ""` runtime command failed multiple times while Engineer
  iterated. After the command later exited zero, the runtime policy still held
  an outstanding failure count from an earlier identical attempt and blocked
  further probes. The job ended with `circle_detected`.
- No target intervention-debt tickets were created; the loop was kept as
  foundation telemetry.

### Decision From Run 33

AD-188 changes runtime repair accounting so one successful exact rerun clears
all unmatched failures for the same command fingerprint in the current job.
Expected-exit corrections use the same multi-count repair rule. This prevents
repeated attempts at one runtime path from leaving stale blockers after the
path finally succeeds.

### Next Check

Run another clean CLI canary and confirm repeated failures of the same exact
runtime command clear after one successful exact rerun. The canary should then
continue to tests, evidence, QA handoff, or a later product-quality blocker
instead of ending in `circle_detected` from stale runtime counters.

## Run 34: Runtime Repair Reached QA, Review Protocol Caused False Rework

### Setup

Run 34 used a clean Note Stats CLI target at
`<validation-root>` with binary
`<validation-root>`.

### Observations

- Product-first bootstrap reached real target artifacts again: COO updated the
  active plan and feature contract, CTO created `T-001`, and Engineer claimed
  the ticket before source mutation.
- AD-188 held in live behavior. Engineer fixed the `--text ""` acceptance
  path, reran that exact command successfully, then corrected the omitted
  `--text` negative path with `expected_exit_code: 1`; stale runtime counters
  did not trap the job.
- Engineer added `cmd/note-stats-cli/main_test.go`, fixed an unused import
  compile failure, passed `go test ./cmd/note-stats-cli`, and passed
  `docsync_audit`.
- The ticket moved to done and QA started review, but product source, tests,
  and `go.mod` were committed inside `chore(tickets): move T-001 to done`
  instead of a separate product/test commit.
- QA's shallow `grep` over `cmd/*` missed `cmd/note-stats-cli/main.go`, then
  ran `go mod init note-stats-cli` even though the module already existed.
- QA validated the happy path and empty-string path, then ran the intentional
  omitted-flag negative path without `expected_exit_code`. The product exited
  non-zero with a clear error, but review policy counted the command as an
  unexpected failure and forced `changes_requested`.
- Orchestrator followed the false QA rework signal and read sample
  player-movement ticket text from `docs/tickets/README.md` as if it were live
  ticket state.
- No target intervention-debt tickets were created; guardrail and operator-stop
  events stayed foundation-owned telemetry.

### Decision From Run 34

AD-189 constrains QA/Security shell execution to validation-only commands,
blocks package/module initialization and no-op placeholders during review,
requires non-ticket implementation changes to be committed before ticket done
moves, and tells Orchestrator to route from live lifecycle state and
`source_disposition` rather than ticket README examples.

### Next Check

Run another clean CLI canary and confirm Engineer creates a separate
implementation/test commit before the done move, QA sets `expected_exit_code`
on the omitted-flag probe on its first run, and Orchestrator no longer treats
ticket README examples as actionable backlog.

## Run 35: Product Delivery Worked, QA No-Op Completion Loop Remained

### Setup

Run 35 used a clean Note Stats CLI target at
`<validation-root>` with binary
`<validation-root>`.

### Observations

- CEO and COO stayed product-specific, reading the Note Stats CLI README and
  producing the active plan and `F-001` product walking-skeleton contract.
- CTO created a real product ticket,
  `docs/tickets/backlog/T-001-implement-core-cli-functionality-for-note-stats-cli.md`,
  and committed it before handing off to Engineer.
- Engineer hit the claim gate once, followed the exact `git mv` instruction,
  implemented `cmd/note-stats/main.go`, initialized `module note-stats`, fixed
  the empty-string acceptance failure, validated the omitted-flag negative path
  with `expected_exit_code: 1`, and committed product source, README, and
  `go.mod` separately from the ticket lifecycle move.
- The done-ticket commit was lifecycle-only, confirming the AD-189 product
  traceability fix.
- QA read the done ticket and source, ran `docsync_audit`, built a fresh
  `<validation-root>` binary, and validated the happy and empty-string
  paths.
- QA then repeatedly called `shell_exec` with empty `argv` placeholders instead
  of recording `job_disposition_record`, ending with `circle_detected`.
- The runtime quarantined the loop as foundation-owned telemetry; it did not
  create target intervention-debt tickets or dispatch Orchestrator.

### Decision From Run 35

AD-190 adds one required-terminal-tool circle grace turn and clearer reviewer
no-op disposition routing. After successful validation, QA/Security no-op shell
placeholders now point directly to `job_disposition_record`, and policy-blocked
no-op shell calls are counted as no-op failures for loop telemetry.

### Next Check

Run another clean CLI canary and confirm QA records an approved disposition
after successful validation instead of ending in `circle_detected`. The canary
should still remain generic: vary the target brief over time so the factory is
not over-optimized for this Note Stats CLI or Space Invaders-style project.

## Run 36: Product Commit Outran Unresolved Acceptance Failure

### Setup

Run 36 used a clean Note Stats CLI target at
`<validation-root>` with binary
`<validation-root>`.

### Observations

- CEO, COO, and CTO remained product-specific: the active plan, feature
  contract, and `T-001` ticket were derived from the Note Stats CLI brief.
- Engineer claimed the ticket, wrote `cmd/note-stats/main.go` plus `go.mod`,
  and built `<validation-root>` before running runtime probes.
- The happy path passed, but the contracted empty-string acceptance probe
  still returned `Error: --text flag is required` with exit code 1.
- Policy blocked ticket completion, successful disposition, and moving the
  ticket to `docs/tickets/done/` while that failure was unresolved.
- The role still found wasteful side paths: shell-wrapper probes, unrelated
  `go test ./...`, ticket evidence edits, and an implementation commit were
  allowed before the exact failed acceptance command passed.
- The job ended at `max_turns`; runtime telemetry stayed foundation-owned and
  no target intervention-debt ticket or Orchestrator recovery loop was created.

### Decision From Run 36

AD-191 tightens the unresolved-runtime-failure boundary. While an Engineer has
an outstanding positive runtime acceptance failure, `shell_exec` may only
rebuild the same stale `<validation-root>` artifact when required or rerun the
exact failed command after a source edit. Other shell probes, shell wrappers,
tests, ticket moves, and commits are blocked. Product commits are also blocked
until the runtime blocker is repaired so broken implementation state cannot be
preserved as progress.

### Next Check

Run another clean non-game canary and confirm the Engineer repairs the
acceptance failure before any implementation commit or lifecycle move. The run
should still exercise the terminal-disposition off-ramp from AD-190 once QA is
reached.

## Run 37: Alternate CLI Canary Reached Done Ticket, Then Hit Stale Ticket-Creation State

### Setup

Run 37 used a clean Temperature JSON CLI target at
`<validation-root>` with binary
`<validation-root>`.

### Observations

- The first sandboxed start initialized the target and seeded a CEO job, then
  failed to bind the dashboard port. The escalated retry started cleanly and
  processed a single active lifecycle for the target.
- CEO, COO, and CTO remained product-specific. COO updated the canonical
  `F-001` feature contract after a duplicate-feature guardrail block, and CTO
  created `T-001` for the Temperature JSON CLI.
- Engineer hit the claim gate once, claimed `T-001`, wrote product code and
  tests, and corrected a DocSync metadata block on the test file.
- Engineer tried to populate ticket evidence before validation; policy blocked
  that evidence write and the role moved to validation.
- The implementation passed the zero-value `--celsius 0` path, the
  `--celsius 100` path, and the omitted-flag negative path after rerunning the
  exact omitted-flag command with `expected_exit_code: 1`.
- This confirmed the AD-191 runtime-repair lane on a non-Note-Stats target:
  unrelated runtime probes were blocked until the exact missing-argument
  correction ran.
- Engineer committed product code, updated evidence, passed docsync, moved
  `T-001` to done, and committed the lifecycle move.
- The successful Engineer disposition was then falsely blocked because the
  earlier failed in-progress ticket-evidence write was counted as unresolved
  ticket creation debt.
- No target intervention-debt tickets were created.

### Decision From Run 37

AD-192 narrows ticket-creation failure accounting. Failed `ticket_create`
calls and non-Engineer ticket-file creation/bypass attempts still block false
successful planning dispositions, but Engineer ticket evidence update failures
do not poison the ticket-creation state. A blocked evidence write should send
Engineer to validation, not later prevent a valid done-ticket handoff.

### Next Check

Run a fresh alternate canary and confirm Engineer can complete
`job_disposition_record` after a pre-validation ticket-evidence block is later
remedied by successful validation, evidence update, product commit, and
done-ticket lifecycle move.

## Run 38: Alternate CLI Reached Product Completion, QA Then Looped On Review No-Ops

### Setup

Run 38 used a clean Temperature JSON CLI target at
`<validation-root>` with binary
`<validation-root>`.

### Observations

- CEO remained product-specific and selected the Temperature JSON CLI walking
  skeleton from the target README.
- COO wrote product-specific planning artifacts, but then tried to create a
  ticket by direct `file_write`, `mars_cli ticket_create`, and
  `mars_cli tools run ticket_create` despite not owning ticket
  creation. Policy blocked those attempts; COO recovered only by recording a
  blocked ticket-breakdown disposition that Orchestrator routed to CTO.
- CTO used `ticket_create`, committed the product ticket, and handed off to
  Engineer. It first attempted successful disposition before committing the
  ticket, and policy correctly forced the ticket commit first.
- Engineer claimed the ticket, implemented `cmd/temperature-json-cli/main.go`
  and `go.mod`, built an external `<validation-root>`
  binary, fixed an unused import build failure, and passed the `--celsius 0`,
  `--celsius 100`, and omitted-flag runtime probes.
- Engineer still spent turns on avoidable side paths: discovery shell before
  claim, blocked repo-local build output, `go test` reporting `[no test
  files]`, an empty shell placeholder, and a ticket-done move before evidence.
  The policy gates recovered each side path and the role completed a product
  commit, evidence update, lifecycle-only done-ticket commit, and
  `qa_review` disposition.
- QA read the done ticket and source, built a fresh validation binary, and ran
  at least one positive runtime probe. It then repeatedly called empty or
  placeholder `shell_exec` commands instead of recording the terminal
  disposition, ended as `circle_detected`, and was quarantined as
  foundation-owned telemetry with no target intervention-debt ticket.
- QA did not convert the missing durable Go tests into structured
  `changes_requested`, even though the target README and ticket expected
  automated tests.

### Decision From Run 38

AD-193 strengthens the review terminal boundary and planning handoff boundary.
After a review role receives a blocked no-op shell placeholder following
successful validation, policy records that only `job_disposition_record` may be
called next; further shell validation or placeholders are blocked with terminal
disposition guidance. The guidance now routes Go targets with source but no
`_test.go` files to `changes_requested`, not approval. COO guidance also
forbids alternate ticket-creation paths and planning roles can record a clean
`ticket_breakdown` handoff to CTO even if they previously hit ticket-creation
policy blocks for work they do not own.

### Next Check

Run a fresh Temperature JSON CLI canary and confirm COO hands directly to CTO,
Engineer still reaches product implementation, and QA records
`changes_requested` for missing tests or an approved disposition after adequate
test/runtime evidence instead of cycling on shell no-ops.

## Run 39: Failing Tests Outran Runtime Probes And Product Commit

### Setup

Run 39 used a clean Temperature JSON CLI target at
`<validation-root>` with binary
`<validation-root>`.

### Observations

- CEO and COO remained product-specific. COO updated the active plan and
  canonical `F-001` feature contract from the target README, committed the
  planning work, and handed directly to CTO without trying alternate ticket
  creation paths.
- CTO created an ordinary product ticket with `ticket_create`, committed it,
  and handed to Engineer. It still attempted successful disposition before the
  ticket commit, but the clean-tree policy recovered the handoff.
- Engineer claimed the ticket and wrote Temperature JSON CLI source plus a
  `_test.go` file, which confirmed the missing-test review feedback from run
  38 had improved the implementation shape.
- `go test ./cmd/temperature-json-cli/...` failed because helper definitions
  were duplicated across `main.go` and `temperature.go`, while the test's
  subprocess path also diverged from the source layout.
- Engineer proved some runtime probes with `go run main.go`, then attempted
  forbidden `rm` cleanup and continued toward ticket evidence and a product
  commit while the authoritative test command was still failing.
- The run stopped at Engineer `max_turns`; the runtime failure remained
  foundation telemetry and no target intervention-debt ticket was created.

### Decision From Run 39

AD-194 adds a failed test/build repair lane for Engineer jobs. Once Engineer
observes a failing test or build command, runtime probes, unrelated shell
commands, ticket evidence updates, ticket completion, successful disposition,
and product commits stay blocked until source or tests are edited and the
exact failing command passes.

### Next Check

Run a clean Temperature JSON CLI canary with the patched harness and confirm a
failing `go test` forces source/test repair plus exact passing rerun before the
Engineer can commit product work or update ticket evidence. Continue rotating
target briefs after this regression check so the live matrix stays generic.

## Run 40: CTO Crossed From Ticket Shaping Into Product Implementation

### Setup

Run 40 used a clean Temperature JSON CLI target at
`<validation-root>` with binary
`<validation-root>`.

### Observations

- CEO chose the target product slice, and COO produced product-specific active
  plan and feature contract artifacts.
- COO still paid formatting tax on `job_disposition_record` by first sending a
  string-shaped `handoff`, but recovered and routed CTO without Orchestrator
  detour or alternate ticket creation.
- CTO created the product ticket with `ticket_create`, then crossed into
  implementation by writing `go.mod`, repeatedly attempting product source and
  test writes, and updating README usage notes.
- Source-file DocSync guardrails blocked the source/test writes, but `go.mod`
  and README-style product mutations were not blocked by role ownership. CTO
  committed `go.mod` together with the ticket before the operator stopped the
  run for source repair.
- The failure remained foundation-owned guardrail/lifecycle evidence; no target
  intervention-debt ticket was created.

### Decision From Run 40

AD-195 makes CTO a bounded technical-planning writer. CTO may write technical
planning artifacts under `docs/design-docs/`, `docs/reports/strategy/`, or
`docs/goals/observations.md`, and may create tickets with `ticket_create`.
Product implementation, package/module files, README usage notes, source,
tests, build config, and root product files are blocked and routed to
ticket-backed Engineer delivery.

### Next Check

Run a clean Temperature JSON CLI canary with the patched harness and confirm
CTO creates and commits only the implementation ticket before handing to
Engineer. The later Engineer phase should still verify AD-194 by preventing
failing tests from being bypassed by runtime probes or commits.

## Run 41: Exact Test Repair Was Too Narrow

### Setup

Run 41 used a clean Temperature JSON CLI target at
`<validation-root>` with binary
`<validation-root>`.

### Observations

- CEO and COO again produced product-specific planning from the target README.
  COO committed the active plan and `F-001` feature contract, then routed CTO.
- CTO confirmed AD-195 by creating and committing only the ordinary product
  ticket before handing to Engineer. It did not write `go.mod`, README usage,
  source, tests, or other implementation files.
- Engineer claimed the ticket, committed the claim, and wrote `go.mod`,
  source, tests, and README usage notes. That confirmed implementation had
  moved back behind ticket-backed Engineer delivery.
- `go test ./cmd/temperature-json-cli/...` failed first on an unused import,
  then after a test edit failed on the test's subprocess path. The AD-194
  guardrail correctly blocked runtime probes, cleanup, ticket evidence, and
  product commits while the test/build repair lane was unresolved.
- The guardrail was too exact-command-bound. Focused same-lane validation such
  as `go test ./cmd/temperature-json-cli` was rejected because it was not the
  original `go test ./cmd/temperature-json-cli/...` command, trapping the role
  in repeated policy blocks. Engineer then attempted workaround paths,
  including an ad hoc `verify_functionality.sh` root script.
- The run was stopped for source repair; no target intervention-debt ticket
  was created.

### Decision From Run 41

AD-196 changes the Engineer test/build repair lane from exact-command-only to
same-lane validation. After a failing test, Engineer may edit source, tests,
fixtures, or build/package config and rerun any recognized test command; after
a failing build, it may rerun a recognized build command. Runtime probes,
helper scripts, ticket evidence, ticket completion, successful disposition,
and commits remain blocked until the same lane passes. New root scratch files
with validation/scratch/verify naming are also blocked before they enter the
target repo.

### Next Check

Run a clean Temperature JSON CLI canary with the patched harness and confirm
Engineer can recover from a failing test via source/test repair plus focused
same-lane validation, without runtime side probes, helper scripts, or product
commits before validation passes.

## Run 42: Same-Lane Repair Missed Simple CD Shell Validation

### Setup

Run 42 used a clean Temperature JSON CLI target at
`<validation-root>` with binary
`<validation-root>`.

### Observations

- CEO, COO, and CTO stayed product-first. CTO created and committed only the
  implementation ticket, then handed to Engineer.
- Engineer claimed the ticket, wrote product source, Go tests, README usage
  notes, and `cmd/temperature-json-cli/go.mod`.
- The first `go test ./cmd/temperature-json-cli/...` failed because the target
  had no root Go module. AD-196 allowed bounded repair writes such as the
  nested `go.mod`, and continued blocking runtime probes, build substitution,
  helper paths, ticket evidence, and product commits while the test lane was
  unresolved.
- The remaining failure was validation command recognition. The Engineer
  repeatedly used `shell_command: cd cmd/temperature-json-cli && go test -v .`,
  which is a natural focused Go-package validation command, but the repair
  classifier treated all shell control syntax as unclassifiable and blocked it
  as an unrelated side path.
- The run was stopped for source repair. No target intervention-debt ticket was
  created.

### Decision From Run 42

AD-197 recognizes the narrow `cd <dir> && <test-or-build command>` shell shape
as same-lane validation for repair-lane classification. The command after the
simple `cd` is classified as the validation command; arbitrary shell wrappers,
multiple chains, pipes, redirection, substitutions, cleanup, runtime probes,
and ticket moves remain blocked while a failing test/build lane is unresolved.

### Next Check

Run a clean Temperature JSON CLI canary with the patched harness and confirm
`cd cmd/temperature-json-cli && go test -v .` can repair the failed test lane,
then continue toward product commit, ticket evidence, and QA handoff.

## Run 43: Security Needed Terminal Evidence Convergence

### Setup

Run 43 used a clean Temperature JSON CLI target at
`<validation-root>` with binary
`<validation-root>`.

### Observations

- CEO, COO, and CTO stayed product-specific. COO updated the active plan and
  feature contract; CTO created only the implementation ticket before handing
  to Engineer.
- Engineer claimed `T-001`, implemented `cmd/temperature-json/main.go`,
  automated tests, README usage notes, and `go.mod`.
- Engineer corrected the missing-argument runtime probe by rerunning the exact
  command with `expected_exit_code: 1`, then repaired a failing `go test` and
  reran tests successfully.
- Engineer committed product work, updated ticket evidence, moved `T-001` to
  done in a lifecycle-only commit, and recorded a QA handoff.
- QA read the done ticket and source, ran docsync, passed `go test`, validated
  the `--celsius 0` and `--celsius 100` runtime paths, recovered from blocked
  extra probes, and approved.
- Security inspected recent commits, scanned for secrets, ran docsync, ran
  `go test`, read source, and ran a successful `--celsius 0` runtime probe.
  With clean evidence already present, the next model turn spent more than
  five minutes instead of recording `job_disposition_record`.
- Stopping the run cancelled the long LLM call and recorded a
  foundation-owned `llm_unreachable` signal. No target intervention-debt
  ticket was created.

### Decision From Run 43

AD-198 makes clean review evidence a terminal-only boundary. Once QA or
Security has successful read plus validation evidence and no current-job
validation failure, the agent loop injects a required `job_disposition_record`
reminder. The next response may only call that terminal tool, and the terminal
grace response is capped so review completion cannot spend another full
default inference timeout.

### Next Check

Run a clean Temperature JSON CLI canary with the patched harness and confirm
Security records a terminal disposition promptly after clean read and
validation evidence, then continue to Dogfood or release-path validation
without target intervention-debt amplification.

## Run 44: QA Validation Procedure Mistake Triggered False Rework

### Setup

Run 44 used a clean Temperature JSON CLI target at
`<validation-root>` with binary
`<validation-root>`.

### Observations

- CEO, COO, and CTO stayed product-specific. The target received a product
  vision, active goal, feature contract, active plan, and one ordinary product
  ticket for the Temperature JSON CLI.
- Engineer claimed `T-001`, implemented the Go CLI, built an external
  `<validation-root>` binary, proved the `--celsius 0` and
  `--celsius 100` JSON outputs, corrected the missing-argument probe with
  `expected_exit_code: 1`, committed product work, updated evidence, moved the
  ticket to done, and handed to QA.
- Guardrail blocks for discovery, repo-local build output, runtime
  negative-path correction, and no-op placeholders remained foundation-owned
  telemetry and did not create target intervention-debt tickets.
- QA read the ticket, README, and implementation, then ran
  `go build -o <validation-root> cmd/temperature-json-cli`.
  Go rejected the command because the repo-local package target needs
  `./cmd/temperature-json-cli`.
- QA attempted corrected build commands, including the exact `./cmd/...` form,
  but policy had already recorded a build failure and forced
  `changes_requested`. Orchestrator routed the target back to Engineer even
  though the observed failure was review procedure, not product behavior.

### Decision From Run 44

AD-199 records obvious QA/Security validation-procedure failures separately
from target validation failures. Missing `./` on repo-relative Go package
targets and root `.` builds in `cmd/*` CLI repos no longer poison the review
with `validation:command:failure` or `validation:build:failure`; reviewers can
retry the corrected validation command. Real compile, test, and runtime
failures still route to structured rework.

### Next Check

Run a clean Temperature JSON CLI canary with the patched harness and confirm QA
recovers from Go build-target command mistakes, continues to runtime evidence,
and then reaches the AD-198 terminal disposition boundary instead of looping or
creating false rework.

## Run 45: Nested Module Validation Was Poisoned By Argv Shell Syntax

### Setup

Run 45 used a clean Temperature JSON CLI target at
`<validation-root>` with binary
`<validation-root>`.

### Observations

- CEO, COO, and CTO-weekly stayed product-first. COO rewrote the active plan
  and feature contract; CTO-weekly created and committed a single product
  implementation ticket before handing to Engineer.
- Engineer claimed `T-001` and wrote a nested Go module under
  `cmd/temperature-json-cli` with implementation, tests, and `go.mod`.
- Engineer attempted the natural focused validation as
  `argv:["cd","cmd/temperature-json-cli","&&","go","test","./..."]`.
  `shell_exec` rejected it because argv mode cannot run shell builtins or
  control operators.
- Engineer then ran `go test ./cmd/temperature-json-cli/...` from the target
  root. Go rejected it because the nested module is outside the root module,
  and the harness treated that as an unresolved test failure.
- The unresolved test lane then correctly blocked builds, commits, discovery,
  and cleanup, but the root cause was still tool-call formatting drift rather
  than target implementation quality. The run was stopped for source repair.

### Decision From Run 45

AD-200 normalizes only the narrow safe argv-shaped validation pattern
`["cd","<dir>","&&",<test-or-build-command>...]` into `shell_command` before
execution. The right-hand side must classify as a test or build command, and
tokens with pipes, redirects, substitutions, background operators, or arbitrary
shell syntax remain rejected.

### Next Check

Run a clean Temperature JSON CLI canary with the patched harness and confirm
the Engineer can run nested-module `cd ... && go test` validation, continue to
product commit and ticket closure, then reach QA/Security to validate AD-199
and AD-198 in the live review path.

## Run 46: Missing-Input CLI Probe Blocked Product Completion

### Setup

Run 46 used a clean Temperature JSON CLI target at
`<validation-root>` with binary
`<validation-root>`.

### Observations

- CEO, COO, and CTO-weekly stayed product-first again. COO rewrote the active
  plan and feature contract; CTO-weekly created and committed a single product
  implementation ticket before handing to Engineer.
- Engineer claimed `T-001`, wrote `cmd/temperature-json-cli/main.go`, created
  `go.mod`, added tests, and attempted to build a repo-local binary.
- The build-artifact guardrail blocked `-o temperature-json-cli` and provided
  the exact external `<validation-root>` correction.
  Engineer followed it and the build succeeded.
- Engineer proved positive runtime behavior for `--celsius 0` and
  `--celsius 100`, receiving the expected JSON outputs.
- Engineer then ran `<validation-root>` with no arguments.
  The binary exited non-zero with `--celsius flag is required`, which is useful
  negative-path evidence for this CLI brief.
- Because the command omitted `expected_exit_code`, the harness opened an
  unexpected runtime-failure blocker. The model then attempted unrelated
  probes, edits, commits, ticket moves, and terminal dispositions; the
  guardrail blocked those actions, but the job could not progress.

### Decision From Run 46

AD-201 treats obvious no-argument or missing-required-input runtime probes as
expected negative-path validation when they exit non-zero with clear
required/usage output and no crash markers. Explicit `expected_exit_code`
remains supported, while panic/traceback/runtime-error output and positive
acceptance failures still open the strict repair lane.

### Next Check

Run a clean Temperature JSON CLI canary with the patched harness and confirm
Engineer can commit and close the product ticket after positive JSON checks
and missing-input validation, then reach QA/Security to validate AD-199 and
AD-198 in the live review path.

## Run 47: Invalid-Input CLI Probe Needed The Same Negative-Path Treatment

### Setup

Run 47 used a clean Temperature JSON CLI target at
`<validation-root>` with binary
`<validation-root>`.

### Observations

- CEO, COO, and CTO-weekly stayed product-first and created a single ordinary
  implementation ticket.
- Engineer claimed `T-001`, implemented the CLI, updated README, and proved
  positive runtime behavior with `go run cmd/temperature-json-cli/main.go 25`,
  receiving `{"celsius":25,"fahrenheit":77}`.
- Engineer ran `go run cmd/temperature-json-cli/main.go` with no arguments.
  The CLI returned missing-input usage text, and AD-201 allowed the job to
  continue instead of opening the run46 blocker.
- Engineer then ran `go run cmd/temperature-json-cli/main.go invalid`. The CLI
  correctly returned `Invalid temperature value 'invalid'. Must be a number.`,
  but the runtime policy treated this deliberate invalid-input check as an
  unexpected failure.
- Subsequent tests, expected-exit correction attempts, commits, and positive
  reruns were blocked by the unresolved runtime guardrail. The guardrail
  protected completion, but the source classifier was too narrow.

### Decision From Run 47

AD-201 now covers obvious invalid-input probes as well as missing-input probes.
The command must include a recognizable bad input such as `invalid`, output
must show normal input-validation text, and crash markers still keep the result
in the strict repair lane. Positive inputs rejected as invalid remain
unexpected failures.

### Next Check

Run a clean Temperature JSON CLI canary with the patched harness and confirm
Engineer can complete product commit and ticket closure after positive,
missing-input, and invalid-input runtime validation, then reach QA/Security.

## Run 48: Engineer Rework Hit A Command-Procedure Build Trap

### Setup

Run 48 used a clean Temperature JSON CLI target at
`<validation-root>` with binary
`<validation-root>`.

### Observations

- CEO, COO, and CTO-weekly stayed product-first and created the expected
  Temperature JSON CLI implementation ticket.
- Engineer implemented the CLI, built
  `<validation-root>`, proved `25`, `-10`, and `0.5`
  Celsius conversions, then proved both missing-input and invalid-input
  negative paths without opening a runtime repair blocker.
- Engineer ran `go test ./...`, committed product work, moved `T-001` to
  done, and handed off for review. This confirmed the expanded AD-201
  classifier through product commit and ticket closure.
- QA requested changes because the delivered Go package had no `_test.go`
  file. That may be a legitimate product-quality expectation for a Go CLI,
  but it moved the run into Engineer rework.
- The rework Engineer ran `go build -o
  <validation-root> cmd/temperature-json-cli/`. Go rejected
  the package target because repo-relative package paths require
  `./cmd/temperature-json-cli`.
- The corrected command was obvious, but the Engineer repair-lane policy
  treated the first command as a real build failure and blocked corrected
  validation until a source edit occurred.

### Decision From Run 48

AD-202 extends validation-procedure failure classification to Engineer.
Recognizable Go package-target mistakes such as missing `./` on a
repo-relative package path are recorded as procedure failures, not product
test/build failures. Corrected validation stays available in the same job,
while real compile errors, failing tests, and target-owned build failures keep
the strict repair lane.

### Next Check

Run a clean Temperature JSON CLI canary with the patched harness and confirm
Engineer can recover from a `cmd/...` build-target mistake during review
rework without being forced into a meaningless source edit. If QA continues to
request tests, assess whether that is healthy target quality enforcement or an
overly narrow Go-specific review heuristic.

## Run 49: Surplus CLI Arguments Needed Negative-Path Classification

### Setup

Run 49 used a clean Temperature JSON CLI target at
`<validation-root>` with binary
`<validation-root>`.

### Observations

- CEO completed product-specific bootstrap inspection in 12 turns and handed
  off to COO.
- COO updated the active plan and feature contract for the Temperature JSON
  CLI in 13 turns, committed the planning change, and handed ticket breakdown
  to CTO-weekly.
- CTO-weekly created `T-001` for the product slice. It first tried to record
  completion with the new ticket uncommitted; the guardrail kept that as
  foundation telemetry only, then CTO-weekly committed the ticket and
  completed.
- Engineer hit the claim-first shell guard on an exploratory `ls`, then
  correctly claimed `T-001`, committed the claim, and continued.
- Engineer wrote `cmd/temperature-json-cli/main.go` with DocSync metadata and
  added `go.mod`.
- Engineer first tried a repo-local build output. The guardrail blocked it
  and named the corrected `<validation-root>` build path;
  Engineer used that path successfully.
- Engineer proved `<validation-root> 25` returned
  `{"celsius":25,"fahrenheit":77}`.
- Engineer proved `<validation-root>` with no arguments
  returned a missing-input error without opening the AD-201 failure lane.
- Engineer then ran `<validation-root> 25 30`. The CLI
  correctly returned `error: too many arguments provided`, but the runtime
  classifier treated that as an unexpected failure and blocked later runtime
  probes, build/test validation, and completion.
- The role tried to add tests; the first test write lacked DocSync metadata
  and was blocked, then the corrected write with DocSync metadata succeeded.
  The run was stopped there because the actionable classifier gap was clear.

### Decision From Run 49

AD-203 treats surplus-argument CLI probes as expected negative-path runtime
evidence when the command includes more than one product argument, output
clearly reports too many/surplus arguments, and no crash markers are present.
This keeps multi-argument positive paths strict because the output must name a
surplus-input validation failure.

### Next Check

Run a clean Temperature JSON CLI canary with the patched harness and confirm
Engineer can complete the product slice after positive, missing-input,
invalid-input, and surplus-argument runtime validation.

## Run 50: Test-Build Repair Needed Same-Job Test File Removal

### Setup

Run 50 used a clean Temperature JSON CLI target at
`<validation-root>` with binary
`<validation-root>`.

### Observations

- CEO completed product-specific bootstrap in 7 turns and handed off to COO.
- COO updated the active plan and feature contract, committed the change, and
  handed ticket breakdown to CTO-weekly.
- CTO-weekly created and committed `T-001`. It repeated the small
  uncommitted-ticket disposition stumble from run49, which remained foundation
  telemetry only and was corrected in-run.
- Engineer hit the claim-first shell guard, then claimed `T-001`, committed
  the claim, and continued.
- Engineer wrote a Go CLI with DocSync metadata, added `go.mod`, corrected a
  repo-local build-output guardrail by building `<validation-root>`,
  and proved positive Celsius conversions for `25`, `-10`, and `37.5`.
- Engineer used explicit `expected_exit_code: 1` for missing-input and
  invalid-input probes, so AD-203 was not exercised in this run.
- Engineer added Go tests, but the tests failed with duplicate helper/type
  declarations in test files.
- The repair lane correctly blocked runtime probes, build-command switching,
  ticket completion, and unrelated shell commands while the failing test was
  unresolved.
- The role then tried to remove the bad same-job test files with `rm`, but the
  repair lane blocked `rm` as unrelated shell work. With no file-delete tool,
  the role created more duplicate test files instead of repairing the failure.

### Decision From Run 50

AD-204 allows a narrow same-job cleanup path during Engineer test/build repair:
non-recursive `rm` or `unlink` may remove test-like files that the same job
wrote after the test/build failure began. Unmarked tests, source files,
recursive deletion, and ordinary cleanup remain blocked.

### Next Check

Run a clean Temperature JSON CLI canary with the patched harness and confirm
Engineer can recover from bad test-file generation by removing same-job repair
test files, rerunning the same-lane test successfully, and continuing toward
ticket evidence and review handoff.

## Run 51: QA Terminal Evidence Needed One Missed-Tool Correction

### Setup

Run 51 used a clean Temperature JSON CLI target at
`<validation-root>` with binary
`<validation-root>`.

### Observations

- CEO completed product-specific bootstrap and handed off to COO.
- COO rewrote the active plan and feature contract for the Temperature JSON
  CLI and committed the planning update.
- CTO-weekly created one product ticket, hit the known uncommitted-ticket
  disposition guard once, committed the ticket, and handed implementation to
  Engineer. The guardrail stayed foundation telemetry only.
- Engineer claimed `T-001`, wrote the Go CLI and Go tests with DocSync
  metadata, added `go.mod`, corrected the repo-local build-output guardrail by
  building `<validation-root>`, proved positive Celsius
  conversions, proved missing-input and invalid-input validation, ran
  `go test ./...`, passed `docsync_audit`, committed product work, moved
  `T-001` to done, and handed off to QA.
- QA observed a dirty `.harness/learnings.yaml` convention update created by
  the harness, inspected the done ticket and source/test files, then repeated
  the familiar Go package-target procedure mistake:
  `go build -o <validation-root> cmd/temperature-json-cli`.
- QA corrected the command to
  `go build -o <validation-root> ./cmd/temperature-json-cli`,
  and the build passed.
- Clean review evidence was then sufficient, but the next model response did
  not call `job_disposition_record`; the terminal-evidence boundary ended the
  job as `circle_detected`.
- Product progress was preserved, and the runtime failure remained
  foundation-owned telemetry instead of target intervention debt.

### Decision From Run 51

AD-205 keeps review evidence convergence strict but adds one bounded
missed-tool correction. The first non-terminal response after a clean
review-evidence reminder is rejected without executing the tool, then the loop
allows one more response that must call `job_disposition_record`. Repeated
misses still end with `circle_detected`.

### Next Check

Run a clean Temperature JSON CLI canary with the patched harness and confirm
QA converts the terminal-evidence correction into an approved or
changes-requested `job_disposition_record`.

## Run 52: Test-Build Repair Scope Allowed Alternate Root Implementation

### Setup

Run 52 used a clean Temperature JSON CLI target at
`<validation-root>` with binary
`<validation-root>`.

### Observations

- CEO completed in 12 turns and handed directly to CTO-weekly with
  ticket-shaping need.
- CTO-weekly created one product ticket and hit the known uncommitted-ticket
  disposition guard once, then committed the ticket and handed to Engineer.
- Engineer claimed `T-001`, committed the claim, initialized a Go module,
  wrote `cmd/temperature-json-cli/main.go`, and wrote
  `cmd/temperature-json-cli/main_test.go`.
- `go test ./cmd/temperature-json-cli/...` failed first because of an unused
  import, then after repair because the tests still failed behaviorally.
- The repair lane correctly blocked runtime side paths, destructive
  `rm -rf cmd/temperature-json-cli`, commits, ticket moves, and disposition.
- However, `file_write` allowed source writes anywhere while the package test
  failure remained outstanding. Engineer created root `main.go` and
  `main_test.go`, then repeated `go test ./...` while the original
  `cmd/temperature-json-cli` package remained failing.
- The run was stopped once the scope leak was clear. The stop produced
  operator-cancelled inference telemetry, but no target intervention-debt
  ticket.

### Decision From Run 52

AD-206 records a failed Go package repair scope for narrow test/build commands
and limits subsequent source/test/fixture writes to that scope until the
failure is repaired. Package/build config remains editable, and repo-wide
commands keep repo-wide repair behavior.

### Next Check

Run a clean Temperature JSON CLI canary with the patched harness and confirm a
failing package test leads to scoped package repair or a truthful blocked
disposition, not duplicate root implementation.

## Run 53: Release Tag Was Created Before The Release-Note Commit

### Setup

Run 53 used a clean Temperature JSON CLI target at
`<validation-root>` with binary
`<validation-root>`.

### Observations

- CEO, COO, CTO-weekly, Engineer, QA, Security, Dogfood, and Release Manager
  all ran in the generated target lifecycle.
- Engineer reproduced the Run 52 narrow package test failure shape, but the
  AD-206 scope guard kept repair work inside `cmd/temperature-json-cli/`.
  The role fixed `main_test.go`, passed package and repo tests, committed the
  product slice, moved `T-001` to done, and handed off to QA.
- QA approved in 8 turns after reading the done ticket and source files and
  running `go test ./cmd/temperature-json-cli/`, confirming the AD-205
  terminal-evidence path no longer lost a clean review.
- Security validated the product and committed a report, but after `date +%F`
  was blocked as non-validation shell it invented stale report date
  `2024-01-25`. Dogfood later used the correct `2026-05-21` date, so this is
  a narrower Security evidence-quality finding.
- Dogfood completed product validation and committed
  `docs/reports/dogfood/temperature-json-cli-validation-2026-05-21.md`.
- Release Manager first tried the installed `mars release notes`
  binary, hit `unknown command "release"`, then correctly used
  `mars_cli` to generate target release notes.
- Release Manager generated `VERSION=0.2.0` and `CHANGELOG.md`, but then
  created local tag `v0.2.0` at the previous Dogfood commit while those release
  files were still dirty. The disposition guard blocked completion, forcing the
  `release: notes 0.2.0` commit afterward, but the tag remained pointed at the
  pre-release-note commit.
- With no target remote configured, release publication stopped as
  `release_blocked`, no Orchestrator or Dogfood loop followed, and no target
  intervention-debt ticket was created.

### Decision From Run 53

AD-207 makes release tag placement a mechanical policy: `git tag vX.Y.Z`
through `shell_exec` is blocked unless `VERSION` matches, the worktree is
clean, `HEAD` is the `release: notes X.Y.Z` commit, and any explicit target
resolves to that `HEAD`. `git_release_guard` now reports stale local version
tags that do not point at the release-note commit.

### Next Check

Run another clean Temperature JSON CLI canary with the patched harness and
confirm Release Manager commits release notes before creating `v0.2.0`, or
records `release_blocked` without leaving a stale local tag.

## Run 54: Release Tag Placement Fixed; Date Drift Reproduced

### Setup

Run 54 used a clean Temperature JSON CLI target at
`<validation-root>` with binary
`<validation-root>`.

### Observations

- CEO, COO, CTO-weekly, Engineer, QA, Security, Dogfood, and Release Manager
  all ran in order and the target produced real product progress: plan,
  feature contract, ticket, implementation, tests, review, dogfood evidence,
  and release notes.
- The first COO handoff spent one extra loop selecting COO again before the
  second COO wrote the active plan and feature contract. It self-recovered, but
  remains turn waste.
- Engineer still emitted several empty `shell_exec argv: []` no-op calls. The
  current guardrails steered recovery through test repair, implementation
  commit, ticket evidence update, and lifecycle completion, so this was
  performance overhead rather than a product blocker.
- Security hit blocked broad grep and `date +%F` attempts, then still approved
  after successful `go test ./...`. Foundation-owned guardrail telemetry stayed
  out of the target backlog.
- Dogfood performed strong product validation, including external build,
  positive runtime probes, missing-input and invalid-input probes, focused
  tests, and docsync. It then wrote
  `docs/reports/dogfood/2024-05-21-dogfood-validation.md`, inventing a stale
  year despite the run occurring on 2026-05-21.
- Release Manager generated target release notes, committed
  `release: notes 0.2.0`, and only then created local tag `v0.2.0`. Pushing the
  tag failed because the throwaway target has no `origin`, and the job recorded
  a release blocker without dispatching another autonomous loop.

### Decision From Run 54

AD-207 is validated in the live target: release tags are now placed after the
release-note commit instead of on the pre-release Dogfood commit. AD-208 adds
non-droppable run metadata to every server job so report and evidence dates use
the actual run date instead of model memory or examples when shell time commands
are unavailable.

### Next Check

Run another clean canary with the patched harness and confirm Dogfood and
Security write evidence paths and report dates from `## RUN METADATA`
(`2026-05-21` for the current validation window), while Release Manager still
commits release notes before tagging.

## Run 55: Product Work Exists, But Duplicate Test Cleanup Is Too Narrow

### Setup

Run 55 used a clean Temperature JSON CLI target at
`<validation-root>` with binary
`<validation-root>`.

### Observations

- CEO, COO, CTO-weekly, and Engineer all received the non-droppable
  `run_metadata` context section, proving AD-208 context injection reached the
  live server path.
- CEO, COO, and CTO-weekly produced product-specific planning and a single
  ordinary product ticket. No automatic target intervention-debt tickets were
  created before product progress.
- Engineer claimed `T-001` and wrote real product files:
  `go.mod` plus `cmd/temperature-json-cli/main.go` and test files.
- The product implementation was close, but `go test ./cmd/temperature-json-cli`
  failed with duplicate test names and unused imports. Manual inspection after
  the run showed `cmd/temperature-json-cli/main_test.go`,
  `cli_test.go`, and `integration_test.go` conflicted.
- Policy correctly blocked runtime probes, build switching, helper work, ticket
  evidence, commits, and a false successful `job_disposition_record` while the
  failing test command remained unresolved.
- The role tried to delete a duplicate test with
  `rm -f cmd/temperature-json-cli/main_test.go`, but the AD-204 cleanup
  exception only allowed test files written after the failure began. The
  duplicate was written earlier in the same job, so cleanup was blocked and the
  job exhausted `max_turns`.
- The runtime failure stayed contained as foundation telemetry; no Orchestrator
  recovery loop or target intervention-debt ticket followed.

### Decision From Run 55

AD-209 records every successful Engineer `file_write` path, then allows
non-recursive `rm` or `unlink` of same-job test-like files during an unresolved
test/build repair lane. Pre-existing tests and source files remain protected.

### Next Check

Run another clean canary with the patched harness and confirm Engineer can
remove or rewrite duplicate generated tests, rerun the same-lane `go test`,
commit the product work, move `T-001` to done, and hand off to QA.

## Run 56: Test Cleanup Improves; QA Terminal Boundary Fires Before DocSync

### Setup

Run 56 used a clean Temperature JSON CLI target at
`<validation-root>` with binary
`<validation-root>`.

### Observations

- CEO, COO, and CTO-weekly again produced product-specific planning, a feature
  contract, and one ordinary product ticket. The target backlog was not
  polluted with intervention-debt tickets before product progress.
- The first Engineer completed the product slice and handed off to QA:
  `cmd/temperature-json-cli/main.go`, `go.mod`, README usage, ticket evidence,
  commits, and local push simulation were produced.
- QA correctly requested implementation rework because the initial slice had
  runtime evidence but no durable `_test.go` assertions for the explicit CLI
  examples and error paths.
- Orchestrator converted that QA disposition into Engineer rework on `T-001`.
  The second Engineer reopened the ticket, added
  `cmd/temperature-json-cli/cli_test.go`, passed
  `go test ./cmd/temperature-json-cli/`, committed
  `feat(cli): add comprehensive test coverage for temperature-json-cli
  (T-001)`, moved `T-001` back to done, and recorded a successful
  `qa_review` disposition.
- The second QA job read the done ticket, README, `main.go`, and `cli_test.go`,
  then ran `go test ./cmd/temperature-json-cli/` successfully.
- After the successful test, the runtime inserted terminal-only guidance for
  `job_disposition_record`. QA attempted `docsync_audit`, which is required by
  its role doctrine before approval, but the runtime rejected it as
  post-validation churn. The stronger correction also missed, and the QA job
  ended with `circle_detected`.
- The failure stayed foundation-owned telemetry. It did not route through
  Orchestrator and did not create a target intervention-debt ticket.

### Decision From Run 56

AD-209 is partially validated: the lifecycle moved past the previous Engineer
`max_turns` blocker and completed product rework with tests. AD-210 fixes the
new blocker by making review terminal convergence wait for `docsync_audit`
evidence before forcing `job_disposition_record`.

### Next Check

Run another clean canary with the patched harness and confirm the second QA
path runs `docsync_audit`, records an approved or changes-requested
`job_disposition_record`, and proceeds to the next lifecycle role instead of
ending with `circle_detected`.

## Run 57: DocSync Runs, But Build-Only Evidence Forces QA Terminal Too Early

### Setup

Run 57 used a clean Temperature JSON CLI target at
`<validation-root>` with binary
`<validation-root>`.

### Observations

- CEO, COO, and CTO-weekly again produced product-specific planning, a feature
  contract, and one ordinary product ticket before any intervention-debt work.
- A sandboxed first `start` attempt failed during port binding after seeding
  bootstrap work. The escalated retry proceeded, but registered the same repo
  path under a second repo ID. This remains a separate bootstrap-idempotency
  edge to inspect after the QA loop clears.
- Engineer completed the product slice, including `cmd/temperature-json-cli`
  source, tests, ticket evidence, commits, and local push simulation.
- QA read the ticket, BDD feature contract, README, implementation file, and
  ran `docsync_audit` successfully. This validates the AD-210 ordering fix:
  docsync was no longer blocked as post-validation churn.
- QA then ran `go build -o <validation-root>
  ./cmd/temperature-json-cli`. The runtime treated that build as sufficient
  terminal review evidence and forced `job_disposition_record` before QA ran
  the authoritative Go test command, even though
  `cmd/temperature-json-cli/main_test.go` existed.
- QA attempted more shell validation, received the stronger terminal-only
  correction, and ended with `circle_detected`.
- The failure stayed foundation-owned telemetry. It did not dispatch through
  Orchestrator and did not create target intervention-debt tickets.

### Decision From Run 57

AD-210 is validated but incomplete. AD-211 now requires review terminal
convergence to wait for a successful test command when test files exist, so
build-only evidence cannot prematurely cut off QA or Security.

### Next Check

Run another clean canary with the patched harness and confirm QA can proceed
from docsync and external build evidence into `go test`, then record a
structured `job_disposition_record` instead of ending with `circle_detected`.

## Run 58: Review No-Op Recovery Forced Terminal Before Required Tests

### Setup

Run 58 used a clean Temperature JSON CLI target at
`<validation-root>` with binary
`<validation-root>`.

### Observations

- CEO, COO, CTO-weekly, Engineer, QA, Orchestrator, and a second Engineer all
  ran against ordinary product work. The run again avoided early
  intervention-debt backlog pollution.
- The first QA job correctly requested Engineer rework for missing tests.
  Orchestrator routed the feedback to Engineer instead of creating
  intervention debt.
- The second Engineer added tests, committed the work, moved `T-001` to done,
  and handed back to QA. This confirmed the product lifecycle still made
  forward progress after rework.
- The second QA job read the done ticket, README, implementation files, and
  test files. It built `<validation-root>`, then called
  `shell_exec` with an empty argv before running the authoritative test command.
- The no-op policy incorrectly told QA to approve because validation had
  succeeded. The approval policy then rejected `job_disposition_record`
  because `_test.go` files existed and no test command had passed. The terminal
  correction repeated, and QA ended with `circle_detected`.
- The failure stayed foundation-owned telemetry. It did not dispatch through
  Orchestrator and did not create a target intervention-debt ticket.

### Decision From Run 58

AD-211 fixed the direct terminal convergence path but not no-op recovery.
AD-212 now prevents blocked review no-op failures from setting terminal-only
state unless the same evidence gates used for approval are satisfied. Missing
tests and missing docsync produce concrete next-action guidance instead of
approval guidance.

### Next Check

Run another clean canary with the patched harness and confirm QA can recover
from a no-op after build evidence by running tests, then recording a structured
`job_disposition_record`.

## Run 59: Product Lifecycle Reached Local Release, Release CLI Used Stale Shell Binary

### Setup

Run 59 used a clean Temperature JSON CLI target at
`<validation-root>` with binary
`<validation-root>`.

### Observations

- CEO, COO, and CTO-weekly produced a product-specific active goal, feature
  contract, execution plan, and one ordinary product ticket for the Temperature
  JSON CLI. No automatic intervention-debt tickets preempted product work.
- Engineer implemented `cmd/temperature-cli`, added tests, committed the
  feature, moved `T-001` to done, and handed off with build, runtime, and
  `go test ./cmd/temperature-cli` evidence.
- QA approved after reading the relevant product artifacts, building
  `<validation-root>`, running positive and negative runtime
  probes, running `go test ./cmd/temperature-cli`, and running
  `docsync_audit`. This validated AD-212: no-op recovery no longer forced
  premature approval before tests.
- Security and Dogfood both approved with independent validation evidence.
- The first Release Manager pass failed as liveness because it ran
  `mars release notes --repo . --bump auto --dry-run` through
  `shell_exec`. That resolved a stale installed binary which reported
  `unknown command "release" for "mars"`. The role read the
  `mars_cli` reference but repeated the stale shell command until loop
  containment forced a terminal disposition.
- Orchestrator recovered by dispatching Release Manager again. The second pass
  generated `VERSION` and `CHANGELOG.md`, committed `release: notes 0.2.0`,
  created tag `v0.2.0`, and stopped with a clean `release_blocked`
  disposition because the clean demo repo had no remote.
- The target worktree ended clean on `main` with tag `v0.2.0`.

### Decision From Run 59

AD-213 now blocks direct `shell_exec mars ...` invocations inside
agent jobs and routes MARS CLI workflows through `mars_cli`,
whose resolver prefers the active harness executable before PATH. Generated
Release Manager guidance names the structured args for release notes and
backfill so deployed release review should not fail solely because of a stale
installed binary.

### Next Check

Run another clean canary with the patched harness and confirm Release Manager
uses `mars_cli` for release notes on the first release pass, reaches
local release artifacts without a liveness retry, and stops only on real
publication blockers such as a missing remote.

## Run 60: Word Count CLI Reached Release With Structured Harness CLI

### Setup

Run 60 used a clean Word Count JSON CLI target at
`<validation-root>` with binary
`<validation-root>`.

The target brief asked for a generic non-game CLI that accepts one text
argument and prints JSON containing original text, word count, and character
count, with clear non-zero errors for missing or surplus arguments.

### Observations

- The lifecycle completed through CEO, COO, CTO-weekly, Engineer, QA,
  Orchestrator rework routing, second Engineer, QA approval, Security approval,
  Dogfood approval, and Release Manager.
- CEO and COO produced a product-specific active goal, plan, feature contract,
  and product ticket for the Word Count JSON CLI. No automatic
  intervention-debt ticket preempted product work.
- The first Engineer implemented the CLI and runtime probes, but QA requested
  rework because the Go source had no `_test.go` coverage. Orchestrator routed
  this to Engineer as ordinary product rework.
- The second Engineer added tests and passed `go test ./cmd/word-count`, but
  spent 36 turns and 35 tools, including repeated guardrail blocks and a long
  model pause after validation. This remains a pace signal, not a correctness
  blocker.
- QA, Security, and Dogfood approved with `go test`, runtime probes, and
  `docsync_audit` evidence.
- Release Manager called `mars_cli` for release notes at 17:28:19,
  committed `release: notes 0.2.0`, created tag `v0.2.0`, and stopped with a
  real `release_blocked` disposition because the temporary target had no
  configured remote. This validates AD-213 in the live release path.
- Startup retry evidence exposed a new persistence issue: the first sandboxed
  start registered the repo and enqueued a CEO job before bind failure, then the
  retry removed `demo-temp-run60.db-wal` and `demo-temp-run60.db-shm` before
  registering the same target with a new repo ID. The final DB did not retain
  duplicate jobs because WAL-backed state was discarded.
- The target ended clean on `main` at `release: notes 0.2.0` with local tag
  `v0.2.0`.

### Decision From Run 60

AD-213 is live-validated. AD-214 now changes automatic startup cleanup so it
never deletes SQLite `-wal` or `-shm` sidecars; it asks SQLite to recover or
checkpoint them and leaves sidecars in place if recovery fails. This preserves
repo registry and queued bootstrap state across retry-after-bind-failure runs.

### Next Check

Run another clean canary and confirm retry-after-bind-failure startup no longer
logs stale WAL/SHM deletion, while the normal product lifecycle still reaches
product planning and release review. Continue tracking Engineer rework pace
separately from correctness.

## Run 61: Slugify CLI Validated SQLite Retry, Exposed Rework Failure Guidance Gap

### Setup

Run 61 used a clean Slugify JSON CLI target at
`<validation-root>` with binary
`<validation-root>`.

The target brief asked for a generic CLI that accepts one title string and
prints JSON containing original input, a URL-friendly slug, and a word count.
Slug generation lowercases, trims whitespace, replaces runs of non-alphanumeric
characters with hyphens, and trims leading/trailing hyphens. Missing or surplus
arguments must fail with a clear non-zero error.

### Observations

- The first sandboxed `start` attempt initialized the repo, registered repo ID
  `3297728f-63d3-4a67-9262-f6535de8ff2a`, and enqueued one CEO bootstrap job
  `cc65a7ab-0fe7-4ee3-85cd-80479a63b1fc` before bind failure.
- The retry logged `cleanup: sqlite sidecar recovery checked`, reused the same
  repo ID and CEO job, and did not log stale sidecar deletion. This validates
  AD-214 for retry-after-bind-failure startup.
- CEO, COO, and CTO-weekly produced product-specific goals, plan, feature
  contract, and one ordinary product ticket before any intervention-debt work.
- Engineer implemented the CLI and moved `T-001` to done. QA correctly
  requested implementation rework because no `_test.go` assertions covered the
  behavior.
- Orchestrator routed the QA feedback to Engineer as product rework, not target
  intervention debt.
- The rework Engineer reopened the ticket and created
  `cmd/slugify-json/main_test.go`, but the tests exposed a real product
  mismatch: `countWords("Test@#$%Special Characters!")` returned `2` while the
  contract-aligned test expected `3`.
- After the failing `go test`, the guardrail correctly blocked unrelated
  runtime probes, builds, ticket moves, commits, and successful disposition
  while the test/build lane was unresolved. However, the policy guidance named
  only the unresolved command, not the failing assertion output. The role spent
  9m44s, 25 tool calls, and 10 Engineer guardrail blocks without repairing the
  implementation. The operator stopped the run, producing an `llm_unreachable`
  blocker and leaving the failing test uncommitted.
- No automatic intervention-debt target tickets were created. The failure stayed
  foundation-owned telemetry, which is correct for runtime/guardrail friction.

### Decision From Run 61

AD-214 is live-validated. AD-215 now records the latest failing test/build
output in the unresolved-validation session state and includes that output in
subsequent guardrail guidance. The guidance also tells Engineer to edit the
implementation, not delete or weaken tests, when the failing assertion matches
the ticket, README, or BDD contract.

### Next Check

Run another clean non-game canary and confirm Engineer rework can use the
explicit failing-output guidance to repair implementation behavior, rerun the
same-lane test successfully, commit the test/source fix, and hand back to QA
without target intervention-debt churn.

## Run 62: Slugify CLI Completed Full Local Lifecycle After Rework Guidance

### Setup

Run 62 used a fresh Slugify JSON CLI target at
`<validation-root>` with binary
`<validation-root>`.

The target brief repeated the Run 61 Slugify JSON CLI contract: accept exactly
one title string, print JSON fields `original_input`, `slug`, and `word_count`,
lowercase and trim slug input, collapse runs of non-alphanumeric characters to
hyphens, trim surrounding hyphens, count words as alphanumeric runs, and return
clear non-zero errors for missing or surplus arguments.

### Observations

- Startup initialized the target, registered repo ID
  `1c989713-6957-4856-b42c-d1e9748140d6`, and enqueued one CEO bootstrap job
  `c5bfd50d-27b5-4a37-9008-58f2941c14db`.
- The lifecycle completed through CEO, COO, CTO-weekly, Engineer, QA,
  Security, Dogfood, and Release Manager. All nine jobs ended `completed`.
- CEO, COO, and CTO-weekly produced product-specific goals, an active plan,
  a Slugify feature contract, and one ordinary product ticket. No automatic
  intervention-debt ticket preempted product work.
- Engineer claimed `T-001`, implemented `cmd/slugify`, added Go tests, ran
  runtime probes and `go test ./cmd/slugify`, committed the feature, moved the
  ticket to done, and handed off to QA.
- QA, Security, and Dogfood independently approved using runtime probes,
  `go test`, and `docsync_audit` evidence. The target finished with a clean
  worktree and `go test ./...` passing.
- Release Manager used `mars_cli` for release notes, committed
  `release: notes 0.2.0`, created tag `v0.2.0`, checked release status, and
  stopped on the expected missing-remote publication blocker. This was a real
  environment blocker for the temp target, not a lifecycle loop.
- Guardrail events stayed foundation-owned telemetry:
  `engineer=3`, `qa=3`, `security=3`, and `cto-weekly=1`. No target
  intervention-debt tickets were created.
- The target changelog included release notes and delivery evidence, but the
  wording remains generic in places because target release-note generation is
  summarizing local semantic commits without deeper product copy editing.

### Decision From Run 62

AD-215 is live-validated. The patched guidance was sufficient for the next
fresh Slugify canary to reach implementation, tests, QA, Security, Dogfood,
local release notes, and a release publication blocker without re-entering the
Run 61 unresolved test/build churn.

No source-code fix is required from this run. The next improvement target is
the validation matrix itself: include remote-backed release canaries and a
broader mix of non-CLI application archetypes so the factory is not optimized
only around small command-line targets.

### Next Check

Run the next canary against a different application shape with a configured
remote or explicit remote-publication simulation, then confirm the release path
publishes or blocks for a genuinely actionable reason while product delivery
remains product-specific and intervention-debt-free.

## Run 63: Notes API Canary Exposed Missing Module Bootstrap Trap

### Setup

Run 63 used a clean HTTP JSON API target at
`<validation-root>` with binary
`<validation-root>`.

The target brief asked for a small dependency-light Go API for personal notes:
create, list, fetch, update, delete, tag search, free-text search, validation
errors, tests, and README usage. Unlike earlier no-remote canaries, this target
had a local bare `origin` at
`<validation-root>`.

### Observations

- The first sandboxed `start` initialized the deployed harness and enqueued CEO
  before the local sandbox blocked port binding. The escalated retry reused the
  same repo ID `69c62c6a-e3ec-44c3-812c-2817da716409` and CEO job
  `a9dcb9d7-059c-4043-8584-cae01bc3e03d`, validating bootstrap idempotency
  again.
- CEO, COO, and CTO-weekly completed. They produced product-specific Notes API
  planning, updated the active feature contract, and created one ordinary
  product ticket `T-001`; no automatic intervention-debt ticket preempted the
  product backlog.
- Engineer initially hit the claim-first ticket guardrail, and the signal was
  logged as foundation telemetry rather than target backlog. Engineer then
  claimed `T-001`, committed the claim, and `git_push` successfully pushed to
  the local bare `origin`.
- Engineer created Go source for `cmd/demo-notes-api`, `internal/note`,
  `internal/handlers`, and `internal/server`, plus a focused test file and
  README updates.
- The first `go test ./internal/note` failed because the fresh target had no
  `go.mod`: Go reported `cannot find main module` and suggested
  `go mod init`.
- The unresolved test/build repair guardrail correctly blocked runtime probes,
  product commits, ticket moves, and unrelated shell work while keeping all
  guardrail signals as foundation telemetry. However, it also blocked
  `go mod init demo-notes-api`, the direct package-config repair.
- After that block, the role tried worse recovery paths: deleting the test,
  committing product work anyway, removing directories, and creating
  placeholder tests. The operator stopped the run at Engineer turn 41.
- DB state at stop: `ceo`, `coo`, and `cto-weekly` were `completed`; Engineer
  was still `running`; telemetry contained `engineer|guardrail_block|14`; the
  target worktree was dirty with uncommitted application files.

### Decision From Run 63

AD-216 now treats missing Go module bootstrap as bounded test/build repair.
When the latest failing output proves Go cannot find a main module and `go.mod`
is absent, Engineer may run `go mod init <module>` before rerunning same-lane
validation. The exception remains closed when the module file exists or the
failing output is not missing-module evidence.

### Next Check

Rerun a fresh Notes API or other HTTP-service canary and confirm Engineer can
recover by running `go mod init`, rerun same-lane tests, commit product work,
continue to QA, and keep intervention-debt signals quarantined as telemetry.

## Run 64: Notes API Canary Avoided Module Trap, Exposed Dependency And Test-Evidence Holes

### Setup

Run 64 used a fresh copy of the Notes API target at
`<validation-root>`, local bare
remote `<validation-root>`,
and patched binary `<validation-root>`.

### Observations

- CEO, COO, and CTO-weekly again completed product-specific planning and one
  ordinary product ticket. Guardrail blocks for broad discovery and
  commit-before-disposition stayed foundation telemetry.
- Engineer hit the claim-first guardrail, recovered by moving `T-001` to
  in-progress, committed and pushed the claim to the local bare `origin`, then
  began implementation.
- Unlike Run 63, Engineer wrote `go.mod` before validation, so the new
  `go mod init` repair exception was not exercised by this live replay.
- Engineer committed and pushed product implementation before first running
  `go test ./...`. The subsequent test failed on an unused import, which
  Engineer repaired and committed forward.
- Engineer then added tests but also ran raw
  `go get github.com/stretchr/testify`, mutating dependency state outside
  `dependency_sync` and contrary to the brief's standard-library preference.
- The new tests exposed assertion failures. The repair lane blocked switching
  test lanes and unchanged reruns, but still allowed deletion of a same-job
  test file even though the failure was ordinary assertion evidence rather than
  duplicate/generated-test conflict.
- The operator stopped the run at Engineer turn 46. DB state at stop:
  `ceo`, `coo`, and `cto-weekly` completed; Engineer remained running;
  telemetry included `engineer|guardrail_block|6`, `ceo|guardrail_block|1`,
  and `cto-weekly|guardrail_block|1`.

### Decision From Run 64

AD-217 now blocks raw `go get` as dependency mutation and narrows same-job test
cleanup to duplicate/generated-test shaped failures. Assertion failures must
preserve the test and be repaired through implementation/test edits plus
same-lane validation. Patched binary
`<validation-root>` confirmed the raw
`go get` block through `mars tools run shell_exec`, returning:
`policy: shell_exec command "go get" mutates dependency state; use
dependency_sync so workspace hygiene preflight and postflight run`.

### Next Check

Run another HTTP-service canary and confirm raw dependency mutation is blocked
before `go.mod`/`go.sum` changes, assertion-failure test deletion is blocked,
and Engineer either repairs the failing behavior or records an honest blocked
handoff without target intervention-debt creation.

## Run 65: Inventory API Canary Reached Product Rework, Exposed Post-Validation No-Op Failure

### Setup

Run 65 used a fresh Inventory API target at
`<validation-root>`, local bare
remote `<validation-root>`,
database `<validation-root>`,
log `<validation-root>`,
and patched binary `<validation-root>`.
The brief asked for a small standard-library Go HTTP JSON service for inventory
items, quantities, and reorder thresholds.

### Observations

- The harness initialized the deployed target, committed the generated harness
  baseline, registered one repo, and seeded exactly one CEO bootstrap job.
- CEO, COO, and CTO-weekly completed product-specific planning from the README:
  active goal, exec plan, canonical `F-001-product-walking-skeleton.md`, and
  ordinary product ticket `T-001`.
- Engineer claimed `T-001`, pushed the claim to the local bare `origin`, wrote
  a Go module and product source, passed `go build -o <validation-root>`
  after a normal unused-import repair, passed `go test ./internal/inventory`,
  committed and pushed implementation, updated ticket evidence, moved `T-001`
  to done, and recorded `qa_review`.
- QA approved with `go test ./internal/inventory`, `make test`, external build,
  and `docsync_audit`. Security approved after `go test ./...`.
- Dogfood ran end-to-end validation and found a target-owned product bug: the
  generated HTTP server registered conflicting `/items` handlers. It created
  normal target ticket `T-002`, committed and pushed it, and recorded
  `changes_requested`.
- Orchestrator routed `T-002` back to Engineer. The second Engineer job claimed
  and pushed the ticket, repaired the route registration, passed external build,
  `go test ./...`, `go test ./internal/inventory`, started the server with
  `background:true`, and successfully probed `curl http://localhost:8080/health`.
- After that successful runtime evidence, Engineer called no-op `shell_exec`
  placeholders instead of stopping the tracked server, committing the dirty
  route repair, updating evidence, moving `T-002` to done, and handing back to
  QA. The job ended as `circle_detected`.
- DB summary after stopping the run: `ceo`, `coo`, `cto-weekly`, first
  `engineer`, `qa`, `security`, `dogfood`, and `orchestrator` completed; second
  `engineer` failed with `executor: agent ended with circle_detected`.
  Telemetry stayed foundation-owned: `ceo|guardrail_block|3`,
  `cto-weekly|guardrail_block|1`, `engineer|guardrail_block|8`,
  `engineer|circle_detected|1`, `security|guardrail_block|3`, and
  `dogfood|guardrail_block|8`.

### Decision From Run 65

AD-218 now blocks the first Engineer no-op placeholder after successful
validation when implementation or ticket files are dirty. The guardrail points
to tracked background PID cleanup, `git_status`, implementation commit, ticket
evidence update, ticket move to done, lifecycle commit, push, and
`job_disposition_record`. This keeps the useful validation path open while
removing the generic no-op gap that produced `circle_detected`.

### Next Check

Rerun an Inventory/API-style canary or resume a comparable rework path and
confirm Engineer stops the tracked background process, commits the route repair,
updates and closes the rework ticket, pushes, and hands back to QA without
entering no-op loops.
