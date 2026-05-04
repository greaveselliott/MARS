# Changelog

Patch notes are generated with `mars-harness release notes` from semantic commits on `main`.

## [0.33.3] - 2026-05-04
<!-- mars-harness-release: version=0.33.3 commit=914cf98f13ee -->

### Impact
- **release:** Operators and future agents get release notes that explain structural delivery shifts instead of repeating thin commit subjects.

### Why
- **release:** Operating-model, orchestration, persona, documentation-sync, and CLI/tool-sync releases need durable context because the changelog becomes the upgrade narrative.

### What Changed
- **release:** Added topic-aware fallback profiles, covered structured dispatch regression, documented the rule, and backfilled 0.33.1 (914cf98).

### Fixes
- **release:** Enrich structural release-note fallbacks (914cf98)

## [0.33.2] - 2026-05-04
<!-- mars-harness-release: version=0.33.2 commit=67aa3f109904 -->

### Impact
- **cli:** Operators see improved reliability because add root version shortcuts.

### Why
- **cli:** This matters because add root version shortcuts closes a failure mode or degraded path.

### What Changed
- **cli:** Changed add root version shortcuts (67aa3f1).

### Fixes
- **cli:** Add root version shortcuts (67aa3f1)

## [0.33.1] - 2026-05-04
<!-- mars-harness-release: version=0.33.1 commit=c436460e5357 -->

### Impact
- **orchestration:** Operators and agents get a more reliable delivery loop because handoff and feedback now travel as first-class runtime data through Orchestrator dispatch.

### Why
- **orchestration:** This matters because operating-model shifts lose value when the next owner, expected correction, or supporting evidence only exists in free-form transcript text.

### What Changed
- **orchestration:** Dispatch triggers now carry the source disposition, including status, next need, ticket ID, reason, evidence links, trace ID, handoff, and feedback, so Orchestrator can validate one target owner before enqueueing follow-up work (c436460).

### Fixes
- **orchestration:** Carry structured handoff through dispatch (c436460)

## [0.33.0] - 2026-05-04
<!-- mars-harness-release: version=0.33.0 commit=6fbc203cea94 -->

### Impact
- **personas:** Operators gain new capability: add canonical foundation agent manuals.

### Why
- **personas:** This matters because add canonical foundation agent manuals was missing from the shipped capability set.

### What Changed
- **personas:** Changed add canonical foundation agent manuals (6fbc203).

### Features
- **personas:** Add canonical foundation agent manuals (6fbc203)

## [0.32.0] - 2026-05-04
<!-- mars-harness-release: version=0.32.0 commit=bcf54eda8895 -->

### Impact
- **roles:** Operators gain new capability: add optional head of strategy agent.

### Why
- **roles:** This matters because add optional head of strategy agent was missing from the shipped capability set.

### What Changed
- **roles:** Changed add optional head of strategy agent (bcf54ed).

### Features
- **roles:** Add optional head of strategy agent (bcf54ed)

## [0.31.1] - 2026-05-04
<!-- mars-harness-release: version=0.31.1 commit=5c2af02c10ea -->

### Impact
- **telemetry:** Operators see improved reliability because satisfy collector rollback lint.

### Why
- **telemetry:** This matters because satisfy collector rollback lint closes a failure mode or degraded path.

### What Changed
- **telemetry:** Changed satisfy collector rollback lint (5c2af02).

### Fixes
- **telemetry:** Satisfy collector rollback lint (5c2af02)

## [0.31.0] - 2026-05-04
<!-- mars-harness-release: version=0.31.0 commit=3953db758420 -->

### Impact
- **telemetry:** Operators gain new capability: add anonymous foundation telemetry collector.

### Why
- **telemetry:** This matters because add anonymous foundation telemetry collector was missing from the shipped capability set.

### What Changed
- **telemetry:** Changed add anonymous foundation telemetry collector (3953db7).

### Features
- **telemetry:** Add anonymous foundation telemetry collector (3953db7)

## [0.30.1] - 2026-05-04
<!-- mars-harness-release: version=0.30.1 commit=074a9e5391af -->

### Impact
- **orchestration:** Freshly initialized target repos no longer get pulled into CEO/Orchestrator loops when the local model emits function-tag tool calls or when a slugged feature contract already exists. Bootstrap can keep moving from plan to feature contract to ticket shaping instead of manufacturing intervention-debt churn.

### Why
- **orchestration:** A simple README-only project exposed three connected reliability gaps: function-tag tool calls were treated as final prose, Orchestrator looked for exact `docs/features/F-001.md` paths even though generated contracts are slugged, and repeated dispatch decisions could enqueue Orchestrator again instead of stopping. The combined effect created noisy telemetry and intervention-debt tickets before product delivery had even started.

### What Changed
- **orchestration:** The agent parser now normalizes `<function=name>` and `<parameter=arg>` blocks into normal tool calls, dispatch loop guards stop repeated Orchestrator-originated routes without ticket-state changes, and generated CEO/COO/Orchestrator guidance resolves BDD features through `docs/features/F-NNN*.md` slug matches (074a9e5).

### Fixes
- **orchestration:** Stop bootstrap dispatch loops (074a9e5)

## [0.30.0] - 2026-05-04
<!-- mars-harness-release: version=0.30.0 commit=ae9ac01bc65e -->

### Impact
- **cli:** Operators and agents now have a release-blocking operating model for keeping the `mars-harness` CLI, the mirrored `mars_harness_cli` tool, generated target guidance, and CLI-related skills synchronized. A command can no longer quietly land while agents keep reading stale tool reference text or stale workflow skills.
- **cli:** The `mars_harness_cli` repo shortcut now recognizes the repo-aware command paths that had drifted from the CLI surface, including `release backfill-notes`, `docsync audit`, `models evaluate`, `models override`, `scores`, and `trust`.

### Why
- **cli:** The CLI is the foundation control plane, but agents usually discover it through mirrored tools and generated skills. Without an explicit sync model and tests, every CLI change created a chance that target agents would keep invoking old commands, miss new repo flags, or choose generic shell execution over the intended tool path.
- **cli:** The previous documentation-sync model made code-to-doc ownership explicit, but it did not specifically cover the CLI-to-tool-and-skill mirrors that agents depend on when operating a deployed harness.

### What Changed
- **cli:** Added AD-103 in `docs/design-docs/cli-tool-skill-sync.md`, documenting the architecture, universal operating model, required mirrors, evidence commands, invariants, and failure mitigations for CLI tool/skill synchronization (ae9ac01).
- **cli:** Added command-tree tests that compare the live Cobra command graph with the `mars_harness_cli` reference and repo shortcut map, then exported the reference/shortcut helpers so the CLI package can enforce that mirror without stringly guessing (ae9ac01).
- **cli:** Mirrored the model into generated target harnesses with a new `cli-tool-sync` skill, generated AD-103 docs, knowledge routes, AGENTS guidance, F-001 scenario coverage, scanner assertions, and doctrine-sync checks (ae9ac01).

### Features
- **cli:** Enforce tool skill sync (ae9ac01)

## [0.29.1] - 2026-05-04
<!-- mars-harness-release: version=0.29.1 commit=8a718debfcbc -->

### Impact
- **docsync:** Operators and future agents now have a first-class Documentation Sync architecture and universal operating model, so "no stale documentation" is no longer just a rule spread across guidance. Every code change has an explicit path from changed files to associated docs, BDD contracts, audit evidence, and release notes.
- **docsync:** Initialized target harnesses inherit the same documentation-sync doctrine, feature scenario, knowledge route, and generated design doc, which keeps source and deployed operating models aligned as projects are scaffolded.

### Why
- **docsync:** The previous source-to-doc map made ownership auditable, but it did not yet explain the architecture, role responsibilities, maintenance workflows, or failure modes deeply enough for agents to apply the process consistently without chat context.
- **docsync:** The stale-doc problem spans foundation and deployed harnesses. Without a universal operating model mirrored into generated targets, new projects could receive the metadata gate but miss the decision-making workflow that tells agents when to update BDD, design docs, product specs, tool docs, role docs, or release notes.

### What Changed
- **docsync:** Added AD-102 in `docs/design-docs/documentation-sync-architecture.md`, documenting the six-layer architecture, the seven-step universal operating model, role responsibilities, maintenance workflows, invariants, mitigations, observability, and acceptance criteria for documentation sync (8a718de).
- **docsync:** Linked the new architecture from `AGENTS.md`, the design-doc index, the delivery operating model, the code-documentation map, the tools glossary, and the BDD feature catalog so agents can discover it from normal context assembly paths (8a718de).
- **docsync:** Mirrored AD-102 into generated target harness defaults, updated target knowledge routes and F-001 with a universal documentation-sync scenario, and expanded scanner/docs-consistency/formal workflow tests so the architecture remains release-blocking (8a718de).

### Documentation
- **docsync:** Document universal operating model (8a718de)

## [0.29.0] - 2026-05-04
<!-- mars-harness-release: version=0.29.0 commit=cb59e75cacf2 -->

