/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/local-inference.md
- docs/features/F-002-zero-config-shell-path.md
- docs/features/F-003-local-inference-lifecycle.md
*/
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
	llamaCppReleaseFmt = "https://github.com/ggml-org/llama.cpp/releases/download/%s/%s"
)

type llamaCppDocument struct {
	Name   string
	URL    string
	SHA256 string
}

type llamaCppPlatformArtifact struct {
	ArchiveName string
	SizeBytes   int64
	SHA256      string
	Enabled     bool
}

type llamaCppRelease struct {
	Tag          string
	SourceCommit string
	License      llamaCppDocument
	Notices      []llamaCppDocument
	Platforms    map[string]llamaCppPlatformArtifact
}

var pinnedLlamaCppRelease = llamaCppRelease{
	Tag:          "b8833",
	SourceCommit: "45cac7ca703fb9085eae62b9121fca01d20177f6",
	License: llamaCppDocument{
		Name:   "MIT License",
		URL:    "https://raw.githubusercontent.com/ggml-org/llama.cpp/45cac7ca703fb9085eae62b9121fca01d20177f6/LICENSE",
		SHA256: "94f29bbed6a22c35b992c5c6ebf0e7c92f13b836b90f36f461c9cf2f0f1d010d",
	},
	Notices: []llamaCppDocument{
		{
			Name:   "nlohmann/json license notice",
			URL:    "https://raw.githubusercontent.com/ggml-org/llama.cpp/45cac7ca703fb9085eae62b9121fca01d20177f6/licenses/LICENSE-jsonhpp",
			SHA256: "c0d068392ea65358b798b8c165103560f06e9e3b38c4ab4e2d8810a7b931af86",
		},
	},
	Platforms: map[string]llamaCppPlatformArtifact{
		"macos-arm64": {
			ArchiveName: "llama-b8833-bin-macos-arm64.tar.gz",
			SizeBytes:   8_552_033,
			SHA256:      "1b31955f312671a5842521e8fd3b85ef6633c2743d62021b325f0d7f93016423",
			Enabled:     true,
		},
		"macos-x64": {
			ArchiveName: "llama-b8833-bin-macos-x64.tar.gz",
			SizeBytes:   8_576_524,
			SHA256:      "33746603d6b9dc4546770d21f4c3049d7830abcdd7da03597dd4110f77ada7ee",
			Enabled:     true,
		},
		"ubuntu-arm64": {
			ArchiveName: "llama-b8833-bin-ubuntu-arm64.tar.gz",
			SizeBytes:   10_962_028,
			SHA256:      "e7ca8183587d0841dc16a868fa11f1e33c46032b47376c0ceda4160e4ddbbb2a",
			Enabled:     false,
		},
		"ubuntu-x64": {
			ArchiveName: "llama-b8833-bin-ubuntu-x64.tar.gz",
			SizeBytes:   13_872_833,
			SHA256:      "8262b45a82436aefd994f16461d99a02cd1ddf0bb343ef0153186a69229667c7",
			Enabled:     false,
		},
	},
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
			platform, err := llamaPlatformKey()
			if err != nil {
				return err
			}
			artifact, err := prepareLlamaCppInstall(platform, binDir)
			if err != nil {
				return err
			}

			url := fmt.Sprintf(llamaCppReleaseFmt, pinnedLlamaCppRelease.Tag, artifact.ArchiveName)
			slog.Info("downloading llama-server", "url", url, "platform", platform)

			tarPath := filepath.Join(binDir, "llama-server.tar.gz")
			if err := downloadFile(tarPath, url, artifact.SizeBytes, artifact.SHA256); err != nil {
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

func prepareLlamaCppInstall(platform, binDir string) (llamaCppPlatformArtifact, error) {
	artifact, err := enabledLlamaCppArtifact(platform)
	if err != nil {
		return llamaCppPlatformArtifact{}, err
	}
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		return llamaCppPlatformArtifact{}, fmt.Errorf("create bin dir: %w — check directory permissions", err)
	}
	return artifact, nil
}

func enabledLlamaCppArtifact(platform string) (llamaCppPlatformArtifact, error) {
	artifact, ok := pinnedLlamaCppRelease.Platforms[platform]
	if !ok {
		return llamaCppPlatformArtifact{}, fmt.Errorf("no pinned llama.cpp artifact for platform %s — build llama.cpp from source and place llama-server in ~/.mars/bin/", platform)
	}
	if !artifact.Enabled {
		return llamaCppPlatformArtifact{}, fmt.Errorf("pinned llama.cpp artifact for platform %s is recorded but installation is unavailable until T-077 delivers safe extraction — install llama-server manually into ~/.mars/bin/llama-server", platform)
	}
	return artifact, nil
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
		return "", fmt.Errorf("unsupported platform %s/%s — build llama.cpp from source and place llama-server in ~/.mars/bin/", runtime.GOOS, runtime.GOARCH)
	}
}

func downloadFile(destPath, url string, expectedSize int64, expectedSHA256 string) error {
	return downloadFileWithClient(http.DefaultClient, destPath, url, expectedSize, expectedSHA256)
}

func downloadFileWithClient(client *http.Client, destPath, url string, expectedSize int64, expectedSHA256 string) error {
	if expectedSize <= 0 {
		return fmt.Errorf("invalid expected download size %d for %s", expectedSize, url)
	}

	ctx, cancel := context.WithTimeout(context.Background(), downloadTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %s from %s", resp.Status, url)
	}
	if resp.ContentLength >= 0 && resp.ContentLength != expectedSize {
		return fmt.Errorf("size mismatch for %s: got %d bytes, want %d", url, resp.ContentLength, expectedSize)
	}

	out, err := os.Create(destPath)
	if err != nil {
		return err
	}

	hasher := sha256.New()
	written, copyErr := io.Copy(io.MultiWriter(out, hasher), io.LimitReader(resp.Body, expectedSize+1))
	closeErr := out.Close()
	if copyErr != nil {
		os.Remove(destPath)
		return copyErr
	}
	if closeErr != nil {
		os.Remove(destPath)
		return closeErr
	}
	if written != expectedSize {
		os.Remove(destPath)
		return fmt.Errorf("size mismatch for %s: got %d bytes, want %d", url, written, expectedSize)
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
