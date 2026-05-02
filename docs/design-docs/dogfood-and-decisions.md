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

3. **COO manifest updated** — `ticket_create` and `file_search` added to tool allowlist in `internal/scanner/init.go`.

4. **COO prompt rewritten** — STEP 3 now references the TICKET INDEX and instructs the agent to use `ticket_create` instead of `file_write`. "Build a mental list" language removed (the tool handles this mechanically). Commit gate added.

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

`docs/tickets/in-progress/` is now the front of the Engineer queue. The Engineer must complete the lowest-numbered in-progress ticket before claiming backlog work. If the ticket is blocked by build failures, missing scripts, dependency drift, test failures, or local convention gaps, the Engineer fixes those blockers as part of the same ticket rather than handing the problem off.

The mechanical gate now allows a handoff when an Engineer run drains one pre-existing in-progress ticket to `done/`, even if other pre-existing in-progress tickets remain for the next run. It still blocks:

- claiming new backlog work while any in-progress ticket existed at run start
- ending with a newly claimed ticket still in progress
- returning in-progress work to backlog instead of completing it
- making no completion progress while open work existed

### Consequences

- Multiple in-progress tickets become a drainable queue, not a permanent failure state.
- The self-chain keeps running Engineer jobs until the in-progress queue is empty before backlog work resumes.
- Prompt guidance and the injected ticket index both state that in-progress tickets are first priority.
- Blockers are treated as implementation work, preserving the harness expectation that a ticket ends in committed, pushed, verified trunk changes.

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
