# Factory Forward Progress Guard Live Run

Date: 2026-06-16, updated 2026-06-17
Source ref: initial run `52eed0b`; latest sequential proof used the current
source checkout after the source fixes listed below and `make install`.
Binary: `build/mars-harness` built from this checkout, reporting
`mars-harness 0.62.1 darwin/arm64`
Run roots: `<validation-root>`,
`<validation-root>`,
`<validation-root>`,
`<validation-root>`
Owner: foundation

## Primary Outcome Contract

**Primary Outcome:** Greenfield `mars-harness start` reaches Engineer and records
first successful build/smoke evidence across the static web, Phaser/browser-game,
and Go API validation targets.

**Primary Pass Gate:** Each selected target reaches Engineer, records first
product mutation, records first successful build/smoke evidence, and avoids a
planner or CTO loop before that evidence.

**Primary Status:** `primary_passed`

**Current Primary Blocker:** None for the first-slice Engineer build/smoke
gate. Full lifecycle completion after QA/Security and parallel farm execution
remain separate, unclaimed validation objectives.

**Next Primary Action:** Keep this report scoped to the proven first-slice gate,
then use a separate primary outcome contract for any follow-up that claims full
lifecycle completion, concurrent model-farm execution, or QA/Security release
readiness.

**Supporting Evidence:** The static, Phaser/browser-game, and Go API sequential
runs each reached Engineer, recorded product mutation, completed the first
implementation ticket, and recorded build/smoke evidence without planner or CTO
handoff blockage. Earlier parallel and failed sequential attempts remain below
as evidence for the source fixes and non-primary runner limitations.

## Current Aggregate Result

| Target | Latest evidence root | Primary gate |
| --- | --- | --- |
| Static web | `<validation-root>` | Passed: Engineer completed with product HTML, `node --check`, served-page smoke, and done ticket. |
| Phaser/browser-game | `<validation-root>` | Passed: Engineer completed with Phaser/Vite product files, `npm run build`, runtime smoke assertion, and done ticket. |
| Go API | `<validation-root>` | Passed: Engineer completed with Go source/tests, `go build`, `go test`, `curl` evidence, done ticket, and QA handoff. |

## Scope

This report records a model-backed live validation attempt for AD-297 Factory
Forward Progress Guard across three clean ephemeral targets:

- static web focus timer
- Phaser browser game
- Go reading-list API

The run intentionally used the real `mars-harness start` lifecycle path, real
target repositories, per-target SQLite databases, real local inference, and
parallel start processes. A subagent captured independent observations and
actions at:

```text
<validation-root>
```

This was not a fake endpoint or deterministic fake-LLM test.

## Setup

The installed command at `/path/to/local-redacted` reported
`0.62.0`, while the committed source reported `0.62.1`. To avoid validating a
stale binary, the run used a fresh source build:

```bash
env GOCACHE=<validation-root> GOMODCACHE=<validation-root> go build -o build/mars-harness ./cmd/mars-harness
build/mars-harness version
```

Each target was initialized as a clean git repo with a committed README product
brief before `start` ran. The first sandboxed launch seeded CEO jobs but failed
to bind worker ports with `operation not permitted`. The escalated rerun used
the same repos and DBs and correctly resumed the queued CEO jobs instead of
reseeding them.

Representative run command shape:

```bash
env MARS_HARNESS_WEBHOOK_PORT=19101 \
  MARS_HARNESS_DASHBOARD_PORT=19201 \
  MARS_HARNESS_SKIP_START_CLEANUP=1 \
  MARS_HARNESS_LOG_FILE=<validation-root> \
  build/mars-harness start \
    --repo <validation-root> \
    --db <validation-root> \
    --concurrency 1 \
    --log-file <validation-root>
```

Equivalent commands used ports `19102`/`19202` for Phaser and `19103`/`19203`
for Go.

## Results

| Target | Final observed state | Classification |
| --- | --- | --- |
| Static web | CEO and COO completed. CTO remained running until the operator stopped the diagnosed replay. A backlog ticket was created, but CTO encountered repeated scenario-order/ticket guardrail blocks and no Engineer job started. | Foundation-owned routing/tool-policy wedge; no deployed product failure. |
| Phaser game | CEO and COO completed. CTO remained running until the operator stopped the diagnosed replay. No Engineer job started, no Phaser source/package/build/smoke existed, and `.harness/learnings.yaml` plus backlog changes were left dirty. | Foundation-owned lifecycle/guardrail/inference contention wedge; no deployed product failure. |
| Go API | CEO completed, CTO created implementation tickets, Engineer started, implemented product code, ran validation, moved T-001 to done, and handed off to QA. QA was stopped by the operator after evidence collection. | Partial pass for forward progress through Engineer; QA not evaluated in this run. |

