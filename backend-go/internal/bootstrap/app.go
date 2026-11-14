package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"

	adminmodule "hotspot/backend-go/internal/modules/admin"
	"hotspot/backend-go/internal/modules/groups"
	"hotspot/backend-go/internal/modules/notify"
	"hotspot/backend-go/internal/modules/system"
	"hotspot/backend-go/internal/modules/users"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/hotcache"
	"hotspot/backend-go/internal/platform/idempotency"
	"hotspot/backend-go/internal/platform/idgen"
	platformmigrate "hotspot/backend-go/internal/platform/migrate"
	"hotspot/backend-go/internal/platform/modulekit"
	"hotspot/backend-go/internal/platform/observability"
	"hotspot/backend-go/internal/platform/outbox"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/ratelimit"
	"hotspot/backend-go/internal/platform/realtime"
	"hotspot/backend-go/internal/platform/requestmeta"
	"hotspot/backend-go/internal/platform/resourceguard"
	"hotspot/backend-go/migrations"
)

const (
	// warmPGroongaConns 预热连接数：覆盖典型并发（c=10）并留余量；上限受池
	// MaxConns 约束。搜索量增大后新连接仍可能首查多一次规划，但不再集中尖刺。
	warmPGroongaConns = 16
	// warmPGroongaQuery 只触碰 PGroonga 索引，不读业务数据；字面量不命中任何行。
	warmPGroongaQuery = "SELECT 1 FROM users WHERE nickname &@ '__hotspot_warmup__' LIMIT 1"
)

