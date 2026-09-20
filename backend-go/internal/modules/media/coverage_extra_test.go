package media

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/appconfig"
	"hotspot/backend-go/internal/platform/config"
)

// ── provider 对象操作（保留期 GC / EXIF 清洗链路依赖）─────────────────────

func TestObjectStorageProviderDeleteIsIdempotent(t *testing.T) {
	var method string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		if r.URL.Path == "/hotspot/gone.png" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	provider := NewObjectStorageProvider(config.Config{
		S3Endpoint:       server.URL,
		S3Bucket:         "hotspot",
		S3ForcePathStyle: true,
	})
	if err := provider.Delete(context.Background(), "uploads/u1/a.png"); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
	if method != http.MethodDelete {
		t.Fatalf("expected DELETE request, got %s", method)
	}
	// 404 视为已删（幂等）。
	if err := provider.Delete(context.Background(), "gone.png"); err != nil {
		t.Fatalf("Delete on missing object must be idempotent, got %v", err)
	}
}

func TestObjectStorageProviderGetObjectAndPutObject(t *testing.T) {
	var putBody []byte
	var putContentType string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			if r.URL.Path == "/hotspot/missing.png" {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("object-bytes"))
		case http.MethodPut:
			putContentType = r.Header.Get("Content-Type")
			buf := make([]byte, r.ContentLength)
			_, _ = r.Body.Read(buf)
			putBody = buf
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	defer server.Close()

	provider := NewObjectStorageProvider(config.Config{
		S3Endpoint:       server.URL,
		S3Bucket:         "hotspot",
		S3ForcePathStyle: true,
	})
	data, err := provider.GetObject(context.Background(), "uploads/u1/a.png")
	if err != nil || string(data) != "object-bytes" {
		t.Fatalf("GetObject: %v %q", err, data)
	}
	if _, err := provider.GetObject(context.Background(), "missing.png"); !errors.Is(err, ErrUploadNotReady) {
		t.Fatalf("expected ErrUploadNotReady on 404, got %v", err)
	}
	if err := provider.PutObject(context.Background(), "uploads/u1/a.png", "image/png", []byte("cleaned")); err != nil {
		t.Fatalf("PutObject: %v", err)
	}
	if string(putBody) != "cleaned" || putContentType != "image/png" {
		t.Fatalf("unexpected put: body=%q content-type=%q", putBody, putContentType)
	}
}

// ── 保留期清理 sweeper ────────────────────────────────────────────────

func TestMediaRetentionSweeperDeletesStaleUploads(t *testing.T) {
	repo := NewInMemoryRepository()
	provider := &mediaStubProvider{}
	svc := NewServiceWithRepoAndProvider(repo, provider)

	created := time.Now().UTC().AddDate(0, 0, -(staleUploadRetentionDays + 1)).Format(time.RFC3339)
	if err := repo.Save(context.Background(), pendingUpload{UploadID: "upload_stale", UserID: "u1", StorageKey: "uploads/u1/stale.png", FileName: "a.png", DeclaredFileSize: 100, DeclaredMimeType: "image/png", Category: CategoryAvatar, CreatedAt: created}); err != nil {
		t.Fatalf("save: %v", err)
	}

	// 同步跑一轮（ListStaleUploads → provider.Delete → DeleteUploadRow）。
	svc.sweepStaleUploads(context.Background())

	stale, err := repo.ListStaleUploads(context.Background(), time.Now().UTC(), 100)
	if err != nil || len(stale) != 0 {
		t.Fatalf("expected stale rows cleaned, got %#v err=%v", stale, err)
	}
}

func TestMediaServiceStartRetentionSweeperIdempotentGuard(t *testing.T) {
	repo := NewInMemoryRepository()
	svc := NewServiceWithRepoAndProvider(repo, &mediaStubProvider{})

	sweepCtx, sweepCancel := context.WithCancel(context.Background())
	defer sweepCancel()
	svc.StartRetentionSweeper(sweepCtx)
	svc.StartRetentionSweeper(sweepCtx)
	if !retentionSweeperRunning() {
		t.Fatal("expected retention sweeper started flag to be set")
	}
	sweepCancel()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && retentionSweeperRunning() {
		time.Sleep(5 * time.Millisecond)
	}
	if retentionSweeperRunning() {
		t.Fatal("expected retention sweeper started flag to reset after ctx cancel")
	}
}

