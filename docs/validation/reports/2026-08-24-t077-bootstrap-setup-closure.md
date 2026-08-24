# T-077 Bootstrap And Setup Closure

- Date: 2026-08-24
- Ticket: T-077
- Scenario: F-017-S003 (private supporting evidence)
- Source commit: `56b8de336cf4d1439944cc7eb8ea0f5ad4043f2b`
- Implementation commit: `85c689c70ef801a2747acabf537739c9ebad3c12`
- Checkpoint evidence commit: `3e5125c`
- Corrective fixture commit: `56b8de336cf4d1439944cc7eb8ea0f5ad4043f2b`
- Status: `passed_t077_private_bootstrap_and_setup`
- Primary Status: `primary_blocked`
- Publication authority: denied

## Primary Outcome Contract

**Primary Outcome:** publish MARS as a supported open-source project without
exposing confidential material, weakening controls, or distributing unsafe or
unverifiable binaries.

**Primary Pass Gate:** the repository is public; signed `v0.69.1` is supported
with signed `v0.69.0` retained only as a rollback bridge; all F-017 scenarios,
logged-out macOS/Linux lifecycle checks, fork controls, GitHub security and
community surfaces, and the 48-hour canary pass.

**Primary Status:** `primary_blocked`

**Current Primary Blocker:** T-073 still requires qualified trademark
counsel's written MARS-name disposition and the owner's signed authority and
history attestation. GitHub-hosted CI is additionally blocked before step one
by the account Billing & plans condition recorded below.

**Next Primary Action:** execute current T-078's local producer/provenance,
exact-ten rehearsal, and hosted-delta acquisition checkpoints without tags,
publication, visibility mutation, destructive hosted cleanup, or immutable-
Release setting mutation absent its required exact owner approval.

**Supporting Evidence:** the exact-source cross-build matrix, clean-HOME native
macOS and Linux source/setup lanes, implementation and corrective-fixture
commits, frozen installer hashes, and the independently reviewed cleanup plan
and verified cleanup record in this report.

## Outcome

T-077 passes its private implementation and native source/setup boundary. Four
CGO-disabled Darwin/Linux AMD64/ARM64 binaries were built from the exact clean
source commit with Go 1.26.5, `-trimpath`, and matching unmodified VCS metadata.
A clean-HOME macOS arm64 install/setup lane and a native Linux arm64 non-root
install/setup lane both completed twice without GitHub credentials or a local
fallback token, with GitHub CLI and Go absent from the setup subprocess PATH
and environment. Neither lane acquired a llama-server or model. The first setup
ran the four deferred-inference configuration
steps; the second skipped all four idempotently. The Linux lane also ran the
hostile installer suite under GNU `stat`.

This is not proof of a published release. The repository remained private at
`VERSION=0.68.49`; the source fallback remained `0.69.0-dev`; no tag points at
the validation commit; and no tag, Release, signature, upload, hosted setting,
visibility, or announcement changed. The real official-tag Go proxy/SumDB,
signed archive install, update, and rollback lifecycle remains assigned to
T-080/T-081 after T-078 production admission and the T-073 owner/legal gate.
F-017-S003 therefore remains incomplete.

The independently frozen implementation boundary is:

- `scripts/install.sh` SHA-256
  `87c2bc1d9769d1cb9f121e34b706dde25e02f6ede5e35380fc07ffb1fa042192`;
- `internal/selfupdate/install_script_test.go` SHA-256
  `d055e830091fea197549756c02248385182290088a5ad06ced4fbe848e962911`.

The direct installer starts with privileged Bash, so imported functions and
`BASH_ENV` cannot execute before sanitization. Optional GitHub credentials are
snapshotted only in the privileged parent, kept out of `env(1)` arguments,
passed to the clean child over descriptors 3 and 4, and read into unexported
variables. Plain `exec 3<&- 4<&-` then closes both descriptors persistently and
the child verifies their absence before any external command. Go receives
neither credential; only the staged signed-updater invocation receives the
values through its command environment. Hostile-startup and token-canary tests
prove the values do not appear in output or logs.

## Matrix And Source Boundary

The source checkout, `HEAD`, and `origin/main` were all
`56b8de336cf4d1439944cc7eb8ea0f5ad4043f2b` before the accepted lanes. The
exact Go executable was `<private-go-toolchain>/bin/go`, mechanically resolved
to the standard exact `go1.26.5` toolchain module on the host without
publishing its owner-local absolute path.
Accepted builds used a clean environment, the public Go proxy and public
SumDB, `CGO_ENABLED=0`, `-trimpath`, and no module replacement. Their standard
Go build metadata reported Go 1.26.5, canonical command/module paths, the exact
source revision, and `vcs.modified=false`.

