# Release Blocker Ledger

**Status:** Evidence
**Extracted:** 2026-06-11
**Source:** docs/exec-plans/active/current-operating-plan.md Current Truth section (extracted verbatim under T-022 / tenet 9 context efficiency)

Per-version release publication and asset-verification blocker evidence for
`v0.21.0` through `v0.42.20`. New release blockers are recorded here; the
active plan keeps only a one-line pointer plus any currently unresolved
blocker that gates dispatch.

## Ledger

- `v0.21.0` release notes and tag were pushed on 2026-05-03, but
  `mars-harness release verify-assets --version v0.21.0` is blocked because
  GitHub returned `404 Not Found` for the tag release immediately after the
  tag push.
- `v0.23.0` release notes and tag were pushed on 2026-05-03 for `MH-047`, but
  `go run ./cmd/mars-harness release verify-assets --version v0.23.0` is
  blocked because GitHub returned `404 Not Found` for the tag release
  immediately after the tag push.
- `v0.36.4`, `v0.36.5`, `v0.36.6`, `v0.37.0`, `v0.38.0`, `v0.39.0`,
  `v0.40.0`, `v0.40.1`, `v0.41.0`, `v0.41.1`, `v0.41.2`, `v0.41.3`,
  `v0.41.4`, `v0.41.5`, `v0.41.6`, `v0.41.7`, `v0.41.8`, `v0.41.9`,
  `v0.41.10`, `v0.41.11`, `v0.41.12`, `v0.41.13`, `v0.41.14`,
  `v0.41.15`, `v0.41.16`, `v0.41.17`, `v0.41.18`, `v0.41.19`, `v0.41.20`,
  `v0.41.21`, `v0.41.22`, `v0.41.23`, `v0.41.24`, `v0.41.25`, `v0.41.26`,
  and `v0.41.27`
  release notes and tags were pushed on 2026-05-19, but CI and Release workflow
  jobs were not started because GitHub reported recent account payment failure
  or a spending-limit increase requirement.
  Notes-only GitHub Releases for `v0.36.4` through `v0.41.27` were created from
  the generated changelog entries on 2026-05-19 so the Releases page is no
  longer stale at `v0.36.3`. `mars-harness release verify-assets --version
  v0.41.27` is still blocked because the `v0.41.27` release is missing
  `mars-harness-linux-amd64`, `mars-harness-linux-arm64`,
  `mars-harness-darwin-amd64`, `mars-harness-darwin-arm64`, and
  `checksums.txt`; GitHub Actions Release run `26129676755` failed with
  "recent account payments have failed or your spending limit needs to be
  increased" before assets could be built. The `v0.41.27` main-branch CI run
  `26129592162` hit the same runner-start billing blocker. `v0.41.16`,
  `v0.41.17`, `v0.41.18`, `v0.41.19`, `v0.41.20`, `v0.41.21`, `v0.41.22`,
  `v0.41.23`, `v0.41.24`, `v0.41.25`, and `v0.41.26` have the same missing-asset blocker via runs
  `26126035892`, `26126035944`, `26126461151`, `26127153189`, `26127529878`,
  `26127808895`, `26128342280`, `26128605770`, `26128778584`, and
  `26129025297`, and `26129336033`.
- `v0.41.28` release notes and tag were pushed on 2026-05-19 for `T-011`, but
  `mars-harness release verify-assets --version v0.41.28` is blocked because
  GitHub returned `404 Not Found` for the tag release after Release workflow
  run `26130558543` failed with the same "recent account payments have failed
  or your spending limit needs to be increased" runner-start blocker before
  assets or a release object could be created.
- `v0.41.29` release notes and tag were pushed on 2026-05-19 for the
  Homebrew/install-doc correction, but `mars-harness release verify-assets
  --version v0.41.29` is blocked because the notes-only GitHub Release is
  missing `mars-harness-linux-amd64`, `mars-harness-linux-arm64`,
  `mars-harness-darwin-amd64`, `mars-harness-darwin-arm64`, and
  `checksums.txt`. Release workflow run `26130724103` and main-branch CI run
  `26130717374` failed with the same runner-start billing blocker before
  assets could be built.
- `v0.41.30` release notes and tag were pushed on 2026-05-19 for the
  dirty-target survey handoff fix. The tag workflow did not create the release
  object because Release workflow run `26132165422` failed before runner
  startup with GitHub's "recent account payments have failed or your spending
  limit needs to be increased" blocker, so a notes-only GitHub Release was
  created from the generated changelog entry. `mars-harness release
  verify-assets --version v0.41.30` is blocked because the notes-only release is
  missing `mars-harness-linux-amd64`, `mars-harness-linux-arm64`,
  `mars-harness-darwin-amd64`, `mars-harness-darwin-arm64`, and
  `checksums.txt`. Main-branch CI run `26132157714` hit the same runner-start
  billing blocker before tests could run on GitHub.