// ── 内存 repo 与 service 解析路径 ─────────────────────────────────────

func TestInMemoryRepositoryResolveAndMetaMethods(t *testing.T) {
	repo := NewInMemoryRepository()
	upload := pendingUpload{
		UploadID: "upload_resolve", UserID: "u1", StorageKey: "uploads/u1/r.png",
		FileName: "r.png", DeclaredFileSize: 100, DeclaredMimeType: "image/png",
		Category: CategoryMomentImage,
	}
	if err := repo.Save(context.Background(), upload); err != nil {
		t.Fatalf("save: %v", err)
	}
	completed := UploadedMedia{FileID: "file_resolve", FileSize: 100, MimeType: "image/png"}
	if _, found, err := repo.Complete(context.Background(), "upload_resolve", completed, ""); err != nil || !found {
		t.Fatalf("complete: found=%v err=%v", found, err)
	}

	assets, err := repo.ResolveByFileIDs(context.Background(), []string{"file_resolve", "file_missing"}, "u1", CategoryMomentImage)
	if err != nil {
		t.Fatalf("ResolveByFileIDs: %v", err)
	}
	if len(assets) != 1 {
		t.Fatalf("expected only owned completed file resolved, got %#v", assets)
	}
	if _, _, err := repo.CompletedFileMetaByID(context.Background(), "file_resolve"); err != nil {
		t.Fatalf("CompletedFileMetaByID: %v", err)
	}
	if _, found, _ := repo.CompletedFileMetaByID(context.Background(), "file_missing"); found {
		t.Fatal("expected missing file meta to be not found")
	}
}

func TestServiceResolveMomentImagesAndAvatar(t *testing.T) {
	repo := NewInMemoryRepository()
	provider := &mediaStubProvider{presignResp: PresignedUpload{UploadID: "upload_rm", StorageKey: "uploads/u1/rm.png"}}
	svc := NewServiceWithRepoAndProvider(repo, provider)

	if _, err := svc.ResolveMomentImages(context.Background(), nil, "u1"); err != nil {
		t.Fatalf("empty ids should be a no-op, got %v", err)
	}
	if err := repo.Save(context.Background(), pendingUpload{UploadID: "upload_rm", UserID: "u1", StorageKey: "uploads/u1/rm.png", FileName: "r.png", DeclaredFileSize: 100, DeclaredMimeType: "image/png", Category: CategoryMomentImage}); err != nil {
		t.Fatalf("save: %v", err)
	}
	if _, found, err := repo.Complete(context.Background(), "upload_rm", UploadedMedia{FileID: "file_rm", FileSize: 100, MimeType: "image/png"}, ""); err != nil || !found {
		t.Fatalf("complete: found=%v err=%v", found, err)
	}
	assets, err := svc.ResolveMomentImages(context.Background(), []string{"file_rm"}, "u1")
	if err != nil || len(assets) != 1 {
		t.Fatalf("ResolveMomentImages: %v %#v", err, assets)
	}
	// 非本人资源：整批拒绝（all-or-nothing，spec api_spec_media.md）。
	if _, err := svc.ResolveMomentImages(context.Background(), []string{"file_rm"}, "u2"); !errors.Is(err, ErrMediaNotOwned) {
		t.Fatalf("expected ErrMediaNotOwned for foreign viewer, got %v", err)
	}
	if err := svc.ResolveAvatarImage(context.Background(), "file_rm", "u2"); !errors.Is(err, ErrMediaNotOwned) {
		t.Fatalf("avatar category mismatch should be not-owned, got %v", err)
	}
	svc.SetAppConfig(appconfig.NewStaticSource(config.Config{}))
	if _, err := svc.PresignGet(context.Background(), "uploads/u1/rm.png", time.Minute); err != nil {
		t.Fatalf("PresignGet: %v", err)
	}
}
