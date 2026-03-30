package search

import "context"

type InMemoryRepository struct{}

func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{}
}

func (r *InMemoryRepository) Suggest(_ context.Context, req SearchSuggestRequest) (SearchSuggestResponse, error) {
	items := []SearchSuggestItem{}
	if req.Prefix == "" {
		return SearchSuggestResponse{Items: items}, nil
	}
	allowed := normalizeSuggestTypes(req.Types)
	if allowed["tag"] {
		items = append(items, SearchSuggestItem{Type: "tag", Payload: map[string]any{"name": req.Prefix, "usage_count": 1}})
	}
	if allowed["user"] {
		items = append(items, SearchSuggestItem{Type: "user", Payload: map[string]any{"user_id": "user_" + req.Prefix, "nickname": req.Prefix}})
	}
	if allowed["group"] {
		items = append(items, SearchSuggestItem{Type: "group", Payload: map[string]any{"group_id": "group_" + req.Prefix, "name": req.Prefix, "members_count": 1}})
	}
	if allowed["moment"] {
		items = append(items, SearchSuggestItem{Type: "moment", Payload: map[string]any{"moment_id": "moment_" + req.Prefix, "description_preview": req.Prefix, "author": map[string]any{"user_id": "user_" + req.Prefix, "nickname": req.Prefix}}})
	}
	return SearchSuggestResponse{Items: items}, nil
}
