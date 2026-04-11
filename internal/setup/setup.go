package setup

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"

	"github.com/greaveselliott/mars-harness/internal/config"
	"github.com/greaveselliott/mars-harness/internal/hardware"
	"gopkg.in/yaml.v3"
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
		steps = append(steps, createModelsDirStep(baseDir))
	}

	if !cfg.SkipGitHub && !cfg.TestMode {
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

func createModelsDirStep(baseDir string) Step {
	modelsDir := filepath.Join(baseDir, "models")
	readmePath := filepath.Join(modelsDir, "README.md")
	return Step{
		Name: "prepare-models",
		Check: func() (bool, error) {
			_, err := os.Stat(readmePath)
			return err == nil, nil
		},
		Execute: func() error {
			if err := os.MkdirAll(modelsDir, 0o755); err != nil {
				return fmt.Errorf("create models dir: %w — check directory permissions", err)
			}
			readme := "# Models\n\nModel GGUF files are downloaded here by `mars-harness setup`.\n"
			return os.WriteFile(readmePath, []byte(readme), 0o644)
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
