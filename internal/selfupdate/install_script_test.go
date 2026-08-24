/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/release-versioning.md
- docs/features/F-009-release-update-lifecycle.md
- docs/features/F-018-goreleaser-distribution.md
*/
package selfupdate

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	installScriptTestVersion = "v0.69.1"
	installScriptCommandPath = "github.com/greaveselliott/mars/cmd/mars"
	installScriptModulePath  = "github.com/greaveselliott/mars"
)

func TestInstallScriptStagesExactGoModuleAndDelegatesToSignedUpdater(t *testing.T) {
	fixture := newInstallScriptFixture(t)
	output, err := fixture.run(installScriptTestVersion, fixture.installDir)
	require.NoError(t, err, string(output))

	log := fixture.readLog()
	require.Contains(t, log, "go <env> <GOVERSION>")
	require.Contains(t, log, "go <install> <"+installScriptCommandPath+"@"+installScriptTestVersion+">")
	require.Contains(t, log, "go <version> <-m>")
	require.Contains(t, log, "mars <update> <tool> <--version> <"+installScriptTestVersion+"> <--install-dir> <"+fixture.installDir+"> <--bootstrap-exact-module> <--skip-shell-path>")
	require.Contains(t, log, "GOPROXY=https://proxy.golang.org")
	require.Contains(t, log, "GOSUMDB=sum.golang.org")
	require.Contains(t, log, "GONOPROXY= GOPRIVATE= GONOSUMDB= GOINSECURE=")
	require.Contains(t, log, "GOENV=off GOTOOLCHAIN=local GOWORK=off GOFLAGS=-modcacherw GOAUTH=off CGO_ENABLED=0")
	require.Contains(t, log, "module-cache read-only-file=0444")
	require.Contains(t, log, "GOBIN="+fixture.stagingRoot+string(filepath.Separator)+"mars-bootstrap.")
	require.Contains(t, log, "TMPDIR="+fixture.stagingRoot+string(filepath.Separator)+"mars-bootstrap.")
	require.Contains(t, log, "GOTMPDIR="+fixture.stagingRoot+string(filepath.Separator)+"mars-bootstrap.")
	require.Contains(t, log, "controls GOROOT= GOCACHEPROG= GOEXPERIMENT= GCCGO= CC= CXX= AR= GOOS= GOARCH= CGO_CFLAGS= CGO_LDFLAGS=")
	require.Contains(t, log, "go-auth GH_TOKEN=empty GITHUB_TOKEN=empty")
	require.Contains(t, log, "updater-auth GH_TOKEN=set GITHUB_TOKEN=set")
	require.Contains(t, log, "internal-token-vars go=absent")
	require.Contains(t, log, "internal-token-vars updater=absent")

	fixture.requirePriorUnchanged()
	fixture.requireStagingClean()
	text := string(output)
	require.Contains(t, text, "public Go proxy and checksum database")
	require.Contains(t, text, "delegating archive and signature verification to MARS")
	require.Contains(t, text, "signed release installation completed")
	for _, sensitive := range []string{
		"gh-token-canary", "github-token-canary", fixture.stagingRoot, "modcache", "buildcache",
	} {
		require.NotContains(t, text, sensitive)
	}
}

