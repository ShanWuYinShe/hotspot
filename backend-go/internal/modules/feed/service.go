package feed

import "context"

type Service struct{ repo Repository }

func NewService() *Service                        { return NewServiceWithRepo(NewInMemoryRepository()) }
func NewServiceWithRepo(repo Repository) *Service { return &Service{repo: repo} }

func (s *Service) ListMoments(ctx context.Context) (PaginatedFeedItems, error) {
	return s.repo.ListMoments(ctx)
}

func (s *Service) ListGroups(ctx context.Context) (PaginatedFeedItems, error) {
	return s.repo.ListGroups(ctx)
}

func (s *Service) ListUsers(ctx context.Context) (PaginatedFeedItems, error) {
	return s.repo.ListUsers(ctx)
}

func (s *Service) ListTrending(ctx context.Context) (TrendingBundle, error) {
	return s.repo.ListTrending(ctx)
}