## Timeline Evidence

Times below are local BST. SQLite output stores the same timestamps in UTC.

| Target | Role sequence | First Engineer | First product mutation | First successful validation |
| --- | --- | --- | --- | --- |
| Static web | CEO completed 21:24:17; COO completed 21:30:29; CTO stopped 21:50:05 | Not reached | Not reached | Not reached |
| Phaser game | CEO completed 21:25:12; COO completed 21:30:45; CTO stopped 21:50:05 | Not reached | Not reached | Not reached |
| Go API | CEO completed 21:26:48; CTO completed 21:35:47; Engineer completed 21:48:04; QA stopped 21:50:06 | 21:35:47 | Product file writes began during Engineer; implementation commit `e995847` | `go test ./...` passed at 21:42:57; guarded build reran as `go build -o <validation-root> ./cmd/api` |

Go API evidence:

```text
?   	reading-list-api/cmd/api	[no test files]
ok  	reading-list-api/internal/api	0.328s
```

Go API commits:

```text
05f6969 chore(tickets): move T-001 to done
a327e58 chore(tickets): move T-001 to done
e995847 feat(api): implement minimal Go HTTP API for reading-list items (T-001)
5b48115 chore(tickets): claim T-001
fc43545 tickets: create implementation tickets for current scenario
```

## 2026-06-16 CTO First-Slice Handoff Rerun

### Primary Outcome Contract

**Primary Outcome:** Greenfield `mars-harness start` reaches Engineer and
records first successful build/smoke evidence for static web,
Phaser/browser-game, and Go API targets.

**Primary Pass Gate:** Each target reaches Engineer, gets product mutation, and
records first build/smoke evidence without planner or CTO handoff blockage.

**Primary Status:** `primary_failed`

**Current Primary Blocker:** The source fix for CTO first-slice handoff did not
receive live lifecycle proof because the three parallel local-inference runs did
not leave CEO before the bounded rerun was stopped.

**Next Primary Action:** Rerun on sufficient harness-native runner capacity, or
a machine farm, until the static web, Phaser/browser-game, and Go API targets
all reach Engineer product mutation and build/smoke evidence.

**Supporting Evidence:** Unit tests, docs consistency, generated CTO prompt
checks, and docsync passed for the first-slice policy. The live rerun proved
startup isolation again and exposed runner-capacity limits, but did not prove
the primary lifecycle outcome.

### Source Change Evidence

The source fix changed CTO handoff from pre-proof batching to exact first-slice
handoff:

- Fresh first-proof mode requires one ordinary first-slice ticket for the active
  current failing scenario or earliest uncovered product scenario.
- A grouped first ticket such as `F-001-S001,F-001-S002` no longer satisfies CTO
  handoff before first proof.
- Post-proof backlog expansion only unlocks after a done product ticket has
  build/smoke evidence, not generic evidence metadata.
- Generated target CTO guidance now says one first-slice ticket before broader
  expansion.

Validation passed before live rerun:

```bash
go test ./internal/tools ./internal/scanner ./internal/docsconsistency/...
go test ./cmd/mars-harness ./internal/serve ./internal/agent
build/mars-harness tools run docsync_audit --repo . --args-json '{}'
git diff --check
go build -o build/mars-harness ./cmd/mars-harness
```

### Live Rerun Setup

Run root:

```text
<validation-root>
```

Commands:

```bash
build/mars-harness start --repo <validation-root> --db <validation-root> --log-file <validation-root> --debug
build/mars-harness start --repo <validation-root> --db <validation-root> --log-file <validation-root> --debug
build/mars-harness start --repo <validation-root> --db <validation-root> --log-file <validation-root> --debug
```

No fake endpoint or scripted model fallback was used.

### Live Rerun Result

