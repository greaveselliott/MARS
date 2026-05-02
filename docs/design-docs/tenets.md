# The Nine Tenets

**Status:** Accepted
**Date:** 2026-04-11
**Author:** Agent-assisted

These are the product's constitution. Every feature, architecture decision, and default is filtered through them. They are ordered by priority — when two conflict, the higher one wins.

---

## 1. Plug and Play

**"Zero to a working autonomous pipeline in one command with no debugging. And that ease extends to every operation for the lifetime of the product."**

This covers the full lifecycle, not just first boot:

- **First setup:** `mars-harness setup` auto-detects GPU, downloads pinned models, scaffolds the bundle, verifies the local loop, and starts serving. GitHub integration is optional and only marked healthy after credentials and webhooks are validated.
- **Adding a repo:** `mars-harness init` in a new repo scaffolds everything. Register it and start running on `main`.
- **Upgrading models:** `mars-harness models upgrade` detects newer weights, downloads them, swaps seamlessly.
- **Upgrading the harness:** `mars-harness upgrade` pulls the latest binary and migrates config.
- **Permission fallback:** GitHub is optional telemetry and integration infrastructure. Local-only operation remains complete for ticket, commit, push, scoring, and dashboard workflows.

If a user has to debug anything during normal operation, we failed.

---

## 2. Self-Improving System

**"The harness gets better over time from two signals: human interventions it should have handled, and its own failures. Both trigger root cause analysis and concrete evolution."**

Two input signals, one evolution system:

**Signal A — Human intervention:** The harness monitors repo and optional integration signals for human actions that overlap with what it should have done (manual follow-up commits, reverts, hand-edits of harness output, manual CI reruns). Each is recorded, classified, and feeds the evolution loop.

**Signal B — Own failures:** When a role's job scores below threshold (tenet 3), the Reviewer meta-role analyses the full execution trace, classifies the root cause, and proposes an evolution.

**Root cause classifications:**

- **Prompt gap:** The role prompt didn't cover this case. Evolution: add instructions or examples to `.harness/roles/<role>.md`.
- **Guardrail gap:** No rule caught this class of mistake. Evolution: propose a new guardrail in `.harness/guardrails/`.
- **Trigger gap:** The harness missed work or a trigger didn't fire. Evolution: adjust triggers or schedule in `manifest.yaml`.
- **Policy gap:** Trust level, permissions, or push policy blocked useful work. Evolution: adjust `.harness/policies/`.
- **Context gap:** The role didn't have enough information to succeed. Evolution: add a knowledge route in `.harness/knowledge-routes.yaml`.
- **Model limitation:** The model isn't capable enough for this task class regardless of prompt. Evolution: log as signal for model upgrade, no prompt change.

**The evolution commit:** At autonomous trust level, a bounded evolution may be committed directly to `main` under `.harness/` with: the failure or intervention that triggered it, the root cause classification, the specific change and why it prevents recurrence, and the role's current accuracy score with expected impact. Lower-trust roles record a proposed change for a human-triggered contributor run.

**Safety rails:**

- The Reviewer cannot modify its own prompt (prevents self-reinforcing loops).
- Maximum one evolution commit per role and scope per day (prevents prompt churn).
- Before/after tracking: post-evolution jobs are tagged, and if the score drops after an evolution, the Reviewer can propose a revert.
- If evolutions consistently worsen scores, the feature automatically disables for that role and flags for human review.

**North star metric:** Interventions trending toward zero, accuracy trending upward. When the intervention count hits zero sustained over 30 days, the harness is fully autonomous for that repo.

**Intervention classification nuance:** Not every human action is an intervention. Clear interventions (human edited the same files, reverted a harness commit, or fixed a failed check by hand) trigger evolution. Ambiguous actions are logged but do not automatically trigger evolution. Routine comments are ignored unless they imply missed work.

---

## 3. Accuracy and Value Scoring

**"Every role has a health score derived from real outcomes. The score measures both correctness and value delivered — doing nothing when there's work to do is a failure, not a success."**

**Per-job outcome scoring:**

