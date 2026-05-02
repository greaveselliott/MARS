# Changelog

Patch notes are generated with `mars-harness release notes` from semantic commits on `main`.

## [0.5.3] - 2026-05-02
<!-- mars-harness-release: version=0.5.3 commit=781c1e5051dd -->

### Fixes
- **setup:** Clarify source install workflow (781c1e5)

## [0.5.2] - 2026-05-02
<!-- mars-harness-release: version=0.5.2 commit=4a599310de29 -->

### Documentation
- **models:** Define ollama swap policy (4a59931)

## [0.5.1] - 2026-05-02
<!-- mars-harness-release: version=0.5.1 commit=8f0a44f12017 -->

### Fixes
- **telemetry:** Keep intervention tickets independent (8f0a44f)

## [0.5.0] - 2026-05-02
<!-- mars-harness-release: version=0.5.0 commit=0ca0257223cd -->

### Features
- **telemetry:** Create intervention-debt tickets (0ca0257)

## [0.4.1] - 2026-05-02
<!-- mars-harness-release: version=0.4.1 commit=548fb73403a1 -->

### Fixes
- **inference:** Route roles by manifest tier (548fb73)

## [0.4.0] - 2026-05-02
<!-- mars-harness-release: version=0.4.0 commit=72032c5985e4 -->

### Features
- **models:** Add benchmark evaluation path (72032c5)

## [0.3.6] - 2026-05-02
<!-- mars-harness-release: version=0.3.6 commit=ecf0f5596249 -->

### Fixes
- **queue:** Self-heal recovery storms (ecf0f55)

## [0.3.5] - 2026-05-02
<!-- mars-harness-release: version=0.3.5 commit=4769fb4172da -->

### Fixes
- **serve:** Contain recursive recovery jobs (4769fb4)

## [0.3.4] - 2026-05-02
<!-- mars-harness-release: version=0.3.4 commit=5fef93f4bc04 -->

### Documentation
- **release:** Require github release publication (5fef93f)

## [0.3.3] - 2026-05-02
<!-- mars-harness-release: version=0.3.3 commit=3232920f527f -->

### Documentation
- **harness:** Mirror operating rules into targets (3232920)

## [0.3.2] - 2026-05-02
<!-- mars-harness-release: version=0.3.2 commit=5c5bc2d6761b -->

### Documentation
- **release:** Mirror versioning rule into targets (5c5bc2d)

## [0.3.1] - 2026-05-02
<!-- mars-harness-release: version=0.3.1 commit=466bc65ad438 -->

### Documentation
- **release:** Require versioning after source commits (466bc65)

## [0.3.0] - 2026-05-02
<!-- mars-harness-release: version=0.3.0 commit=b2cd7df5f2e5 -->

### Features
- **skills:** Guide self-improving skill evolution (b2cd7df)

## [0.2.0] - 2026-05-02
<!-- mars-harness-release: version=0.2.0 commit=15f4b154182d -->

### Features
- **release:** Automate semantic patch notes (15f4b15)

## [0.1.0] - 2026-05-02
<!-- mars-harness-release: version=0.1.0 commit=edaafeacae3a -->

