package auth

import (
	"context"
	"errors"
	"testing"

	"hotspot/backend-go/internal/platform/config"
)

// countingChecker 记录调用次数，并允许测试中途翻转封禁事实——
// 用于验证缓存命中不穿透、失效后重新回源。
type countingChecker struct {
	calls  map[string]int
	banned map[string]bool
	errFor map[string]error
}

func (c *countingChecker) IsBanned(_ context.Context, userID string) (bool, error) {
	c.calls[userID]++
	if err, ok := c.errFor[userID]; ok {
		return false, err
	}
	return c.banned[userID], nil
}

func TestIsUserBannedCachesCheckerResult(t *testing.T) {
	checker := &countingChecker{calls: map[string]int{}, banned: map[string]bool{"u_ban": true}}
	service := NewService(config.Config{AuthTokenSecret: "test-secret"})
	service.SetBanChecker(checker)

	for i := 0; i < 5; i++ {
		if !service.IsUserBanned(context.Background(), "u_ban") {
			t.Fatalf("call %d: expected banned", i+1)
		}
		if service.IsUserBanned(context.Background(), "u_ok") {
			t.Fatalf("call %d: expected not banned", i+1)
		}
	}
	if checker.calls["u_ban"] != 1 || checker.calls["u_ok"] != 1 {
		t.Fatalf("expected checker consulted once per user, got %d/%d", checker.calls["u_ban"], checker.calls["u_ok"])
	}
}

func TestInvalidateBanCacheForcesRecheck(t *testing.T) {
	checker := &countingChecker{calls: map[string]int{}, banned: map[string]bool{"u1": false}}
	service := NewService(config.Config{AuthTokenSecret: "test-secret"})
	service.SetBanChecker(checker)

	if service.IsUserBanned(context.Background(), "u1") {
		t.Fatal("expected initially not banned")
	}
	// 封禁事实在缓存窗口内变化（解封的逆操作：直接改库绕过 admin）——
	// 缓存仍返回旧值
	checker.banned["u1"] = true
	if service.IsUserBanned(context.Background(), "u1") {
		t.Fatal("expected stale cached value within TTL window")
	}
	// admin 状态变更走主动失效：下一个请求立即回到真实结果
	service.InvalidateBanCache("u1")
	if !service.IsUserBanned(context.Background(), "u1") {
		t.Fatal("expected fresh banned=true after invalidation")
	}
	if checker.calls["u1"] != 2 {
		t.Fatalf("expected exactly 2 checker calls, got %d", checker.calls["u1"])
	}
}

func TestBanCheckErrorsAreNotCached(t *testing.T) {
	checker := &countingChecker{calls: map[string]int{}, errFor: map[string]error{"u_err": errors.New("db down")}}
	service := NewService(config.Config{AuthTokenSecret: "test-secret"})
	service.SetBanChecker(checker)

	// fail-open 且不落缓存：DB 恢复后下一个请求立即回到真实结果
	for i := 0; i < 3; i++ {
		if service.IsUserBanned(context.Background(), "u_err") {
			t.Fatalf("call %d: expected fail-open", i+1)
		}
	}
	if checker.calls["u_err"] != 3 {
		t.Fatalf("expected erroring user re-checked every call, got %d", checker.calls["u_err"])
	}

	checker.errFor = map[string]error{}
	if service.IsUserBanned(context.Background(), "u_err") {
		t.Fatal("expected recovery to real result after DB heals")
	}
}

func TestInvalidateBanCacheWithoutCheckerOrEntriesIsSafe(t *testing.T) {
	service := NewService(config.Config{AuthTokenSecret: "test-secret"})
	// 未配置 checker / 从未缓存时调用失效不得 panic
	service.InvalidateBanCache("u1")
	service.InvalidateBanCache()
}
