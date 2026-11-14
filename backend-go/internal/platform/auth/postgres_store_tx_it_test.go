package auth

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/testdb"
)

// 事务感知回归测试：PostgresSessionStore / PostgresRefreshStore 的读写必须经
// persistence.RepoQueryable 加入 ctx 事务（事务上下文一致性约定，见
// docs/backend-audit.md）。修复前两者直连连接池：admin 封禁在事务内逐条撤销
// 会话（RevokeSessionAsAdmin）会立即提交，事务后续步骤失败回滚后出现
// 「用户没被封但全端会话已失效」。
//
// 隔离方式同 store_contract_test.go：连接池 + 用例级唯一 ID + 显式清理
// （RunInTransaction 需要真实 *PostgresDB 池，无法用 BeginScope 的事务工厂）。

var authStoreTxITSeq int64

type authStoreTxITState struct {
	factory   *persistence.Factory
	userID    string
	sessionID string
}

// newAuthStoreTxITState 在真实库上建用户 + 活跃会话，并注册用例级清理。
func newAuthStoreTxITState(t *testing.T) authStoreTxITState {
	t.Helper()
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	seq := atomic.AddInt64(&authStoreTxITSeq, 1)
	state := authStoreTxITState{
		factory:   factory,
		userID:    fmt.Sprintf("it_auth_tx_u_%d", seq),
		sessionID: fmt.Sprintf("it_auth_tx_s_%d", seq),
	}
	testdb.InsertUser(t, factory, state.userID, "nick_"+state.userID)
	now := time.Now().UTC().Truncate(time.Millisecond)
	if err := NewPostgresSessionStore(factory, nil).Save(context.Background(), SessionRecord{
		SessionID:    state.sessionID,
		UserID:       state.userID,
		CreatedAt:    now,
		LastActiveAt: now,
		DeviceID:     "it-tx-device",
	}); err != nil {
		t.Fatalf("seed session: %v", err)
	}
	t.Cleanup(func() {
		qdb, ok := factory.DB.(persistence.QueryableDB)
		if !ok {
			return
		}
		ctx := context.Background()
		_ = qdb.Exec(ctx, "DELETE FROM auth_refresh_tokens WHERE user_id = $1", state.userID)
		_ = qdb.Exec(ctx, "DELETE FROM auth_sessions WHERE user_id = $1", state.userID)
		_ = qdb.Exec(ctx, "DELETE FROM users WHERE user_id = $1", state.userID)
	})
	return state
}

// assertSessionActive 用池（无事务）视角断言会话仍未被撤销。
func assertSessionActive(t *testing.T, factory *persistence.Factory, sessionID string) {
	t.Helper()
	record, found, err := NewPostgresSessionStore(factory, nil).Get(context.Background(), sessionID)
	if err != nil || !found {
		t.Fatalf("get session %s: found=%v err=%v", sessionID, found, err)
	}
	if record.RevokedAt != nil {
		t.Fatal("事务回滚后会话仍处于撤销状态：撤销写逃出了事务（事务感知回归）")
	}
}

func TestPostgresSessionStoreRevokeRollsBackWithTransaction(t *testing.T) {
	state := newAuthStoreTxITState(t)
	store := NewPostgresSessionStore(state.factory, nil)
	rollbackErr := errors.New("it: force rollback")
	err := persistence.RunInTransaction(context.Background(), state.factory, func(txCtx context.Context) error {
		revoked, revokeErr := store.Revoke(txCtx, state.sessionID)
		if revokeErr != nil {
			return revokeErr
		}
		if !revoked {
			return errors.New("事务内 Revoke 未命中活跃会话")
		}
		// read-your-writes：事务内必须看得到本事务的撤销。
		record, found, getErr := store.Get(txCtx, state.sessionID)
		if getErr != nil {
			return getErr
		}
		if !found || record.RevokedAt == nil {
			return fmt.Errorf("事务内看不到本事务撤销: found=%v revoked_at=%v", found, record.RevokedAt)
		}
		return rollbackErr
	})
	if !errors.Is(err, rollbackErr) {
		t.Fatalf("RunInTransaction 返回 %v，期望回滚哨兵 %v", err, rollbackErr)
	}
	assertSessionActive(t, state.factory, state.sessionID)
}

