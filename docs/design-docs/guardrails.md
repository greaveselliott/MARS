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

_(None yet.)_
