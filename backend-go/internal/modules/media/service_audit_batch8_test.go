package media

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/appconfig"
	"hotspot/backend-go/internal/platform/httpx"
)

// ── 第 82 轮审计 P1-1：扩展名黑名单 × MIME 一致性（spec api_spec_media.md）──

// spec「chat_file 安全约束」：可执行/脚本标记/服务端脚本扩展名必须拒绝，
// 且「扩展名 + MIME 双判定」（任一命中即拒）。黑名单对所有上传类别生效。
func TestValidatePresignUploadRejectsForbiddenExtensions(t *testing.T) {
	cases := []struct {
		name string
		req  PresignFileInput
	}{
		{"exe with octet-stream", PresignFileInput{FileName: "malware.exe", FileSize: 1024, MimeType: "application/octet-stream", Category: CategoryChatFile}},
		{"html disguised as text/plain", PresignFileInput{FileName: "page.html", FileSize: 1024, MimeType: "text/plain", Category: CategoryChatFile}},
		{"shell script", PresignFileInput{FileName: "run.sh", FileSize: 1024, MimeType: "text/x-sh", Category: CategoryChatFile}},
		{"js script", PresignFileInput{FileName: "payload.js", FileSize: 1024, MimeType: "application/octet-stream", Category: CategoryChatFile}},
		{"php server script", PresignFileInput{FileName: "shell.php", FileSize: 1024, MimeType: "text/plain", Category: CategoryChatFile}},
		{"aspx server script", PresignFileInput{FileName: "shell.aspx", FileSize: 1024, MimeType: "text/plain", Category: CategoryChatFile}},
		{"uppercase extension", PresignFileInput{FileName: "MALWARE.EXE", FileSize: 1024, MimeType: "application/octet-stream", Category: CategoryChatFile}},
		{"trailing dot bypass", PresignFileInput{FileName: "page.html.", FileSize: 1024, MimeType: "text/plain", Category: CategoryChatFile}},
		{"trailing space bypass", PresignFileInput{FileName: "page.html ", FileSize: 1024, MimeType: "text/plain", Category: CategoryChatFile}},
		// 黑名单不限于 chat_file：图片/视频/音频类别同样不得以可执行/标记扩展名落库。
		{"avatar html", PresignFileInput{FileName: "page.html", FileSize: 100, MimeType: "image/png", Category: CategoryAvatar}},
		{"moment svg", PresignFileInput{FileName: "clip.svg", FileSize: 100, MimeType: "image/png", Category: CategoryMomentImage}},
		{"chat video html", PresignFileInput{FileName: "clip.html", FileSize: 100, MimeType: "video/mp4", Category: CategoryChatVideo}},
		{"chat audio exe", PresignFileInput{FileName: "voice.exe", FileSize: 100, MimeType: "audio/mpeg", Category: CategoryChatAudio}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := callValidate(tc.req); !errors.Is(err, ErrInvalidMediaRequest) {
				t.Fatalf("expected forbidden extension to be rejected, req=%+v got %v", tc.req, err)
			}
		})
	}

	// 黑名单不是把类别封死：普通附件、未知扩展名、无扩展名仍放行（MIME 维度
	// 继续由类别白名单/活跃内容黑名单把关）。
	allowed := []PresignFileInput{
		{FileName: "notes.txt", FileSize: 100, MimeType: "text/plain", Category: CategoryChatFile},
		{FileName: "archive.bin", FileSize: 100, MimeType: "application/octet-stream", Category: CategoryChatFile},
		{FileName: "README", FileSize: 100, MimeType: "text/plain", Category: CategoryChatFile},
		{FileName: "photo.PNG", FileSize: 100, MimeType: "image/png", Category: CategoryAvatar},
		{FileName: "upload_1700000000", FileSize: 100, MimeType: "image/jpeg", Category: CategoryChatImage},
	}
	for _, req := range allowed {
		if err := callValidate(req); err != nil {
			t.Fatalf("expected ordinary attachment %+v to pass, got %v", req, err)
		}
	}
}