### Features
- **tools:** Mechanical ticket deduplication with ticket_create tool (AD-030) (0322c0c)
- **tools,scanner:** Wire git tools into manifests and add commit gates (AD-028) (1091028)
- **pipeline:** Chain dogfood tester after engineer completes a feature (10c845d)
- Dogfood E2E validation with Podman + decision recording system (147952e)
- **dashboard,scanner:** Dynamic pipeline chain + tsconfig path alias check (168a354)
- **m6,m7:** Scoring engine, progressive autonomy, evolution system, guardrails engine (MH-017 through MH-021) (1b9890e)
- **telemetry:** Triage self-improvement signals (28082ae)
- **tools:** Add registry, executor, and core tools (M1.2 / MH-002) (3083f9e)
- **scanner:** Add bootability checks for framework-specific validation (AD-026) (38c491f)
- **m5:** Job queue, scheduler, sandbox, safety, serve command (MH-013, MH-014, MH-015, MH-016) (45365ca)
- **serve:** Wire autonomous orchestrator with repo registry, trigger router, and executor (498d349)
- **agent:** Add conversation loop, parser, and tests (M1.3 / MH-003) (5394612)
- **scanner:** Add mars-harness upgrade command to sync target project manifests and prompts (5b53d26)
- **setup:** Plug-and-play model download from HuggingFace (5c91811)
- **skills:** Add agent skills across Cursor, AGENTS.md, and .harness/skills/ (73a0a82)
- **ui:** Cursor-quality agent output with role banner, tool trace, and handoff (7bf1294)
- **setup:** Auto-install llama-server and wire run command (7ddea6e)
- **llm:** Add OpenAI-compatible chat client and testutil helpers (7e50d1d)
- **init:** Provision full 11-role Mars pipeline by default (Tenet 1) (88de7a0)
- **serve:** Parallel pipeline tracks with sleep resilience (89b7895)
- **dashboard:** Wire dashboard into orchestrator with SSE events (8dce248)
- Implement two-level self-learning system with janitor agent (8f86add)
- **init:** Auto-run git init when .git is missing (8fc2260)
- **cli:** Auto-init .harness when manifest is missing (907d23a)
- **m9:** Embedded dashboard with htmx, Chart.js, SSE, emergency stop (MH-024) (9f7c083)
- **context:** Add context assembler with token budget (M1.4 / MH-004) (a07bfc9)
- **m8:** Setup wizard, init + scanner, doctor command (MH-022, MH-023, MH-028) (a26dccc)
- **init:** Scaffold full docs/ structure with Mars-quality role prompts (a6f7b26)
- **telemetry:** Add telemetry collector with error classification and auto-fix (ab150c5)
- **core:** Enforce strict trunk execution safety (b11daca)
- **tools:** Add background mode to shell_exec for long-running processes (b898608)
- **m3,m4:** CLI framework, bundle reader, terminal UI, GitHub integration (MH-009, MH-010, MH-011, MH-012) (bb95350)
- **inference:** Add llama-server subprocess manager and role router (ce12a9d)
- **context:** File filter, budget partitioning, .harnessignore support (M1.4.2, M1.4.4, MH-004) (d264010)
- **m10:** Role prompts, dogfood workflow, cross-compile release, user docs (MH-025, MH-026, MH-027) (dcac2af)
- **start:** Add `mars-harness start` for full e2e pipeline orchestration (deb1cd3)
- **dashboard:** Implement throughput page with chart, stats, and job table (df3b70d)
- **m2:** Hardware detection, model download, llama-server management, LLM router (MH-006, MH-007, MH-008) (e3436b9)
- **orchestrator:** Configurable agent triggers with chaining and custom cron (ec4db54)
- **agent:** M1 closeout — 3+ tool-call test, trace error propagation, realistic 5-turn test (MH-003, MH-005) (ece2768)
- **inference:** Expose local performance tuning (ed24f83)
- **prompts:** Add git push after every commit in all role prompts (efce0b9)
- **serve:** Per-repo database isolation to prevent cross-project contamination (AD-029) (f098c2e)
- **init:** Mirror harness context glossary (f23663d)
- **trace:** Add JSONL recorder and SQLite store (M1.5 / MH-005) (f862f46)

### Fixes
- **tools:** Kill entire process group on shell_exec timeout (2ecc1ad)
- **inference:** Mitigate LLM timeout, context overflow, and connection refused failures (64ca767)
- **prompts:** Cold-start CEO/COO prompts for new projects (6eff4e9)
- **setup:** Pin local runtime artifacts (7e1a85e)
- **serve:** Block engineer handoff with active tickets (7fd00a8)
- **agent:** Always serialize message content field for llama.cpp compat (927cdb5)
- **agent:** Add context window guard to prevent token overflow (a045bd0)
- Resolve audit findings — broken links, naming inconsistencies, missing docs (M0) (cb17a6e)
- **init:** Preserve existing user content on --force re-init (ccacf88)
- **core:** Auto-tune inference and drain active tickets (e1fd6e0)
- **github:** Keep integrations trunk oriented (e45a90e)
- Correct module path, add gitkeeps, fix placeholders (M0) (ea287fc)
- **pipeline:** Restore COO → Engineer handoff for delivery kickoff (ebfaa56)
- **upgrade:** Preserve user configured agents (edaafea)
- **serve:** Auto-cleanup stale processes and corrupt DB on start/serve (f3af248)
- **dashboard:** Constrain chart canvas height to 280px (f9620aa)

### Documentation
- **references:** Carry mars agent-first references (009358b)
- **generated:** Define generated docs contract (1c0f043)
- **tickets:** Populate full backlog MH-001 through MH-028 (M0) (2c508cf)
- Add terminology definitions and dual-repo commit discipline (3759dc9)
- **product:** Refresh living product specs (4806d8f)
- Switch to trunk-based development, drop branch/PR requirement (584e4d7)
- **workflow:** Align generated bundles with strict trunk (69b608b)
- **design:** Record AD-031 inference resilience decisions (838e29c)
- **references:** Audit mars relevance for harness parity (92d7b8b)
- **exec-plans:** Add master execution plan with M0–M10 + MH-001–MH-028 coverage (9d13b8e)
- **plans:** Add mars supersession parity plan (b8f2a35)
- Add AD-021 through AD-025 for dogfood, decisions, and lean pipeline (bd6293b)
- **tickets:** Reconcile 19 ticket-vs-schedule contradictions (C1–C19) (d315dfc)

### Maintenance
- **tickets:** Move MH-001 through MH-005 to done/ (M1 closeout) (00bbe6f)
- **m0:** Audit and fix M0 quality gate gaps (3419f69)
- **tickets:** Move MH-006 through MH-008 to done/ (M2 closeout) (431fb77)
- Initialize mars-harness repo (M0) (451a632)
- **tickets:** Move MH-009 through MH-012 to done/ (M3+M4 closeout) (88c182e)
- **tickets:** Move MH-017 through MH-021 to done/ (M6+M7 closeout) (8d48467)
- **tickets:** Move MH-013 through MH-016 to done/ (M5a+M5b closeout) (95dbee2)

### Tests
- **serve:** Fix serve tests for new Config requirements, add skills loader tests (56e169a)
