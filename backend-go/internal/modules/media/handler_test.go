package media

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/idempotency"
)

type mediaErrorRepo struct {
	saveErr     error
	getErr      error
	completeErr error
}

func (r *mediaErrorRepo) Save(context.Context, pendingUpload) error { return r.saveErr }
func (r *mediaErrorRepo) Get(context.Context, string) (pendingUpload, bool, error) {
	return pendingUpload{}, false, r.getErr
}
func (r *mediaErrorRepo) Complete(context.Context, string, UploadedMedia, string) (pendingUpload, bool, error) {
	return pendingUpload{}, false, r.completeErr
}
func (r *mediaErrorRepo) ResolveByFileIDs(context.Context, []string, string, MediaCategory) (map[string]MediaAsset, error) {
	return nil, r.getErr
}

type mediaStubProvider struct {
	presignResp PresignedUpload
	presignErr  error
	headResp    ObjectMetadata
	headErr     error
}

func (p *mediaStubProvider) Presign(context.Context, PresignUploadRequest, string) (PresignedUpload, error) {
	return p.presignResp, p.presignErr
}
func (p *mediaStubProvider) PresignGet(context.Context, string, time.Duration) (string, error) {
	return "https://storage.example.com/presigned-get", nil
}
func (p *mediaStubProvider) Head(context.Context, string) (ObjectMetadata, error) {
	return p.headResp, p.headErr
}

func mediaAuthedRequest(method, target, body string) *http.Request {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	ctx := platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "u1"})
	return req.WithContext(ctx)
}

func mediaAuthedCompleteRequest(uploadID, key string) *http.Request {
	req := mediaAuthedRequest(http.MethodPost, "/complete_upload", `{"upload_id":"`+uploadID+`"}`)
	if key != "" {
		req.Header.Set("Idempotency-Key", key)
	}
	return req
}

