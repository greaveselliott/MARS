# Mars Harness — Master Execution Plan

**Status:** Active
**Created:** 2026-04-11
**Source:** [delivery-schedule.md](delivery-schedule.md) + [docs/tickets/](../../tickets/)

---

## How to read this document

This is the **agent-operable source of truth** for the Mars Harness build. It cross-references:

- **Delivery schedule milestones** (M0–M10) with task-level checkboxes.
- **Ticket acceptance criteria** (MH-001–MH-028) with per-ticket status.
- **Conflict resolutions** where ticket and schedule disagree.
- **Dependency graph** dictating execution order.

**Checkbox convention:**
- `[x]` = done and verified
- `[~]` = partial (code exists, not all AC met)
- `[ ]` = not started
- `[SKIP: reason]` = blocked by external dependency

---

## Traceability matrix

| MH | Title | Milestone | Primary packages | Status |
|----|-------|-----------|-----------------|--------|
| MH-001 | LLM client | M1.1 | `internal/llm` | Partial |
| MH-002 | Tool system | M1.2 | `internal/tools` | Partial |
| MH-003 | Conversation loop | M1.3 | `internal/agent` | Partial |
| MH-004 | Context assembly | M1.4 | `internal/context` | Partial |
| MH-005 | Execution trace | M1.5 | `internal/trace` | Partial |
| MH-006 | Hardware + registry | M2.1, M2.2 | `internal/hardware` | Not started |
| MH-007 | Model download | M2.3 | `internal/models` | Not started |
| MH-008 | llama-server + router | M2.4, M2.5 | `internal/inference` | Not started |
| MH-009 | CLI + bundle + demo | M3 | `cmd/mars-harness`, `internal/bundle`, `internal/ui` | Not started |
| MH-010 | GitHub API client | M4.1 | `internal/github` | Not started |
| MH-011 | GitHub App setup | M4.2 | `internal/github` | Not started |
| MH-012 | Webhook receiver | M4.3 | `internal/github` | Not started |
| MH-013 | Job queue | M5a.1, M5a.2 | `internal/queue` | Not started |
| MH-014 | Scheduler | M5a.3 | `internal/scheduler` | Not started |
| MH-015 | Sandbox + safety | M5b.1–M5b.3 | `internal/sandbox`, `internal/safety` | Not started |
| MH-016 | `serve` command | M5b.4 | `cmd/mars-harness` | Not started |
| MH-017 | Accuracy scoring | M6.1, M6.2 | `internal/scoring` | Not started |
| MH-018 | Progressive autonomy | M6.3, M6.4 | `internal/trust` | Not started |
| MH-019 | Intervention detector | M7.1 | `internal/evolution` | Not started |
| MH-020 | Reviewer meta-role | M7.2, M7.3 | `internal/evolution` | Not started |
| MH-021 | Guardrails engine | M7.4 | `internal/guardrails` | Not started |
| MH-022 | Setup wizard | M8.1 | `cmd/mars-harness` | Not started |
| MH-023 | Init + scanner | M8.2, M8.3 | `internal/scanner` | Not started |
| MH-024 | Dashboard | M9 | `internal/dashboard` | Not started |
| MH-025 | Mars prompt port | M10.1–M10.4 | `examples/`, `bundles/` | Not started |
| MH-026 | Dogfood | M10.5 | `.github/workflows/` | Not started |
| MH-027 | Distribution | M10.6 | `.github/workflows/`, `scripts/` | Not started |
| MH-028 | Doctor command | M8.4 | `cmd/mars-harness` | Not started |

---

## Conflict resolutions applied

