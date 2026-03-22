package media

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"hotspot/backend-go/internal/platform/idempotency"
)

type mediaErrorRepo struct {
	saveErr     error
	getErr      error
	completeErr error
}

func (r *mediaErrorRepo) Save(context.Context, string, pendingUpload) error {
	return r.saveErr
}

func (r *mediaErrorRepo) Get(context.Context, string) (pendingUpload, bool, error) {
	return pendingUpload{}, false, r.getErr
}

func (r *mediaErrorRepo) Complete(context.Context, string, UploadedMedia) (pendingUpload, bool, error) {
	return pendingUpload{}, false, r.completeErr
}

type mediaCompleteErrorRepo struct {
	upload pendingUpload
	err    error
}

func (r *mediaCompleteErrorRepo) Save(context.Context, string, pendingUpload) error {
	return nil
}

func (r *mediaCompleteErrorRepo) Get(context.Context, string) (pendingUpload, bool, error) {
	return r.upload, true, nil
}

func (r *mediaCompleteErrorRepo) Complete(context.Context, string, UploadedMedia) (pendingUpload, bool, error) {
	return pendingUpload{}, false, r.err
}

type mediaStubProvider struct {
	presignResp  PresignedUpload
	presignErr   error
	completeResp UploadedMedia
	completeErr  error
}

func (p *mediaStubProvider) Presign(context.Context, PresignUploadRequest) (PresignedUpload, error) {
	return p.presignResp, p.presignErr
}

func (p *mediaStubProvider) Complete(context.Context, string, PresignUploadRequest) (UploadedMedia, error) {
	return p.completeResp, p.completeErr
}

func TestPresignAndCompleteUpload(t *testing.T) {
	idempotency.SetStore(idempotency.NewInMemoryStore())
	handler := NewHandler(NewService())
	presignReq := httptest.NewRequest(http.MethodPost, "/presign_upload", strings.NewReader(`{"file_name":"a.png","file_size":100,"mime_type":"image/png","category":"avatar"}`))
	presignRR := httptest.NewRecorder()
	handler.PresignUpload(presignRR, presignReq)

	var presignResp struct {
		Code int             `json:"code"`
		Data PresignedUpload `json:"data"`
	}
	if err := json.NewDecoder(presignRR.Body).Decode(&presignResp); err != nil {
		t.Fatalf("decode presign response: %v", err)
	}
	if presignResp.Code != 0 || presignResp.Data.UploadID == "" {
		t.Fatalf("unexpected presign response: %#v", presignResp)
	}

	completeReq := httptest.NewRequest(http.MethodPost, "/complete_upload", strings.NewReader(`{"upload_id":"`+presignResp.Data.UploadID+`"}`))
	completeReq.Header.Set("Idempotency-Key", "abcd1234abcd1234abcd1234abcd1234")
	completeRR := httptest.NewRecorder()
	handler.CompleteUpload(completeRR, completeReq)
	completeReq2 := httptest.NewRequest(http.MethodPost, "/complete_upload", strings.NewReader(`{"upload_id":"`+presignResp.Data.UploadID+`"}`))
	completeReq2.Header.Set("Idempotency-Key", "abcd1234abcd1234abcd1234abcd1234")
	completeRR2 := httptest.NewRecorder()
	handler.CompleteUpload(completeRR2, completeReq2)

	var completeResp struct {
		Code int           `json:"code"`
		Data UploadedMedia `json:"data"`
	}
	if err := json.NewDecoder(completeRR.Body).Decode(&completeResp); err != nil {
		t.Fatalf("decode complete response: %v", err)
	}
	var completeResp2 struct {
		Code int           `json:"code"`
		Data UploadedMedia `json:"data"`
	}
	_ = json.NewDecoder(completeRR2.Body).Decode(&completeResp2)
	if completeResp.Code != 0 || completeResp.Data.FileID == "" || completeResp.Data.FileID != completeResp2.Data.FileID {
		t.Fatalf("unexpected complete response: %#v %#v", completeResp, completeResp2)
	}
}

