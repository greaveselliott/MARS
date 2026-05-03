# Changelog

Patch notes are generated with `mars-harness release notes` from semantic commits on `main`.

## [0.24.7] - 2026-05-03
<!-- mars-harness-release: version=0.24.7 commit=31f16cb2cfc7 -->

### Why This Release Matters
It improves reliability through work to migrate legacy job columns before indexes in queue.

### Fixes
- **queue:** Migrate legacy job columns before indexes (31f16cb)

## [0.24.6] - 2026-05-03
<!-- mars-harness-release: version=0.24.6 commit=c63ef60fc301 -->

### Why This Release Matters
It improves reliability through work to make evidence stores actionable in cli.

### Fixes
- **cli:** Make evidence stores actionable (c63ef60)

## [0.24.5] - 2026-05-03
<!-- mars-harness-release: version=0.24.5 commit=d3add7e85a82 -->

### Why This Release Matters
It improves reliability through work to check doctor test file write in ci.

### Fixes
- **ci:** Check doctor test file write (d3add7e)

## [0.24.4] - 2026-05-03
<!-- mars-harness-release: version=0.24.4 commit=eda0526868fd -->

### Why This Release Matters
It improves reliability through work to check serve test file setup in ci.

### Fixes
- **ci:** Check serve test file setup (eda0526)

## [0.24.3] - 2026-05-03
<!-- mars-harness-release: version=0.24.3 commit=59f889ea7ed0 -->

### Why This Release Matters
It improves reliability through work to clear static lint findings in ci.

### Fixes
- **ci:** Clear static lint findings (59f889e)

## [0.24.2] - 2026-05-03
<!-- mars-harness-release: version=0.24.2 commit=571bf7138d6c -->

### Why This Release Matters
It improves reliability through work to clear remaining lint findings in ci.

### Fixes
- **ci:** Clear remaining lint findings (571bf71)

## [0.24.1] - 2026-05-03
<!-- mars-harness-release: version=0.24.1 commit=5a1472e12000 -->

### Why This Release Matters
It improves reliability through work to satisfy lint checks in ci.

### Fixes
- **ci:** Satisfy lint checks (5a1472e)

## [0.24.0] - 2026-05-03
<!-- mars-harness-release: version=0.24.0 commit=8d27b4ed3583 -->

### Why This Release Matters
This release matters because it gives operators new capability through work to add dispatch organization layer in orchestration.

### Features
- **orchestration:** Add dispatch organization layer (8d27b4e)

## [0.23.0] - 2026-05-03
<!-- mars-harness-release: version=0.23.0 commit=deccb88b12bb -->

### Why This Release Matters
This release matters because it gives operators new capability through work to add native orchestrator survey loop (MH-047) in serve.

### Features
- **serve:** Add native orchestrator survey loop (MH-047) (deccb88)

### Delivery Evidence
- Enabler work: MH-047: Add native Orchestrator survey loop

## [0.22.1] - 2026-05-03
<!-- mars-harness-release: version=0.22.1 commit=ef3c15d1c115 -->

### Why This Release Matters
It improves reliability through work to fall back when linux namespaces are unavailable in sandbox.

### Fixes
- **sandbox:** Fall back when linux namespaces are unavailable (ef3c15d)

## [0.22.0] - 2026-05-03
<!-- mars-harness-release: version=0.22.0 commit=44f2c8464e91 -->

### Why This Release Matters
This release matters because it gives operators new capability through work to harden recovery evidence and tool surface in quality.
It makes the harness easier to understand and operate through work to add OpenHarness comparator in references.

### Features
- **quality:** Harden recovery evidence and tool surface (44f2c84)

### Documentation
- **references:** Add OpenHarness comparator (82efaa6)

## [0.21.0] - 2026-05-03
<!-- mars-harness-release: version=0.21.0 commit=cb32661e015e -->

### Features
- **tickets:** Enforce in-progress drain states (MH-046) (cb32661)

### Delivery Evidence
- Enabler work: MH-046: Enforce in-progress ticket drain

## [0.20.0] - 2026-05-03
<!-- mars-harness-release: version=0.20.0 commit=5546e12b1874 -->

### Features
- **serve:** Ingest intervention debt signals (MH-045) (5546e12)

### Delivery Evidence
- Enabler work: MH-045: Complete intervention-debt signal ingestion