### Impact
- **docsync:** Operators and agents now have a repo-wide source-to-documentation map. Every audited source file carries `MarsDocSync` metadata that points to the feature contracts, design docs, product specs, role docs, or release docs that must be checked when that file changes.
- **docsync:** The new `mars-harness docsync audit --repo .` command and mirrored `docsync_audit` tool turn no-stale-documentation from guidance into a repeatable quality gate.

### Why
- **docsync:** Documentation drift was still possible because a reviewer had to infer which docs belonged to a changed file. The new map makes that relationship explicit and lets automation fail when source files lack metadata, reference missing docs, or drift from the canonical package map.
- **docsync:** Generated target harnesses need the same universal operating model, so the scaffolded guidance, role allowlists, tools glossary, role registry, and F-001 feature contract now teach agents to run docsync before claiming code and docs are in sync.

### What Changed
- **docsync:** Added the `internal/docsync` package, `mars-harness docsync audit`, and the mirrored `docsync_audit` workflow tool, with tests for metadata parsing, missing-doc failures, CLI behavior, and docs-consistency enforcement (cb59e75).
- **docsync:** Added `docs/design-docs/code-documentation-map.md` as AD-101, extended F-001 with the source-wide docsync audit scenario, and updated no-stale-documentation doctrine to require a structured `docs:` array (cb59e75).
- **docsync:** Seeded `MarsDocSync` metadata across all audited source roots and updated generated target defaults so implementation, review, release, dogfood, and maintenance roles can use `docsync_audit` before git handoff (cb59e75).

### Features
- **docsync:** Map source files to documentation (cb59e75)

## [0.28.3] - 2026-05-04
<!-- mars-harness-release: version=0.28.3 commit=970e1659b42e -->

### Impact
- **release:** Operators see improved reliability because remove unused commit group helper.

### Why
- **release:** This matters because remove unused commit group helper closes a failure mode or degraded path.

### What Changed
- **release:** Changed remove unused commit group helper (970e165).

### Fixes
- **release:** Remove unused commit group helper (970e165)

## [0.28.2] - 2026-05-04
<!-- mars-harness-release: version=0.28.2 commit=4d26d7c9bf43 -->

### Impact
- **update:** Operators see improved reliability because resolve tagged private release assets.

### Why
- **update:** This matters because resolve tagged private release assets closes a failure mode or degraded path.

### What Changed
- **update:** Changed resolve tagged private release assets (4d26d7c).

### Fixes
- **update:** Resolve tagged private release assets (4d26d7c)

## [0.28.1] - 2026-05-04
<!-- mars-harness-release: version=0.28.1 commit=830ce4782671 -->

### Impact
- **update:** Operators see improved reliability because authenticate private release asset downloads.

### Why
- **update:** This matters because authenticate private release asset downloads closes a failure mode or degraded path.

### What Changed
- **update:** Changed authenticate private release asset downloads (830ce47).

### Fixes
- **update:** Authenticate private release asset downloads (830ce47)

## [0.28.0] - 2026-05-04
<!-- mars-harness-release: version=0.28.0 commit=2c912e4e3425 -->

### Impact
- **release:** Operators can bring every historical changelog entry onto the current Impact, Why, and What Changed release-note standard with a reusable checked command.

### Why
- **release:** Historical release notes were still on mixed narrative formats, which made the changelog a stale and uneven source of product communication.

### What Changed
- **release:** Added release backfill-notes with dry-run, check, and version-range support; backfilled 0.1.0 through 0.26.2; documented the BDD feature audit; and added consistency tests for changelog narrative sections (2c912e4).

### Features
- **release:** Backfill historical release narratives (2c912e4)

## [0.27.0] - 2026-05-04
<!-- mars-harness-release: version=0.27.0 commit=7d285a8ca7ad -->

### Impact
- **release:** Operators and maintainers get release notes that explain the actual impact of a change before the commit buckets.

### Why
- **release:** The previous generated summary was too thin for humans to understand why a release mattered or what changed without rereading the commits.

### What Changed
- **release:** Added generated Impact, Why, and What Changed sections, documented the universal release-note rule, and mirrored the guidance into target defaults (7d285a8).

### Features
- **release:** Generate detailed release narratives (7d285a8)

## [0.26.2] - 2026-05-04
<!-- mars-harness-release: version=0.26.2 commit=1dcd96de5bf9 -->

### Impact
- **operating-model:** Operators and future agents get clearer guidance because add no stale documentation rule.

### Why
- **operating-model:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.

### What Changed
- **operating-model:** Changed add no stale documentation rule (1dcd96d).

### Documentation
- **operating-model:** Add no stale documentation rule (1dcd96d)

## [0.26.1] - 2026-05-04
<!-- mars-harness-release: version=0.26.1 commit=dad878a5206c -->

### Impact
- **operating-model:** Operators and future agents get clearer guidance because make business logic first-class BDD.

### Why
- **operating-model:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.

### What Changed
- **operating-model:** Changed make business logic first-class BDD (dad878a).

### Documentation
- **operating-model:** Make business logic first-class BDD (dad878a)

## [0.26.0] - 2026-05-04
<!-- mars-harness-release: version=0.26.0 commit=38c1627fb039 -->

### Impact
- **cli:** Operators gain new capability: add target harness eject kill switch.

### Why
- **cli:** This matters because add target harness eject kill switch was missing from the shipped capability set.

### What Changed
- **cli:** Changed add target harness eject kill switch (38c1627).

### Features
- **cli:** Add target harness eject kill switch (38c1627)

## [0.25.1] - 2026-05-04
<!-- mars-harness-release: version=0.25.1 commit=42393b327478 -->

### Impact
- **planning:** Operators see improved reliability because enforce bootstrap artifact order.

### Why
- **planning:** This matters because enforce bootstrap artifact order closes a failure mode or degraded path.

### What Changed
- **planning:** Changed enforce bootstrap artifact order (42393b3).

### Fixes
- **planning:** Enforce bootstrap artifact order (42393b3)

## [0.25.0] - 2026-05-04
<!-- mars-harness-release: version=0.25.0 commit=8622d4122e83 -->

### Impact
- **orchestration:** Operators gain new capability: return dispatch handoffs to orchestrator.

### Why
- **orchestration:** This matters because return dispatch handoffs to orchestrator was missing from the shipped capability set.

### What Changed
- **orchestration:** Changed return dispatch handoffs to orchestrator (8622d41).

### Features
- **orchestration:** Return dispatch handoffs to orchestrator (8622d41)

## [0.24.16] - 2026-05-04
<!-- mars-harness-release: version=0.24.16 commit=b78f9f4e4b50 -->

### Impact
- **tickets:** Operators see improved reliability because enforce canonical ticket creation.

### Why
- **tickets:** This matters because enforce canonical ticket creation closes a failure mode or degraded path.

### What Changed
- **tickets:** Changed enforce canonical ticket creation (b78f9f4).

### Fixes
- **tickets:** Enforce canonical ticket creation (b78f9f4)

## [0.24.15] - 2026-05-04
<!-- mars-harness-release: version=0.24.15 commit=eb7e43cbf37e -->

### Impact
- **safety:** Operators see improved reliability because disable default file-count blast radius cap.

### Why
- **safety:** This matters because disable default file-count blast radius cap closes a failure mode or degraded path.

### What Changed
- **safety:** Changed disable default file-count blast radius cap (eb7e43c).

### Fixes
- **safety:** Disable default file-count blast radius cap (eb7e43c)

## [0.24.14] - 2026-05-04
<!-- mars-harness-release: version=0.24.14 commit=1bf62cd85b50 -->

### Impact
- **harness:** Operators and future agents get clearer guidance because define mirrored skill glossary.

### Why
- **harness:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.

### What Changed
- **harness:** Changed define mirrored skill glossary (1bf62cd).

### Documentation
- **harness:** Define mirrored skill glossary (1bf62cd)

## [0.24.13] - 2026-05-04
<!-- mars-harness-release: version=0.24.13 commit=4a6370f43596 -->

### Impact
- **init:** Operators see improved reliability because baseline generated scaffold across entrypoints.

### Why
- **init:** This matters because baseline generated scaffold across entrypoints closes a failure mode or degraded path.

### What Changed
- **init:** Changed baseline generated scaffold across entrypoints (4a6370f).

### Fixes
- **init:** Baseline generated scaffold across entrypoints (4a6370f)

## [0.24.12] - 2026-05-03
<!-- mars-harness-release: version=0.24.12 commit=7a09c57bc3aa -->

### Impact
- **start:** Operators see improved reliability because commit generated harness baseline.

### Why
- **start:** This matters because commit generated harness baseline closes a failure mode or degraded path.

### What Changed
- **start:** Changed commit generated harness baseline (7a09c57).

### Fixes
- **start:** Commit generated harness baseline (7a09c57)

## [0.24.11] - 2026-05-03
<!-- mars-harness-release: version=0.24.11 commit=310a5b052c6a -->

### Impact
- **harness:** Operators see improved reliability because add foundation containment gate.

### Why
- **harness:** This matters because add foundation containment gate closes a failure mode or degraded path.

### What Changed
- **harness:** Changed add foundation containment gate (310a5b0).

