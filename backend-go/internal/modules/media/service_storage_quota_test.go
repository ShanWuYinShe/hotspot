package media

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/testdb"
)

// 账号级完成态媒体配额（第 82 轮审计 P2-3）：complete 阶段校验
// 「已完成媒体总字节 + 本次文件」≤ completedMediaStorageLimitBytes，超限在
// 发布/拷贝之前拒绝（零副作用），handler 映射 10400。

const testChatFileMime = "text/plain"

// seedCompletedMedia 直接写入已完成会话行（绕过 complete 流程）：配额聚合只
// 关心 status='completed' 与 final_file_size，与写入来源无关。
func seedCompletedMedia(t *testing.T, repo *InMemoryRepository, userID, uploadID string, size int) {
	t.Helper()
	upload := pendingUpload{
		UploadID:         uploadID,
		UserID:           userID,
		StorageKey:       "published/" + userID + "/" + uploadID + ".png",
		FileName:         uploadID + ".png",
		DeclaredFileSize: size,
		DeclaredMimeType: "image/png",
		Category:         CategoryMomentImage,
		Completed:        true,
		File:             UploadedMedia{FileID: "file_" + uploadID, FileSize: size, MimeType: "image/png"},
	}
	if err := repo.Save(context.Background(), upload); err != nil {
		t.Fatalf("seed completed media %s: %v", uploadID, err)
	}
}

// seedPendingChatFile 写入一条待完成会话（非图片类别：complete 走 CopyObject，
// 不触发魔数/EXIF 链路，便于聚焦配额判定）。
func seedPendingChatFile(t *testing.T, repo Repository, userID, uploadID string, size int) string {
	t.Helper()
	storageKey := "uploads/" + userID + "/" + uploadID + ".txt"
	if err := repo.Save(context.Background(), pendingUpload{
		UploadID:         uploadID,
		UserID:           userID,
		StorageKey:       storageKey,
		FileName:         "a.txt",
		DeclaredFileSize: size,
		DeclaredMimeType: testChatFileMime,
		Category:         CategoryChatFile,
	}); err != nil {
		t.Fatalf("seed pending chat file %s: %v", uploadID, err)
	}
	return storageKey
}

func newStorageQuotaService(repo Repository, objectSize int) (*Service, *mediaStubProvider) {
	provider := &mediaStubProvider{headResp: ObjectMetadata{FileSize: objectSize, MimeType: testChatFileMime}}
	return NewServiceWithRepoAndProvider(repo, provider), provider
}

// storageQuotaErrorRepo 注入 CompletedStorageBytes 错误：查询失败必须上抛
// （10600），不能把「用量未知」当未超限放行，否则 DB 故障期间存储上界失效。
type storageQuotaErrorRepo struct {
	*InMemoryRepository
	err error
}

func (r *storageQuotaErrorRepo) CompletedStorageBytes(context.Context, string) (int64, error) {
	return 0, r.err
}

// 边界：恰好到上限放行；超 1 字节拒绝且不产生任何发布副作用。
func TestCompleteMediaStorageQuotaBoundary(t *testing.T) {
	ctx := mediaContextWithUserAndKey("u1", "")

	t.Run("exactly at limit passes", func(t *testing.T) {
		repo := NewInMemoryRepository()
		const size = 1024
		seedCompletedMedia(t, repo, "u1", "upload_done", int(completedMediaStorageLimitBytes)-size)
		seedPendingChatFile(t, repo, "u1", "upload_pending", size)
		svc, provider := newStorageQuotaService(repo, size)
		file, err := svc.Complete(ctx, "upload_pending")
		if err != nil {
			t.Fatalf("exactly at limit must pass, got %v", err)
		}
		if file.FileSize != size {
			t.Fatalf("file size = %d, want %d", file.FileSize, size)
		}
		used, err := repo.CompletedStorageBytes(context.Background(), "u1")
		if err != nil {
			t.Fatalf("completed storage bytes: %v", err)
		}
		if used != completedMediaStorageLimitBytes {
			t.Fatalf("used = %d, want %d", used, completedMediaStorageLimitBytes)
		}
		if len(provider.copies) != 1 {
			t.Fatalf("copies = %v, want 1 publish copy", provider.copies)
		}
	})

	t.Run("one byte over limit rejected without publish side effects", func(t *testing.T) {
		repo := NewInMemoryRepository()
		const size = 1024
		seedCompletedMedia(t, repo, "u1", "upload_done", int(completedMediaStorageLimitBytes)-size+1)
		storageKey := seedPendingChatFile(t, repo, "u1", "upload_pending", size)
		svc, provider := newStorageQuotaService(repo, size)
		if _, err := svc.Complete(ctx, "upload_pending"); !errors.Is(err, ErrStorageQuotaExceeded) {
			t.Fatalf("expected ErrStorageQuotaExceeded, got %v", err)
		}
		// 拒绝零副作用：不拷贝/不删除、会话行保持 pending（不释放——上限是
		// 账号级状态，释放单个会话也不会腾出空间，留给保留期 GC）。
		if len(provider.copies) != 0 || len(provider.deletedKeys) != 0 {
			t.Fatalf("quota rejection must not publish or delete: copies=%v deleted=%v", provider.copies, provider.deletedKeys)
		}
		upload, found, err := repo.Get(context.Background(), "upload_pending")
		if err != nil || !found {
			t.Fatalf("pending session must be kept: found=%v err=%v", found, err)
		}
		if upload.Completed {
			t.Fatal("pending session must not be completed on quota rejection")
		}
		if upload.StorageKey != storageKey {
			t.Fatalf("storage key changed on rejection: %q, want %q", upload.StorageKey, storageKey)
		}
	})
}

