# Mars Harness — Delivery Schedule

**Status:** Active
**Date:** 2026-04-11
**Author:** Elliott Greaves

---

## Overview

Exhaustive delivery schedule for the mars-harness project: a standalone Go binary that runs AI agent roles locally against GitHub repositories. Each milestone includes numbered tasks, a quality gate, and architecture decisions made in that phase.

Total estimated duration: **41–53 working days** across 11 milestones (M0–M10).

---

## Milestone 0: Repo Foundation (1 day)

### Tasks

- **0.1** Go module init (`github.com/greaveselliott/mars-harness`), set Go version floor (1.22+), configure `CGO_ENABLED=0`
- **0.2** Directory structure
  - **0.2.1** `cmd/mars-harness/` — entrypoint
  - **0.2.2** `internal/` — domain-grouped packages (`agent/`, `llm/`, `tools/`, `github/`, `pipeline/`, `safety/`, `scoring/`, `dashboard/`, `config/`, `scanner/`)
  - **0.2.3** `pkg/` — public library code (types, errors, testutil)
  - **0.2.4** `docs/` — design docs, exec plans, tenets
  - **0.2.5** `.cursor/rules/` — agent rules for the harness repo
  - **0.2.6** `testdata/` — shared fixtures
- **0.3** `AGENTS.md` — package graph, directory map, key constraints, build/test commands
- **0.4** Tenets document (`docs/tenets.md`) — local-first, accuracy over speed, safety by default, observable, self-improving
- **0.5** Cursor rules
  - **0.5.1** Go conventions rule (error handling, naming, test style)
  - **0.5.2** Documentation discipline rule (mirrors Mars pattern)
  - **0.5.3** No-push-to-main rule
- **0.6** CI setup
  - **0.6.1** GitHub Actions workflow: `golangci-lint` (vet, staticcheck, errcheck, gosimple)
  - **0.6.2** `go test ./...` with race detector
  - **0.6.3** `go build -o /dev/null ./cmd/mars-harness` (compile check)
  - **0.6.4** Coverage report upload
- **0.7** Test infrastructure
  - **0.7.1** Table-driven test conventions documented in `AGENTS.md`
  - **0.7.2** `testify` for assertions and mocks
  - **0.7.3** `pkg/testutil/` — shared helpers (temp dirs, fixture loaders, mock LLM)
- **0.8** README with project description, quickstart placeholder, architecture overview
- **0.9** Design doc stubs
  - **0.9.1** `docs/design-docs/index.md` — table of contents
  - **0.9.2** Stub files for each AD (AD-001 through AD-015)
- **0.10** Package layout decision — domain-grouped under `internal/` with explicit public surface in `pkg/`
- **0.11** Error handling convention — always return errors, never `panic` in library code, wrap with `fmt.Errorf("context: %w", err)`
- **0.12** Logging decision — `log/slog` from stdlib, structured JSON in production, text in development
- **0.13** Configuration decision — YAML config file (`~/.mars-harness/config.yaml`) merged with environment variable overrides (`MARS_HARNESS_` prefix)

### Quality Gate

- [ ] `go build ./cmd/mars-harness` produces a binary without errors
- [ ] `go test ./...` passes (at least one test exists in `pkg/testutil/`)
- [ ] `golangci-lint run` reports zero issues
- [ ] CI workflow runs green on push and PR
- [ ] `AGENTS.md` exists and documents directory structure, build commands, constraints
- [ ] `docs/tenets.md` exists with all five tenets
- [ ] `docs/design-docs/index.md` exists and lists all AD stubs

### Architecture Decisions

| ID | Decision |
|----|----------|
| AD-001 | Pure Go, `CGO_ENABLED=0`, single static binary |
| AD-002 | Apache 2.0 license |
| AD-003 | Mars conventions adopted (exec plans, design docs, AGENTS.md, cursor rules) |
| AD-013 | `log/slog` stdlib for all structured logging |
| AD-014 | Domain-grouped packages under `internal/` |
| AD-015 | YAML config file + environment variable overrides |

---

## Milestone 1: Agent Runtime (5–7 days)

### Tasks

- **1.1** LLM client interface
  - **1.1.1** OpenAI-compatible HTTP client (chat completions, streaming, tool calls)
  - **1.1.2** Request/response types matching OpenAI API schema
  - **1.1.3** Configurable base URL, API key, model, temperature, max tokens
  - **1.1.4** Retry with exponential backoff on 429/5xx
  - **1.1.5** Token counting (tiktoken-compatible estimation)
- **1.2** Tool system
  - **1.2.1** Tool registry — register tools by name with JSON Schema parameter definitions
  - **1.2.2** Tool executor — dispatch tool calls, capture stdout/stderr/exit code, enforce per-tool timeout
  - **1.2.3** Core tools: `read_file`, `write_file`, `list_directory`, `search_text` (ripgrep), `run_command`, `patch_file`
  - **1.2.4** Tool result formatting — truncation for large outputs, binary detection
  - **1.2.5** Tool permission model — allow/deny lists per role
