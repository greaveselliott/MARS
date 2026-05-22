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

### Non-Static API Replay: Bare Port Tokens Must Be Tool-Shape Errors

After repo-local build-output prevention was added, a clean Task Notes API
replay against `<validation-root>` confirmed
the guardrail and exposed the next malformed command pattern.

Positive evidence:

- CEO, COO, and CTO completed once, created product-specific planning, and
  handed off to Engineer.
- Engineer claimed `T-001`, committed a Go `GET /health` implementation, and
  pushed locally with a clean no-remote skip.
- `go build -o task-notes-api src/main.go` was blocked before process execution;
  no `task-notes-api` binary was created in the target repo.
- `mars-harness scores export` wrote quality evidence showing zero target
  intervention-debt tickets, and the target was committed cleanly after the run.

Residual finding:

- After the build-output block, Engineer called `shell_exec` with `argv:
  [":8080"]` twice. The runtime treated it as a missing executable, and the
  repeated malformed call triggered `circle_detected`.

Decision: bare port tokens such as `:8080` are invalid validation commands.
`shell_exec` now rejects them before process execution and tells roles to start
the actual server command with `background:true`, then probe with
`curl http://localhost:8080/health` or the target route. The build-output
blocker now also names a stable external validation-binary shape such as
`<validation-root>` so the model sees the correction directly.

### Non-Static API Replay: Scratch Validation Must Stay Out Of The Target Root

After bare-port rejection was added, a clean Task Notes API replay against
`<validation-root>` confirmed the validation
path and exposed the next source-owned turn sink.

Positive evidence:

- CEO, COO, CTO, Engineer, QA, Security, Dogfood, and Release Manager each
  completed once.
- CTO created one ordinary product ticket; Engineer claimed it, implemented a
  Go `GET /health` endpoint, added tests, moved the ticket to done, and the
  release-manager generated local `0.2.0` release notes before stopping on the
  expected no-remote publication blocker.
- `go build -o task-notes-api` was blocked before creating a repo-local binary,
  and Engineer recovered by using `<validation-root>`.
- Dogfood proved the product with `go test ./...`, an external build output,
  managed `background:true` server startup, `curl /health`, POST 405 evidence,
  and cleanup.
- Intervention-debt count remained `0`, and `scores export` graded the target
  overall `A` while surfacing guardrail-block telemetry as improvement targets.

Residual finding:

- Engineer wrote a root `validate.sh` as temporary validation, then could not
  remove it because broad `rm` is blocked. The script was committed with the
  done-ticket move. Dogfood then ran it and exposed a portability bug:
  `timeout` was not available on the host. QA and Security did not reject the
  script, so a scratch validation file became accidental product surface.

Decision: scratch validation should be prevented before cleanup is needed.
`file_write` now blocks new root-level validation shell scripts such as
`validate.sh` while allowing existing project-owned scripts to be updated.
`shell_exec` rejects external `timeout` and `gtimeout` commands and points roles
to tool `timeout_seconds` or managed `background:true` probes. Generated
Engineer and Dogfood guidance mirrors the same rule.

### Non-Static API Replay: Background Cleanup Must Kill Wrapper Children

The first replay after scratch-validation prevention used
`<validation-root>` and exposed an older
cleanup leak before it could reach Dogfood.

Positive evidence:

- CEO, COO, and CTO again reached product-specific planning and a single
  ordinary product ticket.
- Engineer claimed the ticket, wrote Go source and tests, initialized the Go
  module, and passed `go test ./...`.
- Bare `:8080` commands were blocked as tool-shape errors and kept out of the
  target backlog as foundation telemetry.

Residual finding:

- A `go run` child server from the previous canary still owned port `8080`.
  The harness had killed the tracked background wrapper PID, but not the
  compiled child process. Engineer hit `listen tcp :8080: bind: address already
  in use`, repeated malformed `:8080` cleanup attempts, and stopped with
  `circle_detected`.

Decision: background cleanup is responsible for wrapper children, not only the
tracked process. `shell_exec` now discovers known descendants, kills them from
leaf to root, then kills the tracked process group and process. This preserves
the `background:true` validation contract while preventing stale dev servers
from leaking across jobs or replay targets.

### Non-Static API Replay: Default Go Build Must Not Dirty The Target

The next replay used `<validation-root>` to
validate background descendant cleanup in a fresh Task Notes API target.

Positive evidence:

- CEO, COO, CTO, Engineer, QA, and Security completed product-specific work with
  one ordinary product ticket and zero target intervention-debt tickets.
- Engineer implemented the Go `/health` endpoint, passed `go test .`, recovered
  from a repo-local build-output block by using `<validation-root>`,
  validated the live endpoint with `curl`, and moved `T-001` to done.
- Dogfood validated `GET /health` with HTTP 200 JSON evidence, confirmed a
  missing route returned 404, ran `go test -v ./...`, wrote an E2E report, and
  committed the report before hitting the turn limit.
- Runtime failures stayed in foundation telemetry: Dogfood `max_turns` did not
  create target intervention-debt or dispatch another autonomous recovery loop.

Residual finding:

- Dogfood ran `go build ./...` with no `-o`. The command created the default
  root binary `task-notes-api` before post-command blast-radius validation
  rejected the diff. Dogfood eventually removed the artifact, but the turn sink
  proved that implicit Go build outputs need the same preflight protection as
  explicit repo-local `go build -o` outputs.
- Manual `kill -9 <go-run-wrapper-pid>` can still leave a compiled `go run`
  child server behind during the same job. Dogfood recovered with `lsof` and a
  targeted child kill, but the live trace shows background process ownership is
  not yet obvious enough to roles.

Decision: validation builds should never rely on Go's default repo-local output
path. `shell_exec` now rejects `go build` without `-o` before process execution
and tells roles to use `go test ./...` for compile validation or an external
`-o /tmp/<name>-validation` path when a runnable binary is required.

### Non-Static API Replay: Same-Job Kill Must Clean Background Children

The next replay used `<validation-root>` to
validate default Go build output preflight in a fresh Task Notes API target.

Positive evidence:

- CEO, COO, and CTO again reached product-specific planning and a single
  ordinary product ticket with zero target intervention-debt tickets.
- COO corrected a commit-before-disposition guardrail block and committed the
  updated plan and feature contract before handoff.
- Engineer claimed `T-001`, implemented a Go `/health` endpoint under `src/`,
  recovered from a foreground server timeout by using `background:true`, and
  validated `/health`, POST method rejection, and missing-route behavior.
- The explicit repo-local `go build -o task-notes-api` guardrail fired before
  artifact creation, and Engineer recovered with `<validation-root>`.

Residual finding:

- Engineer killed the tracked `go run` wrapper PID after live validation, but
  the compiled child server kept port `8080` bound during the same job. The
  external `/tmp` validation binary then exited during startup because the port
  was still occupied. Engineer repeated bare `:8080` commands and stopped with
  `circle_detected`.

Decision: targeted cleanup of a tracked background PID should behave like
job-boundary cleanup. `shell_exec` now intercepts `kill <tracked-background-pid>`
and kills the tracked process tree, including known descendants, before
returning success. Untracked PIDs still fall through to normal `kill` behavior.

### Non-Static API Replay: No-Op Shell Calls Should Exit Toward Completion

The next replay used `<validation-root>` to
validate tracked background kill interception in a fresh Task Notes API target.

Positive evidence:

- CEO, COO, and CTO again reached product-specific planning and ordinary
  product ticketing with zero target intervention-debt tickets.
- COO's implementation write attempt was blocked by ownership policy and
  recovered into committed planning artifacts.
- Engineer claimed `T-001`, implemented Go product code and tests, recovered
  from the repo-local build-output block with `<validation-root>`,
  and got `go test ./src` passing.
- Same-job tracked PID cleanup worked: after `go run src/main.go` started with
  `background:true`, `shell_exec` intercepted `kill -9 <tracked-pid>` and killed
  the process tree. The external validation binary then started successfully on
  port `8080`.
- The terminal runtime failure stayed foundation-owned and created no target
  intervention-debt tickets.

Residual finding:

- After successful live validation, Engineer restarted the external validation
  binary and then called `shell_exec` with empty `argv` and repeated single `:`
  commands. Those no-op calls became guardrail blocks and triggered
  `circle_detected` before ticket completion.

Decision: no-op shell calls are model-shape drift, not target defects. Empty
`argv`, blank `argv`, and single `:` now return completion guidance instead of
guardrail errors. When a managed background process is active, the guidance
names the tracked PID and tells the role to stop it, update ticket evidence,
commit, push, and record `job_disposition_record`.

### Non-Static API Replay: DocSync Failures Must Block Approval

The next replay used `<validation-root>` to
validate no-op shell guidance in a fresh Task Notes API target.

Positive evidence:

- CEO, COO, and CTO reached product-specific planning and one ordinary product
  ticket with zero target intervention-debt tickets.
- Engineer claimed `T-001`, implemented the Go `/health` endpoint, added tests,
  used managed `background:true` validation, stopped tracked PIDs, moved the
  ticket to done, and recorded a terminal disposition.
- The prior empty-argv/single-`:` loop did not recur. Background startup output
  guided the role toward `kill <tracked-pid>`, ticket evidence, commit, and
  disposition.
- Security, Dogfood, and Release Manager progressed far enough to expose
  downstream quality and release behavior. Release Manager generated local
  release notes and stopped with a no-remote blocker instead of mutating
  remotes.

Residual findings:

- Engineer added malformed one-line `MarsDocSync` metadata pointing at
  `docs/features/F-001-S002.md`; `F-001-S002` is a scenario ID, not a feature
  contract path.
- QA and Security both ran `docsync_audit`, saw `FAIL:` findings for missing
  metadata, and still approved. Dogfood also approved release readiness while
  the target source failed DocSync.
- A policy-blocked external `timeout` command during Dogfood validation was
  classified as retryable `tool_timeout`, enqueuing a duplicate Dogfood retry
  even though the event was a deterministic guardrail block.
- `scores export` graded the target A from terminal outcomes despite the
  DocSync escape, which remains a later scoring-quality follow-up.

Decision: successful implementation, review, validation, and release
dispositions now mechanically run DocSync and are rejected while findings
exist. Deployed target repos still require valid metadata, but are not forced
to cite foundation-only source docs. External `timeout` policy blocks classify
as guardrail blocks rather than retryable tool timeouts. Generated role
guidance now distinguishes scenario IDs from feature-contract paths and treats
DocSync `FAIL:` output as a release/readiness blocker.

### Non-Static API Replay: No-Op Guidance Must Fail The Tool Call

The next replay used `<validation-root>` to
validate the DocSync disposition gate and timeout classification in a fresh Task
Notes API target.

Positive evidence:

- CEO, COO, and CTO again produced product-specific planning and a single
  ordinary product ticket with zero target intervention-debt tickets.
- Runtime guardrail signals remained foundation-owned telemetry. The run did
  not route a failed Engineer job back through Orchestrator.
- The target `scores export` produced `Insufficient evidence` after the failed
  run, which is more conservative than the earlier A-grade escape.

Residual findings:

- Engineer wrote `src/main.go` and `src/main_test.go` without `MarsDocSync`
  metadata. A manual `docsync audit` reported both files as missing metadata.
- After successful `go test ./...` and an external build, Engineer repeatedly
  called `shell_exec` with empty `argv`. The soft no-op guidance was treated as
  progress, and the run ended with `circle_detected` before ticket evidence,
  ticket completion, or disposition.

Decision: no-op `shell_exec` calls now return a tool error while preserving the
same completion guidance, so empty `argv` and single `:` calls are no longer
successful turns. `file_write` also rejects source/test writes that lack valid
top-of-file `MarsDocSync` metadata or reference a missing doc path, catching the
run15 source drift before files enter the target worktree.

### Non-Static API Replay: Policy Must Normalize Shell Argv Like Execution

The next replay used `<validation-root>` to
validate the no-op hard-error and source-write DocSync preflight in a fresh Task
Notes API target.

Positive evidence:

- The source-write preflight did not block useful code. Engineer's first source
  write already included a structured `MarsDocSync` block pointing at
  `docs/features/F-001-product-walking-skeleton.md`.
- Duplicate feature-contract creation, CEO ownership boundaries, ticket-file
  hand-writing, and clean-handoff commits all remained mechanical guardrails.
- No target intervention-debt tickets were created.

Residual findings:

- CEO and COO still spent many turns recovering from broad discovery and
  role-boundary mistakes before completing their handoffs.
- Engineer attempted the correct claim command, but emitted `argv` as a
  JSON-encoded array string. Execution can normalize that shape, but the
  claim-exception policy decoded `argv` directly as `[]string`, so the command
  was blocked by the claim guard it was supposed to satisfy.

Decision: shell tool policy now decodes `shell_exec` arguments through the same
normalizing parser used by execution before checking the backlog-to-in-progress
ticket claim exception. Ownership policy and execution therefore agree on
simple malformed argv shapes.

### Non-Static API Replay: Engineer Shell Work Must Start With Claim

The next replay used `<validation-root>` to
validate the claim-argv normalization fix in a fresh Task Notes API target.

Positive evidence:

- CEO, COO, and CTO again reached product-specific planning and a committed
  ordinary product ticket with zero target intervention-debt tickets.
- The lifecycle reached Engineer without duplicate bootstrap jobs or automatic
  intervention-debt amplification.
- CTO created and committed `T-001` for the current BDD scenario, confirming
  the previous shell-argv policy fix was no longer the immediate blocker.
- Runtime failures and guardrail blocks stayed foundation-owned and did not
  route through Orchestrator after Engineer failed.

Residual findings:

- COO initially emitted `job_disposition_record.evidence_links` as a string
  containing list syntax and needed extra attempts before recording the
  terminal disposition.
