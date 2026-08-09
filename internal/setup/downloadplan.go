/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/local-inference.md
- docs/features/F-002-zero-config-shell-path.md
- docs/features/F-003-local-inference-lifecycle.md
- docs/features/F-009-release-update-lifecycle.md
*/
package setup

import (
	"encoding/hex"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/greaveselliott/mars/internal/hardware"
	"github.com/greaveselliott/mars/internal/models"
)

const (
	downloadArtifactLlama = "llama.cpp"
	downloadArtifactModel = "model"
)

// DownloadArtifact is one immutable third-party artifact pending setup.
type DownloadArtifact struct {
	Kind              string   `json:"kind"`
	Identity          string   `json:"identity"`
	SizeBytes         int64    `json:"size_bytes"`
	LicenseID         string   `json:"license_id"`
	LicenseURL        string   `json:"license_url"`
	TermsOrNoticeURLs []string `json:"terms_or_notice_urls"`
}

// DownloadPlan is the deterministic set of missing third-party artifacts that
// setup would download after acknowledgement.
type DownloadPlan struct {
	LocalBundle string             `json:"local_bundle,omitempty"`
	Artifacts   []DownloadArtifact `json:"artifacts"`
	TotalBytes  int64              `json:"total_bytes"`
}

// Empty reports whether setup has any third-party downloads pending.
func (p DownloadPlan) Empty() bool {
	return len(p.Artifacts) == 0
}

func (p DownloadPlan) allows(kind, identity string) bool {
	for _, artifact := range p.Artifacts {
		if artifact.Kind == kind && artifact.Identity == identity {
			return true
		}
	}
	return false
}

func (p DownloadPlan) hasKind(kind string) bool {
	for _, artifact := range p.Artifacts {
		if artifact.Kind == kind {
			return true
		}
	}
	return false
}

func (p DownloadPlan) equal(other DownloadPlan) bool {
	if p.LocalBundle != other.LocalBundle || p.TotalBytes != other.TotalBytes || len(p.Artifacts) != len(other.Artifacts) {
		return false
	}
	for i := range p.Artifacts {
		left, right := p.Artifacts[i], other.Artifacts[i]
		if left.Kind != right.Kind || left.Identity != right.Identity || left.SizeBytes != right.SizeBytes || left.LicenseID != right.LicenseID || left.LicenseURL != right.LicenseURL || len(left.TermsOrNoticeURLs) != len(right.TermsOrNoticeURLs) {
			return false
		}
		for j := range left.TermsOrNoticeURLs {
			if left.TermsOrNoticeURLs[j] != right.TermsOrNoticeURLs[j] {
				return false
			}
		}
	}
	return true
}

// ResolveDownloadPlan resolves and validates every pending local-inference
// artifact without creating directories, writing files, or making requests.
func ResolveDownloadPlan(cfg Config) (DownloadPlan, error) {
	if !setupDownloadsEnabled(cfg) {
		return emptyDownloadPlan(), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return DownloadPlan{}, fmt.Errorf("setup: cannot determine home directory: %w — set $HOME and retry", err)
	}
	return resolveDownloadPlan(filepath.Join(home, ".mars"), cfg, hardware.Detect(), "")
}

func resolveDownloadPlan(baseDir string, cfg Config, hw hardware.Summary, platform string) (DownloadPlan, error) {
	if !setupDownloadsEnabled(cfg) {
		return emptyDownloadPlan(), nil
	}

	artifacts := make([]DownloadArtifact, 0)
	binaryPath := filepath.Join(baseDir, "bin", "llama-server")
	llamaPresent, err := observeLlamaServer(binaryPath)
	if err != nil {
		return DownloadPlan{}, err
	}
	if !llamaPresent {
		if platform == "" {
			platform, err = llamaPlatformKey()
			if err != nil {
				return DownloadPlan{}, err
			}
		}
		artifact, err := plannedLlamaDownload(platform)
		if err != nil {
			return DownloadPlan{}, err
		}
		artifacts = append(artifacts, artifact)
	}

	bundle, _, err := models.ResolveLocalBundle(hw, cfg.LocalBundle)
	if err != nil {
		return DownloadPlan{}, err
	}
	modelArtifacts, err := plannedModelDownloads(filepath.Join(baseDir, "models"), hardware.UniqueModels(bundle.Models))
	if err != nil {
		return DownloadPlan{}, err
	}
	artifacts = append(artifacts, modelArtifacts...)

	sort.Slice(artifacts, func(i, j int) bool {
		return artifacts[i].Identity < artifacts[j].Identity
	})
	plan := DownloadPlan{LocalBundle: bundle.ID, Artifacts: artifacts}
	seen := make(map[string]struct{}, len(artifacts))
	for _, artifact := range artifacts {
		if err := validatePlannedDownload(artifact); err != nil {
			return DownloadPlan{}, err
		}
		if _, ok := seen[artifact.Identity]; ok {
			return DownloadPlan{}, fmt.Errorf("setup: duplicate planned download %q — update the pinned artifact registry", artifact.Identity)
		}
		seen[artifact.Identity] = struct{}{}
		plan.TotalBytes += artifact.SizeBytes
	}
	return plan, nil
}

