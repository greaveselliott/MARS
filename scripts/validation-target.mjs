#!/usr/bin/env node
/**
 * Ephemeral validation targets for foundation replays (AD-293).
 *
 * Creates fresh git + harness-init folders under a grouped parent directory,
 * tracks run metadata, and discards runs with their per-repo DBs when done.
 *
 * Usage:
 *   node scripts/validation-target.mjs create --profile inventory-api --label wsd-closure
 *   node scripts/validation-target.mjs list
 *   node scripts/validation-target.mjs show run-20260613-153045-inventory-api-wsd-closure
 *   node scripts/validation-target.mjs discard run-20260613-153045-inventory-api-wsd-closure
 *   node scripts/validation-target.mjs cleanup --keep 3
 *   node scripts/validation-target.mjs profiles
 *
 * Environment:
 *   MH_VALIDATION_ROOT — parent directory (default: ../demo/validation-runs from repo root)
 */
import { execSync, spawnSync } from "node:child_process";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const HARNESS_ROOT = path.resolve(__dirname, "..");
const PROFILES_DIR = path.join(HARNESS_ROOT, "docs/validation/profiles");
const DISCARDED_DIRNAME = ".discarded";

export function parseProfile(content) {
  const match = content.match(/^---\r?\n([\s\S]*?)\r?\n---\r?\n([\s\S]*)$/);
  if (!match) {
    throw new Error("profile must start with YAML frontmatter delimited by ---");
  }
  const meta = {};
  for (const line of match[1].split("\n")) {
    const trimmed = line.trim();
    if (!trimmed || trimmed.startsWith("#")) continue;
    const idx = trimmed.indexOf(":");
    if (idx === -1) continue;
    meta[trimmed.slice(0, idx).trim()] = trimmed.slice(idx + 1).trim();
  }
  if (!meta.slug) throw new Error("profile frontmatter requires slug");
  if (!meta.archetype) throw new Error("profile frontmatter requires archetype");
  return { meta, body: match[2].trimStart() };
}

export function loadProfile(slug) {
  const filePath = path.join(PROFILES_DIR, `${slug}.md`);
  if (!fs.existsSync(filePath)) {
    throw new Error(`unknown profile "${slug}" — run "profiles" for the list`);
  }
  const parsed = parseProfile(fs.readFileSync(filePath, "utf8"));
  if (parsed.meta.slug !== slug) {
    throw new Error(`profile file ${slug}.md declares slug ${parsed.meta.slug}`);
  }
  return { filePath, ...parsed };
}

export function defaultValidationRoot(harnessRoot = HARNESS_ROOT) {
  if (process.env.MH_VALIDATION_ROOT) {
    return path.resolve(process.env.MH_VALIDATION_ROOT);
  }
  return path.resolve(harnessRoot, "../demo/validation-runs");
}

export function formatRunStamp(date = new Date()) {
  const pad = (n) => String(n).padStart(2, "0");
  const day = `${date.getUTCFullYear()}${pad(date.getUTCMonth() + 1)}${pad(date.getUTCDate())}`;
  const time = `${pad(date.getUTCHours())}${pad(date.getUTCMinutes())}${pad(date.getUTCSeconds())}`;
  return `${day}-${time}`;
}

export function buildRunId(profileSlug, label, date = new Date()) {
  const stamp = formatRunStamp(date);
  const safeLabel = label ? slugify(label) : "";
  return safeLabel
    ? `run-${stamp}-${profileSlug}-${safeLabel}`
    : `run-${stamp}-${profileSlug}`;
}

export function readmeFromProfile(meta, body) {
  const title = meta.title || meta.slug;
  const content = body.trimStart();
  const withoutLeadingTitle = content.replace(/^#\s+.*(?:\r?\n)+/, "").trimStart();
  return `# ${title}\n\n${withoutLeadingTitle || content}`.trimEnd() + "\n";
}

export function slugify(value) {
  return value
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "")
    .slice(0, 48);
}

export function dbPathForRepoSlug(repoSlug) {
  return path.join(os.homedir(), ".mars-harness/db", repoSlug, "mars.db");
}

function run(cmd, opts = {}) {
  return execSync(cmd, { encoding: "utf8", stdio: ["pipe", "pipe", "pipe"], ...opts }).trim();
}

function tryRun(cmd) {
  try {
    return run(cmd);
  } catch {
    return null;
  }
}

function readManifest(runDir) {
  const manifestPath = path.join(runDir, ".validation-run.json");
  if (!fs.existsSync(manifestPath)) return null;
  return JSON.parse(fs.readFileSync(manifestPath, "utf8"));
}

function writeManifest(runDir, manifest) {
  fs.writeFileSync(path.join(runDir, ".validation-run.json"), JSON.stringify(manifest, null, 2) + "\n");
}

function listRunDirs(root) {
  if (!fs.existsSync(root)) return [];
  return fs
    .readdirSync(root, { withFileTypes: true })
    .filter((d) => d.isDirectory() && d.name.startsWith("run-") && !d.name.startsWith("."))
    .map((d) => path.join(root, d.name));
}

