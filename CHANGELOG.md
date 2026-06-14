# Changelog

Patch notes are generated with `mars-harness release notes` from semantic commits on `main`.

## [0.58.2] - 2026-06-14
<!-- mars-harness-release: version=0.58.2 commit=f68ca5c76d4c -->

### Impact
- **tools:** Operators see improved reliability because repeated guardrail blocks now return actionable repair guidance to the active agent.

### Why
- **tools:** This closes a failure mode where an agent could keep retrying a blocked tool call without a compact repair path.

### What Changed
- **tools:** Added repeated-guardrail repair guidance, COO feature-contract repair hints, and unresolved test/build lane guidance (f68ca5c).

### Fixes
- **tools:** Add repair guidance for repeated guardrails (f68ca5c)

## [0.58.1] - 2026-06-14
<!-- mars-harness-release: version=0.58.1 commit=9d31ea00b64d -->

### Impact
- **telemetry:** Operators see improved reliability because record guardrail loops for self improvement.

### Why
- **telemetry:** This matters because record guardrail loops for self improvement closes a failure mode or degraded path.

### What Changed
- **telemetry:** Changed record guardrail loops for self improvement (9d31ea0).

### Fixes
- **telemetry:** Record guardrail loops for self improvement (9d31ea0)

## [0.58.0] - 2026-06-14
<!-- mars-harness-release: version=0.58.0 commit=fa782db3454d -->

### Impact
- **codeintel:** Operators gain new capability: add mirrored code graph capability.
- Operators and future agents get clearer guidance because explain Go implementation language decision.

### Why
- **codeintel:** This matters because add mirrored code graph capability was missing from the shipped capability set.
- This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.

### What Changed
- **codeintel:** Changed add mirrored code graph capability (fa782db).
- Changed explain Go implementation language decision (ebc069f).

### Features
- **codeintel:** Add mirrored code graph capability (fa782db)

### Documentation
- Explain Go implementation language decision (ebc069f)

## [0.55.1] - 2026-06-13
<!-- mars-harness-release: version=0.55.1 commit=a66fd33b3b9c -->

### Impact
- **validation:** Operators and future agents get clearer guidance because start WS-D closure replay report on clean demo-11 seed.
- **validation:** Operators and future agents get clearer guidance because document ephemeral runtime validation.

### Why
- **validation:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.
- **validation:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.

### What Changed
- **validation:** Changed start WS-D closure replay report on clean demo-11 seed (24e5aa7).
- **validation:** Changed document ephemeral runtime validation (a66fd33).

### Documentation
- **validation:** Start WS-D closure replay report on clean demo-11 seed (24e5aa7)
- **validation:** Document ephemeral runtime validation (a66fd33)

## [0.55.0] - 2026-06-13
<!-- mars-harness-release: version=0.55.0 commit=d4fbff4ca284 -->

### Impact
- **serve,tools:** Operators gain new capability: break CTO ticket-gate loop and complete WS-D slices 6-8.
- Operators and future agents get clearer guidance because update operating plan for WS-D slices 4-5.
- Operators and future agents get clearer delivery behavior because an operating-model rule, boundary, or workflow contract is now explicit in repo-owned guidance.
- Operators and future agents get clearer guidance because record WS-D slice 5 in convergence state machine.

### Why
- **serve,tools:** This matters because break CTO ticket-gate loop and complete WS-D slices 6-8 was missing from the shipped capability set.
- This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.
- This matters because autonomous work needs durable routing, evidence, and ownership rules rather than relying on chat memory or implicit handoffs.
- This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.

### What Changed
- **serve,tools:** Changed break CTO ticket-gate loop and complete WS-D slices 6-8 (e87c1d4).
- Changed update operating plan for WS-D slices 4-5 (7d5c8a2).
- The operating-model guidance was updated so adjacent docs, roles, tools, evidence paths, and generated target defaults describe the new workflow consistently (d4fbff4).
- Changed record WS-D slice 5 in convergence state machine (e5e2846).

### Features
- **serve,tools:** Break CTO ticket-gate loop and complete WS-D slices 6-8 (e87c1d4)

### Documentation
- Update operating plan for WS-D slices 4-5 (7d5c8a2)
- Add foundation operating model AD-291/292 (d4fbff4)
- Record WS-D slice 5 in convergence state machine (e5e2846)

## [0.54.0] - 2026-06-13
<!-- mars-harness-release: version=0.54.0 commit=04a82852ad33 -->

### Impact
- **tools:** Operators gain new capability: wS-D slice 5 — validated phase and post-validation shell guards.

### Why
- **tools:** This matters because wS-D slice 5 — validated phase and post-validation shell guards was missing from the shipped capability set.

### What Changed
- **tools:** Changed wS-D slice 5 — validated phase and post-validation shell guards (04a8285).

### Features
- **tools:** WS-D slice 5 — validated phase and post-validation shell guards (04a8285)

## [0.53.0] - 2026-06-13
<!-- mars-harness-release: version=0.53.0 commit=128afecc0cfa -->

### Impact
- **tools:** Operators gain new capability: wS-D slice 4 — file_write and disposition DeliveryState gates.
- The release includes visible project movement: merge pull request #2 from greaveselliott/foundation-restart.

### Why
- **tools:** This matters because wS-D slice 4 — file_write and disposition DeliveryState gates was missing from the shipped capability set.
- This matters because the release should explain why merge pull request #2 from greaveselliott/foundation-restart belongs in the shipped state.

### What Changed
- **tools:** Changed wS-D slice 4 — file_write and disposition DeliveryState gates (128afec).
- Changed merge pull request #2 from greaveselliott/foundation-restart (37cd5eb).

### Features
- **tools:** WS-D slice 4 — file_write and disposition DeliveryState gates (128afec)

### Other
- Merge pull request #2 from greaveselliott/foundation-restart (37cd5eb)

### Delivery Evidence
- Enabler work: T-030: Unwedge CTO scenario-batch handoff from ticket_create false-duplicate title matching
- Enabler work: T-033: Verify profile-required model files at start so a profile swap cannot fail jobs with model_unavailable
- Enabler work: T-035: Drain or disposition pending jobs when the orchestrator is stopped so preemption cannot orphan mid-lifecycle work
- Enabler work: T-043: Carve the validation lane and repair guardrails out of the policy monolith to close the AD-287 sequence

## [0.52.0] - 2026-06-13
<!-- mars-harness-release: version=0.52.0 commit=0a01dd9699f6 -->

### Impact
- **pace:** Operators gain new capability: close T-011 with max-turn calibration and WS-D slice 3.

### Why
- **pace:** This matters because close T-011 with max-turn calibration and WS-D slice 3 was missing from the shipped capability set.

### What Changed
- **pace:** Changed close T-011 with max-turn calibration and WS-D slice 3 (0a01dd9).

### Features
- **pace:** Close T-011 with max-turn calibration and WS-D slice 3 (0a01dd9)

### Delivery Evidence
- Enabler work: T-011: Measure and optimize factory pace
- Enabler work: T-035: Drain or disposition pending jobs when the orchestrator is stopped so preemption cannot orphan mid-lifecycle work

## [0.51.1] - 2026-06-13
<!-- mars-harness-release: version=0.51.1 commit=e59ca9d82a8b -->

### Impact
- **queue:** Operators see improved reliability because preempt pending jobs on stop (T-035) and doctor RAM check (T-034).

### Why
- **queue:** This matters because preempt pending jobs on stop (T-035) and doctor RAM check (T-034) closes a failure mode or degraded path.

### What Changed
- **queue:** Changed preempt pending jobs on stop (T-035) and doctor RAM check (T-034) (e59ca9d).

### Fixes
- **queue:** Preempt pending jobs on stop (T-035) and doctor RAM check (T-034) (e59ca9d)

### Delivery Evidence
- Enabler work: T-034: Surface degraded-inference anomalies mechanically via pace telemetry and a doctor RAM-footprint check
- Enabler work: T-035: Drain or disposition pending jobs when the orchestrator is stopped so preemption cannot orphan mid-lifecycle work

## [0.51.0] - 2026-06-13
<!-- mars-harness-release: version=0.51.0 commit=5196f6d72feb -->

### Impact
- **tools:** Operators gain new capability: close T-030/T-043 and land WS-D DeliveryState slices.

### Why
- **tools:** This matters because close T-030/T-043 and land WS-D DeliveryState slices was missing from the shipped capability set.

### What Changed
- **tools:** Changed close T-030/T-043 and land WS-D DeliveryState slices (5196f6d).

### Features
- **tools:** Close T-030/T-043 and land WS-D DeliveryState slices (5196f6d)

### Delivery Evidence
- Enabler work: T-030: Unwedge CTO scenario-batch handoff from ticket_create false-duplicate title matching
- Enabler work: T-033: Verify profile-required model files at start so a profile swap cannot fail jobs with model_unavailable
- Enabler work: T-043: Carve the validation lane and repair guardrails out of the policy monolith to close the AD-287 sequence

## [0.50.25] - 2026-06-12
<!-- mars-harness-release: version=0.50.25 commit=1a1322fd4368 -->

### Impact
- **validation:** Operators and future agents get clearer guidance because record AD-287 final-checkpoint demo-12 Run 4 PASS and demo-15 pause state (T-043).

### Why
- **validation:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.

### What Changed
- **validation:** Changed record AD-287 final-checkpoint demo-12 Run 4 PASS and demo-15 pause state (T-043) (1a1322f).

### Documentation
- **validation:** Record AD-287 final-checkpoint demo-12 Run 4 PASS and demo-15 pause state (T-043) (1a1322f)

### Delivery Evidence
- Enabler work: T-001: Intervention debt: Calibrate guardrail workflow for engineer
- Enabler work: T-004: Verify foundation deployed doctrine consistency and drift gates

## [0.50.24] - 2026-06-12
<!-- mars-harness-release: version=0.50.24 commit=5cd4eb3af936 -->

### Impact
- **tools:** Operators and agents get stronger no-stale-docs enforcement because documentation sync is described and validated as part of the delivery workflow.

### Why
- **tools:** This matters because behavior changes become risky when code, BDD contracts, design docs, generated target guidance, and release notes drift apart.

### What Changed
- **tools:** The release documentation path now ties changed source files to associated docs, docsync evidence, and generated target doctrine instead of treating docs as an after-the-fact checklist (5cd4eb3).

### Maintenance
- **tools:** Extract validation-lane policy domain (AD-287 slice 8, T-043) (5cd4eb3)

### Delivery Evidence
- Enabler work: T-038: Move the shell-safety guardrail checks out of the policy monolith (AD-287 step 3)

## [0.50.23] - 2026-06-12
<!-- mars-harness-release: version=0.50.23 commit=3b73d4c5a309 -->

### Impact
- **tools:** Operators and agents get stronger no-stale-docs enforcement because documentation sync is described and validated as part of the delivery workflow.

### Why
- **tools:** This matters because behavior changes become risky when code, BDD contracts, design docs, generated target guidance, and release notes drift apart.

### What Changed
- **tools:** The release documentation path now ties changed source files to associated docs, docsync evidence, and generated target doctrine instead of treating docs as an after-the-fact checklist (3b73d4c).

### Maintenance
- **tools:** Extract disposition/handoff policy domain (AD-287 slice 7, T-042) (3b73d4c)

### Delivery Evidence
- Enabler work: T-039: Move capability and brief-parsing helpers out of the policy monolith (AD-287 step 4)
- Enabler work: T-040: Extract ticket-lifecycle policy domain into policy_ticket.go (AD-287 step 5)
- Enabler work: T-041: Give review terminal gates a dedicated policy file (AD-287 step 6)
- Enabler work: T-042: Carve job-disposition and CTO handoff gates from the monolith (AD-287 step 7)

## [0.50.22] - 2026-06-12
<!-- mars-harness-release: version=0.50.22 commit=8a8d9d32529a -->

### Impact
- **tools:** Maintainers get a healthier project surface because extract review-gates policy domain (AD-287 slice 6, T-041).

### Why
- **tools:** This matters because project health work keeps future delivery predictable.

### What Changed
- **tools:** Changed extract review-gates policy domain (AD-287 slice 6, T-041) (8a8d9d3).

### Maintenance
- **tools:** Extract review-gates policy domain (AD-287 slice 6, T-041) (8a8d9d3)

### Delivery Evidence
- Enabler work: T-040: Extract ticket-lifecycle policy domain into policy_ticket.go (AD-287 step 5)
- Enabler work: T-041: Give review terminal gates a dedicated policy file (AD-287 step 6)

## [0.50.21] - 2026-06-12
<!-- mars-harness-release: version=0.50.21 commit=9f877d85c45c -->

### Impact
- **tools:** Maintainers get a healthier project surface because extract ticket-lifecycle policy domain (AD-287 slice 5, T-040).

### Why
- **tools:** This matters because project health work keeps future delivery predictable.

### What Changed
- **tools:** Changed extract ticket-lifecycle policy domain (AD-287 slice 5, T-040) (9f877d8).

### Maintenance
- **tools:** Extract ticket-lifecycle policy domain (AD-287 slice 5, T-040) (9f877d8)

### Delivery Evidence
- Enabler work: T-038: Move the shell-safety guardrail checks out of the policy monolith (AD-287 step 3)
- Enabler work: T-039: Move capability and brief-parsing helpers out of the policy monolith (AD-287 step 4)
- Enabler work: T-040: Extract ticket-lifecycle policy domain into policy_ticket.go (AD-287 step 5)

## [0.50.20] - 2026-06-12
<!-- mars-harness-release: version=0.50.20 commit=63a37bf82dad -->

### Impact
- **tools:** Maintainers get a healthier project surface because extract capability/brief-parsing policy domain (AD-287 slice 4, T-039).

### Why
- **tools:** This matters because project health work keeps future delivery predictable.

### What Changed
- **tools:** Changed extract capability/brief-parsing policy domain (AD-287 slice 4, T-039) (63a37bf).

### Maintenance
- **tools:** Extract capability/brief-parsing policy domain (AD-287 slice 4, T-039) (63a37bf)

### Delivery Evidence
- Enabler work: T-036: Extract browser-framework static-analysis policy domain into policy_browser.go (AD-287 slice 1)
- Enabler work: T-039: Move capability and brief-parsing helpers out of the policy monolith (AD-287 step 4)

## [0.50.19] - 2026-06-12
<!-- mars-harness-release: version=0.50.19 commit=85787cfd7da3 -->

### Impact
- **tools:** Maintainers get a healthier project surface because extract shell-safety policy domain (AD-287 slice 3, T-038).

### Why
- **tools:** This matters because project health work keeps future delivery predictable.

### What Changed
- **tools:** Changed extract shell-safety policy domain (AD-287 slice 3, T-038) (85787cf).

### Maintenance
- **tools:** Extract shell-safety policy domain (AD-287 slice 3, T-038) (85787cf)

### Delivery Evidence
- Enabler work: T-038: Move the shell-safety guardrail checks out of the policy monolith (AD-287 step 3)

## [0.50.18] - 2026-06-12
<!-- mars-harness-release: version=0.50.18 commit=03122e6a8281 -->

### Impact
- **tools:** Maintainers get a healthier project surface because extract release-gate and diff/secrets policy domains (AD-287 slice 2, T-037).

### Why
- **tools:** This matters because project health work keeps future delivery predictable.

### What Changed
- **tools:** Changed extract release-gate and diff/secrets policy domains (AD-287 slice 2, T-037) (03122e6).

### Maintenance
- **tools:** Extract release-gate and diff/secrets policy domains (AD-287 slice 2, T-037) (03122e6)

### Delivery Evidence
- Enabler work: T-037: Extract release-gate and diff/secrets policy domains into policy_release.go and policy_diff.go (AD-287 slice 2)

## [0.50.17] - 2026-06-12
<!-- mars-harness-release: version=0.50.17 commit=a1f9b9f74d5d -->

### Impact
- **validation:** Operators and future agents get clearer guidance because record demo-12 AD-287 slice-1 checkpoint replay and close T-036.

### Why
- **validation:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.

### What Changed
- **validation:** Changed record demo-12 AD-287 slice-1 checkpoint replay and close T-036 (a1f9b9f).

### Documentation
- **validation:** Record demo-12 AD-287 slice-1 checkpoint replay and close T-036 (a1f9b9f)

### Delivery Evidence
- Enabler work: T-001: Intervention debt: Calibrate guardrail workflow for engineer
- Enabler work: T-002: Document foundation and deployed harness architecture
- Enabler work: T-036: Extract browser-framework static-analysis policy domain into policy_browser.go (AD-287 slice 1)

## [0.50.16] - 2026-06-12
<!-- mars-harness-release: version=0.50.16 commit=f5a1d6a7c218 -->

### Impact
- **tools:** Maintainers get a healthier project surface because extract browser-framework policy domain into policy_browser.go (AD-287 slice 1, T-036).

### Why
- **tools:** This matters because project health work keeps future delivery predictable.

### What Changed
- **tools:** Changed extract browser-framework policy domain into policy_browser.go (AD-287 slice 1, T-036) (f5a1d6a).

### Maintenance
- **tools:** Extract browser-framework policy domain into policy_browser.go (AD-287 slice 1, T-036) (f5a1d6a)

## [0.50.15] - 2026-06-12
<!-- mars-harness-release: version=0.50.15 commit=27b28a3889a3 -->

### Impact
- **validation:** Operators and future agents get clearer guidance because record demo-14 AD-289 replay evidence and close T-031.

### Why
- **validation:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.

### What Changed
- **validation:** Changed record demo-14 AD-289 replay evidence and close T-031 (27b28a3).

### Documentation
- **validation:** Record demo-14 AD-289 replay evidence and close T-031 (27b28a3)

### Delivery Evidence
- Enabler work: T-001: Intervention debt: Calibrate guardrail workflow for engineer
- Enabler work: T-031: Harden qa and dogfood terminal convergence so circle_detected runtime failures do not cap lifecycle reach

## [0.50.14] - 2026-06-12
<!-- mars-harness-release: version=0.50.14 commit=26d4f391db52 -->

### Impact
- **validation:** Operators and future agents get clearer guidance because fold independent monitor T-032 cross-check into demo-12/demo-13 observer evidence.

### Why
- **validation:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.

### What Changed
- **validation:** Changed fold independent monitor T-032 cross-check into demo-12/demo-13 observer evidence (26d4f39).

### Documentation
- **validation:** Fold independent monitor T-032 cross-check into demo-12/demo-13 observer evidence (26d4f39)

### Delivery Evidence
- Enabler work: T-001: Intervention debt: Calibrate guardrail workflow for engineer
- Enabler work: T-032: Fit engineer context assembly inside the balanced coding-tier window for package-managed frontend targets

## [0.50.13] - 2026-06-12
<!-- mars-harness-release: version=0.50.13 commit=222392cbdf16 -->

### Impact
- **serve:** Operators see improved reliability because give runtime convergence failures one bounded automatic retry per fingerprint (T-031, AD-289).

### Why
- **serve:** This matters because give runtime convergence failures one bounded automatic retry per fingerprint (T-031, AD-289) closes a failure mode or degraded path.

### What Changed
- **serve:** Changed give runtime convergence failures one bounded automatic retry per fingerprint (T-031, AD-289) (222392c).

### Fixes
- **serve:** Give runtime convergence failures one bounded automatic retry per fingerprint (T-031, AD-289) (222392c)

## [0.50.12] - 2026-06-12
<!-- mars-harness-release: version=0.50.12 commit=dc32dca28072 -->

### Impact
- **validation:** Operators and future agents get clearer guidance because record T-032 replay evidence and close the ticket (AD-288).

### Why
- **validation:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.

### What Changed
- **validation:** Changed record T-032 replay evidence and close the ticket (AD-288) (dc32dca).

### Documentation
- **validation:** Record T-032 replay evidence and close the ticket (AD-288) (dc32dca)

### Delivery Evidence
- Enabler work: T-001: Intervention debt: Calibrate guardrail workflow for engineer
- Enabler work: T-002: Document foundation and deployed harness architecture
- Enabler work: T-032: Fit engineer context assembly inside the balanced coding-tier window for package-managed frontend targets

## [0.50.11] - 2026-06-12
<!-- mars-harness-release: version=0.50.11 commit=bee4f5bbb283 -->

### Impact
- **agent:** Operators see improved reliability because clamp context budgeting to the served inference window (T-032, AD-288).

### Why
- **agent:** This matters because clamp context budgeting to the served inference window (T-032, AD-288) closes a failure mode or degraded path.

### What Changed
- **agent:** Changed clamp context budgeting to the served inference window (T-032, AD-288) (bee4f5b).

### Fixes
- **agent:** Clamp context budgeting to the served inference window (T-032, AD-288) (bee4f5b)

## [0.50.10] - 2026-06-12
<!-- mars-harness-release: version=0.50.10 commit=1b0ad79aa040 -->

### Impact
- **plan:** Operators and future agents get clearer guidance because sequence T-032 context overflow as first Phase 3 slice ahead of T-031 routing and extraction work.

### Why
- **plan:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.

### What Changed
- **plan:** Changed sequence T-032 context overflow as first Phase 3 slice ahead of T-031 routing and extraction work (1b0ad79).

### Documentation
- **plan:** Sequence T-032 context overflow as first Phase 3 slice ahead of T-031 routing and extraction work (1b0ad79)

## [0.50.9] - 2026-06-12
<!-- mars-harness-release: version=0.50.9 commit=00b556666b63 -->

### Impact
- **design:** Operators and future agents get clearer guidance because add AD-287 policy.go decomposition AD with ordered same-package extraction sequence (T-030).

### Why
- **design:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.

### What Changed
- **design:** Changed add AD-287 policy.go decomposition AD with ordered same-package extraction sequence (T-030) (00b5566).

