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

### AD-024: Strict-trunk default pipeline for local use

The default pipeline is local-first and strict trunk. GitHub webhooks can still trigger checks and CI repair, but generated bundles do not depend on review-system events to move work forward.

| Role | Purpose |
|------|---------|
| CEO | Strategy and priorities |
| CTO | Architecture review |
| COO | Ticket creation |
| Engineer | Feature delivery |
| QA | Code review |
| Security | Security audit and review |
| Dependency Manager | Dependency review |
| Release Manager | Changelog and release readiness |
| Dogfood | Build and runtime validation |
| Pipeline Fixer | CI repair when check/webhook failures are available |
| Janitor | Backlog entropy management |

Mutating roles commit and push `main` within their trust level and policy limits. Optional integration events can enqueue pipeline-fixer or other explicitly configured compatibility roles without changing the trunk contract.

### AD-025: Dogfood chains from Engineer

The dogfood tester was originally schedule-only (`0 10 * * 1-5`), which never fires when the harness is not running continuously. It now chains from the Engineer (`then: [qa, engineer, dogfood]`) so every completed feature triggers a build validation. The schedule is retained as a safety net for always-on deployments.

### AD-026: Bootability checks in scanner and role prompts

The scanner's static analysis previously detected structural gaps (missing CI, README, license, tests) but not whether a project could actually build and start. This led to agents producing code that passed tests but was unbootable — missing `dev`/`build` scripts in `package.json`, missing root `layout.tsx` for Next.js App Router, missing Tailwind/PostCSS config files, conflicting `app/` and `pages/` directories.

Three changes close this gap:

1. **Scanner bootability checks** (`checkBootability` in `scanner.go`): framework-specific validation that runs during `mars-harness scan` and generates tickets for structural issues. Checks for:
   - Missing `dev`/`start`/`build` scripts in `package.json`
   - Missing root `layout.tsx` for Next.js App Router projects
   - Conflicting `app/` and `pages/` directories at different levels
   - CSS files using Tailwind directives without `tailwind.config.*` or `postcss.config.*`
   - Deprecated `next.config.js` options (e.g. `experimental.appDir`)

2. **Engineer build verification** (step 5 in engineer prompt): mandatory build + dev server boot check before any ticket can be moved to done. The engineer must run the build command and briefly start the dev server to verify it doesn't crash. Failures must be fixed in the same run.

3. **Dogfood pre-flight checks** (Phase 0 in dogfood prompt): structural validation that runs before attempting to build, catching issues like missing scripts or config files that would cause the build to fail. Pre-flight failures produce high-priority `[Dogfood][Pre-flight]` tickets.

4. **QA structural integrity** (checklist item 3 in QA prompt): bootability review added to the QA checklist so code review also validates that the project structure is sound.

### Discoveries

- The recruiter-workflow-portal ran through the full CEO → CTO → COO → Engineer pipeline and accumulated 14+ tickets of implemented code, but the app could not start because `package.json` had no `dev` script, no root `layout.tsx` existed, Tailwind CSS was referenced in CSS but not installed or configured, and a conflicting root `app/` directory clashed with `src/pages/`. All of these were structural issues that no role caught.
- The Dogfood role was designed to catch exactly these issues but never ran successfully (zero dogfood commits in git history).
- Test suites (jest) can pass while the app is completely unbootable, because unit tests don't exercise the build/start pipeline.

## Consequences

- Local pipeline is fast and focused: strict-trunk roles, no dead default triggers
- Decisions accumulate across runs, making agents progressively smarter about the repo
- Optional CI/webhook integrations add telemetry and repair triggers without replacing direct commits to `main`
- Container-based testing catches environment-specific bugs that native runs miss
- Bootability checks catch structural issues at scan time (before agents even start) and at every stage of the pipeline (engineer, QA, dogfood)

---

### AD-028: Git tools in default manifest and commit gates in role prompts

**Status:** Accepted
**Date:** 2026-04-12
**Author:** Agent (intervention recovery)

### Context

After running the dogfood agent against `recruiter-workflow-portal`, the repository had significant uncommitted changes. Investigation revealed two root causes:

1. **No `.gitignore`** — build artifacts (`.next/`, `node_modules/`) appeared in `git status`, making it look like a sea of changes.
2. **Agents never committed** — the `git_commit`, `git_status`, `git_diff`, and `git_push` tools existed in the registry (`internal/tools/git.go`) but were not included in any role's tool allowlist in the default manifest. Agents could only commit via `shell_exec` with raw git commands, and prompts did not enforce committing strongly enough.

### Changes

1. **Default manifest tool lists** — added `git_status`, `git_diff`, `git_commit`, `git_push` to all write-capable roles (engineer, dogfood, janitor, pipeline-fixer, CEO, COO, CTO, release-manager, security, dependency-manager).

2. **Commit gates in role prompts** — every write-capable role prompt now includes a "COMMIT GATE" section that requires the agent to run `git_status` before finishing and commit any uncommitted changes. The Engineer, Dogfood, Janitor, Pipeline Fixer, and strategy/review roles all enforce this.

3. **Prompt language updated** — all prompts now reference `git_commit` and `git_push` tools instead of raw `git commit`/`git push` shell commands, making it clear these are first-class tools.

4. **Scanner `.gitignore` check** — `no_gitignore` finding (severity: high) is now emitted when a repository lacks a root `.gitignore`, since missing gitignore causes agent confusion from noisy `git status` output.

### Discoveries

- Git tools were fully implemented since M2 but never wired into any role's tool list — a classic "built but not deployed" gap.
- The `shell_exec` fallback for git operations is fragile: agents may format git commands incorrectly, and shell piping/quoting issues can silently fail.
- Without a `.gitignore`, `git status` output for a Node.js project can be thousands of lines of `node_modules/` entries, overwhelming the agent's context window and masking real changes.

---

### AD-029: Per-repo database isolation

**Status:** Accepted
**Date:** 2026-04-12
**Author:** Agent (cross-project contamination fix)

### Context

When running `mars-harness start --repo A` then `mars-harness start --repo B`, both shared the same SQLite database at `~/.mars-harness/db/mars.db`. This caused three contamination vectors:

1. **Stale job pickup** — `Claim()` in `internal/queue/queue.go` grabs the oldest pending job across all repos, not scoped to the current repo. Pending/orphaned jobs from repo A could be claimed by a process started for repo B.
2. **Cron schedules for wrong repos** — `Start()` in `internal/serve/server.go` calls `s.repos.List(ctx)` which loads every repo ever registered, then `registerCronSchedules(repos)` creates scheduled jobs for all of them.
3. **Trigger index pollution** — `s.triggers.Rebuild(repos)` builds webhook trigger mappings for all registered repos, not just the one passed to `--repo`.

The agent loop, LLM client, and learnings store were all correctly isolated per-job. The contamination was entirely at the queue/scheduler/registry level.

### Changes

1. **Per-repo database path** — Default DB path changed from `~/.mars-harness/db/mars.db` to `~/.mars-harness/db/{repo-slug}/mars.db`, where `repo-slug` is `filepath.Base(absPath)`. Applied to `startCmd`, `registerCmd`, and `doctorCmd`. The `serveCmd` keeps the legacy shared path (for multi-repo orchestration) with an informational log.

2. **`defaultDBPath()` / `legacyDBPath()` helpers** — Centralised in `cmd/mars-harness/main.go` to keep derivation DRY and consistent across commands.

3. **`--repo` flag on `doctorCmd`** — Doctor can now target a specific repo's database via `--repo /path/to/repo`, which derives the per-repo DB path automatically. The `--db` flag still works as an explicit override.

4. **`RepoScope` in `serve.Config`** — Defense-in-depth filter. When set (wired from `startCmd`), `Start()` filters `repos.List()` to only the scoped repo before building triggers and scheduling cron. Prevents stale DB entries from activating wrong repos even if someone manually shares a DB path.

5. **Migration warning** — `startCmd` and `registerCmd` log a warning if the legacy shared `mars.db` exists but the per-repo DB does not yet, guiding users to copy history if desired.

### What this does NOT change

- `mars-harness run` — already fully isolated (no DB, no queue)
- `record_decision` — already per-repo (writes to `{repo}/.harness/learnings.yaml`)
- Agent loop / LLM client — already fresh per job
- Learnings / context assembly — already per-repo

### Edge case: multi-repo orchestration

Users managing multiple repos from a single orchestrator use `mars-harness serve` with explicit `--db` and multiple `mars-harness register` calls. The per-repo default only applies to the `start` convenience command.

### Discoveries

- The shared DB was the original design for `serve` (multi-repo orchestrator), but `start` (single-repo convenience command) reused the same default without scoping. This is a common pattern when a convenience wrapper inherits infrastructure assumptions from the full-featured command.
- Queue `Claim()` intentionally lacks repo filtering — it's designed for a multi-repo orchestrator where any worker handles any repo. Per-repo DBs are a simpler isolation boundary than adding repo-aware claim logic.

---

### AD-030: Mechanical ticket deduplication enforcement

**Status:** Accepted
**Date:** 2026-04-12
**Author:** Agent (self-improvement — ticket duplication prevention)

### Context

After running agents against `wave-shooter` and `recruiter-workflow-portal`, both repos had massive ticket duplication: wave-shooter had 30+ tickets for 7 unique topics, recruiter-portal had similar overlap. The COO prompt (STEP 3) already instructed agents to deduplicate, but local models consistently ignored it across runs. Root causes:

1. No ticket state in context — agents started blind every run
2. No write gate — `file_write` always succeeds with no duplicate check
3. Context pruning could discard ticket enumeration results mid-run
4. `file_search` was missing from the COO's tool list
5. Repeat trigger chains re-processed unchanged weekly priorities

### Changes

1. **`ticket_create` tool** (`internal/tools/ticket_create.go`) — purpose-built tool that mechanically prevents duplicates. Before writing, it scans `docs/tickets/{backlog,in-progress,done}/*.md`, reads YAML frontmatter, normalizes titles to keyword sets, and uses subset matching to detect duplicate topics. If a match is found, returns the existing ticket path and skips creation. Auto-assigns the next ticket number (MAX + 1).

2. **Ticket index in context assembly** (`internal/context/assembler.go`) — new `## TICKET INDEX` block (`truncPri: 75`) injected into the system prompt for ticket-aware roles (COO, engineer, janitor, QA, dogfood). Built by `BuildTicketIndex()` in `internal/serve/executor.go`, which globs existing tickets and formats a compact inventory. Agents start every run knowing what already exists.

3. **Historical COO manifest update** — `ticket_create` and `file_search` were added to COO in this slice. AD-105 supersedes the ownership boundary: COO now keeps `file_search` for planning context, while CTO owns `ticket_create` and technical ticket shaping.

4. **Historical COO prompt rewrite** — the original ticketing prompt taught COO to use `ticket_create`. AD-105 supersedes that split: COO writes exec plans and BDD contracts, then hands `ticket_breakdown` to CTO.

5. **Scaffold template cleaned** — `docs/tickets/README.md` template in `init.go` no longer contains "CEO's North star / tier + pillar" boilerplate or "COO creates" lifecycle language. Replaced with project-neutral wording.

### Design decisions

