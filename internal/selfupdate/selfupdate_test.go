package selfupdate

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolvePlan_defaultsToLatestInCurrentExecutableDir(t *testing.T) {
	plan, err := ResolvePlan(Config{DryRun: true})
	require.NoError(t, err)

	require.Equal(t, DefaultPackage, plan.Package)
	require.Equal(t, DefaultVersion, plan.Version)
	require.Equal(t, []string{"go", "install", DefaultPackage + "@latest"}, plan.Command)
	require.True(t, filepath.IsAbs(plan.InstallDir))
	require.Equal(t, filepath.Join(plan.InstallDir, DefaultBinary), plan.BinaryPath)
	require.True(t, plan.DryRun)
}

func TestResolvePlan_acceptsVersionAndInstallDir(t *testing.T) {
	installDir := t.TempDir()
	plan, err := ResolvePlan(Config{
		Version:    "@main",
		InstallDir: installDir,
		BinaryName: "mars-harness-dev",
	})
	require.NoError(t, err)

	require.Equal(t, "main", plan.Version)
	require.Equal(t, []string{"go", "install", DefaultPackage + "@main"}, plan.Command)
	require.Equal(t, installDir, plan.InstallDir)
	require.Equal(t, filepath.Join(installDir, "mars-harness-dev"), plan.BinaryPath)
}
