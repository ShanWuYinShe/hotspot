package system

import (
	"context"

	"hotspot/backend-go/internal/platform/appconfig"
)

type Service struct {
	store    Store
	provider Provider
}

func NewService(appConfig appconfig.Source) *Service {
	return NewServiceWithDeps(NewInMemoryStore(), NewAppConfigProvider(appConfig))
}

func NewServiceWithDeps(store Store, provider Provider) *Service {
	return &Service{store: store, provider: provider}
}

func (s *Service) GetMeta(ctx context.Context) ApiVersionMeta {
	return s.store.GetMeta(ctx)
}

func (s *Service) Health(ctx context.Context) (int, string, *HealthStatus) {
	return s.store.Health(ctx)
}

func (s *Service) ListAnnouncements(ctx context.Context) SystemAnnouncementList {
	return s.store.ListAnnouncements(ctx)
}

func (s *Service) GetClientConfig(ctx context.Context) ClientConfig {
	return s.provider.GetClientConfig(ctx)
}
