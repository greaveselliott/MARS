package learnings

import (
	"encoding/json"
	"fmt"
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
	conv.StartCommand = detectStartCommand(repoRoot, conv.Framework, conv.Language, conv.PackageManager)
	conv.DevPort = detectDevPort(conv.Framework, repoRoot)

	slog.Debug("learnings: detected conventions",
		"repo", repoRoot,
		"package_manager", conv.PackageManager,
		"language", conv.Language,
		"framework", conv.Framework,
		"start_command", conv.StartCommand,
		"dev_port", conv.DevPort,
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

func detectStartCommand(root, framework, language, pm string) string {
	pkgJSON := filepath.Join(root, "package.json")
	if fileExists(pkgJSON) {
		data, err := os.ReadFile(pkgJSON)
		if err == nil {
			var pkg struct {
				Scripts map[string]string `json:"scripts"`
			}
			if json.Unmarshal(data, &pkg) == nil {
				runner := pm
				if runner == "" {
					runner = "npm"
				}
				if _, ok := pkg.Scripts["dev"]; ok {
					return runner + " run dev"
				}
				if _, ok := pkg.Scripts["start"]; ok {
					return runner + " run start"
				}
			}
		}
	}

	switch language {
	case "go":
		if fileExists(filepath.Join(root, "cmd")) {
			return "go run ./cmd/..."
		}
		return "go run ."
	case "python":
		if fileExists(filepath.Join(root, "manage.py")) {
			return "python manage.py runserver"
		}
		return "python -m http.server 8000"
	}

	if fileExists(filepath.Join(root, "index.html")) {
		return "python -m http.server 8080"
	}

	return ""
}

func detectDevPort(framework, root string) string {
	switch framework {
	case "next.js":
		return "3000"
	case "vite":
		return "5173"
	case "nuxt":
		return "3000"
	case "angular":
		return "4200"
	case "svelte":
		return "5173"
	case "remix":
		return "3000"
	case "astro":
		return "4321"
	}

	if fileExists(filepath.Join(root, "index.html")) {
		return "8080"
	}
	if fileExists(filepath.Join(root, "go.mod")) {
		return "8080"
	}
	if fileExists(filepath.Join(root, "manage.py")) {
		return "8000"
	}

	if fileExists(filepath.Join(root, "package.json")) {
		return "3000"
	}

	return ""
}

// ContainerfileTemplate generates a Containerfile based on detected conventions.
func ContainerfileTemplate(conv Conventions) string {
	switch conv.Language {
	case "typescript", "javascript":
		return containerfileNode(conv)
	case "go":
		return containerfileGo(conv)
	case "python":
		return containerfilePython(conv)
	}
	if conv.Framework != "" {
		return containerfileNode(conv)
	}
	if conv.StartCommand != "" {
		return containerfileFallback(conv)
	}
	return containerfileStatic(conv)
}

func containerfileNode(conv Conventions) string {
	install := "npm ci"
	switch conv.PackageManager {
	case "yarn":
		install = "yarn install --frozen-lockfile"
	case "pnpm":
		install = "pnpm install --frozen-lockfile"
	case "bun":
		install = "bun install --frozen-lockfile"
	}
	port := conv.DevPort
	if port == "" {
		port = "3000"
	}
	build := conv.BuildCommand
	if build == "" {
		build = conv.PackageManager + " run build"
		if conv.PackageManager == "" {
			build = "npm run build"
		}
	}
	start := conv.StartCommand
	if start == "" {
		start = "node ."
	}

	return fmt.Sprintf(`FROM node:22-alpine AS builder
WORKDIR /app
COPY package*.json yarn.lock* pnpm-lock.yaml* bun.lockb* ./
RUN %s
COPY . .
RUN %s

FROM node:22-alpine
WORKDIR /app
COPY --from=builder /app .
EXPOSE %s
USER node
CMD ["%s"]
`, install, build, port, start)
}

func containerfileGo(conv Conventions) string {
	build := conv.BuildCommand
	if build == "" {
		build = "go build -o /app/server ./..."
	}
	port := conv.DevPort
	if port == "" {
		port = "8080"
	}
	return fmt.Sprintf(`FROM golang:1.24-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 %s

FROM alpine:3.20
WORKDIR /app
COPY --from=builder /app/server .
EXPOSE %s
USER nobody
CMD ["./server"]
`, build, port)
}

func containerfilePython(conv Conventions) string {
	port := conv.DevPort
	if port == "" {
		port = "8000"
	}
	start := conv.StartCommand
	if start == "" {
		start = "python -m http.server " + port
	}
	return fmt.Sprintf(`FROM python:3.12-slim
WORKDIR /app
COPY requirements.txt* pyproject.toml* ./
RUN pip install --no-cache-dir -r requirements.txt 2>/dev/null || pip install --no-cache-dir . 2>/dev/null || true
COPY . .
EXPOSE %s
USER nobody
CMD ["%s"]
`, port, start)
}

func containerfileStatic(conv Conventions) string {
	return `FROM nginx:alpine
COPY . /usr/share/nginx/html
EXPOSE 80
`
}

func containerfileFallback(conv Conventions) string {
	port := conv.DevPort
	if port == "" {
		port = "3000"
	}
	return fmt.Sprintf(`FROM alpine:3.20
RUN apk add --no-cache bash curl
WORKDIR /app
COPY . .
EXPOSE %s
CMD ["%s"]
`, port, conv.StartCommand)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
