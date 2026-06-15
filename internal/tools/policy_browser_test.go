/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/delivery-operating-model.md
- docs/design-docs/guardrails.md
- docs/design-docs/tools-glossary.md
- docs/features/F-005-agent-execution-runtime.md
- docs/features/F-007-guardrails-and-safety.md
*/
package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBrowserProductSmokeGuidanceUsesReactSourceSmoke(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"scripts":{"build":"vite build"},"dependencies":{"react":"latest","react-dom":"latest","vite":"latest"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte(`<script type="module" src="/src/main.jsx"></script>`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "src", "main.jsx"), []byte(`createRoot(document.getElementById('root'));`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "src", "App.jsx"), []byte(`export default function App(){return <main id="game">Score</main>}`), 0o644); err != nil {
		t.Fatal(err)
	}
	root, err := NewRoot(dir)
	if err != nil {
		t.Fatal(err)
	}

	got := browserProductSmokeCommandGuidance(root)
	if !strings.Contains(got, "browser smoke: React document.querySelector #game score UI state") ||
		!strings.Contains(got, "src/App.jsx") ||
		!strings.Contains(got, "createRoot") {
		t.Fatalf("expected React source smoke guidance, got %s", got)
	}
}

func TestDogfoodPostBuildRequiresReactProductSmokeBeforeMoreShell(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"scripts":{"build":"vite build"},"dependencies":{"react":"latest","react-dom":"latest","vite":"latest"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte(`<main id="root"></main><script type="module" src="/src/main.jsx"></script>`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "src", "main.jsx"), []byte(`import { createRoot } from 'react-dom/client'; createRoot(document.getElementById('root'));`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "src", "App.jsx"), []byte(`export default function App(){return <main id="game">Score</main>}`), 0o644); err != nil {
		t.Fatal(err)
	}
	root, err := NewRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	ctx := WithSession(context.Background(), Session{Role: "dogfood", ToolCounts: map[string]int{buildCommandSuccessKey: 1}})
	distRaw, err := json.Marshal(shellExecArgs{Argv: []string{"ls", "-la", "dist"}})
	if err != nil {
		t.Fatal(err)
	}
	err = preToolPolicy(ctx, root, "shell_exec", distRaw)
	if err == nil {
		t.Fatal("expected dogfood post-build dist inspection to be blocked")
	}
	if !strings.Contains(err.Error(), "dogfood has successful browser-framework build evidence") ||
		!strings.Contains(err.Error(), "browser smoke: React document.querySelector #game score UI state") ||
		!strings.Contains(err.Error(), "Do not inspect dist/assets") {
		t.Fatalf("expected React dogfood source-smoke guidance, got %v", err)
	}

	smokeRaw, err := json.Marshal(shellExecArgs{Argv: []string{"node", "-e", "console.log('browser smoke: React document.querySelector #game score UI state')"}})
	if err != nil {
		t.Fatal(err)
	}
	if err := preToolPolicy(ctx, root, "shell_exec", smokeRaw); err != nil {
		t.Fatalf("expected React browser-product smoke to be allowed, got %v", err)
	}
}

func TestEngineerPostValidationAllowsMissingBrowserBuildAfterCommit(t *testing.T) {
	t.Parallel()
	requireGit(t)
	dir, root := setupPolicyTicketRepo(t)
	initGitRepo(t, dir)
	writePhaserBrief(t, dir)
	writePolicyFeature(t, dir, "F-001-product-walking-skeleton.md")
	writePhaserPackage(t, dir, true)
	writePolicyTicket(t, dir, "in-progress", "T-001-phaser.md", `---
id: T-001
title: Ship Phaser slice
work_type: feature
bdd_scenarios:
- F-001-S001
end_to_end_evidence: required
evidence_links:
- npm run build
verified_by:
- engineer
blocker: none
blocked_by: []
---

# Ship Phaser slice
`)
	if err := runGitExit0(context.Background(), root, "add", "."); err != nil {
		t.Fatalf("git add: %v", err)
	}
	if err := runGitExit0(context.Background(), root, "commit", "-m", "feat: implement phaser slice"); err != nil {
		t.Fatalf("git commit: %v", err)
	}
	ctx := WithSession(context.Background(), Session{Role: "engineer", ToolCounts: map[string]int{
		validationCommandSuccessKey: 1,
		"tool:git_commit:success":   1,
	}})

	err := preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":["npm","run","build"]}`))
	if err != nil {
		t.Fatalf("expected missing browser build validation to remain allowed after implementation commit, got %v", err)
	}
}

func TestEngineerPostValidationAllowsMissingBrowserSmokeAfterBuild(t *testing.T) {
	t.Parallel()
	requireGit(t)
	dir, root := setupPolicyTicketRepo(t)
	initGitRepo(t, dir)
	writePhaserBrief(t, dir)
	writePolicyFeature(t, dir, "F-001-product-walking-skeleton.md")
	writePhaserPackage(t, dir, true)
	writePolicyTicket(t, dir, "in-progress", "T-001-phaser.md", `---
id: T-001
title: Ship Phaser slice
work_type: feature
bdd_scenarios:
- F-001-S001
end_to_end_evidence: required
evidence_links:
- npm run build
- browser smoke
verified_by:
- engineer
blocker: none
blocked_by: []
---

# Ship Phaser slice
`)
	if err := runGitExit0(context.Background(), root, "add", "."); err != nil {
		t.Fatalf("git add: %v", err)
	}
	if err := runGitExit0(context.Background(), root, "commit", "-m", "feat: implement phaser slice"); err != nil {
		t.Fatalf("git commit: %v", err)
	}
	ctx := WithSession(context.Background(), Session{Role: "engineer", ToolCounts: map[string]int{
		validationCommandSuccessKey: 1,
		buildCommandSuccessKey:      1,
		"tool:git_commit:success":   1,
	}})
	raw, err := json.Marshal(shellExecArgs{Argv: []string{
		"node", "-e", "console.log('browser smoke: Phaser canvas #game new Phaser.Game')",
	}})
	if err != nil {
		t.Fatalf("marshal shell args: %v", err)
	}

	err = preToolPolicy(ctx, root, "shell_exec", raw)
	if err != nil {
		t.Fatalf("expected missing browser smoke validation to remain allowed after build, got %v", err)
	}
}

