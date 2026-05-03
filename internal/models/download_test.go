package models

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDownloadWritesFileAndReportsProgress(t *testing.T) {
	t.Parallel()
	body := []byte("model-bytes")
	sum := sha256Hex(body)
	httpClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return byteResponse(http.StatusOK, body), nil
	})}

	var progressCalls int
	result, err := Download(context.Background(), DownloadConfig{
		URL:        "http://download.test/model.gguf",
		DestDir:    t.TempDir(),
		SHA256:     sum,
		Filename:   "model.gguf",
		HTTPClient: httpClient,
		OnProgress: func(downloaded, total int64) {
			progressCalls++
			require.LessOrEqual(t, downloaded, total)
		},
	})
	require.NoError(t, err)
	require.False(t, result.Cached)
	require.Equal(t, int64(len(body)), result.Size)
	require.Greater(t, progressCalls, 0)
	data, err := os.ReadFile(result.Path)
	require.NoError(t, err)
	require.Equal(t, body, data)
}

func TestDownloadUsesValidCachedFile(t *testing.T) {
	t.Parallel()
	body := []byte("cached-model")
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "model.gguf"), body, 0o644))

	var hits int
	httpClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		hits++
		t.Fatalf("server should not be called for valid cache")
		return byteResponse(http.StatusInternalServerError, nil), nil
	})}

	result, err := Download(context.Background(), DownloadConfig{
		URL:        "http://download.test/model.gguf",
		DestDir:    dir,
		Filename:   "model.gguf",
		SHA256:     sha256Hex(body),
		HTTPClient: httpClient,
	})
	require.NoError(t, err)
	require.True(t, result.Cached)
	require.Equal(t, 0, hits)
}

func TestDownloadReplacesCorruptCachedFile(t *testing.T) {
	t.Parallel()
	body := []byte("fresh-model")
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "model.gguf"), []byte("corrupt"), 0o644))

	httpClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return byteResponse(http.StatusOK, body), nil
	})}

	result, err := Download(context.Background(), DownloadConfig{
		URL:        "http://download.test/model.gguf",
		DestDir:    dir,
		Filename:   "model.gguf",
		SHA256:     sha256Hex(body),
		HTTPClient: httpClient,
	})
	require.NoError(t, err)
	require.False(t, result.Cached)
	data, err := os.ReadFile(result.Path)
	require.NoError(t, err)
	require.Equal(t, body, data)
}

func TestDownloadResumesPartialFile(t *testing.T) {
	t.Parallel()
	body := []byte("model-bytes")
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "model.gguf.part"), body[:5], 0o644))

	httpClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		require.Equal(t, "bytes=5-", r.Header.Get("Range"))
		resp := byteResponse(http.StatusPartialContent, body[5:])
		resp.Header.Set("Content-Range", fmt.Sprintf("bytes 5-%d/%d", len(body)-1, len(body)))
		return resp, nil
	})}

	result, err := Download(context.Background(), DownloadConfig{
		URL:        "http://download.test/model.gguf",
		DestDir:    dir,
		Filename:   "model.gguf",
		SHA256:     sha256Hex(body),
		HTTPClient: httpClient,
	})
	require.NoError(t, err)
	require.True(t, result.Resumed)
	data, err := os.ReadFile(result.Path)
	require.NoError(t, err)
	require.Equal(t, body, data)
}

func TestDownloadRejectsChecksumMismatch(t *testing.T) {
	t.Parallel()
	httpClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return byteResponse(http.StatusOK, []byte("bad")), nil
	})}

	_, err := Download(context.Background(), DownloadConfig{
		URL:        "http://download.test/model.gguf",
		DestDir:    t.TempDir(),
		Filename:   "model.gguf",
		SHA256:     sha256Hex([]byte("good")),
		HTTPClient: httpClient,
	})
	require.ErrorContains(t, err, "checksum mismatch")
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func byteResponse(status int, body []byte) *http.Response {
	if body == nil {
		body = []byte{}
	}
	return &http.Response{
		StatusCode:    status,
		Status:        fmt.Sprintf("%d %s", status, http.StatusText(status)),
		Header:        make(http.Header),
		Body:          io.NopCloser(bytes.NewReader(body)),
		ContentLength: int64(len(body)),
	}
}

func jsonResponse(v any) *http.Response {
	var buf bytes.Buffer
	_ = json.NewEncoder(&buf).Encode(v)
	resp := byteResponse(http.StatusOK, buf.Bytes())
	resp.Header.Set("Content-Type", "application/json")
	return resp
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
