# T-080 Public Cutover Preflight

**Date:** 2026-08-26
**Ticket:** T-080
**Repository:** `greaveselliott/MARS`
**Repository ID:** `1279592869`
**Source checkpoint:** `b49a46dedc24b309891ec91dc5f2336c05dc9de6`
**Visibility:** private
**Publication status:** not authorized or performed

## Primary Outcome Contract

**Primary Outcome:** freeze the shortest credible public-cutover and two-release
transaction from an exact read-only preflight, without reviving T-078's bespoke
release-security platform.

**Primary Pass Gate:** the private pre-state, public visibility consequences,
public-only control sequence, immutable release sequence, rollback boundaries,
and remaining approvals are explicit before any hosted mutation.

**Primary Status:** `primary_blocked`

**Current Primary Blocker:** separate owner approval of the exact visibility and
publication transaction below.

**Next Primary Action:** commit and push this private preflight checkpoint, then
present the exact hosted transaction for approval. Do not change visibility,
Pages, security settings, tags, Releases, or workflow activation first.

**Supporting Evidence:** exact source/ref receipts, the signed-in read-only
GitHub inspection, the conventional workflow contract, and the passing local
preflight gates recorded below.

**Private evidence checkpoint:** commit
`32d0679616e26c070d3dffd914adbf7639704393` and hosted
[`source-compatibility` run `32911253683`](https://github.com/greaveselliott/MARS/actions/runs/32911253683).

## Scope Re-Baseline

The owner stopped expansion into custom Docker Engine/API orchestration,
ptrace/Landlock, executable-format parsing, transcript-pinned SPDX parsing, and
recursive proof machinery. That work remains preserved as historical and
non-authorizing evidence. The launch path is the conventional repository-owned
Go producer, upstream Syft, and GitHub `actions/attest` workflow already checked
in at `.github/workflows/release.yml`.

GitHub Apps, account-wide App installation scope, trademark registration, and
account funding are not launch blockers under the recorded owner dispositions.
No vulnerability exception changed.

## Exact Private Pre-State

The following facts were revalidated read-only through the signed-in GitHub UI,
local Git, and a temporary read-only clone:

- local `HEAD`, `origin/main`, and the live default branch are exactly
  `b49a46dedc24b309891ec91dc5f2336c05dc9de6`; the repository was clean before
  this T-080 evidence change;
- the repository is private, owned by `greaveselliott`, is not a fork, and has
  929 commits on the visible main surface;
- `VERSION` is still `0.68.49`; neither `v0.69.0` nor `v0.69.1` exists as a tag
  or Release;
- 301 tag refs and 11 advertised branch refs exist. The 10 non-main branch
  heads add zero commits outside `main` and the retained tag graph, so the
  branch surface exposes no branch-only history. The temporary-clone remote-ref
  manifest digest is
  `399c8008d32974bb29ff2c4eac199e1f5aae01e9468f88929ffeefda9d5d0c9e`
  and its tag-ref manifest digest is
  `0076c1b8184ce1e2dbd30bd408504412aaf51cba7ecfdff21383b9b4f216acc5`;
- 56 historical Release objects remain. Exact pagination exposes no uploaded
  Release-asset link; each historical Release shows only GitHub's generated
  source archives. The launch tag pages return not found;
- Actions has 34 retained runs, with zero queued, in-progress, or waiting runs.
  Actions caches are empty. The earlier exact T-078 receipt remains the owning
  zero-artifact and zero-deployment evidence; those two legacy UI routes are not
  exposed by the current GitHub UI;
- future-only immutable Releases are enabled. GitHub warns that published
  Release tags and assets cannot then be changed;
- the checked-in release workflow is tag-push-only, has top-level
  `permissions: {}`, uses full-SHA GitHub actions, splits read-only production,
  OIDC attestation, read-only verification, and release-only publication, and
  keeps all four jobs disabled with literal `false`. It has never run and is not
  registered as a runnable workflow in the Actions UI while dormant;
- Actions permits GitHub-owned actions only, requires full-length SHA pins,
  gives the default token read-only contents/packages authority, and prevents
  Actions from creating or approving pull requests. Private-fork workflows
  require approval and receive no write token, secrets, or variables;
- repository Actions secrets, Actions variables, environment secrets,
  environment variables, and Dependabot secrets are all empty;
- Dependabot alerts and security updates are enabled. Public-only CodeQL,
  repository secret scanning, repository push protection, and private
  vulnerability reporting are not yet available on the private repository;
- active branch ruleset `21491158`, `Protect main and require contributor
  review`, is the only ruleset;
- Pages is disabled and no publishing branch is selected. Historical
  `github-pages` environment `17186549775` has no secrets or variables; and
- Issues and Discussions are enabled. Wiki and pull-request surfaces are
  unchanged.

The GitHub CLI credential is expired, so this checkpoint used the existing
signed-in browser only for read-only hosted inspection. That credential must be
re-authenticated before any CLI/API mutation; no browser or CLI mutation was
performed.

## Dormant Workflow Validation

The conventional workflow contract remains the supported route. Existing
tests verify its full-SHA pins, tag/ref/SHA guards, exact Go 1.27 toolchain,
Syft 1.51 producer, eight checksummed subjects, offline Sigstore bundle,
exact-ten output, independent verification, and draft-before-publish sequence.

Preflight gates passed:

```text
mars guardrails secret-scan --repo . --json                 PASS (0 findings)
go test ./internal/release -count=1                         PASS
go test ./internal/selfupdate -count=1                      PASS (outside app sandbox)
go test ./internal/docsconsistency ./internal/docsync       PASS
go vet ./internal/release ./internal/selfupdate             PASS
go run ./cmd/mars docsync audit --repo .                    PASS (366 files, 0 findings)
git diff --check                                            PASS
```

The self-update suite's first in-app run could not create its deliberately
owner-local test directories. The unchanged suite passed when rerun outside
the app filesystem sandbox; this was environment evidence, not a source fix.

The private checkpoint then passed hosted run `32911253683` in 4m 40s:
dependency notices, Go 1.25.13, Go 1.27.0, and the intentional below-minimum
rejection all completed successfully. The run produced no artifact.

## Consequences of Public Visibility

GitHub's current documentation states that changing a private repository to
public makes its code visible to everyone, allows anyone to fork it, publishes
repository activity, makes Actions history and logs public, erases stars and
watchers, and disconnects existing private forks into a separate network.
GitHub also warns that applicable push rulesets may be disabled during a
visibility change. The live branch ruleset and every Actions restriction must
therefore be revalidated immediately afterward.

This transaction is not safely reversible in the ordinary sense: returning to
private does not undo copies, forks, clones, published logs, lost social state,
or immutable published Release contents. After the change, recovery means
stopping publication, repairing forward, and issuing a new patch release—not
pretending the public event did not occur.

## Frozen Launch Transaction

No step below is authorized by this report. After exact owner approval, execute
the phases in order and stop on any mismatch.

1. **Final private gate:** re-authenticate the GitHub CLI; revalidate repository
   ID `1279592869`, private visibility, exact `main`, tag/Release absence,
   immutable Releases, zero active runs, secret/variable emptiness, workflow
   pins, and the final advertised-ref manifest. Activate only the existing
   conventional tag workflow and land the `0.69.0` source/release-note commit;
   do not create the tag yet.
2. **Required public-only rehearsal:** use a disposable public repository to
   prove CodeQL default setup, secret scanning, push protection, private
   vulnerability reporting, Pages, fork-safe Actions, and GitHub attestation
   availability. Destroy no MARS state and stop if the platform behavior differs
   from the checked-in contract.
3. **Visibility cutover:** revalidate the exact dialog target and consequences,
   then change only `greaveselliott/MARS` from private to public. Immediately
   enable and verify CodeQL default setup, secret scanning, repository push
   protection, private vulnerability reporting, and Pages from `main:/docs`.
   Revalidate ruleset `21491158`, Actions restrictions, secrets/variables,
   Discussions, and anonymous source visibility before continuing.
4. **Immutable `v0.69.0`:** create the exact annotated tag only after the
   version commit and hosted source gates pass. Require the workflow to produce,
   attest, independently verify, and publish exactly ten assets. Verify the
   Release, immutable state, asset digests, attestation, anonymous download,
   install, and startup before continuing.
5. **Immutable `v0.69.1`:** land the evidence/patch release commit, create the
   exact annotated tag, and repeat the same exact-ten verification. Require
   `v0.69.1` to be latest and retain `v0.69.0` only as its tested rollback
   bridge. Verify anonymous update to `v0.69.1` and rollback to `v0.69.0`.
6. **Handoff:** record the exact postcondition receipt and hand off to T-081 for
   the 48-hour canary. Do not announce before that canary passes.

## Stop and Recovery Boundaries

- Before visibility, any mismatch stops with the repository still private.
- After visibility but before a launch tag, keep the repository public, repair
  the failed public control, and do not publish a Release.
- After an immutable Release is published, never delete, replace, retag, or
  overwrite it. A correction is `v0.69.2` or later.
- If `v0.69.0` fails verification before publication, stop and repair before a
  Release becomes immutable. If failure is discovered after publication, mark
  it unsupported without mutation and repair forward.
- Never expose credentials in logs, inputs, artifacts, reports, or chat. Any
  secret finding stops the transaction and requires rotation outside MARS.

## References

- [GitHub repository visibility consequences](https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/managing-repository-settings/setting-repository-visibility)
- [GitHub forks](https://docs.github.com/en/pull-requests/reference/forks)
- [GitHub artifact attestations](https://docs.github.com/en/actions/how-tos/secure-your-work/use-artifact-attestations/use-artifact-attestations)
- [GitHub immutable releases](https://docs.github.com/en/code-security/concepts/supply-chain-security/immutable-releases)
- [GitHub code scanning default setup](https://docs.github.com/en/code-security/how-tos/find-and-fix-code-vulnerabilities/configure-code-scanning/configure-code-scanning)
- [GitHub secret scanning](https://docs.github.com/en/code-security/how-tos/secure-your-secrets/detect-secret-leaks/enable-secret-scanning)
- [GitHub push protection](https://docs.github.com/en/code-security/concepts/secret-security/push-protection)
- [GitHub private vulnerability reporting](https://docs.github.com/en/code-security/how-tos/report-and-fix-vulnerabilities/configure-vulnerability-reporting/configure-for-a-repository)
- [GitHub Pages publishing source](https://docs.github.com/en/pages/getting-started-with-github-pages/configuring-a-publishing-source-for-your-github-pages-site)
