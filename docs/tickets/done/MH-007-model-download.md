---
id: MH-007
title: Implement model weight download with progress and caching
priority: high
complexity: medium
source: delivery-schedule M2.3
created: 2026-04-11
---

# MH-007: Implement model weight download with progress and caching

## Context

Models are 15-50 GB. The download experience must be reliable (resume on failure), transparent (progress bar), and efficient (cache and skip if already present).

Reference: [docs/design-docs/local-inference.md](../../design-docs/local-inference.md) (AD-008)

## Requirements

Implement `internal/models/download.go`:

- Download GGUF files from HuggingFace with progress bar (bytes downloaded / total, ETA)
- HTTP Range header for resume on interrupted downloads
- SHA256 checksum verification after download
- Cache: skip download if file exists and checksum matches
- Disk space check before download (fail early with actionable message if insufficient)

## Acceptance Criteria

### Functional
- [ ] Downloads a file from HuggingFace and saves to `~/.mars-harness/models/`
- [ ] Progress bar shows during download
- [ ] Checksum verified after download
- [ ] Cached file skipped on subsequent runs

### Edge cases
- [ ] Interrupted download resumes from where it left off
- [ ] Corrupted cached file (wrong checksum) triggers re-download
- [ ] Insufficient disk space detected before download starts with actionable error
- [ ] Network timeout returns descriptive error with retry suggestion

### Non-goals
- [ ] Model selection logic (MH-006)
- [ ] Serving the model (MH-008)

## Notes

Test with a small fixture file served by a mock HTTP server, not real 50 GB models.