| Case | Result | SHA-256 | Accepted build facts |
| --- | --- | --- | --- |
| Darwin AMD64 | pass | `3bc8c4a05a634eecbe43a9a197f6d0c5ff41c1565eef775cbda1019e21a4a115` | Go 1.26.5; CGO disabled; `darwin/amd64`; clean exact VCS revision |
| Darwin arm64 | pass | `74ed9ebb72e681718e45367efbf4f228d9adc54ff5b25467202b61bd29e93317` | Go 1.26.5; CGO disabled; `darwin/arm64`; clean exact VCS revision |
| Linux AMD64 | pass | `23ed28fc68f78d9ff76032b29e4d8e84efcaa8647a7f28ae0cf8136cfbe940de` | Go 1.26.5; CGO disabled; `linux/amd64`; clean exact VCS revision |
| Linux arm64 | pass | `1365778e87c044dd9d99eac9734b12a101ad50543d4e10f67827e9a6c915c17e` | Go 1.26.5; CGO disabled; `linux/arm64`; clean exact VCS revision |

The representative build form was:

```text
env -i HOME=<private-home> PATH=<go-bin>:/usr/bin:/bin \
  GOPROXY=https://proxy.golang.org GOSUMDB=sum.golang.org \
  GOPRIVATE= GONOPROXY=none GONOSUMDB=none GOAUTH=off \
  GOTOOLCHAIN=local CGO_ENABLED=0 GOOS=<os> GOARCH=<arch> \
  <go1.26.5> build -trimpath -o <private-output> ./cmd/mars
```

The private validation root was
`/private/tmp/mars-t077-close.MdTdWHfd`. It contained only reproducible build,
clean-home, and setup artifacts, was retained through report review, and was
then removed with absence verified. No model or LLM lifecycle ran; model
identity is not applicable.

## Clean macOS Arm64 Lane

The native host was macOS 26.3.1 arm64 with Bash 3.2.57. The accepted installed
binary SHA-256 was
`44415f661a41c4c1c270730382fa06011497169e1d902592d2e57d6682d0880f`.
Build metadata reported Go 1.26.5, the canonical module/command, CGO disabled,
`darwin/arm64`, the exact source revision, and `vcs.modified=false`.

The lane used clean home `<validation-root>/macos-home`. The setup subprocess
had `GH_TOKEN`, `GITHUB_TOKEN`, and MARS token variables unset; its PATH and
environment exposed neither GitHub CLI nor Go; and canonical and legacy config
were absent before the run. It ran:

```text
mars setup --inference defer --skip-download --skip-github --yes --plain \
  --json --install-dir <clean-bin>
```

The first JSON result was `status=ok`, `steps_run=4`, `steps_skipped=0`,
`artifacts=[]`, and `total_bytes=0`. The second was `status=ok`, `steps_run=0`,
`steps_skipped=4`, `artifacts=[]`, and `total_bytes=0`. The canonical config was
mode `0600` with an empty `github_token`; no legacy config, llama-server, or
model appeared; and the shell path marker occurred exactly once.

## Clean Linux Arm64 Lane

The accepted native-architecture container used official
`golang:1.26.5-bookworm` at digest
`sha256:53eeac89074db483fdf0ab3be1df32bf6e47562263d2d0d6baa7f26acb4957dd`
on Docker Server 20.10.23. It ran as UID 1000, mounted the source read-only,
used an internal clean `/tmp/mars-t077` boundary, and contained no Git
credentials or `http.extraheader` configuration.

After QA rejected the first ephemeral lane as non-recomputable, the accepted
closure rerun mounted a bounded host evidence directory and retained the exact
installed binary, SHA-256, complete `go version -m`, installer-test transcript,
both setup JSON documents and empty stderr files, the command boundary, and a
postcondition manifest. The source clone was clean at the exact revision; its
Git config contained no credential or `extraheader` entry. Those files remained
outside the repository through independent review and were then removed
as recorded below.

Before setup, the native Linux lane passed:

```text
go test ./internal/selfupdate -run TestInstallScript -count=1
```

This exercised the installer contract with GNU `stat`. The accepted installed
binary SHA-256 was
`ddf7ee7805c8086e445e12bf8be468bbba73e704f31ace4819f99b5ca15c0099`.
Its build metadata reported Go 1.26.5, the canonical module/command, CGO
disabled, `linux/arm64`, the exact source revision, and
`vcs.modified=false`.

