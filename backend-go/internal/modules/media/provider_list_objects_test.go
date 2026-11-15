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

// ListObjectsV2 的请求形态是孤儿清理的安全/正确性边界：S3 按实际收到的
// 全部 query 参数重算 SigV4 canonical query，list-type/prefix/max-keys/
// continuation-token 任一漏签都会 403。用例独立重算一份 canonical request
// 比对签名，锁定「业务 query 参数真的进了签名」而不只是「参数出现在 URL 上」。
func TestObjectStorageProviderListObjectsRequestShapeAndSignature(t *testing.T) {
	const (
		bucket    = "hotspot"
		accessKey = "access-key"
		secretKey = "secret-key"
		region    = "us-east-1"
	)
	var (
		gotMethod string
		gotPath   string
		gotQuery  url.Values
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotQuery = r.URL.Query()
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<ListBucketResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
  <Name>hotspot</Name>
  <Prefix>uploads/</Prefix>
  <KeyCount>2</KeyCount>
  <MaxKeys>2</MaxKeys>
  <IsTruncated>true</IsTruncated>
  <NextContinuationToken>tok+/=2026</NextContinuationToken>
  <Contents>
    <Key>uploads/u1/a.txt</Key>
    <LastModified>2026-11-01T00:00:00.000Z</LastModified>
    <ETag>"etag-a"</ETag>
    <Size>5</Size>
  </Contents>
  <Contents>
    <Key>uploads/u1/b.txt</Key>
    <LastModified>2026-11-02T08:30:00Z</LastModified>
  </Contents>
</ListBucketResult>`))
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

	page, err := provider.ListObjects(context.Background(), "uploads/", "tok+/=2026", 2)
	if err != nil {
		t.Fatalf("ListObjects returned error: %v", err)
	}
	if gotMethod != http.MethodGet {
		t.Fatalf("expected GET, got %q", gotMethod)
	}
	// 桶级请求走 path-style 的 /<bucket>（不带尾斜杠）：带尾斜杠的 canonical
	// URI 会被部分 S3 实现归一化，与签名串不一致导致 403。
	if gotPath != "/hotspot" {
		t.Fatalf("unexpected bucket-level path: %q", gotPath)
	}
	for name, want := range map[string]string{
		"list-type":           "2",
		"prefix":              "uploads/",
		"max-keys":            "2",
		"continuation-token":  "tok+/=2026",
		"X-Amz-Algorithm":     "AWS4-HMAC-SHA256",
		"X-Amz-Date":          "20260330T080000Z",
		"X-Amz-SignedHeaders": "host",
	} {
		if got := gotQuery.Get(name); got != want {
			t.Fatalf("query %s = %q, want %q (query=%v)", name, got, want, gotQuery)
		}
	}
	if gotQuery.Get("X-Amz-Signature") == "" {
		t.Fatal("expected non-empty X-Amz-Signature")
	}

	// 独立重算签名：canonical query 必须包含 list-type/prefix/max-keys/
	// continuation-token（与 server 实际收到的一致，X-Amz-Signature 除外）。
	endpoint, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse server url: %v", err)
	}
	canonicalQuery := url.Values{}
	for name, values := range gotQuery {
		if name == "X-Amz-Signature" {
			continue
		}
		for _, value := range values {
			canonicalQuery.Add(name, value)
		}
	}
	amzDate, dateStamp := "20260330T080000Z", "20260330"
	credentialScope := dateStamp + "/" + region + "/s3/aws4_request"
	canonicalRequest := strings.Join([]string{
		http.MethodGet,
		"/hotspot",
		canonicalQuery.Encode(),
		"host:" + endpoint.Host + "\n",
		"host",
		"UNSIGNED-PAYLOAD",
	}, "\n")
	stringToSign := strings.Join([]string{
		"AWS4-HMAC-SHA256",
		amzDate,
		credentialScope,
		hex.EncodeToString(hashSHA256([]byte(canonicalRequest))),
	}, "\n")
	if expected := signForCopyTest(provider, dateStamp, region, stringToSign); gotQuery.Get("X-Amz-Signature") != expected {
		t.Fatalf("signature does not cover list query params: got %q, want %q", gotQuery.Get("X-Amz-Signature"), expected)
	}

	// 分页字段解析：Key/LastModified/IsTruncated/NextContinuationToken。
	if !page.IsTruncated || page.NextContinuationToken != "tok+/=2026" {
		t.Fatalf("unexpected pagination: %+v", page)
	}
	if len(page.Objects) != 2 {
		t.Fatalf("objects = %+v, want 2", page.Objects)
	}
	if page.Objects[0].Key != "uploads/u1/a.txt" || !page.Objects[0].LastModified.Equal(time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("unexpected first object: %+v", page.Objects[0])
	}
	if page.Objects[1].Key != "uploads/u1/b.txt" || !page.Objects[1].LastModified.Equal(time.Date(2026, 11, 2, 8, 30, 0, 0, time.UTC)) {
		t.Fatalf("unexpected second object: %+v", page.Objects[1])
	}
}

// max-keys 夹紧与默认值：<=0 或 >1000 统一取 S3 单页硬上限 1000；首页不携带
// continuation-token（空串不写进 query，避免 S3 把空 token 当非法参数）。
func TestObjectStorageProviderListObjectsClampsMaxKeys(t *testing.T) {
	cases := []struct {
		name    string
		maxKeys int
		want    string
	}{
		{"zero defaults to hard limit", 0, "1000"},
		{"negative defaults to hard limit", -5, "1000"},
		{"over hard limit clamps", 5000, "1000"},
		{"valid passthrough", 200, "200"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var gotQuery url.Values
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotQuery = r.URL.Query()
				_, _ = w.Write([]byte(`<ListBucketResult><IsTruncated>false</IsTruncated></ListBucketResult>`))
			}))
			defer server.Close()
			provider := NewObjectStorageProvider(config.Config{S3Endpoint: server.URL, S3Bucket: "hotspot", S3ForcePathStyle: true})
			if _, err := provider.ListObjects(context.Background(), "uploads/", "  ", tc.maxKeys); err != nil {
				t.Fatalf("ListObjects returned error: %v", err)
			}
			if got := gotQuery.Get("max-keys"); got != tc.want {
				t.Fatalf("max-keys = %q, want %q", got, tc.want)
			}
			if _, ok := gotQuery["continuation-token"]; ok {
				t.Fatalf("blank continuation token must be omitted, query=%v", gotQuery)
			}
		})
	}
}

// 失败/脏数据路径：空前缀不发起请求；非 2xx、非法 XML、网络错误上抛；
// 单条 LastModified 解析失败不得让整页失败，空 Key 跳过。
func TestObjectStorageProviderListObjectsErrorPaths(t *testing.T) {
	ctx := context.Background()

	t.Run("empty prefix rejected before request", func(t *testing.T) {
		called := false
		server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))
		defer server.Close()
		provider := NewObjectStorageProvider(config.Config{S3Endpoint: server.URL, S3Bucket: "hotspot", S3ForcePathStyle: true})
		if _, err := provider.ListObjects(ctx, "  ", "", 100); err == nil || !strings.Contains(err.Error(), "empty prefix") {
			t.Fatalf("expected empty-prefix error, got %v", err)
		}
		if called {
			t.Fatal("empty prefix must not hit the storage endpoint")
		}
	})

	for _, status := range []int{http.StatusInternalServerError, http.StatusNotFound, http.StatusForbidden} {
		status := status
		t.Run(http.StatusText(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(status)
			}))
			defer server.Close()
			provider := NewObjectStorageProvider(config.Config{S3Endpoint: server.URL, S3Bucket: "hotspot", S3ForcePathStyle: true})
			if _, err := provider.ListObjects(ctx, "uploads/", "", 100); err == nil || !strings.Contains(err.Error(), "status") {
				t.Fatalf("expected status error for %d, got %v", status, err)
			}
		})
	}

	t.Run("malformed xml", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("<ListBucketResult><IsTruncated>true"))
		}))
		defer server.Close()
		provider := NewObjectStorageProvider(config.Config{S3Endpoint: server.URL, S3Bucket: "hotspot", S3ForcePathStyle: true})
		if _, err := provider.ListObjects(ctx, "uploads/", "", 100); err == nil || !strings.Contains(err.Error(), "decode failed") {
			t.Fatalf("expected decode error, got %v", err)
		}
	})

	t.Run("network error surfaces", func(t *testing.T) {
		closed := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
		closedURL := closed.URL
		closed.Close()
		provider := NewObjectStorageProvider(config.Config{S3Endpoint: closedURL, S3Bucket: "hotspot", S3ForcePathStyle: true})
		if _, err := provider.ListObjects(ctx, "uploads/", "", 100); err == nil {
			t.Fatal("network errors must surface")
		}
	})

	t.Run("dirty entries do not fail the page", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`<ListBucketResult>
<IsTruncated>false</IsTruncated>
<Contents><Key></Key><LastModified>2026-11-01T00:00:00Z</LastModified></Contents>
<Contents><Key>uploads/u1/ok.txt</Key><LastModified>not-a-timestamp</LastModified></Contents>
</ListBucketResult>`))
		}))
		defer server.Close()
		provider := NewObjectStorageProvider(config.Config{S3Endpoint: server.URL, S3Bucket: "hotspot", S3ForcePathStyle: true})
		page, err := provider.ListObjects(ctx, "uploads/", "", 100)
		if err != nil {
			t.Fatalf("dirty entries must not fail the page: %v", err)
		}
		if len(page.Objects) != 1 || page.Objects[0].Key != "uploads/u1/ok.txt" {
			t.Fatalf("unexpected objects: %+v", page.Objects)
		}
		// 解析失败保留零值：调用方（孤儿清理）据此保守跳过删除。
		if !page.Objects[0].LastModified.IsZero() {
			t.Fatalf("unparseable last-modified must stay zero, got %s", page.Objects[0].LastModified)
		}
	})
}

// 桶级请求的 canonical URI：path-style 是 /<bucket>（不带尾斜杠），virtual-host
// style 是 /；对象级路径保持原有形态不变。
func TestCanonicalURIForBucketLevelRequests(t *testing.T) {
	pathStyle, err := url.Parse("http://127.0.0.1:8333")
	if err != nil {
		t.Fatalf("parse path-style endpoint: %v", err)
	}
	hostStyle, err := url.Parse("https://s3.amazonaws.com")
	if err != nil {
		t.Fatalf("parse host-style endpoint: %v", err)
	}
	cases := []struct {
		name       string
		endpoint   *url.URL
		storageKey string
		forcePath  bool
		want       string
	}{
		{"path-style bucket level", pathStyle, "", true, "/hotspot"},
		{"path-style object", pathStyle, "uploads/u1/a.txt", true, "/hotspot/uploads/u1/a.txt"},
		{"virtual-host bucket level", hostStyle, "", false, "/"},
		{"virtual-host object", hostStyle, "uploads/u1/a.txt", false, "/uploads/u1/a.txt"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := canonicalURIForEndpoint(tc.endpoint, "hotspot", tc.storageKey, tc.forcePath); got != tc.want {
				t.Fatalf("canonical URI = %q, want %q", got, tc.want)
			}
		})
	}
}