### Fixes
- **harness:** Add foundation containment gate (310a5b0)

## [0.24.10] - 2026-05-03
<!-- mars-harness-release: version=0.24.10 commit=7396ea0bc26f -->

### Impact
- **telemetry:** Operators see improved reliability because dedupe secondary intervention debt.

### Why
- **telemetry:** This matters because dedupe secondary intervention debt closes a failure mode or degraded path.

### What Changed
- **telemetry:** Changed dedupe secondary intervention debt (7396ea0).

### Fixes
- **telemetry:** Dedupe secondary intervention debt (7396ea0)

## [0.24.9] - 2026-05-03
<!-- mars-harness-release: version=0.24.9 commit=48655a2079dd -->

### Impact
- **trust:** Operators see improved reliability because honor bootstrap trust defaults.

### Why
- **trust:** This matters because honor bootstrap trust defaults closes a failure mode or degraded path.

### What Changed
- **trust:** Changed honor bootstrap trust defaults (48655a2).

### Fixes
- **trust:** Honor bootstrap trust defaults (48655a2)

## [0.24.8] - 2026-05-03
<!-- mars-harness-release: version=0.24.8 commit=0507ef14d1a4 -->

### Impact
- **tickets:** Operators and future agents get clearer guidance because record persistent store upgrade coverage gap.

### Why
- **tickets:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.

### What Changed
- **tickets:** Changed record persistent store upgrade coverage gap (0507ef1).

### Documentation
- **tickets:** Record persistent store upgrade coverage gap (0507ef1)

## [0.24.7] - 2026-05-03
<!-- mars-harness-release: version=0.24.7 commit=31f16cb2cfc7 -->

### Impact
- **queue:** Operators see improved reliability because migrate legacy job columns before indexes.

### Why
- **queue:** This matters because migrate legacy job columns before indexes closes a failure mode or degraded path.

### What Changed
- **queue:** Changed migrate legacy job columns before indexes (31f16cb).

### Fixes
- **queue:** Migrate legacy job columns before indexes (31f16cb)

## [0.24.6] - 2026-05-03
<!-- mars-harness-release: version=0.24.6 commit=c63ef60fc301 -->

### Impact
- **cli:** Operators see improved reliability because make evidence stores actionable.

### Why
- **cli:** This matters because make evidence stores actionable closes a failure mode or degraded path.

### What Changed
- **cli:** Changed make evidence stores actionable (c63ef60).

### Fixes
- **cli:** Make evidence stores actionable (c63ef60)

## [0.24.5] - 2026-05-03
<!-- mars-harness-release: version=0.24.5 commit=d3add7e85a82 -->

### Impact
- **ci:** Operators see improved reliability because check doctor test file write.

### Why
- **ci:** This matters because check doctor test file write closes a failure mode or degraded path.

### What Changed
- **ci:** Changed check doctor test file write (d3add7e).

### Fixes
- **ci:** Check doctor test file write (d3add7e)

## [0.24.4] - 2026-05-03
<!-- mars-harness-release: version=0.24.4 commit=eda0526868fd -->

### Impact
- **ci:** Operators see improved reliability because check serve test file setup.

### Why
- **ci:** This matters because check serve test file setup closes a failure mode or degraded path.

### What Changed
- **ci:** Changed check serve test file setup (eda0526).

### Fixes
- **ci:** Check serve test file setup (eda0526)

## [0.24.3] - 2026-05-03
<!-- mars-harness-release: version=0.24.3 commit=59f889ea7ed0 -->

### Impact
- **ci:** Operators see improved reliability because clear static lint findings.

### Why
- **ci:** This matters because clear static lint findings closes a failure mode or degraded path.

### What Changed
- **ci:** Changed clear static lint findings (59f889e).

### Fixes
- **ci:** Clear static lint findings (59f889e)

## [0.24.2] - 2026-05-03
<!-- mars-harness-release: version=0.24.2 commit=571bf7138d6c -->

### Impact
- **ci:** Operators see improved reliability because clear remaining lint findings.

### Why
- **ci:** This matters because clear remaining lint findings closes a failure mode or degraded path.

### What Changed
- **ci:** Changed clear remaining lint findings (571bf71).

### Fixes
- **ci:** Clear remaining lint findings (571bf71)

## [0.24.1] - 2026-05-03
<!-- mars-harness-release: version=0.24.1 commit=5a1472e12000 -->

### Impact
- **ci:** Operators see improved reliability because satisfy lint checks.

### Why
- **ci:** This matters because satisfy lint checks closes a failure mode or degraded path.

### What Changed
- **ci:** Changed satisfy lint checks (5a1472e).

### Fixes
- **ci:** Satisfy lint checks (5a1472e)

## [0.24.0] - 2026-05-03
<!-- mars-harness-release: version=0.24.0 commit=8d27b4ed3583 -->

### Impact
- **orchestration:** Operators gain new capability: add dispatch organization layer.

### Why
- **orchestration:** This matters because add dispatch organization layer was missing from the shipped capability set.

### What Changed
- **orchestration:** Changed add dispatch organization layer (8d27b4e).

### Features
- **orchestration:** Add dispatch organization layer (8d27b4e)

## [0.23.0] - 2026-05-03
<!-- mars-harness-release: version=0.23.0 commit=deccb88b12bb -->

### Impact
- **serve:** Operators gain new capability: add native orchestrator survey loop (MH-047).

### Why
- **serve:** This matters because add native orchestrator survey loop (MH-047) was missing from the shipped capability set.

### What Changed
- **serve:** Changed add native orchestrator survey loop (MH-047) (deccb88).

### Features
- **serve:** Add native orchestrator survey loop (MH-047) (deccb88)

### Delivery Evidence
- Enabler work: MH-047: Add native Orchestrator survey loop

## [0.22.1] - 2026-05-03
<!-- mars-harness-release: version=0.22.1 commit=ef3c15d1c115 -->

### Impact
- **sandbox:** Operators see improved reliability because fall back when linux namespaces are unavailable.

### Why
- **sandbox:** This matters because fall back when linux namespaces are unavailable closes a failure mode or degraded path.

### What Changed
- **sandbox:** Changed fall back when linux namespaces are unavailable (ef3c15d).

### Fixes
- **sandbox:** Fall back when linux namespaces are unavailable (ef3c15d)

## [0.22.0] - 2026-05-03
<!-- mars-harness-release: version=0.22.0 commit=44f2c8464e91 -->

### Impact
- **quality:** Operators gain new capability: harden recovery evidence and tool surface.
- **references:** Operators and future agents get clearer guidance because add OpenHarness comparator.

### Why
- **quality:** This matters because harden recovery evidence and tool surface was missing from the shipped capability set.
- **references:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.

### What Changed
- **quality:** Changed harden recovery evidence and tool surface (44f2c84).
- **references:** Changed add OpenHarness comparator (82efaa6).

### Features
- **quality:** Harden recovery evidence and tool surface (44f2c84)

### Documentation
- **references:** Add OpenHarness comparator (82efaa6)

## [0.21.0] - 2026-05-03
<!-- mars-harness-release: version=0.21.0 commit=cb32661e015e -->

### Impact
- **tickets:** Operators gain new capability: enforce in-progress drain states (MH-046).

### Why
- **tickets:** This matters because enforce in-progress drain states (MH-046) was missing from the shipped capability set.

### What Changed
- **tickets:** Changed enforce in-progress drain states (MH-046) (cb32661).

### Features
- **tickets:** Enforce in-progress drain states (MH-046) (cb32661)

### Delivery Evidence
- Enabler work: MH-046: Enforce in-progress ticket drain

## [0.20.0] - 2026-05-03
<!-- mars-harness-release: version=0.20.0 commit=5546e12b1874 -->

### Impact
- **serve:** Operators gain new capability: ingest intervention debt signals (MH-045).

### Why
- **serve:** This matters because ingest intervention debt signals (MH-045) was missing from the shipped capability set.

### What Changed
- **serve:** Changed ingest intervention debt signals (MH-045) (5546e12).

### Features
- **serve:** Ingest intervention debt signals (MH-045) (5546e12)

### Delivery Evidence
- Enabler work: MH-045: Complete intervention-debt signal ingestion

## [0.19.1] - 2026-05-03
<!-- mars-harness-release: version=0.19.1 commit=9b7e4bb50117 -->

### Impact
- **operating-model:** Operators and future agents get clearer guidance because add conversation system record guidance (MH-044).

### Why
- **operating-model:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.

### What Changed
- **operating-model:** Changed add conversation system record guidance (MH-044) (9b7e4bb).

### Documentation
- **operating-model:** Add conversation system record guidance (MH-044) (9b7e4bb)

### Delivery Evidence
- Enabler work: MH-044: Add conversation system record guidance

## [0.19.0] - 2026-05-03
<!-- mars-harness-release: version=0.19.0 commit=6a2d36a9be79 -->

### Impact
- **role-registry:** Operators gain new capability: add checked role inventory (MH-043).

### Why
- **role-registry:** This matters because add checked role inventory (MH-043) was missing from the shipped capability set.

