package scanner

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Finding represents a detected gap in the repo.
type Finding struct {
	Type        string // "missing_tests", "todo", "no_ci", "no_readme", "no_license", "large_function", "missing_dev_script", "missing_root_layout", "conflicting_app_pages", "missing_tailwind_config", "deprecated_next_config"
	Path        string
	Description string
	Severity    string // "high", "medium", "low"
}

// Config controls scanner behaviour.
type Config struct {
	RepoRoot    string
	MaxPackages int      // concurrency limit for monorepos
	SkipDirs    []string // dirs to skip (node_modules, vendor, .git)
}

// ScanResult holds all scanner findings plus detected metadata.
type ScanResult struct {
	Language   string
	Framework  string
	HasCI      bool
	HasTests   bool
	HasReadme  bool
	HasLicense bool
	Findings   []Finding
}

var defaultSkipDirs = map[string]bool{
	".git":         true,
	"node_modules": true,
	"vendor":       true,
	"__pycache__":  true,
	".next":        true,
	"dist":         true,
	"build":        true,
}

var languageExtensions = map[string]string{
	".go":    "Go",
	".ts":    "TypeScript",
	".tsx":   "TypeScript",
	".js":    "JavaScript",
	".jsx":   "JavaScript",
	".py":    "Python",
	".rb":    "Ruby",
	".rs":    "Rust",
	".java":  "Java",
	".cs":    "C#",
	".cpp":   "C++",
	".c":     "C",
	".php":   "PHP",
	".kt":    "Kotlin",
	".swift": "Swift",
}

func applyDefaults(cfg *Config) {
	if cfg.MaxPackages <= 0 {
		cfg.MaxPackages = 4
	}
}

