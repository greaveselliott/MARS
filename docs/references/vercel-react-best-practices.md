# Reference: Vercel React Best Practices

**Source:** [github.com/vercel-labs/agent-skills/tree/main/skills/react-best-practices](https://github.com/vercel-labs/agent-skills/tree/main/skills/react-best-practices)
**Author:** Shu Ding and Vercel Engineering
**Published:** January 2026
**Type:** Structured agent skill
**Carry-over source:** `../mars/docs/references/vercel-react-best-practices.md`
**MARS relevance:** Reference for generated React/Next.js target projects and rule packaging.

## Summary

The Vercel React Best Practices skill is a structured ruleset for React and Next.js performance work. Mars adopted its rule format, impact levels, examples, and generated-output model. MARS should carry it as a reference for target-project bundle design and agent-legible rule authoring.

## Concepts Worth Carrying Forward

### Rules as source

Individual rule files can be compiled into agent-facing outputs. This is relevant to MARS because generated `.harness` bundles, role prompts, and target-project instructions need drift-resistant source material.

### Impact levels

Rules with severity or impact labels help agents choose what matters first.

Harness implication: guardrails, doctor checks, and role instructions should distinguish hard blockers from guidance.

### Examples plus remediation

Good agent rules include the reason, the incorrect pattern, the corrected pattern, and a concrete remediation path.

Harness implication: policy and doctor errors should be written as instructions the next agent can act on.

### Testable skills

The reference extracts test cases from rules. MARS should follow the same spirit: prompts, generated bundles, and guardrails should be validated by golden tests or matrix tests where practical.

## Status For MARS

This is not a default Harness runtime dependency. It is a reference for:

- generated target-project frontend guidance,
- future rule/skill packaging,
- doc consistency tests,
- target-specific frontend guardrails.
