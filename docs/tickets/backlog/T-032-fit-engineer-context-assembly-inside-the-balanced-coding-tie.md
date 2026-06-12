---
id: T-032
title: Fit engineer context assembly inside the balanced coding-tier window for package-managed frontend targets
priority: P1
complexity: medium
work_type: intervention-debt
bdd_scenarios: []
end_to_end_evidence: not_applicable
evidence_links: ["docs/validation/reports/2026-06-12-demo-12-frontend-baseline.md", "docs/validation/reports/2026-06-12-demo-13-maintenance-baseline.md"]
verified_by: "TBD"
owner: "TBD"
last_attempt: "2026-06-12: recorded from demo-12 baseline; demo-13 existing-repo-maintenance replay reproduced the same overflow (32923 tokens vs 32768 ctx) on both engineer jobs, confirming the wedge generalizes to any package-managed JS target"
blocker: "none"
blocked_by: []
trace_id: "TBD"
next_action: "Pick a fix (trim engineer context assembly to the tier window, route oversized turns to the 131k reasoning endpoint, or raise balanced coding ctx) and replay BOTH demo-12 and demo-13 archetypes per AD-284 before claiming improvement."
kind: intervention-debt
source: weekly-priorities.md
created: 2026-06-12
depends_on: []
---

# T-032: Fit engineer context assembly inside the balanced coding-tier window for package-managed frontend targets

During the 2026-06-12 demo-12 package-managed-frontend baseline (docs/validation/reports/2026-06-12-demo-12-frontend-baseline.md, finding F1), engineer prompts crossed the balanced coding tier 32768-token window (33281 and 32883 prompt tokens) once the Vite/React scaffold existed, and llama.cpp rejected the requests as non-retryable exceed_context_size_error. The wedge reproduced across an operator retry and blocks the archetype past initial scaffolding. Options: trim context assembly to fit the configured tier window (Tenet 9), route oversized engineer turns to the 131k reasoning endpoint, or raise the balanced coding ctx. The heavy-model probe of the same archetype wedged engineer differently (dependency_sync ENOENT against a nested project dir) — both belong to this archetype gap.

2026-06-12 update: the demo-13 existing-repo-maintenance baseline
(docs/validation/reports/2026-06-12-demo-13-maintenance-baseline.md, finding
F1) reproduced the identical overflow (32,923 prompt tokens vs 32,768 ctx)
on both engineer jobs against an existing Phaser/Tetris project. The wedge
generalizes to any package-managed JS target with a non-trivial file tree,
so the fix must replay both archetypes per AD-284 before claiming
improvement.
