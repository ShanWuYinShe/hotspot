package media

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

// 保留期 uploads/ 孤儿清理（第 82 轮审计 P2-2）：complete 后旧上传键在预签名
// PUT TTL 内被重放会生成无行引用的永久孤儿；sweeper 每轮按「对象早于保留期
// （2 天）且 DB 无对应 storage_key 行」删除，单轮扫描页数/删除数有界，失败
// 下一轮重试。

func newOrphanSweepService(t *testing.T) (*Service, *InMemoryRepository, *coverageProvider) {
	t.Helper()
	repo := NewInMemoryRepository()
	provider := &coverageProvider{}
	return NewServiceWithRepoAndProvider(repo, provider), repo, provider
}

func orphanObject(key string, age time.Duration) ObjectSummary {
	return ObjectSummary{Key: key, LastModified: time.Now().UTC().Add(-age)}
}

// seedStorageKeyRow 写入一条引用 storageKey 的会话行（completed 控制状态）：
// 孤儿判定必须保留任何被行引用的对象。
func seedStorageKeyRow(t *testing.T, repo *InMemoryRepository, uploadID, storageKey string, completed bool) {
	t.Helper()
	upload := pendingUpload{
		UploadID: uploadID, UserID: "u1", StorageKey: storageKey, FileName: "a.bin",
		DeclaredFileSize: 10, DeclaredMimeType: "application/octet-stream", Category: CategoryChatFile,
		Completed: completed,
		CreatedAt: time.Now().UTC().Add(-72 * time.Hour).Format(time.RFC3339),
	}
	if completed {
		upload.File = UploadedMedia{FileID: "file_" + uploadID, FileSize: 10, MimeType: "application/octet-stream"}
	}
	if err := repo.Save(context.Background(), upload); err != nil {
		t.Fatalf("seed storage key row %s: %v", uploadID, err)
	}
}

// 引用判定与保留期：只删「超过 2 天且无任何会话行引用」的对象；未过期对象、
// pending 行/历史完成态行（storage_key 仍为 uploads/ 键）引用的对象、
// LastModified 缺失（零值，年龄未知）的对象都必须保留。
func TestSweepOrphanUploadObjectsDeletesOnlyUnreferencedExpiredObjects(t *testing.T) {
	svc, repo, provider := newOrphanSweepService(t)
	ctx := context.Background()
	old := staleUploadRetentionDays*24*time.Hour + time.Hour
	pendingKey := "uploads/u1/upload_pending.bin"
	legacyKey := "uploads/u1/upload_legacy_completed.bin"
	seedStorageKeyRow(t, repo, "upload_pending", pendingKey, false)
	seedStorageKeyRow(t, repo, "upload_legacy_completed", legacyKey, true)
	orphanKey := "uploads/u1/upload_orphan.bin"
	freshKey := "uploads/u1/upload_fresh.bin"
	provider.listPages = []ObjectListPage{{
		Objects: []ObjectSummary{
			orphanObject(orphanKey, old),
			orphanObject(pendingKey, old),
			orphanObject(legacyKey, old),
			orphanObject(freshKey, time.Minute),
			{Key: "uploads/u1/upload_no_time.bin"},
		},
	}}

	svc.sweepOrphanUploadObjects(ctx)

	if got := provider.deletedKeys; len(got) != 1 || got[0] != orphanKey {
		t.Fatalf("deleted = %v, want only the unreferenced expired key %q", got, orphanKey)
	}
	if len(provider.listPrefixes) != 1 || provider.listPrefixes[0] != orphanUploadObjectPrefix {
		t.Fatalf("list prefixes = %v, want [%q]", provider.listPrefixes, orphanUploadObjectPrefix)
	}
	if provider.listMaxKeys[0] != orphanUploadListPageSize {
		t.Fatalf("list page size = %d, want %d", provider.listMaxKeys[0], orphanUploadListPageSize)
	}
	// 引用行一律保留（行删除是保留期会话阶段的职责，孤儿阶段只删对象）。
	for _, uploadID := range []string{"upload_pending", "upload_legacy_completed"} {
		if _, found, err := repo.Get(ctx, uploadID); err != nil || !found {
			t.Fatalf("referenced row %s must be kept: found=%v err=%v", uploadID, found, err)
		}
	}
}

