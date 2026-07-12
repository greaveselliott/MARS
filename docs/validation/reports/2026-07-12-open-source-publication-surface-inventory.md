# Open-Source Publication Surface Inventory

- Date: 2026-07-12
- Ticket: T-056
- Scenario: F-017-S001
- Classification: evidence-only, with legal and ownership conclusions mixed/unclear
- Primary Status: `primary_blocked`
- Collection mode: read-only local Git and authenticated GitHub metadata projections
- Local aggregate snapshot: `2026-07-12T13:47:01Z` UTC at source HEAD `29c75ab05d37e1bccaaefd39513c2f65917234e7`
- Local publishable-ref manifest SHA-256 at snapshot: `61f83b94fae8c5dd316e4d1993578cf4d305d481d433c56ac38db21211683d2e`
- Technical recommendation: `undecided`

## Primary Outcome Contract

**Primary Outcome:** Publish MARS as a supported open-source project without
exposing confidential material, weakening controls, or distributing unsafe or
unverifiable binaries.

**Primary Pass Gate:** A logged-out user can clone, build, install, update,
report vulnerabilities, and submit a safe fork PR; exposed history is approved;
runtime P0s are closed; and public artifacts are licensed, signed, and tied to
an immutable source commit.

**Primary Status:** `primary_blocked`

**Current Primary Blocker:** Publication authority and trademark/name clearance
are not established, and restricted-evidence scans and manual privacy/IP/
provenance dispositions remain incomplete.

**Next Primary Action:** Complete independent review of this redacted inventory,
then have the owner provision the restricted evidence boundary for pinned
offline scans and inaccessible-surface collection while legal review proceeds.

**Supporting Evidence:** T-056 inventories local Git and GitHub publication
surfaces without mutation or restricted-content disclosure and records every
observed gap as unknown or pending rather than inferring it clean.

## Outcome

The bounded inventory completed without changing Git refs, GitHub state,
releases, credentials, or repository visibility. Local publishable refs match
the advertised branch and tag manifest. Git object integrity has no missing,
broken, garbage, or error objects. The inventory also identified local-only
administrative refs and unreachable object categories that must be included in
the later owner-controlled restricted review.

GitHub aggregate metadata was available for most repository, release, Actions,
and access surfaces. Package, security-alert, private-vulnerability-reporting,
and GitHub App detail could not be cleanly distinguished between absent,
disabled, and inaccessible with the current credential. Those states remain
unknown; they are not evidence of a clean surface.

This report does not pass F-017-S001. Pinned all-ref scans, release/workflow
artifact scans, manual privacy/IP/provenance review, publication-authority
attestation, and trademark/name clearance remain pending.

## Evidence Boundary

Only aggregate counts, status booleans, classifications, and opaque SHA-256
manifest identifiers were projected into the agent session and this report.
Read-only API responses were reduced inside the CLI; no issue, pull-request,
comment, discussion, workflow-log, artifact, wiki, deployment, release-asset,
or attachment body/name was exposed to the agent transcript or recorded. No
author email, webhook configuration, credential value, authenticated URL,
candidate path/value, or raw API/scanner JSON was exposed or recorded.

The remote identity is deliberately redacted here. It is one private GitHub
origin, and the fetched `origin/main`, advertised `main`, and local `HEAD`
matched at collection time.

## Local Git Inventory

| Surface | Status | Redacted evidence |
| --- | --- | --- |
| Publishable advertised refs | Inventoried | 296 refs: 1 head and 295 tags |
| Local publishable refs | Inventoried | 1 head and 295 tags; no local-only or missing advertised tags |
| Ref manifest | Inventoried | Advertised and matching local SHA-256: `61f83b94fae8c5dd316e4d1993578cf4d305d481d433c56ac38db21211683d2e` |
| Local-only refs | Requires restricted review | 2 remote-tracking refs, 5 agent administrative refs, and 1 preserved operator stash; none are advertised publication refs |
| Notes and replace refs | Absent | 0 notes refs; 0 replace refs |
| Publishable history | Inventoried | 782 commits and 10,995 reachable objects across local heads/tags |
| All local refs | Requires restricted review | 784 commits and 11,001 reachable objects; the delta comes from local-only refs |
| Tags | Inventoried | 295 tag refs, including 30 annotated tag objects |
| Unreachable objects without reflogs | Requires restricted review | 36 commits, 1,319 trees, 191 blobs, 0 tags |
| Object integrity | Inventoried | `git fsck --full`: 0 missing, 0 broken, 0 errors; 0 garbage objects |
| Loose/dangling storage | Requires restricted review | 3,038 loose objects and 546 dangling-object reports at `2026-07-12T13:47:01Z`; content was not inspected in this ticket and these mutable counts must be refreshed when the restricted audit starts |
| LFS | Absent at current HEAD | 0 LFS attributes and 0 detected pointer files; Git LFS client is available |
| Submodules | Absent at current HEAD | 0 declarations |

