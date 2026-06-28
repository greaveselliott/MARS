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
	"sort"
	"strings"
	"time"
)

// BenchmarkOptions controls a local, model-free control/treatment measurement.
type BenchmarkOptions struct {
	RepoPath      string
	DBPath        string
	Case          string
	Trials        int
	ChangedPaths  []string
	ExpectedFiles []string
	ExpectedTests []string
	ExpectedDocs  []string
}

// BenchmarkReport captures self-contained evidence for graph usefulness.
type BenchmarkReport struct {
	GeneratedAt string           `json:"generated_at"`
	RepoPath    string           `json:"repo_path"`
	DBPath      string           `json:"db_path"`
	Case        string           `json:"case"`
	Trials      int              `json:"trials"`
	LocalOnly   bool             `json:"local_only"`
	Control     []BenchmarkTrial `json:"control"`
	Treatment   []BenchmarkTrial `json:"treatment"`
	Summary     BenchmarkSummary `json:"summary"`
	Notes       []string         `json:"notes,omitempty"`
}

// BenchmarkTrial is one disabled/enabled graph measurement.
type BenchmarkTrial struct {
	Mode               string    `json:"mode"`
	Status             Freshness `json:"status,omitempty"`
	DurationMS         int64     `json:"duration_ms"`
	ContextBytes       int       `json:"context_bytes"`
	FilesIndexed       int       `json:"files_indexed,omitempty"`
	ChangedPaths       int       `json:"changed_paths,omitempty"`
	Symbols            int       `json:"symbols,omitempty"`
	Tests              int       `json:"tests,omitempty"`
	Docs               int       `json:"docs,omitempty"`
	Features           int       `json:"features,omitempty"`
	Tickets            int       `json:"tickets,omitempty"`
	ExpectedFilesHit   int       `json:"expected_files_hit,omitempty"`
	ExpectedFilesTotal int       `json:"expected_files_total,omitempty"`
	ExpectedTestsHit   int       `json:"expected_tests_hit,omitempty"`
	ExpectedTestsTotal int       `json:"expected_tests_total,omitempty"`
	ExpectedDocsHit    int       `json:"expected_docs_hit,omitempty"`
	ExpectedDocsTotal  int       `json:"expected_docs_total,omitempty"`
	Message            string    `json:"message,omitempty"`
}

// BenchmarkSummary aggregates treatment/control measurements.
type BenchmarkSummary struct {
	ControlAvgDurationMS     int64   `json:"control_avg_duration_ms"`
	TreatmentAvgDurationMS   int64   `json:"treatment_avg_duration_ms"`
	ControlAvgContextBytes   int     `json:"control_avg_context_bytes"`
	TreatmentAvgContextBytes int     `json:"treatment_avg_context_bytes"`
	ExpectedFilesHitRate     float64 `json:"expected_files_hit_rate,omitempty"`
	ExpectedTestsHitRate     float64 `json:"expected_tests_hit_rate,omitempty"`
	ExpectedDocsHitRate      float64 `json:"expected_docs_hit_rate,omitempty"`
}

// Benchmark runs a local proof pass without contacting an inference endpoint.
func Benchmark(ctx context.Context, opts BenchmarkOptions) (BenchmarkReport, error) {
	opts = normalizeBenchmarkOptions(opts)
	report := BenchmarkReport{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339Nano),
		RepoPath:    opts.RepoPath,
		DBPath:      opts.DBPath,
		Case:        opts.Case,
		Trials:      opts.Trials,
		LocalOnly:   true,
		Notes: []string{
			"control measures the disabled graph path without injected context",
			"treatment measures BuildContext with automatic local refresh",
			"no model endpoint, fake endpoint, or external service is used",
		},
	}
	if report.DBPath == "" {
		report.DBPath = ResolveDBPath(report.RepoPath)
	}

	for i := 0; i < opts.Trials; i++ {
		control, err := runBenchmarkControl(ctx, opts.RepoPath, report.DBPath)
		if err != nil {
			return BenchmarkReport{}, err
		}
		report.Control = append(report.Control, control)

		treatment, err := runBenchmarkTreatment(ctx, opts)
		if err != nil {
			return BenchmarkReport{}, err
		}
		report.Treatment = append(report.Treatment, treatment)
	}
	report.Summary = summarizeBenchmark(report.Control, report.Treatment)
	return report, nil
}

func normalizeBenchmarkOptions(opts BenchmarkOptions) BenchmarkOptions {
	opts.RepoPath = strings.TrimSpace(opts.RepoPath)
	opts.DBPath = strings.TrimSpace(opts.DBPath)
	opts.Case = strings.TrimSpace(opts.Case)
	if opts.Case == "" {
		opts.Case = "current"
	}
	if opts.Trials <= 0 {
		opts.Trials = 2
	}
	opts.ChangedPaths = normalizeExpectedPaths(opts.ChangedPaths)
	opts.ExpectedFiles = normalizeExpectedPaths(opts.ExpectedFiles)
	opts.ExpectedTests = normalizeExpectedPaths(opts.ExpectedTests)
	opts.ExpectedDocs = normalizeExpectedPaths(opts.ExpectedDocs)
	return opts
}

