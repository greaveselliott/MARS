# Contributing to MARS

## Getting Started

```bash
git clone https://github.com/greaveselliott/MARS.git
cd mars
go build ./cmd/mars
go test ./...
```

## Development Workflow

1. Work directly on `main`
2. Make one coherent change with tests
3. Run `go test ./...` and `golangci-lint run`
4. Commit with a semantic message referencing the milestone if applicable
5. Check the active execution plan before changing version or release state
6. During T-065 through T-067, retain the `0.68.49` version floor and do not generate a release-note commit
7. Push the validated semantic checkpoint to `main`
8. Run only the exact pinned, publication-disabled snapshot and verification steps in `.github/workflows/release-snapshot.yml`
9. Record unresolved producer, consumer, signing, rehearsal, or cutover gates as blockers
10. Do not create or move a tag, GitHub Release, upload, signature, announcement, or supported-release claim without the separately approved F-017/F-018 cutover

Outside an explicitly recorded transition exception, every non-release
semantic commit follows the versioning rule in `AGENTS.md`; release-note commits
do not run the generator again. The active F-018 transition exception above is
authoritative until T-067 closes it. Missing production, verification, or
publication authority is a blocker, not permission to create a notes-only
fallback.

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