- **1.3** Conversation loop
  - **1.3.1** Multi-turn orchestration: system prompt → user message → assistant → tool calls → tool results → loop
  - **1.3.2** Error recovery — malformed assistant responses retried with correction prompt
  - **1.3.3** Token budget enforcement — track cumulative tokens, warn at 80%, terminate at limit
  - **1.3.4** Circle detection — detect repeated identical tool call sequences (3 consecutive), inject nudge, abort after 5
  - **1.3.5** Conversation compaction — summarize earlier turns when approaching context window
  - **1.3.6** Robust tool call parser — handle model-specific format variations (JSON in markdown fences, partial JSON, multiple tool calls in single response, XML-style tool use); test suite built from real model output recordings
- **1.4** Context assembly
  - **1.4.1** System prompt builder — role prompt + guardrails + knowledge routes + repo context
  - **1.4.2** Scope-aware file inclusion — respect `.harnessignore`, max file size, binary exclusion
  - **1.4.3** Priority-based context packing — fill context window with highest-relevance files first
  - **1.4.4** Token budget partitioning — reserve tokens for system, conversation history, and tool results
- **1.5** Execution trace recorder
  - **1.5.1** Structured trace format (JSON Lines): timestamps, role, content, tool calls, tool results, token counts, latency
  - **1.5.2** Write traces to `~/.mars-harness/traces/{job_id}.jsonl`
  - **1.5.3** Trace metadata header (job ID, role, repo, model, start time, config snapshot)
  - **1.5.4** Trace tail/follow support for real-time streaming

### Quality Gate

- [ ] Agent loop completes multi-turn conversation with mock LLM (at least 5 turns with tool calls)
- [ ] All core tools tested individually with table-driven tests
- [ ] Malformed tool calls (broken JSON, missing params, unknown tool) handled gracefully with retry
- [ ] Budget exceeded terminates loop cleanly with trace summary
- [ ] Circle detection triggers after 3 identical sequences, aborts after 5
- [ ] Context assembly respects scope exclusions and token budget partitions
- [ ] Integration test: mock LLM replaying real model output recordings produces expected tool call sequence
- [ ] Trace file written and parseable for every completed run

### Architecture Decisions

| ID | Decision |
|----|----------|
| AD-004 | Synchronous execution per job — one goroutine per agent run, no intra-job concurrency |
| AD-005 | Sequential tool execution — tools run one at a time within a turn (no parallel tool calls in v1) |
| AD-006 | Additive context — context grows monotonically within a run; compaction summarizes but never drops |

---

## Milestone 2: Local Inference (3–4 days)

### Tasks

- **2.1** Hardware detection
  - **2.1.1** Detect available RAM, CPU cores, and GPU (Metal on macOS, CUDA on Linux)
  - **2.1.2** Recommend quantization level based on available VRAM/RAM (Q4_K_M for 16GB, Q5_K_M for 32GB+)
  - **2.1.3** Output hardware summary to slog and config
- **2.2** Model registry
  - **2.2.1** Registry data structure: model name, parameter count, quantization, required RAM, context length, download URL, SHA256
  - **2.2.2** Default model set: coding tier (Qwen 2.5 Coder 32B Q4), reasoning tier (DeepSeek R1 14B Q5), fast tier (Qwen 2.5 7B Q4)
  - **2.2.3** Custom model support — user-provided GGUF path or HuggingFace repo
- **2.3** Model download
  - **2.3.1** HTTP download with resume support (Range headers)
  - **2.3.2** Progress reporting (bytes downloaded, speed, ETA) via callback interface
  - **2.3.3** SHA256 checksum verification after download
  - **2.3.4** Store weights in `~/.mars-harness/models/`
  - **2.3.5** Skip download if file exists and checksum matches
- **2.4** llama.cpp server management
  - **2.4.1** Download llama-server binary (platform-specific: darwin-arm64, linux-x86_64) with checksum
  - **2.4.2** Start llama-server subprocess with model path, context length, GPU layers, port
  - **2.4.3** Health check loop — poll `/health` endpoint until ready, timeout after 60s
  - **2.4.4** Graceful restart on model switch — drain, stop, start with new model
  - **2.4.5** Graceful shutdown on process exit (SIGTERM → wait 10s → SIGKILL)
  - **2.4.6** Multi-model support — multiple llama-server instances on different ports, keyed by model name
- **2.5** LLM router
  - **2.5.1** Route by role → model tier mapping (e.g., Reviewer → reasoning, Engineer → coding, Pipeline Fixer → fast)
  - **2.5.2** Ensure target model's server is running before dispatching request
  - **2.5.3** Fallback: if local model unavailable, route to configured remote API (OpenAI, Anthropic)

### Quality Gate

- [ ] Hardware detection reports correct RAM and GPU on macOS (Metal) and Linux (CUDA)
- [ ] Model download completes end-to-end: fetch, resume after interrupt, verify checksum
- [ ] llama-server starts, passes health check, serves completions on expected port
- [ ] Agent runtime (M1) works end-to-end with a real local model (small quantization for CI)
- [ ] Router maps roles to correct model tiers and starts servers on demand
- [ ] Fallback to remote API works when local model is unavailable

### Architecture Decisions

| ID | Decision |
|----|----------|
| AD-007 | llama.cpp as subprocess (not CGO binding) — isolates crashes, simplifies builds |
| AD-008 | Model weights stored in `~/.mars-harness/models/`, binaries in `~/.mars-harness/bin/` |

---

## Milestone 3: `mars run` — FIRST DEMO (2–3 days)

### Tasks

