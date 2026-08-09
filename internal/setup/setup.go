/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/local-inference.md
- docs/design-docs/release-versioning.md
- docs/features/F-002-zero-config-shell-path.md
- docs/features/F-003-local-inference-lifecycle.md
- docs/features/F-009-release-update-lifecycle.md
*/
package setup

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/greaveselliott/mars/internal/config"
	"github.com/greaveselliott/mars/internal/githubauth"
	"github.com/greaveselliott/mars/internal/hardware"
	"github.com/greaveselliott/mars/internal/models"
	"github.com/greaveselliott/mars/internal/shellpath"
	"gopkg.in/yaml.v3"
)

const (
	httpTimeout     = 30 * time.Second
	downloadTimeout = 30 * time.Minute
)

// Step represents a single idempotent setup action.
type Step struct {
	Name    string
	Check   func() (bool, error)
	Execute func() error
}

// Config controls setup behaviour.
type Config struct {
	SkipDownload bool
	SkipGitHub   bool
	EnableGitHub bool
	TestMode     bool
	DryRun       bool
	InstallDir   string
	Inference    string
	LocalBundle  string
}

// Result reports what happened during setup.
type Result struct {
	StepsRun     int
	StepsSkipped int
	Errors       []string
}

// Run executes the first-time setup wizard. Each step is idempotent:
// the Check func returns true if the step is already satisfied.
func Run(cfg Config) (*Result, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("setup: cannot determine home directory: %w — set $HOME and retry", err)
	}
	baseDir := filepath.Join(home, ".mars")

	steps := buildSteps(baseDir, cfg)
	result := &Result{}

	for _, step := range steps {
		if cfg.DryRun {
			slog.Info("dry-run: would execute step", "step", step.Name)
			result.StepsSkipped++
			continue
		}

		done, checkErr := step.Check()
		if checkErr != nil {
			slog.Warn("step check failed, will execute", "step", step.Name, "err", checkErr)
		}
		if done {
			slog.Info("step already satisfied", "step", step.Name)
			result.StepsSkipped++
			continue
		}

		slog.Info("executing step", "step", step.Name)
		if execErr := step.Execute(); execErr != nil {
			msg := fmt.Sprintf("step %q failed: %v", step.Name, execErr)
			slog.Error(msg)
			result.Errors = append(result.Errors, msg)
			return result, fmt.Errorf("setup: %s", msg)
		}
		result.StepsRun++
		slog.Info("step complete", "step", step.Name)
	}

	slog.Info("setup complete", "steps_run", result.StepsRun, "steps_skipped", result.StepsSkipped)
	return result, nil
}

func buildSteps(baseDir string, cfg Config) []Step {
	inferenceMode := strings.ToLower(strings.TrimSpace(cfg.Inference))
	if inferenceMode == "" {
		inferenceMode = models.RoutingLocal
	}
	steps := []Step{
		createDirectoriesStep(baseDir),
		writeDefaultConfigStep(baseDir),
		detectHardwareStep(baseDir),
		configureShellPathStep(cfg),
	}

	if !cfg.SkipGitHub && !cfg.TestMode {
		steps = append(steps, githubPrivateReleaseAuthStep())
	}

	if inferenceMode != models.RoutingLocal {
		cfg.SkipDownload = true
	}
	if !cfg.SkipDownload && !cfg.TestMode {
		steps = append(steps, installLlamaServerStep(baseDir))
		steps = append(steps, downloadModelsStep(baseDir, cfg.LocalBundle))
	}

	if cfg.EnableGitHub && !cfg.SkipGitHub && !cfg.TestMode {
		steps = append(steps, githubPlaceholderStep(baseDir))
	}

	return steps
}

func githubPrivateReleaseAuthStep() Step {
	return Step{
		Name: "github-private-release-auth",
		Check: func() (bool, error) {
			report := githubauth.Check(context.Background(), githubauth.Options{})
			return report.Status == githubauth.StatusOK, nil
		},
		Execute: func() error {
			report := githubauth.Check(context.Background(), githubauth.Options{})
			if report.Status == githubauth.StatusOK {
				slog.Info("setup: GitHub private release auth ready", "auth_source", report.AuthSource)
				return nil
			}
			return fmt.Errorf("%s — %s", report.Message, report.NextAction)
		},
	}
}

