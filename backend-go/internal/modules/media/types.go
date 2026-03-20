package media

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

type PresignUploadRequest struct {
	FileName string        `json:"file_name"`
	FileSize int           `json:"file_size"`
	MimeType string        `json:"mime_type"`
	Category MediaCategory `json:"category"`
	MD5      string        `json:"md5,omitempty"`
}

type PresignedUpload struct {
	UploadID  string            `json:"upload_id"`
	UploadURL string            `json:"upload_url"`
	Headers   map[string]string `json:"headers"`
	ExpiresAt string            `json:"expires_at"`
}

type BatchPresignUploadRequest struct {
	Files []PresignUploadRequest `json:"files"`
}

type BatchPresignedUpload struct {
	Uploads []PresignedUpload `json:"uploads"`
}

type CompleteUploadRequest struct {
	UploadID string `json:"upload_id"`
}

type UploadedMedia struct {
	FileID       string `json:"file_id"`
	ThumbnailURL string `json:"thumbnail_url,omitempty"`
	FileSize     int    `json:"file_size"`
	MimeType     string `json:"mime_type"`
	Duration     int    `json:"duration,omitempty"`
	Width        int    `json:"width,omitempty"`
	Height       int    `json:"height,omitempty"`
}
