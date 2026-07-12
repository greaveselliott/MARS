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
  var button = document.getElementById("login-submit");
  var input = document.getElementById("login-secret");
  var status = document.getElementById("login-status");
  if (!button || !input || !status) return;

  function login() {
    button.disabled = true;
    status.textContent = "Authenticating…";
    fetch("/api/login", {
      method: "POST",
      credentials: "same-origin",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ secret: input.value })
    }).then(function (response) {
      input.value = "";
      if (!response.ok) throw new Error("Dashboard authentication failed.");
      window.location.replace("/pipeline");
    }).catch(function (error) {
      status.textContent = error.message;
      button.disabled = false;
      input.focus();
    });
  }

  button.addEventListener("click", login);
  input.addEventListener("keydown", function (event) {
    if (event.key === "Enter") login();
  });
})();
