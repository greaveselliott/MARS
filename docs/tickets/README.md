# Tickets

Work items live as markdown files in this directory. The repo is the source of truth.

## Directory Structure

```
docs/tickets/
  backlog/       Tickets waiting to be picked up
  in-progress/   Tickets actively being worked on
  done/          Completed tickets committed directly to main
```

## Ticket Format

```markdown
---
id: MH-001
title: Short description
priority: high | medium | low
complexity: small | medium | large
source: delivery-schedule M1.3.1
created: 2026-04-11
---

# MH-001: Short description

## Context
[Link to the delivery schedule milestone and task]

## Requirements
[Specific implementation requirements]

## Affected Files
[File paths or packages]

## Acceptance Criteria

### Functional (happy path)
- [ ] Primary behaviour works as specified

### Edge cases and negative paths
- [ ] Each known failure mode has an explicit line
- [ ] Invalid input, error paths, boundary conditions covered

### Non-goals
- What this ticket does NOT do

### Observability, docs, and regressions
- [ ] Docs updated if behaviour changed
- [ ] Tests cover the new functionality

## Notes
[Implementation notes, discoveries]
```

## Naming Convention

`MH-NNN-short-description.md` where NNN is a zero-padded sequential number.

## Lifecycle

1. Ticket created in `backlog/`
2. Implementation starts: move to `in-progress/`
3. Implementation completes: move to `done/` in a direct semantic commit on `main`
4. Push `main` after the commit so the repo remains the system of record
