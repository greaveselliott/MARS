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
	"fmt"
	"io/fs"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

func checkBrowserFrameworkTicketCreatePolicy(root Root, session Session, hasSession bool, args ticketCreateArgs) error {
	if !hasSession {
		return nil
	}
	role := strings.ToLower(strings.TrimSpace(session.Role))
	if role != "cto" && role != "cto-weekly" {
		return nil
	}
	workType := normalizeWorkType(args.Kind, args.WorkType)
	if workType != "feature" {
		return nil
	}
	if err := checkProductCapabilityScenarioCoverage(root); err != nil {
		return err
	}
	body := strings.ToLower(ticketCreatePolicySurface(args))
	if projectBriefForbidsPackageManager(root) && ticketPrescribesPackageManager(body) {
		return fmt.Errorf("policy: this target explicitly forbids package managers or external dependencies. CTO tickets must not prescribe package.json, npm/yarn/pnpm/bun commands, or framework dependency setup; use plain static files and direct browser/source smoke evidence instead")
	}
	if !projectBriefMentionsFramework(root, "phaser") || projectBriefNamesGoBackend(root) {
		return nil
	}
	badGoShape := []string{"go.mod", "go module", "go cli", "golang", "cmd/"}
	for _, marker := range badGoShape {
		if strings.Contains(body, marker) {
			return fmt.Errorf("policy: Phaser/JavaScript target tickets must default to a browser JavaScript shape such as package.json, index.html, and src/*.js with npm run build evidence. Do not prescribe Go CLI paths, go.mod, or cmd/* unless the README explicitly names a Go backend")
		}
	}
	if phaserTicketPrescribesCDNRuntime(body) {
		return fmt.Errorf("policy: Phaser/JavaScript target tickets must require a local phaser npm dependency, package build evidence, and browser-product smoke evidence. Do not prescribe CDN-only Phaser script tags or CDN loading acceptance criteria")
	}
	return nil
}

func ticketCreatePolicySurface(args ticketCreateArgs) string {
	return strings.Join([]string{
		args.Title,
		args.Source,
		args.Body,
		args.VerifiedBy,
		strings.Join(args.EvidenceLinks, "\n"),
		strings.Join(args.BDDScenarios, "\n"),
	}, "\n")
}

