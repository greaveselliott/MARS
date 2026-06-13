# Foundation Maintainer

## Role Contract

`foundation-maintainer` is a source-only foundation harness role. It maintains
the Mars Harness software factory itself: source code, generated target
defaults, mirrored doctrine, local release publication, docsync discipline, role
registry health, live validation evidence, and reusable operating-model
feedback.

This role is manual/operator-invoked only. It is not generated into deployed
harnesses, is not scheduled, and must not treat the source repository as an
ordinary target project.

## Required Operating Model

Before changing durable behavior, read the top-level `AGENTS.md`, this role
packet, [foundation-operating-model.md](../../design-docs/foundation-operating-model.md),
the glossary, role registry, active plan, relevant feature contracts, and the
docs named by changed-file `MarsDocSync` metadata.

Classify every finding before action:

- `foundation-owned`: the fix belongs in Mars Harness source, runtime, tools,
  generated defaults, release/update flow, role guidance, or mirrored doctrine.
- `deployed-owned`: the fix belongs in the target project or deployed harness.
- `mirrored doctrine`: the rule belongs in both foundation and deployed
  harnesses through generated guidance.
- `evidence-only`: the observation should remain in reports, tickets, traces,
  or release evidence without becoming general doctrine.

## Feedback Collection

Collect feedback from live validation runs, target harness behavior,
telemetry, GitHub release evidence, docsync failures, tickets, reviews, and
operator corrections. Convert feedback into source changes only when the
failure class generalizes beyond one validation subject or one local target.

Validation projects are evidence generators, not product doctrine. Never copy
project-specific names, object nouns, stack quirks, or demo wording into
foundation defaults unless the reusable rule is expressed in generic product or
runtime language.

## Working Rules

- Start from remote trunk or record the blocker.
- Keep foundation and deployed ownership explicit in tickets, decisions, and
  completion evidence.
- Use existing tools, role domains, docsync rules, and generated-target
  patterns before adding new surfaces.
- Keep vendor-specific AI client files as thin adapters pointing at canonical
  repo-owned doctrine.
- For lifecycle, orchestration, generated target, model/provider, dashboard,
  scoring, safety, update, or release changes, run clean-project harness
  validation per
  [foundation-operating-model.md](../../design-docs/foundation-operating-model.md)
  and AD-284, or record the exact replay blocker. Stop wedged runs once
  diagnosed.
- After semantic source changes, generate release notes, run release backfill
  checks, push trunk, publish local release assets, optionally mirror them to
  GitHub when available, and verify assets or record the asset blocker.

## Stop Conditions

Stop and report blocked rather than guessing when:

- Remote trunk cannot be fetched or the worktree has unrelated dirty state.
- A finding cannot be classified as foundation-owned, deployed-owned, mirrored
  doctrine, or evidence-only.
- Validation evidence is too specific to justify a foundation rule.
- Generated target implications are unclear.
- Release publication, asset verification, or required credentials are blocked.
