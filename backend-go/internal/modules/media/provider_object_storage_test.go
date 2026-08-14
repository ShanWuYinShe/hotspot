package media

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
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

	resp, err := provider.Presign(context.Background(), PresignFileInput{FileName: "a.png", MimeType: "image/png", MD5: "d41d8cd98f00b204e9800998ecf8427e"}, "u1")
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

func TestObjectStorageProviderHeadRejectsInvalidMetadata(t *testing.T) {
	t.Run("returns error on server failure", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		provider := NewObjectStorageProvider(config.Config{S3Endpoint: server.URL, S3Bucket: "hotspot", S3ForcePathStyle: true})
		if _, err := provider.Head(context.Background(), "uploads/u1/upload_1.png"); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("returns error when content length missing", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "image/png")
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		provider := NewObjectStorageProvider(config.Config{S3Endpoint: server.URL, S3Bucket: "hotspot", S3ForcePathStyle: true})
		if _, err := provider.Head(context.Background(), "uploads/u1/upload_1.png"); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("returns error when content type missing", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Length", "123")
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		provider := NewObjectStorageProvider(config.Config{S3Endpoint: server.URL, S3Bucket: "hotspot", S3ForcePathStyle: true})
		if _, err := provider.Head(context.Background(), "uploads/u1/upload_1.png"); err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestObjectStorageProviderRequiresConfiguration(t *testing.T) {
	provider := NewObjectStorageProvider(config.Config{})
	if _, err := provider.Presign(context.Background(), PresignFileInput{FileName: "a.png", MimeType: "image/png"}, "u1"); err == nil {
		t.Fatal("expected presign configuration error")
	}
	if _, err := provider.Head(context.Background(), "uploads/u1/upload_1.png"); err == nil {
		t.Fatal("expected head configuration error")
	}
}

// ---------------------------------------------------------------------------
// presignURL tests (unexported, tested from within package media)
// ---------------------------------------------------------------------------

func TestPresignURLGeneratesSignedHEADURL(t *testing.T) {
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

	rawURL, err := provider.presignURL(context.Background(), http.MethodHead, "uploads/u1/upload_1.png", 5*time.Minute)
	if err != nil {
		t.Fatalf("presignURL returned error: %v", err)
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("failed to parse generated URL: %v", err)
	}

	// Host must be present
	if parsed.Host == "" {
		t.Fatal("expected non-empty host in signed URL")
	}

	// Path must contain bucket prefix (path-style)
	expectedPath := "/hotspot/uploads/u1/upload_1.png"
	if parsed.Path != expectedPath {
		t.Fatalf("expected path %q, got %q", expectedPath, parsed.Path)
	}

	query := parsed.Query()
	if query.Get("X-Amz-Algorithm") != "AWS4-HMAC-SHA256" {
		t.Fatalf("expected X-Amz-Algorithm=AWS4-HMAC-SHA256, got %q", query.Get("X-Amz-Algorithm"))
	}
	if query.Get("X-Amz-Date") != "20260330T080000Z" {
		t.Fatalf("expected X-Amz-Date=20260330T080000Z, got %q", query.Get("X-Amz-Date"))
	}
	if query.Get("X-Amz-Expires") != "300" {
		t.Fatalf("expected X-Amz-Expires=300 (5 min), got %q", query.Get("X-Amz-Expires"))
	}
	if query.Get("X-Amz-SignedHeaders") != "host" {
		t.Fatalf("expected X-Amz-SignedHeaders=host, got %q", query.Get("X-Amz-SignedHeaders"))
	}
	if query.Get("X-Amz-Signature") == "" {
		t.Fatal("expected non-empty X-Amz-Signature")
	}
}

func TestPresignURLGeneratesSignedPUTURL(t *testing.T) {
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

	// TTL=15 minutes (mediaPresignTTL) should produce X-Amz-Expires=900
	rawURL, err := provider.presignURL(context.Background(), http.MethodPut, "uploads/u1/upload_1.png", 15*time.Minute)
	if err != nil {
		t.Fatalf("presignURL returned error: %v", err)
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("failed to parse generated URL: %v", err)
	}
	query := parsed.Query()
	if query.Get("X-Amz-Expires") != "900" {
		t.Fatalf("expected X-Amz-Expires=900 (15 min), got %q", query.Get("X-Amz-Expires"))
	}
	if query.Get("X-Amz-Signature") == "" {
		t.Fatal("expected non-empty X-Amz-Signature")
	}
}

func TestPresignURLDifferentMethodsProduceDifferentSignatures(t *testing.T) {
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

	headURL, err := provider.presignURL(context.Background(), http.MethodHead, "uploads/u1/upload_1.png", 5*time.Minute)
	if err != nil {
		t.Fatalf("presignURL HEAD returned error: %v", err)
	}
	putURL, err := provider.presignURL(context.Background(), http.MethodPut, "uploads/u1/upload_1.png", 15*time.Minute)
	if err != nil {
		t.Fatalf("presignURL PUT returned error: %v", err)
	}

	headParsed, _ := url.Parse(headURL)
	putParsed, _ := url.Parse(putURL)

	headSig := headParsed.Query().Get("X-Amz-Signature")
	putSig := putParsed.Query().Get("X-Amz-Signature")
	if headSig == "" || putSig == "" {
		t.Fatal("expected non-empty signatures")
	}
	if headSig == putSig {
		t.Fatalf("HEAD and PUT produced the same signature; they must differ. Signature=%q", headSig)
	}
}

func TestPresignURLUsesRegionFromConfig(t *testing.T) {
	provider := NewObjectStorageProvider(config.Config{
		S3Endpoint:       "https://s3.example.com",
		S3Region:         "eu-west-1",
		S3Bucket:         "hotspot",
		S3AccessKey:      "access-key",
		S3SecretKey:      "secret-key",
		S3ForcePathStyle: true,
	})
	provider.now = func() time.Time {
		return time.Date(2026, 3, 30, 8, 0, 0, 0, time.UTC)
	}

	presignedURL, err := provider.presignURL(context.Background(), http.MethodHead, "uploads/u1/upload_1.png", 5*time.Minute)
	if err != nil {
		t.Fatalf("presignURL returned error: %v", err)
	}

	parsed, _ := url.Parse(presignedURL)
	cred := parsed.Query().Get("X-Amz-Credential")
	if !strings.Contains(cred, "/eu-west-1/s3/aws4_request") {
		t.Fatalf("expected X-Amz-Credential to contain /eu-west-1/s3/aws4_request, got %q", cred)
	}
}

func TestPresignURLDefaultsToUSEast1(t *testing.T) {
	provider := NewObjectStorageProvider(config.Config{
		S3Endpoint:       "https://s3.example.com",
		S3Region:         "", // empty → defaults to us-east-1
		S3Bucket:         "hotspot",
		S3AccessKey:      "access-key",
		S3SecretKey:      "secret-key",
		S3ForcePathStyle: true,
	})
	provider.now = func() time.Time {
		return time.Date(2026, 3, 30, 8, 0, 0, 0, time.UTC)
	}

	presignedURL, err := provider.presignURL(context.Background(), http.MethodHead, "uploads/u1/upload_1.png", 5*time.Minute)
	if err != nil {
		t.Fatalf("presignURL returned error: %v", err)
	}

	parsed, _ := url.Parse(presignedURL)
	cred := parsed.Query().Get("X-Amz-Credential")
	if !strings.Contains(cred, "/us-east-1/s3/aws4_request") {
		t.Fatalf("expected X-Amz-Credential to contain /us-east-1/s3/aws4_request, got %q", cred)
	}
}

func TestPresignURLVirtualHostedStyle(t *testing.T) {
	provider := NewObjectStorageProvider(config.Config{
		S3Endpoint:       "https://hotspot.s3.amazonaws.com",
		S3Region:         "us-east-1",
		S3Bucket:         "hotspot",
		S3AccessKey:      "access-key",
		S3SecretKey:      "secret-key",
		S3ForcePathStyle: false,
	})
	provider.now = func() time.Time {
		return time.Date(2026, 3, 30, 8, 0, 0, 0, time.UTC)
	}

	presignedURL, err := provider.presignURL(context.Background(), http.MethodHead, "uploads/u1/upload_1.png", 5*time.Minute)
	if err != nil {
		t.Fatalf("presignURL returned error: %v", err)
	}

	parsed, _ := url.Parse(presignedURL)
	if parsed.Host != "hotspot.s3.amazonaws.com" {
		t.Fatalf("expected host hotspot.s3.amazonaws.com, got %q", parsed.Host)
	}
	if parsed.Path != "/uploads/u1/upload_1.png" {
		t.Fatalf("expected path /uploads/u1/upload_1.png (no bucket prefix), got %q", parsed.Path)
	}
}

func TestPresignURLPathStyle(t *testing.T) {
	provider := NewObjectStorageProvider(config.Config{
		S3Endpoint:       "http://seaweedfs:8333",
		S3Region:         "us-east-1",
		S3Bucket:         "hotspot",
		S3AccessKey:      "access-key",
		S3SecretKey:      "secret-key",
		S3ForcePathStyle: true,
	})
	provider.now = func() time.Time {
		return time.Date(2026, 3, 30, 8, 0, 0, 0, time.UTC)
	}

	presignedURL, err := provider.presignURL(context.Background(), http.MethodHead, "uploads/u1/upload_1.png", 5*time.Minute)
	if err != nil {
		t.Fatalf("presignURL returned error: %v", err)
	}

	parsed, _ := url.Parse(presignedURL)
	if !strings.HasPrefix(parsed.Path, "/hotspot/uploads/") {
		t.Fatalf("expected path to start with /hotspot/uploads/, got %q", parsed.Path)
	}
}

func TestPresignURLUsesPublicEndpointForClientFacingURL(t *testing.T) {
	provider := NewObjectStorageProvider(config.Config{
		S3Endpoint:             "http://seaweedfs:8333",
		S3PublicEndpoint:       "http://localhost:8333",
		S3Region:               "us-east-1",
		S3Bucket:               "hotspot",
		S3AccessKey:            "access-key",
		S3SecretKey:            "secret-key",
		S3ForcePathStyle:       true,
		S3PublicForcePathStyle: true,
	})
	provider.now = func() time.Time {
		return time.Date(2026, 3, 30, 8, 0, 0, 0, time.UTC)
	}

	presignedURL, err := provider.presignURL(context.Background(), http.MethodPut, "uploads/u1/upload_1.png", 15*time.Minute)
	if err != nil {
		t.Fatalf("presignURL returned error: %v", err)
	}

	parsed, err := url.Parse(presignedURL)
	if err != nil {
		t.Fatalf("failed to parse presigned url: %v", err)
	}
	if parsed.Host != "localhost:8333" {
		t.Fatalf("expected client-facing host localhost:8333, got %q", parsed.Host)
	}
	if !strings.HasPrefix(parsed.Path, "/hotspot/uploads/") {
		t.Fatalf("expected path to start with /hotspot/uploads/, got %q", parsed.Path)
	}
	if parsed.Query().Get("X-Amz-SignedHeaders") != "host" {
		t.Fatalf("expected signed host header, got %q", parsed.Query().Get("X-Amz-SignedHeaders"))
	}
}

func TestPresignURLPathStyleAutoDetect(t *testing.T) {
	t.Run("localhost endpoint auto-detects path style", func(t *testing.T) {
		provider := NewObjectStorageProvider(config.Config{
			S3Endpoint:       "http://localhost:8333",
			S3Region:         "us-east-1",
			S3Bucket:         "hotspot",
			S3AccessKey:      "access-key",
			S3SecretKey:      "secret-key",
			S3ForcePathStyle: false, // explicitly false, but host is localhost
		})
		provider.now = func() time.Time {
			return time.Date(2026, 3, 30, 8, 0, 0, 0, time.UTC)
		}

		presignedURL, err := provider.presignURL(context.Background(), http.MethodHead, "uploads/u1/upload_1.png", 5*time.Minute)
		if err != nil {
			t.Fatalf("presignURL returned error: %v", err)
		}
		parsed, _ := url.Parse(presignedURL)
		if !strings.HasPrefix(parsed.Path, "/hotspot/uploads/") {
			t.Fatalf("expected path to start with /hotspot/uploads/ for localhost, got %q", parsed.Path)
		}
	})

	t.Run("IP address with port auto-detects path style", func(t *testing.T) {
		provider := NewObjectStorageProvider(config.Config{
			S3Endpoint:       "http://192.168.1.1:8333",
			S3Region:         "us-east-1",
			S3Bucket:         "hotspot",
			S3AccessKey:      "access-key",
			S3SecretKey:      "secret-key",
			S3ForcePathStyle: false, // explicitly false, but host contains colon
		})
		provider.now = func() time.Time {
			return time.Date(2026, 3, 30, 8, 0, 0, 0, time.UTC)
		}

		presignedURL, err := provider.presignURL(context.Background(), http.MethodHead, "uploads/u1/upload_1.png", 5*time.Minute)
		if err != nil {
			t.Fatalf("presignURL returned error: %v", err)
		}
		parsed, _ := url.Parse(presignedURL)
		if !strings.HasPrefix(parsed.Path, "/hotspot/uploads/") {
			t.Fatalf("expected path to start with /hotspot/uploads/ for IP:port, got %q", parsed.Path)
		}
	})
}

func TestHeadSendsSignedRequest(t *testing.T) {
	var capturedURL string
	var capturedMethod string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedURL = r.URL.String()
		capturedMethod = r.Method
		w.Header().Set("Content-Length", "123")
		w.Header().Set("Content-Type", "image/png")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	provider := NewObjectStorageProvider(config.Config{
		S3Endpoint:       server.URL,
		S3Region:         "us-east-1",
		S3Bucket:         "hotspot",
		S3AccessKey:      "access-key",
		S3SecretKey:      "secret-key",
		S3ForcePathStyle: true,
	})
	provider.now = func() time.Time {
		return time.Date(2026, 3, 30, 8, 0, 0, 0, time.UTC)
	}

	_, err := provider.Head(context.Background(), "uploads/u1/upload_1.png")
	if err != nil {
		t.Fatalf("Head returned error: %v", err)
	}

	if capturedMethod != http.MethodHead {
		t.Fatalf("expected HEAD method, got %q", capturedMethod)
	}
	if !strings.Contains(capturedURL, "X-Amz-Algorithm=AWS4-HMAC-SHA256") {
		t.Fatalf("expected signed URL with X-Amz-Algorithm, got %q", capturedURL)
	}
	if !strings.Contains(capturedURL, "X-Amz-Signature=") {
		t.Fatalf("expected signed URL with X-Amz-Signature, got %q", capturedURL)
	}
}

func TestHeadUsesInternalEndpointEvenWhenPublicEndpointConfigured(t *testing.T) {
	var capturedHost string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedHost = r.Host
		w.Header().Set("Content-Length", "123")
		w.Header().Set("Content-Type", "image/png")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	provider := NewObjectStorageProvider(config.Config{
		S3Endpoint:             server.URL,
		S3PublicEndpoint:       "http://localhost:1",
		S3Region:               "us-east-1",
		S3Bucket:               "hotspot",
		S3AccessKey:            "access-key",
		S3SecretKey:            "secret-key",
		S3ForcePathStyle:       true,
		S3PublicForcePathStyle: true,
	})
	provider.now = func() time.Time {
		return time.Date(2026, 3, 30, 8, 0, 0, 0, time.UTC)
	}

	_, err := provider.Head(context.Background(), "uploads/u1/upload_1.png")
	if err != nil {
		t.Fatalf("Head returned error: %v", err)
	}
	if capturedHost == "" {
		t.Fatal("expected HEAD request to reach internal endpoint")
	}
}

func TestPresignURLRejectsMissingEndpoint(t *testing.T) {
	provider := NewObjectStorageProvider(config.Config{
		S3Endpoint: "",
		S3Bucket:   "hotspot",
	})
	_, err := provider.presignURL(context.Background(), http.MethodHead, "uploads/u1/upload_1.png", 5*time.Minute)
	if err == nil {
		t.Fatal("expected error for missing endpoint, got nil")
	}
}

func TestPresignURLRejectsMissingBucket(t *testing.T) {
	provider := NewObjectStorageProvider(config.Config{
		S3Endpoint: "https://s3.example.com",
		S3Bucket:   "",
	})
	_, err := provider.presignURL(context.Background(), http.MethodHead, "uploads/u1/upload_1.png", 5*time.Minute)
	if err == nil {
		t.Fatal("expected error for missing bucket, got nil")
	}
}

func TestPresignURLRejectsInvalidEndpointURL(t *testing.T) {
	provider := NewObjectStorageProvider(config.Config{
		S3Endpoint: "://bad",
		S3Bucket:   "hotspot",
	})
	_, err := provider.presignURL(context.Background(), http.MethodHead, "uploads/u1/upload_1.png", 5*time.Minute)
	if err == nil {
		t.Fatal("expected error for invalid endpoint URL, got nil")
	}
}
