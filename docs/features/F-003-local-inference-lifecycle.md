# F-003: Local Inference Lifecycle

- Feature ID: F-003
- Goals: G-003, G-004
- Status: partially-passing
- Owner: Release Manager

## Business Logic

This feature contract is the durable home for business logic in this area. Product rules, workflow branches, state transitions, validations, permissions, scoring or trust decisions, routing rules, release classification, and user-visible outcomes must be documented here before or alongside implementation. Do not rely on ticket text or code comments as the only description of behavior.

## Step-By-Step Behavior

The scenarios below are the step-by-step BDD contract for this feature. Each scenario should describe the starting state, the action or event, and the observable outcome. When implementation changes business logic, update these steps and their evidence before claiming the feature is complete.

## Scenario Schedule

1. F-003-S001 - Setup detects hardware and chooses a zero-config performance profile.
2. F-003-S002 - Model downloads are resumable, checksummed, cached, and rejected when corrupt.
3. F-003-S003 - llama.cpp runs as a supervised subprocess with actionable health and startup state.
4. F-003-S004 - Role inference routing honors manifest model tiers before role-name fallbacks.
5. F-003-S005 - Missing model errors name the tier, expected artifact, and remediation command.
6. F-003-S006 - Model evaluation supports benchmark-backed candidate comparison before default promotion.
7. F-003-S007 - Ollama can be used for explicit catalog/evaluation/swap workflows without changing zero-config defaults automatically.

## Scenarios

### F-003-S001: Hardware Profile Selection

Given a user runs setup on a supported machine
When hardware detection and config initialization complete
Then the harness selects a practical default model profile without requiring manual thread, quantization, or parallel-slot tuning

### F-003-S002: Verified Model Downloads

Given setup needs a model artifact
When a download is complete, cached, partial, or corrupt
Then the model store verifies SHA256, resumes partial files when possible, reuses valid cache files, and rejects checksum mismatches before serving

### F-003-S003: Supervised llama.cpp Server

Given an agent needs a local model endpoint
When the llama.cpp server is started or reused
Then argv, base URL, state transitions, health checks, and shutdown are managed by the harness instead of embedding llama.cpp in the Go binary
And multi-slot local servers preserve each slot's tier context window by scaling total server context with the requested parallel slot count

### F-003-S004: Manifest-Tier Routing

Given a role declares `model: fast`, `model: reasoning`, or `model: coding` in the manifest
When the inference router resolves a server for that role
Then the manifest tier is used before role-name fallback routing
And validation-only single-server mode can explicitly force all roles and manifest hints onto one configured tier for batch smoke evidence

### F-003-S005: Actionable Missing Model Errors

Given a role requests a tier whose local model is missing
When the router cannot resolve a fallback
Then the error names the missing tier or artifact and points the operator to setup or configuration repair

### F-003-S006: Benchmark-Backed Model Evaluation

Given a candidate model is being considered for Mars Harness defaults
When evaluation runs against an endpoint and model name
Then the command exercises mechanical harness-relevant probes and reports promotion criteria rather than accepting freshness claims alone

### F-003-S007: Explicit Ollama Provider

Given an operator wants to try an Ollama model
When the model is used as an ad-hoc candidate or explicit override
Then the harness keeps that state separate from default registry promotion, which still requires pinned artifacts and benchmark evidence

## Out of Scope

- Hosted inference as the default runtime.
- Embedding llama.cpp through CGO.
- Promoting models from leaderboard claims without local harness evidence.
- Treating every locally installed Ollama model as a zero-config default.

## Descoped Scenarios

None.

## Evidence

- F-003-S001: `go test ./internal/hardware/... ./internal/setup/...`
- F-003-S002: `go test ./internal/models -run TestDownload`
- F-003-S003: `go test ./internal/inference -run TestServer`
- F-003-S004: `go test ./internal/inference -run TestTierForRoleModel_usesManifestTierBeforeRoleFallback`
- F-003-S005: `go test ./internal/inference -run TestRouter_serverForRole`
- F-003-S006: `go test ./internal/models -run TestEvaluate`
- F-003-S007: planned evidence for the remaining Ollama catalog and swap workflow tracked by `MH-030`
