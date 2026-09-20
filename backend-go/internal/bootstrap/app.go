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
	resourceguardChecker := newResourceGuardChecker(persistenceFactory)
	resourceguard.SetDefaultChecker(resourceguardChecker)
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
	// resourceguard missing 缓存定期清扫：条目仅在同 key 再次探测时惰性删除，
	// 扫描器打随机 ID 可让 map 无上限增长。
	if dbChecker, ok := resourceguardChecker.(*resourceguard.DBChecker); ok {
		dbChecker.StartSweeper(rootCtx, time.Minute)
	}
	// Prune expired idempotency records from the in-memory store. Get/Reserve
	// already apply lazy expiry so this is purely about bounding map growth.
	// Redis 可用时切共享状态存储：幂等记录（同一 Idempotency-Key 只执行一次）
	// 与限流计数（多实例共享阈值，进程内各实例独立会放大 N 倍）。
	if redisClient != nil {
		idempotency.SetStore(idempotency.NewRedisStore(redisClient))
		ratelimit.SetRedisStore(redisClient)
		logger.Info("redis shared stores enabled", "idempotency", "redis", "ratelimit", "redis")
	} else {
		logger.Info("redis not configured, idempotency running in single-instance mode")
	}
	idempotency.StartSweeper(rootCtx)
	modules, adminRepo := BuildModulesWithRedisPinger(rootCtx, cfg, persistenceFactory, redisClient)
	// bridge 在下方 redisClient 块内创建（instanceID 需同时注入 resumeStore）
	var bridge *realtime.RedisBridge
	router := NewHTTPRouterWithLogger(modules, logger)
	// /metrics 自定义采集器装配：WS 连接/订阅规模 + DB 连接池统计
	//（HTTP 请求指标经 RequestMetricsMiddleware 在路由层采集）。
	observability.Register(realtime.NewHubCollector(realtime.DefaultHub()))
	if persistenceFactory.DB != nil {
		observability.Register(NewDBPoolCollector(persistenceFactory.DB))
	}
	rootMux := http.NewServeMux()
	rootMux.Handle("/", router)
	userRepo := users.NewPostgresRepository(persistenceFactory)
	groupRepo := groups.NewPostgresRepository(persistenceFactory)
	userService := users.NewServiceWithRepo(userRepo)
	realtimeHandler := realtime.NewHandlerWithContext(rootCtx, platformauth.DefaultService(), realtimeUserRepo{repo: userRepo}, realtimeGroupRepo{repo: groupRepo})
	realtimeHandler.SetActorProvider(realtimeActorProvider{svc: userService})
	// WS 握手封禁检查：与 HTTP RequiredAuth 同源（admin repo IsBanned）、同口径（fail-open）
	realtimeHandler.SetBanAuthorizer(realtimeBanAuthorizer{repo: adminRepo})
	// 运行时 presence 落库（首条连接 online / 最后一条断开 offline）：groups/search
	// 的 presence_status='online' 统计以此为准（写失败仅记日志，不中断连接）。
	realtimeHandler.SetPresenceWriter(realtimePresenceWriter{repo: userRepo})
	// resume 增量恢复（spec L156, L159）：Redis 可用时创建跨实例事件存储并注入
	// Hub（seq 分配/事件累积）与 Handler（token 生命周期/回放）。Redis 不可用时
	// Hub/Handler 的 resumeStore 为 nil，退化为全量重放（spec L161）。
	// instanceID 提前生成并同时注入 resumeStore 与 bridge（审计 perf P1-4）：
	// 组合脚本发布的信封以 origin 标记本实例，bridge 靠它跳过自收消息——
	// 两处 origin 不一致会导致本实例重复投递，必须同源。
	if redisClient != nil {
		instanceID := fmt.Sprintf("hs-%d", time.Now().UnixNano())
		resumeStore := realtime.NewRedisResumeStore(redisClient, time.Duration(cfg.ResumeTTLSeconds)*time.Second, instanceID)
		realtime.DefaultHub().SetResumeStore(resumeStore)
		realtimeHandler.SetResumeStore(resumeStore)
		bridge = initRedisBridgeFromClient(cfg, logger, redisClient, instanceID)
	}
	// 握手 per-IP 限流（审查 L-6）：升级请求发生在鉴权之前，须与其他 HTTP
	// 入口同样限流；APP_ENV=test 同口径绕过。
	rootMux.Handle("/ws/v1", ratelimit.WebsocketHandshakeMiddleware(realtimeHandler))

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
	// 先排空 WS 连接（1001 Going Away + 有界等待）：server.Shutdown 不等待
	// hijacked 连接，若先关 Redis/DB，存活连接的后续操作全部报错，客户端也要
	// 等 ping 超时才发现断线。排空只需给各连接发 close frame，不依赖后端服务。
	if remaining := realtime.DefaultHub().DrainConnections(3 * time.Second); remaining > 0 {
		a.logger.Warn("ws drain left active connections", "remaining", remaining)
	}
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
	// fail-slow（审计 perf P2-5）：Redis 故障/网络分区时命令必须在秒级失败并
	// 降级，而不是挂到调用方 ctx 超时（resume 2s / ratelimit 2s）甚至无限阻塞，
	// 把延迟放大到所有共享 Redis 的热路径。dial 500ms / read+write 1s 只约束
	// 单命令，健康实例不受影响。
	opts.DialTimeout = 500 * time.Millisecond
	opts.ReadTimeout = time.Second
	opts.WriteTimeout = time.Second
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
// instanceID 必须与注入 RedisResumeStore 的 origin 同源（组合脚本发布的信封
// 靠它跳过自收消息，见 NewApp 内的创建顺序）。
// Takes ownership of rdb: when no bridge is created (nil client or init
// failure) the caller is responsible for closing rdb; when a bridge is created
// its Close() closes the underlying client.
func initRedisBridgeFromClient(cfg config.Config, logger *observability.Logger, rdb *redis.Client, instanceID string) *realtime.RedisBridge {
	if rdb == nil {
		return nil
	}
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