- **3.1** CLI framework
  - **3.1.1** Adopt `cobra` for command structure, `pflag` for flags
  - **3.1.2** Root command with `--config`, `--verbose`, `--trace-dir` global flags
  - **3.1.3** `run` subcommand: `mars-harness run <role> --repo <path> [--model <name>] [--budget <tokens>] [--dry-run]`
  - **3.1.4** Version command sourced from build-time ldflags
- **3.2** Bundle reader
  - **3.2.1** Bundle manifest format (`bundle.yaml`): role name, model tier, system prompt path, guardrails refs, knowledge routes, tool permissions
  - **3.2.2** Load and validate manifest — required fields, file references resolve, schema version check
  - **3.2.3** Parse guardrails references (inline + file refs)
  - **3.2.4** Parse knowledge routes (glob patterns → file lists, priority weights)
  - **3.2.5** Bundle integrity hash — SHA256 of all bundle files for reproducibility
- **3.3** Terminal trace output
  - **3.3.1** Colour-coded real-time output: role name (cyan), tool calls (yellow), tool results (dim), errors (red), thinking (grey)
  - **3.3.2** Streaming display — tokens appear as they arrive from LLM
  - **3.3.3** Summary footer: total tokens, tool calls, duration, cost estimate
  - **3.3.4** `--quiet` mode: suppress trace, show only final output
  - **3.3.5** Non-TTY detection — disable colours and streaming for piped output
- **3.4** Sample bundle: Pipeline Fixer demo
  - **3.4.1** Create `bundles/pipeline-fixer/` with system prompt, guardrails, knowledge routes
  - **3.4.2** Create test repo fixture (`testdata/repos/broken-ts-project/`) — a small TypeScript project with a deliberate type error in `src/index.ts`
  - **3.4.3** Pipeline Fixer prompt instructs agent to: read CI logs, locate the failing file, diagnose the type error, apply a fix, verify the fix compiles
  - **3.4.4** End-to-end walkthrough documented in `docs/demos/pipeline-fixer.md`

### Quality Gate

- [ ] `mars-harness run pipeline-fixer --repo testdata/repos/broken-ts-project/` executes with visible colour-coded output
- [ ] Trace streams in real-time (tokens visible as they arrive)
- [ ] Missing bundle gives actionable error: "bundle 'foo' not found; available bundles: pipeline-fixer. See docs/bundles/ for how to create one."
- [ ] `--dry-run` shows what would execute without calling LLM
- [ ] Sample Pipeline Fixer bundle diagnoses and fixes the type error in the test repo
- [ ] Demo recorded (terminal recording or annotated transcript in `docs/demos/`)

### Architecture Decisions

None new — builds on AD-004, AD-005, AD-006 from M1.

---

## Milestone 4: GitHub Integration (4–5 days)

### Tasks

- **4.1** GitHub API client
  - **4.1.1** GitHub App JWT authentication (private key → JWT → installation token)
  - **4.1.2** PAT fallback authentication
  - **4.1.3** PR operations: create, update body/title, add comment, request review, merge
  - **4.1.4** Check run operations: create, update (in_progress, completed), set output summary
  - **4.1.5** Repository operations: get file contents, list directory, get commit, compare commits
  - **4.1.6** Rate limiting: respect `X-RateLimit-Remaining`, sleep on 403 rate limit, secondary rate limit handling
  - **4.1.7** Pagination: auto-paginate all list endpoints
- **4.2** GitHub App manifest flow
  - **4.2.1** `mars-harness github setup` command — opens browser to GitHub App creation URL with pre-filled manifest
  - **4.2.2** Local HTTP callback server to receive App credentials after creation
  - **4.2.3** Store App ID, private key, webhook secret in `~/.mars-harness/github/` (file permissions 0600)
  - **4.2.4** Installation ID discovery — list installations, prompt user to select org/repo
- **4.3** Webhook receiver
  - **4.3.1** HTTP server on configurable port (default `:9091`)
  - **4.3.2** `X-Hub-Signature-256` validation — reject unsigned or invalid payloads
  - **4.3.3** Event normalization — parse `pull_request`, `check_suite`, `issue_comment`, `push` events into internal types
  - **4.3.4** Event deduplication — track delivery IDs, reject replays within 1-hour window
  - **4.3.5** Event routing — map events to appropriate roles (PR opened → QA, check_suite failed → Pipeline Fixer)
- **4.4** PAT fallback mode
  - **4.4.1** Detect when no GitHub App configured, use `GITHUB_TOKEN` env var
  - **4.4.2** Disable webhook receiver in PAT mode (polling-only)
  - **4.4.3** Polling mode: check for new PRs/failures on configurable interval
- **4.5** Wire GitHub tools into agent
  - **4.5.1** `github_create_pr`, `github_comment`, `github_create_check_run`, `github_get_file`, `github_push_branch`
  - **4.5.2** Tools use authenticated client from M4.1
  - **4.5.3** Integration tests against real test repo (`mars-harness-test`)

### Quality Gate

- [ ] GitHub App created via manifest flow, credentials stored securely
- [ ] Webhook signature validation rejects tampered payloads
- [ ] Events normalized correctly for all supported types (table-driven tests with real GitHub webhook payloads)
- [ ] Deduplication rejects replayed delivery IDs
- [ ] Agent creates a real PR on `mars-harness-test` repo via GitHub tools
- [ ] PAT fallback works when no App is configured
- [ ] Rate limiting sleeps appropriately on 403 (integration test with rate limit simulation)

