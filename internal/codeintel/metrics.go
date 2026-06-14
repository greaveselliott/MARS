/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/context-efficiency.md
- docs/design-docs/tools-glossary.md
- docs/product-specs/product-surface.md
- docs/features/F-005-agent-execution-runtime.md
*/
package codeintel

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/greaveselliott/mars-harness/internal/trace"
)

const (
	ModeEnabled     = "enabled"
	ModeDisabled    = "disabled"
	ModeUnavailable = "unavailable"
)

const (
	metricContextInjected    = "codeintel:context_injected"
	metricContextUnavailable = "codeintel:context_unavailable"
	metricContextRefreshed   = "codeintel:context_refreshed"
	metricContextBytes       = "codeintel:context_bytes"
	metricContextFiles       = "codeintel:context_files_indexed"
	metricContextStale       = "codeintel:context_stale_files"
	metricContextNew         = "codeintel:context_new_files"

	metricToolCalls       = "codeintel:tool_calls"
	metricToolOutputBytes = "codeintel:output_bytes"
	metricBroadCalls      = "repo_exploration:broad_tool_calls"
	metricBroadBytes      = "repo_exploration:broad_output_bytes"
	metricBulkReads       = "repo_exploration:bulk_file_read_calls"
	metricBroadShell      = "repo_exploration:broad_shell_search_calls"
)

// Runtime captures the resolved automatic code-intel behavior for a job.
type Runtime struct {
	Enabled bool
	Mode    string
	Source  string
}

// NewRuntime returns a normalized runtime mode for automatic graph assistance.
func NewRuntime(enabled bool, source string) Runtime {
	mode := ModeDisabled
	if enabled {
		mode = ModeEnabled
	}
	source = strings.TrimSpace(source)
	if source == "" {
		source = "default"
	}
	return Runtime{Enabled: enabled, Mode: mode, Source: source}
}

// RecordContextCounters merges graph-context build facts into session counters.
func RecordContextCounters(counts map[string]int, graph ContextResult, graphErr error) {
	if counts == nil {
		return
	}
	if graphErr != nil {
		counts[metricContextUnavailable]++
		if graph.Text != "" {
			counts[metricContextBytes] += len(graph.Text)
		}
		return
	}
	counts[metricContextInjected]++
	counts[metricContextBytes] += len(graph.Text)
	counts[metricContextStale] += graph.Status.StaleFiles
	counts[metricContextNew] += graph.Status.NewFiles
	if graph.Refreshed {
		counts[metricContextRefreshed]++
		counts[metricContextFiles] += graph.Index.FilesIndexed
	}
}

// MetricsOptions controls local aggregation from persisted trace summaries.
type MetricsOptions struct {
	RepoPath   string
	DBPath     string
	WindowDays int
}

// MetricsReport summarizes local code-intel efficiency evidence.
type MetricsReport struct {
	RepoPath              string         `json:"repo_path"`
	DBPath                string         `json:"db_path"`
	WindowDays            int            `json:"window_days"`
	Jobs                  int            `json:"jobs"`
	GraphEnabledJobs      int            `json:"graph_enabled_jobs"`
	GraphDisabledJobs     int            `json:"graph_disabled_jobs"`
	GraphUnavailableJobs  int            `json:"graph_unavailable_jobs"`
	CodeIntelToolCalls    int            `json:"codeintel_tool_calls"`
	CodeIntelOutputBytes  int            `json:"codeintel_output_bytes"`
	CodeIntelContextBytes int            `json:"codeintel_context_bytes"`
	CodeIntelRefreshes    int            `json:"codeintel_refreshes"`
	BroadExplorationCalls int            `json:"broad_exploration_calls"`
	BroadExplorationBytes int            `json:"broad_exploration_bytes"`
	BulkFileReadCalls     int            `json:"bulk_file_read_calls"`
	BroadShellSearchCalls int            `json:"broad_shell_search_calls"`
	LLMCalls              int            `json:"llm_calls"`
	ToolInvocations       int            `json:"tool_invocations"`
	WallMs                int64          `json:"wall_ms"`
	TokenEstimate         int            `json:"token_estimate"`
	UnusedGraphJobs       int            `json:"unused_graph_jobs"`
	ModeSources           map[string]int `json:"mode_sources,omitempty"`
}

// Metrics aggregates persisted trace summaries from the per-repo Mars DB.
func Metrics(ctx context.Context, opts MetricsOptions) (MetricsReport, error) {
	repoPath := strings.TrimSpace(opts.RepoPath)
	dbPath := strings.TrimSpace(opts.DBPath)
	if dbPath == "" {
		dbPath = DefaultDBPath(repoPath)
	}
	if opts.WindowDays <= 0 {
		opts.WindowDays = 30
	}
	store, err := trace.OpenStore(dbPath)
	if err != nil {
		return MetricsReport{}, err
	}
	defer store.Close()

	records, err := store.ListSince(ctx, time.Now().UTC().AddDate(0, 0, -opts.WindowDays))
	if err != nil {
		return MetricsReport{}, err
	}
	report := MetricsReport{
		RepoPath:    repoPath,
		DBPath:      dbPath,
		WindowDays:  opts.WindowDays,
		ModeSources: map[string]int{},
	}
	for _, rec := range records {
		var summary trace.Summary
		if err := json.Unmarshal([]byte(rec.SummaryJSON), &summary); err != nil {
			return MetricsReport{}, fmt.Errorf("codeintel metrics: parse trace %s summary: %w", rec.TraceID, err)
		}
		report.Jobs++
		report.LLMCalls += summary.LLMCalls
		report.ToolInvocations += summary.ToolInvocations
		report.WallMs += summary.WallMs
		report.TokenEstimate += summary.TotalTokens
		counts := summary.ToolCounts
		if counts == nil {
			counts = map[string]int{}
		}
		mode := ""
		source := ""
		if summary.CodeIntel != nil {
			mode = strings.TrimSpace(summary.CodeIntel.Mode)
			source = strings.TrimSpace(summary.CodeIntel.Source)
		}
		unavailable := mode == ModeUnavailable || counts[metricContextUnavailable] > 0
		switch {
		case unavailable:
			report.GraphUnavailableJobs++
		case mode == ModeEnabled:
			report.GraphEnabledJobs++
			if counts[metricToolCalls] == 0 {
				report.UnusedGraphJobs++
			}
		case mode == ModeDisabled:
			report.GraphDisabledJobs++
		}
		if source != "" {
			report.ModeSources[source]++
		}
		report.CodeIntelToolCalls += counts[metricToolCalls]
		report.CodeIntelOutputBytes += counts[metricToolOutputBytes]
		report.CodeIntelContextBytes += counts[metricContextBytes]
		report.CodeIntelRefreshes += counts[metricContextRefreshed]
		report.BroadExplorationCalls += counts[metricBroadCalls]
		report.BroadExplorationBytes += counts[metricBroadBytes]
		report.BulkFileReadCalls += counts[metricBulkReads]
		report.BroadShellSearchCalls += counts[metricBroadShell]
	}
	if len(report.ModeSources) == 0 {
		report.ModeSources = nil
	}
	return report, nil
}

// SortedModeSources renders mode-source buckets in stable order.
func SortedModeSources(sources map[string]int) []string {
	if len(sources) == 0 {
		return nil
	}
	keys := make([]string, 0, len(sources))
	for key := range sources {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	lines := make([]string, 0, len(keys))
	for _, key := range keys {
		lines = append(lines, fmt.Sprintf("%s=%d", key, sources[key]))
	}
	return lines
}
