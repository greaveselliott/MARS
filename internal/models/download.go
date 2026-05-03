package models

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

// DownloadConfig controls a single model-file download.
type DownloadConfig struct {
	URL        string
	DestDir    string
	Filename   string
	SHA256     string
	HTTPClient *http.Client
	OnProgress func(downloaded, total int64)
}

// DownloadResult describes the downloaded or cached model file.
type DownloadResult struct {
	Path    string
	Size    int64
	Cached  bool
	Resumed bool
}

// Download downloads a model file with cache reuse, partial-file resume, and
// optional SHA256 verification.
func Download(ctx context.Context, cfg DownloadConfig) (DownloadResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if strings.TrimSpace(cfg.URL) == "" {
		return DownloadResult{}, fmt.Errorf("models download: URL is required")
	}
	if strings.TrimSpace(cfg.DestDir) == "" {
		return DownloadResult{}, fmt.Errorf("models download: destination directory is required")
	}
	filename, err := downloadFilename(cfg)
	if err != nil {
		return DownloadResult{}, err
	}
	if err := os.MkdirAll(cfg.DestDir, 0o755); err != nil {
		return DownloadResult{}, fmt.Errorf("models download: create destination directory: %w", err)
	}

	destPath := filepath.Join(cfg.DestDir, filename)
	if info, err := os.Stat(destPath); err == nil {
		ok, verifyErr := verifyFile(destPath, cfg.SHA256)
		if verifyErr != nil {
			return DownloadResult{}, verifyErr
		}
		if ok {
			return DownloadResult{Path: destPath, Size: info.Size(), Cached: true}, nil
		}
		if err := os.Remove(destPath); err != nil {
			return DownloadResult{}, fmt.Errorf("models download: remove corrupt cached file: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return DownloadResult{}, fmt.Errorf("models download: stat destination: %w", err)
	}

	partialPath := destPath + ".part"
	startOffset := int64(0)
	if info, err := os.Stat(partialPath); err == nil {
		startOffset = info.Size()
	} else if !os.IsNotExist(err) {
		return DownloadResult{}, fmt.Errorf("models download: stat partial file: %w", err)
	}

	result, err := downloadToPartial(ctx, cfg, partialPath, startOffset)
	if err != nil {
		return DownloadResult{}, err
	}
	ok, err := verifyFile(partialPath, cfg.SHA256)
	if err != nil {
		return DownloadResult{}, err
	}
	if !ok {
		_ = os.Remove(partialPath)
		return DownloadResult{}, fmt.Errorf("models download: checksum mismatch for %s", filename)
	}
	if err := os.Rename(partialPath, destPath); err != nil {
		return DownloadResult{}, fmt.Errorf("models download: finalize file: %w", err)
	}
	result.Path = destPath
	return result, nil
}

func downloadFilename(cfg DownloadConfig) (string, error) {
	if strings.TrimSpace(cfg.Filename) != "" {
		return filepath.Base(cfg.Filename), nil
	}
	parsed, err := url.Parse(cfg.URL)
	if err != nil {
		return "", fmt.Errorf("models download: parse URL: %w", err)
	}
	name := path.Base(parsed.Path)
	if name == "." || name == "/" || strings.TrimSpace(name) == "" {
		return "", fmt.Errorf("models download: filename is required when URL has no file path")
	}
	return name, nil
}

func downloadToPartial(ctx context.Context, cfg DownloadConfig, partialPath string, startOffset int64) (DownloadResult, error) {
	client := cfg.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cfg.URL, nil)
	if err != nil {
		return DownloadResult{}, fmt.Errorf("models download: create request: %w", err)
	}
	if startOffset > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", startOffset))
	}

	resp, err := client.Do(req)
	if err != nil {
		return DownloadResult{}, fmt.Errorf("models download: request: %w", err)
	}
	defer resp.Body.Close()

	flags := os.O_CREATE | os.O_WRONLY
	downloaded := startOffset
	resumed := startOffset > 0 && resp.StatusCode == http.StatusPartialContent
	switch {
	case resp.StatusCode == http.StatusOK:
		flags |= os.O_TRUNC
		downloaded = 0
		resumed = false
	case resp.StatusCode == http.StatusPartialContent:
		flags |= os.O_APPEND
	case resp.StatusCode == http.StatusRequestedRangeNotSatisfiable:
		_ = os.Remove(partialPath)
		return downloadToPartial(ctx, cfg, partialPath, 0)
	default:
		return DownloadResult{}, fmt.Errorf("models download: HTTP %s from %s", resp.Status, cfg.URL)
	}

	out, err := os.OpenFile(partialPath, flags, 0o644)
	if err != nil {
		return DownloadResult{}, fmt.Errorf("models download: open partial file: %w", err)
	}
	defer out.Close()

	total := responseTotal(resp, downloaded)
	if cfg.OnProgress != nil {
		cfg.OnProgress(downloaded, total)
	}
	reader := &progressReader{
		reader:     resp.Body,
		downloaded: downloaded,
		total:      total,
		onProgress: cfg.OnProgress,
	}
	if _, err := io.Copy(out, reader); err != nil {
		return DownloadResult{}, fmt.Errorf("models download: write partial file: %w", err)
	}
	if err := out.Close(); err != nil {
		return DownloadResult{}, fmt.Errorf("models download: close partial file: %w", err)
	}
	info, err := os.Stat(partialPath)
	if err != nil {
		return DownloadResult{}, fmt.Errorf("models download: stat partial file: %w", err)
	}
	return DownloadResult{Path: partialPath, Size: info.Size(), Resumed: resumed}, nil
}

func responseTotal(resp *http.Response, downloaded int64) int64 {
	if contentRange := resp.Header.Get("Content-Range"); contentRange != "" {
		if slash := strings.LastIndex(contentRange, "/"); slash >= 0 {
			if total, err := strconv.ParseInt(contentRange[slash+1:], 10, 64); err == nil {
				return total
			}
		}
	}
	if resp.ContentLength >= 0 {
		return downloaded + resp.ContentLength
	}
	return 0
}

type progressReader struct {
	reader     io.Reader
	downloaded int64
	total      int64
	onProgress func(downloaded, total int64)
}

func (r *progressReader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	if n > 0 {
		r.downloaded += int64(n)
		if r.onProgress != nil {
			r.onProgress(r.downloaded, r.total)
		}
	}
	return n, err
}

func verifyFile(filePath, expectedSHA256 string) (bool, error) {
	if strings.TrimSpace(expectedSHA256) == "" {
		return true, nil
	}
	f, err := os.Open(filePath)
	if err != nil {
		return false, fmt.Errorf("models download: open for checksum: %w", err)
	}
	defer f.Close()
	hasher := sha256.New()
	if _, err := io.Copy(hasher, f); err != nil {
		return false, fmt.Errorf("models download: checksum read: %w", err)
	}
	got := hex.EncodeToString(hasher.Sum(nil))
	return strings.EqualFold(got, strings.TrimSpace(expectedSHA256)), nil
}