- **Informational duplicate response, not Go error** — matches `record_decision` pattern. The agent sees "DUPLICATE: ticket X exists at path Y" and gracefully skips rather than treating it as a tool failure that could derail the run.
- **Subset matching for titles** — "implement wave progression" matches "implement wave progression system" and vice versa. Uses normalized keyword sets with stop-word removal. Threshold: all words must match for short titles, 80% for titles with 5+ words.
- **`truncPri: 75` for ticket index** — higher than learnings (60), knowledge routes (40), trigger (20), and repo summary (10). Ticket awareness survives budget pressure for roles that need it.

### Discoveries

- Prompt-only enforcement is fundamentally unreliable with local models for invariants that span multiple tool calls. Any rule that says "check X before doing Y" will eventually be ignored, especially when context pruning discards early results.
- The `record_decision` tool already demonstrated the right pattern: dedup at the tool level, not the prompt level. This should have been applied to ticket creation from the start.
- Ticket duplication was not a cross-project contamination issue (as initially suspected) but a single-project, multi-run accumulation bug. Each COO run independently recreated all "This week" items because it couldn't see what previous runs had already created.

---

### AD-033: In-progress tickets are drained before backlog work

**Status:** Accepted
**Date:** 2026-05-02
**Author:** Agent (self-improvement — ticket completion discipline)

### Context

Dogfood and scanner failures can create new backlog tickets while the Engineer has already claimed other work. Previous guidance treated any remaining `docs/tickets/in-progress/` ticket as a failed handoff. That prevented silent abandonment for a single ticket, but it also created a dead-end when several tickets were already in progress: the Engineer could either get stuck failing the gate or move work back to backlog just to satisfy the rule.

### Decision

`docs/tickets/in-progress/` is now the front of the Engineer queue when the ticket is eligible. A ticket is eligible when it has no meaningful `blocker` or `blocked_by` metadata. The Engineer must complete the lowest-numbered eligible in-progress ticket before claiming backlog work. If the ticket is blocked by build failures, missing scripts, dependency drift, test failures, local convention gaps, or guardrails, the run must leave explicit recovery metadata instead of silently abandoning the ticket.

The mechanical gate now allows a handoff when an Engineer run drains one pre-existing in-progress ticket to `done/`, returns it to backlog with `blocker` and `next_action`, or leaves it in progress with `blocked_by` pointing at a dependency or intervention-debt ticket. It still blocks:

- claiming ordinary backlog work while any eligible in-progress ticket existed at run start
- ending with a newly claimed ticket still in progress
- returning in-progress work to backlog without blocker metadata
- making no completion or explicit blocker progress while eligible work existed

`ticket_create` enforces the same policy for Engineer runs: ordinary backlog tickets are blocked while eligible in-progress work remains. Dependency tickets are allowed only when they carry a dedupe key and metadata such as `metadata.blocks` that links back to the blocked ticket. Dogfood ticket creation is capped per run by total count, severity, group, and repeated dedupe key so validation failures cannot flood backlog.

Scanner and Doctor now treat stale eligible in-progress tickets as ticket-drain findings. A scanner stale finding creates deduped intervention debt and enqueues the Janitor (`ticket.stale_in_progress`) when configured. Doctor reports concrete remediation: complete the ticket, return it to backlog with blocker metadata, or link a dependency ticket through `blocked_by`.

### Consequences

- Multiple eligible in-progress tickets become a drainable queue, not a permanent failure state.
- The self-chain keeps running Engineer jobs until the in-progress queue is empty before backlog work resumes.
- Prompt guidance and the injected ticket index both state that eligible in-progress tickets are first priority and blocked tickets must carry recovery metadata.
- Blockers are either treated as implementation work or recorded as explicit dependency/intervention debt, preserving the harness expectation that a ticket ends in committed, pushed, verified trunk changes or a truthful blocked state.

---

### AD-060: Failed recovery jobs do not recursively recover

**Status:** Accepted
**Date:** 2026-05-02
**Author:** Agent (self-improvement — recovery loop containment)

### Context

Dogfood against `sample-target` showed an Engineer self-chain recovery loop: several tickets were already in `docs/tickets/in-progress/`, a ticket gate failure triggered an `auto_recover` Engineer job, that recovery job failed for the same gate reason, and the failure handler enqueued another recovery job with a timestamped idempotency key. The queue accumulated pending/running Engineer recovery jobs without making ticket progress.

### Decision

Self-chaining roles get at most one active auto-recovery job per repo and role. Recovery jobs use a stable idempotency key of `recover:<repo_id>:<role>`, so repeated failures collapse into the active recovery job instead of creating a queue storm.

If a job whose trigger type is already `auto_recover` fails, the server records telemetry/evolution signals but does not enqueue another recovery job. Recursive recovery is treated as a failed intervention attempt, not as a reason to keep scheduling the same role forever.

### Consequences

- A stale ticket gate state can no longer create an unbounded Engineer recovery queue.
- The queue remains inspectable: one recovery attempt captures the failure, and follow-up improvement work comes from telemetry/tickets rather than repeated identical jobs.
- Operators should restart any old running harness process after this change so the process uses the contained recovery behavior.

---

### AD-062: Recovery Queue Storms Are Self-Healed

**Status:** Accepted
**Date:** 2026-05-02
**Author:** Agent (self-improvement — autonomous recovery)

### Context

Containment prevents new recursive recovery jobs, but dogfood showed that an already-stuck target repo can still contain stale `running` recovery jobs and duplicate pending recovery jobs from a previous harness version. Requiring a human to inspect SQLite and cancel those records violates Plug and Play, Self-Improving System, and Execution Truth.

### Decision

The queue now has a recovery self-heal pass. On serve/start startup, warm restart, wake recovery, and a periodic watchdog, active `auto_recover` jobs are inspected by repo and role. The repair keeps at most one fresh active recovery job per repo/role, prefers the canonical `recover:<repo_id>:<role>` idempotency key, fails stale claimed/running recovery jobs, and cancels duplicate pending recovery jobs.

This repair is deliberately narrow. It only targets active recovery jobs identified by `auto_recover` trigger payloads or `recover:` idempotency keys, leaving normal user or role-triggered jobs alone.

### Consequences

- A pre-fix recovery storm can be repaired automatically after restart without direct database surgery.
- The harness can recover from its own queue-control failure mode while preserving audit rows for what was failed or cancelled.
- Future self-heal routines should stay scoped and observable: repair the mechanical stuck state, record why, and avoid broad queue deletion.

---

### AD-067: Source Development Installs The Command Before Operating Targets

**Status:** Accepted
**Date:** 2026-05-02
**Author:** Agent (operator feedback — source checkout workflow)

### Context

During dogfood, the operator was running:

```bash
cd /path/to/target-repo && go build ./cmd/mars-harness;
./mars-harness start --repo /path/to/target-repo
```

That workflow violates the product shape in two ways. First, the semicolon runs the second command even when the build fails, so the operator can accidentally run a stale source-root binary. Second, requiring the operator to sit inside the harness source repo blurs the harness/target boundary. Mars Harness should be an installed command that operates on target repos through `--repo`.

### Decision

Source development now has an explicit `make install` path. It installs `mars-harness` into the Go bin directory using `go install`, after which the operator runs:

```bash
mars-harness start --repo /path/to/target-repo
```

One-off builds should write to `build/mars-harness`, not the repo root. The root-level `./mars-harness` binary path is treated as a stale-binary trap, not the normal operating interface.

### Consequences

- The harness and target project stay visually and operationally separate.
- Failed builds no longer silently fall through to an old binary in the source tree.
- Docs and agent guidance point to the same development loop.

---

### AD-077: Tool Creation Is Scaffolded By A Meta Tool

**Status:** Accepted
**Date:** 2026-05-03
**Author:** User direction and Agent (tool-system self-improvement)

### Context

Mars Harness tools are first-class model capabilities. Adding one requires a
consistent Go file, JSON Schema, argument type, handler, tests, default-registry
registration, trust-policy classification, and role allowlist exposure. Agents
can do that work manually, but repeated manual scaffolding wastes model turns
and invites small inconsistencies.

At the same time, a tool that accepts arbitrary implementation code would be too
powerful. It would collapse design, safety review, and executable behavior into
one opaque model action.

### Decision

Add `tool_create` as a built-in meta tool. It accepts a snake_case tool name,
description, and JSON Schema field list, then scaffolds:

- `internal/tools/<name>.go`
- `internal/tools/<name>_test.go`

The generated handler intentionally returns "handler not implemented yet".
Agents must still implement deterministic behavior, register the tool in
`internal/tools/register_default.go`, update trust policy if it mutates state,
and add meaningful tests before exposing it in role allowlists.

`tool_create` is itself mutating and therefore blocked at observer trust. It is
a mirrored tool: the built-in exists in the foundation harness registry and can
be included in deployed harness role allowlists for trusted implementation and
self-improvement roles.

### Consequences

- Agents can create consistent tool boilerplate quickly.
- The implementation remains reviewable and test-driven instead of hidden
  inside an arbitrary-code tool argument.
- Tool creation has an explicit, repeatable path that can later grow into richer
  inventories, validators, and generated tool references.

### Discovery: bypassing `tool_create` breaks the doctrine it represents

On 2026-05-03, seven related workflow tools were implemented manually in one
shared file instead of being scaffolded iteratively through `tool_create`. The
end state had registry, glossary, generated-harness, tests, release, and docs,
but the creation path bypassed the meta-tool designed to govern tool creation.

The correction is operational rather than feature-heavy: new built-in tools
must originate through `tool_create`, one tool at a time, before manual
implementation or consolidation into shared helpers. Bypassing `tool_create`
requires an explicit `record_decision` entry and design-doc rationale before the
change can be considered complete.

---

### AD-079: Mars Harness CLI Interaction Is A Mirrored Tool

**Status:** Accepted
**Date:** 2026-05-03
**Author:** User direction and Agent (tool-system self-improvement)

### Context

The `mars-harness` CLI is the operational control plane for setup, init,
upgrade, start/serve, scans, doctor checks, release notes, score exports, trust
updates, model evaluation, and self-update flows. Agents could call these
commands through `shell_exec`, but that hides the command surface behind an
arbitrary shell and forces the model to rediscover CLI affordances from memory.

Foundation and deployed harnesses both need this surface. The source repo uses
it to evolve the software factory; target repos use it to operate their deployed
harness and keep it synchronized.

### Decision

Add `mars_harness_cli` as a mirrored built-in tool. It provides:

- a `reference` mode with an exhaustive LLM-facing CLI reference
- a `run` mode that executes `mars-harness` with structured argv
- a `repo` shortcut that safely appends `--repo <workspace path>` for commands
  that support it
- timeout and background support for long-running `serve` and `start`

The tool accepts `MARS_HARNESS_CLI_BIN` for explicit operator/test
configuration, then prefers the current running `mars-harness` executable when
the active process is itself a harness binary, then resolves an installed
`mars-harness` binary from `PATH`, and finally falls back to
`go run ./cmd/mars-harness` only when operating inside the foundation source
checkout. If a resolved binary is stale enough to reject a known command, the
tool output names the binary and tells the operator to set `MARS_HARNESS_CLI_BIN`
or update the installed tool.

