package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/appconfig"
)

// stubAppConfigSource 是可控的 AppConfig 源（测试用）。
type stubAppConfigSource struct {
	cfg appconfig.AppConfig
}

func (s stubAppConfigSource) Get(context.Context) appconfig.AppConfig { return s.cfg }

// recordingLoginTracker 记录 SetLimits 调用，其余方法空实现。
type recordingLoginTracker struct {
	maxFailures  int
	lockDuration time.Duration
}

func (r *recordingLoginTracker) Check(context.Context, string) error  { return nil }
func (r *recordingLoginTracker) RecordFailed(context.Context, string) {}
func (r *recordingLoginTracker) Clear(context.Context, string)        {}
func (r *recordingLoginTracker) SetLimits(maxFailures int, lock time.Duration) {
	r.maxFailures = maxFailures
	r.lockDuration = lock
}

// TestServiceReadsAuthParamsFromAppConfig 防回归（审查遗留项）：验证码 TTL
// 与登录锁定阈值从 AppConfig.Auth 读取，配置可运行期调整；未配置时回退默认。
func TestServiceReadsAuthParamsFromAppConfig(t *testing.T) {
	ctx := context.Background()

	// 未配置：回退迁移前常量。
	svc := &Service{appConfig: stubAppConfigSource{}}
	if got := svc.verificationCodeTTL(ctx); got != 10*time.Minute {
		t.Fatalf("default code TTL=%v want 10m", got)
	}
	tracker := &recordingLoginTracker{}
	svc.loginTracker = tracker
	svc.syncLoginTrackerLimits(ctx)
	if tracker.maxFailures != 5 || tracker.lockDuration != 15*time.Minute {
		t.Fatalf("default tracker limits=%d/%v want 5/15m", tracker.maxFailures, tracker.lockDuration)
	}

	// 显式配置：按配置生效。
	svc = &Service{appConfig: stubAppConfigSource{cfg: appconfig.AppConfig{Auth: appconfig.AuthConfig{
		VerificationCodeTTLMinutes: 5,
		LoginLockoutThreshold:      3,
		LoginLockoutMinutes:        30,
	}}}}
	if got := svc.verificationCodeTTL(ctx); got != 5*time.Minute {
		t.Fatalf("configured code TTL=%v want 5m", got)
	}
	tracker = &recordingLoginTracker{}
	svc.loginTracker = tracker
	svc.syncLoginTrackerLimits(ctx)
	if tracker.maxFailures != 3 || tracker.lockDuration != 30*time.Minute {
		t.Fatalf("configured tracker limits=%d/%v want 3/30m", tracker.maxFailures, tracker.lockDuration)
	}
}

// TestInMemoryLoginTrackerHonorsConfiguredLimits 防回归：tracker 按 SetLimits
// 的阈值锁定，不再固定 5 次。
func TestInMemoryLoginTrackerHonorsConfiguredLimits(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	tracker := newInMemoryLoginTracker(ctx)
	tracker.SetLimits(2, time.Minute)
	tracker.RecordFailed(ctx, "u1")
	if err := tracker.Check(ctx, "u1"); err != nil {
		t.Fatalf("1 failure must not lock: %v", err)
	}
	tracker.RecordFailed(ctx, "u1")
	if err := tracker.Check(ctx, "u1"); !errors.Is(err, ErrAccountLocked) {
		t.Fatalf("2nd failure must lock, err=%v", err)
	}
}
