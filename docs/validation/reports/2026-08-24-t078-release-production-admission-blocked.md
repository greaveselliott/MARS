# T-078 Release Production Admission — Blocked

**Date:** 2026-08-24
**Ticket:** T-078
**Scenarios:** F-017-S003, F-018-S004
**Source:** `3090fb8dcaefd26f1cc17dd6f437c4c2b7d8ce92`
**Status:** `blocked`
**Primary Status:** `primary_blocked`

## Primary Outcome Contract

**Primary Outcome:** publish MARS as a supported open-source project without
exposing confidential material, weakening controls, or distributing unsafe or
unverifiable binaries.

**Primary Pass Gate:** the repository is public; signed `v0.69.1` is supported
with signed `v0.69.0` retained only as its rollback bridge; all F-017 scenarios,
logged-out macOS/Linux lifecycle checks, fork controls, GitHub security and
community surfaces, and the 48-hour canary pass.

**Primary Status:** `primary_blocked`

**Current Primary Blocker:** T-078 cannot admit the selected production
producer. The exact SumDB-authenticated GoReleaser v2.17.1 binary built with the
repository-pinned Go 1.26.5 toolchain reports 12 called-symbol vulnerabilities,
including two with no fixed module version. The production toolchain is also
behind the current supported Go 1.26 patch. Independently, T-073 still requires
qualified trademark counsel's written disposition and the owner's signed
authority/history attestation; the two all-repository write-capable Apps remain
launch no-gos; and GitHub-hosted jobs fail before step one on the owner-account
Billing & plans condition.

**Next Primary Action:** replan T-078 around a current supported Go patch and an
exact scan-clean GoReleaser release/dependency graph, then rerun provenance,
BuildInfo, pre-execution called-symbol, fork-authority, and two-root artifact
gates. Do not implement or execute the production producer, activate a signer,
delete hosted objects, change hosted settings, create a launch tag, sign,
upload, publish, or change visibility under the failed selection.

**Supporting Evidence:** the exact producer provenance and pre-execution scan,
the independently reconciled 65-run Actions delta, the zero-finding offline
secret scans, and the redacted digests below.

## Admission Boundary And Result

The repository and `origin/main` were clean and equal at the exact source above.
`VERSION` remained `0.68.49`; the source fallback remained `0.69.0-dev`. No
repository source or workflow file changed. No GoReleaser, Syft, or Cosign
binary was executed. No GitHub object or setting changed.

The required admission sequence stopped correctly: acquire authoritative
identity, build with the pinned toolchain and CGO disabled, inspect BuildInfo,
scan the binary, and only then consider execution. GoReleaser failed the scan
stage. Syft's exact binary build completed, but it was not scanned or executed
after GoReleaser stopped the shared producer admission. Cosign was not built.
Checkpoint A was attempted and stopped at the GoReleaser scan gate with no
partial pass; the exact-ten rehearsal remains unattempted.

## Authoritative Tool Identities

| Tool | Exact module identity | Source commit | Admission result |
|---|---|---|---|
| GoReleaser v2.17.1 | `github.com/goreleaser/goreleaser/v2@v2.17.1`, `h1:7nWdnNZSeiutF2PKAKmCEdEnLO+n0BWDoj4AqNiMX1E=` | `83f4c19a5c5c0b9efef6bf2aedc6805bbcb9dfe2` | Built and scanned before execution; blocked |
| Syft v1.50.0 | `github.com/anchore/syft@v1.50.0`, `h1:kSQ4oshw6dwHxcYhrH1jUZl/M05kiCfyPoGJgvXe61s=` | `16223e6dd7893fe578787658ceb876257483d404` | Built; BuildInfo inspected; not scanned or executed after the producer stop |
| Cosign v3.0.6 | `github.com/sigstore/cosign/v3@v3.0.6`, `h1:k8XaUd9pmLknHBst/v0rUGHVdB4D9cfaBmWUaMAOocE=` | `f1ad3ee952313be5d74a49d67ba0aa8d0d5e351f` | Identity acquired; not built, scanned, or executed |

The public Go proxy `.info` records and SumDB module and `go.mod` records bind
all three versions. GoReleaser's annotated tag verifies and peels to the
recorded commit. Syft's annotated tag is unsigned, while its peeled commit is
verified; this report does not describe the Syft tag as signed. Cosign's tag is
verified, but its GitHub Release is not immutable at this observation, so
mutable Release assets are not its admission root. Exact module/SumDB evidence
is the common authoritative path.