### What Changed
- **role-registry:** Changed add checked role inventory (MH-043) (6a2d36a).

### Features
- **role-registry:** Add checked role inventory (MH-043) (6a2d36a)

### Delivery Evidence
- Enabler work: MH-043: Add checked role registry

## [0.18.0] - 2026-05-03
<!-- mars-harness-release: version=0.18.0 commit=d5436fdefd23 -->

### Impact
- **role-model:** Operators gain new capability: add canonical harness operating domains (MH-042).

### Why
- **role-model:** This matters because add canonical harness operating domains (MH-042) was missing from the shipped capability set.

### What Changed
- **role-model:** Changed add canonical harness operating domains (MH-042) (d5436fd).

### Features
- **role-model:** Add canonical harness operating domains (MH-042) (d5436fd)

### Delivery Evidence
- Enabler work: MH-042: Create canonical harness operating model

## [0.17.0] - 2026-05-03
<!-- mars-harness-release: version=0.17.0 commit=ed664ab2a36e -->

### Impact
- **tools:** Operators gain new capability: add tool creation guard.

### Why
- **tools:** This matters because add tool creation guard was missing from the shipped capability set.

### What Changed
- **tools:** Changed add tool creation guard (ed664ab).

### Features
- **tools:** Add tool creation guard (ed664ab)

## [0.16.0] - 2026-05-03
<!-- mars-harness-release: version=0.16.0 commit=5f9870bd9b08 -->

### Impact
- **models:** Operators gain new capability: add benchmark-backed provider workflow (MH-030).

### Why
- **models:** This matters because add benchmark-backed provider workflow (MH-030) was missing from the shipped capability set.

### What Changed
- **models:** Changed add benchmark-backed provider workflow (MH-030) (5f9870b).

### Features
- **models:** Add benchmark-backed provider workflow (MH-030) (5f9870b)

### Delivery Evidence
- Enabler work: MH-030: Benchmark-backed model refresh and promotion

## [0.15.2] - 2026-05-03
<!-- mars-harness-release: version=0.15.2 commit=027449036856 -->

### Impact
- **tools:** Operators and future agents get clearer guidance because require governed tool creation.

### Why
- **tools:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.

### What Changed
- **tools:** Changed require governed tool creation (0274490).

### Documentation
- **tools:** Require governed tool creation (0274490)

## [0.15.1] - 2026-05-03
<!-- mars-harness-release: version=0.15.1 commit=358216584c40 -->

### Impact
- **features:** Operators and future agents get clearer guidance because expand BDD contract catalog.

### Why
- **features:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.

### What Changed
- **features:** Changed expand BDD contract catalog (3582165).

### Documentation
- **features:** Expand BDD contract catalog (3582165)

## [0.15.0] - 2026-05-03
<!-- mars-harness-release: version=0.15.0 commit=b9b84535812f -->

### Impact
- **tools:** Operators gain new capability: formalize repeated workflow tools.

### Why
- **tools:** This matters because formalize repeated workflow tools was missing from the shipped capability set.

### What Changed
- **tools:** Changed formalize repeated workflow tools (b9b8453).

### Features
- **tools:** Formalize repeated workflow tools (b9b8453)

## [0.14.6] - 2026-05-03
<!-- mars-harness-release: version=0.14.6 commit=3ca1a420b043 -->

### Impact
- **tickets:** Operators and future agents get clearer guidance because complete release asset contract (MH-031).

### Why
- **tickets:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.

### What Changed
- **tickets:** Changed complete release asset contract (MH-031) (3ca1a42).

### Documentation
- **tickets:** Complete release asset contract (MH-031) (3ca1a42)

### Delivery Evidence
- Enabler work: MH-031: Publish release binary assets for installer

## [0.14.5] - 2026-05-03
<!-- mars-harness-release: version=0.14.5 commit=227b6f718abf -->

### Impact
- **ui:** Operators see improved reliability because support linux terminal ioctl constants (MH-031).

### Why
- **ui:** This matters because support linux terminal ioctl constants (MH-031) closes a failure mode or degraded path.

### What Changed
- **ui:** Changed support linux terminal ioctl constants (MH-031) (227b6f7).

### Fixes
- **ui:** Support linux terminal ioctl constants (MH-031) (227b6f7)

## [0.14.4] - 2026-05-03
<!-- mars-harness-release: version=0.14.4 commit=be63396bb21a -->

### Impact
- **release:** Operators see improved reliability because backfill notes-only release assets (MH-031).

### Why
- **release:** This matters because backfill notes-only release assets (MH-031) closes a failure mode or degraded path.

### What Changed
- **release:** Changed backfill notes-only release assets (MH-031) (be63396).

### Fixes
- **release:** Backfill notes-only release assets (MH-031) (be63396)

## [0.14.3] - 2026-05-03
<!-- mars-harness-release: version=0.14.3 commit=ed9853b52bd3 -->

### Impact
- **architecture:** Operators and future agents get clearer guidance because update current system map.

### Why
- **architecture:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.

### What Changed
- **architecture:** Changed update current system map (ed9853b).

### Documentation
- **architecture:** Update current system map (ed9853b)

## [0.14.2] - 2026-05-03
<!-- mars-harness-release: version=0.14.2 commit=9fe9b5857df7 -->

### Impact
- **operating-model:** Operators and future agents get clearer guidance because require symbiotic workflow changes.

### Why
- **operating-model:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.

### What Changed
- **operating-model:** Changed require symbiotic workflow changes (9fe9b58).

### Documentation
- **operating-model:** Require symbiotic workflow changes (9fe9b58)

## [0.14.1] - 2026-05-03
<!-- mars-harness-release: version=0.14.1 commit=195c73e1183c -->

### Impact
- **tools:** Operators and future agents get clearer guidance because add mirrored tools glossary.

### Why
- **tools:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.

### What Changed
- **tools:** Changed add mirrored tools glossary (195c73e).

### Documentation
- **tools:** Add mirrored tools glossary (195c73e)

## [0.14.0] - 2026-05-03
<!-- mars-harness-release: version=0.14.0 commit=422adac6f6ad -->

### Impact
- **tools:** Operators gain new capability: add mirrored mars harness cli tool.

### Why
- **tools:** This matters because add mirrored mars harness cli tool was missing from the shipped capability set.

### What Changed
- **tools:** Changed add mirrored mars harness cli tool (422adac).

### Features
- **tools:** Add mirrored mars harness cli tool (422adac)

## [0.13.1] - 2026-05-03
<!-- mars-harness-release: version=0.13.1 commit=cd0ffd67d96a -->

### Impact
- **tickets:** Operators and future agents get clearer guidance because record release asset blocker (MH-031).

### Why
- **tickets:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.

### What Changed
- **tickets:** Changed record release asset blocker (MH-031) (cd0ffd6).

### Documentation
- **tickets:** Record release asset blocker (MH-031) (cd0ffd6)

## [0.13.0] - 2026-05-03
<!-- mars-harness-release: version=0.13.0 commit=ccd36bc3bf3d -->

### Impact
- **release:** Operators gain new capability: verify release assets for self-update (MH-031).
- **release:** Operators see improved reliability because ignore stale changelog markers (MH-031).

### Why
- **release:** This matters because verify release assets for self-update (MH-031) was missing from the shipped capability set.
- **release:** This matters because ignore stale changelog markers (MH-031) closes a failure mode or degraded path.

### What Changed
- **release:** Changed verify release assets for self-update (MH-031) (9e027a2).
- **release:** Changed ignore stale changelog markers (MH-031) (ccd36bc).

### Features
- **release:** Verify release assets for self-update (MH-031) (9e027a2)

### Fixes
- **release:** Ignore stale changelog markers (MH-031) (ccd36bc)

## [0.12.1] - 2026-05-03
<!-- mars-harness-release: version=0.12.1 commit=d8e8c6fcc990 -->

### Impact
- **glossary:** Operators and future agents get clearer guidance because define operating model distinctions.

### Why
- **glossary:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.

### What Changed
- **glossary:** Changed define operating model distinctions (d8e8c6f).

### Documentation
- **glossary:** Define operating model distinctions (d8e8c6f)

## [0.12.0] - 2026-05-03
<!-- mars-harness-release: version=0.12.0 commit=416a91bd5fa1 -->

### Impact
- **scoring:** Operators gain new capability: export repo quality score (MH-037).

### Why
- **scoring:** This matters because export repo quality score (MH-037) was missing from the shipped capability set.

### What Changed
- **scoring:** Changed export repo quality score (MH-037) (416a91b).

### Features
- **scoring:** Export repo quality score (MH-037) (416a91b)

### Delivery Evidence
- Enabler work: MH-037: Automate quality score export

## [0.11.1] - 2026-05-03
<!-- mars-harness-release: version=0.11.1 commit=450d1bbbbbd9 -->

### Impact
- **tools:** Operators see improved reliability because mirror tool_create in target harness.

### Why
- **tools:** This matters because mirror tool_create in target harness closes a failure mode or degraded path.

### What Changed
- **tools:** Changed mirror tool_create in target harness (450d1bb).

### Fixes
- **tools:** Mirror tool_create in target harness (450d1bb)