- `v0.41.31` release notes and tag were pushed on 2026-05-20 for the
  static-demo lifecycle stabilization. The Release workflow failed before
  producing assets via release run `26151264493` and tag-push run `26151243177`,
  so a notes-only GitHub Release was created from the generated changelog entry.
  `mars-harness release verify-assets --version v0.41.31` is blocked because
  the notes-only release is missing `mars-harness-linux-amd64`,
  `mars-harness-linux-arm64`, `mars-harness-darwin-amd64`,
  `mars-harness-darwin-arm64`, and `checksums.txt`. Main-branch CI run
  `26151228459` also failed before running repo tests on GitHub.
- `v0.41.32` release notes and tag were pushed on 2026-05-20 for the
  release-blocked dispatch-loop fix. The Release workflow failed before
  producing assets via release run `26152986765` and tag-push run
  `26152960011`, so a notes-only GitHub Release was created from the generated
  changelog entry. `mars-harness release verify-assets --version v0.41.32` is
  blocked because the notes-only release is missing `mars-harness-linux-amd64`,
  `mars-harness-linux-arm64`, `mars-harness-darwin-amd64`,
  `mars-harness-darwin-arm64`, and `checksums.txt`. Main-branch CI runs
  `26152911907` and `26152949174` also failed before running repo tests on
  GitHub.
- `v0.41.33` release notes and tag were pushed on 2026-05-20 for generic
  list-string argument normalization and validation-matrix doctrine. The
  Release workflow failed before producing assets via release run `26154560201`
  and tag-push run `26154538133`, so a notes-only GitHub Release was created
  from the generated changelog entry. `mars-harness release verify-assets
  --version v0.41.33` is blocked because the notes-only release is missing
  `mars-harness-linux-amd64`, `mars-harness-linux-arm64`,
  `mars-harness-darwin-amd64`, `mars-harness-darwin-arm64`, and
  `checksums.txt`. Main-branch CI runs `26154482134` and `26154521037` also
  failed before running repo tests on GitHub.
- `v0.41.34` release notes and tag were pushed on 2026-05-20 for deployed
  static app-root docsync auditing. A notes-only GitHub Release was created
  from the generated changelog entry. The release-triggered workflow failed
  before producing assets via run `26159856206`, and the tag-push Release
  workflow run `26159792017` failed before asset publication. `mars-harness
  release verify-assets --version v0.41.34` is blocked because the notes-only
  release is missing `mars-harness-linux-amd64`, `mars-harness-linux-arm64`,
  `mars-harness-darwin-amd64`, `mars-harness-darwin-arm64`, and
  `checksums.txt`. Main-branch CI runs `26159740804` and `26159784258` also
  failed before running repo tests on GitHub.
- `v0.42.0` release notes and tag were pushed on 2026-05-20 for Factory Pace
  quality-score export. A notes-only GitHub Release was created from the
  generated changelog entry. The release-triggered workflow failed before
  producing assets via run `26160547817`, and the tag-push Release workflow run
  `26160517742` failed before asset publication. `mars-harness release
  verify-assets --version v0.42.0` is blocked because the notes-only release is
  missing `mars-harness-linux-amd64`, `mars-harness-linux-arm64`,
  `mars-harness-darwin-amd64`, `mars-harness-darwin-arm64`, and
  `checksums.txt`. Main-branch CI runs `26160441827` and `26160496062` also
  failed before running repo tests on GitHub.
- `v0.42.1` release notes and tag were pushed on 2026-05-20 for scheduler
  duplicate-work suppression. A notes-only GitHub Release was created from the
  generated changelog entry. The release-triggered workflow failed before
  producing assets via run `26161650092`, and the tag-push Release workflow run
  `26161635964` failed before asset publication. `mars-harness release
  verify-assets --version v0.42.1` is blocked because the notes-only release is
  missing `mars-harness-linux-amd64`, `mars-harness-linux-arm64`,
  `mars-harness-darwin-amd64`, `mars-harness-darwin-arm64`, and
  `checksums.txt`. Main-branch CI runs `26161599109` and `26161621672` also
  failed before running repo tests on GitHub.