func TestInstallScriptIgnoresExportedHostileUtilityFunctions(t *testing.T) {
	fixture := newInstallScriptFixture(t)
	canary := filepath.Join(t.TempDir(), "function-canary")
	bashEnvCanary := filepath.Join(t.TempDir(), "bash-env-canary")
	bashEnv := filepath.Join(t.TempDir(), "hostile-bash-env")
	require.NoError(t, os.WriteFile(bashEnv, []byte("/bin/echo sourced > "+shellQuote(bashEnvCanary)+"\n"), 0o600))
	substitute := filepath.Join(t.TempDir(), "substituted-stage")
	fixture.extraEnv = append(fixture.extraEnv,
		"BOOTSTRAP_FUNCTION_CANARY="+canary,
		"BOOTSTRAP_SUBSTITUTE_PATH="+substitute,
		"BASH_ENV="+bashEnv,
		"UNRELATED_SECRET=must-not-survive",
		"mars_inherited_gh_token=exported-internal-gh-canary",
		"mars_inherited_github_token=exported-internal-github-canary",
	)
	for _, name := range []string{
		"builtin", "command", "stat", "mktemp", "chmod", "mkdir", "rm", "cd", "pwd", "printf", "read", "type", "go",
		"set", "readonly", "local", "export", "unset", "trap", "umask", "return", "exit", "true", "break",
		"/usr/bin/env", "/bin/bash",
	} {
		body := `() { /bin/echo ` + name + ` >> "$BOOTSTRAP_FUNCTION_CANARY"; return 91; }`
		if name == "mktemp" {
			body = `() { /bin/echo mktemp >> "$BOOTSTRAP_FUNCTION_CANARY"; /bin/echo "$BOOTSTRAP_SUBSTITUTE_PATH"; return 0; }`
		}
		fixture.extraEnv = append(fixture.extraEnv, "BASH_FUNC_"+name+"%%="+body)
	}

	output, err := fixture.run(installScriptTestVersion, fixture.installDir)
	if err != nil {
		invoked, _ := os.ReadFile(canary)
		t.Fatalf("install failed: %v\noutput:\n%s\nhostile functions invoked:\n%s", err, output, invoked)
	}
	require.NoFileExists(t, canary)
	require.NoFileExists(t, bashEnvCanary)
	require.NoFileExists(t, substitute)
	require.NotContains(t, fixture.readLog(), "must-not-survive")
	require.NotContains(t, fixture.readLog(), "exported-internal-")
	fixture.requirePriorUnchanged()
	fixture.requireStagingClean()
}

func TestInstallScriptRejectsNonPrivilegedShellInvocation(t *testing.T) {
	fixture := newInstallScriptFixture(t)
	script := filepath.Join("..", "..", "scripts", "install.sh")
	cmd := exec.Command("/bin/bash", script, installScriptTestVersion, fixture.installDir)
	cmd.Env = append([]string{
		"PATH=" + fixture.fakeBin + ":/usr/bin:/bin",
		"HOME=" + t.TempDir(),
		"TMPDIR=" + fixture.stagingRoot,
	}, fixture.extraEnv...)
	output, err := cmd.CombinedOutput()
	require.Error(t, err, string(output))
	require.Equal(t, "Error: execute ./scripts/install.sh directly so #!/bin/bash -p establishes startup isolation.\n", string(output))
	fixture.requireLogAbsent()
	fixture.requirePriorUnchanged()
	fixture.requireStagingClean()
}

func TestInstallScriptRequiresExactArityAndSafeDestination(t *testing.T) {
	fixture := newInstallScriptFixture(t)
	for name, args := range map[string][]string{
		"none":     {},
		"tag only": {installScriptTestVersion},
		"extra":    {installScriptTestVersion, fixture.installDir, "unexpected"},
	} {
		t.Run(name, func(t *testing.T) {
			output, err := fixture.run(args...)
			require.Error(t, err, string(output))
			require.Contains(t, string(output), "exactly one stable release tag and one final install directory")
		})
	}

	fileDestination := filepath.Join(t.TempDir(), "not-a-directory")
	require.NoError(t, os.WriteFile(fileDestination, []byte("file"), 0o600))
	for _, destination := range []string{"relative", "/", filepath.Join(t.TempDir(), "missing"), fileDestination} {
		output, err := fixture.run(installScriptTestVersion, destination)
		require.Error(t, err, string(output))
		require.Contains(t, string(output), "owner-controlled absolute directory")
	}
	fixture.requireLogAbsent()
	fixture.requirePriorUnchanged()
}

