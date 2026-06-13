import assert from "node:assert/strict";
import test from "node:test";
import {
  buildRunId,
  defaultValidationRoot,
  parseProfile,
  slugify,
} from "./validation-target.mjs";

test("parseProfile reads frontmatter and body", () => {
  const { meta, body } = parseProfile(`---
slug: inventory-api
archetype: api-service
title: Inventory API Demo
---

# Inventory API Demo

Brief text.
`);
  assert.equal(meta.slug, "inventory-api");
  assert.equal(meta.archetype, "api-service");
  assert.match(body, /^# Inventory API Demo/);
});

test("buildRunId includes profile and optional label", () => {
  const date = new Date("2026-06-13T15:30:45Z");
  assert.equal(
    buildRunId("inventory-api", "wsd-closure", date),
    "run-20260613-153045-inventory-api-wsd-closure",
  );
  assert.equal(buildRunId("static-browser-todo", "", date), "run-20260613-153045-static-browser-todo");
});

test("slugify normalizes labels", () => {
  assert.equal(slugify("WS-D Closure Replay"), "ws-d-closure-replay");
});

test("defaultValidationRoot honors MH_VALIDATION_ROOT", () => {
  const prev = process.env.MH_VALIDATION_ROOT;
  process.env.MH_VALIDATION_ROOT = "/tmp/custom-validation";
  assert.equal(defaultValidationRoot("/repo/mars-harness"), "/tmp/custom-validation");
  if (prev == null) delete process.env.MH_VALIDATION_ROOT;
  else process.env.MH_VALIDATION_ROOT = prev;
});