### Architecture Decisions

None new — GitHub client design follows AD-004 (sync per-job) and AD-014 (domain-grouped in `internal/github/`).

---

## Milestone 5a: Pipeline Mechanics (3–4 days)

### Tasks

- **5a.1** SQLite job queue
  - **5a.1.1** Schema: `jobs` table with `id`, `repo_id`, `role`, `status` (queued/running/completed/failed), `trigger_event`, `created_at`, `started_at`, `completed_at`, `trace_path`, `result_json`
  - **5a.1.2** Job lifecycle: queued → running → completed/failed, with atomic status transitions
  - **5a.1.3** Per-repo serialization — only one job runs per `repo_id` at a time; others queue in FIFO order
  - **5a.1.4** Idempotency — duplicate trigger events (same repo + role + event hash) within 5-minute window are deduplicated
  - **5a.1.5** Job TTL — failed/completed jobs retained for 30 days, then pruned
- **5a.2** Worker dispatcher
  - **5a.2.1** Configurable concurrency limit (default: number of available model slots)
  - **5a.2.2** Worker pool — goroutines pull from queue, respecting per-repo serialization
  - **5a.2.3** Per-job timeout (configurable, default 30 minutes) — kill agent loop and record failure
  - **5a.2.4** Graceful shutdown — on SIGTERM/SIGINT, stop accepting new jobs, wait for running jobs to complete (with 60s hard deadline)
- **5a.3** Cron scheduler
  - **5a.3.1** Cron expression parsing (standard 5-field + `@daily`/`@hourly` shortcuts)
  - **5a.3.2** Timezone-aware scheduling (default UTC, configurable per schedule)
  - **5a.3.3** Missed schedule handling — if harness was offline, run at most one catch-up execution per schedule
  - **5a.3.4** Schedule sources: config file entries and per-repo `.harness/schedules.yaml`

### Quality Gate

- [ ] Jobs queued via API and dispatched to workers in FIFO order
- [ ] Per-repo serialization enforced — concurrent jobs for same repo queue, different repos run in parallel
- [ ] Duplicate trigger events within 5-minute window are deduplicated
- [ ] Scheduler emits jobs on cron tick (test with 1-second cron for speed)
- [ ] Graceful shutdown completes running jobs and persists queue state
- [ ] Job TTL pruning removes old records

### Architecture Decisions

| ID | Decision |
|----|----------|
| AD-009 | SQLite for job queue and state — no external database dependency, WAL mode for concurrent reads |
| AD-010 | `repo_id` as first-class dimension from day one — all queries, serialization, and scoring are repo-scoped |

---

## Milestone 5b: Safety Layer (2–3 days)

### Tasks

- **5b.1** Process sandbox
  - **5b.1.1** Linux: `clone()` with `CLONE_NEWNS | CLONE_NEWPID | CLONE_NEWNET` — mount namespace (read-only bind mounts except workdir), PID namespace, network namespace (no outbound except GitHub API)
  - **5b.1.2** macOS fallback: `sandbox-exec` profile restricting file writes to workdir, network to GitHub API hosts only
  - **5b.1.3** Sandbox disabled gracefully if unsupported (log warning, continue with filesystem restrictions only)
- **5b.2** Blast radius controls
  - **5b.2.1** Max files changed per job (default: 20)
  - **5b.2.2** Max lines changed per file (default: 500)
  - **5b.2.3** Max total lines changed per job (default: 2000)
  - **5b.2.4** Rate limit: max PRs per repo per hour (default: 3)
  - **5b.2.5** No-delete policy: agent cannot delete files unless explicitly allowed in role config
  - **5b.2.6** Secret scanner: regex patterns for AWS keys, GitHub tokens, private keys, database URLs — block commits containing matches
- **5b.3** Emergency stop
  - **5b.3.1** `mars-harness stop` command — immediately halts all running jobs
  - **5b.3.2** GitHub state cleanup: convert open PRs to draft, delete branches created by halted jobs, cancel in-progress check runs
  - **5b.3.3** Record stop event in job history with reason
  - **5b.3.4** Dashboard emergency stop button (wired in M9)
- **5b.4** `mars-harness serve` command
  - **5b.4.1** Wire together: webhook receiver (M4.3) + job queue (M5a.1) + worker dispatcher (M5a.2) + scheduler (M5a.3) + safety layer (M5b)
  - **5b.4.2** Health endpoint at `/healthz` returning JSON status
  - **5b.4.3** Structured startup log showing all active components and their configuration

### Quality Gate

- [ ] Sandbox isolates file writes to workdir (attempt to write outside fails)
- [ ] Blast radius blocks a diff exceeding max files/lines limits
- [ ] Secret scanner catches AWS key pattern, GitHub token pattern, and PEM private key
- [ ] Emergency stop halts running jobs and cleans up GitHub state (drafts PRs, deletes branches, cancels check runs)
- [ ] `mars-harness serve` starts, accepts webhooks, dispatches jobs through the full pipeline
- [ ] `/healthz` returns accurate component status

### Architecture Decisions

None new — safety is an enforcement layer atop existing AD-004 and AD-010 decisions.

---

## Milestone 6: Accuracy and Autonomy (3–4 days)

**Pre-requisite:** M4 webhook receiver must be stable — outcome tracking relies on receiving GitHub events.

