---
id: T-034
title: Surface degraded-inference anomalies mechanically via pace telemetry and a doctor RAM-footprint check
priority: medium
complexity: medium
work_type: enabler
bdd_scenarios: []
end_to_end_evidence: required
evidence_links: ["docs/validation/reports/2026-06-11-demo-11-pace-baseline.md", "docs/design-docs/local-inference.md"]
verified_by: "TBD"
owner: "TBD"
last_attempt: "TBD"
blocker: "none"
blocked_by: []
trace_id: "TBD"
next_action: "Decide which signal lands first (tokens/sec or per-role wall-time anomaly vs the dated baseline in the T-027 Convergence And Guardrails export, or the doctor model-footprint-vs-physical-RAM check) and implement it with tests."
kind: intervention-debt
source: intervention-to-automation: 2026-06-11 heavy-model RAM-pressure degradation was caught by human observation, not telemetry
created: 2026-06-12
depends_on: []
---

# T-034: Surface degraded-inference anomalies mechanically via pace telemetry and a doctor RAM-footprint check

## Context

The 2026-06-11 heavy-model demo-11 baseline ran with `Qwen3-Coder-30B-A3B-Instruct-Q8_0` weights (32.5 GB per server) maxing 64 GiB unified memory; inference degraded drastically (cto-weekly 12.1-minute wedge vs 205s for the same stage on the balanced model — same harness, target, and prompts). The degradation was only caught because a human operator noticed the machine was struggling; nothing in the harness surfaced it. The pace data captured under RAM pressure was confounded and had to be reclassified evidence-only (see the local-inference.md Discoveries entry and the AD-285 model-identity amendment).

Degraded inference must become a visible finding, not a human observation (intervention-to-automation).

## Requirements (either or both; pick the cheaper first)

1. **Telemetry anomaly row:** the T-027 Convergence And Guardrails / Factory Pace export (`scores export`, `docs/QUALITY_SCORE.md`) flags per-role tokens/sec or wall-time anomalies vs the dated pace baseline for the same model identity (e.g. avg wall > Nx baseline for the same role and archetype).
2. **Doctor RAM-footprint check:** `mars-harness doctor` (and optionally setup/start preflight) flags when the configured profile model footprint (sum of resident tier weights + KV/context overhead) approaches physical RAM/unified memory, with the remediation suggestion to drop to the balanced/speed profile.

## Affected Files

- internal/qualityscore and/or internal/scoring (anomaly rows)
- internal/hardware, internal/inference, cmd/mars-harness (doctor check)
- docs/design-docs/local-inference.md, docs/design-docs/self-reflective-telemetry.md (AD)

## Acceptance Criteria

- A run whose per-role pace degrades materially vs the same-model-identity baseline produces a visible anomaly flag in the export, with tests.
- doctor warns when the configured model footprint approaches physical RAM, with tests.
- Quality-export change class per AD-284: package tests plus the next scheduled baseline replay doubles as live validation (this ticket records that flag); the doctor check is model/provider-adjacent and takes one archetype replay on the affected path or a recorded blocker.
