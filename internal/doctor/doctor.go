/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/guardrails.md
- docs/design-docs/release-versioning.md
- docs/design-docs/self-reflective-telemetry.md
- docs/features/F-004-target-harness-lifecycle.md
- docs/features/F-009-release-update-lifecycle.md
- docs/features/F-007-guardrails-and-safety.md
- docs/features/F-012-self-improvement-loop.md
- docs/product-specs/product-surface.md
*/
package doctor

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/greaveselliott/mars-harness/internal/githubauth"
	"github.com/greaveselliott/mars-harness/internal/hardware"
	"github.com/greaveselliott/mars-harness/internal/operatingmodel"
	"github.com/greaveselliott/mars-harness/internal/planhygiene"
	"github.com/greaveselliott/mars-harness/internal/roleregistry"
	ticketstate "github.com/greaveselliott/mars-harness/internal/tickets"
	"github.com/greaveselliott/mars-harness/internal/tools"
	"github.com/greaveselliott/mars-harness/internal/updatecheck"
)

// CheckResult represents the outcome of a single health check.
type CheckResult struct {
	Name     string        `json:"name"`
	Status   string        `json:"status"` // "ok", "warn", "fail"
	Message  string        `json:"message"`
	Duration time.Duration `json:"duration"`
	Fix      string        `json:"fix,omitempty"`
}

// Config controls doctor behaviour.
type Config struct {
	ConfigPath       string
	DBPath           string
	RepoPath         string
	CurrentVersion   string
	LatestReleaseURL string
	SkipRemote       bool
	JSONOutput       bool
}

const (
	statusOK   = "ok"
	statusWarn = "warn"
	statusFail = "fail"

	minGoMajor      = 1
	minGoMinor      = 22
	minDiskSpaceMiB = 5120 // 5 GiB
)

// Run executes all health checks and returns results.
func Run(cfg Config) []CheckResult {
	checks := []func(Config) CheckResult{
		checkGoVersion,
		checkConfigFile,
		checkModelRegistry,
		checkModelsDir,
		checkProfileRequiredModels,
		checkDBAccessible,
		checkLlamaServer,
		checkDiskSpace,
		checkPrivateReleaseAuth,
		checkVersionDrift,
		checkOperatingModelHealth,
		checkDeterministicRemediationHealth,
		checkRoleRegistryHealth,
		checkActivePlanHygiene,
		checkTicketDrainHealth,
		checkWorkspaceHygieneHealth,
	}

	results := make([]CheckResult, 0, len(checks))
	for _, check := range checks {
		result := check(cfg)
		if result.Duration == 0 {
			result.Duration = time.Nanosecond
		}
		slog.Info("doctor check",
			"name", result.Name,
			"status", result.Status,
			"message", result.Message,
			"duration", result.Duration,
		)
		results = append(results, result)
	}
	return results
}