func TestBatchPresignUpload(t *testing.T) {
	handler := NewHandler(NewService())
	req := httptest.NewRequest(http.MethodPost, "/batch_presign_upload", strings.NewReader(`{"files":[{"file_name":"a.png","file_size":100,"mime_type":"image/png","category":"avatar"},{"file_name":"b.png","file_size":200,"mime_type":"image/png","category":"chat_image"}]}`))
	rr := httptest.NewRecorder()
	handler.BatchPresignUpload(rr, req)

	var resp struct {
		Code int                  `json:"code"`
		Data BatchPresignedUpload `json:"data"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode batch presign response: %v", err)
	}
	if resp.Code != 0 || len(resp.Data.Uploads) != 2 || resp.Data.Uploads[0].UploadID == "" || resp.Data.Uploads[1].UploadID == "" {
		t.Fatalf("unexpected batch response: %#v", resp)
	}
}

func TestMediaHandlerReturns10600OnServiceError(t *testing.T) {
	idempotency.SetStore(idempotency.NewInMemoryStore())
	handler := NewHandler(NewServiceWithRepoAndProvider(
		&mediaErrorRepo{saveErr: errors.New("boom"), getErr: errors.New("boom")},
		&mediaStubProvider{presignResp: PresignedUpload{UploadID: "upload_1", UploadURL: "https://upload.example.com/upload_1", Headers: map[string]string{"Content-Type": "image/png"}, ExpiresAt: "2026-03-22T00:00:00Z"}},
	))

	presignReq := httptest.NewRequest(http.MethodPost, "/presign_upload", strings.NewReader(`{"file_name":"a.png","file_size":100,"mime_type":"image/png","category":"avatar"}`))
	presignRR := httptest.NewRecorder()
	handler.PresignUpload(presignRR, presignReq)
	assertMediaCode(t, presignRR, 10600)

	batchReq := httptest.NewRequest(http.MethodPost, "/batch_presign_upload", strings.NewReader(`{"files":[{"file_name":"a.png","file_size":100,"mime_type":"image/png","category":"avatar"}]}`))
	batchRR := httptest.NewRecorder()
	handler.BatchPresignUpload(batchRR, batchReq)
	assertMediaCode(t, batchRR, 10600)

	completeReq := httptest.NewRequest(http.MethodPost, "/complete_upload", strings.NewReader(`{"upload_id":"upload_1"}`))
	completeReq.Header.Set("Idempotency-Key", "ijkl1234ijkl1234ijkl1234ijkl1234")
	completeRR := httptest.NewRecorder()
	handler.CompleteUpload(completeRR, completeReq)
	assertMediaCode(t, completeRR, 10600)
}

func TestServicePropagatesProviderAndRepositoryErrors(t *testing.T) {
	repo := NewInMemoryRepository()
	provider := &mediaStubProvider{
		presignResp:  PresignedUpload{UploadID: "upload_ok", UploadURL: "https://upload.example.com/upload_ok", Headers: map[string]string{"Content-Type": "image/png"}, ExpiresAt: "2026-03-22T00:00:00Z"},
		completeResp: UploadedMedia{FileID: "file_upload_ok", FileSize: 100, MimeType: "image/png"},
	}
	service := NewServiceWithRepoAndProvider(repo, provider)

	if _, err := NewServiceWithRepoAndProvider(NewInMemoryRepository(), &mediaStubProvider{presignErr: errors.New("presign boom")}).Presign(context.Background(), PresignUploadRequest{FileName: "a.png", FileSize: 100, MimeType: "image/png", Category: CategoryAvatar}); err == nil {
		t.Fatal("expected presign provider error")
	}

	if _, err := NewServiceWithRepoAndProvider(&mediaErrorRepo{saveErr: errors.New("save boom")}, provider).Presign(context.Background(), PresignUploadRequest{FileName: "a.png", FileSize: 100, MimeType: "image/png", Category: CategoryAvatar}); err == nil {
		t.Fatal("expected presign repository error")
	}

	if _, err := service.Presign(context.Background(), PresignUploadRequest{FileName: "a.png", FileSize: 100, MimeType: "image/png", Category: CategoryAvatar}); err != nil {
		t.Fatalf("presign upload: %v", err)
	}

	if _, _, err := NewServiceWithRepoAndProvider(repo, &mediaStubProvider{completeErr: errors.New("complete boom")}).Complete(context.Background(), "upload_ok"); err == nil {
		t.Fatal("expected complete provider error")
	}

	if _, _, err := NewServiceWithRepoAndProvider(&mediaErrorRepo{getErr: errors.New("get boom")}, provider).Complete(context.Background(), "upload_ok"); err == nil {
		t.Fatal("expected get repository error")
	}

	if _, _, err := NewServiceWithRepoAndProvider(&mediaCompleteErrorRepo{
		upload: pendingUpload{Request: PresignUploadRequest{FileName: "a.png", FileSize: 100, MimeType: "image/png", Category: CategoryAvatar}},
		err:    errors.New("complete persist boom"),
	}, provider).Complete(context.Background(), "upload_2"); err == nil {
		t.Fatal("expected complete repository error")
	}
}

func assertMediaCode(t *testing.T, rr *httptest.ResponseRecorder, expected int) {
	t.Helper()
	var resp struct {
		Code int `json:"code"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != expected {
		t.Fatalf("expected code %d, got %#v", expected, resp)
	}
}
