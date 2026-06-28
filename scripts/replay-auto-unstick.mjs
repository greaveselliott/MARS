#!/usr/bin/env node
/**
 * Auto-unstick mars replays stuck on operator_retry dispositions.
 * Polls per-repo DB and POSTs /api/run-role for suggested_role when blocked.
 *
 * Usage:
 *   node scripts/replay-auto-unstick.mjs --repo demo-15
 *   node scripts/replay-auto-unstick.mjs --repo demo-15 --watch --interval 30
 *   node scripts/replay-auto-unstick.mjs --repo demo-15 --once --role qa
 */
import { execSync } from "node:child_process";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";

const args = process.argv.slice(2);
function flag(name) {
  const i = args.indexOf(name);
  return i !== -1 ? args[i + 1] : undefined;
}
const has = (name) => args.includes(name);

const repoSlug = flag("--repo") ?? "demo-15";
const watch = has("--watch");
const once = has("--once");
const intervalSec = Number(flag("--interval") ?? "30");
const dashboardURL = flag("--dashboard") ?? "http://localhost:9090";
const forceRole = flag("--role");

const dbPath = path.join(os.homedir(), ".mars/db", repoSlug, "mars.db");

function sqlite(query) {
  if (!fs.existsSync(dbPath)) return [];
  try {
    const out = execSync(`sqlite3 -json "${dbPath}" ${JSON.stringify(query)}`, {
      encoding: "utf8",
      stdio: ["pipe", "pipe", "pipe"],
    });
    return out.trim() ? JSON.parse(out) : [];
  } catch {
    return [];
  }
}

function fetchStatus() {
  try {
    const out = execSync(`curl -s --max-time 2 "${dashboardURL}/api/status"`, {
      encoding: "utf8",
    });
    return JSON.parse(out);
  } catch {
    return null;
  }
}

function repoIdForSlug(slug) {
  const status = fetchStatus();
  const match = status?.repos?.find((r) => r.path?.includes(slug));
  return match?.id ?? null;
}

function latestDisposition() {
  const rows = sqlite(
    `SELECT role, status, next_need, suggested_role, reason, recorded_at FROM job_dispositions ORDER BY recorded_at DESC LIMIT 1`,
  );
  return rows[0] ?? null;
}

function activeJobs() {
  const status = fetchStatus();
  return status?.active_jobs ?? 0;
}

function dispatchRunRole(repoId, role) {
  const body = JSON.stringify({ repo_id: repoId, role });
  execSync(
    `curl -s -X POST "${dashboardURL}/api/run-role" -H 'Content-Type: application/json' -d ${JSON.stringify(body)}`,
    { encoding: "utf8" },
  );
  console.log(`[unstick] dispatched ${role} for ${repoSlug} (${repoId})`);
}

function tick() {
  if (activeJobs() > 0) {
    console.log(`[unstick] ${repoSlug}: ${activeJobs()} active job(s) — skip`);
    return false;
  }

  const disp = latestDisposition();
  const repoId = repoIdForSlug(repoSlug);
  if (!repoId) {
    console.log(`[unstick] ${repoSlug}: repo id not found on dashboard`);
    return false;
  }

  const role = forceRole ?? disp?.suggested_role ?? disp?.role;
  const needsRetry =
    forceRole ||
    (disp?.status === "blocked" && disp?.next_need === "operator_retry");

  if (!needsRetry || !role) {
    console.log(
      `[unstick] ${repoSlug}: no operator_retry (${disp?.status}/${disp?.next_need ?? "none"})`,
    );
    return false;
  }

  dispatchRunRole(repoId, role);
  return true;
}

if (once || !watch) {
  tick();
} else {
  console.log(`[unstick] watching ${repoSlug} every ${intervalSec}s`);
  setInterval(tick, intervalSec * 1000);
  tick();
}
