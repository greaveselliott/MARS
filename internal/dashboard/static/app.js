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

  function esc(str) {
    var div = document.createElement("div");
    div.textContent = str;
    return div.innerHTML;
  }

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

  connect();
})();