### Documentation
- **design:** Add AD-287 policy.go decomposition AD with ordered same-package extraction sequence (T-030) (00b5566)

## [0.50.8] - 2026-06-12
<!-- mars-harness-release: version=0.50.8 commit=019a67ed15a0 -->

### Impact
- **design:** Operators and future agents get clearer guidance because add AD-286 convergence state-machine design doc mapping AD-164..AD-275 onto explicit delivery states and transitions (T-028).

### Why
- **design:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.

### What Changed
- **design:** Changed add AD-286 convergence state-machine design doc mapping AD-164..AD-275 onto explicit delivery states and transitions (T-028) (019a67e).

### Documentation
- **design:** Add AD-286 convergence state-machine design doc mapping AD-164..AD-275 onto explicit delivery states and transitions (T-028) (019a67e)

### Delivery Evidence
- Enabler work: T-028: Define matrix-gating doctrine for source-change classes and validation evidence

## [0.50.7] - 2026-06-12
<!-- mars-harness-release: version=0.50.7 commit=771dc5960d27 -->

### Impact
- **validation:** Operators and future agents get clearer guidance because correct demo-11 baseline stop-reason and lifecycle-health claims from second monitor shift (T-011, T-031, T-035).

### Why
- **validation:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.

### What Changed
- **validation:** Changed correct demo-11 baseline stop-reason and lifecycle-health claims from second monitor shift (T-011, T-031, T-035) (771dc59).

### Documentation
- **validation:** Correct demo-11 baseline stop-reason and lifecycle-health claims from second monitor shift (T-011, T-031, T-035) (771dc59)

## [0.50.6] - 2026-06-12
<!-- mars-harness-release: version=0.50.6 commit=43f11b44847b -->

### Impact
- **validation:** Operators and future agents get clearer guidance because fold independent replay-monitor evidence into baseline reports and record RAM-pressure discovery (T-031, T-033, T-034).

### Why
- **validation:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.

### What Changed
- **validation:** Changed fold independent replay-monitor evidence into baseline reports and record RAM-pressure discovery (T-031, T-033, T-034) (43f11b4).

### Documentation
- **validation:** Fold independent replay-monitor evidence into baseline reports and record RAM-pressure discovery (T-031, T-033, T-034) (43f11b4)

## [0.50.5] - 2026-06-12
<!-- mars-harness-release: version=0.50.5 commit=2517626f590e -->

### Impact
- **validation:** Operators and future agents get clearer guidance because record frontend and maintenance archetype baselines, close T-029 (T-029, T-032).

### Why
- **validation:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.

### What Changed
- **validation:** Changed record frontend and maintenance archetype baselines, close T-029 (T-029, T-032) (2517626).

### Documentation
- **validation:** Record frontend and maintenance archetype baselines, close T-029 (T-029, T-032) (2517626)

### Delivery Evidence
- Enabler work: T-029: Close validation archetype baseline gaps

## [0.50.4] - 2026-06-12
<!-- mars-harness-release: version=0.50.4 commit=3d62536b9939 -->

### Impact
- **validation:** Operators and future agents get clearer guidance because record balanced-model factory-pace baseline from demo-11 full lifecycle (T-011).

### Why
- **validation:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.

### What Changed
- **validation:** Changed record balanced-model factory-pace baseline from demo-11 full lifecycle (T-011) (3d62536).

### Documentation
- **validation:** Record balanced-model factory-pace baseline from demo-11 full lifecycle (T-011) (3d62536)

### Delivery Evidence
- Enabler work: T-001: Intervention debt: Calibrate guardrail workflow for engineer
- Enabler work: T-027: Promote convergence failures and guardrail block rates into scores export

## [0.50.3] - 2026-06-12
<!-- mars-harness-release: version=0.50.3 commit=932695e7578f -->

### Impact
- **validation:** Operators and future agents get clearer guidance because reclassify heavy-model demo-11 baseline as evidence-only and add model identity to AD-285 (T-011).

### Why
- **validation:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.

### What Changed
- **validation:** Changed reclassify heavy-model demo-11 baseline as evidence-only and add model identity to AD-285 (T-011) (932695e).

### Documentation
- **validation:** Reclassify heavy-model demo-11 baseline as evidence-only and add model identity to AD-285 (T-011) (932695e)

## [0.50.2] - 2026-06-11
<!-- mars-harness-release: version=0.50.2 commit=ff3d42d80f32 -->

### Impact
- **validation:** Operators and future agents get clearer guidance because record dated factory-pace baseline from demo-11 Inventory/API replay (T-011 measurement floor).

### Why
- **validation:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.

### What Changed
- **validation:** Changed record dated factory-pace baseline from demo-11 Inventory/API replay (T-011 measurement floor) (ff3d42d).

### Documentation
- **validation:** Record dated factory-pace baseline from demo-11 Inventory/API replay (T-011 measurement floor) (ff3d42d)

### Delivery Evidence
- Enabler work: T-027: Promote convergence failures and guardrail block rates into scores export

## [0.50.1] - 2026-06-11
<!-- mars-harness-release: version=0.50.1 commit=4f663c010cb3 -->

### Impact
- **validation:** Operators and future agents get clearer guidance because gate source-change classes on minimum archetype replays with a fixed evidence contract (T-028).

### Why
- **validation:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.

### What Changed
- **validation:** Changed gate source-change classes on minimum archetype replays with a fixed evidence contract (T-028) (4f663c0).

### Documentation
- **validation:** Gate source-change classes on minimum archetype replays with a fixed evidence contract (T-028) (4f663c0)

### Delivery Evidence
- Enabler work: T-028: Define matrix-gating doctrine for source-change classes and validation evidence

## [0.50.0] - 2026-06-11
<!-- mars-harness-release: version=0.50.0 commit=f53bb74ea98d -->

### Impact
- **qualityscore:** Operators gain new capability: break out convergence failures and guardrail block rates in scores export (T-027).

### Why
- **qualityscore:** This matters because break out convergence failures and guardrail block rates in scores export (T-027) was missing from the shipped capability set.

### What Changed
- **qualityscore:** Changed break out convergence failures and guardrail block rates in scores export (T-027) (f53bb74).

### Features
- **qualityscore:** Break out convergence failures and guardrail block rates in scores export (T-027) (f53bb74)

### Delivery Evidence
- Enabler work: T-027: Promote convergence failures and guardrail block rates into scores export

## [0.49.0] - 2026-06-11
<!-- mars-harness-release: version=0.49.0 commit=d13e68eff964 -->

### Impact
- **release:** Operators gain new capability: add release audit command to detect notes-only and missing GitHub releases (T-026).

### Why
- **release:** This matters because add release audit command to detect notes-only and missing GitHub releases (T-026) was missing from the shipped capability set.

### What Changed
- **release:** Changed add release audit command to detect notes-only and missing GitHub releases (T-026) (d13e68e).

### Features
- **release:** Add release audit command to detect notes-only and missing GitHub releases (T-026) (d13e68e)

### Delivery Evidence
- Enabler work: T-026: Make the release pipeline self-verifying with a release audit command

## [0.48.0] - 2026-06-11
<!-- mars-harness-release: version=0.48.0 commit=b0e62051710c -->

### Impact
- **quality:** Operators gain new capability: fuzz hostile model output parsers and gate the module with govulncheck (T-025).

### Why
- **quality:** This matters because fuzz hostile model output parsers and gate the module with govulncheck (T-025) was missing from the shipped capability set.

### What Changed
- **quality:** Changed fuzz hostile model output parsers and gate the module with govulncheck (T-025) (b0e6205).

### Features
- **quality:** Fuzz hostile model output parsers and gate the module with govulncheck (T-025) (b0e6205)

### Delivery Evidence
- Enabler work: T-025: Add govulncheck and fuzz targets for hostile model output parsers

## [0.47.0] - 2026-06-11
<!-- mars-harness-release: version=0.47.0 commit=51719fa4e1b9 -->

### Impact
- **quality:** Operators gain new capability: add per-package coverage ratchet gate to the local delivery gate (T-024).

### Why
- **quality:** This matters because add per-package coverage ratchet gate to the local delivery gate (T-024) was missing from the shipped capability set.

### What Changed
- **quality:** Changed add per-package coverage ratchet gate to the local delivery gate (T-024) (51719fa).

### Features
- **quality:** Add per-package coverage ratchet gate to the local delivery gate (T-024) (51719fa)

### Delivery Evidence
- Enabler work: T-024: Add per-package coverage ratchet gate to CI

## [0.46.4] - 2026-06-11
<!-- mars-harness-release: version=0.46.4 commit=39fdf6d0d908 -->

### Impact
- **release:** Operators and future agents get clearer guidance because record divergent-branch version-collision incident and TD-008 guard.

### Why
- **release:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.

### What Changed
- **release:** Changed record divergent-branch version-collision incident and TD-008 guard (39fdf6d).

### Documentation
- **release:** Record divergent-branch version-collision incident and TD-008 guard (39fdf6d)

## [0.46.3] - 2026-06-11
<!-- mars-harness-release: version=0.46.3 commit=538877f02c88 -->

### Impact
- **scoring:** Operators see improved reliability because evaluate score windows against the caller's reference time.

### Why
- **scoring:** This matters because evaluate score windows against the caller's reference time closes a failure mode or degraded path.

### What Changed
- **scoring:** Changed evaluate score windows against the caller's reference time (538877f).

### Fixes
- **scoring:** Evaluate score windows against the caller's reference time (538877f)

## [0.46.2] - 2026-06-11
<!-- mars-harness-release: version=0.46.2 commit=082f0916c508 -->

### Impact
- **dashboard:** Operators and future agents get clearer guidance because record AD-279 constraint scope and defer epic until T-011 closes (T-023).

### Why
- **dashboard:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.

### What Changed
- **dashboard:** Changed record AD-279 constraint scope and defer epic until T-011 closes (T-023) (082f091).

### Documentation
- **dashboard:** Record AD-279 constraint scope and defer epic until T-011 closes (T-023) (082f091)

### Delivery Evidence
- Enabler work: T-023: Record dashboard architecture decision and schedule-or-defer outcome

## [0.46.1] - 2026-06-11
<!-- mars-harness-release: version=0.46.1 commit=ca7d214045d3 -->

### Impact
- **plan:** Operators and future agents get clearer guidance because extract release-blocker ledger to validation evidence (T-022).

### Why
- **plan:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.

### What Changed
- **plan:** Changed extract release-blocker ledger to validation evidence (T-022) (ca7d214).

### Documentation
- **plan:** Extract release-blocker ledger to validation evidence (T-022) (ca7d214)

### Delivery Evidence
- Enabler work: T-022: Slim active plan by extracting release-blocker ledger to validation evidence

## [0.46.0] - 2026-06-11
<!-- mars-harness-release: version=0.46.0 commit=2cd9f4df91f8 -->

### Impact
- **qualityscore:** Operators gain new capability: define quality-score regeneration cadence with AD-278 (T-021).

### Why
- **qualityscore:** This matters because define quality-score regeneration cadence with AD-278 (T-021) was missing from the shipped capability set.

### What Changed
- **qualityscore:** Changed define quality-score regeneration cadence with AD-278 (T-021) (2cd9f4d).

### Features
- **qualityscore:** Define quality-score regeneration cadence with AD-278 (T-021) (2cd9f4d)

### Delivery Evidence
- Enabler work: T-021: Define QUALITY_SCORE regeneration cadence via scores export

## [0.45.4] - 2026-06-11
<!-- mars-harness-release: version=0.45.4 commit=4543475447fc -->

### Impact
- **hygiene:** Operators and future agents get clearer guidance because retire prompt-port-status and fix quickstart drift with AD-277 (T-020).

### Why
- **hygiene:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.

### What Changed
- **hygiene:** Changed retire prompt-port-status and fix quickstart drift with AD-277 (T-020) (4543475).

### Documentation
- **hygiene:** Retire prompt-port-status and fix quickstart drift with AD-277 (T-020) (4543475)

### Delivery Evidence
- Enabler work: T-020: Retire prompt-port-status and reconcile quickstart command drift

## [0.45.3] - 2026-06-11
<!-- mars-harness-release: version=0.45.3 commit=fb535fc82311 -->

### Impact
- **hygiene:** Operators and future agents get clearer guidance because retire pipeline-learnings tracker with AD-276 (T-019).

### Why
- **hygiene:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.

### What Changed
- **hygiene:** Changed retire pipeline-learnings tracker with AD-276 (T-019) (fb535fc).

### Documentation
- **hygiene:** Retire pipeline-learnings tracker with AD-276 (T-019) (fb535fc)

### Delivery Evidence
- Enabler work: T-019: Retire pipeline-learnings standing tracker with recorded decision

## [0.45.2] - 2026-06-11
<!-- mars-harness-release: version=0.45.2 commit=8ce8d95524fa -->

### Impact
- **plan:** Operators and future agents get clearer guidance because register foundation improvement workstreams and WS-A/WS-B tickets.

### Why
- **plan:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.

### What Changed
- **plan:** Changed register foundation improvement workstreams and WS-A/WS-B tickets (8ce8d95).

### Documentation
- **plan:** Register foundation improvement workstreams and WS-A/WS-B tickets (8ce8d95)

## [0.45.1] - 2026-05-24
<!-- mars-harness-release: version=0.45.1 commit=43a575dd0483 -->

### Impact
- **references:** Operators and future agents get clearer guidance because add AI engineering reading list.

### Why
- **references:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.

### What Changed
- **references:** Changed add AI engineering reading list (43a575d).

### Documentation
- **references:** Add AI engineering reading list (43a575d)

## [0.45.0] - 2026-05-24
<!-- mars-harness-release: version=0.45.0 commit=f0065ad06d42 -->

### Impact
- **onboarding:** Operators gain new capability: add source checkout update path.

### Why
- **onboarding:** This matters because add source checkout update path was missing from the shipped capability set.

### What Changed
- **onboarding:** Changed add source checkout update path (f0065ad).

### Features
- **onboarding:** Add source checkout update path (f0065ad)

## [0.44.3] - 2026-05-23
<!-- mars-harness-release: version=0.44.3 commit=3601809bfd4a -->

### Impact
- **architecture:** Operators and future agents get clearer guidance because add local delivery diagrams.

### Why
- **architecture:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.

### What Changed
- **architecture:** Changed add local delivery diagrams (3601809).

### Documentation
- **architecture:** Add local delivery diagrams (3601809)

## [0.44.2] - 2026-05-23
<!-- mars-harness-release: version=0.44.2 commit=e712ec44dd81 -->

### Impact
- **tools:** Operators see improved reliability because scope cto handoff to active plan feature.

### Why
- **tools:** This matters because scope cto handoff to active plan feature closes a failure mode or degraded path.

### What Changed
- **tools:** Changed scope cto handoff to active plan feature (e712ec4).

### Fixes
- **tools:** Scope cto handoff to active plan feature (e712ec4)

## [0.44.1] - 2026-05-23
<!-- mars-harness-release: version=0.44.1 commit=7343cee455f8 -->

### Impact
- **ui:** Operators see improved reliability because expose model wait phase in terminal dashboard.

### Why
- **ui:** This matters because expose model wait phase in terminal dashboard closes a failure mode or degraded path.

### What Changed
- **ui:** Changed expose model wait phase in terminal dashboard (7343cee).

### Fixes
- **ui:** Expose model wait phase in terminal dashboard (7343cee)

## [0.44.0] - 2026-05-23
<!-- mars-harness-release: version=0.44.0 commit=41f858359867 -->

### Impact
- **release:** Operators gain new capability: move delivery gates local.

### Why
- **release:** This matters because move delivery gates local was missing from the shipped capability set.

### What Changed
- **release:** Changed move delivery gates local (41f8583).

### Features
- **release:** Move delivery gates local (41f8583)

## [0.43.2] - 2026-05-23
<!-- mars-harness-release: version=0.43.2 commit=d21df208ce43 -->

### Impact
- **update:** Operators see improved reliability because private release auth fallback caching.

### Why
- **update:** This matters because private release auth fallback caching closes a failure mode or degraded path.

### What Changed
- **update:** Changed private release auth fallback caching (d21df20).

### Fixes
- **update:** Private release auth fallback caching (d21df20)

## [0.43.1] - 2026-05-23
<!-- mars-harness-release: version=0.43.1 commit=491c7593cbc7 -->

### Impact
- **tools:** Operators see improved reliability because satisfy browser smoke lint.

### Why
- **tools:** This matters because satisfy browser smoke lint closes a failure mode or degraded path.

### What Changed
- **tools:** Changed satisfy browser smoke lint (491c759).

### Fixes
- **tools:** Satisfy browser smoke lint (491c759)

## [0.43.0] - 2026-05-23
<!-- mars-harness-release: version=0.43.0 commit=0b023fe637ae -->

### Impact
- **foundation:** Operators gain new capability: add vendor-neutral foundation role.

### Why
- **foundation:** This matters because add vendor-neutral foundation role was missing from the shipped capability set.

### What Changed
- **foundation:** Changed add vendor-neutral foundation role (0b023fe).

### Features
- **foundation:** Add vendor-neutral foundation role (0b023fe)

## [0.42.28] - 2026-05-23
<!-- mars-harness-release: version=0.42.28 commit=35fb75a21a38 -->

### Impact
- **lifecycle:** Product-first planning is less likely to stall on named-demo vocabulary: target product names and readable outcome prose no longer become phantom capabilities when the concrete behaviors are already covered.

### Why
- **lifecycle:** The live improvement loop showed that useful demo evidence can accidentally become product-specific foundation doctrine. Capability matching needed to keep the reusable lesson while removing hardcoded demo names and object synonyms from runtime policy.

### What Changed
- **lifecycle:** Added dynamic product-label stripping for brief-derived capability checks, ignored readable outcome glue such as see/useful/usable/playable, and removed product-specific global stopwords or object-noun mappings from generic policy.
- **lifecycle:** Updated foundation doctrine and feature contracts to state that representative demo names are evidence anchors only; reusable rules must be expressed by failure class, project class, or stack class.
- **lifecycle:** Cleaned the generic generated/role guidance surface so no Tetris or tetromino vocabulary remains outside historical evidence and test fixtures.

### Fixes
- **lifecycle:** Genericize demo capability matching (35fb75a)

## [0.42.27] - 2026-05-22
<!-- mars-harness-release: version=0.42.27 commit=9878b4107790 -->

### Impact
- **lifecycle:** Fresh target bootstraps now move past CEO/COO planning into CTO-created product tickets instead of looping on planning guardrails when a brief excludes polish, previews, sound, or similar optional extensions.

### Why
- **lifecycle:** The live Tetris replay loop showed two foundation-owned blockers after the previous stabilization: COO tried to create a sibling active-plan file for the current failing scenario, and optional Out-of-Scope lines such as animation polish or next-piece preview were interpreted as descoping covered core gameplay. Those loops kept the factory from reaching implementation even though the target product plan was valid.

### What Changed
- **lifecycle:** Added a single-active-plan policy error and mirrored COO guidance so current-failing-scenario recovery updates `docs/exec-plans/active/current-operating-plan.md` instead of creating a second active plan.
- **lifecycle:** Broadened Out-of-Scope parsing so enhancement-only exclusions for animation/visual polish, optional previews, sound/audio, multiplayer, mobile touch controls, hold-piece, and hard-drop variants do not descope already-covered basic capabilities.
- **lifecycle:** Recorded live `demo-tetris-64` through `demo-tetris-68` dogfood evidence; the confirmation replay completed CEO, COO, and CTO, created ordinary product tickets, and reached Engineer implementation.

### Fixes
- **lifecycle:** Unblock product-first planning handoff (9878b41)

## [0.42.26] - 2026-05-22
<!-- mars-harness-release: version=0.42.26 commit=262aa0960d04 -->

### Impact
- **lifecycle:** Operators see improved reliability because stabilize product-first demo loop.

### Why
- **lifecycle:** This matters because stabilize product-first demo loop closes a failure mode or degraded path.

### What Changed
- **lifecycle:** Changed stabilize product-first demo loop (262aa09).

### Fixes
- **lifecycle:** Stabilize product-first demo loop (262aa09)

## [0.42.25] - 2026-05-22
<!-- mars-harness-release: version=0.42.25 commit=da7779359b0d -->

### Impact
- **orchestration:** The foundation returns to the previously validated lifecycle behavior before starting a broader live project-completion loop.

### Why
- **orchestration:** The post-ticket-gate loop fix was too narrow for the next objective; the factory now needs evidence from a complete Phaser Tetris build before accepting more lifecycle changes.

### What Changed
- **orchestration:** Reverted the ticket-shaping loop guard, generated CTO prompt changes, scenario-ticketing contract edits, and empty `bdd_scenarios` repair text from `v0.42.24` (da77793).

### Other
- **orchestration:** Scrap ticket shaping loop fix (da77793)

## [0.42.24] - 2026-05-22
<!-- mars-harness-release: version=0.42.24 commit=1d1e4323f5eb -->

### Impact
- **orchestration:** Fresh and continuing target runs no longer burn turns in a repeated CTO ticket-shaping loop after the first product ticket is done.

### Why
- **orchestration:** The live `demo-6` run showed one completed product ticket followed by repeated CTO handoffs with no ticket-state change; this release keeps product planning moving toward the next uncovered BDD scenario or stops with a clear blocker.

### What Changed
- **orchestration:** Added a post-ticket-gate loop guard, tightened CTO generated guidance around current-or-next uncovered scenarios, and improved empty `bdd_scenarios` repair guidance (1d1e432).

### Fixes
- **orchestration:** Stop repeated ticket shaping loops (1d1e432)

## [0.42.23] - 2026-05-21
<!-- mars-harness-release: version=0.42.23 commit=8309291edb20 -->

### Impact
- **guardrails:** Operators see improved reliability because ignore workspace metadata noise.

