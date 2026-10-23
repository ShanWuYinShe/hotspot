package auth

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/testdb"
)

// TestPostgresIT_DeviceSessionUniqueness 验证迁移 057 的「同设备至多一个活跃
// 会话」部分唯一索引在真实库生效，并且应用层「插入冲突 → 撤销旧会话 → 重试」
// 的逐出流程最终只留一个活跃会话（审计未修限制 1 的 TOCTOU 修复）。
//
// 不走 testdb.TxScope：唯一冲突会中止当前事务（25P02），而生产路径是逐条
// autocommit；这里用独立注册用户 + defer 清理保证可重复执行。
func TestPostgresIT_DeviceSessionUniqueness(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	ctx := context.Background()

	// 迁移已创建部分唯一索引（启动 EnsureSchema 会跑全部迁移）。
	queryable, ok := factory.DB.(persistence.QueryableDB)
	if !ok {
		t.Fatalf("factory DB is not QueryableDB: %T", factory.DB)
	}
	var indexName string
	if err := queryable.QueryRow(ctx,
		"SELECT indexname FROM pg_indexes WHERE tablename = 'auth_sessions' AND indexname = 'uq_auth_sessions_active_device'",
	).Scan(&indexName); err != nil || indexName == "" {
		t.Fatalf("partial unique index missing: index=%q err=%v", indexName, err)
	}

	repo := NewPostgresRepository(factory)
	loginID := fmt.Sprintf("device-it-%d@example.com", time.Now().UnixNano())
	account, err := repo.Register(ctx, RegisterParams{
		LoginID:  loginID,
		Nickname: "Device IT",
		Password: "secret123",
		Defaults: ProfileDefaults{
			LocationName: "Test City",
			Latitude:     35.0,
			Longitude:    112.0,
			CreatedAt:    time.Now().UTC(),
		},
		RegisterIP:       "203.0.113.10",
		RegisterLocation: "US",
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	t.Cleanup(func() {
		// users 删除级联清理 auth_sessions/auth_accounts 等子表。
		_ = queryable.Exec(context.Background(), "DELETE FROM users WHERE user_id = $1", account.UserID)
	})

	svc := platformauth.NewServiceWithStores(
		config.Config{AuthTokenSecret: "device-eviction-it-secret-at-least-32-bytes"},
		platformauth.NewPostgresSessionStore(factory, nil),
		platformauth.NewInMemoryRefreshStore(),
	)
	meta := platformauth.SessionMeta{DeviceID: "device-it-1"}
	if _, err := svc.IssueSessionWithMeta(ctx, account.UserID, meta); err != nil {
		t.Fatalf("first session: %v", err)
	}
	// 同设备第二个活跃会话必须被索引拦截（这是 TOCTOU 的最终防线）。
	if _, err := svc.IssueSessionWithMeta(ctx, account.UserID, meta); !errors.Is(err, platformauth.ErrDeviceSessionConflict) {
		t.Fatalf("second session err=%v want ErrDeviceSessionConflict", err)
	}
	// 应用层逐出流程：撤销旧会话后重试成功。
	if _, err := svc.IssueSessionEvictingDevice(ctx, account.UserID, meta); err != nil {
		t.Fatalf("IssueSessionEvictingDevice: %v", err)
	}
	active := 0
	for _, session := range svc.ListSessions(ctx, account.UserID, "") {
		if session.DeviceID == "device-it-1" {
			active++
		}
	}
	if active != 1 {
		t.Fatalf("active sessions for device=%d want 1", active)
	}
}
