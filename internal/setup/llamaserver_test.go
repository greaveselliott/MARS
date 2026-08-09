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
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPinnedLlamaCppReleaseMetadata(t *testing.T) {
	expected := llamaCppRelease{
		Tag:          "b8833",
		SourceCommit: "45cac7ca703fb9085eae62b9121fca01d20177f6",
		LicenseID:    "MIT",
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

	assert.Equal(t, expected, pinnedLlamaCppRelease)
}

func TestEnabledLlamaCppArtifactAdmission(t *testing.T) {
	for _, platform := range []string{"macos-arm64", "macos-x64"} {
		t.Run(platform, func(t *testing.T) {
			artifact, err := enabledLlamaCppArtifact(platform)
			require.NoError(t, err)
			assert.True(t, artifact.Enabled)
			assert.NotEmpty(t, artifact.ArchiveName)
			assert.NotEmpty(t, artifact.SHA256)
		})
	}

	for _, platform := range []string{"ubuntu-arm64", "ubuntu-x64"} {
		t.Run(platform, func(t *testing.T) {
			_, err := enabledLlamaCppArtifact(platform)
			require.Error(t, err)
			assert.ErrorContains(t, err, "recorded but installation is unavailable")
		})
	}

	_, err := enabledLlamaCppArtifact("windows-amd64")
	require.Error(t, err)
	assert.ErrorContains(t, err, "no pinned llama.cpp artifact")
}

func TestPrepareLlamaCppInstallRejectsDisabledLinuxBeforeFilesystemMutation(t *testing.T) {
	binDir := filepath.Join(t.TempDir(), "missing", "bin")

	_, err := prepareLlamaCppInstall("ubuntu-arm64", binDir)
	require.Error(t, err)
	assert.ErrorContains(t, err, "unavailable until T-077")
	_, statErr := os.Stat(binDir)
	assert.ErrorIs(t, statErr, os.ErrNotExist)
}

func TestInstallLlamaServerRejectsUnplannedArtifactBeforeFilesystemMutation(t *testing.T) {
	baseDir := filepath.Join(t.TempDir(), "missing", ".mars")

	err := installLlamaServerStep(baseDir, emptyDownloadPlan()).Execute()
	require.Error(t, err)
	assert.ErrorContains(t, err, "absent or unusable")
	assert.ErrorContains(t, err, "remove/move it aside")
	_, statErr := os.Stat(baseDir)
	assert.ErrorIs(t, statErr, os.ErrNotExist)
}

func TestDownloadFileEnforcesSizeAndHashBeforeArtifactUse(t *testing.T) {
	payload := []byte("verified llama.cpp archive fixture")
	digest := sha256.Sum256(payload)
	expectedSHA := hex.EncodeToString(digest[:])

	tests := []struct {
		name         string
		body         []byte
		expectedSize int64
		expectedSHA  string
		chunked      bool
		wantErr      string
	}{
		{
			name:         "valid",
			body:         payload,
			expectedSize: int64(len(payload)),
			expectedSHA:  expectedSHA,
		},
		{
			name:         "under expected size",
			body:         payload[:len(payload)-1],
			expectedSize: int64(len(payload)),
			expectedSHA:  expectedSHA,
			wantErr:      "size mismatch",
		},
		{
			name:         "over hard read bound",
			body:         append(append([]byte(nil), payload...), 'x', 'y'),
			expectedSize: int64(len(payload)),
			expectedSHA:  expectedSHA,
			chunked:      true,
			wantErr:      "size mismatch",
		},
		{
			name:         "wrong hash",
			body:         payload,
			expectedSize: int64(len(payload)),
			expectedSHA:  "0000000000000000000000000000000000000000000000000000000000000000",
			wantErr:      "checksum mismatch",
		},
		{
			name:         "invalid expected size",
			body:         payload,
			expectedSize: 0,
			expectedSHA:  expectedSHA,
			wantErr:      "invalid expected download size",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			contentLength := int64(len(tt.body))
			if tt.chunked {
				contentLength = -1
			}
			client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode:    http.StatusOK,
					Status:        "200 OK",
					Header:        make(http.Header),
					Body:          io.NopCloser(bytes.NewReader(tt.body)),
					ContentLength: contentLength,
				}, nil
			})}

			dest := filepath.Join(t.TempDir(), "llama-server.tar.gz")
			err := downloadFileWithClient(client, dest, "https://example.invalid/llama.tar.gz", tt.expectedSize, tt.expectedSHA)
			if tt.wantErr == "" {
				require.NoError(t, err)
				assert.FileExists(t, dest)
				return
			}

			require.Error(t, err)
			assert.ErrorContains(t, err, tt.wantErr)
			_, statErr := os.Stat(dest)
			assert.ErrorIs(t, statErr, os.ErrNotExist)
		})
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}
