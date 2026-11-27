package media

// uploaded_media_json_test.go — complete_upload 响应项的 JSON 形状契约
//（spec api_spec_media.md UploadedMedia）：thumbnail_url/duration/width/height
// 为恒定 nullable，无值时必须显式下发 null 而不是缺键。

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
)

func TestCompleteUploadResponseAlwaysCarriesNullableMediaKeys(t *testing.T) {
	repo := NewInMemoryRepository()
	if err := repo.Save(context.Background(), pendingUpload{
		UploadID:         "upload_json_shape",
		UserID:           "u1",
		StorageKey:       "uploads/u1/upload_json_shape.png",
		FileName:         "a.png",
		DeclaredFileSize: 100,
		DeclaredMimeType: "image/png",
		Category:         CategoryAvatar,
	}); err != nil {
		t.Fatalf("save upload: %v", err)
	}
	handler := NewHandler(NewServiceWithRepoAndProvider(repo, &mediaStubProvider{
		headResp: ObjectMetadata{FileSize: 100, MimeType: "image/png"},
		prefix:   magicPNG,
	}))
	req := mediaAuthedCompleteRequest("upload_json_shape", "ffff1234ffff1234ffff1234ffff1234")
	rr := httptest.NewRecorder()
	handler.CompleteUpload(rr, req)
	// assertMediaCode 会用 json.Decoder 消费响应体，先留存原始 JSON 供键集合断言。
	body := append([]byte(nil), rr.Body.Bytes()...)
	assertMediaCode(t, rr, 0)

	var resp struct {
		Code int `json:"code"`
		Data struct {
			Uploads []map[string]json.RawMessage `json:"uploads"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("decode response: %v (%s)", err, string(body))
	}
	if len(resp.Data.Uploads) != 1 {
		t.Fatalf("want 1 upload, got %d (%s)", len(resp.Data.Uploads), string(body))
	}
	upload := resp.Data.Uploads[0]
	if _, ok := upload["file_id"]; !ok {
		t.Fatalf("file_id missing: %s", string(body))
	}
	if string(upload["mime_type"]) != "\"image/png\"" {
		t.Fatalf("unexpected mime_type: %s", string(body))
	}
	for _, key := range []string{"thumbnail_url", "duration", "width", "height"} {
		raw, ok := upload[key]
		if !ok {
			t.Fatalf("nullable key %q must always be present: %s", key, string(body))
		}
		if string(raw) != "null" {
			t.Fatalf("nullable key %q must be null when unset, got %s", key, raw)
		}
	}
}
