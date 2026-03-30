package media

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/config"
)

func TestObjectStorageProviderPresignBuildsSignedPutURL(t *testing.T) {
	provider := NewObjectStorageProvider(config.Config{
		S3Endpoint:       "https://s3.example.com",
		S3Region:         "us-east-1",
		S3Bucket:         "hotspot",
		S3AccessKey:      "access-key",
		S3SecretKey:      "secret-key",
		S3ForcePathStyle: true,
	})
	provider.now = func() time.Time {
		return time.Date(2026, 3, 30, 8, 0, 0, 0, time.UTC)
	}

	resp, err := provider.Presign(context.Background(), PresignUploadRequest{FileName: "a.png", MimeType: "image/png", MD5: "d41d8cd98f00b204e9800998ecf8427e"}, "u1")
	if err != nil {
		t.Fatalf("Presign returned error: %v", err)
	}
	if resp.UploadID == "" || resp.StorageKey != "uploads/u1/"+resp.UploadID+".png" {
		t.Fatalf("unexpected upload identity: %#v", resp)
	}
	if !strings.HasPrefix(resp.UploadURL, "https://s3.example.com/hotspot/uploads/u1/") {
		t.Fatalf("unexpected upload url: %s", resp.UploadURL)
	}
	for _, key := range []string{"X-Amz-Algorithm=AWS4-HMAC-SHA256", "X-Amz-Credential=access-key%2F20260330%2Fus-east-1%2Fs3%2Faws4_request", "X-Amz-Date=20260330T080000Z", "X-Amz-Expires=900", "X-Amz-SignedHeaders=host", "X-Amz-Signature="} {
		if !strings.Contains(resp.UploadURL, key) {
			t.Fatalf("expected %q in presigned url: %s", key, resp.UploadURL)
		}
	}
	if resp.Headers["Content-Type"] != "image/png" {
		t.Fatalf("unexpected content type header: %#v", resp.Headers)
	}
	if resp.Headers["Content-MD5"] != "d41d8cd98f00b204e9800998ecf8427e" {
		t.Fatalf("unexpected content md5 header: %#v", resp.Headers)
	}
	if resp.ExpiresAt != "2026-03-30T08:15:00Z" {
		t.Fatalf("unexpected expires at: %s", resp.ExpiresAt)
	}
}

func TestObjectStorageProviderHeadMapsObjectMetadata(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodHead {
			t.Fatalf("expected HEAD request, got %s", r.Method)
		}
		if r.URL.Path != "/hotspot/uploads/u1/upload_1.png" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Length", "123")
		w.Header().Set("Content-Type", "image/png")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	provider := NewObjectStorageProvider(config.Config{
		S3Endpoint:       server.URL,
		S3Bucket:         "hotspot",
		S3ForcePathStyle: true,
	})
	meta, err := provider.Head(context.Background(), "uploads/u1/upload_1.png")
	if err != nil {
		t.Fatalf("Head returned error: %v", err)
	}
	if meta.FileSize != 123 || meta.MimeType != "image/png" {
		t.Fatalf("unexpected metadata: %#v", meta)
	}
}

func TestObjectStorageProviderHeadReturnsNotReadyOn404(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	provider := NewObjectStorageProvider(config.Config{
		S3Endpoint:       server.URL,
		S3Bucket:         "hotspot",
		S3ForcePathStyle: true,
	})
	if _, err := provider.Head(context.Background(), "uploads/u1/upload_1.png"); err != ErrUploadNotReady {
		t.Fatalf("expected ErrUploadNotReady, got %v", err)
	}
}