| ID | Conflict | Winner | Action |
|----|----------|--------|--------|
| C1 | MH-009 CLI: `run <role>` positional vs `--role` flag | Schedule | Update ticket to positional |
| C2 | MH-009 `--dry-run` missing from ticket | Schedule | Add to ticket AC |
| C3 | MH-009 bundle path: `bundles/` vs `examples/` | Ticket | Errata note in schedule context |
| C4 | MH-012 event types: `push` vs `workflow_run`/`merge_group` | Union | Support all; update both |
| C5 | MH-012 dedup window: 1-hour vs delivery-id only | Schedule | Add time window to ticket |
| C6 | MH-013 status names: queued/running vs pending/claimed | Ticket | Errata in schedule context |
| C7 | MH-013 TTL: 30-day missing from ticket | Schedule | Add to ticket |
| C8 | MH-014 cron source: config+schedules.yaml vs manifest | Schedule | Update ticket |
| C9 | MH-015 macOS sandbox: sandbox-exec vs cwd+ulimit | Ticket | sandbox-exec deprecated; errata |
| C10 | MH-015 blast radius: per-file+total vs single | Schedule | Update ticket with finer controls |
| C11 | MH-016 health endpoint: /healthz vs /api/status | Schedule | Update ticket |
| C12 | MH-017 scoring window: 30-day vs 20-jobs | Schedule | Update ticket |
| C13 | MH-017 formula: simple ratio vs weighted | Schedule | Update ticket for v1 |
| C14 | MH-018 trial: runs-to-promote vs expires-and-reverts | Schedule | Update ticket |
| C15 | MH-020 auto-disable: 3 worsening vs score-post-merge | Schedule | Update ticket |
| C16 | MH-021 staleness: 90 days vs 30 days | Schedule | Update ticket |
| C17 | MH-022 test mode: --test-mode vs --dry-run+--skip-* | Ticket | --test-mode aliases --skip-download --skip-github |
| C18 | MH-023 ticket output: .harness/tickets/ vs docs/tickets/ | Schedule | Update ticket |
| C19 | MH-023 roles.yaml vs manifest.yaml | Schedule | Ticket references both |

---

## Dependency graph

```
M0 ──► M1 ──┬──► M2 ──► M3 ──┐
             │                 ├──► M8 ──► M10
             ├──► M4 ──┐      │
             │          ├──► M5b ──┬──► M9
             ├──► M5a ──┤          │
             │          ├──► M6 ──►├──► M9
             │          │    │     │
             │          │    ▼     │
             │          │   M7 ────┘
             │          │    │
             │          │    ▼
             │          │   M10
             │          │
             └──────────┘
```

**Parallel lanes after M1:**
- **Lane A (local):** M2 → M3
- **Lane B (online):** M4 + M5a (concurrent) → M5b → M6 → M7
- **Merge:** M8 needs both lanes; M9 needs Lane B through M7; M10 needs everything.

---

## Milestone 0: Repo Foundation

**Objective:** Establish Go module, directory structure, CI, documentation conventions, and test infrastructure.

### Tasks

- [x] 0.1 Go module init, Go 1.22+, CGO_ENABLED=0
- [x] 0.2 Directory structure (cmd/, internal/, pkg/, docs/, .cursor/rules/, testdata/)
- [x] 0.3 AGENTS.md
- [x] 0.4 Tenets document
- [x] 0.5 Cursor rules (Go conventions, docs discipline, no-push-to-main)
- [~] 0.6 CI setup
  - [x] 0.6.1 golangci-lint
  - [x] 0.6.2 go test with race detector
  - [x] 0.6.3 go build compile check
  - [ ] 0.6.4 Coverage report upload
- [~] 0.7 Test infrastructure
  - [x] 0.7.1 Table-driven conventions in AGENTS.md
  - [x] 0.7.2 testify
  - [~] 0.7.3 pkg/testutil/ (exists, completeness unverified)
- [x] 0.8 README
- [x] 0.9 Design doc stubs (index.md + AD-001 through AD-015)
- [x] 0.10 Package layout decision (AD-014)
- [x] 0.11 Error handling convention
- [ ] 0.12 Logging decision — slog usage not yet present in code
- [ ] 0.13 Configuration decision — no config package exists

### Quality gate