The unreachable, dangling, stash, and local administrative surfaces are not
part of the advertised ref set, but they remain possible privacy/IP evidence.
They must be copied into the restricted evidence boundary for classification;
their absence from advertised refs is not a clean disposition.

## GitHub Publication Surfaces

| Surface | Status | Redacted evidence or exact gap |
| --- | --- | --- |
| Repository | Inventoried | Private; not archived, disabled, or a fork |
| Repository features | Inventoried | Issues, Projects, and Wiki enabled; Discussions disabled |
| Branches and rules | Inventoried | 1 branch; 0 protected branches; 0 repository rulesets |
| Releases and assets | Inventoried, content scan pending | 50 releases and 446 assets; names/content were not retrieved |
| Issues, PRs, Discussions | Absent by aggregate count | 0 Issues, 0 PRs, 0 Discussions; bodies were not queried |
| Collaborators/access | Inventoried | 1 collaborator with admin access |
| Deploy keys and webhooks | Absent by aggregate count | 0 deploy keys; 0 webhooks; configurations were not queried |
| Pages and Wiki | Inventoried, content scan pending | Pages reports `built`; Wiki is enabled; content/history was not retrieved |
| Actions | Inventoried, restricted scan pending | 2 workflows, 335 runs, 1 retained artifact, 0 caches |
| Actions permissions | Inventoried | Actions enabled; all action sources currently allowed; default workflow token is read-only and cannot approve PR reviews |
| Repository secrets/variables | Inventoried by count | 0 Actions secrets, 0 Actions variables, 0 Dependabot secrets; values cannot be returned by GitHub and were not requested |
| Environment surfaces | Partially inventoried | 1 environment; environment identifiers, protection rules, secrets, and variables were not queried in this bounded pass and require owner-controlled review |
| Deployments | Inventoried, content scan pending | 63 deployment records; payloads and environment identifiers were not retrieved |
| Packages | Inaccessible | All five applicable package-type count requests failed with the current credential; absence cannot be inferred |
| GitHub App installation | Inaccessible or absent | The read-only repository-installation query did not distinguish permission denial from no installation |
| Security configuration | Inaccessible or unreported | Advanced Security, secret scanning, push protection, and Dependabot security-update states were not returned |
| Security alerts | Inaccessible or disabled | Dependabot, code-scanning, and secret-scanning aggregate alert endpoints did not return counts; findings cannot be inferred absent |
| Private vulnerability reporting | Inaccessible | Current credential did not return the feature state |
| Repository vulnerability alerts | Disabled or inaccessible | Read-only feature probe did not distinguish the two states |
| Attachments, logs, and historical workflow artifacts | Not attempted | Routed directly to the restricted-evidence audit to avoid private body/name disclosure |
| Org/user-level settings, variables, secrets, runners, and access | Not attempted | Outside this repository-only metadata slice; requires owner-authorized account/org inventory |

## Surface-To-Restricted-Scan Matrix

| Surface | Later restricted input | Required treatment |
| --- | --- | --- |
| Reachable advertised refs | Encrypted mirror of every advertised branch/tag and reachable object | Gitleaks and TruffleHog all-ref scans plus manual privacy, path, IP, license, and provenance review |
| Local-only refs and unreachable objects | Owner-captured local ref/reflog manifest and safely materialized unreachable commit/tree/blob set | Scan and manually classify separately from advertised history; never infer clean from non-advertisement |
| LFS | Pointer history plus every referenced LFS object from all refs | Secret scan object content; verify completeness, license, provenance, and object hashes |
| Releases | Release metadata, notes, every asset, checksums, and source linkage | Secret/malware/license/provenance review; validate archives under the hostile-archive rules below |
| Pages and Wiki | Pages source/build output and complete Wiki export/history | Secret, personal-data, path, third-party-text, license, and provenance review |
| Actions logs and artifacts | Complete retained run-log, artifact, cache, and workflow export | Secret/personal-data scan, retention review, and artifact provenance classification |
| Attachments and packages | Issue/discussion attachment export and repository/account package metadata plus payloads | Secret/malware/license/provenance review with hostile-archive handling for every package/archive |

Every row is owned by the operator-appointed Security audit operator inside the
restricted environment. Owner/legal counsel owns the resulting privacy, IP,
publication-authority, and preserve-history versus clean-snapshot disposition.

## Restricted-Evidence Contract

The next audit must use an operator-approved encrypted
`MARS_OSS_AUDIT_ROOT` outside this repository and outside ordinary temporary
directories. The operator creates it before any collection with `umask 077`,
directory mode `0700`, and file mode `0600`. The actual path must never appear
in tickets, docs, chat, CI, traces, screenshots, or command output.

Within that boundary:

1. Create an encrypted off-repo mirror and a manifest of all refs and audited
   GitHub surfaces.
