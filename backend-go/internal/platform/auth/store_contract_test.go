package auth

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/testdb"
)

// authStoreContractSeq 为每个用例生成唯一用户/会话 ID：平台 auth 存储的 PG 实现
// 直连连接池（不参与 ctx 事务，见 docs/backend-audit.md「事务上下文」残留项），
// 故契约测试用 pool + 用例级唯一 ID + 显式清理，避免污染共享库。
var authStoreContractSeq int64

type authStoreContractState struct {
	ctx       context.Context
	userID    string
	otherUser string
	seedUser  string
	sessionID string
}

func newAuthStoreContractState(seq int64) authStoreContractState {
	return authStoreContractState{
		ctx:       context.Background(),
		userID:    fmt.Sprintf("it_auth_store_u_%d", seq),
		otherUser: fmt.Sprintf("it_auth_store_o_%d", seq),
		seedUser:  fmt.Sprintf("it_auth_store_seed_%d", seq),
		sessionID: fmt.Sprintf("it_auth_store_s_%d", seq),
	}
}

func authStorePostgresNew(t *testing.T) *persistence.Factory {
	t.Helper()
	testdb.SkipIfShort(t)
	return testdb.Open(t)
}

func authStorePostgresState(t *testing.T, factory *persistence.Factory, state authStoreContractState) {
	t.Helper()
	testdb.InsertUser(t, factory, state.userID, "nick_"+state.userID)
	testdb.InsertUser(t, factory, state.otherUser, "nick_"+state.otherUser)
	testdb.InsertUser(t, factory, state.seedUser, "nick_"+state.seedUser)
	// refresh token 有 session_id 外键：预置会话行（契约用例内两个存储共用）。
	sessionStore := NewPostgresSessionStore(factory, nil)
	now := time.Now().UTC().Truncate(time.Millisecond)
	if err := sessionStore.Save(context.Background(), SessionRecord{
		SessionID:    state.sessionID,
		UserID:       state.seedUser,
		CreatedAt:    now,
		LastActiveAt: now,
		DeviceID:     "contract-device",
		DeviceType:   "web",
	}); err != nil {
		t.Fatalf("seed session: %v", err)
	}
}

func authStorePostgresCleanup(t *testing.T, factory *persistence.Factory, state authStoreContractState) func() {
	t.Helper()
	return func() {
		qdb, ok := factory.DB.(persistence.QueryableDB)
		if !ok {
			return
		}
		ctx := context.Background()
		users := []string{state.userID, state.otherUser, state.seedUser}
		_ = qdb.Exec(ctx, "DELETE FROM auth_refresh_tokens WHERE user_id = ANY($1::text[])", users)
		_ = qdb.Exec(ctx, "DELETE FROM auth_sessions WHERE user_id = ANY($1::text[])", users)
		_ = qdb.Exec(ctx, "DELETE FROM users WHERE user_id = ANY($1::text[])", users)
	}
}

