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

// PendingUploadUsage 是单个用户「未完成上传会话」的聚合快照（预签名配额
// 判定用，见 Service.checkPresignQuota）。只统计 status <> 'completed' 的
// 会话：已完成上传是业务数据，不占配额。DeclaredBytes 来自客户端声明的
// file_size，不可信，仅用于抑制海量 presign 弃置（真实 body 由网关
// client_max_body_size 上限兜底）。
type PendingUploadUsage struct {
	Count         int64
	DeclaredBytes int64
}

type Repository interface {
	Save(context.Context, pendingUpload) error
	Get(context.Context, string) (pendingUpload, bool, error)
	// Complete 把上传会话置为 completed 并写入发布键 storageKey（copy-on-complete
	// 后的不可写键）；updated=false 表示未更新（会话不存在或已被并发请求完成）。
	// 调用方已持有 file 元数据，无需再读回整行。
	Complete(ctx context.Context, uploadID string, file UploadedMedia, storageKey, idempotencyKey string) (updated bool, err error)
	// PendingUploadUsage 按 user_id 聚合未完成上传会话数与声明字节总量
	// （预签名配额判定；completed 会话不占配额）。SQL 见
	// repo_postgres_queries.go 的 queryPendingUploadUsage，内存实现同口径。
	PendingUploadUsage(context.Context, string) (PendingUploadUsage, error)
	// ListStaleUploads 返回超过保留期仍未完成的上传会话（retention.go GC 入口）。
	ListStaleUploads(context.Context, time.Time, int) ([]pendingUpload, error)
	// ListStaleUploadsForUser 返回指定用户超过宽限期仍未完成的上传会话（按
	// created_at 升序，限量；预签名配额被拒时的 best-effort 释放路径用，见
	// Service.cleanupStalePendingUploads）。配额只统计未完成会话，弃置会话
	// 若不提前清理会把用户上传功能锁到保留期。
	ListStaleUploadsForUser(context.Context, string, time.Time, int) ([]pendingUpload, error)
	// DeleteUploadRow 按 upload_id 删除会话行（GC 第二步，对象删除成功后）。
	DeleteUploadRow(context.Context, string) error
	// CompletedStorageBytes 返回该用户已完成上传（status='completed'）的
	// final_file_size 总和：账号级完成态媒体配额判定用（见
	// Service.checkCompletedStorageQuota）。空集返回 0；query 命中
	// idx_media_upload_sessions_user_created_at 的 user_id 前缀。
	CompletedStorageBytes(context.Context, string) (int64, error)
	// ExistingStorageKeys 返回给定存储键中仍被 media_upload_sessions 行引用的
	// 子集（保留期 uploads/ 孤儿清理用：只有「无行引用且超过保留期」的对象才可
	// 删除）。批量查询命中 idx_media_upload_sessions_storage_key，调用方按页
	// 提交候选键，避免逐键查库。
	ExistingStorageKeys(context.Context, []string) (map[string]struct{}, error)
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
