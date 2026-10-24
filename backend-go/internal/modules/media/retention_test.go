package media

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// 保留期 GC 的失败路径此前无测试：对象删除失败必须保留会话行（下一轮重试），
// 不能出现「对象还在但行已删 → 孤儿对象永久泄漏」；已完成上传不得被 GC 删除。
type retentionProvider struct {
	mu      sync.Mutex
	deleted []string
	failOn  map[string]error
}

func (p *retentionProvider) Presign(context.Context, PresignFileInput, string) (PresignedUpload, error) {
	return PresignedUpload{}, errors.New("not implemented")
}
func (p *retentionProvider) PresignGet(context.Context, string, time.Duration) (string, error) {
	return "", errors.New("not implemented")
}
func (p *retentionProvider) Head(context.Context, string) (ObjectMetadata, error) {
	return ObjectMetadata{}, errors.New("not implemented")
}
func (p *retentionProvider) ReadPrefix(context.Context, string, int64) ([]byte, error) {
	return nil, errors.New("not implemented")
}
func (p *retentionProvider) Delete(_ context.Context, storageKey string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.deleted = append(p.deleted, storageKey)
	if err := p.failOn[storageKey]; err != nil {
		return err
	}
	return nil
}
func (p *retentionProvider) GetObject(context.Context, string) ([]byte, error) {
	return nil, errors.New("not implemented")
}
func (p *retentionProvider) PutObject(context.Context, string, string, []byte) error {
	return errors.New("not implemented")
}

func (p *retentionProvider) deletedKeys() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]string(nil), p.deleted...)
}

func newRetentionService(t *testing.T) (*Service, *InMemoryRepository, *retentionProvider) {
	t.Helper()
	repo := NewInMemoryRepository()
	provider := &retentionProvider{failOn: map[string]error{}}
	return NewServiceWithRepoAndProvider(repo, provider), repo, provider
}

func seedPendingUpload(t *testing.T, repo Repository, uploadID string, createdAt time.Time) {
	t.Helper()
	if err := repo.Save(context.Background(), pendingUpload{
		UploadID:         uploadID,
		UserID:           "it_retention_user",
		StorageKey:       "uploads/" + uploadID,
		FileName:         uploadID + ".png",
		DeclaredFileSize: 1024,
		DeclaredMimeType: "image/png",
		Category:         CategoryAvatar,
		CreatedAt:        createdAt.UTC().Format(time.RFC3339),
	}); err != nil {
		t.Fatalf("save %s: %v", uploadID, err)
	}
}

func TestSweepStaleUploadsDeletesObjectAndRow(t *testing.T) {
	svc, repo, provider := newRetentionService(t)
	ctx := context.Background()
	now := time.Now().UTC()
	seedPendingUpload(t, repo, "upload_stale", now.AddDate(0, 0, -8))
	seedPendingUpload(t, repo, "upload_fresh", now.Add(-time.Hour))
	// 已完成上传即使 created_at 很旧也不得被保留期 GC 清理（业务数据）。
	completed := pendingUpload{
		UploadID: "upload_completed", UserID: "it_retention_user", StorageKey: "uploads/upload_completed",
		FileName: "done.png", DeclaredFileSize: 1024, DeclaredMimeType: "image/png",
		Category: CategoryAvatar, CreatedAt: now.AddDate(0, 0, -8).Format(time.RFC3339),
	}
	if err := repo.Save(ctx, completed); err != nil {
		t.Fatalf("save completed: %v", err)
	}
	if ok, err := repo.Complete(ctx, "upload_completed", UploadedMedia{FileID: "file_completed", FileSize: 1024, MimeType: "image/png"}, "idem"); err != nil || !ok {
		t.Fatalf("complete: %v %v", ok, err)
	}

	svc.sweepStaleUploads(ctx)

	if keys := provider.deletedKeys(); len(keys) != 1 || keys[0] != "uploads/upload_stale" {
		t.Fatalf("deleted objects = %v, want [uploads/upload_stale]", keys)
	}
	if _, found, _ := repo.Get(ctx, "upload_stale"); found {
		t.Fatal("stale upload row must be deleted")
	}
	if _, found, _ := repo.Get(ctx, "upload_fresh"); !found {
		t.Fatal("fresh upload must be kept")
	}
	if got, found, _ := repo.Get(ctx, "upload_completed"); !found || !got.Completed {
		t.Fatalf("completed upload wrongly removed: found=%v record=%#v", found, got)
	}

	// 第二轮无待清理项：不再调用 Delete。
	before := len(provider.deletedKeys())
	svc.sweepStaleUploads(ctx)
	if after := len(provider.deletedKeys()); after != before {
		t.Fatalf("second sweep deleted %d more objects", after-before)
	}
}

func TestSweepStaleUploadsKeepsRowWhenObjectDeleteFails(t *testing.T) {
	svc, repo, provider := newRetentionService(t)
	ctx := context.Background()
	now := time.Now().UTC()
	seedPendingUpload(t, repo, "upload_retry", now.AddDate(0, 0, -9))
	provider.failOn["uploads/upload_retry"] = errors.New("storage temporarily unavailable")

	svc.sweepStaleUploads(ctx)

	// 对象删除失败：行必须保留，下一轮重试（否则对象成为永久孤儿）。
	if _, found, _ := repo.Get(ctx, "upload_retry"); !found {
		t.Fatal("row must be kept when object delete fails")
	}
	if keys := provider.deletedKeys(); len(keys) != 1 {
		t.Fatalf("expected one delete attempt, got %v", keys)
	}

	// 恢复后下一轮成功删除对象 + 行。
	provider.mu.Lock()
	delete(provider.failOn, "uploads/upload_retry")
	provider.mu.Unlock()
	svc.sweepStaleUploads(ctx)
	if _, found, _ := repo.Get(ctx, "upload_retry"); found {
		t.Fatal("row must be deleted on retry success")
	}
}
