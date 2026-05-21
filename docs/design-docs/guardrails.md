# Guardrails Engine

**Status:** Draft  
**Date:** 2026-04-11  
**Author:** Mars Harness contributors

Mechanical checks on harness outputs and repo mutations: what is enforced, how overrides work, and how the engine stays maintainable as rules grow.

## Context

Guardrails prevent destructive or policy-violating changes while keeping **false positives** manageable. v1 prioritizes **fast, deterministic** checks over deep semantic analysis; richer analysis is explicitly deferred.

Rules are loaded from the repo’s `.harness/` tree so teams can customize policy without recompiling the binary. Invalid rule files should **fail closed** at job start with actionable parse errors.

## Key Design Decisions

### AD-012: Hard guardrails are syntactic in v1

**Hard** guardrails in the first release are limited to **syntactic** checks: regular expressions, path/file patterns, and **file existence** predicates. **AST-based** or language-aware analysis is **v2**, to avoid shipping a slow or incomplete analyzer that blocks legitimate edits.

“Hard” means a failing check blocks promotion or fails the job per [pipeline-engine.md](pipeline-engine.md) policy hooks.

### AD-107: Workspace Hygiene Gates Generated Churn Before LLM Work

Generated dependency and build output is a hard guardrail surface because it
can overwhelm blast-radius checks, context windows, and dispatch recovery. The
server runs `workspace_hygiene` before model loading, package-manager mutation
must flow through `dependency_sync`, and raw install/fetch commands in
`shell_exec` are blocked. Blast-radius checks classify generated paths
separately from implementation paths so ignored dependency trees do not masquerade
as source-risk. Root package-manager lockfiles and checksum files are generated
dependency metadata for line-count blast-radius purposes, while remaining
git-visible and secret-scanned. When missing generated-directory ignore policy
is safely inferable, `serve` appends the ignore entry and commits only
`.gitignore` before the model starts. Blocking results are otherwise actionable
recipe outputs, not automatic cleanup: the harness reports the generated path,
recipe ID, and next action, then waits for an explicit source/tracking fix.

The gate treats missing generated-directory ignores, tracked generated
directories, dirty generated output, large generated diffs, and file deletions
as deterministic workspace problems. It keeps target-owned repo hygiene in the
target backlog rather than classifying it as a foundation-only telemetry issue.
Auto-repair is deliberately narrow: it never removes files, unstages user work,
or commits package manifests, lockfiles, source files, or tracked generated
trees.

### AD-115: Ticket Lifecycle Moves Are A Bounded Deletion Exception

**Status:** Accepted
**Date:** 2026-05-19

Blast-radius deletion checks still block arbitrary file deletion by default,
but ticket lifecycle moves are an explicit bounded exception. Moving the same
ticket ID between `docs/tickets/backlog/`, `docs/tickets/in-progress/`,
`docs/tickets/in-review/`, and `docs/tickets/done/` is required for normal
delivery truth. The diff-stat policy therefore ignores the deletion side of a
ticket move only when the same ticket ID appears as a new ticket markdown file
in another lifecycle directory in the same worktree diff, as a staged
`git mv` destination, or as an already-present lifecycle counterpart being kept
while a duplicate old-state ticket is removed.

This keeps destructive deletion containment intact while allowing agents to
claim, review, and complete tickets without being trapped by the deletion guard.
Unpaired ticket deletions, root ticket markdown churn, and arbitrary source or
doc deletions remain blocked by the normal blast-radius policy.

The related successful-disposition clean-tree gate remains strict for product,
ticket, source, and documentation paths, but it does not count the
runtime-managed `.harness/learnings.yaml` file by itself. That file can be
updated by convention detection while a job runs; treating it as product work
caused repair and review handoffs to fail despite a clean user-facing target
diff. When that runtime file is the only dirty path after a server job, the
executor commits it as a `chore(learnings)` update; mixed dirty trees remain
visible and blocked by the normal clean-handoff rule. The exception is
intentionally local to runtime-owned learning metadata; secret scanning,
blast-radius checks, and ordinary product dirty paths still apply.

