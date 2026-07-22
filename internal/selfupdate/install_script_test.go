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
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInstallScriptFailsClosedToSourceInstall(t *testing.T) {
	for _, version := range []string{"latest", "v0.69.0"} {
		t.Run(version, func(t *testing.T) {
			fakeBin := t.TempDir()
			calls := filepath.Join(t.TempDir(), "calls.log")
			for _, name := range []string{
				"curl", "wget", "gh", "git", "go", "make", "mars", "sudo",
				"mv", "chmod", "uname", "mktemp", "sha256sum", "shasum",
			} {
				writeInstallScriptCommandStub(t, fakeBin, name)
			}

			installDir := filepath.Join(t.TempDir(), "install-dir-canary")
			require.NoError(t, os.Mkdir(installDir, 0o700))
			destination := filepath.Join(installDir, "mars")
			prior := []byte("trusted-prior-binary")
			require.NoError(t, os.WriteFile(destination, prior, 0o755))

			script := filepath.Join("..", "..", "scripts", "install.sh")
			cmd := exec.Command("/bin/bash", script)
			cmd.Env = []string{
				"PATH=" + fakeBin,
				"CALL_LOG=" + calls,
				"VERSION=" + version,
				"INSTALL_DIR=" + installDir,
				"GH_TOKEN=gh-token-canary",
				"GITHUB_TOKEN=github-token-canary",
			}
			output, err := cmd.CombinedOutput()

			var exitErr *exec.ExitError
			require.ErrorAs(t, err, &exitErr, string(output))
			require.Equal(t, 1, exitErr.ExitCode(), string(output))
			text := string(output)
			require.Contains(t, text, "signed binary bootstrap is not available")
			require.Contains(t, text, "independently trusted bootstrap")
			require.Contains(t, text, "Go 1.25.12 or newer")
			require.Contains(t, text, "git clone https://github.com/greaveselliott/MARS.git")
			require.Contains(t, text, "cd MARS")
			require.Contains(t, text, "make install")
			for _, secret := range []string{version, installDir, "gh-token-canary", "github-token-canary"} {
				require.NotContains(t, text, secret)
			}

			_, err = os.Stat(calls)
			require.True(t, errors.Is(err, os.ErrNotExist), "installer invoked an external command")
			got, err := os.ReadFile(destination)
			require.NoError(t, err)
			require.Equal(t, prior, got)
			entries, err := os.ReadDir(installDir)
			require.NoError(t, err)
			require.Len(t, entries, 1)
			require.Equal(t, "mars", entries[0].Name())
		})
	}
}

func TestInstallScriptContainsNoUnsignedBinaryBootstrap(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "scripts", "install.sh"))
	require.NoError(t, err)
	text := string(raw)

	for _, forbidden := range []string{
		"curl", "wget", "/releases/", "releases/latest", "checksums.txt",
		"sha256sum", "shasum", "mars-${os}-${arch}", "mars-harness-", "sudo",
		"mktemp", "chmod +x", "mv \"${tmpdir}",
	} {
		require.NotContains(t, text, forbidden)
	}
	require.Contains(t, text, "signed binary bootstrap is not available")
	require.Contains(t, text, "make install")
}

func writeInstallScriptCommandStub(t *testing.T, dir, name string) {
	t.Helper()
	stub := "#!/bin/sh\nprintf '%s\\n' " + name + " >> \"$CALL_LOG\"\nexit 99\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(stub), 0o755))
}