// spec：MIME 类型与扩展名不一致时拒绝（如 .png 扩展名但 application/pdf MIME）。
func TestValidatePresignUploadRejectsMimeExtensionMismatch(t *testing.T) {
	mismatches := []PresignFileInput{
		{FileName: "photo.png", FileSize: 100, MimeType: "application/pdf", Category: CategoryChatFile},
		{FileName: "doc.pdf", FileSize: 100, MimeType: "image/png", Category: CategoryChatFile},
		{FileName: "a.png", FileSize: 100, MimeType: "image/jpeg", Category: CategoryAvatar},
		{FileName: "clip.mp4", FileSize: 100, MimeType: "video/quicktime", Category: CategoryChatVideo},
		{FileName: "song.mp3", FileSize: 100, MimeType: "audio/ogg", Category: CategoryChatAudio},
		{FileName: "notes.txt", FileSize: 100, MimeType: "image/png", Category: CategoryChatFile},
	}
	for _, req := range mismatches {
		if err := callValidate(req); !errors.Is(err, ErrInvalidMediaRequest) {
			t.Fatalf("expected mime/extension mismatch for %+v to be rejected, got %v", req, err)
		}
	}

	allowed := []PresignFileInput{
		{FileName: "photo.png", FileSize: 100, MimeType: "image/png; charset=binary", Category: CategoryAvatar},
		{FileName: "photo.jpeg", FileSize: 100, MimeType: "image/jpeg", Category: CategoryAvatar},
		{FileName: "doc.pdf", FileSize: 100, MimeType: "application/pdf", Category: CategoryChatFile},
		{FileName: "data.dat", FileSize: 100, MimeType: "application/octet-stream", Category: CategoryChatFile},
	}
	for _, req := range allowed {
		if err := callValidate(req); err != nil {
			t.Fatalf("expected consistent mime/extension %+v to pass, got %v", req, err)
		}
	}
}

// 服务入口与 handler 入口都必须在 presign 阶段拒绝：不签发上传 URL、不落库。
func TestBatchPresignRejectsForbiddenExtensionWithoutSigningURL(t *testing.T) {
	provider := &quotaStubProvider{}
	svc := NewServiceWithRepoAndProvider(NewInMemoryRepository(), provider)
	_, err := svc.BatchPresign(mediaContextWithUserAndKey("u1", ""), []PresignFileInput{{
		FileName: "evil.exe", FileSize: 1024, MimeType: "application/octet-stream", Category: CategoryChatFile,
	}})
	if !errors.Is(err, ErrInvalidMediaRequest) {
		t.Fatalf("expected invalid request, got %v", err)
	}
	if provider.presignCalls != 0 {
		t.Fatalf("rejected file must not be presigned, calls = %d", provider.presignCalls)
	}
}

func TestMediaHandlerPresignRejectsForbiddenExtension(t *testing.T) {
	handler := NewHandler(NewServiceWithRepoAndProvider(NewInMemoryRepository(), &mediaStubProvider{
		presignResp: PresignedUpload{UploadID: "upload_evil", UploadURL: "https://storage.example.com/upload", ExpiresAt: "2026-03-30T00:00:00Z"},
	}))
	body, err := json.Marshal(map[string]any{"files": []map[string]any{{
		"file_name": "evil.exe", "file_size": 1024, "mime_type": "application/octet-stream", "category": "chat_file",
	}}})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	rr := httptest.NewRecorder()
	handler.PresignUpload(rr, mediaAuthedRequest(http.MethodPost, "/presign_upload", string(body)))
	assertMediaCode(t, rr, httpx.CodeInvalidRequest)
}

// ── 第 82 轮审计 P1-2：批量上限（presign 取批内类别最大值 / complete 与之一致）──

func TestBatchLimitForFilesUsesMaxAcrossCategories(t *testing.T) {
	svc := NewServiceWithRepoAndProvider(NewInMemoryRepository(), &coverageProvider{})
	svc.SetAppConfig(staticSource(t, func(cfg *appconfig.AppConfig) {
		cfg.Limits.MaxMomentImages = 2
		cfg.Limits.MaxChatFiles = 5
	}))
	cases := []struct {
		name  string
		files []PresignFileInput
		want  int
	}{
		{"empty falls back to default", nil, 9},
		{"moment only", []PresignFileInput{{Category: CategoryMomentImage}}, 2},
		{"chat only", []PresignFileInput{{Category: CategoryChatFile}}, 5},
		{"mixed takes category max", []PresignFileInput{{Category: CategoryMomentImage}, {Category: CategoryChatFile}}, 5},
		{"avatar keeps default", []PresignFileInput{{Category: CategoryAvatar}}, 9},
	}
	for _, tc := range cases {
		if got := svc.BatchLimitForFiles(tc.files); got != tc.want {
			t.Fatalf("BatchLimitForFiles(%s) = %d, want %d", tc.name, got, tc.want)
		}
	}
}