### Tasks

- **6.1** Outcome tracking
  - **6.1.1** Monitor GitHub events post-job: PR merged (positive), PR closed without merge (negative), check run passed (positive), check run failed after fix (negative), comment requesting changes (negative)
  - **6.1.2** Match events to originating jobs via branch name / check run external ID
  - **6.1.3** Store outcomes in SQLite: `outcomes` table with `job_id`, `repo_id`, `role`, `outcome_type`, `timestamp`, `details_json`
  - **6.1.4** Timeout: if no outcome event within 48 hours, record as `unknown`
- **6.2** Scoring engine
  - **6.2.1** Score formula: `(merged + passed) / (merged + passed + closed + failed + noop)` — noop detection (job ran but produced no output or empty diff) counts as negative
  - **6.2.2** Rolling window: 30-day window, recalculated on each new outcome
  - **6.2.3** Per-role, per-repo scores stored in `scores` table
  - **6.2.4** Minimum sample size: require 5 outcomes before score is considered valid
- **6.3** Progressive autonomy
  - **6.3.1** Trust levels: `observer` (read-only, no PRs), `trial` (create draft PRs, require human approval), `autonomous` (create PRs, auto-merge if checks pass)
  - **6.3.2** Promotion rules: observer → trial after 3 successful read-only runs; trial → autonomous after score ≥ 0.8 over 10+ outcomes
  - **6.3.3** Demotion rules: autonomous → trial if score drops below 0.6; trial → observer if score drops below 0.4
  - **6.3.4** Enforcement: trust level checked before GitHub write operations; observer cannot call `github_create_pr`
  - **6.3.5** Override: `--trust-level` flag on `mars-harness run` for manual override (logged)
- **6.4** CLI commands
  - **6.4.1** `mars-harness scores` — table of per-role, per-repo scores with sample size and trend arrow
  - **6.4.2** `mars-harness trust` — current trust level per role per repo, promotion/demotion history
  - **6.4.3** `mars-harness trust set <role> <repo> <level>` — manual override with recorded reason

### Quality Gate

- [ ] Each outcome type (merged, closed, passed, failed, noop) scored correctly with unit tests
- [ ] Rolling 30-day score computed accurately across window boundaries
- [ ] Role promotes from observer → trial → autonomous as scores improve (integration test with synthetic outcomes)
- [ ] Role demotes when score drops below threshold
- [ ] Observer trust level cannot create PRs (enforcement test)
- [ ] Trial mode creates draft PRs only
- [ ] `mars-harness scores` displays correct data formatted as table
- [ ] `mars-harness trust` shows current levels and history

### Architecture Decisions

None new — scoring uses AD-009 (SQLite) and AD-010 (repo_id scoping).

---

## Milestone 7: Self-Improvement and Guardrails (5–6 days)

### Tasks

- **7.1** Intervention detector
  - **7.1.1** Detect human interventions on agent-created PRs: manual commits pushed to agent branch, PR description edited, review comments with code suggestions applied, PR closed and reopened with changes
  - **7.1.2** Classify interventions: `clear` (exact diff between agent version and merged version), `ambiguous` (structural changes that may or may not be corrections), `non-intervention` (merge commits, CI-only changes)
  - **7.1.3** Store interventions in SQLite: `interventions` table with `job_id`, `repo_id`, `role`, `classification`, `diff`, `timestamp`
- **7.2** Reviewer meta-role
  - **7.2.1** Dedicated agent role that reads execution traces and intervention diffs
  - **7.2.2** Uses coding-tier model (strongest available) with higher context budget (2x standard) for analysis quality
  - **7.2.3** Classifies root cause: prompt gap, missing context, tool limitation, model capability, correct-as-is
  - **7.2.4** Proposes prompt evolution: specific additions/modifications to the role's system prompt or guardrails
  - **7.2.5** Rate limit: max 1 evolution proposal per role per day
  - **7.2.6** Auto-disable: if last 3 evolutions for a role worsened its score, suspend Reviewer proposals for that role until manual review
- **7.3** Evolution PR creation
  - **7.3.1** Create branch `harness/evolve/{role}/{date}` with proposed prompt changes
  - **7.3.2** PR body includes: intervention diff, root cause analysis, proposed change, expected impact
  - **7.3.3** PR requires human approval (never auto-merged)
  - **7.3.4** Track evolution PR outcomes (merged/closed) for Reviewer accuracy scoring
- **7.4** Guardrails engine
  - **7.4.1** Guardrail definition format (YAML): `type` (advisory | hard), `name`, `description`, `check` (for hard: function reference; for advisory: prompt text)
  - **7.4.2** Advisory guardrails: injected into system prompt as constraints (e.g., "never modify package.json without running install")
  - **7.4.3** Hard guardrails: mechanical validation run after each tool call or before PR creation (e.g., max diff size, no secret patterns, required file patterns)
  - **7.4.4** Guardrail override: `--override-guardrail <name>` flag with mandatory reason logged
  - **7.4.5** Staleness detection: flag guardrails not triggered in 90 days for review
  - **7.4.6** Guardrail inheritance: global guardrails + per-role guardrails + per-repo guardrails (most specific wins on conflict)
- **7.5** CLI commands
  - **7.5.1** `mars-harness interventions` — list recent interventions with classification and status
  - **7.5.2** `mars-harness interventions show <id>` — detailed view with diff and Reviewer analysis