- Engineer read the backlog ticket and feature contract, but did not claim
  `T-001` before shell discovery. A broad `find .` command was blocked, then
  repeated empty `shell_exec` calls ended the job with `circle_detected`.
- No product source was created, and the ticket remained in backlog.

Decision: Engineer `shell_exec` is now claim-first whenever an ordinary
backlog product ticket exists and no in-progress product ticket is present.
The only allowed pre-claim shell command is the `git mv` ticket claim; all
other shell calls receive the exact claim command and commit message guidance.
`job_disposition_record` also normalizes simple list-as-string fields before
validation so harmless evidence-list formatting drift does not spend role
turns.

### Non-Static API Replay: Server Validation Must Use Managed Background Mode

The next replay used `<validation-root>` to
validate claim-first shell execution in a fresh Task Notes API target. The
target brief asked for a tiny HTTP API with a `GET /health` endpoint.

Positive evidence:

- Engineer first tried `shell_exec` `ls -la` before claiming the ticket. The
  claim-first guard blocked it and returned the exact `git mv` claim command.
  Engineer immediately moved `T-001` into `docs/tickets/in-progress/` and
  committed `chore(tickets): claim T-001`.
- CEO, COO, and CTO again reached product-specific planning and a committed
  ordinary product ticket with zero target intervention-debt tickets.
- Engineer recovered from source-write DocSync preflight by writing valid
  top-of-file `MarsDocSync` metadata on `main.go` and `main_test.go`.
- `docsync_audit` checked the implementation with zero findings,
  `go test ./...` passed, and an external `/tmp` validation build succeeded.
- Runtime failures stayed foundation-owned. The failed Engineer job recorded
  telemetry but did not dispatch Orchestrator and did not create target
  intervention-debt tickets.

Residual findings:

- Engineer attempted `go build -o task-notes-api` inside the repository; the
  existing build-output guard blocked it and Engineer recovered by building to
  `<validation-root>`.
- Engineer then ran `go run main.go` as a foreground validation command. The
  tool spent its timeout while the server was running on `:8080`, then Engineer
  repeated empty `shell_exec` calls until `circle_detected`.
- The role tried to remove the external validation binary with `rm`, which is
  still blocked by the destructive shell policy. That is noisy, but smaller
  than the foreground server timeout because it did not dirty the target repo.

Decision: likely server and watcher commands now have a pre-execution shell
guard. HTTP-shaped `go run`, common JavaScript dev servers, Python HTTP
servers, ASGI/WSGI servers, Vite, Next, and similar watch commands must use
`background:true`, separate readiness probes, and tracked PID cleanup. The
next API canary should confirm Engineer reaches the same implementation point
and then validates the service through managed background execution instead of
a foreground timeout.

### Non-Static API Replay: Security Needs Bounded Terminal Evidence

The next replay used `<validation-root>` to
validate foreground server preflight and managed runtime validation after the
run18 fix.

Positive evidence:

- CEO, COO, CTO, Engineer, and QA completed. `T-001` moved through backlog,
  in-progress, and done, and the target worktree ended clean after quality
  evidence was committed.
- Engineer again hit the claim-first shell guard, recovered immediately with
  the exact `git mv` claim, implemented the Go health endpoint, wrote
  MarsDocSync metadata on source and test files, passed `go test ./...`, and
  passed `docsync_audit`.
- Runtime validation used managed background execution. The external
  `<validation-root>` binary started with `background:true`,
  `curl /health` returned service/status/timestamp JSON, and the tracked PID
  was killed.
- `scores export` produced overall grade `C`, one done ticket, zero open
  intervention-debt tickets, and Factory Pace rows for each role.
- Security failed with `max_turns`, but the failure stayed foundation-owned:
  no Orchestrator dispatch loop and no target intervention-debt ticket.

Residual findings:

- COO still tries ticket creation through the CLI and direct `file_write`
  before handing off to CTO, costing turns even though recovery works.
- Engineer still spends discovery and recovery turns after the ticket claim,
  including one attempt to run a repo-local binary name after an external build.
- Security has enough evidence to approve, but repeats validation and reaches
  `max_turns` before writing an audit report or recording terminal
  disposition.

Decision: generated Security guidance now treats post-QA feature review as a
bounded pass. Once diffs, secrets scan, changed code, done ticket,
`docsync_audit`, tests, and at most one managed smoke probe are complete,
Security should write the audit, commit, push when possible, and record
`job_disposition_record` instead of running more liveness checks.

### Non-Static API Replay: Bounded Security And Release Notes Complete

The next replay used `<validation-root>` to
validate the bounded Security guidance and the target release-note loop after
the run19 fix.

Positive evidence:

- CEO, COO, CTO, Engineer, QA, Security, Dogfood, and Release Manager all
  completed. The target finished with one done product ticket, local release
  notes, tag `v0.2.0`, and a clean worktree after quality evidence was
  committed.
- Security followed the bounded review path: inspected recent commits, scanned
  for secrets, ran `docsync_audit`, ran `go test ./...`, built the validation
  binary outside the repo, performed one managed `/health` probe, killed the
  tracked PID, wrote `docs/reports/security/security-audit-2026-05-20.md`,
  committed it, and recorded terminal disposition in 14 turns.
- Dogfood then revalidated tests, external build, HTTP 200 response, JSON
  payload, and POST 405 behavior, wrote
  `docs/reports/dogfood/dogfood-validation-2026-05-20.md`, committed it, and
  recorded terminal disposition.
- Release Manager generated target release notes with
  `mars-harness release notes --repo . --bump auto`, committed
  `release: notes 0.2.0`, created tag `v0.2.0`, and stopped with a release
  publication blocker because the disposable target had no `origin` remote.
- `scores export` reported overall grade `A`, one done product ticket, zero
  backlog or in-progress tickets, and zero open intervention-debt tickets.

Residual findings:

- Engineer still paid guardrail tax before completion: one pre-claim shell
  discovery block, a broad `find .` block, a DocSync preflight block, an
  in-repo `go build -o task-notes-api` block, and a few malformed shell argv
  recovery attempts while killing the validation process.
- Dogfood still initially tried `go build ./...`, recovered to an external
  `-o <validation-root>` build, and completed. This is now an optimization target, not
  a lifecycle blocker.
- The validation evidence is still concentrated in the static-game and
  non-static HTTP API canaries. The next foundation claim should come from a
  representative target matrix rather than further overfitting this one API
  path.

Decision: bounded Security review is validated as a foundation improvement.
The next continuous-improvement slice should treat residual guardrail blocks as
pace telemetry and expand the live validation matrix across target archetypes
before broadening role prompts or tool policy.

### CLI Matrix Replay: Security Must Report Product Rework

The first CLI representative replay used
`<validation-root>` with a Note Stats CLI
brief: build `note-stats --text "some words"` and print JSON counts for words,
characters, and lines.

Positive evidence:

- The product-first lifecycle generalized beyond the static-game and HTTP API
  canaries. CEO, COO, CTO, Engineer, and QA reached product-specific planning,
  a feature contract, an ordinary product ticket, implementation, and review
  without creating intervention-debt tickets.
- Claim-first shell policy worked for the CLI target: Engineer was blocked from
  shell discovery before claiming `T-001`, then claimed and committed the
  ticket before implementation.
- Repo-local build-output prevention generalized: an in-repo
  `go build -o note-stats` was blocked, and the agent recovered to an external
  `<validation-root>` binary.
- `scores export` wrote `docs/QUALITY_SCORE.md` and reported grade `C`, with
  zero open intervention-debt tickets.

Residual findings:

- Engineer created a repo-root `debug.go` scratch probe with valid DocSync
  metadata. The metadata gate was technically satisfied, but the file was
  scratch validation noise and was later committed with ticket lifecycle work.
- QA approved based on repository inspection, but Security later ran
  `go test ./cmd/note-stats` and found a failing line-count edge case.
- Security fixed `cmd/note-stats/main.go` directly, reran the test
  successfully, then hit `max_turns` before committing, writing the audit, or
  recording terminal disposition. The target was left dirty with the uncommitted
  product fix.
- Runtime containment behaved correctly: the Security `max_turns` failure
  stayed in foundation telemetry, created no target intervention-debt ticket,
  and the dirty-target survey paused rather than dispatching more work.

Decision: Security remains a reviewer, not an opportunistic implementation
role. Security `file_write` is limited to
`docs/reports/security/security-audit-<date>.md`; functional test failures,
bad evidence, stale docs, or product code defects are recorded as
`changes_requested` feedback for Engineer with exact failing commands and
requested remediation. Root scratch probes such as `debug.go`, `probe.go`, and
`scratch.py` are blocked before creation, just like root validation scripts.

### CLI Matrix Replay: Review Rework Must Stay Bounded

The follow-up CLI replay used
`<validation-root>` with the same Note Stats
CLI brief to validate the Security authority and scratch-probe fixes.

Positive evidence:

- CEO, COO, CTO, Engineer, QA, Security, Orchestrator, and Engineer rework ran
  against a clean disposable CLI target. The product reached an ordinary done
  ticket, passing `go test ./cmd/note-stats`, and a clean target worktree
  before quality export.
- The root scratch-probe fix held: no `debug.go` or equivalent root probe was
  added to the target.
- Security no longer patched product code. It wrote and committed
  `docs/reports/security/security-audit-2026-05-20.md`, then recorded
  `changes_requested` for Engineer.
- Runtime containment held again. The later Engineer `max_turns` signal stayed
  foundation-owned telemetry, did not dispatch Orchestrator, and created zero
  target intervention-debt tickets.
- `scores export` produced overall grade `B`, one done product ticket, and
  zero open intervention-debt tickets.

Residual findings:

- Security still converted a speculative or already-safe CLI flag concern into
  release-blocking remediation. It observed missing and empty `--text` inputs
  failing safely, but still wrote a `NEEDS_REMEDIATION` finding.
- Engineer treated the Security feedback as broad rework, repeated path
  discovery, made a small patch, committed it, then kept adding extra smoke and
  newline probes until `max_turns` before recording a disposition.
- CLI convention detection still reports `start_command: go run ./cmd/...` and
  `dev_port: 8080` for a non-server CLI target.

Decision: Security findings must be grounded in current failing or exploitable
evidence, not speculative future hardening. Commands that already fail safely
or concerns about possible future extensions become PASS notes or low-severity
observations, not `changes_requested` rework. Engineer review rework is now a
bounded fast path: reproduce the exact requested failure, make one minimal
patch only when required, run the exact proof plus a relevant test/docsync
check, commit, push, and record terminal disposition without expanding into
new exploratory validation.

### CLI Matrix Replay: Engineer Must Obey The Ticket Contract

The next CLI replay used
`<validation-root>` to validate the patched
generated role guidance.

Positive evidence:

- The generated target included the new Security evidence-grounding and
  Engineer review-rework guidance.
- CEO, COO, and CTO completed product-specific planning and ticketing.
- Engineer respected the claim-first shell boundary, claimed `T-001`, and did
  not create root scratch probes.
- Runtime containment held: the Engineer `max_turns` failure stayed
  foundation-owned telemetry, did not dispatch Orchestrator, and created no
  target intervention-debt ticket.

Residual findings:

- Engineer never reached QA or Security. It spent 50 turns in initial
  implementation, repeatedly testing and rewriting CLI edge-case behavior.
- The ticket explicitly said empty `--text ""` should produce zero words,
  zero characters, and zero lines. Engineer drifted to an implementation that
  treated empty text as one line and rewrote tests around that behavior.
- The job ended with uncommitted `cmd/` and `go.mod` files and a failing
  `go test ./cmd/note-stats`.

Decision: Engineer's initial implementation path now has the same bounded
discipline as review rework. The selected ticket acceptance criteria and BDD
feature contract are the product contract for the run. Tests and code should
match that contract; useful extra edge cases become follow-up evidence rather
than same-run exploration. Once required acceptance evidence passes, Engineer
should move the ticket to done and record disposition instead of adding more
probes.

### CLI Matrix Replay: Engineer Must Close Before Packaging

The next CLI replay used
`<validation-root>` to validate the
contract-first Engineer guidance.

Positive evidence:

- CEO, COO, and CTO completed product-specific planning and ticketing, and
  Engineer claimed `T-001`.
- Engineer honored the explicit empty-text contract: the validation binary
  returned `{"word_count":0,"character_count":0,"line_count":0}` for
  `--text ""`.
- Engineer proved hello-world and multiline behavior, ran
  `go test ./cmd/note-stats`, updated ticket evidence, and committed the
  product implementation.
- Runtime containment still held. The later Engineer `max_turns` failure
  remained foundation telemetry, did not route through Orchestrator, and
  created no target intervention-debt ticket.

Residual findings:

- After acceptance evidence had passed and the product commit existed, Engineer
  did not move `T-001` to done or record disposition.
- It continued into packaging/build-output exploration, tried
  `go build -o bin/note-stats`, then wrapped the same repo-local build in a
  shell command. The shell-wrapped path created an untracked `bin/` artifact
  before blast-radius validation blocked the tool.
- The job hit `max_turns` with one done-ready product commit and an untracked
  generated build-output directory.

Decision: feature-ticket closure must happen before packaging exploration.
Once required acceptance evidence passes and the implementation is committed,
Engineer should update ticket evidence, move the selected ticket to done, and
record disposition before creating packaging, installer, release, or
repo-local build-output artifacts. `shell_exec` build-output policy now scans
shell command segments so `mkdir -p bin && go build -o bin/...` is blocked
before the binary enters the target repo.

### CLI Matrix Replay: Dogfood Findings Must Become Clean Handoffs

The next CLI replay used
`<validation-root>` to validate the
closure-before-packaging fix.

Positive evidence:

- CEO, COO, CTO, Engineer, QA, Security, and Dogfood ran against a fresh CLI
  target with zero intervention-debt ticket materialization.
