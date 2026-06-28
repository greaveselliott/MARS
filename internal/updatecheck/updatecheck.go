/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/release-versioning.md
- docs/features/F-004-target-harness-lifecycle.md
*/
package updatecheck

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/greaveselliott/mars/internal/operatingmodel"
	"github.com/greaveselliott/mars/internal/scanner"
	"github.com/greaveselliott/mars/internal/selfupdate"
)

type Status string

const (
	StatusUpToDate Status = "up_to_date"
	StatusBehind   Status = "behind"
	StatusAhead    Status = "ahead"
	StatusUnknown  Status = "unknown"
)

// Config controls version drift checks for the installed tool and optional
// deployed target harness.
type Config struct {
	CurrentVersion   string
	RepoPath         string
	LatestReleaseURL string
	SkipRemote       bool
	HTTPClient       *http.Client
}

// Component reports version state for one updatable surface.
type Component struct {
	Name           string `json:"name"`
	Status         Status `json:"status"`
	CurrentVersion string `json:"current_version,omitempty"`
	LatestVersion  string `json:"latest_version,omitempty"`
	Command        string `json:"command,omitempty"`
	Message        string `json:"message"`
}

// Report is the complete update check result.
type Report struct {
	Tool    Component `json:"tool"`
	Harness Component `json:"harness"`
	Actions []string  `json:"actions"`
}

// Run checks whether the installed CLI and deployed target harness are current.
func Run(ctx context.Context, cfg Config) (Report, error) {
	if strings.TrimSpace(cfg.CurrentVersion) == "" {
		return Report{}, fmt.Errorf("update check: current version is required")
	}
	report := Report{
		Tool:    checkTool(ctx, cfg),
		Harness: checkHarness(cfg),
	}
	for _, component := range []Component{report.Tool, report.Harness} {
		if component.Command != "" && (component.Status == StatusBehind || component.Status == StatusUnknown) {
			report.Actions = append(report.Actions, component.Command)
		}
	}
	return report, nil
}

func checkTool(ctx context.Context, cfg Config) Component {
	current := selfupdate.NormalizeVersion(cfg.CurrentVersion)
	component := Component{
		Name:           "tool",
		Status:         StatusUnknown,
		CurrentVersion: current,
		Message:        "remote version check not run",
	}
	if cfg.SkipRemote {
		component.Message = "remote version check skipped"
		return component
	}

	latest, err := selfupdate.LatestRelease(ctx, cfg.HTTPClient, cfg.LatestReleaseURL)
	if err != nil {
		component.Message = err.Error()
		component.Command = "mars update tool"
		return component
	}
	component.LatestVersion = latest
	switch selfupdate.CompareVersions(current, latest) {
	case selfupdate.VersionEqual:
		component.Status = StatusUpToDate
		component.Message = "installed tool is current"
	case selfupdate.VersionBehind:
		component.Status = StatusBehind
		component.Command = fmt.Sprintf("mars update tool --version v%s", latest)
		component.Message = "installed tool is behind latest release"
	case selfupdate.VersionAhead:
		component.Status = StatusAhead
		component.Message = "installed tool is ahead of latest release"
	default:
		component.Status = StatusUnknown
		component.Command = "mars update tool"
		component.Message = "could not compare installed tool with latest release"
	}
	return component
}

func checkHarness(cfg Config) Component {
	repoPath := strings.TrimSpace(cfg.RepoPath)
	component := Component{
		Name:    "harness",
		Status:  StatusUnknown,
		Message: "target repo not supplied",
	}
	if repoPath == "" {
		return component
	}
	absRepo, err := filepath.Abs(repoPath)
	if err != nil {
		component.Message = fmt.Sprintf("resolve repo path: %v", err)
		return component
	}
	current := selfupdate.NormalizeVersion(cfg.CurrentVersion)
	component.CurrentVersion = current
	component.LatestVersion = current

	if operatingmodel.IsFoundationHarnessRepo(absRepo) {
		component.Status = StatusUpToDate
		component.Message = "foundation harness source repo uses AGENTS.md and docs instead of generated .harness metadata"
		return component
	}

	harnessDir := filepath.Join(absRepo, ".harness")
	if _, err := os.Stat(harnessDir); os.IsNotExist(err) {
		component.Message = ".harness is missing"
		component.Command = fmt.Sprintf("mars init --repo %s", quotePath(absRepo))
		return component
	} else if err != nil {
		component.Message = fmt.Sprintf("stat .harness: %v", err)
		return component
	}

	metadata, err := scanner.ReadHarnessMetadata(absRepo)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			component.Message = ".harness/metadata.yaml is missing"
			component.Command = fmt.Sprintf("mars update harness --repo %s", quotePath(absRepo))
			return component
		}
		component.Message = err.Error()
		component.Command = fmt.Sprintf("mars update harness --repo %s", quotePath(absRepo))
		return component
	}

	component.LatestVersion = current
	component.CurrentVersion = selfupdate.NormalizeVersion(metadata.GeneratorVersion)
	switch selfupdate.CompareVersions(component.CurrentVersion, current) {
	case selfupdate.VersionEqual:
		component.Status = StatusUpToDate
		component.Message = "target harness metadata matches installed tool"
	case selfupdate.VersionBehind:
		component.Status = StatusBehind
		component.Command = fmt.Sprintf("mars update harness --repo %s", quotePath(absRepo))
		component.Message = "target harness is behind installed tool"
	case selfupdate.VersionAhead:
		component.Status = StatusAhead
		component.Message = "target harness was generated by a newer mars"
	default:
		component.Status = StatusUnknown
		component.Command = fmt.Sprintf("mars update harness --repo %s", quotePath(absRepo))
		component.Message = "could not compare target harness metadata with installed tool"
	}
	if drift, driftErr := operatingmodel.CheckRepo(absRepo); driftErr != nil {
		component.Status = StatusUnknown
		component.Command = fmt.Sprintf("mars update harness --repo %s", quotePath(absRepo))
		component.Message = driftErr.Error()
	} else if !drift.OK() {
		component.Status = StatusBehind
		component.Command = fmt.Sprintf("mars update harness --repo %s", quotePath(absRepo))
		component.Message = drift.Summary() + "; update harness writes missing defaults and leaves user-owned stale files for migration tickets"
	}
	return component
}

func quotePath(path string) string {
	if strings.ContainsAny(path, " \t\n\"'\\") {
		return strconv.Quote(path)
	}
	return path
}