### Why
- **guardrails:** This matters because ignore workspace metadata noise closes a failure mode or degraded path.

### What Changed
- **guardrails:** Changed ignore workspace metadata noise (8309291).

### Fixes
- **guardrails:** Ignore workspace metadata noise (8309291)

## [0.42.22] - 2026-05-21
<!-- mars-harness-release: version=0.42.22 commit=1fe1dd535711 -->

### Impact
- **tools:** Operators see improved reliability because clear policy lint blockers.

### Why
- **tools:** This matters because clear policy lint blockers closes a failure mode or degraded path.

### What Changed
- **tools:** Changed clear policy lint blockers (1fe1dd5).

### Fixes
- **tools:** Clear policy lint blockers (1fe1dd5)

## [0.42.21] - 2026-05-21
<!-- mars-harness-release: version=0.42.21 commit=a9d602f4f728 -->

### Impact
- **tools:** Operators see improved reliability because preserve ticket evidence path case.

### Why
- **tools:** This matters because preserve ticket evidence path case closes a failure mode or degraded path.

### What Changed
- **tools:** Changed preserve ticket evidence path case (a9d602f).

### Fixes
- **tools:** Preserve ticket evidence path case (a9d602f)

## [0.42.20] - 2026-05-21
<!-- mars-harness-release: version=0.42.20 commit=ae14dea916b2 -->

### Impact
- **tools:** Operators see improved reliability because synchronize background output capture.

### Why
- **tools:** This matters because synchronize background output capture closes a failure mode or degraded path.

### What Changed
- **tools:** Changed synchronize background output capture (ae14dea).

### Fixes
- **tools:** Synchronize background output capture (ae14dea)

## [0.42.19] - 2026-05-21
<!-- mars-harness-release: version=0.42.19 commit=7e67f374daef -->

### Impact
- **tools:** Operators see improved reliability because converge post-validation no-op work.

### Why
- **tools:** This matters because converge post-validation no-op work closes a failure mode or degraded path.

### What Changed
- **tools:** Changed converge post-validation no-op work (7e67f37).

### Fixes
- **tools:** Converge post-validation no-op work (7e67f37)

## [0.42.18] - 2026-05-21
<!-- mars-harness-release: version=0.42.18 commit=a7c46aab914b -->

### Impact
- **guardrails:** Operators see improved reliability because preserve test evidence during repair.

### Why
- **guardrails:** This matters because preserve test evidence during repair closes a failure mode or degraded path.

### What Changed
- **guardrails:** Changed preserve test evidence during repair (a7c46aa).

### Fixes
- **guardrails:** Preserve test evidence during repair (a7c46aa)

## [0.42.17] - 2026-05-21
<!-- mars-harness-release: version=0.42.17 commit=9c963901f887 -->

### Impact
- **lifecycle:** Operators see improved reliability because stabilize continuous factory loop.

### Why
- **lifecycle:** This matters because stabilize continuous factory loop closes a failure mode or degraded path.

### What Changed
- **lifecycle:** Changed stabilize continuous factory loop (9c96390).

### Fixes
- **lifecycle:** Stabilize continuous factory loop (9c96390)

## [0.42.16] - 2026-05-20
<!-- mars-harness-release: version=0.42.16 commit=b7cc36ed8138 -->

### Impact
- **dashboard:** Operators and future agents get clearer guidance because plan TanStack control plane epic.

### Why
- **dashboard:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.

### What Changed
- **dashboard:** Changed plan TanStack control plane epic (b7cc36e).

### Documentation
- **dashboard:** Plan TanStack control plane epic (b7cc36e)

## [0.42.15] - 2026-05-20
<!-- mars-harness-release: version=0.42.15 commit=1b0016acffa0 -->

### Impact
- **lifecycle:** Operators see improved reliability because stabilize product-first validation loop.

### Why
- **lifecycle:** This matters because stabilize product-first validation loop closes a failure mode or degraded path.

### What Changed
- **lifecycle:** Changed stabilize product-first validation loop (1b0016a).

### Fixes
- **lifecycle:** Stabilize product-first validation loop (1b0016a)

## [0.42.14] - 2026-05-20
<!-- mars-harness-release: version=0.42.14 commit=05556832502e -->

### Impact
- **orchestration:** Fresh target runs can no longer approve, dogfood, or release source code while `docsync_audit` reports missing or invalid `MarsDocSync` metadata, reducing stale-doc escapes in the autonomous lifecycle.
- **telemetry:** Policy-blocked external `timeout` validation commands now stay as guardrail evidence instead of spawning duplicate retry work.

### Why
- **orchestration:** The `demo-api-run14` canary showed Engineer completed the product ticket successfully, but QA, Security, and Dogfood observed DocSync failures and still approved. That made documentation freshness a judgement call at exactly the point it needed to be a mechanical gate.
- **telemetry:** The same canary showed a rejected external `timeout` wrapper classified as retryable `tool_timeout`, enqueueing a duplicate Dogfood job for a deterministic policy block.

### What Changed
- **orchestration:** Successful Engineer, Pipeline Fixer, QA, Security, Dogfood, Release Manager, and Dependency Manager dispositions now run DocSync and are rejected while findings exist; generated target guidance tells roles that scenario IDs are not feature-contract file paths and that `FAIL:` output requires rework or a blocked disposition (0555683).
- **docsync:** The audit now distinguishes the foundation source checkout from deployed target repos, so target Go apps under layouts such as `cmd/` still need valid metadata but are not forced to cite Mars Harness foundation-only expected docs (0555683).
- **telemetry:** Guardrail and tool-policy matches are classified before generic timeout matching, preventing policy-blocked `timeout` wrappers from entering retry remediation (0555683).

### Fixes
- **orchestration:** Block handoffs on docsync failures (0555683)

## [0.42.13] - 2026-05-20
<!-- mars-harness-release: version=0.42.13 commit=a95a9c0440c0 -->

### Impact
- **tools:** Fresh target runs are less likely to stall after successful live
  validation because empty `shell_exec` argv and single `:` no-op calls now
  redirect agents toward cleanup, ticket completion, commit, push, and
  `job_disposition_record` instead of producing guardrail-loop failures.

### Why
- **tools:** The `demo-api-run13` canary proved the tracked-background cleanup
  fix worked: Engineer started and killed a managed `go run` server, then
  launched the external `/tmp` validation binary successfully. The remaining
  failure was model-shape drift after validation had passed: empty argv and `:`
  calls consumed turns and triggered `circle_detected` before the ticket could
  move to done.

### What Changed
- **tools:** `shell_exec` now treats empty argv, blank argv, and single `:` calls
  as no-op recovery hints, names active tracked background PIDs, and reminds the
  role to stop the PID, update ticket evidence, commit, push, and record a
  disposition. Managed background startup output and generated Engineer guidance
  now carry the same rule (a95a9c0).

### Fixes
- **tools:** Guide shell no-op loops to completion (a95a9c0)

## [0.42.12] - 2026-05-20
<!-- mars-harness-release: version=0.42.12 commit=7b9d8e79e8a1 -->

### Impact
- **tools:** Operators see improved reliability because kill tracked background process trees.

### Why
- **tools:** This matters because kill tracked background process trees closes a failure mode or degraded path.

### What Changed
- **tools:** Changed kill tracked background process trees (7b9d8e7).

### Fixes
- **tools:** Kill tracked background process trees (7b9d8e7)

## [0.42.11] - 2026-05-20
<!-- mars-harness-release: version=0.42.11 commit=f7b5f48aaa6e -->

### Impact
- **tools:** Operators see improved reliability because block implicit go build artifacts.

### Why
- **tools:** This matters because block implicit go build artifacts closes a failure mode or degraded path.

### What Changed
- **tools:** Changed block implicit go build artifacts (f7b5f48).

### Fixes
- **tools:** Block implicit go build artifacts (f7b5f48)

## [0.42.10] - 2026-05-20
<!-- mars-harness-release: version=0.42.10 commit=aafa166a4f24 -->

### Impact
- **tools:** Live target runs are less likely to inherit stale dev servers from earlier validation jobs, reducing false port conflicts and follow-on command loops.

### Why
- **tools:** The `demo-api-run10` canary found that killing the tracked `go run` wrapper was not enough: the compiled child server could survive on port 8080 and derail the next Engineer validation. Background cleanup needs to own the whole process tree it starts.

### What Changed
- **tools:** Background cleanup now discovers known descendants, kills them from leaf to root, then terminates the tracked process group and process; coverage includes an escaped-child regression test (aafa166).

### Fixes
- **tools:** Clean background process descendants (aafa166)

## [0.42.9] - 2026-05-20
<!-- mars-harness-release: version=0.42.9 commit=053a6e20d9c1 -->

### Impact
- **tools:** Fresh target runs are less likely to pollute product commits with throwaway validation scripts, and service validation now avoids host-specific `timeout` utilities.

### Why
- **tools:** The live Task Notes API replay reached local release notes but exposed a generic validation trap: Engineer committed a root `validate.sh` that Dogfood later proved was non-portable. The factory should keep temporary proof work in tool-managed commands or durable tests, not accidental product surface.

### What Changed
- **tools:** Blocks new repo-root validation shell scripts such as `validate.sh`, rejects external `timeout`/`gtimeout` commands, and mirrors the portable validation rule into generated Engineer and Dogfood guidance (053a6e2).

### Fixes
- **tools:** Block scratch validation scripts (053a6e2)

## [0.42.8] - 2026-05-20
<!-- mars-harness-release: version=0.42.8 commit=1aa405f613e5 -->

### Impact
- **tools:** Agents get a clearer recovery path when service validation drifts
  into malformed port-only commands, reducing short `:8080` loops after build or
  server errors.

### Why
- **tools:** The `demo-api-run8` canary showed the build-output guardrail
  working, but Engineer then called `shell_exec` with `argv:[":8080"]` twice and
  hit `circle_detected` instead of returning to a real server command or curl
  probe.

### What Changed
- **tools:** `shell_exec` now rejects bare port tokens such as `:8080` in argv
  and single-token shell-command mode, explains that ports are not executable
  commands, points roles to `background:true` plus curl probes, improves the
  repo-local build-output recovery hint, and mirrors the port-probe rule into
  generated Engineer guidance (1aa405f).

### Fixes
- **tools:** Reject bare port validation commands (1aa405f)

## [0.42.7] - 2026-05-20
<!-- mars-harness-release: version=0.42.7 commit=e9dd4a960838 -->

### Impact
- **tools:** Go/API validation no longer dirties target repositories with
  compiled binaries by default, so agents avoid a blast-radius trap before it
  starts.

### Why
- **tools:** The `demo-api-run7` canary confirmed managed background service
  validation worked, then showed Engineer creating a root `task-notes-api`
  binary with `go build -o`. That artifact triggered blast-radius containment
  and masked later malformed recovery calls until `circle_detected`.

### What Changed
- **tools:** `shell_exec` now blocks explicit `go build -o <path>` outputs
  when `<path>` resolves inside the target repo, suggests an external temp
  output, validates malformed shell payloads before dirty-diff masking, and
  mirrors the external-validation-binary rule into generated Engineer guidance
  (e9dd4a9).

### Fixes
- **tools:** Prevent repo-local validation binaries (e9dd4a9)

## [0.42.6] - 2026-05-20
<!-- mars-harness-release: version=0.42.6 commit=fe2fb41d4e22 -->

### Impact
- **tools:** Web and API target validation is less likely to leak local server
  processes or burn turns on shell cleanup, because long-running checks must use
  the managed `background:true` path.

### Why
- **tools:** The `demo-api-run6` canary showed Engineer could recover from
  generated build artifacts, then got stuck validating a Go HTTP service by
  mixing foreground `go run`, shell `&` backgrounding, port cleanup, and
  malformed `:8080` commands.

### What Changed
- **tools:** `shell_exec` now rejects shell-background `&` inside
  `shell_command`, reports early `background:true` exits as startup errors with
  output and exit code, bounds foreground timeout cleanup with `WaitDelay`, and
  mirrors the managed-background rule into generated Engineer guidance
  (fe2fb41).

### Fixes
- **tools:** Harden managed server validation (fe2fb41)

## [0.42.5] - 2026-05-20
<!-- mars-harness-release: version=0.42.5 commit=1025fe71d0a4 -->

### Impact
- **tools:** Agents get a direct recovery path when a generated repo/module
  binary trips blast-radius limits, reducing max-turn loops after validation
  builds.

### Why
- **tools:** The run5 Task Notes API canary proved the cleanup exception was
  technically available but not discoverable: Engineer never tried
  `rm task-notes-api` because the guardrail error only suggested splitting
  work or raising `MaxLinesPerFile`.

### What Changed
- **tools:** `validateRepoDiff` now appends the exact `rm <artifact>` command
  when the offending blast-radius file is an untracked, root-level,
  binary-looking build artifact named after the repo or Go module (1025fe7).

### Fixes
- **tools:** Hint generated artifact cleanup (1025fe7)

## [0.42.4] - 2026-05-20
<!-- mars-harness-release: version=0.42.4 commit=3e57ba14885d -->

### Impact
- **tools:** Fresh Go targets can recover from validation builds that leave a
  root executable named after the module, so generated binaries no longer trap
  Engineer behind blast-radius checks.

### Why
- **tools:** The live Task Notes API canary reached product implementation but
  failed at max turns after `task-notes-api` was treated as a 33,970-line
  source change. The existing cleanup exception handled repo-named binaries but
  missed module-named artifacts.

### What Changed
- **tools:** The shell policy now allows `rm`/`unlink` for untracked,
  root-level, binary-looking artifacts named after either the repo directory or
  the root `go.mod` module basename, while ordinary deletion, tracked files,
  nested paths, and text files remain blocked (3e57ba1).

### Fixes
- **tools:** Allow module-named build artifact cleanup (3e57ba1)

## [0.42.3] - 2026-05-20
<!-- mars-harness-release: version=0.42.3 commit=decb5f38231c -->

### Impact
- **scanner:** Fresh target bootstraps are less likely to stall in planning
  because generated CEO and COO prompts now reuse the existing canonical
  `docs/features/F-NNN*.md` contract instead of inventing a second `F-001`
  path.

### Why
- **scanner:** The live Task Notes API canary showed the previous guidance let
  planners hit duplicate-contract and duplicate-scenario guardrails before CTO
  ticketing. Reusing the starter contract keeps the product-first lifecycle on
  the CEO -> COO -> CTO path.

### What Changed
- **scanner:** Generated CEO guidance now forbids `docs/features/` writes during
  bootstrap and asks CEO to pass the existing contract path to COO. Generated
  COO guidance now searches `docs/features/F-NNN*.md`, edits the existing path,
  and rewrites starter scenarios in place with unique IDs (decb5f3).

### Fixes
- **scanner:** Reuse canonical bootstrap feature contracts (decb5f3)

## [0.42.2] - 2026-05-20
<!-- mars-harness-release: version=0.42.2 commit=1e42e219c213 -->

### Impact
- **tools:** Operators see improved reliability because allow cleanup of root build artifacts.

### Why
- **tools:** This matters because allow cleanup of root build artifacts closes a failure mode or degraded path.

### What Changed
- **tools:** Changed allow cleanup of root build artifacts (1e42e21).

### Fixes
- **tools:** Allow cleanup of root build artifacts (1e42e21)

## [0.42.1] - 2026-05-20
<!-- mars-harness-release: version=0.42.1 commit=e9187f2e2049 -->

### Impact
- **scheduler:** Operators see improved reliability because skip active same-role scheduled work.

### Why
- **scheduler:** This matters because skip active same-role scheduled work closes a failure mode or degraded path.

### What Changed
- **scheduler:** Changed skip active same-role scheduled work (e9187f2).

### Fixes
- **scheduler:** Skip active same-role scheduled work (e9187f2)

## [0.42.0] - 2026-05-20
<!-- mars-harness-release: version=0.42.0 commit=b6173231a23c -->

### Impact
- **quality:** Operators gain new capability: export factory pace baselines.

### Why
- **quality:** This matters because export factory pace baselines was missing from the shipped capability set.

### What Changed
- **quality:** Changed export factory pace baselines (b617323).

### Features
- **quality:** Export factory pace baselines (b617323)

## [0.41.34] - 2026-05-20
<!-- mars-harness-release: version=0.41.34 commit=a5b34ae68eca -->

### Impact
- **docsync:** Operators and agents get stronger no-stale-docs enforcement because documentation sync is described and validated as part of the delivery workflow.

### Why
- **docsync:** This matters because behavior changes become risky when code, BDD contracts, design docs, generated target guidance, and release notes drift apart.

### What Changed
- **docsync:** The release documentation path now ties changed source files to associated docs, docsync evidence, and generated target doctrine instead of treating docs as an after-the-fact checklist (a5b34ae).

### Fixes
- **docsync:** Audit deployed static app roots (a5b34ae)

## [0.41.33] - 2026-05-20
<!-- mars-harness-release: version=0.41.33 commit=bfc219a8ae48 -->

### Impact
- **tools:** Operators see improved reliability because normalize list string arguments.

### Why
- **tools:** This matters because normalize list string arguments closes a failure mode or degraded path.

### What Changed
- **tools:** Changed normalize list string arguments (bfc219a).

### Fixes
- **tools:** Normalize list string arguments (bfc219a)

## [0.41.32] - 2026-05-20
<!-- mars-harness-release: version=0.41.32 commit=40d9722ec551 -->

### Impact
- **dispatch:** Operators see improved reliability because stop release blocked loops.

### Why
- **dispatch:** This matters because stop release blocked loops closes a failure mode or degraded path.

### What Changed
- **dispatch:** Changed stop release blocked loops (40d9722).

### Fixes
- **dispatch:** Stop release blocked loops (40d9722)

## [0.41.31] - 2026-05-20
<!-- mars-harness-release: version=0.41.31 commit=5c37fee655b9 -->

### Impact
- **factory:** Operators see improved reliability because stabilize static demo lifecycle.

### Why
- **factory:** This matters because stabilize static demo lifecycle closes a failure mode or degraded path.

### What Changed
- **factory:** Changed stabilize static demo lifecycle (5c37fee).

### Fixes
- **factory:** Stabilize static demo lifecycle (5c37fee)

## [0.41.30] - 2026-05-20
<!-- mars-harness-release: version=0.41.30 commit=1c8ff8453634 -->

### Impact
- **dispatch:** Operators see improved reliability because pause dirty target survey handoffs.

### Why
- **dispatch:** This matters because pause dirty target survey handoffs closes a failure mode or degraded path.

### What Changed
- **dispatch:** Changed pause dirty target survey handoffs (1c8ff84).

### Fixes
- **dispatch:** Pause dirty target survey handoffs (1c8ff84)

## [0.41.29] - 2026-05-20
<!-- mars-harness-release: version=0.41.29 commit=29cafc75b953 -->

### Impact
- Operators and future agents get clearer guidance because correct unsupported Homebrew install guidance.

### Why
- This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.

### What Changed
- Changed correct unsupported Homebrew install guidance (29cafc7).

### Documentation
- Correct unsupported Homebrew install guidance (29cafc7)

## [0.41.28] - 2026-05-19
<!-- mars-harness-release: version=0.41.28 commit=353ad7daa350 -->

### Impact
- **tickets:** Operators and future agents get clearer guidance because add factory pace intervention debt (T-011).

### Why
- **tickets:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.

### What Changed
- **tickets:** Changed add factory pace intervention debt (T-011) (353ad7d).

### Documentation
- **tickets:** Add factory pace intervention debt (T-011) (353ad7d)

## [0.41.27] - 2026-05-19
<!-- mars-harness-release: version=0.41.27 commit=4fd474166d7d -->

### Impact
- **qualityscore:** The release carries stronger evidence because keep outcome signals non-mutating by default.

### Why
- **qualityscore:** This matters because the project needs durable evidence that the behavior keeps working.

### What Changed
- **qualityscore:** Changed keep outcome signals non-mutating by default (4fd4741).

### Tests
- **qualityscore:** Keep outcome signals non-mutating by default (4fd4741)

## [0.41.26] - 2026-05-19
<!-- mars-harness-release: version=0.41.26 commit=6e1a8572d841 -->

### Impact
- **tickets:** Operators and future agents get clearer guidance because add shadcn dashboard replacement ticket.

### Why
- **tickets:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.

### What Changed
- **tickets:** Changed add shadcn dashboard replacement ticket (6e1a857).

### Documentation
- **tickets:** Add shadcn dashboard replacement ticket (6e1a857)

## [0.41.25] - 2026-05-19
<!-- mars-harness-release: version=0.41.25 commit=6e69bfa5ea47 -->

### Impact
- **sqlite:** The release carries stronger evidence because cover legacy store fixtures.

### Why
- **sqlite:** This matters because the project needs durable evidence that the behavior keeps working.

### What Changed
- **sqlite:** Changed cover legacy store fixtures (6e69bfa).

### Tests
- **sqlite:** Cover legacy store fixtures (6e69bfa)

## [0.41.24] - 2026-05-19
<!-- mars-harness-release: version=0.41.24 commit=46c58884ba6e -->

### Impact
- **references:** Operators and future agents get clearer guidance because add OpenHarness follow-up review.

### Why
- **references:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.

### What Changed
- **references:** Changed add OpenHarness follow-up review (46c5888).

### Documentation
- **references:** Add OpenHarness follow-up review (46c5888)

## [0.41.23] - 2026-05-19
<!-- mars-harness-release: version=0.41.23 commit=f4220fddd375 -->

### Impact
- **references:** Operators and future agents get a durable Archon comparison note that captures workflow, isolation, UX, provider-default, and adoption-boundary findings for later planning.

### Why
- **references:** This matters because Archon's workflow-engine shape overlaps Mars Harness remediation, dashboard, and execution-roadmap decisions, while some Archon defaults conflict with Mars Harness' local-first and strict-trunk doctrine.