func checkPrivateReleaseAuth(cfg Config) CheckResult {
	start := time.Now()
	name := "private-release-auth"
	if cfg.SkipRemote {
		return CheckResult{
			Name:     name,
			Status:   statusOK,
			Message:  "remote private release auth check skipped",
			Duration: nonZeroDurationSince(start),
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	report := githubauth.Check(ctx, githubauth.Options{})
	if report.Status == githubauth.StatusOK {
		return CheckResult{
			Name:     name,
			Status:   statusOK,
			Message:  fmt.Sprintf("%s via %s", report.Message, report.AuthSource),
			Duration: nonZeroDurationSince(start),
		}
	}
	return CheckResult{
		Name:     name,
		Status:   statusWarn,
		Message:  report.Message,
		Duration: nonZeroDurationSince(start),
		Fix:      report.NextAction,
	}
}

func checkWorkspaceHygieneHealth(cfg Config) CheckResult {
	start := time.Now()
	name := "workspace-hygiene"
	if strings.TrimSpace(cfg.RepoPath) == "" {
		return CheckResult{
			Name:     name,
			Status:   statusOK,
			Message:  "repo not supplied; workspace hygiene skipped",
			Duration: nonZeroDurationSince(start),
		}
	}
	root, err := tools.NewRoot(cfg.RepoPath)
	if err != nil {
		return CheckResult{
			Name:     name,
			Status:   statusWarn,
			Message:  err.Error(),
			Duration: nonZeroDurationSince(start),
			Fix:      "run doctor with --repo pointing at a valid repository root",
		}
	}
	report, err := tools.AuditWorkspaceHygiene(context.Background(), root, tools.WorkspaceHygieneOptions{Mode: "pre_job"})
	if err != nil {
		return CheckResult{
			Name:     name,
			Status:   statusWarn,
			Message:  err.Error(),
			Duration: nonZeroDurationSince(start),
			Fix:      "run 'mars-harness tools run workspace_hygiene --repo <path> --trust contributor' for a detailed recipe",
		}
	}
	if len(report.Findings) == 0 {
		return CheckResult{
			Name:     name,
			Status:   statusOK,
			Message:  "workspace hygiene is clean",
			Duration: nonZeroDurationSince(start),
		}
	}
	status := statusWarn
	if report.Blocking {
		status = statusFail
	}
	if report.Blocking && report.AutoRepairable {
		status = statusWarn
		return CheckResult{
			Name:     name,
			Status:   status,
			Message:  report.Message + " (auto-repairable .gitignore policy)",
			Duration: nonZeroDurationSince(start),
			Fix:      "mars-harness start will commit a .gitignore-only hygiene repair before loading the model; or add the missing generated ignore entries manually",
		}
	}
	return CheckResult{
		Name:     name,
		Status:   status,
		Message:  report.Message,
		Duration: nonZeroDurationSince(start),
		Fix:      report.NextAction,
	}
}

func checkTicketDrainHealth(cfg Config) CheckResult {
	start := time.Now()
	name := "ticket-drain"
	if strings.TrimSpace(cfg.RepoPath) == "" {
		return CheckResult{
			Name:     name,
			Status:   statusOK,
			Message:  "repo not supplied; ticket-drain health skipped",
			Duration: nonZeroDurationSince(start),
		}
	}
	stale, err := ticketstate.StaleInProgress(cfg.RepoPath, time.Now().UTC(), ticketstate.DefaultStaleInProgressAfter)
	if err != nil {
		return CheckResult{
			Name:     name,
			Status:   statusWarn,
			Message:  err.Error(),
			Duration: nonZeroDurationSince(start),
			Fix:      "restore docs/tickets/{backlog,in-progress,done}/ or run 'mars-harness init --repo <path>' for a new target repo",
		}
	}
	if len(stale) == 0 {
		return CheckResult{
			Name:     name,
			Status:   statusOK,
			Message:  "no stale eligible in-progress tickets",
			Duration: nonZeroDurationSince(start),
		}
	}
	return CheckResult{
		Name:     name,
		Status:   statusWarn,
		Message:  fmt.Sprintf("%d stale eligible in-progress ticket(s): %s", len(stale), staleTicketSummary(stale)),
		Duration: nonZeroDurationSince(start),
		Fix:      "complete the ticket, move it back to docs/tickets/backlog with blocker metadata, or add blocked_by linking to a dependency ticket; then run 'mars-harness scan --repo <path> --tickets' or 'mars-harness run janitor --repo <path>'",
	}
}

func staleTicketSummary(stale []ticketstate.Ticket) string {
	parts := make([]string, 0, len(stale))
	for _, t := range stale {
		parts = append(parts, fmt.Sprintf("%s last_attempt=%s", t.RelPath, t.LastActivityLabel()))
	}
	return strings.Join(parts, "; ")
}

func checkActivePlanHygiene(cfg Config) CheckResult {
	start := time.Now()
	name := "active-plan-hygiene"
	if strings.TrimSpace(cfg.RepoPath) == "" {
		return CheckResult{
			Name:     name,
			Status:   statusOK,
			Message:  "repo not supplied; active-plan hygiene skipped",
			Duration: nonZeroDurationSince(start),
		}
	}
	report, err := planhygiene.CheckRepo(cfg.RepoPath)
	if err != nil {
		return CheckResult{
			Name:     name,
			Status:   statusWarn,
			Message:  err.Error(),
			Duration: nonZeroDurationSince(start),
			Fix:      "run 'mars-harness doctor --repo <path>' with a valid git checkout containing docs/exec-plans/",
		}
	}
	if !report.OK() {
		return CheckResult{
			Name:     name,
			Status:   statusWarn,
			Message:  report.Summary(),
			Duration: nonZeroDurationSince(start),
			Fix:      report.Remediation(),
		}
	}
	return CheckResult{
		Name:     name,
		Status:   statusOK,
		Message:  "active-plan hygiene is clean",
		Duration: nonZeroDurationSince(start),
	}
}

func checkRoleRegistryHealth(cfg Config) CheckResult {
	start := time.Now()
	name := "role-registry"
	if strings.TrimSpace(cfg.RepoPath) == "" {
		return CheckResult{
			Name:     name,
			Status:   statusOK,
			Message:  "repo not supplied; role-registry health skipped",
			Duration: nonZeroDurationSince(start),
		}
	}
	report, err := roleregistry.CheckRepo(cfg.RepoPath)
	if err != nil {
		return CheckResult{
			Name:     name,
			Status:   statusWarn,
			Message:  err.Error(),
			Duration: nonZeroDurationSince(start),
			Fix:      "run 'mars-harness init --repo <path>' for new repos, or restore .harness/manifest.yaml before checking role registry",
		}
	}
	if !report.OK() {
		return CheckResult{
			Name:     name,
			Status:   statusWarn,
			Message:  report.Summary(),
			Duration: nonZeroDurationSince(start),
			Fix:      report.Remediation(),
		}
	}
	return CheckResult{
		Name:     name,
		Status:   statusOK,
		Message:  "role registry matches manifest",
		Duration: nonZeroDurationSince(start),
	}
}

func nonZeroDurationSince(start time.Time) time.Duration {
	elapsed := time.Since(start)
	if elapsed == 0 {
		return time.Nanosecond
	}
	return elapsed
}

func checkOperatingModelHealth(cfg Config) CheckResult {
	start := time.Now()
	name := "operating-model"
	if strings.TrimSpace(cfg.RepoPath) == "" {
		return CheckResult{
			Name:     name,
			Status:   statusOK,
			Message:  "repo not supplied; target operating-model health skipped",
			Duration: time.Since(start),
		}
	}
	report, err := operatingmodel.CheckRepo(cfg.RepoPath)
	if err != nil {
		return CheckResult{
			Name:     name,
			Status:   statusWarn,
			Message:  err.Error(),
			Duration: time.Since(start),
			Fix:      "run 'mars-harness update check --repo <path> --skip-remote'",
		}
	}
	if !report.OK() {
		return CheckResult{
			Name:     name,
			Status:   statusWarn,
			Message:  report.Summary(),
			Duration: time.Since(start),
			Fix:      "run 'mars-harness update harness --repo <path>'; create migration tickets for stale user-owned docs",
		}
	}
	return CheckResult{
		Name:     name,
		Status:   statusOK,
		Message:  "BDD-led goal-driven operating model is present",
		Duration: time.Since(start),
	}
}

func checkVersionDrift(cfg Config) CheckResult {
	start := time.Now()
	name := "version-drift"
	current := strings.TrimSpace(cfg.CurrentVersion)
	if current == "" {
		current = "unknown"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	report, err := updatecheck.Run(ctx, updatecheck.Config{
		CurrentVersion:   current,
		RepoPath:         cfg.RepoPath,
		LatestReleaseURL: cfg.LatestReleaseURL,
		SkipRemote:       cfg.SkipRemote,
	})
	if err != nil {
		return CheckResult{
			Name:     name,
			Status:   statusWarn,
			Message:  err.Error(),
			Duration: time.Since(start),
		}
	}
	for _, component := range []updatecheck.Component{report.Tool, report.Harness} {
		if component.Status == updatecheck.StatusBehind {
			return CheckResult{
				Name:     name,
				Status:   statusWarn,
				Message:  fmt.Sprintf("%s is behind: %s", component.Name, component.Message),
				Duration: time.Since(start),
				Fix:      component.Command,
			}
		}
	}
	if report.Tool.Status == updatecheck.StatusUnknown && !cfg.SkipRemote {
		return CheckResult{
			Name:     name,
			Status:   statusWarn,
			Message:  report.Tool.Message,
			Duration: time.Since(start),
			Fix:      "run 'mars-harness update check --skip-remote' to check local target harness state only",
		}
	}
	if report.Harness.Status == updatecheck.StatusUnknown && cfg.RepoPath != "" && report.Harness.Command != "" {
		return CheckResult{
			Name:     name,
			Status:   statusWarn,
			Message:  report.Harness.Message,
			Duration: time.Since(start),
			Fix:      report.Harness.Command,
		}
	}
	return CheckResult{
		Name:     name,
		Status:   statusOK,
		Message:  "no update drift detected",
		Duration: time.Since(start),
	}
}

func checkModelRegistry(_ Config) CheckResult {
	start := time.Now()
	name := "model-registry"
	for _, profile := range []hardware.Profile{hardware.ProfileCPU, hardware.ProfileLow, hardware.ProfileMedium, hardware.ProfileHigh, hardware.ProfileMulti} {
		for tier, spec := range hardware.DefaultModels(profile) {
			if strings.TrimSpace(spec.Revision) == "" || spec.Revision == "main" || strings.TrimSpace(spec.SHA256) == "" {
				return CheckResult{
					Name:     name,
					Status:   statusFail,
					Message:  fmt.Sprintf("%s/%s is not pinned with immutable revision and SHA256", profile, tier),
					Duration: time.Since(start),
					Fix:      "update internal/hardware/registry.go with a non-main revision and SHA256 for every default model",
				}
			}
			if strings.Contains(spec.DownloadURL(), "/resolve/main/") {
				return CheckResult{
					Name:     name,
					Status:   statusFail,
					Message:  fmt.Sprintf("%s/%s still downloads from resolve/main", profile, tier),
					Duration: time.Since(start),
					Fix:      "pin the model revision before running setup",
				}
			}
		}
	}
	return CheckResult{
		Name:     name,
		Status:   statusOK,
		Message:  "default models pinned by revision and SHA256",
		Duration: time.Since(start),
	}
}

// FormatText renders results as human-readable coloured output.
func FormatText(results []CheckResult) string {
	var b strings.Builder
	for _, r := range results {
		icon := statusIcon(r.Status)
		b.WriteString(fmt.Sprintf("  %s %s: %s (%s)\n", icon, r.Name, r.Message, r.Duration.Truncate(time.Millisecond)))
		if r.Fix != "" {
			b.WriteString(fmt.Sprintf("    fix: %s\n", r.Fix))
		}
	}
	return b.String()
}

// FormatJSON renders results as a JSON array.
func FormatJSON(results []CheckResult) (string, error) {
	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return "", fmt.Errorf("doctor: marshal JSON: %w", err)
	}
	return string(data), nil
}

// HasFailures returns true if any check has status "fail".
func HasFailures(results []CheckResult) bool {
	for _, r := range results {
		if r.Status == statusFail {
			return true
		}
	}
	return false
}

func statusIcon(status string) string {
	switch status {
	case statusOK:
		return "ok"
	case statusWarn:
		return "!!"
	case statusFail:
		return "FAIL"
	default:
		return "??"
	}
}

func checkGoVersion(_ Config) CheckResult {
	start := time.Now()
	name := "go-version"

	out, err := exec.Command("go", "version").Output()
	if err != nil {
		return CheckResult{
			Name:     name,
			Status:   statusFail,
			Message:  "go not found in PATH",
			Duration: time.Since(start),
			Fix:      "install Go >= 1.22 from https://go.dev/dl/",
		}
	}

	version := string(out)
	major, minor, parseErr := parseGoVersion(version)
	if parseErr != nil {
		return CheckResult{
			Name:     name,
			Status:   statusWarn,
			Message:  fmt.Sprintf("could not parse version from: %s", strings.TrimSpace(version)),
			Duration: time.Since(start),
		}
	}

	if major < minGoMajor || (major == minGoMajor && minor < minGoMinor) {
		return CheckResult{
			Name:     name,
			Status:   statusFail,
			Message:  fmt.Sprintf("go %d.%d found, need >= %d.%d", major, minor, minGoMajor, minGoMinor),
			Duration: time.Since(start),
			Fix:      fmt.Sprintf("upgrade Go to >= %d.%d from https://go.dev/dl/", minGoMajor, minGoMinor),
		}
	}

	return CheckResult{
		Name:     name,
		Status:   statusOK,
		Message:  fmt.Sprintf("go %d.%d", major, minor),
		Duration: time.Since(start),
	}
}

func parseGoVersion(output string) (major, minor int, err error) {
	// "go version go1.22.4 darwin/arm64"
	fields := strings.Fields(output)
	for _, f := range fields {
		if strings.HasPrefix(f, "go") && strings.Contains(f, ".") {
			ver := strings.TrimPrefix(f, "go")
			parts := strings.SplitN(ver, ".", 3)
			if len(parts) < 2 {
				continue
			}
			maj, e1 := strconv.Atoi(parts[0])
			min, e2 := strconv.Atoi(parts[1])
			if e1 == nil && e2 == nil {
				return maj, min, nil
			}
		}
	}
	return 0, 0, fmt.Errorf("no version found in %q", output)
}

func checkConfigFile(cfg Config) CheckResult {
	start := time.Now()
	name := "config-file"

	path := cfg.ConfigPath
	if path == "" {
		home, _ := os.UserHomeDir()
		path = filepath.Join(home, ".mars-harness", "config.yaml")
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return CheckResult{
			Name:     name,
			Status:   statusWarn,
			Message:  fmt.Sprintf("%s not found", path),
			Duration: time.Since(start),
			Fix:      "run 'mars-harness setup' to create default configuration",
		}
	}

	return CheckResult{
		Name:     name,
		Status:   statusOK,
		Message:  path,
		Duration: time.Since(start),
	}
}

func checkModelsDir(cfg Config) CheckResult {
	start := time.Now()
	name := "models-dir"

	home, _ := os.UserHomeDir()
	modelsDir := filepath.Join(home, ".mars-harness", "models")

	info, err := os.Stat(modelsDir)
	if os.IsNotExist(err) {
		return CheckResult{
			Name:     name,
			Status:   statusWarn,
			Message:  fmt.Sprintf("%s not found", modelsDir),
			Duration: time.Since(start),
			Fix:      "run 'mars-harness setup' to create the models directory",
		}
	}
	if !info.IsDir() {
		return CheckResult{
			Name:     name,
			Status:   statusFail,
			Message:  fmt.Sprintf("%s exists but is not a directory", modelsDir),
			Duration: time.Since(start),
			Fix:      fmt.Sprintf("remove %s and run 'mars-harness setup'", modelsDir),
		}
	}

	entries, _ := os.ReadDir(modelsDir)
	ggufCount := 0
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".gguf") {
			ggufCount++
		}
	}

	if ggufCount == 0 {
		return CheckResult{
			Name:     name,
			Status:   statusWarn,
			Message:  fmt.Sprintf("%s exists but contains no .gguf files", modelsDir),
			Duration: time.Since(start),
			Fix:      "download models with 'mars-harness setup' or place .gguf files in " + modelsDir,
		}
	}

	return CheckResult{
		Name:     name,
		Status:   statusOK,
		Message:  fmt.Sprintf("%d model(s) in %s", ggufCount, modelsDir),
		Duration: time.Since(start),
	}
}

