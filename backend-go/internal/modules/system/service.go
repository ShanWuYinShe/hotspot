package system

import (
	"context"

	"hotspot/backend-go/internal/platform/appconfig"
	"hotspot/backend-go/internal/platform/persistence"
)

type Service struct {
	repo     Repository
	provider Provider
}

func NewService(appConfig appconfig.Source) *Service {
	return NewServiceWithDeps(NewInMemoryRepository(), NewAppConfigProvider(appConfig))
}

func NewServiceWithFactory(factory *persistence.Factory, appConfig appconfig.Source) *Service {
	return NewServiceWithDeps(NewPersistenceRepository(factory), NewAppConfigProvider(appConfig))
}

func NewServiceWithDeps(repo Repository, provider Provider) *Service {
	return &Service{repo: repo, provider: provider}
}

func (s *Service) GetMeta(ctx context.Context) ApiVersionMeta {
	return s.repo.GetMeta(ctx)
}

func (s *Service) Health(ctx context.Context) (int, string, *HealthStatus) {
	return s.repo.Health(ctx)
}

func (s *Service) ListAnnouncements(ctx context.Context) (SystemAnnouncementList, error) {
	items, err := s.repo.ListAnnouncements(ctx, s.provider.GetSystemAnnouncementsCount(ctx))
	if err != nil {
		return SystemAnnouncementList{}, err
	}
	resp := SystemAnnouncementList{Items: make([]SystemAnnouncementItem, 0, len(items))}
	for _, item := range items {
		resp.Items = append(resp.Items, item.PublicItem())
	}
	return resp, nil
}

func (s *Service) GetClientConfig(ctx context.Context) ClientConfig {
	return s.provider.GetClientConfig(ctx)
}