### Quality Gate

- [ ] Clear interventions classified correctly (test: agent PR vs merged version with known edits)
- [ ] Ambiguous interventions classified and flagged for review
- [ ] Reviewer produces plausible root cause analysis for a known prompt gap scenario
- [ ] Evolution PR created with correct branch naming, body content, and diff
- [ ] Reviewer cannot modify its own prompt (self-modification blocked)
- [ ] Max 1 evolution per role per day enforced
- [ ] Auto-disable triggers after 3 consecutive score-worsening evolutions
- [ ] Advisory guardrails appear in system prompt
- [ ] Hard guardrails block violations (test: diff exceeding max size blocked before PR creation)
- [ ] Guardrail override logged with reason
- [ ] Stale guardrails (not triggered in 90 days) flagged in `mars-harness doctor`

### Architecture Decisions

| ID | Decision |
|----|----------|
| AD-012 | Syntactic-only evolution in v1 — Reviewer proposes text changes to prompts, no structural changes to tool definitions or guardrail logic |

---

## Milestone 8: Setup and Init (3–4 days)

### Tasks

- **8.1** `mars-harness setup`
  - **8.1.1** Orchestrated setup flow: hardware detect → display recommendation → model download (with user confirmation) → GitHub App creation (or PAT entry) → verify connectivity → start serve
  - **8.1.2** Idempotent: re-running skips completed steps (check for existing models, GitHub credentials, etc.)
  - **8.1.3** `--test-mode` flag: skip model download (use mock), skip GitHub App (use PAT), reduce timeouts
  - **8.1.4** Progress display: step N/M with status indicators
- **8.2** `mars-harness init`
  - **8.2.1** Scaffold `.harness/` directory in target repo
  - **8.2.2** Generate `roles.yaml` from detected repo characteristics (language, framework, CI system)
  - **8.2.3** Generate starter tickets from repo scanner output (M8.3)
  - **8.2.4** Generate starter exec plans
  - **8.2.5** Generate `AGENTS.md` stub tailored to detected project structure
  - **8.2.6** `.harness/schedules.yaml` with sensible defaults (nightly QA, on-push Pipeline Fixer)
- **8.3** Repo scanner
  - **8.3.1** Detect: language, framework, package manager, CI system, test framework
  - **8.3.2** Find: missing tests (files in `src/` without corresponding test file), TODO/FIXME comments, type safety gaps (`any` in TypeScript), dead code indicators
  - **8.3.3** Smart skipping: ignore `node_modules/`, `vendor/`, generated files, lock files
  - **8.3.4** Output: structured JSON for programmatic use, markdown summary for human reading
  - **8.3.5** Ticket generation: convert findings into `.harness/tickets/` markdown files with priority, category, and suggested approach
- **8.4** `mars-harness doctor`
  - **8.4.1** System health: Go version, disk space, available RAM, GPU detection
  - **8.4.2** Model health: models present and valid (checksum), llama-server binary present
  - **8.4.3** GitHub health: App credentials valid, webhook endpoint reachable, API rate limit status
  - **8.4.4** Pipeline health: SQLite database accessible, queue depth, stuck jobs
  - **8.4.5** Accuracy summary: per-role scores, recent interventions, stale guardrails
  - **8.4.6** Output: coloured pass/warn/fail indicators, actionable remediation for each failure

### Quality Gate

- [ ] `mars-harness setup` completes end-to-end in test mode (mock model, PAT auth)
- [ ] Re-running setup skips already-completed steps
- [ ] `mars-harness init` scaffolds `.harness/` with correct structure for a detected Node.js project
- [ ] Repo scanner produces plausible findings for a known test repo (finds missing tests, TODOs)
- [ ] Scanner skips `node_modules/` and generated files
- [ ] `mars-harness doctor` reports accurate health across all subsystems
- [ ] Doctor provides actionable fix instructions for each failure mode

### Architecture Decisions

None new — setup orchestrates components from prior milestones.

---

## Milestone 9: Dashboard (5–6 days)

### Tasks

- **9.1** Dashboard infrastructure
  - **9.1.1** HTTP server on configurable port (default `:9090`), integrated into `mars-harness serve`
  - **9.1.2** Go `html/template` for server-side rendering
  - **9.1.3** htmx for dynamic updates (partial page swaps, polling, form submission)
  - **9.1.4** Chart.js for time-series and bar charts
  - **9.1.5** Server-Sent Events (SSE) endpoint for real-time updates
  - **9.1.6** All assets (CSS, JS, images) embedded via `embed.FS` — zero external CDN dependencies
  - **9.1.7** Responsive layout with sidebar navigation
- **9.2** Page 1: Pipeline Flow
  - **9.2.1** Simple layered layout (not full Sugiyama) — roles as nodes, trigger→role→outcome as edges
  - **9.2.2** Server-side SVG generation with stable node positions
  - **9.2.3** Live state: node colour reflects current status (idle/grey, running/blue, success/green, failed/red)
  - **9.2.4** Click node → navigate to role detail page
  - **9.2.5** SSE updates for real-time node state changes
- **9.3** Page 2: Role Health
  - **9.3.1** Per-role cards: current score, trust level, outcome breakdown (pie chart)
  - **9.3.2** Accuracy trend: 30-day rolling score as line chart (Chart.js)
  - **9.3.3** Context usage: average tokens used vs budget per role (bar chart)
  - **9.3.4** Recent outcomes list with links to traces
