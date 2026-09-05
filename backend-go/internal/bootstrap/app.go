package bootstrap

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"

	"hotspot/backend-go/internal/modules/groups"
	"hotspot/backend-go/internal/modules/users"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/idempotency"
	platformmigrate "hotspot/backend-go/internal/platform/migrate"
	"hotspot/backend-go/internal/platform/modulekit"
	"hotspot/backend-go/internal/platform/observability"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/ratelimit"
	"hotspot/backend-go/internal/platform/realtime"
	"hotspot/backend-go/internal/platform/requestmeta"
	"hotspot/backend-go/internal/platform/resourceguard"
	"hotspot/backend-go/migrations"
)

type App struct {
	cfg         config.Config
	logger      *observability.Logger
	router      http.Handler
	modules     []modulekit.Module
	persistence *persistence.Factory
	bridge      *realtime.RedisBridge
	realtime    *realtime.Handler
	rootCtx     context.Context
	rootCancel  context.CancelFunc
}

func NewApp(cfg config.Config) (*App, error) {
	if err := requestmeta.InitTrustedProxies(cfg.TrustedProxyCIDRs); err != nil {
		return nil, err
	}
	logger, err := observability.NewLogger(cfg.AppEnv, cfg.LogLevel)
	if err != nil {
		return nil, err
	}

	persistenceFactory := persistence.NewFactory(cfg)
	if err := ensureSchema(cfg, persistenceFactory); err != nil {
		return nil, err
	}
	resourceguard.SetDefaultChecker(newResourceGuardChecker(persistenceFactory))
	// Create the redis client (if configured) before building modules so the
	// system health endpoint can probe it. The same client is handed to the
	// realtime bridge below.
	redisClient := newRedisClient(cfg, logger)
	// Root context governs long-lived goroutines started by platform packages
	// and modules (the realtime typing-throttle sweeper, the ratelimit store
	// sweeper, the in-memory auth session/refresh store sweeper, the idempotency
	// store sweeper, the auth login-tracker sweeper). It is created before
	// building modules so module-owned workers can bind to it, and is cancelled
	// in Shutdown so they exit cleanly.
	rootCtx, rootCancel := context.WithCancel(context.Background())
	ratelimit.StartSweeper(rootCtx)
	platformauth.StartSweeper(rootCtx)
	// Prune expired idempotency records from the in-memory store. Get/Reserve
	// already apply lazy expiry so this is purely about bounding map growth.
	// Redis 可用时切共享状态存储：幂等记录（同一 Idempotency-Key 只执行一次）
	// 与限流计数（多实例共享阈值，进程内各实例独立会放大 N 倍）。
	if redisClient != nil {
		idempotency.SetStore(idempotency.NewRedisStore(redisClient))
		ratelimit.SetRedisStore(redisClient)
	}
	idempotency.StartSweeper(rootCtx)
	modules, adminRepo := BuildModulesWithRedisPinger(rootCtx, cfg, persistenceFactory, redisPinger(redisClient))
	bridge := initRedisBridgeFromClient(cfg, logger, redisClient)
	router := NewHTTPRouterWithLogger(modules, logger)
	rootMux := http.NewServeMux()
	rootMux.Handle("/", router)
	userRepo := users.NewPostgresRepository(persistenceFactory)
	groupRepo := groups.NewPostgresRepository(persistenceFactory)
	userService := users.NewServiceWithRepo(userRepo)
	realtimeHandler := realtime.NewHandlerWithContext(rootCtx, platformauth.DefaultService(), realtimeUserService{svc: userService}, realtimeUserRepo{repo: userRepo}, realtimeGroupRepo{repo: groupRepo})
	realtimeHandler.SetActorProvider(realtimeUserService{svc: userService})
	// WS 握手封禁检查：与 HTTP RequiredAuth 同源（admin repo IsBanned）、同口径（fail-open）
	realtimeHandler.SetBanAuthorizer(realtimeBanAuthorizer{repo: adminRepo})
	// resume 增量恢复（spec L156, L159）：Redis 可用时创建跨实例事件存储并注入
	// Hub（seq 分配/事件累积）与 Handler（token 生命周期/回放）。Redis 不可用时
	// Hub/Handler 的 resumeStore 为 nil，退化为全量重放（spec L161）。
	if redisClient != nil {
		resumeStore := realtime.NewRedisResumeStore(redisClient, time.Duration(cfg.ResumeTTLSeconds)*time.Second)
		realtime.DefaultHub().SetResumeStore(resumeStore)
		realtimeHandler.SetResumeStore(resumeStore)
	}
	rootMux.Handle("/ws/v1", realtimeHandler)

	return &App{
		cfg:         cfg,
		logger:      logger,
		router:      rootMux,
		modules:     modules,
		persistence: persistenceFactory,
		bridge:      bridge,
		realtime:    realtimeHandler,
		rootCtx:     rootCtx,
		rootCancel:  rootCancel,
	}, nil
}

