/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/self-reflective-telemetry.md
- docs/features/F-012-self-improvement-loop.md
*/
package foundationtelemetry

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const ReportsPath = "/v1/anonymous-telemetry/reports"

// Handler returns the collector HTTP surface.
func Handler(store FoundationTelemetryStore) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc(ReportsPath, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if store == nil {
			http.Error(w, "foundation telemetry store unavailable", http.StatusServiceUnavailable)
			return
		}
		data, err := io.ReadAll(io.LimitReader(r.Body, 2<<20))
		if err != nil {
			http.Error(w, fmt.Sprintf("read request: %v", err), http.StatusBadRequest)
			return
		}
		batch, err := DecodeBatch(data)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		for _, report := range batch.Reports {
			if err := store.SaveReport(r.Context(), report); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			for _, pattern := range AggregatesFromReport(report) {
				if err := store.UpsertPattern(r.Context(), pattern); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]int{"accepted": len(batch.Reports)})
	})
	return mux
}