// Scan analyzes a repository and returns findings.
func Scan(ctx context.Context, cfg Config) (*ScanResult, error) {
	if cfg.RepoRoot == "" {
		return nil, fmt.Errorf("scanner: repo root is empty — pass the path to the repository you want to scan")
	}
	info, err := os.Stat(cfg.RepoRoot)
	if err != nil {
		return nil, fmt.Errorf("scanner: cannot access %s: %w — verify the path exists", cfg.RepoRoot, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("scanner: %s is not a directory — point to the repository root", cfg.RepoRoot)
	}
	applyDefaults(&cfg)

	result := &ScanResult{}
	var allFiles []string
	extensionCount := make(map[string]int)

	err = filepath.WalkDir(cfg.RepoRoot, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		rel, _ := filepath.Rel(cfg.RepoRoot, path)
		if d.IsDir() {
			if shouldSkipDir(d.Name(), rel, cfg.SkipDirs) {
				return filepath.SkipDir
			}
			return nil
		}
		allFiles = append(allFiles, rel)
		ext := strings.ToLower(filepath.Ext(rel))
		if ext != "" {
			extensionCount[ext]++
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scanner: walk %s: %w", cfg.RepoRoot, err)
	}

	result.Language = detectLanguage(extensionCount)
	result.Framework = detectFramework(cfg.RepoRoot, allFiles)
	result.HasCI = detectCI(cfg.RepoRoot, allFiles)
	result.HasTests = detectTests(allFiles)
	result.HasReadme = detectReadme(allFiles)
	result.HasLicense = detectLicense(allFiles)

	if !result.HasCI {
		result.Findings = append(result.Findings, Finding{
			Type:        "no_ci",
			Description: "No CI configuration found — add .github/workflows/ or equivalent",
			Severity:    "high",
		})
	}
	if !result.HasReadme {
		result.Findings = append(result.Findings, Finding{
			Type:        "no_readme",
			Description: "No README found — add a README.md to describe the project",
			Severity:    "medium",
		})
	}
	if !result.HasLicense {
		result.Findings = append(result.Findings, Finding{
			Type:        "no_license",
			Description: "No LICENSE file found — add a LICENSE to clarify usage rights",
			Severity:    "medium",
		})
	}

	result.Findings = append(result.Findings, checkBootability(cfg.RepoRoot, allFiles, result.Framework)...)
	result.Findings = append(result.Findings, findUntestedPackages(cfg.RepoRoot, allFiles)...)
	result.Findings = append(result.Findings, findLargeFunctions(ctx, cfg.RepoRoot, allFiles, cfg.MaxPackages)...)
	result.Findings = append(result.Findings, findTodos(ctx, cfg.RepoRoot, allFiles, cfg.MaxPackages)...)

	slog.Info("scan complete",
		"language", result.Language,
		"framework", result.Framework,
		"findings", len(result.Findings),
		"files_scanned", len(allFiles),
	)
	return result, nil
}

func shouldSkipDir(name, rel string, patterns []string) bool {
	if defaultSkipDirs[name] {
		return true
	}
	for _, p := range patterns {
		if matched, _ := filepath.Match(p, name); matched {
			return true
		}
		if matched, _ := filepath.Match(p, rel); matched {
			return true
		}
	}
	return false
}

func detectLanguage(counts map[string]int) string {
	best := ""
	bestCount := 0
	for ext, count := range counts {
		lang, ok := languageExtensions[ext]
		if !ok {
			continue
		}
		if count > bestCount {
			bestCount = count
			best = lang
		}
	}
	return best
}

func detectFramework(root string, files []string) string {
	for _, f := range files {
		switch filepath.Base(f) {
		case "package.json":
			return detectJSFramework(filepath.Join(root, f))
		case "go.mod":
			return "Go Module"
		case "requirements.txt", "pyproject.toml":
			return "Python"
		case "Cargo.toml":
			return "Rust/Cargo"
		case "Gemfile":
			return "Ruby/Bundler"
		}
	}
	return ""
}

func detectJSFramework(pkgPath string) string {
	data, err := os.ReadFile(pkgPath)
	if err != nil {
		return "Node.js"
	}
	content := string(data)
	switch {
	case strings.Contains(content, `"next"`):
		return "Next.js"
	case strings.Contains(content, `"react"`):
		return "React"
	case strings.Contains(content, `"express"`):
		return "Express"
	case strings.Contains(content, `"vue"`):
		return "Vue"
	default:
		return "Node.js"
	}
}

func detectCI(root string, files []string) bool {
	for _, f := range files {
		dir := filepath.Dir(f)
		base := filepath.Base(f)
		if dir == filepath.Join(".github", "workflows") {
			return true
		}
		if base == ".gitlab-ci.yml" || base == "Jenkinsfile" {
			return true
		}
	}
	_, err := os.Stat(filepath.Join(root, ".github", "workflows"))
	return err == nil
}

func detectTests(files []string) bool {
	for _, f := range files {
		base := filepath.Base(f)
		if strings.HasSuffix(base, "_test.go") ||
			strings.HasSuffix(base, ".test.ts") ||
			strings.HasSuffix(base, ".test.tsx") ||
			strings.HasSuffix(base, ".test.js") ||
			strings.HasSuffix(base, ".test.jsx") ||
			strings.HasPrefix(base, "test_") ||
			strings.HasSuffix(base, "_test.py") {
			return true
		}
	}
	return false
}

func detectReadme(files []string) bool {
	for _, f := range files {
		base := strings.ToLower(filepath.Base(f))
		if base == "readme.md" || base == "readme.txt" || base == "readme" || base == "readme.rst" {
			return true
		}
	}
	return false
}

func detectLicense(files []string) bool {
	for _, f := range files {
		base := strings.ToLower(filepath.Base(f))
		if base == "license" || base == "license.md" || base == "license.txt" || base == "licence" || base == "licence.md" {
			return true
		}
	}
	return false
}

// checkBootability runs framework-specific validation to ensure the project
// can actually build and start. Returns findings for structural issues that
// would prevent the app from running.
func checkBootability(root string, files []string, framework string) []Finding {
	var findings []Finding

	switch framework {
	case "Next.js":
		findings = append(findings, checkNextJSBootability(root, files)...)
	case "React", "Vue", "Express", "Node.js":
		findings = append(findings, checkNodeBootability(root)...)
	}

	findings = append(findings, checkTailwindConsistency(root, files)...)

	return findings
}

func checkNextJSBootability(root string, files []string) []Finding {
	var findings []Finding

	findings = append(findings, checkNodeBootability(root)...)

	hasRootLayout := false
	appDirs := map[string]bool{}
	pagesDirs := map[string]bool{}

	for _, f := range files {
		base := filepath.Base(f)
		dir := filepath.Dir(f)

		if base == "layout.tsx" || base == "layout.jsx" || base == "layout.ts" || base == "layout.js" {
			if isRootAppLayout(f) {
				hasRootLayout = true
			}
		}

		if isAppRouterDir(dir) {
			appDirs[appRouterRoot(f)] = true
		}
		if isPagesRouterDir(dir) {
			pagesDirs[pagesRouterRoot(f)] = true
		}
	}

	if !hasRootLayout {
		findings = append(findings, Finding{
			Type:        "missing_root_layout",
			Description: "Next.js App Router requires a root layout.tsx in src/app/ or app/ — create src/app/layout.tsx with <html> and <body> tags",
			Severity:    "high",
		})
	}

	for appRoot := range appDirs {
		for pagesRoot := range pagesDirs {
			if appRoot != pagesRoot {
				findings = append(findings, Finding{
					Type:        "conflicting_app_pages",
					Path:        fmt.Sprintf("app=%s, pages=%s", appRoot, pagesRoot),
					Description: fmt.Sprintf("'app' dir under %s/ and 'pages' dir under %s/ are not siblings — Next.js requires both under the same root (both in src/ or both at project root)", appRoot, pagesRoot),
					Severity:    "high",
				})
			}
		}
	}

	nextConfigPath := filepath.Join(root, "next.config.js")
	if data, err := os.ReadFile(nextConfigPath); err == nil {
		if strings.Contains(string(data), "appDir") {
			findings = append(findings, Finding{
				Type:        "deprecated_next_config",
				Path:        "next.config.js",
				Description: "next.config.js contains deprecated 'appDir' experimental option — remove it (App Router is stable since Next.js 13.4)",
				Severity:    "medium",
			})
		}
	}

	return findings
}

func checkNodeBootability(root string) []Finding {
	var findings []Finding

	pkgPath := filepath.Join(root, "package.json")
	data, err := os.ReadFile(pkgPath)
	if err != nil {
		return nil
	}
	content := string(data)

	hasScript := func(name string) bool {
		return strings.Contains(content, fmt.Sprintf(`"%s"`, name))
	}

	if !hasScript("dev") && !hasScript("start") {
		findings = append(findings, Finding{
			Type:        "missing_dev_script",
			Path:        "package.json",
			Description: "package.json has no 'dev' or 'start' script — the app cannot be started. Add a dev script (e.g. \"dev\": \"next dev\" for Next.js)",
			Severity:    "high",
		})
	}

	if strings.Contains(content, `"next"`) && !hasScript("build") {
		findings = append(findings, Finding{
			Type:        "missing_dev_script",
			Path:        "package.json",
			Description: "Next.js project has no 'build' script — add \"build\": \"next build\" to package.json scripts",
			Severity:    "high",
		})
	}

	return findings
}

func checkTailwindConsistency(root string, files []string) []Finding {
	hasTailwindDirectives := false
	for _, f := range files {
		ext := filepath.Ext(f)
		if ext != ".css" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(root, f))
		if err != nil {
			continue
		}
		if strings.Contains(string(data), "@tailwind") || strings.Contains(string(data), "@import \"tailwindcss\"") {
			hasTailwindDirectives = true
			break
		}
	}
	if !hasTailwindDirectives {
		return nil
	}

	hasTailwindConfig := false
	for _, f := range files {
		base := filepath.Base(f)
		if strings.HasPrefix(base, "tailwind.config") {
			hasTailwindConfig = true
			break
		}
	}

	var findings []Finding
	if !hasTailwindConfig {
		findings = append(findings, Finding{
			Type:        "missing_tailwind_config",
			Description: "CSS files use Tailwind directives (@tailwind or @import \"tailwindcss\") but no tailwind.config.* file exists — create tailwind.config.js with content paths",
			Severity:    "high",
		})
	}

	hasPostCSSConfig := false
	for _, f := range files {
		base := filepath.Base(f)
		if strings.HasPrefix(base, "postcss.config") {
			hasPostCSSConfig = true
			break
		}
	}
	if !hasPostCSSConfig {
		findings = append(findings, Finding{
			Type:        "missing_tailwind_config",
			Path:        "postcss.config.js",
			Description: "Tailwind CSS requires a PostCSS config — create postcss.config.js with tailwindcss and autoprefixer plugins",
			Severity:    "high",
		})
	}

	return findings
}

// isRootAppLayout checks if a layout file is directly inside app/ or src/app/ (root level).
func isRootAppLayout(relPath string) bool {
	dir := filepath.Dir(relPath)
	return dir == "app" || dir == filepath.Join("src", "app")
}

// isAppRouterDir returns true if the path is under an app/ directory.
func isAppRouterDir(dir string) bool {
	parts := strings.Split(filepath.ToSlash(dir), "/")
	for _, p := range parts {
		if p == "app" {
			return true
		}
	}
	return false
}

// isPagesRouterDir returns true if the path is under a pages/ directory.
func isPagesRouterDir(dir string) bool {
	parts := strings.Split(filepath.ToSlash(dir), "/")
	for _, p := range parts {
		if p == "pages" {
			return true
		}
	}
	return false
}

// appRouterRoot returns "src" if the app dir is under src/, or "." if at root.
func appRouterRoot(relPath string) string {
	parts := strings.Split(filepath.ToSlash(relPath), "/")
	for i, p := range parts {
		if p == "app" {
			if i == 0 {
				return "."
			}
			return strings.Join(parts[:i], "/")
		}
	}
	return "."
}

// pagesRouterRoot returns "src" if the pages dir is under src/, or "." if at root.
func pagesRouterRoot(relPath string) string {
	parts := strings.Split(filepath.ToSlash(relPath), "/")
	for i, p := range parts {
		if p == "pages" {
			if i == 0 {
				return "."
			}
			return strings.Join(parts[:i], "/")
		}
	}
	return "."
}

func findUntestedPackages(_ string, files []string) []Finding {
	goPkgs := make(map[string]bool)
	testedPkgs := make(map[string]bool)

	for _, f := range files {
		if !strings.HasSuffix(f, ".go") {
			continue
		}
		dir := filepath.Dir(f)
		base := filepath.Base(f)
		if strings.HasSuffix(base, "_test.go") {
			testedPkgs[dir] = true
		} else {
			goPkgs[dir] = true
		}
	}

	var findings []Finding
	for pkg := range goPkgs {
		if !testedPkgs[pkg] {
			findings = append(findings, Finding{
				Type:        "missing_tests",
				Path:        pkg,
				Description: fmt.Sprintf("Go package %q has no test files — add %s_test.go", pkg, filepath.Base(pkg)),
				Severity:    "medium",
			})
		}
	}
	return findings
}

// findLargeFunctions detects Go functions exceeding 50 lines.
func findLargeFunctions(ctx context.Context, root string, files []string, concurrency int) []Finding {
	var goFiles []string
	for _, f := range files {
		if strings.HasSuffix(f, ".go") {
			goFiles = append(goFiles, f)
		}
	}

	var (
		mu       sync.Mutex
		findings []Finding
		wg       sync.WaitGroup
	)
	sem := make(chan struct{}, concurrency)

	for _, f := range goFiles {
		if ctx.Err() != nil {
			break
		}
		wg.Add(1)
		sem <- struct{}{}
		go func(relPath string) {
			defer wg.Done()
			defer func() { <-sem }()
			ff := scanFileForLargeFuncs(root, relPath)
			if len(ff) > 0 {
				mu.Lock()
				findings = append(findings, ff...)
				mu.Unlock()
			}
		}(f)
	}
	wg.Wait()
	return findings
}

func scanFileForLargeFuncs(root, relPath string) []Finding {
	f, err := os.Open(filepath.Join(root, relPath))
	if err != nil {
		return nil
	}
	defer f.Close()

	const maxLines = 50
	var findings []Finding
	scanner := bufio.NewScanner(f)
	lineNum := 0
	inFunc := false
	funcName := ""
	funcStart := 0
	braceDepth := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if !inFunc && strings.HasPrefix(trimmed, "func ") {
			name := extractFuncName(trimmed)
			if name != "" {
				inFunc = true
				funcName = name
				funcStart = lineNum
				braceDepth = 0
			}
		}

		if inFunc {
			braceDepth += strings.Count(line, "{") - strings.Count(line, "}")
			if braceDepth <= 0 && lineNum > funcStart {
				length := lineNum - funcStart + 1
				if length > maxLines {
					findings = append(findings, Finding{
						Type:        "large_function",
						Path:        fmt.Sprintf("%s:%d", relPath, funcStart),
						Description: fmt.Sprintf("Function %q is %d lines (threshold: %d) — consider splitting", funcName, length, maxLines),
						Severity:    "low",
					})
				}
				inFunc = false
			}
		}
	}
	return findings
}

