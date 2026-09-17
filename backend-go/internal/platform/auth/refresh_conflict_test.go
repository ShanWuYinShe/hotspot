package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/persistence"
)

// conflictRefreshStore 模拟并发轮换败者视角：Rotate 恒返回
// ErrRefreshRotationConflict（UPDATE 守卫 0 行命中），Get 按预置表返回记录，
// 用于驱动 RefreshSession 的「冲突后重读走宽限期分支」。
type conflictRefreshStore struct{ records map[string]RefreshSession }

func (s *conflictRefreshStore) Save(_ context.Context, session RefreshSession) error {
	s.records[session.RefreshToken] = session
	return nil
}
func (s *conflictRefreshStore) Get(_ context.Context, token string) (RefreshSession, bool, error) {
	session, ok := s.records[token]
	return session, ok, nil
}
func (s *conflictRefreshStore) Rotate(_ context.Context, _ string, _, _ RefreshSession) error {
	return ErrRefreshRotationConflict
}

func newConflictService(graceUntil time.Time) (*Service, *conflictRefreshStore) {
	sessions := NewInMemorySessionStore()
	now := time.Now().UTC()
	if err := sessions.Save(context.Background(), SessionRecord{
		SessionID:    "sess_conflict",
		UserID:       "user_1",
		CreatedAt:    now,
		LastActiveAt: now,
	}); err != nil {
		panic(err)
	}
	// 存储键为 token 哈希（migration 053 起全表口径一致）；被轮换的旧行
	// 携带新链 token 的托管密文，供宽限期分支解密回明文。
	escrow, err := NewTokenEscrow("test-secret")
	if err != nil {
		panic(err)
	}
	sealed, err := escrow.Seal("winner_token")
	if err != nil {
		panic(err)
	}
	refreshes := &conflictRefreshStore{records: map[string]RefreshSession{
		HashRefreshToken("old_token"): {
			UserID:        "user_1",
			SessionID:     "sess_conflict",
			RefreshToken:  HashRefreshToken("old_token"),
			ExpiresAt:     now.Add(time.Hour),
			RotatedTo:     HashRefreshToken("winner_token"),
			GraceUntil:    graceUntil,
			GraceTokenEnc: sealed,
		},
		HashRefreshToken("winner_token"): {
			UserID:       "user_1",
			SessionID:    "sess_conflict",
			RefreshToken: HashRefreshToken("winner_token"),
			ExpiresAt:    now.Add(24 * time.Hour),
		},
	}}
	return NewServiceWithStores(config.Config{AuthTokenSecret: "test-secret"}, sessions, refreshes), refreshes
}

// TestRefreshSessionRotationConflictReturnsWinnerChain 验证冲突分支：
// Rotate 败者重读 store 后，在宽限期内拿到获胜链的 refresh token 而非报错。
func TestRefreshSessionRotationConflictReturnsWinnerChain(t *testing.T) {
	service, _ := newConflictService(time.Now().UTC().Add(RefreshRotationGrace))
	result, err := service.RefreshSession(context.Background(), "old_token")
	if err != nil {
		t.Fatalf("refresh within grace window: %v", err)
	}
	if result.RefreshToken != "winner_token" {
		t.Fatalf("expected winner chain token, got %q", result.RefreshToken)
	}
	if result.Principal.UserID != "user_1" {
		t.Fatalf("unexpected principal: %#v", result.Principal)
	}
}

// TestRefreshSessionRotationConflictAfterGraceWindowRejects 验证宽限期
// 过后旧 token 被拒绝（复用窗口关闭）。
func TestRefreshSessionRotationConflictAfterGraceWindowRejects(t *testing.T) {
	service, _ := newConflictService(time.Now().UTC().Add(-time.Second))
	if _, err := service.RefreshSession(context.Background(), "old_token"); err == nil {
		t.Fatal("expected error after grace window closed")
	}
}

// TestPostgresRefreshStoreRotateReturnsConflictOnNoRows 用假后端驱动
// queryRotateRefreshToken 的 ErrNoRows → ErrRefreshRotationConflict 映射
// （UPDATE rotated_to IS NULL 守卫 0 行命中路径）。
func TestPostgresRefreshStoreRotateReturnsConflictOnNoRows(t *testing.T) {
	backend := &authBackendStub{sessions: map[string]SessionRecord{}, refresh: map[string]RefreshSession{}}
	store := NewPostgresRefreshStore(newAuthFactory(backend), nil)
	// 旧 token 从未保存 → 守卫 0 行命中。
	err := store.Rotate(context.Background(), "missing", RefreshSession{}, RefreshSession{RefreshToken: "next"})
	if !errors.Is(err, ErrRefreshRotationConflict) {
		t.Fatalf("expected ErrRefreshRotationConflict, got %v", err)
	}
}

