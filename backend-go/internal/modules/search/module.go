package search

import (
	"github.com/go-chi/chi/v5"

	"hotspot/backend-go/internal/platform/appconfig"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/idempotency"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/ratelimit"
	"hotspot/backend-go/internal/platform/resourceguard"
)

type Module struct {
	handler *Handler
}

func NewModule(factory *persistence.Factory, source appconfig.Source) *Module {
	repo := NewPersistenceRepository(factory)
	return &Module{handler: NewHandler(NewServiceWithSource(repo, source))}
}

func NewModuleWithSearchers(factory *persistence.Factory, source appconfig.Source, userSearcher, groupSearcher, momentSearcher Searcher) *Module {
	repo := NewPersistenceRepository(factory)
	return &Module{handler: NewHandler(NewServiceWithSearchers(repo, source, userSearcher, groupSearcher, momentSearcher))}
}

func (m *Module) Mount(r chi.Router) {
	r.Route("/api/v1/search", func(searchRouter chi.Router) {
		searchRouter.Group(func(group chi.Router) {
			group.Use(platformauth.OptionalAuth)
			group.Use(ratelimit.Middleware)
			group.Use(resourceguard.Middleware)
			group.Use(idempotency.Middleware)
			group.Post("/suggest", m.handler.Suggest)
			group.Post("/all", m.handler.SearchAll)
		})
	})
}
