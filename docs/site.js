/*
MarsDocSync:
docs:
- docs/index.html
- docs/cli-reference.html
- docs/workflows.html
- docs/harness-guide.html
- docs/operations-guide.html
- docs/models-guide.html
- docs/tools-mcp-guide.html
- docs/safety-quality-guide.html
- docs/release-update-guide.html
- docs/integrations-validation-guide.html
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