- Engineer moved `T-001` to done and recorded disposition after proving the
  selected Note Stats behavior. No repo-local `bin/` packaging artifact was
  created.
- QA approved the completed ticket, and Security wrote and committed a bounded
  audit report.
- Dogfood found a real target-owned product gap: running the CLI without
  `--text` produced zero-count JSON instead of a required-argument error.

Residual findings:

- Dogfood created two uncommitted backlog tickets for the same missing-argument
  behavior and then continued validation.
- The run hit `max_turns`, leaving target-owned finding tickets untracked
  rather than committed and handed off for Engineer.
- Runtime containment held again: Dogfood `max_turns` remained foundation
  telemetry and did not route through Orchestrator.

Decision: Dogfood target findings are now a commit-and-handoff boundary. Once
`ticket_create` leaves a finding ticket uncommitted, Dogfood can inspect,
commit, push, and record disposition, but further shell validation and
additional `ticket_create` calls are blocked until the finding is committed.
Generated Dogfood guidance now says to stop additional validation after a
target-owned ticket is created, commit the finding, attempt push when possible,
and record terminal disposition before doing more work.

### CLI Matrix Replay: Review Gates Need Live Validation Evidence

The `demo-cli-run6` replay confirmed that the Dogfood finding handoff fix moved
the lifecycle forward instead of creating ticket churn. Dogfood created one
target-owned finding, committed it, and Orchestrator routed it back to Engineer.
Engineer fixed the Note Stats test mismatch and `go test ./...` passed.

The same run exposed a quality-gate escape:

- QA approved the original ticket without running the test suite despite test
  files being present.
- Security ran `go test ./...`, saw the failure, and still approved.
- Dogfood caught the failure later, meaning review approval was still prompt
  discipline rather than mechanical evidence.

Decision: QA and Security approval is now tied to in-job validation evidence.
The tool session records successful and failing shell validation commands. A
successful review disposition for a named ticket requires at least one
successful validation command, requires a successful test command when test
files are present, and is blocked after any failing build or test command in
the same job. The correct response to a failing build or test in review is
`changes_requested` with the command, output, and Engineer next action.
Expected non-zero runtime probes remain valid negative-path evidence when they
are documented and paired with positive validation plus passing tests.

Two supporting fixes came from the same run. Dogfood finding creation now
freezes further same-run validation until disposition, not only while the
ticket is uncommitted. `shell_exec` argv mode now accepts literal newline
arguments because argv does not invoke shell parsing. Evidence-required enabler
tickets can now move to done with evidence links and verifier metadata without
being converted into feature tickets.

### CLI Matrix Replay: Scenario IDs Must Match Feature Contracts

The `demo-cli-run7` replay deliberately used a fresh generic CLI target
(`note-stats`) rather than the earlier game demo. It confirmed that the first
bootstrap roles remained product-specific:

- CEO read `README.md`, updated goals for Note Stats, committed the decision,
  and handed off to COO.
- COO updated the active plan and `docs/features/F-001-product-walking-skeleton.md`
  for Note Stats, committed the planning artifacts, and handed off to CTO.
- No intervention-debt tickets were created. Guardrail blocks stayed as
  foundation telemetry.

The run then exposed a fresh planning-shape failure. COO wrote `F-002-S001`,
`F-002-S002`, and `F-002-S003` headings inside the existing `F-001` feature
contract. CTO copied those scenario IDs into `ticket_create`, which correctly
looked for `docs/features/F-002*.md` and blocked ticket creation. CTO repeated
the blocked ticket attempt and then spent turns rereading plan ranges rather
than routing a clean correction back to COO.

Evidence:

- Target repo:
  `<validation-root>`
- DB:
  `<validation-root>`
- Jobs: CEO completed, COO completed, CTO failed after operator stop.
- Telemetry: CEO guardrail blocks 4, COO guardrail block 1, CTO guardrail
  blocks 3, CTO `llm_unreachable` 1 from the manual stop.
- Target commits:
  `385959d CEO: Define first product slice and implementation approach for note-stats CLI`,
  `b7066c4 chore(learnings): update runtime learnings for ceo`, and
  `649e5f7 plan: update active scenario schedule and feature contract for note-stats CLI`.

Decision: feature contract writes now reject scenario headings whose feature ID
does not match the contract path. Generated COO guidance says scenario headings
inside `docs/features/F-001*.md` must use `F-001-SNNN`; generated CTO guidance
says mismatched scenario IDs are a COO feedback item rather than a ticketing
problem to brute force.

### CLI Matrix Replay: Engineer Must Converge After Evidence

The `demo-cli-run8` replay confirmed the scenario-ID fix in the live path. CTO
created `T-001`, and Engineer implemented a working Note Stats CLI with passing
run/build evidence. The failure shifted later: after the implementation commit,
Engineer kept running broad shell probes and `/tmp` listings instead of moving
the evidenced ticket to `done/` and recording the QA handoff. The job ended in
`context_overflow` around 41k prompt tokens.

Containment was healthy: the overflow stayed foundation telemetry, created no
target intervention-debt ticket, and did not dispatch Orchestrator into another
loop. The remaining issue was convergence after product progress.

Decision: once Engineer has successful validation and a successful
implementation commit while a product ticket remains in progress, exploratory
`shell_exec` calls are blocked. The tool error now points to ticket evidence,
`git mv ... docs/tickets/done/`, a lifecycle commit, and
`job_disposition_record`. The agent loop also prunes old assistant tool-call
arguments and older prose so long jobs do not overflow from historical
`file_write` payloads after tool outputs were already pruned.

### CLI Matrix Replay: Review Rework Must Reopen Tickets

The `demo-cli-run9` replay confirmed the post-validation convergence gate moved
the live lifecycle forward. Engineer moved `T-001` to done and recorded a QA
handoff instead of overflowing context. QA then correctly refused approval
because the implementation lacked authoritative post-implementation test
evidence. The runtime kept that policy block as foundation telemetry and
created no target intervention-debt ticket.

The rework path exposed two state-machine issues:

- the first version of the post-validation gate could fire while implementation
  files were still dirty because it counted a prior ticket-claim commit plus
  validation success as if implementation had been cleanly committed; and
- QA `changes_requested` routed Engineer back to the same ticket while the
  ticket still lived in `docs/tickets/done/`, allowing product/test edits while
  the repo-visible lifecycle still said the ticket was complete.

The second Engineer pass proved the review was real: it ran `go test ./...`,
found failing tests, repaired the test expectations, and got the test suite
passing before the operator stopped the canary to implement the source fix.

Decision: Engineer review rework now has to reopen done or in-review product
tickets into `docs/tickets/in-progress/` before product mutation or validation
shell commands. The post-validation shell gate now checks for a clean worktree
before blocking further shell calls, so legitimate validation while code or
ticket files are dirty remains available.

The follow-up `demo-cli-run10` replay confirmed the lifecycle reached product
planning, ticket creation, implementation, tests, and docsync without target
intervention-debt tickets. It also exposed the completion-path exception:
after Engineer moved `T-001` from `docs/tickets/in-progress/` to
`docs/tickets/done/`, the rework guard treated the pending done ticket as
already complete and blocked the final implementation commit until
`max_turns`. The guard now permits `git_commit` when the worktree contains an
in-progress-to-done ticket move, preserving the rework rule without blocking
ordinary completion.

The `demo-cli-run11` replay then found the next convergence problem one step
earlier. Engineer implemented and externally validated the Note Stats CLI but
repeated empty `shell_exec` no-op calls instead of committing dirty work and
closing the ticket. The generic no-op tool error was not enough to break the
loop. Repeated Engineer no-op calls after validation and dirty in-progress
ticket work now become a policy boundary that redirects to `git_status`,
commit, evidence, lifecycle completion, and terminal disposition.

The `demo-cli-run12` replay confirmed the lifecycle reached QA after Engineer
implemented the product slice, moved `T-001` to `docs/tickets/done/`, committed
the lifecycle, and recorded a QA handoff. The next contradiction was between
review policy and tool surface: QA approval required in-job validation, but the
generated QA manifest did not allow `shell_exec`, so QA could only request
changes. The rework pass also showed that `<validation-root>` binaries need
same-session freshness; otherwise an old canary binary can be reused as if it
proved the current work. Generated QA now gets bounded validation-only
`shell_exec`, and the shell session records external validation binaries built
with `go build -o <validation-root>` so later execution is trusted only when
the same role session built that path first.

The `demo-cli-run13` replay confirmed the QA tool-surface patch did not regress
the product-first path: CEO, COO, and CTO again reached product-specific
planning and ticket creation with no target intervention-debt tickets. Engineer
claimed `T-001`, implemented the Note Stats CLI, committed product code, and
proved the happy path with `go run cmd/note-stats/main.go --text "hello world"`.
The remaining failure was classification: direct runtime probes were not counted
as validation evidence, so the post-validation convergence gate never fired and
Engineer repeated empty `shell_exec` placeholders until `circle_detected` while
the ticket stayed in progress. Direct runtime commands that execute product
behavior now count as validation evidence, allowing the existing completion gate
to redirect post-commit shell drift toward ticket evidence, `docs/tickets/done/`,
and `job_disposition_record`.

The `demo-cli-run14` replay confirmed that Engineer-side direct runtime
validation now closes the ticket lifecycle: Engineer implemented Note Stats,
ran product probes and tests, moved `T-001` to `docs/tickets/done/`, committed
the lifecycle, and handed off to QA. QA then exposed a review-evidence
classification edge. It ran a fresh `<validation-root>` binary,
positive CLI probes, an expected missing-argument rejection, and `go test`.
The expected rejection exited non-zero, so approval policy treated it like a
failed validation command and blocked approval until `max_turns`. Review
approval now distinguishes failing builds/tests from expected negative runtime
probes: failed builds or tests still block, while documented non-zero runtime
probes can support invalid-input evidence when paired with positive proof and
passing tests.

The `demo-cli-run15` replay showed AD-169 was not yet live-reachable because
Engineer hit a ticket-completion loop first. The role implemented Note Stats,
committed source, fixed an unused-import build failure, built a fresh external
validation binary, and ran a successful runtime probe. The post-validation
completion gate blocked further shell work exactly as intended, but the model
kept retrying empty `shell_exec` placeholders instead of leaving the shell path
to update ticket evidence. The policy message and generated Engineer guidance
now make the non-shell next step explicit: use `file_read`/`file_write` on the
in-progress ticket to fill evidence, then only use `shell_exec` for the exact
`git mv` lifecycle move to `docs/tickets/done/`.

The `demo-cli-run16` replay confirmed that non-shell completion guidance moved
Engineer past the prior loop: the role updated ticket evidence, moved `T-001`
to done, committed the lifecycle move, and reached QA. It then exposed the
review side of expected-runtime-error handling. QA observed
`<validation-root> --text ""` returning `error: --text flag is
required`, even though the brief and ticket required empty text to produce zero
counts. The generated tests had encoded that bug as expected behavior, so tests
passed and QA approved despite contradictory runtime evidence. Decision:
`shell_exec` now has explicit `expected_exit_code` support for intentional
negative-path probes, and QA/Security approval blocks any unexpected failing
validation command. Expected non-zero runtime probes remain valid only when the
tool call declares the expected code. Cleanup of external `<validation-root>`
binaries is also allowed after ticket completion without forcing Engineer to
reopen a finished ticket.

The `demo-cli-run17` replay confirmed QA no longer rubber-stamps the empty-text
runtime contradiction: the target behavior was corrected before review, and QA
verified both positive and missing-argument behavior. The next failure moved to
the structured-exit boundary. QA ran `go test ./...` on its final turn, found a
failing multi-line count expectation, and then hit `max_turns` before recording
`changes_requested`. Engineer also used `<validation-root>` instead of the
freshness-tracked `<validation-root>` path. Decision: QA/Security now
stop further shell validation after any failing build, test, or unexpected
runtime probe and record a structured rework handoff; dispatch jobs get one
final terminal-tool prompt at the turn-budget edge; and external Go validation
builds must use `/tmp/<project>-validation` so stale temp artifacts are blocked
consistently.

The `demo-cli-run18` replay confirmed AD-172 changed the live loop. QA found
validation failures, received the terminal-tool grace prompt, and recorded
`changes_requested` rather than ending as `max_turns`. Orchestrator routed the
work back to Engineer, and Engineer reopened `T-001` from done to
in-progress before rework. The run also confirmed the `<validation-root>`
freshness guard across role sessions: a rework Engineer could not execute the
old validation binary until it rebuilt that exact path in the current job.

Run 18 exposed the next review-policy edge. QA intentionally checked the
missing-argument path but first ran `<validation-root>` without
`expected_exit_code`; the safe non-zero exit was classified as an unexpected
runtime failure. QA then tried to approve and was blocked, then tried to rerun
the same command with `expected_exit_code: 1` and was blocked by the shell-stop
rule. Decision: expected-negative runtime probes should declare
`expected_exit_code` on the first attempt, but a reviewer may correct the exact
same runtime command once with a matching expected non-zero exit code before
any other shell validation. Build/test failures and different runtime commands
still require structured rework.

The `demo-cli-run19` replay confirmed the product-first path still reaches
implementation in a fresh non-game target. CEO, COO, and CTO produced
product-specific goals, plan, feature contract, and a Note Stats implementation
ticket. Engineer claimed `T-001`, implemented the CLI, committed source,
updated ticket evidence, moved the ticket to done, and committed the lifecycle
move. No target intervention-debt tickets were created.