- `v0.42.2` release notes and tag were pushed on 2026-05-20 for bounded
  repo-local build artifact cleanup. A notes-only GitHub Release was created
  from the generated changelog entry. The release-triggered workflow failed
  before producing assets via run `26162740026`, and the tag-push Release
  workflow run `26162728208` failed before asset publication. `mars-harness
  release verify-assets --version v0.42.2` is blocked because the notes-only
  release is missing `mars-harness-linux-amd64`, `mars-harness-linux-arm64`,
  `mars-harness-darwin-amd64`, `mars-harness-darwin-arm64`, and
  `checksums.txt`. Main-branch CI runs `26162683182` and `26162714822` also
  failed before running repo tests on GitHub.
- `v0.42.3` release notes and tag were pushed on 2026-05-20 for canonical
  bootstrap feature-contract reuse. A notes-only GitHub Release was created
  from the generated changelog entry. The release-triggered workflow failed
  before producing assets via run `26163699850`, and the tag-push Release
  workflow run `26163661221` failed before asset publication. `mars-harness
  release verify-assets --version v0.42.3` is blocked because the notes-only
  release is missing `mars-harness-linux-amd64`, `mars-harness-linux-arm64`,
  `mars-harness-darwin-amd64`, `mars-harness-darwin-arm64`, and
  `checksums.txt`. Main-branch CI runs `26163611221` and `26163652078` also
  failed before running repo tests on GitHub.
- `v0.42.4` release notes and tag were pushed on 2026-05-20 for module-named
  build artifact cleanup. A notes-only GitHub Release was created from the
  generated changelog entry. The release-triggered workflow failed before
  producing assets via run `26165171985`, and the tag-push Release workflow run
  `26165154660` failed before asset publication. `mars-harness release
  verify-assets --version v0.42.4` is blocked because the notes-only release is
  missing `mars-harness-linux-amd64`, `mars-harness-linux-arm64`,
  `mars-harness-darwin-amd64`, `mars-harness-darwin-arm64`, and
  `checksums.txt`. Main-branch CI runs `26165079226` and `26165139112` also
  failed before running repo tests on GitHub.
- `v0.42.5` release notes and tag were pushed on 2026-05-20 for generated
  artifact cleanup hints. A notes-only GitHub Release was created from the
  generated changelog entry. The release-triggered workflow failed before
  producing assets via run `26166260868`, and the tag-push Release workflow run
  `26166239400` failed before asset publication. `mars-harness release
  verify-assets --version v0.42.5` is blocked because the notes-only release is
  missing `mars-harness-linux-amd64`, `mars-harness-linux-arm64`,
  `mars-harness-darwin-amd64`, `mars-harness-darwin-arm64`, and
  `checksums.txt`. Main-branch CI runs `26166129334` and `26166214988` also
  failed before running repo tests on GitHub.
- `v0.42.6` release notes and tag were pushed on 2026-05-20 for managed server
  validation hardening. A notes-only GitHub Release was created from the
  generated changelog entry. The release-triggered workflow failed before
  producing assets via run `26171023219`, and the tag-push Release workflow run
  `26170992204` failed before asset publication. `mars-harness release
  verify-assets --version v0.42.6` is blocked because the notes-only release is
  missing `mars-harness-linux-amd64`, `mars-harness-linux-arm64`,
  `mars-harness-darwin-amd64`, `mars-harness-darwin-arm64`, and
  `checksums.txt`. Main-branch CI runs `26170919394` and `26170977972` also
  failed before running repo tests on GitHub.
- `v0.42.7` release notes and tag were pushed on 2026-05-20 for repo-local
  validation binary prevention. A notes-only GitHub Release was created from the
  generated changelog entry. The release-triggered workflow failed before
  producing assets via run `26172414689`, and the tag-push Release workflow run
  `26172375090` failed before asset publication. `mars-harness release
  verify-assets --version v0.42.7` is blocked because the notes-only release is
  missing `mars-harness-linux-amd64`, `mars-harness-linux-arm64`,
  `mars-harness-darwin-amd64`, `mars-harness-darwin-arm64`, and
  `checksums.txt`. Main-branch CI runs `26172314220` and `26172360280` also
  failed before running repo tests on GitHub.
- `v0.42.8` release notes and tag were pushed on 2026-05-20 for bare port
  validation command rejection. A notes-only GitHub Release was created from the
  generated changelog entry. The release-triggered workflow failed before
  producing assets via run `26173496452`, and the tag-push Release workflow run
  `26173478220` failed before asset publication. `mars-harness release
  verify-assets --version v0.42.8` is blocked because the notes-only release is
  missing `mars-harness-linux-amd64`, `mars-harness-linux-arm64`,
  `mars-harness-darwin-amd64`, `mars-harness-darwin-arm64`, and
  `checksums.txt`. Main-branch CI runs `26173398213` and `26173463431` also
  failed before running repo tests on GitHub.
