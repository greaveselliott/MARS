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
	"strconv"
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

// ValidateLiteralLoopbackAddress accepts only a literal loopback IP and a
// concrete TCP port. It deliberately rejects localhost and other DNS names for
// entry points whose request Host must match the configured listener exactly.
func ValidateLiteralLoopbackAddress(surface, addr string) error {
	host, portText, err := net.SplitHostPort(strings.TrimSpace(addr))
	if err != nil {
		return literalLoopbackError(surface)
	}
	ip := net.ParseIP(strings.TrimSpace(host))
	port, err := strconv.Atoi(portText)
	if err != nil || port < 1 || port > 65535 || ip == nil || !ip.IsLoopback() {
		return literalLoopbackError(surface)
	}
	return nil
}

func literalLoopbackError(surface string) error {
	return fmt.Errorf("%s must use a literal loopback IP and TCP port such as 127.0.0.1:9092 or [::1]:9092", surface)
}
