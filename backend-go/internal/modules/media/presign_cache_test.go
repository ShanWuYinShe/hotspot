package media

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/config"
)

// 审计 perf P3-4：PresignGet 两代缓存——窗口内同 key 返回同一签名 URL（客户端
// 缓存友好 + 免重复签名），窗口轮换后重签；PUT 上传签名不缓存（要求新鲜）。
func TestPresignGetCacheStableWithinWindow(t *testing.T) {
	provider := newTestProvider(t)
	ctx := context.Background()

	first, err := provider.PresignGet(ctx, "uploads/u1/a.png", 24*time.Hour)
	if err != nil {
		t.Fatalf("PresignGet: %v", err)
	}
	second, err := provider.PresignGet(ctx, "uploads/u1/a.png", 24*time.Hour)
	if err != nil {
		t.Fatalf("PresignGet (2nd): %v", err)
	}
	if first != second {
		t.Fatalf("URL should be stable within cache window:\n%s\n%s", first, second)
	}

	// 不同 key 各自独立
	other, err := provider.PresignGet(ctx, "uploads/u1/b.png", 24*time.Hour)
	if err != nil {
		t.Fatalf("PresignGet (other key): %v", err)
	}
	if strings.Contains(other, "a.png") {
		t.Fatalf("other key URL should not reuse first key: %s", other)
	}
}

func TestPresignGetCacheRotatesAfterWindow(t *testing.T) {
	provider := newTestProvider(t)
	base := time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)
	clock := base
	provider.now = func() time.Time { return clock }
	ctx := context.Background()

	first, err := provider.PresignGet(ctx, "uploads/u1/rot.png", 24*time.Hour)
	if err != nil {
		t.Fatalf("PresignGet: %v", err)
	}

	// 推进 2 小时（≥1 个轮换窗口，<2 个）：上一代缓存仍服务旧 URL（条目
	// 存活 1–2 个窗口，远短于 24h 签名有效期，安全）
	clock = base.Add(2 * time.Hour)
	stale, err := provider.PresignGet(ctx, "uploads/u1/rot.png", 24*time.Hour)
	if err != nil {
		t.Fatalf("PresignGet (after 1 window): %v", err)
	}
	if stale != first {
		t.Fatal("URL should stay stable within two generations")
	}

	// 推进超过两个窗口：两代均未命中 → 重签（X-Amz-Date 变化）
	clock = base.Add(3 * time.Hour)
	second, err := provider.PresignGet(ctx, "uploads/u1/rot.png", 24*time.Hour)
	if err != nil {
		t.Fatalf("PresignGet (after 2 windows): %v", err)
	}
	if first == second {
		t.Fatal("URL should be re-signed after both cache generations expired")
	}
}

// PUT 上传签名不缓存：两次调用必须产生独立结果（签名时间戳/随机要素新鲜）。
func TestPresignPutNotCached(t *testing.T) {
	provider := newTestProvider(t)
	base := time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)
	clock := base
	provider.now = func() time.Time { return clock }
	ctx := context.Background()

	_, err := provider.presignURL(ctx, http.MethodPut, "uploads/u1/up.bin", 15*time.Minute)
	if err != nil {
		t.Fatalf("presign PUT: %v", err)
	}
	// 推进数秒后重签（未过轮换窗口）：若走了缓存会返回同一 URL
	clock = base.Add(5 * time.Second)
	signed, err := provider.presignURL(ctx, http.MethodPut, "uploads/u1/up.bin", 15*time.Minute)
	if err != nil {
		t.Fatalf("presign PUT (2nd): %v", err)
	}
	if !strings.Contains(signed, "X-Amz-Date=20260916T120005Z") {
		t.Fatalf("PUT presign must bypass cache (fresh X-Amz-Date): %s", signed)
	}
}

func newTestProvider(t *testing.T) *ObjectStorageProvider {
	t.Helper()
	return NewObjectStorageProvider(config.Config{
		S3Endpoint:       "https://s3.example.com",
		S3Region:         "us-east-1",
		S3Bucket:         "hotspot",
		S3AccessKey:      "access-key",
		S3SecretKey:      "secret-key",
		S3ForcePathStyle: true,
	})
}
