package types

// MediaAsset 是媒体资源基础信息（spec api_spec_types.md MediaAsset）：
// MessageMedia 等消息媒体类型在其基础上扩展文件元数据。
type MediaAsset struct {
	URL          string `json:"url"`
	ThumbnailURL string `json:"thumbnail_url,omitempty"`
	Width        int    `json:"width,omitempty"`
	Height       int    `json:"height,omitempty"`
	MimeType     string `json:"mime_type,omitempty"`
}

// MomentSummary 是瞬间的摘要信息（spec api_spec_types.md MomentSummary）：
// 用于通知/评论等场景的 moment_summary 字段（截断描述 + 首图缩略图）。
type MomentSummary struct {
	MomentID           string `json:"moment_id"`
	DescriptionPreview string `json:"description_preview"`
	CoverURL           string `json:"cover_url,omitempty"`
}

// TagItem 是标签项（spec api_spec_types.md TagItem）：搜索建议、Feed 等
// 模块共用。
type TagItem struct {
	Name       string `json:"name"`
	UsageCount int    `json:"usage_count"`
	Trend      string `json:"trend,omitempty"`
}
