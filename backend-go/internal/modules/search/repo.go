package search

import "context"

type Repository interface {
	Suggest(context.Context, SearchSuggestRequest) (SearchSuggestResponse, error)
}
