package auth

import (
	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/persistence"
)

func NewServiceWithPersistence(cfg config.Config, factory *persistence.Factory) *Service {
	if factory == nil {
		return NewService(cfg)
	}
	return NewServiceWithStores(cfg, NewPostgresSessionStore(factory, nil), NewPostgresRefreshStore(factory, nil))
}