func TestEngineerPostBuildBrowserFrameworkBlocksSmokeSubstitutesWhileDirty(t *testing.T) {
	t.Parallel()
	requireGit(t)
	dir, root := setupPolicyTicketRepo(t)
	initGitRepo(t, dir)
	writePhaserBrief(t, dir)
	writePolicyFeature(t, dir, "F-001-product-walking-skeleton.md")
	writePhaserPackage(t, dir, true)
	writeValidPhaserSource(t, dir)
	writePolicyTicket(t, dir, "in-progress", "T-001-phaser.md", `---
id: T-001
title: Ship Phaser slice
work_type: feature
bdd_scenarios:
- F-001-S001
end_to_end_evidence: required
evidence_links: []
verified_by: TBD
blocker: none
blocked_by: []
---

# Ship Phaser slice
`)
	if err := runGitExit0(context.Background(), root, "add", "."); err != nil {
		t.Fatalf("git add: %v", err)
	}
	if err := runGitExit0(context.Background(), root, "commit", "-m", "chore: seed phaser project"); err != nil {
		t.Fatalf("git commit: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "package-lock.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("write dirty package lock: %v", err)
	}
	ctx := WithSession(context.Background(), Session{Role: "engineer", ToolCounts: map[string]int{
		validationCommandSuccessKey: 2,
		buildCommandSuccessKey:      1,
	}})

	err := preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":["node","-e","const fs=require('fs'); const js=fs.readFileSync('dist/assets/index.js','utf8'); console.log(js.length > 0)"]}`))
	if err == nil {
		t.Fatal("expected post-build browser-framework smoke substitute to be blocked")
	}
	if !strings.Contains(err.Error(), "still needs browser-product smoke") ||
		!strings.Contains(err.Error(), "browser smoke: Phaser canvas #game new Phaser.Game") ||
		!strings.Contains(err.Error(), "require('phaser')") ||
		!strings.Contains(err.Error(), "dist/assets") {
		t.Fatalf("expected browser smoke guidance, got %v", err)
	}

	err = preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":["npm","run","build"]}`))
	if err != nil {
		t.Fatalf("expected build rerun to remain allowed, got %v", err)
	}
	err = preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":["node","-e","const fs=require('fs'); const main=fs.readFileSync('src/main.js','utf8'); if(!main.includes('new Phaser.Game')) throw new Error('missing game'); console.log('browser smoke: Phaser canvas #game new Phaser.Game')"]}`))
	if err != nil {
		t.Fatalf("expected canonical browser product smoke to remain allowed, got %v", err)
	}
}

func TestEngineerPostValidationBrowserEvidenceBlocksDirtyExploration(t *testing.T) {
	t.Parallel()
	requireGit(t)
	dir, root := setupPolicyTicketRepo(t)
	initGitRepo(t, dir)
	writePhaserBrief(t, dir)
	writePolicyFeature(t, dir, "F-001-product-walking-skeleton.md")
	writePhaserPackage(t, dir, true)
	writeValidPhaserSource(t, dir)
	writePolicyTicket(t, dir, "in-progress", "T-001-phaser.md", `---
id: T-001
title: Ship Phaser slice
work_type: feature
bdd_scenarios:
- F-001-S001
end_to_end_evidence: required
evidence_links:
- npm run build
- browser smoke
verified_by:
- engineer
blocker: none
blocked_by: []
---

# Ship Phaser slice
`)
	if err := runGitExit0(context.Background(), root, "add", "."); err != nil {
		t.Fatalf("git add: %v", err)
	}
	if err := runGitExit0(context.Background(), root, "commit", "-m", "feat: implement phaser slice"); err != nil {
		t.Fatalf("git commit: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "package-lock.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("write dirty package lock: %v", err)
	}
	ctx := WithSession(context.Background(), Session{Role: "engineer", ToolCounts: map[string]int{
		validationCommandSuccessKey:   3,
		buildCommandSuccessKey:        1,
		browserProductSmokeSuccessKey: 1,
		"tool:git_commit:success":     1,
	}})

	err := preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":["find","dist/assets","-name","*.js"]}`))
	if err == nil {
		t.Fatal("expected post-browser-validation exploration with dirty work to be blocked")
	}
	if !strings.Contains(err.Error(), "browser-framework build and product-smoke validation") ||
		!strings.Contains(err.Error(), "package-lock.json") ||
		!strings.Contains(err.Error(), "git_commit") ||
		!strings.Contains(err.Error(), "docs/tickets/done") ||
		!strings.Contains(err.Error(), "job_disposition_record") {
		t.Fatalf("expected browser-framework dirty-work convergence guidance, got %v", err)
	}
}

func TestEngineerBrowserFrameworkTicketEvidenceRequiresBuildScript(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePhaserPackage(t, dir, false)
	writePolicyTicket(t, dir, "in-progress", "T-001-phaser-tetris.md", `---
id: T-001
title: Ship Phaser Tetris
work_type: feature
bdd_scenarios: ["F-001-S002"]
end_to_end_evidence: required
evidence_links: []
verified_by: TBD
---

# Ship Phaser Tetris
`)
	path := filepath.Join("docs", "tickets", "in-progress", "T-001-phaser-tetris.md")
	content := `---
id: T-001
title: Ship Phaser Tetris
work_type: feature
bdd_scenarios: ["F-001-S002"]
end_to_end_evidence: required
evidence_links:
- curl -fsS http://127.0.0.1:8080/
verified_by:
- engineer
---

# Ship Phaser Tetris
`
	ctx := WithSession(context.Background(), Session{Role: "engineer", ToolCounts: map[string]int{
		validationCommandSuccessKey: 1,
	}})
	raw, err := json.Marshal(fileWriteArgs{Path: path, Content: content})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	err = preToolPolicy(ctx, root, "file_write", raw)
	if err == nil {
		t.Fatal("expected browser-framework evidence write to require build script")
	}
	if !strings.Contains(err.Error(), "browser-framework") || !strings.Contains(err.Error(), "no build script") {
		t.Fatalf("expected build-script guidance, got %v", err)
	}
}

func TestEngineerBrowserFrameworkTicketEvidenceRequiresBuildSuccess(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePhaserPackage(t, dir, true)
	writePolicyTicket(t, dir, "in-progress", "T-001-phaser-tetris.md", `---
id: T-001
title: Ship Phaser Tetris
work_type: feature
bdd_scenarios: ["F-001-S002"]
end_to_end_evidence: required
evidence_links: []
verified_by: TBD
---

# Ship Phaser Tetris
`)
	path := filepath.Join("docs", "tickets", "in-progress", "T-001-phaser-tetris.md")
	content := `---
id: T-001
title: Ship Phaser Tetris
work_type: feature
bdd_scenarios: ["F-001-S002"]
end_to_end_evidence: required
evidence_links:
- curl -fsS http://127.0.0.1:8080/
verified_by:
- engineer
---

# Ship Phaser Tetris
`
	ctx := WithSession(context.Background(), Session{Role: "engineer", ToolCounts: map[string]int{
		validationCommandSuccessKey: 1,
	}})
	raw, err := json.Marshal(fileWriteArgs{Path: path, Content: content})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	err = preToolPolicy(ctx, root, "file_write", raw)
	if err == nil {
		t.Fatal("expected browser-framework evidence write to require build success")
	}
	if !strings.Contains(err.Error(), "npm run build") || !strings.Contains(err.Error(), "has not passed") {
		t.Fatalf("expected build-success guidance, got %v", err)
	}
}

func TestEngineerBrowserFrameworkTicketEvidenceRejectsNoopBuildScript(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePhaserPackageWithScripts(t, dir, map[string]string{
		"start": "http-server -p 8080",
		"build": "echo 'Building Phaser Tetris Demo...'",
	}, map[string]string{"phaser": "^3.70.0"}, nil)
	writePolicyTicket(t, dir, "in-progress", "T-001-phaser-tetris.md", `---
id: T-001
title: Ship Phaser Tetris
work_type: feature
bdd_scenarios: ["F-001-S002"]
end_to_end_evidence: required
evidence_links: []
verified_by: TBD
---

# Ship Phaser Tetris
`)
	path := filepath.Join("docs", "tickets", "in-progress", "T-001-phaser-tetris.md")
	content := `---
id: T-001
title: Ship Phaser Tetris
work_type: feature
bdd_scenarios: ["F-001-S002"]
end_to_end_evidence: required
evidence_links:
- npm run build
verified_by:
- engineer
---

# Ship Phaser Tetris
`
	ctx := WithSession(context.Background(), Session{Role: "engineer", ToolCounts: map[string]int{
		validationCommandSuccessKey: 1,
		buildCommandSuccessKey:      1,
	}})
	raw, err := json.Marshal(fileWriteArgs{Path: path, Content: content})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	err = preToolPolicy(ctx, root, "file_write", raw)
	if err == nil {
		t.Fatal("expected browser-framework evidence write to reject no-op build script")
	}
	if !strings.Contains(err.Error(), "no-op") || !strings.Contains(err.Error(), "vite build") {
		t.Fatalf("expected no-op build guidance, got %v", err)
	}
}

func TestEngineerBrowserFrameworkTicketEvidenceRejectsSyntaxOnlyBuildScript(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePhaserPackageWithScripts(t, dir, map[string]string{
		"start": "vite --host 127.0.0.1",
		"build": "node --check src/main.js && node --check src/game.js",
	}, map[string]string{"phaser": "^3.70.0"}, map[string]string{"vite": "^5.0.0"})
	writePolicyTicket(t, dir, "in-progress", "T-001-phaser-tetris.md", `---
id: T-001
title: Ship Phaser Tetris
work_type: feature
bdd_scenarios: ["F-001-S002"]
end_to_end_evidence: required
evidence_links: []
verified_by: TBD
---

# Ship Phaser Tetris
`)
	path := filepath.Join("docs", "tickets", "in-progress", "T-001-phaser-tetris.md")
	content := `---
id: T-001
title: Ship Phaser Tetris
work_type: feature
bdd_scenarios: ["F-001-S002"]
end_to_end_evidence: required
evidence_links:
- npm run build
verified_by:
- engineer
---

# Ship Phaser Tetris
`
	ctx := WithSession(context.Background(), Session{Role: "engineer", ToolCounts: map[string]int{
		validationCommandSuccessKey: 1,
		buildCommandSuccessKey:      1,
	}})
	raw, err := json.Marshal(fileWriteArgs{Path: path, Content: content})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	err = preToolPolicy(ctx, root, "file_write", raw)
	if err == nil {
		t.Fatal("expected browser-framework evidence write to reject syntax-only build script")
	}
	if !strings.Contains(err.Error(), "no-op") || !strings.Contains(err.Error(), "vite build") {
		t.Fatalf("expected syntax-only build guidance, got %v", err)
	}
}

func TestEngineerBrowserFrameworkTicketEvidenceRejectsCopyOnlyBuildScript(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePhaserPackageWithScripts(t, dir, map[string]string{
		"start": "live-server --port=8080 --host=127.0.0.1 --open=src/index.html",
		"build": "mkdir -p dist && cp src/index.html dist/index.html && echo 'Build completed successfully'",
	}, map[string]string{"phaser": "^3.70.0"}, map[string]string{"live-server": "^1.2.2"})
	writePolicyTicket(t, dir, "in-progress", "T-001-phaser-tetris.md", `---
id: T-001
title: Ship Phaser Tetris
work_type: feature
bdd_scenarios: ["F-001-S002"]
end_to_end_evidence: required
evidence_links: []
verified_by: TBD
---

# Ship Phaser Tetris
`)
	path := filepath.Join("docs", "tickets", "in-progress", "T-001-phaser-tetris.md")
	content := `---
id: T-001
title: Ship Phaser Tetris
work_type: feature
bdd_scenarios: ["F-001-S002"]
end_to_end_evidence: required
evidence_links:
- npm run build
- browser product smoke
verified_by:
- engineer
---

# Ship Phaser Tetris
`
	ctx := WithSession(context.Background(), Session{Role: "engineer", ToolCounts: map[string]int{
		validationCommandSuccessKey:   1,
		buildCommandSuccessKey:        1,
		browserProductSmokeSuccessKey: 1,
	}})
	raw, err := json.Marshal(fileWriteArgs{Path: path, Content: content})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	err = preToolPolicy(ctx, root, "file_write", raw)
	if err == nil {
		t.Fatal("expected browser-framework evidence write to reject copy-only build script")
	}
	if !strings.Contains(err.Error(), "no-op") || !strings.Contains(err.Error(), "vite build") {
		t.Fatalf("expected copy-only build guidance, got %v", err)
	}
}

func TestEngineerBrowserFrameworkTicketEvidenceRequiresProductSmoke(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePhaserPackage(t, dir, true)
	writePolicyTicket(t, dir, "in-progress", "T-001-phaser-tetris.md", `---
id: T-001
title: Ship Phaser Tetris
work_type: feature
bdd_scenarios: ["F-001-S002"]
end_to_end_evidence: required
evidence_links: []
verified_by: TBD
---

# Ship Phaser Tetris
`)
	path := filepath.Join("docs", "tickets", "in-progress", "T-001-phaser-tetris.md")
	content := `---
id: T-001
title: Ship Phaser Tetris
work_type: feature
bdd_scenarios: ["F-001-S002"]
end_to_end_evidence: required
evidence_links:
- npm run build
verified_by:
- engineer
---

# Ship Phaser Tetris
`
	ctx := WithSession(context.Background(), Session{Role: "engineer", ToolCounts: map[string]int{
		validationCommandSuccessKey: 1,
		buildCommandSuccessKey:      1,
	}})
	raw, err := json.Marshal(fileWriteArgs{Path: path, Content: content})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	err = preToolPolicy(ctx, root, "file_write", raw)
	if err == nil {
		t.Fatal("expected browser-framework evidence write to require product smoke")
	}
	if !strings.Contains(err.Error(), "product smoke") ||
		!strings.Contains(err.Error(), `shell_exec argv ["node","-e"`) ||
		!strings.Contains(err.Error(), "node --check") {
		t.Fatalf("expected browser product smoke guidance, got %v", err)
	}
}

func TestEngineerBrowserFrameworkTicketEvidenceBlocksMissingNamedExports(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePhaserPackage(t, dir, true)
	if err := os.MkdirAll(filepath.Join(dir, "src"), 0o755); err != nil {
		t.Fatalf("mkdir src: %v", err)
	}
	files := map[string]string{
		"src/main.js": `import { TetrisGame } from './game.js';
new TetrisGame();
`,
		"src/game.js": `import { Playfield } from './playfield.js';
class TetrisGame {}
`,
		"src/playfield.js": `class Playfield {}
`,
	}
	for rel, content := range files {
		if err := os.WriteFile(filepath.Join(dir, filepath.FromSlash(rel)), []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}
	writePolicyTicket(t, dir, "in-progress", "T-001-phaser-tetris.md", `---
id: T-001
title: Ship Phaser Tetris
work_type: feature
bdd_scenarios: ["F-001-S002"]
end_to_end_evidence: required
evidence_links: []
verified_by: TBD
---

# Ship Phaser Tetris
`)
	path := filepath.Join("docs", "tickets", "in-progress", "T-001-phaser-tetris.md")
	content := `---
id: T-001
title: Ship Phaser Tetris
work_type: feature
bdd_scenarios: ["F-001-S002"]
end_to_end_evidence: required
evidence_links:
- npm run build
- node -e "const fs=require('fs'); const src=fs.readFileSync('src/main.js','utf8'); if(!src.includes('TetrisGame')) process.exit(1)"
verified_by:
- engineer
---

# Ship Phaser Tetris
`
	ctx := WithSession(context.Background(), Session{Role: "engineer", ToolCounts: map[string]int{
		validationCommandSuccessKey:   1,
		buildCommandSuccessKey:        1,
		browserProductSmokeSuccessKey: 1,
	}})
	raw, err := json.Marshal(fileWriteArgs{Path: path, Content: content})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	err = preToolPolicy(ctx, root, "file_write", raw)
	if err == nil {
		t.Fatal("expected browser-framework evidence write to block missing named exports")
	}
	if !strings.Contains(err.Error(), "does not export") || !strings.Contains(err.Error(), "src/game.js") {
		t.Fatalf("expected missing export guidance, got %v", err)
	}
}

func TestEngineerBrowserFrameworkTicketEvidenceBlocksClassicScriptModuleEntry(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePhaserPackage(t, dir, true)
	if err := os.MkdirAll(filepath.Join(dir, "src"), 0o755); err != nil {
		t.Fatalf("mkdir src: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte(`<script src="src/main.js"></script>`), 0o644); err != nil {
		t.Fatalf("write index.html: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "src", "main.js"), []byte(`import Phaser from 'phaser';
export function boot() { return Phaser.AUTO; }
`), 0o644); err != nil {
		t.Fatalf("write main.js: %v", err)
	}
	writePolicyTicket(t, dir, "in-progress", "T-001-phaser-tetris.md", `---
id: T-001
title: Ship Phaser Tetris
work_type: feature
bdd_scenarios: ["F-001-S002"]
end_to_end_evidence: required
evidence_links: []
verified_by: TBD
---

# Ship Phaser Tetris
`)
	path := filepath.Join("docs", "tickets", "in-progress", "T-001-phaser-tetris.md")
	content := `---
id: T-001
title: Ship Phaser Tetris
work_type: feature
bdd_scenarios: ["F-001-S002"]
end_to_end_evidence: required
evidence_links:
- npm run build
- browser product smoke
verified_by:
- engineer
---

# Ship Phaser Tetris
`
	ctx := WithSession(context.Background(), Session{Role: "engineer", ToolCounts: map[string]int{
		validationCommandSuccessKey:   1,
		buildCommandSuccessKey:        1,
		browserProductSmokeSuccessKey: 1,
	}})
	raw, err := json.Marshal(fileWriteArgs{Path: path, Content: content})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	err = preToolPolicy(ctx, root, "file_write", raw)
	if err == nil {
		t.Fatal("expected browser-framework evidence write to block classic script module entry")
	}
	if !strings.Contains(err.Error(), "classic script") || !strings.Contains(err.Error(), "type=\"module\"") {
		t.Fatalf("expected classic script module guidance, got %v", err)
	}
}

func TestEngineerBrowserFrameworkTicketEvidenceBlocksPhaserExternalInViteConfig(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePhaserPackage(t, dir, true)
	content := `/*
MarsDocSync:
docs:
- docs/features/F-001-product-walking-skeleton.md
*/
import { defineConfig } from 'vite';

export default defineConfig({
  build: {
    rollupOptions: {
      external: ['phaser']
    }
  }
});
`
	if err := os.WriteFile(filepath.Join(dir, "vite.config.js"), []byte(content), 0o644); err != nil {
		t.Fatalf("write vite config: %v", err)
	}
	writePolicyTicket(t, dir, "in-progress", "T-001-phaser-tetris.md", `---
id: T-001
title: Ship Phaser Tetris
work_type: feature
bdd_scenarios: ["F-001-S002"]
end_to_end_evidence: required
evidence_links: []
verified_by: TBD
---

# Ship Phaser Tetris
`)
	path := filepath.Join("docs", "tickets", "in-progress", "T-001-phaser-tetris.md")
	ticketContent := `---
id: T-001
title: Ship Phaser Tetris
work_type: feature
bdd_scenarios: ["F-001-S002"]
end_to_end_evidence: required
evidence_links:
- npm run build
- browser product smoke
verified_by:
- engineer
---

# Ship Phaser Tetris
`
	ctx := WithSession(context.Background(), Session{Role: "engineer", ToolCounts: map[string]int{
		validationCommandSuccessKey:   1,
		buildCommandSuccessKey:        1,
		browserProductSmokeSuccessKey: 1,
	}})
	raw, err := json.Marshal(fileWriteArgs{Path: path, Content: ticketContent})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	err = preToolPolicy(ctx, root, "file_write", raw)
	if err == nil {
		t.Fatal("expected browser-framework evidence write to block phaser externalization")
	}
	if !strings.Contains(err.Error(), "externalizes phaser") || !strings.Contains(err.Error(), "Vite bundle") {
		t.Fatalf("expected Vite externalization source guidance, got %v", err)
	}
}

func TestEngineerBrowserFrameworkTicketEvidenceBlocksMissingPhaserImport(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePhaserPackage(t, dir, true)
	if err := os.MkdirAll(filepath.Join(dir, "src", "scenes"), 0o755); err != nil {
		t.Fatalf("mkdir scenes: %v", err)
	}
	files := map[string]string{
		"src/index.html": `<div id="game-container"></div><script type="module" src="main.js"></script>`,
		"src/main.js": `import Phaser from 'phaser';
import GameScene from './scenes/GameScene.js';
new Phaser.Game({ type: Phaser.AUTO, parent: 'game-container', scene: [GameScene] });
`,
		"src/scenes/GameScene.js": `export default class GameScene extends Phaser.Scene {
  create() {
    this.add.text(20, 20, 'Tetris');
  }
}
`,
	}
	for rel, content := range files {
		if err := os.WriteFile(filepath.Join(dir, filepath.FromSlash(rel)), []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}
	writePolicyTicket(t, dir, "in-progress", "T-001-phaser-tetris.md", `---
id: T-001
title: Ship Phaser Tetris
work_type: feature
bdd_scenarios: ["F-001-S002"]
end_to_end_evidence: required
evidence_links: []
verified_by: TBD
---

# Ship Phaser Tetris
`)
	path := filepath.Join("docs", "tickets", "in-progress", "T-001-phaser-tetris.md")
	content := `---
id: T-001
title: Ship Phaser Tetris
work_type: feature
bdd_scenarios: ["F-001-S002"]
end_to_end_evidence: required
evidence_links:
- npm run build
- browser product smoke
verified_by:
- engineer
---

# Ship Phaser Tetris
`
	ctx := WithSession(context.Background(), Session{Role: "engineer", ToolCounts: map[string]int{
		validationCommandSuccessKey:   1,
		buildCommandSuccessKey:        1,
		browserProductSmokeSuccessKey: 1,
	}})
	raw, err := json.Marshal(fileWriteArgs{Path: path, Content: content})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	err = preToolPolicy(ctx, root, "file_write", raw)
	if err == nil {
		t.Fatal("expected browser-framework evidence write to block missing Phaser import")
	}
	if !strings.Contains(err.Error(), "uses Phaser global APIs") || !strings.Contains(err.Error(), "src/scenes/GameScene.js") {
		t.Fatalf("expected missing Phaser import guidance, got %v", err)
	}
}

func TestEngineerBrowserFrameworkTicketEvidenceBlocksMissingLocalExportImport(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePhaserPackage(t, dir, true)
	if err := os.MkdirAll(filepath.Join(dir, "src"), 0o755); err != nil {
		t.Fatalf("mkdir src: %v", err)
	}
	files := map[string]string{
		"src/main.js": `import { TetrisGame } from './game.js';
new TetrisGame();
`,
		"src/game.js": `export class TetrisGame {
  spawn() {
    return createRandomTetromino();
  }
}
`,
		"src/tetromino.js": `export function createRandomTetromino() {
  return { shape: [[1]], x: 0, y: 0 };
}
`,
	}
	for rel, content := range files {
		if err := os.WriteFile(filepath.Join(dir, filepath.FromSlash(rel)), []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}
	writePolicyTicket(t, dir, "in-progress", "T-001-phaser-tetris.md", `---
id: T-001
title: Ship Phaser Tetris
work_type: feature
bdd_scenarios: ["F-001-S002"]
end_to_end_evidence: required
evidence_links: []
verified_by: TBD
---

# Ship Phaser Tetris
`)
	path := filepath.Join("docs", "tickets", "in-progress", "T-001-phaser-tetris.md")
	content := `---
id: T-001
title: Ship Phaser Tetris
work_type: feature
bdd_scenarios: ["F-001-S002"]
end_to_end_evidence: required
evidence_links:
- npm run build
- browser product smoke
verified_by:
- engineer
---

# Ship Phaser Tetris
`
	ctx := WithSession(context.Background(), Session{Role: "engineer", ToolCounts: map[string]int{
		validationCommandSuccessKey:   1,
		buildCommandSuccessKey:        1,
		browserProductSmokeSuccessKey: 1,
	}})
	raw, err := json.Marshal(fileWriteArgs{Path: path, Content: content})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	err = preToolPolicy(ctx, root, "file_write", raw)
	if err == nil {
		t.Fatal("expected browser-framework evidence write to block missing local export import")
	}
	if !strings.Contains(err.Error(), "does not import") || !strings.Contains(err.Error(), "src/tetromino.js") {
		t.Fatalf("expected missing local import guidance, got %v", err)
	}
}

func TestQABrowserFrameworkApprovalBlocksPhaserLifecycleDefect(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePhaserPackage(t, dir, true)
	if err := os.MkdirAll(filepath.Join(dir, "src"), 0o755); err != nil {
		t.Fatalf("mkdir src: %v", err)
	}
	brokenConfig := `/*
MarsDocSync:
docs:
- docs/features/F-001-product-walking-skeleton.md
*/
const config = {
  type: Phaser.AUTO,
  scene: { preload: preload, create: create, update: update }
};
export { config };
`
	if err := os.WriteFile(filepath.Join(dir, "src", "config.js"), []byte(brokenConfig), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	ctx := WithSession(context.Background(), Session{Role: "qa", ToolCounts: map[string]int{
		validationCommandSuccessKey:  1,
		buildCommandSuccessKey:       1,
		"tool:file_read:success":     1,
		"tool:docsync_audit:success": 1,
	}})

	err := preToolPolicy(ctx, root, "job_disposition_record", json.RawMessage(`{"status":"approved","ticket_id":"T-001","next_need":"security_review"}`))
	if err == nil {
		t.Fatal("expected QA approval to block Phaser lifecycle defect")
	}
	if !strings.Contains(err.Error(), "cannot approve browser-framework ticket") || !strings.Contains(err.Error(), "scene callback") {
		t.Fatalf("expected Phaser lifecycle guidance, got %v", err)
	}
}

func TestQABrowserFrameworkApprovalRequiresProductSmoke(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePhaserPackage(t, dir, true)
	ctx := WithSession(context.Background(), Session{Role: "qa", ToolCounts: map[string]int{
		validationCommandSuccessKey:  1,
		buildCommandSuccessKey:       1,
		"tool:file_read:success":     1,
		"tool:docsync_audit:success": 1,
	}})

	err := preToolPolicy(ctx, root, "job_disposition_record", json.RawMessage(`{"status":"approved","ticket_id":"T-001","next_need":"security_review"}`))
	if err == nil {
		t.Fatal("expected QA approval to require browser product smoke")
	}
	if !strings.Contains(err.Error(), "HTTP/build evidence alone") ||
		!strings.Contains(err.Error(), "Phaser game/canvas") ||
		!strings.Contains(err.Error(), `shell_exec argv ["node","-e"`) ||
		!strings.Contains(err.Error(), "new Phaser.Game") {
		t.Fatalf("expected browser product smoke guidance, got %v", err)
	}
}

func TestPhaserBrowserProductSmokeGuidanceAvoidsRegexEscapes(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePhaserPackage(t, dir, true)

	guidance := browserProductSmokeCommandGuidance(root)
	if !strings.Contains(guidance, "main.split('new Phaser.Game')") {
		t.Fatalf("expected string-count smoke guidance, got %q", guidance)
	}
	if strings.Contains(guidance, `new\\s+Phaser`) || strings.Contains(guidance, `Phaser\\.Game`) {
		t.Fatalf("expected guidance to avoid JSON-escaped regex evidence, got %q", guidance)
	}
}

func TestDogfoodBrowserFrameworkApprovalRequiresProductSmoke(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePhaserPackage(t, dir, true)
	ctx := WithSession(context.Background(), Session{Role: "dogfood", ToolCounts: map[string]int{
		validationCommandSuccessKey:  1,
		buildCommandSuccessKey:       1,
		"tool:docsync_audit:success": 1,
	}})

	err := preToolPolicy(ctx, root, "job_disposition_record", json.RawMessage(`{"status":"approved","next_need":"no_need"}`))
	if err == nil {
		t.Fatal("expected Dogfood approval to require browser product smoke")
	}
	if !strings.Contains(err.Error(), "curl/HTTP reachability alone") {
		t.Fatalf("expected dogfood browser smoke guidance, got %v", err)
	}
}

func TestReviewTerminalEvidenceForBrowserFrameworkWithoutBuildRequestsChanges(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePhaserPackage(t, dir, false)
	session := &Session{Role: "qa", ToolCounts: map[string]int{
		validationCommandSuccessKey:  1,
		"tool:file_read:success":     1,
		"tool:docsync_audit:success": 1,
	}}

	if !ReviewTerminalEvidenceSatisfied(root, session) {
		t.Fatal("expected terminal decision after curl/read/docsync when browser framework has no build surface")
	}
	guidance := ReviewTerminalDispositionGuidance(root, session)
	if !strings.Contains(guidance, "changes_requested") || !strings.Contains(guidance, "no build script") {
		t.Fatalf("expected changes_requested build-surface guidance, got %q", guidance)
	}
}

func TestCTOTicketCreateBlocksGoShapeForPhaserBrief(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePhaserBrief(t, dir)
	writePolicyPlan(t, dir)
	writePolicyFeature(t, dir, "F-001-product-walking-skeleton.md")
	ctx := WithSession(context.Background(), Session{Role: "cto-weekly", ToolCounts: map[string]int{}})
	raw, err := json.Marshal(ticketCreateArgs{
		Title:            "Implement Phaser Tetris",
		Priority:         "high",
		WorkType:         "feature",
		BDDScenarios:     []string{"F-001-S002"},
		EndToEndEvidence: "required",
		Body:             "## Affected Files\n- `cmd/phaser-tetris-demo/main.go`\n- `go.mod`\n\n## Design Guidance\nFollow the established project structure for a Go CLI with web-based frontend.",
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	err = preToolPolicy(ctx, root, "ticket_create", raw)
	if err == nil {
		t.Fatal("expected Phaser ticket_create to reject Go CLI shape")
	}
	if !strings.Contains(err.Error(), "Phaser/JavaScript target tickets") || !strings.Contains(err.Error(), "package.json") {
		t.Fatalf("expected browser JavaScript ticket guidance, got %v", err)
	}
}

func TestCTOTicketCreateBlocksCDNShapeForPhaserBrief(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePhaserBrief(t, dir)
	writePolicyPlan(t, dir)
	writePolicyFeature(t, dir, "F-001-product-walking-skeleton.md")
	ctx := WithSession(context.Background(), Session{Role: "cto-weekly", ToolCounts: map[string]int{}})
	raw, err := json.Marshal(ticketCreateArgs{
		Title:            "Implement Phaser Tetris",
		Priority:         "high",
		WorkType:         "feature",
		BDDScenarios:     []string{"F-001-S002"},
		EndToEndEvidence: "required",
		Body:             "## Acceptance Criteria\n- Phaser JS library loads from CDN\n- Game renders in the browser\n\n## Evidence\n- curl http://localhost:5173",
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	err = preToolPolicy(ctx, root, "ticket_create", raw)
	if err == nil {
		t.Fatal("expected Phaser ticket_create to reject CDN runtime shape")
	}
	if !strings.Contains(err.Error(), "local phaser npm dependency") || !strings.Contains(err.Error(), "CDN loading acceptance criteria") {
		t.Fatalf("expected local dependency ticket guidance, got %v", err)
	}
}

func TestEngineerFileWriteBlocksGoScaffoldForPhaserBrief(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePhaserBrief(t, dir)
	writePolicyFeature(t, dir, "F-001-product-walking-skeleton.md")
	writePolicyTicket(t, dir, "in-progress", "T-001-phaser-tetris.md", `---
id: T-001
title: Ship Phaser Tetris
work_type: feature
bdd_scenarios: ["F-001-S002"]
end_to_end_evidence: required
evidence_links: []
verified_by: TBD
---

# Ship Phaser Tetris
`)
	content := `/*
MarsDocSync:
docs:
- docs/features/F-001-product-walking-skeleton.md
*/
package main

func main() {}
`
	raw, err := json.Marshal(fileWriteArgs{Path: "cmd/phaser-tetris-demo/main.go", Content: content})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	ctx := WithSession(context.Background(), Session{Role: "engineer", ToolCounts: map[string]int{}})

	err = preToolPolicy(ctx, root, "file_write", raw)
	if err == nil {
		t.Fatal("expected Phaser implementation to reject Go scaffold")
	}
	if !strings.Contains(err.Error(), "Phaser/JavaScript target implementation") || !strings.Contains(err.Error(), "package.json") {
		t.Fatalf("expected browser JavaScript implementation guidance, got %v", err)
	}
}

func TestEngineerFileWriteAllowsPhaserValidationHelperProbeStringsUnderScripts(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePhaserBrief(t, dir)
	writePolicyFeature(t, dir, "F-001-product-walking-skeleton.md")
	writePolicyTicket(t, dir, "in-progress", "T-001-phaser-tetris.md", `---
id: T-001
title: Ship Phaser Tetris
work_type: feature
bdd_scenarios: ["F-001-S001"]
end_to_end_evidence: required
evidence_links: []
verified_by: TBD
---

# Ship Phaser Tetris
`)
	content := `/*
MarsDocSync:
docs:
- docs/features/F-001-product-walking-skeleton.md
*/
const fs = require('fs');
const main = fs.readFileSync('src/main.js', 'utf8');
const games = main.split('new Phaser.Game').length - 1;
	if (games !== 1) throw new Error('expected exactly one new Phaser.Game');
console.log('browser smoke helper passed');
`
	raw, err := json.Marshal(fileWriteArgs{Path: "scripts/validate-phaser.js", Content: content})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	ctx := WithSession(context.Background(), Session{Role: "engineer", ToolCounts: map[string]int{}})

	if err := preToolPolicy(ctx, root, "file_write", raw); err != nil {
		t.Fatalf("expected validation helper probe file to bypass product-source Phaser lifecycle checks, got %v", err)
	}
}

func TestEngineerFileWriteBlocksRootBrowserValidationHelperProbe(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePhaserBrief(t, dir)
	writePolicyFeature(t, dir, "F-001-product-walking-skeleton.md")
	writePolicyTicket(t, dir, "in-progress", "T-001-phaser-tetris.md", "# T-001\n")
	content := `/*
MarsDocSync:
docs:
- docs/features/F-001-product-walking-skeleton.md
*/
const fs = require('fs');
const main = fs.readFileSync('src/main.js', 'utf8');
console.log(main.includes('new Phaser.Game'));
`
	raw, err := json.Marshal(fileWriteArgs{Path: "validate-game.js", Content: content})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	ctx := WithSession(context.Background(), Session{Role: "engineer", ToolCounts: map[string]int{}})

	err = preToolPolicy(ctx, root, "file_write", raw)
	if err == nil {
		t.Fatal("expected root validation helper to be blocked")
	}
	if !strings.Contains(err.Error(), "repo-root scratch validation file") ||
		!strings.Contains(err.Error(), "direct shell_exec") {
		t.Fatalf("expected scratch validation guidance, got %v", err)
	}
}

func TestBrowserFrameworkSourceFindingsIgnorePhaserValidationHelper(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePhaserPackage(t, dir, true)
	if err := os.WriteFile(filepath.Join(dir, "validate-phaser.js"), []byte(`const fs = require('fs');
const main = fs.readFileSync('src/main.js', 'utf8');
const games = main.split('new Phaser.Game').length - 1;
if (games !== 1) throw new Error('expected exactly one new Phaser.Game');
`), 0o644); err != nil {
		t.Fatalf("write helper: %v", err)
	}

	if findings := browserFrameworkSourceFindings(root); len(findings) != 0 {
		t.Fatalf("expected validation helper to be ignored by product source findings, got %v", findings)
	}
}

func TestEngineerFileWriteBlocksPhaserPackageWithoutBuildScript(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePhaserBrief(t, dir)
	writePolicyFeature(t, dir, "F-001-product-walking-skeleton.md")
	writePolicyTicket(t, dir, "in-progress", "T-001-phaser-tetris.md", "# T-001\n")
	content := `{
  "name": "phaser-tetris-demo",
  "scripts": {
    "start": "python3 -m http.server 18081 --bind 127.0.0.1"
  },
  "dependencies": {
    "phaser": "^3.70.0"
  }
}`
	raw, err := json.Marshal(fileWriteArgs{Path: "package.json", Content: content})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	ctx := WithSession(context.Background(), Session{Role: "engineer", ToolCounts: map[string]int{}})

	err = preToolPolicy(ctx, root, "file_write", raw)
	if err == nil {
		t.Fatal("expected Phaser package write to require build script")
	}
	if !strings.Contains(err.Error(), "deterministic package build script") || !strings.Contains(err.Error(), "vite build") {
		t.Fatalf("expected build-script guidance, got %v", err)
	}
}

func TestEngineerFileWriteBlocksPhaserPackageCopyOnlyBuildScript(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePhaserBrief(t, dir)
	writePolicyTicket(t, dir, "in-progress", "T-001-phaser-tetris.md", "# T-001\n")
	content := `{
  "name": "phaser-tetris-demo",
  "scripts": {
    "build": "mkdir -p dist && cp src/index.html dist/index.html && echo 'Build completed successfully'",
    "start": "live-server --port=8080 --host=127.0.0.1 --open=src/index.html"
  },
  "dependencies": {
    "phaser": "^3.70.0"
  },
  "devDependencies": {
    "live-server": "^1.2.2"
  }
}`
	raw, err := json.Marshal(fileWriteArgs{Path: "package.json", Content: content})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	ctx := WithSession(context.Background(), Session{Role: "engineer", ToolCounts: map[string]int{}})

	err = preToolPolicy(ctx, root, "file_write", raw)
	if err == nil {
		t.Fatal("expected Phaser package write to reject copy-only build script")
	}
	if !strings.Contains(err.Error(), "deterministic package build script") || !strings.Contains(err.Error(), "vite build") {
		t.Fatalf("expected copy-only build guidance, got %v", err)
	}
}

func TestEngineerFileWriteBlocksPhaserCDNScriptTag(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePhaserBrief(t, dir)
	writePolicyTicket(t, dir, "in-progress", "T-001-phaser-tetris.md", "# T-001\n")
	content := `<!doctype html>
<html>
<body>
  <div id="game-container"></div>
  <script src="https://cdn.jsdelivr.net/npm/phaser@3/dist/phaser.min.js"></script>
  <script src="src/main.js"></script>
</body>
</html>`
	raw, err := json.Marshal(fileWriteArgs{Path: "index.html", Content: content})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	ctx := WithSession(context.Background(), Session{Role: "engineer", ToolCounts: map[string]int{}})

	err = preToolPolicy(ctx, root, "file_write", raw)
	if err == nil {
		t.Fatal("expected Phaser CDN script tag to be blocked")
	}
	if !strings.Contains(err.Error(), "local phaser npm dependency") || !strings.Contains(err.Error(), "CDN-only") {
		t.Fatalf("expected local dependency guidance, got %v", err)
	}
}

func TestEngineerFileWriteBlocksNestedPhaserCDNScriptTag(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePhaserBrief(t, dir)
	writePolicyTicket(t, dir, "in-progress", "T-001-phaser-tetris.md", "# T-001\n")
	content := `<!doctype html>
<html>
<body>
  <div id="game-container"></div>
  <script src="https://cdn.jsdelivr.net/npm/phaser@3/dist/phaser.min.js"></script>
  <script src="main.js"></script>
</body>
</html>`
	raw, err := json.Marshal(fileWriteArgs{Path: "src/index.html", Content: content})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	ctx := WithSession(context.Background(), Session{Role: "engineer", ToolCounts: map[string]int{}})

	err = preToolPolicy(ctx, root, "file_write", raw)
	if err == nil {
		t.Fatal("expected nested Phaser CDN script tag to be blocked")
	}
	if !strings.Contains(err.Error(), "local phaser npm dependency") || !strings.Contains(err.Error(), "CDN-only") {
		t.Fatalf("expected local dependency guidance, got %v", err)
	}
}

func TestEngineerFileWriteBlocksPhaserPackageReservedRuntimePort(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePhaserBrief(t, dir)
	writePolicyTicket(t, dir, "in-progress", "T-001-phaser-tetris.md", "# T-001\n")
	content := `{
  "name": "phaser-tetris-demo",
  "scripts": {
    "build": "vite build",
    "start": "vite --host 127.0.0.1 --port 18081"
  },
  "dependencies": {
    "phaser": "^3.70.0"
  },
  "devDependencies": {
    "vite": "^5.0.0"
  }
}`
	raw, err := json.Marshal(fileWriteArgs{Path: "package.json", Content: content})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	ctx := WithSession(context.Background(), Session{Role: "engineer", ToolCounts: map[string]int{}})

	err = preToolPolicy(ctx, root, "file_write", raw)
	if err == nil {
		t.Fatal("expected reserved runtime port to be blocked")
	}
	if !strings.Contains(err.Error(), "reserved Mars Harness port 18081") || !strings.Contains(err.Error(), "5173") {
		t.Fatalf("expected reserved-port guidance, got %v", err)
	}
}

func TestEngineerFileWriteBlocksPhaserPackageStaticSourceServer(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePhaserBrief(t, dir)
	writePolicyTicket(t, dir, "in-progress", "T-001-phaser-tetris.md", "# T-001\n")
	content := `{
  "name": "phaser-tetris-demo",
  "scripts": {
    "build": "vite build",
    "start": "python3 -m http.server 5173 --bind 127.0.0.1"
  },
  "dependencies": {
    "phaser": "^3.70.0"
  },
  "devDependencies": {
    "vite": "^5.0.0"
  }
}`
	raw, err := json.Marshal(fileWriteArgs{Path: "package.json", Content: content})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	ctx := WithSession(context.Background(), Session{Role: "engineer", ToolCounts: map[string]int{}})

	err = preToolPolicy(ctx, root, "file_write", raw)
	if err == nil {
		t.Fatal("expected static source server to be blocked for Phaser package")
	}
	if !strings.Contains(err.Error(), "static source server") || !strings.Contains(err.Error(), "Vite dev/preview") {
		t.Fatalf("expected Vite runtime guidance, got %v", err)
	}
}

func TestEngineerFileWriteBlocksPhaserSourceWithoutModuleImport(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePhaserBrief(t, dir)
	writePolicyFeature(t, dir, "F-001-product-walking-skeleton.md")
	writePolicyTicket(t, dir, "in-progress", "T-001-phaser-tetris.md", "# T-001\n")
	content := `/*
MarsDocSync:
docs:
- docs/features/F-001-product-walking-skeleton.md
*/
const config = { type: Phaser.AUTO, parent: 'game-container' };
new Phaser.Game(config);
`
	raw, err := json.Marshal(fileWriteArgs{Path: "src/main.js", Content: content})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	ctx := WithSession(context.Background(), Session{Role: "engineer", ToolCounts: map[string]int{}})

	err = preToolPolicy(ctx, root, "file_write", raw)
	if err == nil {
		t.Fatal("expected Phaser source without module import to be blocked")
	}
	if !strings.Contains(err.Error(), "uses Phaser global APIs") || !strings.Contains(err.Error(), "import Phaser from 'phaser'") {
		t.Fatalf("expected Phaser import guidance, got %v", err)
	}
}

func TestEngineerFileWriteBlocksRecursivePhaserGameConstruction(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePhaserBrief(t, dir)
	writePolicyTicket(t, dir, "in-progress", "T-001-phaser-tetris.md", "# T-001\n")
	content := `/*
MarsDocSync:
docs:
- docs/features/F-001-product-walking-skeleton.md
*/
import Phaser from 'phaser';

const config = {
  type: Phaser.AUTO,
  parent: 'game-container',
  scene: { create }
};

function create() {
  new Phaser.Game(config);
}
`
	raw, err := json.Marshal(fileWriteArgs{Path: "src/main.js", Content: content})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	ctx := WithSession(context.Background(), Session{Role: "engineer", ToolCounts: map[string]int{}})

	err = preToolPolicy(ctx, root, "file_write", raw)
	if err == nil {
		t.Fatal("expected recursive Phaser game construction to be blocked")
	}
	if !strings.Contains(err.Error(), "inside scene callback create") || !strings.Contains(err.Error(), "module startup") {
		t.Fatalf("expected recursive construction guidance, got %v", err)
	}
}

func TestEngineerFileWriteAllowsTopLevelPhaserGameAfterSceneCallbacks(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePhaserBrief(t, dir)
	writePolicyFeature(t, dir, "F-001-product-walking-skeleton.md")
	writePolicyTicket(t, dir, "in-progress", "T-001-phaser-tetris.md", "# T-001\n")
	content := `/*
MarsDocSync:
docs:
- docs/features/F-001-product-walking-skeleton.md
*/
import Phaser from 'phaser';

const config = {
  type: Phaser.AUTO,
  parent: 'game',
  scene: { preload, create, update }
};

function preload() {
  // No assets for the first slice.
}

function create() {
  const cell = this.add.rectangle(15, 15, 29, 29, 0x00ffff);
  cell.setOrigin(0, 0);
}

function update() {
  // First slice animation is handled by Phaser's game loop.
}

const game = new Phaser.Game(config);
void game;
`
	raw, err := json.Marshal(fileWriteArgs{Path: "src/main.js", Content: content})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	ctx := WithSession(context.Background(), Session{Role: "engineer", ToolCounts: map[string]int{}})

	if err := preToolPolicy(ctx, root, "file_write", raw); err != nil {
		t.Fatalf("expected top-level Phaser.Game construction after callbacks to pass, got %v", err)
	}
}

func TestEngineerFileWriteBlocksPhaserRuntimeInViteConfig(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePhaserBrief(t, dir)
	writePolicyTicket(t, dir, "in-progress", "T-001-phaser-tetris.md", "# T-001\n")
	content := `/*
MarsDocSync:
docs:
- docs/features/F-001-product-walking-skeleton.md
*/
import { defineConfig } from 'vite';
import Phaser from 'phaser';
import { createGame } from './src/game.js';

export default defineConfig({});
`
	raw, err := json.Marshal(fileWriteArgs{Path: "vite.config.js", Content: content})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	ctx := WithSession(context.Background(), Session{Role: "engineer", ToolCounts: map[string]int{}})

	err = preToolPolicy(ctx, root, "file_write", raw)
	if err == nil {
		t.Fatal("expected Phaser runtime imports in Vite config to be blocked")
	}
	if !strings.Contains(err.Error(), "Vite config runs in Node") || !strings.Contains(err.Error(), "browser entrypoint") {
		t.Fatalf("expected Vite/Phaser config guidance, got %v", err)
	}
}

func TestEngineerFileWriteBlocksPhaserExternalInViteConfig(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePhaserBrief(t, dir)
	writePolicyTicket(t, dir, "in-progress", "T-001-phaser-tetris.md", "# T-001\n")
	content := `/*
MarsDocSync:
docs:
- docs/features/F-001-product-walking-skeleton.md
*/
import { defineConfig } from 'vite';

export default defineConfig({
  build: {
    rollupOptions: {
      external: ['phaser']
    }
  }
});
`
	raw, err := json.Marshal(fileWriteArgs{Path: "vite.config.js", Content: content})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	ctx := WithSession(context.Background(), Session{Role: "engineer", ToolCounts: map[string]int{}})

	err = preToolPolicy(ctx, root, "file_write", raw)
	if err == nil {
		t.Fatal("expected Phaser externalization in Vite config to be blocked")
	}
	if !strings.Contains(err.Error(), "must not externalize phaser") || !strings.Contains(err.Error(), "production bundle") {
		t.Fatalf("expected Vite externalization guidance, got %v", err)
	}
}

func TestEngineerBrowserFrameworkEvidenceRequiresPackageForPhaserBrief(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePhaserBrief(t, dir)
	writePolicyTicket(t, dir, "in-progress", "T-001-phaser-tetris.md", `---
id: T-001
title: Ship Phaser Tetris
work_type: feature
bdd_scenarios: ["F-001-S002"]
end_to_end_evidence: required
evidence_links: []
verified_by: TBD
---

# Ship Phaser Tetris
`)
	path := filepath.Join("docs", "tickets", "in-progress", "T-001-phaser-tetris.md")
	content := `---
id: T-001
title: Ship Phaser Tetris
work_type: feature
bdd_scenarios: ["F-001-S002"]
end_to_end_evidence: required
evidence_links:
- curl -fsS http://127.0.0.1:8080/
verified_by:
- engineer
---

# Ship Phaser Tetris
`
	ctx := WithSession(context.Background(), Session{Role: "engineer", ToolCounts: map[string]int{
		validationCommandSuccessKey: 1,
	}})
	raw, err := json.Marshal(fileWriteArgs{Path: path, Content: content})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	err = preToolPolicy(ctx, root, "file_write", raw)
	if err == nil {
		t.Fatal("expected Phaser brief evidence write to require package manifest")
	}
	if !strings.Contains(err.Error(), "no package.json") || !strings.Contains(err.Error(), "local phaser dependency") {
		t.Fatalf("expected package-manifest guidance, got %v", err)
	}
}

func TestShellExecPolicyBlocksNodeCheckHTML(t *testing.T) {
	t.Parallel()
	_, root := setupPolicyTicketRepo(t)
	ctx := WithSession(context.Background(), Session{Role: "engineer", ToolCounts: map[string]int{}})

	err := preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":["node","--check","index.html"]}`))
	if err == nil {
		t.Fatal("expected node --check HTML to be blocked")
	}
	if !strings.Contains(err.Error(), "only validates JavaScript") || !strings.Contains(err.Error(), "npm run build") {
		t.Fatalf("expected HTML validation guidance, got %v", err)
	}
}

func TestRecordSessionToolOutcomeTreatsNodeCheckHTMLAsProcedureFailure(t *testing.T) {
	t.Parallel()
	_, root := setupPolicyTicketRepo(t)
	session := &Session{Role: "engineer", ToolCounts: map[string]int{}, ToolState: map[string]string{}}
	raw := json.RawMessage(`{"argv":["node","--check","index.html"]}`)

	recordSessionToolOutcome(session, root, "shell_exec", raw, ToolResult{ExitCode: 1, Stderr: "SyntaxError: Unexpected token '<'"}, nil)

	if session.ToolCounts[validationProcedureFailureKey] != 1 {
		t.Fatalf("expected validation procedure failure, got counts %+v", session.ToolCounts)
	}
	if session.ToolCounts[unexpectedRuntimeValidationOutstandingKey] != 0 {
		t.Fatalf("expected no unresolved runtime validation failure, got counts %+v", session.ToolCounts)
	}
}

func TestRecordSessionToolOutcomeTreatsNodeEvalBrowserFrameworkGlobalAsProcedureFailure(t *testing.T) {
	t.Parallel()
	dir, root := setupPolicyTicketRepo(t)
	writePhaserPackage(t, dir, true)
	session := &Session{Role: "engineer", ToolCounts: map[string]int{}, ToolState: map[string]string{}}
	raw := json.RawMessage(`{"argv":["node","-e","const { GameScene } = require('./src/scenes/GameScene.js'); console.log('browser smoke: Phaser canvas #game new Phaser.Game');"]}`)
	result := ToolResult{
		ExitCode: 1,
		Stderr:   "/tmp/demo/node_modules/phaser/src/device/OS.js:153\n        if (window.cordova !== undefined)\n        ^\nReferenceError: window is not defined\n",
	}

	recordSessionToolOutcome(session, root, "shell_exec", raw, result, nil)

	if session.ToolCounts[validationProcedureFailureKey] != 1 {
		t.Fatalf("expected validation procedure failure, got counts %+v", session.ToolCounts)
	}
	if session.ToolCounts[unexpectedRuntimeValidationOutstandingKey] != 0 {
		t.Fatalf("expected no unresolved runtime validation failure, got counts %+v", session.ToolCounts)
	}
	if session.ToolState[validationProcedureFailureCommandKey] == "" {
		t.Fatalf("expected procedure failure command to be recorded, got state %+v", session.ToolState)
	}

	ctx := WithSession(context.Background(), *session)
	err := preToolPolicy(ctx, root, "shell_exec", []byte(`{"argv":["npm","run","build"]}`))
	if err != nil {
		t.Fatalf("expected corrected browser build validation to remain available, got %v", err)
	}
}

func writePhaserPackage(t *testing.T, repoRoot string, buildScript bool) {
	t.Helper()
	scripts := `"start":"http-server -p 8080"`
	if buildScript {
		scripts += `,"build":"vite build"`
	}
	content := `{
  "scripts": {` + scripts + `},
  "dependencies": {
    "phaser": "^3.70.0"
  },
  "devDependencies": {
    "vite": "^5.0.0"
  }
}
`
	if err := os.WriteFile(filepath.Join(repoRoot, "package.json"), []byte(content), 0o644); err != nil {
		t.Fatalf("write package.json: %v", err)
	}
}

func writeValidPhaserSource(t *testing.T, repoRoot string) {
	t.Helper()
	srcDir := filepath.Join(repoRoot, "src")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatalf("mkdir src: %v", err)
	}
	index := `<!--
MarsDocSync:
docs:
- docs/features/F-001-product-walking-skeleton.md
-->
<div id="game-container"></div>
<script type="module" src="./src/main.js"></script>
`
	if err := os.WriteFile(filepath.Join(repoRoot, "index.html"), []byte(index), 0o644); err != nil {
		t.Fatalf("write index.html: %v", err)
	}
	main := `/*
MarsDocSync:
docs:
- docs/features/F-001-product-walking-skeleton.md
*/
import Phaser from 'phaser';

class GameScene extends Phaser.Scene {
  create() {
    this.add.text(10, 10, 'Tetris');
  }
}

const config = {
  type: Phaser.AUTO,
  width: 300,
  height: 600,
  parent: 'game-container',
  scene: [GameScene],
};

new Phaser.Game(config);
`
	if err := os.WriteFile(filepath.Join(srcDir, "main.js"), []byte(main), 0o644); err != nil {
		t.Fatalf("write src/main.js: %v", err)
	}
}

func writePhaserPackageWithScripts(t *testing.T, repoRoot string, scripts map[string]string, dependencies map[string]string, devDependencies map[string]string) {
	t.Helper()
	var b strings.Builder
	b.WriteString("{\n  \"scripts\": {")
	first := true
	for name, script := range scripts {
		if !first {
			b.WriteString(",")
		}
		first = false
		b.WriteString("\n    ")
		key, _ := json.Marshal(name)
		value, _ := json.Marshal(script)
		b.Write(key)
		b.WriteString(": ")
		b.Write(value)
	}
	b.WriteString("\n  },\n  \"dependencies\": {")
	first = true
	for name, version := range dependencies {
		if !first {
			b.WriteString(",")
		}
		first = false
		b.WriteString("\n    ")
		key, _ := json.Marshal(name)
		value, _ := json.Marshal(version)
		b.Write(key)
		b.WriteString(": ")
		b.Write(value)
	}
	b.WriteString("\n  },\n  \"devDependencies\": {")
	first = true
	for name, version := range devDependencies {
		if !first {
			b.WriteString(",")
		}
		first = false
		b.WriteString("\n    ")
		key, _ := json.Marshal(name)
		value, _ := json.Marshal(version)
		b.Write(key)
		b.WriteString(": ")
		b.Write(value)
	}
	b.WriteString("\n  }\n}\n")
	if err := os.WriteFile(filepath.Join(repoRoot, "package.json"), []byte(b.String()), 0o644); err != nil {
		t.Fatalf("write package.json: %v", err)
	}
}

func writePhaserBrief(t *testing.T, repoRoot string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(repoRoot, "README.md"), []byte("# Phaser Tetris Demo\n\nCreate Tetris using Phaser JS.\n"), 0o644); err != nil {
		t.Fatalf("write README: %v", err)
	}
}
