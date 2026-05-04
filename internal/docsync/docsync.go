/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/delivery-operating-model.md
- docs/features/F-001-delivery-operating-model.md
*/
package docsync

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
)

type Config struct {
	RepoRoot string
}

type Rule struct {
	Prefix string   `json:"prefix"`
	Docs   []string `json:"docs"`
}

type Finding struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

type FileReport struct {
	Path         string   `json:"path"`
	Docs         []string `json:"docs"`
	ExpectedDocs []string `json:"expected_docs,omitempty"`
}

type Report struct {
	Files    []FileReport `json:"files"`
	Findings []Finding    `json:"findings"`
}

func (r Report) OK() bool {
	return len(r.Findings) == 0
}

func (r Report) Summary() string {
	return fmt.Sprintf("docsync: checked %d files, findings %d", len(r.Files), len(r.Findings))
}

func Audit(cfg Config) (Report, error) {
	root := strings.TrimSpace(cfg.RepoRoot)
	if root == "" {
		root = "."
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return Report{}, fmt.Errorf("docsync: resolve repo path: %w", err)
	}
	files, err := SourceFiles(absRoot)
	if err != nil {
		return Report{}, err
	}
	var report Report
	for _, rel := range files {
		abs := filepath.Join(absRoot, filepath.FromSlash(rel))
		data, err := os.ReadFile(abs)
		if err != nil {
			return Report{}, fmt.Errorf("docsync: read %s: %w", rel, err)
		}
		docs := MetadataDocs(string(data))
		expected := ExpectedDocs(rel)
		report.Files = append(report.Files, FileReport{Path: rel, Docs: docs, ExpectedDocs: expected})
		if len(docs) == 0 {
			report.Findings = append(report.Findings, Finding{Path: rel, Message: "missing MarsDocSync docs metadata"})
			continue
		}
		for _, doc := range docs {
			if !strings.HasPrefix(doc, "docs/") && doc != "AGENTS.md" && doc != "README.md" && doc != "ARCHITECTURE.md" && doc != "CONTRIBUTING.md" {
				report.Findings = append(report.Findings, Finding{Path: rel, Message: "metadata references non-documentation path " + doc})
				continue
			}
			if _, err := os.Stat(filepath.Join(absRoot, filepath.FromSlash(doc))); err != nil {
				report.Findings = append(report.Findings, Finding{Path: rel, Message: "metadata references missing doc " + doc})
			}
		}
		for _, doc := range expected {
			if !slices.Contains(docs, doc) {
				report.Findings = append(report.Findings, Finding{Path: rel, Message: "metadata missing expected doc " + doc})
			}
		}
	}
	return report, nil
}

func SourceFiles(root string) ([]string, error) {
	var files []string
	sourceRoots := []string{"cmd", "internal", "pkg", ".github/workflows", "examples"}
	for _, sourceRoot := range sourceRoots {
		absSourceRoot := filepath.Join(root, filepath.FromSlash(sourceRoot))
		if _, err := os.Stat(absSourceRoot); os.IsNotExist(err) {
			continue
		}
		err := filepath.WalkDir(absSourceRoot, func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				switch entry.Name() {
				case ".git", "build", "dist", "vendor":
					return filepath.SkipDir
				}
				return nil
			}
			if !isSourceExtension(filepath.Ext(path)) {
				return nil
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			files = append(files, filepath.ToSlash(rel))
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("docsync: walk %s: %w", sourceRoot, err)
		}
	}
	sort.Strings(files)
	return files, nil
}

func isSourceExtension(ext string) bool {
	switch ext {
	case ".go", ".html", ".css", ".js", ".yaml", ".yml":
		return true
	default:
		return false
	}
}