func extractFuncName(line string) string {
	rest := strings.TrimPrefix(line, "func ")
	if idx := strings.Index(rest, "("); idx > 0 {
		candidate := rest[:idx]
		if strings.Contains(candidate, ")") {
			after := candidate[strings.Index(candidate, ")")+1:]
			after = strings.TrimSpace(after)
			if after == "" {
				return ""
			}
			return after
		}
		return candidate
	}
	return ""
}

func findTodos(ctx context.Context, root string, files []string, concurrency int) []Finding {
	var sourceFiles []string
	for _, f := range files {
		ext := strings.ToLower(filepath.Ext(f))
		switch ext {
		case ".go", ".ts", ".tsx", ".js", ".jsx", ".py", ".rb", ".rs", ".java", ".c", ".cpp", ".cs", ".php", ".kt", ".swift":
			sourceFiles = append(sourceFiles, f)
		}
	}

	var (
		mu       sync.Mutex
		findings []Finding
		wg       sync.WaitGroup
	)
	sem := make(chan struct{}, concurrency)

	for _, f := range sourceFiles {
		if ctx.Err() != nil {
			break
		}
		wg.Add(1)
		sem <- struct{}{}
		go func(relPath string) {
			defer wg.Done()
			defer func() { <-sem }()
			ff := scanFileForTodos(root, relPath)
			if len(ff) > 0 {
				mu.Lock()
				findings = append(findings, ff...)
				mu.Unlock()
			}
		}(f)
	}
	wg.Wait()
	return findings
}

