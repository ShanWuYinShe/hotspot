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
	"backend-go/internal/middlewares"
	"backend-go/internal/repository"
	"backend-go/internal/service"
)

type App struct {
	DB     *pgxpool.Pool
	Redis  *redis.Client
	Logger *zap.Logger

	mapService   service.MapService
	authService  service.AuthService
	tokenManager auth.TokenManager
	blacklists   repository.TokenBlacklistRepository
	userService  service.UserService
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
	tokenBlacklistRepo := repository.NewTokenBlacklistRepository(redisClient)

	// Token TTL 可根据配置调整，这里先给出合理默认值
	accessTTL := 15 * time.Minute
	refreshTTL := 30 * 24 * time.Hour
	tokenManager := auth.NewJWTTokenManager(cfg.JWTSecret, accessTTL, refreshTTL)

	authSvc := service.NewAuthService(userRepo, sessionRepo, tokenManager, tokenBlacklistRepo)
	userSvc := service.NewUserService(userRepo, sessionRepo)

	return &App{
		DB:           pool,
		Redis:        redisClient,
		Logger:       logger,
		mapService:   mapSvc,
		authService:  authSvc,
		tokenManager: tokenManager,
		blacklists:   tokenBlacklistRepo,
		userService:  userSvc,
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

	// 按 api_spec_auth.md 注册认证相关端点，并串联认证中间件：
	// - register/login/guest/request_password_reset/reset_password: 无认证
	// - refresh_token / check_status: Token 可选
	// - logout / me/*: Token 必须
	r.Route("/api/v1/auth", func(ar chi.Router) {
		// 无认证端点
		ar.Post("/register", authHandler.Register)
		ar.Post("/login", authHandler.Login)
		ar.Post("/guest", authHandler.Guest)
		ar.Post("/request_password_reset", authHandler.RequestPasswordReset)
		ar.Post("/reset_password", authHandler.ResetPassword)

		// Token 可选端点
		ar.With(middlewares.AuthOptional(a.tokenManager, a.blacklists)).Post("/refresh_token", authHandler.RefreshToken)
		ar.With(middlewares.AuthOptional(a.tokenManager, a.blacklists)).Post("/check_status", authHandler.CheckStatus)

		// Token 必须端点
		ar.Group(func(pr chi.Router) {
			pr.Use(middlewares.AuthRequired(a.tokenManager, a.blacklists))
			pr.Post("/logout", authHandler.Logout)
			pr.Post("/me/change_password", authHandler.ChangePassword)
			pr.Post("/me/sessions/list", authHandler.ListSessions)
			pr.Post("/me/sessions/revoke", authHandler.RevokeSession)
			pr.Post("/me/sessions/revoke_all", authHandler.RevokeAllSessions)
		})
	})

	// users/me/* 端点：统一挂载 token_required 中间件。
	userHandler := handler.NewUserHandler(a.userService)
	r.Group(func(ur chi.Router) {
		ur.Use(middlewares.AuthRequired(a.tokenManager, a.blacklists))
		handler.RegisterUserRoutes(ur, userHandler)
	})

	// TODO: 后续在此注册其他模块（user/group/moment/chat/feed 等）
}
