package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func probeResponse(t *testing.T, handler http.Handler) (int, map[string]any) {
	t.Helper()
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	var body map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("decode readiness response: %v", err)
	}
	return rr.Code, body
}

func okPing(context.Context) error { return nil }

func failPing(context.Context) error { return errors.New("ping failed") }

func TestReadinessProbeReady(t *testing.T) {
	code, body := probeResponse(t, newReadinessProbe(
		readinessCheck{name: "database", required: true, ping: okPing},
		readinessCheck{name: "redis", ping: okPing},
	))
	if code != http.StatusOK || body["status"] != "ok" {
		t.Fatalf("expected ready, got code=%d body=%#v", code, body)
	}
}

func TestReadinessProbeDatabaseDownIs503(t *testing.T) {
	code, body := probeResponse(t, newReadinessProbe(
		readinessCheck{name: "database", required: true, ping: failPing},
		readinessCheck{name: "redis", ping: okPing},
	))
	if code != http.StatusServiceUnavailable || body["status"] != "unavailable" {
		t.Fatalf("expected unavailable, got code=%d body=%#v", code, body)
	}
	checks, _ := body["checks"].(map[string]any)
	if checks["database"] != "down" {
		t.Fatalf("expected database down, got %#v", body)
	}
}

func TestReadinessProbeRedisDownIsDegradedNotUnavailable(t *testing.T) {
	code, body := probeResponse(t, newReadinessProbe(
		readinessCheck{name: "database", required: true, ping: okPing},
		readinessCheck{name: "redis", ping: failPing},
	))
	if code != http.StatusOK || body["status"] != "degraded" {
		t.Fatalf("expected degraded 200, got code=%d body=%#v", code, body)
	}
}

func TestReadinessProbeNilPingIsSkipped(t *testing.T) {
	code, body := probeResponse(t, newReadinessProbe(readinessCheck{name: "redis", ping: nil}))
	if code != http.StatusOK || body["status"] != "ok" {
		t.Fatalf("expected ready for unconfigured dependency, got code=%d body=%#v", code, body)
	}
}

func TestRouterRegistersReadyz(t *testing.T) {
	router := NewHTTPRouter(nil)
	code, body := probeResponse(t, router)
	data, _ := body["data"].(map[string]any)
	if code != http.StatusOK || data["status"] != "ok" {
		t.Fatalf("default readiness should be ok, got code=%d body=%#v", code, body)
	}
}

func TestRedisReadinessPingNilClient(t *testing.T) {
	if redisReadinessPing(nil) != nil {
		t.Fatal("nil redis client must yield nil ping (treated as ok)")
	}
}