func scanFileForTodos(root, relPath string) []Finding {
	f, err := os.Open(filepath.Join(root, relPath))
	if err != nil {
		return nil
	}
	defer f.Close()

	var findings []Finding
	s := bufio.NewScanner(f)
	lineNum := 0
	for s.Scan() {
		lineNum++
		line := s.Text()
		upper := strings.ToUpper(line)
		if strings.Contains(upper, "TODO") || strings.Contains(upper, "FIXME") || strings.Contains(upper, "HACK") {
			findings = append(findings, Finding{
				Type:        "todo",
				Path:        fmt.Sprintf("%s:%d", relPath, lineNum),
				Description: strings.TrimSpace(line),
				Severity:    "low",
			})
		}
	}
	return findings
}

// GenerateTickets creates markdown ticket files from scan findings.
func GenerateTickets(findings []Finding, outputDir string) error {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("scanner: create ticket directory %s: %w — check directory permissions", outputDir, err)
	}

	count := 0
	for _, f := range findings {
		if f.Type == "todo" {
			continue
		}
		count++
		filename := fmt.Sprintf("scan-%03d-%s.md", count, f.Type)
		path := filepath.Join(outputDir, filename)
		content := formatTicket(f, count)
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return fmt.Errorf("scanner: write ticket %s: %w — check directory permissions", path, err)
		}
		slog.Info("ticket created", "path", path, "type", f.Type)
	}
	slog.Info("ticket generation complete", "tickets", count, "output_dir", outputDir)
	return nil
}