func projectBriefForbidsPackageManager(root Root) bool {
	lower := strings.ToLower(projectBriefSourceText(root) + "\n" + projectFeatureContractText(root))
	for _, marker := range []string{
		"no package manager",
		"no package managers",
		"without package manager",
		"without a package manager",
		"no external dependencies",
		"without external dependencies",
		"single-page browser app with local state only",
	} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func projectFeatureContractText(root Root) string {
	data, err := root.RepoFS().ReadFile(filepath.Join("docs", "features", "F-001-product-walking-skeleton.md"))
	if err != nil {
		return ""
	}
	return string(data)
}

func ticketPrescribesPackageManager(body string) bool {
	for _, marker := range []string{
		"package.json",
		"package-lock.json",
		"node_modules",
		"npm ",
		"npm run",
		"yarn ",
		"pnpm ",
		"bun ",
		"vite",
		"vitest",
		"jest",
		"@testing-library",
	} {
		if strings.Contains(body, marker) {
			return true
		}
	}
	return false
}

func phaserTicketPrescribesCDNRuntime(body string) bool {
	body = strings.ToLower(strings.TrimSpace(body))
	if !strings.Contains(body, "phaser") || !strings.Contains(body, "cdn") {
		return false
	}
	negated := regexp.MustCompile(`\b(no|not|avoid|without|disallow|reject|block|cannot|can't|must not|do not|don't|never)\b[^.\n]{0,64}\bcdn\b`)
	if negated.MatchString(body) {
		return false
	}
	badPhrases := []string{
		"cdn-only",
		"cdn script",
		"script tag",
		"load from cdn",
		"loads from cdn",
		"loaded from cdn",
		"loaded by cdn",
		"use cdn",
		"uses cdn",
		"using cdn",
	}
	for _, phrase := range badPhrases {
		if strings.Contains(body, phrase) {
			return true
		}
	}
	return false
}

func engineerBrowserFrameworkEvidenceComplete(root Root, session Session) bool {
	info := repoBrowserFrameworkInfo(root)
	if !info.UsesFramework {
		return false
	}
	counts := session.ToolCounts
	if counts == nil || counts[validationCommandSuccessKey] == 0 {
		return false
	}
	return len(engineerBrowserFrameworkCompletionBlockers(root, session)) == 0
}

func checkEngineerBrowserPostBuildSmokeOnlyPolicy(ctx context.Context, root Root, session Session, args shellExecArgs) error {
	info := repoBrowserFrameworkInfo(root)
	if !info.UsesFramework || !browserFrameworkRequiresProductSmoke(root) {
		return nil
	}
	counts := session.ToolCounts
	if counts == nil || counts[buildCommandSuccessKey] == 0 || counts[browserProductSmokeSuccessKey] > 0 {
		return nil
	}
	if shellExecRunsBuildCommand(args) || shellExecRunsBrowserProductSmokeCommand(args) || shellExecStopsTrackedBackgroundPID(session, args) {
		return nil
	}
	files, err := changedFiles(ctx, root)
	if err != nil || len(dispositionBlockingFiles(files)) == 0 {
		return nil
	}
	return fmt.Errorf(
		"policy: engineer has successful browser-framework build evidence but still needs browser-product smoke before more shell validation. Run %s. Do not inspect dist/assets, require('phaser'), require browser bundles from Node, run node --check on HTML, or use trivial environment probes as substitutes for mounted product UI evidence",
		browserProductSmokeCommandGuidance(root),
	)
}

func checkDogfoodBrowserPostBuildSmokeOnlyPolicy(root Root, session Session, hasSession bool, args shellExecArgs) error {
	if !hasSession || strings.ToLower(strings.TrimSpace(session.Role)) != "dogfood" {
		return nil
	}
	info := repoBrowserFrameworkInfo(root)
	if !info.UsesFramework || !browserFrameworkRequiresProductSmoke(root) {
		return nil
	}
	counts := session.ToolCounts
	if counts == nil || counts[buildCommandSuccessKey] == 0 || counts[browserProductSmokeSuccessKey] > 0 {
		return nil
	}
	if shellExecRunsBuildCommand(args) || shellExecRunsBrowserProductSmokeCommand(args) || shellExecStopsTrackedBackgroundPID(session, args) {
		return nil
	}
	return fmt.Errorf(
		"policy: dogfood has successful browser-framework build evidence but still needs browser-product smoke before more shell validation. Run %s. Do not inspect dist/assets, start static servers, sleep, use no-op shell commands, or treat curl/HTTP reachability as mounted product UI evidence",
		browserProductSmokeCommandGuidance(root),
	)
}

func engineerPostCommitBrowserValidationAllowed(root Root, session Session, args shellExecArgs) bool {
	info := repoBrowserFrameworkInfo(root)
	if !info.UsesFramework {
		return false
	}
	counts := session.ToolCounts
	if counts == nil {
		return false
	}
	if info.HasBuildScript && counts[buildCommandSuccessKey] == 0 && shellExecRunsBuildCommand(args) {
		return true
	}
	if browserFrameworkRequiresProductSmoke(root) && counts[buildCommandSuccessKey] > 0 &&
		counts[browserProductSmokeSuccessKey] == 0 && shellExecRunsBrowserProductSmokeCommand(args) {
		return true
	}
	return false
}

func engineerPostCommitStaticValidationAllowed(root Root, session Session, args shellExecArgs) bool {
	if !staticBrowserRequiresProductSmoke(root) {
		return false
	}
	counts := session.ToolCounts
	if counts == nil || counts[staticProductSmokeSuccessKey] > 0 {
		return false
	}
	return shellExecStartsStaticServerOnApplicationPort(args) || shellExecRunsStaticProductSmokeCommand(args)
}

func shellExecStartsStaticServerOnApplicationPort(args shellExecArgs) bool {
	if !args.Background {
		return false
	}
	fields := normalizedShellExecFields(args)
	if len(fields) == 0 {
		return false
	}
	cmd := filepathBase(fields[0])
	switch cmd {
	case "python", "python3":
		for i := 1; i < len(fields)-1; i++ {
			if fields[i] != "-m" || fields[i+1] != "http.server" {
				continue
			}
			port := "8000"
			if i+2 < len(fields) && regexp.MustCompile(`^[0-9]+$`).MatchString(fields[i+2]) {
				port = fields[i+2]
			}
			return !reservedHarnessPort(port)
		}
	case "npm", "pnpm", "yarn", "bun":
		if len(fields) < 2 {
			return false
		}
		if fields[1] == "run" {
			return len(fields) >= 3 && runtimeScriptName(fields[2])
		}
		return runtimeScriptName(fields[1])
	}
	return false
}

func checkEngineerBrowserFrameworkImplementationShapePolicy(root Root, session Session, hasSession bool, rel string) error {
	if !hasSession || strings.ToLower(strings.TrimSpace(session.Role)) != "engineer" {
		return nil
	}
	if !projectBriefMentionsFramework(root, "phaser") || projectBriefNamesGoBackend(root) {
		return nil
	}
	rel = cleanRepoPath(rel)
	lower := strings.ToLower(rel)
	if lower == "go.mod" || strings.HasSuffix(lower, "/go.mod") || strings.HasSuffix(lower, ".go") && strings.HasPrefix(lower, "cmd/") {
		return fmt.Errorf("policy: Phaser/JavaScript target implementation should use package.json, index.html, and src/*.js with local phaser dependency/build evidence. Do not add Go module or cmd/*.go scaffolding unless README explicitly names a Go backend")
	}
	return nil
}

func checkEngineerBrowserFrameworkPackageWritePolicy(root Root, session Session, hasSession bool, rel, content string) error {
	if !hasSession || strings.ToLower(strings.TrimSpace(session.Role)) != "engineer" {
		return nil
	}
	if !projectBriefMentionsFramework(root, "phaser") || projectBriefNamesGoBackend(root) {
		return nil
	}
	rel = cleanRepoPath(rel)
	lower := strings.ToLower(rel)
	switch lower {
	case "package.json":
		var pkg struct {
			Scripts         map[string]string `json:"scripts"`
			Dependencies    map[string]string `json:"dependencies"`
			DevDependencies map[string]string `json:"devDependencies"`
		}
		if err := json.Unmarshal([]byte(content), &pkg); err != nil {
			return nil
		}
		hasPhaser := false
		for dep := range pkg.Dependencies {
			if strings.EqualFold(strings.TrimSpace(dep), "phaser") {
				hasPhaser = true
			}
		}
		for dep := range pkg.DevDependencies {
			if strings.EqualFold(strings.TrimSpace(dep), "phaser") {
				hasPhaser = true
			}
		}
		if !hasPhaser {
			return fmt.Errorf("policy: Phaser browser targets must declare a local phaser npm dependency in package.json; do not rely on CDN-only runtime")
		}
		hasBuild := false
		for name, script := range pkg.Scripts {
			if buildScriptName(name) && !packageBuildScriptNoop(script) {
				hasBuild = true
				break
			}
		}
		if !hasBuild {
			return fmt.Errorf("policy: Phaser browser targets must include a deterministic package build script in package.json, such as vite build, tsc --noEmit, or another command that fails on broken source; echo, true, and node --check-only scripts are not enough")
		}
		for name, script := range pkg.Scripts {
			if !runtimeScriptName(name) {
				continue
			}
			if port := reservedHarnessPortInScript(script); port != "" {
				return fmt.Errorf("policy: package.json script %q uses reserved MARS port %s. Use an application dev port such as 5173 so target servers do not collide with local inference/runtime ports", name, port)
			}
			if phaserRuntimeScriptUsesStaticSourceServer(script) {
				return fmt.Errorf("policy: package.json script %q starts a static source server for a Phaser app. Use Vite dev/preview, for example `vite --host 127.0.0.1 --port 5173` or `npm run build && vite preview --host 127.0.0.1 --port 5173`, so local npm modules are bundled correctly", name)
			}
		}
	}
	if htmlSourcePath(lower) {
		lowerContent := strings.ToLower(content)
		if strings.Contains(lowerContent, "<script") && strings.Contains(lowerContent, "phaser") &&
			(strings.Contains(lowerContent, "http://") || strings.Contains(lowerContent, "https://") || strings.Contains(lowerContent, "cdn.")) {
			return fmt.Errorf("policy: Phaser browser targets should use the local phaser npm dependency and package build/runtime validation, not a CDN-only Phaser script tag in index.html")
		}
	}
	switch lower {
	case "vite.config.js", "vite.config.ts":
		if viteConfigImportsPhaserRuntime(content) {
			return fmt.Errorf("policy: Phaser Vite config runs in Node during build and must not import Phaser, browser globals, or src/* game modules. Keep vite.config limited to Vite/plugin configuration, and import Phaser/game code from the browser entrypoint instead")
		}
		if viteConfigExternalizesPhaser(content) {
			return fmt.Errorf("policy: Phaser Vite config must not externalize phaser from the production bundle; remove rollupOptions.external entries for phaser so npm run build proves the browser can load the local dependency")
		}
	}
	if javascriptSourcePath(lower) && !browserFrameworkValidationHelperPath(lower) {
		if findings := phaserSingleFileSourceFindings(rel, content); len(findings) > 0 {
			return fmt.Errorf("policy: Phaser source file has lifecycle/import issue: %s", strings.Join(findings, "; "))
		}
	}
	return nil
}

func checkPackageScriptRuntimePolicy(rel, content string) error {
	rel = cleanRepoPath(rel)
	if strings.ToLower(rel) != "package.json" {
		return nil
	}
	var pkg struct {
		Scripts map[string]string `json:"scripts"`
	}
	if err := json.Unmarshal([]byte(content), &pkg); err != nil {
		return nil
	}
	for name, script := range pkg.Scripts {
		if port := reservedHarnessPortInScript(script); port != "" {
			return fmt.Errorf("policy: package.json script %q uses reserved MARS port %s. Use an application dev port such as 5173 or 5174 so target servers do not collide with local inference/runtime ports", name, port)
		}
		if smokeScriptName(name) && packageSmokeScriptNoop(script) {
			return fmt.Errorf("policy: package.json script %q is not real smoke evidence. Replace canned console output, echo, true, or server-start-only scripts with a command that can fail, such as a curl probe against a served page, Playwright/Puppeteer, or a source/runtime assertion that reads product files and throws on mismatch", name)
		}
	}
	return nil
}

func smokeScriptName(name string) bool {
	name = strings.TrimSpace(strings.ToLower(name))
	return name == "smoke" || strings.HasPrefix(name, "smoke:") || strings.Contains(name, "smoke")
}

func packageSmokeScriptNoop(script string) bool {
	script = strings.TrimSpace(script)
	if script == "" {
		return true
	}
	for _, rawPart := range strings.Split(strings.ToLower(script), "&&") {
		part := strings.TrimSpace(rawPart)
		if part == "" {
			continue
		}
		if strings.Contains(part, "curl ") ||
			strings.Contains(part, "wget ") ||
			strings.Contains(part, "playwright") ||
			strings.Contains(part, "puppeteer") ||
			strings.Contains(part, "cypress") ||
			strings.Contains(part, "vitest") ||
			strings.Contains(part, "jest") ||
			strings.Contains(part, "node scripts/") ||
			strings.Contains(part, "node ./scripts/") ||
			strings.Contains(part, "node tests/") ||
			strings.Contains(part, "node ./tests/") ||
			strings.Contains(part, "npm run ") ||
			strings.Contains(part, "pnpm run ") ||
			strings.Contains(part, "yarn ") ||
			strings.Contains(part, "bun run ") ||
			strings.Contains(part, "go test") ||
			strings.Contains(part, "go run") ||
			strings.Contains(part, "python -m pytest") ||
			strings.Contains(part, "python3 -m pytest") {
			return false
		}
		if strings.Contains(part, "node -e") || strings.Contains(part, "node --eval") {
			raw := json.RawMessage(fmt.Sprintf(`{"argv":["node","-e",%q]}`, script))
			args, err := decodeShellExecArgs(raw)
			if err == nil && !shellExecRunsCannedConsoleValidation(args) {
				return false
			}
		}
		if part == ":" || part == "true" || part == "exit 0" ||
			strings.HasPrefix(part, "echo ") ||
			strings.HasPrefix(part, "printf ") ||
			strings.Contains(part, "console.log") ||
			strings.Contains(part, "python -m http.server") ||
			strings.Contains(part, "python3 -m http.server") ||
			strings.HasPrefix(part, "http-server") ||
			strings.HasPrefix(part, "serve ") {
			continue
		}
		return false
	}
	return true
}

func phaserRuntimeScriptUsesStaticSourceServer(script string) bool {
	script = strings.ToLower(strings.TrimSpace(script))
	for _, marker := range []string{
		"python -m http.server",
		"python3 -m http.server",
		"http-server",
		"live-server",
	} {
		if strings.Contains(script, marker) {
			return true
		}
	}
	if regexp.MustCompile(`(^|[;&|]\s*)serve(?:\s+|$)`).MatchString(script) && !strings.Contains(script, "vite preview") {
		return true
	}
	return false
}

func viteConfigImportsPhaserRuntime(content string) bool {
	lower := strings.ToLower(content)
	runtimeMarkers := []string{
		"from 'phaser'",
		`from "phaser"`,
		"require('phaser')",
		`require("phaser")`,
		"from './src",
		`from "./src`,
		"from '../src",
		`from "../src`,
		"require('./src",
		`require("./src`,
		"require('../src",
		`require("../src`,
	}
	for _, marker := range runtimeMarkers {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func viteConfigExternalizesPhaser(content string) bool {
	lower := strings.ToLower(content)
	if !strings.Contains(lower, "external") || !strings.Contains(lower, "phaser") {
		return false
	}
	externalRe := regexp.MustCompile(`(?s)\bexternal\s*:\s*(?:\[[^\]]*['"]phaser['"]|['"]phaser['"]|\([^)]*phaser)`)
	return externalRe.MatchString(lower)
}

func checkShellNodeCheckHTMLPolicy(args shellExecArgs) error {
	if !shellExecNodeCheckHTML(args) {
		return nil
	}
	return fmt.Errorf("policy: node --check only validates JavaScript source, not HTML files. Do not run node --check on .html/.htm entries; validate browser targets with a real package build such as npm run build and a browser/product smoke that loads the HTML")
}

func shellExecNodeCheckHTML(args shellExecArgs) bool {
	fields := normalizedShellExecFields(args)
	for i := 0; i < len(fields)-2; i++ {
		if filepathBase(fields[i]) != "node" {
			continue
		}
		flag := fields[i+1]
		if flag != "--check" && flag != "-c" {
			continue
		}
		target := strings.ToLower(cleanShellPathToken(fields[i+2]))
		if strings.HasSuffix(target, ".html") || strings.HasSuffix(target, ".htm") {
			return true
		}
	}
	return false
}

func shellExecRunsBrowserProductSmokeCommand(args shellExecArgs) bool {
	display := strings.ToLower(shellExecCommandDisplay(args))
	if display == "" {
		return false
	}
	if strings.Contains(display, "playwright") ||
		strings.Contains(display, "puppeteer") ||
		strings.Contains(display, "document.queryselector") ||
		strings.Contains(display, "getelementbyid") ||
		strings.Contains(display, "queryselector") {
		return strings.Contains(display, "canvas") ||
			strings.Contains(display, "#game") ||
			strings.Contains(display, "phaser") ||
			strings.Contains(display, "score") ||
			strings.Contains(display, "game over")
	}
	if strings.Contains(display, "phaser") &&
		(strings.Contains(display, "canvas") ||
			strings.Contains(display, "new phaser.game") ||
			strings.Contains(display, "scene") ||
			strings.Contains(display, "sprite") ||
			strings.Contains(display, "game object")) {
		return true
	}
	return false
}

type browserFrameworkInfo struct {
	UsesFramework          bool
	FrameworkNames         []string
	DeclaredFrameworkNames []string
	HasPackageManifest     bool
	HasBuildScript         bool
	NoopBuildScripts       []string
}

func repoBrowserFrameworkInfo(root Root) browserFrameworkInfo {
	var info browserFrameworkInfo
	seen := map[string]bool{}
	addFramework := func(name string, declared bool) {
		name = strings.ToLower(strings.TrimSpace(name))
		if name == "" {
			return
		}
		if !seen[name] {
			seen[name] = true
			info.FrameworkNames = append(info.FrameworkNames, name)
		}
		if declared && !frameworkListContains(info.DeclaredFrameworkNames, name) {
			info.DeclaredFrameworkNames = append(info.DeclaredFrameworkNames, name)
		}
	}
	if data, err := root.RepoFS().ReadFile("package.json"); err == nil {
		var pkg struct {
			Scripts         map[string]string `json:"scripts"`
			Dependencies    map[string]string `json:"dependencies"`
			DevDependencies map[string]string `json:"devDependencies"`
		}
		if err := json.Unmarshal(data, &pkg); err == nil {
			info.HasPackageManifest = true
			for name, script := range pkg.Scripts {
				if buildScriptName(name) {
					if packageBuildScriptNoop(script) {
						info.NoopBuildScripts = append(info.NoopBuildScripts, name)
						continue
					}
					info.HasBuildScript = true
				}
			}
			frameworks := map[string]string{
				"@vitejs/plugin-react": "react",
				"@vitejs/plugin-vue":   "vue",
				"babylonjs":            "babylon",
				"next":                 "next",
				"phaser":               "phaser",
				"pixi.js":              "pixi",
				"react":                "react",
				"svelte":               "svelte",
				"three":                "three",
				"vite":                 "vite",
				"vue":                  "vue",
			}
			for dep := range pkg.Dependencies {
				if name, ok := frameworks[strings.ToLower(strings.TrimSpace(dep))]; ok {
					addFramework(name, true)
				}
			}
			for dep := range pkg.DevDependencies {
				if name, ok := frameworks[strings.ToLower(strings.TrimSpace(dep))]; ok {
					addFramework(name, true)
				}
			}
		}
	}
	if projectBriefMentionsFramework(root, "phaser") || repoHasPhaserScriptTag(root) {
		addFramework("phaser", false)
	}
	info.UsesFramework = len(info.FrameworkNames) > 0
	return info
}

func browserFrameworkCompletionBlockers(root Root, session Session, requireBuildRun bool) []string {
	info := repoBrowserFrameworkInfo(root)
	if !info.UsesFramework {
		return nil
	}
	var blockers []string
	frameworks := strings.Join(info.FrameworkNames, ", ")
	if frameworks == "" {
		frameworks = "browser framework"
	}
	if frameworkListContains(info.FrameworkNames, "phaser") && !frameworkListContains(info.DeclaredFrameworkNames, "phaser") {
		if info.HasPackageManifest {
			blockers = append(blockers, "project references Phaser but package.json does not declare a local phaser dependency; add phaser to package.json instead of relying on CDN-only runtime")
		} else {
			blockers = append(blockers, "project references Phaser but has no package.json; create a JavaScript package manifest with a local phaser dependency and deterministic npm run build")
		}
	}
	if !info.HasPackageManifest {
		blockers = append(blockers, fmt.Sprintf("project references %s but no package.json is present; add package.json with a deterministic build/static validation command such as npm run build", frameworks))
	} else if len(info.NoopBuildScripts) > 0 && !info.HasBuildScript {
		blockers = append(blockers, fmt.Sprintf("package.json build script for %s is a no-op (%s); replace it with a deterministic build/static validation command such as vite build, tsc --noEmit, or another command that can fail when the browser app is broken", frameworks, strings.Join(info.NoopBuildScripts, ", ")))
	} else if !info.HasBuildScript {
		blockers = append(blockers, fmt.Sprintf("package.json declares %s but no build script; add a deterministic build/static validation command such as npm run build", frameworks))
	} else if requireBuildRun {
		counts := session.ToolCounts
		if counts == nil || counts[buildCommandSuccessKey] == 0 {
			blockers = append(blockers, fmt.Sprintf("package.json declares %s but npm run build or equivalent has not passed in this job", frameworks))
		}
	}
	blockers = append(blockers, browserFrameworkSourceFindings(root)...)
	return blockers
}

func engineerBrowserFrameworkCompletionBlockers(root Root, session Session) []string {
	blockers := browserFrameworkCompletionBlockers(root, session, true)
	if !browserFrameworkRequiresProductSmoke(root) {
		return blockers
	}
	counts := session.ToolCounts
	if counts == nil || counts[browserProductSmokeSuccessKey] == 0 {
		blockers = append(blockers, "browser-framework product smoke has not passed in this job; run "+browserProductSmokeCommandGuidance(root)+". node --check, grep-only evidence, and repo-root scratch scripts are insufficient")
	}
	return blockers
}

func browserProductSmokeCommandGuidance(root Root) string {
	info := repoBrowserFrameworkInfo(root)
	if frameworkListContains(info.FrameworkNames, "phaser") {
		return `shell_exec argv ["node","-e","const fs=require('fs'); const htmlPath=['src/index.html','index.html'].find(p=>fs.existsSync(p)); if(!htmlPath) throw new Error('missing index.html'); const html=fs.readFileSync(htmlPath,'utf8'); const lower=html.toLowerCase(); if(lower.includes('phaser')&&(lower.includes('cdn')||lower.includes('http'))) throw new Error('CDN Phaser script tag is not bundled'); if(!html.includes('main.js')) throw new Error('missing main.js module script'); const mainPath=fs.existsSync('src/main.js')?'src/main.js':'main.js'; const main=fs.readFileSync(mainPath,'utf8'); if(!main.includes(\"import Phaser from 'phaser'\")&&!main.includes('import Phaser from \"phaser\"')) throw new Error('missing import Phaser from phaser'); const games=main.split('new Phaser.Game').length-1; if(games!==1) throw new Error('expected exactly one new Phaser.Game'); if(!main.includes('parent')) throw new Error('missing parent game container'); console.log('browser smoke: Phaser canvas #game new Phaser.Game');"]`
	}
	if frameworkListContains(info.FrameworkNames, "react") {
		return `shell_exec argv ["node","-e","const fs=require('fs'); const html=fs.readFileSync('index.html','utf8'); const main=fs.readFileSync('src/main.jsx','utf8'); const app=fs.readFileSync('src/App.jsx','utf8'); if(!html.includes('/src/main.jsx')) throw new Error('missing main.jsx module script'); if(!main.includes('createRoot')) throw new Error('missing createRoot mount'); if(!app.includes('id=\"game\"')) throw new Error('missing #game UI marker'); if(!app.toLowerCase().includes('score')) throw new Error('missing score UI state'); console.log('browser smoke: React document.querySelector #game score UI state');"]`
	}
	return "Playwright/Puppeteer or an equivalent source/runtime assertion that proves the browser app mounts real UI state"
}

func browserFrameworkTerminalDispositionGuidance(root Root, session Session) string {
	info := repoBrowserFrameworkInfo(root)
	if !info.UsesFramework {
		return ""
	}
	counts := session.ToolCounts
	if counts == nil {
		counts = map[string]int{}
	}
	sourceFindings := browserFrameworkSourceFindings(root)
	if len(sourceFindings) > 0 || !info.HasBuildScript {
		blockers := browserFrameworkCompletionBlockers(root, session, false)
		return "Call job_disposition_record with status changes_requested, ticket_id, next_need implementation_rework, feedback.for_role engineer, and evidence_links explaining that browser-framework completion is not proven: " + strings.Join(blockers, "; ") + "."
	}
	if counts[buildCommandSuccessKey] == 0 {
		return "Run the browser-framework build command such as npm run build before approval, or record job_disposition_record with status changes_requested if the project has no runnable build validation."
	}
	if browserFrameworkRequiresProductSmoke(root) && counts[browserProductSmokeSuccessKey] == 0 {
		return "Run a browser product smoke or equivalent source/runtime assertion that checks real product UI state such as Phaser game/canvas behavior before approval, such as " + browserProductSmokeCommandGuidance(root) + ", or record job_disposition_record with status changes_requested."
	}
	return ""
}

func browserFrameworkRequiresProductSmoke(root Root) bool {
	info := repoBrowserFrameworkInfo(root)
	return info.UsesFramework
}

func staticBrowserRequiresProductSmoke(root Root) bool {
	if repoBrowserFrameworkInfo(root).UsesFramework {
		return false
	}
	return repoHasStaticBrowserSurface(root)
}

func staticBrowserCompletionBlockers(root Root, session Session) []string {
	if !staticBrowserRequiresProductSmoke(root) {
		return nil
	}
	var blockers []string
	if counts := session.ToolCounts; counts == nil || counts[staticProductSmokeSuccessKey] == 0 {
		blockers = append(blockers, "static browser product smoke has not passed in this job; start the static server on an application port such as 5173/5174 with background:true, run a separate curl -fsS http://127.0.0.1:<port>/ probe, stop the tracked PID, and record those exact commands")
	}
	if findings := packageRuntimeScriptFindings(root); len(findings) > 0 {
		blockers = append(blockers, findings...)
	}
	return blockers
}

func repoHasStaticBrowserSurface(root Root) bool {
	for _, rel := range []string{
		"index.html",
		filepath.ToSlash(filepath.Join("src", "index.html")),
		filepath.ToSlash(filepath.Join("public", "index.html")),
	} {
		if _, err := root.RepoFS().Stat(filepath.FromSlash(rel)); err == nil {
			return true
		}
	}
	return packageRuntimeScriptFindings(root) != nil
}

func packageRuntimeScriptFindings(root Root) []string {
	data, err := root.RepoFS().ReadFile("package.json")
	if err != nil {
		return nil
	}
	var pkg struct {
		Scripts map[string]string `json:"scripts"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil
	}
	var findings []string
	for name, script := range pkg.Scripts {
		if port := reservedHarnessPortInScript(script); port != "" {
			findings = append(findings, fmt.Sprintf("package.json script %q uses reserved MARS port %s", name, port))
		}
		if smokeScriptName(name) && packageSmokeScriptNoop(script) {
			findings = append(findings, fmt.Sprintf("package.json script %q is canned/no-op smoke evidence", name))
		}
	}
	return findings
}

func browserFrameworkSourceFindings(root Root) []string {
	info := repoBrowserFrameworkInfo(root)
	if !frameworkListContains(info.FrameworkNames, "phaser") {
		return nil
	}
	var findings []string
	findings = append(findings, phaserGoModuleFindings(root)...)
	jsModules := map[string]string{}
	jsModulePaths := []string{}
	htmlFiles := map[string]string{}
	htmlPaths := []string{}
	_ = fs.WalkDir(root.RepoFS(), ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			switch strings.ToLower(d.Name()) {
			case ".git", ".harness", "node_modules", "vendor", "dist", "build", "coverage":
				return fs.SkipDir
			default:
				return nil
			}
		}
		rel := filepath.ToSlash(path)
		lowerRel := strings.ToLower(rel)
		if !javascriptSourcePath(lowerRel) && !htmlSourcePath(lowerRel) {
			return nil
		}
		if browserFrameworkValidationHelperPath(lowerRel) {
			return nil
		}
		data, err := root.RepoFS().ReadFile(path)
		if err != nil {
			return nil
		}
		content := string(data)
		if (lowerRel == "vite.config.js" || lowerRel == "vite.config.ts") && viteConfigExternalizesPhaser(content) {
			findings = append(findings, fmt.Sprintf("%s externalizes phaser from the Vite bundle; remove rollupOptions.external for phaser so the browser build proves the local dependency loads", rel))
		}
		if htmlSourcePath(lowerRel) {
			findings = append(findings, phaserHTMLFindings(rel, content)...)
			htmlFiles[rel] = content
			htmlPaths = append(htmlPaths, rel)
			return nil
		}
		jsModules[rel] = content
		jsModulePaths = append(jsModulePaths, rel)
		for _, id := range []string{"preload", "create", "update"} {
			if phaserSceneReferencesIdentifier(content, id) && !jsDefinesOrImportsIdentifier(content, id) {
				findings = append(findings, fmt.Sprintf("%s references Phaser scene callback %q without defining or importing it in the same module", rel, id))
			}
		}
		findings = append(findings, phaserSingleFileSourceFindings(rel, content)...)
		return nil
	})
	for _, rel := range jsModulePaths {
		findings = append(findings, jsLocalNamedImportFindings(rel, jsModules[rel], jsModules)...)
		findings = append(findings, jsMissingLocalExportImportFindings(rel, jsModules[rel], jsModules)...)
	}
	for _, rel := range htmlPaths {
		findings = append(findings, htmlClassicScriptModuleFindings(rel, htmlFiles[rel], jsModules)...)
	}
	return findings
}

func javascriptSourcePath(lowerRel string) bool {
	return strings.HasSuffix(lowerRel, ".js") ||
		strings.HasSuffix(lowerRel, ".mjs") ||
		strings.HasSuffix(lowerRel, ".jsx") ||
		strings.HasSuffix(lowerRel, ".ts") ||
		strings.HasSuffix(lowerRel, ".tsx")
}

func browserFrameworkValidationHelperPath(lowerRel string) bool {
	lowerRel = filepath.ToSlash(strings.ToLower(strings.TrimSpace(lowerRel)))
	base := filepath.Base(lowerRel)
	if strings.Contains(base, ".test.") || strings.Contains(base, ".spec.") {
		return true
	}
	if strings.HasPrefix(lowerRel, "test/") || strings.HasPrefix(lowerRel, "tests/") {
		return true
	}
	if strings.HasPrefix(lowerRel, "scripts/") &&
		(strings.Contains(base, "validate") || strings.Contains(base, "smoke") || strings.Contains(base, "probe")) {
		return true
	}
	if strings.Contains(lowerRel, "/") {
		return false
	}
	return strings.HasPrefix(base, "validate-") ||
		strings.HasSuffix(base, "-validation.js") ||
		strings.Contains(base, "smoke") ||
		strings.Contains(base, "probe")
}

func htmlSourcePath(lowerRel string) bool {
	return strings.HasSuffix(lowerRel, ".html") || strings.HasSuffix(lowerRel, ".htm")
}

func repoHasPhaserScriptTag(root Root) bool {
	found := false
	_ = fs.WalkDir(root.RepoFS(), ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || found {
			return nil
		}
		if d.IsDir() {
			switch strings.ToLower(d.Name()) {
			case ".git", ".harness", "node_modules", "vendor", "dist", "build", "coverage":
				return fs.SkipDir
			default:
				return nil
			}
		}
		if !strings.HasSuffix(strings.ToLower(d.Name()), ".html") {
			return nil
		}
		data, err := root.RepoFS().ReadFile(path)
		if err != nil {
			return nil
		}
		lower := strings.ToLower(string(data))
		if strings.Contains(lower, "<script") && strings.Contains(lower, "phaser") {
			found = true
			return fs.SkipAll
		}
		return nil
	})
	return found
}

func phaserGoModuleFindings(root Root) []string {
	var findings []string
	_ = fs.WalkDir(root.RepoFS(), ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			switch strings.ToLower(d.Name()) {
			case ".git", ".harness", "node_modules", "vendor", "dist", "build", "coverage":
				return fs.SkipDir
			default:
				return nil
			}
		}
		if strings.ToLower(d.Name()) != "go.mod" {
			return nil
		}
		data, err := root.RepoFS().ReadFile(path)
		if err != nil {
			return nil
		}
		if !strings.Contains(strings.ToLower(string(data)), "phaser") {
			return nil
		}
		findings = append(findings, fmt.Sprintf("%s declares a Phaser-related Go module dependency; Phaser JS targets should use package.json with the phaser npm dependency, not go.mod", filepath.ToSlash(path)))
		return nil
	})
	return findings
}

func phaserHTMLFindings(rel, content string) []string {
	lower := strings.ToLower(content)
	if strings.Contains(lower, "<script") && strings.Contains(lower, "phaser") &&
		(strings.Contains(lower, "http://") || strings.Contains(lower, "https://") || strings.Contains(lower, "cdn.")) {
		return []string{fmt.Sprintf("%s loads Phaser from a CDN/script tag; Phaser JS targets must use the local phaser npm dependency through the module/bundler entrypoint", rel)}
	}
	return nil
}

func phaserSingleFileSourceFindings(rel, content string) []string {
	var findings []string
	findings = append(findings, phaserMissingImportFindings(rel, content)...)
	findings = append(findings, phaserUnboundSceneHelperFindings(rel, content)...)
	findings = append(findings, phaserSceneContextFindings(rel, content)...)
	findings = append(findings, phaserGameConstructionFindings(rel, content)...)
	return findings
}

func frameworkListContains(values []string, want string) bool {
	want = strings.ToLower(strings.TrimSpace(want))
	for _, value := range values {
		if strings.ToLower(strings.TrimSpace(value)) == want {
			return true
		}
	}
	return false
}

func phaserSceneReferencesIdentifier(content, id string) bool {
	pattern := regexp.MustCompile(`(?m)\b` + regexp.QuoteMeta(id) + `\s*:\s*` + regexp.QuoteMeta(id) + `\b`)
	return pattern.MatchString(content)
}

func jsDefinesOrImportsIdentifier(content, id string) bool {
	quoted := regexp.QuoteMeta(id)
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?m)\bfunction\s+` + quoted + `\s*\(`),
		regexp.MustCompile(`(?m)\b(?:const|let|var)\s+` + quoted + `\b`),
		regexp.MustCompile(`(?m)\bimport\b[^\n;]*\b` + quoted + `\b`),
	}
	for _, pattern := range patterns {
		if pattern.MatchString(content) {
			return true
		}
	}
	return false
}

func phaserUnboundSceneHelperFindings(rel, content string) []string {
	var findings []string
	helperRe := regexp.MustCompile(`(?s)function\s+([A-Za-z_$][A-Za-z0-9_$]*)\s*\([^)]*\)\s*\{[^}]*\bthis\.add\.`)
	for _, match := range helperRe.FindAllStringSubmatch(content, -1) {
		if len(match) < 2 {
			continue
		}
		name := match[1]
		bareCall := regexp.MustCompile(`\b` + regexp.QuoteMeta(name) + `\s*\(\s*\)`)
		callCount := len(bareCall.FindAllStringIndex(content, -1))
		if callCount > 1 {
			findings = append(findings, fmt.Sprintf("%s defines Phaser helper %s using this.add but calls it without binding or passing the scene", rel, name))
		}
	}
	return findings
}

func phaserMissingImportFindings(rel, content string) []string {
	if !strings.Contains(content, "Phaser.") && !regexp.MustCompile(`\bextends\s+Phaser\.`).MatchString(content) {
		return nil
	}
	if jsDefinesOrImportsIdentifier(content, "Phaser") {
		return nil
	}
	return []string{fmt.Sprintf("%s uses Phaser global APIs without importing or defining Phaser in the same module; import Phaser from 'phaser' or avoid referencing the global", rel)}
}

func phaserSceneContextFindings(rel, content string) []string {
	var findings []string
	sceneAPIRe := regexp.MustCompile(`\bthis\.(?:add|cameras|input|load|make|physics|time|tweens)\b`)
	if strings.Contains(content, "new Phaser.Game") && strings.Contains(content, ".bind(this)") && sceneAPIRe.MatchString(content) {
		findings = append(findings, fmt.Sprintf("%s binds Phaser scene callbacks to wrapper this while using Phaser scene APIs; Phaser scene lifecycle methods must run with scene context or receive the scene explicitly", rel))
	}
	gameInstanceSceneAPIRe := regexp.MustCompile(`\bthis\.(?:game|gameInstance)\.(?:add|input|load|make|physics)\b`)
	if gameInstanceSceneAPIRe.MatchString(content) {
		findings = append(findings, fmt.Sprintf("%s uses the Phaser game instance as a scene API surface; drawing, input, and factory calls must run from the Phaser scene context", rel))
	}
	return findings
}

func phaserGameConstructionFindings(rel, content string) []string {
	if !strings.Contains(content, "new Phaser.Game") {
		return nil
	}
	var findings []string
	count := strings.Count(content, "new Phaser.Game")
	if count > 1 {
		findings = append(findings, fmt.Sprintf("%s constructs Phaser.Game %d times; create exactly one game instance from the browser entrypoint", rel, count))
	}
	for _, callback := range []string{"preload", "create", "update"} {
		if phaserNewGameInsideFunction(content, callback) {
			findings = append(findings, fmt.Sprintf("%s constructs new Phaser.Game inside scene callback %s; create the game once at module startup and let scene callbacks use the scene instance", rel, callback))
		}
	}
	return findings
}

func phaserNewGameInsideFunction(content, name string) bool {
	re := regexp.MustCompile(`\bfunction\s+` + regexp.QuoteMeta(name) + `\s*\([^)]*\)\s*\{`)
	for _, loc := range re.FindAllStringIndex(content, -1) {
		open := strings.LastIndex(content[loc[0]:loc[1]], "{")
		if open < 0 {
			continue
		}
		open += loc[0]
		close := jsMatchingBrace(content, open)
		if close < 0 {
			continue
		}
		body := content[open+1 : close]
		if regexp.MustCompile(`\bnew\s+Phaser\.Game\b`).MatchString(body) {
			return true
		}
	}
	return false
}

func jsMatchingBrace(content string, open int) int {
	if open < 0 || open >= len(content) || content[open] != '{' {
		return -1
	}
	depth := 0
	inSingle := false
	inDouble := false
	inTemplate := false
	lineComment := false
	blockComment := false
	escaped := false
	for i := open; i < len(content); i++ {
		ch := content[i]
		next := byte(0)
		if i+1 < len(content) {
			next = content[i+1]
		}
		if lineComment {
			if ch == '\n' || ch == '\r' {
				lineComment = false
			}
			continue
		}
		if blockComment {
			if ch == '*' && next == '/' {
				blockComment = false
				i++
			}
			continue
		}
		if inSingle || inDouble || inTemplate {
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if inSingle && ch == '\'' {
				inSingle = false
			}
			if inDouble && ch == '"' {
				inDouble = false
			}
			if inTemplate && ch == '`' {
				inTemplate = false
			}
			continue
		}
		if ch == '/' && next == '/' {
			lineComment = true
			i++
			continue
		}
		if ch == '/' && next == '*' {
			blockComment = true
			i++
			continue
		}
		switch ch {
		case '\'':
			inSingle = true
		case '"':
			inDouble = true
		case '`':
			inTemplate = true
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func jsLocalNamedImportFindings(rel, content string, modules map[string]string) []string {
	var findings []string
	importRe := regexp.MustCompile(`(?m)\bimport\s*(?:type\s*)?\{([^}]+)\}\s*from\s*["']([^"']+)["']`)
	for _, match := range importRe.FindAllStringSubmatch(content, -1) {
		if len(match) < 3 {
			continue
		}
		spec := strings.TrimSpace(match[2])
		if !strings.HasPrefix(spec, ".") {
			continue
		}
		names := jsNamedImportNames(match[1])
		if len(names) == 0 {
			continue
		}
		targetRel, ok := resolveLocalJSModuleRel(rel, spec, modules)
		if !ok {
			findings = append(findings, fmt.Sprintf("%s imports {%s} from %s but no matching local module file was found", rel, strings.Join(names, ", "), spec))
			continue
		}
		exported := jsExportedNames(modules[targetRel])
		for _, name := range names {
			if name == "default" {
				continue
			}
			if !exported[name] {
				findings = append(findings, fmt.Sprintf("%s imports {%s} from %s but %s does not export it", rel, name, targetRel, targetRel))
			}
		}
	}
	return findings
}

func jsMissingLocalExportImportFindings(rel, content string, modules map[string]string) []string {
	if !jsContainsModuleSyntax(content) {
		return nil
	}
	exportsByName := map[string][]string{}
	for moduleRel, moduleContent := range modules {
		if moduleRel == rel {
			continue
		}
		for name := range jsExportedNames(moduleContent) {
			exportsByName[name] = append(exportsByName[name], moduleRel)
		}
	}
	var findings []string
	for name, exporters := range exportsByName {
		if !jsUsesIdentifier(content, name) || jsDefinesOrImportsIdentifier(content, name) {
			continue
		}
		sort.Strings(exporters)
		findings = append(findings, fmt.Sprintf("%s uses %s but does not import it from local module %s", rel, name, exporters[0]))
	}
	sort.Strings(findings)
	return findings
}

func jsUsesIdentifier(content, id string) bool {
	quoted := regexp.QuoteMeta(id)
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?m)\b` + quoted + `\s*\(`),
		regexp.MustCompile(`(?m)\bnew\s+` + quoted + `\b`),
		regexp.MustCompile(`(?m)\b` + quoted + `\s*\.`),
	}
	for _, pattern := range patterns {
		if pattern.MatchString(content) {
			return true
		}
	}
	return false
}

func jsNamedImportNames(raw string) []string {
	var names []string
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		part = strings.TrimPrefix(part, "type ")
		if part == "" {
			continue
		}
		asParts := regexp.MustCompile(`(?i)\s+as\s+`).Split(part, 2)
		name := strings.TrimSpace(asParts[0])
		fields := strings.Fields(name)
		if len(fields) == 0 {
			continue
		}
		name = fields[0]
		if regexp.MustCompile(`^[A-Za-z_$][A-Za-z0-9_$]*$`).MatchString(name) {
			names = append(names, name)
		}
	}
	return names
}

func resolveLocalJSModuleRel(sourceRel, spec string, modules map[string]string) (string, bool) {
	spec = strings.TrimSpace(spec)
	if i := strings.IndexAny(spec, "?#"); i >= 0 {
		spec = spec[:i]
	}
	if spec == "" || !strings.HasPrefix(spec, ".") {
		return "", false
	}
	sourceDir := filepath.ToSlash(filepath.Dir(sourceRel))
	if sourceDir == "." {
		sourceDir = ""
	}
	base := filepath.ToSlash(filepath.Clean(filepath.Join(sourceDir, spec)))
	if base == "." || strings.HasPrefix(base, "../") || base == ".." {
		return "", false
	}
	candidates := []string{base}
	if ext := strings.ToLower(filepath.Ext(base)); ext == "" {
		for _, suffix := range []string{".js", ".mjs", ".jsx", ".ts", ".tsx"} {
			candidates = append(candidates, base+suffix)
		}
		for _, suffix := range []string{"index.js", "index.mjs", "index.jsx", "index.ts", "index.tsx"} {
			candidates = append(candidates, filepath.ToSlash(filepath.Join(base, suffix)))
		}
	}
	for _, candidate := range candidates {
		candidate = cleanRepoPath(candidate)
		if _, ok := modules[candidate]; ok {
			return candidate, true
		}
	}
	return "", false
}

func jsExportedNames(content string) map[string]bool {
	exported := map[string]bool{}
	for _, pattern := range []*regexp.Regexp{
		regexp.MustCompile(`(?m)\bexport\s+(?:async\s+)?function\s+([A-Za-z_$][A-Za-z0-9_$]*)\b`),
		regexp.MustCompile(`(?m)\bexport\s+class\s+([A-Za-z_$][A-Za-z0-9_$]*)\b`),
		regexp.MustCompile(`(?m)\bexport\s+(?:const|let|var)\s+([A-Za-z_$][A-Za-z0-9_$]*)\b`),
	} {
		for _, match := range pattern.FindAllStringSubmatch(content, -1) {
			if len(match) >= 2 {
				exported[match[1]] = true
			}
		}
	}
	exportListRe := regexp.MustCompile(`(?s)\bexport\s*\{([^}]+)\}`)
	for _, match := range exportListRe.FindAllStringSubmatch(content, -1) {
		if len(match) < 2 {
			continue
		}
		for _, part := range strings.Split(match[1], ",") {
			part = strings.TrimSpace(part)
			part = strings.TrimPrefix(part, "type ")
			if part == "" {
				continue
			}
			asParts := regexp.MustCompile(`(?i)\s+as\s+`).Split(part, 2)
			name := strings.TrimSpace(asParts[len(asParts)-1])
			if fields := strings.Fields(name); len(fields) > 0 {
				name = fields[0]
			}
			if regexp.MustCompile(`^[A-Za-z_$][A-Za-z0-9_$]*$`).MatchString(name) {
				exported[name] = true
			}
		}
	}
	return exported
}

func htmlClassicScriptModuleFindings(rel, content string, modules map[string]string) []string {
	var findings []string
	tagRe := regexp.MustCompile(`(?is)<script\b([^>]*)>`)
	srcRe := regexp.MustCompile(`(?is)\bsrc\s*=\s*["']([^"']+)["']`)
	typeModuleRe := regexp.MustCompile(`(?is)\btype\s*=\s*["']module["']`)
	for _, tag := range tagRe.FindAllStringSubmatch(content, -1) {
		if len(tag) < 2 {
			continue
		}
		attrs := tag[1]
		if typeModuleRe.MatchString(attrs) {
			continue
		}
		src := srcRe.FindStringSubmatch(attrs)
		if len(src) < 2 {
			continue
		}
		targetRel, ok := resolveHTMLScriptRel(rel, src[1], modules)
		if !ok {
			continue
		}
		if jsContainsModuleSyntax(modules[targetRel]) {
			findings = append(findings, fmt.Sprintf("%s loads %s as a classic script but %s contains ES module import/export syntax; use type=\"module\" or bundle the entrypoint", rel, targetRel, targetRel))
		}
	}
	return findings
}

func resolveHTMLScriptRel(sourceRel, spec string, modules map[string]string) (string, bool) {
	spec = strings.TrimSpace(spec)
	if strings.HasPrefix(spec, "http://") || strings.HasPrefix(spec, "https://") || strings.HasPrefix(spec, "//") {
		return "", false
	}
	if i := strings.IndexAny(spec, "?#"); i >= 0 {
		spec = spec[:i]
	}
	sourceDir := filepath.ToSlash(filepath.Dir(sourceRel))
	if sourceDir == "." {
		sourceDir = ""
	}
	base := filepath.ToSlash(filepath.Clean(filepath.Join(sourceDir, spec)))
	if base == "." || strings.HasPrefix(base, "../") || base == ".." {
		return "", false
	}
	base = cleanRepoPath(base)
	_, ok := modules[base]
	return base, ok
}

func jsContainsModuleSyntax(content string) bool {
	return regexp.MustCompile(`(?m)^\s*(?:import|export)\b`).MatchString(content)
}

func checkEngineerBrowserFrameworkTicketDoneMovePolicy(root Root, session Session, hasSession bool, raw json.RawMessage) error {
	if !hasSession || strings.ToLower(strings.TrimSpace(session.Role)) != "engineer" {
		return nil
	}
	args, err := decodeShellExecArgs(raw)
	if err != nil {
		return nil
	}
	fields := args.Argv
	if strings.TrimSpace(args.ShellCommand) != "" {
		fields = shellFieldsPreserveCase(args.ShellCommand)
	}
	if len(ticketDoneMoveSources(fields)) == 0 {
		return nil
	}
	if blockers := engineerBrowserFrameworkCompletionBlockers(root, session); len(blockers) > 0 {
		return fmt.Errorf(
			"policy: engineer cannot move browser-framework ticket to docs/tickets/done yet: %s. Fix the implementation or package build surface, rerun validation, then update evidence and move the ticket",
			strings.Join(blockers, "; "),
		)
	}
	return nil
}

func reservedHarnessPortInScript(script string) string {
	for _, port := range []string{"18080", "18081", "18082", "18083", "18084", "18085", "18086", "18087", "18088", "18089"} {
		if regexp.MustCompile(`(^|[^0-9])` + regexp.QuoteMeta(port) + `([^0-9]|$)`).MatchString(script) {
			return port
		}
	}
	return ""
}

func packageBuildScriptNoop(script string) bool {
	script = strings.TrimSpace(strings.ToLower(script))
	if script == "" {
		return true
	}
	if packageBuildScriptOnlySyntaxCheck(script) {
		return true
	}
	for _, marker := range []string{
		"vite build", "next build", "webpack", "rollup", "parcel", "astro build",
		"tsc", "esbuild", "npm run", "pnpm run", "yarn ", "bun run",
		"node ", "deno ", "make ",
	} {
		if strings.Contains(script, marker) {
			return false
		}
	}
	parts := strings.Split(script, "&&")
	if len(parts) == 0 {
		return true
	}
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if part == ":" || part == "true" || part == "exit 0" ||
			strings.HasPrefix(part, "echo ") ||
			strings.HasPrefix(part, "printf ") ||
			strings.HasPrefix(part, "mkdir ") ||
			strings.HasPrefix(part, "cp ") ||
			strings.HasPrefix(part, "copy ") ||
			strings.HasPrefix(part, "rsync ") ||
			strings.HasPrefix(part, "touch ") ||
			strings.HasPrefix(part, "live-server") ||
			strings.HasPrefix(part, "http-server") ||
			strings.HasPrefix(part, "serve ") ||
			strings.Contains(part, "python -m http.server") ||
			strings.Contains(part, "python3 -m http.server") {
			continue
		}
		return false
	}
	return true
}

func packageBuildScriptOnlySyntaxCheck(script string) bool {
	parts := strings.Split(script, "&&")
	checked := false
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if strings.HasPrefix(part, "node --check ") || strings.HasPrefix(part, "node -c ") {
			checked = true
			continue
		}
		return false
	}
	return checked
}