func TestInstallScriptRejectsNonExactStableVersionsBeforeBootstrap(t *testing.T) {
	for _, version := range []string{
		"", "latest", "main", "0.69.1", "v0.69", "v0.69.01", "v0.69.1-rc.1",
		"v0.69.1+build", "v0.0.0-20260809-abcdef", "abcdef123456",
	} {
		t.Run(strings.ReplaceAll(version, "/", "_"), func(t *testing.T) {
			fixture := newInstallScriptFixture(t)
			output, err := fixture.run(version, fixture.installDir)
			require.Error(t, err, string(output))
			require.Contains(t, string(output), "one exact stable semantic tag")
			fixture.requireLogAbsent()
			fixture.requirePriorUnchanged()
			fixture.requireStagingClean()
		})
	}
}

func TestInstallScriptRejectsBelowMinimumGoBeforeStaging(t *testing.T) {
	fixture := newInstallScriptFixture(t)
	fixture.extraEnv = append(fixture.extraEnv, "BOOTSTRAP_GO_VERSION=go1.25.11")
	output, err := fixture.run(installScriptTestVersion, fixture.installDir)
	require.Error(t, err, string(output))
	require.Contains(t, string(output), "Go 1.25.12 or newer")
	require.Equal(t, "go <env> <GOVERSION>\n", fixture.readLog())
	fixture.requirePriorUnchanged()
	fixture.requireStagingClean()
}

func TestInstallScriptRejectsMalformedGoVersionsBeforeStaging(t *testing.T) {
	for _, version := range []string{"", "go1.26", "1.26.5", "go1.26.5rc1", "devel go1.27"} {
		t.Run(strings.ReplaceAll(version, " ", "_"), func(t *testing.T) {
			fixture := newInstallScriptFixture(t)
			fixture.extraEnv = append(fixture.extraEnv, "BOOTSTRAP_GO_VERSION="+version)
			output, err := fixture.run(installScriptTestVersion, fixture.installDir)
			require.Error(t, err, string(output))
			require.Contains(t, string(output), "unsupported version")
			fixture.requirePriorUnchanged()
			fixture.requireStagingClean()
		})
	}
}

func TestInstallScriptRejectsUnsafeOwnerTemporaryRoots(t *testing.T) {
	for _, mode := range []os.FileMode{0o0777, os.ModeSticky | 0o0777} {
		t.Run(mode.String(), func(t *testing.T) {
			fixture := newInstallScriptFixture(t)
			require.NoError(t, os.Chmod(fixture.stagingRoot, mode))
			output, err := fixture.run(installScriptTestVersion, fixture.installDir)
			require.Error(t, err, string(output))
			require.Contains(t, string(output), "must not be group- or world-writable")
			require.NotContains(t, fixture.readLog(), "go <install>")
			fixture.requirePriorUnchanged()
			fixture.requireStagingClean()
		})
	}
}

func TestInstallScriptMechanicallyRestrictsSharedRootAndMktempIdentity(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "scripts", "install.sh"))
	require.NoError(t, err)
	text := string(raw)
	for _, contract := range []string{
		`owner == EUID && !(mode & 0022)`,
		`owner == 0 && !(mode & 0022)`,
		`owner == 0 && (mode & 01000) && (mode & 0002)`,
		`elif (( owner != 0 ))`,
		`suffix="${staging_dir#"$staging_prefix"}"`,
		`"$suffix" =~ ^[A-Za-z0-9]{8}$`,
		`owner != EUID || mode != 0700`,
	} {
		require.Contains(t, text, contract)
	}
}

func TestInstallScriptCleansStagingAfterGoInstallFailure(t *testing.T) {
	fixture := newInstallScriptFixture(t)
	fixture.extraEnv = append(fixture.extraEnv, "BOOTSTRAP_GO_INSTALL_EXIT=17")
	output, err := fixture.run(installScriptTestVersion, fixture.installDir)
	require.Error(t, err, string(output))
	require.Contains(t, string(output), "exact-version Go/SumDB bootstrap failed")
	require.NotContains(t, fixture.readLog(), "mars <update>")
	fixture.requirePriorUnchanged()
	fixture.requireStagingClean()
}

