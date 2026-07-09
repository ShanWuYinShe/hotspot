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
	platformmigrate "hotspot/backend-go/internal/platform/migrate"
	"hotspot/backend-go/internal/platform/modulekit"
	"hotspot/backend-go/internal/platform/observability"
	"hotspot/backend-go/internal/platform/persistence"
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
	modules := BuildModules(cfg, persistenceFactory)
	bridge := initRedisBridge(cfg, logger)
	router := NewHTTPRouterWithLogger(modules, logger)
	rootMux := http.NewServeMux()
	rootMux.Handle("/", router)
	userRepo := users.NewPersistenceRepository(persistenceFactory)
	groupRepo := groups.NewPersistenceRepository(persistenceFactory)
	userService := users.NewServiceWithRepo(userRepo)
	rootMux.Handle("/ws/v1", realtime.NewHandler(platformauth.DefaultService(), realtimeUserService{svc: userService}, realtimeUserRepo{repo: userRepo}, realtimeGroupRepo{repo: groupRepo}))

	return &App{
		cfg:         cfg,
		logger:      logger,
		router:      rootMux,
		modules:     modules,
		persistence: persistenceFactory,
		bridge:      bridge,
	}, nil
}

func (a *App) Router() http.Handler {
	return a.router
}

func (a *App) Logger() *observability.Logger {
	return a.logger
}

func (a *App) LogField(key string, value any) (string, any) {
	return key, value
}

func (a *App) Shutdown() {
	if a.bridge != nil {
		a.bridge.Close()
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

func initRedisBridge(cfg config.Config, logger *observability.Logger) *realtime.RedisBridge {
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
