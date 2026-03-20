package persistence

import "hotspot/backend-go/internal/platform/config"

type Factory struct {
	Config Config
	DB     DB
}

func NewFactory(cfg config.Config) *Factory {
	factory := &Factory{Config: Config{PostgresDSN: cfg.PostgresDSN, RedisDSN: cfg.RedisDSN}}
	if cfg.PostgresDSN != "" {
		factory.DB = NewPostgresDB(cfg.PostgresDSN)
	} else {
		factory.DB = NewInMemoryDB()
	}
	return factory
}