- **9.4** Page 3: Throughput
  - **9.4.1** Job statistics: jobs/day, average duration, queue depth over time
  - **9.4.2** Inference metrics: tokens/second, model utilization, context window usage
  - **9.4.3** GitHub API metrics: requests/hour, rate limit headroom, cache hit rate
  - **9.4.4** Resource usage: RAM, CPU, GPU utilization (sampled every 30s)
- **9.5** Page 4: Debug
  - **9.5.1** Job timeline: Gantt-style view of recent jobs with duration bars
  - **9.5.2** Trace viewer: full conversation display with collapsible tool calls/results
  - **9.5.3** Live trace: SSE-powered real-time trace view for running jobs
  - **9.5.4** Webhook log: recent received webhooks with payload preview
  - **9.5.5** Error log: aggregated errors with frequency and last occurrence
- **9.6** Page 5: Evolution History
  - **9.6.1** Timeline of Reviewer evolution proposals
  - **9.6.2** Each entry: role, date, root cause summary, proposed change diff, PR link, outcome (merged/closed/pending)
  - **9.6.3** Score impact graph: role score before and after each merged evolution
- **9.7** Emergency stop button
  - **9.7.1** Prominent red button in navigation header
  - **9.7.2** Confirmation dialog before executing
  - **9.7.3** Triggers `mars-harness stop` logic (M5b.3), shows cleanup progress

### Quality Gate

- [ ] Dashboard loads self-contained at `:9090` with no external network requests
- [ ] Pipeline graph renders with correct node layout and edges
- [ ] Nodes update live via SSE when job status changes
- [ ] Role health page shows real data from scoring engine
- [ ] Trace viewer displays full conversation with expandable tool calls
- [ ] Live trace streams tokens in real-time for a running job
- [ ] Emergency stop button halts jobs and shows cleanup progress
- [ ] All pages render correctly on 1280px+ viewport (responsive down to 1024px)

### Architecture Decisions

| ID | Decision |
|----|----------|
| AD-011 | htmx + Chart.js embedded (via `embed.FS`) — no external CDN, no SPA framework, no Node.js build step |

---

## Milestone 10: Mars Migration and Distribution (6–7 days)

### Tasks

- **10.1** Port Mars prompts — batch 1 (priority)
  - **10.1.1** Engineer — adapt for harness tool names, local model context limits, structured output format
  - **10.1.2** Pipeline Fixer — adapt CI log reading, fix application, verification loop
  - **10.1.3** QA — adapt test generation, coverage analysis, fixture creation
- **10.2** Port Mars prompts — batch 2
  - **10.2.1** CEO Vision
  - **10.2.2** CTO Harness
  - **10.2.3** COO Tickets
  - **10.2.4** Reviewer (already partially built in M7)
  - **10.2.5** Code Reviewer
- **10.3** Port Mars prompts — batch 3
  - **10.3.1** Docs Writer
  - **10.3.2** Release Manager
  - **10.3.3** Security Auditor
- **10.4** Validate each ported role
  - **10.4.1** Run each role against the Mars monorepo with a controlled test scenario
  - **10.4.2** Compare output quality to equivalent Cursor automation output
  - **10.4.3** Document quality delta for each role: equivalent, minor regression, significant regression
  - **10.4.4** Iterate on prompts for roles with significant regression (budget: 0.5 day per role)
- **10.5** Dogfood
  - **10.5.1** Run Pipeline Fixer on the mars-harness repo itself (introduce deliberate CI failure, verify fix)
  - **10.5.2** Run QA on the mars-harness repo (verify it generates meaningful test suggestions)
  - **10.5.3** Run Code Reviewer on a mars-harness PR (verify review quality)
  - **10.5.4** Document dogfood results and any prompt adjustments made
- **10.6** Distribution
  - **10.6.1** Cross-compile: `darwin/arm64`, `darwin/amd64`, `linux/amd64`, `linux/arm64`
  - **10.6.2** `curl` installer script: detect OS/arch, download binary, verify checksum, place in `$PATH`
  - **10.6.3** Homebrew formula: `mars-harness` tap with bottle support
  - **10.6.4** GitHub Releases: automated release workflow (tag → build → upload binaries → generate release notes)
  - **10.6.5** Binary size target: < 30MB (strip debug, compress if needed)
- **10.7** Documentation
  - **10.7.1** Quick start guide: install → setup → first run (under 10 minutes)
  - **10.7.2** Tenets reference: expanded explanation of each tenet with examples
  - **10.7.3** Bundle reference: format specification, examples, best practices
  - **10.7.4** Guardrails guide: how to write advisory and hard guardrails, testing guardrails
  - **10.7.5** Model guide: recommended models by use case, VRAM requirements, performance benchmarks
  - **10.7.6** Architecture overview: component diagram, data flow, extension points

### Quality Gate

- [ ] All 11 roles ported with working bundles
- [ ] At least 7 roles produce output equivalent to Cursor automation (documented comparison)
- [ ] Remaining roles have documented quality delta with specific improvement plan
- [ ] Mars-harness runs Pipeline Fixer on its own repo and fixes a deliberate CI failure
- [ ] Mars-harness runs QA on its own repo and produces meaningful test suggestions
- [ ] `curl -fsSL https://get.mars-harness.dev | sh` works on clean Linux (Ubuntu 22.04) and macOS (14+)
- [ ] Homebrew `brew install mars-stack/tap/mars-harness` installs and runs
- [ ] Quick start guide enables a new user to go from zero to first successful run

