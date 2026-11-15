package media

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/testdb"
)

// 真实库 IT（testdb.BeginScope：事务内写入、结束回滚，不提交任何行）：配额
// 释放/清理路径必须走 PG 仓储的真实 SQL——complete 确定性失败释放该会话行后
// 用量归零；presign 配额被拒时清理该用户超宽限期的弃置会话后放行、且不误删
// 宽限期内会话与他人会话（第 82 轮审计 P1-3）。
func TestMediaQuotaReleasePostgresIT(t *testing.T) {
	testdb.SkipIfShort(t)
	testdb.Open(t)

	t.Run("deterministic complete failure releases pending session", func(t *testing.T) {
		factory, cleanup := testdb.BeginScope(t)
		defer cleanup()
		repo := NewPostgresRepository(factory)
		ctx := context.Background()
		const userID = "it_media_quota_release"
		const uploadID = "upload_it_release_html"
		const storageKey = "uploads/" + userID + "/" + uploadID + ".html"
		// 黑名单扩展名的历史会话（模拟修复前写入、未经扩展名校验的行）：
		// complete 用会话行 file_name 复核时确定性失败。
		if err := repo.Save(ctx, pendingUpload{
			UploadID: uploadID, UserID: userID, StorageKey: storageKey,
			FileName: "page.html", DeclaredFileSize: 14, DeclaredMimeType: "text/plain",
			Category: CategoryChatFile, CreatedAt: time.Now().UTC().Format(time.RFC3339),
		}); err != nil {
			t.Fatalf("save pending session: %v", err)
		}
		provider := &mediaStubProvider{headResp: ObjectMetadata{FileSize: 14, MimeType: "text/plain"}}
		svc := NewServiceWithRepoAndProvider(repo, provider)
		if _, err := svc.Complete(mediaContextWithUserAndKey(userID, "it-key"), uploadID); !errors.Is(err, ErrUploadConflict) {
			t.Fatalf("expected ErrUploadConflict, got %v", err)
		}
		if _, found, err := repo.Get(ctx, uploadID); err != nil || found {
			t.Fatalf("rejected session must be released: found=%v err=%v", found, err)
		}
		if usage, err := repo.PendingUploadUsage(ctx, userID); err != nil || usage.Count != 0 || usage.DeclaredBytes != 0 {
			t.Fatalf("released session must free quota: usage=%#v err=%v", usage, err)
		}
		if len(provider.deletedKeys) != 1 || provider.deletedKeys[0] != storageKey {
			t.Fatalf("upload object must be deleted, deleted = %#v", provider.deletedKeys)
		}
	})

	t.Run("quota rejection cleans only this user's expired sessions", func(t *testing.T) {
		factory, cleanup := testdb.BeginScope(t)
		defer cleanup()
		repo := NewPostgresRepository(factory)
		ctx := context.Background()
		const userID = "it_media_quota_cleanup"
		const otherUserID = "it_media_quota_cleanup_other"
		staleAt := time.Now().UTC().Add(-(pendingUploadCleanupGrace + time.Minute))
		for i := 0; i < pendingUploadSessionLimit; i++ {
			uploadID := fmt.Sprintf("upload_it_cleanup_stale_%02d", i)
			if err := repo.Save(ctx, pendingUpload{
				UploadID: uploadID, UserID: userID, StorageKey: "uploads/" + userID + "/" + uploadID + ".png",
				FileName: "a.png", DeclaredFileSize: 100, DeclaredMimeType: "image/png",
				Category: CategoryAvatar, CreatedAt: staleAt.Format(time.RFC3339),
			}); err != nil {
				t.Fatalf("save stale session: %v", err)
			}
		}
		if err := repo.Save(ctx, pendingUpload{
			UploadID: "upload_it_cleanup_fresh", UserID: userID, StorageKey: "uploads/" + userID + "/upload_it_cleanup_fresh.png",
			FileName: "a.png", DeclaredFileSize: 100, DeclaredMimeType: "image/png",
			Category: CategoryAvatar, CreatedAt: time.Now().UTC().Format(time.RFC3339),
		}); err != nil {
			t.Fatalf("save fresh session: %v", err)
		}
		if err := repo.Save(ctx, pendingUpload{
			UploadID: "upload_it_cleanup_other", UserID: otherUserID, StorageKey: "uploads/" + otherUserID + "/upload_it_cleanup_other.png",
			FileName: "a.png", DeclaredFileSize: 100, DeclaredMimeType: "image/png",
			Category: CategoryAvatar, CreatedAt: staleAt.Format(time.RFC3339),
		}); err != nil {
			t.Fatalf("save other user's session: %v", err)
		}

		provider := &quotaStubProvider{mediaStubProvider: mediaStubProvider{
			presignResp: PresignedUpload{
				UploadID: "upload_it_cleanup_new", UploadURL: "https://storage.example.com/upload",
				Headers: map[string]string{"Content-Type": "image/png"}, ExpiresAt: "2026-03-30T00:00:00Z",
				StorageKey: "uploads/" + userID + "/upload_it_cleanup_new.png",
			},
		}}
		svc := NewServiceWithRepoAndProvider(repo, provider)
		resp, err := svc.BatchPresign(mediaContextWithUserAndKey(userID, ""), []PresignFileInput{{
			FileName: "a.png", FileSize: 100, MimeType: "image/png", Category: CategoryAvatar,
		}})
		if err != nil {
			t.Fatalf("cleanup must free quota and allow presign, got %v", err)
		}
		if len(resp.Uploads) != 1 || provider.presignCalls != 1 {
			t.Fatalf("presign uploads = %d calls = %d, want 1/1", len(resp.Uploads), provider.presignCalls)
		}
		if len(provider.deletedKeys) != pendingUploadSessionLimit {
			t.Fatalf("deleted objects = %d, want %d", len(provider.deletedKeys), pendingUploadSessionLimit)
		}
		for i := 0; i < pendingUploadSessionLimit; i++ {
			uploadID := fmt.Sprintf("upload_it_cleanup_stale_%02d", i)
			if _, found, err := repo.Get(ctx, uploadID); err != nil || found {
				t.Fatalf("stale session %s must be released: found=%v err=%v", uploadID, found, err)
			}
		}
		if _, found, err := repo.Get(ctx, "upload_it_cleanup_fresh"); err != nil || !found {
			t.Fatalf("session within grace must be kept: found=%v err=%v", found, err)
		}
		if _, found, err := repo.Get(ctx, "upload_it_cleanup_other"); err != nil || !found {
			t.Fatalf("other user's session must be kept: found=%v err=%v", found, err)
		}
		if usage, err := repo.PendingUploadUsage(ctx, userID); err != nil || usage.Count != 2 {
			t.Fatalf("usage after cleanup = %#v err=%v, want 2 pending sessions", usage, err)
		}
		if usage, err := repo.PendingUploadUsage(ctx, otherUserID); err != nil || usage.Count != 1 {
			t.Fatalf("other user's usage = %#v err=%v, want 1 pending session", usage, err)
		}
	})
}