Run 19 exposed the next generic evidence-integrity failure. Engineer had
observed `<validation-root> --text ""` fail with
`error: --text flag is required`, but later marked the empty-text acceptance
criteria complete and moved the ticket to `docs/tickets/done/`. Manual replay
of the committed target confirmed the product still failed the contract.
Engineer also spent the terminal grace turn on a lifecycle command instead of
the required `job_disposition_record`, so the side effect happened and the job
still ended as `max_turns`. Decision: current-job unexpected runtime validation
failures now remain outstanding until the exact command passes or is corrected
with matching `expected_exit_code`; Engineer cannot move/write/commit product
tickets to done or record successful disposition while such failures remain.
The terminal-tool grace turn now rejects non-terminal tool calls without
executing them.

The `demo-cli-run20` replay confirmed that AD-174 blocks the bad completion
path. Engineer again observed the empty-text runtime failure, attempted to move
`T-001` to done, and policy rejected the lifecycle move before the ticket could
claim completion. The target stayed product-first and did not materialize
intervention-debt tickets. The next problem was narrower: Engineer tried to
rerun the same failed acceptance path with `expected_exit_code: 1`, creating an
expected-exit loop instead of repairing the product behavior. Decision: the
one-time expected-exit correction is review-only for QA/Security mistakes.
Engineer can use `expected_exit_code` only when the error-path probe is
declared on the first run; after an unexpected runtime failure, Engineer must
make the exact command pass to clear the blocker.

The `demo-cli-run21` replay confirmed AD-175 closed the expected-exit bypass
and kept the target backlog clean. CEO/COO/CTO reached product-specific Note
Stats planning and ticketing, Engineer claimed `T-001`, wrote target code, and
proved the happy path. The next failure was convergence: after
`go run ./cmd/note-stats --text ""` failed with the old required-text behavior,
Engineer repeated runtime probes until `circle_detected` instead of editing the
implementation. Decision: Engineer runtime validation failures now require a
post-failure implementation edit before runtime probes can continue, and the
exact failed command must later pass to clear the blocker.

The `demo-cli-run22` replay confirmed AD-176 changed Engineer behavior: the
unchanged empty-text rerun was blocked, Engineer edited the implementation, and
`<validation-root> --text ""` exited successfully. The next generic
edge was expected negative-path validation. Engineer ran
`<validation-root>` without `expected_exit_code`; the command should
fail for the missing-argument contract, but policy treated it as an unresolved
unexpected runtime failure and blocked later probes. Decision: Engineer may
correct an obvious missing-argument runtime probe by rerunning the exact command
once with matching `expected_exit_code`, while positive acceptance failures
with supplied input still require implementation rework.

The `demo-cli-run23` replay confirmed the missing-argument correction path, then
exposed a process-status blind spot. The target validation binary returned exit
code zero for `<validation-root> --text ""` while printing
`error: --text flag is required` and usage text to stderr. Policy counted the
probe as successful runtime evidence even though the product-visible behavior
still failed the empty-text acceptance path. Decision: direct runtime validation
now treats conservative error-shaped stderr as failed evidence, and only a later
clean exact rerun repairs the blocker.

The `demo-cli-run24` replay confirmed the product-first bootstrap path again:
CEO, COO, and CTO produced a Note Stats plan, feature contract, and `T-001`
implementation ticket without target intervention-debt tickets. Engineer
claimed and committed `T-001`, then repeated empty `shell_exec` placeholders
before creating product files and ended as `circle_detected`. Decision:
Engineer repeated no-op shell calls after ticket claim but before validation now
route to reading the ticket and feature contract, then `file_write`
implementation or a blocked disposition.

The `demo-cli-run25` replay confirmed the pre-implementation no-op fix:
Engineer claimed the ticket, wrote product files, and ran runtime probes. The
remaining failure was guidance clarity. After Engineer ran the missing-argument
probe without `expected_exit_code`, policy blocked completion but did not
clearly name the allowed exact-command `expected_exit_code` correction, so the
role kept trying other probes and completion. Decision: unresolved runtime
failure messages now explicitly name the missing-required-input correction path
while preserving strict repair for positive acceptance failures.

The `demo-cli-run26` replay confirmed the bootstrap loop remained
product-first through CEO and COO, but exposed false ticket progress. CTO called
`ticket_create` with `bdd_scenarios` as a quoted list string, repeated the same
parse failure, attempted to write a ticket file directly, then recorded a
successful disposition even though no implementation ticket existed. The
dispatcher re-entered CTO instead of moving product work forward. Decision:
ticket-creation failures now remain unresolved session state until a later
successful `ticket_create` occurs, and successful dispositions are blocked with
explicit JSON-array guidance for `bdd_scenarios`.

The `demo-cli-run27` replay confirmed the AD-181 ticket-creation fix in a fresh
target: CTO produced a valid product ticket and Engineer began implementation.
The next convergence failure was narrower. Engineer ran the missing-argument
validation binary without `expected_exit_code`, then continued with adjacent
work instead of immediately correcting that exact negative-path probe. Decision:
missing-argument runtime failures now store the exact failed `shell_exec`
command and exact `expected_exit_code` correction in session state, and block
unrelated Engineer mutations until the correction runs or the role records an
honest blocked disposition.

The `demo-cli-run28` replay confirmed fresh ticket creation and Engineer claim,
then exposed stale validation artifact reuse after positive acceptance failure.
Engineer built `<validation-root>`, saw the empty-text acceptance
probe fail, edited `main.go`, and reran the same external binary without
rebuilding it. The stale binary still reflected the pre-edit source and the job
ended as `circle_detected`. Decision: external `<validation-root>` artifacts
now record the runtime-edit watermark at build time and must be rebuilt after
post-failure implementation edits before runtime evidence from that artifact is
trusted.

The `demo-cli-run29` replay confirmed fresh product-specific bootstrap again:
CEO, COO, and CTO produced the Note Stats plan, feature contract, and ordinary
product ticket, and Engineer began implementation. The next failure was ticket
evidence outrunning validation. Engineer wrote in-progress ticket
`evidence_links` and `verified_by` before any successful validation command had
run in that job, then stalled in no-op placeholders. Decision: Engineer can no
longer populate in-progress ticket evidence fields until the same job has at
least one successful validation signal, such as a test, build, or direct
runtime probe.

The `demo-cli-run30` replay confirmed that evidence-first rule. Engineer ran
validation before updating ticket evidence, fixed the target, moved `T-001` to
done, and handed off to QA. QA then hit the same-session `<validation-root>`
freshness guard and stalled instead of rebuilding the binary in the review job.
Decision: external validation artifact freshness errors now name the exact
`shell_exec argv ["go","build","-o","/tmp/<project>-validation","."]`
correction when appropriate, and QA/Security guidance says to run the exact
tool-error correction before rerunning the binary.

The `demo-cli-run31` replay confirmed product-first delivery through a real
implementation and QA handoff, but showed that build correction and semantic
evidence still needed tightening. Engineer closed `T-001` even though the
empty-text acceptance probe returned `{"words":0,"lines":1,"characters":0}`
instead of the contracted zero-line output, and QA guessed an invalid root
build correction after `go build ./cmd/note-stats` was blocked for repo-local
output. Decision: Go build-output guardrails now emit exact corrected
`shell_exec argv` commands that preserve package targets such as
`./cmd/<name>`, and generated role guidance requires automated assertions for
explicit expected-output examples instead of relying on exit-code-only smoke
evidence. QA approval is mechanically blocked for Go source changes when no
`_test.go` files exist.

The `demo-cli-run32` replay confirmed that CTO created a product ticket and
Engineer wrote Go tests before trying to complete the Note Stats CLI. It also
found the next containment loop: after a missing-input runtime probe panicked,
the harness required the exact `expected_exit_code` repro, but when that repro
still failed it continued blocking `file_write` and sent Engineer back to the
same command. The run also showed target/foundation naming drift: the ticket
named `cmd/mars-harness/main.go`, and Engineer initialized `module
mars-harness` inside a Note Stats target. Decision: missing-input correction
attempts now unlock implementation edits while still blocking completion until
the exact runtime failure is repaired, and generated CTO/Engineer guidance
requires target-derived module, command, and binary names rather than
foundation defaults.

The `demo-cli-run33` replay confirmed the target-naming fix: CTO created
`T-001` with `cmd/note-stats/main.go`, and Engineer initialized `module
note-stats`. It also confirmed missing-input repair edits were allowed after
runtime failure. The next fault was in session accounting: repeated failures
of the exact same runtime command left multiple outstanding counters, and one
later process-successful exact rerun cleared only one, so policy still blocked
progress and the job ended as `circle_detected`. Decision: exact runtime repair
now clears all unmatched failures for that same command fingerprint in the
current job.

The `demo-cli-run34` replay confirmed that repeated exact runtime failures now
clear correctly: Engineer fixed the empty-text path, reran the exact command,
corrected the omitted-flag negative path with `expected_exit_code: 1`, added Go
tests, passed docsync, moved `T-001` to done, and reached QA without target
intervention-debt tickets. The next faults were review and traceability
boundaries. QA ran `go mod init` during review after shallow grep missed the
cmd-layout source, then ran an intentional no-argument error-path probe without
`expected_exit_code`, causing false `changes_requested`. Orchestrator then
treated sample player-movement prose from `docs/tickets/README.md` as live
ticket state, and the product implementation was bundled into the ticket done
move commit. Decision: QA/Security shell access is now validation-only, ticket
done moves require non-ticket product changes to be committed first, and
Orchestrator guidance treats ticket README prose as examples rather than live
backlog.

The `demo-cli-run35` replay confirmed the ticket-closure half of AD-189.
Engineer committed product source, README, and `go.mod` in a separate
`feat(cli)` commit before moving `T-001` to done in a lifecycle-only commit.
QA then read the done ticket and source, passed `docsync_audit`, built a fresh
`<validation-root>` binary, and validated the happy and empty-string
paths. The next fault was review completion: after passing evidence, QA
repeated empty `shell_exec` placeholders until `circle_detected`. The failure
was contained as foundation-owned telemetry and did not create target
intervention debt or route Orchestrator. Decision: required terminal-tool jobs
now get one circle-grace reminder, reviewer no-op placeholders after successful
validation route directly to `job_disposition_record`, and policy-blocked
no-op shell calls are counted as no-op failures for loop telemetry.

The `demo-cli-run36` replay confirmed product-first planning and ticketing
again, and confirmed unresolved acceptance failures no longer create target
intervention debt or dispatch Orchestrator. It exposed a narrower progress
integrity gap: Engineer saw the empty-string acceptance probe fail, but still
used shell-wrapper probes, unrelated validation, ticket evidence edits, and an
implementation commit before the exact failed command passed. Ticket completion
and successful disposition were blocked, but the bad source state was
committed. Decision: while Engineer has an unresolved positive runtime
acceptance failure, shell execution is limited to rebuilding the same stale
validation artifact or rerunning the exact failed command after source edits,
and product commits are blocked until that runtime failure is repaired.

The `demo-temp-run37` replay used a different Temperature JSON CLI target to
avoid overfitting the factory to the Note Stats canary. The lifecycle reached a
real product implementation, exact runtime evidence for `--celsius 0`,
`--celsius 100`, and omitted `--celsius`, product/test commit, evidence update,
and lifecycle-only done-ticket commit. It confirmed the runtime repair lane on
a non-Note-Stats target: after the omitted-flag probe first failed without
`expected_exit_code`, unrelated runtime probes were blocked until Engineer
reran that exact command with the expected non-zero exit. The next fault was a
false session blocker. Engineer's earlier pre-validation ticket evidence write
was correctly blocked, but that failed `file_write` was counted as unresolved
ticket-creation debt and later blocked a valid successful disposition after
the ticket was done. Decision: ticket-creation failure accounting now ignores
Engineer ticket evidence update failures while preserving false-progress
blocks for failed `ticket_create` and non-Engineer ticket-file bypass attempts.

The `demo-temp-run38` replay confirmed that product-first delivery can now
reach implementation and done-ticket closure on the alternate Temperature JSON
CLI target. It exposed two generic coordination faults. COO treated ticket
creation as something to recover through direct file writes or CLI indirection
despite not owning that step, spending extra turns before CTO created the real
ticket. QA then built and ran useful validation but alternated empty and
placeholder shell calls until `circle_detected`, and did not convert missing
durable Go tests into structured rework. Decision: non-ticket-owning planning
roles must hand off `ticket_breakdown` to CTO and are allowed to do so even
after unowned ticket-creation policy blocks; reviewer no-op recovery after
successful validation becomes terminal-only; and Go source with no `_test.go`
files routes QA to `changes_requested` rather than approval.

The `demo-temp-run39` replay confirmed the planning half of that fix: COO
updated product-specific planning artifacts and handed directly to CTO, and
Engineer created durable Go tests for the alternate CLI target. The next fault
was validation integrity rather than planning. Engineer observed a failing
`go test` command, proved a few runtime probes with a narrower `go run
main.go` path, attempted forbidden file deletion, and committed product work
while the authoritative test command still failed. Decision: Engineer failing
test/build evidence now creates its own repair lane. Runtime probes, unrelated
shell work, ticket evidence, ticket completion, successful disposition, and
product commits stay blocked until source or tests are edited and the exact
failing test/build command passes.

The `demo-temp-run40` replay confirmed the early planning path but exposed a
role-boundary fault before Engineer could validate AD-194. CTO created the
implementation ticket, then wrote `go.mod`, attempted source/test files, and
updated README usage notes. DocSync blocked some source writes, but package and
README product edits still escaped the technical-planning role. Decision: CTO
file writes are limited to bounded technical planning artifacts and product
implementation files now belong strictly behind `ticket_create` and
ticket-backed Engineer delivery.

The `demo-temp-run41` replay confirmed that CTO no longer mutates product
files before Engineer. Engineer claimed the ticket, wrote implementation and
tests, and hit a failing `go test`. AD-194 successfully blocked runtime probes,
cleanup, evidence updates, and product commits while the failing test/build
lane remained unresolved, but the exact-command requirement overconstrained
the repair path: focused same-lane test commands such as
`go test ./cmd/temperature-json-cli` were blocked after
`go test ./cmd/temperature-json-cli/...` failed. The role then tried workaround
behavior, including a root verification script. Decision: test/build repair
lanes are same-lane rather than exact-command-only, allow only bounded repair
writes before validation, and block ad hoc root scratch verification files.

