/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/delivery-operating-model.md
- docs/design-docs/guardrails.md
- docs/design-docs/tools-glossary.md
- docs/features/F-005-agent-execution-runtime.md
- docs/features/F-007-guardrails-and-safety.md
*/
package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

const dependencySyncSchema = `{
  "type": "object",
  "additionalProperties": false,
  "properties": {
    "action": { "type": "string", "description": "Dependency action: install or fetch." },
    "frozen": { "type": "boolean", "description": "Whether to require reproducible lockfile-respecting dependency sync." },
    "package_manager": { "type": "string", "description": "Package manager to use, or auto to detect from lockfiles and manifests." },
    "reason": { "type": "string", "description": "Required rationale when frozen is false or when changing dependency state intentionally." }
  },
  "required": []
}`

type dependencySyncArgs struct {
	Action         string `json:"action"`
	Frozen         *bool  `json:"frozen"`
	PackageManager string `json:"package_manager"`
	Reason         string `json:"reason"`
}

func registerDependencySync(r *Registry) error {
	return r.Register("dependency_sync", "Run package-manager dependency sync through deterministic workspace hygiene gates.", json.RawMessage(dependencySyncSchema), handleDependencySync)
}

func handleDependencySync(ctx context.Context, root Root, raw json.RawMessage) (ToolResult, error) {
	var args dependencySyncArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return ToolResult{}, fmt.Errorf("dependency_sync: parse arguments: %w", err)
	}
	report, err := RunDependencySync(ctx, root, args)
	if err != nil {
		out, _ := json.MarshalIndent(report, "", "  ")
		return ToolResult{Output: string(out), ExitCode: report.ExitCode}, err
	}
	out, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return ToolResult{}, fmt.Errorf("dependency_sync: marshal report: %w", err)
	}
	return ToolResult{Output: string(out), ExitCode: report.ExitCode}, nil
}

// DependencySyncReport captures the command run and deterministic hygiene result.
type DependencySyncReport struct {
	Action         string                        `json:"action"`
	PackageManager string                        `json:"package_manager"`
	Frozen         bool                          `json:"frozen"`
	Command        []string                      `json:"command,omitempty"`
	ExitCode       int                           `json:"exit_code"`
	Stdout         string                        `json:"stdout,omitempty"`
	Stderr         string                        `json:"stderr,omitempty"`
	Repair         *WorkspaceHygieneRepairResult `json:"repair,omitempty"`
	Hygiene        WorkspaceHygieneReport        `json:"hygiene"`
	Message        string                        `json:"message"`
}

type dependencyCommand struct {
	Manager string
	Action  string
	Frozen  bool
	Args    []string
}

// RunDependencySync executes dependency setup through workspace hygiene gates.
func RunDependencySync(ctx context.Context, root Root, args dependencySyncArgs) (DependencySyncReport, error) {
	action := strings.TrimSpace(strings.ToLower(args.Action))
	if action == "" {
		action = "install"
	}
	if action != "install" && action != "fetch" {
		return DependencySyncReport{Action: action, Message: "unsupported dependency action"}, fmt.Errorf("dependency_sync: action must be install or fetch")
	}
	manager := strings.TrimSpace(strings.ToLower(args.PackageManager))
	if manager == "" || manager == "auto" {
		manager = detectDependencyPackageManager(root)
	}
	if manager == "" {
		return DependencySyncReport{Action: action, Message: "no supported package manifest found"}, fmt.Errorf("dependency_sync: no supported package manifest found")
	}
	frozen := dependencyFrozenDefault(root, manager)
	if args.Frozen != nil {
		frozen = *args.Frozen
	}
	if args.Frozen != nil && !frozen && strings.TrimSpace(args.Reason) == "" {
		return DependencySyncReport{Action: action, PackageManager: manager, Frozen: frozen, Message: "reason is required when frozen is false"}, fmt.Errorf("dependency_sync: reason is required when frozen is false")
	}

	report := DependencySyncReport{Action: action, PackageManager: manager, Frozen: frozen}
	repair, err := RepairWorkspaceHygieneIgnorePolicy(ctx, root)
	if err != nil {
		report.Message = err.Error()
		return report, err
	}
	if repair.Changed || repair.Committed {
		report.Repair = &repair
	}

	pre, err := AuditWorkspaceHygiene(ctx, root, WorkspaceHygieneOptions{Mode: workspaceHygieneModePreDependency})
	if err != nil {
		report.Hygiene = pre
		return report, err
	}
	report.Hygiene = pre
	if pre.Blocking {
		report.Message = pre.Message
		return report, fmt.Errorf("dependency_sync: workspace hygiene preflight blocked: %s", pre.Message)
	}

	cmd, err := dependencySyncCommand(root, manager, action, frozen)
	if err != nil {
		report.Message = err.Error()
		return report, err
	}
	report.Command = append([]string{cmd.Manager}, cmd.Args...)

	execCmd := exec.CommandContext(ctx, cmd.Manager, cmd.Args...)
	execCmd.Dir = root.Abs()
	var stdout, stderr bytes.Buffer
	execCmd.Stdout = &stdout
	execCmd.Stderr = &stderr
	runErr := execCmd.Run()
	report.Stdout, _ = capDependencySyncText(stdout.String())
	report.Stderr, _ = capDependencySyncText(stderr.String())
	if runErr != nil {
		if ee, ok := runErr.(*exec.ExitError); ok {
			report.ExitCode = ee.ExitCode()
			report.Message = "dependency command failed"
			return report, fmt.Errorf("dependency_sync: command failed with exit %d", report.ExitCode)
		}
		report.ExitCode = -1
		report.Message = runErr.Error()
		return report, fmt.Errorf("dependency_sync: run %s: %w", cmd.Manager, runErr)
	}

	post, err := AuditWorkspaceHygiene(ctx, root, WorkspaceHygieneOptions{Mode: workspaceHygieneModePostDependency})
	if err != nil {
		return report, err
	}
	report.Hygiene = post
	if post.Blocking {
		report.Message = post.Message
		return report, fmt.Errorf("dependency_sync: workspace hygiene postflight blocked: %s", post.Message)
	}
	report.Message = "dependency sync completed"
	return report, nil
}

