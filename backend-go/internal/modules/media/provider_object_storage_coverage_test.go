package media

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/config"
)

func newCoverageObjectProvider(cfg config.Config) *ObjectStorageProvider {
	return NewObjectStorageProvider(cfg)
}

func coverageS3Config(endpoint string) config.Config {
	return config.Config{
		S3Endpoint:       endpoint,
		S3Bucket:         "hotspot",
		S3Region:         "us-east-1",
		S3AccessKey:      "access-key",
		S3SecretKey:      "secret-key",
		S3ForcePathStyle: true,
	}
}

// 内部端点无法解析（含控制字符）时，所有内部签名操作都必须直接报错，
// 不能发出未签名请求。
func TestObjectStorageProviderInternalPresignErrors(t *testing.T) {
	provider := newCoverageObjectProvider(coverageS3Config("http://\x7f"))
	ctx := context.Background()
	if err := provider.Delete(ctx, "uploads/u1/a.png"); err == nil {
		t.Fatal("Delete must fail when the internal endpoint is unparsable")
	}
	if _, err := provider.Head(ctx, "uploads/u1/a.png"); err == nil {
		t.Fatal("Head must fail when the internal endpoint is unparsable")
	}
	if _, err := provider.ReadPrefix(ctx, "uploads/u1/a.png", 8); err == nil {
		t.Fatal("ReadPrefix must fail when the internal endpoint is unparsable")
	}
	if _, err := provider.GetObject(ctx, "uploads/u1/a.png"); err == nil {
		t.Fatal("GetObject must fail when the internal endpoint is unparsable")
	}
	if err := provider.PutObject(ctx, "uploads/u1/a.png", "image/png", []byte("x")); err == nil {
		t.Fatal("PutObject must fail when the internal endpoint is unparsable")
	}
}

// 客户端可见端点无法解析时读签名必须报错（不允许回退成未签名 URL）。
func TestObjectStorageProviderClientFacingEndpointParseError(t *testing.T) {
	cfg := coverageS3Config("http://seaweedfs:8333")
	cfg.S3PublicEndpoint = "http://\x7f"
	provider := newCoverageObjectProvider(cfg)
	if _, err := provider.PresignGet(context.Background(), "uploads/u1/a.png", time.Minute); err == nil {
		t.Fatal("PresignGet must fail when S3_PUBLIC_ENDPOINT is unparsable")
	}
}

// 网络不可达：所有对象操作返回传输层错误。
func TestObjectStorageProviderNetworkErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	endpoint := server.URL
	server.Close()

	provider := newCoverageObjectProvider(coverageS3Config(endpoint))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := provider.Delete(ctx, "uploads/u1/a.png"); err == nil {
		t.Fatal("Delete must surface connection errors")
	}
	if _, err := provider.Head(ctx, "uploads/u1/a.png"); err == nil {
		t.Fatal("Head must surface connection errors")
	}
	if _, err := provider.ReadPrefix(ctx, "uploads/u1/a.png", 8); err == nil {
		t.Fatal("ReadPrefix must surface connection errors")
	}
	if _, err := provider.GetObject(ctx, "uploads/u1/a.png"); err == nil {
		t.Fatal("GetObject must surface connection errors")
	}
	if err := provider.PutObject(ctx, "uploads/u1/a.png", "image/png", []byte("x")); err == nil {
		t.Fatal("PutObject must surface connection errors")
	}
}

// 非 2xx（5xx）映射为可读错误，调用方据此决定重试。
func TestObjectStorageProviderServerErrorStatuses(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	provider := newCoverageObjectProvider(coverageS3Config(server.URL))
	ctx := context.Background()
	if err := provider.Delete(ctx, "uploads/u1/a.png"); err == nil || !strings.Contains(err.Error(), "delete failed") {
		t.Fatalf("Delete 500 = %v", err)
	}
	if _, err := provider.ReadPrefix(ctx, "uploads/u1/a.png", 8); err == nil || !strings.Contains(err.Error(), "read prefix failed") {
		t.Fatalf("ReadPrefix 500 = %v", err)
	}
	if _, err := provider.GetObject(ctx, "uploads/u1/a.png"); err == nil || !strings.Contains(err.Error(), "get failed") {
		t.Fatalf("GetObject 500 = %v", err)
	}
	if err := provider.PutObject(ctx, "uploads/u1/a.png", "", []byte("x")); err == nil || !strings.Contains(err.Error(), "put failed") {
		t.Fatalf("PutObject 500 = %v", err)
	}
}

func TestContentMD5HeaderValueNormalisesHex(t *testing.T) {
	// base64(md5(""))，证明 32 位 hex 被转成 RFC 3230 / S3 的标准形态。
	if got := contentMD5HeaderValue("d41d8cd98f00b204e9800998ecf8427e"); got != "1B2M2Y8AsgTpgAmY7PhCfg==" {
		t.Fatalf("contentMD5HeaderValue = %q", got)
	}
	// 非 32 位 hex 原样透传（交给存储端判定，不阻断上传）。
	for _, value := range []string{"not-hex", "00", ""} {
		if got := contentMD5HeaderValue(value); got != value {
			t.Fatalf("contentMD5HeaderValue(%q) = %q, want passthrough", value, got)
		}
	}
}

func TestPresignCacheStoreInitialisesAndCapsMap(t *testing.T) {
	provider := newCoverageObjectProvider(coverageS3Config("https://s3.example.com"))
	provider.presignCur = nil
	provider.storePresignedURL("get:static:key", "https://s3.example.com/key", time.Minute)
	if provider.presignCur == nil || provider.presignCur["get:static:key"].url != "https://s3.example.com/key" {
		t.Fatalf("cache map must be initialised before store, got %#v", provider.presignCur)
	}

	full := make(map[string]presignCacheEntry, presignCacheMaxEntries)
	for i := 0; i < presignCacheMaxEntries; i++ {
		full[fmt.Sprintf("key-%d", i)] = presignCacheEntry{url: "https://s3.example.com/cached", expiresAt: time.Now().Add(time.Hour)}
	}
	provider.presignCur = full
	provider.storePresignedURL("get:static:overflow", "https://s3.example.com/overflow", time.Minute)
	if _, ok := provider.presignCur["get:static:overflow"]; ok {
		t.Fatal("full cache must refuse new entries (bounded memory)")
	}
	if len(provider.presignCur) != presignCacheMaxEntries {
		t.Fatalf("cache size = %d, want %d", len(provider.presignCur), presignCacheMaxEntries)
	}
}