The `demo-temp-run42` replay confirmed the repaired CTO path and showed that
AD-196 kept the Engineer in the failing-test lane without allowing runtime or
commit bypasses. The remaining fault was classifier friction: the local model
naturally tried `cd cmd/temperature-json-cli && go test -v .` after adding a
nested Go module, but repair-lane recognition discarded shell commands with
control syntax and blocked the focused test as a side path. Decision: the
narrow `cd <dir> && <test-or-build command>` shell shape now counts as
same-lane validation for repair purposes, while arbitrary shell wrappers remain
blocked.

The `demo-temp-run43` replay confirmed the product path through Engineer and
QA on the alternate Temperature JSON CLI target. Engineer corrected an
expected missing-input runtime probe, repaired a failing `go test`, committed
product work, updated evidence, moved the ticket to done, and QA approved with
read, docsync, test, and runtime evidence. Security then gathered enough clean
review evidence but spent more than five minutes in the next model turn instead
of recording the terminal disposition. Decision: clean QA/Security review
evidence now becomes a mechanical terminal-only boundary. After read plus
successful validation with no outstanding validation failure, the loop tells
the reviewer to call `job_disposition_record`, rejects any other next tool or
prose-only answer, and bounds that final grace completion.

The `demo-temp-run44` replay confirmed product-first planning and Engineer
delivery again, but QA exposed a false review rework path. It ran
`go build -o <validation-root> cmd/temperature-json-cli`,
which Go rejected because repo-local package paths need `./cmd/...`. QA then
tried corrected build commands, but the review policy had already counted the
first command as a product build failure and forced `changes_requested` back to
Engineer. Decision: QA/Security validation-procedure failures for obvious Go
package-target mistakes are tracked separately from target failures so a
reviewer can correct the command and continue validation without falsely
claiming implementation rework.

The `demo-temp-run45` replay confirmed the same product-first bootstrap path
through CTO-weekly ticket creation and Engineer ticket claim, then exposed a
validation-interface mismatch before QA. Engineer created a nested Go module
under `cmd/temperature-json-cli` and attempted the safe focused validation as
argv tokens: `["cd","cmd/temperature-json-cli","&&","go","test","./..."]`.
`shell_exec` rejected shell syntax in argv mode, after which the model tried a
root `go test ./cmd/...` command that Go rejected because of the nested module.
Decision: the tool now normalizes only the narrow `cd <dir> &&
<test-or-build>` argv pattern into the existing shell-command validation path;
arbitrary shell syntax remains blocked.

The `demo-temp-run46` replay confirmed AD-196's repo-local artifact correction
and AD-200's earlier path was no longer the active blocker. Engineer built the
Temperature JSON CLI to `<validation-root>` after the
guardrail rejected a repo-local binary, then proved correct JSON output for
`--celsius 0` and `--celsius 100`. The next validation command ran the binary
with no arguments. That is valid product evidence for this CLI because the
brief requires a required Celsius input, and the binary returned
`--celsius flag is required`; however, the policy treated the non-zero exit as
an unexpected runtime failure because `expected_exit_code` was absent. The
agent then tried unrelated probes, edits, commits, ticket moves, and terminal
dispositions, all correctly blocked by the unresolved runtime guardrail.
Decision: obvious missing-required-input CLI probes with clear required/usage
output and no crash markers now count as expected negative-path validation on
the first run, while panic/traceback/runtime-error output and positive
acceptance failures still open the strict repair lane.

The `demo-temp-run47` replay confirmed the missing-input part of that
decision: Engineer ran `go run cmd/temperature-json-cli/main.go` with no
arguments, received clear missing-argument usage text, and the job continued.
It then exposed the matching invalid-input edge. Engineer ran
`go run cmd/temperature-json-cli/main.go invalid`; the CLI correctly rejected
the value with `Must be a number`, but the runtime guardrail still classified
the non-zero exit as unexpected because the first AD-201 classifier only
covered missing input. Decision: the expected negative-path classifier now
covers deliberate invalid-input probes as well, requiring both an obvious bad
argument such as `invalid` and normal input-validation output, while numeric
positive inputs rejected as invalid still remain unexpected failures.

The `demo-temp-run48` replay confirmed AD-201 end to end in the live
lifecycle. Engineer implemented the Temperature JSON CLI, built the
same-session `<validation-root>` binary, proved positive
Celsius conversions, proved missing-input behavior, proved invalid-input
behavior, committed product work, moved `T-001` to done, and handed off to
QA. QA requested rework because no `_test.go` file accompanied the runtime
evidence, and the orchestrated Engineer rework then hit a command-procedure
trap: `go build -o <validation-root>
cmd/temperature-json-cli/` missed the required `./` package prefix. The
corrected command was obvious, but Engineer policy treated the first command
as a real build failure and blocked corrected validation until a source edit
occurred. Decision: the validation-procedure classification now applies to
Engineer as well as QA/Security, so recognizable Go package-target mistakes
stay out of the product repair lane while real compile, test, and runtime
failures remain blocking.

The `demo-temp-run49` replay moved one step further into the same alternate
CLI target. CEO, COO, and CTO-weekly produced product-specific planning and a
single product ticket. Engineer claimed the ticket, wrote the Go CLI with
DocSync metadata, corrected a repo-local build-output guardrail by using the
external `<validation-root>` path, proved the positive
`25` Celsius conversion, and proved the missing-input negative path. The next
probe, `<validation-root> 25 30`, correctly returned
`error: too many arguments provided`, but the runtime guardrail classified the
surplus-argument rejection as unexpected because AD-201 only recognized
missing input and obviously invalid single values. Decision: surplus-argument
CLI probes now count as expected negative-path evidence when the command has
more than one product argument, output clearly reports too many or surplus
arguments, and no crash markers are present.

The `demo-temp-run50` replay did not exercise AD-203 because Engineer used
explicit `expected_exit_code` for its missing-input and invalid-input probes.
It did expose the next generic repair-lane gap. Engineer added Go tests after
valid runtime evidence, but those tests failed with duplicate helper/type
definitions. The test/build repair policy correctly blocked runtime probes,
build-command switching, ticket moves, and unrelated shell work while the test
failure was unresolved. However, the only available deletion path for the bad
same-job test files was `rm`, and that was blocked too; the role responded by
creating more duplicate test files. Decision: while an Engineer test/build
failure is outstanding, non-recursive `rm` or `unlink` may remove only
test-like files written by the same job after the failure began. Unmarked
tests, product source, recursive cleanup, and old files remain blocked.

The `demo-temp-run51` replay confirmed product-first delivery through
Engineer: planning, ticket creation, implementation, tests, runtime evidence,
ticket closure, and QA handoff all happened without target intervention-debt
pollution. QA then corrected the familiar Go package-target validation
procedure mistake from `cmd/temperature-json-cli` to
`./cmd/temperature-json-cli`; the corrected build passed. The terminal-review
boundary correctly detected sufficient evidence, but the next model response
missed `job_disposition_record` and the job ended as `circle_detected`.
Decision: the first non-terminal response after a clean review-evidence
reminder is now rejected in-band and not executed, with one stronger
terminal-only correction before repeated misses fail.

The `demo-temp-run52` replay then took a different product path: CEO handed
directly to CTO-weekly, CTO created the product ticket, and Engineer wrote a
Go module under `cmd/temperature-json-cli`. A failing
`go test ./cmd/temperature-json-cli/...` correctly opened the test/build
repair lane and blocked runtime probes, destructive cleanup, commits, and
ticket moves. The remaining gap was repair scope: `file_write` allowed a
parallel root `main.go` and `main_test.go`, so Engineer started validating a
second implementation instead of repairing the failed package. Decision:
narrow Go package test/build failures now record their package repair scope,
and source/test/fixture writes outside that scope are blocked until the lane is
repaired.

The `demo-temp-run53` replay validated that scoped repair works in the live
CLI path: Engineer fixed the failing package test in
`cmd/temperature-json-cli/`, reached QA, and the review chain continued through
Security, Dogfood, and Release Manager. Release Manager generated local target
release notes but then created `v0.2.0` at the previous Dogfood commit while
`VERSION` and `CHANGELOG.md` were still dirty. The disposition guard forced a
`release: notes 0.2.0` commit before the release-blocked stop, but the stale
local tag showed that release tag placement needs code enforcement. Decision:
release tag creation now requires a clean worktree, matching `VERSION`, a
release-note `HEAD`, and any explicit tag target resolving to that `HEAD`;
`git_release_guard` reports stale version tags that point elsewhere.

The `demo-temp-run54` replay validated that release tag placement fix in the
same live target shape. Release Manager generated notes, committed
`release: notes 0.2.0`, and only then created `v0.2.0`, which pointed at the
release-note commit. The expected no-remote publication failure stopped the
dispatch chain cleanly. The run also reproduced a separate evidence-quality
gap: Dogfood wrote `docs/reports/dogfood/2024-05-21-dogfood-validation.md`
during a 2026-05-21 run after generic shell date access was unavailable.
Decision: server jobs now inject a compact non-droppable `## RUN METADATA`
context section with the current date, timestamp, and timezone, and roles are
instructed to use that value for report paths, evidence dates, release entries,
and ticket timestamps instead of inferring dates from examples or model memory.

The `demo-temp-run55` replay confirmed `## RUN METADATA` was injected for CEO,
COO, CTO-weekly, and Engineer, but the run did not reach Dogfood date
validation. Engineer created real Temperature JSON CLI product files, then hit
an unresolved `go test ./cmd/temperature-json-cli -run TestTemperatureCLI`
failure caused by duplicate or placeholder generated tests. Policy correctly
blocked runtime probes, build switching, ticket evidence, commits, and a false
successful disposition, and the runtime failure stayed as foundation telemetry
without target intervention debt. The missing escape hatch was safe cleanup:
Engineer tried `rm -f cmd/temperature-json-cli/main_test.go`, but same-job test
removal only covered files written after the failing command. Decision: test
cleanup now tracks every successful Engineer `file_write`, allowing
non-recursive removal of duplicate/generated test files created or rewritten
earlier in the same job while continuing to block pre-existing tests and source
deletion.

The `demo-temp-run56` replay validated the AD-209 direction. The first
Engineer completed the product slice, QA requested ordinary test-coverage
rework, Orchestrator routed that feedback back to Engineer, and the second
Engineer added focused Go tests, passed `go test ./cmd/temperature-json-cli/`,
committed, moved `T-001` to done, and recorded a successful disposition. The
next QA job read the ticket, README, implementation, and tests, then ran the
same package test successfully. The remaining blocker was a foundation
terminal-boundary contradiction: QA doctrine required `docsync_audit` before
approval, but the runtime had already decided review evidence was sufficient
and rejected the attempted `docsync_audit`, eventually ending the job with
`circle_detected`. Decision: review terminal convergence now waits for
`docsync_audit` evidence before forcing `job_disposition_record`, while
successful disposition still enforces docsync mechanically as a final guard.

The `demo-temp-run57` replay confirmed that QA could run `docsync_audit`
before terminal convergence, but showed that the convergence heuristic was
still too broad. After docsync and a successful external `go build -o
<validation-root> ./cmd/temperature-json-cli`, the runtime
forced terminal disposition before QA ran the repository test command even
though `_test.go` files were present. QA attempted more validation, received
the stronger terminal-only correction, and ended with `circle_detected`.
Decision: review terminal convergence now waits for a successful test command
when test files exist, matching the approval policy and preventing build-only
evidence from preempting required tests.

The `demo-temp-run58` replay confirmed that the direct convergence heuristic
waited for tests, but exposed the no-op recovery path as a separate stale
terminal trigger. QA built `<validation-root>`, then called
`shell_exec` with an empty argv. The no-op guard told QA to approve because
some validation had succeeded, while the disposition policy correctly rejected
approval because tests had not passed. Decision: blocked review no-op failures
no longer set terminal-only disposition state by themselves; the agent loop
uses the same evidence-aware gate as approval, and no-op guidance points to
missing tests or docsync before approval guidance appears.

The `demo-temp-run59` replay validated the review fixes end to end. A clean
Temperature JSON CLI target advanced through product-specific planning,
ordinary ticket creation, Engineer implementation, QA approval, Security
approval, Dogfood approval, and local release-note/tag creation without
intervention-debt starvation. QA ran build, runtime probes, `go test`, and
`docsync_audit` before approval. The remaining release finding was command
resolution: Release Manager first invoked `mars-harness release notes` through
`shell_exec`, which hit a stale installed binary with no `release` command,
then repeated that failing command after reading the `mars_harness_cli`
reference. Orchestrator recovered by dispatching Release Manager again, and the
second pass produced `release: notes 0.2.0`, tag `v0.2.0`, and a clean
missing-remote publication blocker. Decision: agent jobs now block direct
`shell_exec mars-harness ...` invocations and route Mars Harness CLI workflows
through `mars_harness_cli`, whose resolver prefers the active harness binary.

The `demo-temp-run60` replay switched to a Word Count JSON CLI target to avoid
overfitting the loop to game or temperature-conversion examples. The lifecycle
again reached product-specific planning, one ordinary ticket, Engineer
implementation, QA-requested test rework, Orchestrator-to-Engineer routing,
QA approval, Security approval, Dogfood approval, and local release review.
Release Manager used `mars_harness_cli` for release notes on the first pass,
committed `release: notes 0.2.0`, created tag `v0.2.0`, and stopped only on
the real missing-remote publication blocker. The new finding came from the
retry setup: a sandboxed start registered the repo and enqueued CEO before bind
failure, then the retry removed SQLite `-wal`/`-shm` sidecars and registered
the same target under a new repo ID because the uncheckpointed state was gone.
Decision: automatic startup cleanup now preserves SQLite sidecars and lets
SQLite recover or checkpoint them instead of deleting queue or repo registry
state.

