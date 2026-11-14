package media

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/testdb"
)

func newAvatarTestRouter(handler *Handler) http.Handler {
	router := chi.NewRouter()
	router.Get("/media/avatars/{file_id}", handler.ServeAvatar)
	return router
}

func requestAvatar(t *testing.T, router http.Handler, fileID string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/media/avatars/"+fileID, nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}

// seedCompletedAvatar 在内存仓储落一条已完成上传，返回其 file_id（file_<uploadID>）。
func seedCompletedAvatar(t *testing.T, repo *InMemoryRepository, uploadID, userID string, category MediaCategory) string {
	t.Helper()
	ctx := context.Background()
	if err := repo.Save(ctx, pendingUpload{
		UploadID:         uploadID,
		UserID:           userID,
		StorageKey:       "uploads/" + userID + "/" + uploadID + ".png",
		FileName:         uploadID + ".png",
		DeclaredFileSize: 100,
		DeclaredMimeType: "image/png",
		Category:         category,
	}); err != nil {
		t.Fatalf("save %s: %v", uploadID, err)
	}
	fileID := "file_" + uploadID
	if ok, err := repo.Complete(ctx, uploadID, UploadedMedia{FileID: fileID, FileSize: 100, MimeType: "image/png"}, "published/u1/"+uploadID+".png", "idem-"+uploadID); err != nil || !ok {
		t.Fatalf("complete %s: ok=%v err=%v", uploadID, ok, err)
	}
	return fileID
}

func assertAvatarNotFound(t *testing.T, rr *httptest.ResponseRecorder) {
	t.Helper()
	if rr.Code != http.StatusOK {
		t.Fatalf("avatar failures must be HTTP 200 envelope, got %d", rr.Code)
	}
	if code := testdb.DecodeAPICode(t, rr); code != httpx.CodeNotFound {
		t.Fatalf("expected CodeNotFound, got %d", code)
	}
}

func TestServeAvatarRedirectsToSignedURL(t *testing.T) {
	repo := NewInMemoryRepository()
	fileID := seedCompletedAvatar(t, repo, "mediacov_avatar_ok", "u1", CategoryAvatar)
	provider := &coverageProvider{presignGetURL: "https://storage.example.com/hotspot/uploads/u1/mediacov_avatar_ok.png?X-Amz-Signature=abc"}
	handler := NewHandler(NewServiceWithRepoAndProvider(repo, provider))

	rr := requestAvatar(t, newAvatarTestRouter(handler), fileID)
	if rr.Code != http.StatusFound {
		t.Fatalf("expected 302 redirect, got %d", rr.Code)
	}
	if location := rr.Header().Get("Location"); location != provider.presignGetURL {
		t.Fatalf("Location = %q, want %q", location, provider.presignGetURL)
	}

	// group_avatar 与 avatar 同为公开资料类，同样放行。
	groupAvatarID := seedCompletedAvatar(t, repo, "mediacov_avatar_group", "u1", CategoryGroupAvatar)
	if rr := requestAvatar(t, newAvatarTestRouter(handler), groupAvatarID); rr.Code != http.StatusFound {
		t.Fatalf("group avatar must redirect, got %d", rr.Code)
	}
}

func TestServeAvatarHidesUnavailableFiles(t *testing.T) {
	repo := NewInMemoryRepository()
	handler := NewHandler(NewServiceWithRepoAndProvider(repo, &coverageProvider{presignGetURL: "https://storage.example.com/signed"}))
	router := newAvatarTestRouter(handler)

	// 不存在。
	assertAvatarNotFound(t, requestAvatar(t, router, "file_mediacov_absent"))
	// 非头像类别（瞬间图不得经匿名代理读取）。
	nonAvatarID := seedCompletedAvatar(t, repo, "mediacov_avatar_moment", "u1", CategoryMomentImage)
	assertAvatarNotFound(t, requestAvatar(t, router, nonAvatarID))
	// 未完成上传不对外。
	if err := repo.Save(context.Background(), pendingUpload{UploadID: "mediacov_avatar_pending", UserID: "u1", StorageKey: "uploads/u1/pending.png", FileName: "pending.png", DeclaredFileSize: 100, DeclaredMimeType: "image/png", Category: CategoryAvatar}); err != nil {
		t.Fatalf("save pending avatar: %v", err)
	}
	assertAvatarNotFound(t, requestAvatar(t, router, "file_mediacov_avatar_pending"))
}

func TestServeAvatarMapsRepositoryError(t *testing.T) {
	repo := &mediaRepoOverride{InMemoryRepository: NewInMemoryRepository(), metaErr: errors.New("db down")}
	handler := NewHandler(NewServiceWithRepoAndProvider(repo, &coverageProvider{presignGetURL: "https://storage.example.com/signed"}))
	assertAvatarNotFound(t, requestAvatar(t, newAvatarTestRouter(handler), "file_mediacov_dberror"))
}

func TestServeAvatarRejectsUnsafeReadURL(t *testing.T) {
	repo := NewInMemoryRepository()
	fileID := seedCompletedAvatar(t, repo, "mediacov_avatar_unsafe", "u1", CategoryAvatar)

	for _, readURL := range []string{"javascript:alert(1)", "https:///missing-host", "not-a-url"} {
		handler := NewHandler(NewServiceWithRepoAndProvider(repo, &coverageProvider{presignGetURL: readURL}))
		assertAvatarNotFound(t, requestAvatar(t, newAvatarTestRouter(handler), fileID))
	}

	// 空读 URL（provider 无错误但未签名）同样按 404 处理。
	handler := NewHandler(NewServiceWithRepoAndProvider(repo, &coverageProvider{}))
	assertAvatarNotFound(t, requestAvatar(t, newAvatarTestRouter(handler), fileID))
}
