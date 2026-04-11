---
id: MH-009
title: "Implement mars-harness run: CLI, bundle reader, terminal trace, first demo"
priority: high
complexity: large
source: delivery-schedule M3
created: 2026-04-11
---

# MH-009: Implement `mars-harness run` — CLI, bundle reader, terminal trace, first demo

## Context

This is the first visible, demonstrable milestone. A user runs `mars-harness run pipeline-fixer --repo /path` and sees a role execute against a real repo with the full trace streaming in the terminal.

Reference: delivery schedule M3

## Requirements

### CLI framework (`cmd/mars-harness/main.go`)
- Replace placeholder with cobra (or equivalent) CLI framework
- `run` subcommand: `--role`, `--repo`, `--model-endpoint` (optional override), `--trace` (verbosity)
- `version` subcommand

### Bundle reader (`internal/bundle/reader.go`)
- Read `.harness/` from a local path
- Parse `manifest.yaml`, role prompts (`.md`), guardrails (`.yaml`), knowledge routes
- Compute bundle hash (SHA256 over all bundle files, sorted)
- Validate: required fields present, role files exist for declared roles

### Terminal trace output (`internal/ui/trace.go`)
- Render agent execution trace in real-time to terminal
- Colour-coded: model responses, tool calls, tool results, errors
- Token counter per turn and cumulative
- Truncate long tool results in display (full in trace record)

### Sample bundle
- `examples/sample-bundle/.harness/` with a Pipeline Fixer role
- Includes a deliberate TypeScript type error for the agent to diagnose and fix
- Works as a self-contained demo

### End-to-end wiring
- CLI → bundle reader → context assembly → agent loop → LLM router → llama.cpp → tools → trace output

## Acceptance Criteria

### Functional
- [ ] `mars-harness run pipeline-fixer --repo /path` executes and produces visible output
- [ ] Trace streams to terminal in real-time with colour coding
- [ ] Tool calls execute against real files in the repo
- [ ] Budget limit terminates the run cleanly with clear message
- [ ] Sample bundle works: agent diagnoses the type error and proposes a fix

### Edge cases
- [ ] Missing `.harness/` directory → actionable error ("Run mars-harness init to create one")
- [ ] Invalid manifest YAML → actionable error with line number
- [ ] Role not found in bundle → actionable error listing available roles
- [ ] Repo path doesn't exist → actionable error

### Non-goals
- [ ] GitHub operations (stubs only, real in M4)
- [ ] Webhook receiving (M4)
- [ ] Persistent job queue (M5)

### Observability
- [ ] Demo recorded: terminal session showing mars-harness run against a repo

## Notes

This is the "wow" moment. If this doesn't look impressive, nothing else matters. Prioritise the terminal trace UX.
