package app

import (
	"context"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"backend-go/internal/auth"
	"backend-go/internal/config"
	"backend-go/internal/handler"
	"backend-go/internal/repository"
	"backend-go/internal/service"
)

type App struct {
	DB     *pgxpool.Pool
	Redis  *redis.Client
	Logger *zap.Logger

	mapService  service.MapService
	authService service.AuthService
}

func NewApp(ctx context.Context, cfg config.Config, logger *zap.Logger) (*App, error) {
	pool, err := pgxpool.New(ctx, cfg.PostgresURL)
	if err != nil {
		return nil, err
	}

	redisClient := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr})

	geoRepo := repository.NewGeoRepository(pool)
	mapSvc := service.NewMapService(geoRepo)

	userRepo := repository.NewUserRepository(pool)
	sessionRepo := repository.NewSessionRepository(pool)

	// Token TTL 可根据配置调整，这里先给出合理默认值
	accessTTL := 15 * time.Minute
	refreshTTL := 30 * 24 * time.Hour
	tokenManager := auth.NewJWTTokenManager(cfg.JWTSecret, accessTTL, refreshTTL)

	authSvc := service.NewAuthService(userRepo, sessionRepo, tokenManager)

	return &App{
		DB:         pool,
		Redis:      redisClient,
		Logger:     logger,
		mapService:  mapSvc,
		authService: authSvc,
	}, nil
}

// Close 释放底层资源。
func (a *App) Close() {
	if a.DB != nil {
		a.DB.Close()
	}
	if a.Redis != nil {
		_ = a.Redis.Close()
	}
}

// RegisterRoutes 注册所有模块的 HTTP 路由。
func (a *App) RegisterRoutes(r chi.Router) {
	mapHandler := handler.NewMapHandler(a.mapService)
	handler.RegisterMapRoutes(r, mapHandler)

	authHandler := handler.NewAuthHandler(a.authService)
	handler.RegisterAuthRoutes(r, authHandler)

	// TODO: 后续在此注册其他模块（user/group/moment/chat/feed 等）
}
