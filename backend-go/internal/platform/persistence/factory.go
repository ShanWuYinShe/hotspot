package persistence

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"hotspot/backend-go/internal/platform/config"
)

type Factory struct {
	Config Config
	DB     DB
}

func NewFactory(cfg config.Config) *Factory {
	factory := &Factory{Config: Config{PostgresDSN: cfg.PostgresDSN}}
	if cfg.PostgresDSN != "" {
		db := NewPostgresDB(cfg.PostgresDSN)
		if pool, err := pgxpool.New(context.Background(), cfg.PostgresDSN); err == nil {
			db.Pool = pool
		}
		factory.DB = db
	} else {
		factory.DB = NewInMemoryDB()
	}
	return factory
}