The `demo-slug-run61` replay switched to a Slugify JSON CLI target and
validated that SQLite sidecar preservation fixed the retry-after-bind-failure
case: the retry reused repo ID `3297728f-63d3-4a67-9262-f6535de8ff2a` and CEO
job `cc65a7ab-0fe7-4ee3-85cd-80479a63b1fc` instead of discarding WAL-backed
state. The product lifecycle again reached ordinary planning, ticket creation,
implementation, QA rework, and Orchestrator-to-Engineer routing without
intervention-debt tickets. The new finding was repair guidance: the rework
Engineer added `cmd/slugify-json/main_test.go`, `go test` exposed a real
contract mismatch (`countWords("Test@#$%Special Characters!") = 2, expected 3`),
and guardrails blocked unrelated commands but only repeated the unresolved
command, not the failing assertion. Decision: unresolved test/build repair
state now stores the latest failing output and repeats it in guardrail guidance,
including an instruction to edit implementation rather than weaken a
contract-matching test.

The `demo-slug-run62` replay reran the same Slugify JSON CLI brief with that
guidance in place. The lifecycle completed through CEO, COO, CTO-weekly,
Engineer, QA, Security, Dogfood, and Release Manager in a single local run.
Engineer implemented product code and tests, QA/Security/Dogfood approved with
runtime, `go test`, and `docsync_audit` evidence, Release Manager generated
`release: notes 0.2.0`, and the target ended clean with tag `v0.2.0`. The only
terminal blocker was the expected missing remote for GitHub publication in the
temporary target. Decision: the Run 61 repair-guidance gap is closed for the
fresh Slugify canary, and the next dogfood surface should broaden the canary
matrix with remote-backed release validation and non-CLI application shapes.

The `demo-7` replay switched to a static browser-game target: "Create Tetris
using Phaser JS." The lifecycle correctly initialized a fresh target, produced
product-specific CEO/COO/CTO planning, created one ordinary product ticket, and
committed a Phaser grid slice. It then failed in Engineer ticket closure:
validation used file listings and echo placeholders rather than a real static
HTTP smoke probe, policy blocked evidence writes and done-ticket moves, and
the job repeated forbidden disposition attempts until `max_turns`. Decision:
static HTTP probes now count as validation evidence, generated Engineer
guidance names the exact static smoke path, and `max_turns` following
ticket-lifecycle policy blocks enqueues one bounded `ticket_gate_repair`
instead of leaving the target stalled with an in-progress ticket.

The `demo-8` replay reran the Phaser Tetris target after the first static
validation fixes. The lifecycle progressed through product-specific planning,
ticket creation, implementation, QA rework, bounded ticket-gate repair, QA
approval, and Security review without creating intervention-debt tickets.
Security then found a runtime accessibility issue and Orchestrator correctly
selected Engineer rework, but dispatch stopped because the loop guard counted
earlier QA rework routes with the same ticket-state hash as the same loop.
Decision: the repeated-route guard now distinguishes later review-stage
`changes_requested` feedback from earlier review-stage rework history, allowing
one bounded Engineer rework for fresh Security or later-stage findings while
still stopping repeated same-reviewer routes without ticket-state progress.

The `demo-tetris-9` replay restarted the Phaser Tetris target with the newer
static guidance. The lifecycle again reached product-specific planning, one
ordinary ticket, Engineer implementation, and QA without target intervention
debt. Engineer produced and committed static `src/` files, moved `T-001` to
done, and handed off to QA. QA read the ticket, feature contract, README, and
implementation files, ran `docsync_audit`, then started a static HTTP server.
The runtime incorrectly treated background server startup as sufficient review
evidence before a separate `curl` probe ran, injected terminal-only approval
guidance, and QA ended with `circle_detected`. Manual browser inspection of
the served target showed the deeper product issue that QA should have routed
to Engineer: `src/main.js` constructed `new Phaser.Game(config)` inside the
scene `create()` callback and used `game.add.rectangle`, producing repeated
`TypeError: Cannot read properties of undefined (reading 'rectangle')` console
errors instead of a working Tetris slice. Decision: background server startup
is validation setup, not successful validation evidence. Static HTTP probes
still count when a separate `curl` command succeeds, and generated Engineer/QA
guidance now treats static curl as file-delivery proof only while requiring
framework lifecycle inspection for browser-game tickets, including Phaser's
single-game/scene-instance conventions.

The `demo-tetris-10` replay reran the static Phaser Tetris target after the
review-evidence fix. CEO, COO, and CTO-weekly again produced product-specific
planning and one ordinary product ticket. Engineer implemented and committed
`index.html`, `style.css`, and `main.js`, but the first static HTTP validation
failed with `curl: (52) Empty reply from server` immediately after starting
`python3 -m http.server 8080 --bind 127.0.0.1` through `shell_exec`
`background:true`. The job then used grep, echo, and file inspection as
substitute evidence, so policy correctly blocked ticket evidence writes,
ticket-done moves, and successful dispositions until `max_turns`. The bounded
ticket-gate repair reproduced the same evidence gap and was interrupted while
stopping the run. Decision: the root issue was in `shell_exec` background
process handling. The tool captured startup output through pipes, returned the
PID, then closed the read side of stdout/stderr when the tool call returned.
Python's HTTP server logs each request to stderr, so later request logging could
break the server pipe and produce empty HTTP replies. Background processes now
keep stdout/stderr drained until exit with capped capture buffers. Static
target guidance also tells Engineer to run `node --check main.js` before the
HTTP smoke and to use high validation ports 18081/18082 rather than defaulting
to commonly occupied 8080.

The `demo-tetris-11` replay reran the Phaser Tetris target after background
server output was fixed. CEO, COO, and CTO-weekly again created product-specific
planning and one ordinary product ticket, and Engineer implemented a Phaser
slice without intervention-debt tickets. Manual browser inspection of the
served target showed the product was still blank: `index.html` loaded Phaser
only from a CDN, `src/config.js` referenced `preload`, `create`, and `update`
callbacks that were not defined or imported in that module, and helper
functions in `src/main.js` used `this.add` while being called without a Phaser
scene binding. QA originally approved from static file delivery evidence, then
Security requested rework because server validation was flaky, and the rework
Engineer moved `T-001` back to done without changing product code. The second
QA pass read the files and ran `curl`, but the runtime injected terminal
approval guidance after static HTTP delivery, leaving QA in `circle_detected`
instead of a structured `changes_requested`. Decision: browser-framework
completion now requires deterministic package build evidence before Engineer
ticket evidence, done moves, or successful dispositions. QA/Security approval
is blocked when a browser-framework package has no build script, when a build
has not passed, or when static Phaser lifecycle inspection finds obvious
runtime defects. Terminal review convergence now routes those cases to
`changes_requested` rather than approving from `curl` alone.

The `demo-tetris-12` replay reran the Phaser Tetris target with the
browser-framework build gate installed. CEO and COO produced product-specific
planning and CTO created one ordinary ticket, but the ticket itself inherited a
foundation-shaped Go implementation: `cmd/phaser-tetris-demo/main.go`,
`cmd/phaser-tetris-demo/go.mod`, and a "Go CLI with web-based frontend"
guidance block for a brief that explicitly said "Phaser JS." Engineer followed
that ticket literally, producing a Go static-file server, a CDN-only Phaser HTML
page, and a Phaser-related Go module dependency before the run was stopped. The
observed failure was earlier than QA: the system let the wrong technical shape
become the authoritative target ticket. Decision: CTO ticket creation now blocks
Go CLI/module affected files for Phaser/JavaScript briefs unless the README
explicitly names a Go backend. Generated CTO guidance tells planners to derive
browser JavaScript targets as `package.json`, `index.html`, `src/*.js` or
`src/*.ts`, tests, and build config with `npm run build` evidence. Engineer file
writes also block Go scaffolding for Phaser-only briefs, and browser-framework
completion detects Phaser from README/vision or CDN script tags, not only from
an existing package manifest.

The `demo-tetris-13` replay reran the Phaser Tetris target after the target
shape guard. This time CTO created a JavaScript/Phaser-shaped ticket and
Engineer produced `package.json`, `index.html`, `src/main.js`, local Phaser
dependency metadata, and a clean target commit. The lifecycle progressed through
Engineer, QA, Security, Dogfood, and Release Manager without intervention-debt
ticket flooding. Two new quality gaps appeared. First, `package.json` satisfied
the build gate with `build: echo 'Building Phaser Tetris Demo...'`, so build
evidence could not actually fail when the browser game was broken. Second,
Dogfood wrote a positive E2E report from HTTP 200 reachability, syntax checks,
and file inspection even though the product was still a sample movable block
rather than complete Tetris. The run also reached release after one walking
skeleton ticket while the generated feature scenarios remained uncovered.
Decision: browser-framework evidence now rejects no-op build scripts and
requires product-state smoke evidence beyond HTTP reachability for QA,
Security, and Dogfood. Release-bound dispatch now routes Engineer while open
product tickets remain and routes CTO when generated target feature scenarios
are still uncovered by done tickets.

The `demo-tetris-14` replay reran the Phaser Tetris target after the no-op
build and release-coverage fixes. The lifecycle still progressed through CEO,
COO, CTO, and Engineer without intervention-debt flooding, and CTO kept the
browser JavaScript target shape. A deeper planning failure remained: COO moved
the Tetris capability list into Business Logic but left the Scenario Schedule
as three generic starter scenarios. CTO then created only
`T-001-implement-visible-playfield-grid-for-phaser-tetris-demo.md`, explicitly
out-of-scoping falling tetrominoes, movement, rotation, line clearing, scoring,
game over, and restart. Engineer implemented a playfield/grid slice with
`package.json`, `index.html`, and `src/main.js`, but used no-op build/validate
scripts and eventually failed with local-model context overflow
(`request (32865 tokens) exceeds the available context size (32768 tokens)`).
Decision: explicit product capabilities from README, active goals, or the
brief must become concrete generated feature scenarios or be deliberately
descoped before COO handoff or CTO ticket creation. This keeps one-ticket-at-a
time delivery compatible with complete product builds, because the future
product behavior remains visible in the scenario schedule instead of being lost
as "out of scope" in the first ticket.

The `demo-tetris-15` replay reran the Phaser Tetris target after capability
coverage was made mechanical. The run finally produced the right early shape:
CEO defined a Phaser Tetris slice, COO expanded `F-001` into scenarios for
playfield visibility, falling tetromino controls, line clearing and scoring,
game over and restart, and local browser execution, and CTO created an ordinary
product ticket instead of intervention debt. The first Engineer implementation
was broken, QA caught the Phaser integration defect, and Orchestrator routed
rework. The rework then exposed a remaining evidence-ordering issue: Engineer
changed `index.html` to module loading and added `src/main.js`, but local
modules did not export the names being imported and the Phaser scene callbacks
used wrapper `this` where Phaser scene context was required. `npm run build`
still passed because the package build was only a collection of `node --check`
syntax checks. Engineer moved the ticket to done, QA policy blocked approval
for missing browser-product smoke, then the review job exhausted turns while
trying to start the app. Decision: browser-framework completion must fail
before QA on syntax-only builds, broken local module exports, classic-script
module loading, and Phaser scene-context misuse. Engineer now needs successful
same-job build evidence plus product smoke/source-runtime proof before ticket
evidence, done moves, or successful disposition can proceed.

The `demo-tetris-16` replay reran the Phaser Tetris target after the
syntax-only build and module-graph gates were installed. The lifecycle again
made product-first progress with no intervention-debt flood: CEO established a
Tetris goal, COO wrote a product-specific active plan and feature contract, CTO
created an ordinary Phaser-shaped implementation ticket, and Engineer wrote and
committed `package.json`, `index.html`, and `src/*.js` product files. The
remaining failure moved from planning/ticketing into lifecycle continuation.
Engineer hit `max_turns` immediately after a final commit and push, leaving
`T-001` in progress and `package-lock.json` untracked rather than handing off
to QA. The implementation also showed the exact blank-browser class that HTTP
reachability misses: `index.html` loaded `src/main.js` as a classic script even
though it contained ES module imports, and `src/game.js` used exported
tetromino helpers without importing them. Decision: Engineer max-turns with an
ordinary in-progress product ticket now enqueues one bounded
`product_continuation` job instead of dead-stopping, and browser-framework
source inspection detects local exported symbols used without imports as well
as classic-script module entrypoints.

The `demo-tetris-17` replay verified the bounded continuation path and exposed
the next browser-delivery failure class. The first Engineer run reached
`max_turns` on an active product ticket and the harness correctly enqueued a
same-ticket `product_continuation` job. That continuation did not reach product
completion because the first run had recorded `node --check index.html` as an
unexpected runtime validation failure. Node's `--check` validates JavaScript
source, not HTML entrypoints, so the policy kept demanding an impossible clean
rerun before commits, ticket evidence, or successful disposition could proceed;
the repeated feedback eventually pushed the local model over the 32k context
limit. The same target also showed that invalid Phaser shape was arriving too
late: Engineer wrote a local `phaser` dependency without a real build script,
loaded Phaser from a CDN-only HTML script tag, and added repo-root scratch
validation files. Decision: `node --check` on `.html`/`.htm` is now blocked or
classified as validation-procedure failure instead of product runtime failure;
Phaser package and entrypoint writes fail immediately unless they use a local
npm dependency, deterministic build script, and non-CDN entrypoint shape; and
root scratch validation files named like `test-*` are blocked before they enter
the target repo.

