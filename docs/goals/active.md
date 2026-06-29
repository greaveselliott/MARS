# Active Goals

## G-001: Establish BDD-Led Delivery Evidence

- ID: G-001
- Status: active
- Category: operational
- Priority: P0
- Confidence: high
- Source: user_chat
- Dedupe Key: operating-model:bdd-led-walking-skeleton
- Hypothesis: Mapping goals, feature contracts, exec plans, tickets, and evidence into one loop will reduce half-finished work and make completion auditable.
- Success Evidence: Feature tickets reference scenario IDs and move to done only with E2E/integration evidence or explicit descoping.
- Falsification Evidence: Tickets move to done without evidence, feature contracts are not referenced by plans, or in-progress work piles up without completion.
- Competes With: raw throughput, low-documentation experimentation
- Supports: G-002, G-003
- Last Reviewed: 2026-05-02
- Review Trigger: Feature ticket completion, dogfood failure, telemetry triage, quality score change, or user feedback that delivery still feels half-done.
- Owner: CEO

## G-002: Keep Source And Target Harnesses Mirrored

- ID: G-002
- Status: active
- Category: operational
- Priority: P0
- Confidence: high
- Source: user_chat
- Dedupe Key: operating-model:source-target-mirror
- Hypothesis: Applying operating rules to both MARS and initialized targets prevents generated harnesses from falling behind the source doctrine.
- Success Evidence: `mars init` scaffolds goals, feature contracts, AD-074, updated role prompts, knowledge routes, exec-plan metadata, ticket metadata, and quality guidance.
- Falsification Evidence: A generated target lacks AD-074 artifacts, update check misses drift, or source-only behavior appears without an explicit source-only label.
- Competes With: preserving every existing target doc verbatim
- Supports: G-001
- Last Reviewed: 2026-05-02
- Review Trigger: Scanner/init changes, target update changes, role prompt changes, or operating-rule changes.
- Owner: CEO

## G-003: Prefer E2E/Integration Evidence Over Unit-Only Confidence

- ID: G-003
- Status: active
- Category: quality
- Priority: P1
- Confidence: medium
- Source: user_chat
- Dedupe Key: quality:bdd-e2e-integration-default
- Hypothesis: BDD scenarios backed by integration/E2E evidence catch operating-loop failures that unit tests miss.
- Success Evidence: New operating-model features include BDD-style integration or E2E tests, with unit tests limited to deterministic helper behavior.
- Falsification Evidence: Unit tests pass while harness runs still leave stale in-progress tickets, false-done tickets, or unverified scenarios.
- Competes With: fast isolated unit-only delivery
- Supports: G-001
- Last Reviewed: 2026-05-02
- Review Trigger: New feature contracts, completion gates, doctor/update drift checks, or dogfood failures.
- Owner: QA

## G-004: Make CLI Availability Zero-Config Across Shells

- ID: G-004
- Status: active
- Category: distribution
- Priority: P0
- Confidence: high
- Source: user_chat
- Dedupe Key: setup:shell-path-zero-config
- Hypothesis: Automatically configuring the installed binary directory in the user's shell profile removes a first-run failure mode and keeps plug-and-play true for Fish, Zsh, Bash, and POSIX shell users.
- Success Evidence: `make install`, `mars setup`, and `mars update tool` all converge on the same idempotent PATH setup, and tests prove supported shells are configured without duplicate profile entries.
- Falsification Evidence: A supported-shell user installs the binary and still gets `Unknown command: mars` in a new terminal.
- Competes With: leaving PATH setup to Go tooling or manual docs
- Supports: G-002
- Last Reviewed: 2026-05-02
- Review Trigger: Installer, setup, update-tool, or shell support changes.
- Owner: Release Manager

## G-DOCS-IA-001: Rebuild Documentation Site Information Architecture

- ID: G-DOCS-IA-001
- Status: active
- Category: distribution
- Priority: P1
- Confidence: high
- Source: user_chat
- Dedupe Key: docs-site-information-architecture-trust-governance
- Hypothesis: If the docs site is rebuilt around reader intent, security, ownership, guardrails, evidence, and canonical harness docs, evaluators and operators will understand MARS as a governed AI product engineering team rather than a loose agent runner.
- Success Evidence: Homepage first viewport explains MARS as a local AI product engineering team that can be inspected, governed, and improved; public docs route by safe action, proof need, governance evidence, operating recovery, and source-of-truth inspection; public pages identify safe actions, file-writing actions, ownership boundaries, evidence paths, and canonical source-of-truth docs; long catalog content is moved out of the homepage into a dedicated documentation map; `mars docsync audit --repo .`, HTML link sweep, docs consistency tests, and `go test ./...` pass.
- Falsification Evidence: Homepage still acts as a link wall; security, guardrails, and ownership evidence is not visible before command references; public docs and harness-consumed docs describe different truths; new docs duplicate canonical docs without clear source-of-truth labels.
- Competes With: catalog-first homepage expansion, duplicating canonical harness doctrine in public pages
- Supports: G-001, G-002, G-003
- Last Reviewed: 2026-06-29
- Review Trigger: Public docs IA changes, DocSync audit findings, governance feedback, homepage conversion feedback, or changes to canonical harness documentation.
- Owner: COO with Product/Docs Maintainer

## G-FOUNDATION-PLANNING-001: Make Foundation Planning Provider-Neutral

- ID: G-FOUNDATION-PLANNING-001
- Status: active
- Category: operational
- Priority: P0
- Confidence: high
- Source: user_chat
- Dedupe Key: operating-model:provider-neutral-feature-planning
- Hypothesis: If every AI coding provider consumes the same MARS Orchestrator planning model when building the foundation harness, foundation feature delivery will remain auditable, resumable, and consistent across Claude, Codex, Copilot, Cursor, Windsurf, and other clients.
- Success Evidence: `AGENTS.md`, the foundation maintainer role packet, `docs/design-docs/foundation-operating-model.md`, `docs/features/F-016-foundation-provider-planning-doctrine.md`, the active exec plan, and T-054 all require goal -> exec plan -> BDD feature -> tickets -> implementation evidence for non-trivial foundation feature work.
- Falsification Evidence: A provider can plan or build a non-trivial foundation feature from chat-only or provider-native task state; foundation feature tickets exist without an active goal, active exec plan, and BDD feature contract; vendor adapters carry independent doctrine; or deployed target harnesses are told to consume this source-only rule without a separate mirroring decision.
- Competes With: direct chat-to-code delivery, provider-specific planning checklists, branch-only planning, issue-only planning
- Supports: G-001, G-002, G-003
- Last Reviewed: 2026-06-29
- Review Trigger: AI client adapter changes, foundation feature-planning failures, ticket creation without feature contracts, or user feedback that providers are bypassing MARS foundation doctrine.
- Owner: foundation-maintainer with COO and CTO-weekly
