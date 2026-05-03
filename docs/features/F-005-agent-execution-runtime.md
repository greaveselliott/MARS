# F-005: Agent Execution Runtime

- Feature ID: F-005
- Goals: G-001, G-003
- Status: partially-passing
- Owner: CTO

## Scenario Schedule

1. F-005-S001 - Context assembly builds bounded, role-scoped prompts from repo-owned harness artifacts.
2. F-005-S002 - The agent loop handles multi-turn tool calling, completion, error outcomes, and retryable inference failures.
3. F-005-S003 - Tools are allowlisted, validated, and executed through repo-root containment.
4. F-005-S004 - Execution traces persist enough turn and tool detail to make outcomes auditable.
5. F-005-S005 - Budgets, max turns, max tool calls, context pruning, and wall time stop runaway jobs.
6. F-005-S006 - `mars-harness run <role> --repo <path>` executes one role with terminal-result truth.
7. F-005-S007 - Tool creation and CLI-operation tools are first-class mirrored harness capabilities.

## Scenarios

### F-005-S001: Role-Scoped Context Assembly

Given a role, repo, ticket index, guardrail scope, knowledge routes, and context budget
When a job prompt is assembled
Then high-priority role and workflow context is included, irrelevant guardrails are filtered, and lower-priority context is truncated before exceeding budget

### F-005-S002: Multi-Turn Tool Loop

Given an LLM response includes tool calls and later a completion signal
When the agent loop runs
Then tool calls execute in order, tool results are returned to the model, and the job ends with a classified outcome

### F-005-S003: Tool Execution Containment

Given a role has an allowlist and trust level
When it asks to call a tool
Then unknown tools, disallowed tools, invalid JSON, observer mutations, path escapes, and secret writes fail closed

### F-005-S004: Auditable Trace

Given an agent job runs
When turns and tool calls occur
Then trace storage records enough context to diagnose what happened after the run

### F-005-S005: Runtime Limits

Given a model loops, exceeds budget, calls too many tools, or exceeds wall time
When the agent loop reaches the configured limit
Then the run stops with a non-success outcome that telemetry can classify

### F-005-S006: Manual Role Run

Given a user runs `mars-harness run <role> --repo <path>`
When manifest, context, tools, and inference endpoint are available
Then exactly one role run executes against the target repo and reports the terminal result

### F-005-S007: Mirrored Built-In Tools

Given a recurring deterministic harness operation exists
When it is implemented as a built-in tool
Then the tool has schema, tests, registry exposure, trust-policy review, glossary documentation, and target mirroring when applicable

## Out of Scope

- Parallel tool execution inside a single agent turn.
- Unbounded prompt stuffing.
- Treating chat transcript memory as the system of record.

## Descoped Scenarios

None.

## Evidence

- F-005-S001: `go test ./internal/context`
- F-005-S002: `go test ./internal/agent -run TestRun`
- F-005-S003: `go test ./internal/tools -run TestExecutor`
- F-005-S004: `go test ./internal/agent -run TestRun_persistsTraceToSQLite`
- F-005-S005: `go test ./internal/agent -run 'TestRun_(max|token|wall|circle|empty)'`
- F-005-S006: covered by command-level run behavior and planned broader E2E dogfood evidence
- F-005-S007: `go test ./internal/tools -run 'TestToolCreate|TestMarsHarnessCLI'`
