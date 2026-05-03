package models

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

const DefaultOllamaEndpoint = "http://127.0.0.1:11434/v1"

// OllamaModel is one model row from `ollama list`.
type OllamaModel struct {
	Name     string `json:"name"`
	ID       string `json:"id,omitempty"`
	Size     string `json:"size,omitempty"`
	Modified string `json:"modified,omitempty"`
}

// CommandRunner abstracts shell commands for provider catalog tests.
type CommandRunner interface {
	CombinedOutput(ctx context.Context, name string, args ...string) ([]byte, error)
}

type shellRunner struct{}

func (shellRunner) CombinedOutput(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	return cmd.CombinedOutput()
}

// ListOllamaModels returns locally installed Ollama models.
func ListOllamaModels(ctx context.Context, runner CommandRunner) ([]OllamaModel, error) {
	if runner == nil {
		runner = shellRunner{}
	}
	out, err := runner.CombinedOutput(ctx, "ollama", "list")
	if err != nil {
		detail := strings.TrimSpace(string(out))
		if detail != "" {
			return nil, fmt.Errorf("models: cannot list Ollama models with `ollama list`: %w: %s — install/start Ollama or run `ollama pull <model>` first", err, detail)
		}
		return nil, fmt.Errorf("models: cannot list Ollama models with `ollama list`: %w — install/start Ollama or run `ollama pull <model>` first", err)
	}
	return ParseOllamaList(string(out)), nil
}

// ParseOllamaList parses Ollama's tabular list output.
func ParseOllamaList(output string) []OllamaModel {
	var models []OllamaModel
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		if strings.EqualFold(fields[0], "NAME") {
			continue
		}
		model := OllamaModel{Name: fields[0], ID: fields[1]}
		if len(fields) >= 4 {
			if looksLikeSizeUnit(fields[3]) {
				model.Size = fields[2] + " " + fields[3]
				model.Modified = strings.Join(fields[4:], " ")
			} else {
				model.Size = fields[2]
				model.Modified = strings.Join(fields[3:], " ")
			}
		} else if len(fields) == 3 {
			model.Size = fields[2]
		}
		models = append(models, model)
	}
	return models
}

func looksLikeSizeUnit(value string) bool {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "B", "KB", "MB", "GB", "TB":
		return true
	default:
		return false
	}
}
