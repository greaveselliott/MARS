---
id: T-077
title: Make bootstrap and setup anonymous-first
priority: high
complexity: large
work_type: enabler
bdd_scenarios: ["F-017-S003"]
end_to_end_evidence: required
evidence_links: ["commit:10b62f7d59620022b2e1030c5f33856d0c16e70f", "commit:04d6ba6844126dc84eb6bedc13c78bd31f8d371d", "commit:85c689c70ef801a2747acabf537739c9ebad3c12", "commit:56b8de336cf4d1439944cc7eb8ea0f5ad4043f2b", "docs/validation/reports/2026-08-24-t077-bootstrap-setup-closure.md", "docs/features/F-017-open-source-publication.md#f-017-s003-anonymous-immutable-verifiable-release-lifecycle", "docs/features/F-009-release-update-lifecycle.md"]
verified_by: "Engineer and Release Manager implementation GO; independent QA and Security closure GO; Orchestrator executed and verified the native Dogfood matrix on 2026-08-24"
owner: "foundation-maintainer"
last_attempt: "2026-08-24: four exact Go 1.26.5 builds and independently reviewed clean macOS/Linux source/setup lanes passed at 56b8de3; temporary evidence roots were removed and verified absent"
blocker: "none"
blocked_by: []
trace_id: "launch-anonymous-bootstrap:2026-08-09"
next_action: "Complete; create T-078 through ticket_create and begin the production-producer, signing, and legacy-hosted-object sanitation gate without tags or destructive hosted mutation absent exact authority."
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

T-066 completed the fail-closed signed archive consumer and deliberately removed the circular shell binary bootstrap. At ticket start, the installer stopped at source-checkout instructions; setup still required private-release authentication unless the operator knew to skip it; `auth github check` reported only private-token readiness; no command removed only MARS's stored fallback token; and setup's `--download`/`--yes` flags did not yet create a truthful license-aware download boundary. Checkpoints A and B closed the setup/auth/local-fallback and download-acknowledgement gaps. Checkpoint C implements the exact-version bootstrap into the signed updater and passes the required native build plus clean macOS/Linux source/setup evidence. T-077 is complete. F-017-S003 additionally requires the later production and public lifecycle gates while the repository remains private until cutover.

## Scope And Authority

Keep the repository private, retain `VERSION=0.68.49` and source fallback `0.69.0-dev`, and make no tag, GitHub Release, signature, upload, settings, visibility, or announcement mutation. Reuse the standard Go module proxy/SumDB trust path and the existing signed updater; do not add a second signature verifier, curl-pipe binary downloader, custom cryptography, installer framework, VM lab, or rolling audit. Unsupported automatic Linux llama.cpp acquisition remains disabled under the T-073 disposition. Real official `v0.69.0`/`v0.69.1` publication and logged-out public lifecycle proof remain T-080/T-081.

## Checkpoint A — Anonymous Setup And Auth Semantics — Complete

- Make ordinary `mars setup` independent of GitHub credentials. GitHub App/status integration remains an explicit `--github` opt-in, and private-fork/custom-repository authentication remains optional rather than a prerequisite for official public source or releases. Preserve compatible skip flags without retaining a hidden private-auth step.
- Make `mars auth github check` test the exact official metadata endpoint anonymously first, then report one fixed `anonymous`, `authenticated`, or `unavailable` class. Optional credentials may be resolved and retried only after an exact 401, 403, or 404 from the no-redirect `api.github.com` official metadata request, and only to that same origin/path. Transport/TLS failure, redirect, unexpected status, or a custom/non-GitHub origin must never trigger credential resolution or receive Authorization. No token, credential-derived text, response body, or private path may appear in text/JSON/errors; a representative regression must prove the first request is unauthenticated and no token crosses origins.
- Add `mars auth github clear-local`. It removes only `github_token` from the selected MARS config, preserves every other field and owner-only mode, is idempotent, and never alters GH_TOKEN, GITHUB_TOKEN, GitHub CLI/keychain state, GitHub App credentials, repository files, or remote state.
- Preserve the existing signed release consumer's anonymous-first, redirect-bounded, credential-scoped behavior; do not redesign acquisition. Commit and push this independently green checkpoint.

Exact pushed commit `10b62f7d59620022b2e1030c5f33856d0c16e70f` completes Checkpoint A in a bounded 33-path packet. Ordinary `mars setup` no longer performs an auth step. `mars auth github check` makes the exact no-redirect official metadata request anonymously first, and only an exact `401`, `403`, or `404` may resolve optional credentials for one retry to the same origin and path. `mars auth github clear-local` removes only the stored config `github_token` while preserving unrelated YAML and owner-only mode. Doctor retains custom `ConfigPath` compatibility when it performs the same release-access check. Transport, redirect, unexpected-status, custom-origin, environment/legacy-contamination, idempotence, setup, doctor, tool, CLI, and generated-guidance regressions pass without changing the signed downloader.