func TestInstallScriptRejectsUnsafeInstallDirectoriesBeforeBootstrap(t *testing.T) {
	fixture := newInstallScriptFixture(t)
	symlink := filepath.Join(t.TempDir(), "install-link")
	require.NoError(t, os.Symlink(fixture.installDir, symlink))

	for _, installDir := range []string{"relative/install", "/", symlink} {
		output, err := fixture.run(installScriptTestVersion, installDir)
		require.Error(t, err, string(output))
		require.Contains(t, string(output), "owner-controlled absolute directory")
	}
	fixture.requireLogAbsent()
	fixture.requirePriorUnchanged()
	fixture.requireStagingClean()
}

func TestInstallScriptRejectsNoncanonicalStagedBuildMetadata(t *testing.T) {
	tests := map[string]string{
		"wrong command path": "BOOTSTRAP_COMMAND_PATH=example.com/not-mars/cmd/mars",
		"wrong module path":  "BOOTSTRAP_MODULE_PATH=example.com/not-mars",
		"wrong version":      "BOOTSTRAP_MODULE_VERSION=v0.69.0",
		"missing sum":        "BOOTSTRAP_MODULE_SUM=",
		"short sum":          "BOOTSTRAP_MODULE_SUM=h1:testsum",
		"invalid alphabet":   "BOOTSTRAP_MODULE_SUM=h1:47DEQpj8HBSa-_TImW-5JCeuQeRkm5NMpJWZG3hSuFU=",
		"replacement":        "BOOTSTRAP_REPLACEMENT=1",
	}
	for name, setting := range tests {
		t.Run(name, func(t *testing.T) {
			fixture := newInstallScriptFixture(t)
			fixture.extraEnv = append(fixture.extraEnv, setting)
			output, err := fixture.run(installScriptTestVersion, fixture.installDir)
			require.Error(t, err, string(output))
			require.NotContains(t, fixture.readLog(), "mars <update>")
			fixture.requirePriorUnchanged()
			fixture.requireStagingClean()
			for _, sensitive := range []string{fixture.stagingRoot, "modcache", "buildcache"} {
				require.NotContains(t, string(output), sensitive)
			}
		})
	}
}

func TestInstallScriptUpdaterFailureUsesRecoveryTruthAndCleansStaging(t *testing.T) {
	fixture := newInstallScriptFixture(t)
	fixture.extraEnv = append(fixture.extraEnv, "BOOTSTRAP_UPDATE_EXIT=23")
	output, err := fixture.run(installScriptTestVersion, fixture.installDir)
	require.Error(t, err, string(output))
	require.Contains(t, string(output), "follow the updater recovery guidance")
	require.Contains(t, string(output), "preserve .mars-update.transaction if instructed")
	require.NotContains(t, string(output), "prior final binary was preserved")
	require.Contains(t, fixture.readLog(), "mars <update> <tool>")
	fixture.requirePriorUnchanged()
	fixture.requireStagingClean()
}

func TestInstallScriptReportsInstalledBinaryWhenStagingCleanupIsIncomplete(t *testing.T) {
	fixture := newInstallScriptFixture(t)
	fixture.extraEnv = append(fixture.extraEnv, "BOOTSTRAP_CLEANUP_FAIL=1")
	output, err := fixture.run(installScriptTestVersion, fixture.installDir)
	require.Error(t, err, string(output))
	require.Contains(t, string(output), "binary installed but bootstrap staging cleanup is incomplete")
	require.NotContains(t, string(output), "signed release installation completed")
	require.NotContains(t, string(output), fixture.stagingRoot)
	fixture.requirePriorUnchanged()
	fixture.requireRetainedStagingThenRemove()
}

func TestInstallScriptPreservesFailureTruthWhenCleanupIsIncomplete(t *testing.T) {
	fixture := newInstallScriptFixture(t)
	fixture.extraEnv = append(fixture.extraEnv, "BOOTSTRAP_UPDATE_EXIT=23", "BOOTSTRAP_CLEANUP_FAIL=1")
	output, err := fixture.run(installScriptTestVersion, fixture.installDir)
	require.Error(t, err, string(output))
	require.Contains(t, string(output), "signed release installation failed")
	require.Contains(t, string(output), "Warning: bootstrap staging cleanup incomplete")
	require.NotContains(t, string(output), "binary installed but")
	require.NotContains(t, string(output), fixture.stagingRoot)
	fixture.requirePriorUnchanged()
	fixture.requireRetainedStagingThenRemove()
}

