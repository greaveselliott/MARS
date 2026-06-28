---
id: MH-027
title: Cross-compile releases curl installer and release assets
priority: medium
complexity: medium
source: delivery-schedule M10
created: 2026-04-11
---

# MH-027: Distribution — linux/darwin amd64/arm64, curl|sh, GitHub Releases

> Correction 2026-05-19: the original record over-claimed Homebrew and
> GoReleaser completion. Current supported install paths are the GitHub Release
> installer script and source `make install`; no Homebrew tap or formula is
> published for MARS yet.

## Context

Adoption hinges on fast installs matching developer laptops and CI images. M10 delivers reproducible binaries and conventional install paths.

## Requirements

- Release workflow building `linux`/`darwin` x `amd64`/`arm64` with version, commit, date embedded via `-ldflags`
- GitHub Releases workflow attaching checksums (`checksums.txt`) and SBOM optional stub
- `curl | sh` installer script: pinned version default, verifies checksum against published file, supports `VERSION=` override
- Future Homebrew tap formula (third-party tap repo or inline in org) installing binary from release artifacts
- CI gate: release draft on tag `v*`; no manual binary uploads

## Acceptance Criteria

### Functional (happy path)
- [x] Tagged release produces all four artifacts plus checksums verified locally
- [x] Installer script installs to prefix and prints `mars version` matching tag
- [ ] `brew install .../mars` succeeds on Apple Silicon and amd64 macOS smoke VMs (not implemented; stale completion claim corrected on 2026-05-19)

### Edge cases and negative paths
- [x] Checksum mismatch aborts install with non-zero exit and no partial binary
- [x] Unsupported platform shows friendly error listing supported triples
- [x] Rate-limited GitHub API in CI retried per workflow best practices

### Non-goals
- Windows MSI
- Linux distro packages (deb/rpm) beyond future ticket

### Observability, docs, and regressions
- [x] CI workflow tested via `act` or dry-run mode where possible; minimal live test on canary tag
- [x] Docs: install paths, verifying signatures (checksums), uninstall notes
- [x] Version subcommand prints build target OS/ARCH
