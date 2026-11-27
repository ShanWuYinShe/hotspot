package media

import "time"

type MediaCategory string

const (
	CategoryAvatar      MediaCategory = "avatar"
	CategoryGroupAvatar MediaCategory = "group_avatar"
	CategoryMomentImage MediaCategory = "moment_image"
	CategoryChatImage   MediaCategory = "chat_image"
	CategoryChatVideo   MediaCategory = "chat_video"
	CategoryChatAudio   MediaCategory = "chat_audio"
	CategoryChatFile    MediaCategory = "chat_file"
)

type PresignFileInput struct {
	FileName string        `json:"file_name"`
	FileSize int           `json:"file_size"`
	MimeType string        `json:"mime_type"`
	Category MediaCategory `json:"category"`
	MD5      string        `json:"md5,omitempty"`
}

type PresignUploadRequest struct {
	Files []PresignFileInput `json:"files"` // 1..N（spec api_spec_media.md：数组天然覆盖单/批，无独立批量端点）
}

type PresignedUpload struct {
	UploadID   string            `json:"upload_id"`
	UploadURL  string            `json:"upload_url"`
	Headers    map[string]string `json:"headers"`
	ExpiresAt  string            `json:"expires_at"`
	StorageKey string            `json:"-"`
}

type PresignedUploads struct {
	Uploads []PresignedUpload `json:"uploads"` // 与请求 files 一一对应且顺序一致
}

type CompleteUploadRequest struct {
	UploadIDs []string `json:"upload_ids"` // 1..N
}

type CompletedUploads struct {
	Uploads []UploadedMedia `json:"uploads"` // 与请求 upload_ids 一一对应且顺序一致
}

// UploadedMedia 是 complete_upload 响应项（spec api_spec_media.md）：四个
// 元数据字段的契约形态为「恒定 nullable」——图片没有 duration、非音视频没有
// thumbnail_url、部分类型没有宽高，无值时必须显式下发 null 而不是缺键。
// 故用指针 + 不带 omitempty（内部零值经 ptrString/ptrInt 归一为 nil）。
type UploadedMedia struct {
	FileID       string  `json:"file_id"`
	ThumbnailURL *string `json:"thumbnail_url"`
	FileSize     int     `json:"file_size"`
	MimeType     string  `json:"mime_type"`
	Duration     *int    `json:"duration"`
	Width        *int    `json:"width"`
	Height       *int    `json:"height"`
}

// ptrString 把「空串即无值」的存储形态归一为 null；ptrInt 同理（0 即无值）。
func ptrString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func ptrInt(value int) *int {
	if value == 0 {
		return nil
	}
	return &value
}

// valueString/valueInt 是 ptrString/ptrInt 的逆运算，供需要值语义的消费方
// （MediaAsset / FileMeta 等内部聚合）使用。
func valueString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func valueInt(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}

type ObjectMetadata struct {
	FileSize int
	MimeType string
}

// ObjectSummary 是对象存储列举结果中的单个对象摘要（S3 ListObjectsV2 的
// Contents 项）。LastModified 解析失败时为零值，调用方应按「无法判断对象年龄」
// 保守跳过，不做删除。
type ObjectSummary struct {
	Key          string
	LastModified time.Time
}

// ObjectListPage 是 S3 ListObjectsV2 的单页结果。IsTruncated 为 true 时用
// NextContinuationToken 取下一页（令牌不透明，调用方原样回传）。
type ObjectListPage struct {
	Objects               []ObjectSummary
	IsTruncated           bool
	NextContinuationToken string
}

// MediaAsset is the resolved view of a completed upload, used by other modules
// (e.g. moments) to attach media to their entities and to presign read URLs.
type MediaAsset struct {
	FileID       string
	StorageKey   string
	ThumbnailURL string
	Width        int
	Height       int
	MimeType     string
}

// FileMeta 是按 file_id 读取的完成态文件展示元数据（举报快照等跨模块
// 只读场景消费；与 MediaAsset 的差异是附带 file_name/file_size/duration）。
type FileMeta struct {
	Category     MediaCategory
	StorageKey   string
	ThumbnailURL string
	Width        int
	Height       int
	MimeType     string
	FileName     string
	FileSize     int
	Duration     int
}