func TestSessionStoreContract(t *testing.T) {
	base := time.Now().UTC().Truncate(time.Millisecond)

	impls := []testdb.ContractImpl[SessionStore, authStoreContractState]{
		{
			Name: "memory",
			New: func(t *testing.T) (SessionStore, authStoreContractState, func()) {
				seq := atomic.AddInt64(&authStoreContractSeq, 1)
				return NewInMemorySessionStore(), newAuthStoreContractState(seq), nil
			},
		},
		{
			Name: "postgres",
			New: func(t *testing.T) (SessionStore, authStoreContractState, func()) {
				seq := atomic.AddInt64(&authStoreContractSeq, 1)
				state := newAuthStoreContractState(seq)
				factory := authStorePostgresNew(t)
				authStorePostgresState(t, factory, state)
				return NewPostgresSessionStore(factory, nil), state, authStorePostgresCleanup(t, factory, state)
			},
		},
	}

	session := func(state authStoreContractState, sessionID string, lastActive time.Time) SessionRecord {
		return SessionRecord{
			SessionID:    sessionID,
			UserID:       state.userID,
			CreatedAt:    base,
			LastActiveAt: lastActive,
			DeviceType:   "web",
			DeviceName:   "Chrome",
			IPAddress:    "203.0.113.9",
			IPLocation:   "CN",
		}
	}

	testdb.RunContract(t, impls, []testdb.ContractCase[SessionStore, authStoreContractState]{
		{
			Name: "save and get round-trips fields",
			Run: func(t *testing.T, store SessionStore, state authStoreContractState) {
				record := session(state, "sess_roundtrip", base)
				record.DeviceID = "device-rt"
				if err := store.Save(state.ctx, record); err != nil {
					t.Fatalf("save: %v", err)
				}
				got, found, err := store.Get(state.ctx, "sess_roundtrip")
				if err != nil || !found {
					t.Fatalf("get: found=%v err=%v", found, err)
				}
				if got.UserID != state.userID || got.DeviceID != "device-rt" || got.DeviceType != "web" || got.DeviceName != "Chrome" {
					t.Fatalf("record = %#v", got)
				}
				if got.IPAddress != "203.0.113.9" || got.IPLocation != "CN" {
					t.Fatalf("ip fields = %#v", got)
				}
				if got.RevokedAt != nil {
					t.Fatalf("revoked_at = %#v", got.RevokedAt)
				}
				if !got.CreatedAt.Equal(base) {
					t.Fatalf("created_at = %v, want %v", got.CreatedAt, base)
				}
				if _, found, err := store.Get(state.ctx, "sess_missing"); err != nil || found {
					t.Fatalf("get missing: found=%v err=%v", found, err)
				}
			},
		},
		{
			Name: "touch refreshes last active and ignores revoked/missing",
			Run: func(t *testing.T, store SessionStore, state authStoreContractState) {
				stale := base
				if err := store.Save(state.ctx, session(state, "sess_touch", stale)); err != nil {
					t.Fatalf("save: %v", err)
				}
				if err := store.Touch(state.ctx, "sess_touch"); err != nil {
					t.Fatalf("touch: %v", err)
				}
				got, found, err := store.Get(state.ctx, "sess_touch")
				if err != nil || !found {
					t.Fatalf("get after touch: found=%v err=%v", found, err)
				}
				if !got.LastActiveAt.After(stale) {
					t.Fatalf("last_active_at = %v, want after %v", got.LastActiveAt, stale)
				}
				if err := store.Touch(state.ctx, "sess_missing"); err != nil {
					t.Fatalf("touch missing: %v", err)
				}
				revoked := session(state, "sess_touch_revoked", stale)
				revokedAt := stale.Add(time.Minute)
				revoked.RevokedAt = &revokedAt
				if err := store.Save(state.ctx, revoked); err != nil {
					t.Fatalf("save revoked: %v", err)
				}
				before, _, err := store.Get(state.ctx, "sess_touch_revoked")
				if err != nil {
					t.Fatalf("get revoked: %v", err)
				}
				if err := store.Touch(state.ctx, "sess_touch_revoked"); err != nil {
					t.Fatalf("touch revoked: %v", err)
				}
				after, _, err := store.Get(state.ctx, "sess_touch_revoked")
				if err != nil {
					t.Fatalf("get revoked after touch: %v", err)
				}
				if !after.LastActiveAt.Equal(before.LastActiveAt) {
					t.Fatalf("revoked session touched: %v -> %v", before.LastActiveAt, after.LastActiveAt)
				}
			},
		},
		{
			Name: "list active excludes revoked and orders by recency",
			Run: func(t *testing.T, store SessionStore, state authStoreContractState) {
				newest := session(state, "sess_newest", base)
				newest.DeviceID = "device-a"
				middle := session(state, "sess_middle", base.Add(-time.Hour))
				middle.DeviceID = "device-b"
				oldest := session(state, "sess_oldest", base.Add(-2*time.Hour))
				oldest.DeviceID = "device-c"
				for _, record := range []SessionRecord{newest, middle, oldest} {
					if err := store.Save(state.ctx, record); err != nil {
						t.Fatalf("save %s: %v", record.SessionID, err)
					}
				}
				if _, err := store.Revoke(state.ctx, "sess_middle"); err != nil {
					t.Fatalf("revoke middle: %v", err)
				}
				items, err := store.ListActiveByUser(state.ctx, state.userID)
				if err != nil {
					t.Fatalf("list active: %v", err)
				}
				if len(items) != 2 || items[0].SessionID != "sess_newest" || items[1].SessionID != "sess_oldest" {
					t.Fatalf("active sessions = %#v", items)
				}
				others, err := store.ListActiveByUser(state.ctx, state.otherUser)
				if err != nil {
					t.Fatalf("list active other: %v", err)
				}
				if len(others) != 0 {
					t.Fatalf("other user sessions = %#v", others)
				}
			},
		},
		{
			Name: "revoke is idempotent and reports missing",
			Run: func(t *testing.T, store SessionStore, state authStoreContractState) {
				if err := store.Save(state.ctx, session(state, "sess_revoke", base)); err != nil {
					t.Fatalf("save: %v", err)
				}
				changed, err := store.Revoke(state.ctx, "sess_revoke")
				if err != nil || !changed {
					t.Fatalf("first revoke = %v err=%v", changed, err)
				}
				if changed, err := store.Revoke(state.ctx, "sess_revoke"); err != nil || changed {
					t.Fatalf("second revoke = %v err=%v", changed, err)
				}
				if changed, err := store.Revoke(state.ctx, "sess_missing"); err != nil || changed {
					t.Fatalf("revoke missing = %v err=%v", changed, err)
				}
				got, found, err := store.Get(state.ctx, "sess_revoke")
				if err != nil || !found {
					t.Fatalf("get revoked: found=%v err=%v", found, err)
				}
				if got.RevokedAt == nil {
					t.Fatalf("revoked_at = nil, want set")
				}
			},
		},
		{
			Name: "revoke others excludes current and already revoked",
			Run: func(t *testing.T, store SessionStore, state authStoreContractState) {
				current := session(state, "sess_current", base)
				current.DeviceID = "device-cur"
				otherOne := session(state, "sess_other_1", base.Add(-time.Hour))
				otherOne.DeviceID = "device-o1"
				otherTwo := session(state, "sess_other_2", base.Add(-2*time.Hour))
				otherTwo.DeviceID = "device-o2"
				for _, record := range []SessionRecord{current, otherOne, otherTwo} {
					if err := store.Save(state.ctx, record); err != nil {
						t.Fatalf("save %s: %v", record.SessionID, err)
					}
				}
				count, err := store.RevokeOthers(state.ctx, state.userID, "sess_current")
				if err != nil || count != 2 {
					t.Fatalf("revoke others = %d err=%v", count, err)
				}
				if count, err := store.RevokeOthers(state.ctx, state.userID, "sess_current"); err != nil || count != 0 {
					t.Fatalf("revoke others again = %d err=%v", count, err)
				}
				items, err := store.ListActiveByUser(state.ctx, state.userID)
				if err != nil {
					t.Fatalf("list active: %v", err)
				}
				if len(items) != 1 || items[0].SessionID != "sess_current" {
					t.Fatalf("active after revoke others = %#v", items)
				}
			},
		},
		{
			Name: "same device active session conflict is reported",
			Run: func(t *testing.T, store SessionStore, state authStoreContractState) {
				first := session(state, "sess_dev_1", base)
				first.DeviceID = "device-conflict"
				if err := store.Save(state.ctx, first); err != nil {
					t.Fatalf("save first: %v", err)
				}
				second := session(state, "sess_dev_2", base.Add(-time.Minute))
				second.DeviceID = "device-conflict"
				if err := store.Save(state.ctx, second); !errors.Is(err, ErrDeviceSessionConflict) {
					t.Fatalf("save second err = %v, want ErrDeviceSessionConflict", err)
				}
			},
		},
	})
}