// TestPostgresRefreshStoreRotateConflictWhenAlreadyRotated 验证已被轮换的
// 旧 token 二次轮换同样命中守卫（并发败者）。
func TestPostgresRefreshStoreRotateConflictWhenAlreadyRotated(t *testing.T) {
	now := time.Now().UTC()
	backend := &authBackendStub{
		sessions: map[string]SessionRecord{},
		refresh: map[string]RefreshSession{
			"old": {
				UserID: "user_1", SessionID: "sess_1", RefreshToken: "old",
				ExpiresAt: now.Add(time.Hour), RotatedTo: "taken",
			},
		},
	}
	store := NewPostgresRefreshStore(newAuthFactory(backend), nil)
	err := store.Rotate(context.Background(), "old", RefreshSession{}, RefreshSession{RefreshToken: "next"})
	if !errors.Is(err, ErrRefreshRotationConflict) {
		t.Fatalf("expected ErrRefreshRotationConflict, got %v", err)
	}
}

// TestPostgresRefreshStoreGetParsesRotatedFields 验证 Get 正确回填
// RotatedTo/GraceUntil 可空列。
func TestPostgresRefreshStoreGetParsesRotatedFields(t *testing.T) {
	now := time.Now().UTC()
	grace := now.Add(RefreshRotationGrace)
	backend := &authBackendStub{
		sessions: map[string]SessionRecord{},
		refresh: map[string]RefreshSession{
			"old": {
				UserID: "user_1", SessionID: "sess_1", RefreshToken: "old",
				ExpiresAt: now.Add(time.Hour), RotatedTo: "next", GraceUntil: grace,
			},
		},
	}
	store := NewPostgresRefreshStore(newAuthFactory(backend), nil)
	session, ok, err := store.Get(context.Background(), "old")
	if err != nil || !ok {
		t.Fatalf("get: ok=%v err=%v", ok, err)
	}
	if session.RotatedTo != "next" || !session.GraceUntil.Equal(grace) {
		t.Fatalf("unexpected parsed session: %#v", session)
	}
}

// TestPostgresRefreshStoreFallsBackWithoutFactory 验证未接 PG 时回落内存
// 实现（dev/test 默认路径）。
func TestPostgresRefreshStoreFallsBackWithoutFactory(t *testing.T) {
	fallback := NewInMemoryRefreshStore()
	store := NewPostgresRefreshStore(&persistence.Factory{}, fallback)
	if err := store.Save(context.Background(), RefreshSession{RefreshToken: "t", ExpiresAt: time.Now().Add(time.Hour)}); err != nil {
		t.Fatalf("fallback save: %v", err)
	}
	if _, ok, _ := store.Get(context.Background(), "t"); !ok {
		t.Fatal("expected fallback store to hold token")
	}
}

// TestSetAndClearAuthCookies 验证双 cookie 的名称、路径与 MaxAge 语义。
func TestSetAndClearAuthCookies(t *testing.T) {
	recorder := httptest.NewRecorder()
	SetAuthCookies(recorder, "at", time.Minute, "rt", time.Hour)
	cookies := recorder.Result().Cookies()
	if len(cookies) != 2 {
		t.Fatalf("expected 2 cookies, got %d", len(cookies))
	}
	byName := map[string]*http.Cookie{}
	for _, c := range cookies {
		byName[c.Name] = c
	}
	access, ok := byName[AccessTokenCookieName]
	if !ok || access.Value != "at" || access.Path != "/" || !access.HttpOnly || access.MaxAge != 60 {
		t.Fatalf("unexpected access cookie: %#v", access)
	}
	refresh, ok := byName[RefreshTokenCookieName]
	if !ok || refresh.Value != "rt" || refresh.Path != RefreshCookiePath || refresh.MaxAge != 3600 {
		t.Fatalf("unexpected refresh cookie: %#v", refresh)
	}

	clearRecorder := httptest.NewRecorder()
	ClearAuthCookies(clearRecorder)
	for _, c := range clearRecorder.Result().Cookies() {
		if c.MaxAge != 0 {
			t.Fatalf("expected MaxAge=0 to expire cookie %s, got %d", c.Name, c.MaxAge)
		}
	}
}

// TestEnsureDeviceIDCookieIssuesOnceAndKeepsExisting 验证设备 cookie
// 首次下发、已有保留的契约（非 HttpOnly、Lax）。
func TestEnsureDeviceIDCookieIssuesOnceAndKeepsExisting(t *testing.T) {
	recorder := httptest.NewRecorder()
	EnsureDeviceIDCookie(recorder, httptest.NewRequest(http.MethodGet, "/", nil))
	cookies := recorder.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected device cookie, got %d", len(cookies))
	}
	issued := cookies[0]
	if issued.Name != DeviceIDCookieName || issued.Value == "" || issued.HttpOnly {
		t.Fatalf("unexpected device cookie: %#v", issued)
	}

	kept := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.AddCookie(&http.Cookie{Name: DeviceIDCookieName, Value: "existing"})
	EnsureDeviceIDCookie(kept, request)
	if len(kept.Result().Cookies()) != 0 {
		t.Fatal("existing device cookie should be kept as-is")
	}
}