func runBenchmarkControl(ctx context.Context, repoPath, dbPath string) (BenchmarkTrial, error) {
	started := time.Now()
	store, err := Open(repoPath, dbPath)
	if err != nil {
		return BenchmarkTrial{}, err
	}
	defer store.Close()
	status, err := store.Status(ctx)
	if err != nil {
		return BenchmarkTrial{}, err
	}
	return BenchmarkTrial{
		Mode:       ModeDisabled,
		Status:     status.Status,
		DurationMS: time.Since(started).Milliseconds(),
		Message:    "graph disabled; no context injected",
	}, nil
}

func runBenchmarkTreatment(ctx context.Context, opts BenchmarkOptions) (BenchmarkTrial, error) {
	started := time.Now()
	graph, err := BuildContext(ctx, opts.RepoPath, ContextOptions{Refresh: true, DBPath: opts.DBPath})
	if err != nil {
		return BenchmarkTrial{}, err
	}
	impact := graph.Impact
	if len(opts.ChangedPaths) > 0 {
		store, err := Open(opts.RepoPath, opts.DBPath)
		if err != nil {
			return BenchmarkTrial{}, err
		}
		defer store.Close()
		impact, err = store.Impact(ctx, opts.ChangedPaths, "")
		if err != nil {
			return BenchmarkTrial{}, err
		}
	}
	changed := normalizeExpectedPaths(impact.ChangedPaths)
	tests := normalizeExpectedPaths(impact.Tests)
	docs := normalizeExpectedPaths(impact.Docs)
	return BenchmarkTrial{
		Mode:               ModeEnabled,
		Status:             graph.Status.Status,
		DurationMS:         time.Since(started).Milliseconds(),
		ContextBytes:       len(graph.Text),
		FilesIndexed:       graph.Index.FilesIndexed,
		ChangedPaths:       len(changed),
		Symbols:            len(impact.Symbols),
		Tests:              len(tests),
		Docs:               len(docs),
		Features:           len(impact.Features),
		Tickets:            len(impact.Tickets),
		ExpectedFilesHit:   countHits(opts.ExpectedFiles, changed),
		ExpectedFilesTotal: len(opts.ExpectedFiles),
		ExpectedTestsHit:   countHits(opts.ExpectedTests, tests),
		ExpectedTestsTotal: len(opts.ExpectedTests),
		ExpectedDocsHit:    countHits(opts.ExpectedDocs, docs),
		ExpectedDocsTotal:  len(opts.ExpectedDocs),
		Message:            graph.Status.Message,
	}, nil
}

func summarizeBenchmark(control, treatment []BenchmarkTrial) BenchmarkSummary {
	var s BenchmarkSummary
	s.ControlAvgDurationMS = avgDuration(control)
	s.TreatmentAvgDurationMS = avgDuration(treatment)
	s.ControlAvgContextBytes = avgContextBytes(control)
	s.TreatmentAvgContextBytes = avgContextBytes(treatment)
	s.ExpectedFilesHitRate = hitRate(treatment, func(t BenchmarkTrial) (int, int) {
		return t.ExpectedFilesHit, t.ExpectedFilesTotal
	})
	s.ExpectedTestsHitRate = hitRate(treatment, func(t BenchmarkTrial) (int, int) {
		return t.ExpectedTestsHit, t.ExpectedTestsTotal
	})
	s.ExpectedDocsHitRate = hitRate(treatment, func(t BenchmarkTrial) (int, int) {
		return t.ExpectedDocsHit, t.ExpectedDocsTotal
	})
	return s
}

func avgDuration(trials []BenchmarkTrial) int64 {
	if len(trials) == 0 {
		return 0
	}
	var total int64
	for _, trial := range trials {
		total += trial.DurationMS
	}
	return total / int64(len(trials))
}

func avgContextBytes(trials []BenchmarkTrial) int {
	if len(trials) == 0 {
		return 0
	}
	var total int
	for _, trial := range trials {
		total += trial.ContextBytes
	}
	return total / len(trials)
}

func hitRate(trials []BenchmarkTrial, pick func(BenchmarkTrial) (int, int)) float64 {
	var hits, total int
	for _, trial := range trials {
		h, t := pick(trial)
		hits += h
		total += t
	}
	if total == 0 {
		return 0
	}
	return float64(hits) / float64(total)
}

func normalizeExpectedPaths(paths []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, path := range paths {
		path = strings.TrimSpace(strings.ReplaceAll(path, "\\", "/"))
		path = strings.TrimPrefix(path, "./")
		if path == "" || seen[path] {
			continue
		}
		seen[path] = true
		out = append(out, path)
	}
	sort.Strings(out)
	return out
}

func countHits(expected, actual []string) int {
	if len(expected) == 0 || len(actual) == 0 {
		return 0
	}
	actualSet := map[string]bool{}
	for _, path := range actual {
		actualSet[path] = true
	}
	var hits int
	for _, path := range expected {
		if actualSet[path] {
			hits++
		}
	}
	return hits
}
