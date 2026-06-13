# AD-293: Ephemeral Validation Runtimes

**Status:** Accepted
**Date:** 2026-06-13
**Owner:** Mars Harness maintainers
**Related:** AD-001, AD-011, AD-021, AD-088, AD-139, AD-279, AD-284, AD-285

## Context

Mars Harness is distributed and operated as a single Go binary. That rule keeps
the product plug-and-play: setup, serve, start, run, queue, scoring, trust,
release/update, local inference management, and the default dashboard cannot
require Node, npm, pnpm, Postgres, Redis, Grafana, or another always-on external
service.

At the same time, target validation sometimes needs tools outside the core
binary. Browser visual validation is the current example: Midscene, Playwright,
Puppeteer, browser drivers, or similar tools can inspect rendered UI state that
`curl`, source checks, and unit tests cannot see. The same pattern already
exists in smaller forms: Dogfood may use Podman when available, target projects
may require package managers for their own dependencies, and validation may
start dev servers as managed background processes.

The architectural line must be clear enough for agents:

- persistent infrastructure as a Mars Harness prerequisite violates the single
  binary constraint;
- job-scoped validation tooling can be allowed when it is optional, bounded,
  observable, and cleaned up.

## Decision

Mars Harness allows **ephemeral external runtimes** for optional validation
lanes.

An ephemeral external runtime is a non-core runtime, toolchain, service, or
process started for one validation job and then torn down before the job claims
success. Examples include a Node-based visual testing runner, a browser driver,
a temporary dev server, a container build/run, or a language-specific package
manager used to prepare target-owned dependencies.

This exception does **not** weaken the single-binary tenet:

- the installed Mars Harness control plane remains one Go binary;
- external runtimes are never required for setup, serve/start/run, queue,
  scoring, trust, release/update, local inference management, or the default
  embedded dashboard;
- Mars Harness does not silently install or bundle external runtimes;
- missing optional runtime prerequisites produce actionable skip/blocker output
  rather than degrading core operation.

Ephemeral runtime use is valid only when all of the following are true:

1. **Optional lane:** The validation path is supplemental or target-adopted, not
   a universal release gate for every generated target.
2. **Explicit selection:** The role or tool names why the external runtime is
   needed and which product path it validates.
3. **Managed setup:** Dependency fetch/install runs through `dependency_sync`
   or a first-class tool, not raw package-manager shell commands.
4. **Workspace hygiene:** `workspace_hygiene` runs before setup when repository
   state or ignored generated paths matter.
5. **Tracked processes:** Servers, browsers, containers, and watchers are
   started through managed background/process tracking or an equivalent
   first-class tool.
6. **Contained artifacts:** Caches, downloaded dependencies, screenshots,
   reports, temp configs, and validation binaries are written to ignored paths,
   configured target-owned test/report paths, or `/tmp`.
7. **Cleanup proof:** The role records process/container shutdown, postflight
   hygiene, and `git_status` evidence before claiming pass.
8. **Failure ownership:** Runtime setup, model, tool-policy, cleanup, or leaked
   artifact failures are foundation/runtime evidence by default; target backlog
   is used only for target-owned product defects or deliberate target-owned
   tool adoption.

## Consequences

- Midscene and similar tools can be evaluated without treating Node as a Mars
  Harness prerequisite.
- Dogfood and QA can eventually gain richer visual evidence lanes while the
  baseline generated harness remains lightweight.
- Cleanup is part of correctness. A visual smoke that passes but leaves dirty
  files, occupied ports, orphaned browser processes, or untracked dependency
  trees is a failed validation procedure.
- Tool promotion remains governed by the formalized tool path: recurring or
  risky ephemeral-runtime workflows should become first-class tools with tests,
  role allowlists, generated guidance, and cleanup checks.

## Non-Goals

- Do not add a general permission to run arbitrary package-manager commands.
- Do not make Midscene, Playwright, Node, pnpm, Podman, Docker, or browser
  drivers required infrastructure for all users.
- Do not let optional runtime failures create noisy target product tickets.
- Do not use this exception for production services or long-lived sidecars.
  Optional operator sidecars, such as the planned TanStack control plane, remain
  governed by their own design decisions and explicit prerequisite checks.

## Midscene Application

The Midscene evaluation in
[midscene-dogfood-agent-evaluation.md](../references/midscene-dogfood-agent-evaluation.md)
is the first concrete candidate for this principle. Midscene should enter Mars
Harness, if at all, as an optional visual browser validation lane after a clean
source canary proves model setup, report usefulness, repeatability, and cleanup.

Until that canary passes, deterministic build/start evidence plus a
browser-product smoke or source/runtime assertion remains the required Dogfood
baseline for visual browser targets.
