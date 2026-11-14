package media

import (
	"context"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/config"
)

// CopyObject 的请求形状是 copy-on-complete 的安全边界：预签名 PUT 必须把
// x-amz-copy-source 纳入 SigV4 签名（X-Amz-SignedHeaders=host;x-amz-copy-source），
// 否则该头可被篡改，发布对象可能被复制自任意源对象。用例独立重算一份 canonical
// request 比对签名，锁定「头真的进了签名」而不只是「头存在」。
func TestObjectStorageProviderCopyObjectRequestShape(t *testing.T) {
	const (
		bucket    = "hotspot"
		accessKey = "access-key"
		secretKey = "secret-key"
		region    = "us-east-1"
		srcKey    = "uploads/u1/upload_1.jpg"
		dstKey    = "published/u1/upload_1.jpg"
	)
	var (
		gotMethod string
		gotPath   string
		gotQuery  url.Values
		gotSource string
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotQuery = r.URL.Query()
		gotSource = r.Header.Get("x-amz-copy-source")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	provider := NewObjectStorageProvider(config.Config{
		S3Endpoint:       server.URL,
		S3Bucket:         bucket,
		S3Region:         region,
		S3AccessKey:      accessKey,
		S3SecretKey:      secretKey,
		S3ForcePathStyle: true,
	})
	now := time.Date(2026, 3, 30, 8, 0, 0, 0, time.UTC)
	provider.now = func() time.Time { return now }
	if err := provider.CopyObject(context.Background(), srcKey, dstKey); err != nil {
		t.Fatalf("CopyObject returned error: %v", err)
	}

	if gotMethod != http.MethodPut {
		t.Fatalf("expected PUT method, got %q", gotMethod)
	}
	if gotPath != "/hotspot/published/u1/upload_1.jpg" {
		t.Fatalf("unexpected destination path: %q", gotPath)
	}
	if gotSource != "/hotspot/uploads/u1/upload_1.jpg" {
		t.Fatalf("unexpected x-amz-copy-source: %q", gotSource)
	}
	if signed := gotQuery.Get("X-Amz-SignedHeaders"); signed != "host;x-amz-copy-source" {
		t.Fatalf("X-Amz-SignedHeaders = %q, want host;x-amz-copy-source", signed)
	}
	if gotQuery.Get("X-Amz-Signature") == "" {
		t.Fatal("expected non-empty X-Amz-Signature")
	}

	// 独立重算签名：canonical request 的 query/header 形态必须与 server 实际收到的
	// 请求一致（X-Amz-Signature 除外），证明 x-amz-copy-source 参与签名。
	endpoint, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse server url: %v", err)
	}
	host := endpoint.Host
	amzDate := "20260330T080000Z"
	dateStamp := "20260330"
	credentialScope := dateStamp + "/" + region + "/s3/aws4_request"
	canonicalQuery := url.Values{}
	for name, values := range gotQuery {
		if name == "X-Amz-Signature" {
			continue
		}
		for _, value := range values {
			canonicalQuery.Add(name, value)
		}
	}
	canonicalHeaders := "host:" + host + "\n" + "x-amz-copy-source:/hotspot/uploads/u1/upload_1.jpg" + "\n"
	canonicalRequest := strings.Join([]string{
		http.MethodPut,
		"/hotspot/published/u1/upload_1.jpg",
		canonicalQuery.Encode(),
		canonicalHeaders,
		"host;x-amz-copy-source",
		"UNSIGNED-PAYLOAD",
	}, "\n")
	stringToSign := strings.Join([]string{
		"AWS4-HMAC-SHA256",
		amzDate,
		credentialScope,
		hexHashForCopyTest(canonicalRequest),
	}, "\n")
	expected := signForCopyTest(provider, dateStamp, region, stringToSign)
	if gotQuery.Get("X-Amz-Signature") != expected {
		t.Fatalf("signature does not cover the copy-source header: got %q, want %q", gotQuery.Get("X-Amz-Signature"), expected)
	}
}

// 辅助：与 provider 内部实现同一套哈希/HMAC 原语（签名串与 canonical request
// 由测试独立构造，不调用被测量的预签名函数）。
func hexHashForCopyTest(value string) string {
	return hex.EncodeToString(hashSHA256([]byte(value)))
}

func signForCopyTest(p *ObjectStorageProvider, dateStamp, region, stringToSign string) string {
	return hex.EncodeToString(hmacSHA256(p.signingKey(dateStamp, region), stringToSign))
}

// 失败路径：源对象不存在 → ErrUploadNotReady（与 Head 同口径，客户端可重试
// complete）；存储端 5xx / 网络错误 → 错误上抛，complete 不得提交。
func TestObjectStorageProviderCopyObjectErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-amz-copy-source") == "/hotspot/uploads/u1/missing.png" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()
	provider := NewObjectStorageProvider(config.Config{
		S3Endpoint: server.URL, S3Bucket: "hotspot", S3Region: "us-east-1",
		S3AccessKey: "access-key", S3SecretKey: "secret-key", S3ForcePathStyle: true,
	})
	ctx := context.Background()
	if err := provider.CopyObject(ctx, "uploads/u1/missing.png", "published/u1/missing.png"); err != ErrUploadNotReady {
		t.Fatalf("missing source = %v, want ErrUploadNotReady", err)
	}
	if err := provider.CopyObject(ctx, "uploads/u1/boom.png", "published/u1/boom.png"); err == nil || !strings.Contains(err.Error(), "copy failed") {
		t.Fatalf("5xx = %v, want copy failed error", err)
	}
	if err := provider.CopyObject(ctx, "  ", "published/u1/x.png"); err == nil {
		t.Fatal("empty source key must be rejected")
	}
	if err := provider.CopyObject(ctx, "uploads/u1/x.png", " "); err == nil {
		t.Fatal("empty destination key must be rejected")
	}

	// 网络不可达：错误上抛（不允许静默成功）。
	closed := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	closedURL := closed.URL
	closed.Close()
	broken := NewObjectStorageProvider(config.Config{
		S3Endpoint: closedURL, S3Bucket: "hotspot", S3ForcePathStyle: true,
	})
	if err := broken.CopyObject(ctx, "uploads/u1/x.png", "published/u1/x.png"); err == nil {
		t.Fatal("network errors must surface")
	}
}

// canonicalCopySource 形态：标准 S3 复制源 /<bucket>/<key>，对象键不额外百分号
// 编码（键字符集受服务端生成约束，含 '/' 保留原样供 SeaweedFS 直接解析）。
func TestCanonicalCopySource(t *testing.T) {
	cases := map[string]string{
		"uploads/u1/upload_1.jpg":   "/hotspot/uploads/u1/upload_1.jpg",
		"/uploads/u1/upload_1.jpg":  "/hotspot/uploads/u1/upload_1.jpg",
		"published/u1/upload_1.png": "/hotspot/published/u1/upload_1.png",
	}
	for input, want := range cases {
		if got := canonicalCopySource("hotspot", input); got != want {
			t.Fatalf("canonicalCopySource(%q) = %q, want %q", input, got, want)
		}
	}
}