func formatTicket(f Finding, seq int) string {
	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString(fmt.Sprintf("id: SCAN-%03d\n", seq))
	b.WriteString(fmt.Sprintf("title: %s\n", titleFromType(f.Type)))
	b.WriteString(fmt.Sprintf("priority: %s\n", f.Severity))
	b.WriteString("complexity: small\n")
	b.WriteString("source: scanner\n")
	if f.Path != "" {
		b.WriteString(fmt.Sprintf("path: %s\n", f.Path))
	}
	b.WriteString("---\n\n")
	b.WriteString(fmt.Sprintf("# SCAN-%03d: %s\n\n", seq, titleFromType(f.Type)))
	b.WriteString("## Context\n\n")
	b.WriteString("Detected by `mars-harness scan` static analysis.\n\n")
	b.WriteString("## Description\n\n")
	b.WriteString(f.Description + "\n")
	return b.String()
}

func titleFromType(t string) string {
	switch t {
	case "no_ci":
		return "Add CI Configuration"
	case "no_readme":
		return "Add README"
	case "no_license":
		return "Add LICENSE"
	case "missing_tests":
		return "Add Missing Tests"
	case "large_function":
		return "Refactor Large Function"
	case "missing_dev_script":
		return "Add Missing Package Scripts"
	case "missing_root_layout":
		return "Add Root Layout for Next.js App Router"
	case "conflicting_app_pages":
		return "Fix Conflicting App and Pages Directories"
	case "missing_tailwind_config":
		return "Add Missing Tailwind CSS Configuration"
	case "deprecated_next_config":
		return "Remove Deprecated Next.js Config Options"
	default:
		return strings.ReplaceAll(t, "_", " ")
	}
}
