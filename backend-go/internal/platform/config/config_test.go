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
		RedisDSN:    "redis://test-redis:6379/1",
	}
	if cfg.PostgresDSN != "postgres://test-db/hotspot_test" {
		t.Fatalf("unexpected postgres dsn: %s", cfg.PostgresDSN)
	}
	if cfg.RedisDSN != "redis://test-redis:6379/1" {
		t.Fatalf("unexpected redis dsn: %s", cfg.RedisDSN)
	}
}

func TestLoadLeavesPostgresDSNEmptyWhenUnset(t *testing.T) {
	oldPostgres := os.Getenv("POSTGRES_DSN")
	oldRedis := os.Getenv("REDIS_DSN")
	defer func() {
		if oldPostgres == "" {
			_ = os.Unsetenv("POSTGRES_DSN")
		} else {
			_ = os.Setenv("POSTGRES_DSN", oldPostgres)
		}
		if oldRedis == "" {
			_ = os.Unsetenv("REDIS_DSN")
		} else {
			_ = os.Setenv("REDIS_DSN", oldRedis)
		}
	}()

	_ = os.Unsetenv("POSTGRES_DSN")
	_ = os.Unsetenv("REDIS_DSN")

	cfg := Load()
	if cfg.PostgresDSN != "" {
		t.Fatalf("expected empty postgres dsn, got %q", cfg.PostgresDSN)
	}
	if cfg.RedisDSN != "" {
		t.Fatalf("expected empty redis dsn, got %q", cfg.RedisDSN)
	}
}
