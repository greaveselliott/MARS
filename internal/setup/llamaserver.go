package setup

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	llamaCppReleasesAPI = "https://api.github.com/repos/ggml-org/llama.cpp/releases/latest"
	llamaCppReleaseFmt  = "https://github.com/ggml-org/llama.cpp/releases/download/%s/llama-%s-bin-%s.tar.gz"
)

func installLlamaServerStep(baseDir string) Step {
	binDir := filepath.Join(baseDir, "bin")
	binaryPath := filepath.Join(binDir, "llama-server")

	return Step{
		Name: "install-llama-server",
		Check: func() (bool, error) {
			info, err := os.Stat(binaryPath)
			if err != nil {
				return false, nil
			}
			if info.Mode()&0o111 == 0 {
				return false, nil
			}
			out, err := exec.Command(binaryPath, "--version").CombinedOutput()
			if err != nil {
				slog.Debug("llama-server exists but --version failed", "err", err)
				return false, nil
			}
			slog.Info("llama-server already installed", "version", strings.TrimSpace(string(out)))
			return true, nil
		},
		Execute: func() error {
			if err := os.MkdirAll(binDir, 0o755); err != nil {
				return fmt.Errorf("create bin dir: %w — check directory permissions", err)
			}

			tag, err := fetchLatestLlamaCppTag()
			if err != nil {
				return fmt.Errorf("fetch llama.cpp release: %w — check network connectivity", err)
			}
			slog.Info("latest llama.cpp release", "tag", tag)

			platform, err := llamaPlatformKey()
			if err != nil {
				return err
			}

			url := fmt.Sprintf(llamaCppReleaseFmt, tag, tag, platform)
			slog.Info("downloading llama-server", "url", url, "platform", platform)

			tarPath := filepath.Join(binDir, "llama-server.tar.gz")
			if err := downloadFile(tarPath, url); err != nil {
				return fmt.Errorf("download llama-server: %w — check network connectivity", err)
			}
			defer os.Remove(tarPath)

			if err := extractLlamaServer(tarPath, binDir); err != nil {
				return fmt.Errorf("extract llama-server: %w", err)
			}

			if err := os.Chmod(binaryPath, 0o755); err != nil {
				return fmt.Errorf("chmod llama-server: %w", err)
			}

			out, err := exec.Command(binaryPath, "--version").CombinedOutput()
			if err != nil {
				return fmt.Errorf("llama-server verification failed: %w — binary may be incompatible with this OS/arch", err)
			}
			slog.Info("llama-server installed", "version", strings.TrimSpace(string(out)))
			return nil
		},
	}
}

type ghRelease struct {
	TagName string `json:"tag_name"`
}

func fetchLatestLlamaCppTag() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), httpTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, llamaCppReleasesAPI, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned %s", resp.Status)
	}

	var rel ghRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return "", fmt.Errorf("decode release JSON: %w", err)
	}
	if rel.TagName == "" {
		return "", fmt.Errorf("release has no tag_name")
	}
	return rel.TagName, nil
}

func llamaPlatformKey() (string, error) {
	switch {
	case runtime.GOOS == "darwin" && runtime.GOARCH == "arm64":
		return "macos-arm64", nil
	case runtime.GOOS == "darwin" && runtime.GOARCH == "amd64":
		return "macos-x64", nil
	case runtime.GOOS == "linux" && runtime.GOARCH == "amd64":
		return "ubuntu-x64", nil
	case runtime.GOOS == "linux" && runtime.GOARCH == "arm64":
		return "ubuntu-arm64", nil
	default:
		return "", fmt.Errorf("unsupported platform %s/%s — build llama.cpp from source and place llama-server in ~/.mars-harness/bin/", runtime.GOOS, runtime.GOARCH)
	}
}

func downloadFile(destPath, url string) error {
	ctx, cancel := context.WithTimeout(context.Background(), downloadTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %s from %s", resp.Status, url)
	}

	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, resp.Body); err != nil {
		os.Remove(destPath)
		return err
	}
	return out.Close()
}

func extractLlamaServer(tarPath, destDir string) error {
	cmd := exec.Command("tar", "xzf", tarPath, "-C", destDir, "--strip-components=1")
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("tar extract: %w", err)
	}

	binaryPath := filepath.Join(destDir, "llama-server")
	if _, err := os.Stat(binaryPath); err != nil {
		return fmt.Errorf("llama-server not found after extraction — archive structure may have changed: %w", err)
	}
	return nil
}
