package media

import (
	"context"
	"time"
)

type pendingUpload struct {
	UploadID                string
	UserID                  string
	StorageKey              string
	FileName                string
	DeclaredFileSize        int
	DeclaredMimeType        string
	Category                MediaCategory
	MD5                     string
	Completed               bool
	File                    UploadedMedia
	CompletedIdempotencyKey string
	CreatedAt               string
	CompletedAt             string
}

type Repository interface {
	Save(context.Context, pendingUpload) error
	Get(context.Context, string) (pendingUpload, bool, error)
	// Complete 把上传会话置为 completed；updated=false 表示未更新（会话不存在或
	// 已被并发请求完成）。调用方已持有 file 元数据，无需再读回整行。
	Complete(ctx context.Context, uploadID string, file UploadedMedia, idempotencyKey string) (updated bool, err error)
	// ListStaleUploads 返回超过保留期仍未完成的上传会话（retention.go GC 入口）。
	ListStaleUploads(context.Context, time.Time, int) ([]pendingUpload, error)
	// DeleteUploadRow 按 upload_id 删除会话行（GC 第二步，对象删除成功后）。
	DeleteUploadRow(context.Context, string) error
	// ResolveByFileIDs returns the completed uploads owned by userID for the
	// given file_ids, filtered to the given category. Only rows with
	// status='completed' are returned. Unmatched file_ids are simply absent
	// from the result; callers treat absence as an authorization failure.
	ResolveByFileIDs(context.Context, []string, string, MediaCategory) (map[string]MediaAsset, error)
	// CompletedFileMetaByID returns display metadata for a completed upload by
	// its file_id, regardless of owner/category — report snapshots need the
	// file truth value, not ownership scoping. found=false when the file does
	// not exist or is not in status='completed'.
	CompletedFileMetaByID(context.Context, string) (FileMeta, bool, error)
}
