---
id: MH-061
title: Fix MARS install PATH precedence
priority: high
complexity: small
kind: standard
work_type: fix
bdd_scenarios: ["F-002-S007"]
end_to_end_evidence: required
evidence_links: ["docs/features/F-002-zero-config-shell-path.md", "go test ./internal/shellpath", "make install"]
verified_by: "go test ./internal/shellpath; make install; zsh -ic 'command -v mars; mars version'; zsh -lc 'source ~/.zshrc; command -v mars; mars version'"
owner: "foundation-maintainer"
last_attempt: 2026-06-28
completed: 2026-06-28
blocker: none
blocked_by: []
trace_id: none
dedupe_key: foundation:mars-install-path-precedence
source: operator request to install MARS locally on 2026-06-28
created: 2026-06-28
depends_on: ["MH-060"]
---

# MH-061: Fix MARS Install PATH Precedence

## Context

`make install` installed `/Users/elliottgreaves/go/bin/mars`, but a separate
`/opt/homebrew/bin/mars` symlink still resolved first in a fresh shell. The
existing managed profile block only checked whether the Go bin directory was
present anywhere in PATH; it did not move the install directory ahead of
same-name commands.

## Acceptance Criteria

- [x] Legacy `mars-harness` profile markers are repaired into canonical `mars` markers.
- [x] The generated shell block moves the install directory to the front of PATH.
- [x] Duplicate install-dir PATH entries are collapsed.
- [x] `make install` leaves a sourced Zsh profile resolving `mars` to `/Users/elliottgreaves/go/bin/mars`.

## Evidence

- `GOCACHE=/private/tmp/mars-go-cache go test ./internal/shellpath`
- `GOCACHE=/private/tmp/mars-go-cache make install`
- `zsh -ic 'command -v mars; mars version'` resolved `/Users/elliottgreaves/go/bin/mars` and printed `mars 0.66.1`.
- `zsh -lc 'source ~/.zshrc; command -v mars; mars version'` resolved `/Users/elliottgreaves/go/bin/mars` and printed `mars 0.66.1`.
