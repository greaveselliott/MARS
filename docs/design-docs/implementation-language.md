# Implementation Language

**Status:** Accepted
**Date:** 2026-06-14
**Author:** foundation-maintainer

## Context

MARS is a local autonomous delivery system, not a hosted SaaS control
plane. Its core job is to install cleanly, run on operator machines, supervise
subprocesses, manage repositories, persist queue and telemetry state, expose a
small dashboard, and remain inspectable by humans and other agents.

The language choice therefore has to optimize for operational distribution and
maintenance:

- a single cross-platform binary,
- no required runtime package manager,
- no database, queue, or web server dependency outside the process,
- predictable filesystem, process, signal, and network behavior,
- readable code that autonomous agents can safely modify,
- enough performance for local orchestration without turning inference into an
  in-process concern.

AD-001 records the source-level implementation decision. Related runtime
packaging decisions live in [local-inference.md](local-inference.md), including
AD-007, which keeps llama.cpp outside the Go binary as a supervised subprocess.

## Key Design Decisions

### AD-001: Go Is The Implementation Language

MARS is implemented in Go, with the main binary expected to build
without CGO for normal distribution.

Go is the best fit because the product is mostly an operations control plane:
CLI commands, long-running daemons, HTTP handlers, SQLite-backed local state,
subprocess supervision, filesystem and git orchestration, release packaging,
and small server-rendered UI surfaces. Go's standard library and deployment
model match those responsibilities with little extra machinery.

The choice deliberately keeps the heavy AI work out of process. The harness
does not try to embed model inference or GPU kernels in Go. It manages model
servers such as llama.cpp over process and HTTP boundaries, preserving a simple
binary while letting specialized native inference runtimes own accelerator
support.

## Why Go

**Single-binary distribution.** A self-hosted tool should be easy to copy,
install, update, and diagnose. Go's static binary path keeps the default
operator experience close to `mars setup` followed by normal CLI use.

**Low runtime dependency burden.** Python, Node.js, Postgres, Redis, and
external dashboard build chains are useful in other products, but they would
weaken the plug-and-play tenet here. Go lets the queue, server, CLI, scheduler,
embedded assets, and release tooling ship together.

**Strong fit for local orchestration.** MARS spends most of its time
coordinating processes, files, HTTP APIs, SQLite, signals, timers, and logs.
Those are ordinary Go strengths, and the code stays explicit enough for
operators and agents to inspect.

**Reasonable safety without high ceremony.** Rust would offer stronger memory
and type guarantees, but at noticeably higher implementation friction for the
agent-maintained codebase. Go's simpler type system, fast compiler, test tools,
and conventional package structure are a better fit for frequent autonomous
maintenance.

**Boring concurrency.** The queue, scheduler, server, subprocess supervisors,
and dashboard streams need concurrency, but not exotic concurrency. Goroutines,
contexts, channels, and `net/http` cover the default shape while keeping
failure handling visible.

**Readable agent-editable code.** MARS is meant to improve through
agent work. Go's formatting, explicit error returns, package conventions, and
standard testing flow reduce style variance and make mechanical edits easier to
review.

## Alternatives Considered

**Rust** would strengthen compile-time guarantees and produce excellent
binaries. It was rejected as the default implementation language because the
extra complexity would slow broad product iteration, make agent edits harder,
and add ceremony around integrations that are mostly process, file, and HTTP
orchestration.

**Python** would make early prototyping and AI ecosystem integration faster.
It was rejected because the product promise depends on a local binary with no
Python environment, virtualenv, pip, native wheel, or system package debugging
loop.

**TypeScript/Node.js** would speed dashboard and API development. It was
rejected for the core because it would introduce a required runtime and package
manager into the operator path. Node may still be appropriate for optional
development-only sidecars or generated UI experiments when production remains
served by the Go harness boundary.

**Java/Kotlin or C#** would provide mature service ecosystems, but they add a
larger runtime story than the project needs and do not improve the local
single-binary operator experience enough to justify that burden.

## Consequences

- Core extension points should preserve the CGO-free default unless a design
  decision explicitly accepts native build complexity.
- Heavy inference, GPU, and model-provider concerns stay behind subprocess,
  HTTP, or provider interfaces instead of becoming Go-native compute.
- Embedded dashboard work should remain compatible with single-binary
  distribution unless a design doc narrows the exception.
- New dependencies should be judged against the operator install path, not only
  developer convenience.
- Runtime errors should continue to use Go's explicit remediation style:
  actionable messages, concrete commands, and visible trace evidence.

## Discoveries

- **Go fits the product thesis because MARS is mostly control-plane
  software:** the hardest product constraints are packaging, orchestration,
  state, trust, and evidence, not writing inference kernels.
- **The inference boundary makes the language choice cleaner:** keeping
  llama.cpp out of process lets MARS stay portable while still using
  native accelerator-aware tooling.
