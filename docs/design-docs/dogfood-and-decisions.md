**Status:** Accepted
**Date:** 2026-04-12
**Author:** mars-harness

# Dogfood Validation and Decision Recording

Covers containerised E2E testing, the decision recording system, git push discipline, and the lean local pipeline profile.

## Key Design Decisions

### AD-021: Podman containers with native fallback

The dogfood tester builds and runs the application in a Podman container to avoid "works on my machine" bugs. Podman was chosen over Docker because it is rootless and daemonless. If Podman is not installed, the agent falls back to native build + start on the host.

A `ContainerfileTemplate()` function auto-generates multi-stage builds based on detected conventions (Node/Next.js, Go, Python, static HTML, generic fallback). The generated file lives at `.harness/Containerfile` so it is versioned and customisable.

Container cleanup (`podman rm -f`) is deferred in the executor after every dogfood run to prevent orphaned containers.

### AD-022: record_decision as a first-class agent tool

Rather than relying on agents to write decisions into `docs/design-docs/` (which requires the LLM to follow prompt guidance), decisions are captured via a dedicated `record_decision` tool. This tool:

- Persists decisions in `.harness/learnings.yaml` alongside conventions and lessons
- Deduplicates by summary
- Injects all past decisions into the REPO LEARNINGS context block for future agents
- Is available to every role via the tool allowlist

This creates a two-level documentation system:
1. **Target repo** (`.harness/learnings.yaml`) — agent-readable, survives across runs
2. **Harness** (SQLite stores) — dashboard-visible, evolution-reviewable

### AD-023: Push after every semantic commit

Agents were committing locally but never pushing, so all work was trapped on the machine. Every role prompt now includes `git push` after each semantic commit. This ensures:

- Work is never lost if the machine crashes or the harness stops
- Other agents (or humans) can see progress immediately
- The remote is always the source of truth

### AD-024: Lean 7-role pipeline for local use

The full 15-role pipeline includes 8 event-driven roles that never fire without GitHub webhooks (PR triggers, CI failure triggers, PR review comment triggers). For local use, only 7 roles are active:

| Role | Purpose |
|------|---------|
| CEO | Strategy and priorities |
| CTO | Architecture review |
| COO | Ticket creation |
| Engineer | Feature delivery |
| QA | Code review |
| Dogfood | Build and runtime validation |
| Janitor | Backlog entropy management |

The dormant roles (security, dependency-manager, release-manager, pipeline-fixer, pr-comment-fixer, cto-pr-merge) have prompt files on disk and are commented out in the manifest. Uncomment to activate when GitHub webhooks and CI are connected.

### AD-025: Dogfood chains from Engineer

The dogfood tester was originally schedule-only (`0 10 * * 1-5`), which never fires when the harness is not running continuously. It now chains from the Engineer (`then: [qa, engineer, dogfood]`) so every completed feature triggers a build validation. The schedule is retained as a safety net for always-on deployments.

## Consequences

- Local pipeline is fast and focused: 7 roles, no dead triggers
- Decisions accumulate across runs, making agents progressively smarter about the repo
- Moving to a PR/CI model later requires only uncommenting manifest entries
- Container-based testing catches environment-specific bugs that native runs miss
