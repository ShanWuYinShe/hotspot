package share

import (
	"hotspot/backend-go/internal/modules/moments"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/idempotency"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/ratelimit"
	"hotspot/backend-go/internal/platform/resourceguard"

	"github.com/go-chi/chi/v5"
)

type Module struct {
	handler *Handler
	Service *Service
}

func NewModule(factory *persistence.Factory) *Module {
	repo := NewPostgresRepository(factory)
	service := NewServiceWithRepo(repo)
	service.SetMomentsSharer(moments.NewServiceWithRepo(moments.NewPostgresRepository(factory)))
	return &Module{handler: NewHandler(service), Service: service}
}

func (m *Module) Mount(r chi.Router) {
	r.Route("/api/v1/share", func(sr chi.Router) {
		sr.Group(func(group chi.Router) {
			group.Use(platformauth.RequiredAuth)
			group.Use(ratelimit.Middleware)
			group.Use(resourceguard.Middleware)
			group.Use(idempotency.Middleware)
			group.Post("/create", m.handler.Create)
			group.Post("/fetch_created", m.handler.List)
			group.Post("/revoke", m.handler.Revoke)
		})
	})

	// 爬虫预览落地页（GET /share/{code}，登记例外见 og.go 头注释）：仅限流
	// 不鉴权——解析 share code 本身即匿名能力（open_the_door 同源）。流量来自
	// nginx 的 share_crawler UA 分支（真实用户走 SPA，见 docker/nginx*.conf）。
	r.With(ratelimit.Middleware).Get("/share/{code}", m.handler.ServeCrawlerHTML)
}

// MountPublic mounts the anonymous share-code resolution endpoint on the
// /public/v1 node (spec api_spec_public.md — public-only, no /api/v1 mirror).
func (m *Module) MountPublic(r chi.Router) {
	r.Route("/share", func(sr chi.Router) {
		sr.Post("/open_the_door", m.handler.Get)
	})
}