Primary upstream references:

- <https://goreleaser.com/getting-started/install/oss/>
- <https://github.com/goreleaser/goreleaser/releases/tag/v2.17.1>
- <https://github.com/anchore/syft/releases/tag/v1.50.0>
- <https://oss.anchore.com/docs/installation/verification/>
- <https://github.com/sigstore/cosign/releases/tag/v3.0.6>
- <https://docs.sigstore.dev/cosign/system_config/installation/>
- <https://go.dev/doc/devel/release>
- <https://go.dev/doc/security/vuln/database>

The reviewed SumDB provenance projection had SHA-256
`669c827bed9d722abf9cd45cc87b4f4b3653bda980d9e952c55288a9ff3464c1`.

## Exact Producer Scan Failure

The GoReleaser binary was built with CGO disabled using Go 1.26.5. Its SHA-256
is `5c8a99c75b7ed74d6c815e62af481e8324d52cb68642b0b8af8e7f8a107db08c`.
`go version -m` proved the exact main module, version, canonical module sum,
Go 1.26.5, and zero replacements. The BuildInfo record has SHA-256
`c00bc8618d681057d8e702699810a6f9d69d375951434f9b595369d7a93dd3ea`.

Exact `govulncheck v1.6.0` was itself SumDB-authenticated as
`golang.org/x/vuln/cmd/govulncheck@v1.6.0` with
`h1:FeMO9Rm/HwyduOztbvKcOw+zvDEPr4I4aQNSfevFcKY=`. Its binary SHA-256 was
`c04b245fe27e4eeba7ede49411f6a7de1f90fa0150263820d9b3636b95a87d76`.
The binary's BuildInfo and its own version output both recorded Go 1.26.5. The
official database was `https://vuln.go.dev` at `2026-08-21T20:38:00Z`.

The binary-mode structured scan found 12 called vulnerability IDs and 104
unique terminal called-symbol records:

| Disposition | IDs | Required correction |
|---|---|---|
| No fixed dependency version | `GO-2026-6225`, `GO-2026-5932` | Upstream removal/fix or separately authorized, narrowly reviewed disposition; no waiver exists in T-078 |
| Dependency fix available | `GO-2026-6214`, `GO-2026-6162` | A producer release using go-git/v5 v5.19.2 or newer and sigstore-go v1.2.1 or newer, followed by a fresh binary scan |
| Standard-library fixes begin in Go 1.26.6 | `GO-2026-6218`, `GO-2026-6091`, `GO-2026-6090`, `GO-2026-6089`, `GO-2026-6088`, `GO-2026-5972`, `GO-2026-5942`, `GO-2026-5026` | Rebuild with the current supported patch, not the already superseded minimum fix |

Go 1.26.7 was the current 1.26 patch on 2026-08-24; Go's official release
history records Go 1.26.6 on 2026-08-13 and Go 1.26.7 on 2026-08-19. Updating
only to the minimum 1.26.6 fix would already select a stale patch and would not
resolve the four dependency/no-fix records.

The full JSON scan exited zero despite the called-symbol findings. A future
admission gate must parse the structured stream and fail on called symbol
traces, or use a separately proven non-JSON finding-fails mode; process exit
alone is not an admission result. The full JSON SHA-256 is
`294e05d859e885c3b8457dcbf302a76e853a3c084ecd4df41dadb63e89ed645e`;
the redacted called-summary SHA-256 is
`ca4c6e0d4c9fef80e5d0263c7fb66e4094b102ce203e8dc267ce97be7f9a2365`.

The reviewed Syft binary was Mach-O arm64 with SHA-256
`dee30767063e61c08067d6af327bbb841793fcdc47013336eeac3b840781acf1`.
Its BuildInfo records Go 1.26.5, canonical command
`github.com/anchore/syft/cmd/syft`, exact module version and canonical sum, and
no replacement. It was not scanned or executed because the shared producer
selection had already stopped at GoReleaser.

## Read-Only Hosted Run Reconciliation

T-072 sealed 401 completed workflow-run IDs with SHA-256
`cdfb0cc95e6dc8f4e2b9e910f3c897834fa17d2d18f3a9e454c2b04db74b938b`.
On 2026-08-24, the oldest 401 IDs reconstructed from the live repository
reproduced that digest exactly. Start and end inventories then contained 466
completed runs and zero active runs. The exact 65-run delta has sorted-ID
SHA-256 `2cb0b770e3915f6fcacc6cd086ba9fe808e8279c6f50e7b6a7279b5da8d9e0fc`.