## [0.19.1] - 2026-05-03
<!-- mars-harness-release: version=0.19.1 commit=9b7e4bb50117 -->

### Documentation
- **operating-model:** Add conversation system record guidance (MH-044) (9b7e4bb)

### Delivery Evidence
- Enabler work: MH-044: Add conversation system record guidance

## [0.19.0] - 2026-05-03
<!-- mars-harness-release: version=0.19.0 commit=6a2d36a9be79 -->

### Features
- **role-registry:** Add checked role inventory (MH-043) (6a2d36a)

### Delivery Evidence
- Enabler work: MH-043: Add checked role registry

## [0.18.0] - 2026-05-03
<!-- mars-harness-release: version=0.18.0 commit=d5436fdefd23 -->

### Features
- **role-model:** Add canonical harness operating domains (MH-042) (d5436fd)

### Delivery Evidence
- Enabler work: MH-042: Create canonical harness operating model

## [0.17.0] - 2026-05-03
<!-- mars-harness-release: version=0.17.0 commit=ed664ab2a36e -->

### Why This Release Matters
This release made tool creation safer by adding a guard that pushes repeated, risky, or validation-heavy work into governed first-class tools instead of fragile chat memory.

### Features
- **tools:** Add tool creation guard (ed664ab)

## [0.16.0] - 2026-05-03
<!-- mars-harness-release: version=0.16.0 commit=5f9870bd9b08 -->

### Why This Release Matters
This release made model choices more evidence-driven by adding a benchmark-backed workflow for evaluating and promoting local inference providers.

### Features
- **models:** Add benchmark-backed provider workflow (MH-030) (5f9870b)

### Delivery Evidence
- Enabler work: MH-030: Benchmark-backed model refresh and promotion

## [0.15.2] - 2026-05-03
<!-- mars-harness-release: version=0.15.2 commit=027449036856 -->

### Why This Release Matters
This release tightened the operating discipline around tools so new automation capabilities are created through documented, reviewable paths.

### Documentation
- **tools:** Require governed tool creation (0274490)

## [0.15.1] - 2026-05-03
<!-- mars-harness-release: version=0.15.1 commit=358216584c40 -->

### Why This Release Matters
This release made done criteria clearer by expanding the BDD feature contract catalog that anchors delivery claims to observable scenarios.

### Documentation
- **features:** Expand BDD contract catalog (3582165)

## [0.15.0] - 2026-05-03
<!-- mars-harness-release: version=0.15.0 commit=b9b84535812f -->

### Why This Release Matters
This release turned recurring operational rituals into formal workflow tools, making important release and quality steps easier for agents to repeat correctly.

### Features
- **tools:** Formalize repeated workflow tools (b9b8453)

## [0.14.6] - 2026-05-03
<!-- mars-harness-release: version=0.14.6 commit=3ca1a420b043 -->

### Why This Release Matters
This release clarified the release asset contract so installer-facing releases have an explicit definition of what must be published and verified.

### Documentation
- **tickets:** Complete release asset contract (MH-031) (3ca1a42)

### Delivery Evidence
- Enabler work: MH-031: Publish release binary assets for installer

## [0.14.5] - 2026-05-03
<!-- mars-harness-release: version=0.14.5 commit=227b6f718abf -->

### Why This Release Matters
This release improved Linux terminal support so interactive UI behavior can work across more operator environments.

### Fixes
- **ui:** Support linux terminal ioctl constants (MH-031) (227b6f7)

## [0.14.4] - 2026-05-03
<!-- mars-harness-release: version=0.14.4 commit=be63396bb21a -->

### Why This Release Matters
This release improved release reliability by adding a path to backfill binary assets when a GitHub Release had notes but no downloadable artifacts.

### Fixes
- **release:** Backfill notes-only release assets (MH-031) (be63396)

## [0.14.3] - 2026-05-03
<!-- mars-harness-release: version=0.14.3 commit=ed9853b52bd3 -->

### Why This Release Matters
This release refreshed the architecture map so future agents and maintainers can understand the current system without reconstructing it from code.

### Documentation
- **architecture:** Update current system map (ed9853b)

## [0.14.2] - 2026-05-03
<!-- mars-harness-release: version=0.14.2 commit=9fe9b5857df7 -->

### Why This Release Matters
This release protected the operating model from piecemeal drift by requiring workflow changes to fit the surrounding delivery loop.