| Outcome signal | Score impact |
|---|---|
| Commit produced and pushed to main | +1.0 (delivery signal) |
| Checks passed after harness commit | +1.0 |
| Checks failed after harness commit | 0.0 |
| CI fix resolved the failure | +1.0 |
| CI fix did not resolve | 0.0 |
| Guardrail blocked unsafe mutation | 0.0 for delivery, positive for containment |
| Revert of harness commit | -1.0 |
| Human follow-up commit touched same files | +0.5 partial or -0.5 if correcting an error |
| Human intervention on harness output | -0.5 penalty |
| Meaningful noop (correctly no action) | neutral |
| Noop when work was available | -0.5 (value failure) |

**Per-role rolling score** over the last 20 scored jobs, normalised to 0-100 scale.

**Score-driven alerts:**

- Below 70: warning, suggest reviewing recent failures and guardrails.
- Below 50: critical, suggest pausing the role and investigating.
- Rising after an evolution: note the improvement with attribution.

**Honest framing:** Scoring is inherently lagging and imperfect. Some outcomes take days to observe (a regression might appear later). Scores are a health signal, not a verdict. They drive progressive autonomy (tenet 8) and self-improvement (tenet 2), but a human should investigate underlying causes, not just react to numbers.

**Time horizon acknowledgement:** Weekly roles (CEO) need 20 weeks before the rolling score is meaningful. Low-frequency roles should use manual trust overrides (tenet 8) during the bootstrap period.

---

## 4. Customisable Guardrails

**"Users define how their application should be built. The harness enforces those rules mechanically, with clear distinction between advisory guidance and hard constraints."**

**Two enforcement tiers:**

- **Advisory guardrails** (prompt-injected): Appended to the agent's system prompt as constraints. Best-effort — the model should follow them but might not. Used for conventions, style preferences, architectural guidance.
- **Hard guardrails** (mechanically validated): Checked against agent output before commit via regex, file pattern matching, or file existence checks. A violation blocks the commit. Used for security rules, import boundaries, required patterns.

**V1 limitation:** Hard guardrails are limited to syntactic checks (regex, pattern matching, file existence). Semantic checks ("all error paths must have user-facing messages") are advisory only. AST-level validation is a v2 feature.

**Override mechanism:** A role can request a guardrail override by including a structured justification in its commit message. The override is logged and surfaced in the dashboard. Frequent overrides of the same guardrail signal that the guardrail may be wrong.

**Staleness detection:** If a guardrail hasn't triggered in 30 days, the dashboard flags it for review. Stale guardrails consume context budget (tenet 9) for no value.

**Evolution integration:** The self-improvement loop (tenet 2) can propose new guardrails from failure patterns, and can propose removing stale ones.

---

## 5. Roadmap from Init

**"When the harness initialises a repo, it deploys a work management system so the autonomous pipeline has input from day one. The structure is configurable, not imposed."**

**What `mars-harness init` scaffolds** (default layout, configurable in manifest):

- `docs/tickets/` with `backlog/`, `in-progress/`, `done/` and a README
- `docs/exec-plans/backlog/`, `active/`, `completed/`, and `superseded/`
- `AGENTS.md` describing the repo for the harness
- A starter backlog generated by scanning the repo (missing tests, TODOs, type safety gaps)

**Configurable structure:** Directory paths are settings in the manifest, not hardcoded. Teams with existing structures can point the harness at their layout.

**External ticket integration:** For teams already using GitHub Issues, Linear, or Jira, the harness can read tickets from those sources. The default is self-contained markdown tickets (zero dependency), but integration is a config option.

**Smart scanner:** The repo scanner skips well-covered areas. A mature repo with good test coverage won't get 50 "missing test" tickets. The scanner is additive, not noisy.

---

## 6. Blast Radius Containment

**"The harness must never be able to cause irreversible damage. Bounded trunk commits, rate limiting, and an emergency stop."**

**Hard system limits (not configurable lower, only higher):**

- Only push the configured trunk branch, normally `main`.
- Never force-push or rewrite shared history.
- No secrets in output. Scan all generated content for API key / token / password patterns before committing.

