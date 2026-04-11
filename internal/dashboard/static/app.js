// app.js — htmx SSE event handlers for Mars Harness dashboard.

document.body.addEventListener("sse:job_update", function () {
  var table = document.getElementById("job-table");
  if (table) {
    htmx.trigger(table, "refresh");
  }
});

document.body.addEventListener("sse:score_update", function () {
  var grid = document.getElementById("role-grid");
  if (grid) {
    htmx.trigger(grid, "refresh");
  }
});

document.body.addEventListener("sse:trace_event", function () {
  var viewer = document.getElementById("trace-viewer");
  if (viewer) {
    htmx.trigger(viewer, "refresh");
  }
});
