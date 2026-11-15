package media

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"hotspot/backend-go/internal/platform/httpx"
)

// 预签名配额（第 81 轮审计残留：预签名 PUT 的 body 大小不受签名约束，弃置
// 未完成会话会放大存储/带宽）：BatchPresign 在签发上传 URL 前按用户聚合
// 「未完成会话数 + 声明字节总量」，超限整批拒绝。阈值取舍见 service.go 的
// checkPresignQuota 注释。

// quotaStubProvider 记录 Presign 调用次数，供「拒绝时不得签发 URL」断言。
type quotaStubProvider struct {
	mediaStubProvider
	presignCalls int
}

func (p *quotaStubProvider) Presign(ctx context.Context, req PresignFileInput, userID string) (PresignedUpload, error) {
	p.presignCalls++
	return p.mediaStubProvider.Presign(ctx, req, userID)
}

// seedQuotaUploads 直接向仓储写入 n 条未完成会话（声明 size 字节），不经过
// service 校验——配额聚合只关心行数与声明字节，与写入来源无关。
func seedQuotaUploads(t *testing.T, repo *InMemoryRepository, userID, prefix string, n, size int) {
	t.Helper()
	ctx := context.Background()
	for i := 0; i < n; i++ {
		uploadID := fmt.Sprintf("%s_%d", prefix, i)
		if err := repo.Save(ctx, pendingUpload{
			UploadID:         uploadID,
			UserID:           userID,
			StorageKey:       "uploads/" + userID + "/" + uploadID + ".png",
			FileName:         uploadID + ".png",
			DeclaredFileSize: size,
			DeclaredMimeType: "image/png",
			Category:         CategoryAvatar,
		}); err != nil {
			t.Fatalf("seed %s: %v", uploadID, err)
		}
	}
}

func quotaAvatarFile(name string, size int) PresignFileInput {
	return PresignFileInput{FileName: name, FileSize: size, MimeType: "image/png", Category: CategoryAvatar}
}

func newQuotaService(repo Repository) (*Service, *quotaStubProvider) {
	provider := &quotaStubProvider{mediaStubProvider: mediaStubProvider{
		presignResp: PresignedUpload{
			UploadID:   "upload_quota_new",
			UploadURL:  "https://storage.example.com/upload",
			Headers:    map[string]string{"Content-Type": "image/png"},
			ExpiresAt:  "2026-03-30T00:00:00Z",
			StorageKey: "uploads/u1/upload_quota_new.png",
		},
	}}
	return NewServiceWithRepoAndProvider(repo, provider), provider
}

func repoUsage(t *testing.T, repo *InMemoryRepository, userID string) PendingUploadUsage {
	t.Helper()
	usage, err := repo.PendingUploadUsage(context.Background(), userID)
	if err != nil {
		t.Fatalf("pending usage: %v", err)
	}
	return usage
}