func checkProfileRequiredModels(cfg Config) CheckResult {
	start := time.Now()
	name := "profile-required-models"

	home, _ := os.UserHomeDir()
	modelsDir := filepath.Join(home, ".mars-harness", "models")
	performanceProfile := "auto"
	if cfgPath := strings.TrimSpace(cfg.ConfigPath); cfgPath != "" {
		if data, err := os.ReadFile(cfgPath); err == nil {
			for _, line := range strings.Split(string(data), "\n") {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "performance_profile:") {
					performanceProfile = strings.TrimSpace(strings.TrimPrefix(line, "performance_profile:"))
					performanceProfile = strings.Trim(performanceProfile, `"'`)
				}
			}
		}
	}

	missing, err := hardware.MissingRequiredModelFiles(modelsDir, performanceProfile)
	if err != nil {
		return CheckResult{
			Name:     name,
			Status:   statusFail,
			Message:  err.Error(),
			Duration: time.Since(start),
			Fix:      "run 'mars-harness setup' to download required model weights",
		}
	}
	if len(missing) > 0 {
		effective := hardware.EffectivePerformanceProfile(hardware.Detect(), performanceProfile)
		return CheckResult{
			Name:     name,
			Status:   statusFail,
			Message:  fmt.Sprintf("performance_profile %q requires missing file(s): %s", effective, strings.Join(missing, ", ")),
			Duration: time.Since(start),
			Fix:      "run 'mars-harness setup' to download the weights for your active profile",
		}
	}

	return CheckResult{
		Name:     name,
		Status:   statusOK,
		Message:  "required profile model files present",
		Duration: time.Since(start),
	}
}