func MetadataDocs(text string) []string {
	prefix := text
	if len(prefix) > 2500 {
		prefix = prefix[:2500]
	}
	lines := strings.Split(prefix, "\n")
	inBlock := false
	inDocs := false
	pendingMode := ""
	commentMode := ""
	var docs []string
	for _, rawLine := range lines {
		rawTrimmed := strings.TrimSpace(rawLine)
		if !inBlock {
			switch {
			case strings.HasPrefix(rawTrimmed, "/*"):
				pendingMode = "block"
			case strings.HasPrefix(rawTrimmed, "<!--"):
				pendingMode = "html"
			case strings.HasPrefix(rawTrimmed, "#") || strings.HasPrefix(rawTrimmed, "//"):
				pendingMode = "line"
			}
		}
		if inBlock && commentMode == "line" && rawTrimmed != "" && !strings.HasPrefix(rawTrimmed, "#") && !strings.HasPrefix(rawTrimmed, "//") {
			break
		}
		line := cleanCommentLine(rawLine)
		if strings.TrimSpace(line) == "MarsDocSync:" {
			inBlock = true
			commentMode = pendingMode
			if commentMode == "" {
				commentMode = "block"
			}
			continue
		}
		if !inBlock {
			continue
		}
		trimmed := strings.TrimSpace(line)
		if trimmed == "docs:" {
			inDocs = true
			continue
		}
		if strings.HasPrefix(trimmed, "- ") && (inDocs || strings.Contains(trimmed, "docs/")) {
			doc := strings.TrimSpace(strings.TrimPrefix(trimmed, "- "))
			doc = strings.Trim(doc, `"'`)
			if doc != "" && !slices.Contains(docs, doc) {
				docs = append(docs, doc)
			}
			continue
		}
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "package ") || strings.HasPrefix(trimmed, "import ") || strings.HasPrefix(trimmed, "func ") || strings.HasPrefix(trimmed, "<") {
			break
		}
		if strings.Contains(rawTrimmed, "*/") || strings.Contains(rawTrimmed, "-->") {
			break
		}
	}
	return docs
}

func cleanCommentLine(line string) string {
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, "/*")
	line = strings.TrimPrefix(line, "*/")
	line = strings.TrimPrefix(line, "*")
	line = strings.TrimPrefix(line, "//")
	line = strings.TrimPrefix(line, "#")
	line = strings.TrimPrefix(line, "<!--")
	line = strings.TrimSuffix(line, "-->")
	return strings.TrimSpace(line)
}

func ExpectedDocs(rel string) []string {
	for _, rule := range Rules() {
		if strings.HasPrefix(rel, rule.Prefix) {
			return append([]string{}, rule.Docs...)
		}
	}
	return []string{"docs/design-docs/code-documentation-map.md", "docs/product-specs/product-surface.md"}
}

