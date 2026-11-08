package media

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/appconfig"
	"hotspot/backend-go/internal/platform/config"
)

// coverageProvider 是可逐方法注入返回值/错误的内存 Provider，用于覆盖服务层
// 与 provider 的错误分支（生产实现只能经 HTTP 触发这些分支）。
type coverageProvider struct {
	presignResp    PresignedUpload
	presignErr     error
	presignGetURL  string
	presignGetErr  error
	headResp       ObjectMetadata
	headErr        error
	readPrefix     []byte
	readPrefixErr  error
	deleteErr      error
	getObject      []byte
	getObjectErr   error
	putErr         error
	deletedKeys    []string
	putContentType string
	putData        []byte
}

func (p *coverageProvider) Presign(context.Context, PresignFileInput, string) (PresignedUpload, error) {
	return p.presignResp, p.presignErr
}

func (p *coverageProvider) PresignGet(context.Context, string, time.Duration) (string, error) {
	return p.presignGetURL, p.presignGetErr
}

func (p *coverageProvider) Head(context.Context, string) (ObjectMetadata, error) {
	return p.headResp, p.headErr
}

func (p *coverageProvider) ReadPrefix(context.Context, string, int64) ([]byte, error) {
	return p.readPrefix, p.readPrefixErr
}

func (p *coverageProvider) Delete(_ context.Context, storageKey string) error {
	p.deletedKeys = append(p.deletedKeys, storageKey)
	return p.deleteErr
}

func (p *coverageProvider) GetObject(context.Context, string) ([]byte, error) {
	return p.getObject, p.getObjectErr
}

func (p *coverageProvider) PutObject(_ context.Context, _ string, contentType string, data []byte) error {
	p.putContentType = contentType
	p.putData = append([]byte(nil), data...)
	return p.putErr
}

// mediaRepoOverride 在内存仓储基础上注入单方法错误/行为，覆盖服务层对仓储
// 错误的防御分支；未覆盖的方法仍走内存实现。
type mediaRepoOverride struct {
	*InMemoryRepository
	resolveErr   error
	metaErr      error
	completeErr  error
	completeNoop bool
	listErr      error
	deleteRowErr error
	getErr       error
}

func (r *mediaRepoOverride) Get(ctx context.Context, uploadID string) (pendingUpload, bool, error) {
	if r.getErr != nil {
		return pendingUpload{}, false, r.getErr
	}
	return r.InMemoryRepository.Get(ctx, uploadID)
}

func (r *mediaRepoOverride) Complete(ctx context.Context, uploadID string, file UploadedMedia, key string) (bool, error) {
	if r.completeErr != nil {
		return false, r.completeErr
	}
	if r.completeNoop {
		return false, nil
	}
	return r.InMemoryRepository.Complete(ctx, uploadID, file, key)
}

func (r *mediaRepoOverride) ResolveByFileIDs(ctx context.Context, fileIDs []string, userID string, category MediaCategory) (map[string]MediaAsset, error) {
	if r.resolveErr != nil {
		return nil, r.resolveErr
	}
	return r.InMemoryRepository.ResolveByFileIDs(ctx, fileIDs, userID, category)
}

func (r *mediaRepoOverride) CompletedFileMetaByID(ctx context.Context, fileID string) (FileMeta, bool, error) {
	if r.metaErr != nil {
		return FileMeta{}, false, r.metaErr
	}
	return r.InMemoryRepository.CompletedFileMetaByID(ctx, fileID)
}

func (r *mediaRepoOverride) ListStaleUploads(ctx context.Context, cutoff time.Time, limit int) ([]pendingUpload, error) {
	if r.listErr != nil {
		return nil, r.listErr
	}
	return r.InMemoryRepository.ListStaleUploads(ctx, cutoff, limit)
}

func (r *mediaRepoOverride) DeleteUploadRow(ctx context.Context, uploadID string) error {
	if r.deleteRowErr != nil {
		return r.deleteRowErr
	}
	return r.InMemoryRepository.DeleteUploadRow(ctx, uploadID)
}

