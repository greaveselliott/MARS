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
)

const (
	defaultContextMaxFiles              = 20000
	defaultContextMaxAutoRefreshChanges = 250
	defaultContextMaxChangedPaths       = 12
	defaultContextMaxSymbols            = 12
	defaultContextMaxTests              = 10
	defaultContextMaxDocs               = 8
	defaultContextMaxFeatures           = 8
	defaultContextMaxTickets            = 8
	DefaultImpactPreflightMaxPaths      = 8
)

// ContextOptions controls the compact graph bundle injected into Mars role context.
type ContextOptions struct {
	Refresh               bool
	DBPath                string
	MaxFiles              int
	MaxAutoRefreshChanges int
	MaxChangedPaths       int
	MaxSymbols            int
	MaxTests              int
	MaxDocs               int
	MaxFeatures           int
	MaxTickets            int
}

// ContextResult is the rendered graph bundle plus the structured facts used to build it.
type ContextResult struct {
	Text      string
	Status    Status
	Index     IndexResult
	Impact    ImpactResult
	Refreshed bool
}

// ToolAllowed reports whether a role allowlist can use the code graph directly.
func ToolAllowed(tools []string) bool {
	for _, name := range tools {
		switch strings.TrimSpace(name) {
		case "code_index", "code_search", "code_snippet", "code_trace", "code_impact":
			return true
		}
	}
	return false
}

// ImpactPreflightArgs returns bounded code_impact arguments for agent preflight.
// Large dirty sets are already summarized in the compact context block; replaying
// them as a full tool result can waste the model window before the first turn.
func ImpactPreflightArgs(graph ContextResult, maxPaths int) (string, bool) {
	if maxPaths <= 0 {
		maxPaths = DefaultImpactPreflightMaxPaths
	}
	paths := cleanRelPaths(graph.Impact.ChangedPaths)
	if len(paths) > maxPaths {
		return "", false
	}
	if len(paths) == 0 {
		return `{}`, true
	}
	raw, err := json.Marshal(struct {
		Paths []string `json:"paths"`
	}{Paths: paths})
	if err != nil {
		return "", false
	}
	return string(raw), true
}

// UnavailableContext returns a short explicit block when the graph cannot be built.
func UnavailableContext(err error) string {
	msg := "unknown error"
	if err != nil && strings.TrimSpace(err.Error()) != "" {
		msg = strings.TrimSpace(err.Error())
	}
	return fmt.Sprintf("freshness: unavailable\nreason: %s\noperator_hint: code graph unavailable; use the narrowest possible file reads and rerun code_index when the underlying issue is repaired.", msg)
}

// BuildContext creates the bounded code graph routing block used by both CLI
// runs and the deployed Mars orchestrator. It writes only Mars DB state.
func BuildContext(ctx context.Context, repoRoot string, opts ContextOptions) (ContextResult, error) {
	opts = normalizeContextOptions(opts)
	store, err := Open(repoRoot, opts.DBPath)
	if err != nil {
		return ContextResult{}, err
	}
	defer store.Close()

	status, err := store.Status(ctx)
	if err != nil {
		return ContextResult{}, err
	}

	var index IndexResult
	var refreshed bool
	if opts.Refresh && shouldRefreshForContext(status, opts.MaxAutoRefreshChanges) {
		index, err = store.Index(ctx, IndexOptions{MaxFiles: opts.MaxFiles})
		if err != nil {
			return ContextResult{}, err
		}
		refreshed = true
		status, err = store.Status(ctx)
		if err != nil {
			return ContextResult{}, err
		}
	}

	impact, err := store.Impact(ctx, nil, "")
	if err != nil {
		return ContextResult{}, err
	}

	text := renderContext(status, index, impact, refreshed, opts)
	return ContextResult{Text: text, Status: status, Index: index, Impact: impact, Refreshed: refreshed}, nil
}

func normalizeContextOptions(opts ContextOptions) ContextOptions {
	if opts.MaxFiles <= 0 {
		opts.MaxFiles = defaultContextMaxFiles
	}
	if opts.MaxAutoRefreshChanges <= 0 {
		opts.MaxAutoRefreshChanges = defaultContextMaxAutoRefreshChanges
	}
	if opts.MaxChangedPaths <= 0 {
		opts.MaxChangedPaths = defaultContextMaxChangedPaths
	}
	if opts.MaxSymbols <= 0 {
		opts.MaxSymbols = defaultContextMaxSymbols
	}
	if opts.MaxTests <= 0 {
		opts.MaxTests = defaultContextMaxTests
	}
	if opts.MaxDocs <= 0 {
		opts.MaxDocs = defaultContextMaxDocs
	}
	if opts.MaxFeatures <= 0 {
		opts.MaxFeatures = defaultContextMaxFeatures
	}
	if opts.MaxTickets <= 0 {
		opts.MaxTickets = defaultContextMaxTickets
	}
	return opts
}

