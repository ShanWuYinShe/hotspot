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

	// 推进 2 小时（≥1 个轮换窗口，<2 个）：上一代缓存仍服务旧 URL（本例用
	// 24h 签名 TTL，条目未过期；生产读 TTL 仅 5–15min，过期条目会重签，见下）
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

// 短 TTL 回归：生产读签名仅 5–15min 有效，缓存窗口（1h）远长于它——条目
// 过期后必须重签，不能拿窗口内旧签名喂给客户端（否则 X-Amz-Expires 已过，
// 图片随时间全灭，线上实测）。
func TestPresignGetCacheExpiresWithSignatureTTL(t *testing.T) {
	provider := newTestProvider(t)
	base := time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)
	clock := base
	provider.now = func() time.Time { return clock }
	ctx := context.Background()

	first, err := provider.PresignGet(ctx, "uploads/u1/short.png", 5*time.Minute)
	if err != nil {
		t.Fatalf("PresignGet: %v", err)
	}
	if !strings.Contains(first, "X-Amz-Date=20260916T120000Z") {
		t.Fatalf("fresh sign carries sign time: %s", first)
	}

	// TTL 内（4min < 5min − margin）：缓存命中，同一 URL
	clock = base.Add(4 * time.Minute)
	cached, err := provider.PresignGet(ctx, "uploads/u1/short.png", 5*time.Minute)
	if err != nil {
		t.Fatalf("PresignGet (within TTL): %v", err)
	}
	if cached != first {
		t.Fatal("URL should stay stable while signature is servable")
	}

	// TTL 耗尽后（6min，仍在 1h 轮换窗口内）：必须重签，不能返回旧签名
	clock = base.Add(6 * time.Minute)
	second, err := provider.PresignGet(ctx, "uploads/u1/short.png", 5*time.Minute)
	if err != nil {
		t.Fatalf("PresignGet (after TTL): %v", err)
	}
	if second == first {
		t.Fatal("expired signature must be re-signed even within cache window")
	}
	if !strings.Contains(second, "X-Amz-Date=20260916T120600Z") {
		t.Fatalf("re-sign carries fresh sign time: %s", second)
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