func checkDBAccessible(cfg Config) CheckResult {
	start := time.Now()
	name := "database"

	path := cfg.DBPath
	if path == "" {
		home, _ := os.UserHomeDir()
		path = filepath.Join(home, ".mars-harness", "db", "mars.db")
	}

	dir := filepath.Dir(path)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return CheckResult{
			Name:     name,
			Status:   statusWarn,
			Message:  fmt.Sprintf("database directory %s not found", dir),
			Duration: time.Since(start),
			Fix:      "run 'mars-harness setup' to create the database directory",
		}
	}

	return CheckResult{
		Name:     name,
		Status:   statusOK,
		Message:  fmt.Sprintf("database directory exists: %s", dir),
		Duration: time.Since(start),
	}
}

func checkLlamaServer(_ Config) CheckResult {
	start := time.Now()
	name := "llama-server"

	path, err := exec.LookPath("llama-server")
	if err != nil {
		home, _ := os.UserHomeDir()
		binPath := filepath.Join(home, ".mars-harness", "bin", "llama-server")
		if _, statErr := os.Stat(binPath); statErr == nil {
			return CheckResult{
				Name:     name,
				Status:   statusOK,
				Message:  binPath,
				Duration: time.Since(start),
			}
		}
		return CheckResult{
			Name:     name,
			Status:   statusWarn,
			Message:  "llama-server not found in PATH or ~/.mars-harness/bin/",
			Duration: time.Since(start),
			Fix:      "install llama.cpp: https://github.com/ggml-org/llama.cpp#build",
		}
	}

	return CheckResult{
		Name:     name,
		Status:   statusOK,
		Message:  path,
		Duration: time.Since(start),
	}
}