func setupDownloadsEnabled(cfg Config) bool {
	if cfg.SkipDownload || cfg.TestMode {
		return false
	}
	mode := strings.ToLower(strings.TrimSpace(cfg.Inference))
	return mode == "" || mode == models.RoutingLocal
}

func emptyDownloadPlan() DownloadPlan {
	return DownloadPlan{Artifacts: []DownloadArtifact{}}
}

func plannedLlamaDownload(platform string) (DownloadArtifact, error) {
	artifact, err := enabledLlamaCppArtifact(platform)
	if err != nil {
		return DownloadArtifact{}, err
	}
	return plannedLlamaDownloadForRelease(pinnedLlamaCppRelease, artifact)
}

func plannedLlamaDownloadForRelease(release llamaCppRelease, artifact llamaCppPlatformArtifact) (DownloadArtifact, error) {
	if err := validateLlamaCppReleaseProvenance(release, artifact); err != nil {
		return DownloadArtifact{}, fmt.Errorf("setup: llama.cpp download has incomplete provenance: %w — use --skip-download and install llama-server independently", err)
	}
	noticeURLs := make([]string, 0, len(release.Notices))
	for _, notice := range release.Notices {
		noticeURLs = append(noticeURLs, notice.URL)
	}
	sort.Strings(noticeURLs)
	planned := DownloadArtifact{
		Kind:              downloadArtifactLlama,
		Identity:          llamaDownloadIdentity(release, artifact),
		SizeBytes:         artifact.SizeBytes,
		LicenseID:         release.LicenseID,
		LicenseURL:        release.License.URL,
		TermsOrNoticeURLs: noticeURLs,
	}
	if err := validatePlannedDownload(planned); err != nil {
		return DownloadArtifact{}, fmt.Errorf("setup: llama.cpp download has incomplete provenance: %w — use --skip-download and install llama-server independently", err)
	}
	return planned, nil
}

func llamaDownloadIdentity(release llamaCppRelease, artifact llamaCppPlatformArtifact) string {
	return fmt.Sprintf("github.com/ggml-org/llama.cpp@%s/%s", release.SourceCommit, artifact.ArchiveName)
}

func observeLlamaServer(binaryPath string) (bool, error) {
	_, err := os.Lstat(binaryPath)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, fmt.Errorf("setup: cannot inspect existing llama-server — check the bin directory permissions")
}

func validateLlamaCppReleaseProvenance(release llamaCppRelease, artifact llamaCppPlatformArtifact) error {
	if !isPinnedLlamaTag(release.Tag) || !isLowerHexString(release.SourceCommit, 40) {
		return fmt.Errorf("a pinned tag and full source commit are required")
	}
	if !strings.Contains(artifact.ArchiveName, "-"+release.Tag+"-") || artifact.SizeBytes <= 0 || !isLowerHexString(artifact.SHA256, 64) {
		return fmt.Errorf("an archive name, positive byte size, and exact SHA256 are required")
	}
	if strings.TrimSpace(release.LicenseID) == "" || strings.TrimSpace(release.License.Name) == "" || !isHTTPSURL(release.License.URL) || !isLowerHexString(release.License.SHA256, 64) {
		return fmt.Errorf("a license ID and hashed license record are required")
	}
	if len(release.Notices) == 0 {
		return fmt.Errorf("at least one hashed notice record is required")
	}
	for _, notice := range release.Notices {
		if strings.TrimSpace(notice.Name) == "" || !isHTTPSURL(notice.URL) || !isLowerHexString(notice.SHA256, 64) {
			return fmt.Errorf("each notice requires a name, HTTPS URL, and exact SHA256")
		}
	}
	return nil
}