### What Changed
- **references:** Added `docs/references/archon-comparator.md` and indexed it from `docs/references/README.md` (f4220fd).

### Documentation
- **references:** Added the Archon comparator reference note (f4220fd)

## [0.41.22] - 2026-05-19
<!-- mars-harness-release: version=0.41.22 commit=b61c659b1f92 -->

### Impact
- **run:** Operators see improved reliability because add observer-safe no-init dry-run.

### Why
- **run:** This matters because add observer-safe no-init dry-run closes a failure mode or degraded path.

### What Changed
- **run:** Changed add observer-safe no-init dry-run (b61c659).

### Fixes
- **run:** Add observer-safe no-init dry-run (b61c659)

## [0.41.21] - 2026-05-19
<!-- mars-harness-release: version=0.41.21 commit=9f18594b5806 -->

### Impact
- **skills:** Operators and future agents get clearer guidance because add release publication workflow.

### Why
- **skills:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.

### What Changed
- **skills:** Changed add release publication workflow (9f18594).

### Documentation
- **skills:** Add release publication workflow (9f18594)

## [0.41.20] - 2026-05-19
<!-- mars-harness-release: version=0.41.20 commit=eb701aa57e76 -->

### Impact
- **validation:** Operators and future agents get clearer guidance because record mars observer trial.

### Why
- **validation:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.

### What Changed
- **validation:** Changed record mars observer trial (eb701aa).

### Documentation
- **validation:** Record mars observer trial (eb701aa)

## [0.41.19] - 2026-05-19
<!-- mars-harness-release: version=0.41.19 commit=7ed38e599deb -->

### Impact
- **serve:** Operators see improved reliability because route dashboard stop through server loop.

### Why
- **serve:** This matters because route dashboard stop through server loop closes a failure mode or degraded path.

### What Changed
- **serve:** Changed route dashboard stop through server loop (7ed38e5).

### Fixes
- **serve:** Route dashboard stop through server loop (7ed38e5)

## [0.41.18] - 2026-05-19
<!-- mars-harness-release: version=0.41.18 commit=5b2a46910435 -->

### Impact
- **tools:** Operators see improved reliability because prefer active harness cli binary.

### Why
- **tools:** This matters because prefer active harness cli binary closes a failure mode or degraded path.

### What Changed
- **tools:** Changed prefer active harness cli binary (5b2a469).

### Fixes
- **tools:** Prefer active harness cli binary (5b2a469)

## [0.41.17] - 2026-05-19
<!-- mars-harness-release: version=0.41.17 commit=27882d6a27e6 -->

### Impact
- **orchestration:** Operators see improved reliability because route completed planning forward.

### Why
- **orchestration:** This matters because route completed planning forward closes a failure mode or degraded path.

### What Changed
- **orchestration:** Changed route completed planning forward (27882d6).

### Fixes
- **orchestration:** Route completed planning forward (27882d6)

## [0.41.16] - 2026-05-19
<!-- mars-harness-release: version=0.41.16 commit=cb0ef00bb9ce -->

### Impact
- **dogfood:** The release carries stronger evidence because broaden foundation validation loop.

### Why
- **dogfood:** This matters because the project needs durable evidence that the behavior keeps working.

### What Changed
- **dogfood:** Changed broaden foundation validation loop (cb0ef00).

### Tests
- **dogfood:** Broaden foundation validation loop (cb0ef00)

## [0.41.15] - 2026-05-19
<!-- mars-harness-release: version=0.41.15 commit=b67f96729a99 -->

### Impact
- **remediation:** Deterministic repair is safer and more honest at the edges:
  dirty target worktrees remain operator-visible blockers, destructive repair
  recipes stay approval-gated, and missing optional tools become setup/skip
  guidance instead of false success.

### Why
- **remediation:** The live `demo-123` feedback loop showed that product-first
  bootstrap depends on runtime failures staying out of target backlog unless
  the target owns the cause. Closing the remaining `MH-048` edge cases keeps
  remediation evidence useful without letting it mutate user work or hide
  missing local tooling.

### What Changed
- **remediation:** Added an explicit `optional-tool:install-guidance` recipe,
  tests that destructive recipes never auto-ready without approval, and tests
  that dirty worktrees route to `git status --short` blocker guidance rather
  than cleanup commands.
- **planning:** Moved `MH-048` to done, recorded a clean `demo-123` replay
  through product-specific planning and feature-contract creation, and advanced
  the active plan to `MH-049` for broader dogfood matrix evidence.

### Fixes
- **remediation:** Close deterministic recipe edge cases (b67f967)

## [0.41.14] - 2026-05-19
<!-- mars-harness-release: version=0.41.14 commit=11433811939f -->

### Impact
- **skills:** Operators and future agents now have a clear boundary: the broad
  recursive improvement loop remains operating doctrine, while the repeated
  GitHub release-publication ritual becomes a queued foundation Release Manager
  skill instead of an immediate tool.

### Why
- **skills:** The live improvement loop crosses roles, tools, tests, target
  replay, trunk publication, release notes, and blocker evidence, so turning the
  whole loop into a skill would hide system-level doctrine inside procedural
  memory. Release publication is narrower and judgment-heavy enough to benefit
  from compact skill guidance before a deterministic release-publish tool is
  justified.

### What Changed
- **skills:** Added AD-140 to `skill-evolution.md`, keeping recursive
  improvement as doctrine and selecting a foundation Release Manager skill for
  GitHub release object publication, notes-only fallback, asset verification,
  and missing-asset blocker recording.
- **planning:** Closed `T-005`, created tool-backed `T-006` for the foundation
  release publication skill, and returned the active plan to the Mars parity
  remediation lane.

### Documentation
- **skills:** Decide recursive improvement skill boundary (1143381)

## [0.41.13] - 2026-05-19
<!-- mars-harness-release: version=0.41.13 commit=dc18ff12ecc5 -->

### Impact
- **review:** Operators and future agents now have explicit evidence that the
  source architecture, generated target route, glossary, tools, release
  doctrine, and scanner tests agree on the foundation/deployed boundary.

### Why
- **review:** Mirroring AD-139 into generated targets was not enough by itself.
  The operating model needed a recorded drift pass so future work can trust the
  source-only versus mirrored split without rereading every generated default.

### What Changed
- **review:** Added a 2026-05-19 drift review table to AD-139 covering source
  `AGENTS.md`, the harness glossary, mirrored doctrine, tools glossary, release
  doctrine, generated knowledge routes, generated glossary routes, generated
  mirrored docs, and scanner assertions.
- **planning:** Closed `T-004`, found no unowned doctrine mismatch, and advanced
  the active plan to `T-005` for the remaining skill/tool/doctrine decision.

### Documentation
- **review:** Record foundation deployed doctrine drift review (dc18ff1)

## [0.41.12] - 2026-05-19
<!-- mars-harness-release: version=0.41.12 commit=f84abcd3a978 -->

### Impact
- **generated:** Fresh target harnesses now receive the reusable
  foundation/deployed architecture route and core doctrine, so target agents
  can distinguish mirrored operating rules from source-only `mars-harness`
  release and runtime mechanics.

### Why
- **generated:** AD-139 would drift if it lived only in the foundation repo.
  Target agents need the routing language for feedback ownership, doctrine
  drift, tool/skill authority, and recursive improvement boundaries without
  importing foundation-only binary asset duties.

### What Changed
- **generated:** Added a generated knowledge route and harness-glossary context
  entry for foundation/deployed boundaries, mirrored operating doctrine,
  runtime feedback routing, and source-only versus deployed-target
  requirements.
- **generated:** Mirrored the AD-139 core into generated target design docs and
  scanner tests while asserting generated doctrine does not import source binary
  asset names.
- **planning:** Closed `T-003`, marked `F-004-S007` and `F-012-S007` passing,
  and advanced the active plan to the doctrine drift review.

### Documentation
- **generated:** Mirror foundation deployed doctrine route (f84abcd)

## [0.41.11] - 2026-05-19
<!-- mars-harness-release: version=0.41.11 commit=c8b5968f4913 -->

### Impact
- **architecture:** Operators and future agents now have a single architecture
  reference for separating the foundation harness, runtime substrate, deployed
  harness, target project ownership, mirrored doctrine, and source-only release
  mechanics.

### Why
- **architecture:** The recursive improvement loop was proving useful, but it
  made the layer split more important: the binary executes orchestration, the
  foundation harness evolves the software factory, and deployed harnesses
  should inherit only the reusable operating core unless a rule is explicitly
  mirrored.

### What Changed
- **architecture:** Added AD-139 with boundary tables, architecture and
  doctrine-flow diagrams, feedback collection/routing rules, tool/skill/binary
  authority levels, generated-target implications, and doctrine-maintenance
  duties.
- **planning:** Closed `T-002`, marked `F-001-S015` passing, and advanced the
  active plan toward `T-003` generated-target mirroring as the next failing
  scenario.

### Documentation
- **architecture:** Document foundation deployed harness boundary (c8b5968)

## [0.41.10] - 2026-05-19
<!-- mars-harness-release: version=0.41.10 commit=a81fccee532c -->

### Impact
- **planning:** Future foundation and deployed harness agents now have a
  ticket-backed path for clarifying how recursive improvement, feedback
  routing, mirrored doctrine, and source-only release mechanics fit together.

### Why
- **planning:** The live improvement loop was producing useful outcomes, but
  the surrounding architecture was still easy to blur: the binary executes the
  runtime, the foundation harness evolves Mars Harness doctrine, and deployed
  harnesses inherit only the reusable core unless a rule is deliberately
  source-only.

### What Changed
- **planning:** Added `F-001-S015` and refreshed the active plan so foundation
  and deployed doctrine boundaries are scheduled before implementation work.
- **planning:** Created four tool-backed backlog tickets covering the
  architecture document, generated target mirroring, doctrine drift review, and
  later recursive-improvement skill evaluation.
- **docs:** Marked F-001 as partially passing until the new architecture slice
  has durable evidence.

### Documentation
- **planning:** Materialize foundation deployed architecture tickets (a81fcce)

## [0.41.9] - 2026-05-19
<!-- mars-harness-release: version=0.41.9 commit=c7159e6d5484 -->

### Impact
- **release:** Operators and future agents now have a release-object gate, so
  pushed tags and changelog entries cannot leave the GitHub Releases page stale.

### Why
- **release:** The live release loop produced tags through `v0.41.8`, but the
  GitHub Releases page still showed `v0.36.3` because the Release workflow was
  blocked before creating release objects. The operating model now treats that
  visible release object as a separate gate from binary asset verification.

### What Changed
- **release:** Source workflow docs, generated target doctrine, and Release
  Manager guidance now require `gh release view vX.Y.Z` after each tag push.
- **release:** If the tag workflow cannot create the release object but the
  GitHub API is available, Release Manager must create a notes-only release from
  the generated `CHANGELOG.md` entry for the existing tag, then keep missing
  binary assets as the remaining blocker.
- **tests:** Docs-consistency and scanner coverage now assert the release-object
  fallback remains present in source and generated target guidance.

### Documentation
- **release:** Require GitHub release object fallback (c7159e6)

## [0.41.8] - 2026-05-19
<!-- mars-harness-release: version=0.41.8 commit=dd6b0035c2a8 -->

### Impact
- **tools:** A single stuck tool handler can no longer strand an agent job
  indefinitely with dirty target files and no new LLM activity.

### Why
- **tools:** The patched `demo-123` replay reached Engineer and produced
  Space Invaders source files, then went quiet with the job still marked
  `running`. The executor had a context timeout, but a non-returning handler
  could still keep the loop from recording a blocker or failure.

### What Changed
- **tools:** Tool handlers now execute behind a hard executor TTL. If the TTL
  expires, the executor stops waiting and returns an actionable timeout error
  to the model instead of letting the loop hang.
- **docs:** The runtime feature contract and dogfood evidence now record stuck
  tool handlers as a foundation runtime failure mode with regression coverage.

### Fixes
- **tools:** Hard timeout stuck tool handlers (dd6b003)

## [0.41.7] - 2026-05-19
<!-- mars-harness-release: version=0.41.7 commit=082ec783f38a -->

### Impact
- **orchestration:** Clean target lifecycles now keep moving from product
  validation into release/versioning instead of stopping with unreleased
  semantic commits.
- **release:** Historical changelog backfill can be used as a compliance check
  without flattening release entries that already have good Impact, Why, and
  What Changed narrative.

### Why
- **orchestration:** The latest `demo-123` replay proved the release rule was
  documented but not actually enforced by dispatch: target `VERSION` stayed
  `0.1.0` and `CHANGELOG.md` remained a stub after semantic target commits.
- **release:** The first retrospective backfill attempt also showed the checker
  could mistake richer existing entries for stale entries and replace them with
  generic commit-subject prose.

### What Changed
- **orchestration:** Review roles that complete with their own review
  `next_need` now advance to the next review owner, and Dogfood completion
  routes to Release Manager when that role is configured.
- **scanner:** Generated target Dogfood and Release Manager guidance now names
  the release-review handoff and requires `release backfill-notes --check`
  during versioning.
- **release:** `release backfill-notes` now preserves entries that already have
  complete current narrative sections, while still filling legacy or missing
  narrative when needed.
- **docs:** The operating model, feature contracts, and live dogfood evidence
  record release review as part of the product lifecycle rather than a weekly
  afterthought.

### Fixes
- **orchestration:** Route validated work to release notes (082ec78)

## [0.41.6] - 2026-05-19
<!-- mars-harness-release: version=0.41.6 commit=ec88640bbb88 -->

### Impact
- **orchestration:** Direct dispatch no longer turns a role's own unfinished
  work signal into an immediate same-role job.

### Why
- **orchestration:** The `v0.41.5` clean `demo-123` replay completed end to
  end, but the first COO job finished with `next_need: exec_plan`, which mapped
  right back to COO and added an avoidable second planning pass before CTO. That
  is forward progress, but it is still autonomous loop noise.

### What Changed
- **orchestration:** Non-Orchestrator completed and no-work dispositions now
  stop when their only direct route is a `next_need` that resolves back to the
  same role.
- **scanner:** Generated COO guidance now tells COO not to finish with planning
  `next_need` values that route back to COO; it must continue planning, record a
  blocker, or hand off to another owner.
- **docs:** The live demo evidence trail records the same-role COO replay
  finding and the new dispatch rule.

### Fixes
- **orchestration:** Stop same-role next-need loops (ec88640)

## [0.41.5] - 2026-05-19
<!-- mars-harness-release: version=0.41.5 commit=1df263c139c3 -->

### Impact
- **scanner:** Fresh static demos should spend fewer turns on package-manager,
  container, and ticket-metadata shell churn after the product lifecycle is
  already healthy.

### Why
- **scanner:** The latest clean `demo-123` replay completed CEO through Dogfood
  with no intervention-debt flood, but Engineer and Dogfood still burned
  excessive tool calls proving a tiny static HTML game. The generated role
  defaults now match that target shape instead of pushing every demo toward a
  full package-managed app workflow.

### What Changed
- **scanner:** Generated Engineer guidance now treats no-manifest static
  HTML/CSS/JS targets as valid with bounded HTTP smoke evidence and asks for
  one full-file ticket evidence update rather than repeated shell substitutions.
- **scanner:** Generated Dogfood guidance now skips irrelevant package/container
  expectations for no-manifest static targets, requires background-only static
  server smoke tests, and keeps validation evidence bounded.
- **scanner:** The live-demo evidence trail and feature contract now capture
  this evidence-cost stabilization loop.

### Fixes
- **scanner:** Tighten static demo role guidance (1df263c)

## [0.41.4] - 2026-05-19
<!-- mars-harness-release: version=0.41.4 commit=9db85c479fec -->

### Impact
- **serve:** The native survey loop no longer turns an Engineer `max_turns`
  failure into an immediate same-role retry just because the ticket remains
  in progress.

### Why
- **serve:** Live `demo-123` validation showed failure handling correctly
  quarantining `max_turns` as foundation telemetry, while the survey watchdog
  immediately re-enqueued another `ticket_delivery` Engineer for the same
  in-progress ticket.

### What Changed
- **serve:** Ticket-owner survey routing now pauses after recent same-role
  runtime failures during a cooldown window, preserving in-progress ticket
  priority without bypassing runtime-failure containment (9db85c4).

### Fixes
- **serve:** Pause ticket-owner survey after runtime failure (9db85c4)

## [0.41.3] - 2026-05-19
<!-- mars-harness-release: version=0.41.3 commit=acba3ebe8d2f -->

### Impact
- **guardrails:** Agents get a faster recovery path when a feature-contract
  rewrite accidentally duplicates a BDD scenario heading.

### Why
- **guardrails:** Live `demo-123` validation completed, but Engineer spent 43
  LLM calls and nine guardrail blocks partly because the duplicate-scenario
  error did not distinguish real headings from Scenario Schedule references.

### What Changed
- **guardrails:** Duplicate feature-scenario errors now name the duplicate
  heading line numbers and tell the role that Scenario Schedule list references
  are allowed, keeping the fix to one targeted full-file rewrite (acba3eb).

### Fixes
- **guardrails:** Point duplicate scenario errors at headings (acba3eb)

## [0.41.2] - 2026-05-19
<!-- mars-harness-release: version=0.41.2 commit=f7ff1f0ca2b2 -->

### Impact
- **tools:** Agent runs recover faster when a model accidentally passes shell
  redirection or control syntax through structured `argv` instead of a shell.

### Why
- **tools:** Live `demo-123` validation showed an Engineer burning completion
  turns on low-level `exec` failures such as `exec: ":" not found` after useful
  product commits had already landed.

### What Changed
- **tools:** `shell_exec` now rejects shell-only argv tokens before process
  execution and points the role to `shell_command` or file tools for the
  intended operation (f7ff1f0).

### Fixes
- **tools:** Reject shell syntax in argv mode (f7ff1f0)

## [0.41.1] - 2026-05-19
<!-- mars-harness-release: version=0.41.1 commit=b18aed669531 -->

### Impact
- **guardrails:** Operators see improved reliability because ignore new lockfile line churn.

### Why
- **guardrails:** This matters because ignore new lockfile line churn closes a failure mode or degraded path.

### What Changed
- **guardrails:** Changed ignore new lockfile line churn (b18aed6).

### Fixes
- **guardrails:** Ignore new lockfile line churn (b18aed6)

## [0.41.0] - 2026-05-19
<!-- mars-harness-release: version=0.41.0 commit=53bbb5e8170d -->

### Impact
- **qualityscore:** Operators gain new capability: summarize remediation evidence.

### Why
- **qualityscore:** This matters because summarize remediation evidence was missing from the shipped capability set.

### What Changed
- **qualityscore:** Changed summarize remediation evidence (53bbb5e).

### Features
- **qualityscore:** Summarize remediation evidence (53bbb5e)

## [0.40.1] - 2026-05-19
<!-- mars-harness-release: version=0.40.1 commit=aae9b0640ca4 -->

### Impact
- **serve:** Operators see improved reliability because require executors before suppressing remediation retry.

### Why
- **serve:** This matters because require executors before suppressing remediation retry closes a failure mode or degraded path.

### What Changed
- **serve:** Changed require executors before suppressing remediation retry (aae9b06).

### Fixes
- **serve:** Require executors before suppressing remediation retry (aae9b06)

## [0.40.0] - 2026-05-19
<!-- mars-harness-release: version=0.40.0 commit=1930ae4e340b -->

### Impact
- **doctor:** Operators gain new capability: surface deterministic remediation checks.

### Why
- **doctor:** This matters because surface deterministic remediation checks was missing from the shipped capability set.

### What Changed
- **doctor:** Changed surface deterministic remediation checks (1930ae4).

### Features
- **doctor:** Surface deterministic remediation checks (1930ae4)

## [0.39.0] - 2026-05-19
<!-- mars-harness-release: version=0.39.0 commit=5f2adf60661f -->

### Impact
- **serve:** Operators gain new capability: execute generated-docs remediation.

### Why
- **serve:** This matters because execute generated-docs remediation was missing from the shipped capability set.

### What Changed
- **serve:** Changed execute generated-docs remediation (5f2adf6).

### Features
- **serve:** Execute generated-docs remediation (5f2adf6)

## [0.38.0] - 2026-05-19
<!-- mars-harness-release: version=0.38.0 commit=be6cd0f2174a -->

### Impact
- **serve:** Operators gain new capability: record deterministic remediation plans.

### Why
- **serve:** This matters because record deterministic remediation plans was missing from the shipped capability set.

### What Changed
- **serve:** Changed record deterministic remediation plans (be6cd0f).

### Features
- **serve:** Record deterministic remediation plans (be6cd0f)

## [0.37.0] - 2026-05-19
<!-- mars-harness-release: version=0.37.0 commit=24f0aa60f2a3 -->

### Impact
- **remediation:** Operators gain new capability: add deterministic recipe registry.

### Why
- **remediation:** This matters because add deterministic recipe registry was missing from the shipped capability set.

### What Changed
- **remediation:** Changed add deterministic recipe registry (24f0aa6).

### Features
- **remediation:** Add deterministic recipe registry (24f0aa6)

## [0.36.6] - 2026-05-19
<!-- mars-harness-release: version=0.36.6 commit=1a34a002d019 -->

### Impact
- **operating-model:** Operators and future agents get clearer delivery behavior because an operating-model rule, boundary, or workflow contract is now explicit in repo-owned guidance.

### Why
- **operating-model:** This matters because autonomous work needs durable routing, evidence, and ownership rules rather than relying on chat memory or implicit handoffs.

### What Changed
- **operating-model:** The operating-model guidance was updated so adjacent docs, roles, tools, evidence paths, and generated target defaults describe the new workflow consistently (1a34a00).

