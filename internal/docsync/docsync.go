/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/cli-tool-skill-sync.md
- docs/design-docs/delivery-operating-model.md
- docs/design-docs/documentation-sync-architecture.md
- docs/features/F-001-delivery-operating-model.md
- docs/features/F-019-typescript-monorepo-docsync.md
*/
package docsync

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/greaveselliott/mars/internal/bundle"
)

type Config struct {
	RepoRoot          string
	IncludeRoots      []string
	IncludeExtensions []string
	ExcludeGlobs      []string
}

var defaultIncludeRoots = []string{
	"cmd", "internal", "pkg", "examples",
	"src", "app", "apps", "pages", "packages", "public", "web", "workers", "static", "tests",
	".github/workflows",
}

var defaultIncludeExtensions = []string{
	".go", ".html", ".css", ".js", ".jsx", ".mjs", ".cjs",
	".ts", ".tsx", ".mts", ".cts", ".yaml", ".yml",
}

var defaultExcludeGlobs = []string{
	"**/.git/**",
	"**/node_modules/**",
	"**/build/**",
	"**/dist/**",
	"**/vendor/**",
	"**/coverage/**",
	"**/.expo/**",
	"**/.react-router/**",
	"**/*.generated.*",
}

type sourceSelection struct {
	includeRoots      []string
	includeExtensions map[string]struct{}
	excludeGlobs      []compiledGlob
}

type compiledGlob struct {
	raw string
	re  *regexp.Regexp
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
	selection, err := resolveSourceSelection(absRoot, cfg)
	if err != nil {
		return Report{}, err
	}
	files, err := sourceFiles(absRoot, selection)
	if err != nil {
		return Report{}, err
	}
	foundationRoot := isFoundationHarnessRoot(absRoot)
	var report Report
	for _, rel := range files {
		abs := filepath.Join(absRoot, filepath.FromSlash(rel))
		data, err := os.ReadFile(abs)
		if err != nil {
			return Report{}, fmt.Errorf("docsync: read %s: %w", rel, err)
		}
		docs := MetadataDocs(string(data))
		var expected []string
		if foundationRoot {
			expected = ExpectedDocs(rel)
		}
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

func isFoundationHarnessRoot(root string) bool {
	data, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			if strings.TrimSpace(line) == "module github.com/greaveselliott/mars" {
				return true
			}
		}
	}
	if _, err := os.Stat(filepath.Join(root, "cmd", "mars", "main.go")); err != nil {
		return false
	}
	if _, err := os.Stat(filepath.Join(root, "internal", "scanner", "init.go")); err != nil {
		return false
	}
	return true
}

func SourceFiles(root string) ([]string, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("docsync: resolve repo path: %w", err)
	}
	selection, err := resolveSourceSelection(absRoot, Config{RepoRoot: absRoot})
	if err != nil {
		return nil, err
	}
	return sourceFiles(absRoot, selection)
}

// RequiresMetadata reports whether rel is selected by the repository's
// effective DocSync configuration. It keeps file-write policy aligned with the
// same roots, extensions, and exclusions used by Audit.
func RequiresMetadata(repoRoot, rel string) (bool, error) {
	absRoot, err := filepath.Abs(strings.TrimSpace(repoRoot))
	if err != nil {
		return false, fmt.Errorf("docsync: resolve repo path: %w", err)
	}
	selection, err := resolveSourceSelection(absRoot, Config{RepoRoot: absRoot})
	if err != nil {
		return false, err
	}
	rel = filepath.ToSlash(filepath.Clean(strings.TrimSpace(rel)))
	if rel == "." || rel == "" || strings.HasPrefix(rel, "../") || path.IsAbs(rel) {
		return false, nil
	}
	if selection.excluded(rel) || !selection.includesExtension(filepath.Ext(rel)) {
		return false, nil
	}
	if !strings.Contains(rel, "/") {
		return true, nil
	}
	for _, root := range selection.includeRoots {
		if rel == root || strings.HasPrefix(rel, root+"/") {
			return true, nil
		}
	}
	return false, nil
}

