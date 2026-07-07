package search

import "context"

// SearchParams is the common input contract passed to each Searcher when
// running a cross-module search/all request. Each Searcher adapts the fields
// to its own module-specific search request.
type SearchParams struct {
	Keyword        string
	Tag            string
	Radius         int
	SortBy         string
	ReferencePoint GeoPoint
	Limit          int
	Cursor         string
}

// SearchItem is a single, type-tagged result returned by a Searcher. The
// concrete payload is carried verbatim (as map[string]any) so callers can
// forward the module-native JSON shape without lossy conversions.
type SearchItem struct {
	Type    string         `json:"type"`
	Payload map[string]any `json:"payload"`
}

// SearchResult is the normalized output of a Searcher.
type SearchResult struct {
	Items      []SearchItem
	NextCursor string
}

// Searcher abstracts a single searchable module (users, groups, moments) so
// that the search module can fan out a search/all request across them without
// depending on any module's concrete types.
type Searcher interface {
	Search(ctx context.Context, params SearchParams) (SearchResult, error)
}