func staticSource(t *testing.T, apply func(*appconfig.AppConfig)) *appconfig.StaticSource {
	t.Helper()
	source := appconfig.NewStaticSource(config.Config{})
	if _, err := source.Update(context.Background(), func(cfg appconfig.AppConfig) appconfig.AppConfig {
		apply(&cfg)
		return cfg
	}); err != nil {
		t.Fatalf("update app config: %v", err)
	}
	return source
}

func TestServiceBatchLimitForUsesAppConfig(t *testing.T) {
	svc := NewServiceWithRepoAndProvider(NewInMemoryRepository(), &coverageProvider{})
	if got := svc.BatchLimitFor("moment_image"); got != 9 {
		t.Fatalf("batch limit without app config = %d, want 9", got)
	}

	source := staticSource(t, func(cfg *appconfig.AppConfig) {
		cfg.Limits.MaxMomentImages = 6
		cfg.Limits.MaxChatFiles = 4
	})
	svc.SetAppConfig(source)
	for category, want := range map[string]int{
		"moment_image": 6,
		"chat_image":   4,
		"chat_video":   4,
		"chat_audio":   4,
		"chat_file":    4,
		"avatar":       9,
	} {
		if got := svc.BatchLimitFor(category); got != want {
			t.Fatalf("BatchLimitFor(%q) = %d, want %d", category, got, want)
		}
	}

	// 配置字段为 0 → 回退默认 9（switch 内不提前返回）。
	source = staticSource(t, func(cfg *appconfig.AppConfig) {
		cfg.Limits.MaxMomentImages = 0
		cfg.Limits.MaxChatFiles = 0
	})
	svc.SetAppConfig(source)
	for _, category := range []string{"moment_image", "chat_image"} {
		if got := svc.BatchLimitFor(category); got != 9 {
			t.Fatalf("BatchLimitFor(%q) with zero config = %d, want 9", category, got)
		}
	}
}

func TestServiceMaxFileSizeUsesAppConfigUploadLimits(t *testing.T) {
	svc := NewServiceWithRepoAndProvider(NewInMemoryRepository(), &coverageProvider{})
	svc.SetAppConfig(staticSource(t, func(cfg *appconfig.AppConfig) {
		cfg.Upload.AvatarMaxSize = 1024
		cfg.Upload.ImageMaxSize = 2048
		cfg.Upload.VideoMaxSize = 4096
		cfg.Upload.AudioMaxSize = 8192
		cfg.Upload.ChatFileMaxSize = 16384
	}))
	for category, want := range map[MediaCategory]int{
		CategoryAvatar:           1024,
		CategoryGroupAvatar:      1024,
		CategoryMomentImage:      2048,
		CategoryChatImage:        2048,
		CategoryChatVideo:        4096,
		CategoryChatAudio:        8192,
		CategoryChatFile:         16384,
		MediaCategory("unknown"): 0,
	} {
		if got := svc.maxFileSize(category); got != want {
			t.Fatalf("maxFileSize(%q) = %d, want %d", category, got, want)
		}
	}

	// 配置为 0 → 回退到与迁移前一致的兜底常量。
	svc.SetAppConfig(staticSource(t, func(cfg *appconfig.AppConfig) {
		cfg.Upload = appconfig.UploadConfig{}
	}))
	if got := svc.maxFileSize(CategoryAvatar); got != defaultAvatarMaxSize {
		t.Fatalf("avatar fallback size = %d, want %d", got, defaultAvatarMaxSize)
	}
	if got := svc.maxFileSize(CategoryChatFile); got != defaultChatFileMaxSize {
		t.Fatalf("chat file fallback size = %d, want %d", got, defaultChatFileMaxSize)
	}
}

