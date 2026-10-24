package media

import (
	"context"
	"sort"
	"sync"
	"time"
)

type InMemoryRepository struct {
	mu      sync.RWMutex
	uploads map[string]pendingUpload
}

func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{uploads: make(map[string]pendingUpload)}
}

func (r *InMemoryRepository) Save(ctx context.Context, upload pendingUpload) error {
	_ = ctx
	r.mu.Lock()
	defer r.mu.Unlock()
	r.uploads[upload.UploadID] = upload
	return nil
}

func (r *InMemoryRepository) Get(ctx context.Context, uploadID string) (pendingUpload, bool, error) {
	_ = ctx
	r.mu.RLock()
	defer r.mu.RUnlock()
	upload, ok := r.uploads[uploadID]
	return upload, ok, nil
}

func (r *InMemoryRepository) Complete(ctx context.Context, uploadID string, file UploadedMedia, idempotencyKey string) (bool, error) {
	_ = ctx
	r.mu.Lock()
	defer r.mu.Unlock()
	upload, ok := r.uploads[uploadID]
	if !ok {
		return false, nil
	}
	// 与 PG 实现的 status <> 'completed' 守卫同口径：已完成的会话不重复改写。
	if upload.Completed {
		return false, nil
	}
	upload.Completed = true
	upload.File = file
	upload.CompletedIdempotencyKey = idempotencyKey
	// PG queryCompleteUploadSession 同时写 completed_at；内存实现此前漏写，
	// 导致完成态会话的审计字段在降级模式下恒为空（契约测试发现）。
	upload.CompletedAt = time.Now().UTC().Format(time.RFC3339)
	r.uploads[uploadID] = upload
	return true, nil
}

func (r *InMemoryRepository) ResolveByFileIDs(_ context.Context, fileIDs []string, userID string, category MediaCategory) (map[string]MediaAsset, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	want := make(map[string]struct{}, len(fileIDs))
	for _, id := range fileIDs {
		want[id] = struct{}{}
	}
	assets := make(map[string]MediaAsset)
	for _, upload := range r.uploads {
		if !upload.Completed || upload.UserID != userID || upload.Category != category {
			continue
		}
		if upload.File.FileID == "" {
			continue
		}
		if _, ok := want[upload.File.FileID]; !ok {
			continue
		}
		assets[upload.File.FileID] = MediaAsset{
			FileID:       upload.File.FileID,
			StorageKey:   upload.StorageKey,
			ThumbnailURL: upload.File.ThumbnailURL,
			Width:        upload.File.Width,
			Height:       upload.File.Height,
			MimeType:     upload.File.MimeType,
		}
	}
	return assets, nil
}

// CompletedFileMetaByID 在内存实现里按 uploadID 复用已完成上传的元数据
// （测试桩场景：file_id 与 upload_id 同值约定）。
func (r *InMemoryRepository) CompletedFileMetaByID(_ context.Context, fileID string) (FileMeta, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	// 与 PG 同键：按完成态的 file_id 查找（此前以 upload_id 充当 file_id，
	// 两者不同值时内存实现恒查不到；契约测试发现）。
	for _, upload := range r.uploads {
		if !upload.Completed || upload.File.FileID != fileID {
			continue
		}
		return FileMeta{
			Category:     upload.Category,
			StorageKey:   upload.StorageKey,
			ThumbnailURL: upload.File.ThumbnailURL,
			Width:        upload.File.Width,
			Height:       upload.File.Height,
			MimeType:     upload.File.MimeType,
			FileName:     upload.FileName,
			FileSize:     upload.File.FileSize,
			Duration:     upload.File.Duration,
		}, true, nil
	}
	return FileMeta{}, false, nil
}

func (r *InMemoryRepository) ListStaleUploads(_ context.Context, cutoff time.Time, limit int) ([]pendingUpload, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]pendingUpload, 0)
	for _, item := range r.uploads {
		if item.Completed {
			continue
		}
		created, err := time.Parse(time.RFC3339, item.CreatedAt)
		if err != nil || !created.Before(cutoff) {
			continue
		}
		items = append(items, item)
	}
	// 与 PG queryListStaleUploads（ORDER BY created_at ASC LIMIT $2）同序：
	// 旧行优先处理；此前按 map 随机序并提前 break，GC 顺序与批次不稳定。
	sort.Slice(items, func(i, j int) bool {
		if items[i].CreatedAt == items[j].CreatedAt {
			return items[i].UploadID < items[j].UploadID
		}
		return items[i].CreatedAt < items[j].CreatedAt
	})
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}

func (r *InMemoryRepository) DeleteUploadRow(_ context.Context, uploadID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	// 与 PG queryDeleteUploadRow（WHERE status <> 'completed'）同口径：保留期 GC
	// 只清理未完成会话，已完成上传是业务数据，不能被 GC 误删（内存实现此前无条件删除）。
	if upload, ok := r.uploads[uploadID]; ok && upload.Completed {
		return nil
	}
	delete(r.uploads, uploadID)
	return nil
}