Planner ownership is also a hard tool-policy surface. COO is allowed to write
the active exec plan, feature contracts, backlog plans, and goal observations,
but attempts to create implementation files or run mutating shell commands are
blocked before mutation. This keeps first-run product code behind CTO ticketing
and Engineer delivery even when an existing target manifest still exposes broad
tools to COO.

Feature ticket completion evidence is now enforced at the same pre-mutation
layer. A role may still move tickets through the lifecycle, but `git mv`/`mv`
into `docs/tickets/done/` and `file_write` saves to done feature tickets are
blocked while required BDD evidence fields remain empty. This turns a late
post-run ticket-gate failure into an immediate tool error that the active role
can repair before it records disposition or triggers a separate repair job.
Feature-contract scenario uniqueness uses the same recovery shape: duplicate
scenario-heading errors name the duplicate heading line numbers and clarify
that Scenario Schedule list references are allowed, so roles can replace the
existing scenario section instead of repeatedly appending another heading.

Dogfood observation is also a hard tool-policy boundary. Dogfood may create
target-owned findings through `ticket_create` and write bounded evidence under
`docs/reports/dogfood/`, but direct `file_write` changes to product source,
package manifests, lockfiles, config, or harness scaffold are blocked. A
validator that needs those changes must report a finding or blocked disposition
instead of silently becoming an implementation role.

### AD-142: Repo-Local Build Artifact Cleanup Is A Bounded Removal Exception

**Status:** Accepted
**Date:** 2026-05-20

Validation commands can create untracked root-level binaries in ordinary target
repos. The live `demo-api-run2` replay showed `go build .` writing a binary
named after the repo, and the live `demo-api-run4` replay showed an Engineer
leaving a root binary named after the Go module (`task-notes-api`). Both made
blast-radius checks see tens of thousands of changed "lines" and then blocked
the Engineer from cleaning the artifact with `rm`. That trapped later ticket
and evidence updates behind the very guardrail intended to protect the repo.

`shell_exec` therefore has a narrow removal exception for repo-local compiled
artifacts: `rm <artifact>` or `unlink <artifact>` is allowed only when the path
is root-level, untracked, named exactly after the repository directory or the
basename of the root `go.mod` module path, and binary-looking. Recursive
removal, tracked files, ordinary source/docs, nested paths, and arbitrary
filenames remain blocked by the normal destructive-command policy. Post-tool
blast-radius validation still runs after cleanup, so the exception only clears
generated build output and does not waive review for the remaining source or
ticket changes. When blast-radius validation fails because one of these
cleanable artifacts is present, the error names the exact `rm <artifact>`
command so the agent can recover instead of trying unrelated build or write
operations until it reaches a turn limit.

The follow-up `demo-api-run7` replay showed that cleanup hints are still a
second-best recovery path: Engineer used managed `background:true` validation
successfully, then created `task-notes-api` with `go build -o task-notes-api
main.go` and got trapped after a malformed recovery call while the dirty binary
remained present. Guardrails now prevent that class before mutation by rejecting
explicit `go build -o <path>` outputs that resolve inside the target repo. Roles
must put validation binaries in an external temp path, keeping the target diff
limited to source, tests, docs, tickets, and intentional config.

The follow-up `demo-api-run11` replay showed the same artifact trap through the
implicit Go build output path: Dogfood ran `go build ./...`, which created a
repo-root `task-notes-api` binary before post-command blast-radius validation
could reject it. Guardrails now also reject `go build` without `-o` before
process execution. Roles that only need compile evidence should use
`go test ./...`; roles that need a runnable validation binary must use an
external `-o /tmp/<name>-validation` output.

The follow-up `demo-api-run9` replay showed a related scratch-file trap:
Engineer created a root `validate.sh`, could not delete it because broad `rm`
remains blocked, and committed the script with the ticket completion. Dogfood
later proved the script was not portable because it depended on the host
`timeout` command. Guardrails now block creation of new repo-root validation
shell scripts such as `validate.sh` and `validation.sh` while allowing existing
project-owned scripts to be updated. `shell_exec` also rejects external
`timeout`/`gtimeout` executables so roles use tool `timeout_seconds` or managed
`background:true` validation instead of platform-specific process wrappers.

