package bootstrap

import (
	"context"
	"testing"
	"time"

	"hotspot/backend-go/internal/modules/media"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/idempotency"
)

// stubMediaProvider 满足 media.Provider 的最小桩（Presign 固定返回可预算的
// upload 会话；Head 返回与声明一致的元数据，让 complete 校验通过）。
type stubMediaProvider struct {
	head media.ObjectMetadata
}

func (p *stubMediaProvider) Presign(_ context.Context, req media.PresignFileInput, owner string) (media.PresignedUpload, error) {
	id := "upload_report"
	return media.PresignedUpload{
		UploadID:   id,
		UploadURL:  "https://storage.example.com/" + id,
		Headers:    map[string]string{"Content-Type": req.MimeType},
		ExpiresAt:  time.Now().Add(time.Hour).Format(time.RFC3339),
		StorageKey: "uploads/" + owner + "/" + id + ".bin",
	}, nil
}

func (p *stubMediaProvider) PresignGet(context.Context, string, time.Duration) (string, error) {
	return "", nil
}

func (p *stubMediaProvider) Head(context.Context, string) (media.ObjectMetadata, error) {
	return p.head, nil
}

func (p *stubMediaProvider) ReadPrefix(context.Context, string, int64) ([]byte, error) {
	return nil, nil
}

func (p *stubMediaProvider) Delete(context.Context, string) error { return nil }

func (p *stubMediaProvider) GetObject(context.Context, string) ([]byte, error) { return nil, nil }

func (p *stubMediaProvider) PutObject(context.Context, string, string, []byte) error { return nil }

// 走真实 Presign → Complete 流程构造完成态上传（chat_file 类别：无魔数嗅探/EXIF 清洗）。
// 返回服务、完成态 FileMeta 与 uploadID。注意：内存仓 CompletedFileMetaByID
// 实际按 uploadID 键查找（与 Postgres 的 file_id 查找口径不一致，见
// repo_memory.go Get），因此查找键必须用 uploadID。
func seedCompletedUpload(t *testing.T, ctx context.Context) (*media.Service, media.FileMeta, string) {
	t.Helper()
	repo := media.NewInMemoryRepository()
	svc := media.NewServiceWithRepoAndProvider(repo, &stubMediaProvider{
		head: media.ObjectMetadata{FileSize: 2048, MimeType: "application/pdf"},
	})
	ctx = platformauth.WithPrincipal(ctx, platformauth.Principal{UserID: "u1"})
	ctx = idempotency.WithKey(ctx, "idem_report")
	resp, err := svc.Presign(ctx, media.PresignFileInput{
		FileName: "evidence.pdf", FileSize: 2048, MimeType: "application/pdf", Category: media.CategoryChatFile,
	})
	if err != nil {
		t.Fatalf("Presign: %v", err)
	}
	if _, err := svc.Complete(ctx, resp.UploadID); err != nil {
		t.Fatalf("Complete: %v", err)
	}
	fm, found, err := repo.CompletedFileMetaByID(ctx, resp.UploadID)
	if err != nil || !found {
		t.Fatalf("expected completed file meta, found=%v err=%v", found, err)
	}
	return svc, fm, resp.UploadID
}

func TestReportsMediaSourceMapsAllFields(t *testing.T) {
	ctx := context.Background()
	svc, fm, fileID := seedCompletedUpload(t, ctx)
	src := reportsMediaSource{svc: svc}

	got, found, err := src.CompletedFileMetaByID(ctx, fileID)
	if err != nil || !found {
		t.Fatalf("expected found=true err=nil, got found=%v err=%v", found, err)
	}
	// FileMeta.Category 不在 reports.MediaFileMeta 中（reports 用不到类别），
	// 其余字段逐一比对：漏拷或错位都会在这里失败（举报证据不完整）。
	if got.StorageKey != fm.StorageKey {
		t.Fatalf("StorageKey mismatch: got %q want %q", got.StorageKey, fm.StorageKey)
	}
	if got.ThumbnailURL != fm.ThumbnailURL {
		t.Fatalf("ThumbnailURL mismatch: got %q want %q", got.ThumbnailURL, fm.ThumbnailURL)
	}
	if got.Width != fm.Width || got.Height != fm.Height {
		t.Fatalf("dimensions mismatch: got %dx%d want %dx%d", got.Width, got.Height, fm.Width, fm.Height)
	}
	if got.MimeType != fm.MimeType {
		t.Fatalf("MimeType mismatch: got %q want %q", got.MimeType, fm.MimeType)
	}
	if got.FileName != fm.FileName {
		t.Fatalf("FileName mismatch: got %q want %q", got.FileName, fm.FileName)
	}
	if got.FileSize != fm.FileSize {
		t.Fatalf("FileSize mismatch: got %d want %d", got.FileSize, fm.FileSize)
	}
	if got.Duration != fm.Duration {
		t.Fatalf("Duration mismatch: got %d want %d", got.Duration, fm.Duration)
	}
}

func TestReportsMediaSourceUnknownFile(t *testing.T) {
	ctx := context.Background()
	repo := media.NewInMemoryRepository()
	src := reportsMediaSource{svc: media.NewServiceWithRepo(repo)}

	_, found, err := src.CompletedFileMetaByID(ctx, "upload_missing")
	if err != nil || found {
		t.Fatalf("expected found=false err=nil for unknown file, got found=%v err=%v", found, err)
	}
}

// presign 后未 complete 的上传不暴露给举报快照——只认完成态。
func TestReportsMediaSourceIgnoresPendingUpload(t *testing.T) {
	ctx := context.Background()
	repo := media.NewInMemoryRepository()
	svc := media.NewServiceWithRepoAndProvider(repo, &stubMediaProvider{})
	ictx := platformauth.WithPrincipal(ctx, platformauth.Principal{UserID: "u1"})
	if _, err := svc.Presign(ictx, media.PresignFileInput{
		FileName: "pending.pdf", FileSize: 100, MimeType: "application/pdf", Category: media.CategoryChatFile,
	}); err != nil {
		t.Fatalf("Presign: %v", err)
	}
	src := reportsMediaSource{svc: svc}

	if _, found, err := src.CompletedFileMetaByID(ctx, "upload_report"); err != nil || found {
		t.Fatalf("pending upload must not be visible, got found=%v err=%v", found, err)
	}
}
