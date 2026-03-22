package search

import "context"

type Service struct{ repo Repository }

func NewService() *Service                        { return NewServiceWithRepo(NewInMemoryRepository()) }
func NewServiceWithRepo(repo Repository) *Service { return &Service{repo: repo} }

func (s *Service) Suggest(ctx context.Context, req SearchSuggestRequest) (SearchSuggestResponse, error) {
	return s.repo.Suggest(ctx, req)
}
