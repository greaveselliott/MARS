# Mars Harness — Product Vision

**Status:** Accepted
**Date:** 2026-04-11

## What this is

A self-hosted autonomous AI delivery system. You provide a machine with a GPU, run one command, and it autonomously manages your development pipeline: CI diagnosis and repair, code generation from tickets, PR review, release management, documentation maintenance. All inference runs locally on open models. No cloud API costs, no data exfiltration, no vendor lock-in.

Over time, the harness gets better: accuracy scores improve as prompts and guardrails are refined, interventions decrease as the system learns what the human used to do, and the roadmap advances as the pipeline executes tickets autonomously.

## The analogy

Like Jarvis in Iron Man: an AI system built by you, running in your lab, on your hardware, managing your systems autonomously. You provide strategic direction (exec plans, priorities, product decisions). The harness handles execution (integration, review, fix, merge, release). No S.H.I.E.L.D. dependency — no Cursor, no OpenAI billing, no cloud vendor lock-in.

## The Mars lineage

The Mars monorepo proved the pipeline model works:

- 11 autonomous roles (CEO, COO, CTO, Engineer, QA, Security, Dependency Manager, Release Manager, Dogfood Tester, Pipeline Fixer, PR Comment Fixer) running a full development lifecycle.
- Self-correcting CI: deterministic fixes before probabilistic (changeset generation, auto-merge, dependabot rebase).
- Intervention debt tracking: every manual step treated as a signal to automate.
- Documentation as system of record: AGENTS.md, design docs, exec plans, tickets as markdown.

Mars used Cursor Cloud Automations as the execution plane. Mars Harness replaces that with a self-hosted system. Mars becomes the first customer, not the only one.

## What this is not

- **Not a code completion tool.** This is a pipeline automation system, not an IDE plugin.
- **Not a hosted service.** It runs on your hardware. There is no SaaS version (yet).
- **Not Mars-specific.** The harness is repo-agnostic. Mars's 11 roles are the seed content; users define their own roles, guardrails, and triggers.
- **Not a replacement for CI.** The harness works with GitHub Actions (or any CI), not instead of it. It reacts to CI events and produces PRs that CI then validates.

## Who this is for

Developers and teams who:

- Want autonomous AI managing their development pipeline
- Care about data sovereignty (code never leaves their infrastructure)
- Want to eliminate per-token API costs
- Have a machine with a GPU (consumer or workstation grade)
- Value transparency and auditability over black-box convenience

## The pitch (one paragraph)

Stop paying cloud providers per token to run AI agents on your code. Mars Harness runs on your own hardware with open models, autonomously managing your development pipeline — from ticket to merged PR. It scores its own accuracy, learns from its mistakes, and gets better over time. One command to set up. Zero ongoing costs beyond electricity.
