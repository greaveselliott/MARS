package setup

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
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
	llamaCppPinnedTag  = "b8833"
	llamaCppReleaseFmt = "https://github.com/ggml-org/llama.cpp/releases/download/%s/llama-%s-bin-%s.tar.gz"
)

var llamaServerSHA256 = map[string]string{
	"macos-arm64": "1b31955f312671a5842521e8fd3b85ef6633c2743d62021b325f0d7f93016423",
	"macos-x64":   "33746603d6b9dc4546770d21f4c3049d7830abcdd7da03597dd4110f77ada7ee",
}

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

			platform, err := llamaPlatformKey()
			if err != nil {
				return err
			}
			expectedSHA := llamaServerSHA256[platform]
			if expectedSHA == "" {
				return fmt.Errorf("no pinned llama.cpp checksum for platform %s — install llama-server manually into ~/.mars-harness/bin/llama-server or update the checksum table", platform)
			}

			url := fmt.Sprintf(llamaCppReleaseFmt, llamaCppPinnedTag, llamaCppPinnedTag, platform)
			slog.Info("downloading llama-server", "url", url, "platform", platform)

			tarPath := filepath.Join(binDir, "llama-server.tar.gz")
			if err := downloadFile(tarPath, url, expectedSHA); err != nil {
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

func downloadFile(destPath, url, expectedSHA256 string) error {
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

	hasher := sha256.New()
	if _, err := io.Copy(io.MultiWriter(out, hasher), resp.Body); err != nil {
		os.Remove(destPath)
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	sum := hex.EncodeToString(hasher.Sum(nil))
	if !strings.EqualFold(sum, expectedSHA256) {
		os.Remove(destPath)
		return fmt.Errorf("checksum mismatch for %s: got %s, want %s", url, sum, expectedSHA256)
	}
	return nil
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
