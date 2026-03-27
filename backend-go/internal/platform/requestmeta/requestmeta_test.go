package requestmeta

import (
	"net/http/httptest"
	"testing"
)

func TestClientIPUsesCloudflareHeaderOnlyForCloudflareRemote(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "173.245.48.5:443"
	req.Header.Set("CF-Connecting-IP", "8.8.8.8")
	if got := ClientIP(req); got != "8.8.8.8" {
		t.Fatalf("unexpected client ip: %s", got)
	}
}

func TestClientIPFallsBackToRemoteWhenNotCloudflare(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "203.0.113.10:443"
	req.Header.Set("CF-Connecting-IP", "8.8.8.8")
	if got := ClientIP(req); got != "203.0.113.10" {
		t.Fatalf("unexpected client ip: %s", got)
	}
}

func TestClientIPUsesFirstNonPrivateForwardedIP(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "173.245.48.5:443"
	req.Header.Set("X-Forwarded-For", "10.0.0.1, 172.16.0.2, 8.8.4.4")
	if got := ClientIP(req); got != "8.8.4.4" {
		t.Fatalf("unexpected client ip: %s", got)
	}
}