The follow-up `demo-api-run15` replay showed that documentation-sync failures
should also be caught before source files are written, not only at successful
handoff. `file_write` now blocks source, test, static asset, and workflow
writes under audited source roots, plus root-level source files such as
`main.go` or `index.html`, unless the content already carries top-of-file
`MarsDocSync` metadata pointing at existing documentation. This keeps scenario
ID paths and missing metadata from becoming committed product state.

The `demo-api-run17` replay showed that Engineer can also spend implementation
turns on shell discovery before visible ticket ownership. When ordinary product
backlog work exists and no ticket is in progress, Engineer `shell_exec` now
allows only the backlog-to-in-progress `git mv` claim. Discovery, validation,
and no-op shell calls are blocked until the claim is committed.

### AD-215: Test/Build Repair Guardrails Repeat The Failing Assertion

**Status:** Accepted
**Date:** 2026-05-21

The `demo-slug-run61` replay showed the right repair boundary but weak
feedback. QA requested rework because the Slugify CLI implementation had no Go
tests. Engineer added `cmd/slugify-json/main_test.go`; the same-lane `go test`
then failed because a contract-shaped assertion expected punctuation-separated
words to count as separate words while the implementation returned `2`.
Guardrails correctly blocked unrelated runtime probes, builds, commits, ticket
moves, and successful disposition while that failure was unresolved, but the
repeated policy message only named the unresolved command. The role could keep
bouncing off the same boundary without the assertion failure being the most
salient next action.

Unresolved test/build session state now records a compact copy of the latest
failing stdout/stderr or exit code. Every later guardrail message for that
unresolved lane includes both the exact command and the failing output. The
guidance also says that when a failing assertion matches the ticket, README, or
BDD contract, Engineer must edit the implementation instead of deleting or
weakening the test. Successful same-lane validation clears the stored failure
output with the rest of the unresolved repair state.

### Open topics

- **Advisory vs hard tiers:** advisory rules surface warnings in traces and UI; hard rules fail the job or block merge paths per policy; same schema with a `severity` field is the likely shape.
- **YAML format** for rule definitions: schema versioning, validation at load time, clear error messages pointing to file and line.
- **Prompt injection:** treat untrusted repo content as hostile; rules and prompts must assume adversarial README/docs; never `eval` rule bodies as code.
- **Mechanical validation:** deterministic execution, no network dependency for core checks; optional advisory fetches are explicitly labeled.
- **Override mechanism:** explicit allowlists / break-glass with audit trail (who, when, why); time-bounded overrides preferred over permanent silence.
- **Staleness detection:** rules that reference deleted paths or obsolete globs should warn or fail lint of `.harness/` on `harness doctor` or pre-job validation.

### Relationship

- [agent-runtime.md](agent-runtime.md) invokes guardrails at defined checkpoints (post-tool, pre-commit simulation, pre-push).

### Performance budget

Rule evaluation should stay sub-second for typical repos on laptop hardware; pathological regexes may need timeouts or complexity caps in v2.

## Discoveries

- **2026-05-21 — Scoped test/build repair writes:** A clean target replay
  showed Engineer responding to a failing `go test ./cmd/...` command by
  creating a parallel root `main.go` and `main_test.go`. Guardrails now record
  the failed Go package target when it is narrow and block source/test writes
  outside that scope until the same test/build lane is repaired.
- **2026-05-21 — Structured Mars Harness CLI routing:** A clean release replay
  showed Release Manager resolving an older installed `mars-harness` through
  `shell_exec`, even though the active running harness binary had the needed
  `release` command. `shell_exec` now blocks direct `mars-harness` binary
  invocations and points the role at equivalent `mars_harness_cli` args.
- **2026-05-21 — Failing test output in repair guidance:** A Slugify CLI replay
  showed Engineer stuck after adding failing tests for a real implementation
  mismatch. Guardrails now repeat the latest failing assertion output in
  unresolved test/build guidance instead of naming only the command.