func TestCompleteBatchLimitCoversAllCategories(t *testing.T) {
	svc := NewServiceWithRepoAndProvider(NewInMemoryRepository(), &coverageProvider{})
	if got := svc.CompleteBatchLimit(); got != 9 {
		t.Fatalf("default CompleteBatchLimit = %d, want 9", got)
	}
	// 档位调低不得把 complete 上限压到 9 以下（avatar 等类别固定 9，presign 对
	// avatar 批仍可到 9）；调高则必须跟随放大。
	svc.SetAppConfig(staticSource(t, func(cfg *appconfig.AppConfig) {
		cfg.Limits.MaxMomentImages = 2
		cfg.Limits.MaxChatFiles = 4
	}))
	if got := svc.CompleteBatchLimit(); got != 9 {
		t.Fatalf("tightened CompleteBatchLimit = %d, want 9 (avatar floor)", got)
	}
	svc.SetAppConfig(staticSource(t, func(cfg *appconfig.AppConfig) {
		cfg.Limits.MaxMomentImages = 12
		cfg.Limits.MaxChatFiles = 4
	}))
	if got := svc.CompleteBatchLimit(); got != 12 {
		t.Fatalf("raised CompleteBatchLimit = %d, want 12", got)
	}
}

// 混合类别批次不得被 files[0] 的类别误限（第 82 轮审计 P2-1 备注）：config
// max_moment_images=2、max_chat_files=9 时，[moment, chat, chat] 应放行。
func TestMediaHandlerPresignMixedBatchUsesMaxCategoryLimit(t *testing.T) {
	handler := NewHandler(NewServiceWithRepoAndProvider(NewInMemoryRepository(), &mediaStubProvider{
		presignResp: PresignedUpload{
			UploadID: "upload_mixed", UploadURL: "https://storage.example.com/upload",
			Headers: map[string]string{"Content-Type": "image/png"}, ExpiresAt: "2026-03-30T00:00:00Z",
			StorageKey: "uploads/u1/upload_mixed.png",
		},
	}))
	handler.service.SetAppConfig(staticSource(t, func(cfg *appconfig.AppConfig) {
		cfg.Limits.MaxMomentImages = 2
		cfg.Limits.MaxChatFiles = 9
	}))
	body, err := json.Marshal(map[string]any{"files": []map[string]any{
		{"file_name": "a.png", "file_size": 100, "mime_type": "image/png", "category": "moment_image"},
		{"file_name": "b.pdf", "file_size": 100, "mime_type": "application/pdf", "category": "chat_file"},
		{"file_name": "c.pdf", "file_size": 100, "mime_type": "application/pdf", "category": "chat_file"},
	}})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	rr := httptest.NewRecorder()
	handler.PresignUpload(rr, mediaAuthedRequest(http.MethodPost, "/presign_upload", string(body)))
	var resp struct {
		Code httpx.Code       "json:\"code\""
		Data PresignedUploads "json:\"data\""
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != 0 || len(resp.Data.Uploads) != 3 {
		t.Fatalf("mixed batch within max category limit must pass, got %#v", resp)
	}
}

// complete_upload 的 upload_ids 上限与 presign 一致：默认 9；超限 10000。
func TestMediaHandlerCompleteRejectsOversizeBatch(t *testing.T) {
	handler := NewHandler(NewServiceWithRepoAndProvider(NewInMemoryRepository(), &mediaStubProvider{}))
	makeBody := func(n int) string {
		raw := make([]string, n)
		for i := 0; i < n; i++ {
			raw[i] = fmt.Sprintf("%q", fmt.Sprintf("upload_missing_%02d", i))
		}
		return "{\"upload_ids\":[" + strings.Join(raw, ",") + "]}"
	}
	oversize := httptest.NewRecorder()
	handler.CompleteUpload(oversize, mediaAuthedRequest(http.MethodPost, "/complete_upload", makeBody(10)))
	assertMediaCode(t, oversize, httpx.CodeInvalidRequest)

	// 恰好 9 个（不存在的 id）：通过批量上限后落到 10300，证明上限就是 9。
	within := httptest.NewRecorder()
	handler.CompleteUpload(within, mediaAuthedRequest(http.MethodPost, "/complete_upload", makeBody(9)))
	assertMediaCode(t, within, httpx.CodeNotFound)
}

// ── 第 82 轮审计 P1-3(a)：complete 确定性失败立即释放会话（对象+行）────────

func savePendingSession(t *testing.T, repo Repository, upload pendingUpload) {
	t.Helper()
	if err := repo.Save(context.Background(), upload); err != nil {
		t.Fatalf("save %s: %v", upload.UploadID, err)
	}
}

func TestCompleteDeterministicFailureReleasesPendingSession(t *testing.T) {
	now := time.Now().UTC().Format(time.RFC3339)
	cases := []struct {
		name       string
		storedName string
		cat        MediaCategory
		mime       string
	}{
		{"forbidden extension from legacy row", "page.html", CategoryChatFile, "text/plain"},
		{"mime extension mismatch", "photo.png", CategoryChatFile, "application/pdf"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := NewInMemoryRepository()
			provider := &mediaStubProvider{headResp: ObjectMetadata{FileSize: 100, MimeType: tc.mime}}
			svc := NewServiceWithRepoAndProvider(repo, provider)
			uploadID := "upload_release_" + strings.ReplaceAll(tc.name, " ", "_")
			storageKey := "uploads/u1/" + uploadID + ".bin"
			savePendingSession(t, repo, pendingUpload{
				UploadID: uploadID, UserID: "u1", StorageKey: storageKey,
				FileName: tc.storedName, DeclaredFileSize: 100, DeclaredMimeType: tc.mime,
				Category: tc.cat, CreatedAt: now,
			})
			if _, err := svc.Complete(mediaContextWithUserAndKey("u1", "key1"), uploadID); !errors.Is(err, ErrUploadConflict) {
				t.Fatalf("expected ErrUploadConflict, got %v", err)
			}
			if _, found, err := repo.Get(context.Background(), uploadID); err != nil || found {
				t.Fatalf("rejected session must be released, found=%v err=%v", found, err)
			}
			if usage := repoUsage(t, repo, "u1"); usage.Count != 0 || usage.DeclaredBytes != 0 {
				t.Fatalf("released session must free quota, usage = %#v", usage)
			}
			if len(provider.deletedKeys) != 1 || provider.deletedKeys[0] != storageKey {
				t.Fatalf("upload object must be deleted, deleted = %#v", provider.deletedKeys)
			}
		})
	}
}

