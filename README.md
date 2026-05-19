# Mars Harness

A self-hosted autonomous AI delivery system. Run it on your own hardware with open models. No cloud API costs, no data exfiltration, no vendor lock-in.

You provide a machine with a GPU. Mars Harness autonomously manages your development pipeline: CI diagnosis and repair, code generation from tickets, trunk checks and review, release management, documentation maintenance. All inference runs locally.

**Status:** Under active development. See the [current operating plan](docs/exec-plans/active/current-operating-plan.md).

## The Nine Tenets

1. **Plug and Play** — zero to running in one command, extends to full lifecycle
2. **Self-Improving System** — evolves from human interventions and its own failures
3. **Accuracy and Value Scoring** — per-role health scores from real outcomes
4. **Customisable Guardrails** — user-defined rules enforced during execution
5. **Roadmap from Init** — tickets and backlog deployed on day one
6. **Blast Radius Containment** — never cause irreversible damage
7. **Execution Truth and Transparency** — auditable, attributable, everything in git
8. **Progressive Autonomy** — earn trust, graduate from observer to autonomous
9. **Context Efficiency** — minimal context assembly, retrieval over stuffing

Full text: [docs/design-docs/tenets.md](docs/design-docs/tenets.md)

## Quick Start

```bash
# Install from published GitHub Release assets
curl -sSfL https://raw.githubusercontent.com/greaveselliott/mars-harness/main/scripts/install.sh | bash

# Private release auth and setup
mars-harness auth github setup
mars-harness setup
mars-harness doctor
mars-harness update tool

# Run a single role against a repo
mars-harness run pipeline-fixer --repo /path/to/your/repo

# Start the full autonomous pipeline
mars-harness start --repo /path/to/your/repo
```

Homebrew is not a supported install path yet; use the release installer above
or the source-development path below until a tap is published.

When working from a source checkout, install the dev binary once instead of running `./mars-harness` from the repo root:

```bash
cd /path/to/target-repo
make install
mars-harness start --repo /path/to/target-repo
```

Avoid `go build ./cmd/mars-harness; ./mars-harness ...`: the semicolon can run a stale old binary if the build fails.

Upgrade the installed command without changing directories:

```bash
mars-harness update check --repo /path/to/target-repo
mars-harness update tool
mars-harness update harness --repo /path/to/target-repo
```

`update tool` uses checksum-verified private GitHub Release assets by default.
Run `mars-harness auth github setup` once during getting started so update,
version-drift, and release-asset workflows can reuse the same auth model. The
resolver tries `GH_TOKEN`, `GITHUB_TOKEN`, GitHub CLI auth from `gh auth token`,
then an optional local token stored under `~/.mars-harness/`. Source development
channels remain available with `mars-harness update tool --source --version
main`.

Remove Mars Harness from a target repo with a dry-run kill switch:

```bash
mars-harness eject --repo /path/to/target-repo
mars-harness eject --repo /path/to/target-repo --apply --confirm sample-target
```

The apply path removes generated harness artifacts and the associated per-repo
SQLite database, but does not rewrite git history.

Generate semantic-versioned patch notes from commits:

```bash
mars-harness release notes --repo . --bump auto
```

For changes to this source repo and repos initialized by Mars Harness, that release command is part of the commit flow: every non-release semantic commit is followed by a generated `release: notes X.Y.Z` commit before `main` is pushed.
When GitHub release credentials are configured, push tag `vX.Y.Z` at the
release-note commit so the Release workflow publishes the changelog entry and
checksum-verified binaries. Verify it with `mars-harness release verify-assets
--version vX.Y.Z`.

## Lineage

Mars Harness is an evolution of the [Mars](https://github.com/elliottgreaves/mars) monorepo's automation pipeline. Mars proved the model works — 11 autonomous roles running a full development lifecycle via Cursor Automations. This product extracts that into a standalone system that runs on your own hardware against any repo.

## Documentation

- [AGENTS.md](AGENTS.md) — project guide for AI agents working in this repo
- [ARCHITECTURE.md](ARCHITECTURE.md) — system architecture
- [docs/design-docs/](docs/design-docs/) — architectural decisions
- [docs/exec-plans/](docs/exec-plans/) — delivery plans and trackers
- [docs/product-specs/index.md](docs/product-specs/index.md) — living product specs
- [docs/references/](docs/references/) — research findings and external sources

## License

Apache 2.0 — see [LICENSE](LICENSE).
