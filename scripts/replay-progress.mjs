#!/usr/bin/env node
/**
 * Live progress + rolling ETA for a mars-harness per-repo replay.
 * Reads ~/.mars-harness/db/<repo>/mars.db and optional dashboard /api/status.
 *
 * Usage:
 *   node scripts/replay-progress.mjs --repo demo-15
 *   node scripts/replay-progress.mjs --repo demo-15 --watch --interval 15
 *   node scripts/replay-progress.mjs --repo demo-15 --json
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
const intervalSec = Number(flag("--interval") ?? "15");
const jsonOut = has("--json");

const dbPath = path.join(os.homedir(), ".mars-harness/db", repoSlug, "mars.db");
const dashboardURL = flag("--dashboard") ?? "http://localhost:9090";

/** demo-14 Inventory/API baseline (126.5 min, 48 jobs) — seconds per phase archetype */
const BASELINE = {
  ceo: 65,
  coo: 90,
  "cto-weekly": 120,
  engineerCycle: 600,
  qa: 45,
  security: 30,
  dogfood: 110,
  orchestrator: 60,
  ticketsPerCycle: 1,
  expectedCycles: 4,
  totalRunSec: 126.5 * 60,
};

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

function fmtDuration(sec) {
  if (!Number.isFinite(sec) || sec < 0) return "—";
  if (sec < 60) return `${Math.round(sec)}s`;
  const m = Math.floor(sec / 60);
  const s = Math.round(sec % 60);
  if (m < 60) return s ? `${m}m ${s}s` : `${m}m`;
  const h = Math.floor(m / 60);
  const rm = m % 60;
  return rm ? `${h}h ${rm}m` : `${h}h`;
}

function fmtClock(unix) {
  if (!unix) return "—";
  return new Date(unix * 1000).toLocaleTimeString(undefined, {
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  });
}

function bar(ratio, width = 28) {
  const clamped = Math.max(0, Math.min(1, ratio));
  const filled = Math.round(clamped * width);
  return `${"█".repeat(filled)}${"░".repeat(width - filled)}`;
}

function buildSnapshot() {
  const now = Date.now() / 1000;
  const jobs = sqlite(
    `SELECT id, role, status, created_at, updated_at, completed_at, error_msg FROM jobs ORDER BY created_at`,
  );

  const status = fetchStatus();
  const firstStart = jobs[0]?.created_at ?? now;
  const elapsed = now - firstStart;

  const roleCounts = {};
  for (const j of jobs) {
    roleCounts[j.role] = roleCounts[j.role] ?? { completed: 0, failed: 0, running: 0, pending: 0 };
    const bucket = roleCounts[j.role];
    if (j.status === "completed") bucket.completed++;
    else if (j.status === "failed") bucket.failed++;
    else if (j.status === "running" || j.status === "claimed") bucket.running++;
    else if (j.status === "pending") bucket.pending++;
  }

  const dispositions = sqlite(
    `SELECT role, status, next_need, suggested_role, reason, recorded_at FROM job_dispositions ORDER BY recorded_at DESC LIMIT 1`,
  );
  const latestDisp = dispositions[0];

  const runningJob = jobs.find((j) => j.status === "running" || j.status === "claimed");
  const pendingJobs = jobs.filter((j) => j.status === "pending");
  const failedJobs = jobs.filter((j) => j.status === "failed");
  const completedJobs = jobs.filter((j) => j.status === "completed");

  const blocked =
    latestDisp?.status === "blocked" &&
    latestDisp?.next_need === "operator_retry" &&
    !runningJob &&
    pendingJobs.length === 0;

  const phases = [
    {
      id: "bootstrap",
      label: "1. CEO — vision & goals",
      baselineSec: BASELINE.ceo,
      status: roleCounts.ceo?.completed ? "completed" : runningJob?.role === "ceo" ? "active" : "pending",
      actualSec: jobs.find((j) => j.role === "ceo" && j.status === "completed")
        ? jobs.find((j) => j.role === "ceo").updated_at - jobs.find((j) => j.role === "ceo").created_at
        : null,
    },
    {
      id: "planning",
      label: "2. COO — exec plan & feature contract",
      baselineSec: BASELINE.coo,
      status: roleCounts.coo?.completed ? "completed" : runningJob?.role === "coo" ? "active" : roleCounts.ceo?.completed ? "pending" : "pending",
      actualSec: jobs.find((j) => j.role === "coo" && j.status === "completed")
        ? jobs.find((j) => j.role === "coo").updated_at - jobs.find((j) => j.role === "coo").created_at
        : null,
    },
    {
      id: "ticketing",
      label: "3. CTO — scenario tickets",
      baselineSec: BASELINE["cto-weekly"],
      status: roleCounts["cto-weekly"]?.completed
        ? "completed"
        : runningJob?.role === "cto-weekly"
          ? "active"
          : roleCounts.coo?.completed
            ? "pending"
            : "pending",
      actualSec: jobs.find((j) => j.role === "cto-weekly" && j.status === "completed")
        ? jobs.find((j) => j.role === "cto-weekly").updated_at -
          jobs.find((j) => j.role === "cto-weekly").created_at
        : null,
    },
    {
      id: "delivery",
      label: "4. Product delivery (Engineer → QA → Security → Dogfood)",
      baselineSec: BASELINE.totalRunSec - BASELINE.ceo - BASELINE.coo - BASELINE["cto-weekly"],
      status: blocked
        ? "blocked"
        : runningJob
          ? "active"
          : roleCounts["cto-weekly"]?.completed
            ? completedJobs.some((j) => ["qa", "security", "dogfood"].includes(j.role))
              ? "active"
              : roleCounts.engineer?.failed
                ? "active"
                : "pending"
            : "pending",
      actualSec: null,
      detail: `${roleCounts.engineer?.completed ?? 0} eng ✓ · ${roleCounts.engineer?.failed ?? 0} eng ✗ · ${roleCounts.qa?.completed ?? 0} qa · ${roleCounts.dogfood?.completed ?? 0} dogfood`,
    },
    {
      id: "drain",
      label: "5. Queue drain (natural completion)",
      baselineSec: 0,
      status:
        status && !status.paused && status.active_jobs === 0 && pendingJobs.length === 0 && !blocked && jobs.length > 10
          ? "completed"
          : "pending",
      actualSec: null,
    },
  ];

  // Fix delivery actualSec
  const planningDone = phases.slice(0, 3).reduce((s, p) => s + (p.actualSec ?? 0), 0);
  phases[3].actualSec = Math.max(0, elapsed - planningDone);

  let rollingRemaining = 0;
  for (const phase of phases) {
    if (phase.status === "completed") {
      phase.etaSec = 0;
      phase.etaLabel = "Done";
      continue;
    }
    if (phase.status === "blocked") {
      phase.etaSec = null;
      phase.etaLabel = "Blocked — operator retry";
      rollingRemaining += phase.baselineSec * 0.5;
      continue;
    }
    if (phase.status === "active") {
      const spent = phase.actualSec ?? 0;
      const remain = Math.max(0, phase.baselineSec - spent);
      phase.etaSec = remain;
      phase.etaLabel = fmtDuration(remain);
      rollingRemaining += remain;
      continue;
    }
    phase.etaSec = phase.baselineSec;
    phase.etaLabel = fmtDuration(phase.baselineSec);
    rollingRemaining += phase.baselineSec;
  }

  const completedPhaseCount = phases.filter((p) => p.status === "completed").length;
  const progressRatio = completedPhaseCount / phases.length;

  return {
    repoSlug,
    dbPath,
    now,
    elapsed,
    elapsedLabel: fmtDuration(elapsed),
    rollingEtaSec: blocked ? null : rollingRemaining,
    rollingEtaLabel: blocked ? "Paused (operator)" : fmtDuration(rollingRemaining),
    progressRatio,
    completedPhaseCount,
    phaseCount: phases.length,
    phases,
    jobs: {
      total: jobs.length,
      completed: completedJobs.length,
      failed: failedJobs.length,
      pending: pendingJobs.length,
      running: runningJob ? 1 : 0,
    },
    runningJob: runningJob
      ? { role: runningJob.role, id: runningJob.id.slice(0, 8), started: fmtClock(runningJob.created_at) }
      : null,
    blocked,
    blockReason: blocked ? latestDisp?.reason || "convergence failure escalated to operator_retry" : null,
    dashboard: status,
    recentFailures: failedJobs.slice(-3).map((j) => ({
      role: j.role,
      error: (j.error_msg || "").slice(0, 60),
      at: fmtClock(j.updated_at),
    })),
  };
}