2. Run collection inspection and scanners in a disposable, unprivileged
   environment with an empty or explicitly sanitized environment; no host
   credentials, agent sockets, cloud metadata access, host home directory, or
   ambient token/config mounts are available.
3. Mount acquired inputs read-only and mount only the restricted audit output
   directory writable. Apply explicit CPU, memory, elapsed-time, output-size,
   file-count, and process-count limits, and terminate the environment after
   each input class.
4. Enforce network `none` during inspection and scanning. Acquisition and
   signature/provenance verification happen in a separate trusted staging step;
   unverified inputs never gain network access.
5. Treat every archive/package as hostile. List entries without extraction,
   reject absolute paths, `..` traversal, symlinks, hardlinks, devices, FIFOs,
   and entries exceeding file/count/expanded-size limits, then extract only in
   the disposable environment. Reapply the same checks to nested archives.
6. Write raw stdout, stderr, JSON, candidate fragments, paths, bodies, names,
   and manifests only to owner-only files under the audit root.
7. Project only counts, classifications, opaque random finding IDs, tool/file
   hashes, and redacted dispositions back into repository evidence.
8. Do not pass raw output through agents, chat, CI, traces, SaaS uploads, or
   screenshots.
9. If a plausible credential appears, stop without reproducing it. Rotation is
   separately approved incident work and must precede removal, asset deletion,
   or history rewriting.

## Pinned Offline Scanner Prerequisites

The scanners were neither installed nor executed in T-056.

| Tool | Required pin | Offline/restricted execution contract |
| --- | --- | --- |
| Gitleaks | v8.30.1 | Acquire from the trusted upstream release channel; verify its publisher-signed checksum/signature or immutable image digest and available provenance before staging; run with network disabled, `--no-banner`, `--no-color`, and `--redact`; route any JSON/report file directly to the restricted root; scan the encrypted mirror across all refs |
| TruffleHog | v3.95.9 | Acquire from the trusted upstream release channel; verify its publisher-signed checksum/signature or immutable image digest and provenance, including vendor cosign signature/attestation verification where the publisher supports it; run with network disabled, `--no-update`, and `--no-verification`; route JSON directly to the restricted root; do not use SaaS/provider verification or upload features |

The version pins and a locally calculated SHA-256 alone are not trusted
provenance. The operator must verify each artifact against an independently
trusted publisher signature, signed checksum, transparency/provenance record,
or immutable signed image digest, then record that verification and the local
binary/image SHA-256. Flags must be confirmed against those exact pinned
binaries inside the restricted session; if a pin does not support the required
offline/redaction behavior, the scan stops rather than falling back to `latest`
or enabling network verification.

## Classification And Recommendation

- `evidence-only`: counts, ref-manifest hash, integrity result, GitHub surface
  states, and access gaps in this report.
- `mixed/unclear`: rights, predecessor provenance, personal-data acceptance,
  trademark/name clearance, and the final history-publication choice.
- `foundation-owned`: later secure defaults, release integrity, update access,
  CI, and governance implementation tickets.
- `deployed-owned`: none.

Technical recommendation: `undecided`. Preserve-audited-history is still a
viable default because advertised refs match local publishable refs and no Git
integrity defect was found. It cannot be recommended yet because unreachable
objects, 446 release assets, retained Actions evidence, Wiki/Pages content,
inaccessible security/package surfaces, full-history secret scanning, and
manual privacy/IP/provenance review are unresolved. A real secret, unlicensed
material, or unresolvable personal-data finding may force
`clean_public_snapshot` after rotation/removal and owner/legal review.

## Next Actions And No-Go State

**Downstream blocker:** The restricted audit environment, trusted scanner
provenance, inaccessible GitHub/account exports, content scans, and legal
dispositions do not yet exist. **Owners:** the repository owner provisions and
authorizes collection; the appointed Security audit operator runs the isolated
audit; owner/legal counsel decides publication authority, name clearance, and
history disposition.

1. Owner provisions the restricted evidence root and isolated execution
   environment, then verifies trusted scanner signatures/digests/provenance.
2. Collect the off-repo backup/mirror and the currently inaccessible GitHub,
   account, security, package, environment, Pages/Wiki, release-asset, and
   workflow-artifact surfaces without exposing raw output to agents.
3. Run pinned offline all-ref/LFS/artifact scans and manually review privacy,
   IP, provenance, predecessor material, and absolute personal-path exposure.
4. Rotate any plausible real secret before removal and assign only opaque
   finding IDs to durable dispositions.
5. Obtain explicit publication-authority and trademark/name clearance, then
   record the owner/legal choice between audited history and a clean snapshot.

Repository visibility, history rewriting, release deletion or republication,
copyright/NOTICE finalization, credential rotation, and announcement remain
blocked. Primary Status remains `primary_blocked`.