// 分页：IsTruncated 驱动 continuation-token 翻页，跨页候选都参与判定。
func TestSweepOrphanUploadObjectsPaginates(t *testing.T) {
	svc, _, provider := newOrphanSweepService(t)
	old := staleUploadRetentionDays*24*time.Hour + time.Hour
	provider.listPages = []ObjectListPage{
		{Objects: []ObjectSummary{orphanObject("uploads/u1/a.bin", old)}, IsTruncated: true, NextContinuationToken: "token-1"},
		{Objects: []ObjectSummary{orphanObject("uploads/u1/b.bin", old)}, IsTruncated: true, NextContinuationToken: "token-2"},
		{Objects: []ObjectSummary{orphanObject("uploads/u1/c.bin", old)}},
	}

	svc.sweepOrphanUploadObjects(context.Background())

	if len(provider.listTokens) != 3 {
		t.Fatalf("list calls = %d, want 3", len(provider.listTokens))
	}
	for i, want := range []string{"", "token-1", "token-2"} {
		if provider.listTokens[i] != want {
			t.Fatalf("list call %d token = %q, want %q", i, provider.listTokens[i], want)
		}
	}
	if len(provider.deletedKeys) != 3 {
		t.Fatalf("deleted = %v, want 3 objects", provider.deletedKeys)
	}
}

// 单轮删除上限：一页内候选过多时只删 orphanUploadDeleteLimit 条并停止（剩余
// 对象仍满足条件，下一轮继续）。
func TestSweepOrphanUploadObjectsStopsAtDeleteLimit(t *testing.T) {
	svc, _, provider := newOrphanSweepService(t)
	old := staleUploadRetentionDays*24*time.Hour + time.Hour
	objects := make([]ObjectSummary, 0, orphanUploadDeleteLimit+10)
	for i := 0; i < orphanUploadDeleteLimit+10; i++ {
		objects = append(objects, orphanObject(fmt.Sprintf("uploads/u1/orphan_%03d.bin", i), old))
	}
	provider.listPages = []ObjectListPage{{Objects: objects, IsTruncated: true, NextContinuationToken: "more"}}

	svc.sweepOrphanUploadObjects(context.Background())

	if len(provider.deletedKeys) != orphanUploadDeleteLimit {
		t.Fatalf("deleted = %d, want delete limit %d", len(provider.deletedKeys), orphanUploadDeleteLimit)
	}
	if len(provider.listTokens) != 1 {
		t.Fatalf("delete limit must stop pagination, list calls = %d", len(provider.listTokens))
	}
}

// 单轮扫描页数上限：异常多页时不会无限翻页，最多 orphanUploadListMaxPages 页。
func TestSweepOrphanUploadObjectsBoundedPages(t *testing.T) {
	svc, _, provider := newOrphanSweepService(t)
	old := staleUploadRetentionDays*24*time.Hour + time.Hour
	for i := 0; i < orphanUploadListMaxPages+5; i++ {
		provider.listPages = append(provider.listPages, ObjectListPage{
			Objects:               []ObjectSummary{orphanObject(fmt.Sprintf("uploads/u1/page_%02d.bin", i), old)},
			IsTruncated:           true,
			NextContinuationToken: fmt.Sprintf("token-%02d", i),
		})
	}

	svc.sweepOrphanUploadObjects(context.Background())

	if len(provider.listTokens) != orphanUploadListMaxPages {
		t.Fatalf("list calls = %d, want page cap %d", len(provider.listTokens), orphanUploadListMaxPages)
	}
	if len(provider.deletedKeys) != orphanUploadListMaxPages {
		t.Fatalf("deleted = %d, want %d", len(provider.deletedKeys), orphanUploadListMaxPages)
	}
}

