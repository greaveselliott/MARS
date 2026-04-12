(function () {
  "use strict";

  var eventSource = null;

  function connect() {
    eventSource = new EventSource("/api/events");

    eventSource.onmessage = function (e) {
      try {
        var msg = JSON.parse(e.data);
        var inner = {};
        try {
          inner = JSON.parse(msg.data);
        } catch (_) {
          inner = { raw: msg.data };
        }
        handleEvent(msg.type, inner);
      } catch (err) {
        console.error("SSE parse error:", err);
      }
    };

    eventSource.onerror = function () {
      eventSource.close();
      setTimeout(connect, 3000);
    };
  }

  function handleEvent(type, data) {
    appendEventLog(type, data);
    appendDebugLog(type, data);
    updateRoleGrid(type, data);
    updatePipelineChain(type, data);
    appendEvolutionLog(type, data);
  }

  function appendEventLog(type, data) {
    var log = document.getElementById("event-log");
    if (!log) return;

    var placeholder = log.querySelector(".muted");
    if (placeholder) placeholder.remove();

    var entry = document.createElement("div");
    entry.className = "event-entry " + type;
    if (data.outcome === "error") entry.className += " error";

    var now = new Date().toLocaleTimeString();
    var detail = formatDetail(type, data);
    entry.innerHTML =
      '<span class="event-time">' + now + "</span>" +
      '<span class="event-type">' + type + "</span>" +
      "<span>" + detail + "</span>";

    log.prepend(entry);

    while (log.children.length > 200) {
      log.removeChild(log.lastChild);
    }
  }

  function appendDebugLog(type, data) {
    var log = document.getElementById("debug-log");
    if (!log) return;
    var line = new Date().toISOString() + " [" + type + "] " + JSON.stringify(data) + "\n";
    log.textContent = line + log.textContent;
  }

  function formatDetail(type, data) {
    switch (type) {
      case "job_start":
        return '<strong>' + esc(data.role) + '</strong> started (job ' + esc(data.job_id || "") + ')';
      case "job_complete":
        return '<strong>' + esc(data.role) + '</strong> ' + esc(data.outcome || "done") + ' in ' + esc(data.duration || "?");
      case "chain":
        return esc(data.from) + ' &rarr; ' + esc(data.to);
      default:
        return JSON.stringify(data);
    }
  }

  function updateRoleGrid(type, data) {
    var grid = document.getElementById("role-grid");
    if (!grid) return;

    var placeholder = grid.querySelector(".muted");
    if (placeholder) placeholder.remove();

    var role = data.role || data.from;
    if (!role) return;

    var cardId = "role-" + role;
    var card = document.getElementById(cardId);
    if (!card) {
      card = document.createElement("div");
      card.id = cardId;
      card.className = "role-card";
      card.innerHTML = '<div class="role-name">' + esc(role) + '</div><div class="role-status">idle</div>';
      grid.appendChild(card);
    }

    var status = card.querySelector(".role-status");
    card.className = "role-card";

    if (type === "job_start") {
      card.className += " running";
      status.textContent = "running…";
    } else if (type === "job_complete") {
      card.className += " " + (data.outcome === "error" ? "error" : "success");
      status.textContent = (data.outcome || "done") + " (" + (data.duration || "?") + ")";
    }
  }

  function updatePipelineChain(type, data) {
    if (type !== "job_start" && type !== "job_complete") return;
    var role = data.role;
    if (!role) return;

    var nodes = document.querySelectorAll(".chain-node");
    nodes.forEach(function (node) {
      if (node.textContent.toLowerCase() === role.toLowerCase() ||
          role.toLowerCase().indexOf(node.textContent.toLowerCase()) === 0) {
        if (type === "job_start") {
          node.classList.add("active");
          node.classList.remove("done");
        } else if (type === "job_complete" && data.outcome !== "error") {
          node.classList.remove("active");
          node.classList.add("done");
        }
      }
    });
  }

  function appendEvolutionLog(type, data) {
    if (type !== "telemetry" && type !== "telemetry_pattern" && type !== "job_failed") return;
    var log = document.getElementById("evolution-log");
    if (!log) return;

    var placeholder = log.querySelector(".muted");
    if (placeholder) placeholder.remove();

    var entry = document.createElement("div");
    entry.className = "event-entry " + type;
    var now = new Date().toLocaleTimeString();
    var detail = "";

    if (type === "telemetry") {
      detail = '<strong>' + esc(data.role || "") + '</strong> ' +
        '<span class="badge">' + esc(data.category || "") + '</span> ' +
        (data.remedied ? '<span class="badge success">remedied</span>' : '');
    } else if (type === "telemetry_pattern") {
      detail = 'Recurring: <strong>' + esc(data.role || "") + '</strong> ' +
        esc(data.category || "") + ' (' + (data.count || 0) + 'x)';
    } else {
      detail = '<strong>' + esc(data.role || "") + '</strong> failed: ' + esc(data.error || "");
    }

    entry.innerHTML =
      '<span class="event-time">' + now + '</span>' +
      '<span class="event-type">' + type + '</span>' +
      '<span>' + detail + '</span>';

    log.prepend(entry);
    while (log.children.length > 100) log.removeChild(log.lastChild);
  }

  function esc(str) {
    var div = document.createElement("div");
    div.textContent = str;
    return div.innerHTML;
  }

  // --- Control surface ---

  var isPaused = false;

  function updateStatusUI(state, activeJobs) {
    var dot = document.getElementById("status-dot");
    var label = document.getElementById("status-label");
    var jobsEl = document.getElementById("active-jobs");
    var pauseLabel = document.getElementById("pause-label");

    if (!dot) return;

    dot.className = "status-dot";
    if (state === "paused") {
      dot.classList.add("paused");
      label.textContent = "Paused";
      isPaused = true;
      if (pauseLabel) pauseLabel.textContent = "Resume";
    } else if (state === "restarting") {
      dot.classList.add("restarting");
      label.textContent = "Restarting…";
    } else if (state === "stopped") {
      dot.classList.add("stopped");
      label.textContent = "Stopped";
    } else {
      label.textContent = "Running";
      isPaused = false;
      if (pauseLabel) pauseLabel.textContent = "Pause";
    }

    if (jobsEl && activeJobs !== undefined) {
      jobsEl.textContent = activeJobs > 0 ? activeJobs + " active" : "";
    }
  }

  function fetchStatus() {
    fetch("/api/status")
      .then(function (r) { return r.json(); })
      .then(function (data) {
        var state = data.paused ? "paused" : (data.healthy ? "running" : "stopped");
        updateStatusUI(state, data.active_jobs);
      })
      .catch(function () {});
  }

  function loadRepos() {
    fetch("/api/repos")
      .then(function (r) { return r.json(); })
      .then(function (repos) {
        var sel = document.getElementById("select-repo");
        if (!sel) return;
        sel.innerHTML = '<option value="">Repo...</option>';
        (repos || []).forEach(function (repo) {
          var opt = document.createElement("option");
          opt.value = repo.id;
          opt.textContent = repo.path.split("/").pop();
          sel.appendChild(opt);
        });
        if (repos && repos.length === 1) {
          sel.value = repos[0].id;
          loadRoles();
        }
      })
      .catch(function () {});
  }

  window.loadRoles = function () {
    var repoID = (document.getElementById("select-repo") || {}).value;
    var sel = document.getElementById("select-role");
    if (!sel) return;
    sel.innerHTML = '<option value="">Role...</option>';
    if (!repoID) return;

    fetch("/api/repo-roles?repo_id=" + encodeURIComponent(repoID))
      .then(function (r) { return r.json(); })
      .then(function (roles) {
        (roles || []).forEach(function (role) {
          var opt = document.createElement("option");
          opt.value = role;
          opt.textContent = role;
          sel.appendChild(opt);
        });
      })
      .catch(function () {});
  };

  window.togglePause = function () {
    var endpoint = isPaused ? "/api/resume" : "/api/pause";
    fetch(endpoint, { method: "POST" })
      .then(function (r) { return r.json(); })
      .then(function () { fetchStatus(); })
      .catch(function (err) { alert("Error: " + err); });
  };

  window.confirmRestart = function () {
    if (!confirm("Warm restart: workers will stop, config reloads, workers restart. Continue?")) return;
    var btn = document.getElementById("btn-restart");
    btn.disabled = true;
    btn.textContent = "Restarting…";
    fetch("/api/restart", { method: "POST" })
      .then(function (r) { return r.json(); })
      .then(function (resp) {
        btn.disabled = false;
        btn.innerHTML = '<span class="key-hint">R</span>Restart';
        if (!resp.ok && resp.error) alert("Restart failed: " + resp.error);
        fetchStatus();
      })
      .catch(function (err) {
        btn.disabled = false;
        btn.innerHTML = '<span class="key-hint">R</span>Restart';
        alert("Restart failed: " + err);
      });
  };

  window.confirmStop = function () {
    if (!confirm("Gracefully stop the orchestrator?")) return;
    fetch("/api/stop", { method: "POST" })
      .then(function (r) { return r.json(); })
      .then(function () { updateStatusUI("stopped"); })
      .catch(function (err) { alert("Stop failed: " + err); });
  };

  window.triggerScan = function () {
    var repoID = (document.getElementById("select-repo") || {}).value;
    if (!repoID) { alert("Select a repo first"); return; }
    var btn = document.getElementById("btn-scan");
    btn.disabled = true;
    fetch("/api/scan", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ repo_id: repoID })
    })
    .then(function (r) { return r.json(); })
    .then(function (resp) {
      btn.disabled = false;
      if (!resp.ok && resp.error) alert("Scan error: " + resp.error);
    })
    .catch(function (err) {
      btn.disabled = false;
      alert("Scan failed: " + err);
    });
  };

  window.triggerRunRole = function () {
    var repoID = (document.getElementById("select-repo") || {}).value;
    var role = (document.getElementById("select-role") || {}).value;
    if (!repoID || !role) { alert("Select both a repo and a role"); return; }
    fetch("/api/run-role", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ repo_id: repoID, role: role })
    })
    .then(function (r) { return r.json(); })
    .then(function (resp) {
      if (!resp.ok && resp.error) alert("Run role error: " + resp.error);
    })
    .catch(function (err) { alert("Run role failed: " + err); });
  };

  window.emergencyStop = function () {
    if (!confirm("This will stop ALL running agents. Continue?")) return;

    var btn = document.getElementById("estop-btn");
    btn.textContent = "Stopping…";
    btn.disabled = true;

    fetch("/api/emergency-stop", { method: "POST" })
      .then(function (r) { return r.json(); })
      .then(function (resp) {
        if (resp.ok) {
          btn.textContent = "Stopped";
          btn.style.background = "#16a34a";
          updateStatusUI("stopped");
        } else {
          btn.textContent = "Failed";
          alert("Errors: " + (resp.errors || []).join(", "));
        }
      })
      .catch(function (err) {
        btn.textContent = "Error";
        alert("Emergency stop request failed: " + err);
      });
  };

  // Handle status_change SSE events
  var origHandleEvent = handleEvent;
  handleEvent = function (type, data) {
    origHandleEvent(type, data);
    if (type === "status_change" && data.state) {
      updateStatusUI(data.state, data.active_jobs);
    }
    if (type === "scan_complete") {
      appendEventLog("scan_complete", data);
    }
  };

  // Bootstrap
  fetchStatus();
  loadRepos();
  setInterval(fetchStatus, 15000);

  connect();
})();