// TestBatchPresignQuotaCountLimit 覆盖会话数边界：恰好到上限放行，超 1 条拒绝
// （批量本身计入配额，「先签满再追加」不可绕过）。
func TestBatchPresignQuotaCountLimit(t *testing.T) {
	ctx := mediaContextWithUserAndKey("u1", "")

	t.Run("at limit passes", func(t *testing.T) {
		repo := NewInMemoryRepository()
		seedQuotaUploads(t, repo, "u1", "upload_pending", pendingUploadSessionLimit-1, 100)
		svc, provider := newQuotaService(repo)
		if _, err := svc.BatchPresign(ctx, []PresignFileInput{quotaAvatarFile("a.png", 100)}); err != nil {
			t.Fatalf("at limit must pass, got %v", err)
		}
		if provider.presignCalls != 1 {
			t.Fatalf("presign calls = %d, want 1", provider.presignCalls)
		}
		if got := repoUsage(t, repo, "u1"); got.Count != pendingUploadSessionLimit {
			t.Fatalf("pending count = %d, want %d", got.Count, pendingUploadSessionLimit)
		}
	})

	t.Run("one over limit rejected without side effects", func(t *testing.T) {
		repo := NewInMemoryRepository()
		seedQuotaUploads(t, repo, "u1", "upload_pending", pendingUploadSessionLimit, 100)
		svc, provider := newQuotaService(repo)
		before := repoUsage(t, repo, "u1")
		if _, err := svc.BatchPresign(ctx, []PresignFileInput{quotaAvatarFile("a.png", 100)}); !errors.Is(err, ErrUploadQuotaExceeded) {
			t.Fatalf("expected ErrUploadQuotaExceeded, got %v", err)
		}
		if provider.presignCalls != 0 {
			t.Fatalf("rejected batch must not presign, calls = %d", provider.presignCalls)
		}
		if after := repoUsage(t, repo, "u1"); after != before {
			t.Fatalf("rejected batch must not save sessions: before=%+v after=%+v", before, after)
		}
	})

	t.Run("batch would exceed limit rejected as a whole", func(t *testing.T) {
		repo := NewInMemoryRepository()
		seedQuotaUploads(t, repo, "u1", "upload_pending", pendingUploadSessionLimit-1, 100)
		svc, provider := newQuotaService(repo)
		files := []PresignFileInput{quotaAvatarFile("a.png", 100), quotaAvatarFile("b.png", 100)}
		if _, err := svc.BatchPresign(ctx, files); !errors.Is(err, ErrUploadQuotaExceeded) {
			t.Fatalf("expected ErrUploadQuotaExceeded, got %v", err)
		}
		if provider.presignCalls != 0 {
			t.Fatalf("whole batch must be rejected before presign, calls = %d", provider.presignCalls)
		}
	})

	t.Run("completed sessions do not occupy quota", func(t *testing.T) {
		repo := NewInMemoryRepository()
		seedQuotaUploads(t, repo, "u1", "upload_done", pendingUploadSessionLimit, 100)
		for i := 0; i < pendingUploadSessionLimit; i++ {
			uploadID := fmt.Sprintf("upload_done_%d", i)
			if ok, err := repo.Complete(context.Background(), uploadID, UploadedMedia{FileID: "file_" + uploadID, FileSize: 100, MimeType: "image/png"}, "published/u1/"+uploadID+".png", "idem"); err != nil || !ok {
				t.Fatalf("complete %s: ok=%v err=%v", uploadID, ok, err)
			}
		}
		svc, _ := newQuotaService(repo)
		if _, err := svc.BatchPresign(ctx, []PresignFileInput{quotaAvatarFile("a.png", 100)}); err != nil {
			t.Fatalf("completed sessions must not count, got %v", err)
		}
	})

	t.Run("other users pending sessions do not count", func(t *testing.T) {
		repo := NewInMemoryRepository()
		seedQuotaUploads(t, repo, "u2", "upload_other", pendingUploadSessionLimit, 100)
		svc, _ := newQuotaService(repo)
		if _, err := svc.BatchPresign(ctx, []PresignFileInput{quotaAvatarFile("a.png", 100)}); err != nil {
			t.Fatalf("other owner sessions must not count, got %v", err)
		}
	})
}

// TestBatchPresignQuotaBytesLimit 覆盖声明字节总量边界：等于上限放行、超 1
// 字节即拒绝；已完成的会话声明字节同样不计入。
func TestBatchPresignQuotaBytesLimit(t *testing.T) {
	ctx := mediaContextWithUserAndKey("u1", "")

	t.Run("at byte limit passes", func(t *testing.T) {
		repo := NewInMemoryRepository()
		seedQuotaUploads(t, repo, "u1", "upload_bytes", 1, pendingUploadBytesLimit-100)
		svc, _ := newQuotaService(repo)
		if _, err := svc.BatchPresign(ctx, []PresignFileInput{quotaAvatarFile("a.png", 100)}); err != nil {
			t.Fatalf("exactly at byte limit must pass, got %v", err)
		}
	})

	t.Run("one byte over limit rejected", func(t *testing.T) {
		repo := NewInMemoryRepository()
		seedQuotaUploads(t, repo, "u1", "upload_bytes", 1, pendingUploadBytesLimit-100)
		svc, provider := newQuotaService(repo)
		if _, err := svc.BatchPresign(ctx, []PresignFileInput{quotaAvatarFile("a.png", 101)}); !errors.Is(err, ErrUploadQuotaExceeded) {
			t.Fatalf("expected ErrUploadQuotaExceeded, got %v", err)
		}
		if provider.presignCalls != 0 {
			t.Fatalf("rejected batch must not presign, calls = %d", provider.presignCalls)
		}
	})

	t.Run("single large declared bytes session consumes quota", func(t *testing.T) {
		repo := NewInMemoryRepository()
		seedQuotaUploads(t, repo, "u1", "upload_bytes", 1, pendingUploadBytesLimit)
		svc, _ := newQuotaService(repo)
		if _, err := svc.BatchPresign(ctx, []PresignFileInput{quotaAvatarFile("a.png", 1)}); !errors.Is(err, ErrUploadQuotaExceeded) {
			t.Fatalf("expected ErrUploadQuotaExceeded, got %v", err)
		}
	})

	t.Run("completed declared bytes do not count", func(t *testing.T) {
		repo := NewInMemoryRepository()
		seedQuotaUploads(t, repo, "u1", "upload_bytes_done", 1, pendingUploadBytesLimit-50)
		uploadID := "upload_bytes_done_0"
		if ok, err := repo.Complete(context.Background(), uploadID, UploadedMedia{FileID: "file_" + uploadID, FileSize: 100, MimeType: "image/png"}, "published/u1/"+uploadID+".png", "idem"); err != nil || !ok {
			t.Fatalf("complete: ok=%v err=%v", ok, err)
		}
		svc, _ := newQuotaService(repo)
		if _, err := svc.BatchPresign(ctx, []PresignFileInput{quotaAvatarFile("a.png", 100)}); err != nil {
			t.Fatalf("completed declared bytes must not count, got %v", err)
		}
	})
}

