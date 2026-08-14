package groupmessages

import (
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

func NewModule(factory *persistence.Factory, users userReader, mentions groupMentionNotifier, shares shareValidator) *Module {
	svc := NewServiceWithDeps(NewPostgresRepository(factory), users, mentions, shares)
	return &Module{handler: NewHandler(svc), Service: svc}
}
func (m *Module) Mount(r chi.Router) {
	r.Route("/api/v1/group_messages", func(gr chi.Router) {
		gr.Group(func(group chi.Router) {
			group.Use(platformauth.RequiredAuth)
			group.Use(ratelimit.Middleware)
			group.Use(resourceguard.Middleware)
			group.Use(idempotency.Middleware)
			group.Post("/send", m.handler.Send)
			group.Post("/list", m.handler.History)
			group.Post("/update", m.handler.Update)
			group.Post("/recall", m.handler.Recall)
			group.Post("/mark_read", m.handler.MarkRead)
		})
	})
}
