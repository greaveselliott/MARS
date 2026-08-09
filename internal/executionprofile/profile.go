/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/tools-glossary.md
- docs/product-specs/product-surface.md
- docs/features/F-005-agent-execution-runtime.md
- docs/features/F-017-open-source-publication.md
*/
package executionprofile

import (
	"fmt"
	"strings"
)

// Profile is the operator-selected boundary for agent execution.
type Profile string

const (
	Observer Profile = "observer"
	Host     Profile = "host"
	Isolated Profile = "isolated"
)

// Admit validates an execution profile before a command creates runtime state
// or starts a subprocess. The isolated profile is reserved until MARS has an
// enforceable isolation adapter.
func Admit(command, raw string, acknowledgeHost bool) (Profile, error) {
	profile, err := Parse(raw)
	if err != nil {
		return "", fmt.Errorf("%s: %w", command, err)
	}
	switch profile {
	case Isolated:
		return "", fmt.Errorf("%s: execution profile %q is unavailable because MARS has no enforceable isolation adapter; use --execution-profile observer for read-only operation, or use --execution-profile host --acknowledge-host-execution only if you accept current-user host authority", command, profile)
	case Host:
		if !acknowledgeHost {
			return "", fmt.Errorf("%s: execution profile %q requires --acknowledge-host-execution before target mutation or model-controlled host execution; acknowledged host execution has the current OS user's filesystem, network, process, keychain, and credential authority and is not containment", command, profile)
		}
	}
	return profile, nil
}

// Parse normalizes an execution profile. An omitted value defaults to observer.
func Parse(raw string) (Profile, error) {
	switch Profile(strings.ToLower(strings.TrimSpace(raw))) {
	case "", Observer:
		return Observer, nil
	case Host:
		return Host, nil
	case Isolated:
		return Isolated, nil
	default:
		return "", fmt.Errorf("execution profile %q is invalid; choose observer, host, or isolated", raw)
	}
}

// RequireTargetMutation rejects command-owned target writes outside host mode.
func (p Profile) RequireTargetMutation(operation string) error {
	if p == Host {
		return nil
	}
	if p == "" {
		p = Observer
	}
	operation = strings.TrimSpace(operation)
	if operation == "" {
		operation = "this target mutation"
	}
	return fmt.Errorf("execution profile %q blocks %s; rerun with --execution-profile host --acknowledge-host-execution only if current-user host authority is intended", p, operation)
}