- `v0.42.9` release notes and tag were pushed on 2026-05-20 for scratch
  validation script prevention. A notes-only GitHub Release was created from
  the generated changelog entry. The release-triggered workflow failed before
  producing assets via run `26175039706`, and the tag-push Release workflow run
  `26174999000` failed before asset publication. `mars-harness release
  verify-assets --version v0.42.9` is blocked because the notes-only release is
  missing `mars-harness-linux-amd64`, `mars-harness-linux-arm64`,
  `mars-harness-darwin-amd64`, `mars-harness-darwin-arm64`, and
  `checksums.txt`. Main-branch CI runs `26174936601` and `26174987693` also
  failed before running repo tests on GitHub.
- `v0.42.10` release notes and tag were pushed on 2026-05-20 for background
  descendant cleanup. A notes-only GitHub Release was created from the generated
  changelog entry. The release-triggered workflow failed before producing
  assets via run `26176038467`, and the tag-push Release workflow run
  `26176025806` failed before asset publication. `mars-harness release
  verify-assets --version v0.42.10` is blocked because the notes-only release is
  missing `mars-harness-linux-amd64`, `mars-harness-linux-arm64`,
  `mars-harness-darwin-amd64`, `mars-harness-darwin-arm64`, and
  `checksums.txt`. Main-branch CI runs `26175947976` and `26176011152` were
  still queued at blocker-record time and had not run repo tests on GitHub.
- `v0.42.11` release notes and tag were pushed on 2026-05-20 for implicit Go
  build artifact prevention. A notes-only GitHub Release was created from the
  generated changelog entry. The release-triggered workflow failed before
  producing assets via run `26177423152`, and the tag-push Release workflow run
  `26177399129` failed before asset publication. `mars-harness release
  verify-assets --version v0.42.11` is blocked because the notes-only release is
  missing `mars-harness-linux-amd64`, `mars-harness-linux-arm64`,
  `mars-harness-darwin-amd64`, `mars-harness-darwin-arm64`, and
  `checksums.txt`. Main-branch CI runs `26177359092` and `26177391754` failed
  before running repo tests on GitHub.
- `v0.42.12` release notes and tag were pushed on 2026-05-20 for tracked
  background process tree cleanup. A notes-only GitHub Release was created from
  the generated changelog entry. The release-triggered workflow failed before
  producing assets via run `26178472121`, and the tag-push Release workflow run
  `26178462481` failed before asset publication. `mars-harness release
  verify-assets --version v0.42.12` is blocked because the notes-only release is
  missing `mars-harness-linux-amd64`, `mars-harness-linux-arm64`,
  `mars-harness-darwin-amd64`, `mars-harness-darwin-arm64`, and
  `checksums.txt`. Main-branch CI run `26178418597` failed before running repo
  tests on GitHub; release-note CI run `26178451590` was still queued at
  blocker-record time.
- `v0.42.13` release notes and tag were pushed on 2026-05-20 for no-op
  shell-call completion guidance. A notes-only GitHub Release was created from
  the generated changelog entry. The release-triggered workflow failed before
  producing assets via run `26179822054`, and the tag-push Release workflow run
  `26179799379` failed before asset publication. `mars-harness release
  verify-assets --version v0.42.13` is blocked because the notes-only release is
  missing `mars-harness-linux-amd64`, `mars-harness-linux-arm64`,
  `mars-harness-darwin-amd64`, `mars-harness-darwin-arm64`, and
  `checksums.txt`. Main-branch CI run `26179784299` failed before running repo
  tests on GitHub.
- `v0.42.14` release notes and tag were pushed on 2026-05-20 for DocSync
  handoff gates and timeout-policy telemetry classification. A notes-only
  GitHub Release was created from the generated changelog entry. The
  release-triggered workflow failed before producing assets via run
  `26181460762`, and the tag-push Release workflow run `26181415533` failed
  before asset publication. `mars-harness release verify-assets --version
  v0.42.14` is blocked because the notes-only release is missing
  `mars-harness-linux-amd64`, `mars-harness-linux-arm64`,
  `mars-harness-darwin-amd64`, `mars-harness-darwin-arm64`, and
  `checksums.txt`. Main-branch CI run `26181405499` failed before running repo
  tests on GitHub.
