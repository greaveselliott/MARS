/*
MarsDocSync:
docs:
- docs/configuration-reference.html
- docs/design-docs/code-documentation-map.md
- docs/features/F-005-agent-execution-runtime.md
- docs/product-specs/product-surface.md
*/
// Package childenv builds inherited environments for MARS-managed children.
package childenv

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"unicode"
)

// AllowlistVariable is read only from the MARS parent environment and is never
// propagated to a child. Its comma- or whitespace-separated names explicitly
// restore variables that the default name filter would remove.
const AllowlistVariable = "MARS_CHILD_ENV_ALLOWLIST"

// Current returns the current process environment after name-based filtering.
func Current() ([]string, error) {
	return Filter(os.Environ())
}

// Filter preserves ordinary inherited variables, removes credential-like
// names, and restores names explicitly selected by AllowlistVariable. Values
// are intentionally opaque: this boundary does not scan or rewrite them.
func Filter(parent []string) ([]string, error) {
	allowed, err := ownerAllowlist(parent)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(parent))
	for _, entry := range parent {
		name, ok := entryName(entry)
		if !ok || name == AllowlistVariable {
			continue
		}
		if sensitiveName(name) && !allowed[name] {
			continue
		}
		out = append(out, entry)
	}
	return out, nil
}

// Apply sets cmd.Env to the filtered current process environment.
func Apply(cmd *exec.Cmd) error {
	if cmd == nil {
		return fmt.Errorf("child environment: command is nil")
	}
	env, err := Current()
	if err != nil {
		return err
	}
	cmd.Env = env
	return nil
}

// ApplyWith applies the filtered current environment and then sets explicit
// code-owned overrides, replacing any inherited entry with the same name.
func ApplyWith(cmd *exec.Cmd, overrides ...string) error {
	if err := Apply(cmd); err != nil {
		return err
	}
	for _, override := range overrides {
		name, ok := entryName(override)
		if !ok {
			return fmt.Errorf("child environment: invalid explicit override %q", override)
		}
		cmd.Env = withoutName(cmd.Env, name)
		cmd.Env = append(cmd.Env, override)
	}
	return nil
}

// OwnerAllows reports whether the current parent explicitly names a variable
// in AllowlistVariable.
func OwnerAllows(name string) (bool, error) {
	allowed, err := ownerAllowlist(os.Environ())
	if err != nil {
		return false, err
	}
	return allowed[name], nil
}

func ownerAllowlist(parent []string) (map[string]bool, error) {
	raw := ""
	for _, entry := range parent {
		name, ok := entryName(entry)
		if ok && name == AllowlistVariable {
			raw = strings.TrimPrefix(entry, name+"=")
		}
	}
	allowed := map[string]bool{}
	for _, name := range strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || unicode.IsSpace(r)
	}) {
		name = strings.TrimSpace(name)
		if !validName(name) {
			return nil, fmt.Errorf("child environment: %s contains invalid variable name %q; use comma-separated environment variable names", AllowlistVariable, name)
		}
		if name != AllowlistVariable {
			allowed[name] = true
		}
	}
	return allowed, nil
}

func entryName(entry string) (string, bool) {
	idx := strings.IndexByte(entry, '=')
	if idx <= 0 {
		return "", false
	}
	name := entry[:idx]
	return name, validName(name)
}

func validName(name string) bool {
	if name == "" {
		return false
	}
	for i, r := range name {
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || r == '_' || (i > 0 && r >= '0' && r <= '9') {
			continue
		}
		return false
	}
	return true
}

func sensitiveName(name string) bool {
	upper := strings.ToUpper(name)
	compact := strings.NewReplacer("_", "", "-", "").Replace(upper)
	for _, segment := range strings.FieldsFunc(upper, func(r rune) bool {
		return r == '_' || r == '-'
	}) {
		if segment == "AUTH" {
			return true
		}
	}
	for _, fragment := range []string{
		"AUTHORIZATION", "BEARER", "TOKEN", "SECRET", "PASSWORD", "PASSWD",
		"APIKEY", "PRIVATEKEY", "CREDENTIAL", "ACCESSKEY",
	} {
		if strings.Contains(compact, fragment) {
			return true
		}
	}
	for _, prefix := range []string{
		"MARS_", "GITHUB_", "GH_", "AWS_", "AZURE_", "GCP_", "GOOGLE_",
		"CLOUDSDK_", "OPENAI_", "ANTHROPIC_", "COHERE_", "MISTRAL_",
		"GEMINI_", "XAI_", "DEEPSEEK_", "GROQ_", "HUGGINGFACE_",
		"HUGGING_FACE_", "HF_", "OCI_",
		"CLOUDFLARE_", "VERCEL_", "NETLIFY_", "HEROKU_", "SSH_",
	} {
		if strings.HasPrefix(upper, prefix) {
			return true
		}
	}
	for _, exact := range []string{
		"MARS", "AUTH", "KUBECONFIG", "DOCKER_AUTH_CONFIG", "GIT_ASKPASS",
		"GIT_SSH", "GIT_SSH_COMMAND", "GIT_TERMINAL_PROMPT",
	} {
		if upper == exact {
			return true
		}
	}
	return strings.HasPrefix(upper, "GIT_CONFIG_")
}

func withoutName(env []string, name string) []string {
	out := env[:0]
	for _, entry := range env {
		entryName, ok := entryName(entry)
		if !ok || entryName == name {
			continue
		}
		out = append(out, entry)
	}
	return out
}