// 对象删除失败时保留行（下一轮重试再释放），避免「行删了对象还在」的永久孤儿。
func TestCompleteDeterministicFailureKeepsRowWhenObjectDeleteFails(t *testing.T) {
	repo := NewInMemoryRepository()
	provider := &mediaStubProvider{
		headResp:  ObjectMetadata{FileSize: 100, MimeType: "text/plain"},
		deleteErr: errors.New("storage temporarily unavailable"),
	}
	svc := NewServiceWithRepoAndProvider(repo, provider)
	const uploadID = "upload_release_retry"
	savePendingSession(t, repo, pendingUpload{
		UploadID: uploadID, UserID: "u1", StorageKey: "uploads/u1/" + uploadID + ".html",
		FileName: "page.html", DeclaredFileSize: 100, DeclaredMimeType: "text/plain",
		Category: CategoryChatFile, CreatedAt: time.Now().UTC().Format(time.RFC3339),
	})
	if _, err := svc.Complete(mediaContextWithUserAndKey("u1", "key1"), uploadID); !errors.Is(err, ErrUploadConflict) {
		t.Fatalf("expected ErrUploadConflict, got %v", err)
	}
	if _, found, _ := repo.Get(context.Background(), uploadID); !found {
		t.Fatal("row must be kept when object delete fails")
	}
	// 存储恢复后重试：确定性失败再次触发释放，对象+行一起删除。
	provider.deleteErr = nil
	if _, err := svc.Complete(mediaContextWithUserAndKey("u1", "key1"), uploadID); !errors.Is(err, ErrUploadConflict) {
		t.Fatalf("expected ErrUploadConflict on retry, got %v", err)
	}
	if _, found, _ := repo.Get(context.Background(), uploadID); found {
		t.Fatal("row must be released on retry success")
	}
}

