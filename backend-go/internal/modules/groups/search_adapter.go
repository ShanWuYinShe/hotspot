package groups

import (
	"context"
	"encoding/json"

	"hotspot/backend-go/internal/modules/search"
)

// SearchAdapter adapts the groups Service to the search.Searcher interface so
// that the search module can fan out cross-module search/all requests.
type SearchAdapter struct {
	service *Service
}

func NewSearchAdapter(service *Service) *SearchAdapter {
	return &SearchAdapter{service: service}
}

func (a *SearchAdapter) Search(ctx context.Context, params search.SearchParams) (search.SearchResult, error) {
	var radius *int
	if params.Radius > 0 {
		r := params.Radius
		radius = &r
	}
	req := SearchGroupsRequest{
		Keyword: params.Keyword,
		Tag:     params.Tag,
		Radius:  radius,
		SortBy:  params.SortBy,
		ReferencePoint: GeoPoint{
			Latitude:  params.ReferencePoint.Latitude,
			Longitude: params.ReferencePoint.Longitude,
		},
		Limit:  params.Limit,
		Cursor: params.Cursor,
	}
	result, err := a.service.Search(ctx, req)
	if err != nil {
		return search.SearchResult{}, err
	}
	items := make([]search.SearchItem, 0, len(result.Items))
	for _, item := range result.Items {
		payload, err := structToMap(item)
		if err != nil {
			return search.SearchResult{}, err
		}
		items = append(items, search.SearchItem{Type: "group", Payload: payload})
	}
	return search.SearchResult{Items: items, NextCursor: result.NextCursor}, nil
}

func structToMap(value any) (map[string]any, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	out := make(map[string]any)
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return out, nil
}