### Documentation
- **operating-model:** Require remote publication after live confirmation (1a34a00)

## [0.36.5] - 2026-05-19
<!-- mars-harness-release: version=0.36.5 commit=c5b17a58fc35 -->

### Impact
- **operating-model:** Operators and future agents get clearer delivery behavior because an operating-model rule, boundary, or workflow contract is now explicit in repo-owned guidance.

### Why
- **operating-model:** This matters because autonomous work needs durable routing, evidence, and ownership rules rather than relying on chat memory or implicit handoffs.

### What Changed
- **operating-model:** The operating-model guidance was updated so adjacent docs, roles, tools, evidence paths, and generated target defaults describe the new workflow consistently (c5b17a5).

### Documentation
- **operating-model:** Formalize live demo improvement loop (c5b17a5)

## [0.36.4] - 2026-05-19
<!-- mars-harness-release: version=0.36.4 commit=533e532eabdf -->

### Impact
- **lifecycle:** Operators see improved reliability because prioritize product progress before intervention debt.

### Why
- **lifecycle:** This matters because prioritize product progress before intervention debt closes a failure mode or degraded path.

### What Changed
- **lifecycle:** Changed prioritize product progress before intervention debt (533e532).

### Fixes
- **lifecycle:** Prioritize product progress before intervention debt (533e532)

## [0.36.3] - 2026-05-05
<!-- mars-harness-release: version=0.36.3 commit=4fc0c2a7c5e9 -->

### Impact
- **dashboard:** Operators no longer see the legacy blue dashboard in freshly installed harness binaries. The embedded web dashboard now presents the current neutral operations theme consistently across navigation, cards, controls, status pills, and chart surfaces.

### Why
- **dashboard:** The dashboard is an operator trust surface. Reintroducing the old hard-coded slate/blue palette made target runs such as `paul-demo` look stale even when the binary was freshly installed, so the visual contract needed a regression check instead of relying on manual inspection.

### What Changed
- **dashboard:** Replaced the legacy palette with semantic operations-theme CSS tokens, routed Chart.js colors through those tokens, documented the theme contract, and added a dashboard regression test that fails if the old blue palette literals return (4fc0c2a).

### Fixes
- **dashboard:** Refresh embedded operations theme (4fc0c2a)

## [0.36.2] - 2026-05-05
<!-- mars-harness-release: version=0.36.2 commit=cf60686d77fa -->

### Impact
- **ui:** The release carries stronger evidence because avoid dashboard buffer race.

### Why
- **ui:** This matters because the project needs durable evidence that the behavior keeps working.

### What Changed
- **ui:** Changed avoid dashboard buffer race (cf60686).

### Tests
- **ui:** Avoid dashboard buffer race (cf60686)

## [0.36.1] - 2026-05-05
<!-- mars-harness-release: version=0.36.1 commit=7c350a8321a8 -->

### Impact
- **operating-model:** Operators and future agents get clearer delivery behavior because an operating-model rule, boundary, or workflow contract is now explicit in repo-owned guidance.

### Why
- **operating-model:** This matters because autonomous work needs durable routing, evidence, and ownership rules rather than relying on chat memory or implicit handoffs.

### What Changed
- **operating-model:** The operating-model guidance was updated so adjacent docs, roles, tools, evidence paths, and generated target defaults describe the new workflow consistently (7c350a8).

### Documentation
- **operating-model:** Require remote trunk freshness (7c350a8)

## [0.36.0] - 2026-05-05
<!-- mars-harness-release: version=0.36.0 commit=8bcb789067b1 -->

### Impact
- **cli:** Operators gain new capability: add tty dashboard logging.

### Why
- **cli:** This matters because add tty dashboard logging was missing from the shipped capability set.

### What Changed
- **cli:** Changed add tty dashboard logging (8bcb789).

### Features
- **cli:** Add tty dashboard logging (8bcb789)

## [0.35.1] - 2026-05-05
<!-- mars-harness-release: version=0.35.1 commit=7ef248d424fd -->

### Impact
- **tools:** Main CI can complete after the private release auth change because the workspace hygiene helper code no longer trips golangci-lint.

### Why
- **tools:** The prior workspace hygiene change left unused helpers and a simplifiable struct conversion that tests allowed but CI lint rejected.

### What Changed
- **tools:** Removed unused generated-path helper functions and the slices import, and converted workspace_hygiene arguments directly into WorkspaceHygieneOptions (7ef248d).

### Fixes
- **tools:** Satisfy workspace hygiene lint (7ef248d)

## [0.35.0] - 2026-05-05
<!-- mars-harness-release: version=0.35.0 commit=ec917142c353 -->

### Impact
- **auth:** Operators and agents get a first-class private release auth path with setup/check commands, doctor readiness, setup gating, update guidance, a universal github_auth_check tool, and mirrored getting-started docs.

### Why
- **auth:** Private GitHub Release access was a hidden prerequisite for update tool and release asset workflows, which made version drift and install repair feel like mysterious token failures instead of an explicit onboarding step.

### What Changed
- **auth:** Added the githubauth resolver with GH_TOKEN, GITHUB_TOKEN, GitHub CLI, then local config precedence; wired auth github setup/check into the CLI, setup, doctor, self-update, tool registry, role allowlists, generated target guidance, docs, skills, and tests while preserving token redaction (ec91714).

### Features
- **auth:** Add private release auth setup model (ec91714)

## [0.34.1] - 2026-05-05
<!-- mars-harness-release: version=0.34.1 commit=8d96adccffe2 -->

### Impact
- **guardrails:** The blast-radius guard now evaluates implementation changes separately from generated dependency/build output, so untracked `node_modules`-style churn no longer blocks the guardrail before hygiene can repair repo policy.
- **workspace hygiene:** `serve` and `dependency_sync` can safely append missing generated-directory ignores and commit only `.gitignore` before model work or package installs continue.

### Why
- **guardrails:** Missing ignore policy is usually a deterministic repo hygiene issue, not a reason to send hundreds of dependency files into blast-radius accounting or context. The harness should fix the policy when it can prove the generated files are untracked.
- **workspace hygiene:** Auto-repair must stay narrow: no generated-file deletion, no unstaging user work, no source/lockfile commits, and no tracked generated tree cleanup without an explicit human-reviewed change.

### What Changed
- **guardrails:** `ValidateRepoDiff` now omits generated workspace paths from generic file, line, deletion, and secret-scan accounting, leaving those paths to the workspace hygiene operating model (8d96adc).
- **workspace hygiene:** Added repair planning and `.gitignore`-only commits for safe missing ignore entries, surfaced `auto_repairable` in hygiene reports, taught doctor to downgrade repairable hygiene failures to warnings, and made `dependency_sync` perform the same safe repair before install/fetch preflight (8d96adc).

### Fixes
- **guardrails:** Auto-repair generated ignore policy (8d96adc)

## [0.34.0] - 2026-05-05
<!-- mars-harness-release: version=0.34.0 commit=1518a9e53593 -->

### Impact
- **guardrails:** Agents now stop before model loading when dependency/build artifacts pollute a target worktree, avoiding the dirty `node_modules`, blast-radius, oversized diff, context overflow, and repeated Orchestrator recovery loop seen during dogfood.
- **tools:** All roles can audit workspace hygiene through `workspace_hygiene`, while dependency-changing roles must use `dependency_sync` for package installs/fetches so ignore policy, lockfile intent, and post-run git state are checked deterministically.

### Why
- **guardrails:** Raw dependency commands are a common autonomous-agent failure mode: they mutate generated trees, then ordinary diff/search tools stuff those generated files into context and recovery keeps retrying the same dirty state.
- **tools:** Making dependency mutation a governed tool preserves user work by blocking with an exact remediation recipe instead of silently cleaning, unstaging, or deleting generated artifacts.

### What Changed
- **guardrails:** Added pre-job hygiene gating, shell policy blocks for raw package-manager install/fetch commands, generated-directory exclusions for broad context tools, scanner/doctor findings for missing ignores and tracked generated trees, and `workspace_hygiene` telemetry that can become target intervention debt (1518a9e).
- **tools:** Added `workspace_hygiene` and `dependency_sync`, updated role allowlists and generated target guidance, documented the accepted operating-model decision, and covered the workflow with unit, policy, serve, scanner, doctor, telemetry, docs consistency, and full-suite tests (1518a9e).

### Features
- **guardrails:** Add workspace hygiene dependency gates (1518a9e)

## [0.33.3] - 2026-05-04
<!-- mars-harness-release: version=0.33.3 commit=914cf98f13ee -->

### Impact
- **release:** Operators and future agents get release notes that explain structural delivery shifts instead of repeating thin commit subjects.

### Why
- **release:** Operating-model, orchestration, persona, documentation-sync, and CLI/tool-sync releases need durable context because the changelog becomes the upgrade narrative.

### What Changed
- **release:** Added topic-aware fallback profiles, covered structured dispatch regression, documented the rule, and backfilled 0.33.1 (914cf98).

### Fixes
- **release:** Enrich structural release-note fallbacks (914cf98)

## [0.33.2] - 2026-05-04
<!-- mars-harness-release: version=0.33.2 commit=67aa3f109904 -->

### Impact
- **cli:** Operators see improved reliability because add root version shortcuts.

### Why
- **cli:** This matters because add root version shortcuts closes a failure mode or degraded path.

### What Changed
- **cli:** Changed add root version shortcuts (67aa3f1).

### Fixes
- **cli:** Add root version shortcuts (67aa3f1)

## [0.33.1] - 2026-05-04
<!-- mars-harness-release: version=0.33.1 commit=c436460e5357 -->

### Impact
- **orchestration:** Operators and agents get a more reliable delivery loop because handoff and feedback now travel as first-class runtime data through Orchestrator dispatch.

### Why
- **orchestration:** This matters because operating-model shifts lose value when the next owner, expected correction, or supporting evidence only exists in free-form transcript text.

### What Changed
- **orchestration:** Dispatch triggers now carry the source disposition, including status, next need, ticket ID, reason, evidence links, trace ID, handoff, and feedback, so Orchestrator can validate one target owner before enqueueing follow-up work (c436460).

### Fixes
- **orchestration:** Carry structured handoff through dispatch (c436460)

## [0.33.0] - 2026-05-04
<!-- mars-harness-release: version=0.33.0 commit=6fbc203cea94 -->

### Impact
- **personas:** Operators gain new capability: add canonical foundation agent manuals.

### Why
- **personas:** This matters because add canonical foundation agent manuals was missing from the shipped capability set.

### What Changed
- **personas:** Changed add canonical foundation agent manuals (6fbc203).

### Features
- **personas:** Add canonical foundation agent manuals (6fbc203)

## [0.32.0] - 2026-05-04
<!-- mars-harness-release: version=0.32.0 commit=bcf54eda8895 -->

### Impact
- **roles:** Operators gain new capability: add optional head of strategy agent.

### Why
- **roles:** This matters because add optional head of strategy agent was missing from the shipped capability set.

### What Changed
- **roles:** Changed add optional head of strategy agent (bcf54ed).

### Features
- **roles:** Add optional head of strategy agent (bcf54ed)

## [0.31.1] - 2026-05-04
<!-- mars-harness-release: version=0.31.1 commit=5c2af02c10ea -->

### Impact
- **telemetry:** Operators see improved reliability because satisfy collector rollback lint.

### Why
- **telemetry:** This matters because satisfy collector rollback lint closes a failure mode or degraded path.

### What Changed
- **telemetry:** Changed satisfy collector rollback lint (5c2af02).

### Fixes
- **telemetry:** Satisfy collector rollback lint (5c2af02)

## [0.31.0] - 2026-05-04
<!-- mars-harness-release: version=0.31.0 commit=3953db758420 -->

### Impact
- **telemetry:** Operators gain new capability: add anonymous foundation telemetry collector.

### Why
- **telemetry:** This matters because add anonymous foundation telemetry collector was missing from the shipped capability set.

### What Changed
- **telemetry:** Changed add anonymous foundation telemetry collector (3953db7).

### Features
- **telemetry:** Add anonymous foundation telemetry collector (3953db7)

## [0.30.1] - 2026-05-04
<!-- mars-harness-release: version=0.30.1 commit=074a9e5391af -->

### Impact
- **orchestration:** Freshly initialized target repos no longer get pulled into CEO/Orchestrator loops when the local model emits function-tag tool calls or when a slugged feature contract already exists. Bootstrap can keep moving from plan to feature contract to ticket shaping instead of manufacturing intervention-debt churn.

### Why
- **orchestration:** A simple README-only project exposed three connected reliability gaps: function-tag tool calls were treated as final prose, Orchestrator looked for exact `docs/features/F-001.md` paths even though generated contracts are slugged, and repeated dispatch decisions could enqueue Orchestrator again instead of stopping. The combined effect created noisy telemetry and intervention-debt tickets before product delivery had even started.

### What Changed
- **orchestration:** The agent parser now normalizes `<function=name>` and `<parameter=arg>` blocks into normal tool calls, dispatch loop guards stop repeated Orchestrator-originated routes without ticket-state changes, and generated CEO/COO/Orchestrator guidance resolves BDD features through `docs/features/F-NNN*.md` slug matches (074a9e5).

### Fixes
- **orchestration:** Stop bootstrap dispatch loops (074a9e5)

## [0.30.0] - 2026-05-04
<!-- mars-harness-release: version=0.30.0 commit=ae9ac01bc65e -->

### Impact
- **cli:** Operators and agents now have a release-blocking operating model for keeping the `mars-harness` CLI, the mirrored `mars_harness_cli` tool, generated target guidance, and CLI-related skills synchronized. A command can no longer quietly land while agents keep reading stale tool reference text or stale workflow skills.
- **cli:** The `mars_harness_cli` repo shortcut now recognizes the repo-aware command paths that had drifted from the CLI surface, including `release backfill-notes`, `docsync audit`, `models evaluate`, `models override`, `scores`, and `trust`.

### Why
- **cli:** The CLI is the foundation control plane, but agents usually discover it through mirrored tools and generated skills. Without an explicit sync model and tests, every CLI change created a chance that target agents would keep invoking old commands, miss new repo flags, or choose generic shell execution over the intended tool path.
- **cli:** The previous documentation-sync model made code-to-doc ownership explicit, but it did not specifically cover the CLI-to-tool-and-skill mirrors that agents depend on when operating a deployed harness.

### What Changed
- **cli:** Added AD-103 in `docs/design-docs/cli-tool-skill-sync.md`, documenting the architecture, universal operating model, required mirrors, evidence commands, invariants, and failure mitigations for CLI tool/skill synchronization (ae9ac01).
- **cli:** Added command-tree tests that compare the live Cobra command graph with the `mars_harness_cli` reference and repo shortcut map, then exported the reference/shortcut helpers so the CLI package can enforce that mirror without stringly guessing (ae9ac01).
- **cli:** Mirrored the model into generated target harnesses with a new `cli-tool-sync` skill, generated AD-103 docs, knowledge routes, AGENTS guidance, F-001 scenario coverage, scanner assertions, and doctrine-sync checks (ae9ac01).

### Features
- **cli:** Enforce tool skill sync (ae9ac01)

## [0.29.1] - 2026-05-04
<!-- mars-harness-release: version=0.29.1 commit=8a718debfcbc -->

### Impact
- **docsync:** Operators and future agents now have a first-class Documentation Sync architecture and universal operating model, so "no stale documentation" is no longer just a rule spread across guidance. Every code change has an explicit path from changed files to associated docs, BDD contracts, audit evidence, and release notes.
- **docsync:** Initialized target harnesses inherit the same documentation-sync doctrine, feature scenario, knowledge route, and generated design doc, which keeps source and deployed operating models aligned as projects are scaffolded.

### Why
- **docsync:** The previous source-to-doc map made ownership auditable, but it did not yet explain the architecture, role responsibilities, maintenance workflows, or failure modes deeply enough for agents to apply the process consistently without chat context.
- **docsync:** The stale-doc problem spans foundation and deployed harnesses. Without a universal operating model mirrored into generated targets, new projects could receive the metadata gate but miss the decision-making workflow that tells agents when to update BDD, design docs, product specs, tool docs, role docs, or release notes.

### What Changed
- **docsync:** Added AD-102 in `docs/design-docs/documentation-sync-architecture.md`, documenting the six-layer architecture, the seven-step universal operating model, role responsibilities, maintenance workflows, invariants, mitigations, observability, and acceptance criteria for documentation sync (8a718de).
- **docsync:** Linked the new architecture from `AGENTS.md`, the design-doc index, the delivery operating model, the code-documentation map, the tools glossary, and the BDD feature catalog so agents can discover it from normal context assembly paths (8a718de).
- **docsync:** Mirrored AD-102 into generated target harness defaults, updated target knowledge routes and F-001 with a universal documentation-sync scenario, and expanded scanner/docs-consistency/formal workflow tests so the architecture remains release-blocking (8a718de).

### Documentation
- **docsync:** Document universal operating model (8a718de)

## [0.29.0] - 2026-05-04
<!-- mars-harness-release: version=0.29.0 commit=cb59e75cacf2 -->

### Impact
- **docsync:** Operators and agents now have a repo-wide source-to-documentation map. Every audited source file carries `MarsDocSync` metadata that points to the feature contracts, design docs, product specs, role docs, or release docs that must be checked when that file changes.
- **docsync:** The new `mars-harness docsync audit --repo .` command and mirrored `docsync_audit` tool turn no-stale-documentation from guidance into a repeatable quality gate.

### Why
- **docsync:** Documentation drift was still possible because a reviewer had to infer which docs belonged to a changed file. The new map makes that relationship explicit and lets automation fail when source files lack metadata, reference missing docs, or drift from the canonical package map.
- **docsync:** Generated target harnesses need the same universal operating model, so the scaffolded guidance, role allowlists, tools glossary, role registry, and F-001 feature contract now teach agents to run docsync before claiming code and docs are in sync.

### What Changed
- **docsync:** Added the `internal/docsync` package, `mars-harness docsync audit`, and the mirrored `docsync_audit` workflow tool, with tests for metadata parsing, missing-doc failures, CLI behavior, and docs-consistency enforcement (cb59e75).
- **docsync:** Added `docs/design-docs/code-documentation-map.md` as AD-101, extended F-001 with the source-wide docsync audit scenario, and updated no-stale-documentation doctrine to require a structured `docs:` array (cb59e75).
- **docsync:** Seeded `MarsDocSync` metadata across all audited source roots and updated generated target defaults so implementation, review, release, dogfood, and maintenance roles can use `docsync_audit` before git handoff (cb59e75).

### Features
- **docsync:** Map source files to documentation (cb59e75)

## [0.28.3] - 2026-05-04
<!-- mars-harness-release: version=0.28.3 commit=970e1659b42e -->

### Impact
- **release:** Operators see improved reliability because remove unused commit group helper.

### Why
- **release:** This matters because remove unused commit group helper closes a failure mode or degraded path.

### What Changed
- **release:** Changed remove unused commit group helper (970e165).

### Fixes
- **release:** Remove unused commit group helper (970e165)

## [0.28.2] - 2026-05-04
<!-- mars-harness-release: version=0.28.2 commit=4d26d7c9bf43 -->

### Impact
- **update:** Operators see improved reliability because resolve tagged private release assets.

### Why
- **update:** This matters because resolve tagged private release assets closes a failure mode or degraded path.

### What Changed
- **update:** Changed resolve tagged private release assets (4d26d7c).

### Fixes
- **update:** Resolve tagged private release assets (4d26d7c)

## [0.28.1] - 2026-05-04
<!-- mars-harness-release: version=0.28.1 commit=830ce4782671 -->

### Impact
- **update:** Operators see improved reliability because authenticate private release asset downloads.

### Why
- **update:** This matters because authenticate private release asset downloads closes a failure mode or degraded path.

### What Changed
- **update:** Changed authenticate private release asset downloads (830ce47).

### Fixes
- **update:** Authenticate private release asset downloads (830ce47)

## [0.28.0] - 2026-05-04
<!-- mars-harness-release: version=0.28.0 commit=2c912e4e3425 -->

### Impact
- **release:** Operators can bring every historical changelog entry onto the current Impact, Why, and What Changed release-note standard with a reusable checked command.

### Why
- **release:** Historical release notes were still on mixed narrative formats, which made the changelog a stale and uneven source of product communication.

### What Changed
- **release:** Added release backfill-notes with dry-run, check, and version-range support; backfilled 0.1.0 through 0.26.2; documented the BDD feature audit; and added consistency tests for changelog narrative sections (2c912e4).

### Features
- **release:** Backfill historical release narratives (2c912e4)

## [0.27.0] - 2026-05-04
<!-- mars-harness-release: version=0.27.0 commit=7d285a8ca7ad -->

### Impact
- **release:** Operators and maintainers get release notes that explain the actual impact of a change before the commit buckets.

### Why
- **release:** The previous generated summary was too thin for humans to understand why a release mattered or what changed without rereading the commits.

### What Changed
- **release:** Added generated Impact, Why, and What Changed sections, documented the universal release-note rule, and mirrored the guidance into target defaults (7d285a8).

### Features
- **release:** Generate detailed release narratives (7d285a8)

## [0.26.2] - 2026-05-04
<!-- mars-harness-release: version=0.26.2 commit=1dcd96de5bf9 -->

### Impact
- **operating-model:** Operators and future agents get clearer guidance because add no stale documentation rule.

### Why
- **operating-model:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.

### What Changed
- **operating-model:** Changed add no stale documentation rule (1dcd96d).

### Documentation
- **operating-model:** Add no stale documentation rule (1dcd96d)

## [0.26.1] - 2026-05-04
<!-- mars-harness-release: version=0.26.1 commit=dad878a5206c -->

