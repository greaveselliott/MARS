# Contributing to Mars Harness

## Getting Started

```bash
git clone https://github.com/greaveselliott/mars-harness.git
cd mars-harness
go build ./cmd/mars-harness
go test ./...
```

## Development Workflow

1. Work directly on `main`
2. Make one coherent change with tests
3. Run `go test ./...` and `golangci-lint run`
4. Commit with a semantic message referencing the milestone if applicable
5. Run `mars-harness release notes --repo . --bump auto`
6. Verify the generated `VERSION`, `CHANGELOG.md`, and `internal/buildinfo/version.go` changes
7. Run `mars-harness release backfill-notes --repo . --check`; if it reports legacy entries, run `mars-harness release backfill-notes --repo .` and include those changelog corrections
8. Commit them as `release: notes X.Y.Z`
9. Push `main`
10. Publish or update GitHub Release `vX.Y.Z` with the generated changelog entry when GitHub release credentials are configured
11. Confirm `gh release view vX.Y.Z` succeeds; if the release object is missing after the tag workflow, create a notes-only release from the generated `CHANGELOG.md` entry for the existing tag
12. Run any repo-required asset backfill or verification and record missing assets as a blocker

Every non-release semantic commit must follow this versioning step. Release-note commits are the only exception: do not run the release generator again for a `release: notes X.Y.Z` commit.

If GitHub release publication or asset verification is unavailable, record the blocker explicitly before ending the task.

## Commit Messages

Follow conventional commits. When working from the delivery schedule, reference the milestone and task:

```
feat(agent): implement conversation loop (M1.3.1)
fix(llm): handle malformed tool call JSON (M1.3.6)
docs(design): record AD-004 synchronous agent loop
```

## Code Style

- `golangci-lint` with default configuration
- Table-driven tests
- Error messages must be actionable (state what went wrong and how to fix it)
- No `panic` in library code; return errors

## Documentation

Every architecture decision goes in `docs/design-docs/` and is indexed in `docs/design-docs/index.md`. Read the [documentation discipline](.cursor/rules/documentation-discipline.mdc) rule.

## License

By contributing, you agree that your contributions will be licensed under the Apache 2.0 License.
