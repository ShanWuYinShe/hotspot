package requestmeta

import (
	"net/http/httptest"
	"testing"
)

func preserveTrustedProxies(t *testing.T) {
	t.Helper()
	original := cloneCIDRs(currentTrustedProxyCIDRs())
	t.Cleanup(func() {
		trustedProxyCIDRs.Store(original)
	})
}

func TestClientIPUsesCloudflareHeaderForDefaultTrustedProxy(t *testing.T) {
	preserveTrustedProxies(t)
	if err := InitTrustedProxies(""); err != nil {
		t.Fatalf("init trusted proxies: %v", err)
	}
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "173.245.48.5:443"
	req.Header.Set("CF-Connecting-IP", "8.8.8.8")
	if got := ClientIP(req); got != "8.8.8.8" {
		t.Fatalf("unexpected client ip: %s", got)
	}
}

func TestClientIPUsesTrustedProxyHeaderWhenConfigured(t *testing.T) {
	preserveTrustedProxies(t)
	if err := InitTrustedProxies("10.0.0.0/8"); err != nil {
		t.Fatalf("init trusted proxies: %v", err)
	}
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "10.1.2.3:443"
	req.Header.Set("CF-Connecting-IP", "8.8.8.8")
	if got := ClientIP(req); got != "8.8.8.8" {
		t.Fatalf("unexpected client ip: %s", got)
	}
}

func TestClientIPFallsBackToRemoteWhenNotTrustedProxy(t *testing.T) {
	preserveTrustedProxies(t)
	if err := InitTrustedProxies("10.0.0.0/8"); err != nil {
		t.Fatalf("init trusted proxies: %v", err)
	}
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "203.0.113.10:443"
	req.Header.Set("CF-Connecting-IP", "8.8.8.8")
	if got := ClientIP(req); got != "203.0.113.10" {
		t.Fatalf("unexpected client ip: %s", got)
	}
}

func TestClientIPUsesFirstNonPrivateForwardedIPForTrustedProxy(t *testing.T) {
	preserveTrustedProxies(t)
	if err := InitTrustedProxies("10.0.0.0/8"); err != nil {
		t.Fatalf("init trusted proxies: %v", err)
	}
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "10.1.2.3:443"
	req.Header.Set("X-Forwarded-For", "10.0.0.1, 172.16.0.2, 8.8.4.4")
	if got := ClientIP(req); got != "8.8.4.4" {
		t.Fatalf("unexpected client ip: %s", got)
	}
}

func TestClientCountryUsesHeaderForTrustedProxy(t *testing.T) {
	preserveTrustedProxies(t)
	if err := InitTrustedProxies(""); err != nil {
		t.Fatalf("init trusted proxies: %v", err)
	}
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "173.245.48.5:443"
	req.Header.Set("CF-IPCountry", "cn")
	if got := ClientCountry(req); got != "CN" {
		t.Fatalf("unexpected client country: %s", got)
	}
}

func TestTrustedProxyMetadataHeadersIgnoredForDirectRequest(t *testing.T) {
	preserveTrustedProxies(t)
	if err := InitTrustedProxies("10.0.0.0/8"); err != nil {
		t.Fatalf("init trusted proxies: %v", err)
	}
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "203.0.113.10:443"
	req.Header.Set("CF-IPCountry", "US")
	req.Header.Set("CF-Ray", "abc123")
	req.Header.Set("X-Forwarded-Proto", "https")
	if got := ClientCountry(req); got != "" {
		t.Fatalf("expected empty client country, got %s", got)
	}
	if got := RequestRay(req); got != "" {
		t.Fatalf("expected empty ray id, got %s", got)
	}
	if got := ForwardedProto(req); got != "" {
		t.Fatalf("expected empty forwarded proto, got %s", got)
	}
}

func TestRequestRayAndForwardedProtoUseTrustedProxyHeaders(t *testing.T) {
	preserveTrustedProxies(t)
	if err := InitTrustedProxies("10.0.0.0/8"); err != nil {
		t.Fatalf("init trusted proxies: %v", err)
	}
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "10.1.2.3:443"
	req.Header.Set("CF-Ray", "abc123-HKG")
	req.Header.Set("X-Forwarded-Proto", "HTTPS")
	if got := RequestRay(req); got != "abc123-HKG" {
		t.Fatalf("unexpected ray id: %s", got)
	}
	if got := ForwardedProto(req); got != "https" {
		t.Fatalf("unexpected forwarded proto: %s", got)
	}
}

func TestInitTrustedProxiesRejectsInvalidCIDR(t *testing.T) {
	preserveTrustedProxies(t)
	if err := InitTrustedProxies("invalid-cidr"); err == nil {
		t.Fatal("expected invalid cidr error")
	}
}