func sourceFiles(root string, selection sourceSelection) ([]string, error) {
	var files []string
	seen := map[string]struct{}{}
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("docsync: read repo root: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 {
			continue
		}
		rel := filepath.ToSlash(entry.Name())
		if selection.includesExtension(filepath.Ext(entry.Name())) && !selection.excluded(rel) {
			seen[rel] = struct{}{}
		}
	}
	for _, sourceRoot := range selection.includeRoots {
		absSourceRoot := filepath.Join(root, filepath.FromSlash(sourceRoot))
		if _, err := os.Stat(absSourceRoot); os.IsNotExist(err) {
			continue
		} else if err != nil {
			return nil, fmt.Errorf("docsync: stat source root %s: %w", sourceRoot, err)
		}
		err := filepath.WalkDir(absSourceRoot, func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			rel = filepath.ToSlash(rel)
			if entry.Type()&os.ModeSymlink != 0 {
				if entry.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if entry.IsDir() {
				if selection.excluded(rel) {
					return filepath.SkipDir
				}
				return nil
			}
			if !selection.includesExtension(filepath.Ext(path)) || selection.excluded(rel) {
				return nil
			}
			seen[rel] = struct{}{}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("docsync: walk %s: %w", sourceRoot, err)
		}
	}
	for rel := range seen {
		files = append(files, rel)
	}
	sort.Strings(files)
	return files, nil
}

func resolveSourceSelection(root string, cfg Config) (sourceSelection, error) {
	manifestCfg, err := readManifestDocSyncConfig(root)
	if err != nil {
		return sourceSelection{}, err
	}
	roots := firstNonEmpty(cfg.IncludeRoots, manifestCfg.IncludeRoots, defaultIncludeRoots)
	extensions := firstNonEmpty(cfg.IncludeExtensions, manifestCfg.IncludeExtensions, defaultIncludeExtensions)
	excludes := firstNonEmpty(cfg.ExcludeGlobs, manifestCfg.ExcludeGlobs, defaultExcludeGlobs)

	normalizedRoots, err := normalizeRoots(roots)
	if err != nil {
		return sourceSelection{}, err
	}
	normalizedExtensions, err := normalizeExtensions(extensions)
	if err != nil {
		return sourceSelection{}, err
	}
	compiledExcludes, err := compileGlobs(excludes)
	if err != nil {
		return sourceSelection{}, err
	}
	extensionSet := make(map[string]struct{}, len(normalizedExtensions))
	for _, ext := range normalizedExtensions {
		extensionSet[ext] = struct{}{}
	}
	return sourceSelection{
		includeRoots:      normalizedRoots,
		includeExtensions: extensionSet,
		excludeGlobs:      compiledExcludes,
	}, nil
}

func readManifestDocSyncConfig(root string) (Config, error) {
	manifest, _, err := bundle.LoadDocSyncConfig(root)
	if err != nil {
		return Config{}, fmt.Errorf("docsync: %w", err)
	}
	return Config{
		IncludeRoots:      manifest.IncludeRoots,
		IncludeExtensions: manifest.IncludeExtensions,
		ExcludeGlobs:      manifest.ExcludeGlobs,
	}, nil
}

func firstNonEmpty(values ...[]string) []string {
	for _, value := range values {
		if len(value) > 0 {
			return append([]string(nil), value...)
		}
	}
	return nil
}

func normalizeRoots(values []string) ([]string, error) {
	var roots []string
	for _, value := range values {
		raw := strings.TrimSpace(value)
		if raw == "" {
			return nil, fmt.Errorf("docsync: include_roots contains an empty path — use repository-relative directories such as apps or packages")
		}
		if filepath.IsAbs(raw) || path.IsAbs(filepath.ToSlash(raw)) || windowsAbsolutePattern.MatchString(raw) {
			return nil, fmt.Errorf("docsync: include_root %q must be repository-relative", raw)
		}
		normalized := path.Clean(filepath.ToSlash(raw))
		if normalized == "." || normalized == ".." || strings.HasPrefix(normalized, "../") {
			return nil, fmt.Errorf("docsync: include_root %q escapes or selects the repository root — use a contained source directory", raw)
		}
		if strings.Contains(normalized, "\\") {
			return nil, fmt.Errorf("docsync: include_root %q must use repository-relative slash-separated paths", raw)
		}
		if !slices.Contains(roots, normalized) {
			roots = append(roots, normalized)
		}
	}
	return roots, nil
}

var extensionPattern = regexp.MustCompile(`^\.[A-Za-z0-9]+$`)
var windowsAbsolutePattern = regexp.MustCompile(`^[A-Za-z]:[\\/]`)

func normalizeExtensions(values []string) ([]string, error) {
	var extensions []string
	for _, value := range values {
		ext := strings.ToLower(strings.TrimSpace(value))
		if !extensionPattern.MatchString(ext) {
			return nil, fmt.Errorf("docsync: include_extension %q is invalid — use a dot-prefixed suffix such as .ts or .tsx", value)
		}
		if !slices.Contains(extensions, ext) {
			extensions = append(extensions, ext)
		}
	}
	return extensions, nil
}

