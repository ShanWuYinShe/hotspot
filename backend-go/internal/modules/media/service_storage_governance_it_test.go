package media

import (
	"context"
	"errors"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/testdb"
)

// 真实库 IT（testdb.BeginScope：事务内写入、结束回滚，不提交/不污染测试库）：
// 账号级完成态媒体配额与 uploads/ 孤儿清理的引用判定必须走 PG 仓储真实 SQL
//（列歧义、参数个数、类型转换只有真实库能暴露；stub 单测无法覆盖）。

func saveITUploadRow(t *testing.T, repo Repository, upload pendingUpload) {
	t.Helper()
	if upload.CreatedAt == "" {
		upload.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	if err := repo.Save(context.Background(), upload); err != nil {
		t.Fatalf("save upload row %s: %v", upload.UploadID, err)
	}
}

// CompletedStorageBytes：只汇总该用户 status='completed' 行的 final_file_size，
// 忽略未完成行与他人行；final_file_size 为 NULL 的历史完成态行按 0 计。
func TestMediaCompletedStorageBytesPostgresIT(t *testing.T) {
	testdb.SkipIfShort(t)
	testdb.Open(t)
	ctx := context.Background()
	factory, cleanup := testdb.BeginScope(t)
	defer cleanup()
	repo := NewPostgresRepository(factory)
	const userID = "it_media_storage_bytes"
	const otherUserID = "it_media_storage_bytes_other"

	saveITUploadRow(t, repo, pendingUpload{
		UploadID: "upload_it_bytes_done_a", UserID: userID, StorageKey: "published/" + userID + "/upload_it_bytes_done_a.png",
		FileName: "a.png", DeclaredFileSize: 100, DeclaredMimeType: "image/png", Category: CategoryMomentImage,
		Completed: true, File: UploadedMedia{FileID: "file_it_bytes_a", FileSize: 100, MimeType: "image/png"},
	})
	saveITUploadRow(t, repo, pendingUpload{
		UploadID: "upload_it_bytes_done_b", UserID: userID, StorageKey: "published/" + userID + "/upload_it_bytes_done_b.png",
		FileName: "b.png", DeclaredFileSize: 250, DeclaredMimeType: "image/png", Category: CategoryMomentImage,
		Completed: true, File: UploadedMedia{FileID: "file_it_bytes_b", FileSize: 250, MimeType: "image/png"},
	})
	// 未完成行：声明 999 字节，不得计入完成态总量。
	saveITUploadRow(t, repo, pendingUpload{
		UploadID: "upload_it_bytes_pending", UserID: userID, StorageKey: "uploads/" + userID + "/upload_it_bytes_pending.png",
		FileName: "c.png", DeclaredFileSize: 999, DeclaredMimeType: "image/png", Category: CategoryMomentImage,
	})
	// 他人完成态行：不得计入。
	saveITUploadRow(t, repo, pendingUpload{
		UploadID: "upload_it_bytes_other", UserID: otherUserID, StorageKey: "published/" + otherUserID + "/upload_it_bytes_other.png",
		FileName: "d.png", DeclaredFileSize: 500, DeclaredMimeType: "image/png", Category: CategoryMomentImage,
		Completed: true, File: UploadedMedia{FileID: "file_it_bytes_other", FileSize: 500, MimeType: "image/png"},
	})

	total, err := repo.CompletedStorageBytes(ctx, userID)
	if err != nil {
		t.Fatalf("CompletedStorageBytes: %v", err)
	}
	if total != 350 {
		t.Fatalf("completed storage bytes = %d, want 350 (100+250)", total)
	}
	if total, err := repo.CompletedStorageBytes(ctx, "it_media_storage_bytes_empty"); err != nil || total != 0 {
		t.Fatalf("empty account = (%d, %v), want (0, nil) — COALESCE 必须让空集返回 0", total, err)
	}
}

// ExistingStorageKeys：批量引用判定（孤儿清理用）——任意状态的行都算引用，
// 未命中的键不得出现在结果中；空入参不查库直接返回空集合。
func TestMediaExistingStorageKeysPostgresIT(t *testing.T) {
	testdb.SkipIfShort(t)
	testdb.Open(t)
	ctx := context.Background()
	factory, cleanup := testdb.BeginScope(t)
	defer cleanup()
	repo := NewPostgresRepository(factory)
	const userID = "it_media_orphan_keys"
	pendingKey := "uploads/" + userID + "/upload_it_keys_pending.png"
	publishedKey := "published/" + userID + "/upload_it_keys_done.png"
	legacyKey := "uploads/" + userID + "/upload_it_keys_legacy.png"
	saveITUploadRow(t, repo, pendingUpload{
		UploadID: "upload_it_keys_pending", UserID: userID, StorageKey: pendingKey,
		FileName: "a.png", DeclaredFileSize: 10, DeclaredMimeType: "image/png", Category: CategoryMomentImage,
	})
	saveITUploadRow(t, repo, pendingUpload{
		UploadID: "upload_it_keys_done", UserID: userID, StorageKey: publishedKey,
		FileName: "b.png", DeclaredFileSize: 10, DeclaredMimeType: "image/png", Category: CategoryMomentImage,
		Completed: true, File: UploadedMedia{FileID: "file_it_keys_done", FileSize: 10, MimeType: "image/png"},
	})
	// copy-on-complete 之前的历史完成态行：storage_key 仍是 uploads/ 键。
	saveITUploadRow(t, repo, pendingUpload{
		UploadID: "upload_it_keys_legacy", UserID: userID, StorageKey: legacyKey,
		FileName: "c.png", DeclaredFileSize: 10, DeclaredMimeType: "image/png", Category: CategoryMomentImage,
		Completed: true, File: UploadedMedia{FileID: "file_it_keys_legacy", FileSize: 10, MimeType: "image/png"},
	})

	candidates := []string{pendingKey, publishedKey, legacyKey, "uploads/" + userID + "/upload_it_keys_orphan.png"}
	existing, err := repo.ExistingStorageKeys(ctx, candidates)
	if err != nil {
		t.Fatalf("ExistingStorageKeys: %v", err)
	}
	for _, key := range []string{pendingKey, publishedKey, legacyKey} {
		if _, ok := existing[key]; !ok {
			t.Fatalf("referenced key %q missing from result %v", key, existing)
		}
	}
	if _, ok := existing["uploads/"+userID+"/upload_it_keys_orphan.png"]; ok {
		t.Fatalf("unreferenced key must not be reported: %v", existing)
	}
	if empty, err := repo.ExistingStorageKeys(ctx, nil); err != nil || len(empty) != 0 {
		t.Fatalf("empty input = (%v, %v), want empty map without error", empty, err)
	}
}

// 端到端（真实库 + 真实 attempt 流程）：配额在发布前拒绝、已完成行重复
// complete 不被配额误伤；孤儿清理用真实引用查询删除无行对象、保留有行对象。
func TestMediaStorageGovernancePostgresIT(t *testing.T) {
	testdb.SkipIfShort(t)
	testdb.Open(t)

	t.Run("complete rejected over account quota before publishing", func(t *testing.T) {
		factory, cleanup := testdb.BeginScope(t)
		defer cleanup()
		repo := NewPostgresRepository(factory)
		ctx := context.Background()
		const userID = "it_media_quota_full"
		// 已接近上限：只差 1 字节，本次 1024 字节必超。
		saveITUploadRow(t, repo, pendingUpload{
			UploadID: "upload_it_quota_done", UserID: userID, StorageKey: "published/" + userID + "/upload_it_quota_done.png",
			FileName: "a.png", DeclaredFileSize: int(completedMediaStorageLimitBytes) - 1,
			DeclaredMimeType: "image/png", Category: CategoryMomentImage, Completed: true,
			File: UploadedMedia{FileID: "file_it_quota_done", FileSize: int(completedMediaStorageLimitBytes) - 1, MimeType: "image/png"},
		})
		const uploadID = "upload_it_quota_pending"
		saveITUploadRow(t, repo, pendingUpload{
			UploadID: uploadID, UserID: userID, StorageKey: "uploads/" + userID + "/" + uploadID + ".txt",
			FileName: "a.txt", DeclaredFileSize: 1024, DeclaredMimeType: "text/plain", Category: CategoryChatFile,
		})
		provider := &mediaStubProvider{headResp: ObjectMetadata{FileSize: 1024, MimeType: "text/plain"}}
		svc := NewServiceWithRepoAndProvider(repo, provider)
		if _, err := svc.Complete(mediaContextWithUserAndKey(userID, ""), uploadID); !errors.Is(err, ErrStorageQuotaExceeded) {
			t.Fatalf("expected ErrStorageQuotaExceeded, got %v", err)
		}
		if len(provider.copies) != 0 || len(provider.deletedKeys) != 0 {
			t.Fatalf("quota rejection must not publish: copies=%v deleted=%v", provider.copies, provider.deletedKeys)
		}
		upload, found, err := repo.Get(ctx, uploadID)
		if err != nil || !found || upload.Completed {
			t.Fatalf("pending session must stay pending: found=%v upload=%#v err=%v", found, upload, err)
		}
	})

	t.Run("orphan sweep deletes only unreferenced expired objects", func(t *testing.T) {
		factory, cleanup := testdb.BeginScope(t)
		defer cleanup()
		repo := NewPostgresRepository(factory)
		ctx := context.Background()
		const userID = "it_media_orphan_sweep"
		referencedKey := "uploads/" + userID + "/upload_it_orphan_referenced.bin"
		saveITUploadRow(t, repo, pendingUpload{
			UploadID: "upload_it_orphan_referenced", UserID: userID, StorageKey: referencedKey,
			FileName: "a.bin", DeclaredFileSize: 10, DeclaredMimeType: "application/octet-stream", Category: CategoryChatFile,
		})
		old := staleUploadRetentionDays*24*time.Hour + time.Hour
		orphanKey := "uploads/" + userID + "/upload_it_orphan_replayed.bin"
		freshKey := "uploads/" + userID + "/upload_it_orphan_fresh.bin"
		provider := &coverageProvider{listPages: []ObjectListPage{{
			Objects: []ObjectSummary{
				{Key: referencedKey, LastModified: time.Now().UTC().Add(-old)},
				{Key: orphanKey, LastModified: time.Now().UTC().Add(-old)},
				{Key: freshKey, LastModified: time.Now().UTC().Add(-time.Minute)},
			},
		}}}
		svc := NewServiceWithRepoAndProvider(repo, provider)
		svc.sweepOrphanUploadObjects(ctx)
		if len(provider.deletedKeys) != 1 || provider.deletedKeys[0] != orphanKey {
			t.Fatalf("deleted = %v, want only %q (referenced row kept, fresh kept)", provider.deletedKeys, orphanKey)
		}
		if _, found, err := repo.Get(ctx, "upload_it_orphan_referenced"); err != nil || !found {
			t.Fatalf("referenced row must be kept: found=%v err=%v", found, err)
		}
	})
}
