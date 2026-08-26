/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/release-versioning.md
- docs/features/F-009-release-update-lifecycle.md
- docs/features/F-018-goreleaser-distribution.md
*/
package release

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

type syftTestConfig struct {
	CheckForAppUpdate bool     `yaml:"check-for-app-update"`
	Enrich            []string `yaml:"enrich"`
	Cache             struct {
		Dir string `yaml:"dir"`
		TTL string `yaml:"ttl"`
	} `yaml:"cache"`
	CPP struct {
		AllowVCPKGGitClone bool `yaml:"vcpkg-allow-git-clone"`
	} `yaml:"cpp"`
	Golang struct {
		SearchLocalModuleCacheLicenses bool `yaml:"search-local-mod-cache-licenses"`
		SearchLocalVendorLicenses      bool `yaml:"search-local-vendor-licenses"`
		SearchRemoteLicenses           bool `yaml:"search-remote-licenses"`
		UsePackagesLib                 bool `yaml:"use-packages-lib"`
	} `yaml:"golang"`
	Java struct {
		UseNetwork bool `yaml:"use-network"`
	} `yaml:"java"`
	JavaScript struct {
		SearchRemoteLicenses bool `yaml:"search-remote-licenses"`
	} `yaml:"javascript"`
	Python struct {
		SearchRemoteLicenses bool `yaml:"search-remote-licenses"`
	} `yaml:"python"`
}

type workflowContract struct {
	Name        string                 `yaml:"name"`
	On          map[string]any         `yaml:"on"`
	Permissions map[string]string      `yaml:"permissions"`
	Concurrency map[string]any         `yaml:"concurrency"`
	Jobs        map[string]workflowJob `yaml:"jobs"`
}

type workflowJob struct {
	Name           string            `yaml:"name"`
	If             string            `yaml:"if"`
	Needs          string            `yaml:"needs"`
	RunsOn         string            `yaml:"runs-on"`
	TimeoutMinutes int               `yaml:"timeout-minutes"`
	Permissions    map[string]string `yaml:"permissions"`
	Strategy       workflowStrategy  `yaml:"strategy"`
	Steps          []workflowStep    `yaml:"steps"`
}

type workflowStrategy struct {
	FailFast bool                `yaml:"fail-fast"`
	Matrix   map[string][]string `yaml:"matrix"`
}

type workflowStep struct {
	Name string            `yaml:"name"`
	ID   string            `yaml:"id"`
	Uses string            `yaml:"uses"`
	If   string            `yaml:"if"`
	With map[string]any    `yaml:"with"`
	Env  map[string]string `yaml:"env"`
	Run  string            `yaml:"run"`
}