func checkDiskSpace(_ Config) CheckResult {
	start := time.Now()
	name := "disk-space"

	home, _ := os.UserHomeDir()
	target := filepath.Join(home, ".mars-harness")
	if _, err := os.Stat(target); os.IsNotExist(err) {
		target = home
	}

	freeMiB, err := freeDiskMiB(target)
	if err != nil {
		return CheckResult{
			Name:     name,
			Status:   statusWarn,
			Message:  fmt.Sprintf("could not determine free disk space: %v", err),
			Duration: time.Since(start),
		}
	}

	if freeMiB < minDiskSpaceMiB {
		return CheckResult{
			Name:     name,
			Status:   statusFail,
			Message:  fmt.Sprintf("%d MiB free (need >= %d MiB)", freeMiB, minDiskSpaceMiB),
			Duration: time.Since(start),
			Fix:      "free up disk space — models alone require several GiB",
		}
	}

	return CheckResult{
		Name:     name,
		Status:   statusOK,
		Message:  fmt.Sprintf("%d MiB free", freeMiB),
		Duration: time.Since(start),
	}
}

func freeDiskMiB(path string) (int, error) {
	switch runtime.GOOS {
	case "linux", "darwin":
		return freeDiskUnix(path)
	default:
		return 0, fmt.Errorf("unsupported OS %q for disk space check", runtime.GOOS)
	}
}

func freeDiskUnix(path string) (int, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, err
	}
	freeBytes := uint64(stat.Bavail) * uint64(stat.Bsize)
	return int(freeBytes / (1024 * 1024)), nil
}
