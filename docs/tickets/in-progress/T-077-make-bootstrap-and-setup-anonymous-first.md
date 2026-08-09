---
id: T-077
title: Make bootstrap and setup anonymous-first
priority: high
complexity: large
work_type: enabler
bdd_scenarios: ["F-017-S003"]
end_to_end_evidence: required
evidence_links: ["docs/exec-plans/active/current-operating-plan.md", "docs/features/F-017-open-source-publication.md#f-017-s003-anonymous-immutable-verifiable-release-lifecycle", "docs/features/F-009-release-update-lifecycle.md", "docs/tickets/done/T-066-migrate-release-consumers-to-fail-closed-signed-archives.md"]
verified_by: "pending QA, Security, Dogfood, Release Manager, and Orchestrator"
owner: "foundation-maintainer"
last_attempt: "TBD"
blocker: "none"
blocked_by: []
trace_id: "launch-anonymous-bootstrap:2026-08-09"
next_action: "Checkpoint A: make official setup and auth readiness anonymous-first, add clear-local, and keep GitHub integration explicitly optional."
dedupe_key: "open-source:anonymous-bootstrap-setup"
metadata:
  classification: "foundation-owned-and-mirrored-doctrine"
  mutation_authority: "repository-source-tests-docs-only"
  primary_status: "primary_blocked"
  publication_authority: "denied"
  supports: "F-017-S003"
source: MARS Launch-Complete Open-Source Delivery Plan — T-077
created: 2026-08-09
depends_on: [T-058]
---

# T-077: Make bootstrap and setup anonymous-first

## Context

T-066 completed the fail-closed signed archive consumer and deliberately removed the circular shell binary bootstrap. The current installer therefore stops at source-checkout instructions; setup still requires private-release authentication unless the operator knows to skip it; `auth github check` reports only private-token readiness; no command removes only MARS's stored fallback token; and setup's `--download`/`--yes` flags do not yet create a truthful license-aware download boundary. F-017-S003 requires an anonymous-first path into the already-reviewed signed updater while the repository remains private until the later launch gates.

## Scope And Authority

Keep the repository private, retain `VERSION=0.68.49` and source fallback `0.69.0-dev`, and make no tag, GitHub Release, signature, upload, settings, visibility, or announcement mutation. Reuse the standard Go module proxy/SumDB trust path and the existing signed updater; do not add a second signature verifier, curl-pipe binary downloader, custom cryptography, installer framework, VM lab, or rolling audit. Unsupported automatic Linux llama.cpp acquisition remains disabled under the T-073 disposition. Real official `v0.69.0`/`v0.69.1` publication and logged-out public lifecycle proof remain T-080/T-081.

## Checkpoint A — Anonymous Setup And Auth Semantics

- Make ordinary `mars setup` independent of GitHub credentials. GitHub App/status integration remains an explicit `--github` opt-in, and private-fork/custom-repository authentication remains optional rather than a prerequisite for official public source or releases. Preserve compatible skip flags without retaining a hidden private-auth step.
- Make `mars auth github check` test the exact official metadata endpoint anonymously first, then report one fixed `anonymous`, `authenticated`, or `unavailable` class. Optional credentials may be resolved and retried only after an exact 401, 403, or 404 from the no-redirect `api.github.com` official metadata request, and only to that same origin/path. Transport/TLS failure, redirect, unexpected status, or a custom/non-GitHub origin must never trigger credential resolution or receive Authorization. No token, credential-derived text, response body, or private path may appear in text/JSON/errors; a representative regression must prove the first request is unauthenticated and no token crosses origins.
- Add `mars auth github clear-local`. It removes only `github_token` from the selected MARS config, preserves every other field and owner-only mode, is idempotent, and never alters GH_TOKEN, GITHUB_TOKEN, GitHub CLI/keychain state, GitHub App credentials, repository files, or remote state.
- Preserve the existing signed release consumer's anonymous-first, redirect-bounded, credential-scoped behavior; do not redesign acquisition. Commit and push this independently green checkpoint.

## Checkpoint B — License-Aware Third-Party Downloads

