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
	"errors"
	"io"
	"mime"
	"net/http"
	"strings"
)

const ReportsPath = "/v1/anonymous-telemetry/reports"

const maxRequestBodyBytes int64 = 2 << 20

// Handler returns the collector HTTP surface.
func Handler(store FoundationTelemetryStore, expectedHost string) http.Handler {
	expectedHost = strings.TrimSpace(expectedHost)
	mux := http.NewServeMux()
	mux.HandleFunc(ReportsPath, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			writeHTTPError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		if r.Host != expectedHost {
			writeHTTPError(w, http.StatusBadRequest, "invalid request")
			return
		}
		origins := r.Header.Values("Origin")
		if len(origins) > 1 || (len(origins) == 1 && origins[0] != "http://"+expectedHost) {
			writeHTTPError(w, http.StatusForbidden, "origin not allowed")
			return
		}
		mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil || mediaType != "application/json" {
			writeHTTPError(w, http.StatusUnsupportedMediaType, "application/json required")
			return
		}
		if store == nil {
			writeHTTPError(w, http.StatusServiceUnavailable, "collector unavailable")
			return
		}
		if r.Context().Err() != nil {
			writeHTTPError(w, http.StatusBadRequest, "invalid request")
			return
		}
		if r.ContentLength > maxRequestBodyBytes {
			writeHTTPError(w, http.StatusRequestEntityTooLarge, "request too large")
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
		data, err := io.ReadAll(r.Body)
		if err != nil {
			var maxBytesErr *http.MaxBytesError
			if errors.As(err, &maxBytesErr) {
				writeHTTPError(w, http.StatusRequestEntityTooLarge, "request too large")
				return
			}
			writeHTTPError(w, http.StatusBadRequest, "invalid request")
			return
		}
		batch, err := DecodeBatch(data)
		if err != nil {
			writeHTTPError(w, http.StatusBadRequest, "invalid request")
			return
		}
		if r.Context().Err() != nil {
			writeHTTPError(w, http.StatusBadRequest, "invalid request")
			return
		}
		for _, report := range batch.Reports {
			if err := store.SaveReport(r.Context(), report); err != nil {
				writeHTTPError(w, http.StatusInternalServerError, "storage failed")
				return
			}
			for _, pattern := range AggregatesFromReport(report) {
				if err := store.UpsertPattern(r.Context(), pattern); err != nil {
					writeHTTPError(w, http.StatusInternalServerError, "storage failed")
					return
				}
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]int{"accepted": len(batch.Reports)})
	})
	return mux
}

func writeHTTPError(w http.ResponseWriter, status int, message string) {
	http.Error(w, message, status)
}