`mars_harness_cli` is classified as mutating because many CLI commands can
write files, change trust, start workers, or change release state. Observer
trust therefore blocks it.

### Consequences

- Agents can discover and use the full CLI without relying on generic shell
  recall.
- Deployed harnesses can operate their own harness lifecycle through the same
  mirrored tool language as the foundation harness.
- CLI changes should update the tool reference and tests so the LLM-facing
  command surface remains current.

---

### AD-085: Universal Tools Are Exposed Through MCP

**Status:** Accepted
**Date:** 2026-05-03
**Author:** User direction and Agent (tool-system self-improvement)

### Context

Mirrored tools are not useful enough if external agents or local harness agents
can only reach them through an ad hoc shell convention. MCP-compatible clients
already have a standard tool mechanism: MCP stdio servers. Mars Harness needs
the same built-in registry to be available through that native path for both the
foundation harness and deployed target harnesses, without assuming any specific
model provider. Deployed harnesses use local models by default.

### Decision

Expose the Mars Harness built-in registry through `mars-harness mcp serve`.
The server implements newline-delimited JSON-RPC stdio, supports `initialize`,
`ping`, `tools/list`, and `tools/call`, and delegates execution to the same
tool executor, repo root, trust policy, allowlist, and JSON argument path used
by agent runs.

The default MCP trust level is `observer`, so mutating tools are visible but
blocked until an operator deliberately starts the server with
`--trust contributor` or another explicit trust level.

### Consequences

- MCP-compatible hosts and local harness agents can attach to Mars Harness tools
  as native tools.
- The CLI remains useful for operator smoke tests through `tools list` and
  `tools run`, but MCP is the preferred integration surface for external LLM
  clients.
- The universal tool surface remains model-provider agnostic: tool transport and
  policy must not depend on frontier cloud model access.
- Tool additions must keep the registry, role allowlists, MCP exposure,
  `mars_harness_cli` reference, tools glossary, generated target defaults, and
  tests aligned.

---

### AD-091: Bootstrap Trust Defaults And Destructive Shell Gates

**Status:** Accepted
**Date:** 2026-05-03
**Author:** Agent (target bootstrap diagnosis)

### Context

A target harness run in `../sample-target` produced repeated intervention-debt
tickets for `guardrail_block` instead of useful bootstrap work. Telemetry showed
two foundation failures:

- generated roles listed mutating tools such as `record_decision`,
  `ticket_create`, `file_write`, and `shell_exec`, but queued jobs ignored
  manifest trust defaults and fell back to observer trust unless SQLite already
  had a role entry
- `shell_exec` enforced blast-radius limits after command execution, so
  destructive commands could delete many repo files before the policy reported
  that the diff was too large

### Decision

Generated target manifests seed the default starter roles with
`trust_level: contributor`. The runtime parses manifest trust levels and uses
them only when no persisted SQLite trust entry exists; explicit DB trust
overrides still win.

The shell preflight policy blocks obvious deletion operations before execution,
including `rm`, `rmdir`, `unlink`, `git rm`, and `find -delete`. The post-tool
diff and secret checks remain as defense-in-depth for less obvious mutation.

### Consequences

- Fresh `start` pipelines no longer create observer-trust intervention debt for
  roles that are intentionally write-capable during bootstrap.
- Deployed harnesses that already wrote `trust_level: operator` during early
  self-healing are treated as contributor rather than becoming invalid.
- Blast-radius tickets should no longer be created after obvious repo deletion
  commands have already modified the worktree.

---

### AD-092: Intervention Debt Signals Avoid Secondary Ticket Amplification

**Status:** Accepted
**Date:** 2026-05-03
**Author:** Agent (target telemetry diagnosis)

### Context

The same `../sample-target` bootstrap failure produced additional
intervention-debt tickets after the primary guardrail failures:

- blast-radius policy blocks were classified as `unknown`, creating a
  high-priority "Classify unknown failure" ticket for a known guardrail event
- the Engineer later failed the ticket gate because it could not complete the
  newly claimed intervention-debt ticket after those policy blocks, creating a
  second "Fix ticket completion workflow" ticket for the same job

This made one root failure look like several independent process failures.

### Decision

Telemetry classification treats `tool policy blocked` and
`blast radius exceeded` messages as `guardrail_block`.

When a job already has a policy or guardrail telemetry event, a later terminal
`ticket_gate` failure from the same job is still recorded as telemetry, but it
does not create a second intervention-debt ticket. The primary guardrail ticket
remains the work item to investigate.

### Consequences

- Intervention debt stays closer to root cause instead of multiplying symptoms.
- Ticket-gate telemetry remains visible for scoring and dashboards.
- Genuine ticket-gate failures without prior policy blocks still create
  intervention-debt tickets.

---

### AD-093: Foundation Containment Gate Is Release-Blocking

**Status:** Accepted
**Date:** 2026-05-03
**Author:** Agent (target dogfood containment diagnosis)

### Context

The live `../sample-target` harness run showed a second-order foundation failure.
Once the target worktree had already exceeded blast-radius limits, every
subsequent `shell_exec` call, including read-only commands such as `ls` and
`find`, emitted guardrail telemetry. The repeated guardrail, ticket-gate,
circle-detected, and context-overflow signals produced intervention-debt churn
instead of containing the unsafe target state.

The dogfood matrix had already named the need for a deterministic fake-LLM
loop, but it was not executable in normal CI. Doctrine and backlog were present;
the release-blocking foundation proof was missing.

### Decision

The normal Go test suite includes a fast foundation acceptance gate that drives
a generated target harness through the real executor, OpenAI-compatible client,
router fallback, tool registry, trust policy, telemetry, scoring, ticket gate,
and intervention-debt paths with deterministic fake LLM responses.

Jobs now perform dirty-worktree containment before LLM invocation. If the repo
already exceeds blast-radius limits at job start, the executor fails with one
clear containment signal instead of spending tokens or starting inference.

Fresh bootstrap commits the generated harness scaffold before any role can run.
The same behavior applies to `init` and every command that auto-initializes a
missing harness (`start`, `run`, `register`, and `scan`). The commit stages only
files that appeared during harness initialization, preserving any pre-existing
target work as uncommitted user state. This keeps generated baseline size from
tripping dirty-worktree containment while still making the scaffold auditable in
git.

`shell_exec` remains mutating for trust purposes, and destructive shell
operations remain blocked before execution. Conservative inspection commands
are treated as read-only for post-diff policy so dirty targets can still be
inspected without producing new blast-radius noise.

Deterministic failures such as guardrail blocks, context overflow, missing
models, ticket-gate failures, max turns, and circle detection do not blindly
enqueue same-role recovery. Intervention-debt ticket updates are compacted so
repeated signals do not inflate later prompts.

### Consequences

- Foundation releases are blocked by an executable containment gate rather than
  relying on dogfood doctrine alone.
- Dirty target repos fail closed before model work begins.
- Fresh generated harness baselines are clean before the first autonomous role
  runs.
- Read-only investigation stays possible without amplifying intervention debt.
- Repeated intervention signals remain visible but bounded.

---

### AD-094: File Count Is Not a Default Blast-Radius Gate

**Status:** Accepted
**Date:** 2026-05-04
**Author:** Codex (sample-target intervention-debt review)

### Context

The live `../sample-target` bootstrap produced a medium-priority
intervention-debt ticket after `cto-weekly` hit `MaxFilesPerJob` with 11 changed
files against a limit of 10. The underlying work was a small documentation and
harness-context sweep, not a destructive code rewrite. The file-count cap made
ordinary progress look unsafe and promoted a non-urgent self-improvement ticket
ahead of product work.

The original tenets already emphasized line volume, deletion safety, secret
scanning, trunk discipline, and emergency stop. File count alone proved too
noisy to be a default safety signal.

### Decision

`MaxFilesPerJob` is disabled by default. Repositories that explicitly want a
file-count cap can still configure a positive value, and the safety checker will
enforce it.

Default blast-radius containment relies on stronger signals:

- secret scanning
- per-file line volume
- total line volume
- deletion blocking
- strict trunk push policy

Intervention debt remains durable and visible, but product backlog progress is
the default priority unless an active product ticket explicitly names the debt
as a blocker. AD-109 supersedes the earlier high-priority preemption rule.

### Consequences

- Small multi-file documentation, harness, and planning updates no longer stop
  role progress just because they touch more than 10 files.
- Large rewrites, secret leaks, and deletions remain blocked by more meaningful
  guardrails.
- Intervention debt remains visible without starving product work by default.

---

### AD-095: Ticket Creation Is Tool-Owned

**Status:** Accepted
**Date:** 2026-05-04
**Author:** Codex (sample-target ticket-path review)

### Context

The live `../sample-target` harness created product tickets directly under
`docs/tickets/` while intervention-debt tickets landed correctly under
`docs/tickets/backlog/`. The canonical `ticket_create` tool already wrote new
tickets to the backlog and deduped them, but Dogfood was allowed to create
findings without that tool. Direct `file_write` made it possible to bypass both
backlog placement and dedupe.

### Decision

New ticket markdown is owned by `ticket_create`. `file_write` blocks direct
ticket markdown at `docs/tickets/*.md` and blocks new markdown files inside
ticket lifecycle directories. Existing tickets in `backlog/`, `in-progress/`,
`in-review/`, and `done/` can still be edited in place for blocker metadata,
completion notes, and evidence updates.

Dogfood now has `ticket_create` in the generated manifest and role registry,
and its prompt tells it to call `ticket_create` for findings instead of
hand-writing ticket files. Generated target ticket guidance and tool guidance
state that root-level ticket markdown is invalid.

### Consequences

- New product, dogfood, dependency, and intervention-debt tickets enter the same
  backlog path with numbering and dedupe.
- Agents can still update real lifecycle tickets without losing the ability to
  record blockers or evidence.
- Misplaced root tickets become an obvious policy violation instead of hidden
  backlog state.

---

### AD-096: Target Harnesses Have a Dry-Run Kill Switch

**Status:** Accepted
**Date:** 2026-05-04
**Author:** Codex (sample-target reset review)

### Context

Live target experimentation exposed an escape-hatch gap: once a target repo had
been initialized, operators could stop processes and manually delete Git or
database files, but Mars Harness had no first-class way to remove itself from a
target directory. That made reset workflows error-prone and encouraged ad hoc
deletion of `.harness/`, generated docs, tickets, and `~/.mars-harness/db/*`
files without a preview of the blast radius.

### Decision

`mars-harness eject` is the target-harness kill switch. It is dry-run by
default and requires `--apply --confirm <repo-name>` before deleting anything.
The command removes the deployed harness working-tree surface:

- `.harness/`
- generated `AGENTS.md`, `VERSION`, and `CHANGELOG.md`
- generated planning, goals, feature, ticket, role, reference, report, quality,
  and design-doc directories/files under `docs/`
- the associated per-repo SQLite database and its WAL/SHM sidecars

The command does not rewrite git history. If pointed at the legacy shared
database, it removes the repo registration but keeps the shared database unless
the operator explicitly passes `--delete-shared-db`.

### Consequences

- Target repos can return to a pre-harness working-tree state through one
  auditable command instead of manual deletion.
- Dry-run and repo-name confirmation keep the destructive path deliberate.
- Per-repo database isolation now has a matching cleanup operation.

