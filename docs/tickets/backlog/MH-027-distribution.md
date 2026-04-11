---
id: MH-027
title: Cross-compile releases curl installer Homebrew and GoReleaser
priority: medium
complexity: medium
source: delivery-schedule M10
created: 2026-04-11
---

# MH-027: Distribution — linux/darwin amd64/arm64, curl|sh, Homebrew, GitHub Releases

## Context

Adoption hinges on fast installs matching developer laptops and CI images. M10 delivers reproducible binaries and conventional install paths.

## Requirements

- GoReleaser configuration building `linux`/`darwin` × `amd64`/`arm64` with version, commit, date embedded via `-ldflags`
- GitHub Releases workflow attaching checksums (`checksums.txt`) and SBOM optional stub
- `curl | sh` installer script: pinned version default, verifies checksum against published file, supports `VERSION=` override
- Homebrew tap formula (third-party tap repo or inline in org) installing binary from release artifacts
- CI gate: release draft on tag `v*`; no manual binary uploads

## Acceptance Criteria

### Functional (happy path)
- [ ] Tagged release produces all four artifacts plus checksums verified locally
- [ ] Installer script installs to prefix and prints `mars-harness version` matching tag
- [ ] `brew install …/mars-harness` succeeds on Apple Silicon and amd64 macOS smoke VMs (documented matrix)

### Edge cases and negative paths
- [ ] Checksum mismatch aborts install with non-zero exit and no partial binary
- [ ] Unsupported platform shows friendly error listing supported triples
- [ ] Rate-limited GitHub API in CI retried per workflow best practices

### Non-goals
- [ ] Windows MSI
- [ ] Linux distro packages (deb/rpm) beyond future ticket

### Observability, docs, and regressions
- [ ] CI workflow tested via `act` or dry-run mode where possible; minimal live test on canary tag
- [ ] Docs: install paths, verifying signatures (checksums), uninstall notes
- [ ] Version subcommand prints build target OS/ARCH