| Target | Startup Action | Final Observed State | Primary Gate Result |
| --- | --- | --- | --- |
| Static web | `startup_action=seeded_ceo` | CEO interrupted after 15m38s, 3 turns, 4 tool invocations. No CTO, Engineer, product mutation, build, or smoke. | Failed |
| Phaser/browser-game | `startup_action=seeded_ceo` | CEO interrupted after 15m39s, 4 turns, 5 tool invocations. No CTO, Engineer, product mutation, build, or smoke. | Failed |
| Go API | `startup_action=seeded_ceo` | CEO interrupted after 15m50s, 2 turns, 3 tool invocations. No CTO, Engineer, product mutation, build, or smoke. | Failed |

SQLite job evidence after stopping:

```text
static-web: ceo|failed|2026-06-16 23:35:36|2026-06-16 23:51:51|executor: agent loop error (llm_unreachable): context canceled
phaser-game: ceo|failed|2026-06-16 23:35:36|2026-06-16 23:51:52|executor: agent loop error (llm_unreachable): context canceled
go-api: ceo|failed|2026-06-16 23:35:36|2026-06-16 23:51:53|executor: agent loop error (llm_unreachable): context canceled
```

The `llm_unreachable/context canceled` category was caused by the operator stop
after diagnosing that the rerun had not reached the primary code path. The
evidence before stop showed very slow local generation under three parallel
30B-model servers, with no lifecycle role handoff beyond CEO.

### Conclusion

At the end of this 2026-06-16 attempt, the implementation was
source-validated but not live primary-validated. This attempt remained
`primary_failed` because no real greenfield run had reached Engineer
build/smoke evidence on all three target types.

## 2026-06-17 Sequential First-Slice Reruns

### Primary Outcome Contract

**Primary Outcome:** Greenfield `mars-harness start` reaches Engineer and
records first successful build/smoke evidence for static web,
Phaser/browser-game, and Go API targets.

**Primary Pass Gate:** Each target reaches Engineer, gets product mutation, and
records first build/smoke evidence without planner or CTO handoff blockage.

**Primary Status:** `primary_passed`

**Current Primary Blocker:** None for the first-slice Engineer build/smoke
gate. The Go API rerun4 target reached Engineer, recorded product mutation,
moved T-001 to `done/`, recorded `go build`, `go test`, and `curl` evidence, and
handed to QA.

**Next Primary Action:** Do not treat this as a full lifecycle or QA/Security
release-readiness pass. Any broader claim needs a separate primary outcome
contract and live proof.

**Supporting Evidence:** Static and Phaser sequential runs prove the earlier
CTO/COO blockers can be removed target by target. The Go rerun4 target proves
the post-mutation convergence blocker was cleared for the first-slice Go API
case after the Makefile build-output guard steered validation away from
repo-local binaries.

### Source Changes Validated Before Sequential Reruns

Validation passed for the source changes exercised by these runs:

```bash
go test ./internal/tools
go test ./internal/personas
go test ./internal/scanner ./internal/docsconsistency/...
go test ./cmd/mars-harness ./internal/serve ./internal/agent
go build -o build/mars-harness ./cmd/mars-harness
build/mars-harness tools run docsync_audit --repo . --args-json '{}'
make install
```

Source fixes exercised:

- Static browser smoke evidence now requires a served product page response,
  rejects directory listings/repository indexes, rejects reserved inference
  ports, and ignores canned console-output validation.
- Capability matching accepts split left/right/down movement scenarios for a
  combined directional movement brief.
- Fresh CTO first-slice handoff uses the earliest concrete first scenario
  before first proof, while still skipping obvious process/governance-only
  placeholder scenarios.
- `ticket_create` and CTO implementation handoff now agree on Go API
  `F-001-S001` rather than contradicting each other with `F-001-S003`.
- `make build` is blocked when its Makefile recipe writes Go build output into
  the target repository, steering validation toward `go test ./...` or
  `<validation-root>` artifacts.

### Sequential Run Results

