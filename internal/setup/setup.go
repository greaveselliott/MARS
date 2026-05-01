package setup

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/greaveselliott/mars-harness/internal/config"
	"github.com/greaveselliott/mars-harness/internal/hardware"
	"github.com/greaveselliott/mars-harness/internal/models"
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
	baseDir := filepath.Join(home, ".mars-harness")

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
	steps := []Step{
		createDirectoriesStep(baseDir),
		writeDefaultConfigStep(baseDir),
		detectHardwareStep(baseDir),
	}

	if !cfg.SkipDownload && !cfg.TestMode {
		steps = append(steps, installLlamaServerStep(baseDir))
		steps = append(steps, downloadModelsStep(baseDir))
	}

	if cfg.EnableGitHub && !cfg.SkipGitHub && !cfg.TestMode {
		steps = append(steps, githubPlaceholderStep(baseDir))
	}

	return steps
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
				ModelsDir:     filepath.Join(baseDir, "models"),
				BinDir:        filepath.Join(baseDir, "bin"),
				TracesDir:     filepath.Join(baseDir, "traces"),
				LogFormat:     "text",
				WebhookPort:   9091,
				DashboardPort: 9090,
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
func downloadModelsStep(baseDir string) Step {
	modelsDir := filepath.Join(baseDir, "models")
	markerPath := filepath.Join(modelsDir, ".download-complete")

	return Step{
		Name: "download-models",
		Check: func() (bool, error) {
			_, err := os.Stat(markerPath)
			return err == nil, nil
		},
		Execute: func() error {
			if err := os.MkdirAll(modelsDir, 0o755); err != nil {
				return fmt.Errorf("create models dir: %w — check directory permissions", err)
			}

			hw := hardware.Detect()
			modelSet := hardware.DefaultModels(hw.Profile)
			unique := hardware.UniqueModels(modelSet)

			slog.Info("model download plan",
				"profile", string(hw.Profile),
				"models_to_download", len(unique),
			)

			for i, spec := range unique {
				if spec.Revision == "" || spec.SHA256 == "" {
					return fmt.Errorf("model %s is not pinned with both revision and SHA256 — update the model registry or run setup with --skip-download for local-only configuration", spec.Name)
				}
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

func githubPlaceholderStep(baseDir string) Step {
	markerPath := filepath.Join(baseDir, ".github-configured")
	return Step{
		Name: "github-setup",
		Check: func() (bool, error) {
			_, err := os.Stat(markerPath)
			return err == nil, nil
		},
		Execute: func() error {
			slog.Info("GitHub App setup: configure via 'mars-harness setup --github' or set MARS_HARNESS_GITHUB_TOKEN")
			return os.WriteFile(markerPath, []byte("pending\n"), 0o644)
		},
	}
}
