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
	"fmt"
	"net"
	"strings"
)

// ValidateLoopbackAddress accepts only an explicit loopback host and TCP port.
// Hostnames other than literal localhost are rejected without DNS resolution so
// a later DNS change cannot widen the control-plane bind boundary.
func ValidateLoopbackAddress(surface, addr string) error {
	host, port, err := net.SplitHostPort(strings.TrimSpace(addr))
	if err != nil || strings.TrimSpace(port) == "" {
		return fmt.Errorf("%s address %q is invalid; use an explicit loopback address such as 127.0.0.1:9091", surface, addr)
	}
	host = strings.TrimSpace(host)
	if strings.EqualFold(host, "localhost") {
		return nil
	}
	ip := net.ParseIP(host)
	if ip != nil && ip.IsLoopback() {
		return nil
	}
	return fmt.Errorf("%s address %q is not loopback; bind MARS to 127.0.0.1 or [::1] and place any authenticated remote gateway or reverse proxy in front of it", surface, addr)
}
