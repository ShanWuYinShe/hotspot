package system

import (
	"context"

	"hotspot/backend-go/internal/platform/appconfig"
	"hotspot/backend-go/internal/platform/observability"
	"hotspot/backend-go/internal/platform/persistence"
)

type Service struct {
	repo      Repository
	provider  Provider
	bootstrap BootstrapResolver
}

func NewService(appConfig appconfig.Source) *Service {
	return NewServiceWithDeps(NewInMemoryRepository(), NewAppConfigProvider(appConfig))
}

func NewServiceWithFactory(factory *persistence.Factory, appConfig appconfig.Source) *Service {
	return NewServiceWithDeps(NewPostgresRepository(factory), NewAppConfigProvider(appConfig))
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

// SetHealthRedisPinger wires an optional redis liveness probe onto the
// underlying repository, when it supports one. No-op for in-memory repos.
func (s *Service) SetHealthRedisPinger(p func(ctx context.Context) error) {
	type redisPingerSetter interface {
		SetRedisPinger(func(ctx context.Context) error)
	}
	if setter, ok := s.repo.(redisPingerSetter); ok {
		setter.SetRedisPinger(p)
	}
}

func (s *Service) ListAnnouncements(ctx context.Context) (SystemAnnouncementList, error) {
	items, err := s.repo.ListAnnouncements(ctx, s.provider.GetSystemAnnouncementsCount(ctx))
	if err != nil {
		observability.L().Error("system.list_announcements failed", "error", err)
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

func (s *Service) Bootstrap(ctx context.Context, bootstrapResolver BootstrapResolver) BootstrapData {
	bootstrapVersion := s.repo.GetMeta(ctx).Version
	meta := s.GetMeta(ctx)
	clientConfig := s.GetClientConfig(ctx)
	bd := BootstrapData{
		BootstrapVersion: bootstrapVersion,
		Meta:             &meta,
		ClientConfig:     &clientConfig,
	}
	if bootstrapResolver != nil {
		if profile, ok := bootstrapResolver.ResolveProfile(ctx); ok {
			bd.Profile = profile
			// 仅登录态解析未读摘要；匿名时 profile / unread_summary 均为 null（恒定 nullable）
			summary := bootstrapResolver.ResolveUnreadSummary(ctx)
			bd.UnreadSummary = &summary
		}
	}
	return bd
}

func (s *Service) GetBootstrapResolver() BootstrapResolver  { return s.bootstrap }
func (s *Service) SetBootstrapResolver(r BootstrapResolver) { s.bootstrap = r }
