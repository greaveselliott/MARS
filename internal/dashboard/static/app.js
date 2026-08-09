/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/dashboard.md
- docs/features/F-010-dashboard-control-plane.md
- docs/features/F-017-open-source-publication.md
*/
(function () {
  "use strict";

  var csrfToken = "";
  var eventSource = null;
  var reconnects = 0;
  var throughputChart = null;
  var failureChart = null;

  function byID(id) { return document.getElementById(id); }
  function clear(node) { if (node) node.replaceChildren(); }
  function node(tag, className, text) {
    var result = document.createElement(tag);
    if (className) result.className = className;
    if (text !== undefined && text !== null) result.textContent = String(text);
    return result;
  }
  function append(parent) {
    for (var i = 1; parent && i < arguments.length; i += 1) parent.appendChild(arguments[i]);
    return parent;
  }
  function fixedClass(value, allowed, fallback) {
    return allowed.indexOf(value) >= 0 ? value : fallback;
  }
  function bounded(value, max) {
    var text = String(value === undefined || value === null ? "" : value).replace(/[\u0000-\u0008\u000b\u000c\u000e-\u001f\u007f]/g, "");
    return text.length > max ? text.substring(0, max) + "…" : text;
  }
  function emptyMessage(parent, message) {
    clear(parent);
    append(parent, node("p", "muted", message));
  }
  function themeVar(name, fallback) {
    var value = getComputedStyle(document.documentElement).getPropertyValue(name).trim();
    return value || fallback;
  }

  function request(path, options) {
    var opts = options || {};
    opts.credentials = "same-origin";
    opts.headers = opts.headers || {};
    if (opts.method === "POST") {
      if (path !== "/api/login") opts.headers["X-MARS-CSRF-Token"] = csrfToken;
      if (opts.body !== undefined) opts.headers["Content-Type"] = "application/json";
    }
    return fetch(path, opts).then(function (response) {
      if (response.status === 401 || response.status === 403 || response.status === 503) showAuthState(response.status);
      return response.text().then(function (text) {
        var parsed = {};
        if (text) {
          try { parsed = JSON.parse(text); } catch (_) { parsed = { error: "invalid bounded dashboard response" }; }
        }
        if (!response.ok) throw new Error(bounded(parsed.error || ("request failed with status " + response.status), 240));
        return parsed;
      });
    });
  }

  function installAuthPanel() {
    var main = document.querySelector("main");
    if (!main || byID("dashboard-auth")) return;
    var panel = node("section", "auth-panel", "");
    panel.id = "dashboard-auth";
    var label = node("label", "auth-label", "Dashboard control secret");
    label.htmlFor = "dashboard-secret";
    var input = node("input", "auth-input", "");
    input.id = "dashboard-secret";
    input.type = "password";
    input.autocomplete = "current-password";
    input.minLength = 32;
    input.maxLength = 4096;
    var login = node("button", "ctrl-btn", "Unlock controls");
    login.id = "dashboard-login";
    login.type = "button";
    var logout = node("button", "ctrl-btn", "Log out");
    logout.id = "dashboard-logout";
    logout.type = "button";
    logout.hidden = true;
    var status = node("span", "muted", "Observer mode: controls require MARS_DASHBOARD_CONTROL_SECRET.");
    status.id = "dashboard-auth-status";
    append(panel, label, input, login, logout, status);
    main.insertBefore(panel, main.firstChild);
    login.addEventListener("click", loginDashboard);
    logout.addEventListener("click", logoutDashboard);
  }

  function loginDashboard() {
    var input = byID("dashboard-secret");
    var secret = input ? input.value : "";
    request("/api/login", { method: "POST", body: JSON.stringify({ secret: secret }) })
      .then(function (response) {
        csrfToken = bounded(response.csrf_token, 128);
        if (input) input.value = "";
        showAuthState(200);
        loadPrivileged();
      })
      .catch(function (err) { showAuthError(err); });
  }

  function logoutDashboard() {
    request("/api/logout", { method: "POST" })
      .then(function () { window.location.reload(); })
      .catch(function (err) { showAuthError(err); });
  }

  function showAuthState(statusCode) {
    var status = byID("dashboard-auth-status");
    var login = byID("dashboard-login");
    var logout = byID("dashboard-logout");
    if (!status) return;
    if (statusCode === 200 && csrfToken) {
      status.textContent = "Controls unlocked for this browser session.";
      if (login) login.hidden = true;
      if (logout) logout.hidden = false;
    } else if (statusCode === 503) {
      status.textContent = "Controls disabled. Set MARS_DASHBOARD_CONTROL_SECRET and restart MARS.";
      if (login) login.hidden = false;
      if (logout) logout.hidden = true;
    } else {
      csrfToken = "";
      status.textContent = "Observer mode: authenticate to use controls.";
      if (login) login.hidden = false;
      if (logout) logout.hidden = true;
    }
  }

  function showAuthError(err) {
    var status = byID("dashboard-auth-status");
    if (status) status.textContent = bounded(err && err.message ? err.message : "authentication failed", 240);
  }

  function bootstrapSession() {
    return request("/api/session").then(function (response) {
      csrfToken = bounded(response.csrf_token, 128);
      showAuthState(200);
      loadPrivileged();
    }).catch(function () {});
  }

  function connectEvents() {
    if (eventSource) eventSource.close();
    eventSource = new EventSource("/api/events");
    eventSource.onopen = function () { reconnects = 0; };
    eventSource.onmessage = function (event) {
      try {
        var envelope = JSON.parse(event.data);
        var data = {};
        try { data = JSON.parse(envelope.data); } catch (_) { data = { raw: bounded(envelope.data, 2048) }; }
        handleEvent(bounded(envelope.type, 64), data);
      } catch (_) { /* malformed events are ignored */ }
    };
    eventSource.onerror = function () {
      eventSource.close();
      fetch("/api/status", { credentials: "same-origin" }).then(function (response) {
        if (response.status === 401 || response.status === 403) {
          showAuthState(response.status);
          return;
        }
        if (reconnects >= 8) return;
        reconnects += 1;
        window.setTimeout(connectEvents, Math.min(30000, 1000 * Math.pow(2, reconnects)));
      }).catch(function () {
        if (reconnects >= 8) return;
        reconnects += 1;
        window.setTimeout(connectEvents, Math.min(30000, 1000 * Math.pow(2, reconnects)));
      });
    };
  }

  function handleEvent(type, data) {
    appendEventLog(type, data);
    appendDebugLog(type, data);
    updateRoleGrid(type, data);
    updatePipelineChain(type, data);
    appendEvolutionEvent(type, data);
    if (type === "status_change") updateStatusUI(data.state, data.active_jobs);
  }

  function appendEventLog(type, data) {
    var log = byID("event-log");
    if (!log) return;
    var placeholder = log.querySelector(".muted");
    if (placeholder) placeholder.remove();
    var entry = node("div", "event-entry " + fixedClass(type, ["job_start", "job_complete", "chain", "dispatch_return", "job_disposition", "orchestration_decision", "dispatch_enqueued", "scan_complete"], "event"));
    append(entry,
      node("span", "event-time", new Date().toLocaleTimeString()),
      node("span", "event-type", bounded(type, 64)),
      node("span", "", eventDetail(type, data))
    );
    log.prepend(entry);
    while (log.children.length > 100) log.lastChild.remove();
  }

  function eventDetail(type, data) {
    if (type === "job_start") return bounded(data.role, 128) + " started (job " + bounded(data.job_id, 128) + ")";
    if (type === "job_complete") return bounded(data.role, 128) + " " + bounded(data.outcome || "done", 64) + " in " + bounded(data.duration || "?", 64);
    if (type === "chain") return bounded(data.from, 128) + " → " + bounded(data.to, 128);
    if (type === "job_failed") return bounded(data.role, 128) + " failed";
    return bounded(JSON.stringify(data), 1024);
  }

  function appendDebugLog(type, data) {
    var log = byID("debug-log");
    if (!log) return;
    var line = new Date().toISOString() + " [" + bounded(type, 64) + "] " + bounded(JSON.stringify(data), 2048) + "\n";
    log.textContent = bounded(line + log.textContent, 32768);
  }

  function updateRoleGrid(type, data) {
    var grid = byID("role-grid");
    var role = bounded(data.role || data.from, 128);
    if (!grid || !role) return;
    var card = Array.prototype.find.call(grid.querySelectorAll(".role-card"), function (candidate) { return candidate.dataset.role === role; });
    if (!card) {
      card = node("div", "role-card");
      card.dataset.role = role;
      append(card, node("div", "role-name", role), node("div", "role-status", "idle"));
      grid.appendChild(card);
    }
    var status = card.querySelector(".role-status");
    card.className = "role-card";
    if (type === "job_start") { card.classList.add("running"); status.textContent = "running…"; }
    if (type === "job_complete") {
      card.classList.add(data.outcome === "error" ? "error" : "success");
      status.textContent = bounded(data.outcome || "done", 64) + " (" + bounded(data.duration || "?", 64) + ")";
    }
  }

  function updatePipelineChain(type, data) {
    if (type !== "job_start" && type !== "job_complete") return;
    var role = bounded(data.role, 128).toLowerCase();
    document.querySelectorAll(".chain-node").forEach(function (candidate) {
      if (candidate.textContent.toLowerCase() !== role) return;
      candidate.classList.toggle("active", type === "job_start");
      candidate.classList.toggle("done", type === "job_complete" && data.outcome !== "error");
    });
  }

  function appendEvolutionEvent(type, data) {
    if (["telemetry", "telemetry_pattern", "job_failed"].indexOf(type) < 0) return;
    var log = byID("evolution-log");
    if (!log) return;
    var entry = node("div", "event-entry event");
    append(entry, node("span", "event-time", new Date().toLocaleTimeString()), node("span", "event-type", type), node("span", "", eventDetail(type, data)));
    log.prepend(entry);
    while (log.children.length > 100) log.lastChild.remove();
  }

  var isPaused = false;
  function updateStatusUI(state, activeJobs) {
    var dot = byID("status-dot");
    var label = byID("status-label");
    var jobs = byID("active-jobs");
    var pauseLabel = byID("pause-label");
    if (!dot || !label) return;
    var safeState = fixedClass(state, ["paused", "restarting", "stopped", "running"], "running");
    dot.className = "status-dot " + safeState;
    label.textContent = safeState.substring(0, 1).toUpperCase() + safeState.substring(1);
    isPaused = safeState === "paused";
    if (pauseLabel) pauseLabel.textContent = isPaused ? "Resume" : "Pause";
    if (jobs) jobs.textContent = Number(activeJobs) > 0 ? Number(activeJobs) + " active" : "";
  }

  function fetchStatus() {
    request("/api/status").then(function (data) {
      updateStatusUI(data.paused ? "paused" : (data.healthy ? "running" : "stopped"), data.active_jobs);
    }).catch(function () {});
  }

  function loadRepos() {
    request("/api/repos").then(function (repos) {
      var select = byID("select-repo");
      if (!select) return;
      clear(select);
      select.appendChild(new Option("Repo…", ""));
      (Array.isArray(repos) ? repos.slice(0, 256) : []).forEach(function (repo) {
        select.appendChild(new Option(bounded(repo.path, 128), bounded(repo.id, 256)));
      });
      if (repos.length === 1) { select.value = bounded(repos[0].id, 256); loadRoles(); }
    }).catch(function () {});
  }

  function loadRoles() {
    var repoID = byID("select-repo") ? byID("select-repo").value : "";
    var select = byID("select-role");
    if (!select) return;
    clear(select);
    select.appendChild(new Option("Role…", ""));
    if (!repoID) return;
    request("/api/repo-roles?repo_id=" + encodeURIComponent(repoID)).then(function (roles) {
      (Array.isArray(roles) ? roles.slice(0, 256) : []).forEach(function (role) { select.appendChild(new Option(bounded(role, 128), bounded(role, 256))); });
    }).catch(function () {});
  }

  function mutate(path, body) {
    var options = { method: "POST" };
    if (body !== undefined) options.body = JSON.stringify(body);
    return request(path, options);
  }

  function installControls() {
    var bindings = [
      ["btn-pause", function () { mutate(isPaused ? "/api/resume" : "/api/pause").then(fetchStatus).catch(showAuthError); }],
      ["btn-restart", function () { if (window.confirm("Warm restart workers and reload configuration?")) mutate("/api/restart").then(fetchStatus).catch(showAuthError); }],
      ["btn-stop", function () { if (window.confirm("Gracefully stop the orchestrator?")) mutate("/api/stop").then(function () { updateStatusUI("stopped"); }).catch(showAuthError); }],
      ["btn-scan", function () { var repo = byID("select-repo"); if (repo && repo.value) mutate("/api/scan", { repo_id: repo.value }).catch(showAuthError); }],
      ["btn-run", function () { var repo = byID("select-repo"); var role = byID("select-role"); if (repo && role && repo.value && role.value) mutate("/api/run-role", { repo_id: repo.value, role: role.value }).catch(showAuthError); }],
      ["estop-btn", function () { if (window.confirm("Stop all running agents?")) mutate("/api/emergency-stop").then(function () { updateStatusUI("stopped"); }).catch(showAuthError); }]
    ];
    bindings.forEach(function (binding) { var target = byID(binding[0]); if (target) target.addEventListener("click", binding[1]); });
    var repoSelect = byID("select-repo");
    if (repoSelect) repoSelect.addEventListener("change", loadRoles);
  }

  function renderRoles(roles) {
    var grid = byID("role-grid");
    if (!grid) return;
    if (!roles.length) { emptyMessage(grid, "No role data yet — run some jobs first"); return; }
    clear(grid);
    roles.slice(0, 128).forEach(function (role) {
      var score = Math.max(0, Math.min(1, Number(role.score) || 0));
      var pct = Math.round(score * 100);
      var scoreClass = score >= 0.8 ? "success" : score >= 0.5 ? "warning" : "error";
      var card = node("div", "role-card " + scoreClass);
      card.dataset.role = bounded(role.role, 128);
      var bar = node("div", "score-bar");
      append(bar, node("div", "score-fill " + scoreClass + " score-width-" + Math.round(pct / 10)));
      append(card,
        node("div", "role-name", bounded(role.role, 128)),
        append(node("div", "role-score"), bar, node("span", "", pct + "%")),
        append(node("div", "role-stats"), node("span", "stat-success", Number(role.success_count) + " passed"), node("span", "stat-fail", Number(role.fail_count) + " failed"), node("span", "stat-total", Number(role.sample_size) + " total")),
        node("div", "role-status", "idle")
      );
      grid.appendChild(card);
    });
  }

  function loadRolesData() { if (byID("role-grid")) request("/api/roles").then(function (data) { renderRoles(data.roles || []); }).catch(function () {}); }

  function renderThroughput(data) {
    [["summary-total", data.summary.total], ["summary-completed", data.summary.completed], ["summary-failed", data.summary.failed], ["summary-running", data.summary.running]].forEach(function (item) { if (byID(item[0])) byID(item[0]).textContent = Number(item[1]) || 0; });
    renderThroughputChart(data.hourly || []);
    var body = byID("job-tbody");
    if (!body) return;
    if (!(data.recent_jobs || []).length) { clear(body); var row = node("tr"); var cell = node("td", "muted", "No jobs recorded yet"); cell.colSpan = 6; row.appendChild(cell); body.appendChild(row); return; }
    clear(body);
    data.recent_jobs.slice(0, 50).forEach(function (job) {
      var row = node("tr");
      var status = fixedClass(job.status, ["completed", "failed", "running", "pending", "claimed"], "pending");
      [bounded(job.id, 32), bounded(job.role, 128), status, bounded(job.created_at, 64), bounded(job.duration || "—", 64), bounded(job.error || "", 120)].forEach(function (value, index) {
        var cell = node("td", index === 0 ? "mono" : index === 5 ? "error-cell" : "", value);
        row.appendChild(cell);
      });
      body.appendChild(row);
    });
  }

  function renderThroughputChart(hourly) {
    var canvas = byID("throughput-chart");
    if (!canvas || typeof window.Chart !== "function") return;
    if (throughputChart) throughputChart.destroy();
    var wrap = canvas.parentElement;
    var prior = wrap.querySelector(".chart-empty"); if (prior) prior.remove();
    if (!hourly.length) { append(wrap, node("p", "muted chart-empty", "No job data yet — run some agents first")); return; }
    throughputChart = new window.Chart(canvas, { type: "bar", data: { labels: hourly.map(function (h) { return bounded(h.hour, 64); }), datasets: [
      { label: "Completed", data: hourly.map(function (h) { return Number(h.completed) || 0; }), backgroundColor: "#22c55e" },
      { label: "Failed", data: hourly.map(function (h) { return Number(h.failed) || 0; }), backgroundColor: "#f43f5e" }
    ] }, options: { responsive: true, maintainAspectRatio: false, scales: { x: { stacked: true }, y: { stacked: true, beginAtZero: true } } } });
  }

  function loadThroughput() { if (byID("job-tbody")) request("/api/throughput").then(renderThroughput).catch(function () {}); }

  function renderEvolution(data) {
    var evolution = byID("evolution-log");
    if (evolution) {
      clear(evolution);
      if (!(data.evolutions || []).length) append(evolution, node("p", "muted", "No evolution events recorded yet…"));
      (data.evolutions || []).slice(0, 50).forEach(function (item) { append(evolution, append(node("div", "event-entry evolution"), node("span", "event-time", bounded(item.created_at, 32)), node("span", "event-type", bounded(item.classification, 64)), node("span", "", bounded(item.role, 128) + ": " + bounded(item.suggestion, 512)))); });
    }
    var patterns = byID("patterns-list");
    if (patterns) { clear(patterns); if (!(data.patterns || []).length) append(patterns, node("p", "muted", "No recurring patterns detected")); (data.patterns || []).slice(0, 50).forEach(function (item) { append(patterns, node("div", "pattern-item", bounded(item.role, 128) + " — " + bounded(item.category, 128) + " (" + Number(item.count) + ")")); }); }
    var telemetry = byID("telemetry-log");
    if (telemetry) { clear(telemetry); if (!(data.telemetry || []).length) append(telemetry, node("p", "muted", "No telemetry events yet…")); (data.telemetry || []).slice(0, 50).forEach(function (item) { append(telemetry, node("div", "event-entry telemetry", bounded(item.role, 128) + " — " + bounded(item.category, 128))); }); }
    renderFailureChart(data.telemetry || []);
  }

  function renderFailureChart(events) {
    var canvas = byID("failure-chart");
    if (!canvas || typeof window.Chart !== "function") return;
    var counts = {};
    events.forEach(function (event) { var category = bounded(event.category || "unknown", 64); counts[category] = (counts[category] || 0) + 1; });
    if (failureChart) failureChart.destroy();
    var prior = canvas.parentElement.querySelector(".chart-empty"); if (prior) prior.remove();
    var labels = Object.keys(counts).slice(0, 32);
    if (!labels.length) { append(canvas.parentElement, node("p", "muted chart-empty", "No failures yet")); return; }
    failureChart = new window.Chart(canvas, { type: "doughnut", data: { labels: labels, datasets: [{ data: labels.map(function (label) { return counts[label]; }), backgroundColor: [themeVar("--danger", "#f43f5e"), themeVar("--warning", "#f59e0b"), themeVar("--primary", "#8b5cf6"), themeVar("--success", "#22c55e")] }] }, options: { responsive: true, maintainAspectRatio: false } });
  }

  function loadEvolution() { if (byID("evolution-log")) request("/api/evolution").then(renderEvolution).catch(function () {}); }

  function renderOrchestration(data) {
    var repos = byID("orchestration-repos");
    if (repos) { clear(repos); if (!(data.repos || []).length) append(repos, node("p", "muted", "No repositories registered.")); (data.repos || []).slice(0, 64).forEach(function (repo) { append(repos, node("div", "repo-row", bounded(repo.path, 128) + " — " + bounded(repo.orchestration_mode || "legacy", 32))); }); }
    var model = byID("dispatch-model");
    if (model) { clear(model); var first = (data.repos || [])[0]; if (!first) append(model, node("p", "muted", "No repositories registered.")); else { append(model, node("div", "hub-node", "Orchestrator"), node("div", "dispatch-return", bounded(first.orchestration_mode || "legacy", 32))); var row = node("div", "spoke-row"); (first.roles || []).slice(0, 64).forEach(function (role) { append(row, node("span", "spoke-node", bounded(role.name, 128))); }); append(model, row); } }
    renderDecisionList(byID("disposition-list"), data.dispositions || [], "No dispositions recorded yet…");
    renderDecisionList(byID("decision-list"), data.decisions || [], "No dispatch decisions recorded yet…");
  }

  function renderDecisionList(parent, items, fallback) {
    if (!parent) return;
    clear(parent);
    if (!items.length) append(parent, node("p", "muted", fallback));
    items.slice(0, 50).forEach(function (item) { append(parent, node("div", "decision-item", bounded((item.role || item.source_role || item.from_role || "") + " — " + (item.status || item.next_role || "stop") + " — " + (item.reason || item.next_need || ""), 768))); });
  }

  function loadOrchestration() { if (byID("orchestration-repos")) request("/api/orchestration").then(renderOrchestration).catch(function () {}); }

  function loadPrivileged() {
    if (!csrfToken) return;
    loadRepos();
    loadRolesData();
    loadThroughput();
    loadEvolution();
    loadOrchestration();
    connectEvents();
  }

  installAuthPanel();
  installControls();
  fetchStatus();
  bootstrapSession();
  window.setInterval(fetchStatus, 15000);
  window.setInterval(function () { if (csrfToken) loadThroughput(); }, 15000);
  window.setInterval(function () { if (csrfToken) loadOrchestration(); }, 15000);
  window.setInterval(function () { if (csrfToken) loadRolesData(); }, 30000);
  window.setInterval(function () { if (csrfToken) loadEvolution(); }, 30000);
})();