// 瞬时错误（直传未完成/读取失败）不得释放会话：客户端可完成直传后重试。
func TestCompleteTransientFailureKeepsPendingSession(t *testing.T) {
	t.Run("upload not ready", func(t *testing.T) {
		repo := NewInMemoryRepository()
		provider := &mediaStubProvider{headErr: ErrUploadNotReady}
		svc := NewServiceWithRepoAndProvider(repo, provider)
		const uploadID = "upload_transient_not_ready"
		savePendingSession(t, repo, pendingUpload{
			UploadID: uploadID, UserID: "u1", StorageKey: "uploads/u1/" + uploadID + ".png",
			FileName: "a.png", DeclaredFileSize: 100, DeclaredMimeType: "image/png",
			Category: CategoryAvatar, CreatedAt: time.Now().UTC().Format(time.RFC3339),
		})
		if _, err := svc.Complete(mediaContextWithUserAndKey("u1", "key1"), uploadID); !errors.Is(err, ErrUploadNotReady) {
			t.Fatalf("expected ErrUploadNotReady, got %v", err)
		}
		if _, found, _ := repo.Get(context.Background(), uploadID); !found {
			t.Fatal("transient failure must keep the pending row")
		}
		if len(provider.deletedKeys) != 0 {
			t.Fatalf("transient failure must not delete objects, deleted = %#v", provider.deletedKeys)
		}
	})

	t.Run("read prefix error", func(t *testing.T) {
		repo := NewInMemoryRepository()
		provider := &mediaStubProvider{
			headResp:  ObjectMetadata{FileSize: 100, MimeType: "image/png"},
			prefixErr: errors.New("read boom"),
		}
		svc := NewServiceWithRepoAndProvider(repo, provider)
		const uploadID = "upload_transient_read"
		savePendingSession(t, repo, pendingUpload{
			UploadID: uploadID, UserID: "u1", StorageKey: "uploads/u1/" + uploadID + ".png",
			FileName: "a.png", DeclaredFileSize: 100, DeclaredMimeType: "image/png",
			Category: CategoryAvatar, CreatedAt: time.Now().UTC().Format(time.RFC3339),
		})
		if _, err := svc.Complete(mediaContextWithUserAndKey("u1", "key1"), uploadID); err == nil || errors.Is(err, ErrUploadConflict) {
			t.Fatalf("read error must propagate as transient error, got %v", err)
		}
		if _, found, _ := repo.Get(context.Background(), uploadID); !found {
			t.Fatal("transient failure must keep the pending row")
		}
		if len(provider.deletedKeys) != 0 {
			t.Fatalf("transient failure must not delete objects, deleted = %#v", provider.deletedKeys)
		}
	})
}

// ── 第 82 轮审计 P1-3(b)：presign 配额拒绝时清理该用户超宽限期的弃置会话 ──

func seedStaleQuotaSession(t *testing.T, repo Repository, uploadID, userID string, createdAt time.Time) {
	t.Helper()
	savePendingSession(t, repo, pendingUpload{
		UploadID: uploadID, UserID: userID, StorageKey: "uploads/" + userID + "/" + uploadID + ".png",
		FileName: "a.png", DeclaredFileSize: 100, DeclaredMimeType: "image/png",
		Category: CategoryAvatar, CreatedAt: createdAt.UTC().Format(time.RFC3339),
	})
}

func TestPresignQuotaCleanupReleasesStaleSessions(t *testing.T) {
	repo := NewInMemoryRepository()
	now := time.Now().UTC()
	staleAt := now.Add(-(pendingUploadCleanupGrace + time.Minute))
	freshAt := now
	for i := 0; i < pendingUploadSessionLimit; i++ {
		seedStaleQuotaSession(t, repo, fmt.Sprintf("upload_cleanup_stale_%02d", i), "u1", staleAt)
	}
	seedStaleQuotaSession(t, repo, "upload_cleanup_fresh", "u1", freshAt)
	// 其它用户的弃置会话不得被本次清理连带删除。
	seedStaleQuotaSession(t, repo, "upload_cleanup_other_1", "u2", staleAt)
	seedStaleQuotaSession(t, repo, "upload_cleanup_other_2", "u2", staleAt)

	svc, provider := newQuotaService(repo)
	resp, err := svc.BatchPresign(mediaContextWithUserAndKey("u1", ""), []PresignFileInput{quotaAvatarFile("a.png", 100)})
	if err != nil {
		t.Fatalf("cleanup must free quota and allow presign, got %v", err)
	}
	if len(resp.Uploads) != 1 {
		t.Fatalf("presign uploads = %d, want 1", len(resp.Uploads))
	}
	if provider.presignCalls != 1 {
		t.Fatalf("presign calls = %d, want 1", provider.presignCalls)
	}
	for i := 0; i < pendingUploadSessionLimit; i++ {
		if _, found, _ := repo.Get(context.Background(), fmt.Sprintf("upload_cleanup_stale_%02d", i)); found {
			t.Fatalf("stale session %02d must be released", i)
		}
	}
	if len(provider.deletedKeys) != pendingUploadSessionLimit {
		t.Fatalf("deleted objects = %d, want %d", len(provider.deletedKeys), pendingUploadSessionLimit)
	}
	if _, found, _ := repo.Get(context.Background(), "upload_cleanup_fresh"); !found {
		t.Fatal("session within grace period must be kept")
	}
	for _, uploadID := range []string{"upload_cleanup_other_1", "upload_cleanup_other_2"} {
		if _, found, _ := repo.Get(context.Background(), uploadID); !found {
			t.Fatalf("other user's session %s must not be released", uploadID)
		}
	}
	// 清理后用量：宽限期内会话 1 条 + 本次签名 1 条。
	if usage := repoUsage(t, repo, "u1"); usage.Count != 2 {
		t.Fatalf("released quota usage = %#v, want 2 pending sessions", usage)
	}
}

