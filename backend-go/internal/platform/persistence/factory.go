package persistence

import (
	"context"
	"fmt"
	"time"

	"hotspot/backend-go/internal/platform/config"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
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
		if err != nil {
			// DSN 解析失败是确定性启动错误，带病启动只会让后续每个请求都失败。
			// panic 消息不含 DSN，避免泄漏连接串中的密码。
			panic(fmt.Sprintf("parse postgres DSN: %v", err))
		}
		poolConfig.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol
		// Pool sizing (pgx defaults are MaxConns=4, too low for production).
		// Tune these via env vars POSTGRES_MAX_CONNS / POSTGRES_MIN_CONNS if needed.
		poolConfig.MaxConns = 25
		poolConfig.MinConns = 2
		poolConfig.MaxConnLifetime = 30 * time.Minute
		poolConfig.MaxConnIdleTime = 5 * time.Minute
		pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
		if err != nil {
			panic(fmt.Sprintf("create postgres pool: %v", err))
		}
		db.Pool = pool
		factory.DB = db
	} else {
		factory.DB = NewInMemoryDB()
	}
	return factory
}