---

### AD-109: Product Progress Comes Before Intervention Debt

**Status:** Accepted
**Date:** 2026-05-19
**Author:** Codex (demo-123 lifecycle stabilization)

### Context

A fresh `demo-123` Space Invaders dogfood run on `main` exposed the same failure
shape as older `sample-target` runs: harness/runtime problems became target
backlog or orchestration work before the target product made progress.

Observed `demo-123` evidence:

- a retry after a bind failure seeded a second active CEO bootstrap job
- two CEO jobs and two Orchestrator jobs ran before any game implementation
  ticket or code appeared
- Orchestrator drifted into operating-model work rather than product-specific
  Space Invaders planning
- repo-local runtime DB/WAL/log artifacts could trip dirty-worktree and
  blast-radius containment
- downstream roles then failed repeatedly, and loop guard stopped the cycle
  without useful product progress

Older runs showed the companion failure mode: repeated guardrail, ticket-gate,
score, and telemetry signals created intervention-debt tickets that starved the
actual project backlog.

### Decision

Automatic harness/runtime failures are quarantined as local telemetry and
foundation evidence by default. Target backlog intervention-debt tickets are
reserved for target-owned causes such as stale target tickets, human follow-up
on target work, reverted target agent commits, or explicit operator/tool-driven
ticket creation.

Intervention debt no longer preempts ordinary product backlog by default. It
stays visible in quality and status evidence, but Engineer context does not put
high-priority intervention debt ahead of product tickets unless an active
product ticket explicitly names that intervention debt in `blocked_by`.

`mars-harness scores export` is non-mutating with respect to tickets by
default. It refreshes `docs/QUALITY_SCORE.md` and reports improvement targets;
operators must pass `--create-intervention-debt` to materialize score/outcome
signals as tickets.

Fresh generated target plans and feature contracts start from the target
README, active goals, and product brief. The starter F-001 is a product walking
skeleton, not the source harness delivery operating model.

Runtime artifacts are not allowed inside target repos. `start` and `register`
reject repo-local `--db` paths, and `start`/`run` reject repo-local
`--log-file` paths before those files can dirty the worktree.

Bootstrap seeding is idempotent by repo/role/bootstrap key so retrying `start`
cannot duplicate the first CEO job.

Deterministic containment failures stop with one operator-visible blocker
instead of dispatching Orchestrator loops.

### Consequences

- Fresh target runs should reach product planning or product tickets before
  intervention-debt work.
- Harness/runtime instability remains observable without polluting the target
  backlog.
- Operators can still opt into intervention-debt ticket materialization when
  they deliberately want it.
- Repo-local DB/WAL/log mistakes fail early with actionable remediation instead
  of becoming blast-radius noise.
- Source-harness lifecycle fixes must be replayed against a live target
  experience such as `demo-123`, or record the exact blocker and replay steps,
  before the improvement is treated as complete.

### Current Verification State

As of 2026-05-19, deterministic package tests and generated-scaffold checks
passed, and a live replay was run against a clean `<validation-root>`
target with a Space Invaders README brief using a patched binary, external
SQLite DB, and external log file:

```bash
go build -o <validation-root> ./cmd/mars-harness
MARS_HARNESS_WEBHOOK_PORT=19091 MARS_HARNESS_DASHBOARD_PORT=19090 \
  <validation-root> start \
  --repo <validation-root> \
  --db <validation-root> \
  --log-file <validation-root> \
  --concurrency 1
```

The replay improved the original failure class but did not fully prove the
desired first lifecycle. Positive evidence:

- bootstrap seeding produced exactly one active CEO bootstrap job with the
  stable `seed:<repo_id>:ceo:bootstrap` idempotency key
- CEO, Orchestrator, and COO stayed product-specific and named the Space
  Invaders walking skeleton instead of source-harness doctrine
- generated goal, active plan, and `F-001-product-walking-skeleton.md` were
  derived from the README brief
- repo-local runtime DB/WAL/log artifacts were not created in the target repo
- guardrail and inference failures were recorded in telemetry and kept out of
  `docs/tickets/`
- no target intervention-debt tickets were created; `docs/tickets/` still only
  contained its README

Residual blockers found by the replay:

- the hourly Engineer schedule fired while COO was still shaping the product
  ticket, so Engineer claimed work before the queued Orchestrator handoff could
  run
- Engineer had product context but no ordinary product ticket, attempted to
  write `docs/tickets/backlog/T-001-...` directly, and was correctly blocked by
  the `ticket_create` policy gate
- Engineer then recorded `no_work/ticket_shaping` repeatedly instead of exiting
  promptly, showing a remaining no-work termination problem
- the run reached product-specific plan and feature-contract state, but did not
  create an ordinary product ticket or Space Invaders implementation before the
  operator stopped the process

Verdict: the stabilization meaningfully reduces intervention-debt starvation,
runtime artifact noise, and generic harness-doctrine drift, but the first-run
lifecycle is still incomplete. The next fix should prevent scheduled Engineer
jobs from preempting bootstrap handoff work and should make no-ticket Engineer
runs terminate after one clear disposition.

### Follow-up `demo-123` Replay: Product-First, Then CTO Fan-Out

After queue priority and terminal-disposition fixes, a second live replay used a
clean `<validation-root>` target and external runtime state:

```bash
MARS_HARNESS_WEBHOOK_PORT=19191 MARS_HARNESS_DASHBOARD_PORT=19190 \
  <validation-root> start \
  --repo <validation-root> \
  --db <validation-root> \
  --log-file <validation-root> \
  --concurrency 1
```

Positive evidence:

- exactly one CEO bootstrap job was seeded
- CEO and Orchestrator completed through accepted terminal dispositions
- COO produced Space Invaders-specific planning and
  `docs/features/F-001-product-walking-skeleton.md`, then committed the target
  change as `b83c767 plan: update active scenario schedule and feature contract
  for Space Invaders`
- no repo-local runtime DB/WAL/log artifacts appeared in the target repo
- no target intervention-debt tickets were created
- Orchestrator routed the next need to technical planning rather than
  implementation without a product ticket

Residual blocker:

- `cto-weekly` opened with governance/audit context before product ticketing
  and then created several independent backlog tickets for the same
  `F-001-S001` scenario:
  `T-001-create-basic-html-structure-with-game-container.md`,
  `T-002-implement-player-ship-movement-controls.md`,
  `T-003-implement-basic-projectile-firing-mechanics.md`,
  `T-004-implement-alien-targets-with-grid-formation.md`,
  `T-005-implement-basic-score-and-lives-display-system.md`, and
  `T-006-implement-basic-game-loop-and-state-management.md`
- the run still had no CTO commit or implementation handoff after more than
  six minutes, and operator shutdown recorded only foundation-local telemetry
  for the cancellation

Decision: first-run technical planning is now explicitly bounded. Generated CTO
guidance creates or confirms one current-scenario implementation ticket before
handoff, the generated CTO allowlist removes broad audit and shell/Mars CLI
tools from the default first-run surface, and `ticket_create` dedupes
independent feature-ticket fan-out for the same BDD scenario unless `depends_on`
makes the decomposition explicit.

### Follow-up `demo-123` Replay: Bounded CTO Reaches Engineer

After narrowing CTO and adding BDD-scenario ticket dedupe, a third clean
`demo-123` replay used an external runtime directory:

```bash
MARS_HARNESS_WEBHOOK_PORT=19291 MARS_HARNESS_DASHBOARD_PORT=19290 \
  <validation-root> start \
  --repo <validation-root> \
  --db <validation-root> \
  --log-file <validation-root> \
  --concurrency 1
```

Positive evidence:

- the retry after an initial local bind failure did not create a duplicate CEO
  bootstrap job; the DB contained one completed CEO job
- no intervention rows were recorded and no intervention-debt ticket appeared
  in the target backlog
- COO produced Space Invaders planning and a slugged
  `docs/features/F-001-product-walking-skeleton.md`
- CTO ran with the narrowed 13-tool surface, did not call broad audit,
  docsync, shell, or Mars CLI tools, and created exactly one ordinary product
  ticket for `F-001-S001`
- Orchestrator routed to Engineer, so the run reached implementation rather
  than stalling in planning or intervention debt

Residual findings:

- COO and CTO recorded successful dispositions while their target changes were
  still uncommitted
- CTO also created duplicate `docs/features/F-001.md` even though the slugged
  F-001 contract already existed
- Orchestrator still spent 18 tool calls inspecting the expected dirty
  planning/ticket state before routing Engineer
- the operator stopped the run after Engineer began; the DB retained the
  Engineer job as running, which should be treated as shutdown recovery
  evidence rather than product failure

Decision: the next stabilization makes clean handoff mechanical. `file_write`
now blocks creation of duplicate `docs/features/F-NNN*.md` contracts that share
the same feature ID, and successful non-Orchestrator `job_disposition_record`
calls are rejected until repo-visible changes are committed. This should turn
the next replay's COO/CTO boundary into committed plan/ticket evidence before
Engineer starts.

### Follow-up `demo-123` Replay: Clean Handoff Exposes Canonicalization

After adding successful-disposition commit gates, a fourth clean `demo-123`
replay used another external runtime directory:

```bash
MARS_HARNESS_WEBHOOK_PORT=19391 MARS_HARNESS_DASHBOARD_PORT=19390 \
  <validation-root> start \
  --repo <validation-root> \
  --db <validation-root> \
  --log-file <validation-root> \
  --concurrency 1
```

Positive evidence:

- CEO attempted to finish while dirty, was blocked by `job_disposition_record`,
  then ran `git_status`, committed
  `8435b7a feat: Record decision for Demo 123 initial goal`, and recorded a
  successful disposition.
- COO produced Space Invaders planning and a product feature contract, committed
  `2712c3d plan: update active scenario schedule and feature contract for Demo
  123 Space Invaders game`, and left the target tree clean.
- No intervention-debt tickets were created, and runtime DB/log artifacts
  stayed outside the target repo.

Residual findings:

- COO created a second slugged `F-001` contract
  (`docs/features/F-001-S001-project-brief-becomes-visible-product-slice.md`)
  even though `docs/features/F-001-product-walking-skeleton.md` already
  existed.
- Orchestrator then recorded `suggested_role: cto`; because the executable
  generated manifest role is `cto-weekly`, dispatch queued another Orchestrator
  job instead of running CTO.
- The run reached committed product planning but still did not create the
  ordinary product ticket before operator shutdown.

Decision: canonicalization is now enforced mechanically. `file_write` blocks
any new feature contract path when another `docs/features/F-NNN*.md` file with
the same feature ID exists, and dispatch routing normalizes small canonical role
aliases (`cto`, `architecture`, `release`, `dependency`) to generated manifest
role keys when present.

### Follow-up `demo-123` Replay: Clean Handoff Exposes Orchestrator Recovery

After feature-contract canonicalization and role alias routing, a fifth clean
`demo-123` replay used another external runtime directory:

```bash
MARS_HARNESS_WEBHOOK_PORT=19491 MARS_HARNESS_DASHBOARD_PORT=19490 \
  <validation-root> start \
  --repo <validation-root> \
  --db <validation-root> \
  --log-file <validation-root> \
  --concurrency 1
```

Positive evidence:

