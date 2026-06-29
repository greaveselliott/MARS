# Cognition Orchestration Patterns

**Status:** Draft
**Date:** 2026-06-29
**Owner:** foundation-maintainer
**Scope:** Source-first foundation architecture investigation. Generated target mirroring requires a follow-up decision.
**Sources:** AWS Prescriptive Guidance, "Agentic AI patterns and workflows on AWS" pages listed in the evidence ledger below.

## Primary Outcome Contract

| Field | Value |
| --- | --- |
| Primary Outcome | Exhaustively read the operator-shared AWS agentic AI pattern resources and map their relevance to MARS. |
| Primary Pass Gate | All shared pages are fetched from AWS, read from canonical markdown, accounted for in a page-by-page matrix, and synthesized into foundation-owned, mirrored-doctrine, and evidence-only implications. |
| Primary Status | `primary_passed` for the research pass. Implementation remains unstarted. |
| Current Primary Blocker | None for the audit. Runtime evolution is a separate future slice. |
| Next Primary Action | Decide whether to open a bounded implementation plan for workflow metadata, route rationale, and trace correlation. |
| Supporting Evidence | 30 of 30 AWS pages fetched as HTML and markdown on 2026-06-29, with no fetch errors; markdown corpus contained 12,373 parsed words after AWS footer removal. |

## Context

The operator asked for an orchestrator-pattern reading of the AWS Prescriptive
Guidance resources on agentic AI patterns, then asked for a stricter second
pass after the first synthesis identified a possible major MARS evolution.
This document is the durable system record for that second pass.

The core finding is that AWS's material does not imply MARS should adopt AWS
managed services. Instead, it gives MARS a sharper architecture language for
something the product already is becoming: a local cognition-orchestration
engine that coordinates roles, tools, memory, traces, validation, trust, and
release evidence through auditable workflow state.

## Evidence Ledger

The second pass fetched each shared AWS page from
`https://docs.aws.amazon.com/prescriptive-guidance/latest/agentic-ai-patterns/`
as HTML and as the page's linked markdown rendition. The markdown files were
read for this matrix because they avoid navigation noise. The hashes below are
the SHA-256 prefixes for the markdown text fetched during the audit; they are
not committed copies of AWS content.

