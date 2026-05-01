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
5. Push `main`

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