func TestValidatePresignUploadRequestBoundaryBranches(t *testing.T) {
	if err := callValidate(PresignFileInput{FileName: strings.Repeat("a", 256), FileSize: 100, MimeType: "image/png", Category: CategoryAvatar}); !errors.Is(err, ErrInvalidMediaRequest) {
		t.Fatalf("file name longer than 255 bytes must be rejected, got %v", err)
	}
	if err := callValidate(PresignFileInput{FileName: "a.png", FileSize: 100, MimeType: "   ", Category: CategoryAvatar}); !errors.Is(err, ErrInvalidMediaRequest) {
		t.Fatalf("blank mime type must be rejected, got %v", err)
	}
	if isAllowedMediaCategory(MediaCategory("bogus")) {
		t.Fatal("unknown category must not be allowed")
	}
	if isMimeAllowed(MediaCategory("bogus"), "image/png") {
		t.Fatal("unknown category must not allow any mime")
	}
}

func TestValidateCompletedObjectMetadataBranches(t *testing.T) {
	upload := pendingUpload{Category: CategoryAvatar}
	if err := validateCompletedObjectMetadata(upload, ObjectMetadata{FileSize: 0, MimeType: "image/png"}); !errors.Is(err, ErrUploadConflict) {
		t.Fatalf("zero-size object must conflict, got %v", err)
	}
	if err := validateCompletedObjectMetadata(upload, ObjectMetadata{FileSize: 10, MimeType: "   "}); !errors.Is(err, ErrUploadConflict) {
		t.Fatalf("missing content type must conflict, got %v", err)
	}
	// 声明 MIME 缺失时由类别白名单兜底：视频 MIME 不属于 avatar 类别。
	if err := validateCompletedObjectMetadata(upload, ObjectMetadata{FileSize: 10, MimeType: "video/mp4"}); !errors.Is(err, ErrUploadConflict) {
		t.Fatalf("category/mime mismatch must conflict, got %v", err)
	}
	if err := validateCompletedObjectMetadata(upload, ObjectMetadata{FileSize: 10, MimeType: "image/png; charset=binary"}); err != nil {
		t.Fatalf("parameterised image mime must normalize and pass, got %v", err)
	}
}

func TestNormalizeMimeTypeAndStorageKeyFallbacks(t *testing.T) {
	if got := normalizeMimeType("text/html; charset"); got != "text/html; charset" {
		t.Fatalf("unparsable mime must fall back to trimmed input, got %q", got)
	}
	if got := normalizeMimeType("  IMAGE/PNG  "); got != "image/png" {
		t.Fatalf("mime must be lower-cased and trimmed, got %q", got)
	}
	if got := storageKeyFor("u1", "upload_1", "README"); got != "uploads/u1/upload_1.bin" {
		t.Fatalf("extension-less file name storage key = %q", got)
	}
	if got := storageKeyFor("u1", "upload_1", "A.PNG"); got != "uploads/u1/upload_1.png" {
		t.Fatalf("extension must be lower-cased, got %q", got)
	}
}

// coveragePNGWithEXIF 构造含 eXIf 块的 PNG（GPS 载体）。
func coveragePNGWithEXIF() []byte {
	data := append([]byte(nil), pngSignature...)
	data = append(data, pngChunk("IHDR", []byte("0000000d"))...)
	data = append(data, pngChunk("eXIf", []byte("MM\x00\x2aGPS-LOCATION"))...)
	data = append(data, pngChunk("IDAT", []byte("imagedata"))...)
	data = append(data, pngChunk("IEND", nil)...)
	return data
}

