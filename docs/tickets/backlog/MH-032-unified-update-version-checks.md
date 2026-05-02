---
id: MH-032
title: Unified update version checks for tool and target harness
priority: high
complexity: medium
source: operator feedback 2026-05-02
created: 2026-05-02
kind: intervention-debt
dedupe_key: "public-example"
metadata:
  role: "release"
  repo_id: "mars-harness"
  target: "update"
  category: "version_drift"
  severity: "high"
---

# MH-032: Unified update version checks for tool and target harness

## Context

Mars Harness now has semantic versioning for the source tool and initialized target repos. Operators should not have to remember whether the binary, source checkout, or deployed target harness is behind. The CLI should use one vocabulary and check version drift automatically.

## Requirements

- Add a version-check path under the unified `mars-harness update` command.
- Compare the installed CLI version with the latest configured source:
  - GitHub latest release when a GitHub source is configured
  - source checkout or Go module version when using source-development mode
- Compare a target repo's deployed harness metadata/version with the installed CLI's generated-harness version.
- Recommend or run the right command:
  - `mars-harness update tool`
  - `mars-harness update harness --repo <path>`
  - both, in the right order, when both are stale
- Keep `mars-harness upgrade --repo` as compatibility language, but prefer `update harness` in docs and prompts.

## Affected Files

- `cmd/mars-harness/main.go`
- `internal/selfupdate/`
- `internal/scanner/`
- `internal/doctor/`
- `docs/quickstart.md`
- `docs/design-docs/release-versioning.md`
- `docs/product-specs/product-surface.md`

## Acceptance Criteria

### Functional (happy path)

- [ ] `mars-harness update check --repo <path> --json` emits parseable status for tool and target harness.
- [ ] Behind CLI version reports latest available version and update command.
- [ ] Behind target harness reports installed generator version and update command.
- [ ] Up-to-date tool and target report no action needed.

### Edge cases and negative paths

- [ ] Missing network or GitHub API failure reports unknown remote status without failing local target checks.
- [ ] Target repos without harness metadata receive an actionable `mars-harness init` or `update harness` recommendation.
- [ ] Source-development installs can check `main`/configured module version without requiring `cd`.

### Non-goals

- Replacing MH-031 release-asset publication.
- Silently mutating target repos during a check-only command.

### Observability, docs, and regressions

- [ ] Doctor includes version drift findings.
- [ ] Docs use `update tool` and `update harness` consistently.
- [ ] Tests cover behind/ahead/equal/unknown states.
