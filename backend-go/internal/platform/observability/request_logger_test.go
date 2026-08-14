package observability

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"hotspot/backend-go/internal/platform/requestmeta"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
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

	req := httptest.NewRequest(http.MethodPost, "/public/v1/system/health", http.NoBody)
	req.RemoteAddr = "173.245.48.5:443"
	req.Header.Set("CF-Connecting-IP", "8.8.8.8")
	req.Header.Set("CF-IPCountry", "US")
	req.Header.Set("CF-Ray", "abc123-HKG")
	req.Header.Set("X-Forwarded-Proto", "https")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	entries := logs.FilterMessage("request completed").All()
	if len(entries) != 1 {
		t.Fatalf("expected one log entry, got %d", len(entries))
	}
	context := entries[0].ContextMap()
	if got := context["client_ip"]; got != "8.8.8.8" {
		t.Fatalf("unexpected client ip: %v", got)
	}
	if got := context["cf_country"]; got != "US" {
		t.Fatalf("unexpected cf country: %v", got)
	}
	if got := context["cf_ray"]; got != "abc123-HKG" {
		t.Fatalf("unexpected cf ray: %v", got)
	}
	if got := context["forwarded_proto"]; got != "https" {
		t.Fatalf("unexpected forwarded proto: %v", got)
	}
}