func TestRefreshTokenStoreContract(t *testing.T) {
	base := time.Now().UTC().Truncate(time.Millisecond)

	impls := []testdb.ContractImpl[RefreshTokenStore, authStoreContractState]{
		{
			Name: "memory",
			New: func(t *testing.T) (RefreshTokenStore, authStoreContractState, func()) {
				seq := atomic.AddInt64(&authStoreContractSeq, 1)
				return NewInMemoryRefreshStore(), newAuthStoreContractState(seq), nil
			},
		},
		{
			Name: "postgres",
			New: func(t *testing.T) (RefreshTokenStore, authStoreContractState, func()) {
				seq := atomic.AddInt64(&authStoreContractSeq, 1)
				state := newAuthStoreContractState(seq)
				factory := authStorePostgresNew(t)
				authStorePostgresState(t, factory, state)
				return NewPostgresRefreshStore(factory, nil), state, authStorePostgresCleanup(t, factory, state)
			},
		},
	}

	refresh := func(state authStoreContractState, token string) RefreshSession {
		return RefreshSession{
			UserID:       state.userID,
			SessionID:    state.sessionID,
			RefreshToken: token,
			ExpiresAt:    base.Add(30 * 24 * time.Hour),
		}
	}

	testdb.RunContract(t, impls, []testdb.ContractCase[RefreshTokenStore, authStoreContractState]{
		{
			Name: "save and get round-trips fields",
			Run: func(t *testing.T, store RefreshTokenStore, state authStoreContractState) {
				if err := store.Save(state.ctx, refresh(state, "refresh_roundtrip")); err != nil {
					t.Fatalf("save: %v", err)
				}
				got, found, err := store.Get(state.ctx, "refresh_roundtrip")
				if err != nil || !found {
					t.Fatalf("get: found=%v err=%v", found, err)
				}
				if got.UserID != state.userID || got.SessionID != state.sessionID || got.RotatedTo != "" {
					t.Fatalf("refresh = %#v", got)
				}
				if !got.ExpiresAt.Equal(base.Add(30 * 24 * time.Hour)) {
					t.Fatalf("expires_at = %v", got.ExpiresAt)
				}
				if _, found, err := store.Get(state.ctx, "refresh_missing"); err != nil || found {
					t.Fatalf("get missing: found=%v err=%v", found, err)
				}
			},
		},
		{
			Name: "rotate links old token and creates next chain token",
			Run: func(t *testing.T, store RefreshTokenStore, state authStoreContractState) {
				if err := store.Save(state.ctx, refresh(state, "refresh_old")); err != nil {
					t.Fatalf("save old: %v", err)
				}
				current, found, err := store.Get(state.ctx, "refresh_old")
				if err != nil || !found {
					t.Fatalf("get old: found=%v err=%v", found, err)
				}
				next := refresh(state, "refresh_next")
				next.GraceTokenEnc = []byte("encrypted-next-chain-token")
				if err := store.Rotate(state.ctx, "refresh_old", current, next); err != nil {
					t.Fatalf("rotate: %v", err)
				}
				rotated, found, err := store.Get(state.ctx, "refresh_old")
				if err != nil || !found {
					t.Fatalf("get rotated: found=%v err=%v", found, err)
				}
				if rotated.RotatedTo != "refresh_next" {
					t.Fatalf("rotated_to = %q", rotated.RotatedTo)
				}
				if rotated.GraceUntil.IsZero() || rotated.GraceUntil.Before(time.Now()) {
					t.Fatalf("grace_until = %v", rotated.GraceUntil)
				}
				if string(rotated.GraceTokenEnc) != "encrypted-next-chain-token" {
					t.Fatalf("grace_token_enc = %q", rotated.GraceTokenEnc)
				}
				nextRecord, found, err := store.Get(state.ctx, "refresh_next")
				if err != nil || !found {
					t.Fatalf("get next: found=%v err=%v", found, err)
				}
				if nextRecord.RotatedTo != "" {
					t.Fatalf("next rotated_to = %q", nextRecord.RotatedTo)
				}
				// 并发败者/复用者：同一旧 token 二次轮换必须报冲突（不得分叉出第二条链）。
				if err := store.Rotate(state.ctx, "refresh_old", rotated, refresh(state, "refresh_third")); !errors.Is(err, ErrRefreshRotationConflict) {
					t.Fatalf("second rotate err = %v, want ErrRefreshRotationConflict", err)
				}
				// 不存在的旧 token：PG 守卫 0 行 → 冲突；内存实现必须一致。
				if err := store.Rotate(state.ctx, "refresh_missing", RefreshSession{}, refresh(state, "refresh_other")); !errors.Is(err, ErrRefreshRotationConflict) {
					t.Fatalf("rotate missing err = %v, want ErrRefreshRotationConflict", err)
				}
			},
		},
		{
			Name: "save overwrites existing token row",
			Run: func(t *testing.T, store RefreshTokenStore, state authStoreContractState) {
				first := refresh(state, "refresh_upsert")
				if err := store.Save(state.ctx, first); err != nil {
					t.Fatalf("save first: %v", err)
				}
				second := refresh(state, "refresh_upsert")
				second.ExpiresAt = base.Add(60 * 24 * time.Hour)
				if err := store.Save(state.ctx, second); err != nil {
					t.Fatalf("save second: %v", err)
				}
				got, found, err := store.Get(state.ctx, "refresh_upsert")
				if err != nil || !found {
					t.Fatalf("get: found=%v err=%v", found, err)
				}
				if !got.ExpiresAt.Equal(second.ExpiresAt) {
					t.Fatalf("expires_at = %v, want %v", got.ExpiresAt, second.ExpiresAt)
				}
			},
		},
	})
}
