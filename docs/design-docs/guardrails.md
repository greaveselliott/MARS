# Guardrails Engine

**Status:** Draft  
**Date:** 2026-04-11  
**Author:** Mars Harness contributors

Mechanical checks on harness outputs and repo mutations: what is enforced, how overrides work, and how the engine stays maintainable as rules grow.

## Context

Guardrails prevent destructive or policy-violating changes while keeping **false positives** manageable. v1 prioritizes **fast, deterministic** checks over deep semantic analysis; richer analysis is explicitly deferred.

Rules are loaded from the repo’s `.harness/` tree so teams can customize policy without recompiling the binary. Invalid rule files should **fail closed** at job start with actionable parse errors.

## Key Design Decisions

### AD-012: Hard guardrails are syntactic in v1

**Hard** guardrails in the first release are limited to **syntactic** checks: regular expressions, path/file patterns, and **file existence** predicates. **AST-based** or language-aware analysis is **v2**, to avoid shipping a slow or incomplete analyzer that blocks legitimate edits.

“Hard” means a failing check blocks promotion or fails the job per [pipeline-engine.md](pipeline-engine.md) policy hooks.

### Open topics

- **Advisory vs hard tiers:** advisory rules surface warnings in traces and UI; hard rules fail the job or block merge paths per policy; same schema with a `severity` field is the likely shape.
- **YAML format** for rule definitions: schema versioning, validation at load time, clear error messages pointing to file and line.
- **Prompt injection:** treat untrusted repo content as hostile; rules and prompts must assume adversarial README/docs; never `eval` rule bodies as code.
- **Mechanical validation:** deterministic execution, no network dependency for core checks; optional advisory fetches are explicitly labeled.
- **Override mechanism:** explicit allowlists / break-glass with audit trail (who, when, why); time-bounded overrides preferred over permanent silence.
- **Staleness detection:** rules that reference deleted paths or obsolete globs should warn or fail lint of `.harness/` on `harness doctor` or pre-job validation.

### Relationship

- [agent-runtime.md](agent-runtime.md) invokes guardrails at defined checkpoints (post-tool, pre-commit simulation, pre-push).

### Performance budget

Rule evaluation should stay sub-second for typical repos on laptop hardware; pathological regexes may need timeouts or complexity caps in v2.

## Discoveries

_(None yet.)_