type App struct {
	cfg         config.Config
	logger      *observability.Logger
	router      http.Handler
	modules     []modulekit.Module
	persistence *persistence.Factory
	bridge      *realtime.RedisBridge
	realtime    *realtime.Handler
	hub         *realtime.Hub
	hotCache    *hotcache.Cache
	// redis 是幂等/限流/热缓存/跨实例广播共享的 client。bridge.Close 只退订阅、
	// 明确不关共享 client（resume/health/缓存仍在用），生命周期归 App：Shutdown
	// 中在 bridge/realtime/hotCache 之后、DB 之前关闭。
	redis      *redis.Client
	rootCtx    context.Context
	rootCancel context.CancelFunc
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
	// PGroonga 首次查询规划按连接惰性初始化（实测每个新连接首查多 ~50-70ms）：
	// 在组合根预热最多 warmPGroongaConns 个池连接，把冷启动/扩容尖刺移出
	// 请求路径。失败只告警（搜索仍可用，仅首查多一次规划）。
	if db, ok := persistenceFactory.DB.(*persistence.PostgresDB); ok {
		warmCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		if err := db.WarmPlanner(warmCtx, warmPGroongaQuery, warmPGroongaConns); err != nil {
			logger.Warn("pgroonga planner warmup failed", "err", err)
		} else {
			logger.Info("pgroonga planner warmed", "conns", warmPGroongaConns)
		}
		cancel()
	}
	resourceguardChecker := newResourceGuardChecker(persistenceFactory)
	// Create the redis client (if configured) before building modules so the
	// system health endpoint can probe it. The same client is handed to the
	// realtime bridge below.
	// newRedisClient 只返回三态（未配置/不可用/可用），日志与指标由这里统一打：
	// 「未配置」是显式支持的单实例模式（Info），「配置了但不可用」是降级（Error），
	// 两者必须在日志里可区分（旧实现把后者也打成 Info「redis not configured」）。
	redisClient, redisStartupErr := newRedisClient(cfg)
	// Root context governs long-lived goroutines started by platform packages
	// and modules (the realtime typing-throttle sweeper, the ratelimit store
	// sweeper, the in-memory auth session/refresh store sweeper, the idempotency
	// store sweeper, the auth login-tracker sweeper). It is created before
	// building modules so module-owned workers can bind to it, and is cancelled
	// in Shutdown so they exit cleanly.
	rootCtx, rootCancel := context.WithCancel(context.Background())
	ratelimit.StartSweeper(rootCtx)
	platformAuth := platformauth.NewServiceWithPersistence(cfg, persistenceFactory)
	platformauth.StartSweeper(rootCtx, platformAuth)
	// resourceguard missing 缓存定期清扫：条目仅在同 key 再次探测时惰性删除，
	// 扫描器打随机 ID 可让 map 无上限增长。
	if dbChecker, ok := resourceguardChecker.(*resourceguard.DBChecker); ok {
		dbChecker.StartSweeper(rootCtx, time.Minute)
	}
	// Prune expired idempotency records from the in-memory store. Get/Reserve
	// already apply lazy expiry so this is purely about bounding map growth.
	// Redis 可用时切共享状态存储：幂等记录（同一 Idempotency-Key 只执行一次）
	// 与限流计数（多实例共享阈值，进程内各实例独立会放大 N 倍）。
	idemStore := idempotency.Store(idempotency.NewInMemoryStore())
	if redisClient != nil {
		idemStore = idempotency.NewRedisStore(redisClient)
		ratelimit.SetRedisStore(redisClient)
		logger.Info("redis shared stores enabled", "idempotency", "redis", "ratelimit", "redis")
	} else if redisStartupErr != nil {
		// 显式配置了 REDIS_URL 却连不上：幂等/限流退化为单实例、WS 跨实例广播
		// 与热缓存共享失效。这是**降级**而非可选默认模式，必须 Error 级并说明
		// 影响面——否则多实例部署下会悄悄出现「同一请求在 N 个实例各执行一次」
		// （幂等失效）与「限流阈值放大 N 倍」，线上无人察觉。
		logger.Error("redis configured but unavailable, degraded to single-instance mode",
			"err", redisStartupErr,
			"endpoint", redisEndpoint(cfg.RedisURL),
			"degraded", []string{"idempotency", "ratelimit", "realtime-fanout", "hotcache"},
		)
	} else {
		// 未配置 Redis 是显式支持的单实例模式（dev/单机部署），Info 级即可。
		logger.Info("redis not configured, idempotency, rate limiting and realtime fanout running in single-instance mode")
	}
	// 启动期降级状态指标（Gauge：持续状态，见 registerRedisStartupMetric）。
	registerRedisStartupMetric(redisStartupErr != nil)
	idempotency.StartSweeper(rootCtx, idemStore)
	// 热数据缓存（读穿 + 写失效）：users 设置与群成员关系；单一实例贯穿模块与
	// realtime 适配器（本地模式下失效才互通；配了 Redis 则多实例共享键）。
	hotCache := hotcache.New(redisClient)
	modules, adminRepo := BuildModulesWithRedisPinger(rootCtx, cfg, persistenceFactory, redisClient, hotCache, platformAuth)
	// bridge 在下方 redisClient 块内创建（instanceID 需同时注入 resumeStore）
	var bridge *realtime.RedisBridge
	readiness := newReadinessProbe(
		// DB 是硬依赖：不可达则摘流量（compose healthcheck/LB 依赖该信号）。
		readinessCheck{name: "database", required: true, ping: persistenceFactory.DB.Ping},
		// Redis 未配置时 ping 为 nil（视为 ok）；配置后故障只降级。启动探测失败
		// 时 client 已丢弃且不会重连，探针固定报 down（degraded），避免就绪面把
		// 「配置了但宕机」误报成全绿。
		readinessCheck{name: "redis", required: false, ping: redisReadinessProbe(redisClient, redisStartupErr)},
	)
	// 应用级实时 Hub：本 App 独占实例，经根中间件注入请求 context；
	// 不再依赖模块隐式读取进程级 DefaultHub。
	hub := realtime.NewHub()
	// 通用 outbox dispatcher：业务事务内写入的事件在 COMMIT 后由后台投递；
	// 进程崩溃重启后未处理行仍会补投。当前接入系统公告 WS 广播（topic 见
	// outbox.TopicSystemAnnouncement）。
	if persistenceFactory.DB != nil && cfg.PostgresDSN != "" {
		dispatcher := outbox.NewDispatcher(persistenceFactory, logger)
		dispatcher.Register(outbox.TopicSystemAnnouncement, systemAnnouncementOutboxHandler(hub))
		dispatcher.Register(outbox.TopicAdminReportHandled, adminReportHandledOutboxHandler(hub))
		dispatcher.Start(rootCtx)
	}
	router := NewHTTPRouterWithReadiness(modules, logger, readiness, resourceguardChecker, idemStore, platformAuth, adminRepo, hub)
	// /metrics 自定义采集器装配：WS 连接/订阅规模 + DB 连接池统计
	//（HTTP 请求指标经 RequestMetricsMiddleware 在路由层采集）。
	observability.Register(realtime.NewHubCollector(hub))
	if persistenceFactory.DB != nil {
		observability.Register(NewDBPoolCollector(persistenceFactory.DB))
	}
	rootMux := http.NewServeMux()
	rootMux.Handle("/", router)
	userRepo := users.NewPostgresRepository(persistenceFactory)
	groupRepo := groups.NewPostgresRepository(persistenceFactory)
	userService := users.NewServiceWithRepo(userRepo)
	realtimeHandler := realtime.NewHandlerWithContext(rootCtx, platformAuth, NewRealtimeUserRepo(userRepo, hotCache), NewRealtimeGroupRepo(groupRepo, hotCache))
	realtimeHandler.SetHub(hub)
	realtimeHandler.SetMaxConnections(cfg.WSMaxConnections)
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
	// 两处 origin 不一致会导致本实例重复投递，必须同源。用 idgen 随机后缀
	// 而非 UnixNano：时间戳在克隆 VM/时钟回拨下可重复，两实例同 ID 时 bridge
	// 会互相把对方消息当自收消息丢弃（跨实例事件静默丢失）。
	if redisClient != nil {
		instanceID := idgen.MustWithPrefix("hs-")
		resumeStore := realtime.NewRedisResumeStore(redisClient, time.Duration(cfg.ResumeTTLSeconds)*time.Second, instanceID)
		hub.SetResumeStore(resumeStore)
		realtimeHandler.SetResumeStore(resumeStore)
		bridge = initRedisBridgeFromClient(cfg, logger, redisClient, instanceID, hub)
	}
	// 握手 per-IP 限流（审查 L-6）：升级请求发生在鉴权之前，须与其他 HTTP
	// 入口同样限流；测试基建经 RATE_LIMIT_BYPASS=1 显式绕过。
	rootMux.Handle("/ws/v1", ratelimit.WebsocketHandshakeMiddleware(realtimeHandler))

	return &App{
		cfg:         cfg,
		logger:      logger,
		router:      rootMux,
		modules:     modules,
		persistence: persistenceFactory,
		bridge:      bridge,
		realtime:    realtimeHandler,
		hub:         hub,
		rootCtx:     rootCtx,
		hotCache:    hotCache,
		redis:       redisClient,
		rootCancel:  rootCancel,
	}, nil
}

