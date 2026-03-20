package persistence

import (
	"context"
	"testing"

	"hotspot/backend-go/internal/platform/config"
)

func TestNewFactoryUsesDatasourceConfig(t *testing.T) {
	factory := NewFactory(config.Config{PostgresDSN: "postgres://test", RedisDSN: "redis://test"})
	if factory.Config.PostgresDSN != "postgres://test" || factory.Config.RedisDSN != "redis://test" {
		t.Fatalf("unexpected factory config: %#v", factory.Config)
	}
	if _, ok := factory.DB.(*PostgresDB); !ok {
		t.Fatalf("expected PostgresDB, got %#v", factory.DB)
	}
	_ = context.Background()
}