func TestInstallScriptContainsOnlyGoBootstrapAndSignedUpdaterHandoff(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "scripts", "install.sh"))
	require.NoError(t, err)
	text := string(raw)

	for _, required := range []string{
		"#!/bin/bash -p", `/usr/bin/env -i`, `/bin/bash -p -s -- "$@"`, `[[ "$-" == *p* ]]`,
		`builtin unset mars_inherited_gh_token mars_inherited_github_token`,
		`GH_TOKEN= GITHUB_TOKEN= /usr/bin/env -i`, `3<<<"$mars_inherited_gh_token"`, `4<<<"$mars_inherited_github_token"`,
		`exec 3<&- 4<&-`, `[[ -e /dev/fd/3 || -e /dev/fd/4 ]]`,
		`bootstrap credential transport descriptors did not close`,
		`GH_TOKEN="$mars_gh_token" GITHUB_TOKEN="$mars_github_token"`,
		`execute ./scripts/install.sh directly so #!/bin/bash -p establishes startup isolation`,
		`${MARS_COMMAND}@${version}`,
		`GOPROXY="$PUBLIC_GO_PROXY"`, `GOSUMDB="$PUBLIC_GO_SUMDB"`,
		"GOPRIVATE=", "GONOSUMDB=", "GONOPROXY=", "GOTOOLCHAIN=local", "GOENV=off", "GOFLAGS=-modcacherw", "GOAUTH=off", "CGO_ENABLED=0",
		`TMPDIR="$staging_dir/tmp"`, `GOTMPDIR="$staging_dir/tmp"`,
		`version -m "$binary"`, `--bootstrap-exact-module --skip-shell-path`,
		`builtin command /usr/bin/stat`, `builtin command /usr/bin/mktemp`,
		`builtin command /bin/mkdir`, `builtin command /bin/rm`,
		`owner == EUID && !(mode & 0022)`, `owner == 0 && !(mode & 0022)`,
		`owner == 0 && (mode & 01000) && (mode & 0002)`, `elif (( owner != 0 ))`,
		`if ! cleanup; then`, `binary installed but bootstrap staging cleanup is incomplete`,
	} {
		require.Contains(t, text, required)
	}
	for _, forbidden := range []string{
		"curl", "wget", "/releases/", "releases/latest", "checksums.txt",
		"sha256sum", "shasum", "mars-${os}-${arch}", "mars-harness-", "sudo",
		"chmod -R", `"GH_TOKEN=${GH_TOKEN:-}"`, `"GITHUB_TOKEN=${GITHUB_TOKEN:-}"`,
		`/usr/bin/env -i GH_TOKEN=`, `/usr/bin/env -i GITHUB_TOKEN=`,
	} {
		require.NotContains(t, text, forbidden)
	}
}

type installScriptFixture struct {
	t            *testing.T
	fakeBin      string
	stagingRoot  string
	installDir   string
	destination  string
	logPath      string
	configPath   string
	marsTemplate string
	prior        []byte
	extraEnv     []string
}