// 口径：只统计该用户已完成行（final_file_size）；未完成会话与其他用户的媒体
// 不占配额（否则 pending 声明量或他人数据会误锁上传）。
func TestCompleteMediaStorageQuotaCountsOnlyOwnCompletedMedia(t *testing.T) {
	const size = 1024

	t.Run("other users' completed media is not counted", func(t *testing.T) {
		repo := NewInMemoryRepository()
		// u1 已满、u2 已有 1GiB：若聚合误把全表算进来，u2 会被 3GiB 拖挂。
		seedCompletedMedia(t, repo, "u1", "upload_u1_done", int(completedMediaStorageLimitBytes))
		seedCompletedMedia(t, repo, "u2", "upload_u2_done", 1<<30)
		seedPendingChatFile(t, repo, "u2", "upload_u2_pending", size)
		svc, _ := newStorageQuotaService(repo, size)
		if _, err := svc.Complete(mediaContextWithUserAndKey("u2", ""), "upload_u2_pending"); err != nil {
			t.Fatalf("other user's completed media must not consume quota, got %v", err)
		}
	})

	t.Run("pending declared bytes are not counted", func(t *testing.T) {
		repo := NewInMemoryRepository()
		// u1 距上限只差本次文件；另有一条声明 10GiB 的未完成会话——未完成会话
		// 只受 pending 配额约束，不得计入完成态配额。
		seedCompletedMedia(t, repo, "u1", "upload_u1_done", int(completedMediaStorageLimitBytes)-size)
		if err := repo.Save(context.Background(), pendingUpload{
			UploadID: "upload_u1_big_pending", UserID: "u1", StorageKey: "uploads/u1/upload_u1_big_pending.bin",
			FileName: "big.bin", DeclaredFileSize: 10 << 30, DeclaredMimeType: "application/octet-stream",
			Category: CategoryChatFile, CreatedAt: "2026-03-30T00:00:00Z",
		}); err != nil {
			t.Fatalf("seed big pending session: %v", err)
		}
		seedPendingChatFile(t, repo, "u1", "upload_u1_pending", size)
		svc, _ := newStorageQuotaService(repo, size)
		if _, err := svc.Complete(mediaContextWithUserAndKey("u1", ""), "upload_u1_pending"); err != nil {
			t.Fatalf("pending declared bytes must not consume completed quota, got %v", err)
		}
	})

	t.Run("own full account is rejected", func(t *testing.T) {
		repo := NewInMemoryRepository()
		seedCompletedMedia(t, repo, "u1", "upload_u1_done", int(completedMediaStorageLimitBytes))
		seedPendingChatFile(t, repo, "u1", "upload_u1_pending", size)
		svc, _ := newStorageQuotaService(repo, size)
		if _, err := svc.Complete(mediaContextWithUserAndKey("u1", ""), "upload_u1_pending"); !errors.Is(err, ErrStorageQuotaExceeded) {
			t.Fatalf("expected ErrStorageQuotaExceeded, got %v", err)
		}
	})
}

