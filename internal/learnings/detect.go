package learnings

import (
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
)

// DetectConventions scans a repo root and returns detected conventions.
func DetectConventions(repoRoot string) Conventions {
	var conv Conventions

	conv.PackageManager = detectPackageManager(repoRoot)
	conv.Language = detectLanguage(repoRoot)
	conv.Framework = detectFramework(repoRoot)

	scripts := detectScripts(repoRoot)
	conv.TestCommand = scripts["test"]
	conv.LintCommand = scripts["lint"]
	conv.BuildCommand = scripts["build"]

	slog.Debug("learnings: detected conventions",
		"repo", repoRoot,
		"package_manager", conv.PackageManager,
		"language", conv.Language,
		"framework", conv.Framework,
	)
	return conv
}

// DetectExcludes returns a standard set of directories to exclude,
// plus any detected project-specific ones.
func DetectExcludes(repoRoot string) []string {
	standard := []string{"node_modules", ".git", "dist", "vendor", "build", ".next", "__pycache__"}
	var result []string
	for _, dir := range standard {
		if info, err := os.Stat(filepath.Join(repoRoot, dir)); err == nil && info.IsDir() {
			result = append(result, dir)
		}
	}
	if len(result) == 0 {
		return standard[:3]
	}
	return result
}

func detectPackageManager(root string) string {
	checks := []struct {
		file    string
		manager string
	}{
		{"yarn.lock", "yarn"},
		{"pnpm-lock.yaml", "pnpm"},
		{"package-lock.json", "npm"},
		{"bun.lockb", "bun"},
		{"go.sum", "go"},
		{"Cargo.lock", "cargo"},
		{"Pipfile.lock", "pipenv"},
		{"poetry.lock", "poetry"},
		{"requirements.txt", "pip"},
	}
	for _, c := range checks {
		if fileExists(filepath.Join(root, c.file)) {
			return c.manager
		}
	}
	return ""
}

func detectLanguage(root string) string {
	checks := []struct {
		file string
		lang string
	}{
		{"tsconfig.json", "typescript"},
		{"package.json", "javascript"},
		{"go.mod", "go"},
		{"Cargo.toml", "rust"},
		{"pyproject.toml", "python"},
		{"setup.py", "python"},
		{"requirements.txt", "python"},
		{"Gemfile", "ruby"},
		{"build.gradle", "java"},
		{"pom.xml", "java"},
	}
	for _, c := range checks {
		if fileExists(filepath.Join(root, c.file)) {
			return c.lang
		}
	}
	return ""
}

func detectFramework(root string) string {
	checks := []struct {
		file      string
		framework string
	}{
		{"next.config.js", "next.js"},
		{"next.config.mjs", "next.js"},
		{"next.config.ts", "next.js"},
		{"nuxt.config.ts", "nuxt"},
		{"vite.config.ts", "vite"},
		{"vite.config.js", "vite"},
		{"angular.json", "angular"},
		{"svelte.config.js", "svelte"},
		{"remix.config.js", "remix"},
		{"astro.config.mjs", "astro"},
	}
	for _, c := range checks {
		if fileExists(filepath.Join(root, c.file)) {
			return c.framework
		}
	}
	return ""
}

func detectScripts(root string) map[string]string {
	result := map[string]string{}

	pkgJSON := filepath.Join(root, "package.json")
	if fileExists(pkgJSON) {
		data, err := os.ReadFile(pkgJSON)
		if err == nil {
			var pkg struct {
				Scripts map[string]string `json:"scripts"`
			}
			if json.Unmarshal(data, &pkg) == nil {
				pm := detectPackageManager(root)
				if pm == "" {
					pm = "npm"
				}
				for _, name := range []string{"test", "lint", "build"} {
					if _, ok := pkg.Scripts[name]; ok {
						result[name] = pm + " run " + name
					}
				}
			}
		}
	}

	makefile := filepath.Join(root, "Makefile")
	if fileExists(makefile) {
		data, _ := os.ReadFile(makefile)
		content := string(data)
		for _, target := range []string{"test", "lint", "build"} {
			if _, ok := result[target]; ok {
				continue
			}
			if containsMakeTarget(content, target) {
				result[target] = "make " + target
			}
		}
	}

	gomod := filepath.Join(root, "go.mod")
	if fileExists(gomod) {
		if _, ok := result["test"]; !ok {
			result["test"] = "go test ./..."
		}
		if _, ok := result["build"]; !ok {
			result["build"] = "go build ./..."
		}
	}

	return result
}

func containsMakeTarget(content, target string) bool {
	needle := "\n" + target + ":"
	return len(content) > 0 && (content[:len(target)+1] == target+":" || contains(content, needle))
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstr(s, substr))
}

func containsSubstr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