### Documentation
- **operating-model:** Require symbiotic workflow changes (9fe9b58)

## [0.14.1] - 2026-05-03
<!-- mars-harness-release: version=0.14.1 commit=195c73e1183c -->

### Why This Release Matters
This release made the shared tool vocabulary explicit, helping source and target harnesses describe the same capabilities consistently.

### Documentation
- **tools:** Add mirrored tools glossary (195c73e)

## [0.14.0] - 2026-05-03
<!-- mars-harness-release: version=0.14.0 commit=422adac6f6ad -->

### Why This Release Matters
This release gave target harnesses a mirrored Mars Harness CLI tool, making core harness operations available through the same universal tool surface.

### Features
- **tools:** Add mirrored mars harness cli tool (422adac)

## [0.13.1] - 2026-05-03
<!-- mars-harness-release: version=0.13.1 commit=cd0ffd67d96a -->

### Why This Release Matters
This release recorded the release asset blocker plainly, keeping unfinished publication work visible instead of burying it in chat or CI logs.

### Documentation
- **tickets:** Record release asset blocker (MH-031) (cd0ffd6)

## [0.13.0] - 2026-05-03
<!-- mars-harness-release: version=0.13.0 commit=ccd36bc3bf3d -->

### Why This Release Matters
This release made self-update releases more trustworthy by verifying GitHub assets and avoiding stale changelog markers when finding new release work.

### Features
- **release:** Verify release assets for self-update (MH-031) (9e027a2)

### Fixes
- **release:** Ignore stale changelog markers (MH-031) (ccd36bc)

## [0.12.1] - 2026-05-03
<!-- mars-harness-release: version=0.12.1 commit=d8e8c6fcc990 -->

### Why This Release Matters
This release clarified the difference between foundation and deployed operating models, reducing ambiguity when rules are mirrored into target projects.

### Documentation
- **glossary:** Define operating model distinctions (d8e8c6f)

## [0.12.0] - 2026-05-03
<!-- mars-harness-release: version=0.12.0 commit=416a91bd5fa1 -->

### Why This Release Matters
This release made repository health measurable by exporting a quality score that agents and operators can use to track delivery confidence.

### Features
- **scoring:** Export repo quality score (MH-037) (416a91b)

### Delivery Evidence
- Enabler work: MH-037: Automate quality score export

## [0.11.1] - 2026-05-03
<!-- mars-harness-release: version=0.11.1 commit=450d1bbbbbd9 -->

### Why This Release Matters
This release fixed target harness parity by ensuring deployed projects receive the same governed `tool_create` capability as the foundation harness.

### Fixes
- **tools:** Mirror tool_create in target harness (450d1bb)

## [0.11.0] - 2026-05-03
<!-- mars-harness-release: version=0.11.0 commit=a00bb9e11730 -->

### Why This Release Matters
This release added the scaffold for creating new tools, turning repeatable operator workflows into durable harness capabilities.

### Features
- **tools:** Add tool creation scaffold (a00bb9e)

## [0.10.4] - 2026-05-03
<!-- mars-harness-release: version=0.10.4 commit=f133e393f97a -->

### Why This Release Matters
This release mirrored harness terminology so initialized target projects inherit the same language and operating context as the source harness.

### Documentation
- **glossary:** Mirror harness terminology (f133e39)

## [0.10.3] - 2026-05-03
<!-- mars-harness-release: version=0.10.3 commit=9e444541196c -->

### Why This Release Matters
This release converted Mars parity workstreams into concrete backlog tickets, making the remaining migration work visible and schedulable.

### Documentation
- **planning:** Materialize mars parity backlog tickets (MH-035) (9e44454)

### Delivery Evidence
- Enabler work: MH-035: Materialize Mars parity workstreams as tickets

## [0.10.2] - 2026-05-03
<!-- mars-harness-release: version=0.10.2 commit=e2bcf2f7a080 -->

### Why This Release Matters
This release improved diagnosis by classifying ticket gate failures in telemetry, making blocked delivery easier to understand and repair.

### Fixes
- **telemetry:** Classify ticket gate failures (e2bcf2f)

## [0.10.1] - 2026-05-03
<!-- mars-harness-release: version=0.10.1 commit=bb885cd5ad7e -->

### Why This Release Matters
This release made installed model state clearer by surfacing available model variants during inference checks.

### Fixes
- **inference:** Surface installed model variants (bb885cd)

