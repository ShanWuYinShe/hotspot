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
	valid := PresignFileInput{FileName: "a.png", FileSize: 100, MimeType: "image/png", Category: CategoryAvatar, MD5: "d41d8cd98f00b204e9800998ecf8427e"}
	if err := validatePresignUploadRequest(valid); err != nil {
		t.Fatalf("expected valid request, got %v", err)
	}
	cases := []PresignFileInput{
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

	resp, err := service.Presign(mediaContextWithUserAndKey("u1", ""), PresignFileInput{FileName: "a.png", FileSize: 100, MimeType: "image/png", Category: CategoryAvatar})
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
	_, err := service.BatchPresign(mediaContextWithUserAndKey("u1", ""), []PresignFileInput{{FileName: "a.png", FileSize: 100, MimeType: "image/png", Category: CategoryAvatar}, {FileName: "b.png", FileSize: 100, MimeType: "video/mp4", Category: CategoryAvatar}})
	if !errors.Is(err, ErrInvalidMediaRequest) {
		t.Fatalf("expected invalid request, got %v", err)
	}
}

// magicPNG PNG 文件头魔数，供 Complete 成功路径 stub 复用。
var magicPNG = []byte("\x89PNG\r\n\x1a\n")

func TestServiceCompleteSemantics(t *testing.T) {
	provider := &mediaStubProvider{headResp: ObjectMetadata{FileSize: 100, MimeType: "image/png"}, prefix: magicPNG}
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
	if file.FileID == "" || file.FileSize != 100 || file.MimeType != "image/png" {
		t.Fatalf("unexpected file: %#v", file)
	}

	replayed, err := service.Complete(mediaContextWithUserAndKey("u1", "key1"), "upload_1")
	if err != nil || replayed.FileID != file.FileID {
		t.Fatalf("expected stable replay, got file=%#v err=%v", replayed, err)
	}

	// 自然幂等（spec api_spec_media.md）：同一 upload_id 重复 complete 返回相同
	// file_id（等价成功），不依赖 Idempotency-Key——不同 key 也视为等价成功。
	replayed2, err := service.Complete(mediaContextWithUserAndKey("u1", "key2"), "upload_1")
	if err != nil || replayed2.FileID != file.FileID {
		t.Fatalf("expected natural-idempotent replay regardless of key, got file=%#v err=%v", replayed2, err)
	}
}

func TestServiceCompleteRejectsMetadataMismatch(t *testing.T) {
	t.Run("rejects mismatched content type", func(t *testing.T) {
		repo := NewInMemoryRepository()
		_ = repo.Save(context.Background(), pendingUpload{UploadID: "upload_mime", UserID: "u1", StorageKey: "uploads/u1/upload_mime.png", FileName: "a.png", DeclaredFileSize: 100, DeclaredMimeType: "image/png", Category: CategoryAvatar})
		service := NewServiceWithRepoAndProvider(repo, &mediaStubProvider{headResp: ObjectMetadata{FileSize: 100, MimeType: "image/jpeg"}})
		if _, err := service.Complete(mediaContextWithUserAndKey("u1", "key1"), "upload_mime"); !errors.Is(err, ErrUploadConflict) {
			t.Fatalf("expected ErrUploadConflict, got %v", err)
		}
	})

	t.Run("rejects category incompatible content type", func(t *testing.T) {
		repo := NewInMemoryRepository()
		_ = repo.Save(context.Background(), pendingUpload{UploadID: "upload_category", UserID: "u1", StorageKey: "uploads/u1/upload_category.png", FileName: "a.png", DeclaredFileSize: 100, DeclaredMimeType: "image/png", Category: CategoryAvatar})
		service := NewServiceWithRepoAndProvider(repo, &mediaStubProvider{headResp: ObjectMetadata{FileSize: 100, MimeType: "video/mp4"}})
		if _, err := service.Complete(mediaContextWithUserAndKey("u1", "key1"), "upload_category"); !errors.Is(err, ErrUploadConflict) {
			t.Fatalf("expected ErrUploadConflict, got %v", err)
		}
	})

	t.Run("rejects mismatched content length", func(t *testing.T) {
		repo := NewInMemoryRepository()
		_ = repo.Save(context.Background(), pendingUpload{UploadID: "upload_size", UserID: "u1", StorageKey: "uploads/u1/upload_size.png", FileName: "a.png", DeclaredFileSize: 100, DeclaredMimeType: "image/png", Category: CategoryAvatar})
		service := NewServiceWithRepoAndProvider(repo, &mediaStubProvider{headResp: ObjectMetadata{FileSize: 101, MimeType: "image/png"}})
		if _, err := service.Complete(mediaContextWithUserAndKey("u1", "key1"), "upload_size"); !errors.Is(err, ErrUploadConflict) {
			t.Fatalf("expected ErrUploadConflict, got %v", err)
		}
	})
}

