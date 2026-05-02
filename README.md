# Mars Harness

A self-hosted autonomous AI delivery system. Run it on your own hardware with open models. No cloud API costs, no data exfiltration, no vendor lock-in.

You provide a machine with a GPU. Mars Harness autonomously manages your development pipeline: CI diagnosis and repair, code generation from tickets, trunk checks and review, release management, documentation maintenance. All inference runs locally.

**Status:** Under active development. See [delivery schedule](docs/exec-plans/active/delivery-schedule.md).

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
# Install
curl -sSL https://get.mars-harness.dev | sh

# Setup (auto-detects GPU, downloads pinned models, optional GitHub integration)
mars-harness setup

# Run a single role against a repo
mars-harness run pipeline-fixer --repo /path/to/your/repo

# Start the full autonomous pipeline
mars-harness serve
```

Generate semantic-versioned patch notes from commits:

```bash
mars-harness release notes --repo . --bump auto
```

For changes to this source repo and repos initialized by Mars Harness, that release command is part of the commit flow: every non-release semantic commit is followed by a generated `release: notes X.Y.Z` commit before `main` is pushed.

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
