package feed

import (
	"context"

	"hotspot/backend-go/internal/platform/appconfig"
	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/observability"
	"hotspot/backend-go/internal/platform/slicex"
)

type Service struct {
	repo      Repository
	appConfig appconfig.Source
}

func NewService(source appconfig.Source) *Service { return NewServiceWithSource(NewInMemoryRepository(), source) }

func NewServiceWithRepo(repo Repository) *Service { return NewServiceWithSource(repo, nil) }

func NewServiceWithSource(repo Repository, source appconfig.Source) *Service {
	if repo == nil {
		repo = NewInMemoryRepository()
	}
	if source == nil {
		source = appconfig.NewStaticSource(config.Config{})
	}
	return &Service{repo: repo, appConfig: source}
}

func (s *Service) ListMoments(ctx context.Context) (PaginatedFeedItems, error) {
	items, err := s.repo.ListMoments(ctx)
	if err != nil {
		observability.L().Error("feed.list_moments failed", "error", err)
		return PaginatedFeedItems{}, err
	}
	return items, nil
}

func (s *Service) ListGroups(ctx context.Context) (PaginatedFeedItems, error) {
	items, err := s.repo.ListGroups(ctx)
	if err != nil {
		observability.L().Error("feed.list_groups failed", "error", err)
		return PaginatedFeedItems{}, err
	}
	return items, nil
}

func (s *Service) ListUsers(ctx context.Context) (PaginatedFeedItems, error) {
	items, err := s.repo.ListUsers(ctx)
	if err != nil {
		observability.L().Error("feed.list_users failed", "error", err)
		return PaginatedFeedItems{}, err
	}
	return items, nil
}

func (s *Service) ListTrending(ctx context.Context) (TrendingBundle, error) {
	bundle, err := s.repo.ListTrending(ctx)
	if err != nil {
		observability.L().Error("feed.list_trending failed", "error", err)
		return TrendingBundle{}, err
	}
	cfg := s.appConfig.Get(ctx)
	bundle.TrendingTags = slicex.LimitCopy(bundle.TrendingTags, cfg.ResponseDefaults.TrendingTagsCount)
	bundle.HotMoments = slicex.LimitCopy(bundle.HotMoments, cfg.ResponseDefaults.HotMomentsCount)
	return bundle, nil
}