// 宽限期内（预签名 URL 未过期）的会话不得被清理：客户端可能刚上传完尚未 complete。
func TestPresignQuotaCleanupRespectsGracePeriod(t *testing.T) {
	repo := NewInMemoryRepository()
	now := time.Now().UTC()
	for i := 0; i < pendingUploadSessionLimit; i++ {
		seedStaleQuotaSession(t, repo, fmt.Sprintf("upload_cleanup_recent_%02d", i), "u1", now.Add(-time.Minute))
	}
	svc, provider := newQuotaService(repo)
	_, err := svc.BatchPresign(mediaContextWithUserAndKey("u1", ""), []PresignFileInput{quotaAvatarFile("a.png", 100)})
	if !errors.Is(err, ErrUploadQuotaExceeded) {
		t.Fatalf("sessions within grace must not be cleaned, got %v", err)
	}
	if provider.presignCalls != 0 {
		t.Fatalf("rejected presign must not sign URL, calls = %d", provider.presignCalls)
	}
	if len(provider.deletedKeys) != 0 {
		t.Fatalf("sessions within grace must not be deleted, deleted = %#v", provider.deletedKeys)
	}
}

// 清理查询失败只是旁路失败：仍按配额拒绝返回 10400 语义，不把内部错误透传成
// 上传成功或 panic。
func TestPresignQuotaCleanupErrorFallsBackToQuotaError(t *testing.T) {
	repo := &mediaRepoOverride{InMemoryRepository: NewInMemoryRepository(), userListErr: errors.New("list boom")}
	now := time.Now().UTC()
	for i := 0; i < pendingUploadSessionLimit; i++ {
		seedStaleQuotaSession(t, repo, fmt.Sprintf("upload_cleanup_err_%02d", i), "u1", now.Add(-(pendingUploadCleanupGrace + time.Minute)))
	}
	svc, provider := newQuotaService(repo)
	_, err := svc.BatchPresign(mediaContextWithUserAndKey("u1", ""), []PresignFileInput{quotaAvatarFile("a.png", 100)})
	if !errors.Is(err, ErrUploadQuotaExceeded) {
		t.Fatalf("cleanup failure must fall back to quota error, got %v", err)
	}
	if provider.presignCalls != 0 {
		t.Fatalf("rejected presign must not sign URL, calls = %d", provider.presignCalls)
	}
}

// ── 第 82 轮审计 P2-1：AppConfig 大小上限只能收紧、不能放宽 ──────────────

func TestMaxFileSizeCannotExceedContractHardLimit(t *testing.T) {
	svc := NewServiceWithRepoAndProvider(NewInMemoryRepository(), &coverageProvider{})
	svc.SetAppConfig(staticSource(t, func(cfg *appconfig.AppConfig) {
		cfg.Upload.AvatarMaxSize = defaultAvatarMaxSize * 10
		cfg.Upload.ImageMaxSize = defaultImageMaxSize * 10
		cfg.Upload.VideoMaxSize = defaultVideoMaxSize * 10
		cfg.Upload.AudioMaxSize = defaultAudioMaxSize * 10
		cfg.Upload.ChatFileMaxSize = defaultChatFileMaxSize * 10
	}))
	for category, want := range map[MediaCategory]int{
		CategoryAvatar:      defaultAvatarMaxSize,
		CategoryGroupAvatar: defaultAvatarMaxSize,
		CategoryMomentImage: defaultImageMaxSize,
		CategoryChatImage:   defaultImageMaxSize,
		CategoryChatVideo:   defaultVideoMaxSize,
		CategoryChatAudio:   defaultAudioMaxSize,
		CategoryChatFile:    defaultChatFileMaxSize,
	} {
		if got := svc.maxFileSize(category); got != want {
			t.Fatalf("maxFileSize(%q) = %d, want clamped contract limit %d", category, got, want)
		}
	}

	// 收紧仍然生效。
	svc.SetAppConfig(staticSource(t, func(cfg *appconfig.AppConfig) {
		cfg.Upload.AudioMaxSize = 1024
	}))
	if got := svc.maxFileSize(CategoryChatAudio); got != 1024 {
		t.Fatalf("tightened audio limit = %d, want 1024", got)
	}

	// 越界配置不得让 presign 放行超契约上限的文件（avatar 契约 5MB）。
	raised := NewServiceWithRepoAndProvider(NewInMemoryRepository(), &coverageProvider{
		presignResp: PresignedUpload{UploadID: "upload_oversize", UploadURL: "https://storage.example.com/upload", ExpiresAt: "2026-03-30T00:00:00Z", StorageKey: "uploads/u1/upload_oversize.png"},
	})
	raised.SetAppConfig(staticSource(t, func(cfg *appconfig.AppConfig) {
		cfg.Upload.AvatarMaxSize = defaultAvatarMaxSize * 10
	}))
	_, err := raised.Presign(mediaContextWithUserAndKey("u1", ""), PresignFileInput{
		FileName: "a.png", FileSize: defaultAvatarMaxSize + 1, MimeType: "image/png", Category: CategoryAvatar,
	})
	if !errors.Is(err, ErrInvalidMediaRequest) {
		t.Fatalf("avatar above contract limit must be rejected even if app config raises it, got %v", err)
	}
}

