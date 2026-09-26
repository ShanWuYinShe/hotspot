package media

import (
	"context"
	"strings"
	"testing"

	"hotspot/backend-go/internal/platform/config"
)

// 上传直传 URL 必须按请求入口 origin 动态签名：静态 S3_PUBLIC_ENDPOINT
// （部署期值，如 http://127.0.0.1:8888/s3）在用户从 localhost/域名入口访问
// 时会产生跨源 PUT，被浏览器 CORS 拦截（实测 Failed to fetch）。
func TestPresignUploadURLFollowsClientRequestEndpoint(t *testing.T) {
	provider := NewObjectStorageProvider(config.Config{
		S3Endpoint:             "http://seaweedfs:8333",
		S3Bucket:               "hotspot",
		S3PublicEndpoint:       "http://127.0.0.1:8888/s3",
		S3PublicForcePathStyle: true,
	})

	ctx := WithClientRequestEndpoint(context.Background(), "https", "chaoset.com")
	upload, err := provider.Presign(ctx, PresignFileInput{FileName: "a.png", FileSize: 10, MimeType: "image/png", Category: "moment_image"}, "user_1")
	if err != nil {
		t.Fatalf("Presign returned error: %v", err)
	}
	if !strings.HasPrefix(upload.UploadURL, "https://chaoset.com/s3/hotspot/uploads/") {
		t.Fatalf("upload URL should follow request origin, got %s", upload.UploadURL)
	}

	// 无入口信息（脚本/内部调用）回退静态 S3_PUBLIC_ENDPOINT。
	fallback, err := provider.Presign(context.Background(), PresignFileInput{FileName: "b.png", FileSize: 10, MimeType: "image/png", Category: "moment_image"}, "user_1")
	if err != nil {
		t.Fatalf("fallback Presign returned error: %v", err)
	}
	if !strings.HasPrefix(fallback.UploadURL, "http://127.0.0.1:8888/s3/hotspot/uploads/") {
		t.Fatalf("fallback upload URL should use configured endpoint, got %s", fallback.UploadURL)
	}
}