func detectDependencyPackageManager(root Root) string {
	checks := []struct {
		file    string
		manager string
	}{
		{"pnpm-lock.yaml", "pnpm"},
		{"yarn.lock", "yarn"},
		{"bun.lockb", "bun"},
		{"bun.lock", "bun"},
		{"package-lock.json", "npm"},
		{"package.json", "npm"},
		{"go.mod", "go"},
		{"Cargo.toml", "cargo"},
		{"requirements.txt", "pip"},
		{"pyproject.toml", "pip"},
		{"Gemfile", "bundle"},
		{"composer.json", "composer"},
	}
	for _, check := range checks {
		if repoFileExists(root, check.file) {
			return check.manager
		}
	}
	return ""
}

func dependencyFrozenDefault(root Root, manager string) bool {
	switch manager {
	case "npm":
		return repoFileExists(root, "package-lock.json")
	case "pnpm":
		return repoFileExists(root, "pnpm-lock.yaml")
	case "yarn":
		return repoFileExists(root, "yarn.lock")
	case "bun":
		return repoFileExists(root, "bun.lockb") || repoFileExists(root, "bun.lock")
	case "cargo":
		return repoFileExists(root, "Cargo.lock")
	case "composer":
		return repoFileExists(root, "composer.lock")
	case "bundle":
		return repoFileExists(root, "Gemfile.lock")
	default:
		return false
	}
}

func dependencySyncCommand(root Root, manager, action string, frozen bool) (dependencyCommand, error) {
	switch manager {
	case "npm":
		if action == "fetch" {
			action = "install"
		}
		if frozen {
			if !repoFileExists(root, "package-lock.json") {
				return dependencyCommand{}, fmt.Errorf("dependency_sync: npm frozen install requires package-lock.json; pass frozen:false with reason to create or update lockfile")
			}
			return dependencyCommand{Manager: "npm", Action: action, Frozen: frozen, Args: []string{"ci"}}, nil
		}
		return dependencyCommand{Manager: "npm", Action: action, Frozen: frozen, Args: []string{"install"}}, nil
	case "pnpm":
		args := []string{"install"}
		if action == "fetch" {
			args = []string{"fetch"}
		} else if frozen {
			args = append(args, "--frozen-lockfile")
		}
		return dependencyCommand{Manager: "pnpm", Action: action, Frozen: frozen, Args: args}, nil
	case "yarn":
		args := []string{"install"}
		if frozen {
			args = append(args, "--frozen-lockfile")
		}
		return dependencyCommand{Manager: "yarn", Action: action, Frozen: frozen, Args: args}, nil
	case "bun":
		args := []string{"install"}
		if frozen {
			args = append(args, "--frozen-lockfile")
		}
		return dependencyCommand{Manager: "bun", Action: action, Frozen: frozen, Args: args}, nil
	case "go":
		return dependencyCommand{Manager: "go", Action: "fetch", Frozen: true, Args: []string{"mod", "download"}}, nil
	case "cargo":
		args := []string{"fetch"}
		if frozen {
			args = append(args, "--locked")
		}
		return dependencyCommand{Manager: "cargo", Action: "fetch", Frozen: frozen, Args: args}, nil
	case "pip":
		if repoFileExists(root, "requirements.txt") {
			return dependencyCommand{Manager: "python", Action: action, Frozen: frozen, Args: []string{"-m", "pip", "install", "-r", "requirements.txt"}}, nil
		}
		return dependencyCommand{}, fmt.Errorf("dependency_sync: pip requires requirements.txt for deterministic sync")
	case "bundle":
		return dependencyCommand{Manager: "bundle", Action: action, Frozen: frozen, Args: []string{"install"}}, nil
	case "composer":
		args := []string{"install"}
		if frozen {
			args = append(args, "--no-update")
		}
		return dependencyCommand{Manager: "composer", Action: action, Frozen: frozen, Args: args}, nil
	default:
		return dependencyCommand{}, fmt.Errorf("dependency_sync: unsupported package manager %q", manager)
	}
}

func capDependencySyncText(value string) (string, bool) {
	return capString(value, DefaultMaxToolOutputBytes/4)
}