// ── 第 82 轮审计 P2-6：final_file_size = 发布对象实际字节数（清洗后）──────

func TestCompleteImageRecordsPublishedFileSize(t *testing.T) {
	src := coveragePNGWithEXIF()
	repo := NewInMemoryRepository()
	provider := &mediaStubProvider{
		headResp: ObjectMetadata{FileSize: len(src), MimeType: "image/png"},
		// 同一 stub 既承担魔数嗅探（前 512 字节）又承担 EXIF 清洗读取：
		// 用带 eXIf 块的真实 PNG 作为对象内容。
		prefix: src,
	}
	svc := NewServiceWithRepoAndProvider(repo, provider)
	const uploadID = "upload_final_size"
	savePendingSession(t, repo, pendingUpload{
		UploadID: uploadID, UserID: "u1", StorageKey: "uploads/u1/" + uploadID + ".png",
		FileName: "photo.png", DeclaredFileSize: len(src), DeclaredMimeType: "image/png",
		Category: CategoryMomentImage, CreatedAt: time.Now().UTC().Format(time.RFC3339),
	})

	file, err := svc.Complete(mediaContextWithUserAndKey("u1", "key1"), uploadID)
	if err != nil {
		t.Fatalf("Complete returned error: %v", err)
	}
	if provider.putBody == nil {
		t.Fatal("image must be published through the scrubbing path")
	}
	if file.FileSize != len(provider.putBody) {
		t.Fatalf("completed file_size = %d, want published bytes %d", file.FileSize, len(provider.putBody))
	}
	if file.FileSize >= len(src) {
		t.Fatalf("scrubbed image must shrink: file_size=%d original=%d", file.FileSize, len(src))
	}

	stored, found, err := repo.Get(context.Background(), uploadID)
	if err != nil || !found || !stored.Completed {
		t.Fatalf("get completed upload: found=%v err=%v", found, err)
	}
	if stored.File.FileSize != len(provider.putBody) {
		t.Fatalf("persisted final_file_size = %d, want %d", stored.File.FileSize, len(provider.putBody))
	}
	meta, found, err := svc.CompletedFileMetaByID(context.Background(), file.FileID)
	if err != nil || !found {
		t.Fatalf("CompletedFileMetaByID = found=%v err=%v", found, err)
	}
	if meta.FileSize != len(provider.putBody) {
		t.Fatalf("file meta size = %d, want published bytes %d", meta.FileSize, len(provider.putBody))
	}
}

