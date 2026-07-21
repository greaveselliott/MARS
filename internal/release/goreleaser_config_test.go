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

type goreleaserTestConfig struct {
	Version     int                     `yaml:"version"`
	ProjectName string                  `yaml:"project_name"`
	Dist        string                  `yaml:"dist"`
	Builds      []goreleaserTestBuild   `yaml:"builds"`
	Archives    []goreleaserTestArchive `yaml:"archives"`
	Snapshot    goreleaserTestSnapshot  `yaml:"snapshot"`
	SBOMs       []goreleaserTestSBOM    `yaml:"sboms"`
	Checksum    goreleaserTestChecksum  `yaml:"checksum"`
	Changelog   struct {
		Disable bool `yaml:"disable"`
	} `yaml:"changelog"`
	Release struct {
		Disable bool `yaml:"disable"`
	} `yaml:"release"`
}

type goreleaserTestBuild struct {
	ID           string   `yaml:"id"`
	Main         string   `yaml:"main"`
	Binary       string   `yaml:"binary"`
	Env          []string `yaml:"env"`
	Flags        []string `yaml:"flags"`
	LDFlags      []string `yaml:"ldflags"`
	GOOS         []string `yaml:"goos"`
	GOARCH       []string `yaml:"goarch"`
	GOAMD64      []string `yaml:"goamd64"`
	GOARM64      []string `yaml:"goarm64"`
	ModTimestamp string   `yaml:"mod_timestamp"`
}

type goreleaserTestFileInfo struct {
	Owner string `yaml:"owner"`
	Group string `yaml:"group"`
	Mode  int    `yaml:"mode"`
	MTime string `yaml:"mtime"`
}

type goreleaserTestArchiveFile struct {
	Src  string                 `yaml:"src"`
	Dst  string                 `yaml:"dst"`
	Info goreleaserTestFileInfo `yaml:"info"`
}

type goreleaserTestArchive struct {
	ID                   string                      `yaml:"id"`
	IDs                  []string                    `yaml:"ids"`
	Formats              []string                    `yaml:"formats"`
	NameTemplate         string                      `yaml:"name_template"`
	WrapInDirectory      bool                        `yaml:"wrap_in_directory"`
	StripBinaryDirectory bool                        `yaml:"strip_binary_directory"`
	BuildsInfo           goreleaserTestFileInfo      `yaml:"builds_info"`
	Files                []goreleaserTestArchiveFile `yaml:"files"`
}

type goreleaserTestSnapshot struct {
	VersionTemplate string `yaml:"version_template"`
}

type goreleaserTestSBOM struct {
	ID        string   `yaml:"id"`
	Artifacts string   `yaml:"artifacts"`
	IDs       []string `yaml:"ids"`
	Documents []string `yaml:"documents"`
	Cmd       string   `yaml:"cmd"`
	Args      []string `yaml:"args"`
	Env       []string `yaml:"env"`
}

type goreleaserTestChecksum struct {
	NameTemplate string   `yaml:"name_template"`
	Algorithm    string   `yaml:"algorithm"`
	IDs          []string `yaml:"ids"`
	ExtraFiles   []struct {
		Glob string `yaml:"glob"`
	} `yaml:"extra_files"`
}

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

type snapshotWorkflow struct {
	Name        string                 `yaml:"name"`
	On          map[string]any         `yaml:"on"`
	Permissions map[string]string      `yaml:"permissions"`
	Jobs        map[string]workflowJob `yaml:"jobs"`
}

type workflowJob struct {
	RunsOn         string         `yaml:"runs-on"`
	TimeoutMinutes int            `yaml:"timeout-minutes"`
	Steps          []workflowStep `yaml:"steps"`
}

type workflowStep struct {
	Name string            `yaml:"name"`
	Uses string            `yaml:"uses"`
	With map[string]any    `yaml:"with"`
	Env  map[string]string `yaml:"env"`
	Run  string            `yaml:"run"`
}