## [0.11.0] - 2026-05-03
<!-- mars-harness-release: version=0.11.0 commit=a00bb9e11730 -->

### Impact
- **tools:** Operators gain new capability: add tool creation scaffold.

### Why
- **tools:** This matters because add tool creation scaffold was missing from the shipped capability set.

### What Changed
- **tools:** Changed add tool creation scaffold (a00bb9e).

### Features
- **tools:** Add tool creation scaffold (a00bb9e)

## [0.10.4] - 2026-05-03
<!-- mars-harness-release: version=0.10.4 commit=f133e393f97a -->

### Impact
- **glossary:** Operators and future agents get clearer guidance because mirror harness terminology.

### Why
- **glossary:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.

### What Changed
- **glossary:** Changed mirror harness terminology (f133e39).

### Documentation
- **glossary:** Mirror harness terminology (f133e39)

## [0.10.3] - 2026-05-03
<!-- mars-harness-release: version=0.10.3 commit=9e444541196c -->

### Impact
- **planning:** Operators and future agents get clearer guidance because materialize mars parity backlog tickets (MH-035).

### Why
- **planning:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.

### What Changed
- **planning:** Changed materialize mars parity backlog tickets (MH-035) (9e44454).

### Documentation
- **planning:** Materialize mars parity backlog tickets (MH-035) (9e44454)

### Delivery Evidence
- Enabler work: MH-035: Materialize Mars parity workstreams as tickets

## [0.10.2] - 2026-05-03
<!-- mars-harness-release: version=0.10.2 commit=e2bcf2f7a080 -->

### Impact
- **telemetry:** Operators see improved reliability because classify ticket gate failures.

### Why
- **telemetry:** This matters because classify ticket gate failures closes a failure mode or degraded path.

### What Changed
- **telemetry:** Changed classify ticket gate failures (e2bcf2f).

### Fixes
- **telemetry:** Classify ticket gate failures (e2bcf2f)

## [0.10.1] - 2026-05-03
<!-- mars-harness-release: version=0.10.1 commit=bb885cd5ad7e -->

### Impact
- **inference:** Operators see improved reliability because surface installed model variants.

### Why
- **inference:** This matters because surface installed model variants closes a failure mode or degraded path.

### What Changed
- **inference:** Changed surface installed model variants (bb885cd).

### Fixes
- **inference:** Surface installed model variants (bb885cd)

## [0.10.0] - 2026-05-03
<!-- mars-harness-release: version=0.10.0 commit=0f4d9ec86ceb -->

### Impact
- **planhygiene:** Operators gain new capability: add active plan hygiene checker (MH-034).

### Why
- **planhygiene:** This matters because add active plan hygiene checker (MH-034) was missing from the shipped capability set.

### What Changed
- **planhygiene:** Changed add active plan hygiene checker (MH-034) (0f4d9ec).

### Features
- **planhygiene:** Add active plan hygiene checker (MH-034) (0f4d9ec)

### Delivery Evidence
- Enabler work: MH-034: Implement active-plan hygiene checker

## [0.9.0] - 2026-05-02
<!-- mars-harness-release: version=0.9.0 commit=c3a87e2179e3 -->

### Impact
- **setup:** Operators gain new capability: configure shell path automatically (MH-041).

### Why
- **setup:** This matters because configure shell path automatically (MH-041) was missing from the shipped capability set.

### What Changed
- **setup:** Changed configure shell path automatically (MH-041) (c3a87e2).

### Features
- **setup:** Configure shell path automatically (MH-041) (c3a87e2)

### Delivery Evidence
- Shipped feature scenarios: MH-041: F-002-S001, F-002-S002, F-002-S003, F-002-S004, F-002-S005

## [0.8.0] - 2026-05-02
<!-- mars-harness-release: version=0.8.0 commit=cd7514dfdce5 -->

### Impact
- **operating-model:** Operators gain new capability: implement BDD-led delivery loop (MH-040).

### Why
- **operating-model:** This matters because implement BDD-led delivery loop (MH-040) was missing from the shipped capability set.

### What Changed
- **operating-model:** Changed implement BDD-led delivery loop (MH-040) (cd7514d).

### Features
- **operating-model:** Implement BDD-led delivery loop (MH-040) (cd7514d)

### Delivery Evidence
- Shipped feature scenarios: MH-040: F-001-S001, F-001-S002, F-001-S003, F-001-S004, F-001-S005, F-001-S006

## [0.7.5] - 2026-05-02
<!-- mars-harness-release: version=0.7.5 commit=e39e335c8fc2 -->

### Impact
- **plans:** Operators and future agents get clearer guidance because add exec plan dependency metadata.

### Why
- **plans:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.

### What Changed
- **plans:** Changed add exec plan dependency metadata (e39e335).

### Documentation
- **plans:** Add exec plan dependency metadata (e39e335)

## [0.7.4] - 2026-05-02
<!-- mars-harness-release: version=0.7.4 commit=c7dbdf3dbb6e -->

### Impact
- **plans:** Operators and future agents get clearer guidance because enforce single active exec plan.

### Why
- **plans:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.

### What Changed
- **plans:** Changed enforce single active exec plan (c7dbdf3).

### Documentation
- **plans:** Enforce single active exec plan (c7dbdf3)

## [0.7.3] - 2026-05-02
<!-- mars-harness-release: version=0.7.3 commit=9a4ced42bdad -->

### Impact
- **scoring:** Operators and future agents get clearer guidance because seed quality score artifact.

### Why
- **scoring:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.

### What Changed
- **scoring:** Changed seed quality score artifact (9a4ced4).

### Documentation
- **scoring:** Seed quality score artifact (9a4ced4)

## [0.7.2] - 2026-05-02
<!-- mars-harness-release: version=0.7.2 commit=dac23b716ce3 -->

### Impact
- **plans:** Operators and future agents get clearer guidance because reconcile current execution state.

### Why
- **plans:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.

### What Changed
- **plans:** Changed reconcile current execution state (dac23b7).

### Documentation
- **plans:** Reconcile current execution state (dac23b7)

## [0.7.1] - 2026-05-02
<!-- mars-harness-release: version=0.7.1 commit=21d617f832ef -->

### Impact
- **update:** The release carries stronger evidence because keep version drift fixtures release-agnostic.

### Why
- **update:** This matters because the project needs durable evidence that the behavior keeps working.

### What Changed
- **update:** Changed keep version drift fixtures release-agnostic (21d617f).

### Tests
- **update:** Keep version drift fixtures release-agnostic (21d617f)

## [0.7.0] - 2026-05-02
<!-- mars-harness-release: version=0.7.0 commit=ce831c5cd4de -->

### Impact
- **update:** Operators gain new capability: check tool and harness version drift.

### Why
- **update:** This matters because check tool and harness version drift was missing from the shipped capability set.

### What Changed
- **update:** Changed check tool and harness version drift (ce831c5).

### Features
- **update:** Check tool and harness version drift (ce831c5)

## [0.6.0] - 2026-05-02
<!-- mars-harness-release: version=0.6.0 commit=2187d5a379c3 -->

### Impact
- **update:** Operators gain new capability: unify tool and harness updates.

### Why
- **update:** This matters because unify tool and harness updates was missing from the shipped capability set.

### What Changed
- **update:** Changed unify tool and harness updates (2187d5a).

### Features
- **update:** Unify tool and harness updates (2187d5a)

## [0.5.3] - 2026-05-02
<!-- mars-harness-release: version=0.5.3 commit=781c1e5051dd -->

### Impact
- **setup:** Operators see improved reliability because clarify source install workflow.

### Why
- **setup:** This matters because clarify source install workflow closes a failure mode or degraded path.

### What Changed
- **setup:** Changed clarify source install workflow (781c1e5).

### Fixes
- **setup:** Clarify source install workflow (781c1e5)

## [0.5.2] - 2026-05-02
<!-- mars-harness-release: version=0.5.2 commit=4a599310de29 -->

### Impact
- **models:** Operators and future agents get clearer guidance because define ollama swap policy.

### Why
- **models:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.

### What Changed
- **models:** Changed define ollama swap policy (4a59931).

### Documentation
- **models:** Define ollama swap policy (4a59931)

## [0.5.1] - 2026-05-02
<!-- mars-harness-release: version=0.5.1 commit=8f0a44f12017 -->

### Impact
- **telemetry:** Operators see improved reliability because keep intervention tickets independent.

### Why
- **telemetry:** This matters because keep intervention tickets independent closes a failure mode or degraded path.

### What Changed
- **telemetry:** Changed keep intervention tickets independent (8f0a44f).

### Fixes
- **telemetry:** Keep intervention tickets independent (8f0a44f)

## [0.5.0] - 2026-05-02
<!-- mars-harness-release: version=0.5.0 commit=0ca0257223cd -->

### Impact
- **telemetry:** Operators gain new capability: create intervention-debt tickets.

### Why
- **telemetry:** This matters because create intervention-debt tickets was missing from the shipped capability set.

### What Changed
- **telemetry:** Changed create intervention-debt tickets (0ca0257).