func TestScrubImageMetadataPaths(t *testing.T) {
	ctx := context.Background()

	t.Run("get object error keeps original", func(t *testing.T) {
		provider := &coverageProvider{getObjectErr: errors.New("get boom")}
		NewServiceWithRepoAndProvider(NewInMemoryRepository(), provider).scrubImageMetadata(ctx, "uploads/u1/a.png", "image/png")
		if provider.putData != nil {
			t.Fatal("must not write back when object read fails")
		}
	})

	t.Run("parse error keeps original", func(t *testing.T) {
		provider := &coverageProvider{getObject: []byte{0xFF, 0xD8, 0xFF, 0xE1, 0x7F, 0xFF, 0x01, 0x02}}
		NewServiceWithRepoAndProvider(NewInMemoryRepository(), provider).scrubImageMetadata(ctx, "uploads/u1/a.png", "image/jpeg")
		if provider.putData != nil {
			t.Fatal("must not write back when metadata parsing fails")
		}
	})

	t.Run("cleaned object is written back", func(t *testing.T) {
		provider := &coverageProvider{getObject: coveragePNGWithEXIF()}
		NewServiceWithRepoAndProvider(NewInMemoryRepository(), provider).scrubImageMetadata(ctx, "uploads/u1/a.png", "image/png")
		if provider.putContentType != "image/png" {
			t.Fatalf("put content type = %q", provider.putContentType)
		}
		if provider.putData == nil || bytes.Contains(provider.putData, []byte("GPS-LOCATION")) {
			t.Fatalf("cleaned object must drop EXIF payload, got %q", provider.putData)
		}
		if !bytes.Contains(provider.putData, []byte("imagedata")) {
			t.Fatal("image payload must survive scrubbing")
		}
	})

	t.Run("put failure is fail-open", func(t *testing.T) {
		provider := &coverageProvider{getObject: coveragePNGWithEXIF(), putErr: errors.New("put boom")}
		NewServiceWithRepoAndProvider(NewInMemoryRepository(), provider).scrubImageMetadata(ctx, "uploads/u1/a.png", "image/png")
		if provider.putData == nil {
			t.Fatal("expected cleaned object write attempt")
		}
	})
}

func TestServicePresignErrorBranches(t *testing.T) {
	valid := PresignFileInput{FileName: "a.png", FileSize: 100, MimeType: "image/png", Category: CategoryAvatar}
	svc := NewServiceWithRepoAndProvider(NewInMemoryRepository(), &coverageProvider{presignResp: PresignedUpload{UploadID: "upload_mediacov", StorageKey: "uploads/u1/upload_mediacov.png"}})

	if _, err := svc.Presign(context.Background(), valid); !errors.Is(err, ErrInvalidMediaRequest) {
		t.Fatalf("unauthenticated presign must be invalid, got %v", err)
	}
	if _, err := svc.Presign(mediaContextWithUserAndKey("u1", ""), PresignFileInput{FileName: "", FileSize: 100, MimeType: "image/png", Category: CategoryAvatar}); !errors.Is(err, ErrInvalidMediaRequest) {
		t.Fatalf("invalid file must be rejected, got %v", err)
	}
	if _, err := svc.BatchPresign(context.Background(), []PresignFileInput{valid}); !errors.Is(err, ErrInvalidMediaRequest) {
		t.Fatalf("unauthenticated batch presign must be invalid, got %v", err)
	}

	providerErr := errors.New("presign boom")
	providerSvc := NewServiceWithRepoAndProvider(NewInMemoryRepository(), &coverageProvider{presignErr: providerErr})
	if _, err := providerSvc.Presign(mediaContextWithUserAndKey("u1", ""), valid); !errors.Is(err, providerErr) {
		t.Fatalf("provider error must propagate, got %v", err)
	}

	saveErr := errors.New("save boom")
	saveSvc := NewServiceWithRepoAndProvider(&mediaErrorRepo{saveErr: saveErr}, &coverageProvider{presignResp: PresignedUpload{UploadID: "upload_mediacov"}})
	if _, err := saveSvc.Presign(mediaContextWithUserAndKey("u1", ""), valid); !errors.Is(err, saveErr) {
		t.Fatalf("repository save error must propagate, got %v", err)
	}
}