Read-only acquisition collected 65 run records, 65 attempt records, 252 jobs,
65 log ZIPs, and every per-run artifact inventory. Four log ZIPs were valid and
empty; the other archives contained 472 regular log entries totaling
14,210,509 expanded bytes. Artifact count was zero. All 65 ZIPs passed two
bounded validations rejecting absolute/traversal/control-character paths,
links, devices, duplicate or case-colliding names, excessive entry counts,
and excessive per-file or total expansion.

The acquired-input manifest contains 797 files and 19,320,002 bytes. Its
SHA-256 is `eb5038c08af8f8b0b24a20251659fc1d01e0db37a3bd05bd0240137e60a3c744`.
Raw provider records and the complete exact-identifier manifests remained
outside the repository in an owner-only temporary boundary through independent
review and were then removed as recorded below.

## Offline Secret Scan

The exact T-072-admitted scanner binaries were reacquired and matched their
recorded identities:

- Gitleaks v8.30.1:
  `ba52fb1bfabbcde42f032afad3d6e0b19dff8ed105229a16e7caa338bbc0e84f`
- TruffleHog v3.95.9:
  `8c6110728eca539ac188a149d8a1e0510e5e59e4d3e3f1ce9daa41fa4961814f`

Both ran against only the acquired input tree with a clean environment.
Updates, credential verification, network-backed sources, profiling, and
archive re-expansion were disabled. Scanner output remained in a separate
owner-only directory and was not rescanned.

Gitleaks returned zero findings; its empty JSON report SHA-256 is
`37517e5f3dc66819f61f5a7bb8ace1921282415f10551d2defa5c3eb0985b570`.
TruffleHog returned zero findings; its empty JSONL report SHA-256 is
`e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`.
Scanner errors, skips, timeouts, rejected inputs, and unresolved candidates are
zero.

This scan-clean delta does not authorize deletion. The report commit and every
later run must be acquired, scanned, reconciled, and added to the exact
transaction before any separately approved deletion.

## Signer And Hosted Authority Finding

Read-only settings inspection found the default Actions token at read, fork
workflow writes/secrets disabled, `allowed_actions=all`, SHA-pinning enforcement
disabled, zero rulesets, no branch protection, and no protected release/signing
environment. Only the unrelated `github-pages` environment exists. The two
all-repository write-capable Apps remain unresolved.

Therefore a production signer cannot yet be described as maintainer-only. A
future workflow must remain structurally dormant until T-079 restricts/removes
the Apps and establishes the admitted environment/ruleset/tag controls, and
T-080 revalidates them and supplies its separate release authority. Merely
naming an environment would not prove protection. An absent-by-default
repository-variable gate must be job-level and combined with the exact upstream
repository, `push` event, and exact `v0.69.0`/`v0.69.1` ref allowlist; it is
fail-closed dormancy, not T-080 authorization. No `workflow_dispatch` path may
enter the privileged signer.

Each supported Release will eventually expose exactly ten **uploaded Release
assets** through the Releases API: four archives, four SPDX SBOMs,
`checksums.txt`, and `checksums.txt.sigstore.json`. GitHub-generated source
archives and an immutable-Release attestation are additional provider surfaces,
not uploaded assets and not part of the 10/20 arithmetic.

## Outcome And Cleanup Status

T-078 remains in progress and blocked. Checkpoint A is NO-GO; the exact-ten
producer/rehearsal was not run. Checkpoint B has supporting scan-clean evidence
for the exact 65-run snapshot but cannot close because later runs remain in
scope and no cleanup transaction is authorized. Engineer, QA/Release Manager,
and Security agree that the selected producer must not be executed or admitted.

Release/QA and Security independently verified the retained producer and hosted
evidence before cleanup. The exact owner-only Actions acquisition root, five
bounded acquisition/helper files, four private build/cache roots, and four
private producer-scan evidence files were then removed. Two Go module-cache
roots contained expected read-only downloaded files; owner write permission was
restored only inside those exact reviewed roots before the successful repeated
removal. Absence of every named target and of any remaining T-078 temporary
sibling was verified.

No repository, release, tag, asset, workflow run, deployment, setting, App,
visibility, signing, or publication state changed. Local temporary evidence is
not recoverable from those removed paths; its redacted hashes, counts, results,
and authority boundary remain in this report.
