# Tech Debt Tracker

Standing tracker. Not an active/completed plan.

| ID | Area | Description | Priority | Added |
|----|------|-------------|----------|-------|
| TD-001 | Guardrails | Hard guardrails limited to syntactic checks (regex, file pattern). AST-level validation deferred to v2. | Medium | 2026-04-11 |
| TD-002 | Dashboard | Pipeline flow graph uses simple layered layout, not full Sugiyama algorithm. Custom role topologies may render suboptimally. | Low | 2026-04-11 |
| TD-003 | Scoring | Low-frequency roles (weekly CEO) take months to accumulate 20 scored jobs for meaningful rolling average. Manual trust override is the workaround. | Medium | 2026-04-11 |
| TD-004 | Sandbox | macOS sandbox is process-group based, not namespace-isolated like Linux. Weaker isolation on dev machines. | Low | 2026-04-11 |
| TD-005 | Multi-repo | Schema has repo_id from day one but v1 UI and CLI assume single repo. Multi-repo support deferred. | Medium | 2026-04-11 |
| TD-006 | Quality score | AD-278 regeneration cadence is doctrine-only; wire `scores export` into a post-run hook or scheduled survey (expected with the WS-C pace/convergence telemetry slice). | Medium | 2026-06-11 |
