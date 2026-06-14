/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/context-efficiency.md
- docs/design-docs/orchestrated-organization-layer.md
- docs/design-docs/pipeline-engine.md
- docs/features/F-005-agent-execution-runtime.md
- docs/features/F-006-queue-and-orchestration.md
*/
package serve

import (
	"fmt"
	"testing"

	"github.com/greaveselliott/mars-harness/internal/codeintel"
)

func TestRecordCodeGraphContextCounters(t *testing.T) {
	counts := map[string]int{}
	recordCodeGraphContextCounters(counts, codeintel.ContextResult{
		Text:      "freshness: fresh",
		Status:    codeintel.Status{StaleFiles: 2, NewFiles: 3},
		Index:     codeintel.IndexResult{FilesIndexed: 4},
		Refreshed: true,
	}, nil)

	if counts["codeintel:context_injected"] != 1 {
		t.Fatalf("expected context injection counter, got %+v", counts)
	}
	if counts["codeintel:context_refreshed"] != 1 || counts["codeintel:context_files_indexed"] != 4 {
		t.Fatalf("expected refresh counters, got %+v", counts)
	}
	if counts["codeintel:context_stale_files"] != 2 || counts["codeintel:context_new_files"] != 3 {
		t.Fatalf("expected freshness counters, got %+v", counts)
	}
	if counts["codeintel:context_bytes"] != len("freshness: fresh") {
		t.Fatalf("expected context bytes, got %+v", counts)
	}

	recordCodeGraphContextCounters(counts, codeintel.ContextResult{Text: "freshness: unavailable"}, fmt.Errorf("boom"))
	if counts["codeintel:context_unavailable"] != 1 {
		t.Fatalf("expected unavailable counter, got %+v", counts)
	}
}

func TestCodeGraphPreflightRequiresFreshImpactTool(t *testing.T) {
	fresh := codeintel.ContextResult{Status: codeintel.Status{Status: codeintel.FreshnessFresh}}
	runtime := codeintel.NewRuntime(true, "test")
	got := codeGraphPreflight([]string{"code_impact"}, fresh, nil, runtime)
	if len(got) != 1 || got[0].Name != "code_impact" {
		t.Fatalf("expected code_impact preflight, got %+v", got)
	}
	stale := codeintel.ContextResult{Status: codeintel.Status{Status: codeintel.FreshnessStale}}
	if got := codeGraphPreflight([]string{"code_impact"}, stale, nil, runtime); len(got) != 0 {
		t.Fatalf("expected no stale preflight, got %+v", got)
	}
	if got := codeGraphPreflight([]string{"file_read"}, fresh, nil, runtime); len(got) != 0 {
		t.Fatalf("expected no preflight without code_impact allowlist, got %+v", got)
	}
	if got := codeGraphPreflight([]string{"code_impact"}, fresh, fmt.Errorf("boom"), runtime); len(got) != 0 {
		t.Fatalf("expected no preflight on graph error, got %+v", got)
	}
	if got := codeGraphPreflight([]string{"code_impact"}, fresh, nil, codeintel.NewRuntime(false, "test")); len(got) != 0 {
		t.Fatalf("expected no preflight when code-intel is disabled, got %+v", got)
	}
	unavailable := codeGraphRuntimeForTrace(runtime, []string{"code_impact"}, fmt.Errorf("boom"))
	if unavailable.Mode != codeintel.ModeUnavailable {
		t.Fatalf("expected unavailable trace mode, got %+v", unavailable)
	}
	bounded := codeintel.ContextResult{
		Status: codeintel.Status{Status: codeintel.FreshnessFresh},
		Impact: codeintel.ImpactResult{ChangedPaths: []string{"main.go"}},
	}
	got = codeGraphPreflight([]string{"code_impact"}, bounded, nil, runtime)
	if len(got) != 1 || got[0].ArgsJSON != `{"paths":["main.go"]}` {
		t.Fatalf("expected bounded path preflight, got %+v", got)
	}
	large := codeintel.ContextResult{Status: codeintel.Status{Status: codeintel.FreshnessFresh}}
	for i := 0; i < codeintel.DefaultImpactPreflightMaxPaths+1; i++ {
		large.Impact.ChangedPaths = append(large.Impact.ChangedPaths, fmt.Sprintf("file-%d.go", i))
	}
	if got := codeGraphPreflight([]string{"code_impact"}, large, nil, runtime); len(got) != 0 {
		t.Fatalf("expected large dirty set to skip preflight, got %+v", got)
	}
}