- retrying `start` after a bind failure did not leave a second active CEO job
- CEO and COO both hit the successful-disposition clean-worktree gate, ran
  `git_status`, committed their product planning work, and only then recorded
  terminal dispositions
- the target retained a single canonical
  `docs/features/F-001-product-walking-skeleton.md` file; no duplicate F-001
  feature contract was created
- the target tree was clean after COO, runtime DB/log artifacts stayed outside
  the repo, and no intervention-debt rows or target intervention-debt tickets
  were created

Residual finding:

- the Orchestrator job after COO reached max turns before recording a
  disposition; failure recovery then treated the failed Orchestrator as the
  source disposition and enqueued another Orchestrator, even though the original
  COO handoff already said `next_need: ticket_breakdown`

Decision: failed Orchestrator recovery is now non-recursive. If the failed
Orchestrator trigger carries a non-Orchestrator source disposition with a
deterministic routing signal, dispatch falls forward from that source handoff
to the target role, such as `cto-weekly` for `ticket_breakdown`. If the source
handoff is absent or would route Orchestrator again, dispatch records a stopped
decision and creates no recursive Orchestrator job.

### Follow-up `demo-123` Replay: Product Implementation Exposes Ticket Move Guard

After non-recursive Orchestrator recovery, the next clean replay used:

```bash
MARS_HARNESS_WEBHOOK_PORT=19591 MARS_HARNESS_DASHBOARD_PORT=19590 \
  <validation-root> start \
  --repo <validation-root> \
  --db <validation-root> \
  --log-file <validation-root> \
  --concurrency 1
```

Positive evidence:

- the lifecycle reached product implementation: CEO, Orchestrator, COO, CTO,
  Orchestrator, and Engineer all ran in order without intervention-debt tickets
- CTO created exactly one ordinary product ticket,
  `docs/tickets/backlog/T-001-implement-first-visible-product-slice-with-player-ship-alien.md`,
  and committed it as `ef91601`
- Engineer created real browser game files under `src/`, including
  `index.html`, `style.css`, and `game.js`
- repeated guardrail blocks stayed foundation-owned telemetry; the target
  backlog still had zero intervention-debt tickets

Residual finding:

- Engineer moved T-001 from backlog to done, which produced one deleted ticket
  path and one new done ticket path; blast-radius deletion policy treated the
  legitimate lifecycle move as a forbidden deletion and trapped the job in
  repeated guardrail blocks before it could commit or hand off to QA

Decision: ticket lifecycle moves are now a bounded deletion exception. The
blast-radius diff policy ignores the deletion side only when the same ticket ID
appears as a new ticket markdown file in another lifecycle directory in the same
diff. Arbitrary deletions and unpaired ticket deletions remain blocked.

### Follow-up `demo-123` Replay: No-Ticket Engineer Dispatch

After the ticket lifecycle move patch, another clean replay used:

```bash
MARS_HARNESS_WEBHOOK_PORT=19691 MARS_HARNESS_DASHBOARD_PORT=19690 \
  <validation-root> start \
  --repo <validation-root> \
  --db <validation-root> \
  --log-file <validation-root> \
  --concurrency 1
```

Positive evidence:

- CEO committed a product-specific first-slice decision and no
  intervention-debt tickets appeared
- Orchestrator and CTO stayed on product delivery surfaces rather than
  foundation intervention debt
- the earlier ticket lifecycle deletion problem is covered by deterministic
  tests before this replay

Residual finding:

- this run skipped COO, CTO completed without creating an ordinary product
  ticket, and Orchestrator still routed Engineer; Engineer then began writing
  `index.html` without any backlog or in-progress product ticket

Decision: Engineer dispatch is now ticket-backed. If dispatch selects Engineer
while no ordinary product ticket exists in `docs/tickets/backlog/` or
`docs/tickets/in-progress/`, the runtime rewrites the handoff to `cto-weekly`
for ticket shaping. Intervention-debt tickets do not satisfy this prerequisite.

### Follow-up `demo-123` Replay: Completed Ticket Needs QA, Not New Ticket Shaping

The next clean replay used:

```bash
MARS_HARNESS_WEBHOOK_PORT=19791 MARS_HARNESS_DASHBOARD_PORT=19790 \
  <validation-root> start \
  --repo <validation-root> \
  --db <validation-root> \
  --log-file <validation-root> \
  --concurrency 1
```

Positive evidence:

- CTO created one ordinary product ticket and committed it
- Engineer claimed T-001, committed product files, moved T-001 to
  `docs/tickets/done/`, and the ticket lifecycle deletion exception allowed the
  commit
- no intervention-debt tickets were created

Residual finding:

- the Orchestrator after Engineer hit max turns; deterministic fallback
  preserved the source handoff, but the Engineer disposition still said
  `next_need: implementation`, so the no-open-ticket prerequisite routed back
  to CTO instead of QA

Decision: completed Engineer source dispositions override stale implementation
wording. If Engineer completed the product ticket and no open product ticket
remains, dispatch routes QA review rather than returning to Engineer or CTO
ticket shaping.

### Follow-up `demo-123` Replay: Ticket-Gate Repair Before Orchestration

The next clean replay used:

```bash
MARS_HARNESS_WEBHOOK_PORT=19891 MARS_HARNESS_DASHBOARD_PORT=19890 \
  <validation-root> start \
  --repo <validation-root> \
  --db <validation-root> \
  --log-file <validation-root> \
  --concurrency 1
```

Positive evidence:

- CEO, COO, CTO, and Engineer progressed on the Space Invaders product lane
- CTO created an ordinary product ticket and Engineer implemented visible game
  files before attempting ticket completion
- no intervention-debt tickets were created

Residual finding:

- Engineer moved the ticket to done with empty `evidence_links`, so the ticket
  gate rejected completion with a missing BDD evidence error; dispatch then
  routed the failed Engineer job through Orchestrator, which hit max turns
  instead of repairing the narrow evidence defect

Decision: ticket-gate failures after Engineer progress now receive one bounded
Engineer `ticket_gate_repair` job. The repair trigger carries the gate error and
is intended to fix ticket evidence, lifecycle placement, or handoff metadata.
If that repair fails the gate again, the runtime records telemetry and stops
without recursive Orchestrator dispatch or another repair job.

### Follow-up `demo-123` Replay: QA Protocol And Stale Survey Cleanup

The next clean replay used:

```bash
MARS_HARNESS_WEBHOOK_PORT=19991 MARS_HARNESS_DASHBOARD_PORT=19990 \
  <validation-root> start \
  --repo <validation-root> \
  --db <validation-root> \
  --log-file <validation-root> \
  --concurrency 1
```

Positive evidence:

- CEO, COO, CTO, Engineer, and the bounded Engineer repair all stayed on the
  Space Invaders product lane
- the repair job was enqueued with trigger type `ticket_gate_repair`, moved
  T-001 to `docs/tickets/done/`, filled `evidence_links`, and handed to
  Orchestrator
- Orchestrator routed the completed Engineer ticket to QA, not back to CTO or
  Engineer
- no target intervention-debt tickets were created

Residual findings:

- while T-001 was briefly in progress, the orchestrator survey enqueued a
  separate pending Engineer ticket-owner job; after the repair moved T-001 to
  done, that pending job became stale
- generated QA guidance still described writing a review file even though the
  default QA role is read-only; QA replied in prose, skipped
  `job_disposition_record`, and failed the dispatch protocol

Decision: dispatch protocol failures are now contained as foundation telemetry
and do not route through Orchestrator. Generated QA guidance now treats
disposition output as the durable read-only review handoff. Successful Engineer
completion also cancels stale pending ticket-owner survey jobs whose referenced
tickets are no longer eligible in-progress work.

### Follow-up `demo-123` Replay: Repair Prompt Is Too Broad

The next replay showed the bounded repair path again, but exposed a narrower
performance issue: the repair job received a missing evidence failure after the
ticket had already been moved to `docs/tickets/done/`, then spent several
minutes running as a broad Engineer job without editing the clean working tree.

Decision: `ticket_gate_repair` now carries an explicit
`ticket_lifecycle_and_evidence_only` repair scope, and generated Engineer
guidance has a fast path for that trigger. The repair should update the ticket
evidence/lifecycle metadata, commit, and hand off to QA instead of restarting
implementation reasoning unless the gate reason says product code is invalid.

### Follow-up `demo-123` Replay: Security Approval Routed Backward

The next replay showed the product-first path working through CEO, COO, CTO,
Engineer, bounded ticket-gate repair, QA, and Security without creating target
intervention-debt tickets. The remaining lifecycle defect appeared after
Security approved T-001: Orchestrator suggested QA again for the same ticket,
and the duplicated read-only QA job failed the dispatch protocol.

Decision: completed or approved review handoffs now move forward through the
review chain. When Orchestrator suggests an earlier reviewer after QA,
Security, Dependency Manager, or Release Manager has approved/completed its
stage, deterministic dispatch rewrites to the next forward review owner in the
manifest or stops if none remains.

### Follow-up `demo-123` Replay: Runtime Learnings Blocked Repair Handoff

The next clean replay used:

```bash
MARS_HARNESS_WEBHOOK_PORT=20191 MARS_HARNESS_DASHBOARD_PORT=20190 \
  <validation-root> start \
  --repo <validation-root> \
  --db <validation-root> \
  --log-file <validation-root> \
  --concurrency 1
```

Positive evidence:

- exactly one CEO bootstrap job was seeded
- CEO and CTO stayed product-first, and CTO created one ordinary product ticket
  for the Space Invaders game
- Engineer produced real product files under `src/`
- the ticket-gate failure created one bounded `ticket_gate_repair` Engineer job
  and no target intervention-debt tickets

Residual finding:

- the repair job was blocked while recording a successful disposition because
  runtime convention learning left `.harness/learnings.yaml` dirty; the target
  had no remaining product dirty path, but the clean-handoff policy still
  treated the learning metadata as uncommitted product work

Decision: runtime-managed `.harness/learnings.yaml` metadata no longer blocks a
successful `job_disposition_record` by itself. Product, ticket, documentation,
and source dirty paths still block clean handoff and must be committed before a
successful disposition.

### Follow-up `demo-123` Replay: COO Implemented Before Ticketing

After the runtime-learning handoff patch, another clean replay used:

```bash
MARS_HARNESS_WEBHOOK_PORT=20291 MARS_HARNESS_DASHBOARD_PORT=20290 \
  <validation-root> start \
  --repo <validation-root> \
  --db <validation-root> \
  --log-file <validation-root> \
  --concurrency 1
```

Positive evidence:

- exactly one CEO bootstrap job was seeded
- runtime `.harness/learnings.yaml` no longer blocked CEO's successful handoff
  after the actual goal-doc changes were committed
- no target intervention-debt tickets were created
- the lifecycle advanced from CEO to Orchestrator to COO

Residual finding:

- COO wrote and committed root `index.html` as
  `feat: add walking skeleton HTML for Space Invaders game demo` before CTO
  created any ordinary product ticket; COO also committed
  `.harness/learnings.yaml`, proving product code could still bypass the
  intended COO -> CTO -> Engineer boundary

Decision: COO is now planning-only at the tool-policy layer. COO `file_write`
may update exec plans, feature contracts, backlog plans, and goal observations,
but implementation paths are blocked. Mutating `shell_exec` is blocked for COO,
and generated COO manifests no longer expose `shell_exec` by default.