func TestMediaHandlerPresignValidationAndSuccess(t *testing.T) {
	handler := NewHandler(NewServiceWithRepoAndProvider(NewInMemoryRepository(), &mediaStubProvider{
		presignResp: PresignedUpload{UploadID: "upload_1", UploadURL: "https://storage.example.com/hotspot/uploads/u1/upload_1.png", Headers: map[string]string{"Content-Type": "image/png"}, ExpiresAt: "2026-03-30T00:00:00Z", StorageKey: "uploads/u1/upload_1.png"},
	}))

	badReq := mediaAuthedRequest(http.MethodPost, "/presign_upload", `{"file_name":"","file_size":100,"mime_type":"image/png","category":"avatar"}`)
	badRR := httptest.NewRecorder()
	handler.PresignUpload(badRR, badReq)
	assertMediaCode(t, badRR, httpx.CodeInvalidRequest)

	okReq := mediaAuthedRequest(http.MethodPost, "/presign_upload", `{"file_name":"a.png","file_size":100,"mime_type":"image/png","category":"avatar"}`)
	okRR := httptest.NewRecorder()
	handler.PresignUpload(okRR, okReq)
	var resp struct {
		Code httpx.Code      `json:"code"`
		Data PresignedUpload `json:"data"`
	}
	if err := json.NewDecoder(okRR.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != 0 || resp.Data.UploadID != "upload_1" || resp.Data.StorageKey != "" {
		t.Fatalf("unexpected success response: %#v", resp)
	}
}

func TestMediaHandlerBatchValidationAndSuccess(t *testing.T) {
	handler := NewHandler(NewServiceWithRepoAndProvider(NewInMemoryRepository(), &mediaStubProvider{
		presignResp: PresignedUpload{UploadID: "upload_batch", UploadURL: "https://storage.example.com/hotspot/uploads/u1/upload_batch.png", Headers: map[string]string{"Content-Type": "image/png"}, ExpiresAt: "2026-03-30T00:00:00Z", StorageKey: "uploads/u1/upload_batch.png"},
	}))

	emptyReq := mediaAuthedRequest(http.MethodPost, "/batch_presign_upload", `{"files":[]}`)
	emptyRR := httptest.NewRecorder()
	handler.BatchPresignUpload(emptyRR, emptyReq)
	assertMediaCode(t, emptyRR, httpx.CodeInvalidRequest)

	invalidItemReq := mediaAuthedRequest(http.MethodPost, "/batch_presign_upload", `{"files":[{"file_name":"a.png","file_size":100,"mime_type":"video/mp4","category":"avatar"}]}`)
	invalidItemRR := httptest.NewRecorder()
	handler.BatchPresignUpload(invalidItemRR, invalidItemReq)
	assertMediaCode(t, invalidItemRR, httpx.CodeInvalidRequest)

	okReq := mediaAuthedRequest(http.MethodPost, "/batch_presign_upload", `{"files":[{"file_name":"a.png","file_size":100,"mime_type":"image/png","category":"avatar"}]}`)
	okRR := httptest.NewRecorder()
	handler.BatchPresignUpload(okRR, okReq)
	var resp struct {
		Code httpx.Code           `json:"code"`
		Data BatchPresignedUpload `json:"data"`
	}
	if err := json.NewDecoder(okRR.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != 0 || len(resp.Data.Uploads) != 1 || resp.Data.Uploads[0].UploadID == "" {
		t.Fatalf("unexpected batch response: %#v", resp)
	}
}

func TestMediaHandlerCompleteMapsSpecCodes(t *testing.T) {
	idempotency.SetStore(idempotency.NewInMemoryStore())
	repo := NewInMemoryRepository()
	_ = repo.Save(context.Background(), pendingUpload{UploadID: "upload_ready", UserID: "u1", StorageKey: "uploads/u1/upload_ready.png", FileName: "a.png", DeclaredFileSize: 100, DeclaredMimeType: "image/png", Category: CategoryAvatar})
	handler := NewHandler(NewServiceWithRepoAndProvider(repo, &mediaStubProvider{headResp: ObjectMetadata{FileSize: 100, MimeType: "image/png"}}))

	notFoundReq := mediaAuthedCompleteRequest("upload_missing", "abcd1234abcd1234abcd1234abcd1234")
	notFoundRR := httptest.NewRecorder()
	handler.CompleteUpload(notFoundRR, notFoundReq)
	assertMediaCode(t, notFoundRR, httpx.CodeNotFound)

	notReadyRepo := NewInMemoryRepository()
	_ = notReadyRepo.Save(context.Background(), pendingUpload{UploadID: "upload_not_ready", UserID: "u1", StorageKey: "uploads/u1/upload_not_ready.png", FileName: "a.png", DeclaredFileSize: 100, DeclaredMimeType: "image/png", Category: CategoryAvatar})
	notReadyHandler := NewHandler(NewServiceWithRepoAndProvider(notReadyRepo, &mediaStubProvider{headErr: ErrUploadNotReady}))
	notReadyReq := mediaAuthedCompleteRequest("upload_not_ready", "bbbb1234bbbb1234bbbb1234bbbb1234")
	notReadyRR := httptest.NewRecorder()
	notReadyHandler.CompleteUpload(notReadyRR, notReadyReq)
	assertMediaCode(t, notReadyRR, httpx.CodeConflict)

	foreignReq := mediaAuthedCompleteRequest("upload_ready", "ffff1234ffff1234ffff1234ffff1234")
	foreignCtx := platformauth.WithPrincipal(foreignReq.Context(), platformauth.Principal{UserID: "u2"})
	foreignRR := httptest.NewRecorder()
	handler.CompleteUpload(foreignRR, foreignReq.WithContext(foreignCtx))
	assertMediaCode(t, foreignRR, httpx.CodeNotFound)

	okReq := mediaAuthedCompleteRequest("upload_ready", "cccc1234cccc1234cccc1234cccc1234")
	okRR := httptest.NewRecorder()
	handler.CompleteUpload(okRR, okReq)
	assertMediaCode(t, okRR, 0)

	// 自然幂等（spec api_spec_media.md）：同一 upload_id 重复 complete 返回相同
	// file_id（等价成功 0），即使换用不同 Idempotency-Key。
	replayReq := mediaAuthedCompleteRequest("upload_ready", "dddd1234dddd1234dddd1234dddd1234")
	replayRR := httptest.NewRecorder()
	handler.CompleteUpload(replayRR, replayReq)
	assertMediaCode(t, replayRR, 0)
}

// TestMediaHandlerCompleteSucceedsWithoutIdempotencyKey verifies that
// complete_upload no longer requires an Idempotency-Key header (the endpoint
// is naturally idempotent at the handler level; per-upload conflict detection
// remains in the service layer).
func TestMediaHandlerCompleteSucceedsWithoutIdempotencyKey(t *testing.T) {
	repo := NewInMemoryRepository()
	_ = repo.Save(context.Background(), pendingUpload{UploadID: "upload_naked", UserID: "u1", StorageKey: "uploads/u1/upload_naked.png", FileName: "a.png", DeclaredFileSize: 100, DeclaredMimeType: "image/png", Category: CategoryAvatar})
	handler := NewHandler(NewServiceWithRepoAndProvider(repo, &mediaStubProvider{headResp: ObjectMetadata{FileSize: 100, MimeType: "image/png"}}))

	req := mediaAuthedCompleteRequest("upload_naked", "")
	rr := httptest.NewRecorder()
	handler.CompleteUpload(rr, req)
	assertMediaCode(t, rr, 0)
}

func TestMediaHandlerReturns10600OnServerErrors(t *testing.T) {
	idempotency.SetStore(idempotency.NewInMemoryStore())
	handler := NewHandler(NewServiceWithRepoAndProvider(
		&mediaErrorRepo{saveErr: errors.New("boom"), getErr: errors.New("boom")},
		&mediaStubProvider{presignResp: PresignedUpload{UploadID: "upload_1", UploadURL: "https://storage.example.com/upload_1", Headers: map[string]string{"Content-Type": "image/png"}, ExpiresAt: "2026-03-30T00:00:00Z"}},
	))

	presignReq := mediaAuthedRequest(http.MethodPost, "/presign_upload", `{"file_name":"a.png","file_size":100,"mime_type":"image/png","category":"avatar"}`)
	presignRR := httptest.NewRecorder()
	handler.PresignUpload(presignRR, presignReq)
	assertMediaCode(t, presignRR, httpx.CodeInternalUnavailable)

	completeReq := mediaAuthedCompleteRequest("upload_1", "eeee1234eeee1234eeee1234eeee1234")
	completeRR := httptest.NewRecorder()
	handler.CompleteUpload(completeRR, completeReq)
	assertMediaCode(t, completeRR, httpx.CodeInternalUnavailable)
}

func TestBatchPresignRejectsMoreThan9Files(t *testing.T) {
	handler := NewHandler(NewServiceWithRepoAndProvider(NewInMemoryRepository(), &mediaStubProvider{
		presignResp: PresignedUpload{UploadID: "upload_1", UploadURL: "https://storage.example.com/upload", ExpiresAt: "2026-03-30T00:00:00Z"},
	}))
	files := make([]string, 10)
	for i := 0; i < 10; i++ {
		files[i] = `{"file_name":"a.png","file_size":100,"mime_type":"image/png","category":"avatar"}`
	}
	reqBody := `{"files":[` + strings.Join(files, ",") + `]}`
	req := mediaAuthedRequest(http.MethodPost, "/batch_presign_upload", reqBody)
	rr := httptest.NewRecorder()
	handler.BatchPresignUpload(rr, req)
	assertMediaCode(t, rr, httpx.CodeInvalidRequest)
}

func TestBatchPresignReturns10600OnServiceError(t *testing.T) {
	handler := NewHandler(NewServiceWithRepoAndProvider(NewInMemoryRepository(), &mediaStubProvider{
		presignErr: errors.New("internal provider error"),
	}))
	req := mediaAuthedRequest(http.MethodPost, "/batch_presign_upload", `{"files":[{"file_name":"a.png","file_size":100,"mime_type":"image/png","category":"avatar"}]}`)
	rr := httptest.NewRecorder()
	handler.BatchPresignUpload(rr, req)
	assertMediaCode(t, rr, httpx.CodeInternalUnavailable)
}

func TestPresignRejectsInvalidCategory(t *testing.T) {
	handler := NewHandler(NewServiceWithRepoAndProvider(NewInMemoryRepository(), &mediaStubProvider{
		presignResp: PresignedUpload{UploadID: "upload_1", UploadURL: "https://storage.example.com/upload", ExpiresAt: "2026-03-30T00:00:00Z"},
	}))
	req := mediaAuthedRequest(http.MethodPost, "/presign_upload", `{"file_name":"a.png","file_size":100,"mime_type":"image/png","category":"unknown_category"}`)
	rr := httptest.NewRecorder()
	handler.PresignUpload(rr, req)
	assertMediaCode(t, rr, httpx.CodeInvalidRequest)
}

func assertMediaCode(t *testing.T, rr *httptest.ResponseRecorder, expected httpx.Code) {
	t.Helper()
	var resp struct {
		Code httpx.Code `json:"code"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != expected {
		t.Fatalf("expected code %d, got %#v", expected, resp)
	}
}
