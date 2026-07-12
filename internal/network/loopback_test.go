/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/dashboard.md
- docs/design-docs/github-app-integration.md
- docs/features/F-017-open-source-publication.md
- docs/product-specs/product-surface.md
*/
package network

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateLoopbackAddress(t *testing.T) {
	t.Parallel()
	for _, addr := range []string{"127.0.0.1:9091", "localhost:9091", "LOCALHOST:0", "[::1]:9091"} {
		addr := addr
		t.Run("accept_"+addr, func(t *testing.T) {
			t.Parallel()
			require.NoError(t, ValidateLoopbackAddress("webhook", addr))
		})
	}
	for _, addr := range []string{":9091", "0.0.0.0:9091", "[::]:9091", "192.168.1.5:9091", "mars.local:9091", "127.0.0.1"} {
		addr := addr
		t.Run("reject_"+addr, func(t *testing.T) {
			t.Parallel()
			err := ValidateLoopbackAddress("webhook", addr)
			require.Error(t, err)
			require.Contains(t, err.Error(), "loopback")
		})
	}
}