func (a *App) Router() http.Handler {
	return a.router
}

func (a *App) Logger() *observability.Logger {
	return a.logger
}

func (a *App) Shutdown() {
	if a.bridge != nil {
		a.bridge.Close()
	}
	// Stop the realtime typing-throttle sweeper and wait for it to exit so it
	// doesn't outlive the process (and so tests don't leak goroutines).
	if a.realtime != nil {
		a.realtime.Close()
	}
	// Cancel the root context to stop any other platform-level background
	// goroutines (e.g. the ratelimit store sweeper).
	if a.rootCancel != nil {
		a.rootCancel()
	}
	// Close the Postgres pool too — otherwise a graceful shutdown leaks the
	// pool and its connections until process exit.
	if a.persistence != nil && a.persistence.DB != nil {
		_ = a.persistence.DB.Close()
	}
}

func ensureSchema(cfg config.Config, factory *persistence.Factory) error {
	if cfg.PostgresDSN == "" || factory == nil || factory.DB == nil {
		return nil
	}
	queryable, ok := factory.DB.(persistence.QueryableDB)
	if !ok {
		return fmt.Errorf("postgres persistence does not support schema bootstrap")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := platformmigrate.EnsureSchema(ctx, queryable, migrations.Files); err != nil {
		return fmt.Errorf("ensure backend-go schema: %w", err)
	}
	return nil
}

func newResourceGuardChecker(factory *persistence.Factory) resourceguard.Checker {
	if factory == nil || factory.DB == nil {
		return resourceguard.NoopChecker{}
	}
	queryable, ok := factory.DB.(persistence.QueryableDB)
	if !ok {
		return resourceguard.NoopChecker{}
	}
	return resourceguard.NewDBChecker(queryable)
}

// newRedisClient creates and ping-verifies a redis client from cfg.RedisURL.
// Returns nil when redis is not configured or unreachable, in which case the
// realtime hub runs in single-instance mode and the health endpoint skips the
// redis probe.
func newRedisClient(cfg config.Config, logger *observability.Logger) *redis.Client {
	if cfg.RedisURL == "" {
		logger.Info("redis not configured, realtime hub running in single-instance mode")
		return nil
	}
	opts, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		logger.Error("invalid redis url, falling back to single-instance mode", "err", err)
		return nil
	}
	rdb := redis.NewClient(opts)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		logger.Error("redis connection failed, falling back to single-instance mode", "err", err)
		rdb.Close()
		return nil
	}
	return rdb
}

// redisPinger adapts a redis client into the health-probe function signature
// expected by the system module. Nil client => nil pinger (probe is skipped).
func redisPinger(rdb *redis.Client) func(context.Context) error {
	if rdb == nil {
		return nil
	}
	return func(ctx context.Context) error { return rdb.Ping(ctx).Err() }
}

// initRedisBridgeFromClient starts the realtime pub/sub bridge using rdb.
// Takes ownership of rdb: when no bridge is created (nil client or init
// failure) the caller is responsible for closing rdb; when a bridge is created
// its Close() closes the underlying client.
func initRedisBridgeFromClient(cfg config.Config, logger *observability.Logger, rdb *redis.Client) *realtime.RedisBridge {
	if rdb == nil {
		return nil
	}
	instanceID := fmt.Sprintf("hs-%d", time.Now().UnixNano())
	bridge, err := realtime.NewRedisBridge(rdb, instanceID, realtime.DefaultHub())
	if err != nil {
		logger.Error("redis bridge init failed, falling back to single-instance mode", "err", err)
		rdb.Close()
		return nil
	}
	realtime.DefaultHub().SetPublisher(bridge)
	logger.Info("redis bridge initialized", "instance", instanceID)
	return bridge
}