func Rules() []Rule {
	return []Rule{
		{Prefix: ".github/workflows/", Docs: []string{"docs/design-docs/code-documentation-map.md", "docs/design-docs/release-versioning.md", "docs/features/F-009-release-update-lifecycle.md"}},
		{Prefix: "cmd/mars-harness/", Docs: []string{"docs/design-docs/code-documentation-map.md", "docs/design-docs/delivery-operating-model.md", "docs/design-docs/release-versioning.md", "docs/product-specs/product-surface.md", "docs/features/F-001-delivery-operating-model.md", "docs/features/F-002-zero-config-shell-path.md", "docs/features/F-004-target-harness-lifecycle.md", "docs/features/F-009-release-update-lifecycle.md"}},
		{Prefix: "examples/", Docs: []string{"docs/design-docs/code-documentation-map.md", "docs/design-docs/role-customization.md", "docs/features/F-004-target-harness-lifecycle.md"}},
		{Prefix: "internal/agent/", Docs: []string{"docs/design-docs/code-documentation-map.md", "docs/design-docs/agent-runtime.md", "docs/features/F-005-agent-execution-runtime.md"}},
		{Prefix: "internal/buildinfo/", Docs: []string{"docs/design-docs/code-documentation-map.md", "docs/design-docs/release-versioning.md", "docs/features/F-009-release-update-lifecycle.md"}},
		{Prefix: "internal/bundle/", Docs: []string{"docs/design-docs/code-documentation-map.md", "docs/design-docs/context-efficiency.md", "docs/design-docs/role-customization.md", "docs/features/F-004-target-harness-lifecycle.md", "docs/features/F-005-agent-execution-runtime.md"}},
		{Prefix: "internal/config/", Docs: []string{"docs/design-docs/code-documentation-map.md", "docs/product-specs/product-surface.md", "docs/features/F-003-local-inference-lifecycle.md"}},
		{Prefix: "internal/context/", Docs: []string{"docs/design-docs/code-documentation-map.md", "docs/design-docs/context-efficiency.md", "docs/features/F-005-agent-execution-runtime.md"}},
		{Prefix: "internal/dashboard/", Docs: []string{"docs/design-docs/code-documentation-map.md", "docs/design-docs/dashboard.md", "docs/features/F-010-dashboard-control-plane.md"}},
		{Prefix: "internal/docsconsistency/", Docs: []string{"docs/design-docs/code-documentation-map.md", "docs/design-docs/delivery-operating-model.md", "docs/features/F-001-delivery-operating-model.md"}},
		{Prefix: "internal/docsync/", Docs: []string{"docs/design-docs/code-documentation-map.md", "docs/design-docs/delivery-operating-model.md", "docs/features/F-001-delivery-operating-model.md"}},
		{Prefix: "internal/doctor/", Docs: []string{"docs/design-docs/code-documentation-map.md", "docs/features/F-004-target-harness-lifecycle.md", "docs/product-specs/product-surface.md"}},
		{Prefix: "internal/evolution/", Docs: []string{"docs/design-docs/code-documentation-map.md", "docs/design-docs/self-improvement.md", "docs/features/F-012-self-improvement-loop.md"}},
		{Prefix: "internal/github/", Docs: []string{"docs/design-docs/code-documentation-map.md", "docs/features/F-011-optional-github-integration.md", "docs/product-specs/product-surface.md"}},
		{Prefix: "internal/guardrails/", Docs: []string{"docs/design-docs/code-documentation-map.md", "docs/design-docs/guardrails.md", "docs/features/F-007-guardrails-and-safety.md"}},
		{Prefix: "internal/hardware/", Docs: []string{"docs/design-docs/code-documentation-map.md", "docs/design-docs/local-inference.md", "docs/features/F-003-local-inference-lifecycle.md"}},
		{Prefix: "internal/inference/", Docs: []string{"docs/design-docs/code-documentation-map.md", "docs/design-docs/local-inference.md", "docs/features/F-003-local-inference-lifecycle.md"}},
		{Prefix: "internal/learnings/", Docs: []string{"docs/design-docs/code-documentation-map.md", "docs/design-docs/dogfood-and-decisions.md", "docs/features/F-012-self-improvement-loop.md"}},
		{Prefix: "internal/llm/", Docs: []string{"docs/design-docs/code-documentation-map.md", "docs/design-docs/agent-runtime.md", "docs/design-docs/local-inference.md", "docs/features/F-003-local-inference-lifecycle.md", "docs/features/F-005-agent-execution-runtime.md"}},
		{Prefix: "internal/mcpstdio/", Docs: []string{"docs/design-docs/code-documentation-map.md", "docs/design-docs/tools-glossary.md", "docs/features/F-005-agent-execution-runtime.md"}},
		{Prefix: "internal/models/", Docs: []string{"docs/design-docs/code-documentation-map.md", "docs/design-docs/local-inference.md", "docs/features/F-003-local-inference-lifecycle.md"}},
		{Prefix: "internal/operatingmodel/", Docs: []string{"docs/design-docs/code-documentation-map.md", "docs/design-docs/harness-operating-model.md", "docs/features/F-001-delivery-operating-model.md"}},
		{Prefix: "internal/orchestration/", Docs: []string{"docs/design-docs/code-documentation-map.md", "docs/design-docs/orchestrated-organization-layer.md", "docs/features/F-006-queue-and-orchestration.md"}},
		{Prefix: "internal/orgstate/", Docs: []string{"docs/design-docs/code-documentation-map.md", "docs/design-docs/orchestrated-organization-layer.md", "docs/features/F-006-queue-and-orchestration.md"}},
		{Prefix: "internal/planhygiene/", Docs: []string{"docs/design-docs/code-documentation-map.md", "docs/design-docs/self-improvement.md", "docs/features/F-001-delivery-operating-model.md"}},
		{Prefix: "internal/power/", Docs: []string{"docs/design-docs/code-documentation-map.md", "docs/features/F-006-queue-and-orchestration.md", "docs/product-specs/product-surface.md"}},
		{Prefix: "internal/qualityscore/", Docs: []string{"docs/design-docs/code-documentation-map.md", "docs/design-docs/scoring-system.md", "docs/features/F-008-scoring-trust-quality.md"}},
		{Prefix: "internal/queue/", Docs: []string{"docs/design-docs/code-documentation-map.md", "docs/design-docs/pipeline-engine.md", "docs/features/F-006-queue-and-orchestration.md"}},
		{Prefix: "internal/release/", Docs: []string{"docs/design-docs/code-documentation-map.md", "docs/design-docs/release-versioning.md", "docs/features/F-009-release-update-lifecycle.md"}},
		{Prefix: "internal/roleregistry/", Docs: []string{"docs/design-docs/code-documentation-map.md", "docs/design-docs/harness-operating-model.md", "docs/features/F-001-delivery-operating-model.md"}},
		{Prefix: "internal/safety/", Docs: []string{"docs/design-docs/code-documentation-map.md", "docs/design-docs/guardrails.md", "docs/features/F-007-guardrails-and-safety.md"}},
		{Prefix: "internal/sandbox/", Docs: []string{"docs/design-docs/code-documentation-map.md", "docs/design-docs/guardrails.md", "docs/features/F-007-guardrails-and-safety.md"}},
		{Prefix: "internal/scanner/", Docs: []string{"docs/design-docs/code-documentation-map.md", "docs/design-docs/delivery-operating-model.md", "docs/features/F-004-target-harness-lifecycle.md"}},
		{Prefix: "internal/scheduler/", Docs: []string{"docs/design-docs/code-documentation-map.md", "docs/design-docs/pipeline-engine.md", "docs/features/F-006-queue-and-orchestration.md"}},
		{Prefix: "internal/scoring/", Docs: []string{"docs/design-docs/code-documentation-map.md", "docs/design-docs/scoring-system.md", "docs/features/F-008-scoring-trust-quality.md"}},
		{Prefix: "internal/selfupdate/", Docs: []string{"docs/design-docs/code-documentation-map.md", "docs/design-docs/release-versioning.md", "docs/features/F-009-release-update-lifecycle.md"}},
		{Prefix: "internal/serve/", Docs: []string{"docs/design-docs/code-documentation-map.md", "docs/design-docs/pipeline-engine.md", "docs/design-docs/orchestrated-organization-layer.md", "docs/features/F-006-queue-and-orchestration.md"}},
		{Prefix: "internal/setup/", Docs: []string{"docs/design-docs/code-documentation-map.md", "docs/design-docs/local-inference.md", "docs/features/F-002-zero-config-shell-path.md", "docs/features/F-003-local-inference-lifecycle.md"}},
		{Prefix: "internal/shellpath/", Docs: []string{"docs/design-docs/code-documentation-map.md", "docs/design-docs/release-versioning.md", "docs/features/F-002-zero-config-shell-path.md"}},
		{Prefix: "internal/telemetry/", Docs: []string{"docs/design-docs/code-documentation-map.md", "docs/design-docs/self-reflective-telemetry.md", "docs/features/F-012-self-improvement-loop.md"}},
		{Prefix: "internal/tickets/", Docs: []string{"docs/design-docs/code-documentation-map.md", "docs/design-docs/delivery-operating-model.md", "docs/features/F-001-delivery-operating-model.md"}},
		{Prefix: "internal/tools/", Docs: []string{"docs/design-docs/code-documentation-map.md", "docs/design-docs/tools-glossary.md", "docs/features/F-005-agent-execution-runtime.md"}},
		{Prefix: "internal/trace/", Docs: []string{"docs/design-docs/code-documentation-map.md", "docs/design-docs/agent-runtime.md", "docs/features/F-005-agent-execution-runtime.md"}},
		{Prefix: "internal/trust/", Docs: []string{"docs/design-docs/code-documentation-map.md", "docs/design-docs/scoring-system.md", "docs/features/F-008-scoring-trust-quality.md"}},
		{Prefix: "internal/ui/", Docs: []string{"docs/design-docs/code-documentation-map.md", "docs/design-docs/agent-runtime.md", "docs/features/F-005-agent-execution-runtime.md"}},
		{Prefix: "internal/updatecheck/", Docs: []string{"docs/design-docs/code-documentation-map.md", "docs/design-docs/release-versioning.md", "docs/features/F-004-target-harness-lifecycle.md"}},
		{Prefix: "pkg/testutil/", Docs: []string{"docs/design-docs/code-documentation-map.md", "docs/design-docs/agent-runtime.md", "docs/features/F-005-agent-execution-runtime.md"}},
	}
}

func MarshalReport(report Report) ([]byte, error) {
	return json.MarshalIndent(report, "", "  ")
}
