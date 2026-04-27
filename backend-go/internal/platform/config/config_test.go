package config

import (
	"os"
	"testing"
)

func TestConfigHTTPAddr(t *testing.T) {
	cfg := Config{Host: "127.0.0.1", Port: "9000"}
	if got := cfg.HTTPAddr(); got != "127.0.0.1:9000" {
		t.Fatalf("unexpected http addr: %s", got)
	}
}

func TestConfigCarriesDatasourceValues(t *testing.T) {
	cfg := Config{
		PostgresDSN: "postgres://test-db/hotspot_test",
	}
	if cfg.PostgresDSN != "postgres://test-db/hotspot_test" {
		t.Fatalf("unexpected postgres dsn: %s", cfg.PostgresDSN)
	}
}

func TestLoadLeavesPostgresDSNEmptyWhenUnset(t *testing.T) {
	oldPostgres := os.Getenv("POSTGRES_DSN")
	oldTrustedProxyCIDRs := os.Getenv("TRUSTED_PROXY_CIDRS")
	defer func() {
		if oldPostgres == "" {
			_ = os.Unsetenv("POSTGRES_DSN")
		} else {
			_ = os.Setenv("POSTGRES_DSN", oldPostgres)
		}
		if oldTrustedProxyCIDRs == "" {
			_ = os.Unsetenv("TRUSTED_PROXY_CIDRs")
		} else {
			_ = os.Setenv("TRUSTED_PROXY_CIDRs", oldTrustedProxyCIDRs)
		}
	}()

	_ = os.Unsetenv("POSTGRES_DSN")
	_ = os.Unsetenv("TRUSTED_PROXY_CIDRs")

	cfg := Load()
	if cfg.PostgresDSN != "" {
		t.Fatalf("expected empty postgres dsn, got %q", cfg.PostgresDSN)
	}
	if cfg.TrustedProxyCIDRs != "" {
		t.Fatalf("expected empty trusted proxy cidrs, got %q", cfg.TrustedProxyCIDRs)
	}
}

func TestLoadCarriesTrustedProxyCIDRs(t *testing.T) {
	oldValue := os.Getenv("TRUSTED_PROXY_CIDRS")
	defer func() {
		if oldValue == "" {
			_ = os.Unsetenv("TRUSTED_PROXY_CIDRS")
		} else {
			_ = os.Setenv("TRUSTED_PROXY_CIDRS", oldValue)
		}
	}()

	_ = os.Setenv("TRUSTED_PROXY_CIDRS", "10.0.0.0/8,192.168.0.0/16")
	cfg := Load()
	if cfg.TrustedProxyCIDRs != "10.0.0.0/8,192.168.0.0/16" {
		t.Fatalf("unexpected trusted proxy cidrs: %q", cfg.TrustedProxyCIDRs)
	}
}

func TestLoadCarriesS3PublicEndpointSettings(t *testing.T) {
	oldEndpoint := os.Getenv("S3_PUBLIC_ENDPOINT")
	oldForcePathStyle := os.Getenv("S3_PUBLIC_FORCE_PATH_STYLE")
	defer func() {
		if oldEndpoint == "" {
			_ = os.Unsetenv("S3_PUBLIC_ENDPOINT")
		} else {
			_ = os.Setenv("S3_PUBLIC_ENDPOINT", oldEndpoint)
		}
		if oldForcePathStyle == "" {
			_ = os.Unsetenv("S3_PUBLIC_FORCE_PATH_STYLE")
		} else {
			_ = os.Setenv("S3_PUBLIC_FORCE_PATH_STYLE", oldForcePathStyle)
		}
	}()

	_ = os.Setenv("S3_PUBLIC_ENDPOINT", "http://localhost:8333")
	_ = os.Setenv("S3_PUBLIC_FORCE_PATH_STYLE", "false")

	cfg := Load()
	if cfg.S3PublicEndpoint != "http://localhost:8333" {
		t.Fatalf("unexpected s3 public endpoint: %q", cfg.S3PublicEndpoint)
	}
	if cfg.S3PublicForcePathStyle {
		t.Fatal("expected s3 public force path style to be false")
	}
}