func plannedModelDownloads(modelsDir string, specs []hardware.ModelSpec) ([]DownloadArtifact, error) {
	if err := validateDownloadModelProvenance(specs); err != nil {
		return nil, err
	}
	artifacts := make([]DownloadArtifact, 0, len(specs))
	for _, spec := range specs {
		destPath := filepath.Join(modelsDir, spec.File)
		if _, err := os.Stat(destPath); err == nil {
			continue
		} else if !os.IsNotExist(err) {
			return nil, fmt.Errorf("setup: cannot inspect pending model artifact %q — check models directory permissions", spec.File)
		}
		artifacts = append(artifacts, DownloadArtifact{
			Kind:              downloadArtifactModel,
			Identity:          modelDownloadIdentity(spec),
			SizeBytes:         spec.SizeBytes,
			LicenseID:         spec.Provenance.LicenseID,
			LicenseURL:        spec.Provenance.LicenseURL,
			TermsOrNoticeURLs: modelTermsOrNoticeURLs(spec),
		})
	}
	return artifacts, nil
}

func modelDownloadIdentity(spec hardware.ModelSpec) string {
	return fmt.Sprintf("huggingface.co/%s@%s/%s", spec.Repo, spec.Revision, spec.File)
}

func modelTermsOrNoticeURLs(spec hardware.ModelSpec) []string {
	urls := []string{spec.Provenance.TermsURL, spec.Provenance.QuantizationToolLicenseURL}
	sort.Strings(urls)
	if urls[0] == urls[1] {
		return urls[:1]
	}
	return urls
}

func pendingPlannedModels(modelsDir string, specs []hardware.ModelSpec, plan DownloadPlan) ([]hardware.ModelSpec, error) {
	pending := make([]hardware.ModelSpec, 0, len(specs))
	for _, spec := range specs {
		if _, err := os.Stat(filepath.Join(modelsDir, spec.File)); err == nil {
			continue
		} else if !os.IsNotExist(err) {
			return nil, fmt.Errorf("setup: cannot inspect pending model artifact %q — check models directory permissions", spec.File)
		}
		if !plan.allows(downloadArtifactModel, modelDownloadIdentity(spec)) {
			return nil, fmt.Errorf("setup: pending model artifacts changed after acknowledgement — review the new plan and rerun `mars setup --download --yes`")
		}
		pending = append(pending, spec)
	}
	return pending, nil
}

func validatePlannedDownload(artifact DownloadArtifact) error {
	if strings.TrimSpace(artifact.Kind) == "" || strings.TrimSpace(artifact.Identity) == "" || artifact.SizeBytes <= 0 {
		return fmt.Errorf("artifact identity, kind, and positive byte size are required")
	}
	if strings.TrimSpace(artifact.LicenseID) == "" || !isHTTPSURL(artifact.LicenseURL) {
		return fmt.Errorf("artifact %s requires a license ID and HTTPS license URL", artifact.Identity)
	}
	if len(artifact.TermsOrNoticeURLs) == 0 {
		return fmt.Errorf("artifact %s requires at least one terms or notice URL", artifact.Identity)
	}
	for _, noticeURL := range artifact.TermsOrNoticeURLs {
		if !isHTTPSURL(noticeURL) {
			return fmt.Errorf("artifact %s requires HTTPS terms or notice URLs", artifact.Identity)
		}
	}
	return nil
}

func isHTTPSURL(value string) bool {
	parsed, err := url.Parse(strings.TrimSpace(value))
	return err == nil && parsed.Scheme == "https" && parsed.Host != ""
}

func isLowerHexString(value string, length int) bool {
	if len(value) != length || value != strings.ToLower(value) {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func isPinnedLlamaTag(value string) bool {
	if len(value) < 2 || value[0] != 'b' {
		return false
	}
	for _, char := range value[1:] {
		if char < '0' || char > '9' {
			return false
		}
	}
	return true
}