func TestGoReleaserConfigContract(t *testing.T) {
	t.Parallel()
	path := filepath.Join(releaseRepoRoot(t), ".goreleaser.yaml")
	raw := readStrictYAML(t, path, new(goreleaserTestConfig))
	var cfg goreleaserTestConfig
	readStrictYAMLInto(t, raw, &cfg)

	require.Equal(t, 2, cfg.Version)
	require.Equal(t, "mars", cfg.ProjectName)
	require.Equal(t, "dist", cfg.Dist)
	require.Len(t, cfg.Builds, 1)
	build := cfg.Builds[0]
	require.Equal(t, "mars", build.ID)
	require.Equal(t, "./cmd/mars", build.Main)
	require.Equal(t, "mars", build.Binary)
	require.Equal(t, []string{"CGO_ENABLED=0"}, build.Env)
	require.Equal(t, []string{"-trimpath"}, build.Flags)
	require.Equal(t, []string{"darwin", "linux"}, build.GOOS)
	require.Equal(t, []string{"amd64", "arm64"}, build.GOARCH)
	require.Equal(t, []string{"v1"}, build.GOAMD64)
	require.Equal(t, []string{"v8.0"}, build.GOARM64)
	require.Equal(t, "{{ .CommitTimestamp }}", build.ModTimestamp)
	require.Len(t, build.LDFlags, 1)
	require.Equal(t,
		"-s -w -X main.version={{ .Version }} -X main.commit={{ .FullCommit }} -X main.date={{ .CommitDate }}",
		strings.Join(strings.Fields(build.LDFlags[0]), " "))

	require.Len(t, cfg.Archives, 1)
	archive := cfg.Archives[0]
	require.Equal(t, "mars-archives", archive.ID)
	require.Equal(t, []string{"mars"}, archive.IDs)
	require.Equal(t, []string{"tar.gz"}, archive.Formats)
	require.Equal(t, "mars_{{ .Version }}_{{ .Os }}_{{ .Arch }}", archive.NameTemplate)
	require.False(t, archive.WrapInDirectory)
	require.True(t, archive.StripBinaryDirectory)
	require.Equal(t, goreleaserTestFileInfo{Owner: "root", Group: "root", Mode: 0o755, MTime: "{{ .CommitDate }}"}, archive.BuildsInfo)
	require.Equal(t, []string{"LICENSE", "NOTICE", "THIRD_PARTY_NOTICES"}, archiveSources(archive.Files))
	for _, file := range archive.Files {
		require.Equal(t, file.Src, file.Dst)
		require.Equal(t, goreleaserTestFileInfo{Owner: "root", Group: "root", Mode: 0o644, MTime: "{{ .CommitDate }}"}, file.Info)
	}

	require.Equal(t, "0.69.0-dev.{{ .ShortCommit }}", cfg.Snapshot.VersionTemplate)
	require.Len(t, cfg.SBOMs, 1)
	sbom := cfg.SBOMs[0]
	require.Equal(t, "mars-sboms", sbom.ID)
	require.Equal(t, "archive", sbom.Artifacts)
	require.Equal(t, []string{"mars-archives"}, sbom.IDs)
	require.Equal(t, []string{"${artifact}.sbom.json"}, sbom.Documents)
	require.Equal(t, "syft", sbom.Cmd)
	require.Equal(t, []string{"file:${artifact}", "--config", "../.syft.yaml", "--quiet", "--output", "spdx-json=${document}"}, sbom.Args)
	require.Equal(t, []string{"SYFT_CHECK_FOR_APP_UPDATE=false", "SYFT_ENRICH="}, sbom.Env)

	require.Equal(t, "checksums.txt", cfg.Checksum.NameTemplate)
	require.Equal(t, "sha256", cfg.Checksum.Algorithm)
	require.Equal(t, []string{"mars-archives"}, cfg.Checksum.IDs)
	require.Len(t, cfg.Checksum.ExtraFiles, 1)
	require.Equal(t, "./dist/*.sbom.json", cfg.Checksum.ExtraFiles[0].Glob)
	require.True(t, cfg.Changelog.Disable)
	require.True(t, cfg.Release.Disable)

	text := string(raw)
	for _, forbidden := range []string{
		"{{ .Date }}", "{{ .Now }}", "{{ .Timestamp }}", "{{ .Tag }}",
		"\nsigns:", "\nbinary_signs:", "\npublishers:", "\nuploads:",
		"\nannounce:", "\nbefore:", "\nhooks:", "\nkos:", "\nsource:", "--enrich",
	} {
		require.NotContains(t, text, forbidden)
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

func TestSnapshotWorkflowContract(t *testing.T) {
	t.Parallel()
	path := filepath.Join(releaseRepoRoot(t), ".github", "workflows", "release-snapshot.yml")
	raw := readStrictYAML(t, path, new(snapshotWorkflow))
	var workflow snapshotWorkflow
	readStrictYAMLInto(t, raw, &workflow)

	require.Equal(t, "release-snapshot", workflow.Name)
	require.Equal(t, []string{"pull_request", "workflow_dispatch"}, sortedKeys(workflow.On))
	require.Equal(t, map[string]string{"contents": "read"}, workflow.Permissions)
	require.Equal(t, []string{"snapshot"}, sortedKeys(workflow.Jobs))
	job := workflow.Jobs["snapshot"]
	require.Equal(t, "ubuntu-24.04", job.RunsOn)
	require.Equal(t, 30, job.TimeoutMinutes)
	require.Len(t, job.Steps, 6)

	require.Equal(t, "actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1", job.Steps[0].Uses)
	require.Equal(t, map[string]any{"fetch-depth": 0, "persist-credentials": false}, job.Steps[0].With)
	require.Equal(t, "actions/setup-go@b7ad1dad31e06c5925ef5d2fc7ad053ef454303e", job.Steps[1].Uses)
	require.Equal(t, map[string]any{"go-version": "1.26.5", "cache": false}, job.Steps[1].With)

	toolStep := job.Steps[2]
	require.Empty(t, toolStep.Uses)
	require.Equal(t, map[string]string{
		"GOBIN": "${{ runner.temp }}", "GONOSUMDB": "", "GOPRIVATE": "",
		"GOPROXY": "https://proxy.golang.org", "GOSUMDB": "sum.golang.org", "GOTOOLCHAIN": "local",
	}, toolStep.Env)
	for _, command := range []string{
		"go install github.com/goreleaser/goreleaser/v2@v2.17.0",
		"go install github.com/anchore/syft/cmd/syft@v1.49.0",
		`"$RUNNER_TEMP/goreleaser" --version`,
		`"$RUNNER_TEMP/syft" version`,
		`go version -m "$RUNNER_TEMP/goreleaser" | grep -F $'\tmod\tgithub.com/goreleaser/goreleaser/v2\tv2.17.0\t'`,
		`go version -m "$RUNNER_TEMP/goreleaser" | grep -F ': go1.26.5'`,
		`go version -m "$RUNNER_TEMP/syft" | grep -F $'\tmod\tgithub.com/anchore/syft\tv1.49.0\t'`,
		`go version -m "$RUNNER_TEMP/syft" | grep -F ': go1.26.5'`,
	} {
		require.Contains(t, toolStep.Run, command)
	}
	require.Equal(t, "set -euo pipefail\n\"$RUNNER_TEMP/goreleaser\" check", strings.TrimSpace(job.Steps[3].Run))
	require.Equal(t, "set -euo pipefail\nPATH=\"$RUNNER_TEMP:$PATH\" \"$RUNNER_TEMP/goreleaser\" release --snapshot --clean --skip=ko,sign,announce,publish", strings.TrimSpace(job.Steps[4].Run))
	verifyStep := job.Steps[5]
	require.Empty(t, verifyStep.Uses)
	for _, command := range []string{
		`snapshot_commit="$(git rev-parse HEAD)"`,
		`snapshot_version="0.69.0-dev.$(git rev-parse --short HEAD)"`,
		`snapshot_commit_time="$(git show -s --format=%cI HEAD)"`,
		`MARS_GORELEASER_VERIFY=1`,
		`MARS_GORELEASER_DIST="$PWD/dist"`,
		`MARS_GORELEASER_VERSION="$snapshot_version"`,
		`MARS_GORELEASER_COMMIT="$snapshot_commit"`,
		`MARS_GORELEASER_COMMIT_TIME="$snapshot_commit_time"`,
		`MARS_GORELEASER_GO_VERSION="$(go env GOVERSION)"`,
		`go test ./internal/release -run '^TestVerifyGoReleaserSnapshotDistFromEnvironment$' -count=1`,
	} {
		require.Contains(t, verifyStep.Run, command)
	}

	fullSHA := regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_./-]+@[0-9a-f]{40}$`)
	for _, step := range job.Steps {
		if step.Uses != "" {
			require.Regexp(t, fullSHA, step.Uses)
		}
	}
	text := string(raw)
	for _, forbidden := range []string{
		"pull_request_target", "secrets.", "GITHUB_TOKEN", "GH_TOKEN", "id-token:",
		"contents: write", "actions/cache", "upload-artifact", "goreleaser-action",
		"download-syft", "continue-on-error", "curl ", "wget ",
	} {
		require.NotContains(t, text, forbidden)
	}
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

func archiveSources(files []goreleaserTestArchiveFile) []string {
	result := make([]string, len(files))
	for i, file := range files {
		result[i] = file.Src
	}
	return result
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