// 魔数嗅探（P2-1）：complete 阶段图片类别必须通过魔数校验，防止以声明的
// Content-Type 伪装非图片内容（预签名直传的 Content-Type 由客户端控制）。
func TestServiceCompleteRejectsMagicMismatch(t *testing.T) {
	newAvatarUpload := func(t *testing.T, uploadID string) *Service {
		t.Helper()
		repo := NewInMemoryRepository()
		_ = repo.Save(context.Background(), pendingUpload{UploadID: uploadID, UserID: "u1", StorageKey: "uploads/u1/" + uploadID + ".png", FileName: "a.png", DeclaredFileSize: 100, DeclaredMimeType: "image/png", Category: CategoryAvatar})
		return NewServiceWithRepoAndProvider(repo, &mediaStubProvider{headResp: ObjectMetadata{FileSize: 100, MimeType: "image/png"}})
	}

	t.Run("rejects html content declared as png", func(t *testing.T) {
		service := newAvatarUpload(t, "upload_html")
		service.provider.(*mediaStubProvider).prefix = []byte("<html><body>evil</body></html>")
		if _, err := service.Complete(mediaContextWithUserAndKey("u1", "key1"), "upload_html"); !errors.Is(err, ErrUploadConflict) {
			t.Fatalf("expected ErrUploadConflict, got %v", err)
		}
	})

	t.Run("rejects empty object body", func(t *testing.T) {
		service := newAvatarUpload(t, "upload_empty")
		if _, err := service.Complete(mediaContextWithUserAndKey("u1", "key1"), "upload_empty"); !errors.Is(err, ErrUploadConflict) {
			t.Fatalf("expected ErrUploadConflict, got %v", err)
		}
	})

	t.Run("propagates read prefix failure", func(t *testing.T) {
		service := newAvatarUpload(t, "upload_readerr")
		service.provider.(*mediaStubProvider).prefixErr = errors.New("read boom")
		if _, err := service.Complete(mediaContextWithUserAndKey("u1", "key1"), "upload_readerr"); err == nil || errors.Is(err, ErrUploadConflict) {
			t.Fatalf("expected propagated read error, got %v", err)
		}
	})

	t.Run("accepts real png magic", func(t *testing.T) {
		service := newAvatarUpload(t, "upload_png")
		service.provider.(*mediaStubProvider).prefix = magicPNG
		if _, err := service.Complete(mediaContextWithUserAndKey("u1", "key1"), "upload_png"); err != nil {
			t.Fatalf("expected success, got %v", err)
		}
	})

	t.Run("skips non image categories", func(t *testing.T) {
		repo := NewInMemoryRepository()
		_ = repo.Save(context.Background(), pendingUpload{UploadID: "upload_file", UserID: "u1", StorageKey: "uploads/u1/upload_file.bin", FileName: "a.bin", DeclaredFileSize: 100, DeclaredMimeType: "application/pdf", Category: CategoryChatFile})
		service := NewServiceWithRepoAndProvider(repo, &mediaStubProvider{headResp: ObjectMetadata{FileSize: 100, MimeType: "application/pdf"}})
		if _, err := service.Complete(mediaContextWithUserAndKey("u1", "key1"), "upload_file"); err != nil {
			t.Fatalf("expected non-image category to skip magic check, got %v", err)
		}
	})
}

func TestMagicMatchesImageType(t *testing.T) {
	cases := []struct {
		name     string
		prefix   []byte
		mimeType string
		want     bool
	}{
		{"jpeg", []byte{0xFF, 0xD8, 0xFF, 0xE0}, "image/jpeg", true},
		{"jpeg truncated", []byte{0xFF, 0xD8}, "image/jpeg", false},
		{"png", magicPNG, "image/png", true},
		{"png broken", []byte("\x89PNG\x00\x00"), "image/png", false},
		{"gif87a", []byte("GIF87a rest"), "image/gif", true},
		{"gif89a", []byte("GIF89a rest"), "image/gif", true},
		{"gif wrong", []byte("GIF86a rest"), "image/gif", false},
		{"webp", []byte("RIFF\xf4\x01\x00\x00WEBPVP8 "), "image/webp", true},
		{"webp riff only", []byte("RIFF\x00\x00\x00\x00JPEG"), "image/webp", false},
		{"html as png", []byte("<html>"), "image/png", false},
		{"non image passthrough", []byte("anything"), "application/pdf", true},
	}
	for _, tc := range cases {
		if got := magicMatchesImageType(tc.prefix, tc.mimeType); got != tc.want {
			t.Fatalf("%s: magicMatchesImageType=%v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestServicePropagatesProviderAndRepositoryErrors(t *testing.T) {
	service := NewServiceWithRepoAndProvider(&mediaErrorRepo{saveErr: errors.New("save boom")}, &mediaStubProvider{presignResp: PresignedUpload{UploadID: "upload_1", UploadURL: "https://storage.example.com/upload_1", Headers: map[string]string{"Content-Type": "image/png"}, ExpiresAt: "2026-03-30T00:00:00Z", StorageKey: "uploads/u1/upload_1.png"}})
	if _, err := service.Presign(mediaContextWithUserAndKey("u1", ""), PresignFileInput{FileName: "a.png", FileSize: 100, MimeType: "image/png", Category: CategoryAvatar}); err == nil {
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
