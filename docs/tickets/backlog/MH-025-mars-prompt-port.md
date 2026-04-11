---
id: MH-025
title: Port Mars automation prompts to .harness roles
priority: medium
complexity: large
source: delivery-schedule M10
created: 2026-04-11
---

# MH-025: Port all 11 Mars automation prompts to `.harness/roles/` for harness tools

## Context

Mars monorepo automations encode hard-won operational behavior. M10 ports them into harness-native roles with correct tool names, local model constraints, and context-efficient instructions.

## Requirements

- Map each Mars prompt to a role file under `.harness/roles/` with frontmatter metadata (model hints, budgets, triggers)
- Replace tool/action names with harness equivalents; document any intentionally unsupported flows
- Phase 1 deliverables: Engineer, Pipeline Fixer, QA roles production-ready; remaining eight tracked with checklist in PR
- Each prompt includes self-check: “verify CI status URLs”, “no push to default branch”, “changeset discipline” adapted to harness policy knobs
- Estimate 0.5–1 day per prompt for review + fixture run; batch in stacked PRs if needed

## Acceptance Criteria

### Functional (happy path)
- [ ] Engineer, Pipeline Fixer, QA roles execute on sample repo with MH-009/MH-016 wiring without manual prompt edits
- [ ] All eleven prompts exist in repo; each has `prompt_version` and `source_mars_commit` note in header comment
- [ ] Manifest lists roles with triggers consistent with ported automation intent

### Edge cases and negative paths
- [ ] Prompts that referenced GitHub-only features include guard text when feature flag off
- [ ] Token-heavy sections trimmed or split using context assembly hooks (MH-004) without losing critical constraints

### Non-goals
- [ ] Parity with every Cursor-only UI instruction
- [ ] Translating non-English operator docs

### Observability, docs, and regressions
- [ ] Golden transcript smoke: each role at least one dry-run conversation recorded in `testdata/`
- [ ] Checklist file `docs/prompt-port-status.md` updated per merged tranche
- [ ] Reviewer (MH-020) can propose diffs to these files under normal safety limits
