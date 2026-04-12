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

- Local pipeline is fast and focused: 7 roles, no dead triggers
- Decisions accumulate across runs, making agents progressively smarter about the repo
- Moving to a PR/CI model later requires only uncommenting manifest entries
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

1. **Default manifest tool lists** — added `git_status`, `git_diff`, `git_commit`, `git_push` to all write-capable roles (engineer, dogfood, janitor, pipeline-fixer, pr-comment-fixer, CEO, COO, CTO, release-manager, security-weekly).

2. **Commit gates in role prompts** — every write-capable role prompt now includes a "COMMIT GATE" section that requires the agent to run `git_status` before finishing and commit any uncommitted changes. The Engineer, Dogfood, Janitor, Pipeline Fixer, and PR Comment Fixer prompts all enforce this.

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
