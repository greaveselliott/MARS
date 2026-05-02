# Reference: Vercel Composition Patterns

**Source:** [github.com/vercel-labs/agent-skills/tree/main/skills/composition-patterns](https://github.com/vercel-labs/agent-skills/tree/main/skills/composition-patterns)
**Author:** Vercel Engineering
**Published:** January 2026
**Type:** Structured agent skill
**Carry-over source:** `../mars/docs/references/vercel-composition-patterns.md`
**Mars Harness relevance:** Reference for generated React target projects and future skill format.

## Summary

The Vercel Composition Patterns skill captures React component composition rules for agents. Mars used it to improve generated UI maintainability. Mars Harness should keep it as a reference for generated target repositories, especially where future bundles include React, Next.js, or component-library scaffolds.

## Concepts Worth Carrying Forward

- Prefer explicit component variants over boolean prop proliferation.
- Use compound components and shared context for complex UI primitives.
- Separate state ownership from visual composition.
- Prefer children-based composition unless data must be passed back.
- Treat composition rules as maintainability guardrails, not stylistic decoration.

## Harness Translation

Mars Harness should not make these React-specific rules part of the core agent runtime. Instead, they should inform:

- generated target `AGENTS.md` files for React projects,
- optional frontend guardrail bundles,
- future skill/rule packaging,
- target-specific doctor checks where the target stack is known.

The transferable lesson is that agents benefit from small, enforceable composition rules with examples and priority levels.