func configureShellPathStep(cfg Config) Step {
	return Step{
		Name: "configure-shell-path",
		Check: func() (bool, error) {
			result, err := shellpath.Evaluate(shellpath.Config{InstallDir: cfg.InstallDir})
			if err != nil {
				return false, err
			}
			return result.UnsupportedShell || result.ProfileAlreadyConfigured, nil
		},
		Execute: func() error {
			result, err := shellpath.Ensure(shellpath.Config{InstallDir: cfg.InstallDir})
			if err != nil {
				return err
			}
			if result.UnsupportedShell {
				slog.Warn("setup: shell PATH unsupported", "shell", result.Shell, "install_dir", result.InstallDir)
				return nil
			}
			slog.Info("setup: shell PATH ready", "profile", result.ProfilePath, "changed", result.Changed, "hint", result.ReloadHint)
			return nil
		},
	}
}

func createDirectoriesStep(baseDir string) Step {
	dirs := []string{
		baseDir,
		filepath.Join(baseDir, "models"),
		filepath.Join(baseDir, "bin"),
		filepath.Join(baseDir, "traces"),
		filepath.Join(baseDir, "db"),
	}
	return Step{
		Name: "create-directories",
		Check: func() (bool, error) {
			for _, d := range dirs {
				if _, err := os.Stat(d); os.IsNotExist(err) {
					return false, nil
				}
			}
			return true, nil
		},
		Execute: func() error {
			for _, d := range dirs {
				if err := os.MkdirAll(d, 0o755); err != nil {
					return fmt.Errorf("create %s: %w — check directory permissions", d, err)
				}
				slog.Debug("created directory", "path", d)
			}
			return nil
		},
	}
}

func writeDefaultConfigStep(baseDir string) Step {
	cfgPath := filepath.Join(baseDir, "config.yaml")
	return Step{
		Name: "write-config",
		Check: func() (bool, error) {
			_, err := os.Stat(cfgPath)
			return err == nil, nil
		},
		Execute: func() error {
			cfg := config.Config{
				ModelsDir:           filepath.Join(baseDir, "models"),
				BinDir:              filepath.Join(baseDir, "bin"),
				TracesDir:           filepath.Join(baseDir, "traces"),
				LogFormat:           "text",
				WebhookPort:         9091,
				DashboardPort:       9090,
				PerformanceProfile:  "auto",
				LlamaParallel:       1,
				LlamaFlashAttention: "auto",
				Telemetry: config.TelemetryConfig{
					Reporting:      "off",
					ReportInterval: "24h",
				},
			}
			data, err := yaml.Marshal(&cfg)
			if err != nil {
				return fmt.Errorf("marshal config: %w", err)
			}
			return os.WriteFile(cfgPath, data, 0o644)
		},
	}
}

func detectHardwareStep(baseDir string) Step {
	hwPath := filepath.Join(baseDir, "hardware.yaml")
	return Step{
		Name: "detect-hardware",
		Check: func() (bool, error) {
			_, err := os.Stat(hwPath)
			return err == nil, nil
		},
		Execute: func() error {
			hw := hardware.Detect()
			out := hardwareSnapshot{
				Profile:  string(hw.Profile),
				GPUCount: len(hw.GPUs),
				RAMMiB:   hw.RAMMiB,
				CPUCores: hw.CPUCores,
				OS:       runtime.GOOS,
				Arch:     runtime.GOARCH,
			}
			data, err := yaml.Marshal(&out)
			if err != nil {
				return fmt.Errorf("marshal hardware: %w", err)
			}
			return os.WriteFile(hwPath, data, 0o644)
		},
	}
}

type hardwareSnapshot struct {
	Profile  string `yaml:"profile"`
	GPUCount int    `yaml:"gpu_count"`
	RAMMiB   int    `yaml:"ram_mib"`
	CPUCores int    `yaml:"cpu_cores"`
	OS       string `yaml:"os"`
	Arch     string `yaml:"arch"`
}