func newInstallScriptFixture(t *testing.T) *installScriptFixture {
	t.Helper()
	stagingRoot := t.TempDir()
	resolvedStagingRoot, err := filepath.EvalSymlinks(stagingRoot)
	require.NoError(t, err)
	fixture := &installScriptFixture{
		t:           t,
		fakeBin:     t.TempDir(),
		stagingRoot: resolvedStagingRoot,
		installDir:  t.TempDir(),
		logPath:     filepath.Join(t.TempDir(), "bootstrap.log"),
		prior:       []byte("trusted-prior-binary"),
	}
	fixture.configPath = filepath.Join(fixture.fakeBin, "bootstrap-fixture.env")
	t.Cleanup(func() {
		entries, _ := os.ReadDir(fixture.stagingRoot)
		for _, entry := range entries {
			stage := filepath.Join(fixture.stagingRoot, entry.Name())
			_ = os.Chmod(stage, 0o700)
			_ = os.Chmod(filepath.Join(stage, "cleanup-locked"), 0o700)
			_ = os.RemoveAll(stage)
		}
	})
	fixture.destination = filepath.Join(fixture.installDir, "mars")
	require.NoError(t, os.WriteFile(fixture.destination, fixture.prior, 0o755))

	fixture.marsTemplate = filepath.Join(fixture.fakeBin, "mars-template")
	marsStub := `#!/bin/bash
. ` + shellQuote(fixture.configPath) + `
printf 'mars' >> "$BOOTSTRAP_LOG"
printf ' <%s>' "$@" >> "$BOOTSTRAP_LOG"
printf '\n' >> "$BOOTSTRAP_LOG"
if [ -n "${GH_TOKEN:-}" ]; then gh_state=set; else gh_state=empty; fi
if [ -n "${GITHUB_TOKEN:-}" ]; then github_state=set; else github_state=empty; fi
printf 'updater-auth GH_TOKEN=%s GITHUB_TOKEN=%s\n' "$gh_state" "$github_state" >> "$BOOTSTRAP_LOG"
if [ -n "${mars_inherited_gh_token:-}${mars_inherited_github_token:-}" ]; then internal_state=present; else internal_state=absent; fi
printf 'internal-token-vars updater=%s\n' "$internal_state" >> "$BOOTSTRAP_LOG"
if [ "${BOOTSTRAP_CLEANUP_FAIL:-0}" = 1 ]; then
  cleanup_stage="${0%/bin/mars}/cleanup-locked"
  /bin/mkdir "$cleanup_stage"
  /usr/bin/touch "$cleanup_stage/file"
  /bin/chmod 0000 "$cleanup_stage"
fi
exit "${BOOTSTRAP_UPDATE_EXIT:-0}"
`
	require.NoError(t, os.WriteFile(fixture.marsTemplate, []byte(marsStub), 0o700))

	goStub := `#!/bin/bash
set -eu
. ` + shellQuote(fixture.configPath) + `
printf 'go' >> "$BOOTSTRAP_LOG"
printf ' <%s>' "$@" >> "$BOOTSTRAP_LOG"
printf '\n' >> "$BOOTSTRAP_LOG"
case "$1" in
  env)
    printf '%s\n' "${BOOTSTRAP_GO_VERSION-go1.26.5}"
    ;;
  install)
	if [ -n "${GH_TOKEN:-}" ]; then gh_state=set; else gh_state=empty; fi
	if [ -n "${GITHUB_TOKEN:-}" ]; then github_state=set; else github_state=empty; fi
	printf 'go-auth GH_TOKEN=%s GITHUB_TOKEN=%s\n' "$gh_state" "$github_state" >> "$BOOTSTRAP_LOG"
	if [ -n "${mars_inherited_gh_token:-}${mars_inherited_github_token:-}" ]; then internal_state=present; else internal_state=absent; fi
	printf 'internal-token-vars go=%s\n' "$internal_state" >> "$BOOTSTRAP_LOG"
    printf 'env GOPROXY=%s GOSUMDB=%s GONOPROXY=%s GOPRIVATE=%s GONOSUMDB=%s GOINSECURE=%s GOENV=%s GOTOOLCHAIN=%s GOWORK=%s GOFLAGS=%s GOAUTH=%s CGO_ENABLED=%s GOBIN=%s TMPDIR=%s GOTMPDIR=%s\n' \
      "$GOPROXY" "$GOSUMDB" "$GONOPROXY" "$GOPRIVATE" "$GONOSUMDB" "$GOINSECURE" "$GOENV" "$GOTOOLCHAIN" "$GOWORK" "$GOFLAGS" "$GOAUTH" "$CGO_ENABLED" "$GOBIN" "$TMPDIR" "$GOTMPDIR" >> "$BOOTSTRAP_LOG"
    printf 'controls GOROOT=%s GOCACHEPROG=%s GOEXPERIMENT=%s GCCGO=%s CC=%s CXX=%s AR=%s GOOS=%s GOARCH=%s CGO_CFLAGS=%s CGO_LDFLAGS=%s\n' \
      "${GOROOT:-}" "${GOCACHEPROG:-}" "${GOEXPERIMENT:-}" "${GCCGO:-}" "${CC:-}" "${CXX:-}" "${AR:-}" "${GOOS:-}" "${GOARCH:-}" "${CGO_CFLAGS:-}" "${CGO_LDFLAGS:-}" >> "$BOOTSTRAP_LOG"
    if [ "${BOOTSTRAP_GO_INSTALL_EXIT:-0}" != 0 ]; then
      exit "$BOOTSTRAP_GO_INSTALL_EXIT"
    fi
    /bin/cp "$BOOTSTRAP_MARS_TEMPLATE" "$GOBIN/mars"
    /bin/chmod 0700 "$GOBIN/mars"
    /bin/mkdir -p "$GOMODCACHE/example.com/module@v1.0.0"
    printf 'verified module cache fixture\n' > "$GOMODCACHE/example.com/module@v1.0.0/source.go"
    /bin/chmod 0444 "$GOMODCACHE/example.com/module@v1.0.0/source.go"
    printf 'module-cache read-only-file=0444\n' >> "$BOOTSTRAP_LOG"
    ;;
  version)
    printf '%s\n' \
      "$3: go1.26.5" \
      $'\tpath\t'"${BOOTSTRAP_COMMAND_PATH:-github.com/greaveselliott/mars/cmd/mars}" \
      $'\tmod\t'"${BOOTSTRAP_MODULE_PATH:-github.com/greaveselliott/mars}"$'\t'"${BOOTSTRAP_MODULE_VERSION:-v0.69.1}"$'\t'"${BOOTSTRAP_MODULE_SUM-h1:47DEQpj8HBSa+/TImW+5JCeuQeRkm5NMpJWZG3hSuFU=}"
    if [ "${BOOTSTRAP_REPLACEMENT:-0}" = 1 ]; then
      printf '%s\n' $'\t=>\t./replacement'
    fi
    ;;
  *)
    exit 97
    ;;
esac
`
	require.NoError(t, os.WriteFile(filepath.Join(fixture.fakeBin, "go"), []byte(goStub), 0o700))
	return fixture
}