Affected-package normal tests, focused race tests, affected-package vet, `go test ./internal/docsync ./internal/docsconsistency`, `go run ./cmd/mars docsync audit --repo .` (`364` files, `0` findings), formatting, and `git diff --check` passed. QA, Security, and Orchestrator returned GO. This is Checkpoint A evidence only and did not complete the then-pending later checkpoints. The repository remains private at `VERSION=0.68.49` with Primary Status `primary_blocked`; T-073 legal/rights disposition and the two installed-App findings remain launch no-gos, and no Release, settings, visibility, signing, publication, or announcement authority changed.

## Checkpoint B — License-Aware Third-Party Downloads — Complete

- Before any llama.cpp archive or model download request, resolve the exact planned artifacts and show a deterministic bounded plan containing artifact identity, byte size, license ID/URL, and applicable terms/notice URLs. Reject incomplete provenance before the download step mutates storage or makes a download request.
- Require an explicit download choice and acknowledgement. Interactive use may confirm the displayed plan; non-interactive/JSON use must provide the documented `--download --yes` form or fail with that remediation. `--skip-download`, `--inference defer`, and configured cloud routing perform no third-party download and require no acceptance. Do not persist a fabricated legal attestation.
- Keep current checksum, size, platform, and extraction controls. Do not enable the recorded-but-disabled Linux llama.cpp archives in this ticket; Linux clean-operation evidence may use deferred inference or an independently installed llama-server and must say so. Commit and push this independently green checkpoint.

Exact pushed commit `04d6ba6844126dc84eb6bedc13c78bd31f8d371d` completes Checkpoint B in a bounded 21-file packet. Setup stable-sorts one unique pending-artifact plan, fixes the concrete local bundle, and shows each immutable identity, exact byte size, license ID/URL, and applicable terms/notice URLs, including the recorded quantization-tool license. Interactive setup displays the plan and confirms once. Non-TTY and JSON use require exact `--download --yes`; JSON emits a complete preflight event on stderr before requests and includes the same plan in final stdout. The exact acknowledged plan is compared again, the concrete bundle is forced through execution, and each downloader admits only listed identities. Decline, missing acknowledgement, incomplete provenance, a changed plan, or failed complete-plan display causes zero download requests and zero download-artifact writes. Skip, test, deferred, and cloud paths download nothing and need no acknowledgement; acceptance is not persisted, and automatic Linux llama.cpp acquisition remains disabled.

Exact affected-package normal and race tests for `internal/setup`, `cmd/mars`, `internal/tools`, and `internal/scanner`, affected-package vet, `go test ./internal/docsconsistency ./internal/docsync -count=1`, `go run ./cmd/mars docsync audit --repo .` (`366` files, `0` findings), formatting, and `git diff --check` passed. QA, Security, and Orchestrator returned GO. Checkpoint B was complete at this boundary; Checkpoint C implementation evidence is recorded below. T-077 and F-017-S003 remain incomplete. The repository remains private at `VERSION=0.68.49` with Primary Status `primary_blocked`; legal/rights disposition, the two installed-App findings, Release state, settings, visibility, signing, publication, and announcement remain unchanged no-gos.

## Checkpoint C — Exact-Version Go/SumDB Bootstrap Into The Signed Updater — Complete

- Replace the fail-closed placeholder with a small Bash bootstrap that accepts only one exact stable semantic tag, requires Go 1.25.12 or newer, and builds the canonical `github.com/greaveselliott/mars/cmd/mars@<exact-tag>` through an enabled public Go proxy and SumDB with canonical-module private/no-sum bypasses disabled. Floating refs, pseudo-versions, replacements, direct/off proxy mode, disabled SumDB, and malformed versions fail closed.
- Build only into an owner-controlled temporary staging directory, verify the staged binary's canonical command/module path and exact module version with standard Go build metadata, then invoke that staged MARS binary's existing signed updater with the same explicit version and the selected final install directory. The signed updater remains the sole archive/signature verifier and durable replacement authority; pre-commit rejection preserves any prior final binary, while a recovery-required updater result preserves transaction evidence and reports exact recovery guidance. Successful packaged operation does not require Go.
- Bootstrap/setup must not require or print GitHub credentials. Keep output bounded and free of tokens, response bodies, temporary paths, and module-cache contents. Use standard Bash/Go tools and focused command stubs or a bounded local module-proxy fixture; do not create an installer subsystem. Commit and push this independently green checkpoint.

Exact pushed commit `85c689c70ef801a2747acabf537739c9ebad3c12`
completes the Checkpoint C implementation in a bounded 13-file packet. Direct
`#!/bin/bash -p` execution suppresses imported functions and `BASH_ENV`, starts
the real body in an allowlisted clean environment, keeps optional GitHub tokens
out of `env(1)` arguments and the Go process, closes their transport
descriptors before any child command, and scopes them only to the staged signed
updater. The exact Go lane fixes the public proxy and SumDB, disables private,
direct, workspace, replacement, Go-auth, CGO, and inherited tool controls, uses
owner/root-trusted temporary ancestry plus exact private staging identity, and
enables `-modcacherw` so successful cleanup is verifiable.