- [x] `go build ./cmd/mars-harness` produces a binary
- [x] `go test ./...` passes
- [ ] `golangci-lint run` reports zero issues (not verified recently)
- [x] CI workflow runs on push and PR
- [x] AGENTS.md exists
- [x] tenets.md exists
- [x] design-docs/index.md exists

---

## Milestone 1: Agent Runtime — MH-001 through MH-005

**Objective:** LLM client, tool system, conversation loop, context assembly, execution traces.

### M1.1 LLM client (MH-001) — Partial

- [x] 1.1.1 OpenAI-compatible HTTP client
- [x] 1.1.2 Request/response types
- [x] 1.1.3 Configurable base URL, API key, model, temperature, max tokens
- [x] 1.1.4 Retry with backoff
- [x] 1.1.5 Token counting estimation

**Ticket AC gaps:** Timeout error message wording (PARTIAL), agent-runtime.md specifics (PARTIAL).

### M1.2 Tool system (MH-002) — Partial

- [x] 1.2.1 Tool registry with JSON Schema definitions
- [x] 1.2.2 Tool executor with timeout and cwd boundary
- [x] 1.2.3 Core tools: file_read, file_write, file_search, grep, shell_exec, git_*
- [x] 1.2.4 Tool result formatting (truncation, binary detection)
- [x] 1.2.5 Tool permission model (allowlist)

**Ticket AC gaps:** Exact "tool not found" wording (PARTIAL), wrong-arg-type test (PARTIAL).

### M1.3 Conversation loop (MH-003) — Partial

- [x] 1.3.1 Multi-turn orchestration
- [x] 1.3.2 Error recovery for malformed responses
- [ ] 1.3.3 Token budget enforcement in loop
- [ ] 1.3.4 Circle detection (3 consecutive → nudge, 5 → abort)
- [ ] 1.3.5 Conversation compaction
- [~] 1.3.6 Robust tool call parser (exists; needs real model output fixtures)

**Ticket AC gaps:** No 3+ tool-call integration test (primary AC NOT MET). Circle detection, budget, compaction not implemented. Boolean corruption test partial.

### M1.4 Context assembly (MH-004) — Partial

- [x] 1.4.1 System prompt builder (role + guardrails + knowledge + trigger + repo)
- [ ] 1.4.2 Scope-aware file inclusion (.harnessignore, max size, binary exclusion)
- [ ] 1.4.3 Priority-based context packing
- [ ] 1.4.4 Token budget partitioning (system vs history vs tools)

**Ticket AC:** All current AC met. Schedule items 1.4.2–1.4.4 not started.

### M1.5 Execution trace (MH-005) — Partial

- [x] 1.5.1 JSONL format with timestamps, role, content, tool calls, token counts
- [x] 1.5.2 Write traces to file
- [x] 1.5.3 Trace metadata header
- [~] 1.5.4 Trace tail/follow (recorder supports io.Writer; not CLI wired)

**Ticket AC gaps:** loop.go swallows trace-store error (PARTIAL). 5-turn test uses synthetic user lines not realistic conversation (PARTIAL).

### M1 Quality gate

- [~] Agent loop completes multi-turn with mock LLM (works but no 3+ tool-call test)
- [x] Core tools tested individually
- [~] Malformed tool calls handled (parser exists; loop test partial)
- [ ] Budget exceeded terminates cleanly
- [ ] Circle detection triggers
- [x] Context assembly respects scope and budget
- [ ] Integration test: mock LLM replaying real recordings
- [x] Trace file written and parseable

---

## Milestone 2: Local Inference — MH-006 through MH-008

**Objective:** Hardware detection, model registry, download, llama-server management, LLM router.

### M2.1 Hardware detection (MH-006)

- [ ] 2.1.1 Detect RAM, CPU cores, GPU (Metal on macOS, CUDA on Linux)
- [ ] 2.1.2 Recommend quantization level by VRAM/RAM
- [ ] 2.1.3 Output hardware summary to slog and config