### Features
- **telemetry:** Create intervention-debt tickets (0ca0257)

## [0.4.1] - 2026-05-02
<!-- mars-harness-release: version=0.4.1 commit=548fb73403a1 -->

### Impact
- **inference:** Operators see improved reliability because route roles by manifest tier.

### Why
- **inference:** This matters because route roles by manifest tier closes a failure mode or degraded path.

### What Changed
- **inference:** Changed route roles by manifest tier (548fb73).

### Fixes
- **inference:** Route roles by manifest tier (548fb73)

## [0.4.0] - 2026-05-02
<!-- mars-harness-release: version=0.4.0 commit=72032c5985e4 -->

### Impact
- **models:** Operators gain new capability: add benchmark evaluation path.

### Why
- **models:** This matters because add benchmark evaluation path was missing from the shipped capability set.

### What Changed
- **models:** Changed add benchmark evaluation path (72032c5).

### Features
- **models:** Add benchmark evaluation path (72032c5)

## [0.3.6] - 2026-05-02
<!-- mars-harness-release: version=0.3.6 commit=ecf0f5596249 -->

### Impact
- **queue:** Operators see improved reliability because self-heal recovery storms.

### Why
- **queue:** This matters because self-heal recovery storms closes a failure mode or degraded path.

### What Changed
- **queue:** Changed self-heal recovery storms (ecf0f55).

### Fixes
- **queue:** Self-heal recovery storms (ecf0f55)

## [0.3.5] - 2026-05-02
<!-- mars-harness-release: version=0.3.5 commit=4769fb4172da -->

### Impact
- **serve:** Operators see improved reliability because contain recursive recovery jobs.

### Why
- **serve:** This matters because contain recursive recovery jobs closes a failure mode or degraded path.

### What Changed
- **serve:** Changed contain recursive recovery jobs (4769fb4).

### Fixes
- **serve:** Contain recursive recovery jobs (4769fb4)

## [0.3.4] - 2026-05-02
<!-- mars-harness-release: version=0.3.4 commit=5fef93f4bc04 -->

### Impact
- **release:** Operators and future agents get clearer guidance because require github release publication.

### Why
- **release:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.

### What Changed
- **release:** Changed require github release publication (5fef93f).

### Documentation
- **release:** Require github release publication (5fef93f)

## [0.3.3] - 2026-05-02
<!-- mars-harness-release: version=0.3.3 commit=3232920f527f -->

### Impact
- **harness:** Operators and future agents get clearer guidance because mirror operating rules into targets.

### Why
- **harness:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.

### What Changed
- **harness:** Changed mirror operating rules into targets (3232920).

### Documentation
- **harness:** Mirror operating rules into targets (3232920)

## [0.3.2] - 2026-05-02
<!-- mars-harness-release: version=0.3.2 commit=5c5bc2d6761b -->

### Impact
- **release:** Operators and future agents get clearer guidance because mirror versioning rule into targets.

### Why
- **release:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.

### What Changed
- **release:** Changed mirror versioning rule into targets (5c5bc2d).

### Documentation
- **release:** Mirror versioning rule into targets (5c5bc2d)

## [0.3.1] - 2026-05-02
<!-- mars-harness-release: version=0.3.1 commit=466bc65ad438 -->

### Impact
- **release:** Operators and future agents get clearer guidance because require versioning after source commits.

### Why
- **release:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.

### What Changed
- **release:** Changed require versioning after source commits (466bc65).

### Documentation
- **release:** Require versioning after source commits (466bc65)

## [0.3.0] - 2026-05-02
<!-- mars-harness-release: version=0.3.0 commit=b2cd7df5f2e5 -->

### Impact
- **skills:** Operators gain new capability: guide self-improving skill evolution.

### Why
- **skills:** This matters because guide self-improving skill evolution was missing from the shipped capability set.

### What Changed
- **skills:** Changed guide self-improving skill evolution (b2cd7df).

### Features
- **skills:** Guide self-improving skill evolution (b2cd7df)

## [0.2.0] - 2026-05-02
<!-- mars-harness-release: version=0.2.0 commit=15f4b154182d -->

### Impact
- **release:** Operators gain new capability: automate semantic patch notes.

### Why
- **release:** This matters because automate semantic patch notes was missing from the shipped capability set.

### What Changed
- **release:** Changed automate semantic patch notes (15f4b15).

### Features
- **release:** Automate semantic patch notes (15f4b15)

## [0.1.0] - 2026-05-02
<!-- mars-harness-release: version=0.1.0 commit=edaafeacae3a -->

### Impact
- **tools:** Operators gain new capability: mechanical ticket deduplication with ticket_create tool (AD-030).
- **tools,scanner:** Operators gain new capability: wire git tools into manifests and add commit gates (AD-028).
- **pipeline:** Operators gain new capability: chain dogfood tester after engineer completes a feature.
- Operators gain new capability: dogfood E2E validation with Podman + decision recording system.
- **dashboard,scanner:** Operators gain new capability: dynamic pipeline chain + tsconfig path alias check.
- **m6,m7:** Operators gain new capability: scoring engine, progressive autonomy, evolution system, guardrails engine (MH-017 through MH-021).
- **telemetry:** Operators gain new capability: triage self-improvement signals.
- **tools:** Operators gain new capability: add registry, executor, and core tools (M1.2 / MH-002).
- **scanner:** Operators gain new capability: add bootability checks for framework-specific validation (AD-026).
- **m5:** Operators gain new capability: job queue, scheduler, sandbox, safety, serve command (MH-013, MH-014, MH-015, MH-016).
- **serve:** Operators gain new capability: wire autonomous orchestrator with repo registry, trigger router, and executor.
- **agent:** Operators gain new capability: add conversation loop, parser, and tests (M1.3 / MH-003).
- **scanner:** Operators gain new capability: add mars-harness upgrade command to sync target project manifests and prompts.
- **setup:** Operators gain new capability: plug-and-play model download from HuggingFace.
- **skills:** Operators gain new capability: add agent skills across Cursor, AGENTS.md, and .harness/skills/.
- **ui:** Operators gain new capability: cursor-quality agent output with role banner, tool trace, and handoff.
- **setup:** Operators gain new capability: auto-install llama-server and wire run command.
- **llm:** Operators gain new capability: add OpenAI-compatible chat client and testutil helpers.
- **init:** Operators gain new capability: provision full 11-role Mars pipeline by default (Tenet 1).
- **serve:** Operators gain new capability: parallel pipeline tracks with sleep resilience.
- **dashboard:** Operators gain new capability: wire dashboard into orchestrator with SSE events.
- Operators gain new capability: implement two-level self-learning system with janitor agent.
- **init:** Operators gain new capability: auto-run git init when .git is missing.
- **cli:** Operators gain new capability: auto-init .harness when manifest is missing.
- **m9:** Operators gain new capability: embedded dashboard with htmx, Chart.js, SSE, emergency stop (MH-024).
- **context:** Operators gain new capability: add context assembler with token budget (M1.4 / MH-004).
- **m8:** Operators gain new capability: setup wizard, init + scanner, doctor command (MH-022, MH-023, MH-028).
- **init:** Operators gain new capability: scaffold full docs/ structure with Mars-quality role prompts.
- **telemetry:** Operators gain new capability: add telemetry collector with error classification and auto-fix.
- **core:** Operators gain new capability: enforce strict trunk execution safety.
- **tools:** Operators gain new capability: add background mode to shell_exec for long-running processes.
- **m3,m4:** Operators gain new capability: cLI framework, bundle reader, terminal UI, GitHub integration (MH-009, MH-010, MH-011, MH-012).
- **inference:** Operators gain new capability: add llama-server subprocess manager and role router.
- **context:** Operators gain new capability: file filter, budget partitioning, .harnessignore support (M1.4.2, M1.4.4, MH-004).
- **m10:** Operators gain new capability: role prompts, dogfood workflow, cross-compile release, user docs (MH-025, MH-026, MH-027).
- **start:** Operators gain new capability: add `mars-harness start` for full e2e pipeline orchestration.
- **dashboard:** Operators gain new capability: implement throughput page with chart, stats, and job table.
- **m2:** Operators gain new capability: hardware detection, model download, llama-server management, LLM router (MH-006, MH-007, MH-008).
- **orchestrator:** Operators gain new capability: configurable agent triggers with chaining and custom cron.
- **agent:** Operators gain new capability: m1 closeout — 3+ tool-call test, trace error propagation, realistic 5-turn test (MH-003, MH-005).
- **inference:** Operators gain new capability: expose local performance tuning.
- **prompts:** Operators gain new capability: add git push after every commit in all role prompts.
- **serve:** Operators gain new capability: per-repo database isolation to prevent cross-project contamination (AD-029).
- **init:** Operators gain new capability: mirror harness context glossary.
- **trace:** Operators gain new capability: add JSONL recorder and SQLite store (M1.5 / MH-005).
- **tools:** Operators see improved reliability because kill entire process group on shell_exec timeout.
- **inference:** Operators see improved reliability because mitigate LLM timeout, context overflow, and connection refused failures.
- **prompts:** Operators see improved reliability because cold-start CEO/COO prompts for new projects.
- **setup:** Operators see improved reliability because pin local runtime artifacts.
- **serve:** Operators see improved reliability because block engineer handoff with active tickets.
- **agent:** Operators see improved reliability because always serialize message content field for llama.cpp compat.
- **agent:** Operators see improved reliability because add context window guard to prevent token overflow.
- Operators see improved reliability because resolve audit findings — broken links, naming inconsistencies, missing docs (M0).
- **init:** Operators see improved reliability because preserve existing user content on --force re-init.
- **core:** Operators see improved reliability because auto-tune inference and drain active tickets.
- **github:** Operators see improved reliability because keep integrations trunk oriented.
- Operators see improved reliability because correct module path, add gitkeeps, fix placeholders (M0).
- **pipeline:** Operators see improved reliability because restore COO → Engineer handoff for delivery kickoff.
- **upgrade:** Operators see improved reliability because preserve user configured agents.
- **serve:** Operators see improved reliability because auto-cleanup stale processes and corrupt DB on start/serve.
- **dashboard:** Operators see improved reliability because constrain chart canvas height to 280px.
- **references:** Operators and future agents get clearer guidance because carry mars agent-first references.
- **generated:** Operators and future agents get clearer guidance because define generated docs contract.
- **tickets:** Operators and future agents get clearer guidance because populate full backlog MH-001 through MH-028 (M0).
- Operators and future agents get clearer guidance because add terminology definitions and dual-repo commit discipline.
- **product:** Operators and future agents get clearer guidance because refresh living product specs.
- Operators and future agents get clearer guidance because switch to trunk-based development, drop branch/PR requirement.
- **workflow:** Operators and future agents get clearer guidance because align generated bundles with strict trunk.
- **design:** Operators and future agents get clearer guidance because record AD-031 inference resilience decisions.
- **references:** Operators and future agents get clearer guidance because audit mars relevance for harness parity.
- **exec-plans:** Operators and future agents get clearer guidance because add master execution plan with M0–M10 + MH-001–MH-028 coverage.
- **plans:** Operators and future agents get clearer guidance because add mars supersession parity plan.
- Operators and future agents get clearer guidance because add AD-021 through AD-025 for dogfood, decisions, and lean pipeline.
- **tickets:** Operators and future agents get clearer guidance because reconcile 19 ticket-vs-schedule contradictions (C1–C19).
- **tickets:** Maintainers get a healthier project surface because move MH-001 through MH-005 to done/ (M1 closeout).
- **m0:** Maintainers get a healthier project surface because audit and fix M0 quality gate gaps.
- **tickets:** Maintainers get a healthier project surface because move MH-006 through MH-008 to done/ (M2 closeout).
- Maintainers get a healthier project surface because initialize mars-harness repo (M0).
- **tickets:** Maintainers get a healthier project surface because move MH-009 through MH-012 to done/ (M3+M4 closeout).
- **tickets:** Maintainers get a healthier project surface because move MH-017 through MH-021 to done/ (M6+M7 closeout).
- **tickets:** Maintainers get a healthier project surface because move MH-013 through MH-016 to done/ (M5a+M5b closeout).
- **serve:** The release carries stronger evidence because fix serve tests for new Config requirements, add skills loader tests.

