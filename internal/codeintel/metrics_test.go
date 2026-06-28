/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/context-efficiency.md
- docs/product-specs/product-surface.md
- docs/features/F-005-agent-execution-runtime.md
*/
package codeintel

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/greaveselliott/mars/internal/trace"
)

func TestMetricsAggregatesTraceSummaries(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "mars.db")
	store, err := trace.OpenStore(dbPath)
	if err != nil {
		t.Fatalf("OpenStore: %v", err)
	}
	defer store.Close()

	summary := trace.Summary{
		TraceID:         "trace-1",
		JobID:           "job-1",
		Outcome:         "completed",
		WallMs:          25,
		TotalTokens:     100,
		ToolInvocations: 3,
		LLMCalls:        2,
		ToolCounts: map[string]int{
			metricToolCalls:        1,
			metricToolOutputBytes:  42,
			metricContextBytes:     120,
			metricContextRefreshed: 1,
			metricBroadCalls:       2,
			metricBroadBytes:       300,
			metricBulkReads:        1,
			metricBroadShell:       1,
		},
		CodeIntel: &trace.CodeIntelSummary{Mode: ModeEnabled, Source: "test"},
	}
	data, err := json.Marshal(summary)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if err := store.Save(context.Background(), summary.JobID, summary.TraceID, "", string(data)); err != nil {
		t.Fatalf("Save: %v", err)
	}

	report, err := Metrics(context.Background(), MetricsOptions{RepoPath: "/repo", DBPath: dbPath, WindowDays: 30})
	if err != nil {
		t.Fatalf("Metrics: %v", err)
	}
	if report.Jobs != 1 || report.GraphEnabledJobs != 1 || report.CodeIntelToolCalls != 1 {
		t.Fatalf("unexpected metrics report: %+v", report)
	}
	if report.BroadExplorationCalls != 2 || report.BulkFileReadCalls != 1 || report.BroadShellSearchCalls != 1 {
		t.Fatalf("expected broad exploration counters, got %+v", report)
	}
	if report.ModeSources["test"] != 1 {
		t.Fatalf("expected mode source bucket, got %+v", report.ModeSources)
	}
}

func TestMetricsClassifiesUnavailableOnce(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "mars.db")
	store, err := trace.OpenStore(dbPath)
	if err != nil {
		t.Fatalf("OpenStore: %v", err)
	}
	defer store.Close()

	summary := trace.Summary{
		TraceID: "trace-1",
		JobID:   "job-1",
		ToolCounts: map[string]int{
			metricContextUnavailable: 1,
		},
		CodeIntel: &trace.CodeIntelSummary{Mode: ModeEnabled, Source: "test"},
	}
	data, err := json.Marshal(summary)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if err := store.Save(context.Background(), summary.JobID, summary.TraceID, "", string(data)); err != nil {
		t.Fatalf("Save: %v", err)
	}

	report, err := Metrics(context.Background(), MetricsOptions{RepoPath: "/repo", DBPath: dbPath, WindowDays: 30})
	if err != nil {
		t.Fatalf("Metrics: %v", err)
	}
	if report.GraphUnavailableJobs != 1 || report.GraphEnabledJobs != 0 {
		t.Fatalf("expected unavailable-only classification, got %+v", report)
	}
}

func TestBenchmarkUsesExplicitChangedPathsForImpactEvidence(t *testing.T) {
	repo := t.TempDir()
	writeFile(t, repo, "go.mod", "module example.test/app\n\ngo 1.22\n")
	writeFile(t, repo, "internal/app/app.go", "package app\n\nfunc Run() string { return \"ok\" }\n")
	writeFile(t, repo, "internal/app/app_test.go", "package app\n\nfunc TestRun(t *testing.T) {}\n")
	writeFile(t, repo, "docs/design-docs/app.md", "Run is documented by internal/app/app.go.\n")

	report, err := Benchmark(context.Background(), BenchmarkOptions{
		RepoPath:      repo,
		DBPath:        filepath.Join(t.TempDir(), "mars.db"),
		Trials:        1,
		ChangedPaths:  []string{"internal/app/app.go"},
		ExpectedFiles: []string{"internal/app/app.go"},
		ExpectedTests: []string{"internal/app/app_test.go"},
		ExpectedDocs:  []string{"docs/design-docs/app.md"},
	})
	if err != nil {
		t.Fatalf("Benchmark: %v", err)
	}
	if !report.LocalOnly || len(report.Control) != 1 || len(report.Treatment) != 1 {
		t.Fatalf("unexpected benchmark shape: %+v", report)
	}
	if report.Summary.TreatmentAvgContextBytes == 0 {
		t.Fatalf("expected treatment context bytes, got %+v", report.Summary)
	}
	if report.Summary.ExpectedFilesHitRate != 1 {
		t.Fatalf("expected changed path hit rate, got %+v", report.Summary)
	}
	if report.Summary.ExpectedTestsHitRate != 1 {
		t.Fatalf("expected test hit rate, got %+v", report.Summary)
	}
	if report.Summary.ExpectedDocsHitRate != 1 {
		t.Fatalf("expected doc hit rate, got %+v", report.Summary)
	}
}