func (f *installScriptFixture) run(args ...string) ([]byte, error) {
	f.t.Helper()
	f.writeConfig()
	script := filepath.Join("..", "..", "scripts", "install.sh")
	cmd := exec.Command(script, args...)
	cmd.Env = append([]string{
		"PATH=" + f.fakeBin + ":/usr/bin:/bin",
		"HOME=" + f.t.TempDir(),
		"TMPDIR=" + f.stagingRoot,
		"GH_TOKEN=gh-token-canary",
		"GITHUB_TOKEN=github-token-canary",
		"GOPROXY=off",
		"GOSUMDB=off",
		"GONOPROXY=" + installScriptModulePath,
		"GOPRIVATE=" + installScriptModulePath,
		"GONOSUMDB=" + installScriptModulePath,
		"GOINSECURE=" + installScriptModulePath,
		"GOWORK=/hostile/workspace",
		"GOFLAGS=-mod=vendor",
		"GOAUTH=netrc",
		"CGO_ENABLED=1",
		"GOROOT=/hostile/goroot",
		"GOCACHEPROG=/hostile/cache-program",
		"GOEXPERIMENT=hostile",
		"GCCGO=/hostile/gccgo",
		"CC=/hostile/cc",
		"CXX=/hostile/cxx",
		"AR=/hostile/ar",
		"GOOS=windows",
		"GOARCH=amd64",
		"CGO_CFLAGS=-DHOSTILE",
		"CGO_LDFLAGS=-L/hostile",
	}, f.extraEnv...)
	return cmd.CombinedOutput()
}