func compileGlobs(values []string) ([]compiledGlob, error) {
	var globs []compiledGlob
	for _, value := range values {
		raw := strings.TrimSpace(value)
		if raw == "" {
			return nil, fmt.Errorf("docsync: exclude_globs contains an empty pattern — remove it or use a repository-relative glob")
		}
		if filepath.IsAbs(raw) || path.IsAbs(filepath.ToSlash(raw)) || windowsAbsolutePattern.MatchString(raw) {
			return nil, fmt.Errorf("docsync: exclude_glob %q must be repository-relative", raw)
		}
		normalized := filepath.ToSlash(raw)
		for _, part := range strings.Split(normalized, "/") {
			if part == ".." {
				return nil, fmt.Errorf("docsync: exclude_glob %q contains parent traversal", raw)
			}
		}
		re, err := compileGlob(normalized)
		if err != nil {
			return nil, fmt.Errorf("docsync: exclude_glob %q is invalid: %w", raw, err)
		}
		if !slices.ContainsFunc(globs, func(existing compiledGlob) bool { return existing.raw == normalized }) {
			globs = append(globs, compiledGlob{raw: normalized, re: re})
		}
	}
	return globs, nil
}

func compileGlob(glob string) (*regexp.Regexp, error) {
	var expression strings.Builder
	expression.WriteString("^")
	for i := 0; i < len(glob); {
		switch glob[i] {
		case '*':
			if i+1 < len(glob) && glob[i+1] == '*' {
				i += 2
				if i < len(glob) && glob[i] == '/' {
					expression.WriteString("(?:.*/)?")
					i++
				} else {
					expression.WriteString(".*")
				}
			} else {
				expression.WriteString("[^/]*")
				i++
			}
		case '?':
			expression.WriteString("[^/]")
			i++
		case '[', ']':
			return nil, fmt.Errorf("character classes are not supported; use *, **, or ?")
		default:
			expression.WriteString(regexp.QuoteMeta(string(glob[i])))
			i++
		}
	}
	expression.WriteString("$")
	return regexp.Compile(expression.String())
}

func (s sourceSelection) includesExtension(ext string) bool {
	_, ok := s.includeExtensions[strings.ToLower(ext)]
	return ok
}

func (s sourceSelection) excluded(rel string) bool {
	rel = filepath.ToSlash(strings.TrimPrefix(rel, "./"))
	for _, glob := range s.excludeGlobs {
		if glob.re.MatchString(rel) || glob.re.MatchString(rel+"/") {
			return true
		}
	}
	return false
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
		for _, doc := range inlineMetadataDocs(rawTrimmed) {
			if !slices.Contains(docs, doc) {
				docs = append(docs, doc)
			}
		}
		if len(docs) > 0 && strings.Contains(rawTrimmed, "MarsDocSync:") && strings.Contains(rawTrimmed, "]") {
			continue
		}
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

func inlineMetadataDocs(line string) []string {
	idx := strings.Index(line, "MarsDocSync:")
	if idx < 0 {
		return nil
	}
	after := strings.TrimSpace(line[idx+len("MarsDocSync:"):])
	start := strings.Index(after, "[")
	end := strings.LastIndex(after, "]")
	if start < 0 || end < start {
		return nil
	}
	var docs []string
	if err := json.Unmarshal([]byte(after[start:end+1]), &docs); err != nil {
		return nil
	}
	return compactDocStrings(docs)
}

func compactDocStrings(values []string) []string {
	var out []string
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" || slices.Contains(out, trimmed) {
			continue
		}
		out = append(out, trimmed)
	}
	return out
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
	if isDeployedSourcePath(rel) {
		return nil
	}
	for _, rule := range Rules() {
		if strings.HasPrefix(rel, rule.Prefix) {
			return append([]string{}, rule.Docs...)
		}
	}
	return []string{"docs/design-docs/code-documentation-map.md", "docs/product-specs/product-surface.md"}
}

func isDeployedSourcePath(rel string) bool {
	for _, prefix := range []string{"src/", "app/", "apps/", "pages/", "packages/", "public/", "web/", "workers/", "static/", "tests/", ".github/workflows/"} {
		if strings.HasPrefix(rel, prefix) {
			return true
		}
	}
	return false
}

