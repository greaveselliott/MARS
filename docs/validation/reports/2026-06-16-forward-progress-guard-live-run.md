# Factory Forward Progress Guard Live Run

Date: 2026-06-16
Source ref: `52eed0b`
Binary: `build/mars-harness` built from this checkout, reporting `mars-harness 0.62.1 darwin/arm64`
Run root: `<validation-root>`
Owner: foundation

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

## What Worked

- Parallel live `start` processes ran against isolated target repos and DBs.
- Restart safety worked: after the first bind failure seeded CEO jobs, the
  escalated rerun printed `startup_action=resumed_lifecycle` and resumed those
  jobs instead of seeding duplicate CEOs.
- No target showed repeated CEO/COO planner oscillation before Engineer.
- The Go API path reached CTO ticketing, Engineer implementation, successful
  test/build validation, ticket done, and QA handoff.
- Tool policy protected Go from an unsafe repo-local `go build ./cmd/api` and
  required the external validation binary path.

## What Did Not Work

- Parallel full lifecycle runs share local inference ports. Reasoning roles
  contended for `127.0.0.1:18081`, producing repeated bind failures, restart
  churn, and `inference server exceeded restart limit` log lines.
- `mars-harness start` has no public `--model-endpoint` or per-run inference
  port override, so operators cannot cleanly isolate parallel lifecycle runs
  without code/config support.
- Static web CTO became trapped in scenario/ticket ordering guardrails:
  `ticket_create` was blocked for later scenarios while the job failed to settle
  a valid small ticket batch and hand off to Engineer.
- Phaser CTO did not reach Engineer before the replay was stopped and left
  dirty harness/ticket state.
- Go Engineer hit a `guardrail_loop` signal after a failing `go test ./...`
  lane and repeated blocked shell commands. It recovered and completed, but the
  loop was only telemetry/evolution evidence; the running job was not stopped or
  routed by the loop signal.

## Final DB State

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

## Follow-Up Actions

Immediate source follow-ups:

- Add inference server ownership/locking for concurrent `start` processes so
  they either reuse one managed server safely or allocate non-conflicting ports.
- Add a public, validated way for `start` to use an explicit real model endpoint
  or inference port when running farmed/parallel lifecycle validation.
- Convert guardrail-loop telemetry into an operator-visible route or terminal
  disposition when the same blocked action repeats without useful state change.
- Fix CTO ticket-shaping guardrails so static/web and Phaser projects can create
  the first small implementation ticket batch without becoming trapped by
  scenario-order policy.
- Record when a guardrail block is remediated so successful recovery does not
  leave only `remedied=false` telemetry.

Validation follow-ups:

- Rerun the three-target live proof after inference isolation and CTO ticket
  shaping fixes.
- Keep the Go API target as a positive partial proof of the forward-progress
  route through Engineer.
- Add a dedicated parallel-start smoke that asserts independent target DBs and
  non-conflicting inference/runtime ports before running long model sweeps.

## Conclusion

The new factory behavior is partially proven, not fully proven.

The Go API target demonstrates the intended forward path through CEO/CTO,
Engineer implementation, successful validation, ticket closure, and QA handoff.
The static web and Phaser targets did not reach Engineer, and the parallel run
also exposed shared inference-port contention. Those are foundation-owned
failures that must be fixed before claiming the forward progress guard works
across the required project matrix.
