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

// defaultPoolMaxConns 与 config.defaultPostgresMaxConns 保持一致：用于直接
// 构造 config.Config（未经 config.Load）时兜底，避免池尺寸为 0 导致启动失败。
const defaultPoolMaxConns = 25

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
		// Pool sizing（pgx 默认 MaxConns=4 对生产偏低）。默认值与 env 覆盖
		// 集中在 config 包：POSTGRES_MAX_CONNS / POSTGRES_MIN_CONNS。
		// 多实例部署时各实例独立持池，需按实例数下调以免撞上 Postgres
		// max_connections（默认 100）。零值回退到默认，避免直接构造
		// config.Config 的调用方（测试/嵌入场景）拿到非法池尺寸。
		maxConns := cfg.PostgresMaxConns
		if maxConns <= 0 {
			maxConns = defaultPoolMaxConns
		}
		minConns := cfg.PostgresMinConns
		if minConns < 0 {
			minConns = 0
		}
		if minConns > maxConns {
			minConns = maxConns
		}
		poolConfig.MaxConns = int32(maxConns)
		poolConfig.MinConns = int32(minConns)
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
