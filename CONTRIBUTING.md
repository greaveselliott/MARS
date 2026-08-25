# Contributing to MARS

## Getting Started

```bash
git clone https://github.com/greaveselliott/MARS.git
cd mars
go build ./cmd/mars
go test ./...
```

## Development Workflow

External contributions use a fork and pull request. Repository maintainers use
the documented trunk workflow and the explicit administrator bypass in the
main-branch ruleset.

1. Fork the repository and create a focused branch
2. Make one coherent change with tests
3. Run `go test ./...` and `go vet ./...`
4. Commit with a semantic message referencing the milestone if applicable and
   add a DCO trailer with `git commit --signoff`
5. Check the active execution plan before changing version or release state
6. During T-071 through T-079, including resumed T-058 corrections, retain the `0.68.49` version floor and do not generate a release-note commit. After separately approved public visibility, T-080 alone publishes attested `v0.69.0` and `v0.69.1`; evidence-only T-081 closeout retains `v0.69.1` unless a canary correction requires `v0.69.2`
7. Push the validated semantic checkpoint to `main`
8. During T-071 through T-079, run only the no-publish producer and verification rehearsal authorized by AD-315; `.github/workflows/release.yml` remains dormant until the protected public launch sequence
9. Record unresolved producer, consumer, signing, rehearsal, or cutover gates as blockers
10. Do not create or move a tag, GitHub Release, upload, attestation, announcement, or supported-release claim during T-071 through T-079; after separate owner visibility approval, T-080 alone creates the two public attested launch releases

Pull requests must pass the read-only source and DCO workflows and receive
CODEOWNERS approval. Fork workflows receive no secrets, write token, OIDC, or
release authority. See [GOVERNANCE.md](GOVERNANCE.md),
[CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md), [SUPPORT.md](SUPPORT.md), and
[SECURITY.md](SECURITY.md).

Outside an explicitly recorded transition exception, every non-release
semantic commit follows the versioning rule in `AGENTS.md`; release-note commits
do not run the generator again. The active launch transition exception above is
authoritative through T-079 and ends only in T-080's two-release sequence.
Missing production, verification, or
publication authority is a blocker, not permission to create a notes-only
fallback.

## Commit Messages

Follow conventional commits. When working from the delivery schedule, reference the milestone and task:

```
feat(agent): implement conversation loop (M1.3.1)
fix(llm): handle malformed tool call JSON (M1.3.6)
docs(design): record AD-004 synchronous agent loop
```

Every non-merge pull-request commit must include an author-matching DCO
trailer. Add it with `git commit --signoff`; this certifies that you have the
right to submit the contribution under the repository's Apache 2.0 license.
The required DCO check recognizes only GitHub's exact Dependabot actor,
author, committer, and `dependabot[bot] <support@github.com>` trailer as the
non-human exception. Human contributors cannot use that exception.

## Code Style

- `golangci-lint` with default configuration
- Table-driven tests
- Error messages must be actionable (state what went wrong and how to fix it)
- No `panic` in library code; return errors

## Documentation

Every architecture decision goes in `docs/design-docs/` and is indexed in `docs/design-docs/index.md`. Read the [documentation discipline](.cursor/rules/documentation-discipline.mdc) rule.

## License

By contributing, you agree that your contributions will be licensed under the Apache 2.0 License.
