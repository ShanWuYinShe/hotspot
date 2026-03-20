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
	if err := factory.DB.Ping(context.Background()); err != nil {
		t.Fatalf("unexpected ping error: %v", err)
	}
}
