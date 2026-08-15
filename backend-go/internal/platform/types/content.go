package types

// MomentSummary 是瞬间的摘要信息（spec api_spec_types.md MomentSummary）：
// 通知 payload 的 moment_summary 字段（截断描述 + 首图缩略图）。用类型
// 承载而非 map[string]any，保证字段名与 spec 一致且编译期可查。
type MomentSummary struct {
	MomentID           string  `json:"moment_id"`
	DescriptionPreview string  `json:"description_preview"`
	CoverURL           *string `json:"cover_url"`
}