The setup subprocess had GitHub/MARS token variables unset, exposed neither
GitHub CLI nor Go through its PATH or environment, and had no pre-existing
config. The first deferred setup returned `status=ok`,
`steps_run=4`, `steps_skipped=0`, `artifacts=[]`, and `total_bytes=0`; the
second returned `status=ok`, `steps_run=0`, `steps_skipped=4`, `artifacts=[]`,
and `total_bytes=0`. Config mode was `0600`, `github_token` was empty, the
legacy config was absent, no llama-server or model appeared, and the shell path
marker occurred once. Automatic Linux llama.cpp acquisition remained disabled;
deferred inference was the deliberate supported lane. The successful container
used `--rm` and left no running validation container; the evidence mount
survived separately for review. The exact source-install command was
`GOBIN=<clean-bin> go install ./cmd/mars`; the setup subprocess used
`SHELL=/bin/bash` and only `<clean-bin>:/usr/bin:/bin` as PATH. The retained
binary and recorded SHA-256 both reproduce the `ddf7ee…` value above.

## Rejected Or Superseded Attempts

- The first host build used the installed base Go 1.26.2. It was rejected as
  evidence and superseded by all four exact Go 1.26.5 builds above.
- A Linux AMD64 Docker lane under arm64 QEMU failed inside Go's TLS certificate
  parsing with a runtime `growslice` panic. It did not become accepted product
  evidence; the native Linux arm64 lane superseded that emulation-only
  infrastructure failure.
- A root container invocation failed the unsafe-temporary-root fixture because
  root-owned sticky system temporary directories are deliberately admitted.
  The required representative non-root lane then passed.
- The first successful non-root Linux lane used an automatically removed
  container but retained no binary or transcript. QA rejected that evidence
  boundary. The closure rerun above reproduced the exact installed-binary hash
  and setup outcomes while retaining a bounded host-mounted evidence set.
- The closure rerun's first postcondition probe used an incorrect profile-marker
  string after all build and setup operations had passed, so it was not accepted
  as closure evidence. A fresh clean source-install/setup rerun retained its
  exact `ddf7ee…` binary, used the supported `SHELL=/bin/bash` boundary,
  recorded the correct marker count, and passed every postcondition.
- Initial public module resolution was blocked by the command sandbox's DNS
  boundary. The same exact public proxy/SumDB build was rerun with approved
  network access and passed.

## Failure Ownership And Hosted Blocker

The pre-existing nine-test `internal/agent` failure was foundation test-fixture
drift following T-075's intentional fail-closed Git-authoritative scan. Commit
`56b8de3` seeds deterministic clean Git fixtures without weakening production
policy; the exact nine tests and the full package passed normally and under the
race detector. The rejected Docker AMD64 attempt is emulation infrastructure,
not MARS behavior. Neither invalidates the accepted native lanes.

GitHub-hosted workflows are externally blocked before step one. Runs
`32676253403`, `32676429256`, and `32676702467` report:
"The job was not started because recent account payments have failed or your
spending limit needs to be increased." The latest validation source run is
`32676702467` (job `97286058559`). This does not replace or invalidate the
local T-077 evidence, but hosted CI and T-078 release-workflow proof cannot pass
until the owner resolves the GitHub Billing & plans condition and reruns the
exact source commit.

## Cleanup And Handoff

The successful Docker container was automatically removed and no validation
container remained. After QA and Security bound the accepted hashes and
postconditions, the exact private temporary roots
`/private/tmp/mars-t077-close.MdTdWHfd` and
`/private/tmp/mars-t077-builds.3lSWPJ7l`, plus the two bounded Linux review
roots, were removed and all four paths were verified absent. The removed
temporary binaries, caches, clean homes, source clone, and transcripts are not
recoverable from those roots; the durable report retains their reviewed hashes,
commands, outcomes, and rejected-attempt classification so the lanes can be
reproduced. The owner-local Linux review paths are deliberately not published
in this public-destined report. The official Docker image may remain in the
local cache and is not part of the evidence boundary.

T-077 closes with independent review and verified temporary-root cleanup.
T-078 must next admit and prove the pinned producer/signing contract, refresh
and rescan the stale workflow-run cleanup set, and retain destructive hosted
cleanup behind exact owner approval. T-073 remains the external owner/legal
gate. T-080 and T-081 retain real signed release and public lifecycle proof.
