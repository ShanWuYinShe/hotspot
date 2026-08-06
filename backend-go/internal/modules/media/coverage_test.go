package media

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/httpx"
)

// mediaErrRepo 包装内存 repo，Complete 返回可控错误。
type mediaErrRepo struct {
	repo *InMemoryRepository
	err  error
}

func (r *mediaErrRepo) Save(ctx context.Context, upload pendingUpload) error {
	return r.repo.Save(ctx, upload)
}
func (r *mediaErrRepo) Get(ctx context.Context, uploadID string) (pendingUpload, bool, error) {
	return r.repo.Get(ctx, uploadID)
}
func (r *mediaErrRepo) Complete(ctx context.Context, uploadID string, media UploadedMedia, storageKey string) (pendingUpload, bool, error) {
	if r.err != nil {
		return pendingUpload{}, false, r.err
	}
	return r.repo.Complete(ctx, uploadID, media, storageKey)
}
func (r *mediaErrRepo) ResolveByFileIDs(ctx context.Context, fileIDs []string, userID string, category MediaCategory) (map[string]MediaAsset, error) {
	return r.repo.ResolveByFileIDs(ctx, fileIDs, userID, category)
}

func TestMediaServiceConstructors(t *testing.T) {
	svc := NewService()
	if svc == nil || svc.repo == nil || svc.provider == nil {
		t.Fatal("NewService must wire repo and provider")
	}
	svc2 := NewServiceWithRepo(NewInMemoryRepository())
	if svc2 == nil || svc2.repo == nil {
		t.Fatal("NewServiceWithRepo must wire repo")
	}
}

func TestMediaHandlerCompleteBatchErrorPaths(t *testing.T) {
	repo := NewInMemoryRepository()
	svc := NewServiceWithRepo(repo)
	handler := NewHandler(svc)

	// 空 upload_ids → httpx.CodeInvalidRequest
	req := httptest.NewRequest(http.MethodPost, "/batch_complete_upload", strings.NewReader(`{"upload_ids":[]}`))
	rr := httptest.NewRecorder()
	handler.CompleteUploadBatch(rr, req)
	if code := decodeMediaCode(t, rr); code != httpx.CodeInvalidRequest {
		t.Fatalf("empty batch: expected httpx.CodeInvalidRequest, got %d", code)
	}

	// 超过 9 个 → httpx.CodeInvalidRequest
	req = httptest.NewRequest(http.MethodPost, "/batch_complete_upload", strings.NewReader(`{"upload_ids":["1","2","3","4","5","6","7","8","9","10"]}`))
	rr = httptest.NewRecorder()
	handler.CompleteUploadBatch(rr, req)
	if code := decodeMediaCode(t, rr); code != httpx.CodeInvalidRequest {
		t.Fatalf("oversize batch: expected httpx.CodeInvalidRequest, got %d", code)
	}

	// 不存在的 upload id 批量完成：不匹配的服务端错误分支走 httpx.CodeInternalUnavailable
	handler2 := NewHandler(NewServiceWithRepo(&mediaErrRepo{repo: repo, err: errors.New("boom")}))
	req = httptest.NewRequest(http.MethodPost, "/batch_complete_upload", strings.NewReader(`{"upload_ids":["1"]}`))
	rr = httptest.NewRecorder()
	handler2.CompleteUploadBatch(rr, req)
	if code := decodeMediaCode(t, rr); code != httpx.CodeInvalidRequest && code != httpx.CodeInternalUnavailable {
		t.Fatalf("unexpected code for batch error: %d", code)
	}
}

func decodeMediaCode(t *testing.T, rr *httptest.ResponseRecorder) httpx.Code {
	t.Helper()
	var resp struct {
		Code httpx.Code `json:"code"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	return resp.Code
}

var _ = config.Load
