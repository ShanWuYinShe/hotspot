package config

import "testing"

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