### Impact
- **operating-model:** Operators and future agents get clearer guidance because make business logic first-class BDD.

### Why
- **operating-model:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.

### What Changed
- **operating-model:** Changed make business logic first-class BDD (dad878a).

### Documentation
- **operating-model:** Make business logic first-class BDD (dad878a)

## [0.26.0] - 2026-05-04
<!-- mars-harness-release: version=0.26.0 commit=38c1627fb039 -->

### Impact
- **cli:** Operators gain new capability: add target harness eject kill switch.

### Why
- **cli:** This matters because add target harness eject kill switch was missing from the shipped capability set.

### What Changed
- **cli:** Changed add target harness eject kill switch (38c1627).

### Features
- **cli:** Add target harness eject kill switch (38c1627)

## [0.25.1] - 2026-05-04
<!-- mars-harness-release: version=0.25.1 commit=42393b327478 -->

### Impact
- **planning:** Operators see improved reliability because enforce bootstrap artifact order.

### Why
- **planning:** This matters because enforce bootstrap artifact order closes a failure mode or degraded path.

### What Changed
- **planning:** Changed enforce bootstrap artifact order (42393b3).

### Fixes
- **planning:** Enforce bootstrap artifact order (42393b3)

## [0.25.0] - 2026-05-04
<!-- mars-harness-release: version=0.25.0 commit=8622d4122e83 -->

### Impact
- **orchestration:** Operators gain new capability: return dispatch handoffs to orchestrator.

### Why
- **orchestration:** This matters because return dispatch handoffs to orchestrator was missing from the shipped capability set.

### What Changed
- **orchestration:** Changed return dispatch handoffs to orchestrator (8622d41).

### Features
- **orchestration:** Return dispatch handoffs to orchestrator (8622d41)

## [0.24.16] - 2026-05-04
<!-- mars-harness-release: version=0.24.16 commit=b78f9f4e4b50 -->

### Impact
- **tickets:** Operators see improved reliability because enforce canonical ticket creation.

### Why
- **tickets:** This matters because enforce canonical ticket creation closes a failure mode or degraded path.

### What Changed
- **tickets:** Changed enforce canonical ticket creation (b78f9f4).

### Fixes
- **tickets:** Enforce canonical ticket creation (b78f9f4)

## [0.24.15] - 2026-05-04
<!-- mars-harness-release: version=0.24.15 commit=eb7e43cbf37e -->

### Impact
- **safety:** Operators see improved reliability because disable default file-count blast radius cap.

### Why
- **safety:** This matters because disable default file-count blast radius cap closes a failure mode or degraded path.

### What Changed
- **safety:** Changed disable default file-count blast radius cap (eb7e43c).

### Fixes
- **safety:** Disable default file-count blast radius cap (eb7e43c)

## [0.24.14] - 2026-05-04
<!-- mars-harness-release: version=0.24.14 commit=1bf62cd85b50 -->

### Impact
- **harness:** Operators and future agents get clearer guidance because define mirrored skill glossary.

### Why
- **harness:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.

### What Changed
- **harness:** Changed define mirrored skill glossary (1bf62cd).

### Documentation
- **harness:** Define mirrored skill glossary (1bf62cd)

## [0.24.13] - 2026-05-04
<!-- mars-harness-release: version=0.24.13 commit=4a6370f43596 -->

### Impact
- **init:** Operators see improved reliability because baseline generated scaffold across entrypoints.

### Why
- **init:** This matters because baseline generated scaffold across entrypoints closes a failure mode or degraded path.

### What Changed
- **init:** Changed baseline generated scaffold across entrypoints (4a6370f).

### Fixes
- **init:** Baseline generated scaffold across entrypoints (4a6370f)

## [0.24.12] - 2026-05-03
<!-- mars-harness-release: version=0.24.12 commit=7a09c57bc3aa -->

### Impact
- **start:** Operators see improved reliability because commit generated harness baseline.

### Why
- **start:** This matters because commit generated harness baseline closes a failure mode or degraded path.

### What Changed
- **start:** Changed commit generated harness baseline (7a09c57).

### Fixes
- **start:** Commit generated harness baseline (7a09c57)

## [0.24.11] - 2026-05-03
<!-- mars-harness-release: version=0.24.11 commit=310a5b052c6a -->

### Impact
- **harness:** Operators see improved reliability because add foundation containment gate.

### Why
- **harness:** This matters because add foundation containment gate closes a failure mode or degraded path.

### What Changed
- **harness:** Changed add foundation containment gate (310a5b0).

### Fixes
- **harness:** Add foundation containment gate (310a5b0)

## [0.24.10] - 2026-05-03
<!-- mars-harness-release: version=0.24.10 commit=7396ea0bc26f -->

### Impact
- **telemetry:** Operators see improved reliability because dedupe secondary intervention debt.

### Why
- **telemetry:** This matters because dedupe secondary intervention debt closes a failure mode or degraded path.

### What Changed
- **telemetry:** Changed dedupe secondary intervention debt (7396ea0).

### Fixes
- **telemetry:** Dedupe secondary intervention debt (7396ea0)

## [0.24.9] - 2026-05-03
<!-- mars-harness-release: version=0.24.9 commit=48655a2079dd -->

### Impact
- **trust:** Operators see improved reliability because honor bootstrap trust defaults.

### Why
- **trust:** This matters because honor bootstrap trust defaults closes a failure mode or degraded path.

### What Changed
- **trust:** Changed honor bootstrap trust defaults (48655a2).

### Fixes
- **trust:** Honor bootstrap trust defaults (48655a2)

## [0.24.8] - 2026-05-03
<!-- mars-harness-release: version=0.24.8 commit=0507ef14d1a4 -->

### Impact
- **tickets:** Operators and future agents get clearer guidance because record persistent store upgrade coverage gap.

### Why
- **tickets:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.

### What Changed
- **tickets:** Changed record persistent store upgrade coverage gap (0507ef1).

### Documentation
- **tickets:** Record persistent store upgrade coverage gap (0507ef1)

## [0.24.7] - 2026-05-03
<!-- mars-harness-release: version=0.24.7 commit=31f16cb2cfc7 -->

### Impact
- **queue:** Operators see improved reliability because migrate legacy job columns before indexes.

### Why
- **queue:** This matters because migrate legacy job columns before indexes closes a failure mode or degraded path.

### What Changed
- **queue:** Changed migrate legacy job columns before indexes (31f16cb).

### Fixes
- **queue:** Migrate legacy job columns before indexes (31f16cb)

## [0.24.6] - 2026-05-03
<!-- mars-harness-release: version=0.24.6 commit=c63ef60fc301 -->

### Impact
- **cli:** Operators see improved reliability because make evidence stores actionable.

### Why
- **cli:** This matters because make evidence stores actionable closes a failure mode or degraded path.

### What Changed
- **cli:** Changed make evidence stores actionable (c63ef60).

### Fixes
- **cli:** Make evidence stores actionable (c63ef60)

## [0.24.5] - 2026-05-03
<!-- mars-harness-release: version=0.24.5 commit=d3add7e85a82 -->

### Impact
- **ci:** Operators see improved reliability because check doctor test file write.

### Why
- **ci:** This matters because check doctor test file write closes a failure mode or degraded path.

### What Changed
- **ci:** Changed check doctor test file write (d3add7e).

### Fixes
- **ci:** Check doctor test file write (d3add7e)

## [0.24.4] - 2026-05-03
<!-- mars-harness-release: version=0.24.4 commit=eda0526868fd -->

### Impact
- **ci:** Operators see improved reliability because check serve test file setup.

### Why
- **ci:** This matters because check serve test file setup closes a failure mode or degraded path.

### What Changed
- **ci:** Changed check serve test file setup (eda0526).

### Fixes
- **ci:** Check serve test file setup (eda0526)

## [0.24.3] - 2026-05-03
<!-- mars-harness-release: version=0.24.3 commit=59f889ea7ed0 -->

### Impact
- **ci:** Operators see improved reliability because clear static lint findings.

### Why
- **ci:** This matters because clear static lint findings closes a failure mode or degraded path.

### What Changed
- **ci:** Changed clear static lint findings (59f889e).

### Fixes
- **ci:** Clear static lint findings (59f889e)

## [0.24.2] - 2026-05-03
<!-- mars-harness-release: version=0.24.2 commit=571bf7138d6c -->

### Impact
- **ci:** Operators see improved reliability because clear remaining lint findings.

### Why
- **ci:** This matters because clear remaining lint findings closes a failure mode or degraded path.

### What Changed
- **ci:** Changed clear remaining lint findings (571bf71).

### Fixes
- **ci:** Clear remaining lint findings (571bf71)

## [0.24.1] - 2026-05-03
<!-- mars-harness-release: version=0.24.1 commit=5a1472e12000 -->

### Impact
- **ci:** Operators see improved reliability because satisfy lint checks.

### Why
- **ci:** This matters because satisfy lint checks closes a failure mode or degraded path.

### What Changed
- **ci:** Changed satisfy lint checks (5a1472e).

### Fixes
- **ci:** Satisfy lint checks (5a1472e)

## [0.24.0] - 2026-05-03
<!-- mars-harness-release: version=0.24.0 commit=8d27b4ed3583 -->

### Impact
- **orchestration:** Operators gain new capability: add dispatch organization layer.

### Why
- **orchestration:** This matters because add dispatch organization layer was missing from the shipped capability set.

### What Changed
- **orchestration:** Changed add dispatch organization layer (8d27b4e).

### Features
- **orchestration:** Add dispatch organization layer (8d27b4e)

## [0.23.0] - 2026-05-03
<!-- mars-harness-release: version=0.23.0 commit=deccb88b12bb -->

### Impact
- **serve:** Operators gain new capability: add native orchestrator survey loop (MH-047).

### Why
- **serve:** This matters because add native orchestrator survey loop (MH-047) was missing from the shipped capability set.

### What Changed
- **serve:** Changed add native orchestrator survey loop (MH-047) (deccb88).

### Features
- **serve:** Add native orchestrator survey loop (MH-047) (deccb88)

### Delivery Evidence
- Enabler work: MH-047: Add native Orchestrator survey loop

## [0.22.1] - 2026-05-03
<!-- mars-harness-release: version=0.22.1 commit=ef3c15d1c115 -->

### Impact
- **sandbox:** Operators see improved reliability because fall back when linux namespaces are unavailable.

### Why
- **sandbox:** This matters because fall back when linux namespaces are unavailable closes a failure mode or degraded path.

### What Changed
- **sandbox:** Changed fall back when linux namespaces are unavailable (ef3c15d).

### Fixes
- **sandbox:** Fall back when linux namespaces are unavailable (ef3c15d)

## [0.22.0] - 2026-05-03
<!-- mars-harness-release: version=0.22.0 commit=44f2c8464e91 -->

### Impact
- **quality:** Operators gain new capability: harden recovery evidence and tool surface.
- **references:** Operators and future agents get clearer guidance because add OpenHarness comparator.

### Why
- **quality:** This matters because harden recovery evidence and tool surface was missing from the shipped capability set.
- **references:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.

### What Changed
- **quality:** Changed harden recovery evidence and tool surface (44f2c84).
- **references:** Changed add OpenHarness comparator (82efaa6).

### Features
- **quality:** Harden recovery evidence and tool surface (44f2c84)

### Documentation
- **references:** Add OpenHarness comparator (82efaa6)

## [0.21.0] - 2026-05-03
<!-- mars-harness-release: version=0.21.0 commit=cb32661e015e -->

### Impact
- **tickets:** Operators gain new capability: enforce in-progress drain states (MH-046).

### Why
- **tickets:** This matters because enforce in-progress drain states (MH-046) was missing from the shipped capability set.

### What Changed
- **tickets:** Changed enforce in-progress drain states (MH-046) (cb32661).

### Features
- **tickets:** Enforce in-progress drain states (MH-046) (cb32661)

### Delivery Evidence
- Enabler work: MH-046: Enforce in-progress ticket drain

## [0.20.0] - 2026-05-03
<!-- mars-harness-release: version=0.20.0 commit=5546e12b1874 -->

### Impact
- **serve:** Operators gain new capability: ingest intervention debt signals (MH-045).

### Why
- **serve:** This matters because ingest intervention debt signals (MH-045) was missing from the shipped capability set.

### What Changed
- **serve:** Changed ingest intervention debt signals (MH-045) (5546e12).

### Features
- **serve:** Ingest intervention debt signals (MH-045) (5546e12)

### Delivery Evidence
- Enabler work: MH-045: Complete intervention-debt signal ingestion

## [0.19.1] - 2026-05-03
<!-- mars-harness-release: version=0.19.1 commit=9b7e4bb50117 -->

### Impact
- **operating-model:** Operators and future agents get clearer guidance because add conversation system record guidance (MH-044).

### Why
- **operating-model:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.

### What Changed
- **operating-model:** Changed add conversation system record guidance (MH-044) (9b7e4bb).

### Documentation
- **operating-model:** Add conversation system record guidance (MH-044) (9b7e4bb)

### Delivery Evidence
- Enabler work: MH-044: Add conversation system record guidance

## [0.19.0] - 2026-05-03
<!-- mars-harness-release: version=0.19.0 commit=6a2d36a9be79 -->

### Impact
- **role-registry:** Operators gain new capability: add checked role inventory (MH-043).

### Why
- **role-registry:** This matters because add checked role inventory (MH-043) was missing from the shipped capability set.

### What Changed
- **role-registry:** Changed add checked role inventory (MH-043) (6a2d36a).

### Features
- **role-registry:** Add checked role inventory (MH-043) (6a2d36a)

### Delivery Evidence
- Enabler work: MH-043: Add checked role registry

## [0.18.0] - 2026-05-03
<!-- mars-harness-release: version=0.18.0 commit=d5436fdefd23 -->

### Impact
- **role-model:** Operators gain new capability: add canonical harness operating domains (MH-042).

### Why
- **role-model:** This matters because add canonical harness operating domains (MH-042) was missing from the shipped capability set.

### What Changed
- **role-model:** Changed add canonical harness operating domains (MH-042) (d5436fd).

### Features
- **role-model:** Add canonical harness operating domains (MH-042) (d5436fd)

### Delivery Evidence
- Enabler work: MH-042: Create canonical harness operating model

## [0.17.0] - 2026-05-03
<!-- mars-harness-release: version=0.17.0 commit=ed664ab2a36e -->

### Impact
- **tools:** Operators gain new capability: add tool creation guard.

### Why
- **tools:** This matters because add tool creation guard was missing from the shipped capability set.

### What Changed
- **tools:** Changed add tool creation guard (ed664ab).

### Features
- **tools:** Add tool creation guard (ed664ab)

## [0.16.0] - 2026-05-03
<!-- mars-harness-release: version=0.16.0 commit=5f9870bd9b08 -->

### Impact
- **models:** Operators gain new capability: add benchmark-backed provider workflow (MH-030).

### Why
- **models:** This matters because add benchmark-backed provider workflow (MH-030) was missing from the shipped capability set.

### What Changed
- **models:** Changed add benchmark-backed provider workflow (MH-030) (5f9870b).

### Features
- **models:** Add benchmark-backed provider workflow (MH-030) (5f9870b)

### Delivery Evidence
- Enabler work: MH-030: Benchmark-backed model refresh and promotion

## [0.15.2] - 2026-05-03
<!-- mars-harness-release: version=0.15.2 commit=027449036856 -->

### Impact
- **tools:** Operators and future agents get clearer guidance because require governed tool creation.

### Why
- **tools:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.

### What Changed
- **tools:** Changed require governed tool creation (0274490).

### Documentation
- **tools:** Require governed tool creation (0274490)

## [0.15.1] - 2026-05-03
<!-- mars-harness-release: version=0.15.1 commit=358216584c40 -->

### Impact
- **features:** Operators and future agents get clearer guidance because expand BDD contract catalog.

### Why
- **features:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.

### What Changed
- **features:** Changed expand BDD contract catalog (3582165).

### Documentation
- **features:** Expand BDD contract catalog (3582165)

## [0.15.0] - 2026-05-03
<!-- mars-harness-release: version=0.15.0 commit=b9b84535812f -->

### Impact
- **tools:** Operators gain new capability: formalize repeated workflow tools.

### Why
- **tools:** This matters because formalize repeated workflow tools was missing from the shipped capability set.

### What Changed
- **tools:** Changed formalize repeated workflow tools (b9b8453).

### Features
- **tools:** Formalize repeated workflow tools (b9b8453)

## [0.14.6] - 2026-05-03
<!-- mars-harness-release: version=0.14.6 commit=3ca1a420b043 -->

### Impact
- **tickets:** Operators and future agents get clearer guidance because complete release asset contract (MH-031).

### Why
- **tickets:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.

### What Changed
- **tickets:** Changed complete release asset contract (MH-031) (3ca1a42).

### Documentation
- **tickets:** Complete release asset contract (MH-031) (3ca1a42)

### Delivery Evidence
- Enabler work: MH-031: Publish release binary assets for installer

## [0.14.5] - 2026-05-03
<!-- mars-harness-release: version=0.14.5 commit=227b6f718abf -->

### Impact
- **ui:** Operators see improved reliability because support linux terminal ioctl constants (MH-031).

### Why
- **ui:** This matters because support linux terminal ioctl constants (MH-031) closes a failure mode or degraded path.

### What Changed
- **ui:** Changed support linux terminal ioctl constants (MH-031) (227b6f7).

### Fixes
- **ui:** Support linux terminal ioctl constants (MH-031) (227b6f7)

## [0.14.4] - 2026-05-03
<!-- mars-harness-release: version=0.14.4 commit=be63396bb21a -->

### Impact
- **release:** Operators see improved reliability because backfill notes-only release assets (MH-031).

### Why
- **release:** This matters because backfill notes-only release assets (MH-031) closes a failure mode or degraded path.

### What Changed
- **release:** Changed backfill notes-only release assets (MH-031) (be63396).

### Fixes
- **release:** Backfill notes-only release assets (MH-031) (be63396)

## [0.14.3] - 2026-05-03
<!-- mars-harness-release: version=0.14.3 commit=ed9853b52bd3 -->

### Impact
- **architecture:** Operators and future agents get clearer guidance because update current system map.

### Why
- **architecture:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.

### What Changed
- **architecture:** Changed update current system map (ed9853b).

### Documentation
- **architecture:** Update current system map (ed9853b)

## [0.14.2] - 2026-05-03
<!-- mars-harness-release: version=0.14.2 commit=9fe9b5857df7 -->

### Impact
- **operating-model:** Operators and future agents get clearer guidance because require symbiotic workflow changes.

### Why
- **operating-model:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.

### What Changed
- **operating-model:** Changed require symbiotic workflow changes (9fe9b58).

### Documentation
- **operating-model:** Require symbiotic workflow changes (9fe9b58)

## [0.14.1] - 2026-05-03
<!-- mars-harness-release: version=0.14.1 commit=195c73e1183c -->

### Impact
- **tools:** Operators and future agents get clearer guidance because add mirrored tools glossary.

### Why
- **tools:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.

### What Changed
- **tools:** Changed add mirrored tools glossary (195c73e).

### Documentation
- **tools:** Add mirrored tools glossary (195c73e)

## [0.14.0] - 2026-05-03
<!-- mars-harness-release: version=0.14.0 commit=422adac6f6ad -->

### Impact
- **tools:** Operators gain new capability: add mirrored mars harness cli tool.

### Why
- **tools:** This matters because add mirrored mars harness cli tool was missing from the shipped capability set.

### What Changed
- **tools:** Changed add mirrored mars harness cli tool (422adac).

### Features
- **tools:** Add mirrored mars harness cli tool (422adac)

## [0.13.1] - 2026-05-03
<!-- mars-harness-release: version=0.13.1 commit=cd0ffd67d96a -->

### Impact
- **tickets:** Operators and future agents get clearer guidance because record release asset blocker (MH-031).

### Why
- **tickets:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.

### What Changed
- **tickets:** Changed record release asset blocker (MH-031) (cd0ffd6).

### Documentation
- **tickets:** Record release asset blocker (MH-031) (cd0ffd6)

## [0.13.0] - 2026-05-03
<!-- mars-harness-release: version=0.13.0 commit=ccd36bc3bf3d -->

### Impact
- **release:** Operators gain new capability: verify release assets for self-update (MH-031).
- **release:** Operators see improved reliability because ignore stale changelog markers (MH-031).

### Why
- **release:** This matters because verify release assets for self-update (MH-031) was missing from the shipped capability set.
- **release:** This matters because ignore stale changelog markers (MH-031) closes a failure mode or degraded path.

### What Changed
- **release:** Changed verify release assets for self-update (MH-031) (9e027a2).
- **release:** Changed ignore stale changelog markers (MH-031) (ccd36bc).

### Features
- **release:** Verify release assets for self-update (MH-031) (9e027a2)

### Fixes
- **release:** Ignore stale changelog markers (MH-031) (ccd36bc)

## [0.12.1] - 2026-05-03
<!-- mars-harness-release: version=0.12.1 commit=d8e8c6fcc990 -->

### Impact
- **glossary:** Operators and future agents get clearer guidance because define operating model distinctions.

### Why
- **glossary:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.

### What Changed
- **glossary:** Changed define operating model distinctions (d8e8c6f).

### Documentation
- **glossary:** Define operating model distinctions (d8e8c6f)

## [0.12.0] - 2026-05-03
<!-- mars-harness-release: version=0.12.0 commit=416a91bd5fa1 -->

### Impact
- **scoring:** Operators gain new capability: export repo quality score (MH-037).

### Why
- **scoring:** This matters because export repo quality score (MH-037) was missing from the shipped capability set.

### What Changed
- **scoring:** Changed export repo quality score (MH-037) (416a91b).

### Features
- **scoring:** Export repo quality score (MH-037) (416a91b)