| Target | Run Root | Startup Action | Final Observed State | Target Gate |
| --- | --- | --- | --- | --- |
| Static web | `<validation-root>` | `startup_action=seeded_ceo` | CEO, COO, CTO, Engineer, and QA completed. Engineer created product source, recovered from reserved port guidance, ran `node --check src/main.js`, served the product on `127.0.0.1:5173`, and `curl -fsS http://127.0.0.1:5173/` returned actual product HTML. QA reran syntax and served-page smoke and approved. Security was intentionally stopped after the primary target gate. | Passed |
| Phaser/browser-game | `<validation-root>` | `startup_action=seeded_ceo` | Failed before CTO because COO capability policy did not recognize split left/right/down movement scenarios as covering the combined movement brief. | Failed, fixed in source |
| Phaser/browser-game | `<validation-root>` | `startup_action=seeded_ceo` | CEO, COO, CTO, Engineer, and QA completed. CTO handed to Engineer. Engineer created Phaser/Vite product files, ran `npm run build`, and ran the policy-required `node -e` Phaser source/runtime smoke assertion that printed `browser smoke: Phaser canvas #game new Phaser.Game`. QA approved. Security was stopped after primary evidence collection. | Passed |
| Go API | `<validation-root>` | `startup_action=seeded_ceo` | Failed at CTO. `ticket_create` correctly rejected `F-001-S003` because earliest uncovered was `F-001-S001`, while CTO handoff policy incorrectly demanded `F-001-S003`, creating an unrecoverable policy contradiction. | Failed, fixed in source |
| Go API | `<validation-root>` | `startup_action=seeded_ceo` | CEO, COO, and CTO completed; CTO handed to Engineer. Engineer claimed T-001 and wrote buildable Go source, tests, Makefile, and `go.mod`, but left files dirty with `bin/` build output and persisted no Engineer trace/disposition before operator stop. Manual `go test ./...` passed afterward. | Failed |
| Go API | `<validation-root>` | `startup_action=seeded_ceo` | CEO, COO, and CTO completed; CTO handed to Engineer. Engineer claimed T-001 and wrote buildable Go source/tests without leaving a `bin/` artifact, but remained running with dirty `README.md`, `Makefile`, `cmd/`, `go.mod`, and `internal/`, persisted no Engineer trace/disposition, and recorded no harness-owned build/test evidence before operator stop. Manual `go test ./...` passed afterward. | Failed |
| Go API | `<validation-root>` | `startup_action=seeded_ceo` | CEO, COO, and CTO completed; CTO handed to Engineer. Engineer recovered from Makefile build-output policy blocks, wrote a safe `<validation-root>` build target, committed product source/tests, moved T-001 to `done/`, and recorded `go build ./cmd/go-api`, `go test ./internal/server`, and `curl -v http://localhost:8080/health` evidence before handing to QA. The run was stopped after QA started because QA/Security were outside this primary gate. | Passed |

### Key Evidence

Static target:

```text
CEO completed 01:03:10
COO completed 01:05:08
CTO completed 01:06:58
Engineer completed 01:10:39
QA approved 01:12:00
Ticket: docs/tickets/done/T-001-implement-static-focus-timer-page-with-basic-controls.md
Evidence: node --check src/main.js; python3 -m http.server 5173 --bind 127.0.0.1; curl -fsS http://127.0.0.1:5173/
```

Phaser target:

```text
CEO completed
COO completed
CTO completed and handed to Engineer
Engineer completed and handed to QA for T-001
Engineer evidence: npm run build; node -e Phaser source/runtime smoke assertion
QA approved with build/smoke evidence
Ticket: docs/tickets/done/T-001-establish-project-brief-as-visible-product-slice.md
```

Go API final target:

```text
Run root: <validation-root>
CEO completed 01:39:45
COO completed 01:44:43
CTO completed 01:46:05 and handed to Engineer
Engineer completed 01:52:18 and handed to QA for T-001
Ticket: docs/tickets/done/T-001-implement-go-http-server-with-health-endpoint.md
Engineer evidence: go build ./cmd/go-api; go test ./internal/server; curl -v http://localhost:8080/health
Commits: 895c22e feat: implement Go HTTP server with /health endpoint (T-001); 154bf3a chore(tickets): move T-001 to done
```

### Classification

| Finding | Evidence | Classification |
| --- | --- | --- |
| Static first-slice lifecycle can reach Engineer build/smoke sequentially after the smoke-evidence fixes. | Static run root and done ticket evidence above. | foundation-owned pass evidence |
| Phaser first-slice lifecycle can reach Engineer build/smoke sequentially after the directional movement capability fix. | Phaser rerun2 trace/dispositions and done ticket evidence. | foundation-owned pass evidence |
| Go first-slice CTO handoff contradiction was foundation-owned and fixed: `ticket_create` and handoff now agree on `F-001-S001`. | Go rerun first failure, source tests, and rerun4 CTO handoff to Engineer. | foundation-owned fixed blocker |
| Go Engineer convergence now passes for the first-slice target after the Makefile build-output guard. | Go rerun4 Engineer completed, ticket moved to done, and evidence recorded before QA handoff. | foundation-owned pass evidence |
| QA/Security completion is intentionally unclaimed. | Go rerun4 was stopped after QA started; static and Phaser were also stopped after primary evidence collection. | evidence-only for separate lifecycle validation |

