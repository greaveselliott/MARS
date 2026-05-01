# Relationship to Mars

**Status:** Accepted
**Date:** 2026-04-11

## Independence

Mars does not depend on Mars Harness. Mars Harness does not depend on Mars. They are separate codebases with separate repos.

## Connection

Mars's 11 automation prompts and operational experience are the seed content:

- **Automation prompts** (`docs/automations/prompts/`): 11 role prompts that become the starter `.harness/roles/` content, adapted for local model capabilities and the harness tool surface.
- **Automation team design doc** (`docs/design-docs/automation-team.md`): The founding principle (humans out of delivery loop), canonical decision registry, intervention debt tracking, and role design directly inform tenets 2 and 6.
- **Pipeline learnings** (`docs/exec-plans/pipeline-learnings.md`): CI failure patterns and fix recipes inform the Pipeline Fixer role's prompt and the knowledge routing system.
- **BOTS.md** (`docs/automations/BOTS.md`): Model-per-role assignment and cost awareness inform the hardware profiles and model routing in the harness manifest.
- **Knowledge base** (`.cursor/rules/knowledge-base.mdc`): The routing pattern ("when working on X, read Y") directly becomes `.harness/knowledge-routes.yaml`.
- **Deep research report** (`docs/future-state/deep-research-report.md`): Architecture A's optional GitHub integration, job protocol, webhook handling, and threat model provide the engineering foundation.

## Migration path

Mars continues on Cursor Cloud Automations. When Mars Harness reaches parity, Mars migrates by:

1. Adding `.harness/` to the Mars monorepo with all 11 roles ported
2. Enabling optional GitHub integration for Mars only after credentials, checks, and webhooks validate
3. Running `mars-harness serve` on a GPU machine
4. Validating each role produces equivalent output to Cursor Automations
5. Turning off Cursor webhooks in `.github/workflows/automations.yml`

## Parity definition

"Parity" means the pipeline flow operates equivalently — not that outputs are identical. Specifically:

- Same pipeline flow: CEO -> COO -> Engineer -> QA/Security -> CTO -> Release Manager
- Same trigger model: cron schedules + webhook events
- Equivalent or better accuracy scores per role (measured after 20+ jobs)
- Acknowledged quality delta on code-generating roles (Engineer, Pipeline Fixer) due to local model ceiling vs Opus 4.6

True output quality parity for code generation may require local models exceeding 80% SWE-bench Verified. As of April 2026, Qwen3-Coder-Next achieves 70.6%. Cloud fallback per role is the interim solution.