### Delivery Evidence
- Enabler work: MH-037: Automate quality score export

## [0.11.1] - 2026-05-03
<!-- mars-harness-release: version=0.11.1 commit=450d1bbbbbd9 -->

### Impact
- **tools:** Operators see improved reliability because mirror tool_create in target harness.

### Why
- **tools:** This matters because mirror tool_create in target harness closes a failure mode or degraded path.

### What Changed
- **tools:** Changed mirror tool_create in target harness (450d1bb).

### Fixes
- **tools:** Mirror tool_create in target harness (450d1bb)

## [0.11.0] - 2026-05-03
<!-- mars-harness-release: version=0.11.0 commit=a00bb9e11730 -->

### Impact
- **tools:** Operators gain new capability: add tool creation scaffold.

### Why
- **tools:** This matters because add tool creation scaffold was missing from the shipped capability set.

### What Changed
- **tools:** Changed add tool creation scaffold (a00bb9e).

### Features
- **tools:** Add tool creation scaffold (a00bb9e)

## [0.10.4] - 2026-05-03
<!-- mars-harness-release: version=0.10.4 commit=f133e393f97a -->

### Impact
- **glossary:** Operators and future agents get clearer guidance because mirror harness terminology.

### Why
- **glossary:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.

### What Changed
- **glossary:** Changed mirror harness terminology (f133e39).

### Documentation
- **glossary:** Mirror harness terminology (f133e39)

## [0.10.3] - 2026-05-03
<!-- mars-harness-release: version=0.10.3 commit=9e444541196c -->

### Impact
- **planning:** Operators and future agents get clearer guidance because materialize mars parity backlog tickets (MH-035).

### Why
- **planning:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.

### What Changed
- **planning:** Changed materialize mars parity backlog tickets (MH-035) (9e44454).

### Documentation
- **planning:** Materialize mars parity backlog tickets (MH-035) (9e44454)

### Delivery Evidence
- Enabler work: MH-035: Materialize Mars parity workstreams as tickets

## [0.10.2] - 2026-05-03
<!-- mars-harness-release: version=0.10.2 commit=e2bcf2f7a080 -->

### Impact
- **telemetry:** Operators see improved reliability because classify ticket gate failures.

### Why
- **telemetry:** This matters because classify ticket gate failures closes a failure mode or degraded path.

### What Changed
- **telemetry:** Changed classify ticket gate failures (e2bcf2f).

### Fixes
- **telemetry:** Classify ticket gate failures (e2bcf2f)

## [0.10.1] - 2026-05-03
<!-- mars-harness-release: version=0.10.1 commit=bb885cd5ad7e -->

### Impact
- **inference:** Operators see improved reliability because surface installed model variants.

### Why
- **inference:** This matters because surface installed model variants closes a failure mode or degraded path.

### What Changed
- **inference:** Changed surface installed model variants (bb885cd).

### Fixes
- **inference:** Surface installed model variants (bb885cd)

## [0.10.0] - 2026-05-03
<!-- mars-harness-release: version=0.10.0 commit=0f4d9ec86ceb -->

### Impact
- **planhygiene:** Operators gain new capability: add active plan hygiene checker (MH-034).

### Why
- **planhygiene:** This matters because add active plan hygiene checker (MH-034) was missing from the shipped capability set.

### What Changed
- **planhygiene:** Changed add active plan hygiene checker (MH-034) (0f4d9ec).

### Features
- **planhygiene:** Add active plan hygiene checker (MH-034) (0f4d9ec)

### Delivery Evidence
- Enabler work: MH-034: Implement active-plan hygiene checker

## [0.9.0] - 2026-05-02
<!-- mars-harness-release: version=0.9.0 commit=c3a87e2179e3 -->

### Impact
- **setup:** Operators gain new capability: configure shell path automatically (MH-041).

### Why
- **setup:** This matters because configure shell path automatically (MH-041) was missing from the shipped capability set.

### What Changed
- **setup:** Changed configure shell path automatically (MH-041) (c3a87e2).

### Features
- **setup:** Configure shell path automatically (MH-041) (c3a87e2)

### Delivery Evidence
- Shipped feature scenarios: MH-041: F-002-S001, F-002-S002, F-002-S003, F-002-S004, F-002-S005

## [0.8.0] - 2026-05-02
<!-- mars-harness-release: version=0.8.0 commit=cd7514dfdce5 -->

### Impact
- **operating-model:** Operators gain new capability: implement BDD-led delivery loop (MH-040).

### Why
- **operating-model:** This matters because implement BDD-led delivery loop (MH-040) was missing from the shipped capability set.

### What Changed
- **operating-model:** Changed implement BDD-led delivery loop (MH-040) (cd7514d).

### Features
- **operating-model:** Implement BDD-led delivery loop (MH-040) (cd7514d)

### Delivery Evidence
- Shipped feature scenarios: MH-040: F-001-S001, F-001-S002, F-001-S003, F-001-S004, F-001-S005, F-001-S006

## [0.7.5] - 2026-05-02
<!-- mars-harness-release: version=0.7.5 commit=e39e335c8fc2 -->

### Impact
- **plans:** Operators and future agents get clearer guidance because add exec plan dependency metadata.

### Why
- **plans:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.

### What Changed
- **plans:** Changed add exec plan dependency metadata (e39e335).

### Documentation
- **plans:** Add exec plan dependency metadata (e39e335)

## [0.7.4] - 2026-05-02
<!-- mars-harness-release: version=0.7.4 commit=c7dbdf3dbb6e -->

### Impact
- **plans:** Operators and future agents get clearer guidance because enforce single active exec plan.

### Why
- **plans:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.

### What Changed
- **plans:** Changed enforce single active exec plan (c7dbdf3).

### Documentation
- **plans:** Enforce single active exec plan (c7dbdf3)

## [0.7.3] - 2026-05-02
<!-- mars-harness-release: version=0.7.3 commit=9a4ced42bdad -->

### Impact
- **scoring:** Operators and future agents get clearer guidance because seed quality score artifact.

### Why
- **scoring:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.

### What Changed
- **scoring:** Changed seed quality score artifact (9a4ced4).

### Documentation
- **scoring:** Seed quality score artifact (9a4ced4)

## [0.7.2] - 2026-05-02
<!-- mars-harness-release: version=0.7.2 commit=dac23b716ce3 -->

### Impact
- **plans:** Operators and future agents get clearer guidance because reconcile current execution state.

### Why
- **plans:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.

### What Changed
- **plans:** Changed reconcile current execution state (dac23b7).

### Documentation
- **plans:** Reconcile current execution state (dac23b7)

## [0.7.1] - 2026-05-02
<!-- mars-harness-release: version=0.7.1 commit=21d617f832ef -->

### Impact
- **update:** The release carries stronger evidence because keep version drift fixtures release-agnostic.

### Why
- **update:** This matters because the project needs durable evidence that the behavior keeps working.

### What Changed
- **update:** Changed keep version drift fixtures release-agnostic (21d617f).

### Tests
- **update:** Keep version drift fixtures release-agnostic (21d617f)

## [0.7.0] - 2026-05-02
<!-- mars-harness-release: version=0.7.0 commit=ce831c5cd4de -->

### Impact
- **update:** Operators gain new capability: check tool and harness version drift.

### Why
- **update:** This matters because check tool and harness version drift was missing from the shipped capability set.

### What Changed
- **update:** Changed check tool and harness version drift (ce831c5).

### Features
- **update:** Check tool and harness version drift (ce831c5)

## [0.6.0] - 2026-05-02
<!-- mars-harness-release: version=0.6.0 commit=2187d5a379c3 -->

### Impact
- **update:** Operators gain new capability: unify tool and harness updates.

### Why
- **update:** This matters because unify tool and harness updates was missing from the shipped capability set.

### What Changed
- **update:** Changed unify tool and harness updates (2187d5a).

### Features
- **update:** Unify tool and harness updates (2187d5a)

## [0.5.3] - 2026-05-02
<!-- mars-harness-release: version=0.5.3 commit=781c1e5051dd -->

### Impact
- **setup:** Operators see improved reliability because clarify source install workflow.

### Why
- **setup:** This matters because clarify source install workflow closes a failure mode or degraded path.

### What Changed
- **setup:** Changed clarify source install workflow (781c1e5).

### Fixes
- **setup:** Clarify source install workflow (781c1e5)

## [0.5.2] - 2026-05-02
<!-- mars-harness-release: version=0.5.2 commit=4a599310de29 -->

### Impact
- **models:** Operators and future agents get clearer guidance because define ollama swap policy.

### Why
- **models:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.

### What Changed
- **models:** Changed define ollama swap policy (4a59931).

### Documentation
- **models:** Define ollama swap policy (4a59931)

## [0.5.1] - 2026-05-02
<!-- mars-harness-release: version=0.5.1 commit=8f0a44f12017 -->

### Impact
- **telemetry:** Operators see improved reliability because keep intervention tickets independent.

### Why
- **telemetry:** This matters because keep intervention tickets independent closes a failure mode or degraded path.

### What Changed
- **telemetry:** Changed keep intervention tickets independent (8f0a44f).

### Fixes
- **telemetry:** Keep intervention tickets independent (8f0a44f)

## [0.5.0] - 2026-05-02
<!-- mars-harness-release: version=0.5.0 commit=0ca0257223cd -->

### Impact
- **telemetry:** Operators gain new capability: create intervention-debt tickets.

### Why
- **telemetry:** This matters because create intervention-debt tickets was missing from the shipped capability set.

### What Changed
- **telemetry:** Changed create intervention-debt tickets (0ca0257).

### Features
- **telemetry:** Create intervention-debt tickets (0ca0257)

## [0.4.1] - 2026-05-02
<!-- mars-harness-release: version=0.4.1 commit=548fb73403a1 -->

### Impact
- **inference:** Operators see improved reliability because route roles by manifest tier.

### Why
- **inference:** This matters because route roles by manifest tier closes a failure mode or degraded path.

### What Changed
- **inference:** Changed route roles by manifest tier (548fb73).

### Fixes
- **inference:** Route roles by manifest tier (548fb73)

## [0.4.0] - 2026-05-02
<!-- mars-harness-release: version=0.4.0 commit=72032c5985e4 -->

### Impact
- **models:** Operators gain new capability: add benchmark evaluation path.

### Why
- **models:** This matters because add benchmark evaluation path was missing from the shipped capability set.

### What Changed
- **models:** Changed add benchmark evaluation path (72032c5).

### Features
- **models:** Add benchmark evaluation path (72032c5)

## [0.3.6] - 2026-05-02
<!-- mars-harness-release: version=0.3.6 commit=ecf0f5596249 -->

### Impact
- **queue:** Operators see improved reliability because self-heal recovery storms.

### Why
- **queue:** This matters because self-heal recovery storms closes a failure mode or degraded path.

### What Changed
- **queue:** Changed self-heal recovery storms (ecf0f55).

### Fixes
- **queue:** Self-heal recovery storms (ecf0f55)

## [0.3.5] - 2026-05-02
<!-- mars-harness-release: version=0.3.5 commit=4769fb4172da -->

### Impact
- **serve:** Operators see improved reliability because contain recursive recovery jobs.

### Why
- **serve:** This matters because contain recursive recovery jobs closes a failure mode or degraded path.

### What Changed
- **serve:** Changed contain recursive recovery jobs (4769fb4).

### Fixes
- **serve:** Contain recursive recovery jobs (4769fb4)

## [0.3.4] - 2026-05-02
<!-- mars-harness-release: version=0.3.4 commit=5fef93f4bc04 -->

### Impact
- **release:** Operators and future agents get clearer guidance because require github release publication.

### Why
- **release:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.

### What Changed
- **release:** Changed require github release publication (5fef93f).

### Documentation
- **release:** Require github release publication (5fef93f)

## [0.3.3] - 2026-05-02
<!-- mars-harness-release: version=0.3.3 commit=3232920f527f -->

### Impact
- **harness:** Operators and future agents get clearer guidance because mirror operating rules into targets.

### Why
- **harness:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.

### What Changed
- **harness:** Changed mirror operating rules into targets (3232920).

### Documentation
- **harness:** Mirror operating rules into targets (3232920)

## [0.3.2] - 2026-05-02
<!-- mars-harness-release: version=0.3.2 commit=5c5bc2d6761b -->

### Impact
- **release:** Operators and future agents get clearer guidance because mirror versioning rule into targets.

### Why
- **release:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.

### What Changed
- **release:** Changed mirror versioning rule into targets (5c5bc2d).

### Documentation
- **release:** Mirror versioning rule into targets (5c5bc2d)

## [0.3.1] - 2026-05-02
<!-- mars-harness-release: version=0.3.1 commit=466bc65ad438 -->

### Impact
- **release:** Operators and future agents get clearer guidance because require versioning after source commits.

### Why
- **release:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.

### What Changed
- **release:** Changed require versioning after source commits (466bc65).

### Documentation
- **release:** Require versioning after source commits (466bc65)

## [0.3.0] - 2026-05-02
<!-- mars-harness-release: version=0.3.0 commit=b2cd7df5f2e5 -->

### Impact
- **skills:** Operators gain new capability: guide self-improving skill evolution.

### Why
- **skills:** This matters because guide self-improving skill evolution was missing from the shipped capability set.

### What Changed
- **skills:** Changed guide self-improving skill evolution (b2cd7df).

### Features
- **skills:** Guide self-improving skill evolution (b2cd7df)

## [0.2.0] - 2026-05-02
<!-- mars-harness-release: version=0.2.0 commit=15f4b154182d -->

### Impact
- **release:** Operators gain new capability: automate semantic patch notes.

### Why
- **release:** This matters because automate semantic patch notes was missing from the shipped capability set.

### What Changed
- **release:** Changed automate semantic patch notes (15f4b15).

### Features
- **release:** Automate semantic patch notes (15f4b15)

## [0.1.0] - 2026-05-02
<!-- mars-harness-release: version=0.1.0 commit=edaafeacae3a -->

### Impact
- **tools:** Operators gain new capability: mechanical ticket deduplication with ticket_create tool (AD-030).
- **tools,scanner:** Operators gain new capability: wire git tools into manifests and add commit gates (AD-028).
- **pipeline:** Operators gain new capability: chain dogfood tester after engineer completes a feature.
- Operators gain new capability: dogfood E2E validation with Podman + decision recording system.
- **dashboard,scanner:** Operators gain new capability: dynamic pipeline chain + tsconfig path alias check.
- **m6,m7:** Operators gain new capability: scoring engine, progressive autonomy, evolution system, guardrails engine (MH-017 through MH-021).
- **telemetry:** Operators gain new capability: triage self-improvement signals.
- **tools:** Operators gain new capability: add registry, executor, and core tools (M1.2 / MH-002).
- **scanner:** Operators gain new capability: add bootability checks for framework-specific validation (AD-026).
- **m5:** Operators gain new capability: job queue, scheduler, sandbox, safety, serve command (MH-013, MH-014, MH-015, MH-016).
- **serve:** Operators gain new capability: wire autonomous orchestrator with repo registry, trigger router, and executor.
- **agent:** Operators gain new capability: add conversation loop, parser, and tests (M1.3 / MH-003).
- **scanner:** Operators gain new capability: add mars-harness upgrade command to sync target project manifests and prompts.
- **setup:** Operators gain new capability: plug-and-play model download from HuggingFace.
- **skills:** Operators gain new capability: add agent skills across Cursor, AGENTS.md, and .harness/skills/.
- **ui:** Operators gain new capability: cursor-quality agent output with role banner, tool trace, and handoff.
- **setup:** Operators gain new capability: auto-install llama-server and wire run command.
- **llm:** Operators gain new capability: add OpenAI-compatible chat client and testutil helpers.
- **init:** Operators gain new capability: provision full 11-role Mars pipeline by default (Tenet 1).
- **serve:** Operators gain new capability: parallel pipeline tracks with sleep resilience.
- **dashboard:** Operators gain new capability: wire dashboard into orchestrator with SSE events.
- Operators gain new capability: implement two-level self-learning system with janitor agent.
- **init:** Operators gain new capability: auto-run git init when .git is missing.
- **cli:** Operators gain new capability: auto-init .harness when manifest is missing.
- **m9:** Operators gain new capability: embedded dashboard with htmx, Chart.js, SSE, emergency stop (MH-024).
- **context:** Operators gain new capability: add context assembler with token budget (M1.4 / MH-004).
- **m8:** Operators gain new capability: setup wizard, init + scanner, doctor command (MH-022, MH-023, MH-028).
- **init:** Operators gain new capability: scaffold full docs/ structure with Mars-quality role prompts.
- **telemetry:** Operators gain new capability: add telemetry collector with error classification and auto-fix.
- **core:** Operators gain new capability: enforce strict trunk execution safety.
- **tools:** Operators gain new capability: add background mode to shell_exec for long-running processes.
- **m3,m4:** Operators gain new capability: cLI framework, bundle reader, terminal UI, GitHub integration (MH-009, MH-010, MH-011, MH-012).
- **inference:** Operators gain new capability: add llama-server subprocess manager and role router.
- **context:** Operators gain new capability: file filter, budget partitioning, .harnessignore support (M1.4.2, M1.4.4, MH-004).
- **m10:** Operators gain new capability: role prompts, dogfood workflow, cross-compile release, user docs (MH-025, MH-026, MH-027).
- **start:** Operators gain new capability: add `mars-harness start` for full e2e pipeline orchestration.
- **dashboard:** Operators gain new capability: implement throughput page with chart, stats, and job table.
- **m2:** Operators gain new capability: hardware detection, model download, llama-server management, LLM router (MH-006, MH-007, MH-008).
- **orchestrator:** Operators gain new capability: configurable agent triggers with chaining and custom cron.
- **agent:** Operators gain new capability: m1 closeout — 3+ tool-call test, trace error propagation, realistic 5-turn test (MH-003, MH-005).
- **inference:** Operators gain new capability: expose local performance tuning.
- **prompts:** Operators gain new capability: add git push after every commit in all role prompts.
- **serve:** Operators gain new capability: per-repo database isolation to prevent cross-project contamination (AD-029).
- **init:** Operators gain new capability: mirror harness context glossary.
- **trace:** Operators gain new capability: add JSONL recorder and SQLite store (M1.5 / MH-005).
- **tools:** Operators see improved reliability because kill entire process group on shell_exec timeout.
- **inference:** Operators see improved reliability because mitigate LLM timeout, context overflow, and connection refused failures.
- **prompts:** Operators see improved reliability because cold-start CEO/COO prompts for new projects.
- **setup:** Operators see improved reliability because pin local runtime artifacts.
- **serve:** Operators see improved reliability because block engineer handoff with active tickets.
- **agent:** Operators see improved reliability because always serialize message content field for llama.cpp compat.
- **agent:** Operators see improved reliability because add context window guard to prevent token overflow.
- Operators see improved reliability because resolve audit findings — broken links, naming inconsistencies, missing docs (M0).
- **init:** Operators see improved reliability because preserve existing user content on --force re-init.
- **core:** Operators see improved reliability because auto-tune inference and drain active tickets.
- **github:** Operators see improved reliability because keep integrations trunk oriented.
- Operators see improved reliability because correct module path, add gitkeeps, fix placeholders (M0).
- **pipeline:** Operators see improved reliability because restore COO → Engineer handoff for delivery kickoff.
- **upgrade:** Operators see improved reliability because preserve user configured agents.
- **serve:** Operators see improved reliability because auto-cleanup stale processes and corrupt DB on start/serve.
- **dashboard:** Operators see improved reliability because constrain chart canvas height to 280px.
- **references:** Operators and future agents get clearer guidance because carry mars agent-first references.
- **generated:** Operators and future agents get clearer guidance because define generated docs contract.
- **tickets:** Operators and future agents get clearer guidance because populate full backlog MH-001 through MH-028 (M0).
- Operators and future agents get clearer guidance because add terminology definitions and dual-repo commit discipline.
- **product:** Operators and future agents get clearer guidance because refresh living product specs.
- Operators and future agents get clearer guidance because switch to trunk-based development, drop branch/PR requirement.
- **workflow:** Operators and future agents get clearer guidance because align generated bundles with strict trunk.
- **design:** Operators and future agents get clearer guidance because record AD-031 inference resilience decisions.
- **references:** Operators and future agents get clearer guidance because audit mars relevance for harness parity.
- **exec-plans:** Operators and future agents get clearer guidance because add master execution plan with M0–M10 + MH-001–MH-028 coverage.
- **plans:** Operators and future agents get clearer guidance because add mars supersession parity plan.
- Operators and future agents get clearer guidance because add AD-021 through AD-025 for dogfood, decisions, and lean pipeline.
- **tickets:** Operators and future agents get clearer guidance because reconcile 19 ticket-vs-schedule contradictions (C1–C19).
- **tickets:** Maintainers get a healthier project surface because move MH-001 through MH-005 to done/ (M1 closeout).
- **m0:** Maintainers get a healthier project surface because audit and fix M0 quality gate gaps.
- **tickets:** Maintainers get a healthier project surface because move MH-006 through MH-008 to done/ (M2 closeout).
- Maintainers get a healthier project surface because initialize mars-harness repo (M0).
- **tickets:** Maintainers get a healthier project surface because move MH-009 through MH-012 to done/ (M3+M4 closeout).
- **tickets:** Maintainers get a healthier project surface because move MH-017 through MH-021 to done/ (M6+M7 closeout).
- **tickets:** Maintainers get a healthier project surface because move MH-013 through MH-016 to done/ (M5a+M5b closeout).
- **serve:** The release carries stronger evidence because fix serve tests for new Config requirements, add skills loader tests.