### Why
- **tools:** This matters because mechanical ticket deduplication with ticket_create tool (AD-030) was missing from the shipped capability set.
- **tools,scanner:** This matters because wire git tools into manifests and add commit gates (AD-028) was missing from the shipped capability set.
- **pipeline:** This matters because chain dogfood tester after engineer completes a feature was missing from the shipped capability set.
- This matters because dogfood E2E validation with Podman + decision recording system was missing from the shipped capability set.
- **dashboard,scanner:** This matters because dynamic pipeline chain + tsconfig path alias check was missing from the shipped capability set.
- **m6,m7:** This matters because scoring engine, progressive autonomy, evolution system, guardrails engine (MH-017 through MH-021) was missing from the shipped capability set.
- **telemetry:** This matters because triage self-improvement signals was missing from the shipped capability set.
- **tools:** This matters because add registry, executor, and core tools (M1.2 / MH-002) was missing from the shipped capability set.
- **scanner:** This matters because add bootability checks for framework-specific validation (AD-026) was missing from the shipped capability set.
- **m5:** This matters because job queue, scheduler, sandbox, safety, serve command (MH-013, MH-014, MH-015, MH-016) was missing from the shipped capability set.
- **serve:** This matters because wire autonomous orchestrator with repo registry, trigger router, and executor was missing from the shipped capability set.
- **agent:** This matters because add conversation loop, parser, and tests (M1.3 / MH-003) was missing from the shipped capability set.
- **scanner:** This matters because add mars-harness upgrade command to sync target project manifests and prompts was missing from the shipped capability set.
- **setup:** This matters because plug-and-play model download from HuggingFace was missing from the shipped capability set.
- **skills:** This matters because add agent skills across Cursor, AGENTS.md, and .harness/skills/ was missing from the shipped capability set.
- **ui:** This matters because cursor-quality agent output with role banner, tool trace, and handoff was missing from the shipped capability set.
- **setup:** This matters because auto-install llama-server and wire run command was missing from the shipped capability set.
- **llm:** This matters because add OpenAI-compatible chat client and testutil helpers was missing from the shipped capability set.
- **init:** This matters because provision full 11-role Mars pipeline by default (Tenet 1) was missing from the shipped capability set.
- **serve:** This matters because parallel pipeline tracks with sleep resilience was missing from the shipped capability set.
- **dashboard:** This matters because wire dashboard into orchestrator with SSE events was missing from the shipped capability set.
- This matters because implement two-level self-learning system with janitor agent was missing from the shipped capability set.
- **init:** This matters because auto-run git init when .git is missing was missing from the shipped capability set.
- **cli:** This matters because auto-init .harness when manifest is missing was missing from the shipped capability set.
- **m9:** This matters because embedded dashboard with htmx, Chart.js, SSE, emergency stop (MH-024) was missing from the shipped capability set.
- **context:** This matters because add context assembler with token budget (M1.4 / MH-004) was missing from the shipped capability set.
- **m8:** This matters because setup wizard, init + scanner, doctor command (MH-022, MH-023, MH-028) was missing from the shipped capability set.
- **init:** This matters because scaffold full docs/ structure with Mars-quality role prompts was missing from the shipped capability set.
- **telemetry:** This matters because add telemetry collector with error classification and auto-fix was missing from the shipped capability set.
- **core:** This matters because enforce strict trunk execution safety was missing from the shipped capability set.
- **tools:** This matters because add background mode to shell_exec for long-running processes was missing from the shipped capability set.
- **m3,m4:** This matters because cLI framework, bundle reader, terminal UI, GitHub integration (MH-009, MH-010, MH-011, MH-012) was missing from the shipped capability set.
- **inference:** This matters because add llama-server subprocess manager and role router was missing from the shipped capability set.
- **context:** This matters because file filter, budget partitioning, .harnessignore support (M1.4.2, M1.4.4, MH-004) was missing from the shipped capability set.
- **m10:** This matters because role prompts, dogfood workflow, cross-compile release, user docs (MH-025, MH-026, MH-027) was missing from the shipped capability set.
- **start:** This matters because add `mars-harness start` for full e2e pipeline orchestration was missing from the shipped capability set.
- **dashboard:** This matters because implement throughput page with chart, stats, and job table was missing from the shipped capability set.
- **m2:** This matters because hardware detection, model download, llama-server management, LLM router (MH-006, MH-007, MH-008) was missing from the shipped capability set.
- **orchestrator:** This matters because configurable agent triggers with chaining and custom cron was missing from the shipped capability set.
- **agent:** This matters because m1 closeout — 3+ tool-call test, trace error propagation, realistic 5-turn test (MH-003, MH-005) was missing from the shipped capability set.
- **inference:** This matters because expose local performance tuning was missing from the shipped capability set.
- **prompts:** This matters because add git push after every commit in all role prompts was missing from the shipped capability set.
- **serve:** This matters because per-repo database isolation to prevent cross-project contamination (AD-029) was missing from the shipped capability set.
- **init:** This matters because mirror harness context glossary was missing from the shipped capability set.
- **trace:** This matters because add JSONL recorder and SQLite store (M1.5 / MH-005) was missing from the shipped capability set.
- **tools:** This matters because kill entire process group on shell_exec timeout closes a failure mode or degraded path.
- **inference:** This matters because mitigate LLM timeout, context overflow, and connection refused failures closes a failure mode or degraded path.
- **prompts:** This matters because cold-start CEO/COO prompts for new projects closes a failure mode or degraded path.
- **setup:** This matters because pin local runtime artifacts closes a failure mode or degraded path.
- **serve:** This matters because block engineer handoff with active tickets closes a failure mode or degraded path.
- **agent:** This matters because always serialize message content field for llama.cpp compat closes a failure mode or degraded path.
- **agent:** This matters because add context window guard to prevent token overflow closes a failure mode or degraded path.
- This matters because resolve audit findings — broken links, naming inconsistencies, missing docs (M0) closes a failure mode or degraded path.
- **init:** This matters because preserve existing user content on --force re-init closes a failure mode or degraded path.
- **core:** This matters because auto-tune inference and drain active tickets closes a failure mode or degraded path.
- **github:** This matters because keep integrations trunk oriented closes a failure mode or degraded path.
- This matters because correct module path, add gitkeeps, fix placeholders (M0) closes a failure mode or degraded path.
- **pipeline:** This matters because restore COO → Engineer handoff for delivery kickoff closes a failure mode or degraded path.
- **upgrade:** This matters because preserve user configured agents closes a failure mode or degraded path.
- **serve:** This matters because auto-cleanup stale processes and corrupt DB on start/serve closes a failure mode or degraded path.
- **dashboard:** This matters because constrain chart canvas height to 280px closes a failure mode or degraded path.
- **references:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.
- **generated:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.
- **tickets:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.
- This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.
- **product:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.
- This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.
- **workflow:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.
- **design:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.
- **references:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.
- **exec-plans:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.
- **plans:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.
- This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.
- **tickets:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.
- **tickets:** This matters because project health work keeps future delivery predictable.
- **m0:** This matters because project health work keeps future delivery predictable.
- **tickets:** This matters because project health work keeps future delivery predictable.
- This matters because project health work keeps future delivery predictable.
- **tickets:** This matters because project health work keeps future delivery predictable.
- **tickets:** This matters because project health work keeps future delivery predictable.
- **tickets:** This matters because project health work keeps future delivery predictable.
- **serve:** This matters because the project needs durable evidence that the behavior keeps working.