func TestPostgresSessionStoreRevokeCommitsWithTransaction(t *testing.T) {
	state := newAuthStoreTxITState(t)
	store := NewPostgresSessionStore(state.factory, nil)
	if err := persistence.RunInTransaction(context.Background(), state.factory, func(txCtx context.Context) error {
		revoked, revokeErr := store.Revoke(txCtx, state.sessionID)
		if revokeErr != nil {
			return revokeErr
		}
		if !revoked {
			return errors.New("事务内 Revoke 未命中活跃会话")
		}
		return nil
	}); err != nil {
		t.Fatalf("RunInTransaction: %v", err)
	}
	record, found, err := store.Get(context.Background(), state.sessionID)
	if err != nil || !found {
		t.Fatalf("提交后 Get: found=%v err=%v", found, err)
	}
	if record.RevokedAt == nil {
		t.Fatal("事务提交后会话未撤销")
	}
}

// TestRevokeSessionAsAdminRollsBackWithTransaction 复现问题现场：admin 封禁在
// 事务内调用 RevokeSessionAsAdmin（生产装配经 CachedSessionStore 装饰），
// 事务失败回滚后会话必须仍然有效。
func TestRevokeSessionAsAdminRollsBackWithTransaction(t *testing.T) {
	state := newAuthStoreTxITState(t)
	svc := NewServiceWithStores(config.Config{},
		NewCachedSessionStore(NewPostgresSessionStore(state.factory, nil), 0, 0),
		NewPostgresRefreshStore(state.factory, nil),
	)
	rollbackErr := errors.New("it: force rollback")
	err := persistence.RunInTransaction(context.Background(), state.factory, func(txCtx context.Context) error {
		revoked, revokeErr := svc.RevokeSessionAsAdmin(txCtx, state.sessionID)
		if revokeErr != nil {
			return revokeErr
		}
		if !revoked {
			return errors.New("事务内 RevokeSessionAsAdmin 未命中活跃会话")
		}
		return rollbackErr
	})
	if !errors.Is(err, rollbackErr) {
		t.Fatalf("RunInTransaction 返回 %v，期望回滚哨兵 %v", err, rollbackErr)
	}
	assertSessionActive(t, state.factory, state.sessionID)
}

func TestPostgresRefreshStoreWritesJoinTransaction(t *testing.T) {
	state := newAuthStoreTxITState(t)
	store := NewPostgresRefreshStore(state.factory, nil)
	token := "it_auth_tx_refresh_" + state.sessionID
	session := RefreshSession{
		UserID:       state.userID,
		SessionID:    state.sessionID,
		RefreshToken: token,
		ExpiresAt:    time.Now().UTC().Add(time.Hour),
	}

	rollbackErr := errors.New("it: force rollback")
	err := persistence.RunInTransaction(context.Background(), state.factory, func(txCtx context.Context) error {
		if saveErr := store.Save(txCtx, session); saveErr != nil {
			return saveErr
		}
		if _, found, getErr := store.Get(txCtx, token); getErr != nil || !found {
			return fmt.Errorf("事务内读不到本事务写入的刷新令牌: found=%v err=%v", found, getErr)
		}
		return rollbackErr
	})
	if !errors.Is(err, rollbackErr) {
		t.Fatalf("RunInTransaction 返回 %v，期望回滚哨兵 %v", err, rollbackErr)
	}
	if _, found, getErr := store.Get(context.Background(), token); getErr != nil || found {
		t.Fatalf("事务回滚后刷新令牌仍存在: found=%v err=%v", found, getErr)
	}

	// 提交路径：事务内写入必须持久化。
	if commitErr := persistence.RunInTransaction(context.Background(), state.factory, func(txCtx context.Context) error {
		return store.Save(txCtx, session)
	}); commitErr != nil {
		t.Fatalf("提交事务写入刷新令牌: %v", commitErr)
	}
	if _, found, getErr := store.Get(context.Background(), token); getErr != nil || !found {
		t.Fatalf("事务提交后刷新令牌不存在: found=%v err=%v", found, getErr)
	}
}
