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
- Hypothesis: Applying operating rules to both Mars Harness and initialized targets prevents generated harnesses from falling behind the source doctrine.
- Success Evidence: `mars-harness init` scaffolds goals, feature contracts, AD-074, updated role prompts, knowledge routes, exec-plan metadata, ticket metadata, and quality guidance.
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
- Success Evidence: `make install`, `mars-harness setup`, and `mars-harness update tool` all converge on the same idempotent PATH setup, and tests prove supported shells are configured without duplicate profile entries.
- Falsification Evidence: A supported-shell user installs the binary and still gets `Unknown command: mars-harness` in a new terminal.
- Competes With: leaving PATH setup to Go tooling or manual docs
- Supports: G-002
- Last Reviewed: 2026-05-02
- Review Trigger: Installer, setup, update-tool, or shell support changes.
- Owner: Release Manager