### Architecture Decisions

None new — distribution is a packaging concern, not an architectural one.

---

## Test Coverage Requirements

| Metric | Target | Enforcement |
|--------|--------|-------------|
| Unit test coverage | ≥ 70% line coverage per package | CI fails below threshold (`go test -coverprofile`) |
| Integration tests | ≥ 1 per milestone | Tracked in milestone quality gates |
| Error paths | Every exported function's error return tested | Enforced by review, `errcheck` linter |
| CI green | All tests pass on every PR | Branch protection rule |
| Lint zero | `golangci-lint run` reports zero issues | CI fails on any lint finding |
| Race detector | All tests run with `-race` | CI flag: `go test -race ./...` |
| Table-driven | Preferred style for ≥ 3 cases | Convention in `AGENTS.md`, enforced by review |

---

## Architecture Decision Log

| ID | Decision | Milestone | Rationale |
|--------|----------|-----------|-----------|
| AD-001 | Pure Go, `CGO_ENABLED=0`, single static binary | M0 | Zero runtime dependencies, trivial cross-compilation, single-file distribution |
| AD-002 | Apache 2.0 license | M0 | Permissive, compatible with all downstream use, patent grant |
| AD-003 | Mars conventions (exec plans, design docs, AGENTS.md, cursor rules) | M0 | Proven documentation and agent workflow patterns from Mars monorepo |
| AD-004 | Synchronous execution per job | M1 | Simplifies reasoning about agent state; concurrency at the job level, not within jobs |
| AD-005 | Sequential tool execution within a turn | M1 | Avoids race conditions in file operations; parallel tool calls deferred to v2 |
| AD-006 | Additive context (never drop, only compact) | M1 | Prevents silent loss of earlier reasoning; compaction preserves intent via summary |
| AD-007 | llama.cpp as subprocess (not CGO) | M2 | Crash isolation, independent version management, preserves `CGO_ENABLED=0` |
| AD-008 | Weights in `~/.mars-harness/models/`, binaries in `~/.mars-harness/bin/` | M2 | User-local storage, no root required, easy cleanup, respects XDG conventions |
| AD-009 | SQLite for job queue and persistent state | M5a | No external database dependency, WAL mode for concurrent reads, embedded in binary |
| AD-010 | `repo_id` as first-class dimension from day one | M5a | All queries, serialization, scoring, and trust are repo-scoped by design |
| AD-011 | htmx + Chart.js embedded via `embed.FS` | M9 | Zero external CDN, no SPA framework, no Node.js build step, single binary serves dashboard |
| AD-012 | Syntactic-only evolution in v1 | M7 | Reviewer proposes prompt text changes only; structural tool/guardrail changes require human design |
| AD-013 | `log/slog` stdlib for structured logging | M0 | Zero dependencies, structured JSON in production, human-readable in development |
| AD-014 | Domain-grouped packages under `internal/` | M0 | Clear ownership boundaries, prevents import cycles, explicit public surface via `pkg/` |
| AD-015 | YAML config + environment variable overrides | M0 | Human-readable config files, twelve-factor env var overrides, no config server dependency |

---

## Risk Register

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| **Agent runtime reliability** — LLM produces unusable output, infinite loops, or hallucinated tool calls despite safeguards | High | High | Circle detection (M1.3.4), robust tool call parser with real model recordings (M1.3.6), budget enforcement (M1.3.3), progressive autonomy gating (M6.3), intervention detection (M7.1) |
| **Local model quality** — open-weight models underperform vs cloud APIs for complex coding tasks | Medium | High | LLM router with remote API fallback (M2.5), model registry allows swapping tiers (M2.2), dogfood validation against Cursor output (M10.4), prompt iteration budget in M10 |
| **Dashboard complexity** — scope creep on 5 dashboard pages delays other milestones | Medium | Medium | Simple layered layout not Sugiyama (M9.2), htmx not SPA (AD-011), defer non-essential pages if behind schedule, dashboard is last feature milestone before migration |
| **GitHub App permissions across orgs** — org admins may block App installation or restrict permissions | Medium | Medium | PAT fallback mode (M4.4) works without any App, polling mode as alternative to webhooks, clear documentation of required permissions and why |
| **Single-developer bus factor** — all knowledge and context in one person | High | High | Mars conventions (AD-003): AGENTS.md, design docs, exec plans, execution traces, conversation-as-system-record pattern; harness dogfoods its own documentation discipline |
| **llama.cpp compatibility** — binary updates may break server management assumptions | Low | Medium | Pin llama.cpp release version with checksum (M2.4.1), health check loop (M2.4.3) detects failures, fallback to remote API (M2.5.3) |
| **SQLite concurrency under load** — WAL mode has limits with many concurrent writers | Low | Medium | Per-repo serialization (AD-010) limits write contention, single writer goroutine pattern, benchmark at 50 concurrent repos before shipping |
| **Secret scanner false positives** — regex patterns may block legitimate code patterns | Medium | Low | Configurable allowlist per repo, guardrail override with logging (M7.4.4), start conservative and tune based on intervention data |