### M2.2 Model registry (MH-006)

- [ ] 2.2.1 Registry data structure (name, params, quant, RAM, context, URL, SHA256)
- [ ] 2.2.2 Default model set (coding/reasoning/fast tiers)
- [ ] 2.2.3 Custom model support (GGUF path or HuggingFace repo)

### M2.3 Model download (MH-007)

- [ ] 2.3.1 HTTP download with resume (Range headers)
- [ ] 2.3.2 Progress reporting via callback
- [ ] 2.3.3 SHA256 verification
- [ ] 2.3.4 Store in ~/.mars-harness/models/
- [ ] 2.3.5 Skip if exists and checksum matches

### M2.4 llama.cpp server (MH-008)

- [ ] 2.4.1 Download llama-server binary (platform-specific) with checksum
- [ ] 2.4.2 Start subprocess with model path, context length, GPU layers, port
- [ ] 2.4.3 Health check loop (/health, timeout 60s)
- [ ] 2.4.4 Graceful restart on model switch
- [ ] 2.4.5 Graceful shutdown (SIGTERM → wait 10s → SIGKILL)
- [ ] 2.4.6 Multi-model support (multiple instances, different ports)

### M2.5 LLM router (MH-008)

- [ ] 2.5.1 Route by role → model tier
- [ ] 2.5.2 Ensure target server running before dispatch
- [ ] 2.5.3 Fallback to configured remote API

### M2 Quality gate

- [ ] Hardware detection correct on macOS and Linux [SKIP: needs hardware]
- [ ] Model download end-to-end with resume and checksum
- [ ] llama-server starts, health check, serves completions
- [ ] Agent runtime works with real local model [SKIP: needs hardware]
- [ ] Router maps roles to tiers and starts servers on demand
- [ ] Fallback to remote API works

---

## Milestone 3: `mars run` — First Demo — MH-009

**Objective:** CLI framework, bundle reader, terminal trace output, sample bundle, end-to-end demo.

### Tasks

- [ ] 3.1 CLI framework (cobra, global flags, run subcommand, version)
- [ ] 3.2 Bundle reader (manifest.yaml, role prompts, guardrails, knowledge routes, bundle hash)
- [ ] 3.3 Terminal trace output (colour-coded, streaming, summary footer, --quiet, non-TTY)
- [ ] 3.4 Sample bundle (Pipeline Fixer demo with broken TypeScript fixture)

### Quality gate

- [ ] `mars-harness run pipeline-fixer --repo testdata/...` executes with colour output
- [ ] Trace streams in real-time
- [ ] Missing bundle gives actionable error with available bundles list
- [ ] --dry-run shows plan without calling LLM
- [ ] Sample bundle diagnoses and fixes type error
- [ ] Demo recorded

---

## Milestone 4: GitHub Integration — MH-010 through MH-012

**Objective:** GitHub API client, App manifest flow, webhook receiver, wire into agent tools.

### Tasks

- [ ] 4.1 GitHub API client (JWT auth, PAT fallback, PR/check/repo ops, rate limiting, pagination)
- [ ] 4.2 GitHub App manifest flow (setup command, callback, credential storage, installation discovery)
- [ ] 4.3 Webhook receiver (HTTP server, HMAC validation, event normalization, dedup, routing)
- [ ] 4.4 PAT fallback mode (polling when no App configured)
- [ ] 4.5 Wire GitHub tools into agent (replace stubs)

### Quality gate

- [ ] App created via manifest flow [SKIP: needs credentials]
- [ ] Webhook signature rejects tampered payloads
- [ ] Events normalized (table-driven with real payloads)
- [ ] Dedup rejects replayed delivery IDs
- [ ] Agent creates PR on test repo [SKIP: needs credentials]
- [ ] PAT fallback works
- [ ] Rate limiting sleeps on 403

---

## Milestone 5a: Pipeline Mechanics — MH-013, MH-014