// 失败路径：列举失败 / 引用查询失败时不得删除任何对象（否则可能误删在途上传），
// 只告警留待下一轮。
func TestSweepOrphanUploadObjectsKeepsObjectsOnFailures(t *testing.T) {
	old := staleUploadRetentionDays*24*time.Hour + time.Hour

	t.Run("list failure", func(t *testing.T) {
		svc, _, provider := newOrphanSweepService(t)
		provider.listErr = errors.New("list boom")
		svc.sweepOrphanUploadObjects(context.Background())
		if len(provider.deletedKeys) != 0 {
			t.Fatalf("no object may be deleted when listing fails, got %v", provider.deletedKeys)
		}
	})

	t.Run("budget exhausted mid-sweep", func(t *testing.T) {
		svc, _, provider := newOrphanSweepService(t)
		provider.listPages = []ObjectListPage{{
			Objects: []ObjectSummary{orphanObject("uploads/u1/orphan.bin", old)},
		}}
		// 已取消的 ctx：列举/引用查询（fake 不检查 ctx）之后、删除之前检查预算，
		// 不得继续向已取消的 ctx 发删除请求。
		canceled, cancel := context.WithCancel(context.Background())
		cancel()
		svc.sweepOrphanUploadObjects(canceled)
		if len(provider.deletedKeys) != 0 {
			t.Fatalf("canceled budget must not delete objects, got %v", provider.deletedKeys)
		}
	})

	t.Run("reference query failure", func(t *testing.T) {
		base := NewInMemoryRepository()
		provider := &coverageProvider{listPages: []ObjectListPage{{
			Objects: []ObjectSummary{orphanObject("uploads/u1/orphan.bin", old)},
		}}}
		repo := &orphanReferenceErrorRepo{InMemoryRepository: base, err: errors.New("reference boom")}
		NewServiceWithRepoAndProvider(repo, provider).sweepOrphanUploadObjects(context.Background())
		if len(provider.deletedKeys) != 0 {
			t.Fatalf("no object may be deleted when reference check fails, got %v", provider.deletedKeys)
		}
	})
}

// orphanReferenceErrorRepo 注入 ExistingStorageKeys 错误：引用状态不可确认时
// 孤儿清理必须整体放弃本轮（可能误删在途上传的对象）。
type orphanReferenceErrorRepo struct {
	*InMemoryRepository
	err error
}

func (r *orphanReferenceErrorRepo) ExistingStorageKeys(context.Context, []string) (map[string]struct{}, error) {
	return nil, r.err
}

// runRetentionSweep 两阶段串联：先回收超期未完成会话（对象+行），再清理
// 无行引用的 uploads/ 孤儿对象；同一轮内两种残留都被回收。
func TestRunRetentionSweepCoversSessionsAndOrphans(t *testing.T) {
	svc, repo, provider := newOrphanSweepService(t)
	ctx := context.Background()
	staleAt := time.Now().UTC().AddDate(0, 0, -(staleUploadRetentionDays + 1))
	if err := repo.Save(ctx, pendingUpload{
		UploadID: "upload_stale_session", UserID: "u1", StorageKey: "uploads/u1/upload_stale_session.bin",
		FileName: "a.bin", DeclaredFileSize: 10, DeclaredMimeType: "application/octet-stream",
		Category: CategoryChatFile, CreatedAt: staleAt.Format(time.RFC3339),
	}); err != nil {
		t.Fatalf("seed stale session: %v", err)
	}
	// 孤儿阶段看到的是会话阶段删除后的列举结果：只剩无行引用的孤儿对象。
	provider.listPages = []ObjectListPage{{
		Objects: []ObjectSummary{orphanObject("uploads/u1/upload_replayed.bin", staleUploadRetentionDays*24*time.Hour+time.Hour)},
	}}

	svc.runRetentionSweep(ctx)

	if _, found, err := repo.Get(ctx, "upload_stale_session"); err != nil || found {
		t.Fatalf("stale session must be swept: found=%v err=%v", found, err)
	}
	deleted := map[string]int{}
	for _, key := range provider.deletedKeys {
		deleted[key]++
	}
	if deleted["uploads/u1/upload_stale_session.bin"] != 1 {
		t.Fatalf("stale session object must be deleted once, deleted=%v", provider.deletedKeys)
	}
	if deleted["uploads/u1/upload_replayed.bin"] != 1 {
		t.Fatalf("orphan object must be deleted once, deleted=%v", provider.deletedKeys)
	}
}