function listDiscardedManifests(root) {
  const discardedRoot = path.join(root, DISCARDED_DIRNAME);
  if (!fs.existsSync(discardedRoot)) return [];
  return fs
    .readdirSync(discardedRoot, { withFileTypes: true })
    .filter((d) => d.isDirectory())
    .map((d) => {
      const manifestPath = path.join(discardedRoot, d.name, "manifest.json");
      if (!fs.existsSync(manifestPath)) return null;
      return JSON.parse(fs.readFileSync(manifestPath, "utf8"));
    })
    .filter(Boolean);
}

function resolveRun(root, idOrPrefix) {
  const dirs = listRunDirs(root).filter((d) => path.basename(d).startsWith(idOrPrefix));
  if (dirs.length === 1) return dirs[0];
  if (dirs.length > 1) {
    throw new Error(`ambiguous run id "${idOrPrefix}" — matches: ${dirs.map((d) => path.basename(d)).join(", ")}`);
  }
  const exact = path.join(root, idOrPrefix);
  if (fs.existsSync(exact)) return exact;
  throw new Error(`run not found: ${idOrPrefix}`);
}

function removeDb(repoSlug) {
  const dbDir = path.dirname(dbPathForRepoSlug(repoSlug));
  if (fs.existsSync(dbDir)) {
    fs.rmSync(dbDir, { recursive: true, force: true });
  }
}

function createRun({ profile, label, root, skipInit }) {
  const { meta, body } = loadProfile(profile);
  const runId = buildRunId(meta.slug, label);
  const runDir = path.join(root, runId);
  if (fs.existsSync(runDir)) {
    throw new Error(`run directory already exists: ${runDir}`);
  }
  fs.mkdirSync(runDir, { recursive: true });

  run(`git init -b main`, { cwd: runDir });
  fs.writeFileSync(path.join(runDir, "spec.md"), body.endsWith("\n") ? body : `${body}\n`);
  fs.writeFileSync(path.join(runDir, "README.md"), readmeFromProfile(meta, body));
  run(`git add README.md spec.md`, { cwd: runDir });
  run(`git commit -m "chore: add validation brief"`, { cwd: runDir });

  if (!skipInit) {
    const init = spawnSync("mars-harness", ["init", "--repo", runDir], {
      encoding: "utf8",
      stdio: ["pipe", "pipe", "pipe"],
    });
    if (init.status !== 0) {
      fs.rmSync(runDir, { recursive: true, force: true });
      throw new Error(init.stderr || init.stdout || "mars-harness init failed");
    }
  }

  const manifest = {
    id: runId,
    profile: meta.slug,
    archetype: meta.archetype,
    title: meta.title ?? meta.slug,
    label: label ?? null,
    createdAt: new Date().toISOString(),
    path: runDir,
    repoSlug: runId,
    dbPath: dbPathForRepoSlug(runId),
    status: "active",
    marsHarnessVersion: tryRun("mars-harness --version"),
    harnessSourceRef: tryRun(`git -C ${JSON.stringify(HARNESS_ROOT)} rev-parse --short HEAD`),
  };
  writeManifest(runDir, manifest);

  return manifest;
}

function discardRun(root, idOrPrefix, { dryRun = false, purge = false } = {}) {
  const runDir = resolveRun(root, idOrPrefix);
  const manifest = readManifest(runDir) ?? {
    id: path.basename(runDir),
    repoSlug: path.basename(runDir),
    path: runDir,
    dbPath: dbPathForRepoSlug(path.basename(runDir)),
  };
  manifest.discardedAt = new Date().toISOString();
  manifest.status = purge ? "purged" : "discarded";

  if (dryRun) {
    return { manifest, dryRun: true };
  }

  removeDb(manifest.repoSlug ?? path.basename(runDir));

  if (purge) {
    fs.rmSync(runDir, { recursive: true, force: true });
    return manifest;
  }

  const discardedRoot = path.join(root, DISCARDED_DIRNAME, manifest.id);
  fs.mkdirSync(path.dirname(discardedRoot), { recursive: true });
  fs.mkdirSync(discardedRoot, { recursive: true });
  fs.writeFileSync(path.join(discardedRoot, "manifest.json"), JSON.stringify(manifest, null, 2) + "\n");
  fs.rmSync(runDir, { recursive: true, force: true });
  return manifest;
}

function cleanupRuns(root, { keep = 3, olderThanDays, dryRun = false } = {}) {
  const runs = listRunDirs(root)
    .map((dir) => readManifest(dir) ?? { id: path.basename(dir), path: dir, createdAt: null })
    .sort((a, b) => (b.createdAt ?? "").localeCompare(a.createdAt ?? ""));

  const toDiscard = [];
  if (olderThanDays != null) {
    const cutoff = Date.now() - olderThanDays * 24 * 60 * 60 * 1000;
    for (const run of runs) {
      const created = run.createdAt ? Date.parse(run.createdAt) : 0;
      if (created && created < cutoff) toDiscard.push(run);
    }
  } else {
    for (const run of runs.slice(keep)) toDiscard.push(run);
  }

  const results = [];
  for (const run of toDiscard) {
    results.push(discardRun(root, run.id, { dryRun }));
  }
  return results;
}