### Conclusion

This follow-up meets the stated first-slice primary outcome: static web,
Phaser/browser-game, and Go API all reached Engineer, product mutation, and
build/smoke evidence without planner or CTO handoff blockage. It does not prove
full lifecycle completion, QA/Security quality, or parallel farm execution.

## What Worked

- Parallel live `start` processes ran against isolated target repos and DBs.
- Restart safety worked: after the first bind failure seeded CEO jobs, the
  escalated rerun printed `startup_action=resumed_lifecycle` and resumed those
  jobs instead of seeding duplicate CEOs.
- No target showed repeated CEO/COO planner oscillation before Engineer.
- Sequential static web, Phaser/browser-game, and Go API runs reached CTO
  handoff, Engineer implementation, successful build/smoke evidence, ticket
  done, and QA handoff.
- Tool policy protected Go from an unsafe repo-local `go build ./cmd/api` and
  required the external validation binary path. The final Go rerun recovered
  from those policy blocks and completed the Engineer disposition.

## What Did Not Work

- Earlier parallel full lifecycle runs shared local inference ports. Reasoning roles
  contended for `127.0.0.1:18081`, producing repeated bind failures, restart
  churn, and `inference server exceeded restart limit` log lines.
- `mars-harness start` has no public `--model-endpoint` or per-run inference
  port override, so operators cannot cleanly isolate parallel lifecycle runs
  without code/config support.
- Earlier static web, Phaser, and Go API reruns exposed source policy blockers
  before the final sequential evidence passed. Those failures remain useful
  evidence for why the policy fixes were needed.
- QA/Security/full lifecycle completion was not evaluated by this report.
  Sequential runs were stopped after the primary first-slice evidence was
  collected, so later-role quality cannot be inferred from this pass.

## Earlier Parallel Run DB State

Static web:

```text
ceo         completed
coo         completed
cto-weekly  failed     executor: agent loop error (llm_unreachable): context canceled
```

Phaser game:

```text
ceo         completed
coo         completed
cto-weekly  failed     executor: agent loop error (llm_unreachable): context canceled
```

Go API:

```text
ceo         completed
cto-weekly  completed
engineer    completed
qa          failed     executor: agent loop error (llm_unreachable): context canceled
```

The `context canceled` failures were caused by operator stop after diagnosis.
They are not the root failure classification for static/Phaser and are not a QA
quality result for Go.

## Latest Sequential Gate DB State

Static web:

```text
ceo         completed
coo         completed
cto-weekly  completed
engineer    completed
qa          completed
security    stopped by operator after primary evidence collection
```

Phaser/browser-game:

```text
ceo         completed
coo         completed
cto-weekly  completed
engineer    completed
qa          completed
security    stopped by operator after primary evidence collection
```

Go API:

```text
ceo         completed
coo         completed
cto-weekly  completed
engineer    completed
qa          running when operator stopped after primary evidence collection
```

## Follow-Up Actions

Immediate source follow-ups:

- Add inference server ownership/locking for concurrent `start` processes so
  they either reuse one managed server safely or allocate non-conflicting ports.
- Add a public, validated way for `start` to use an explicit real model endpoint
  or inference port when running farmed/parallel lifecycle validation.
- Convert guardrail-loop telemetry into an operator-visible route or terminal
  disposition when the same blocked action repeats without useful state change.
- Record when a guardrail block is remediated so successful recovery does not
  leave only `remedied=false` telemetry.

Validation follow-ups:

- Use a separate primary outcome contract for full lifecycle completion through
  QA, Security, Release, and Dogfood if that becomes the next claim.
- Use a separate primary outcome contract for farmed/parallel execution because
  this report proves sequential first-slice progress, not concurrent runner
  economics.
- Add a dedicated parallel-start smoke that asserts independent target DBs and
  non-conflicting inference/runtime ports before running long model sweeps.

## Conclusion

The new factory behavior is proven for the stated first-slice primary gate.

Sequential live runs on static web, Phaser/browser-game, and Go API reached
Engineer, product mutation, successful build/smoke evidence, ticket closure,
and QA handoff without planner or CTO handoff blockage. This does not claim
parallel farm readiness or full lifecycle completion beyond the first Engineer
proof.
