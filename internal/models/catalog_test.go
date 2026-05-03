package models

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseOllamaList(t *testing.T) {
	rows := ParseOllamaList(`NAME              ID              SIZE      MODIFIED
qwen3.6:27b       abc123          17 GB     2 days ago
laguna-xs.2       def456          21GB      1 hour ago
`)

	require.Equal(t, []OllamaModel{
		{Name: "qwen3.6:27b", ID: "abc123", Size: "17 GB", Modified: "2 days ago"},
		{Name: "laguna-xs.2", ID: "def456", Size: "21GB", Modified: "1 hour ago"},
	}, rows)
}

func TestListOllamaModelsUsesRunner(t *testing.T) {
	rows, err := ListOllamaModels(context.Background(), commandRunnerFunc(func(ctx context.Context, name string, args ...string) ([]byte, error) {
		require.Equal(t, "ollama", name)
		require.Equal(t, []string{"list"}, args)
		return []byte("NAME ID SIZE MODIFIED\nqwen3.6:27b abc 17GB today\n"), nil
	}))

	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, "qwen3.6:27b", rows[0].Name)
}

func TestListOllamaModelsActionableError(t *testing.T) {
	_, err := ListOllamaModels(context.Background(), commandRunnerFunc(func(ctx context.Context, name string, args ...string) ([]byte, error) {
		return []byte("ollama not found"), errors.New("exit 127")
	}))

	require.ErrorContains(t, err, "install/start Ollama")
	require.ErrorContains(t, err, "ollama pull <model>")
}

type commandRunnerFunc func(context.Context, string, ...string) ([]byte, error)

func (f commandRunnerFunc) CombinedOutput(ctx context.Context, name string, args ...string) ([]byte, error) {
	return f(ctx, name, args...)
}