function renderText(s) {
  const lines = [];
  lines.push("");
  lines.push(`mars-harness replay · ${s.repoSlug} · ${new Date(s.now * 1000).toLocaleTimeString()}`);
  lines.push(`Elapsed ${s.elapsedLabel}  ·  Rolling ETA ${s.rollingEtaLabel}  ·  Jobs ${s.jobs.completed}/${s.jobs.total} done (${s.jobs.failed} failed)`);
  if (s.blocked) lines.push(`⚠ BLOCKED: ${s.blockReason}`);
  else if (s.runningJob) lines.push(`▶ Running: ${s.runningJob.role} (since ${s.runningJob.started})`);
  else if (s.dashboard?.active_jobs === 0 && s.jobs.pending === 0) lines.push("⏸ Idle — survey paused or between dispatches");
  lines.push("");
  lines.push(`${bar(s.progressRatio)} ${Math.round(s.progressRatio * 100)}% phases (${s.completedPhaseCount}/${s.phaseCount})`);
  lines.push("");
  for (const p of s.phases) {
    const icon =
      p.status === "completed" ? "✓" : p.status === "active" ? "▶" : p.status === "blocked" ? "!" : "○";
    const actual = p.actualSec != null ? fmtDuration(p.actualSec) : "—";
    const baseline = fmtDuration(p.baselineSec);
    lines.push(
      `  ${icon} ${p.label.padEnd(52)} ${actual.padStart(7)} / ~${baseline.padStart(6)}  ETA ${p.etaLabel}`,
    );
    if (p.detail) lines.push(`      ${p.detail}`);
  }
  if (s.recentFailures.length) {
    lines.push("");
    lines.push("Recent failures:");
    for (const f of s.recentFailures) {
      lines.push(`  · ${f.at} ${f.role}: ${f.error}`);
    }
  }
  lines.push("");
  return lines.join("\n");
}

function main() {
  const snap = buildSnapshot();
  if (jsonOut) {
    process.stdout.write(JSON.stringify(snap, null, 2) + "\n");
    return;
  }
  process.stdout.write(renderText(snap));
}

if (watch) {
  const tick = () => {
    console.clear();
    main();
  };
  tick();
  setInterval(tick, intervalSec * 1000);
} else {
  main();
}