### Follow-up `demo-123` Replay: Engineer Done Still Looped To CTO

After the COO planning-only boundary, another clean replay used:

```bash
MARS_HARNESS_WEBHOOK_PORT=20391 MARS_HARNESS_DASHBOARD_PORT=20390 \
  <validation-root> start \
  --repo <validation-root> \
  --db <validation-root> \
  --log-file <validation-root> \
  --concurrency 1
```

Positive evidence:

- CEO, COO, CTO, and Engineer ran in order on the Space Invaders product lane
- COO stayed planning-only and created no product source files before CTO
  ticketing
- CTO created an ordinary product ticket, and Engineer created real browser
  game files under `src/`, added `package.json`, and moved T-001 to `done/`
- no target intervention-debt tickets were created

Residual finding:

- after Engineer completed T-001 and no open ordinary product ticket remained,
  Orchestrator selected `cto-weekly` twice instead of routing QA review; the
  prior no-ticket gate only rewrote direct Engineer redispatches, not CTO
  planning handoffs selected by Orchestrator

Decision: a completed Engineer source disposition with no remaining open
ordinary product ticket is a QA-review boundary. The runtime now rewrites any
Orchestrator-selected pre-review follow-up, including CTO ticket shaping, to QA
before further planning or implementation.

### Follow-up `demo-123` Replay: QA Rework Should Not Create A New Ticket

After the QA-before-planning boundary, another clean replay used:

```bash
MARS_HARNESS_WEBHOOK_PORT=20491 MARS_HARNESS_DASHBOARD_PORT=20490 \
  <validation-root> start \
  --repo <validation-root> \
  --db <validation-root> \
  --log-file <validation-root> \
  --concurrency 1
```

Positive evidence:

- CEO, COO, CTO, Engineer, the bounded ticket-gate repair, and QA all stayed on
  the Space Invaders product lane
- CTO created one ordinary product ticket, Engineer committed browser game
  files, and the ticket-gate repair filled missing ticket evidence before QA
- the Orchestrator decision after completed Engineer work routed to QA with the
  QA-boundary reason instead of returning to CTO
- no target intervention-debt tickets were created

Residual finding:

- QA requested implementation rework on T-001, but T-001 was already in
  `docs/tickets/done/`; the no-open-ticket prerequisite treated that as a fresh
  no-ticket implementation attempt and rewrote the handoff back to CTO, where a
  duplicate ticket-creation path began

Decision: review rework now reuses the existing product ticket. If a reviewer
records `changes_requested` for an ordinary product ticket and the next need is
implementation rework or an Engineer fix, dispatch allows Engineer to repair
that same ticket even when it currently lives in `done/` or `in-review/`.
Fresh implementation without a backlog or in-progress product ticket still
routes to CTO ticket shaping.

### Follow-up `demo-123` Replay: Ticket Move Deletions Need Tracked Counterparts

After the review-rework routing patch, another clean replay used:

```bash
MARS_HARNESS_WEBHOOK_PORT=20591 MARS_HARNESS_DASHBOARD_PORT=20590 \
  <validation-root> start \
  --repo <validation-root> \
  --db <validation-root> \
  --log-file <validation-root> \
  --concurrency 1
```

Positive evidence:

- CEO, COO, CTO, and Engineer all stayed on the Space Invaders product lane
- CTO created one ordinary product ticket, and Engineer committed visible
  browser game files before ticket lifecycle cleanup
- the first ticket-gate failure was quarantined as foundation telemetry and
  produced one bounded `ticket_gate_repair` job, not intervention-debt backlog
  churn
- no target intervention-debt tickets were created

Residual finding:

- Engineer and the bounded repair job attempted ordinary ticket lifecycle
  cleanup, but the blast-radius deletion exception only recognized an untracked
  destination file; a normal staged `git mv`, or cleanup after a tracked
  duplicate ticket already existed in another lifecycle directory, still looked
  like a forbidden deletion and trapped the repair loop

Decision: the ticket lifecycle deletion exception now recognizes staged
`git mv` destinations and already-present lifecycle counterparts for the same
ticket ID, in addition to untracked destination files. Arbitrary deletions and
unpaired ticket deletions remain blocked.

### Follow-up `demo-123` Replay: QA Needs Runtime Terminal-Tool Recovery

After ticket lifecycle move handling was widened, another clean replay used:

```bash
MARS_HARNESS_WEBHOOK_PORT=20691 MARS_HARNESS_DASHBOARD_PORT=20690 \
  <validation-root> start \
  --repo <validation-root> \
  --db <validation-root> \
  --log-file <validation-root> \
  --concurrency 1
```

Positive evidence:

- bind-failure retry kept a single CEO bootstrap job with idempotency key
  `seed:<repoID>:ceo:bootstrap`
- CEO, COO, CTO, Engineer, bounded ticket-gate repair, Orchestrator, and QA
  stayed on the Space Invaders product lane
- CTO created one ordinary product ticket, Engineer committed visible browser
  game files, and the bounded ticket-gate repair moved T-001 to done with
  scenario evidence metadata
- ticket-gate and guardrail signals remained foundation telemetry; no target
  intervention-debt tickets were created

Residual finding:

- QA reached the review stage but replied in prose without calling
  `job_disposition_record`; the generated QA prompt already required the tool,
  so this is a runtime tool-discipline failure rather than just missing role
  documentation

Decision: dispatch-mode jobs now pass `job_disposition_record` as a required
terminal tool to the agent loop. Prose-only completion gets an in-band corrective
turn requiring the tool call. If the role still finishes without the disposition
after that correction, the existing dispatch-protocol failure path records
foundation telemetry and stops without Orchestrator recovery or target backlog
debt.

### Follow-up `demo-123` Replay: QA Must Inspect Repo Files Before Liveness Blocking

After terminal-tool recovery was added, another clean replay used:

```bash
MARS_HARNESS_WEBHOOK_PORT=20791 MARS_HARNESS_DASHBOARD_PORT=20790 \
  <validation-root> start \
  --repo <validation-root> \
  --db <validation-root> \
  --log-file <validation-root> \
  --concurrency 1
```

Positive evidence:

- CEO, Orchestrator, CTO, QA, Engineer, bounded ticket-gate repair, QA rework,
  and Engineer rework all stayed on the Space Invaders product lane
- QA called `job_disposition_record`; the prior prose-only protocol failure did
  not recur
- Orchestrator corrected the premature pre-implementation QA block to Engineer,
  and later routed QA `changes_requested` feedback back to Engineer on the same
  T-001 ticket
- no target intervention-debt tickets were created

Residual finding:

- after Engineer rework, QA blocked with `next_need: liveness` because source
  code was not in the trigger context, even though the target repo contained the
  completed ticket, recent commits, `index.html`, `script.js`, `style.css`, and
  validation scripts; Orchestrator then routed that QA inspection miss back to
  CTO planning

Decision: QA generated guidance now requires reading the ticket, recent commits,
and named implementation files before using a missing-context or liveness block.
Dispatch also rewrites QA trigger-context blocks away from CTO/COO/CEO/Janitor
and back to QA for one repo-inspection retry, so missing trigger prose does not
become planning churn.

### Follow-up `demo-123` Replay: Inline Tool Calls Need Parsing

After QA repo-inspection hardening, another clean replay used:

```bash
MARS_HARNESS_WEBHOOK_PORT=20891 MARS_HARNESS_DASHBOARD_PORT=20890 \
  <validation-root> start \
  --repo <validation-root> \
  --db <validation-root> \
  --log-file <validation-root> \
  --concurrency 1
```

Positive evidence:

- CEO, COO, CTO, Engineer, QA, Engineer rework, Orchestrator, and second QA all
  stayed on the Space Invaders product lane
- QA no longer blocked because source was missing from trigger context; it
  inspected the repo and requested a legitimate product fix for missing
  life-decrement and game-over behavior
- Engineer rework committed `feat(game): implement complete life management and
  damage logic for Space Invaders (T-001)`
- guardrail and dispatch-protocol signals remained foundation telemetry; no
  target intervention-debt tickets were created

Residual finding:

- the second QA job attempted repo inspection and terminal disposition using
  inline `<tool_call>name{...}</tool_call>` text; because the parser did not
  recognize that syntax, both attempts were treated as prose and the job failed
  the dispatch protocol

Decision: the agent parser now recognizes inline `<tool_call>` tags with
unquoted-key arguments, `<|"|>` sentinel strings, arrays, and nested objects.
That keeps tool syntax compatibility in the runtime and allows fast local
models to execute intended tool calls instead of generating false prose-only
protocol failures.

### Follow-up `demo-123` Replay: Product Approval Drifted Into Governance

After inline tool-call parsing, the next clean replay used:

```bash
MARS_HARNESS_WEBHOOK_PORT=20991 MARS_HARNESS_DASHBOARD_PORT=20990 \
  <validation-root> start \
  --repo <validation-root> \
  --db <validation-root> \
  --log-file <validation-root> \
  --concurrency 1
```

Positive evidence:

- one CEO bootstrap job was seeded, then Orchestrator routed the product lane
- CTO created T-001, Engineer committed a playable Space Invaders walking
  skeleton, and Engineer moved T-001 to done with BDD evidence
- QA approved T-001, Security completed with no critical or high findings, and
  no target intervention-debt tickets were created
- the previous inline `<tool_call>` protocol failure did not recur

Residual finding:

- after Security, automatic review progression routed into Dependency Manager,
  which hit `max_turns`; later Orchestrator/CTO repeated ticket-shaping attempts
  without ticket-state change until the loop guard stopped them
- `.harness/learnings.yaml` remained modified in the target repo after runtime
  convention detection

Decision: automatic post-approval review progression now routes QA to Security
and Security to Dogfood when available, then stops unless a role explicitly asks
for dependency or release work. Fresh target product validation should prove the
slice is runnable before broader governance resumes. Runtime-only
`.harness/learnings.yaml` changes are also auto-committed when they are the sole
dirty target path, keeping the dogfood target clean without hiding product
changes.

### Follow-up `demo-123` Replay: Done Evidence Must Fail Before Move

After QA tool-tier hardening, another clean replay used:

```bash
<validation-root> start \
  --repo <validation-root> \
  --db <validation-root> \
  --log-file <validation-root>
```

Positive evidence:

- one CEO bootstrap job was seeded; CEO, COO, CTO, and Engineer stayed on the
  Space Invaders product lane
- CTO created ordinary product ticket T-001 for player-ship movement
- Engineer produced real browser-game artifacts (`index.html`, `style.css`,
  `script.js`, test files, README updates), committed them, and moved T-001 to
  `docs/tickets/done/`
- guardrail and ticket-gate signals stayed foundation telemetry; no target
  intervention-debt tickets were created

Residual finding:

- Engineer moved T-001 to `done/` while `evidence_links` and `verified_by`
  were still empty, so the post-run ticket gate failed the otherwise productive
  job and enqueued a bounded `ticket_gate_repair` Engineer job
- the repair path was bounded, but it still spent another model run on a
  metadata issue the original Engineer should fix before the lifecycle move