func (f *installScriptFixture) writeConfig() {
	f.t.Helper()
	defaults := map[string]string{
		"BOOTSTRAP_LOG":             f.logPath,
		"BOOTSTRAP_MARS_TEMPLATE":   f.marsTemplate,
		"BOOTSTRAP_GO_VERSION":      "go1.26.5",
		"BOOTSTRAP_GO_INSTALL_EXIT": "0",
		"BOOTSTRAP_COMMAND_PATH":    installScriptCommandPath,
		"BOOTSTRAP_MODULE_PATH":     installScriptModulePath,
		"BOOTSTRAP_MODULE_VERSION":  installScriptTestVersion,
		"BOOTSTRAP_MODULE_SUM":      "h1:47DEQpj8HBSa+/TImW+5JCeuQeRkm5NMpJWZG3hSuFU=",
		"BOOTSTRAP_REPLACEMENT":     "0",
		"BOOTSTRAP_UPDATE_EXIT":     "0",
		"BOOTSTRAP_CLEANUP_FAIL":    "0",
	}
	for _, setting := range f.extraEnv {
		name, value, ok := strings.Cut(setting, "=")
		if !ok {
			continue
		}
		if _, allowed := defaults[name]; allowed {
			defaults[name] = value
		}
	}
	names := []string{
		"BOOTSTRAP_LOG", "BOOTSTRAP_MARS_TEMPLATE", "BOOTSTRAP_GO_VERSION", "BOOTSTRAP_GO_INSTALL_EXIT",
		"BOOTSTRAP_COMMAND_PATH", "BOOTSTRAP_MODULE_PATH", "BOOTSTRAP_MODULE_VERSION", "BOOTSTRAP_MODULE_SUM",
		"BOOTSTRAP_REPLACEMENT", "BOOTSTRAP_UPDATE_EXIT", "BOOTSTRAP_CLEANUP_FAIL",
	}
	var content strings.Builder
	for _, name := range names {
		content.WriteString(name)
		content.WriteByte('=')
		content.WriteString(shellQuote(defaults[name]))
		content.WriteByte('\n')
	}
	require.NoError(f.t, os.WriteFile(f.configPath, []byte(content.String()), 0o600))
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

func (f *installScriptFixture) readLog() string {
	f.t.Helper()
	raw, err := os.ReadFile(f.logPath)
	require.NoError(f.t, err)
	return string(raw)
}

func (f *installScriptFixture) requireLogAbsent() {
	f.t.Helper()
	_, err := os.Stat(f.logPath)
	require.ErrorIs(f.t, err, os.ErrNotExist)
}

func (f *installScriptFixture) requirePriorUnchanged() {
	f.t.Helper()
	got, err := os.ReadFile(f.destination)
	require.NoError(f.t, err)
	require.Equal(f.t, f.prior, got)
	entries, err := os.ReadDir(f.installDir)
	require.NoError(f.t, err)
	require.Len(f.t, entries, 1)
	require.Equal(f.t, "mars", entries[0].Name())
}

func (f *installScriptFixture) requireStagingClean() {
	f.t.Helper()
	entries, err := os.ReadDir(f.stagingRoot)
	require.NoError(f.t, err)
	require.Empty(f.t, entries)
}

func (f *installScriptFixture) requireRetainedStagingThenRemove() {
	f.t.Helper()
	entries, err := os.ReadDir(f.stagingRoot)
	require.NoError(f.t, err)
	require.Len(f.t, entries, 1)
	stage := filepath.Join(f.stagingRoot, entries[0].Name())
	require.NoError(f.t, os.Chmod(stage, 0o700))
	locked := filepath.Join(stage, "cleanup-locked")
	if _, err := os.Lstat(locked); err == nil {
		require.NoError(f.t, os.Chmod(locked, 0o700))
	}
	require.NoError(f.t, os.RemoveAll(stage))
	f.requireStagingClean()
}