func TestCommitCompleteBranches(t *testing.T) {
	ctx := mediaContextWithUserAndKey("u1", "key1")
	file := UploadedMedia{FileID: "file_mediacov_commit", FileSize: 100, MimeType: "image/png"}

	t.Run("requires principal", func(t *testing.T) {
		svc := NewServiceWithRepoAndProvider(NewInMemoryRepository(), &coverageProvider{})
		if _, err := svc.CommitComplete(context.Background(), "upload_mediacov_commit", file); !errors.Is(err, ErrInvalidMediaRequest) {
			t.Fatalf("expected ErrInvalidMediaRequest, got %v", err)
		}
	})

	t.Run("complete write error", func(t *testing.T) {
		writeErr := errors.New("write boom")
		repo := &mediaRepoOverride{InMemoryRepository: NewInMemoryRepository(), completeErr: writeErr}
		svc := NewServiceWithRepoAndProvider(repo, &coverageProvider{})
		if _, err := svc.CommitComplete(ctx, "upload_mediacov_commit", file); !errors.Is(err, writeErr) {
			t.Fatalf("expected write error, got %v", err)
		}
	})

	t.Run("read back error", func(t *testing.T) {
		readErr := errors.New("read boom")
		repo := &mediaRepoOverride{InMemoryRepository: NewInMemoryRepository(), completeNoop: true, getErr: readErr}
		svc := NewServiceWithRepoAndProvider(repo, &coverageProvider{})
		if _, err := svc.CommitComplete(ctx, "upload_mediacov_commit", file); !errors.Is(err, readErr) {
			t.Fatalf("expected read error, got %v", err)
		}
	})

	t.Run("upload vanished", func(t *testing.T) {
		repo := &mediaRepoOverride{InMemoryRepository: NewInMemoryRepository(), completeNoop: true}
		svc := NewServiceWithRepoAndProvider(repo, &coverageProvider{})
		if _, err := svc.CommitComplete(ctx, "upload_mediacov_commit", file); !errors.Is(err, ErrUploadNotFound) {
			t.Fatalf("expected ErrUploadNotFound, got %v", err)
		}
	})

	t.Run("upload still pending after no-op update", func(t *testing.T) {
		repo := NewInMemoryRepository()
		if err := repo.Save(context.Background(), pendingUpload{UploadID: "upload_mediacov_pending", UserID: "u1", StorageKey: "uploads/u1/p.png", FileName: "p.png", DeclaredFileSize: 100, DeclaredMimeType: "image/png", Category: CategoryAvatar}); err != nil {
			t.Fatalf("save pending upload: %v", err)
		}
		svc := NewServiceWithRepoAndProvider(&mediaRepoOverride{InMemoryRepository: repo, completeNoop: true}, &coverageProvider{})
		if _, err := svc.CommitComplete(ctx, "upload_mediacov_pending", file); !errors.Is(err, ErrUploadNotFound) {
			t.Fatalf("expected ErrUploadNotFound for pending row, got %v", err)
		}
	})

	t.Run("already completed is idempotent", func(t *testing.T) {
		repo := NewInMemoryRepository()
		seedCompletedAvatar(t, repo, "mediacov_commit_done", "u1", CategoryAvatar)
		svc := NewServiceWithRepoAndProvider(&mediaRepoOverride{InMemoryRepository: repo, completeNoop: true}, &coverageProvider{})
		got, err := svc.CommitComplete(ctx, "mediacov_commit_done", UploadedMedia{FileID: "file_ignored", FileSize: 1, MimeType: "image/jpeg"})
		if err != nil {
			t.Fatalf("completed upload must stay idempotent, got %v", err)
		}
		if got.FileID != "file_mediacov_commit_done" {
			t.Fatalf("idempotent replay must return stored file, got %#v", got)
		}
	})
}

func TestServiceResolveImageErrorBranches(t *testing.T) {
	resolveErr := errors.New("resolve boom")
	repo := &mediaRepoOverride{InMemoryRepository: NewInMemoryRepository(), resolveErr: resolveErr}
	svc := NewServiceWithRepoAndProvider(repo, &coverageProvider{presignGetURL: "https://storage.example.com/signed"})

	ctx := context.Background()
	if _, err := svc.ResolveMomentImages(ctx, []string{"file_mediacov"}, "u1"); !errors.Is(err, resolveErr) {
		t.Fatalf("moment image resolve error must propagate, got %v", err)
	}
	if err := svc.ResolveAvatarImage(ctx, "file_mediacov", "u1"); !errors.Is(err, resolveErr) {
		t.Fatalf("avatar resolve error must propagate, got %v", err)
	}
	if err := svc.ResolveGroupAvatarImage(ctx, "file_mediacov", "u1"); !errors.Is(err, resolveErr) {
		t.Fatalf("group avatar resolve error must propagate, got %v", err)
	}

	emptySvc := NewServiceWithRepoAndProvider(NewInMemoryRepository(), &coverageProvider{})
	if err := emptySvc.ResolveAvatarImage(ctx, "file_mediacov_missing", "u1"); !errors.Is(err, ErrMediaNotOwned) {
		t.Fatalf("expected ErrMediaNotOwned for missing avatar, got %v", err)
	}
	if err := emptySvc.ResolveGroupAvatarImage(ctx, "file_mediacov_missing", "u1"); !errors.Is(err, ErrMediaNotOwned) {
		t.Fatalf("expected ErrMediaNotOwned for missing group avatar, got %v", err)
	}
}