// systemAnnouncementOutboxHandler 解码 admin 生产的公告广播 payload 并做一次
// 全体在线广播（与同步路径共用 notify.SystemAnnouncementRealtimeEvent，
// 保证 WS payload 结构一致）。
func systemAnnouncementOutboxHandler(hub *realtime.Hub) outbox.Handler {
	return func(_ context.Context, payload json.RawMessage) error {
		var event struct {
			Item     system.SystemAnnouncementItem `json:"item"`
			NotifyID string                        `json:"notify_id"`
		}
		if err := json.Unmarshal(payload, &event); err != nil {
			return fmt.Errorf("bootstrap: decode system announcement outbox payload: %w", err)
		}
		if event.Item.AnnouncementID == "" || event.NotifyID == "" {
			return fmt.Errorf("bootstrap: invalid system announcement outbox payload")
		}
		hub.BroadcastToAll(notify.SystemAnnouncementRealtimeEvent(event.Item, event.NotifyID))
		return nil
	}
}

// adminReportHandledOutboxHandler 把举报裁决结果 WS 广播补投到举报人连接：
// 通知行已在裁决事务内落库，这里只负责 COMMIT 后的实时投递（至少一次；
// 客户端按 notify_id 去重）。payload 形状与同步路径完全一致。
func adminReportHandledOutboxHandler(hub *realtime.Hub) outbox.Handler {
	return func(_ context.Context, payload json.RawMessage) error {
		var event adminmodule.ReportHandledBroadcastPayload
		if err := json.Unmarshal(payload, &event); err != nil {
			return fmt.Errorf("bootstrap: decode admin report outbox payload: %w", err)
		}
		if event.ReporterUserID == "" || event.ReportID == "" {
			return fmt.Errorf("bootstrap: invalid admin report outbox payload")
		}
		wsPayload := map[string]any{
			"notify_id":   event.NotifyID,
			"report_id":   event.ReportID,
			"target_type": event.TargetType,
			"target_id":   event.TargetID,
		}
		if event.Note != "" {
			wsPayload["note"] = event.Note
		}
		eventType := realtime.EventTypeReportResolved
		if event.Resolved {
			wsPayload["resolved_at"] = event.OccurredAt
		} else {
			eventType = realtime.EventTypeReportDismissed
			wsPayload["dismissed_at"] = event.OccurredAt
		}
		hub.BroadcastToUser(event.ReporterUserID, realtime.Event{Type: eventType, Payload: wsPayload})
		return nil
	}
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
	if remaining := a.hub.DrainConnections(3 * time.Second); remaining > 0 {
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
	// 热数据缓存的本地清扫协程由 c.stop 控制（不随 rootCtx）：必须显式关闭，
	// 否则优雅停机后 goroutine 仍存活（进程退出才回收），双重 Close 已改为幂等。
	if a.hotCache != nil {
		a.hotCache.Close()
	}
	// 共享 Redis client 最后关闭（必须在 bridge/realtime/hotCache 之后、DB 之前）：
	// bridge.Close 只退订阅、明确不关共享 client（resume/健康探针/缓存仍在用），
	// 生命周期归组合根；放在 DB 之前，保证数据库池是最后一个回收的资源。
	if a.redis != nil {
		if err := a.redis.Close(); err != nil {
			a.logger.Warn("close redis client failed", "err", err)
		}
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

// newRedisClient 根据 REDIS_URL 创建并 ping 校验 Redis client。
//
// 返回值语义（三态，日志由调用方统一打，便于单测）：
//   - (nil, nil)：未配置 Redis——单实例是显式支持的默认模式，Info 级说明即可；
//   - (nil, err)：**显式配置了但不可用**（URL 解析失败 / ping 失败）：幂等、限流、
//     热缓存与跨实例广播整体退化为单实例，调用方必须打 Error 级降级日志；
//   - (client, nil)：可用。
//
// 失败时丢弃 client（fail-fast 降级）：半死连接只会让每条热路径命令各自等超时。
// err 已脱敏（见 sanitizeRedisStartupErr），可直接落日志。
func newRedisClient(cfg config.Config) (*redis.Client, error) {
	if cfg.RedisURL == "" {
		return nil, nil
	}
	opts, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		return nil, fmt.Errorf("parse REDIS_URL: %w", sanitizeRedisStartupErr(err))
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
		_ = rdb.Close()
		return nil, fmt.Errorf("ping redis %s: %w", opts.Addr, sanitizeRedisStartupErr(err))
	}
	return rdb, nil
}

// sanitizeRedisStartupErr 脱敏 Redis 启动错误：net/url 的 *url.Error 会把
// **完整 URL（含口令）**拼进 Error()，直接落日志/告警即泄露凭据。这里只保留
// 底层原因（如 invalid port / missing port），其余错误原样返回——go-redis 的
// 连接错误只含 host:port，不含 userinfo。
func sanitizeRedisStartupErr(err error) error {
	var urlErr *url.Error
	if errors.As(err, &urlErr) && urlErr.Err != nil {
		return urlErr.Err
	}
	return err
}

// redisEndpoint 返回脱敏后的 Redis 端点（scheme://host[:port][/db]）用于日志
// 关联：运维需要知道连的是哪个实例，但 userinfo（用户名/口令）绝不能进日志。
// 解析失败时不回显原值，返回占位串。
func redisEndpoint(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return "<invalid redis endpoint>"
	}
	endpoint := u.Scheme + "://" + u.Host
	if u.Path != "" && u.Path != "/" {
		endpoint += u.Path
	}
	return endpoint
}

// redisReadinessProbe 返回就绪探针用的 Redis 检查函数：
//   - 未配置（无 client 且无启动错误）→ nil，探针按 ok 处理（单实例是支持的模式）；
//   - 已配置且可用 → 每次探测实时 Ping，失败计入 degraded 但不阻塞接流量——限流/
//     幂等在 Redis 故障时 fail-open（审计 ops P0-2）；
//   - 已配置但启动探测失败 → 固定返回启动错误：client 已丢弃且进程内不会重连，
//     就绪面必须持续显示 degraded，而不是误报全绿。
func redisReadinessProbe(client *redis.Client, startupErr error) func(context.Context) error {
	if client != nil {
		return func(ctx context.Context) error { return client.Ping(ctx).Err() }
	}
	if startupErr != nil {
		return func(context.Context) error { return startupErr }
	}
	return nil
}

// registerRedisStartupMetric 暴露启动期 Redis 降级状态（1=显式配置但不可达）。
// 用 Gauge 而非计数器：启动 ping 失败后 client 被丢弃、进程内不会重连，降级是
// 持续状态（恢复需重启），hotspot_redis_startup_degraded == 1 才能被告警规则
// 稳定捕获；计数器只在重启瞬间 +1，容易被 5 分钟速率窗口与 for 时长吃掉。
// 未配置 Redis 时同样暴露 0，保持指标序列连续（生产 Validate 强制要求配置）。
func registerRedisStartupMetric(degraded bool) {
	observability.Register(newRedisStartupDegradedGauge(degraded))
}

// newRedisStartupDegradedGauge 构造降级状态 gauge（拆分出来便于单测断言取值）。
func newRedisStartupDegradedGauge(degraded bool) prometheus.Collector {
	value := 0.0
	if degraded {
		value = 1
	}
	return prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "hotspot_redis_startup_degraded",
		Help: "启动时显式配置了 Redis 但不可达（1=已降级为单实例幂等/限流/跨实例广播；恢复需重启）",
	}, func() float64 { return value })
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
// 不接管 rdb 生命周期：RedisBridge.Close 只退订阅、不关闭共享 client（resume/
// 健康探针/缓存仍在用），client 统一由 App.Shutdown 关闭（见 App.redis）。
func initRedisBridgeFromClient(cfg config.Config, logger *observability.Logger, rdb *redis.Client, instanceID string, hub *realtime.Hub) *realtime.RedisBridge {
	if rdb == nil {
		return nil
	}
	bridge, err := realtime.NewRedisBridge(rdb, instanceID, hub)
	if err != nil {
		logger.Error("redis bridge init failed, falling back to single-instance mode", "err", err)
		rdb.Close()
		return nil
	}
	hub.SetPublisher(bridge)
	logger.Info("redis bridge initialized", "instance", instanceID)
	return bridge
}