// 单文件 Presign 入口走同一配额闸门（两个服务入口的用户级不变量保持一致）。
func TestPresignQuotaAppliesToSingleUpload(t *testing.T) {
	repo := NewInMemoryRepository()
	seedQuotaUploads(t, repo, "u1", "upload_single", pendingUploadSessionLimit, 100)
	svc, provider := newQuotaService(repo)
	if _, err := svc.Presign(mediaContextWithUserAndKey("u1", ""), quotaAvatarFile("a.png", 100)); !errors.Is(err, ErrUploadQuotaExceeded) {
		t.Fatalf("expected ErrUploadQuotaExceeded, got %v", err)
	}
	if provider.presignCalls != 0 {
		t.Fatalf("rejected presign must not sign URL, calls = %d", provider.presignCalls)
	}
}

// 聚合查询失败透传为内部错误（handler 映射 10600），不能被误判成配额拒绝。
func TestPresignQuotaRepositoryErrorPropagates(t *testing.T) {
	repoErr := errors.New("usage boom")
	repo := &mediaRepoOverride{InMemoryRepository: NewInMemoryRepository(), usageErr: repoErr}
	svc, provider := newQuotaService(repo)
	if _, err := svc.BatchPresign(mediaContextWithUserAndKey("u1", ""), []PresignFileInput{quotaAvatarFile("a.png", 100)}); !errors.Is(err, repoErr) {
		t.Fatalf("BatchPresign = %v, want %v", err, repoErr)
	}
	if _, err := svc.Presign(mediaContextWithUserAndKey("u1", ""), quotaAvatarFile("a.png", 100)); !errors.Is(err, repoErr) {
		t.Fatalf("Presign = %v, want %v", err, repoErr)
	}
	if provider.presignCalls != 0 {
		t.Fatalf("repository failure must stop presign, calls = %d", provider.presignCalls)
	}
}

// 配额拒绝映射 spec 已定义的 10400 业务冲突（不得自创错误码）：客户端按
// message 提示用户完成已有上传，不做无意义重试。
func TestMediaHandlerPresignQuotaMapsConflict(t *testing.T) {
	repo := NewInMemoryRepository()
	seedQuotaUploads(t, repo, "u1", "upload_quota_handler", pendingUploadSessionLimit, 100)
	handler := NewHandler(NewServiceWithRepoAndProvider(repo, &mediaStubProvider{
		presignResp: PresignedUpload{UploadID: "upload_quota_handler_new", UploadURL: "https://storage.example.com/upload", ExpiresAt: "2026-03-30T00:00:00Z"},
	}))
	req := mediaAuthedRequest(http.MethodPost, "/presign_upload", `{"files":[{"file_name":"a.png","file_size":100,"mime_type":"image/png","category":"avatar"}]}`)
	rr := httptest.NewRecorder()
	handler.PresignUpload(rr, req)
	assertMediaCode(t, rr, httpx.CodeConflict)
}

// 配额聚合查询失败属内部错误：handler 必须返回 10600（WriteAPIError 默认
// 分支），不得误映射成 10400/10000 的业务拒绝——否则客户端会把存储故障当成
// 「上传配额已满」而放弃上传。
func TestMediaHandlerPresignQuotaRepositoryErrorIsInternal(t *testing.T) {
	handler := NewHandler(NewServiceWithRepoAndProvider(
		&mediaErrorRepo{pendingUsageErr: errors.New("usage boom")},
		&mediaStubProvider{presignResp: PresignedUpload{UploadID: "upload_quota_err", UploadURL: "https://storage.example.com/upload", ExpiresAt: "2026-03-30T00:00:00Z"}},
	))
	req := mediaAuthedRequest(http.MethodPost, "/presign_upload", "{\"files\":[{\"file_name\":\"a.png\",\"file_size\":100,\"mime_type\":\"image/png\",\"category\":\"avatar\"}]}")
	rr := httptest.NewRecorder()
	handler.PresignUpload(rr, req)
	assertMediaCode(t, rr, httpx.CodeInternalUnavailable)
}