| # | AWS page | Markdown words | Hash prefix |
| --- | --- | ---: | --- |
| 1 | [Introduction](https://docs.aws.amazon.com/prescriptive-guidance/latest/agentic-ai-patterns/introduction.html) | 332 | `dbfc5398d81a` |
| 2 | [Agent patterns](https://docs.aws.amazon.com/prescriptive-guidance/latest/agentic-ai-patterns/agent-patterns.html) | 230 | `0451985e42a9` |
| 3 | [Basic reasoning agents](https://docs.aws.amazon.com/prescriptive-guidance/latest/agentic-ai-patterns/basic-reasoning-agents.html) | 910 | `22cdb7ad303b` |
| 4 | [Tool-based agents for calling functions](https://docs.aws.amazon.com/prescriptive-guidance/latest/agentic-ai-patterns/tool-based-agents-for-calling-functions.html) | 449 | `e9510ebe3e9f` |
| 5 | [Tool-based agents for servers](https://docs.aws.amazon.com/prescriptive-guidance/latest/agentic-ai-patterns/tool-based-agents-for-servers.html) | 508 | `3826e2901bba` |
| 6 | [Computer-use agents](https://docs.aws.amazon.com/prescriptive-guidance/latest/agentic-ai-patterns/computer-use-agents.html) | 527 | `64fcc79a4db3` |
| 7 | [Coding agents](https://docs.aws.amazon.com/prescriptive-guidance/latest/agentic-ai-patterns/coding-agents.html) | 433 | `de1e3b7faa47` |
| 8 | [Speech and voice agents](https://docs.aws.amazon.com/prescriptive-guidance/latest/agentic-ai-patterns/speech-and-voice-agents.html) | 526 | `6b02e20b7797` |
| 9 | [Workflow orchestration agents](https://docs.aws.amazon.com/prescriptive-guidance/latest/agentic-ai-patterns/workflow-orchestration-agents.html) | 452 | `199d23d2aeb7` |
| 10 | [Memory-augmented agents](https://docs.aws.amazon.com/prescriptive-guidance/latest/agentic-ai-patterns/memory-augmented-agents.html) | 560 | `b9c605fd155d` |
| 11 | [Simulation and test-bed agents](https://docs.aws.amazon.com/prescriptive-guidance/latest/agentic-ai-patterns/simulation-and-test-bed-agents.html) | 584 | `655aa0645176` |
| 12 | [Observer and monitoring agents](https://docs.aws.amazon.com/prescriptive-guidance/latest/agentic-ai-patterns/observer-and-monitoring-agents.html) | 519 | `d72fe6554e1f` |
| 13 | [Multi-agent collaboration](https://docs.aws.amazon.com/prescriptive-guidance/latest/agentic-ai-patterns/multi-agent-collaboration.html) | 696 | `dc0cdcbb248c` |
| 14 | [LLM workflows](https://docs.aws.amazon.com/prescriptive-guidance/latest/agentic-ai-patterns/llm-workflows.html) | 177 | `d4ba523046c3` |
| 15 | [Overview of LLM-augmented cognition](https://docs.aws.amazon.com/prescriptive-guidance/latest/agentic-ai-patterns/overview-of-llm-augmented-cognition.html) | 150 | `4312f3898035` |
| 16 | [Workflow for prompt chaining](https://docs.aws.amazon.com/prescriptive-guidance/latest/agentic-ai-patterns/workflow-for-prompt-chaining.html) | 262 | `0f6c424e8580` |
| 17 | [Workflow for routing](https://docs.aws.amazon.com/prescriptive-guidance/latest/agentic-ai-patterns/workflow-for-routing.html) | 256 | `7079db0504be` |
| 18 | [Workflow for parallelization](https://docs.aws.amazon.com/prescriptive-guidance/latest/agentic-ai-patterns/workflow-for-parallelization.html) | 267 | `ad65088f4ff9` |
| 19 | [Workflow for orchestration](https://docs.aws.amazon.com/prescriptive-guidance/latest/agentic-ai-patterns/workflow-for-orchestration.html) | 278 | `996fdcb3d76f` |
| 20 | [Workflow for evaluators and reflect-refine loops](https://docs.aws.amazon.com/prescriptive-guidance/latest/agentic-ai-patterns/workflow-for-evaluators-and-reflect-refine-loops.html) | 274 | `27fd0c2cbedf` |
| 21 | [Conclusion: LLM workflows](https://docs.aws.amazon.com/prescriptive-guidance/latest/agentic-ai-patterns/conclusion-llm-workflows.html) | 216 | `9b262d6c2779` |
| 22 | [Agentic workflow patterns](https://docs.aws.amazon.com/prescriptive-guidance/latest/agentic-ai-patterns/agentic-workflow-patterns.html) | 124 | `ffcae0bbc411` |
| 23 | [From event-driven to cognition-augmented systems](https://docs.aws.amazon.com/prescriptive-guidance/latest/agentic-ai-patterns/from-event-driven-to-cognition-augmented-systems.html) | 418 | `346fdbf8891a` |
| 24 | [Prompt chaining saga patterns](https://docs.aws.amazon.com/prescriptive-guidance/latest/agentic-ai-patterns/prompt-chaining-saga-patterns.html) | 559 | `5a8f9f66da46` |
| 25 | [Routing dynamic dispatch patterns](https://docs.aws.amazon.com/prescriptive-guidance/latest/agentic-ai-patterns/routing-dynamic-dispatch-patterns.html) | 549 | `727aaaaaa571` |
| 26 | [Parallelization and scatter-gather patterns](https://docs.aws.amazon.com/prescriptive-guidance/latest/agentic-ai-patterns/parallelization-and-scatter-gather-patterns.html) | 633 | `c2adfc89bd55` |
| 27 | [Saga orchestration patterns](https://docs.aws.amazon.com/prescriptive-guidance/latest/agentic-ai-patterns/saga-orchestration-patterns.html) | 579 | `a1b187ef0a8d` |
| 28 | [Evaluator reflect-refine loop patterns](https://docs.aws.amazon.com/prescriptive-guidance/latest/agentic-ai-patterns/evaluator-reflect-refine-loop-patterns.html) | 666 | `a12f007d1895` |
| 29 | [Designing agentic workflows on AWS](https://docs.aws.amazon.com/prescriptive-guidance/latest/agentic-ai-patterns/designing-agentic-workflows-on-aws.html) | 115 | `27625e5db517` |
| 30 | [Conclusion: agentic workflow patterns](https://docs.aws.amazon.com/prescriptive-guidance/latest/agentic-ai-patterns/conclusion.html) | 124 | `11515e71d3c3` |

## Key Finding

AWS frames agentic systems as evented, stateful compositions of cognition,
tools, memory, routing, parallelism, orchestration, and evaluation. MARS
already has local equivalents for most of those pieces:

| AWS concept | MARS local equivalent |
| --- | --- |
| Amazon Bedrock agent reasoning | Local model router and llama.cpp/OpenAI-compatible inference path |
| AWS Lambda/ECS tool execution | Built-in tools, sandboxed shell, CLI commands, dependency sync, MCP tools |
| Step Functions orchestration | SQLite queue, worker pool, Orchestrator, dispatch mode, role registry |
| EventBridge/SQS routing | Triggers, surveys, job payloads, queue metadata, dispatch decisions |
| S3/DynamoDB/OpenSearch memory | Repo artifacts, SQLite, code graph context, traces, quality scores, docs |
| CloudWatch/X-Ray observability | MARS trace recorder, dashboard, scores, validation reports, release evidence |

The gap is not that MARS lacks agents. The gap is that workflow cognition is
not yet consistently typed, traced, aggregated, compensated, and rendered as a
first-class workflow graph.

## AD-307: MARS Uses Cognition-Orchestration Patterns As A Local Architecture Lens

MARS should adopt the pattern language from the AWS guidance as a local,
self-hosted architecture lens:

- `agent_pattern` describes the role shape: `reasoning`, `rag`,
  `tool-function`, `tool-server`, `computer-use`, `coding`, `voice`,
  `workflow-orchestrator`, `memory-augmented`, `simulation-testbed`,
  `observer`, or `multi-agent-collaboration`.
- `workflow_type` describes the job or handoff shape: `chain`, `route`,
  `parallel`, `orchestrate`, `evaluate`, `saga`, or `simulation-episode`.
- Route decisions should record rationale, confidence, fallback, trust
  constraints, source evidence, and selected role/model/tool.
- Parallel lanes should have explicit aggregators that align, validate,
  dedupe, and synthesize results before claiming completion.
- Evaluator loops should influence scoring and trust only when backed by
  executable or durable evidence, not evaluator prose alone.
- Saga behavior should name retry, compensation, idempotency, escalation, and
  stop conditions for mutating workflows.

This decision is source-first. Generated target mirroring should wait until a
bounded foundation implementation proves which terms are stable and useful.

## Page-By-Page MARS Relevance Matrix

| # | AWS page | MARS relevance | Classification | Candidate action |
| --- | --- | --- | --- | --- |
| 1 | Introduction | Establishes the full pattern set and the goals of autonomous but controllable agent systems. This aligns with MARS tenets for transparency, progressive autonomy, and context efficiency. | Evidence-only now; foundation-owned vocabulary later. | Use as the umbrella source for this design doc. |
| 2 | Agent patterns | Defines asynchronous, autonomous, purposeful agents using perception, reasoning, and action. MARS roles already fit this model, but role metadata could record the agent pattern explicitly. | Foundation-owned. | Extend role registry vocabulary after implementation design. |
| 3 | Basic reasoning agents and RAG | Distinguishes stateless reasoning from retrieval-grounded reasoning. MARS should reserve basic reasoning for small classification/summarization work and rely on repo docs/code graph for grounded operations. | Mirrored doctrine candidate. | Clarify when roles may answer from prompt versus when they must retrieve repo evidence. |
| 4 | Tool-based agents for calling functions | Maps directly to MARS built-in tools and CLI wrappers. The key AWS idea is schemas plus tool metadata in the reasoning loop. | Foundation-owned and mirrored tool doctrine. | Add richer tool metadata: side effects, trust tier, timeout, retry, schema, and remediation. |
| 5 | Tool-based agents for servers | Maps to MCP, sandboxed subprocesses, tool servers, and future external runtimes. It reinforces isolation and delegation for complex tool execution. | Foundation-owned. | Treat MCP/tool-server calls as governed external execution lanes with trace and trust policy. |
| 6 | Computer-use agents | Relevant for browser/desktop UI validation and simulated user flows, but risky for mutation. MARS should prefer APIs/CLI/tools and use computer-use only for bounded evidence. | Foundation-owned safety guidance. | Add future guidance: computer-use evidence is review/test support, not default mutation path. |
| 7 | Coding agents | Directly maps to Engineer and Pipeline Fixer. AWS's context extraction, reasoning, action, and test loop matches MARS code graph, BDD, tickets, tests, and docsync. | Foundation-owned and mirrored doctrine. | Keep coding agents ticket-backed and evidence-gated. |
| 8 | Speech and voice agents | Low near-term relevance. Could support operator control or accessibility later, but does not improve the core autonomous delivery loop. | Evidence-only. | Defer. Record only as optional future interface pattern. |
| 9 | Workflow orchestration agents | Strong direct match to MARS Orchestrator, queue, role registry, scoring, and dispatch. AWS highlights selection by metadata and prior success. | Foundation-owned. | Make role selection score-aware and route-rationale-visible. |
| 10 | Memory-augmented agents | Maps to repo artifacts, SQLite, traces, quality scores, code graph context, release history, and operator preferences. MARS must keep memory inspectable. | Foundation-owned and mirrored doctrine. | Separate memory into run state, repo facts, semantic retrieval, role history, and user/project preferences. |
| 11 | Simulation and test-bed agents | Directly reinforces clean validation targets, agent-smoke, matrix reports, and replay commands. | Foundation-owned. | Treat validation runs as simulation episodes with scenario, seed, evidence, score impact, and replay command. |
| 12 | Observer and monitoring agents | Maps to Dogfood, Janitor, quality export, telemetry triage, docsync, release monitoring, and security review. | Foundation-owned. | Keep observer roles mostly non-mutating and route findings by failure ownership. |
| 13 | Multi-agent collaboration | Useful for research, review, and planning, but AWS distinguishes it from centrally orchestrated workflows. MARS should prefer orchestrated collaboration for delivery. | Foundation-owned. | Use peer/debate patterns only inside bounded research or review lanes; keep final authority in Orchestrator and repo artifacts. |
| 14 | LLM workflows | States the important principle: raw model calls are insufficient; workflow structure creates reliability. This is already MARS doctrine. | Foundation-owned. | Name workflow primitives in jobs/traces. |
| 15 | LLM-augmented cognition | Summarizes prompting, retrieval, tools, and memory as the cognitive module. These map directly to MARS context assembly and tools. | Foundation-owned. | Model MARS job execution as a cognitive module embedded in deterministic controls. |
| 16 | Prompt chaining | Maps to plan -> BDD -> ticket -> implementation -> review -> release. Intermediate outputs should be auditable structured handoffs. | Mirrored doctrine candidate. | Add typed handoff schemas and chain stage metadata. |
| 17 | Routing | Maps to role routing, failure ownership, model routing, tool selection, and context routes. Misrouting is high-leverage risk. | Foundation-owned. | Persist route rationale, confidence, fallback, and matched rule. |
| 18 | Parallelization | Maps to scanner batches, docsync audits, validation matrix, multi-perspective review, and release checks. Aggregation is the critical missing step. | Foundation-owned. | Add explicit scatter-gather aggregators before completion claims. |
| 19 | Orchestration | Maps to foundation-maintainer, Orchestrator, and role-subagent operation. | Foundation-owned. | Render workflow graph/timeline in telemetry and dashboard. |
| 20 | Evaluators and reflect-refine loops | Maps to Reviewer, QA, Security, scoring, self-improvement, and clean reruns. Risk is self-confirming critique. | Foundation-owned. | Bind evaluator results to tests, BDD evidence, docsync, release checks, and validation reports. |
| 21 | LLM workflow conclusion | Confirms the primitives are composable, not mutually exclusive. MARS workflows are hybrid by nature. | Evidence-only now. | Use hybrid workflow labels rather than exclusive role categories. |
| 22 | Agentic workflow patterns | Frames the higher-level shift from static event logic to LLM-augmented context and action. | Foundation-owned. | Treat cognition as evented workflow state. |
| 23 | Event-driven to cognition-augmented systems | Maps event enrichment to semantic enrichment. MARS equivalent is context assembly plus code graph plus tool outputs. | Foundation-owned. | Trace context packs and enrichment sources for each job. |
| 24 | Prompt chaining saga patterns | Strongly relevant: each reasoning or role step can be atomic, recoverable, and metadata-enriched. | Foundation-owned. | Add saga semantics for retry, compensation, replan, rollback, escalation, and stop. |
| 25 | Routing dynamic dispatch patterns | Strongly relevant to dispatch mode. Semantic routing must not erase deterministic guardrails. | Foundation-owned. | Combine LLM routing with registry validation and deterministic safety gates. |
| 26 | Parallelization and scatter-gather patterns | Strongly relevant to validation matrices and role-assuming research lanes. | Foundation-owned. | Make correlation IDs and aggregation decisions first-class in traces. |
| 27 | Saga orchestration patterns | Directly maps to Orchestrator as central coordinator with distributed worker roles. | Foundation-owned. | Record orchestrator plans, branch state, timeouts, retries, and final synthesis as data. |
| 28 | Evaluator reflect-refine loop patterns | Reinforces feedback control loops for CI, policy, quality, and self-improvement. | Foundation-owned. | Define max iterations, objective gates, timeout, known-wedge stop, and human escalation. |
| 29 | Designing agentic workflows on AWS | Useful as a translation table, not as a platform mandate. | Evidence-only. | Maintain local MARS equivalents instead of adopting managed-service dependencies. |
| 30 | Agentic workflow conclusion | States the broad evolution: modular systems grounded in event patterns and augmented by cognition. | Evidence-only now. | Use as strategic rationale for future workflow graph and trace work. |

## Architecture Implications

### 1. Workflow Type Becomes First-Class

MARS already stores jobs, dispositions, traces, scores, and queue metadata, but
the workflow shape is implicit. A future slice should add a stable
`workflow_type` vocabulary to jobs, trigger payloads, trace summaries, and
dashboard views. This allows operators to distinguish a route decision from a
chain step, a scatter-gather branch, a validation simulation episode, or an
evaluator loop.

### 2. Routing Needs Explainability

Routing is currently split across manifest roles, dispatch rules, surveys, and
model/context routing. AWS's dynamic-dispatch pattern makes the missing
operator question clear: why this role, model, context pack, and tool path?
MARS should persist route rationale as structured data rather than prose.

Minimum fields for a future route decision:

| Field | Purpose |
| --- | --- |
| `input_signal` | The event, disposition, ticket, trace, or operator request being routed. |
| `matched_rules` | Registry, manifest, trust, ticket, or failure-ownership rules that applied. |
| `selected_role` | The manifest role selected after validation. |
| `selected_model_tier` | The model tier or provider chosen, without leaking secrets. |
| `selected_tools` | Tool allowlist or tool family needed for the task. |
| `confidence` | Numeric or categorical confidence in the route. |
| `fallback` | What happens if the selected route fails or is unavailable. |
| `classification` | Foundation-owned, deployed-owned, mirrored doctrine, evidence-only, or mixed/unclear. |

### 3. Scatter-Gather Needs Aggregators

Parallel work is only useful when synthesis is explicit. For MARS, scatter
gather should not mean "many jobs completed"; it should mean:

1. A coordinator creates independent branches with a correlation ID.
2. Branches write typed outputs and evidence.
3. An aggregator waits for required branches or timeout policy.
4. The aggregator dedupes, checks contradictions, records unsupported claims,
   and chooses the final route.
5. Completion evidence cites the aggregated result, not individual branch prose.

Likely first adopters are validation matrix reports, docsync audits, scanner
batches, release asset checks, and role-assuming research lanes.

### 4. Saga Semantics Should Be Real, Not Metaphorical

The AWS saga analogy is useful only if MARS names real compensation behavior.
For read-only reasoning, compensation can be replan or discard. For mutating
work, compensation is concrete: do not stage, revert a local patch, reopen a
ticket, record a blocker, stop dispatch, or require human approval. File, git,
release, target-repo, and generated-default mutations cannot be compensated by
the model merely changing its mind.

### 5. Evaluator Loops Must Stay Evidence-Bound

MARS already has QA, Security, Dogfood, scoring, and self-improvement loops.
Evaluator output should be a signal, not proof. Trust and score changes should
require objective evidence where available: tests, BDD scenarios, docsync,
release verification, clean validation replays, or explicit operator approval.

### 6. Memory Must Stay Inspectable

AWS treats memory as short-term state plus long-term structured and semantic
memory. MARS should keep these layers visible:

| Memory layer | MARS source of truth |
| --- | --- |
| Short-term run state | Job payload, active messages, tool results, trace summary |
| Long-term repo facts | Docs, tickets, feature contracts, release notes, code graph |
| Semantic retrieval | Code graph context, search/snippet/trace/impact tools |
| Role performance history | Scores, terminal outcomes, telemetry, quality score |
| Operator/project preferences | Repo-owned docs, config, manifest, local overrides |

Hidden chat memory should not become a source of truth.

## Recommended Implementation Roadmap

### Slice 1: Metadata And Observability

Add `agent_pattern` to role registry docs and generated role metadata, and add
`workflow_type` to job/trace summaries where it can be inferred without
changing dispatch behavior. Expose the fields in CLI/status/dashboard surfaces.

Validation: affected package tests, docs consistency, and no clean-project
replay unless runtime dispatch behavior changes.

### Slice 2: Route Rationale Records

Extend orchestration decisions to record structured route rationale, confidence,
fallback, classification, matched rules, and selected role/model/tool family.

Validation: orchestration unit tests plus a clean dispatch replay if routing
behavior changes.

### Slice 3: Simulation Episode Contract

Extend validation reports and agent-smoke outputs with a stable simulation
episode schema: seed, scenario, model identity, command, DB/log paths, expected
evidence, observed result, failure ownership, score impact, cleanup, and replay.

Validation: agent-smoke report tests and one fixture-only smoke run.

### Slice 4: Scatter-Gather Aggregation

Add an explicit aggregator concept for validation matrices, source research
lanes, docs audits, or release checks. Start with read-only/reporting lanes
before mutating delivery branches.

Validation: unit tests for aggregation policy and a bounded report-generation
exercise.

### Slice 5: Saga Compensation Policy

Define compensation actions for MARS workflow classes: read-only research,
ticket lifecycle moves, source edits, target edits, release notes, release
assets, generated target defaults, and external tool calls.

Validation: policy tests plus clean-project replay for any runtime behavior
change that changes dispatch or mutation handling.

## Assumption Confidence Matrix

| Assumption | Evidence | Confidence | Validation Required |
| --- | --- | ---: | --- |
| The shared AWS resources were fully covered. | 30 of 30 URLs fetched as HTML and markdown; all 30 read from markdown; evidence ledger includes word counts and hashes. | 1.0 | Re-fetch only if AWS updates the guide or the source list changes. |
| AWS's pattern language maps naturally to existing MARS primitives. | Direct correspondences across queue, roles, tools, traces, scoring, trust, docsync, and validation reports. | 0.95 | Confirm through code-level design before implementation. |
| `workflow_type` and `agent_pattern` are the least risky first slice. | They can begin as metadata and observability before changing routing behavior. | 0.85 | Inspect data model and generated target implications; add tests. |
| Route rationale should become structured data. | AWS dynamic dispatch and MARS dispatch-mode docs both identify routing as high leverage. | 0.9 | Implement in orchestration decision tests before live replay. |
| Saga compensation can be generalized across MARS mutation classes. | AWS saga pages provide analogy; MARS has concrete mutation surfaces. | 0.75 | Needs detailed policy design and source/target boundary review. |
| Generated targets should inherit the vocabulary eventually. | Several implications are reusable doctrine, but premature mirroring could expose unstable terms. | 0.7 | Wait for one foundation slice and decide mirrored vocabulary after operator review. |

## Risks And Cautions

- Do not adopt AWS service dependencies as runtime requirements. MARS must keep
  its single-binary, local-first, no-external-runtime-dependency constraints.
- Do not let dynamic reasoning replace BDD contracts. Agents reason inside
  documented product and operating rules; they do not become the rules.
- Do not equate evaluator approval with proof. Executable or durable evidence
  remains the acceptance layer.
- Do not let memory become hidden chat state. Repo artifacts and SQLite-backed
  evidence remain the system of record.
- Do not parallelize beyond local hardware. Scatter-gather must respect model,
  GPU, CPU, DB, and working-tree contention.
- Do not treat peer multi-agent emergence as the default delivery mode.
  Orchestrated collaboration is safer for auditable source and target changes.

## Classification Summary

| Finding | Classification | Reason |
| --- | --- | --- |
| AWS pattern audit itself | Evidence-only | It records external research and does not change runtime behavior. |
| Local equivalence table | Foundation-owned | It names MARS runtime and doctrine surfaces. |
| Tool metadata, route rationale, workflow type, and simulation episodes | Foundation-owned | These belong first in source runtime, docs, trace, and dashboard surfaces. |
| Retrieval discipline, coding-agent bounds, BDD evidence, and no hidden memory | Mirrored doctrine candidate | These are reusable operating rules for deployed harnesses once wording stabilizes. |
| Voice-agent interface pattern | Evidence-only | Low near-term relevance to autonomous delivery. |
| AWS service implementation table | Evidence-only | It is useful only as an analogy to local MARS components. |

## Conclusion

The stricter pass confirms that the AWS material is a significant but natural
evolution for MARS. It does not require a platform pivot. It suggests that the
next generation of MARS should make cognition orchestration explicit: typed
workflow state, structured route rationale, simulation episodes, scatter-gather
aggregation, evidence-bound evaluator loops, and real saga compensation.

The conservative path is to begin with metadata and observability. That lets
MARS expose the workflow graph it already approximates before changing dispatch
behavior or generated target doctrine.
