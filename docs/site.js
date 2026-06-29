/*
MarsDocSync:
docs:
- docs/index.html
- docs/quickstart.html
- docs/install-setup-reference.html
- docs/shell-integration-reference.html
- docs/auth-credentials-reference.html
- docs/cli-reference.html
- docs/workflows.html
- docs/harness-guide.html
- docs/target-lifecycle-reference.html
- docs/bundle-reference.html
- docs/guardrails-reference.html
- docs/operations-guide.html
- docs/models-guide.html
- docs/tools-mcp-guide.html
- docs/code-intel-reference.html
- docs/roles-guide.html
- docs/documentation-sync-guide.html
- docs/configuration-reference.html
- docs/observability-guide.html
- docs/troubleshooting-guide.html
- docs/safety-quality-guide.html
- docs/release-update-guide.html
- docs/integrations-validation-guide.html
- docs/planning-delivery-guide.html
- docs/dashboard-api-reference.html
- docs/files-state-reference.html
- docs/checks-evidence-guide.html
- README.md
- docs/product-specs/product-surface.md
*/
(function () {
  var search = document.getElementById("docSearch");
  var filterTargets = Array.prototype.slice.call(document.querySelectorAll("[data-doc-filter]"));
  var navLinks = Array.prototype.slice.call(document.querySelectorAll(".side-nav a"));
  var sections = navLinks
    .map(function (link) {
      var id = link.getAttribute("href");
      return id && id.charAt(0) === "#" ? document.querySelector(id) : null;
    })
    .filter(Boolean);

  function normalize(value) {
    return (value || "").toLowerCase().replace(/\s+/g, " ").trim();
  }

  function applyFilter() {
    var query = normalize(search.value);
    filterTargets.forEach(function (target) {
      var text = normalize(target.textContent);
      target.classList.toggle("is-hidden", query.length > 0 && text.indexOf(query) === -1);
    });
  }

  if (search) {
    search.addEventListener("input", applyFilter);
  }

  var header = document.querySelector(".site-header");
  var topNav = document.querySelector(".top-nav");

  if (header && topNav) {
    if (!topNav.id) {
      topNav.id = "primary-doc-nav";
    }

    var navToggle = document.createElement("button");
    navToggle.type = "button";
    navToggle.className = "nav-toggle";
    navToggle.setAttribute("aria-controls", topNav.id);
    navToggle.setAttribute("aria-expanded", "false");
    navToggle.textContent = "Menu";
    header.insertBefore(navToggle, topNav);
    header.classList.add("has-nav-toggle");

    navToggle.addEventListener("click", function () {
      var isOpen = header.classList.toggle("is-nav-open");
      navToggle.setAttribute("aria-expanded", isOpen ? "true" : "false");
    });
  }

  document.querySelectorAll(".wide-table table").forEach(function (table) {
    var headers = Array.prototype.slice.call(table.querySelectorAll("thead th"))
      .map(function (header) {
        return (header.textContent || "").replace(/\s+/g, " ").trim();
      });

    if (!headers.some(Boolean)) {
      return;
    }

    table.querySelectorAll("tbody tr").forEach(function (row) {
      Array.prototype.slice.call(row.children).forEach(function (cell, index) {
        if (headers[index]) {
          cell.setAttribute("data-label", headers[index]);
        }
      });
    });
  });

  document.querySelectorAll("[data-copy-target]").forEach(function (button) {
    button.addEventListener("click", function () {
      var target = document.getElementById(button.getAttribute("data-copy-target"));
      if (!target || !navigator.clipboard) {
        return;
      }
      navigator.clipboard.writeText(target.textContent).then(function () {
        var original = button.textContent;
        button.textContent = "Copied";
        window.setTimeout(function () {
          button.textContent = original;
        }, 1400);
      });
    });
  });

  if ("IntersectionObserver" in window) {
    var observer = new IntersectionObserver(function (entries) {
      entries.forEach(function (entry) {
        if (!entry.isIntersecting) {
          return;
        }
        navLinks.forEach(function (link) {
          link.classList.toggle("is-active", link.getAttribute("href") === "#" + entry.target.id);
        });
      });
    }, { rootMargin: "-30% 0px -60% 0px", threshold: 0.01 });

    sections.forEach(function (section) {
      observer.observe(section);
    });
  }
})();
