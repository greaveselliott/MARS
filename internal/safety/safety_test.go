package safety

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

func TestCheck_withinLimits(t *testing.T) {
	stats := DiffStats{
		FilesChanged: 3,
		LinesPerFile: map[string]int{"a.go": 50, "b.go": 30, "c.go": 10},
		TotalLines:   90,
		Deletions:    0,
	}
	if err := Check(stats, DefaultLimits()); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestCheck_exceedsFileCount(t *testing.T) {
	stats := DiffStats{
		FilesChanged: 20,
		LinesPerFile: map[string]int{},
		TotalLines:   100,
	}
	err := Check(stats, DefaultLimits())
	if err == nil {
		t.Fatal("expected error for exceeding file count")
	}
	if !strings.Contains(err.Error(), "files changed") {
		t.Errorf("expected file count error, got %v", err)
	}
}

func TestCheck_exceedsLinesPerFile(t *testing.T) {
	stats := DiffStats{
		FilesChanged: 1,
		LinesPerFile: map[string]int{"huge.go": 1000},
		TotalLines:   1000,
	}
	err := Check(stats, DefaultLimits())
	if err == nil {
		t.Fatal("expected error for exceeding per-file line count")
	}
	if !strings.Contains(err.Error(), "lines changed in huge.go") {
		t.Errorf("expected per-file error, got %v", err)
	}
}

func TestCheck_exceedsTotalLines(t *testing.T) {
	stats := DiffStats{
		FilesChanged: 5,
		LinesPerFile: map[string]int{
			"a.go": 490, "b.go": 490, "c.go": 490,
			"d.go": 490, "e.go": 490,
		},
		TotalLines: 5000,
	}
	err := Check(stats, DefaultLimits())
	if err == nil {
		t.Fatal("expected error for exceeding total lines")
	}
	if !strings.Contains(err.Error(), "total lines") {
		t.Errorf("expected total lines error, got %v", err)
	}
}

func TestCheck_forbidDelete(t *testing.T) {
	stats := DiffStats{
		FilesChanged: 1,
		LinesPerFile: map[string]int{"a.go": 10},
		TotalLines:   10,
		Deletions:    2,
	}
	limits := DefaultLimits()
	err := Check(stats, limits)
	if err == nil {
		t.Fatal("expected error for file deletions")
	}
	if !strings.Contains(err.Error(), "deletions") {
		t.Errorf("expected deletion error, got %v", err)
	}
}

func TestCheck_allowsDeleteWhenPermitted(t *testing.T) {
	stats := DiffStats{
		FilesChanged: 1,
		LinesPerFile: map[string]int{"a.go": 10},
		TotalLines:   10,
		Deletions:    2,
	}
	limits := DefaultLimits()
	limits.ForbidDelete = false
	if err := Check(stats, limits); err != nil {
		t.Errorf("expected no error with ForbidDelete=false, got %v", err)
	}
}

func TestScanForSecrets_detectsAWSKeys(t *testing.T) {
	content := `aws_access_key_id = AKIA_EXAMPLE_REDACTED`
	results := ScanForSecrets("config.yaml", content)
	if len(results) == 0 {
		t.Fatal("expected to detect AWS key")
	}
	if results[0].Pattern != "AWS Access Key" {
		t.Errorf("expected AWS Access Key pattern, got %q", results[0].Pattern)
	}
}

func TestScanForSecrets_detectsGitHubTokens(t *testing.T) {
	tests := []string{
		"token: github-token-placeholder",
		"secret: github-token-placeholder",
		"pat: github-token-placeholder",
	}
	for _, content := range tests {
		results := ScanForSecrets("env.sh", content)
		if len(results) == 0 {
			t.Errorf("expected to detect GitHub token in %q", content)
			continue
		}
		if !strings.Contains(results[0].Pattern, "GitHub Token") {
			t.Errorf("expected GitHub Token pattern, got %q", results[0].Pattern)
		}
	}
}

func TestScanForSecrets_detectsPrivateKeys(t *testing.T) {
	content := `private-key-placeholder`
	results := ScanForSecrets("id_rsa", content)
	if len(results) == 0 {
		t.Fatal("expected to detect private key")
	}
	if !strings.Contains(results[0].Pattern, "Private Key") {
		t.Errorf("expected Private Key pattern, got %q", results[0].Pattern)
	}
}

func TestScanForSecrets_detectsPasswordInURL(t *testing.T) {
	content := `DATABASE_URL=postgres://example-user:<password>@db.example.com/mydb`
	results := ScanForSecrets(".env", content)
	if len(results) == 0 {
		t.Fatal("expected to detect password in URL")
	}
	found := false
	for _, r := range results {
		if r.Pattern == "Password in URL" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected Password in URL pattern among results")
	}
}

func TestScanForSecrets_detectsGenericAPIKey(t *testing.T) {
	content := `api_key = "sk_example_redacted"`
	results := ScanForSecrets("config.toml", content)
	if len(results) == 0 {
		t.Fatal("expected to detect generic API key")
	}
	found := false
	for _, r := range results {
		if r.Pattern == "Generic API Key Assignment" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected Generic API Key Assignment pattern among results")
	}
}

func TestScanForSecrets_cleanFileReturnsNone(t *testing.T) {
	content := `package main

import "fmt"

func main() {
	fmt.Println("Hello, world!")
}
`
	results := ScanForSecrets("main.go", content)
	if len(results) != 0 {
		t.Errorf("expected no secrets in clean file, got %d: %+v", len(results), results)
	}
}

func TestScanForSecrets_reportsCorrectLineNumber(t *testing.T) {
	content := "line1\nline2\nAKIA_EXAMPLE_REDACTED\nline4"
	results := ScanForSecrets("test.txt", content)
	if len(results) == 0 {
		t.Fatal("expected to detect AWS key")
	}
	if results[0].Line != 3 {
		t.Errorf("expected line 3, got %d", results[0].Line)
	}
}

func TestEmergencyStop_executesAll(t *testing.T) {
	es := NewEmergencyStop()
	var calls []int

	es.Register(func(_ context.Context) error {
		calls = append(calls, 1)
		return nil
	})
	es.Register(func(_ context.Context) error {
		calls = append(calls, 2)
		return nil
	})
	es.Register(func(_ context.Context) error {
		calls = append(calls, 3)
		return nil
	})

	errs := es.Execute(context.Background())
	if len(errs) != 0 {
		t.Errorf("expected no errors, got %v", errs)
	}
	if len(calls) != 3 {
		t.Errorf("expected 3 handlers called, got %d", len(calls))
	}
}

func TestEmergencyStop_collectsErrors(t *testing.T) {
	es := NewEmergencyStop()

	es.Register(func(_ context.Context) error {
		return nil
	})
	es.Register(func(_ context.Context) error {
		return fmt.Errorf("handler 1 failed")
	})
	es.Register(func(_ context.Context) error {
		return fmt.Errorf("handler 2 failed")
	})

	errs := es.Execute(context.Background())
	if len(errs) != 2 {
		t.Fatalf("expected 2 errors, got %d: %v", len(errs), errs)
	}
	if !strings.Contains(errs[0].Error(), "handler 1 failed") {
		t.Errorf("expected first error message, got %v", errs[0])
	}
	if !strings.Contains(errs[1].Error(), "handler 2 failed") {
		t.Errorf("expected second error message, got %v", errs[1])
	}
}

func TestEmergencyStop_emptyExecute(t *testing.T) {
	es := NewEmergencyStop()
	errs := es.Execute(context.Background())
	if len(errs) != 0 {
		t.Errorf("expected no errors from empty stop, got %v", errs)
	}
}

func TestDefaultLimits(t *testing.T) {
	limits := DefaultLimits()
	if limits.MaxFilesPerJob != 10 {
		t.Errorf("expected MaxFilesPerJob=10, got %d", limits.MaxFilesPerJob)
	}
	if limits.MaxLinesPerFile != 500 {
		t.Errorf("expected MaxLinesPerFile=500, got %d", limits.MaxLinesPerFile)
	}
	if limits.MaxTotalLines != 2000 {
		t.Errorf("expected MaxTotalLines=2000, got %d", limits.MaxTotalLines)
	}
	if limits.MaxOpenPRsPerRepo != 3 {
		t.Errorf("expected MaxOpenPRsPerRepo=3, got %d", limits.MaxOpenPRsPerRepo)
	}
	if !limits.ForbidDelete {
		t.Error("expected ForbidDelete=true by default")
	}
}