The `demo-tetris-18` replay reran the Phaser Tetris target after the browser
validation-procedure and package-shape fixes. CEO completed cleanly, committed
product goals, and handed off to COO. COO then rewrote the generated
`F-001-product-walking-skeleton.md` into a Tetris-specific contract with
scenarios for visible playfield, falling tetrominoes, movement, rotation, line
clearing, scoring, game over, and restart. The remaining blocker was a false
positive in foundation policy: the starter-placeholder guard rejected the
product-specific contract because its Business Logic section reused the durable
BDD phrase "Product rules, workflow branches, state transitions..." from the
operating model. Decision: starter-placeholder detection should key on actual
scaffold markers such as "starter contract is seeded" and "replace placeholder
nouns", while product capability coverage separately proves that the scenario
schedule has become concrete before CTO ticketing.

The `demo-tetris-19` replay confirmed that the lifecycle now reaches real
product delivery: CEO and COO completed, CTO created an ordinary
`T-001-implement-line-clearing-mechanics-for-tetris-game.md` ticket, and
Engineer claimed the ticket and wrote `package.json`, `index.html`,
`vite.config.js`, and `src/*.js` Phaser/Vite product files. The remaining
failure was an Engineer repair loop after `npm run build` failed with
`ReferenceError: window is not defined` because the Vite config imported
Phaser/browser runtime code. The unresolved-build guard correctly blocked side
paths, but it repeated a long failing-output excerpt on each blocked retry and
the local model eventually exceeded its context window. Decision: Phaser Vite
config writes now reject imports of Phaser/browser runtime or local `src/*`
game modules, and unresolved build/test guardrail guidance includes only a
bounded compact failing-output excerpt so repeated repair-lane feedback does
not cause context overflow.

The `demo-tetris-20` replay confirmed the compact/config-safe patch moved the
Tetris lifecycle forward again: CEO, COO, and CTO completed; the generated
feature contract covered the full Tetris brief; CTO created ordinary product
tickets; and Engineer wrote a Phaser/Vite app plus integration tests. The next
failure was a repair-lane mismatch rather than a product-planning failure.
After `npm run test:integration` failed because Jest could not find
`jest-environment-jsdom`, the unresolved validation guard correctly blocked
side probes, ticket evidence, and switching validation lanes, but it also
blocked a focused edit to `tests/integration/playfield.test.js`. Decision:
test/build repair writes now recognize durable test files under `test/` or
`tests/` and conventional `*.test.*`, `*.spec.*`, or `_test.go` filenames as
same-lane repair surface, while root scratch probes and unrelated files remain
blocked until validation passes.

The `demo-tetris-21` replay showed the next failure class in the live build
loop. CEO, COO, and CTO completed quickly, CTO produced an ordinary
visible-playfield ticket, and Engineer wrote a Phaser implementation. The
target still did not complete because the build surface was weak and dirty:
Engineer used a copy-only build script (`mkdir -p dist && cp ... && echo ...`),
generated an unignored `dist/index.html`, and exhausted its turn budget trying
to satisfy browser-product smoke with static checks. The continuation job then
failed before model execution because workspace hygiene blocked dirty generated
output. Inspection also found `src/scenes/GameScene.js` extended
`Phaser.Scene` without importing Phaser, a module-graph defect that the static
browser inspection did not yet catch. Decision: copy-only/static-server build
scripts count as no-op browser validation, generated target `.gitignore`
includes common frontend output directories from init, and Phaser source
inspection blocks modules that use `Phaser.*` or `extends Phaser.Scene`
without importing or defining Phaser in that module.

The `demo-tetris-22` replay confirmed those fixes helped but exposed the next
completion bottleneck. Engineer now produced a real Phaser/Vite package,
`vite build` passed, generated output was ignored, and the missing Phaser import
was repaired. The job still failed twice at max turns because the
browser-product smoke requirement did not provide a concrete enough path. The
model tried repo-root scratch validation files and a `node -e` eval payload in
argv mode; scratch files should stay blocked, but argv validation should not
treat JavaScript eval code as shell syntax. The same run showed a planning
scope leak: the README required line clearing, scoring, game over, and restart,
while COO put those capabilities under `Out of Scope` and left `Descoped
Scenarios` as `None`. Decision: allow language eval code arguments such as
`node -e` in argv mode, name a concrete Phaser source/runtime smoke assertion
fallback, and block explicit brief requirements that are hidden under generic
`Out of Scope` text without descoping rationale.

The `demo-tetris-23` replay confirmed the planning-scope fix: COO produced a
feature contract with eight in-scope Tetris scenarios, and the first ticket
targeted only the visible playfield scenario while preserving later line
clearing, scoring, game over, and restart scenarios. Engineer created a real
Phaser/Vite package and committed the playfield implementation, but then looped
on ticket evidence/done/disposition without running the browser-product smoke.
The guardrail named `node -e` but did not provide a literal tool call, so the
model kept trying closure paths and even attempted another scratch file. The
same implementation also externalized `phaser` in `vite.config.js`, which can
let `vite build` pass while leaving the browser with an unresolved bare import.
Decision: Phaser smoke guardrails now include a copy-pastable `shell_exec argv
["node","-e", ...]` command, and Vite config writes/source inspection block
externalizing `phaser` from the production bundle.

The `demo-tetris-24` replay found the next lifecycle break after those fixes.
COO stayed product-specific, but advanced the active plan to keyboard controls
before the visible playfield scenario was evidenced; CTO created `T-001` for
`F-001-S002`, so the lifecycle skipped the earliest uncovered scenario. Engineer
then produced real Phaser files but used `python3 -m http.server 18081` as the
target app server, colliding with the local harness runtime/inference port
range, loaded Phaser from a nested CDN script tag, referenced global `Phaser`
without a module import, and constructed `new Phaser.Game` from inside a scene
callback. The run ended with `llm_unreachable` context overflow after 48 turns,
with no done ticket. Decision: feature ticket creation now requires the
earliest uncovered BDD scenario, Phaser package writes reject reserved
`18080`-`18089` app ports and static source-server runtime scripts, nested HTML
entrypoints cannot load CDN-only Phaser, and Phaser source writes reject
global/import gaps or recursive game construction before those defects become a
long Engineer loop.

The `demo-tetris-25` replay confirmed the first half of that fix. CTO first
tried to create a later keyboard-controls ticket, the runtime blocked it with
the earliest uncovered scenario, and CTO retried with `T-001` for the visible
playfield scenario. Engineer then produced a real Phaser/Vite package with a
local `phaser` dependency, Vite scripts on application ports, source under
`src/`, and one top-level game construction. The remaining failure was a
closure-policy mismatch: after the implementation commit, the post-validation
convergence gate blocked the missing `npm run build` and browser-product smoke
commands that browser-framework completion itself required. The replay also
showed that wrapped README capability lists could still leak requirements into
`Out of Scope` because single line breaks split one operator sentence into
fragments. Decision: single newlines in brief capability extraction now behave
as wrapped prose, and post-validation convergence allows only the missing
browser build or product-smoke evidence commands needed to finish the current
browser-framework ticket.

The `demo-tetris-26` replay exercised that fix against the same wrapped README
shape. COO preserved the core Tetris capabilities in the scenario schedule, but
then got trapped by the README's validation sentence: "include enough build or
smoke evidence to prove the game mounts and plays." The product-capability
guard treated that as implementation scope and kept blocking completed planning
even after COO added a build/smoke scenario. The run also showed that "keyboard
controls" with left/right/down/rotate keys is a fair representation of
"keyboard movement and rotation" even when the exact word "movement" is absent.
Decision: capability extraction now ignores validation/evidence fragments and
short proof tails such as "mounts" or "plays", while capability coverage treats
keyboard-control and directional-input language as movement coverage.

The follow-up `demo-tetris-27` replay cleared that validation-evidence loop and
got COO back to the first product scenario. The next loop was an out-of-scope
false positive: the feature contract had line-clearing and scoring scenarios,
but its Out of Scope section said "Advanced scoring or game modes beyond basic
line clearing." The guard matched the base words and kept claiming basic line
clearing and scoring were descoped, even though only advanced variants were
excluded. Decision: out-of-scope capability checks now inspect individual lines
and treat "advanced ... beyond basic ..." as an advanced-only qualifier, not a
descope of the basic capability already present in the scenario schedule.

The later `demo-tetris-28` replay got past planning and ticketing: CTO created
one ordinary `F-001-S001` product ticket, Engineer claimed it, and wrote real
Phaser/Vite package and source files. The write-time guards correctly blocked
bad Vite config imports, missing DocSync metadata, and a broken Phaser helper
using `this.add` without a scene. Engineer then ended as `circle_detected`,
leaving an active product ticket and dirty partial product files. Decision:
Engineer `circle_detected` with an ordinary in-progress product ticket now
enqueues one bounded `product_continuation` job, matching the existing
`max_turns` product-progress behavior while still preventing recursive
continuations, Orchestrator runtime loops, and intervention-debt tickets.

The `demo-tetris-29` replay proved the continuation-era path can ship the first
real slice: CEO/COO/CTO/Engineer completed, Engineer committed a Phaser/Vite
playfield implementation, build and smoke evidence ran, and `T-001` reached
`docs/tickets/done/`. The run then exposed two next bottlenecks. CTO created
only one product ticket despite a clear multi-scenario Tetris contract, so the
factory had no ready backlog for the rest of the game. QA then copied or
interpreted browser-smoke evidence as shell syntax, treated a helper/escaping
problem as a target defect, and requested Engineer rework while saying the
implementation was correct. Decision: fresh bootstrap ticketing may now seed a
small ordered scenario backlog batch, Phaser smoke guidance avoids
JSON-escaped regex evidence, and QA cannot route "the validation helper is
wrong but source is correct" feedback to target Engineer implementation rework
when browser-framework source inspection is clean.

The `demo-tetris-30` replay confirmed the first-slice recovery but exposed the
next product-build stall. CTO still handed off with only `F-001-S001` ticketed,
so later Tetris scenarios had no ready backlog. Engineer initially reached
`circle_detected` after useful partial product work, and the bounded
continuation recovered by committing the Phaser/Vite skeleton and moving
`T-001` to done. QA then requested Engineer rework because its localhost smoke
probe ran against a stopped dev server, despite successful build evidence and
clean source inspection. Decision: CTO handoff now requires the first small
scenario batch to be covered by ordinary tickets, and QA cannot route
dev-server/helper setup failures to target Engineer rework when browser build
and source checks are clean.

The `demo-tetris-31` replay verified that COO can now expand the Phaser Tetris
brief into a full product feature contract: visible grid, falling tetrominoes,
keyboard controls, line clearing, scoring, game over, and restart were all
scheduled as scenarios. It then exposed a capability-parser false positive:
strategy prose saying "all core Tetris mechanics: visible playfield grid..."
was treated as one literal requirement instead of a category label followed by
list items. Decision: capability extraction now strips generic category
prefixes before colon-delimited lists, preserving the individual product
capabilities without requiring a phantom "all core mechanics" scenario.

The `demo-tetris-32` replay confirmed that the category-prefix fix let the
pipeline reach product ticketing again. CEO and COO completed quickly, CTO
created a three-ticket Tetris batch, and Engineer shipped real Phaser/Vite
progress for `T-001` and `T-002`. The next bottleneck was ticket-boundary
discipline. CTO's third ticket duplicated the first two scenario IDs while
trying to satisfy the early-batch gate, and Engineer attempted to claim that
next ticket in the same job after moving `T-002` to `done/` but before
committing the lifecycle move or handing off to QA. The job then burned turns
against claim/evidence guardrails and failed with `max_turns` despite a valid
product implementation commit. Decision: feature ticket creation now rejects
already-covered BDD scenarios in new ordinary tickets, and Engineer sessions
record done-ticket moves so one completed product ticket must be committed,
pushed when possible, and handed off to QA before another ticket can be
claimed.

The `demo-tetris-33` replay verified both fixes under a clean target. CTO
created a compact batch with `T-001` for `F-001-S001` and `T-002` for
`F-001-S002`/`F-001-S003`, without duplicating covered scenario IDs. Engineer
claimed `T-001` and produced a real Phaser/Vite app skeleton. The new loop was
smaller and more concrete: after a browser product-smoke assertion failed,
Engineer tried to create `validate-phaser.js` to inspect source strings, but
the product-source Phaser guard treated the helper's probe text as shipped app
code. Decision: browser-framework validation helper paths are now excluded
from product-source lifecycle findings while `src/`, entrypoints, package
scripts, and Vite config remain guarded.

The `demo-tetris-34` replay showed the next product-start bottleneck. COO
collapsed the requested Tetris mechanics into one broad "user can run or
inspect the first product behavior" scenario, so CTO could only create one
large ticket instead of a useful ordered product backlog. After that ticket was
created, the CTO handoff gate demanded `F-001-S003`, but that scenario was
about product evidence staying ahead of governance and intervention debt, not a
separate implementation slice. CTO then looped through blocked ticket creation
and disposition attempts instead of handing the product ticket to Engineer.
Decision: explicit product capabilities must be visible in scenario schedule
entries or scenario headings, and CTO handoff counts early product scenarios
rather than evidence-only or governance-only scenarios.

The `demo-tetris-35` replay verified that COO now breaks the Tetris brief into
clear product scenarios. The first COO pass still grouped rotation under
generic keyboard control, and the new outline guard forced a correction; the
second pass produced eight product-specific scenarios. CTO then created
`T-001` for `F-001-S001`, but after the handoff gate asked for
`F-001-S002` and `F-001-S003`, the follow-up `ticket_create` calls repeatedly
arrived without a usable `bdd_scenarios` array. Decision: the ticket creation
path now remembers the pending CTO handoff scenario batch and can fill missing
BDD scenario IDs from that state when CTO retries the next ticket.