func shouldRefreshForContext(status Status, maxChanges int) bool {
	switch status.Status {
	case FreshnessMissing:
		return true
	case FreshnessStale:
		return status.StaleFiles+status.NewFiles <= maxChanges
	default:
		return false
	}
}

func renderContext(status Status, index IndexResult, impact ImpactResult, refreshed bool, opts ContextOptions) string {
	var b strings.Builder
	fmt.Fprintf(&b, "freshness: %s\n", status.Status)
	fmt.Fprintf(&b, "indexed: files=%d symbols=%d edges=%d stale_files=%d new_files=%d\n", status.Files, status.Symbols, status.Edges, status.StaleFiles, status.NewFiles)
	if refreshed {
		fmt.Fprintf(&b, "index_refresh: files_seen=%d indexed=%d removed=%d duration_ms=%d\n", index.FilesSeen, index.FilesIndexed, index.FilesRemoved, index.DurationMS)
	} else if status.Status == FreshnessStale {
		fmt.Fprintf(&b, "repair: run `mars-harness tools run code_index --repo <repo> --trust observer --args-json '{}'` before relying on stale relationships\n")
	}
	appendStringList(&b, "changed_paths", prioritizePaths(impact.ChangedPaths), opts.MaxChangedPaths)
	if status.Status == FreshnessStale && !refreshed {
		b.WriteString("impacted_symbols:\n")
		b.WriteString("- omitted: graph relationships are stale; run code_index before relying on symbol, docs, feature, or ticket impact\n")
		b.WriteString("operator_hint: use code_index or the narrowest possible source reads; disclose stale state before making impact claims.\n")
		return strings.TrimSpace(b.String())
	}
	appendSymbolList(&b, prioritizeSymbols(impact.Symbols), opts.MaxSymbols)
	appendStringList(&b, "likely_tests", impact.Tests, opts.MaxTests)
	appendStringList(&b, "docs", impact.Docs, opts.MaxDocs)
	appendStringList(&b, "features", impact.Features, opts.MaxFeatures)
	appendStringList(&b, "tickets", impact.Tickets, opts.MaxTickets)
	b.WriteString("operator_hint: use code_search, code_snippet, code_trace, and code_impact before grep, file_search, or broad file_read; disclose stale or partial state.\n")
	return strings.TrimSpace(b.String())
}

func appendStringList(b *strings.Builder, title string, values []string, limit int) {
	fmt.Fprintf(b, "%s:\n", title)
	if len(values) == 0 {
		b.WriteString("- none\n")
		return
	}
	for _, value := range limitStrings(values, limit) {
		fmt.Fprintf(b, "- %s\n", value)
	}
	if len(values) > limit {
		fmt.Fprintf(b, "- ... %d more\n", len(values)-limit)
	}
}

func appendSymbolList(b *strings.Builder, symbols []Symbol, limit int) {
	b.WriteString("impacted_symbols:\n")
	if len(symbols) == 0 {
		b.WriteString("- none\n")
		return
	}
	for _, sym := range limitSymbols(symbols, limit) {
		fmt.Fprintf(b, "- %s %s:%d-%d\n", sym.QualifiedName, sym.Path, sym.StartLine, sym.EndLine)
	}
	if len(symbols) > limit {
		fmt.Fprintf(b, "- ... %d more\n", len(symbols)-limit)
	}
}

func limitStrings(values []string, limit int) []string {
	if limit <= 0 || len(values) <= limit {
		return values
	}
	return values[:limit]
}

func limitSymbols(values []Symbol, limit int) []Symbol {
	if limit <= 0 || len(values) <= limit {
		return values
	}
	return values[:limit]
}

func prioritizePaths(paths []string) []string {
	out := append([]string(nil), paths...)
	sort.SliceStable(out, func(i, j int) bool {
		ri, rj := contextPathRank(out[i]), contextPathRank(out[j])
		if ri == rj {
			return out[i] < out[j]
		}
		return ri < rj
	})
	return out
}

func prioritizeSymbols(symbols []Symbol) []Symbol {
	out := append([]Symbol(nil), symbols...)
	sort.SliceStable(out, func(i, j int) bool {
		ri, rj := contextPathRank(out[i].Path), contextPathRank(out[j].Path)
		if ri == rj {
			if out[i].Path == out[j].Path {
				return out[i].StartLine < out[j].StartLine
			}
			return out[i].Path < out[j].Path
		}
		return ri < rj
	})
	return out
}

func contextPathRank(path string) int {
	path = strings.TrimSpace(path)
	switch {
	case path == "":
		return 100
	case strings.HasPrefix(path, ".harness/"):
		return 90
	case strings.HasPrefix(path, ".git"):
		return 90
	case strings.HasPrefix(path, "docs/tickets/"):
		return 70
	case strings.HasPrefix(path, "docs/"):
		return 60
	case path == "AGENTS.md", path == "README.md", strings.HasPrefix(path, ".github/"):
		return 40
	case path == "go.mod", path == "go.sum", path == "package.json", path == "pnpm-lock.yaml", path == "package-lock.json", path == "yarn.lock":
		return 20
	default:
		return 0
	}
}
