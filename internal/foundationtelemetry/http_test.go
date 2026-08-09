/*
MarsDocSync:
docs:
- docs/design-docs/code-documentation-map.md
- docs/design-docs/self-reflective-telemetry.md
- docs/features/F-012-self-improvement-loop.md
*/
package foundationtelemetry

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const testCollectorHost = "127.0.0.1:9092"

type countingTelemetryStore struct {
	saveCalls   int
	upsertCalls int
	saveErr     error
	upsertErr   error
}

func (s *countingTelemetryStore) SaveReport(context.Context, AnonymousReport) error {
	s.saveCalls++
	return s.saveErr
}

func (s *countingTelemetryStore) UpsertPattern(context.Context, AggregatedPattern) error {
	s.upsertCalls++
	return s.upsertErr
}

func (*countingTelemetryStore) PatternsSince(context.Context, time.Time) ([]AggregatedPattern, error) {
	return nil, nil
}

func validBatchBytes(t *testing.T) []byte {
	t.Helper()
	data, err := json.Marshal(ReportBatch{Reports: []AnonymousReport{testReport(t)}})
	require.NoError(t, err)
	return data
}

func collectorRequest(t *testing.T, method string, body []byte) *http.Request {
	t.Helper()
	req := httptest.NewRequest(method, ReportsPath, bytes.NewReader(body))
	req.Host = testCollectorHost
	req.Header.Set("Content-Type", "application/json")
	return req
}

func TestHandlerRejectsPreAdmissionFailuresWithoutStoreCalls(t *testing.T) {
	valid := validBatchBytes(t)
	canceledContext, cancel := context.WithCancel(context.Background())
	cancel()

	tests := []struct {
		name   string
		status int
		make   func() *http.Request
	}{
		{name: "method", status: http.StatusMethodNotAllowed, make: func() *http.Request {
			return collectorRequest(t, http.MethodGet, valid)
		}},
		{name: "host", status: http.StatusBadRequest, make: func() *http.Request {
			req := collectorRequest(t, http.MethodPost, valid)
			req.Host = "192.0.2.10:9092"
			return req
		}},
		{name: "origin", status: http.StatusForbidden, make: func() *http.Request {
			req := collectorRequest(t, http.MethodPost, valid)
			req.Header.Set("Origin", "https://hostile.invalid")
			return req
		}},
		{name: "media_type", status: http.StatusUnsupportedMediaType, make: func() *http.Request {
			req := collectorRequest(t, http.MethodPost, valid)
			req.Header.Set("Content-Type", "text/plain")
			return req
		}},
		{name: "oversized", status: http.StatusRequestEntityTooLarge, make: func() *http.Request {
			return collectorRequest(t, http.MethodPost, bytes.Repeat([]byte("x"), int(maxRequestBodyBytes)+1))
		}},
		{name: "oversized_stream", status: http.StatusRequestEntityTooLarge, make: func() *http.Request {
			req := collectorRequest(t, http.MethodPost, bytes.Repeat([]byte("x"), int(maxRequestBodyBytes)+1))
			req.ContentLength = -1
			return req
		}},
		{name: "trailing_value", status: http.StatusBadRequest, make: func() *http.Request {
			return collectorRequest(t, http.MethodPost, append(append([]byte{}, valid...), []byte(` {}`)...))
		}},
		{name: "invalid_envelope", status: http.StatusBadRequest, make: func() *http.Request {
			return collectorRequest(t, http.MethodPost, []byte(`{"raw":"secret-body-candidate"}`))
		}},
		{name: "canceled", status: http.StatusBadRequest, make: func() *http.Request {
			return collectorRequest(t, http.MethodPost, valid).WithContext(canceledContext)
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &countingTelemetryStore{}
			rec := httptest.NewRecorder()
			Handler(store, testCollectorHost).ServeHTTP(rec, tt.make())
			require.Equal(t, tt.status, rec.Code)
			require.Zero(t, store.saveCalls)
			require.Zero(t, store.upsertCalls)
			require.NotContains(t, rec.Body.String(), "hostile.invalid")
			require.NotContains(t, rec.Body.String(), "192.0.2.10")
			require.NotContains(t, rec.Body.String(), "secret-body-candidate")
		})
	}
}

func TestHandlerAcceptsExactBoundaryAndIgnoresForwardedAuthority(t *testing.T) {
	data := validBatchBytes(t)
	require.LessOrEqual(t, int64(len(data)), maxRequestBodyBytes)
	data = append(data, bytes.Repeat([]byte(" "), int(maxRequestBodyBytes)-len(data))...)
	store := &countingTelemetryStore{}
	req := collectorRequest(t, http.MethodPost, data)
	req.Header.Set("Origin", "http://"+testCollectorHost)
	req.Header.Set("X-Forwarded-Host", "hostile.invalid")
	rec := httptest.NewRecorder()

	Handler(store, testCollectorHost).ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, 1, store.saveCalls)
	require.Equal(t, 1, store.upsertCalls)
	require.JSONEq(t, `{"accepted":1}`, rec.Body.String())
}

func TestHandlerReturnsFixedStorageFailures(t *testing.T) {
	sentinel := errors.New("secret-db-path-/owner/private/intake.db")
	for _, tc := range []struct {
		name  string
		store *countingTelemetryStore
	}{
		{name: "save", store: &countingTelemetryStore{saveErr: sentinel}},
		{name: "upsert", store: &countingTelemetryStore{upsertErr: sentinel}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			Handler(tc.store, testCollectorHost).ServeHTTP(rec, collectorRequest(t, http.MethodPost, validBatchBytes(t)))
			require.Equal(t, http.StatusInternalServerError, rec.Code)
			require.Equal(t, "storage failed\n", rec.Body.String())
			require.NotContains(t, rec.Body.String(), sentinel.Error())
			require.NotContains(t, rec.Body.String(), "accepted")
		})
	}
}

func TestHandlerRejectsNilStore(t *testing.T) {
	rec := httptest.NewRecorder()
	Handler(nil, testCollectorHost).ServeHTTP(rec, collectorRequest(t, http.MethodPost, validBatchBytes(t)))
	require.Equal(t, http.StatusServiceUnavailable, rec.Code)
	require.Equal(t, "collector unavailable\n", rec.Body.String())
}

func TestDecodeBatchRejectsTrailingData(t *testing.T) {
	_, err := DecodeBatch(append(validBatchBytes(t), []byte(" trailing-secret-fragment")...))
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "trailing data") || strings.Contains(err.Error(), "multiple JSON values"))
}
