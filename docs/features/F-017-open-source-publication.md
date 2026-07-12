# F-017: Open-Source Publication

- Feature ID: F-017
- Goals: G-OSS-001, G-001, G-002, G-003, G-004
- Status: active
- Owner: foundation-maintainer

## Business Logic

MARS may become public only when publication authority and technical readiness
are independently proven. Reversible technical work may proceed while
ownership/licensing review is pending, but no supporting result authorizes an
irreversible public action.

Publication authority covers the right to license source, predecessor material,
prompts, documentation, and assets, plus explicit disposition of historical
privacy/IP findings and the product name. Agents record evidence but do not make
legal conclusions.

Technical readiness covers secure-by-default network, browser, filesystem,
execution, credential, release/update, contribution, and supply-chain behavior.
Official public release access must work anonymously. Optional authentication
may support rate limits or private forks but cannot be required for the official
public repository.

Visibility conversion is a separately approved manual action after all
scenarios pass. Historical public tags are immutable. Unsafe assets may be
withdrawn, but tags are never moved. Returning a public repository to private
does not restore confidentiality.

## Step-By-Step Behavior

1. The Orchestrator classifies evidence as foundation-owned, mirrored doctrine,
   evidence-only, deployed-owned, or mixed/unclear.
2. COO maintains the Primary Outcome Contract and scenario schedule.
3. CTO creates only the current bounded ticket through `ticket_create`.
4. Engineer implements it with tests, documentation, and `MarsDocSync`.
5. QA and Security independently review acceptance and containment.
6. Dogfood validates affected runtime or release behavior using the installed
   binary on clean matrix targets and records a report or exact blocker.
7. Release Manager versions and verifies only legally permitted private artifacts.
8. Audit evidence committed to the repo is redacted; raw secret candidates
   remain outside the repository with owner-only protection.
9. A real secret is rotated before removal or history rewriting.
10. Legal/owner evidence decides whether audited history is preserved or a
    clean public snapshot is required.
11. Visibility stays private until every scenario, no-go gate, and role
    sign-off passes.
12. After separately approved cutover, logged-out smoke runs immediately,
    integrations are re-enabled least-to-most privileged, and announcement
    waits for a clean 48-hour canary.

## Scenario Schedule

1. F-017-S002 - Runtime and source operations are secure by default or explicitly gated.
2. F-017-S003 - Public release/install/update works anonymously with signed provenance.
3. F-017-S004 - External contributions use fork-safe least-privilege CI and protected refs.
4. F-017-S001 - Ownership, history, privacy, provenance, and GitHub surfaces are approved.
5. F-017-S005 - Private rehearsal, logged-out cutover, rollback, and canary pass.

## Scenarios

### F-017-S001: Publication Authority And Audit Clearance

Given MARS source, history, releases, GitHub surfaces, predecessor material,
prompts, documentation, and assets are candidates for publication
When pinned scans, manual privacy/IP review, provenance review, and owner/legal
review complete
Then every finding has a durable redacted disposition
And real secrets were rotated before removal
And publication authority and name clearance are explicitly recorded
And the decision states whether audited history is preserved or a clean
snapshot is required.

### F-017-S002: Secure Runtime Defaults

Given a new user runs supported MARS commands against a hostile or untrusted
repository
When dashboard, webhook, setup, telemetry, filesystem tools, shell execution,
cleanup, logs, and traces operate
Then listeners default to loopback
And remote control requires explicit authentication and request protections
And unsigned or untrusted GitHub events cannot enqueue autonomous mutation
And repo filesystem operations cannot escape through symlinks
And observer execution is the new-install default
And host execution requires explicit acknowledgement
And child environments, state permissions, logs, and traces contain credentials
and sensitive data according to documented policy
And vulnerability gates fail closed.

### F-017-S003: Anonymous Signed Release Lifecycle

Given a logged-out user has no GitHub token
When they build, install, set up, update, verify, or roll back MARS
Then official public repository metadata and assets are available anonymously
And every supported archive contains the binary, license, notices, third-party
notices, SBOM, and provenance metadata
And a pinned public key verifies the release manifest
And the updater rejects tampered, stale, extra, unsigned, wrong-version,
wrong-platform, or wrong-commit assets before atomic replacement
And optional authentication remains available for rate limits and private forks.

### F-017-S004: Fork-Safe Contribution And Governance

Given an untrusted external contributor opens a fork pull request
When repository automation runs
Then it receives no secrets, write token, release key, or self-hosted MARS runner
And it does not use `pull_request_target`
And required checks, DCO, review, conversation resolution, and protected
`main`/`v*` rules apply
And only narrowly scoped maintainers or the trusted MARS App can bypass
direct-trunk controls
And public security reports have a private disclosure path
And community, support, governance, maintenance, and AI-assisted contribution
expectations are documented.

### F-017-S005: Controlled Cutover And Recovery

Given F-017-S001 through F-017-S004 pass and all required roles sign off
When the owner separately approves visibility conversion
Then a final backup/settings/ref/artifact manifest exists
And the signed cutover release passes private verification
And logged-out clone, build, install, update, vulnerability reporting, and fork
PR smoke pass immediately after conversion
And integrations are re-enabled from least to most privileged
And rollback and incident procedures are executable
And public announcement waits until the 48-hour canary remains clean.

## Out of Scope

- Agents providing legal advice or independently declaring ownership.
- Changing repository visibility before the separately approved cutover.
- Finalizing ownership-dependent copyright or notice text before clearance.
- Deleting or publicly republishing historical assets before clearance.
- Adding cloud runtime dependencies.
- Guaranteeing support for releases older than the latest supported release.

## Descoped Scenarios

None.

## Evidence

- F-017-S001: OSS-00 is the current read-only inventory slice. It may establish surface completeness, access gaps, and an offline restricted-evidence scan contract, but the scenario remains pending pinned scans, manual privacy/IP review, provenance disposition, publication-authority attestation, and name clearance.
- F-017-S002: T-055 technically completes the vulnerability-baseline enabler: Go 1.26.5, the Go-1.22-compatible `x/sys` v0.30.0 disposition, fail-closed scanner regressions with pinned v1.6.0 install remediation, and a v1.6.0 scan with zero reachable findings passed Engineer, QA, Security, and Dogfood evidence. The scenario remains incomplete pending the later runtime P0 tickets.
- F-017-S003: Pending deterministic-build, signature, anonymous install/update, negative artifact, bridge, and rollback evidence.
- F-017-S004: Pending community/CI checks and disposable public hostile-fork rehearsal.
- F-017-S005: Pending private rehearsal, role sign-offs, logged-out smoke, rollback record, and 48-hour canary report.
