# Reference: UI Design Brain

**Source:** [github.com/carmahhawwari/ui-design-brain](https://github.com/carmahhawwari/ui-design-brain)
**Author:** carmahhawwari
**Type:** Cursor skill, open source
**Carry-over source:** `../mars/docs/references/ui-design-brain.md`
**Mars Harness relevance:** Reference for generated target projects and future UI-oriented skills.

## Summary

UI Design Brain is a skill that gives agents structured UI component knowledge: component best practices, layouts, aliases, and anti-patterns. It is relevant to Mars Harness because generated target projects will eventually need agent-legible UI guidance instead of generic "make it look nice" prompting.

## Concepts Worth Carrying Forward

### Component knowledge as skill data

The useful pattern is not the specific component list. It is the shape: short entrypoint, detailed component reference, best practices, common layouts, aliases, and anti-patterns.

Harness implication: when Mars Harness generates target-project skills or docs, UI guidance should be structured and discoverable, not embedded as vague prompt prose.

### Anti-patterns are first-class

Agents improve faster when rules say what to avoid as clearly as they say what to do.

Harness implication: generated docs, guardrails, and future target-project skills should include anti-patterns that can later become checks.

### Skill activation by intent

UI Design Brain activates when the task is UI design related. Mars Harness should use the same idea for role context: load the narrow guidance needed for the current job rather than broad generic guidance.

## Status For Mars Harness

This is not core runtime doctrine. It should influence generated frontend target guidance, role context routing, and future skill/bundle design.
