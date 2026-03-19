package middlewares_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	"backend-go/internal/middlewares"
)

// TestRequestID_setsHeaderAndContext verifies that RequestID middleware
// injects a non-empty request id into both header and context.
func TestRequestID_setsHeaderAndContext(t *testing.T) {
	r := chi.NewRouter()
	r.Use(middlewares.RequestID)

	r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
		if middleware.GetReqID(r.Context()) == "" {
			t.Errorf("expected non-empty request id in context")
		}
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if got := w.Header().Get("X-Request-ID"); got == "" {
		t.Fatalf("expected X-Request-ID header to be set")
	}
}

// TestLogging_writesRequestLogWithRequestID verifies that Logging middleware
// emits a log entry containing the request id field.
func TestLogging_writesRequestLogWithRequestID(t *testing.T) {
	core, recorded := observer.New(zap.InfoLevel)
	logger := zap.New(core)

	r := chi.NewRouter()
	r.Use(middlewares.RequestID)
	r.Use(middlewares.Logging(logger))

	r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	logs := recorded.All()
	if len(logs) == 0 {
		t.Fatalf("expected at least one log entry, got 0")
	}

	foundReqID := false
	for _, field := range logs[0].Context {
		if field.Key == "request_id" && field.String != "" {
			foundReqID = true
			break
		}
	}
	if !foundReqID {
		t.Fatalf("expected log entry to contain non-empty request_id field")
	}
}

// TestRecovery_convertsPanicsToApiError verifies that Recovery middleware
// catches panics and returns a valid API error envelope with HTTP 200.
func TestRecovery_convertsPanicsToApiError(t *testing.T) {
	logger := zap.NewNop()

	r := chi.NewRouter()
	r.Use(middlewares.RequestID)
	r.Use(middlewares.Recovery(logger))

	r.Get("/panic", func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	})

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200 from recovery, got %d", w.Code)
	}
	if w.Body.Len() == 0 {
		t.Fatalf("expected non-empty JSON body from recovery")
	}
}

// TestTimeout_passesThroughWithinDeadline ensures that Timeout middleware
// does not interfere with handlers that complete within deadline.
func TestTimeout_passesThroughWithinDeadline(t *testing.T) {
	logger := zap.NewNop()

	r := chi.NewRouter()
	r.Use(middlewares.RequestID)
	r.Use(middlewares.Recovery(logger))
	r.Use(middlewares.Timeout(200 * time.Millisecond))

	r.Get("/ok", func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-time.After(50 * time.Millisecond):
			w.WriteHeader(http.StatusOK)
		case <-r.Context().Done():
			t.Fatalf("context cancelled before handler completed: %v", r.Context().Err())
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}
