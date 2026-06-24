/*
MarsDocSync:
docs:
- README.md
- docs/product-specs/vision.md
- docs/product-specs/product-surface.md
- docs/design-docs/foundation-deployed-harness-architecture.md
- docs/design-docs/delivery-operating-model.md
- docs/design-docs/documentation-sync-architecture.md
- docs/design-docs/self-reflective-telemetry.md
- docs/design-docs/index.md
- docs/design-docs/implementation-language.md
- docs/design-docs/pipeline-engine.md
- docs/design-docs/dashboard.md
- docs/design-docs/code-documentation-map.md
- docs/design-docs/cli-tool-skill-sync.md
- docs/design-docs/dogfood-and-decisions.md
- docs/design-docs/scoring-system.md
- docs/design-docs/validation-matrix-gating.md
- docs/design-docs/tools-glossary.md
- docs/design-docs/guardrails.md
- docs/design-docs/local-inference.md
- docs/design-docs/release-versioning.md
*/
(function () {
  const surfaces = {
    foundation: {
      title: "Foundation harness",
      body:
        "The Mars Harness source repository owns the operating doctrine, generated target defaults, release discipline, product specifications, role registry, and foundation validation evidence.",
      bullets: [
        "Owns source operating model and generated target doctrine.",
        "Classifies findings as foundation-owned, deployed-owned, mixed, or evidence-only.",
        "Does not become the target of its own agents during active target runs."
      ]
    },
    runtime: {
      title: "Runtime substrate",
      body:
        "The compiled Go binary loads manifests, runs the queue, calls tools, records traces, serves the dashboard, manages local inference, and updates release state.",
      bullets: [
        "Executes deterministic commands through CLI, tools, MCP, queue, and worker surfaces.",
        "Captures telemetry, trust, scores, traces, and guardrail outcomes in per-repo SQLite.",
        "Provides capability; doctrine decides how that capability should be used."
      ]
    },
    deployed: {
      title: "Deployed harness",
      body:
        "A target repository receives a generated agent operating system: AGENTS.md, .harness roles, guardrails, skills, knowledge routes, goals, BDD docs, tickets, score artifacts, and release files.",
      bullets: [
        "Mirrors reusable foundation doctrine without copying source-only release mechanics.",
        "Guides agents building the target product inside that target repo.",
        "Can be upgraded to fill missing defaults without overwriting deliberate local policy."
      ]
    },
    target: {
      title: "Target project",
      body:
        "The product being built owns its application behavior, architecture, tests, local docs, package setup, deployment policy, and user-facing release value.",
      bullets: [
        "Receives product changes, validation, and release notes from harness-guided work.",
        "Owns target-specific bugs and project-specific operating policy.",
        "Should not absorb foundation runtime defects as ordinary product backlog noise."
      ]
    }
  };

  const glossary = [
    {
      term: "Mars Harness",
      category: "architecture",
      text:
        "The software factory: source repo, CLI, runtime, local inference, orchestration, tools, telemetry, scoring, dashboard, release tooling, and generated target defaults."
    },
    {
      term: "Foundation harness",
      category: "architecture",
      text:
        "The harness consumed by agents maintaining Mars Harness itself. It owns source doctrine, generated defaults, release rules, and foundation validation evidence."
    },
    {
      term: "Runtime substrate",
      category: "architecture",
      text:
        "The compiled mars-harness binary and internal packages that execute jobs, tools, queues, telemetry, dashboard, scanner, release, and local inference behavior."
    },
    {
      term: "Deployed harness",
      category: "architecture",
      text:
        "The generated .harness, AGENTS.md, docs, tickets, release files, and quality artifacts inside a target repository."
    },
    {
      term: "Target project",
      category: "architecture",
      text:
        "The product repository Mars Harness is building, testing, documenting, and releasing. It owns product behavior and target-specific policy."
    },
    {
      term: "Operating model",
      category: "delivery",
      text:
        "The documented way intent becomes shipped, verifiable work through goals, BDD contracts, active plans, tickets, roles, evidence, release, scoring, and improvement."
    },
    {
      term: "Architecture decision",
      category: "governance",
      text:
        "A durable design choice recorded with an AD number, rationale, trade-off, consequences, and source document so future work can preserve or deliberately change it."
    },
    {
      term: "BDD feature contract",
      category: "delivery",
      text:
        "A Markdown feature artifact in docs/features that defines business logic, step-by-step behavior, scenario schedules, Given/When/Then scenarios, and evidence."
    },
    {
      term: "Walking skeleton",
      category: "delivery",
      text:
        "The implementation strategy: build the thinnest real end-to-end path that proves the next failing BDD scenario."
    },
    {
      term: "MarsDocSync",
      category: "governance",
      text:
        "Top-of-file metadata listing docs that describe or constrain a source file. A code change is incomplete until those docs are read, updated or verified, and audited."
    },
    {
      term: "No stale documentation",
      category: "governance",
      text:
        "The rule that durable docs are live system artifacts and must change with behavior, instead of becoming retrospective notes."
    },
    {
      term: "Universal tool surface",
      category: "governance",
      text:
        "The shared registered tool set exposed through role runs, mars-harness tools run, and MCP so local agents and external clients use the same governed operations."
    },
    {
      term: "Skills",
      category: "governance",
      text:
        "Compact reusable workflow instructions. Skills guide behavior but do not grant tool authority."
    },
    {
      term: "Trust levels",
      category: "governance",
      text:
        "Observer, contributor, and autonomous capability levels. Trust is earned, audited, and revocable per role and repository."
    },
    {
      term: "Guardrails",
      category: "governance",
      text:
        "Mechanical policy checks that constrain file writes, shell commands, secrets, destructive operations, dependency churn, evidence claims, and blast radius."
    },
    {
      term: "Self-reflective telemetry",
      category: "learning",
      text:
        "The loop that turns traces, outcomes, scores, guardrail blocks, no-ops, human follow-up, and stale work into improvement targets."
    },
    {
      term: "Quality score",
      category: "learning",
      text:
        "A repo-visible artifact generated from real outcomes. Missing evidence is shown as insufficient evidence instead of guessed health."
    },
    {
      term: "Failure ownership classification",
      category: "learning",
      text:
        "The required step of routing every finding as foundation-owned, deployed-owned, mixed or unclear, or evidence-only before creating tickets or fixes."
    },
    {
      term: "Mirrored doctrine",
      category: "architecture",
      text:
        "Reusable operating rules that exist in both the foundation harness and initialized target harnesses unless explicitly marked source-only."
    }
  ];

  const drilldowns = {
    operating: {
      title: "BDD operating model",
      summary:
        "Agent work is valuable only when it proves an intended product behavior. Mars Harness turns goals into BDD contracts, plans, tickets, validation evidence, and release notes.",
      problem:
        "Agents can complete tasks that look busy but do not ship the user-visible capability leadership asked for.",
      value:
        "Delivery leaders can ask whether a scenario passed, not whether an agent produced a large diff.",
      mechanism:
        "Goals define outcomes, BDD contracts define behavior, the active plan ranks failing scenarios, and tickets implement the next walking-skeleton slice.",
      proof:
        "Pick one product slice and require the final claim to cite a BDD scenario, validation command, ticket state, and release-note entry.",
      watch:
        "False-done tickets, scenario evidence gaps, and enabler work described as shipped product value."
    },
    docsync: {
      title: "DocSync",
      summary:
        "DocSync makes documentation freshness an explicit part of source work instead of a social reminder after the fact.",
      problem:
        "Autonomous changes can make product docs, design decisions, role guidance, and release notes stale faster than reviewers notice.",
      value:
        "A reviewer can see which durable docs must be checked for a changed file, reducing archaeology and stale-system risk.",
      mechanism:
        "Audited source files declare MarsDocSync metadata near the top of the file. The audit verifies metadata, doc paths, and required package-level docs.",
      proof:
        "In a pilot, change a small source file and require the agent to identify the listed docs, update or verify them, and pass the DocSync audit.",
      watch:
        "Code changes with no doc owner, broad unhelpful doc links, or tickets that say docs were checked without evidence."
    },
    safety: {
      title: "Safety and trust",
      summary:
        "Mars Harness treats autonomy as an earned capability, not a switch. Tool execution is constrained by role, trust, guardrails, and blast radius.",
      problem:
        "A useful agent can still run the wrong command, write outside scope, leak secrets, delete files, or claim completion with broken evidence.",
      value:
        "Security and platform teams get a mechanical policy layer before mutation, not only prompt instructions and post-hoc review.",
      mechanism:
        "Roles run at observer, contributor, or autonomous trust. Mutating tools enforce repo bounds, secret scanning, destructive command blocks, dependency hygiene, and ticket evidence gates.",
      proof:
        "Run a controlled contributor pilot and record every policy block: what was blocked, whether the block was correct, and how the agent recovered.",
      watch:
        "False-positive guardrails, repeated policy loops, broad shell access, or autonomous mutation before the role has enough scored history."
    },
    telemetry: {
      title: "Telemetry and scoring",
      summary:
        "Telemetry is useful only when it changes future behavior. Mars Harness routes failures into improvement targets instead of leaving them as dashboard trivia.",
      problem:
        "Teams see failures, reverts, no-ops, slow runs, and manual fixes but rarely convert them into reusable process improvements.",
      value:
        "Managers can see whether the factory is learning: fewer repeated failures, healthier role scores, and clearer intervention debt.",
      mechanism:
        "Traces, terminal outcomes, guardrail blocks, dogfood findings, quality scores, stale tickets, and human follow-up are triaged by root cause and ownership.",
      proof:
        "After five to ten pilot runs, export the quality score and inspect the top improvement targets. Each should have evidence and an owning route.",
      watch:
        "Scores without sample counts, raw telemetry treated as truth, or foundation-owned runtime issues becoming target product backlog noise."
    },
    local: {
      title: "Local inference",
      summary:
        "The default operating path keeps source code on company-controlled hardware while still allowing explicit model/provider evaluation.",
      problem:
        "Cloud-only agent systems can create data-exposure concerns, variable API costs, and dependence on opaque external model behavior.",
      value:
        "Teams can start with local open models, measure fit on harness-specific tasks, and choose external providers deliberately when they add value.",
      mechanism:
        "The Go binary manages llama.cpp as a subprocess, stores weights outside the repo, verifies model files, detects hardware, and routes roles by model tier.",
      proof:
        "Run setup and doctor on a pilot machine, then compare role latency, successful tool-call JSON, and completion quality under the selected profile.",
      watch:
        "RAM pressure, missing model tiers, slow completion time, or model changes promoted without pinned artifacts and benchmark evidence."
    },
    tools: {
      title: "Tools, skills, and MCP",
      summary:
        "Mars Harness separates authority from guidance: tools perform governed actions, skills teach workflows, and MCP exposes the same tool surface to external clients.",
      problem:
        "Without a shared tool surface, every AI client invents its own shell conventions, increasing risk and making outcomes hard to compare.",
      value:
        "Codex, Cursor, local harness agents, and other MCP-compatible clients can use the same repo-root resolution, trust policy, and JSON argument path.",
      mechanism:
        "Built-in tools are registered, allowlisted per role, trust-gated, and available through active agent runs, `mars-harness tools run`, and `mars-harness mcp serve`.",
      proof:
        "Pick one recurring process such as DocSync audit or ticket creation and verify that both a harness role and an external client use the same tool path.",
      watch:
        "Manual shell steps that should become tools, skills that imply authority they do not have, or tool allowlists that are broader than the role needs."
    }
  };

  const decisions = {
    distribution: {
      title: "Single-binary default with explicit sidecar exceptions",
      summary:
        "Mars Harness optimizes the default operator path for one Go binary, local state, embedded dashboard assets, and a supervised inference subprocess instead of a mandatory SaaS, Node, Python, Redis, or Postgres stack.",
      refs: ["AD-001", "AD-007", "AD-011", "AD-279"],
      value:
        "Teams can trial and operate the factory on controlled hardware without first standing up a platform team around the tool.",
      tradeoff:
        "The default UI and runtime stay deliberately conservative; richer frontend work becomes an explicit sidecar exception.",
      implication:
        "New dependencies must justify their impact on setup, release, offline use, support, and generated target simplicity.",
      proof:
        "Install from a clean machine, run setup and doctor, start a target, and verify no hidden service dependency appears.",
      revisit:
        "Install friction is no longer the adoption bottleneck, or a sidecar produces proven operator value without weakening the core path.",
      sources: "implementation-language.md, local-inference.md, dashboard.md"
    },
    boundary: {
      title: "Foundation and deployed harnesses stay separate",
      summary:
        "The source harness, runtime binary, deployed target harness, and target product each own different artifacts and failures. The shared operating model mirrors reusable doctrine without leaking source-only mechanics into target repos.",
      refs: ["AD-139", "AD-252"],
      value:
        "Teams avoid turning runtime defects into product backlog noise, and they can tell whether a fix belongs in Mars Harness, generated defaults, or the target application.",
      tradeoff:
        "Every finding needs an ownership classification step before ticketing or patching, which adds discipline to early runs.",
      implication:
        "Reviewers should ask whether a failure is foundation-owned, deployed-owned, mixed, or evidence-only before approving remediation.",
      proof:
        "Run one target pilot and classify every observed failure. A healthy loop should route source issues to foundation evidence and target issues to target work.",
      revisit:
        "Ownership classification repeatedly blocks delivery without improving routing accuracy, or target teams need a deliberate local override.",
      sources: "foundation-deployed-harness-architecture.md, delivery-operating-model.md"
    },
    state: {
      title: "SQLite keeps state local, durable, and repo-scoped",
      summary:
        "Jobs, traces, scores, telemetry, schedules, and repo registry state live in SQLite rather than a mandatory external database. Schema includes repo identity early, and startup preserves WAL sidecars as recoverable state.",
      refs: ["AD-009", "AD-010", "AD-029", "AD-214"],
      value:
        "The factory can survive restarts and support multiple target repos without requiring Postgres, Redis, or hosted infrastructure.",
      tradeoff:
        "The first architecture optimizes single-machine reliability and clear backup paths, not distributed multi-tenant scale.",
      implication:
        "Operators should treat the per-repo database as important run evidence and avoid cleanup flows that erase queue or telemetry history.",
      proof:
        "Start a target, interrupt and restart, then verify pending work, trace state, and repo identity recover instead of reseeding as new work.",
      revisit:
        "Many concurrent repos outgrow single-machine contention, or enterprise deployment requires a managed state backend.",
      sources: "pipeline-engine.md, dogfood-and-decisions.md"
    },
    operating: {
      title: "BDD-led delivery turns agent activity into product proof",
      summary:
        "Goals, BDD feature contracts, the active plan, ticket lifecycle, validation evidence, and release notes form the path from intent to shipped work.",
      refs: ["AD-097", "AD-108", "AD-138"],
      value:
        "Delivery leaders can ask whether a user-visible scenario passed instead of interpreting diff size or agent confidence.",
      tradeoff:
        "Teams pay an up-front cost to write behavior contracts and maintain plans before broad implementation work.",
      implication:
        "A ticket is not done until the scenario evidence, repo state, docs, and release trail support the completion claim.",
      proof:
        "Pick one bounded feature and require the final handoff to cite the scenario, command evidence, ticket move, and release-note entry.",
      revisit:
        "The documentation overhead exceeds the review waste it removes, or a team has an equivalent artifact that can be mechanically checked.",
      sources: "delivery-operating-model.md, validation-matrix-gating.md"
    },
    docsync: {
      title: "DocSync makes documentation freshness mechanical",
      summary:
        "Audited files declare their owning docs through MarsDocSync metadata. Agents must read, update or verify those docs, then run the audit before completion.",
      refs: ["AD-098", "AD-101", "AD-102"],
      value:
        "Reviewers get a concrete doc checklist for changed code, reducing stale architecture, stale product specs, and repeated archaeology.",
      tradeoff:
        "Source files carry metadata, and changes that look code-only may require doc review before they can land.",
      implication:
        "Every new audited source file needs a meaningful doc owner, not a vague link chosen only to satisfy the parser.",
      proof:
        "Change a small JS or Go file in the pilot and verify that the agent names the right docs, updates or confirms them, and passes docsync audit.",
      revisit:
        "Metadata becomes noisy or broad enough that reviewers no longer trust it as a useful checklist.",
      sources: "documentation-sync-architecture.md, code-documentation-map.md"
    },
    safety: {
      title: "Deterministic guardrails come before semantic ambition",
      summary:
        "The first hard safety layer uses path, regex, file-existence, workspace hygiene, lifecycle, dependency, and blast-radius checks before relying on deeper semantic analysis.",
      refs: ["AD-012", "AD-107", "AD-115", "AD-142"],
      value:
        "Security and platform teams get fast pre-mutation blocks for destructive, noisy, or misleading actions instead of prompt-only governance.",
      tradeoff:
        "Syntactic rules can miss nuanced semantic defects and can produce false positives when policy is too broad.",
      implication:
        "Policy blocks should be measured and tuned; repeated false positives are product feedback, not an excuse to remove the safety layer.",
      proof:
        "Track pilot guardrail blocks by class, correctness, recovery path, and whether the block happened before harmful mutation.",
      revisit:
        "False positives routinely stop valid work, or the highest-risk failures are semantic and require a richer analyzer.",
      sources: "guardrails.md, delivery-operating-model.md"
    },
    telemetry: {
      title: "Telemetry and scores route improvement work",
      summary:
        "Traces, outcomes, guardrail blocks, no-ops, timeouts, human follow-up, quality scores, and stale tickets are classified into improvement targets before becoming work.",
      refs: ["AD-037", "AD-038", "AD-065", "AD-072"],
      value:
        "Managers can see whether the factory is learning from failures instead of collecting dashboards that do not change behavior.",
      tradeoff:
        "The system must preserve enough evidence and sample counts to avoid overreacting to one noisy run.",
      implication:
        "Low scores are inspection triggers. They should route to prompt, skill, guardrail, context, inference, manifest, tool policy, or process work based on evidence.",
      proof:
        "After several pilot runs, export the quality score and confirm that repeated failure categories have specific owners and acceptance criteria.",
      revisit:
        "Scores fail to predict review pain, or telemetry creates more triage overhead than repeated failure reduction.",
      sources: "self-reflective-telemetry.md, scoring-system.md"
    },
    tools: {
      title: "Universal tools separate authority from instructions",
      summary:
        "Mars Harness exposes governed operations through role runs, mars-harness tools run, and MCP. Skills teach workflows, while tools carry authority, policy, and structured arguments.",
      refs: ["AD-022", "AD-077", "AD-103"],
      value:
        "Different AI clients can use the same safe operations instead of each inventing shell conventions that are hard to audit.",
      tradeoff:
        "Useful repeated workflows must be promoted into registered tools or maintained skills, which requires inventory discipline.",
      implication:
        "When a process is repeated, risky, validation-heavy, or likely to recur, it should become a first-class tool rather than chat memory.",
      proof:
        "Have a harness role and an external MCP client perform the same DocSync or ticket-create action through the same tool surface.",
      revisit:
        "Tool upkeep exceeds the risk reduction, or external clients need capabilities that cannot be expressed through the shared registry.",
      sources: "tools-glossary.md, cli-tool-skill-sync.md, dogfood-and-decisions.md"
    },
    release: {
      title: "Versioned release evidence follows source work",
      summary:
        "Semantic commits are followed by generated version and changelog updates. Source releases are tagged, assets are built locally, checksums are verified, and GitHub mirroring is optional.",
      refs: ["AD-049", "AD-050", "AD-056", "AD-059", "AD-078", "AD-099", "AD-141"],
      value:
        "Stakeholders can inspect what changed, why it mattered, what impact to expect, and whether the shipped binary assets match the release.",
      tradeoff:
        "Even documentation changes carry versioning overhead in the source harness, which slows casual edits but improves traceability.",
      implication:
        "Completion is not only a code commit. It includes release-note generation, tag discipline, local asset publication, and verification or an explicit blocker.",
      proof:
        "Audit one release from semantic commit through changelog, VERSION, tag, checksums, and release asset verification.",
      revisit:
        "The source release cadence becomes too heavy for low-risk doc-only changes, or target repos need a lighter local policy.",
      sources: "release-versioning.md, product-surface.md"
    },
    dashboard: {
      title: "Embedded dashboard stays core; richer control plane is optional",
      summary:
        "The current dashboard remains embedded and offline-capable. A future TanStack control plane can exist as an operator-installed sidecar behind the Go gateway without weakening the core single-binary path.",
      refs: ["AD-011", "AD-156", "AD-279"],
      value:
        "Operators get local visibility immediately, while the product can still evolve toward a richer UI when evidence justifies the sidecar cost.",
      tradeoff:
        "The product carries two dashboard horizons: a conservative shipped surface and a deferred optional sidecar plan.",
      implication:
        "Dashboard proposals must state whether they affect the core embedded path or the optional sidecar, and they must keep Go as the trusted gateway.",
      proof:
        "Run the embedded dashboard without Node installed, then evaluate whether a sidecar prototype improves operator decisions enough to justify prerequisites.",
      revisit:
        "The embedded dashboard prevents critical operational workflows, or the sidecar proves enough value to justify explicit operator installation.",
      sources: "dashboard.md, product-surface.md"
    }
  };

  const navToggle = document.querySelector(".nav-toggle");
  const mobileNav = document.querySelector("#mobileNav");
  if (navToggle && mobileNav) {
    navToggle.addEventListener("click", () => {
      const isOpen = mobileNav.classList.toggle("is-open");
      navToggle.setAttribute("aria-expanded", String(isOpen));
    });
    mobileNav.addEventListener("click", (event) => {
      if (event.target.matches("a")) {
        mobileNav.classList.remove("is-open");
        navToggle.setAttribute("aria-expanded", "false");
      }
    });
  }

  const progress = document.querySelector("#pageProgress");
  const setProgress = () => {
    const scrollTop = window.scrollY;
    const docHeight = document.documentElement.scrollHeight - window.innerHeight;
    const percent = docHeight > 0 ? (scrollTop / docHeight) * 100 : 0;
    if (progress) {
      progress.style.width = `${Math.min(100, Math.max(0, percent))}%`;
    }
  };
  window.addEventListener("scroll", setProgress, { passive: true });
  setProgress();

  const observedSections = document.querySelectorAll(".section-observed[id]");
  const navLinks = [...document.querySelectorAll(".nav-links a")];
  if ("IntersectionObserver" in window) {
    const sectionObserver = new IntersectionObserver(
      (entries) => {
        const visible = entries
          .filter((entry) => entry.isIntersecting)
          .sort((a, b) => b.intersectionRatio - a.intersectionRatio)[0];
        if (!visible) return;
        navLinks.forEach((link) => {
          const isActive = link.getAttribute("href") === `#${visible.target.id}`;
          link.classList.toggle("is-active", isActive);
        });
      },
      { threshold: [0.2, 0.45, 0.7], rootMargin: "-80px 0px -55% 0px" }
    );
    observedSections.forEach((section) => sectionObserver.observe(section));
  }

  const surfaceButtons = document.querySelectorAll(".surface-card");
  const surfaceDetail = document.querySelector("#surfaceDetail");
  surfaceButtons.forEach((button) => {
    button.addEventListener("click", () => {
      const key = button.dataset.surface;
      const data = surfaces[key];
      if (!data || !surfaceDetail) return;
      surfaceButtons.forEach((item) => item.classList.toggle("is-active", item === button));
      surfaceDetail.innerHTML = `
        <p class="eyebrow">Selected surface</p>
        <h3>${data.title}</h3>
        <p>${data.body}</p>
        <ul>${data.bullets.map((bullet) => `<li>${bullet}</li>`).join("")}</ul>
      `;
    });
  });

  document.querySelectorAll(".concept-toggle").forEach((button) => {
    button.addEventListener("click", () => {
      const card = button.closest(".concept-card");
      const isOpen = card.classList.toggle("is-open");
      button.setAttribute("aria-expanded", String(isOpen));
    });
  });

  const decisionTabs = document.querySelectorAll(".decision-tab");
  const decisionPanel = document.querySelector("#decisionPanel");
  decisionTabs.forEach((button) => {
    button.addEventListener("click", () => {
      const data = decisions[button.dataset.decision];
      if (!data || !decisionPanel) return;
      decisionTabs.forEach((item) => item.classList.toggle("is-active", item === button));
      decisionPanel.innerHTML = `
        <div class="decision-panel-header">
          <p class="eyebrow">Decision cluster</p>
          <h3>${data.title}</h3>
          <p>${data.summary}</p>
          <div class="decision-refs" aria-label="Architecture decision references">
            ${data.refs.map((ref) => `<span>${ref}</span>`).join("")}
          </div>
        </div>
        <div class="decision-columns">
          <div>
            <h4>Company value</h4>
            <p>${data.value}</p>
          </div>
          <div>
            <h4>Trade-off</h4>
            <p>${data.tradeoff}</p>
          </div>
          <div>
            <h4>Operating implication</h4>
            <p>${data.implication}</p>
          </div>
          <div>
            <h4>Pilot proof</h4>
            <p>${data.proof}</p>
          </div>
          <div>
            <h4>Revisit when</h4>
            <p>${data.revisit}</p>
          </div>
          <div>
            <h4>Source docs</h4>
            <p>${data.sources}</p>
          </div>
        </div>
      `;
    });
  });

  const drilldownTabs = document.querySelectorAll(".drilldown-tab");
  const drilldownPanel = document.querySelector("#drilldownPanel");
  drilldownTabs.forEach((button) => {
    button.addEventListener("click", () => {
      const data = drilldowns[button.dataset.drill];
      if (!data || !drilldownPanel) return;
      drilldownTabs.forEach((item) => item.classList.toggle("is-active", item === button));
      drilldownPanel.innerHTML = `
        <div class="drilldown-panel-header">
          <p class="eyebrow">Selected drill-down</p>
          <h3>${data.title}</h3>
          <p>${data.summary}</p>
        </div>
        <div class="drilldown-columns">
          <div>
            <h4>Problem it solves</h4>
            <p>${data.problem}</p>
          </div>
          <div>
            <h4>Company value</h4>
            <p>${data.value}</p>
          </div>
          <div>
            <h4>How it works</h4>
            <p>${data.mechanism}</p>
          </div>
          <div>
            <h4>Proof in a pilot</h4>
            <p>${data.proof}</p>
          </div>
        </div>
        <div class="drilldown-proof">
          <strong>Watch for:</strong>
          <span>${data.watch}</span>
        </div>
      `;
    });
  });

  const glossaryList = document.querySelector("#glossaryList");
  const glossarySearch = document.querySelector("#glossarySearch");
  const filterChips = document.querySelectorAll(".filter-chip");
  let activeFilter = "all";

  function renderGlossary() {
    if (!glossaryList) return;
    const query = (glossarySearch && glossarySearch.value ? glossarySearch.value : "").trim().toLowerCase();
    const filtered = glossary.filter((item) => {
      const matchesFilter = activeFilter === "all" || item.category === activeFilter;
      const haystack = `${item.term} ${item.category} ${item.text}`.toLowerCase();
      return matchesFilter && (!query || haystack.includes(query));
    });
    glossaryList.innerHTML = filtered.length
      ? filtered
          .map(
            (item) => `
              <article class="glossary-item">
                <strong>${item.term}</strong>
                <p>${item.text}</p>
                <span class="tag">${item.category}</span>
              </article>
            `
          )
          .join("")
      : `<article class="glossary-item"><strong>No matches</strong><p>Try another term or clear the filter.</p></article>`;
  }

  filterChips.forEach((chip) => {
    chip.addEventListener("click", () => {
      activeFilter = chip.dataset.filter || "all";
      filterChips.forEach((item) => item.classList.toggle("is-active", item === chip));
      renderGlossary();
    });
  });

  if (glossarySearch) {
    glossarySearch.addEventListener("input", renderGlossary);
  }
  renderGlossary();

  const canvas = document.querySelector("#signalCanvas");
  const prefersReducedMotion = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
  if (canvas && canvas.getContext) {
    const ctx = canvas.getContext("2d");
    const w = canvas.width;
    const h = canvas.height;
    const nodes = [
      { x: 104, y: 250, label: "Trace", color: "#0f8c82" },
      { x: 258, y: 132, label: "Score", color: "#c85f24" },
      { x: 282, y: 374, label: "Guardrail", color: "#2f7d44" },
      { x: 482, y: 160, label: "Triage", color: "#705ca8" },
      { x: 548, y: 346, label: "Improvement", color: "#a33d57" }
    ];
    const links = [
      [0, 1],
      [0, 2],
      [1, 3],
      [2, 3],
      [3, 4],
      [4, 0]
    ];
    const packets = links.map((link, index) => ({ link, t: index / links.length, speed: 0.0024 + index * 0.00028 }));

    function draw(time) {
      ctx.clearRect(0, 0, w, h);
      const gradient = ctx.createLinearGradient(0, 0, w, h);
      gradient.addColorStop(0, "#171916");
      gradient.addColorStop(0.55, "#20231f");
      gradient.addColorStop(1, "#252724");
      ctx.fillStyle = gradient;
      ctx.fillRect(0, 0, w, h);

      ctx.strokeStyle = "rgba(255,253,245,0.07)";
      ctx.lineWidth = 1;
      for (let x = 40; x < w; x += 56) {
        ctx.beginPath();
        ctx.moveTo(x, 30);
        ctx.lineTo(x, h - 30);
        ctx.stroke();
      }
      for (let y = 42; y < h; y += 56) {
        ctx.beginPath();
        ctx.moveTo(30, y);
        ctx.lineTo(w - 30, y);
        ctx.stroke();
      }

      links.forEach(([from, to]) => {
        const a = nodes[from];
        const b = nodes[to];
        ctx.strokeStyle = "rgba(255,253,245,0.22)";
        ctx.lineWidth = 2;
        ctx.beginPath();
        ctx.moveTo(a.x, a.y);
        ctx.lineTo(b.x, b.y);
        ctx.stroke();
      });

      packets.forEach((packet) => {
        if (!prefersReducedMotion) {
          packet.t = (packet.t + packet.speed) % 1;
        }
        const a = nodes[packet.link[0]];
        const b = nodes[packet.link[1]];
        const x = a.x + (b.x - a.x) * packet.t;
        const y = a.y + (b.y - a.y) * packet.t;
        ctx.fillStyle = "rgba(243,177,79,0.92)";
        ctx.beginPath();
        ctx.arc(x, y, 5, 0, Math.PI * 2);
        ctx.fill();
      });

      nodes.forEach((node) => {
        ctx.fillStyle = node.color;
        ctx.beginPath();
        ctx.arc(node.x, node.y, 27, 0, Math.PI * 2);
        ctx.fill();
        ctx.strokeStyle = "rgba(255,253,245,0.56)";
        ctx.lineWidth = 2;
        ctx.stroke();
        ctx.fillStyle = "#fffdf5";
        ctx.font = "700 13px system-ui, sans-serif";
        ctx.textAlign = "center";
        ctx.fillText(node.label, node.x, node.y + 51);
      });

      ctx.fillStyle = "rgba(255,253,245,0.82)";
      ctx.font = "800 22px system-ui, sans-serif";
      ctx.textAlign = "left";
      ctx.fillText("Telemetry routes evidence into improvement", 34, 44);
      ctx.fillStyle = "rgba(255,253,245,0.58)";
      ctx.font = "14px system-ui, sans-serif";
      ctx.fillText("Local signals stay local unless anonymous aggregate reporting is explicitly enabled.", 34, 70);

      if (!prefersReducedMotion) {
        window.requestAnimationFrame(draw);
      }
    }

    draw(0);
  }
})();
