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
	RunsOn         string           `yaml:"runs-on"`
	TimeoutMinutes int              `yaml:"timeout-minutes"`
	Strategy       workflowStrategy `yaml:"strategy"`
	Steps          []workflowStep   `yaml:"steps"`
}

type workflowStrategy struct {
	FailFast bool                `yaml:"fail-fast"`
	Matrix   map[string][]string `yaml:"matrix"`
}

type workflowStep struct {
	Name string            `yaml:"name"`
	Uses string            `yaml:"uses"`
	If   string            `yaml:"if"`
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
	require.Equal(t, 45, job.TimeoutMinutes)
	require.Len(t, job.Steps, 6)

	require.Equal(t, "actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1", job.Steps[0].Uses)
	require.Equal(t, map[string]any{"fetch-depth": 0, "persist-credentials": false}, job.Steps[0].With)
	require.Equal(t, "actions/setup-go@b7ad1dad31e06c5925ef5d2fc7ad053ef454303e", job.Steps[1].Uses)
	require.Equal(t, map[string]any{"go-version": "1.26.5", "cache": false}, job.Steps[1].With)

	preflightStep := job.Steps[2]
	require.Empty(t, preflightStep.Uses)
	require.Empty(t, preflightStep.Env)
	for _, command := range []string{
		`rehearsal_root="$RUNNER_TEMP/mars-t068-rehearsal"`,
		`test ! -e "$rehearsal_root"`,
		`mkdir -m 700 "$rehearsal_root"`,
		`test "$(git rev-parse HEAD)" = "$GITHUB_SHA"`,
		`test "$(go env GOVERSION)" = go1.26.5`,
		`test -z "$(git status --porcelain=v1 --untracked-files=all)"`,
		`test -z "$(git tag --points-at HEAD)"`,
		`git config --local --get-regexp '^http\..*\.extraheader$|^credential\.'`,
		`case "$origin_url" in *://*@*) exit 1;; esac`,
		`test -z "${ACTIONS_ID_TOKEN_REQUEST_URL:-}"`,
		`test -z "${ACTIONS_ID_TOKEN_REQUEST_TOKEN:-}"`,
		`test -z "${SSH_AUTH_SOCK:-}"`,
		`test -z "${COSIGN_PASSWORD:-}"`,
		`test -z "${COSIGN_PRIVATE_KEY:-}"`,
		`/usr/bin/env -i`,
		`HOME="$rehearsal_root/preflight-home" TMPDIR="$rehearsal_root/preflight-tmp"`,
		`GOMODCACHE="$rehearsal_root/go-mod-cache" GOCACHE="$rehearsal_root/preflight-cache"`,
		`GOPROXY=https://proxy.golang.org GOSUMDB=sum.golang.org GOPRIVATE= GONOSUMDB=`,
		`"$go_bin" test ./internal/release`,
		`-run '^(TestGoReleaserConfigContract|TestSyftConfigDisablesNetworkCapableBehavior|TestSnapshotWorkflowContract)$' -count=1`,
	} {
		require.Contains(t, preflightStep.Run, command)
	}

	toolStep := job.Steps[3]
	require.Empty(t, toolStep.Uses)
	require.Empty(t, toolStep.Env)
	for _, command := range []string{
		`rehearsal_root="$RUNNER_TEMP/mars-t068-rehearsal"`,
		`tool_bin="$rehearsal_root/tool-bin"`,
		`tool_env=(/usr/bin/env -i`,
		`GOMODCACHE="$rehearsal_root/go-mod-cache" GOCACHE="$rehearsal_root/tool-cache"`,
		`GOPROXY=https://proxy.golang.org GOSUMDB=sum.golang.org GOPRIVATE= GONOSUMDB=)`,
		`"${tool_env[@]}" "$go_bin" mod download`,
		`"${tool_env[@]}" "$go_bin" install github.com/goreleaser/goreleaser/v2@v2.17.0`,
		`"${tool_env[@]}" "$go_bin" install github.com/anchore/syft/cmd/syft@v1.49.0`,
		`"${inspect_env[@]}" "$tool_bin/goreleaser" --version`,
		`"${inspect_env[@]}" "$tool_bin/syft" version`,
		`"${inspect_env[@]}" "$go_bin" version -m "$tool_bin/goreleaser" | grep -F $'\tmod\tgithub.com/goreleaser/goreleaser/v2\tv2.17.0\t'`,
		`"${inspect_env[@]}" "$go_bin" version -m "$tool_bin/goreleaser" | grep -F ': go1.26.5'`,
		`"${inspect_env[@]}" "$go_bin" version -m "$tool_bin/syft" | grep -F $'\tmod\tgithub.com/anchore/syft\tv1.49.0\t'`,
		`"${inspect_env[@]}" "$go_bin" version -m "$tool_bin/syft" | grep -F ': go1.26.5'`,
		`GOPROXY=off GOSUMDB=sum.golang.org GOPRIVATE= GONOSUMDB=`,
		`"$tool_bin/goreleaser" check`,
	} {
		require.Contains(t, toolStep.Run, command)
	}

	rehearsalStep := job.Steps[4]
	require.Empty(t, rehearsalStep.Uses)
	require.Empty(t, rehearsalStep.Env)
	for _, command := range []string{
		`rehearsal_root="$RUNNER_TEMP/mars-t068-rehearsal"`,
		`trap cleanup EXIT`,
		`chmod -R u+w -- "$rehearsal_root" 2>/dev/null || true`,
		`rm -rf -- "$rehearsal_root" || status=1`,
		`if [ -e "$rehearsal_root" ]; then status=1; fi`,
		`snapshot_commit="$GITHUB_SHA"`,
		`test "$(git rev-parse HEAD)" = "$snapshot_commit"`,
		`snapshot_version="0.69.0-dev.${snapshot_commit:0:7}"`,
		`for lane in a b; do`,
		`git clone --quiet --no-local --no-hardlinks "$GITHUB_WORKSPACE" "$root"`,
		`git -C "$root" checkout --quiet --detach "$snapshot_commit"`,
		`test "$(git -C "$root" remote)" = origin`,
		`test "$(git -C "$root" remote get-url origin)" = "$GITHUB_WORKSPACE"`,
		`git -C "$root" config --local --get-regexp '^http\..*\.extraheader$|^credential\.'`,
		`/usr/bin/env -i HOME="$home" TMPDIR="$tmp" PATH="$rehearsal_path"`,
		`GOPROXY=off GOSUMDB=sum.golang.org GOPRIVATE= GONOSUMDB= GOCACHE="$cache"`,
		`"$tool_bin/goreleaser" release --snapshot --clean --skip=ko,sign,announce,publish`,
		`test -z "$(git -C "$root" tag --points-at HEAD)"`,
		`root_a="$rehearsal_root/source-a"`,
		`root_b="$rehearsal_root/source-b"`,
		`test "$root_a" != "$root_b"`,
		`MARS_GORELEASER_VERIFY=1`,
		`MARS_GORELEASER_DIST="$root_a/dist"`,
		`MARS_GORELEASER_COMPARE_DIST="$root_b/dist"`,
		`MARS_GORELEASER_VERSION="$snapshot_version"`,
		`MARS_GORELEASER_COMMIT="$snapshot_commit"`,
		`MARS_GORELEASER_COMMIT_TIME="$snapshot_commit_time"`,
		`MARS_GORELEASER_GO_VERSION=go1.26.5`,
		`"$go_bin" test ./internal/release -run '^TestVerifyGoReleaserSnapshotDistFromEnvironment$' -count=1`,
		`"$make_bin" -C "$root_a" GO="$go_bin" install`,
		`"$go_bin" version -m "$source_bin/mars" | grep -F $'\tbuild\tvcs.revision='"$snapshot_commit"`,
		`"$go_bin" version -m "$source_bin/mars" | grep -F $'\tbuild\tvcs.modified=false'`,
		`if /usr/bin/env -i PATH="$runtime_bin" /bin/sh -c 'command -v go' >/dev/null 2>&1; then exit 1; fi`,
		`tar --extract --gzip --file="$native_archive" --directory="$archive_dir" --no-same-owner mars`,
		`home="$rehearsal_root/$label-target-home"`,
		`tmp="$rehearsal_root/$label-target-tmp"`,
		`fixture_root="$rehearsal_root/$label-target-root"`,
		`target="$fixture_root/target"`,
		`test ! -e "$target"`,
		`exercise_target source "$source_bin/mars"`,
		`exercise_target archive "$archive_dir/mars"`,
		`source-target-root/target`,
		`archive-target-root/target`,
		`rev-parse 'HEAD^{tree}'`,
		`update_home="$rehearsal_root/update-home"`,
		`cp -a "$update_home" "$update_home_before"`,
		`update tool --dry-run --version v0.69.0`,
		`diff -qr "$update_home_before" "$update_home"`,
		`test ! -e "$update_dir"`,
		`consumer='^(TestVerifySigstoreChecksumsEvidenceRealOfflineFixture|TestFetchVerifiedMARSReleaseAcceptsBoundedAnnotatedTagAndExplicitOlderVersion|TestReplaceVerifiedMARSReleasePreverifiedUpdateAndRollbackLifecycle|TestInstallScriptFailsClosedToSourceInstall|TestInstallScriptContainsNoUnsignedBinaryBootstrap)$'`,
		`"$go_bin" test ./internal/selfupdate -run "$consumer" -count=1`,
	} {
		require.Contains(t, rehearsalStep.Run, command)
	}
	require.Less(t,
		strings.Index(rehearsalStep.Run, "TestVerifyGoReleaserSnapshotDistFromEnvironment"),
		strings.Index(rehearsalStep.Run, "tar --extract"),
	)
	require.Equal(t, 1, strings.Count(rehearsalStep.Run, `"$tool_bin/goreleaser" release`))
	require.Equal(t, 1, strings.Count(rehearsalStep.Run, "for lane in a b; do"))
	require.GreaterOrEqual(t, strings.Count(rehearsalStep.Run, "/usr/bin/env -i"), 7)
	require.NotContains(t, rehearsalStep.Run, "go test -race")

	cleanupStep := job.Steps[5]
	require.Equal(t, "${{ always() }}", cleanupStep.If)
	require.Empty(t, cleanupStep.Uses)
	require.Contains(t, cleanupStep.Run, `rehearsal_root="$RUNNER_TEMP/mars-t068-rehearsal"`)
	require.Contains(t, cleanupStep.Run, `if [ -e "$rehearsal_root" ]; then chmod -R u+w -- "$rehearsal_root" 2>/dev/null || true; fi`)
	require.Contains(t, cleanupStep.Run, `if [ -e "$rehearsal_root" ]; then rm -rf -- "$rehearsal_root"; fi`)
	require.Contains(t, cleanupStep.Run, `test ! -e "$rehearsal_root"`)

	fullSHA := regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_./-]+@[0-9a-f]{40}$`)
	for _, step := range job.Steps {
		if step.Uses != "" {
			require.Regexp(t, fullSHA, step.Uses)
		}
	}
	text := string(raw)
	for _, forbidden := range []string{
		"pull_request_target", "secrets.", "${{ secrets", "github.token", "GITHUB_TOKEN", "GH_TOKEN", "id-token:",
		"contents: write", "actions: write", "packages: write", "actions/cache", "upload-artifact", "goreleaser-action",
		"download-syft", "continue-on-error", "curl ", "wget ", "git push", "gh release", "cosign ", "attest",
	} {
		require.NotContains(t, text, forbidden)
	}
	for _, line := range strings.Split(text, "\n") {
		if strings.Contains(line, `"$tool_bin/goreleaser" release`) {
			require.Contains(t, line, "release --snapshot --clean --skip=ko,sign,announce,publish")
		}
	}
}

func TestSourceCompatibilityWorkflowContract(t *testing.T) {
	t.Parallel()
	path := filepath.Join(releaseRepoRoot(t), ".github", "workflows", "source-compatibility.yml")
	raw := readStrictYAML(t, path, new(snapshotWorkflow))
	var workflow snapshotWorkflow
	readStrictYAMLInto(t, raw, &workflow)

	require.Equal(t, "source-compatibility", workflow.Name)
	require.Equal(t, []string{"pull_request", "push", "workflow_dispatch"}, sortedKeys(workflow.On))
	require.Equal(t, map[string]string{"contents": "read"}, workflow.Permissions)
	require.Equal(t, []string{"below-minimum", "supported-source"}, sortedKeys(workflow.Jobs))

	supported := workflow.Jobs["supported-source"]
	require.Equal(t, "ubuntu-24.04", supported.RunsOn)
	require.Equal(t, 30, supported.TimeoutMinutes)
	require.False(t, supported.Strategy.FailFast)
	require.Equal(t, map[string][]string{"go-version": {"1.25.12", "1.26.5"}}, supported.Strategy.Matrix)
	require.Len(t, supported.Steps, 3)
	require.Equal(t, "actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1", supported.Steps[0].Uses)
	require.Equal(t, map[string]any{"fetch-depth": 0, "persist-credentials": false}, supported.Steps[0].With)
	require.Equal(t, "actions/setup-go@b7ad1dad31e06c5925ef5d2fc7ad053ef454303e", supported.Steps[1].Uses)
	require.Equal(t, map[string]any{"go-version": "${{ matrix.go-version }}", "cache": false}, supported.Steps[1].With)
	require.Equal(t, map[string]string{"GOTOOLCHAIN": "local"}, supported.Steps[2].Env)
	for _, command := range []string{
		`test "$(go env GOVERSION)" = "go${{ matrix.go-version }}"`,
		"go mod tidy -go=1.25.12",
		"git diff --exit-code -- go.mod go.sum",
		"CGO_ENABLED=0 go build ./cmd/mars",
		"go test ./...",
		"go vet ./...",
		"go install golang.org/x/vuln/cmd/govulncheck@v1.6.0",
		`"$RUNNER_TEMP/govulncheck" ./...`,
	} {
		require.Contains(t, supported.Steps[2].Run, command)
	}

	below := workflow.Jobs["below-minimum"]
	require.Equal(t, "ubuntu-24.04", below.RunsOn)
	require.Equal(t, 10, below.TimeoutMinutes)
	require.Len(t, below.Steps, 3)
	require.Equal(t, "actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1", below.Steps[0].Uses)
	require.Equal(t, "actions/setup-go@b7ad1dad31e06c5925ef5d2fc7ad053ef454303e", below.Steps[1].Uses)
	require.Equal(t, map[string]any{"go-version": "1.25.11", "cache": false}, below.Steps[1].With)
	require.Equal(t, map[string]string{"GOTOOLCHAIN": "local"}, below.Steps[2].Env)
	require.Contains(t, below.Steps[2].Run, `test "$(go env GOVERSION)" = "go1.25.11"`)
	require.Contains(t, below.Steps[2].Run, `output="$(go list -m 2>&1)"`)
	require.Contains(t, below.Steps[2].Run, `grep -F "go.mod requires go >= 1.25.12"`)
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
