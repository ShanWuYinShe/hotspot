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
}

func NewModule(cfg config.Config, factory *persistence.Factory) *Module {
	repo := NewPersistenceRepository(factory)
	endpoint, err := url.Parse(strings.TrimSpace(cfg.S3Endpoint))
	if err != nil || endpoint.Scheme == "" || endpoint.Host == "" || strings.TrimSpace(cfg.S3Bucket) == "" {
		panic("media object storage not configured")
	}
	provider := NewObjectStorageProvider(cfg)
	return &Module{handler: NewHandler(NewServiceWithRepoAndProvider(repo, provider))}
}

func (m *Module) Mount(r chi.Router) {
	r.Route("/api/v1/media", func(mediaRouter chi.Router) {
		mediaRouter.Group(func(group chi.Router) {
			group.Use(platformauth.RequiredAuth)
			group.Use(ratelimit.Middleware)
			group.Use(resourceguard.Middleware)
			group.Use(idempotency.Middleware)
			group.Post("/presign_upload", m.handler.PresignUpload)
			group.Post("/batch_presign_upload", m.handler.BatchPresignUpload)
			group.Post("/complete_upload", m.handler.CompleteUpload)
		})
	})
}
