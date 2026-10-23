package auth

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/config"
)

func newDeviceTestService() *Service {
	return NewServiceWithStores(
		config.Config{AuthTokenSecret: "device-eviction-test-secret-32-bytes-long"},
		NewInMemorySessionStore(),
		NewInMemoryRefreshStore(),
	)
}

// TestInMemorySessionStoreRejectsSecondActiveDeviceSession 内存实现必须与 PG
// 的部分唯一索引（迁移 057）同语义，否则降级路径会掩盖 TOCTOU 竞态。
func TestInMemorySessionStoreRejectsSecondActiveDeviceSession(t *testing.T) {
	ctx := context.Background()
	store := NewInMemorySessionStore()
	if err := store.Save(ctx, SessionRecord{SessionID: "s1", UserID: "u1", DeviceID: "d1"}); err != nil {
		t.Fatalf("save first: %v", err)
	}
	if err := store.Save(ctx, SessionRecord{SessionID: "s2", UserID: "u1", DeviceID: "d1"}); !errors.Is(err, ErrDeviceSessionConflict) {
		t.Fatalf("second same-device save err=%v want ErrDeviceSessionConflict", err)
	}
	// 其它设备/空 device_id 不参与约束。
	if err := store.Save(ctx, SessionRecord{SessionID: "s3", UserID: "u1", DeviceID: "d2"}); err != nil {
		t.Fatalf("other device save: %v", err)
	}
	if err := store.Save(ctx, SessionRecord{SessionID: "s4", UserID: "u1", DeviceID: ""}); err != nil {
		t.Fatalf("empty device save: %v", err)
	}
	// 撤销后允许同设备再次登录。
	now := time.Now().UTC()
	if err := store.Save(ctx, SessionRecord{SessionID: "s1", UserID: "u1", DeviceID: "d1", RevokedAt: &now}); err != nil {
		t.Fatalf("revoke save: %v", err)
	}
	if err := store.Save(ctx, SessionRecord{SessionID: "s5", UserID: "u1", DeviceID: "d1"}); err != nil {
		t.Fatalf("save after revoke: %v", err)
	}
}

// TestIssueSessionEvictingDeviceLeavesSingleActiveSession 核心回归：同一设备
// 连续登录（含并发）后，数据库/内存中只能有一个活跃会话。
func TestIssueSessionEvictingDeviceLeavesSingleActiveSession(t *testing.T) {
	ctx := context.Background()
	svc := newDeviceTestService()
	meta := SessionMeta{DeviceID: "device-1"}

	first, err := svc.IssueSessionWithMeta(ctx, "user_1", meta)
	if err != nil {
		t.Fatalf("first issue: %v", err)
	}
	// 直接签发第二会话必须被拒（模拟旧实现竞态窗口里「撤销检查后同时插入」）。
	if _, err := svc.IssueSessionWithMeta(ctx, "user_1", meta); !errors.Is(err, ErrDeviceSessionConflict) {
		t.Fatalf("second direct issue err=%v want ErrDeviceSessionConflict", err)
	}
	second, err := svc.IssueSessionEvictingDevice(ctx, "user_1", meta)
	if err != nil {
		t.Fatalf("evicting issue: %v", err)
	}
	sessions, err := svc.sessions.ListActiveByUser(ctx, "user_1")
	if err != nil {
		t.Fatalf("list active: %v", err)
	}
	if len(sessions) != 1 || sessions[0].SessionID != second.Principal.SessionID {
		t.Fatalf("active sessions=%+v want only %s", sessions, second.Principal.SessionID)
	}
	old, found, err := svc.sessions.Get(ctx, first.Principal.SessionID)
	if err != nil || !found || old.RevokedAt == nil {
		t.Fatalf("old session must be revoked: found=%v revoked=%v err=%v", found, old.RevokedAt, err)
	}
}

// TestIssueSessionEvictingDeviceConcurrent 并发登录同一设备：最终活跃会话必须
// 恰为一个；失败者只能是冲突哨兵（不能留下半状态）。
func TestIssueSessionEvictingDeviceConcurrent(t *testing.T) {
	ctx := context.Background()
	svc := newDeviceTestService()
	meta := SessionMeta{DeviceID: "device-1"}
	const workers = 8
	var wg sync.WaitGroup
	errs := make([]error, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, err := svc.IssueSessionEvictingDevice(ctx, "user_1", meta)
			errs[idx] = err
		}(i)
	}
	wg.Wait()
	sessions, err := svc.sessions.ListActiveByUser(ctx, "user_1")
	if err != nil {
		t.Fatalf("list active: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("concurrent logins left %d active sessions, want 1", len(sessions))
	}
	for i, err := range errs {
		if err != nil && !errors.Is(err, ErrDeviceSessionConflict) {
			t.Fatalf("worker %d unexpected error: %v", i, err)
		}
	}
}

// TestIssueSessionEvictingDeviceSkipsEmptyDeviceID 空 device_id（历史/未知
// 设备）不参与唯一约束，也不触发逐出.
func TestIssueSessionEvictingDeviceSkipsEmptyDeviceID(t *testing.T) {
	ctx := context.Background()
	svc := newDeviceTestService()
	for i := 0; i < 2; i++ {
		if _, err := svc.IssueSessionEvictingDevice(ctx, "user_1", SessionMeta{}); err != nil {
			t.Fatalf("issue %d: %v", i, err)
		}
	}
	sessions, err := svc.sessions.ListActiveByUser(ctx, "user_1")
	if err != nil {
		t.Fatalf("list active: %v", err)
	}
	if len(sessions) != 2 {
		t.Fatalf("empty-device sessions=%d want 2", len(sessions))
	}
}
