---
id: MH-031
title: Publish release binary assets for installer
priority: high
complexity: medium
source: operator feedback 2026-05-02
created: 2026-05-02
kind: intervention-debt
work_type: intervention-debt
end_to_end_evidence: not_applicable
evidence_links:
  - "GOCACHE=<validation-root> go test ./internal/selfupdate ./cmd/mars-harness"
  - "GOCACHE=<validation-root> go test ./internal/scanner ./internal/docsconsistency"
  - "GOCACHE=<validation-root> go test -cover ./internal/selfupdate ./internal/scanner ./internal/docsconsistency ./internal/release ./cmd/mars-harness"
  - "GOCACHE=<validation-root> go test ./cmd/mars-harness ./internal/ui ./internal/docsconsistency ./internal/scanner ./internal/selfupdate ./internal/release ./internal/buildinfo"
  - "GOOS=linux GOARCH=amd64 CGO_ENABLED=0 GOCACHE=<validation-root> go build -o <validation-root> ./cmd/mars-harness"
  - "GOOS=linux GOARCH=arm64 CGO_ENABLED=0 GOCACHE=<validation-root> go test -c -o <validation-root> ./internal/ui"
  - "shasum -a 256 -c <validation-root>"
  - "gh release view v0.14.5 --repo greaveselliott/mars-harness --json tagName,url,assets,publishedAt,targetCommitish"
  - "gh release list --repo greaveselliott/mars-harness --limit 5"
verified_by: command
dedupe_key: "public-example"
metadata:
  role: "release"
  repo_id: "mars-harness"
  target: "distribution"
  category: "missing_release_assets"
  severity: "high"
---

# MH-031: Publish release binary assets for installer

## Context

The documented binary installer downloads release assets named `mars-harness-{os}-{arch}` plus `checksums.txt`. Current GitHub releases such as `v0.5.2` contain release notes but no binary assets. This makes the recommended install path fail and pushes operators into source-root workflows like:

```bash
cd /path/to/target-repo && go build ./cmd/mars-harness;
./mars-harness start --repo /path/to/target-repo
```

That workflow can run stale binaries and violates the Plug and Play promise.

## Requirements

- Verify whether the tag-triggered release workflow is firing when release notes are published.
- Ensure `linux/darwin` x `amd64/arm64` binaries and `checksums.txt` are attached to every GitHub Release.
- Make release publication create or push the matching tag in the way the workflow expects.
- Extend `mars-harness update tool` from Go-install source updates to checksum-verified release-asset updates once assets exist.
- Add a release verification step that fails when the latest release has no assets.
- Update docs if manual release publication must be replaced by tag-first publication.

## Affected Files

- `.github/workflows/release.yml`
- `scripts/install.sh`
- `docs/quickstart.md`
- `docs/design-docs/release-versioning.md`
- `docs/product-specs/product-surface.md`

## Acceptance Criteria

### Functional (happy path)

- [x] Latest release includes four binaries and `checksums.txt`.
- [x] `scripts/install.sh` succeeds against the latest release on supported macOS/Linux platforms.
- [x] `mars-harness update tool` can use release assets without requiring Go or a source checkout.
- [x] Release process documents whether `git tag && git push --tags` or `gh release create` is authoritative.

### Edge cases and negative paths

- [x] Installer fails with an actionable message when assets are missing.
- [x] Checksum mismatch still aborts without leaving a partial binary.
- [x] Release publication cannot silently succeed with notes only.

### Non-goals

- Homebrew tap automation beyond verifying the binary asset contract.

### Observability, docs, and regressions

- [x] Docs explain the source-development `make install` path separately from the binary-release install path.
- [x] A test or release-check script validates expected asset names before announcing a release.

## Progress Notes

2026-05-03:

- Implemented checksum-verified release-asset self-update as the default
  `mars-harness update tool` path, with `--source` and `--version main` retaining
  the Go-install source-development path.
- Added `mars-harness release verify-assets` to fail when a GitHub Release is
  missing any expected binary or `checksums.txt`.
- Updated the tag-triggered Release workflow to use portable checksums, verify
  all expected files before publication, and use the matching `CHANGELOG.md`
  entry as the release body.
- Confirmed the previously latest visible release `v0.12.1` has no assets:
  `gh release view --repo greaveselliott/mars-harness --json tagName,assets,url`
  returned `"assets":[]`.
- Pushed `main` through `release: notes 0.13.0` and pushed tag `v0.13.0` at the
  release-note commit.
- Blocker: after waiting, `gh release view v0.13.0 --repo
  greaveselliott/mars-harness` still returned `release not found`, and `gh run
  list --repo greaveselliott/mars-harness --workflow Release --limit 5` could not
  connect to `api.github.com`. `MH-031` remains in progress until GitHub release
  creation and asset verification can be observed.

2026-05-03 follow-up:

- Confirmed GitHub Releases `v0.14.0`, `v0.14.1`, and `v0.14.2` existed as
  notes-only releases with empty asset lists.
- Updated the source Release workflow so pushed tags remain authoritative, while
  newly published GitHub Releases and manual `workflow_dispatch` runs can
  backfill assets for an existing tag. Mirrored the notes-only release blocker
  into source docs and generated target guidance.
- Found and fixed a release-build blocker in `internal/ui`: Linux
  cross-compilation failed because Darwin-only terminal ioctl constants were
  used directly. Split the termios ioctl constants by platform and verified
  Linux `amd64`/`arm64` compilation.
- Pushed `main` through `release: notes 0.14.5` and pushed tag `v0.14.5`.
- Built release assets for `linux/darwin` x `amd64/arm64`, generated
  `checksums.txt`, and verified the checksums locally with `shasum -a 256 -c`.
- Published GitHub Release `v0.14.5` and uploaded
  `mars-harness-linux-amd64`, `mars-harness-linux-arm64`,
  `mars-harness-darwin-amd64`, `mars-harness-darwin-arm64`, and
  `checksums.txt`.
- `gh release view v0.14.5 --repo greaveselliott/mars-harness --json
  tagName,url,assets,publishedAt,targetCommitish` confirmed all five assets are
  uploaded, and `gh release list --repo greaveselliott/mars-harness --limit 5`
  showed `v0.14.5` as latest.
- Local `mars-harness release verify-assets --version v0.14.5` and
  `INSTALL_DIR=<validation-root> VERSION=v0.14.5 bash
  scripts/install.sh` were blocked by sandbox DNS failures resolving
  `api.github.com` and `github.com`; the installer failure remained actionable
  and the GitHub release metadata verified the asset contract.