**Configurable limits (in manifest, with safe defaults):**

- Max changed files per job: default 20.
- Max lines changed per commit: default 500.
- Max commits per hour per repo: default 5.
- File deletion requires explicit allowlist per role. Default: no deletions.

**Revert capability:** Every harness commit is trace-linked and can be reverted with a generated command or proposed revert commit.

**Emergency stop:** `mars-harness stop --now` immediately halts all jobs, cancels the queue, and stops new mutating tool calls. Dashboard has a red stop button.

**Scope:** Containment is the harness's job before commit and push. After a commit lands, safety remains traceable through scoring, checks, revert detection, and emergency stop.

---

## 7. Execution Truth and Transparency

**"Every action the harness takes is auditable and attributable. All behavior is defined in git. Every output traces back to its specific inputs with a full reasoning chain."**

**Auditability (not reproducibility):** LLMs are non-deterministic. The harness does not promise reproducibility. It promises that for any output, you can trace exactly what produced it:

- **Bundle hash:** Which version of `.harness/` was active.
- **Event payload:** What triggered the job.
- **Repo SHA:** What state the repo was in.
- **Model checkpoint:** Which model weights were used (pinned by hash in `bundle.lock.json`).
- **Full execution trace:** The complete LLM conversation, tool calls, tool results, reasoning.

**Nothing outside the repo:** The harness's behavior is fully determined by the `.harness/` bundle committed in git. No dashboard config, no hidden database state, no learned behavior that isn't committed. Evolution changes land as bounded commits to the bundle in git, not as hidden internal state.

**Trace storage:** Full traces retained 30 days, summaries retained indefinitely. Configurable in manifest.

**Trace access:** Every harness commit is linked to a reasoning summary. The dashboard shows full traces. `mars-harness run` streams traces live in the terminal.

---

## 8. Progressive Autonomy

**"The harness earns trust through demonstrated accuracy. New roles start restricted and graduate to full autonomy. Trust is earned, automatically managed, and revocable."**

**Three trust levels:**

| Level | Capabilities | When |
|---|---|---|
| **Observer** | Read/report only. Cannot mutate the repo. | Default for new setup. |
| **Contributor** | Human-triggered or ticket-bound edit, test, commit, and push to `main`. | After trial, or score 50-80. |
| **Autonomous** | May self-schedule, chain jobs, edit, test, commit, push to `main`, and perform bounded evolution. | Score above threshold, sustained 20 jobs. |

**Configurable thresholds per role** in the manifest. A personal side project might set autonomous at 60. A fintech app might need 95.

**Cold start (trial mode):** New roles get N trial runs at Contributor level for human-triggered or ticket-bound trunk commits. After N trials, the accuracy score determines the trust level. Default N=5.

**Automatic demotion:** Score drops below threshold for 5 consecutive jobs → demoted one level.

**Manual override:** `mars-harness trust <role> --level <level>` overrides the score-based level. Expected path for low-frequency roles during bootstrap.

---

## 9. Context Efficiency

**"The harness assembles the minimum context each role needs, not the maximum it could fit. Retrieval over stuffing. Knowledge routing over context dumping."**

Local models have real costs per token of context: VRAM for KV cache, slower inference, lower throughput.

**Minimal base context per job:** Role prompt (~2K tokens) + in-scope guardrails only + compact trigger context + repo summary (directory tree, not file contents). Everything else retrieved on demand via tools.

**Tool-based retrieval:** The agent has `file_read`, `file_search`, `grep`. The role prompt tells it where to look (via knowledge routes), not what's there. The model reads 5 relevant files instead of having 50 pre-loaded.

**Context budget per role:** Configurable in manifest. The harness assembles base context within this budget.

**Knowledge routing** (`.harness/knowledge-routes.yaml`): Maps task types to relevant files, injected as "when you encounter X, read Y" in the role prompt. Same pattern as Mars's `knowledge-base.mdc`.

**Guardrail scoping:** Only guardrails with `scope: [current-role]` are injected. An Engineer job doesn't load Security guardrails.

## Discoveries

(Record discoveries during implementation here.)
