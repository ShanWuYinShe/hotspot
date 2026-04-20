package persistence

import (
	"context"

	"github.com/jackc/pgx/v5"
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
		poolConfig, err := pgxpool.ParseConfig(cfg.PostgresDSN)
		if err == nil {
			poolConfig.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol
			if pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig); err == nil {
				db.Pool = pool
			}
		}
		factory.DB = db
	} else {
		factory.DB = NewInMemoryDB()
	}
	return factory
}