// 释放路径的剩余防御分支：行删除失败时行必须保留（配额至少不会比对象先消失），
// 清理路径的对象删除/行删除失败也只是跳过该条，不得让 presign 变成 panic 或
// 把内部错误当成上传成功。
func TestQuotaReleaseCleanupErrorBranches(t *testing.T) {
	now := time.Now().UTC()
	staleAt := now.Add(-(pendingUploadCleanupGrace + time.Minute))

	t.Run("release keeps row when row delete fails", func(t *testing.T) {
		repo := &mediaRepoOverride{InMemoryRepository: NewInMemoryRepository(), deleteRowErr: errors.New("row boom")}
		provider := &mediaStubProvider{headResp: ObjectMetadata{FileSize: 100, MimeType: "text/plain"}}
		svc := NewServiceWithRepoAndProvider(repo, provider)
		const uploadID = "upload_release_rowfail"
		savePendingSession(t, repo, pendingUpload{
			UploadID: uploadID, UserID: "u1", StorageKey: "uploads/u1/" + uploadID + ".html",
			FileName: "page.html", DeclaredFileSize: 100, DeclaredMimeType: "text/plain",
			Category: CategoryChatFile, CreatedAt: now.Format(time.RFC3339),
		})
		if _, err := svc.Complete(mediaContextWithUserAndKey("u1", "key1"), uploadID); !errors.Is(err, ErrUploadConflict) {
			t.Fatalf("expected ErrUploadConflict, got %v", err)
		}
		if len(provider.deletedKeys) != 1 {
			t.Fatalf("object delete must be attempted first, deleted = %#v", provider.deletedKeys)
		}
		if _, found, _ := repo.Get(context.Background(), uploadID); !found {
			t.Fatal("row must be kept when row delete fails")
		}
	})

	t.Run("cleanup keeps rows when row delete fails", func(t *testing.T) {
		repo := &mediaRepoOverride{InMemoryRepository: NewInMemoryRepository(), deleteRowErr: errors.New("row boom")}
		for i := 0; i < pendingUploadSessionLimit; i++ {
			seedStaleQuotaSession(t, repo, fmt.Sprintf("upload_cleanup_rowfail_%02d", i), "u1", staleAt)
		}
		svc, provider := newQuotaService(repo)
		if _, err := svc.BatchPresign(mediaContextWithUserAndKey("u1", ""), []PresignFileInput{quotaAvatarFile("a.png", 100)}); !errors.Is(err, ErrUploadQuotaExceeded) {
			t.Fatalf("row delete failure must leave quota exhausted, got %v", err)
		}
		if provider.presignCalls != 0 {
			t.Fatalf("rejected presign must not sign URL, calls = %d", provider.presignCalls)
		}
		if len(provider.deletedKeys) != pendingUploadSessionLimit {
			t.Fatalf("objects deleted before row deletion = %d, want %d", len(provider.deletedKeys), pendingUploadSessionLimit)
		}
		for i := 0; i < pendingUploadSessionLimit; i++ {
			uploadID := fmt.Sprintf("upload_cleanup_rowfail_%02d", i)
			if _, found, _ := repo.Get(context.Background(), uploadID); !found {
				t.Fatalf("row %s must be kept when row delete fails", uploadID)
			}
		}
	})

	t.Run("cleanup keeps rows when object delete fails", func(t *testing.T) {
		repo := NewInMemoryRepository()
		for i := 0; i < pendingUploadSessionLimit; i++ {
			seedStaleQuotaSession(t, repo, fmt.Sprintf("upload_cleanup_objfail_%02d", i), "u1", staleAt)
		}
		svc, provider := newQuotaService(repo)
		provider.deleteErr = errors.New("storage temporarily unavailable")
		if _, err := svc.BatchPresign(mediaContextWithUserAndKey("u1", ""), []PresignFileInput{quotaAvatarFile("a.png", 100)}); !errors.Is(err, ErrUploadQuotaExceeded) {
			t.Fatalf("object delete failure must leave quota exhausted, got %v", err)
		}
		if len(provider.deletedKeys) != pendingUploadSessionLimit {
			t.Fatalf("object delete attempts = %d, want %d", len(provider.deletedKeys), pendingUploadSessionLimit)
		}
		for i := 0; i < pendingUploadSessionLimit; i++ {
			uploadID := fmt.Sprintf("upload_cleanup_objfail_%02d", i)
			if _, found, _ := repo.Get(context.Background(), uploadID); !found {
				t.Fatalf("row %s must be kept when object delete fails", uploadID)
			}
		}
	})
}

// quotaUsageFailRepo 让 PendingUploadUsage 第 failAfter+1 次调用起返回错误：
// 配额清理后的「重试一次用量查询」失败必须作为内部错误透传，不能把存储/DB
// 故障误判成配额拒绝或上传成功。
type quotaUsageFailRepo struct {
	*InMemoryRepository
	calls     int
	failAfter int
	err       error
}

func (r *quotaUsageFailRepo) PendingUploadUsage(ctx context.Context, userID string) (PendingUploadUsage, error) {
	r.calls++
	if r.calls > r.failAfter {
		return PendingUploadUsage{}, r.err
	}
	return r.InMemoryRepository.PendingUploadUsage(ctx, userID)
}

func TestPresignQuotaCleanupRetryUsageErrorPropagates(t *testing.T) {
	repoErr := errors.New("usage retry boom")
	repo := &quotaUsageFailRepo{
		InMemoryRepository: NewInMemoryRepository(),
		failAfter:          1,
		err:                repoErr,
	}
	now := time.Now().UTC()
	for i := 0; i < pendingUploadSessionLimit; i++ {
		seedStaleQuotaSession(t, repo, fmt.Sprintf("upload_cleanup_usageerr_%02d", i), "u1", now.Add(-(pendingUploadCleanupGrace + time.Minute)))
	}
	svc, provider := newQuotaService(repo)
	if _, err := svc.BatchPresign(mediaContextWithUserAndKey("u1", ""), []PresignFileInput{quotaAvatarFile("a.png", 100)}); !errors.Is(err, repoErr) {
		t.Fatalf("retry usage query error must propagate, got %v", err)
	}
	if provider.presignCalls != 0 {
		t.Fatalf("failed quota recheck must not presign, calls = %d", provider.presignCalls)
	}
}