**Objective:** SQLite job queue, worker dispatcher, cron scheduler.

### Tasks

- [ ] 5a.1 SQLite job queue (schema, lifecycle, per-repo serialization, idempotency, TTL)
- [ ] 5a.2 Worker dispatcher (concurrency, pool, per-job timeout, graceful shutdown)
- [ ] 5a.3 Cron scheduler (expression parsing, timezone, missed schedule, sources)

### Quality gate

- [ ] FIFO dispatch, per-repo serialization
- [ ] Idempotency dedup
- [ ] Scheduler emits jobs on tick
- [ ] Graceful shutdown completes running jobs
- [ ] TTL pruning

---

## Milestone 5b: Safety Layer — MH-015, MH-016

**Objective:** Sandbox, blast radius, emergency stop, `mars-harness serve`.

### Tasks

- [ ] 5b.1 Process sandbox (Linux namespaces, macOS cwd+ulimit fallback)
- [ ] 5b.2 Blast radius controls (max files, max lines per file, max total lines, PR rate limit, no-delete, secret scanner)
- [ ] 5b.3 Emergency stop (stop command, GitHub cleanup, audit)
- [ ] 5b.4 `mars-harness serve` (wire webhook + queue + worker + scheduler + safety, /healthz)

### Quality gate

- [ ] Sandbox isolates writes
- [ ] Blast radius blocks excessive diffs
- [ ] Secret scanner catches patterns
- [ ] Emergency stop halts and cleans up [SKIP: needs credentials for GitHub cleanup]
- [ ] `serve` starts and accepts webhooks
- [ ] /healthz returns status

---

## Milestone 6: Accuracy and Autonomy — MH-017, MH-018

**Objective:** Outcome tracking, scoring engine, progressive autonomy, CLI commands.

### Tasks

- [ ] 6.1 Outcome tracking (monitor events, match to jobs, store in SQLite, 48h timeout)
- [ ] 6.2 Scoring engine (30-day rolling window, per-role per-repo, min 5 outcomes)
- [ ] 6.3 Progressive autonomy (observer/contributor/autonomous, promotion/demotion, enforcement, override)
- [ ] 6.4 CLI commands (scores, trust, trust set)

### Quality gate

- [ ] Each outcome type scored correctly
- [ ] Rolling 30-day score accurate
- [ ] Promotion observer → contributor → autonomous
- [ ] Demotion on score drop
- [ ] Observer cannot create PRs
- [ ] CLI displays correct data

---

## Milestone 7: Self-Improvement and Guardrails — MH-019 through MH-021

**Objective:** Intervention detection, Reviewer meta-role, guardrails engine, evolution PRs.

### Tasks

- [ ] 7.1 Intervention detector (detect, classify, store)
- [ ] 7.2 Reviewer meta-role (trace analysis, root cause, prompt evolution proposals, rate limit, auto-disable)
- [ ] 7.3 Evolution PR creation (branch naming, PR body, outcome tracking)
- [ ] 7.4 Guardrails engine (advisory/hard, override, staleness 90-day, inheritance)
- [ ] 7.5 CLI commands (interventions, interventions show)

### Quality gate

- [ ] Clear interventions classified correctly
- [ ] Reviewer produces plausible analysis
- [ ] Evolution PR created with correct format
- [ ] Reviewer cannot modify its own prompt
- [ ] Max 1 evolution per role per day
- [ ] Auto-disable after 3 worsening evolutions
- [ ] Advisory guardrails in system prompt
- [ ] Hard guardrails block violations
- [ ] Override logged
- [ ] Stale guardrails flagged

---

## Milestone 8: Setup and Init — MH-022, MH-023, MH-028

**Objective:** Setup wizard, init + scanner, doctor command.

### Tasks

