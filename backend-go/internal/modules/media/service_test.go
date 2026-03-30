package media

import (
	"context"
	"errors"
	"testing"

	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/idempotency"
)

func mediaContextWithUserAndKey(userID, key string) context.Context {
	ctx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: userID})
	if key != "" {
		ctx = idempotency.WithKey(ctx, key)
	}
	return ctx
}

func TestValidatePresignUploadRequest(t *testing.T) {
	valid := PresignUploadRequest{FileName: "a.png", FileSize: 100, MimeType: "image/png", Category: CategoryAvatar, MD5: "d41d8cd98f00b204e9800998ecf8427e"}
	if err := validatePresignUploadRequest(valid); err != nil {
		t.Fatalf("expected valid request, got %v", err)
	}
	cases := []PresignUploadRequest{
		{FileName: "", FileSize: 100, MimeType: "image/png", Category: CategoryAvatar},
		{FileName: "a.png", FileSize: 0, MimeType: "image/png", Category: CategoryAvatar},
		{FileName: "a.png", FileSize: 100, MimeType: "video/mp4", Category: CategoryAvatar},
		{FileName: "a.png", FileSize: 6 * 1024 * 1024, MimeType: "image/png", Category: CategoryAvatar},
		{FileName: "a.png", FileSize: 100, MimeType: "image/png", Category: CategoryAvatar, MD5: "bad-md5"},
	}
	for _, c := range cases {
		if err := validatePresignUploadRequest(c); !errors.Is(err, ErrInvalidMediaRequest) {
			t.Fatalf("expected invalid request for %#v, got %v", c, err)
		}
	}
}

func TestServicePresignWritesPendingSession(t *testing.T) {
	repo := NewInMemoryRepository()
	provider := &mediaStubProvider{presignResp: PresignedUpload{UploadID: "upload_1", UploadURL: "https://storage.example.com/hotspot/uploads/u1/upload_1.png", Headers: map[string]string{"Content-Type": "image/png"}, ExpiresAt: "2026-03-30T00:00:00Z", StorageKey: "uploads/u1/upload_1.png"}}
	service := NewServiceWithRepoAndProvider(repo, provider)

	resp, err := service.Presign(mediaContextWithUserAndKey("u1", ""), PresignUploadRequest{FileName: "a.png", FileSize: 100, MimeType: "image/png", Category: CategoryAvatar})
	if err != nil {
		t.Fatalf("Presign returned error: %v", err)
	}
	if resp.UploadID != "upload_1" || resp.StorageKey != "" {
		t.Fatalf("unexpected presign response: %#v", resp)
	}
	upload, ok, err := repo.Get(context.Background(), "upload_1")
	if err != nil || !ok {
		t.Fatalf("expected persisted upload, ok=%v err=%v", ok, err)
	}
	if upload.UserID != "u1" || upload.StorageKey != "uploads/u1/upload_1.png" {
		t.Fatalf("unexpected stored upload: %#v", upload)
	}
}

func TestServiceBatchPresignRejectsAnyInvalidItem(t *testing.T) {
	service := NewServiceWithRepoAndProvider(NewInMemoryRepository(), &mediaStubProvider{})
	_, err := service.BatchPresign(mediaContextWithUserAndKey("u1", ""), BatchPresignUploadRequest{Files: []PresignUploadRequest{{FileName: "a.png", FileSize: 100, MimeType: "image/png", Category: CategoryAvatar}, {FileName: "b.png", FileSize: 100, MimeType: "video/mp4", Category: CategoryAvatar}}})
	if !errors.Is(err, ErrInvalidMediaRequest) {
		t.Fatalf("expected invalid request, got %v", err)
	}
}

func TestServiceCompleteSemantics(t *testing.T) {
	provider := &mediaStubProvider{headResp: ObjectMetadata{FileSize: 123, MimeType: "image/png"}}
	repo := NewInMemoryRepository()
	service := NewServiceWithRepoAndProvider(repo, provider)

	if _, err := service.Complete(mediaContextWithUserAndKey("u1", "key1"), "missing"); !errors.Is(err, ErrUploadNotFound) {
		t.Fatalf("expected ErrUploadNotFound, got %v", err)
	}

	_ = repo.Save(context.Background(), pendingUpload{UploadID: "upload_1", UserID: "u1", StorageKey: "uploads/u1/upload_1.png", FileName: "a.png", DeclaredFileSize: 100, DeclaredMimeType: "image/png", Category: CategoryAvatar})
	if _, err := service.Complete(mediaContextWithUserAndKey("u2", "key1"), "upload_1"); !errors.Is(err, ErrUploadNotFound) {
		t.Fatalf("expected ErrUploadNotFound for foreign user, got %v", err)
	}
	provider.headErr = ErrUploadNotReady
	if _, err := service.Complete(mediaContextWithUserAndKey("u1", "key1"), "upload_1"); !errors.Is(err, ErrUploadNotReady) {
		t.Fatalf("expected ErrUploadNotReady, got %v", err)
	}
	provider.headErr = nil

	file, err := service.Complete(mediaContextWithUserAndKey("u1", "key1"), "upload_1")
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if file.FileID == "" || file.FileSize != 123 || file.MimeType != "image/png" {
		t.Fatalf("unexpected file: %#v", file)
	}

	replayed, err := service.Complete(mediaContextWithUserAndKey("u1", "key1"), "upload_1")
	if err != nil || replayed.FileID != file.FileID {
		t.Fatalf("expected stable replay, got file=%#v err=%v", replayed, err)
	}

	if _, err := service.Complete(mediaContextWithUserAndKey("u1", "key2"), "upload_1"); !errors.Is(err, ErrUploadConflict) {
		t.Fatalf("expected ErrUploadConflict, got %v", err)
	}
}

func TestServicePropagatesProviderAndRepositoryErrors(t *testing.T) {
	service := NewServiceWithRepoAndProvider(&mediaErrorRepo{saveErr: errors.New("save boom")}, &mediaStubProvider{presignResp: PresignedUpload{UploadID: "upload_1", UploadURL: "https://storage.example.com/upload_1", Headers: map[string]string{"Content-Type": "image/png"}, ExpiresAt: "2026-03-30T00:00:00Z", StorageKey: "uploads/u1/upload_1.png"}})
	if _, err := service.Presign(mediaContextWithUserAndKey("u1", ""), PresignUploadRequest{FileName: "a.png", FileSize: 100, MimeType: "image/png", Category: CategoryAvatar}); err == nil {
		t.Fatal("expected save error")
	}

	repo := NewInMemoryRepository()
	_ = repo.Save(context.Background(), pendingUpload{UploadID: "upload_2", UserID: "u1", StorageKey: "uploads/u1/upload_2.png", FileName: "a.png", DeclaredFileSize: 100, DeclaredMimeType: "image/png", Category: CategoryAvatar})
	if _, err := NewServiceWithRepoAndProvider(repo, &mediaStubProvider{headErr: errors.New("head boom")}).Complete(mediaContextWithUserAndKey("u1", "key1"), "upload_2"); err == nil {
		t.Fatal("expected head error")
	}

	if _, err := NewServiceWithRepoAndProvider(&mediaErrorRepo{getErr: errors.New("get boom")}, &mediaStubProvider{}).Complete(mediaContextWithUserAndKey("u1", "key1"), "upload_2"); err == nil {
		t.Fatal("expected get error")
	}
}