function printProfiles() {
  const files = fs.readdirSync(PROFILES_DIR).filter((f) => f.endsWith(".md") && f !== "README.md");
  for (const file of files.sort()) {
    const { meta } = parseProfile(fs.readFileSync(path.join(PROFILES_DIR, file), "utf8"));
    console.log(`${meta.slug.padEnd(28)} ${meta.archetype.padEnd(28)} ${meta.title ?? ""}`);
  }
}

function printList(root, { json = false } = {}) {
  const active = listRunDirs(root).map((dir) => readManifest(dir)).filter(Boolean);
  const discarded = listDiscardedManifests(root);
  const payload = { root, active, discarded };
  if (json) {
    console.log(JSON.stringify(payload, null, 2));
    return;
  }
  console.log(`Validation root: ${root}\n`);
  console.log("Active runs:");
  if (!active.length) console.log("  (none)");
  for (const run of active.sort((a, b) => b.createdAt.localeCompare(a.createdAt))) {
    console.log(`  ${run.id}`);
    console.log(`    profile=${run.profile} archetype=${run.archetype} created=${run.createdAt}`);
    console.log(`    path=${run.path}`);
    console.log(`    start: mars-harness start --repo ${run.path}`);
  }
  console.log("\nDiscarded runs:");
  if (!discarded.length) console.log("  (none)");
  for (const run of discarded.sort((a, b) => (b.discardedAt ?? "").localeCompare(a.discardedAt ?? ""))) {
    console.log(`  ${run.id} discarded=${run.discardedAt ?? "?"}`);
  }
}

function usage() {
  console.log(`Usage:
  node scripts/validation-target.mjs create --profile <slug> [--label <text>] [--root <path>] [--skip-init]
  node scripts/validation-target.mjs list [--json]
  node scripts/validation-target.mjs show <run-id>
  node scripts/validation-target.mjs discard <run-id> [--purge] [--dry-run]
  node scripts/validation-target.mjs cleanup [--keep N] [--older-than-days N] [--dry-run]
  node scripts/validation-target.mjs profiles

Environment:
  MH_VALIDATION_ROOT  Parent directory for grouped runs (default ../demo/validation-runs)
`);
}

function main() {
  const args = process.argv.slice(2);
  const command = args[0];
  const flag = (name) => {
    const i = args.indexOf(name);
    return i !== -1 ? args[i + 1] : undefined;
  };
  const has = (name) => args.includes(name);
  const root = path.resolve(flag("--root") ?? defaultValidationRoot());

  if (command === "profiles") {
    printProfiles();
    return;
  }
  if (command === "create") {
    const profile = flag("--profile");
    if (!profile) throw new Error("create requires --profile <slug>");
    fs.mkdirSync(root, { recursive: true });
    const manifest = createRun({
      profile,
      label: flag("--label"),
      root,
      skipInit: has("--skip-init"),
    });
    if (has("--json")) {
      console.log(JSON.stringify(manifest, null, 2));
    } else {
      console.log(`Created validation run ${manifest.id}`);
      console.log(`  path: ${manifest.path}`);
      console.log(`  db:   ${manifest.dbPath}`);
      console.log(`  next: mars-harness start --repo ${manifest.path}`);
    }
    return;
  }
  if (command === "list") {
    printList(root, { json: has("--json") });
    return;
  }
  if (command === "show") {
    const id = args[1];
    if (!id) throw new Error("show requires <run-id>");
    const manifest = readManifest(resolveRun(root, id));
    if (!manifest) throw new Error("run exists but .validation-run.json is missing");
    console.log(JSON.stringify(manifest, null, 2));
    return;
  }
  if (command === "discard") {
    const id = args[1];
    if (!id) throw new Error("discard requires <run-id>");
    const manifest = discardRun(root, id, { dryRun: has("--dry-run"), purge: has("--purge") });
    if (has("--json")) console.log(JSON.stringify(manifest, null, 2));
    else if (manifest.dryRun) console.log(`Would discard ${id}`);
    else console.log(`Discarded ${manifest.id} (DB removed${has("--purge") ? ", folder purged" : ", manifest archived"})`);
    return;
  }
  if (command === "cleanup") {
    const keep = flag("--keep") != null ? Number(flag("--keep")) : 3;
    const olderThanDays = flag("--older-than-days") != null ? Number(flag("--older-than-days")) : undefined;
    const results = cleanupRuns(root, { keep, olderThanDays, dryRun: has("--dry-run") });
    if (has("--json")) console.log(JSON.stringify(results, null, 2));
    else console.log(has("--dry-run") ? `Would discard ${results.length} run(s)` : `Discarded ${results.length} run(s)`);
    return;
  }

  usage();
  process.exit(command ? 1 : 0);
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  try {
    main();
  } catch (err) {
    console.error(String(err.message ?? err));
    process.exit(1);
  }
}