- Before any llama.cpp archive or model download request, resolve the exact planned artifacts and show a deterministic bounded plan containing artifact identity, byte size, license ID/URL, and applicable terms/notice URLs. Reject incomplete provenance before the download step mutates storage or makes a download request.
- Require an explicit download choice and acknowledgement. Interactive use may confirm the displayed plan; non-interactive/JSON use must provide the documented `--download --yes` form or fail with that remediation. `--skip-download`, `--inference defer`, and configured cloud routing perform no third-party download and require no acceptance. Do not persist a fabricated legal attestation.
- Keep current checksum, size, platform, and extraction controls. Do not enable the recorded-but-disabled Linux llama.cpp archives in this ticket; Linux clean-operation evidence may use deferred inference or an independently installed llama-server and must say so. Commit and push this independently green checkpoint.

## Checkpoint C — Exact-Version Go/SumDB Bootstrap Into The Signed Updater

- Replace the fail-closed placeholder with a small Bash bootstrap that accepts only one exact stable semantic tag, requires Go 1.25.12 or newer, and builds the canonical `github.com/greaveselliott/mars/cmd/mars@<exact-tag>` through an enabled public Go proxy and SumDB with canonical-module private/no-sum bypasses disabled. Floating refs, pseudo-versions, replacements, direct/off proxy mode, disabled SumDB, and malformed versions fail closed.
- Build only into an owner-controlled temporary staging directory, verify the staged binary's canonical command/module path and exact module version with standard Go build metadata, then invoke that staged MARS binary's existing signed updater with the same explicit version and the selected final install directory. The signed updater remains the sole archive/signature verifier and durable replacement authority; failure preserves any prior final binary and cleans staging. Successful packaged operation does not require Go.
- Bootstrap/setup must not require or print GitHub credentials. Keep output bounded and free of tokens, response bodies, temporary paths, and module-cache contents. Use standard Bash/Go tools and focused command stubs or a bounded local module-proxy fixture; do not create an installer subsystem. Commit and push this independently green checkpoint.

## Interfaces And DocSync

Expected implementation surfaces are `internal/githubauth`, config-owned local-token clearing, setup command/steps, `scripts/install.sh`, directly owning self-update/setup tests, MarsCLI and generated-target command guidance, README/AGENTS/quickstart/configuration/release/local-inference docs, F-009, F-017, and DocSync metadata only where behavior changes. Do not widen into producer admission, signing workflows, GitHub administration, community governance, legal disposition, or automatic Linux llama.cpp enablement.

## Validation

For each semantic checkpoint run affected-package normal tests, focused hostile/race tests, package vet, formatting, DocSync/docs consistency, and diff checks. At closure run four CGO-disabled Darwin/Linux AMD64/ARM64 builds once and one installed clean-HOME macOS lane plus one Linux lane without GH_TOKEN, GITHUB_TOKEN, GitHub CLI auth, or local fallback. The private bootstrap fixture must prove exact-version/SumDB command composition and prior-binary preservation; actual official-tag anonymous download/update/rollback remains a T-080/T-081 gate and must not be claimed here.

## Acceptance

- Setup and official-readiness checks work without GitHub credentials; optional authenticated fallback is classified without disclosure; clear-local removes only the MARS config fallback.
- No third-party binary/model download begins before the exact provenance plan and explicit acknowledgement; skip/defer paths remain non-downloading.
- The exact-version Go/SumDB bootstrap reaches only the canonical module, stages separately, hands off to the existing signed updater, and preserves the prior installation on every rejected/failing lane.
- Clean macOS and Linux source/setup lanes run without GitHub credentials, with the Linux inference limitation stated truthfully.
- T-077 may complete as private implementation evidence, but F-017-S003 remains incomplete pending T-078, T-080, and the logged-out T-081 lifecycle. Repository visibility stays private and Primary Status stays `primary_blocked`.

## Stop Conditions

Stop on any floating or mutable bootstrap selector, SumDB/private-module bypass for the canonical module, unsigned archive install, duplicated signature logic, credential requirement or disclosure on the official path, unacknowledged third-party download, enabled unsupported Linux llama.cpp acquisition, overwritten prior binary after bootstrap failure, public tag/Release/signing/settings/visibility mutation, or a claim that a not-yet-published official lifecycle passed.
