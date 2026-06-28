# Reference: Midscene For Dogfood Visual Validation

**Source:** [web-infra-dev/midscene](https://github.com/web-infra-dev/midscene), [midscene-skills](https://github.com/web-infra-dev/midscene-skills), [Midscene Web API reference](https://github.com/web-infra-dev/midscene/blob/main/apps/site/docs/en/web-api-reference.mdx)
**Source verified:** 2026-06-13
**Type:** Tool evaluation

## Summary

Midscene is an open source, vision-driven UI automation framework. Its core
value for MARS is screenshot-first browser validation: it can interact
with and assert rendered UI state without depending only on DOM selectors,
accessibility trees, or HTTP reachability. That maps directly to a recurring
Dogfood gap for browser games, canvas surfaces, custom controls, cross-origin
iframes, and visually meaningful UI states where `curl`, build output, and
source/runtime assertions are necessary but not sufficient.

Midscene should not become a required MARS runtime dependency. It is a
candidate optional validation lane for Dogfood and QA when a target is a visual
browser app and a suitable vision model/toolchain is configured.

## Fit For MARS

Midscene is useful when Dogfood needs to answer "can a user see and operate the
product?" rather than only "does a route return 200?" Strong candidate cases:

- browser games and canvas apps that need mounted-state evidence;
- icon-only controls, custom widgets, menus, and drag/drop flows;
- layout, visual state, color, highlight, and selection assertions;
- local browser smoke tests that should produce screenshots and replayable
  reports;
- optional agent/MCP-driven UI exploration when deterministic checks have
  already passed.

Midscene is a poor fit for baseline validation when:

- the target is a CLI/API/service with no meaningful visual surface;
- a deterministic Playwright, unit, or integration test already covers the
  scenario better;
- no vision-capable model endpoint is available;
- validation would require external credentials, production data, or browsing
  arbitrary public sites;
- the tool would install or persist Node artifacts without workspace hygiene.

## Ephemeral External Runtime Principle

The canonical design decision is
[ephemeral-validation-runtimes.md](../design-docs/ephemeral-validation-runtimes.md)
AD-293. The single-binary tenet forbids making Node, npm, browser automation
servers, or other external runtimes part of the installed MARS control
plane. It does not forbid temporary validation runtimes when they are explicitly
bounded.

Required cleanup checks for any Midscene lane:

- run only through a first-class tool or a documented `dependency_sync` /
  `shell_exec` path, never ad hoc package-manager commands;
- preflight `workspace_hygiene` before dependency setup;
- keep dependency caches, reports, screenshots, and temporary configs in
  ignored or `/tmp` paths unless a target-owned test suite intentionally tracks
  them;
- start browser/dev-server processes with managed background tracking;
- stop tracked PIDs and verify no child server/browser process remains;
- run postflight `workspace_hygiene` and `git_status`;
- fail as foundation/runtime evidence, not target product backlog, when the
  validation tool leaks files, leaves ports occupied, or cannot clean up.

## Recommended Adoption Path

1. Run a source-only canary against a clean visual browser target from the
   validation matrix. Prefer a canvas/game/static app where existing Dogfood
   evidence is weakest.
2. Record the exact Midscene package version, model family, redacted model
   endpoint shape, command, screenshots/report path, cleanup evidence, and
   stability across at least two reruns.
3. Treat failures as evidence until ownership is clear. Product rendering
   defects belong to the target; Midscene setup, model, cleanup, prompt, or
   tool-policy failures belong to the foundation.
4. If the canary is stable, create a first-class optional tool such as
   `browser_visual_smoke` through `tool_create`.
5. Expose the tool only to Dogfood/QA roles and only as an optional lane when
   Midscene configuration is present.
6. Update generated Dogfood guidance to say visual browser apps may use the
   optional visual-smoke lane after build/start evidence, but releases must not
   depend on Midscene unless the target has deliberately adopted it.

## Decision

Classify this as `mirrored doctrine candidate`: the cleanup principle should
eventually apply to both foundation and deployed harnesses, but the specific
Midscene integration must prove itself in a source canary before generated
targets inherit new defaults.

Until that canary passes, Dogfood should continue to require real build/start
evidence plus a browser-product smoke or source/runtime assertion for visual
browser targets. Midscene may be used manually as supplemental evidence, but it
is not yet a release-blocking gate.