// downloadModelsStep detects hardware, selects models for the profile, and downloads
// each unique GGUF from HuggingFace with resume support. Idempotent: re-running
// skips models whose files already exist in the models directory.
func downloadModelsStep(baseDir, localBundle string) Step {
	modelsDir := filepath.Join(baseDir, "models")
	markerPath := filepath.Join(modelsDir, ".download-complete")

	return Step{
		Name: "download-models",
		Check: func() (bool, error) {
			if _, err := os.Stat(markerPath); err != nil {
				return false, nil
			}
			hw := hardware.Detect()
			bundle, _, err := models.ResolveLocalBundle(hw, localBundle)
			if err != nil {
				return false, err
			}
			unique := hardware.UniqueModels(bundle.Models)
			if err := validateDownloadModelProvenance(unique); err != nil {
				return false, err
			}
			for _, spec := range unique {
				if _, err := os.Stat(filepath.Join(modelsDir, spec.File)); err != nil {
					return false, nil
				}
			}
			return true, nil
		},
		Execute: func() error {
			hw := hardware.Detect()
			bundle, _, err := models.ResolveLocalBundle(hw, localBundle)
			if err != nil {
				return err
			}
			unique := hardware.UniqueModels(bundle.Models)
			if err := validateDownloadModelProvenance(unique); err != nil {
				return err
			}
			if err := os.MkdirAll(modelsDir, 0o755); err != nil {
				return fmt.Errorf("create models dir: %w — check directory permissions", err)
			}

			slog.Info("model download plan",
				"profile", string(hw.Profile),
				"local_bundle", bundle.ID,
				"models_to_download", len(unique),
			)

			for i, spec := range unique {
				destPath := filepath.Join(modelsDir, spec.File)
				if _, err := os.Stat(destPath); err == nil {
					slog.Info("model already present, skipping",
						"file", spec.File,
						"index", fmt.Sprintf("%d/%d", i+1, len(unique)),
					)
					continue
				}

				url := spec.DownloadURL()
				if url == "" {
					return fmt.Errorf("no download URL for model %s — check registry configuration", spec.Name)
				}

				slog.Info("downloading model",
					"name", spec.Name,
					"quant", spec.Quant,
					"file", spec.File,
					"url", url,
					"index", fmt.Sprintf("%d/%d", i+1, len(unique)),
				)

				started := time.Now()
				_, err := models.Download(context.Background(), models.DownloadConfig{
					URL:      url,
					DestDir:  modelsDir,
					Filename: spec.File,
					SHA256:   spec.SHA256,
					OnProgress: func(downloaded, total int64) {
						if total > 0 {
							pct := float64(downloaded) / float64(total) * 100
							slog.Info("download progress",
								"file", spec.File,
								"percent", fmt.Sprintf("%.1f%%", pct),
								"downloaded_mb", downloaded/(1024*1024),
								"total_mb", total/(1024*1024),
							)
						}
					},
				})
				if err != nil {
					return fmt.Errorf("download %s: %w — check network connectivity and disk space", spec.File, err)
				}

				elapsed := time.Since(started)
				slog.Info("model downloaded",
					"file", spec.File,
					"elapsed", elapsed.Round(time.Second).String(),
				)
			}

			return os.WriteFile(markerPath, []byte("ok\n"), 0o644)
		},
	}
}

func validateDownloadModelProvenance(specs []hardware.ModelSpec) error {
	if len(specs) == 0 {
		return fmt.Errorf("model download has no artifacts — update the model registry or run setup with --skip-download")
	}
	for _, spec := range specs {
		if err := spec.ValidateProvenance(); err != nil {
			return fmt.Errorf("model %s has incomplete provenance: %w — update the model registry or run setup with --skip-download", spec.Name, err)
		}
	}
	return nil
}

func githubPlaceholderStep(baseDir string) Step {
	markerPath := filepath.Join(baseDir, ".github-configured")
	return Step{
		Name: "github-setup",
		Check: func() (bool, error) {
			_, err := os.Stat(markerPath)
			return err == nil, nil
		},
		Execute: func() error {
			slog.Info("GitHub App setup: configure via 'mars setup --github' or set MARS_GITHUB_TOKEN")
			return os.WriteFile(markerPath, []byte("pending\n"), 0o644)
		},
	}
}