Decision: feature-ticket done evidence now fails at tool preflight. `shell_exec`
blocks `git mv`/`mv` into `docs/tickets/done/` while required evidence fields
are empty, and `file_write` blocks saving a done feature ticket with missing
evidence. The live replay loop therefore turns this class from late repair work
into an immediate, actionable same-run correction.

### Follow-up `demo-123` Replay: Runtime Loops Must Stop After Product Progress

After done-evidence preflight, another clean replay used:

```bash
<validation-root> start \
  --repo <validation-root> \
  --db <validation-root> \
  --log-file <validation-root>
```

Positive evidence:

- CEO, COO, CTO, and Engineer stayed on the Space Invaders product lane with no
  target intervention-debt tickets.
- Engineer committed browser-game implementation, package scripts, ticket
  evidence, and workspace hygiene changes.
- Guardrail blocks for duplicate feature-contract paths and missing done-ticket
  evidence remained foundation telemetry.

Residual finding:

- COO appended duplicate `F-001-S001` through `F-001-S003` headings to the
  generated feature contract instead of replacing the starter scenarios.
- CTO's ticket body duplicated the canonical ticket heading.
- Engineer copied T-001 into `docs/tickets/done/` while leaving the backlog
  copy in place, then hit `max_turns`; failure handling routed to Orchestrator,
  which selected CTO ticket shaping even though product implementation had
  already landed.

Decision: runtime failures such as non-Orchestrator `max_turns` now stop as
foundation telemetry instead of being routed through Orchestrator. Ticket
completion is also tightened: copying a ticket into `done/` is blocked so
completion must be a single `git mv`, multiline evidence fields count as valid
preflight evidence, and feature files reject duplicate scenario IDs. The
`ticket_create` tool strips duplicate leading ticket-title headings from model
provided bodies.

### Follow-up `demo-123` Replay: Dogfood Must Not Become Product Mutation

After runtime failure containment, another clean replay used:

```bash
<validation-root> start \
  --repo <validation-root> \
  --db <validation-root> \
  --log-file <validation-root>
```

Positive evidence:

- CEO, COO, CTO, Engineer, bounded Engineer repair, QA, and Security stayed on
  the Space Invaders product lane.
- The run produced a product-specific plan, BDD feature contract, ordinary
  product ticket, browser-game implementation commits, ticket completion, QA
  approval, and security review.
- Guardrail, ticket-gate, and Dogfood `max_turns` failures remained foundation
  telemetry; no target intervention-debt tickets were created.

Residual finding:

- Dogfood spent 40 turns on shell-heavy validation, hit `max_turns`, and left
  `package.json` plus `package-lock.json` dirty after trying to add dev/start
  support itself.
- This was not the original intervention-debt starvation, but it was the same
  class of role-boundary failure: validation work turned into target mutation
  before the validator produced bounded evidence or a disposition.

Decision: Dogfood is now observation-first. It may write bounded reports under
`docs/reports/dogfood/` and create target-owned findings through
`ticket_create`, but tool policy blocks direct product/package `file_write`.
Generated Dogfood guidance now stops after pre-flight failures, records a
structured disposition, forbids editing product files to make validation pass,
and treats foundation/runtime failures as telemetry or blocked dispositions by
default. `git_push` also skips cleanly when a throwaway demo repo has no remote.

### Follow-up `demo-123` Replay: New Lockfiles Must Not Trap Product Delivery

After score-export remediation visibility, another clean replay used:

```bash
<validation-root> start \
  --repo <validation-root> \
  --db <validation-root> \
  --log-file <validation-root>
```

Positive evidence:

- CEO, COO, CTO, and Engineer stayed on the Space Invaders product lane.
- No target intervention-debt tickets were created for CEO, COO, or Engineer
  guardrail blocks.
- CTO created ordinary product ticket T-001 for player movement, and Engineer
  claimed it before producing browser-game source and tests.

Residual finding:

- Engineer generated a new root `package-lock.json` with 2,012 lines while
  adding test support. Tracked root lockfile churn was already exempt from
  source-file blast-radius line counts, but new untracked root lockfiles were
  still counted as ordinary source files. The role then repeated the same
  blocked commit attempt, producing ten Engineer guardrail telemetry events
  while product files sat uncommitted.

Decision: untracked root dependency metadata must use the same blast-radius
treatment as tracked dependency metadata. Root lockfiles remain git-visible and
secret-scanned, but their generated line count does not count as source-file
blast radius. Product source files still count normally, so the guardrail keeps
blocking oversized implementation files while allowing normal package-manager
lockfile creation to accompany a bounded product slice.

### Follow-up `demo-123` Replay: Shell Syntax Must Not Burn Completion Turns

After untracked lockfile blast-radius hardening, another clean replay used:

```bash
<validation-root> start \
  --repo <validation-root> \
  --db <validation-root> \
  --log-file <validation-root>
```

Positive evidence:

- CEO, COO, CTO, and Engineer stayed on the Space Invaders product lane.
- CTO created ordinary product ticket T-001 for player movement.
- Engineer produced and committed product artifacts for ship movement and a
  browser test page.
- The prior package-lock blast-radius loop did not recur, the target worktree
  ended clean, and no target intervention-debt tickets were created.

Residual finding:

- Engineer then spent late turns calling `shell_exec` with shell-only syntax in
  `argv`, including `: > src/test.html`, `: > /dev/null`, an empty argv call,
  and `> /dev/null`. The tool returned low-level process errors such as
  `exec: ":" not found`, so the model repeated the shape until `max_turns`
  fired before moving T-001 to done.

Decision: `shell_exec` argv mode now rejects shell syntax before process
execution. Redirection, pipes, control operators, command substitution, and
shell builtins produce one actionable error that explains argv is not shell
parsed and directs the role to `shell_command` or file tools. The aim is not to
make shell use broader; it is to turn an observed completion-window turn sink
into an immediate same-run correction.

### Follow-up `demo-123` Replay: Duplicate-Scenario Recovery Needs Exact Lines

After shell argv hardening, another clean replay used:

```bash
<validation-root> start \
  --repo <validation-root> \
  --db <validation-root> \
  --log-file <validation-root>
```

Positive evidence:

- CEO, COO, CTO, Engineer, QA, Security, and Dogfood all completed.
- The target produced a Space Invaders walking-skeleton implementation, moved
  T-001 to `docs/tickets/done/`, passed QA and Security review, and ended with
  a clean worktree.
- No target intervention-debt tickets were created; guardrail findings stayed
  foundation telemetry.

Residual finding:

- Engineer completed but needed 43 LLM calls and nine guardrail blocks. The
  dominant loop was duplicate `F-001-S001` scenario-heading writes. The role
  used `grep` and saw both the real scenario heading and the Scenario Schedule
  list entry, then repeatedly tried full-file rewrites, created a `.bak`
  sidecar, and attempted a forbidden `rm` cleanup before finally recovering.

Decision: duplicate-scenario guardrail errors must include exact duplicate
heading lines and explain that Scenario Schedule references are allowed. This
keeps the invariant mechanical while making the recovery step concrete: read
the file, replace the existing scenario section once, and do not append another
heading with the same ID.

### Follow-up `demo-123` Replay: Survey Retry Must Respect Runtime Failure Stops

After duplicate-scenario recovery hardening, another clean replay used:

```bash
<validation-root> start \
  --repo <validation-root> \
  --db <validation-root> \
  --log-file <validation-root>
```

Positive evidence:

- The duplicate-scenario loop did not recur: Engineer had two guardrail blocks
  instead of nine before landing product commits.
- The target had a clean worktree after the first Engineer's product commits.
- No target intervention-debt tickets were created for runtime or guardrail
  signals.

Residual finding:

- The first Engineer still hit `max_turns` with T-001 left in
  `docs/tickets/in-progress/`.
- Failure handling correctly kept the `max_turns` signal out of Orchestrator,
  CTO, and target intervention debt, but the native survey watchdog immediately
  saw an eligible in-progress ticket and enqueued another Engineer
  `ticket_delivery` job from `eligible_in_progress_ticket`.

Decision: ticket-owner survey routing must respect runtime-failure containment.
After a recent same-role runtime failure such as `max_turns`, the survey loop
pauses same-role ticket-owner retry for a cooldown window. Eligible in-progress
tickets still outrank backlog work, but the watchdog cannot bypass the
foundation-telemetry stop by immediately retrying the same failed role.

### Follow-up `demo-123` Replay: Healthy Lifecycle Still Needs Cheaper Evidence

After survey retry containment, another clean replay used:

```bash
<validation-root> start \
  --repo <validation-root> \
  --db <validation-root> \
  --log-file <validation-root>
```

Positive evidence:

- CEO, COO, CTO, Engineer, QA, Security, and Dogfood all completed.
- T-001 reached `docs/tickets/done/`, QA and Security approved, Dogfood
  validated, and the target worktree ended clean.
- No target intervention-debt tickets were created. The remaining guardrail
  blocks stayed foundation evidence.
- The previous same-role retry loop did not recur; the run had no `max_turns`
  failure.

Residual finding:

- The product lifecycle is now progressing, but it is still too expensive for a
  tiny static HTML demo. Engineer used 48 calls and roughly 1.63M trace tokens;
  Dogfood used 32 calls and roughly 772k trace tokens.
- Engineer repeatedly edited ticket metadata through shell substitutions after
  product work was done, and Dogfood performed broad shell discovery plus
  package/container-oriented checks before settling on a static server smoke
  test.

Decision: generated role guidance should make static target validation and
ticket evidence updates explicit. Engineer should accept a bounded static HTTP
smoke test for intentionally static HTML/CSS/JS targets and update ticket
evidence with one full-file replacement. Dogfood should skip package-manager
and container-build expectations that do not apply to no-manifest static
targets, start static servers only in background mode, and keep validation
evidence bounded.

### Follow-up `demo-123` Replay: Same-Role Planning Handoffs Must Stop

After static-target role guidance, a clean replay used:

```bash
<validation-root> start \
  --repo <validation-root> \
  --db <validation-root> \
  --log-file <validation-root>
```

Positive evidence:

- CEO, COO, CTO, Engineer, QA, Security, and Dogfood all completed.
- T-001 reached `docs/tickets/done/`; QA, Security, and Dogfood approved; the
  target worktree ended clean.
- No intervention-debt tickets were created. Runtime guardrail blocks stayed
  foundation telemetry.
- Engineer improved from 48 calls to 35, and Dogfood improved from 32 calls to
  21 while validating the same static game shape.

Residual finding:

- The first COO job completed with `next_need: exec_plan`, no structured
  target owner, and no work products. Direct dispatch interpreted that
  deterministic need literally and enqueued a second COO job. The second COO
  eventually committed the plan and feature contract and routed CTO, so product
  progress survived, but the extra same-role pass was avoidable lifecycle
  noise.

Decision: a direct non-Orchestrator dispatch whose only route is a `next_need`
that maps back to the same role is not forward progress. Dispatch now stops
with a same-role reason instead of enqueueing the same role again, and
generated COO guidance explicitly forbids finishing with planning `next_need`
values that route back to COO.

### Follow-up `demo-123` Replay: Release Notes Must Be In The Live Chain

After same-role handoff containment, a clean replay used:

```bash
<validation-root> start \
  --repo <validation-root> \
  --db <validation-root> \
  --log-file <validation-root>
```