The `demo-tetris-36` replay exposed an earlier role-boundary leak in a fresh
Phaser target. CEO correctly read the Tetris brief, but then shell execution
attempted product/dependency-shaped mutations before COO/CTO/Engineer handoff,
leaving target churn such as a root `package.json` and blocked package-manager
commands. Existing file-write policy protected planning artifacts, but
shell-based mutation still let a planner role bypass the product-ticket spine.
Decision: planner roles such as CEO, Head of Strategy, COO, CTO, and CTO-weekly
may use read-only shell inspection when policy-safe, but mutating `shell_exec`
is blocked until work reaches the role that owns the mutation path.

The `demo-tetris-37` replay validated the planner shell guard and bind-retry
idempotency: retrying after a bind failure reused the same CEO bootstrap job,
and CEO no longer mutated product files through shell. The run then exposed a
COO planning loop. Capability extraction read Markdown active-goal Scope and
Non-Goals as one long statement, so operational constraints such as npm
install/build scripts and descoped optional mechanics such as hold piece were
treated as required product scenarios. Decision: capability extraction now
treats Markdown bullets and headings as sentence boundaries, recognizes
explicit product-action bullets such as must implement/detect/allow, ignores
generic access phrasing such as opening the game locally, and keeps operational
validation constraints and non-goals out of required product scenario coverage.
The next `demo-tetris-39` replay showed the same capability gate was now
product-focused, but still too literal for natural gameplay wording such as
"clear full lines" versus "lines clear" and "restart for another round" versus
"restart functionality." Decision: capability matching treats those qualifier
words as non-essential so the gate validates the product behavior rather than
the exact prose.

The `demo-tetris-41` replay narrowed the same class further. CEO completed
cleanly, COO produced a product-specific plan and a compact Tetris feature
contract, but the outline guard rejected "Game ends when stack fills and user
can restart" as coverage for the README capability "reach game over when the
stack fills." The target contract was healthy; the source matcher was too
literal. Decision: game-over and game-ending wording now canonicalize to the
same capability, allowing natural scenario titles while preserving the
requirement that product capabilities be visible in the scenario schedule or
headings before CTO ticketing.

The `demo-tetris-42` replay kept that progress but showed the matcher still
treated product names and motion modifiers as required outline keywords. A
healthy refined schedule with "Playfield is visible and keyboard controls work"
and "Tetrominoes move and rotate with keyboard" was rejected because README
said "see a Tetris playfield" and "rotate falling tetrominoes with the
keyboard." Decision: product names such as Tetris and modifier words such as
falling no longer act as required capability keywords; the gate still requires
the actual behavior words such as playfield, rotate, tetromino, keyboard, line
clearing, score, game-over/end, and restart.

The `demo-tetris-43` and `demo-tetris-44` replays then reached better product
contracts with separate scenarios for browser access, playfield, movement,
rotation, line clearing with score, game over, and restart. The guard blocked
COO because the Out of Scope section excluded "High score tracking or
persistence," "animations beyond basic movement," and "UI beyond the game grid
and score display," treating those advanced extensions as if basic score or
movement behavior had been removed. Decision: high-score, persistence, and
`beyond ...` qualifier wording now leave basic behavior in scope when the
scenario outline still covers that behavior.

The `demo-tetris-47` clean replay validated the newer product-first planning
path again: CEO handed off to COO, COO converged on a product-specific scenario
schedule, and CTO created ordinary product tickets instead of intervention debt.
It then exposed a foundation-owned recovery defect after a duplicate
`ticket_create` attempt. The valid early scenario ticket batch was already in
the backlog and committed, but successful disposition still required another
successful `ticket_create` after the duplicate failure. Decision: CTO
implementation handoff now ignores the stale ticket-create failure when the
ticket lifecycle already covers the required early product scenario batch.

The `demo-tetris-48` rerun exposed a second foundation-owned matching issue
before the AD-254 path could be revalidated. COO produced a concrete scenario
for "Tetrominoes Lock Into Stack On Contact," but the gate still complained
that the brief capability "lock pieces into the stack" was missing. Decision:
capability matching now treats generic "piece/pieces" wording as the same
product object as concrete game-piece nouns such as tetromino when validating
scenario coverage.

The `demo-tetris-49` rerun moved the next blocker earlier in bootstrap. CEO
found the canonical generated feature file but tried to create
`docs/features/F-001.md`; the duplicate feature path guard correctly rejected
that duplicate, yet told CEO to update the canonical contract. The next CEO
attempt to update the canonical feature contract was then blocked by the role
write boundary, because COO owns feature contracts. Decision: duplicate feature
path guidance is now role-aware, telling CEO to hand off canonical feature
updates to COO while keeping the normal canonical-update guidance for roles
that may write feature contracts.

The `demo-tetris-50` rerun confirmed the CEO fix in the live lifecycle: CEO
completed in 9 turns and COO produced a product-specific active plan and
feature contract. CTO eventually created an ordinary product ticket batch and
handed to Engineer, but Engineer exposed a browser-framework entry-loop
failure. It ran `node --check main.js` before any `main.js` existed, which was
recorded as an unresolved product runtime failure, and the Phaser lifecycle
guard then falsely treated a top-level `new Phaser.Game(config)` after scene
callbacks as recursive construction inside `preload`, `create`, and `update`.
Decision: missing-file `node --check` is validation-procedure failure, and the
Phaser source guard now checks actual function bodies before flagging recursive
game construction.

The `demo-tetris-51` rerun used the AD-257 build and uncovered a planning
handoff mismatch before Engineer. COO created `F-002` as the active,
product-specific core gameplay contract and marked generated `F-001` as
superseded. The capability gate still treated `F-001` as the only contract
that could satisfy README capabilities, so COO looped between editing F-001,
editing F-002, committing no-op planning changes, and retrying handoff.
Decision: product capability coverage and CTO handoff gates now follow active
feature contracts and ignore superseded walking-skeleton contracts except as
bootstrap fallback.

The `demo-tetris-52` rerun validated that active-contract planning can reach
implementation: CEO, COO, and CTO created and committed product-specific plan,
feature, and ticket artifacts, then Engineer claimed the ticket, wrote a Vite
Phaser app, installed dependencies, started Vite, and confirmed the HTTP
entrypoint. The next blocker was a validation-procedure mismatch. Engineer
used plain `node -e` to `require()` a Phaser scene module; Phaser startup then
failed with `ReferenceError: window is not defined`, and the runtime treated
that probe failure as an unresolved product runtime blocker that prevented
`node --check` and `npm run build` from running. Decision: browser-framework
Node eval probes that load browser-only modules and fail on missing browser
globals are validation-procedure failures rather than product runtime blockers.

The `demo-tetris-53` rerun confirmed the browser-framework procedure fix by
reaching product build and validation: Engineer created the package-managed
Vite app, passed `npm run build`, started and stopped the Vite dev server, and
ran the documented `node -e` browser-smoke assertion successfully. The closeout
still failed at `max_turns` because a repo-root `validate-game.js` scratch
helper became committed product clutter; Engineer then spent final turns trying
to remove or inspect that helper instead of moving T-001 to `done`. Decision:
new root validate/smoke/probe helper files are blocked before creation, while
durable validation helpers remain available under `scripts/` or `tests/`.

The `demo-tetris-54` rerun validated that root helper quarantine held: Engineer
used the direct `node -e` smoke assertion and did not create a root helper. The
next blocker was dependency repair state. After `npm run build` failed with
`vite: command not found`, Engineer correctly used `dependency_sync`, but the
unresolved test/build lane still rejected the immediate `npm run build` rerun
as unchanged validation and pushed the role toward unrelated probes and source
edits. Decision: successful Engineer `dependency_sync` now counts as a repair
action for the unresolved test/build lane, while same-lane validation still has
to pass before evidence, completion, commits, or successful disposition.

The `demo-tetris-55` rerun validated the package dependency repair path and
reached a clean committed Phaser product slice. CTO created the early product
ticket batch, but the first ticket still contained CDN acceptance wording for
loading Phaser, which contradicted the foundation's local dependency and build
policy. QA then ran build and HTTP evidence correctly, but the missing-smoke
approval block did not print the canonical smoke command and the validation
policy blocked QA from killing its own tracked Vite background PID. Decision:
Phaser/browser-framework tickets now reject CDN runtime acceptance criteria,
reviewer smoke blockers include the canonical `node -e` product-smoke command,
and QA/Security may stop tracked background validation PIDs while arbitrary
cleanup remains blocked.

The `demo-tetris-56` rerun used the AD-262 build and reached COO planning
again. CEO and COO stayed product-specific and intervention-debt remained
telemetry-only, but COO looped on the capability gate because the active goal
heading "Implement core Tetris gameplay mechanics" produced a required
capability phrase "core Tetris gameplay mechanics". The feature contract already
covered the concrete mechanics: browser access, playfield, movement/rotation,
line clearing, scoring, game over, and restart. Decision: generic summary
labels such as core gameplay mechanics are ignored as standalone capability
keywords, while the concrete behavior words remain required.

The `demo-tetris-57` rerun confirmed that generic gameplay labels no longer
block handoff. CEO completed in 9 turns and COO wrote a product-specific plan
and feature contract. The next guardrail loop came from Out of Scope wording:
"Mobile touch controls" was treated as if the required movement capability had
been descoped because `controls` alone counted as movement coverage. Decision:
movement coverage now requires directional language or keyboard controls; plain
alternate-input controls do not cover or descope the movement capability.

The `demo-tetris-58` rerun confirmed the AD-264 matcher fix and progressed
through product-specific planning, ticketing, implementation, dependency
install, Vite build, and canonical browser-product smoke. Runtime containment
also behaved correctly: guardrail and context-overflow signals stayed in
foundation telemetry instead of target intervention debt. The next bottleneck
was post-validation drift. After browser-framework build and product smoke had
passed, Engineer continued shell inspection of generated `dist/assets` instead
of closing the ticket lifecycle, pushing the prompt over the local context
window. Decision: once browser-framework completion evidence has passed in the
same Engineer job, further shell exploration is blocked while dirty
implementation or ticket work remains; the role must commit, update evidence,
move the ticket to done, and record disposition.

The `demo-tetris-59` rerun validated the post-validation convergence guard and
showed broader lifecycle progress: bootstrap created exactly one CEO job, CEO
and COO produced product-specific planning, CTO created multiple ordinary
Tetris tickets, and Engineer completed the first three product tickets without
intervention-debt flooding. The run also exposed three foundation-owned
efficiency gaps. Dogfood created a target-owned finding for `F-001-S003` while
active product tickets already covered that scenario, so scenario dedupe needed
to catch active overlap rather than only exact scenario-set matches. QA and
Security both eventually satisfied build plus canonical browser-product smoke,
but first spent turns on weaker or invalid validation shapes such as direct
Phaser `require()` from Node or broad recursive secret grep through
`shell_exec`. Orchestrator also tried to read flat `docs/tickets/T-NNN...`
paths and used content grep for filenames instead of preserving lifecycle
ticket paths from context. Decision: active ticket scenario overlap now dedupes
through `ticket_create`, QA/Security prompts route directly to canonical
browser-framework evidence, Security avoids broad shell secret scans, and
Orchestrator uses lifecycle ticket paths or frontmatter IDs rather than
invented flat paths.

The `demo-tetris-60` rerun validated the active-scenario dedupe and reviewer
guidance changes through fresh bootstrap, planning, and ticket creation. The
run created exactly one CEO bootstrap job after a retry, kept deterministic
guardrail failures in foundation telemetry, produced a Tetris-specific active
plan and feature contract, and created three ordinary product tickets. Engineer
then claimed the first ticket, added a local Vite/Phaser package shape, ran
dependency install, and passed `npm run build`. The new bottleneck appeared
between build and product smoke: Engineer treated generated-bundle inspection,
plain Node `require('phaser')`, and requiring Vite browser bundles as
validation substitutes. Those checks are foundation-owned validation-procedure
mistakes for browser apps because they execute browser-only code under Node and
produce missing `window` or `document` failures unrelated to mounted product
behavior. Decision: after browser-framework build passes and dirty work remains,
Engineer shell validation is limited to build reruns, canonical browser-product
smoke or equivalent source/runtime assertion, and tracked PID cleanup until the
smoke passes.

The `demo-tetris-61` rerun validated the post-build smoke substitute guidance
inside a fresh generated target and then exposed a planning-policy blocker
before implementation. CEO and COO stayed product-specific, produced and
committed Tetris planning artifacts, and intervention signals again remained
foundation telemetry. COO's feature contract covered the required product
behaviors but listed advanced-only exclusions under Out of Scope. The capability
guard incorrectly read explanatory text such as "clear reasons" and advanced
scoring-system exclusions as descoping basic line clearing and score tracking,
causing repeated feature-contract rewrites instead of CTO handoff. Decision:
Out of Scope parsing now ignores explanatory/rationale prose and treats
advanced-only extensions as leaving already-covered basic product capabilities
in scope.

The `demo-tetris-62` rerun confirmed the Out-of-Scope parser fix and moved the
blocker to capability-matching glue. COO first compressed several capabilities
into one scenario, the runtime correctly blocked that, and COO then rewrote the
feature contract into separate schedule entries for visible grid, falling
pieces, movement/rotation, locking, line clearing, scoring, game over, restart,
and runnable gameplay. The remaining blocker came from phrases such as "core
Tetris gameplay including visible grid" and "game over detection": policy
treated "including" and "detection" as standalone required keywords even though
the concrete behaviors were covered. Decision: capability matching now ignores
those glue words while preserving the concrete behavior requirements around
them.

The same `demo-tetris-63` replay narrowed one more wording false positive:
"show game over when the stack fills" was treated as requiring the literal
`show` keyword even though the scenario schedule and headings covered game over
state when the stack fills. Decision: show/display variants join
include/detection as glue words; concrete behavior terms such as game over and
stack-fill coverage remain required.

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
