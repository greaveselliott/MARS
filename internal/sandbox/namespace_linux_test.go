//go:build linux

/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/guardrails.md
- docs/features/F-007-guardrails-and-safety.md
*/
package sandbox

import (
	"context"
	"testing"
)

func TestRun_linuxNamespaceDisableEnvUsesUlimitFallback(t *testing.T) {
	t.Setenv("MARS_DISABLE_LINUX_NAMESPACES", "true")

	cmd, err := Run(context.Background(), Config{WorkDir: "/tmp"}, "true")
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if cmd.SysProcAttr == nil {
		t.Fatal("expected SysProcAttr to be set")
	}
	if cmd.SysProcAttr.Cloneflags != 0 {
		t.Fatalf("expected namespace clone flags to be disabled, got %#x", cmd.SysProcAttr.Cloneflags)
	}
}

func TestLinuxNamespacesDisabledByEnv(t *testing.T) {
	for _, value := range []string{"1", "true", "TRUE", "yes", "on"} {
		t.Setenv("MARS_DISABLE_LINUX_NAMESPACES", value)
		if !linuxNamespacesDisabledByEnv() {
			t.Fatalf("expected %q to disable Linux namespaces", value)
		}
	}

	t.Setenv("MARS_DISABLE_LINUX_NAMESPACES", "0")
	if linuxNamespacesDisabledByEnv() {
		t.Fatal("expected 0 to leave Linux namespaces enabled")
	}
}