## [0.10.0] - 2026-05-03
<!-- mars-harness-release: version=0.10.0 commit=0f4d9ec86ceb -->

### Why This Release Matters
This release added an active plan hygiene checker so the harness can detect stale or inconsistent execution planning before it misleads agents.

### Features
- **planhygiene:** Add active plan hygiene checker (MH-034) (0f4d9ec)

### Delivery Evidence
- Enabler work: MH-034: Implement active-plan hygiene checker

## [0.9.0] - 2026-05-02
<!-- mars-harness-release: version=0.9.0 commit=c3a87e2179e3 -->

### Why This Release Matters
This release made installation more plug-and-play by configuring shell paths automatically, reducing the chance that a working install is hidden from new terminals.

### Features
- **setup:** Configure shell path automatically (MH-041) (c3a87e2)

### Delivery Evidence
- Shipped feature scenarios: MH-041: F-002-S001, F-002-S002, F-002-S003, F-002-S004, F-002-S005

## [0.8.0] - 2026-05-02
<!-- mars-harness-release: version=0.8.0 commit=cd7514dfdce5 -->

### Why This Release Matters
This release made delivery more evidence-led by wiring the operating model around BDD scenarios, goals, and quality gates instead of vague task completion.

### Features
- **operating-model:** Implement BDD-led delivery loop (MH-040) (cd7514d)

### Delivery Evidence
- Shipped feature scenarios: MH-040: F-001-S001, F-001-S002, F-001-S003, F-001-S004, F-001-S005, F-001-S006

## [0.7.5] - 2026-05-02
<!-- mars-harness-release: version=0.7.5 commit=e39e335c8fc2 -->

### Why This Release Matters
This release made execution planning clearer by recording dependency metadata, helping agents understand what must happen before a plan can advance.

### Documentation
- **plans:** Add exec plan dependency metadata (e39e335)

## [0.7.4] - 2026-05-02
<!-- mars-harness-release: version=0.7.4 commit=c7dbdf3dbb6e -->

### Why This Release Matters
This release reduced planning ambiguity by enforcing a single active execution plan as the current source of truth.

### Documentation
- **plans:** Enforce single active exec plan (c7dbdf3)

## [0.7.3] - 2026-05-02
<!-- mars-harness-release: version=0.7.3 commit=9a4ced42bdad -->

### Why This Release Matters
This release seeded the quality score artifact so repository health could be tracked as a first-class output, not inferred from scattered notes.

### Documentation
- **scoring:** Seed quality score artifact (9a4ced4)

## [0.7.2] - 2026-05-02
<!-- mars-harness-release: version=0.7.2 commit=dac23b716ce3 -->

### Why This Release Matters
This release reconciled the current execution state, giving future agents a more accurate starting point for planning and delivery.

### Documentation
- **plans:** Reconcile current execution state (dac23b7)

## [0.7.1] - 2026-05-02
<!-- mars-harness-release: version=0.7.1 commit=21d617f832ef -->

### Why This Release Matters
This release made version drift tests less brittle by keeping fixtures independent from the current release number.

### Tests
- **update:** Keep version drift fixtures release-agnostic (21d617f)

## [0.7.0] - 2026-05-02
<!-- mars-harness-release: version=0.7.0 commit=ce831c5cd4de -->

### Why This Release Matters
This release helped operators see when either the installed tool or a target harness has drifted from the expected version.

### Features
- **update:** Check tool and harness version drift (ce831c5)

## [0.6.0] - 2026-05-02
<!-- mars-harness-release: version=0.6.0 commit=2187d5a379c3 -->

### Why This Release Matters
This release unified update workflows so operators have one coherent path for keeping both the CLI tool and harness files current.

### Features
- **update:** Unify tool and harness updates (2187d5a)

## [0.5.3] - 2026-05-02
<!-- mars-harness-release: version=0.5.3 commit=781c1e5051dd -->

### Why This Release Matters
This release made source installs less confusing by clarifying the supported workflow for running Mars Harness from a checkout.

### Fixes
- **setup:** Clarify source install workflow (781c1e5)

## [0.5.2] - 2026-05-02
<!-- mars-harness-release: version=0.5.2 commit=4a599310de29 -->

### Why This Release Matters
This release documented the policy for swapping Ollama-backed models, preserving the rationale behind provider choices.

### Documentation
- **models:** Define ollama swap policy (4a59931)

