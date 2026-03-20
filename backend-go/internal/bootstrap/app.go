package bootstrap

import (
	"net/http"

	"go.uber.org/zap"

	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/modulekit"
	"hotspot/backend-go/internal/platform/observability"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/realtime"
)

type App struct {
	cfg         config.Config
	logger      *observability.Logger
	router      http.Handler
	modules     []modulekit.Module
	persistence *persistence.Factory
}

func NewApp(cfg config.Config) (*App, error) {
	logger, err := observability.NewLogger()
	if err != nil {
		return nil, err
	}

	persistenceFactory := persistence.NewFactory(cfg)
	modules := BuildModules(cfg, persistenceFactory)
	router := NewHTTPRouter(modules)
	rootMux := http.NewServeMux()
	rootMux.Handle("/", router)
	rootMux.Handle("/ws/v1", realtime.NewHandler(platformauth.NewService(cfg)))

	return &App{
		cfg:         cfg,
		logger:      logger,
		router:      rootMux,
		modules:     modules,
		persistence: persistenceFactory,
	}, nil
}

func (a *App) Router() http.Handler {
	return a.router
}

func (a *App) Logger() *observability.Logger {
	return a.logger
}

func (a *App) LogField(key string, value any) zap.Field {
	return observability.Field(key, value)
}