Positive evidence:

- The same-role COO loop was fixed: one COO job handed directly to CTO, and the
  target had product-specific planning, feature-contract, and implementation
  ticket commits.
- The target produced semantic commits such as `CEO: Define scope...`,
  `plan: update active scenario schedule...`, and `tickets: create
  implementation ticket...`.

Residual finding:

- `VERSION` remained `0.1.0` and target `CHANGELOG.md` still contained only
  the generated header after semantic target commits. The release/versioning
  rule existed in generated docs and the Release Manager prompt, but the
  dispatch chain stopped at product validation rather than enqueueing Release
  Manager.
- The same replay also exposed the review-role variant of the same-role guard:
  Security completed with `next_need: security_review`, which mapped to
  Security and stopped before Dogfood. Review roles need to interpret their own
  review `next_need` as completed evidence and move forward to the next review
  or release owner.
- Source `release backfill-notes --check` also reported historical entries
  needing backfill, but the first write pass showed the tool could downgrade
  already-rich entries into generic fallback prose.

Decision: release discipline must be executed by the lifecycle, not left as
passive doctrine. Dogfood approval/completion now routes to Release Manager
when configured, review roles with self-named review needs route forward rather
than stopping, Release Manager guidance runs `release backfill-notes --check`
after generated notes, and the backfill tool preserves complete current
narrative entries instead of flattening them.

### Follow-up `demo-123` Replay: Stuck Tools Must Not Strand Engineer

After the release-chain fix, a clean replay used:

```bash
<validation-root> start \
  --repo <validation-root> \
  --db <validation-root> \
  --log-file <validation-root>
```

Positive evidence:

- CEO, COO, and CTO each completed once and routed directly through the product
  delivery path.
- Engineer created concrete Space Invaders source files under `src/`, showing
  product implementation had started rather than intervention-debt churn.

Residual finding:

- Engineer then went quiet with uncommitted `src/` files, no new llama-server
  activity, and the queue row still marked `running`. The nominal tool timeout
  did not return control to the loop, so the process had to be stopped by the
  operator.

Decision: tool execution needs a hard executor timeout around handlers, not
only context propagation. A stuck or context-ignoring handler must return an
actionable timeout error to the model so the role can record a blocker or fail
closed, and the server can classify the outcome instead of leaving a target
repo dirty forever.

### Follow-up `demo-123` Replay: Remediation Edges Stay Product-First

After completing the remaining `MH-048` negative-path checks for destructive
recipe approval, dirty-worktree blockers, and missing optional-tool guidance, a
clean replay used:

```bash
go run ./cmd/mars-harness start \
  --repo <validation-root> \
  --db <validation-root> \
  --log-file <validation-root> \
  --debug
```

Positive evidence:

- The target started from a clean Git repo containing only a Space Invaders
  README brief and an initial commit.
- `start` auto-initialized the deployed harness, committed the generated
  baseline, registered the repo, and seeded exactly one CEO bootstrap job with
  the stable idempotency key shape `seed:<repoID>:ceo:bootstrap`.
- CEO read the Space Invaders brief, completed in 5 turns, and handed
  `next_need: exec_plan` to COO.
- COO wrote a product-specific active plan and BDD feature contract for the
  Space Invaders game, committed them, and handed `next_need:
  ticket_breakdown` forward.
- The one guardrail block caused by attempting to record a disposition with
  uncommitted plan/feature changes stayed foundation-owned telemetry; no target
  intervention-debt tickets were created.

Residual finding:

- The replay was stopped manually while `cto-weekly` was reading context to
  create implementation tickets, so the scratch DB retained one running
  `cto-weekly` row. The routing itself matched current doctrine because
  `ticket_breakdown` maps to CTO ticket shaping before Engineer.

Decision: `MH-048` can close. The remediation edge changes did not regress the
fresh product-first lifecycle through product planning and feature-contract
creation. The next continuous-build loop should claim `MH-049` and broaden the
dogfood evidence far enough to observe ticket creation, implementation,
quality-score export, release notes, and release publication behavior.

### Non-Static API Replay: Canonical Feature Contract Guidance

After adding scheduler duplicate-work suppression and bounded repo-local build
artifact cleanup, a clean Task Notes API replay against
`<validation-root>` reached CEO and COO
planning but stalled before CTO ticketing.

Positive evidence:

- CEO found the generated canonical
  `docs/features/F-001-product-walking-skeleton.md` contract and completed with
  an `exec_plan` handoff.
- Guardrails prevented a duplicate `docs/features/F-001-task-notes-api.md`
  feature contract and prevented duplicate scenario headings in the canonical
  contract.
- No target intervention-debt tickets were created; the failure stayed as
  foundation evidence.

Residual finding:

- The generated CEO/COO prompts still left enough ambiguity for both roles to
  try product-specific `F-001` contract paths or duplicate starter scenario IDs
  before reaching CTO ticket creation.

Decision: bootstrap role guidance now names canonical feature-contract reuse as
the happy path. CEO does not write `docs/features/` and passes the existing
contract path to COO. COO searches `docs/features/F-NNN*.md`, edits the
existing path when present, and rewrites starter scenarios in place with unique
scenario IDs instead of creating a second slugged contract.

### Non-Static API Replay: Module-Named Build Artifacts

After adding canonical feature-contract guidance, a clean Task Notes API replay
against `<validation-root>` reached CEO, COO,
CTO, and Engineer.

Positive evidence:

- CEO, COO, and CTO each completed once, and the run reached ordinary
  product-ticket implementation instead of stalling in bootstrap planning.
- COO reused the canonical
  `docs/features/F-001-product-walking-skeleton.md` contract instead of
  creating a duplicate `F-001` path.
- Guardrail and max-turn failures stayed foundation-owned telemetry. No target
  intervention-debt tickets were created, and the failed Engineer did not
  dispatch an Orchestrator recovery loop.

Residual finding:

- Engineer generated a root executable named `task-notes-api`, matching the
  Go module basename rather than the repo directory. The cleanup exception only
  allowed a repo-named binary, so blast-radius checks treated the untracked
  executable as a huge source file and blocked later writes, disposition, and
  decision recording until Engineer hit max turns.

Decision: repo-local build-artifact cleanup now accepts the same bounded shape
for module-named Go binaries. `rm`/`unlink` is allowed only for untracked,
root-level, binary-looking files named after either the repo directory or the
root `go.mod` module basename. Recursive removal, tracked files, ordinary
source/docs, nested paths, and arbitrary filenames remain blocked.

### Non-Static API Replay: Cleanup Must Be Discoverable

After adding module-named build-artifact cleanup, a clean Task Notes API replay
against `<validation-root>` again reached
Engineer implementation and validation.

Positive evidence:

- CEO, COO, and CTO completed exactly once and reached Engineer without
  duplicate planning artifacts or intervention-debt amplification.
- Engineer claimed the product ticket, committed source files, added `go.mod`,
  and reached a real `go build -o task-notes-api ...` validation attempt.
- Runtime failures remained foundation telemetry; no target intervention-debt
  tickets were created and no Orchestrator recovery loop was dispatched after
  max turns.

Residual finding:

- The new cleanup exception was available but invisible. The blast-radius error
  named `task-notes-api` as the oversized file but only suggested splitting
  work or raising `MaxLinesPerFile`; Engineer never attempted `rm
  task-notes-api` and instead kept retrying builds, writes, commits, and ticket
  movement against the dirty repo until max turns.

Decision: blast-radius validation now appends an exact cleanup hint when the
oversized file is a cleanable generated artifact. The error keeps the normal
blast-radius failure, then adds `Generated build artifact "<artifact>" can be
cleaned with rm <artifact>`, preserving the narrow removal policy while making
the recovery path model-visible.

### Non-Static API Replay: Server Validation Must Use Managed Background Mode

After adding generated artifact cleanup hints, a clean Task Notes API replay
against `<validation-root>` confirmed the
cleanup path and exposed the next generic service-validation bottleneck.

Positive evidence:

- CEO, COO, CTO, and Engineer reached product-specific planning, ticketing,
  ticket claim, source implementation, and build validation without target
  intervention-debt ticket amplification.
- The blast-radius error named `rm task-notes-api`; Engineer used that command
  and recovered from the generated binary trap instead of reaching max turns on
  artifact cleanup.

Residual finding:

- Engineer then validated the API by running `go run src/main.go` in the
  foreground, spending a 30-second timeout. It recovered with shell syntax that
  backgrounded `go run` inside `shell_command`; that left port `8080` occupied
  by the compiled server process, caused a later `background:true` attempt to
  fail with "address already in use", and produced extra malformed `:8080`
  shell calls before the run was manually stopped.

Decision: long-running application validation is a managed-tool concern, not a
shell convention. `shell_exec` now rejects the shell background operator `&` in
`shell_command` and tells the role to use `background:true` with separate
readiness probes. A `background:true` process that exits during the startup
capture window now returns an error with initial output and exit code, so
crashed or port-conflicted servers do not look successfully started. Generated
Engineer guidance mirrors the same rule.

### Non-Static API Replay: Validation Build Outputs Must Not Enter The Repo

After managed background validation was added, a clean Task Notes API replay
against `<validation-root>` confirmed the
service process fix and exposed the next artifact-prevention gap.

Positive evidence:

- CEO, COO, and CTO each completed once and reached an ordinary product
  implementation ticket.
- Engineer used `shell_exec` with `background:true` for `go run main.go`,
  successfully probed `GET /health`, and killed the managed PID. The previous
  foreground timeout and shell-background process leak did not recur.
- Runtime guardrail and circle-detection failures stayed foundation-owned
  telemetry. No target intervention-debt tickets were created.

Residual finding:

- Engineer then ran `go build -o task-notes-api main.go`, creating an untracked
  binary inside the target repo. Blast-radius validation correctly blocked the
  oversized binary-shaped diff, but because the artifact already existed,
  later malformed empty `shell_exec` calls were also masked by the dirty
  blast-radius precheck until the role ended with `circle_detected`.

Decision: validation build artifacts should be prevented, not only cleaned up
after the fact. `shell_exec` now rejects explicit `go build -o <path>` outputs
that resolve inside the target repo and instructs roles to use external temp
paths for runnable validation binaries. The cleanup exception remains for
artifacts already present, and malformed `shell_exec` payloads now surface their
own validation error before dirty-diff containment can obscure them.

### Mars Observer Replay: Dry-Run Needs An Explicit No-Init Boundary

The first Mars observer validation against
`/path/to/local-redacted` proved that `doctor`, `update check`,
read-only tools, and observer-trust mutation blocks can inspect the real Mars
checkout without writing to it. It also found that normal `run --dry-run`
auto-initializes `.harness/` before context assembly, which is correct for
plug-and-play delivery but unsafe for an observer-mode supersession benchmark
against an uninitialized legacy target.

Decision: normal `run` keeps auto-initialization as the product default, and
observer validation opts into `--no-init`. When `.harness/manifest.yaml` is
missing and `--dry-run --no-init` is supplied, the command reports the
missing-harness boundary, states that no files were written, and exits before
scaffolding or LLM execution. This lets the factory keep running live observer
checks against real targets without blurring the foundation/deployed write
boundary.