## [0.5.1] - 2026-05-02
<!-- mars-harness-release: version=0.5.1 commit=8f0a44f12017 -->

### Why This Release Matters
This release kept intervention-debt tickets independent so self-improvement work can be tracked without contaminating unrelated telemetry.

### Fixes
- **telemetry:** Keep intervention tickets independent (8f0a44f)

## [0.5.0] - 2026-05-02
<!-- mars-harness-release: version=0.5.0 commit=0ca0257223cd -->

### Why This Release Matters
This release gave the harness a durable feedback loop by creating intervention-debt tickets from human corrections and operational failures.

### Features
- **telemetry:** Create intervention-debt tickets (0ca0257)

## [0.4.1] - 2026-05-02
<!-- mars-harness-release: version=0.4.1 commit=548fb73403a1 -->

### Why This Release Matters
This release improved local inference routing by respecting manifest tiers when choosing which model should handle each role.

### Fixes
- **inference:** Route roles by manifest tier (548fb73)

## [0.4.0] - 2026-05-02
<!-- mars-harness-release: version=0.4.0 commit=72032c5985e4 -->

### Why This Release Matters
This release added a benchmark evaluation path so model changes can be judged against evidence instead of preference or guesswork.

### Features
- **models:** Add benchmark evaluation path (72032c5)

## [0.3.6] - 2026-05-02
<!-- mars-harness-release: version=0.3.6 commit=ecf0f5596249 -->

### Why This Release Matters
This release improved queue stability by preventing self-healing logic from creating recovery storms.

### Fixes
- **queue:** Self-heal recovery storms (ecf0f55)

## [0.3.5] - 2026-05-02
<!-- mars-harness-release: version=0.3.5 commit=4769fb4172da -->

### Why This Release Matters
This release contained recursive recovery jobs, reducing the risk that a repair attempt creates more orchestration work than it solves.

### Fixes
- **serve:** Contain recursive recovery jobs (4769fb4)

## [0.3.4] - 2026-05-02
<!-- mars-harness-release: version=0.3.4 commit=5fef93f4bc04 -->

### Why This Release Matters
This release made GitHub Release publication an explicit part of the release contract, so versioned notes and assets are not treated as optional afterthoughts.

### Documentation
- **release:** Require github release publication (5fef93f)

## [0.3.3] - 2026-05-02
<!-- mars-harness-release: version=0.3.3 commit=3232920f527f -->

### Why This Release Matters
This release ensured initialized target projects inherit the same operating rules as the foundation harness unless deliberately overridden.

### Documentation
- **harness:** Mirror operating rules into targets (3232920)

## [0.3.2] - 2026-05-02
<!-- mars-harness-release: version=0.3.2 commit=5c5bc2d6761b -->

### Why This Release Matters
This release mirrored versioning rules into target projects so source and deployed harnesses follow the same release discipline.

### Documentation
- **release:** Mirror versioning rule into targets (5c5bc2d)

## [0.3.1] - 2026-05-02
<!-- mars-harness-release: version=0.3.1 commit=466bc65ad438 -->

### Why This Release Matters
This release required semantic versioning after source commits, making release state explicit and recoverable from the repo.

### Documentation
- **release:** Require versioning after source commits (466bc65)

## [0.3.0] - 2026-05-02
<!-- mars-harness-release: version=0.3.0 commit=b2cd7df5f2e5 -->

### Why This Release Matters
This release documented how agent skills should evolve, supporting the self-improving system tenet without losing safety or reviewability.

### Features
- **skills:** Guide self-improving skill evolution (b2cd7df)

## [0.2.0] - 2026-05-02
<!-- mars-harness-release: version=0.2.0 commit=15f4b154182d -->

### Why This Release Matters
This release introduced automated semantic patch notes, giving Mars Harness a repeatable way to turn commits into versioned release history.

### Features
- **release:** Automate semantic patch notes (15f4b15)

## [0.1.0] - 2026-05-02
<!-- mars-harness-release: version=0.1.0 commit=edaafeacae3a -->

### Why This Release Matters
This foundation release turned Mars Harness from a concept into a working self-hosted AI delivery system. It established the CLI, local model runtime, agent loop, context assembly, queue, scheduler, safety controls, scanner, dashboard, tool registry, release workflow, and the initial operating doctrine needed for autonomous delivery on target repositories.

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
