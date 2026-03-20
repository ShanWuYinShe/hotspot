package search

import "context"

type Service struct{}

func NewService() *Service {
	return &Service{}
}

func (s *Service) Suggest(ctx context.Context, req SearchSuggestRequest) SearchSuggestResponse {
	_ = ctx
	items := []SearchSuggestItem{}
	if req.Prefix == "" {
		return SearchSuggestResponse{Items: items}
	}
	items = append(items,
		SearchSuggestItem{Type: "tag", Payload: map[string]any{"name": req.Prefix, "usage_count": 1}},
		SearchSuggestItem{Type: "user", Payload: map[string]any{"user_id": "user_" + req.Prefix, "nickname": req.Prefix}},
		SearchSuggestItem{Type: "group", Payload: map[string]any{"group_id": "group_" + req.Prefix, "name": req.Prefix, "members_count": 1}},
		SearchSuggestItem{Type: "moment", Payload: map[string]any{"moment_id": "moment_" + req.Prefix, "description_preview": req.Prefix, "author": map[string]any{"user_id": "user_" + req.Prefix, "nickname": req.Prefix}}},
	)
	return SearchSuggestResponse{Items: items}
}