- [ ] 8.1 `mars-harness setup` (orchestrated flow, idempotent, --test-mode alias)
- [ ] 8.2 `mars-harness init` (scaffold .harness/, roles.yaml, starter tickets, exec plans, AGENTS.md stub, schedules.yaml)
- [ ] 8.3 Repo scanner (detect language/framework/CI, find gaps, ticket generation)
- [ ] 8.4 `mars-harness doctor` (system/model/GitHub/pipeline/accuracy health, coloured output, --json)

### Quality gate

- [ ] setup completes in test mode
- [ ] Re-run skips completed steps
- [ ] init scaffolds correct structure
- [ ] Scanner finds missing tests, TODOs
- [ ] Scanner skips node_modules, generated files
- [ ] doctor reports accurate health
- [ ] doctor provides fix instructions

---

## Milestone 9: Dashboard — MH-024

**Objective:** Embedded web dashboard with real-time updates.

### Tasks

- [ ] 9.1 Dashboard infrastructure (HTTP server, html/template, htmx, Chart.js, SSE, embed.FS, responsive)
- [ ] 9.2 Pipeline Flow page (layered layout, SVG, live state, click-to-detail)
- [ ] 9.3 Role Health page (cards, accuracy trend, context usage, recent outcomes)
- [ ] 9.4 Throughput page (job stats, inference metrics, GitHub API, resource usage)
- [ ] 9.5 Debug page (timeline, trace viewer, live trace, webhook log, error log)
- [ ] 9.6 Evolution History page (timeline, entries, score impact)
- [ ] 9.7 Emergency stop button (confirm dialog, triggers stop, cleanup progress)

### Quality gate

- [ ] Dashboard loads self-contained at :9090
- [ ] Pipeline graph renders
- [ ] Nodes update live via SSE
- [ ] Role health shows real scoring data
- [ ] Trace viewer displays conversations
- [ ] Live trace streams tokens
- [ ] Emergency stop halts jobs
- [ ] Responsive down to 1024px

---

## Milestone 10: Mars Migration and Distribution — MH-025 through MH-027

**Objective:** Port all role prompts, dogfood, cross-compile, distribute.

### Tasks

- [ ] 10.1–10.3 Port all 11 Mars prompts (3 batches)
- [ ] 10.4 Validate each ported role against Mars monorepo
- [ ] 10.5 Dogfood (Pipeline Fixer, QA, Code Reviewer on mars-harness itself)
- [ ] 10.6 Distribution (cross-compile, curl installer, Homebrew, GitHub Releases, <30MB binary)
- [ ] 10.7 Documentation (quickstart, tenets reference, bundle reference, guardrails guide, model guide, architecture)

### Quality gate

- [ ] All 11 roles ported with working bundles
- [ ] At least 7 roles equivalent to Cursor automation
- [ ] Dogfood: Pipeline Fixer fixes deliberate CI failure
- [ ] Dogfood: QA produces meaningful suggestions
- [ ] curl installer works on Linux and macOS [SKIP: needs credentials]
- [ ] Homebrew installs and runs [SKIP: needs credentials]
- [ ] Quickstart enables zero-to-first-run

---

## Execution protocol

1. **Branch:** `feat/m{major}-{slug}` or `feat/mh-NNN-short`; never commit to main.
2. **Ticket state:** backlog/ → in-progress/ on start; → done/ on AC completion.
3. **Conflict resolution:** Check Conflict Register; apply resolution; update ticket in same PR.
4. **Implement:** Smallest vertical slice; code + table-driven tests.
5. **Quality:** `go test ./... -race`, `go vet ./...`, `golangci-lint run`; ≥70% coverage.
6. **Docs:** Update Discoveries in relevant design doc; update index.md if new AD.
7. **Commit:** Conventional Commit referencing milestone + ticket.
8. **PR:** Merge order follows dependency graph.

**Failure recovery:**
- Test failure requiring redesign → WIP commit, document blocker, continue to next task.
- Blocked by prerequisite → record in ticket Notes, switch to prerequisite.
- Partial completion → commit passing code, create follow-up ticket.
- Needs credentials/hardware → mark [SKIP: needs X], implement testable parts.