### What Changed
- **tools:** Changed mechanical ticket deduplication with ticket_create tool (AD-030) (0322c0c).
- **tools,scanner:** Changed wire git tools into manifests and add commit gates (AD-028) (1091028).
- **pipeline:** Changed chain dogfood tester after engineer completes a feature (10c845d).
- Changed dogfood E2E validation with Podman + decision recording system (147952e).
- **dashboard,scanner:** Changed dynamic pipeline chain + tsconfig path alias check (168a354).
- **m6,m7:** Changed scoring engine, progressive autonomy, evolution system, guardrails engine (MH-017 through MH-021) (1b9890e).
- **telemetry:** Changed triage self-improvement signals (28082ae).
- **tools:** Changed add registry, executor, and core tools (M1.2 / MH-002) (3083f9e).
- **scanner:** Changed add bootability checks for framework-specific validation (AD-026) (38c491f).
- **m5:** Changed job queue, scheduler, sandbox, safety, serve command (MH-013, MH-014, MH-015, MH-016) (45365ca).
- **serve:** Changed wire autonomous orchestrator with repo registry, trigger router, and executor (498d349).
- **agent:** Changed add conversation loop, parser, and tests (M1.3 / MH-003) (5394612).
- **scanner:** Changed add mars-harness upgrade command to sync target project manifests and prompts (5b53d26).
- **setup:** Changed plug-and-play model download from HuggingFace (5c91811).
- **skills:** Changed add agent skills across Cursor, AGENTS.md, and .harness/skills/ (73a0a82).
- **ui:** Changed cursor-quality agent output with role banner, tool trace, and handoff (7bf1294).
- **setup:** Changed auto-install llama-server and wire run command (7ddea6e).
- **llm:** Changed add OpenAI-compatible chat client and testutil helpers (7e50d1d).
- **init:** Changed provision full 11-role Mars pipeline by default (Tenet 1) (88de7a0).
- **serve:** Changed parallel pipeline tracks with sleep resilience (89b7895).
- **dashboard:** Changed wire dashboard into orchestrator with SSE events (8dce248).
- Changed implement two-level self-learning system with janitor agent (8f86add).
- **init:** Changed auto-run git init when .git is missing (8fc2260).
- **cli:** Changed auto-init .harness when manifest is missing (907d23a).
- **m9:** Changed embedded dashboard with htmx, Chart.js, SSE, emergency stop (MH-024) (9f7c083).
- **context:** Changed add context assembler with token budget (M1.4 / MH-004) (a07bfc9).
- **m8:** Changed setup wizard, init + scanner, doctor command (MH-022, MH-023, MH-028) (a26dccc).
- **init:** Changed scaffold full docs/ structure with Mars-quality role prompts (a6f7b26).
- **telemetry:** Changed add telemetry collector with error classification and auto-fix (ab150c5).
- **core:** Changed enforce strict trunk execution safety (b11daca).
- **tools:** Changed add background mode to shell_exec for long-running processes (b898608).
- **m3,m4:** Changed cLI framework, bundle reader, terminal UI, GitHub integration (MH-009, MH-010, MH-011, MH-012) (bb95350).
- **inference:** Changed add llama-server subprocess manager and role router (ce12a9d).
- **context:** Changed file filter, budget partitioning, .harnessignore support (M1.4.2, M1.4.4, MH-004) (d264010).
- **m10:** Changed role prompts, dogfood workflow, cross-compile release, user docs (MH-025, MH-026, MH-027) (dcac2af).
- **start:** Changed add `mars-harness start` for full e2e pipeline orchestration (deb1cd3).
- **dashboard:** Changed implement throughput page with chart, stats, and job table (df3b70d).
- **m2:** Changed hardware detection, model download, llama-server management, LLM router (MH-006, MH-007, MH-008) (e3436b9).
- **orchestrator:** Changed configurable agent triggers with chaining and custom cron (ec4db54).
- **agent:** Changed m1 closeout — 3+ tool-call test, trace error propagation, realistic 5-turn test (MH-003, MH-005) (ece2768).
- **inference:** Changed expose local performance tuning (ed24f83).
- **prompts:** Changed add git push after every commit in all role prompts (efce0b9).
- **serve:** Changed per-repo database isolation to prevent cross-project contamination (AD-029) (f098c2e).
- **init:** Changed mirror harness context glossary (f23663d).
- **trace:** Changed add JSONL recorder and SQLite store (M1.5 / MH-005) (f862f46).
- **tools:** Changed kill entire process group on shell_exec timeout (2ecc1ad).
- **inference:** Changed mitigate LLM timeout, context overflow, and connection refused failures (64ca767).
- **prompts:** Changed cold-start CEO/COO prompts for new projects (6eff4e9).
- **setup:** Changed pin local runtime artifacts (7e1a85e).
- **serve:** Changed block engineer handoff with active tickets (7fd00a8).
- **agent:** Changed always serialize message content field for llama.cpp compat (927cdb5).
- **agent:** Changed add context window guard to prevent token overflow (a045bd0).
- Changed resolve audit findings — broken links, naming inconsistencies, missing docs (M0) (cb17a6e).
- **init:** Changed preserve existing user content on --force re-init (ccacf88).
- **core:** Changed auto-tune inference and drain active tickets (e1fd6e0).
- **github:** Changed keep integrations trunk oriented (e45a90e).
- Changed correct module path, add gitkeeps, fix placeholders (M0) (ea287fc).
- **pipeline:** Changed restore COO → Engineer handoff for delivery kickoff (ebfaa56).
- **upgrade:** Changed preserve user configured agents (edaafea).
- **serve:** Changed auto-cleanup stale processes and corrupt DB on start/serve (f3af248).
- **dashboard:** Changed constrain chart canvas height to 280px (f9620aa).
- **references:** Changed carry mars agent-first references (009358b).
- **generated:** Changed define generated docs contract (1c0f043).
- **tickets:** Changed populate full backlog MH-001 through MH-028 (M0) (2c508cf).
- Changed add terminology definitions and dual-repo commit discipline (3759dc9).
- **product:** Changed refresh living product specs (4806d8f).
- Changed switch to trunk-based development, drop branch/PR requirement (584e4d7).
- **workflow:** Changed align generated bundles with strict trunk (69b608b).
- **design:** Changed record AD-031 inference resilience decisions (838e29c).
- **references:** Changed audit mars relevance for harness parity (92d7b8b).
- **exec-plans:** Changed add master execution plan with M0–M10 + MH-001–MH-028 coverage (9d13b8e).
- **plans:** Changed add mars supersession parity plan (b8f2a35).
- Changed add AD-021 through AD-025 for dogfood, decisions, and lean pipeline (bd6293b).
- **tickets:** Changed reconcile 19 ticket-vs-schedule contradictions (C1–C19) (d315dfc).
- **tickets:** Changed move MH-001 through MH-005 to done/ (M1 closeout) (00bbe6f).
- **m0:** Changed audit and fix M0 quality gate gaps (3419f69).
- **tickets:** Changed move MH-006 through MH-008 to done/ (M2 closeout) (431fb77).
- Changed initialize mars-harness repo (M0) (451a632).
- **tickets:** Changed move MH-009 through MH-012 to done/ (M3+M4 closeout) (88c182e).
- **tickets:** Changed move MH-017 through MH-021 to done/ (M6+M7 closeout) (8d48467).
- **tickets:** Changed move MH-013 through MH-016 to done/ (M5a+M5b closeout) (95dbee2).
- **serve:** Changed fix serve tests for new Config requirements, add skills loader tests (56e169a).

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
