package types

// PaginationParams is the shared request-pagination shape (api_spec_base.md
// PaginationParams). Modules embed/reference this instead of redefining it,
// so a field rename or limit change doesn't drift across 14 copies.
type PaginationParams struct {
	Limit  int    `json:"limit,omitempty"`
	Cursor string `json:"cursor,omitempty"`
}