func TestServiceAvatarReadURLAndGroupAvatar(t *testing.T) {
	repo := NewInMemoryRepository()
	provider := &coverageProvider{presignGetURL: "https://storage.example.com/signed-avatar"}
	svc := NewServiceWithRepoAndProvider(repo, provider)
	ctx := context.Background()

	if got := svc.AvatarPublicPath("file_mediacov"); got != "/public/v1/media/avatars/file_mediacov" {
		t.Fatalf("AvatarPublicPath = %q", got)
	}

	avatarID := seedCompletedAvatar(t, repo, "mediacov_avatar_service", "u1", CategoryAvatar)
	groupAvatarID := seedCompletedAvatar(t, repo, "mediacov_group_avatar_service", "u1", CategoryGroupAvatar)
	momentID := seedCompletedAvatar(t, repo, "mediacov_moment_service", "u1", CategoryMomentImage)

	if err := svc.ResolveGroupAvatarImage(ctx, groupAvatarID, "u1"); err != nil {
		t.Fatalf("own completed group avatar must resolve, got %v", err)
	}
	if err := svc.ResolveGroupAvatarImage(ctx, groupAvatarID, "u2"); !errors.Is(err, ErrMediaNotOwned) {
		t.Fatalf("foreign group avatar must be rejected, got %v", err)
	}
	// group_avatar 不能经 avatar 类别接口解析。
	if err := svc.ResolveAvatarImage(ctx, groupAvatarID, "u1"); !errors.Is(err, ErrMediaNotOwned) {
		t.Fatalf("group avatar must not resolve as avatar, got %v", err)
	}

	for _, fileID := range []string{avatarID, groupAvatarID} {
		url, err := svc.AvatarReadURL(ctx, fileID)
		if err != nil || url != provider.presignGetURL {
			t.Fatalf("AvatarReadURL(%q) = %q err=%v", fileID, url, err)
		}
	}
	if _, err := svc.AvatarReadURL(ctx, momentID); !errors.Is(err, ErrMediaNotOwned) {
		t.Fatalf("moment image must not be served as avatar, got %v", err)
	}
	if _, err := svc.AvatarReadURL(ctx, "file_mediacov_missing"); !errors.Is(err, ErrMediaNotOwned) {
		t.Fatalf("missing file must be not-owned, got %v", err)
	}

	errRepo := &mediaRepoOverride{InMemoryRepository: repo, metaErr: errors.New("meta boom")}
	if _, err := NewServiceWithRepoAndProvider(errRepo, provider).AvatarReadURL(ctx, avatarID); err == nil {
		t.Fatal("repository error must propagate from AvatarReadURL")
	}
}

func TestServiceCompletedFileMetaByID(t *testing.T) {
	repo := NewInMemoryRepository()
	fileID := seedCompletedAvatar(t, repo, "mediacov_meta_service", "u1", CategoryAvatar)
	svc := NewServiceWithRepoAndProvider(repo, &coverageProvider{})

	meta, found, err := svc.CompletedFileMetaByID(context.Background(), fileID)
	if err != nil || !found {
		t.Fatalf("CompletedFileMetaByID = found=%v err=%v", found, err)
	}
	if meta.Category != CategoryAvatar || meta.FileSize != 100 || meta.MimeType != "image/png" {
		t.Fatalf("unexpected file meta: %#v", meta)
	}
	if _, found, err := svc.CompletedFileMetaByID(context.Background(), "file_mediacov_absent"); err != nil || found {
		t.Fatalf("missing file meta = found=%v err=%v", found, err)
	}
}