The staged command's runtime `BuildInfo` must prove the canonical command and
module, byte-exact requested stable tag, canonical SHA-256 `h1` sum, and absence
of main/dependency replacements before the existing signed updater may acquire
or replace anything. Bootstrap mode alone cannot bypass that proof; normal
stable-version/commit admission remains unchanged. The handoff skips only
shell-profile mutation after durable replacement. Pre-commit rejection leaves
the prior binary unchanged, while recovery-required post-commit results retain
transaction evidence and give truthful remediation. Successful exit requires
verified staging removal; incomplete cleanup preserves the original failure or
reports installed-with-incomplete-cleanup without exposing a path.

Normal and race tests for `internal/selfupdate` and `cmd/mars`, affected vet,
the hostile installer suite, Bash 3.2 syntax and native descriptor-close proof,
CLI/help mirror checks, docs consistency, DocSync across 366 files with zero
findings, and `git diff --check` passed. QA, Security, Release Manager, and
Orchestrator returned GO on exact frozen installer hashes
`87c2bc1d9769d1cb9f121e34b706dde25e02f6ede5e35380fc07ffb1fa042192`
and `d055e830091fea197549756c02248385182290088a5ad06ced4fbe848e962911`.
The native closure passed at exact clean pushed source
`56b8de336cf4d1439944cc7eb8ea0f5ad4043f2b`. Go 1.26.5 produced CGO-disabled
Darwin/Linux AMD64/ARM64 binaries with exact VCS metadata. A clean-HOME macOS
arm64 source install ran deferred setup twice at `4/0` then `0/4` steps. A
native Linux arm64 non-root lane passed the full installer suite under GNU
`stat`, retained its exact `ddf7ee…` binary for independent review, and produced
the same clean setup and postcondition results without GitHub tokens or a local
fallback. Linux automatic llama.cpp acquisition remained disabled. QA and
Security independently recomputed the retained evidence and returned GO; all
four temporary evidence roots were then removed and verified absent. The exact
matrix, hashes, environment, rejected attempts, cleanup, and GitHub-hosted CI
billing blocker are recorded in
`docs/validation/reports/2026-08-24-t077-bootstrap-setup-closure.md`.

A real official-tag proxy/SumDB plus signed install/update/rollback lifecycle
remains deliberately assigned to T-080/T-081. T-077 therefore closes while
F-017-S003 remains incomplete.

## Interfaces And DocSync

Expected implementation surfaces are `internal/githubauth`, config-owned local-token clearing, setup command/steps, `scripts/install.sh`, directly owning self-update/setup tests, MarsCLI and generated-target command guidance, README/AGENTS/quickstart/configuration/release/local-inference docs, F-009, F-017, and DocSync metadata only where behavior changes. Do not widen into producer admission, signing workflows, GitHub administration, community governance, legal disposition, or automatic Linux llama.cpp enablement.

## Validation

For each semantic checkpoint run affected-package normal tests, focused hostile/race tests, package vet, formatting, DocSync/docs consistency, and diff checks. At closure run four CGO-disabled Darwin/Linux AMD64/ARM64 builds once and one installed clean-HOME macOS lane plus one Linux lane without GH_TOKEN, GITHUB_TOKEN, GitHub CLI auth, or local fallback. The private bootstrap fixture must prove exact-version/SumDB command composition and prior-binary preservation; actual official-tag anonymous download/update/rollback remains a T-080/T-081 gate and must not be claimed here.

Complete. The exact closure report records all four builds, clean macOS and
native non-root Linux lanes, frozen installer hashes, independent review,
rejected/superseded attempts, and verified cleanup. Hosted CI did not start due
to the GitHub Billing & plans condition; this external blocker is carried into
T-078 and does not replace the accepted local/native evidence.

## Acceptance

- Setup and official-readiness checks work without GitHub credentials; optional authenticated fallback is classified without disclosure; clear-local removes only the MARS config fallback.
- No third-party binary/model download begins before the exact provenance plan and explicit acknowledgement; skip/defer paths remain non-downloading.
- The exact-version Go/SumDB bootstrap reaches only the canonical module, stages separately, and hands off to the existing signed updater. Every pre-commit rejection preserves the prior installation; recovery-required post-commit results preserve transaction evidence and report exact remediation without claiming that replacement never occurred.
- Clean macOS and Linux source/setup lanes run without GitHub credentials, with the Linux inference limitation stated truthfully.
- T-077 may complete as private implementation evidence, but F-017-S003 remains incomplete pending T-078, T-080, and the logged-out T-081 lifecycle. Repository visibility stays private and Primary Status stays `primary_blocked`.

## Stop Conditions

Stop on any floating or mutable bootstrap selector, SumDB/private-module bypass for the canonical module, unsigned archive install, duplicated signature logic, credential requirement or disclosure on the official path, unacknowledged third-party download, enabled unsupported Linux llama.cpp acquisition, overwritten prior binary after bootstrap failure, public tag/Release/signing/settings/visibility mutation, or a claim that a not-yet-published official lifecycle passed.
