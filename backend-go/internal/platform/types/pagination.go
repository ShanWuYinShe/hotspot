package types

// 全后端统一的分页常量。各模块的分页上限应引用 MaxPageSize 而非内联 100。
const (
	DefaultPageSize = 20
	MaxPageSize     = 100
)

// PaginationParams is the shared request-pagination shape (api_spec_base.md
// PaginationParams). Modules embed/reference this instead of redefining it,
// so a field rename or limit change doesn't drift across 14 copies.
type PaginationParams struct {
	Limit  int    `json:"limit,omitempty"`
	Cursor string `json:"cursor,omitempty"`
}

// NormalizeLimit 归一化分页 limit：≤0 取 DefaultPageSize，>MaxPageSize 截断。
// 全后端统一入口，替代各模块重复的 normalizeLimit/paginationLimit。
func NormalizeLimit(limit int) int {
	switch {
	case limit <= 0:
		return DefaultPageSize
	case limit > MaxPageSize:
		return MaxPageSize
	default:
		return limit
	}
}

// PaginationLimit 返回分页 limit 裸值（nil 安全）。
func PaginationLimit(p *PaginationParams) int {
	if p == nil {
		return 0
	}
	return p.Limit
}

// PaginationLimitWith 返回归一化 limit：≤0 取 fallback，>max 截断 max。
func PaginationLimitWith(p *PaginationParams, fallback, max int) int {
	if p == nil || p.Limit <= 0 {
		return fallback
	}
	if p.Limit > max {
		return max
	}
	return p.Limit
}

// PaginationCursor 返回分页 cursor（nil 安全）。
func PaginationCursor(p *PaginationParams) string {
	if p == nil {
		return ""
	}
	return p.Cursor
}
