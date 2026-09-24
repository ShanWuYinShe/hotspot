package media

import (
	"net/url"
	"strings"

	"github.com/go-chi/chi/v5"

	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/idempotency"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/ratelimit"
	"hotspot/backend-go/internal/platform/resourceguard"
)

type Module struct {
	handler *Handler
	Service *Service
}

func NewModule(cfg config.Config, factory *persistence.Factory) *Module {
	repo := NewPostgresRepository(factory)
	endpoint, err := url.Parse(strings.TrimSpace(cfg.S3Endpoint))
	if err != nil || endpoint.Scheme == "" || endpoint.Host == "" || strings.TrimSpace(cfg.S3Bucket) == "" {
		panic("media object storage not configured")
	}
	provider := NewObjectStorageProvider(cfg)
	service := NewServiceWithRepoAndProvider(repo, provider)
	return &Module{handler: NewHandler(service), Service: service}
}

// MountPublic 挂载匿名头像代理端点（GET /public/v1/media/avatars/{file_id}，
// 302 到签名读 URL）。avatar_url 存储的就是该代理路径（见 service_avatar.go），
// 登记 GET 例外见 handler_avatar.go 头注释。仅限流不鉴权：头像为公开资料，
// 端点只做类别白名单 + 302，不回源文件内容。
func (m *Module) MountPublic(r chi.Router) {
	r.Get("/media/avatars/{file_id}", m.handler.ServeAvatar)
}

func (m *Module) Mount(r chi.Router) {
	r.Route("/api/v1/media", func(mediaRouter chi.Router) {
		mediaRouter.Group(func(group chi.Router) {
			group.Use(platformauth.RequiredAuth)
			group.Use(ratelimit.Middleware)
			group.Use(resourceguard.Middleware)
			group.Use(idempotency.Middleware)
			group.Post("/presign_upload", m.handler.PresignUpload)
			group.Post("/complete_upload", m.handler.CompleteUpload)
		})
	})
}