func Rules() []Rule {
	return []Rule{
		{Prefix: "Makefile", Docs: []string{"docs/design-docs/code-documentation-map.md", "docs/design-docs/release-versioning.md", "docs/design-docs/dogfood-matrix.md", "docs/features/F-009-release-update-lifecycle.md"}},
		{Prefix: "cmd/mars/", Docs: []string{"docs/design-docs/code-documentation-map.md", "docs/design-docs/cli-tool-skill-sync.md", "docs/design-docs/delivery-operating-model.md", "docs/design-docs/documentation-sync-architecture.md", "docs/design-docs/release-versioning.md", "docs/design-docs/self-reflective-telemetry.md", "docs/product-specs/product-surface.md", "docs/features/F-001-delivery-operating-model.md", "docs/features/F-002-zero-config-shell-path.md", "docs/features/F-004-target-harness-lifecycle.md", "docs/features/F-009-release-update-lifecycle.md", "docs/features/F-012-self-improvement-loop.md"}},
		{Prefix: "examples/", Docs: []string{"docs/design-docs/code-documentation-map.md", "docs/design-docs/role-customization.md", "docs/features/F-004-target-harness-lifecycle.md"}},
		{Prefix: "internal/agent/", Docs: []string{"docs/design-docs/code-documentation-map.md", "docs/design-docs/agent-runtime.md", "docs/features/F-005-agent-execution-runtime.md"}},
		{Prefix: "internal/buildinfo/", Docs: []string{"docs/design-docs/code-documentation-map.md", "docs/design-docs/release-versioning.md", "docs/features/F-009-release-update-lifecycle.md"}},
		{Prefix: "internal/bundle/", Docs: []string{"docs/design-docs/code-documentation-map.md", "docs/design-docs/context-efficiency.md", "docs/design-docs/role-customization.md", "docs/features/F-004-target-harness-lifecycle.md", "docs/features/F-005-agent-execution-runtime.md"}},
		{Prefix: "internal/config/", Docs: []string{"docs/design-docs/code-documentation-map.md", "docs/product-specs/product-surface.md", "docs/features/F-003-local-inference-lifecycle.md"}},
		{Prefix: "internal/context/", Docs: []string{"docs/design-docs/code-documentation-map.md", "docs/design-docs/context-efficiency.md", "docs/features/F-005-agent-execution-runtime.md"}},
		{Prefix: "internal/dashboard/", Docs: []string{"docs/design-docs/code-documentation-map.md", "docs/design-docs/dashboard.md", "docs/features/F-010-dashboard-control-plane.md"}},
		{Prefix: "internal/docsconsistency/", Docs: []string{"docs/design-docs/code-documentation-map.md", "docs/design-docs/delivery-operating-model.md", "docs/features/F-001-delivery-operating-model.md"}},
		{Prefix: "internal/docsync/", Docs: []string{"docs/design-docs/code-documentation-map.md", "docs/design-docs/delivery-operating-model.md", "docs/design-docs/documentation-sync-architecture.md", "docs/features/F-001-delivery-operating-model.md"}},
		{Prefix: "internal/doctor/", Docs: []string{"docs/design-docs/code-documentation-map.md", "docs/design-docs/self-reflective-telemetry.md", "docs/features/F-004-target-harness-lifecycle.md", "docs/features/F-012-self-improvement-loop.md", "docs/product-specs/product-surface.md"}},
		{Prefix: "internal/evolution/", Docs: []string{"docs/design-docs/code-documentation-map.md", "docs/design-docs/self-improvement.md", "docs/features/F-012-self-improvement-loop.md"}},
		{Prefix: "internal/foundationtelemetry/", Docs: []string{"docs/design-docs/code-documentation-map.md", "docs/design-docs/self-reflective-telemetry.md", "docs/features/F-012-self-improvement-loop.md"}},
		{Prefix: "internal/github/", Docs: []string{"docs/design-docs/code-documentation-map.md", "docs/features/F-011-optional-github-integration.md", "docs/product-specs/product-surface.md"}},
		{Prefix: "internal/guardrails/", Docs: []string{"docs/design-docs/code-documentation-map.md", "docs/design-docs/guardrails.md", "docs/features/F-007-guardrails-and-safety.md"}},
		{Prefix: "internal/hardware/", Docs: []string{"docs/design-docs/code-documentation-map.md", "docs/design-docs/local-inference.md", "docs/features/F-003-local-inference-lifecycle.md"}},
		{Prefix: "internal/inference/", Docs: []string{"docs/design-docs/code-documentation-map.md", "docs/design-docs/local-inference.md", "docs/features/F-003-local-inference-lifecycle.md"}},
		{Prefix: "internal/integrations/", Docs: []string{"docs/design-docs/code-documentation-map.md", "docs/design-docs/board-driven-integrations.md", "docs/features/F-013-board-driven-integrations.md"}},
		{Prefix: "internal/learnings/", Docs: []string{"docs/design-docs/code-documentation-map.md", "docs/design-docs/dogfood-and-decisions.md", "docs/features/F-012-self-improvement-loop.md"}},
		{Prefix: "internal/llm/", Docs: []string{"docs/design-docs/code-documentation-map.md", "docs/design-docs/agent-runtime.md", "docs/design-docs/local-inference.md", "docs/features/F-003-local-inference-lifecycle.md", "docs/features/F-005-agent-execution-runtime.md"}},
		{Prefix: "internal/mcpstdio/", Docs: []string{"docs/design-docs/code-documentation-map.md", "docs/design-docs/tools-glossary.md", "docs/features/F-005-agent-execution-runtime.md"}},
		{Prefix: "internal/models/", Docs: []string{"docs/design-docs/code-documentation-map.md", "docs/design-docs/local-inference.md", "docs/features/F-003-local-inference-lifecycle.md"}},
		{Prefix: "internal/operatingmodel/", Docs: []string{"docs/design-docs/code-documentation-map.md", "docs/design-docs/harness-operating-model.md", "docs/features/F-001-delivery-operating-model.md"}},
		{Prefix: "internal/orchestration/", Docs: []string{"docs/design-docs/code-documentation-map.md", "docs/design-docs/orchestrated-organization-layer.md", "docs/features/F-006-queue-and-orchestration.md"}},
		{Prefix: "internal/orgstate/", Docs: []string{"docs/design-docs/code-documentation-map.md", "docs/design-docs/orchestrated-organization-layer.md", "docs/features/F-006-queue-and-orchestration.md"}},
		{Prefix: "internal/planhygiene/", Docs: []string{"docs/design-docs/code-documentation-map.md", "docs/design-docs/self-improvement.md", "docs/features/F-001-delivery-operating-model.md"}},
		{Prefix: "internal/power/", Docs: []string{"docs/design-docs/code-documentation-map.md", "docs/features/F-006-queue-and-orchestration.md", "docs/product-specs/product-surface.md"}},
		{Prefix: "internal/qualityscore/", Docs: []string{"docs/design-docs/code-documentation-map.md", "docs/design-docs/scoring-system.md", "docs/design-docs/self-reflective-telemetry.md", "docs/features/F-008-scoring-trust-quality.md", "docs/features/F-012-self-improvement-loop.md"}},
		{Prefix: "internal/queue/", Docs: []string{"docs/design-docs/code-documentation-map.md", "docs/design-docs/pipeline-engine.md", "docs/features/F-006-queue-and-orchestration.md"}},
		{Prefix: "internal/remediation/", Docs: []string{"docs/design-docs/code-documentation-map.md", "docs/design-docs/self-reflective-telemetry.md", "docs/features/F-012-self-improvement-loop.md"}},
		{Prefix: "internal/release/", Docs: []string{"docs/design-docs/code-documentation-map.md", "docs/design-docs/release-versioning.md", "docs/features/F-009-release-update-lifecycle.md"}},
		{Prefix: "internal/roleregistry/", Docs: []string{"docs/design-docs/code-documentation-map.md", "docs/design-docs/harness-operating-model.md", "docs/features/F-001-delivery-operating-model.md"}},
		{Prefix: "internal/safety/", Docs: []string{"docs/design-docs/code-documentation-map.md", "docs/design-docs/guardrails.md", "docs/features/F-007-guardrails-and-safety.md"}},
		{Prefix: "internal/sandbox/", Docs: []string{"docs/design-docs/code-documentation-map.md", "docs/design-docs/guardrails.md", "docs/features/F-007-guardrails-and-safety.md"}},
		{Prefix: "internal/scanner/", Docs: []string{"docs/design-docs/code-documentation-map.md", "docs/design-docs/delivery-operating-model.md", "docs/features/F-004-target-harness-lifecycle.md"}},
		{Prefix: "internal/scheduler/", Docs: []string{"docs/design-docs/code-documentation-map.md", "docs/design-docs/pipeline-engine.md", "docs/features/F-006-queue-and-orchestration.md"}},
		{Prefix: "internal/scoring/", Docs: []string{"docs/design-docs/code-documentation-map.md", "docs/design-docs/scoring-system.md", "docs/features/F-008-scoring-trust-quality.md"}},
		{Prefix: "internal/selfupdate/", Docs: []string{"docs/design-docs/code-documentation-map.md", "docs/design-docs/release-versioning.md", "docs/features/F-009-release-update-lifecycle.md"}},
		{Prefix: "internal/serve/remediation", Docs: []string{"docs/design-docs/code-documentation-map.md", "docs/design-docs/pipeline-engine.md", "docs/design-docs/self-reflective-telemetry.md", "docs/features/F-006-queue-and-orchestration.md", "docs/features/F-012-self-improvement-loop.md"}},
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