// 已完成的会话重复 complete 是自然幂等（不重新发布/不校验配额）：账号已满时
// 重复调用仍返回成功，客户端「complete 成功但响应丢失后重试」不被配额误伤。
func TestRepeatCompleteAfterQuotaFullRemainsIdempotent(t *testing.T) {
	ctx := mediaContextWithUserAndKey("u1", "")
	repo := NewInMemoryRepository()
	const size = 1024
	seedPendingChatFile(t, repo, "u1", "upload_once", size)
	svc, provider := newStorageQuotaService(repo, size)
	first, err := svc.Complete(ctx, "upload_once")
	if err != nil {
		t.Fatalf("first complete: %v", err)
	}
	// 把账号补满到上限，再重复 complete。
	seedCompletedMedia(t, repo, "u1", "upload_filler", int(completedMediaStorageLimitBytes))
	second, err := svc.Complete(ctx, "upload_once")
	if err != nil {
		t.Fatalf("idempotent repeat complete must not hit quota, got %v", err)
	}
	if first.FileID != second.FileID {
		t.Fatalf("idempotent file_id = %q, want %q", second.FileID, first.FileID)
	}
	if len(provider.copies) != 1 {
		t.Fatalf("repeat complete must not republish, copies=%v", provider.copies)
	}
}

// 用量查询失败必须上抛为内部错误（10600 路径），不得当成「未超限」放行。
func TestCompleteMediaStorageQuotaQueryFailureSurfaces(t *testing.T) {
	queryErr := errors.New("storage usage query boom")
	base := NewInMemoryRepository()
	seedPendingChatFile(t, base, "u1", "upload_pending", 1024)
	repo := &storageQuotaErrorRepo{InMemoryRepository: base, err: queryErr}
	svc, _ := newStorageQuotaService(repo, 1024)
	_, err := svc.Complete(mediaContextWithUserAndKey("u1", ""), "upload_pending")
	if !errors.Is(err, queryErr) {
		t.Fatalf("expected query error to surface, got %v", err)
	}
	if errors.Is(err, ErrStorageQuotaExceeded) {
		t.Fatal("query failure must not be reported as quota conflict")
	}
}

// handler 映射：配额超限返回 spec 10400 业务冲突（不是 10600 内部错误）。
func TestCompleteMediaStorageQuotaHandlerMapsConflict(t *testing.T) {
	repo := NewInMemoryRepository()
	seedCompletedMedia(t, repo, "u1", "upload_done", int(completedMediaStorageLimitBytes))
	seedPendingChatFile(t, repo, "u1", "upload_pending", 1024)
	svc, _ := newStorageQuotaService(repo, 1024)
	handler := NewHandler(svc)
	rr := httptest.NewRecorder()
	handler.CompleteUpload(rr, mediaAuthedCompleteRequest("upload_pending", ""))
	if code := testdb.DecodeAPICode(t, rr); code != httpx.CodeConflict {
		t.Fatalf("expected code %d (10400 conflict), got %d: %s", httpx.CodeConflict, code, rr.Body.String())
	}
	if rr.Code != http.StatusOK {
		t.Fatalf("business errors must keep HTTP 200, got %d", rr.Code)
	}
}

// 阈值锁定：常量是账号级存储上界的唯一口径，防后续调参悄悄改变语义。
func TestCompletedMediaStorageLimitValue(t *testing.T) {
	if completedMediaStorageLimitBytes != 2<<30 {
		t.Fatalf("completedMediaStorageLimitBytes = %d, want 2GiB (2<<30)", completedMediaStorageLimitBytes)
	}
}

// 用量查询失败走 10600（服务器错误）而不是 10400：客户端可重试，不把内部
// 故障误报成「存储空间已满」。
func TestCompleteMediaStorageQuotaQueryFailureHandlerMapsInternal(t *testing.T) {
	base := NewInMemoryRepository()
	seedPendingChatFile(t, base, "u1", "upload_pending", 1024)
	repo := &storageQuotaErrorRepo{InMemoryRepository: base, err: errors.New("storage usage query boom")}
	svc, _ := newStorageQuotaService(repo, 1024)
	handler := NewHandler(svc)
	rr := httptest.NewRecorder()
	handler.CompleteUpload(rr, mediaAuthedCompleteRequest("upload_pending", ""))
	if code := testdb.DecodeAPICode(t, rr); code != httpx.CodeInternalUnavailable {
		t.Fatalf("expected code %d (10600 internal), got %d: %s", httpx.CodeInternalUnavailable, code, rr.Body.String())
	}
}
