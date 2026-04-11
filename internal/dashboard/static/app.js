// app.js — Mars Harness dashboard: SSE handlers, Chart.js init, emergency stop.

(function () {
  "use strict";

  // SSE event handlers — trigger htmx refreshes when server pushes updates.
  document.body.addEventListener("sse:job_update", function () {
    var table = document.getElementById("job-table");
    if (table && typeof htmx !== "undefined") {
      htmx.trigger(table, "refresh");
    }
  });

  document.body.addEventListener("sse:score_update", function () {
    var grid = document.getElementById("role-grid");
    if (grid && typeof htmx !== "undefined") {
      htmx.trigger(grid, "refresh");
    }
  });

  document.body.addEventListener("sse:trace_event", function () {
    var viewer = document.getElementById("trace-viewer");
    if (viewer && typeof htmx !== "undefined") {
      htmx.trigger(viewer, "refresh");
    }
  });

  // Chart.js — initialise placeholder charts on the throughput page.
  function initCharts() {
    if (typeof Chart === "undefined") return;

    var placeholderLabels = ["—", "—", "—", "—", "—"];
    var placeholderData = [0, 0, 0, 0, 0];

    var jobsCanvas = document.getElementById("chart-jobs-hour");
    if (jobsCanvas) {
      new Chart(jobsCanvas.getContext("2d"), {
        type: "bar",
        data: {
          labels: placeholderLabels,
          datasets: [{
            label: "Jobs",
            data: placeholderData,
            backgroundColor: "rgba(99,102,241,0.6)"
          }]
        },
        options: {
          responsive: true,
          maintainAspectRatio: false,
          plugins: { legend: { display: false } },
          scales: {
            x: { ticks: { color: "#8b8fa3" }, grid: { color: "#2a2d3a" } },
            y: { ticks: { color: "#8b8fa3" }, grid: { color: "#2a2d3a" }, beginAtZero: true }
          }
        }
      });
    }

    var latencyCanvas = document.getElementById("chart-latency");
    if (latencyCanvas) {
      new Chart(latencyCanvas.getContext("2d"), {
        type: "line",
        data: {
          labels: placeholderLabels,
          datasets: [{
            label: "Latency (ms)",
            data: placeholderData,
            borderColor: "#6366f1",
            backgroundColor: "rgba(99,102,241,0.1)",
            fill: true,
            tension: 0.3
          }]
        },
        options: {
          responsive: true,
          maintainAspectRatio: false,
          plugins: { legend: { display: false } },
          scales: {
            x: { ticks: { color: "#8b8fa3" }, grid: { color: "#2a2d3a" } },
            y: { ticks: { color: "#8b8fa3" }, grid: { color: "#2a2d3a" }, beginAtZero: true }
          }
        }
      });
    }
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", initCharts);
  } else {
    initCharts();
  }
})();

// Emergency stop — called from base template button onclick.
function confirmEmergencyStop() {
  if (!confirm("EMERGENCY STOP: This will halt all running agents and jobs. Continue?")) {
    return;
  }
  fetch("/api/emergency-stop", { method: "POST" })
    .then(function (resp) { return resp.json(); })
    .then(function (data) {
      if (data.ok) {
        alert("Emergency stop executed successfully.");
      } else {
        alert("Emergency stop completed with errors:\n" + (data.errors || []).join("\n"));
      }
    })
    .catch(function (err) {
      alert("Emergency stop request failed: " + err.message);
    });
}
