# F-014: MARS Rename

**Feature ID:** F-014
**Goals:** Rename the old Mars Harness product, repository, module, CLI, generated target guidance, and durable docs to MARS while preserving explicit migration compatibility.
**Status:** active
**Owner:** foundation-maintainer

## Business Logic

MARS is the canonical product and repository name. The command-line binary is
`mars`. The Go module path is `github.com/greaveselliott/mars`.

Canonical user-facing commands, generated target guidance, release/update
commands, installer scripts, shell PATH setup, tool references, docs, tickets,
and feature contracts must name `mars`, not `mars-harness`.

Compatibility is deliberate, bounded, and tested:

- Existing `MARS_HARNESS_*` environment variables are read only as fallbacks
  when the canonical `MARS_*` variable is unset.
- Existing `~/.mars-harness` state can be read as a legacy fallback, but new
  default state is under `~/.mars`.
- Existing `mars_harness_cli` tool calls remain accepted as an alias, but
  `mars_cli` is canonical.
- Existing release, generated metadata, and integration markers using old names
  remain readable, while writers emit new MARS markers.
- References to the separate predecessor Mars monorepo remain only when the
  URL or context proves they are not old product naming.

## Step-By-Step Behavior

1. A source checkout builds the command from `./cmd/mars`.
2. `mars version`, `mars setup`, `mars doctor`, `mars tools`, `mars mcp`,
   `mars release`, `mars update`, and lifecycle commands show canonical MARS
   names in help, errors, remediation, and docs.
3. Source imports use `github.com/greaveselliott/mars`.
4. Config defaults and runtime state write under `~/.mars`.
5. Compatibility readers check old env names, old state paths, old release
   markers, old generated metadata, and old integration markers where existing
   installations or generated targets may already contain them.
6. Generated targets receive MARS guidance, `mars` commands, `mars_cli` tool
   references, and new metadata.
7. Historical tracked docs are rewritten to MARS unless preserving a separate
   Mars monorepo reference or a tested compatibility fixture.
8. Final validation fails on any unclassified old-name string.

## Scenario Schedule

### F-014-S001: Module And Entrypoint Rename

Given the source repo is checked out
When the code is built and tested
Then the module path is `github.com/greaveselliott/mars`
And the CLI entrypoint is `cmd/mars`
And no Go import uses `github.com/greaveselliott/mars-harness`.

### F-014-S002: Canonical CLI Identity

Given a user runs the CLI
When help, version, setup, doctor, release, update, tools, or lifecycle commands print commands or remediation
Then the visible command name is `mars`.

### F-014-S003: Tool And Env Compatibility

Given an agent or operator uses MARS through the mirrored tool surface
When it invokes `mars_cli`
Then the tool runs the active MARS executable with structured args
And legacy `mars_harness_cli` invocations still route to the same handler.

Given canonical `MARS_*` env vars are absent
When a legacy `MARS_HARNESS_*` var is set
Then MARS reads it as a fallback without treating it as canonical output.

### F-014-S004: Runtime State Migration

Given no explicit paths are supplied
When MARS creates config, models, bins, traces, logs, telemetry, or per-repo DBs
Then new defaults use `~/.mars`
And old `~/.mars-harness` paths are read only as compatibility fallbacks where existing state matters.

### F-014-S005: Release And Update Rename

Given a release or update workflow runs
When assets, install scripts, self-update package paths, changelog markers, and repository names are produced
Then they use `mars`, `github.com/greaveselliott/mars`, and `greaveselliott/MARS`
And old release markers remain readable for backfill and audit.

### F-014-S006: Generated Target Doctrine

Given `mars init` or upgrade writes target harness defaults
When generated docs, skills, manifests, tool allowlists, role prompts, and metadata are inspected
Then canonical guidance uses MARS, `mars`, `mars_cli`, and `generator: mars`
And old generated metadata is readable as legacy input.

### F-014-S007: Tracked Docs Rewrite

Given historical tracked docs, tickets, changelog, validation reports, skills, and runbooks are searched
When old product strings are found
Then they are rewritten to MARS unless explicitly classified in the allowlist.

### F-014-S008: Exhaustive Final Audit

Given all implementation slices are complete
When old-name grep and old-path grep run
Then every remaining old-name hit is tied to a compatibility test, parser, migration note, or distinct historical Mars monorepo reference.

## Out of Scope

- Rewriting git commit history.
- Removing old compatibility support in the same change.
- Changing the project’s `MarsDocSync` metadata name.

## Descoped Scenarios

None.

## Evidence

- Pending: targeted rename tests.
- Pending: `go test ./...`.
- Pending: final old-name grep allowlist.
- Pending: generated target smoke.
- Pending: release/backfill/assets evidence or explicit blocker.
