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
| TD-007 | Docs hygiene | Add a docsconsistency length/staleness guard for the active plan (flag ledger-style accretion in Current Truth) so T-022-class extraction becomes mechanical. | Low | 2026-06-11 |
| TD-008 | Release | `release notes` numbers from the local `VERSION` only and silently reuses versions already published from trunk when run on a divergent branch (2026-06-11 incident: branch regenerated 0.43.2–0.44.3 colliding with published tags). Add a guard that warns or fails when `origin/main` `VERSION` or the highest published `vX.Y.Z` tag is ahead of the local base. | High | 2026-06-11 |
| TD-009 | Factory pace | COO `max_turns` on `depot-supplies-api` ephemeral profile wedged 2026-06-13 WS-D closure Run 2 (3 failures before CTO). Calibrate COO turn budget or tighten exec-plan/feature-contract guidance for API briefs; replay: `validation-target.mjs create --profile depot-supplies-api --label wsd-closure`. | Medium | 2026-06-13 |
| TD-010 | God files | Deferred from foundation plan: split `cmd/mars-harness/main.go` (~3.8k lines), `internal/serve/server.go` (~2.7k), and `internal/scanner/init.go` (~5k) per AD-287 follow-on; policy monolith extraction complete (T-043). | Low | 2026-06-13 |
