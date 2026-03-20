package media

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"hotspot/backend-go/internal/platform/idempotency"
)

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
