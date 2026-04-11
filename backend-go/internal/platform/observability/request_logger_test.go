package observability

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
	"hotspot/backend-go/internal/platform/requestmeta"
)

func TestRequestLoggerMiddlewareUsesRequestmetaClientIP(t *testing.T) {
	if err := requestmeta.InitTrustedProxies(""); err != nil {
		t.Fatalf("init trusted proxies: %v", err)
	}
	core, logs := observer.New(zapcore.InfoLevel)
	logger := &Logger{base: zap.New(core).Sugar()}
	handler := RequestIDMiddleware(RequestLoggerMiddleware(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/system/health", http.NoBody)
	req.RemoteAddr = "173.245.48.5:443"
	req.Header.Set("CF-Connecting-IP", "8.8.8.8")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	entries := logs.FilterMessage("request completed").All()
	if len(entries) != 1 {
		t.Fatalf("expected one log entry, got %d", len(entries))
	}
	if got := entries[0].ContextMap()["client_ip"]; got != "8.8.8.8" {
		t.Fatalf("unexpected client ip: %v", got)
	}
}