- `v0.42.15` release notes and tag were pushed on 2026-05-20 for the
  product-first validation loop stabilization. A notes-only GitHub Release was
  created from the generated changelog entry. The tag-push Release workflow run
  `26187543392` failed before asset publication because GitHub did not start
  hosted runners with the account billing/spending-limit blocker. `mars-harness
  release verify-assets --version v0.42.15` is blocked because the notes-only
  release is missing `mars-harness-linux-amd64`, `mars-harness-linux-arm64`,
  `mars-harness-darwin-amd64`, `mars-harness-darwin-arm64`, and
  `checksums.txt`.
- `v0.42.16` release notes and tag were pushed on 2026-05-20 for the dashboard
  control-plane documentation epic. The tag-push Release workflow run
  `26188732882` failed before creating the GitHub Release or publishing assets,
  and the main-branch CI run `26188726401` failed before running repo tests on
  GitHub. `mars-harness release verify-assets --version v0.42.16` is blocked
  because GitHub returned `404 Not Found` for the `v0.42.16` release object.
- `v0.42.17` release notes and tag were pushed on 2026-05-21 for the
  continuous factory-loop stabilization. The tag-push Release workflow run
  `26242138645` failed before creating release assets; the `linux/amd64`
  matrix job failed with no retrievable logs or steps, and the remaining matrix
  jobs were cancelled or skipped. Main-branch CI run `26242173189` also failed
  before exposing retrievable job logs. A notes-only GitHub Release was created
  from the generated changelog entry at
  `https://github.com/greaveselliott/mars-harness/releases/tag/v0.42.17`.
  The release-created workflow run `26242344544` then failed the same way
  before asset publication, and main-branch CI run `26242396980` failed before
  exposing retrievable job logs for the blocker-recording commit.
  `go run ./cmd/mars-harness release verify-assets --version v0.42.17` is
  blocked because the notes-only release is missing
  `mars-harness-linux-amd64`, `mars-harness-linux-arm64`,
  `mars-harness-darwin-amd64`, `mars-harness-darwin-arm64`, and
  `checksums.txt`.
- `v0.42.18` release notes and tag were pushed on 2026-05-21 for the
  guardrail repair-evidence stabilization. The tag-push Release workflow run
  `26247800820` failed before creating the GitHub Release or publishing assets:
  the `linux/arm64` matrix job failed with no retrievable steps, the remaining
  matrix jobs were cancelled, and the release job was skipped. Main-branch CI
  run `26247792812` also failed before exposing retrievable job logs. `gh run
  view --log-failed` returned `log not found` for Release workflow job
  `77250600906` and CI job `77250569043`. `go run ./cmd/mars-harness release
  verify-assets --version v0.42.18` is blocked because GitHub returned
  `404 Not Found` for the `v0.42.18` release object.
- `v0.42.19` release notes and tag were pushed on 2026-05-21 for the
  post-validation no-op convergence guard. After the GitHub Actions budget was
  restored, the tag-push Release workflow run `26249349959` was rerun and
  passed, publishing `mars-harness-linux-amd64`, `mars-harness-linux-arm64`,
  `mars-harness-darwin-amd64`, `mars-harness-darwin-arm64`, and
  `checksums.txt`. `go run ./cmd/mars-harness release verify-assets --version
  v0.42.19` passed against
  `https://github.com/greaveselliott/mars-harness/releases/tag/v0.42.19`.
  The same budget restoration exposed a real current-main CI failure in run
  `26249406435`: `internal/tools` raced while reading managed background
  `shell_exec` startup output during `go test ./... -race -count=1
  -coverprofile=coverage.out -covermode=atomic`.
- `v0.42.20` release notes and tag were pushed on 2026-05-21 for managed
  background output capture synchronization. The tag-push Release workflow run
  `26251082633` passed, publishing all four binary assets and `checksums.txt`.
  Main-branch CI run `26251070633` then failed in `internal/tools` because
  shell-command ticket lifecycle parsing lowercased `T-001`/`T-002` ticket file
  paths before reading frontmatter on Linux, allowing missing-evidence ticket
  done moves to escape only on case-sensitive filesystems.
- `v0.45.1`: the first live `mars-harness release audit --repo . --limit 5`
  run on 2026-06-11 (T-026, AD-282) reported `missing_release` — the tag is
  pushed but no GitHub Release object exists. Remediation: checkout the
  `v0.45.1` release-note commit on `main`, then run `mars-harness release
  publish-assets --repo . --version v0.45.1 --upload github`. Not executed
  from the `codex/main-lifecycle-stabilization-rebased` branch because release
  publication belongs to trunk state, not in-flight branch work.