func TestReleaseWorkflowIsActiveLeastPrivilegeAndConventional(t *testing.T) {
	t.Parallel()
	root := releaseRepoRoot(t)
	require.NoFileExists(t, filepath.Join(root, ".goreleaser.yaml"))
	require.NoFileExists(t, filepath.Join(root, ".github", "workflows", "release-snapshot.yml"))

	path := filepath.Join(root, ".github", "workflows", "release.yml")
	raw := readStrictYAML(t, path, new(workflowContract))
	var workflow workflowContract
	readStrictYAMLInto(t, raw, &workflow)

	require.Equal(t, "release", workflow.Name)
	require.Equal(t, []string{"push"}, sortedKeys(workflow.On))
	push, ok := workflow.On["push"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, []any{"v*.*.*"}, push["tags"])
	require.Empty(t, workflow.Permissions)
	require.Equal(t, []string{"attest", "produce", "publish", "verify"}, sortedKeys(workflow.Jobs))

	expected := map[string]struct {
		needs       string
		timeout     int
		permissions map[string]string
	}{
		"produce": {timeout: 45, permissions: map[string]string{"contents": "read"}},
		"attest":  {needs: "produce", timeout: 15, permissions: map[string]string{"actions": "read", "artifact-metadata": "write", "attestations": "write", "contents": "read", "id-token": "write"}},
		"verify":  {needs: "attest", timeout: 30, permissions: map[string]string{"actions": "read", "contents": "read"}},
		"publish": {needs: "verify", timeout: 15, permissions: map[string]string{"actions": "read", "contents": "write"}},
	}
	expectedIf := "${{ github.repository == 'greaveselliott/MARS' && github.event_name == 'push' && github.ref_type == 'tag' }}"
	for name, want := range expected {
		job := workflow.Jobs[name]
		require.Equal(t, expectedIf, job.If, name)
		require.Equal(t, want.needs, job.Needs, name)
		require.Equal(t, "ubuntu-24.04", job.RunsOn, name)
		require.Equal(t, want.timeout, job.TimeoutMinutes, name)
		require.Equal(t, want.permissions, job.Permissions, name)
		require.NotEmpty(t, job.Steps, name)
	}

	fullSHA := regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_./-]+@[0-9a-f]{40}$`)
	for _, job := range workflow.Jobs {
		for _, step := range job.Steps {
			if step.Uses != "" {
				require.Regexp(t, fullSHA, step.Uses)
			}
		}
	}

	text := string(raw)
	for _, required := range []string{
		"go-version: 1.27.0",
		"github.com/anchore/syft/cmd/syft@v1.51.0",
		"bash scripts/release-produce.sh",
		"subject-checksums: dist/checksums.txt",
		"checksums.txt.sigstore.json",
		"TestVerifyReleaseDistFromEnvironment",
		"TestVerifyMARSReleaseAttestationFromEnvironment",
		"gh release create \"$tag\" --draft --verify-tag",
		"gh release edit \"$tag\" --draft=false --latest",
	} {
		require.Contains(t, text, required)
	}
	for _, forbidden := range []string{
		"pull_request_target", "workflow_dispatch:", "secrets.", "${{ secrets",
		"goreleaser release", "goreleaser-action", "cosign", "docker", "continue-on-error", "environment:",
	} {
		require.NotContains(t, strings.ToLower(text), strings.ToLower(forbidden))
	}
	require.Less(t, strings.Index(text, "gh release create"), strings.Index(text, "gh release upload"))
	require.Less(t, strings.Index(text, "gh release upload"), strings.Index(text, "gh release edit"))
	bundleAside := strings.Index(text, `mv -- "$GITHUB_WORKSPACE/dist/checksums.txt.sigstore.json" "$bundle"`)
	unsignedVerify := strings.LastIndex(text, "TestVerifyReleaseDistFromEnvironment")
	bundleRestore := strings.Index(text, `mv -- "$bundle" "$GITHUB_WORKSPACE/dist/checksums.txt.sigstore.json"`)
	attestationVerify := strings.Index(text, "TestVerifyMARSReleaseAttestationFromEnvironment")
	require.NotEqual(t, -1, bundleAside)
	require.NotEqual(t, -1, unsignedVerify)
	require.NotEqual(t, -1, bundleRestore)
	require.NotEqual(t, -1, attestationVerify)
	require.Less(t, bundleAside, unsignedVerify)
	require.Less(t, unsignedVerify, bundleRestore)
	require.Less(t, bundleRestore, attestationVerify)
}

func TestConventionalReleaseProducerContract(t *testing.T) {
	t.Parallel()
	raw, err := os.ReadFile(filepath.Join(releaseRepoRoot(t), "scripts", "release-produce.sh"))
	require.NoError(t, err)
	text := string(raw)

	for _, required := range []string{
		"go1.27.0", "github.com/anchore/syft", "v1.51.0", "CGO_ENABLED=0",
		"GOOS=\"$goos\"", "GOARCH=\"$goarch\"", "-trimpath", "-buildvcs=true",
		"darwin-amd64 darwin-arm64 linux-amd64 linux-arm64",
		"LICENSE", "NOTICE", "THIRD_PARTY_NOTICES", "mars",
		"--format=ustar", "--owner=root", "--group=root", "gzip -n",
		"date -u -d", "spdx-json", "sha256sum", "sort -k2,2", "checksums.txt",
		"COSIGN_PASSWORD", "COSIGN_PRIVATE_KEY",
	} {
		require.Contains(t, text, required)
	}
	for _, forbidden := range []string{"goreleaser release", "goreleaser-action", "docker", "ptrace", "landlock"} {
		require.NotContains(t, strings.ToLower(text), forbidden)
	}
}

func TestSyftConfigDisablesNetworkCapableBehavior(t *testing.T) {
	t.Parallel()
	path := filepath.Join(releaseRepoRoot(t), ".syft.yaml")
	raw := readStrictYAML(t, path, new(syftTestConfig))
	var cfg syftTestConfig
	readStrictYAMLInto(t, raw, &cfg)

	require.False(t, cfg.CheckForAppUpdate)
	require.Empty(t, cfg.Enrich)
	require.Empty(t, cfg.Cache.Dir)
	require.Equal(t, "0s", cfg.Cache.TTL)
	require.False(t, cfg.CPP.AllowVCPKGGitClone)
	require.False(t, cfg.Golang.SearchLocalModuleCacheLicenses)
	require.False(t, cfg.Golang.SearchLocalVendorLicenses)
	require.False(t, cfg.Golang.SearchRemoteLicenses)
	require.False(t, cfg.Golang.UsePackagesLib)
	require.False(t, cfg.Java.UseNetwork)
	require.False(t, cfg.JavaScript.SearchRemoteLicenses)
	require.False(t, cfg.Python.SearchRemoteLicenses)
}

func TestSourceCompatibilityWorkflowContract(t *testing.T) {
	t.Parallel()
	path := filepath.Join(releaseRepoRoot(t), ".github", "workflows", "source-compatibility.yml")
	raw := readStrictYAML(t, path, new(workflowContract))
	var workflow workflowContract
	readStrictYAMLInto(t, raw, &workflow)

	require.Equal(t, "source-compatibility", workflow.Name)
	require.Equal(t, []string{"pull_request", "push", "workflow_dispatch"}, sortedKeys(workflow.On))
	require.Equal(t, map[string]string{"contents": "read"}, workflow.Permissions)
	require.Equal(t, []string{"below-minimum", "dependency-notices", "supported-source"}, sortedKeys(workflow.Jobs))

	notices := workflow.Jobs["dependency-notices"]
	require.Equal(t, map[string]any{"go-version": "1.27.0", "cache": false}, notices.Steps[1].With)

	supported := workflow.Jobs["supported-source"]
	require.Equal(t, map[string][]string{"go-version": {"1.25.13", "1.27.0"}}, supported.Strategy.Matrix)
	for _, command := range []string{
		"go mod tidy -go=1.25.13", "CGO_ENABLED=0 go build ./cmd/mars",
		"go test ./...", "go vet ./...",
		"go install golang.org/x/vuln/cmd/govulncheck@v1.7.0",
	} {
		require.Contains(t, supported.Steps[2].Run, command)
	}

	below := workflow.Jobs["below-minimum"]
	require.Equal(t, map[string]any{"go-version": "1.25.12", "cache": false}, below.Steps[1].With)
	require.Contains(t, below.Steps[2].Run, "go.mod requires go >= 1.25.13")
}

func readStrictYAML[T any](t *testing.T, path string, target *T) []byte {
	t.Helper()
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	readStrictYAMLInto(t, raw, target)
	return raw
}

func readStrictYAMLInto(t *testing.T, raw []byte, target any) {
	t.Helper()
	var document yaml.Node
	require.NoError(t, yaml.Unmarshal(raw, &document))
	rejectYAMLAliasesAndDuplicateKeys(t, &document)
	decoder := yaml.NewDecoder(bytes.NewReader(raw))
	decoder.KnownFields(true)
	require.NoError(t, decoder.Decode(target))
}

func rejectYAMLAliasesAndDuplicateKeys(t *testing.T, node *yaml.Node) {
	t.Helper()
	require.NotEqual(t, yaml.AliasNode, node.Kind, "YAML aliases are not allowed in release contracts")
	if node.Kind == yaml.MappingNode {
		seen := make(map[string]struct{}, len(node.Content)/2)
		for i := 0; i < len(node.Content); i += 2 {
			key := node.Content[i].Value
			require.NotEqual(t, "<<", key, "YAML merge keys are not allowed in release contracts")
			_, duplicate := seen[key]
			require.False(t, duplicate, "duplicate YAML key %q", key)
			seen[key] = struct{}{}
		}
	}
	for _, child := range node.Content {
		rejectYAMLAliasesAndDuplicateKeys(t, child)
	}
}

func sortedKeys[V any](values map[string]V) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func releaseRepoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}