### Why
- **tools:** This matters because mechanical ticket deduplication with ticket_create tool (AD-030) was missing from the shipped capability set.
- **tools,scanner:** This matters because wire git tools into manifests and add commit gates (AD-028) was missing from the shipped capability set.
- **pipeline:** This matters because chain dogfood tester after engineer completes a feature was missing from the shipped capability set.
- This matters because dogfood E2E validation with Podman + decision recording system was missing from the shipped capability set.
- **dashboard,scanner:** This matters because dynamic pipeline chain + tsconfig path alias check was missing from the shipped capability set.
- **m6,m7:** This matters because scoring engine, progressive autonomy, evolution system, guardrails engine (MH-017 through MH-021) was missing from the shipped capability set.
- **telemetry:** This matters because triage self-improvement signals was missing from the shipped capability set.
- **tools:** This matters because add registry, executor, and core tools (M1.2 / MH-002) was missing from the shipped capability set.
- **scanner:** This matters because add bootability checks for framework-specific validation (AD-026) was missing from the shipped capability set.
- **m5:** This matters because job queue, scheduler, sandbox, safety, serve command (MH-013, MH-014, MH-015, MH-016) was missing from the shipped capability set.
- **serve:** This matters because wire autonomous orchestrator with repo registry, trigger router, and executor was missing from the shipped capability set.
- **agent:** This matters because add conversation loop, parser, and tests (M1.3 / MH-003) was missing from the shipped capability set.
- **scanner:** This matters because add mars-harness upgrade command to sync target project manifests and prompts was missing from the shipped capability set.
- **setup:** This matters because plug-and-play model download from HuggingFace was missing from the shipped capability set.
- **skills:** This matters because add agent skills across Cursor, AGENTS.md, and .harness/skills/ was missing from the shipped capability set.
- **ui:** This matters because cursor-quality agent output with role banner, tool trace, and handoff was missing from the shipped capability set.
- **setup:** This matters because auto-install llama-server and wire run command was missing from the shipped capability set.
- **llm:** This matters because add OpenAI-compatible chat client and testutil helpers was missing from the shipped capability set.
- **init:** This matters because provision full 11-role Mars pipeline by default (Tenet 1) was missing from the shipped capability set.
- **serve:** This matters because parallel pipeline tracks with sleep resilience was missing from the shipped capability set.
- **dashboard:** This matters because wire dashboard into orchestrator with SSE events was missing from the shipped capability set.
- This matters because implement two-level self-learning system with janitor agent was missing from the shipped capability set.
- **init:** This matters because auto-run git init when .git is missing was missing from the shipped capability set.
- **cli:** This matters because auto-init .harness when manifest is missing was missing from the shipped capability set.
- **m9:** This matters because embedded dashboard with htmx, Chart.js, SSE, emergency stop (MH-024) was missing from the shipped capability set.
- **context:** This matters because add context assembler with token budget (M1.4 / MH-004) was missing from the shipped capability set.
- **m8:** This matters because setup wizard, init + scanner, doctor command (MH-022, MH-023, MH-028) was missing from the shipped capability set.
- **init:** This matters because scaffold full docs/ structure with Mars-quality role prompts was missing from the shipped capability set.
- **telemetry:** This matters because add telemetry collector with error classification and auto-fix was missing from the shipped capability set.
- **core:** This matters because enforce strict trunk execution safety was missing from the shipped capability set.
- **tools:** This matters because add background mode to shell_exec for long-running processes was missing from the shipped capability set.
- **m3,m4:** This matters because cLI framework, bundle reader, terminal UI, GitHub integration (MH-009, MH-010, MH-011, MH-012) was missing from the shipped capability set.
- **inference:** This matters because add llama-server subprocess manager and role router was missing from the shipped capability set.
- **context:** This matters because file filter, budget partitioning, .harnessignore support (M1.4.2, M1.4.4, MH-004) was missing from the shipped capability set.
- **m10:** This matters because role prompts, dogfood workflow, cross-compile release, user docs (MH-025, MH-026, MH-027) was missing from the shipped capability set.
- **start:** This matters because add `mars-harness start` for full e2e pipeline orchestration was missing from the shipped capability set.
- **dashboard:** This matters because implement throughput page with chart, stats, and job table was missing from the shipped capability set.
- **m2:** This matters because hardware detection, model download, llama-server management, LLM router (MH-006, MH-007, MH-008) was missing from the shipped capability set.
- **orchestrator:** This matters because configurable agent triggers with chaining and custom cron was missing from the shipped capability set.
- **agent:** This matters because m1 closeout — 3+ tool-call test, trace error propagation, realistic 5-turn test (MH-003, MH-005) was missing from the shipped capability set.
- **inference:** This matters because expose local performance tuning was missing from the shipped capability set.
- **prompts:** This matters because add git push after every commit in all role prompts was missing from the shipped capability set.
- **serve:** This matters because per-repo database isolation to prevent cross-project contamination (AD-029) was missing from the shipped capability set.
- **init:** This matters because mirror harness context glossary was missing from the shipped capability set.
- **trace:** This matters because add JSONL recorder and SQLite store (M1.5 / MH-005) was missing from the shipped capability set.
- **tools:** This matters because kill entire process group on shell_exec timeout closes a failure mode or degraded path.
- **inference:** This matters because mitigate LLM timeout, context overflow, and connection refused failures closes a failure mode or degraded path.
- **prompts:** This matters because cold-start CEO/COO prompts for new projects closes a failure mode or degraded path.
- **setup:** This matters because pin local runtime artifacts closes a failure mode or degraded path.
- **serve:** This matters because block engineer handoff with active tickets closes a failure mode or degraded path.
- **agent:** This matters because always serialize message content field for llama.cpp compat closes a failure mode or degraded path.
- **agent:** This matters because add context window guard to prevent token overflow closes a failure mode or degraded path.
- This matters because resolve audit findings — broken links, naming inconsistencies, missing docs (M0) closes a failure mode or degraded path.
- **init:** This matters because preserve existing user content on --force re-init closes a failure mode or degraded path.
- **core:** This matters because auto-tune inference and drain active tickets closes a failure mode or degraded path.
- **github:** This matters because keep integrations trunk oriented closes a failure mode or degraded path.
- This matters because correct module path, add gitkeeps, fix placeholders (M0) closes a failure mode or degraded path.
- **pipeline:** This matters because restore COO → Engineer handoff for delivery kickoff closes a failure mode or degraded path.
- **upgrade:** This matters because preserve user configured agents closes a failure mode or degraded path.
- **serve:** This matters because auto-cleanup stale processes and corrupt DB on start/serve closes a failure mode or degraded path.
- **dashboard:** This matters because constrain chart canvas height to 280px closes a failure mode or degraded path.
- **references:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.
- **generated:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.
- **tickets:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.
- This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.
- **product:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.
- This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.
- **workflow:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.
- **design:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.
- **references:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.
- **exec-plans:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.
- **plans:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.
- This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.
- **tickets:** This matters because agents and maintainers depend on repo-owned docs to preserve behavior and intent.
- **tickets:** This matters because project health work keeps future delivery predictable.
- **m0:** This matters because project health work keeps future delivery predictable.
- **tickets:** This matters because project health work keeps future delivery predictable.
- This matters because project health work keeps future delivery predictable.
- **tickets:** This matters because project health work keeps future delivery predictable.
- **tickets:** This matters because project health work keeps future delivery predictable.
- **tickets:** This matters because project health work keeps future delivery predictable.
- **serve:** This matters because the project needs durable evidence that the behavior keeps working.

### What Changed
- **tools:** Changed mechanical ticket deduplication with ticket_create tool (AD-030) (0322c0c).
- **tools,scanner:** Changed wire git tools into manifests and add commit gates (AD-028) (1091028).
- **pipeline:** Changed chain dogfood tester after engineer completes a feature (10c845d).
- Changed dogfood E2E validation with Podman + decision recording system (147952e).
- **dashboard,scanner:** Changed dynamic pipeline chain + tsconfig path alias check (168a354).
- **m6,m7:** Changed scoring engine, progressive autonomy, evolution system, guardrails engine (MH-017 through MH-021) (1b9890e).
- **telemetry:** Changed triage self-improvement signals (28082ae).
- **tools:** Changed add registry, executor, and core tools (M1.2 / MH-002) (3083f9e).
- **scanner:** Changed add bootability checks for framework-specific validation (AD-026) (38c491f).
- **m5:** Changed job queue, scheduler, sandbox, safety, serve command (MH-013, MH-014, MH-015, MH-016) (45365ca).
- **serve:** Changed wire autonomous orchestrator with repo registry, trigger router, and executor (498d349).
- **agent:** Changed add conversation loop, parser, and tests (M1.3 / MH-003) (5394612).
- **scanner:** Changed add mars-harness upgrade command to sync target project manifests and prompts (5b53d26).
- **setup:** Changed plug-and-play model download from HuggingFace (5c91811).
- **skills:** Changed add agent skills across Cursor, AGENTS.md, and .harness/skills/ (73a0a82).
- **ui:** Changed cursor-quality agent output with role banner, tool trace, and handoff (7bf1294).
- **setup:** Changed auto-install llama-server and wire run command (7ddea6e).
- **llm:** Changed add OpenAI-compatible chat client and testutil helpers (7e50d1d).
- **init:** Changed provision full 11-role Mars pipeline by default (Tenet 1) (88de7a0).
- **serve:** Changed parallel pipeline tracks with sleep resilience (89b7895).
- **dashboard:** Changed wire dashboard into orchestrator with SSE events (8dce248).
- Changed implement two-level self-learning system with janitor agent (8f86add).
- **init:** Changed auto-run git init when .git is missing (8fc2260).
- **cli:** Changed auto-init .harness when manifest is missing (907d23a).
- **m9:** Changed embedded dashboard with htmx, Chart.js, SSE, emergency stop (MH-024) (9f7c083).
- **context:** Changed add context assembler with token budget (M1.4 / MH-004) (a07bfc9).
- **m8:** Changed setup wizard, init + scanner, doctor command (MH-022, MH-023, MH-028) (a26dccc).
- **init:** Changed scaffold full docs/ structure with Mars-quality role prompts (a6f7b26).
- **telemetry:** Changed add telemetry collector with error classification and auto-fix (ab150c5).
- **core:** Changed enforce strict trunk execution safety (b11daca).
- **tools:** Changed add background mode to shell_exec for long-running processes (b898608).
- **m3,m4:** Changed cLI framework, bundle reader, terminal UI, GitHub integration (MH-009, MH-010, MH-011, MH-012) (bb95350).
- **inference:** Changed add llama-server subprocess manager and role router (ce12a9d).
- **context:** Changed file filter, budget partitioning, .harnessignore support (M1.4.2, M1.4.4, MH-004) (d264010).
- **m10:** Changed role prompts, dogfood workflow, cross-compile release, user docs (MH-025, MH-026, MH-027) (dcac2af).
- **start:** Changed add `mars-harness start` for full e2e pipeline orchestration (deb1cd3).
- **dashboard:** Changed implement throughput page with chart, stats, and job table (df3b70d).
- **m2:** Changed hardware detection, model download, llama-server management, LLM router (MH-006, MH-007, MH-008) (e3436b9).
- **orchestrator:** Changed configurable agent triggers with chaining and custom cron (ec4db54).
- **agent:** Changed m1 closeout — 3+ tool-call test, trace error propagation, realistic 5-turn test (MH-003, MH-005) (ece2768).
- **inference:** Changed expose local performance tuning (ed24f83).
- **prompts:** Changed add git push after every commit in all role prompts (efce0b9).
- **serve:** Changed per-repo database isolation to prevent cross-project contamination (AD-029) (f098c2e).
- **init:** Changed mirror harness context glossary (f23663d).
- **trace:** Changed add JSONL recorder and SQLite store (M1.5 / MH-005) (f862f46).
- **tools:** Changed kill entire process group on shell_exec timeout (2ecc1ad).
- **inference:** Changed mitigate LLM timeout, context overflow, and connection refused failures (64ca767).
- **prompts:** Changed cold-start CEO/COO prompts for new projects (6eff4e9).
- **setup:** Changed pin local runtime artifacts (7e1a85e).
- **serve:** Changed block engineer handoff with active tickets (7fd00a8).
- **agent:** Changed always serialize message content field for llama.cpp compat (927cdb5).
- **agent:** Changed add context window guard to prevent token overflow (a045bd0).
- Changed resolve audit findings — broken links, naming inconsistencies, missing docs (M0) (cb17a6e).
- **init:** Changed preserve existing user content on --force re-init (ccacf88).
- **core:** Changed auto-tune inference and drain active tickets (e1fd6e0).
- **github:** Changed keep integrations trunk oriented (e45a90e).
- Changed correct module path, add gitkeeps, fix placeholders (M0) (ea287fc).
- **pipeline:** Changed restore COO → Engineer handoff for delivery kickoff (ebfaa56).
- **upgrade:** Changed preserve user configured agents (edaafea).
- **serve:** Changed auto-cleanup stale processes and corrupt DB on start/serve (f3af248).
- **dashboard:** Changed constrain chart canvas height to 280px (f9620aa).
- **references:** Changed carry mars agent-first references (009358b).
- **generated:** Changed define generated docs contract (1c0f043).
- **tickets:** Changed populate full backlog MH-001 through MH-028 (M0) (2c508cf).
- Changed add terminology definitions and dual-repo commit discipline (3759dc9).
- **product:** Changed refresh living product specs (4806d8f).
- Changed switch to trunk-based development, drop branch/PR requirement (584e4d7).
- **workflow:** Changed align generated bundles with strict trunk (69b608b).
- **design:** Changed record AD-031 inference resilience decisions (838e29c).
- **references:** Changed audit mars relevance for harness parity (92d7b8b).
- **exec-plans:** Changed add master execution plan with M0–M10 + MH-001–MH-028 coverage (9d13b8e).
- **plans:** Changed add mars supersession parity plan (b8f2a35).
- Changed add AD-021 through AD-025 for dogfood, decisions, and lean pipeline (bd6293b).
- **tickets:** Changed reconcile 19 ticket-vs-schedule contradictions (C1–C19) (d315dfc).
- **tickets:** Changed move MH-001 through MH-005 to done/ (M1 closeout) (00bbe6f).
- **m0:** Changed audit and fix M0 quality gate gaps (3419f69).
- **tickets:** Changed move MH-006 through MH-008 to done/ (M2 closeout) (431fb77).
- Changed initialize mars-harness repo (M0) (451a632).
- **tickets:** Changed move MH-009 through MH-012 to done/ (M3+M4 closeout) (88c182e).
- **tickets:** Changed move MH-017 through MH-021 to done/ (M6+M7 closeout) (8d48467).
- **tickets:** Changed move MH-013 through MH-016 to done/ (M5a+M5b closeout) (95dbee2).
- **serve:** Changed fix serve tests for new Config requirements, add skills loader tests (56e169a).

### Features
- **tools:** Mechanical ticket deduplication with ticket_create tool (AD-030) (0322c0c)
- **tools,scanner:** Wire git tools into manifests and add commit gates (AD-028) (1091028)
- **pipeline:** Chain dogfood tester after engineer completes a feature (10c845d)
- Dogfood E2E validation with Podman + decision recording system (147952e)
- **dashboard,scanner:** Dynamic pipeline chain + tsconfig path alias check (168a354)
- **m6,m7:** Scoring engine, progressive autonomy, evolution system, guardrails engine (MH-017 through MH-021) (1b9890e)
- **telemetry:** Triage self-improvement signals (28082ae)
- **tools:** Add registry, executor, and core tools (M1.2 / MH-002) (3083f9e)
- **scanner:** Add bootability checks for framework-specific validation (AD-026) (38c491f)
- **m5:** Job queue, scheduler, sandbox, safety, serve command (MH-013, MH-014, MH-015, MH-016) (45365ca)
- **serve:** Wire autonomous orchestrator with repo registry, trigger router, and executor (498d349)
- **agent:** Add conversation loop, parser, and tests (M1.3 / MH-003) (5394612)
- **scanner:** Add mars-harness upgrade command to sync target project manifests and prompts (5b53d26)
- **setup:** Plug-and-play model download from HuggingFace (5c91811)
- **skills:** Add agent skills across Cursor, AGENTS.md, and .harness/skills/ (73a0a82)
- **ui:** Cursor-quality agent output with role banner, tool trace, and handoff (7bf1294)
- **setup:** Auto-install llama-server and wire run command (7ddea6e)
- **llm:** Add OpenAI-compatible chat client and testutil helpers (7e50d1d)
- **init:** Provision full 11-role Mars pipeline by default (Tenet 1) (88de7a0)
- **serve:** Parallel pipeline tracks with sleep resilience (89b7895)
- **dashboard:** Wire dashboard into orchestrator with SSE events (8dce248)
- Implement two-level self-learning system with janitor agent (8f86add)
- **init:** Auto-run git init when .git is missing (8fc2260)
- **cli:** Auto-init .harness when manifest is missing (907d23a)
- **m9:** Embedded dashboard with htmx, Chart.js, SSE, emergency stop (MH-024) (9f7c083)
- **context:** Add context assembler with token budget (M1.4 / MH-004) (a07bfc9)
- **m8:** Setup wizard, init + scanner, doctor command (MH-022, MH-023, MH-028) (a26dccc)
- **init:** Scaffold full docs/ structure with Mars-quality role prompts (a6f7b26)
- **telemetry:** Add telemetry collector with error classification and auto-fix (ab150c5)
- **core:** Enforce strict trunk execution safety (b11daca)
- **tools:** Add background mode to shell_exec for long-running processes (b898608)
- **m3,m4:** CLI framework, bundle reader, terminal UI, GitHub integration (MH-009, MH-010, MH-011, MH-012) (bb95350)
- **inference:** Add llama-server subprocess manager and role router (ce12a9d)
- **context:** File filter, budget partitioning, .harnessignore support (M1.4.2, M1.4.4, MH-004) (d264010)
- **m10:** Role prompts, dogfood workflow, cross-compile release, user docs (MH-025, MH-026, MH-027) (dcac2af)
- **start:** Add `mars-harness start` for full e2e pipeline orchestration (deb1cd3)
- **dashboard:** Implement throughput page with chart, stats, and job table (df3b70d)
- **m2:** Hardware detection, model download, llama-server management, LLM router (MH-006, MH-007, MH-008) (e3436b9)
- **orchestrator:** Configurable agent triggers with chaining and custom cron (ec4db54)
- **agent:** M1 closeout — 3+ tool-call test, trace error propagation, realistic 5-turn test (MH-003, MH-005) (ece2768)
- **inference:** Expose local performance tuning (ed24f83)
- **prompts:** Add git push after every commit in all role prompts (efce0b9)
- **serve:** Per-repo database isolation to prevent cross-project contamination (AD-029) (f098c2e)
- **init:** Mirror harness context glossary (f23663d)
- **trace:** Add JSONL recorder and SQLite store (M1.5 / MH-005) (f862f46)

### Fixes
- **tools:** Kill entire process group on shell_exec timeout (2ecc1ad)
- **inference:** Mitigate LLM timeout, context overflow, and connection refused failures (64ca767)
- **prompts:** Cold-start CEO/COO prompts for new projects (6eff4e9)
- **setup:** Pin local runtime artifacts (7e1a85e)
- **serve:** Block engineer handoff with active tickets (7fd00a8)
- **agent:** Always serialize message content field for llama.cpp compat (927cdb5)
- **agent:** Add context window guard to prevent token overflow (a045bd0)
- Resolve audit findings — broken links, naming inconsistencies, missing docs (M0) (cb17a6e)
- **init:** Preserve existing user content on --force re-init (ccacf88)
- **core:** Auto-tune inference and drain active tickets (e1fd6e0)
- **github:** Keep integrations trunk oriented (e45a90e)
- Correct module path, add gitkeeps, fix placeholders (M0) (ea287fc)
- **pipeline:** Restore COO → Engineer handoff for delivery kickoff (ebfaa56)
- **upgrade:** Preserve user configured agents (edaafea)
- **serve:** Auto-cleanup stale processes and corrupt DB on start/serve (f3af248)
- **dashboard:** Constrain chart canvas height to 280px (f9620aa)

### Documentation
- **references:** Carry mars agent-first references (009358b)
- **generated:** Define generated docs contract (1c0f043)
- **tickets:** Populate full backlog MH-001 through MH-028 (M0) (2c508cf)
- Add terminology definitions and dual-repo commit discipline (3759dc9)
- **product:** Refresh living product specs (4806d8f)
- Switch to trunk-based development, drop branch/PR requirement (584e4d7)
- **workflow:** Align generated bundles with strict trunk (69b608b)
- **design:** Record AD-031 inference resilience decisions (838e29c)
- **references:** Audit mars relevance for harness parity (92d7b8b)
- **exec-plans:** Add master execution plan with M0–M10 + MH-001–MH-028 coverage (9d13b8e)
- **plans:** Add mars supersession parity plan (b8f2a35)
- Add AD-021 through AD-025 for dogfood, decisions, and lean pipeline (bd6293b)
- **tickets:** Reconcile 19 ticket-vs-schedule contradictions (C1–C19) (d315dfc)

### Maintenance
- **tickets:** Move MH-001 through MH-005 to done/ (M1 closeout) (00bbe6f)
- **m0:** Audit and fix M0 quality gate gaps (3419f69)
- **tickets:** Move MH-006 through MH-008 to done/ (M2 closeout) (431fb77)
- Initialize mars-harness repo (M0) (451a632)
- **tickets:** Move MH-009 through MH-012 to done/ (M3+M4 closeout) (88c182e)
- **tickets:** Move MH-017 through MH-021 to done/ (M6+M7 closeout) (8d48467)
- **tickets:** Move MH-013 through MH-016 to done/ (M5a+M5b closeout) (95dbee2)

### Tests
- **serve:** Fix serve tests for new Config requirements, add skills loader tests (56e169a)
