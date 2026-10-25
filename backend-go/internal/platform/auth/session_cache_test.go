package auth

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// countingSessionStore 是 SessionStore 的内存实现 + 调用计数，用于验证缓存是否
// 真的减少了底层 Get（以及写路径是否触发失效）。
type countingSessionStore struct {
	mu        sync.Mutex
	getCalls  int
	records   map[string]SessionRecord
	revokeAll int
}

func newCountingSessionStore() *countingSessionStore {
	return &countingSessionStore{records: map[string]SessionRecord{}}
}

func (s *countingSessionStore) Save(_ context.Context, session SessionRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.records[session.SessionID] = session
	return nil
}

func (s *countingSessionStore) Touch(_ context.Context, sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if rec, ok := s.records[sessionID]; ok {
		rec.LastActiveAt = time.Now().UTC()
		s.records[sessionID] = rec
	}
	return nil
}

func (s *countingSessionStore) ListActiveByUser(_ context.Context, userID string) ([]SessionRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []SessionRecord{}
	for _, rec := range s.records {
		if rec.UserID == userID && rec.RevokedAt == nil {
			out = append(out, rec)
		}
	}
	return out, nil
}

func (s *countingSessionStore) Revoke(_ context.Context, sessionID string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, ok := s.records[sessionID]
	if !ok || rec.RevokedAt != nil {
		return false, nil
	}
	now := time.Now().UTC()
	rec.RevokedAt = &now
	s.records[sessionID] = rec
	return true, nil
}

func (s *countingSessionStore) RevokeOthers(_ context.Context, userID, currentSessionID string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	count := 0
	now := time.Now().UTC()
	for id, rec := range s.records {
		if rec.UserID == userID && id != currentSessionID && rec.RevokedAt == nil {
			rec.RevokedAt = &now
			s.records[id] = rec
			count++
		}
	}
	s.revokeAll++
	return count, nil
}

func (s *countingSessionStore) Get(_ context.Context, sessionID string) (SessionRecord, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.getCalls++
	rec, ok := s.records[sessionID]
	return rec, ok, nil
}

func (s *countingSessionStore) getCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.getCalls
}

func TestCachedSessionStoreHitSkipsInnerGet(t *testing.T) {
	inner := newCountingSessionStore()
	if err := inner.Save(context.Background(), SessionRecord{SessionID: "s1", UserID: "u1"}); err != nil {
		t.Fatal(err)
	}
	cache := NewCachedSessionStore(inner, time.Minute, 16)
	for i := 0; i < 3; i++ {
		rec, ok, err := cache.Get(context.Background(), "s1")
		if err != nil || !ok || rec.UserID != "u1" {
			t.Fatalf("Get #%d = (%+v, %v, %v)", i, rec, ok, err)
		}
	}
	if got := inner.getCount(); got != 1 {
		t.Fatalf("inner Get calls = %d, want 1", got)
	}
}

func TestCachedSessionStoreRevokeInvalidatesImmediately(t *testing.T) {
	inner := newCountingSessionStore()
	if err := inner.Save(context.Background(), SessionRecord{SessionID: "s1", UserID: "u1"}); err != nil {
		t.Fatal(err)
	}
	cache := NewCachedSessionStore(inner, time.Minute, 16)
	if _, _, err := cache.Get(context.Background(), "s1"); err != nil {
		t.Fatal(err)
	}
	if _, err := cache.Revoke(context.Background(), "s1"); err != nil {
		t.Fatal(err)
	}
	rec, ok, err := cache.Get(context.Background(), "s1")
	if err != nil || !ok {
		t.Fatalf("Get after revoke = (%+v, %v, %v)", rec, ok, err)
	}
	if rec.RevokedAt == nil {
		t.Fatal("revoked session must not be served from stale cache")
	}
	if got := inner.getCount(); got != 2 {
		t.Fatalf("inner Get calls = %d, want 2 (cache invalidated by Revoke)", got)
	}
}

func TestCachedSessionStoreRevokeOthersClearsCache(t *testing.T) {
	inner := newCountingSessionStore()
	ctx := context.Background()
	for _, id := range []string{"s1", "s2"} {
		if err := inner.Save(ctx, SessionRecord{SessionID: id, UserID: "u1"}); err != nil {
			t.Fatal(err)
		}
	}
	cache := NewCachedSessionStore(inner, time.Minute, 16)
	for _, id := range []string{"s1", "s2"} {
		if _, _, err := cache.Get(ctx, id); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := cache.RevokeOthers(ctx, "u1", "s1"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := cache.Get(ctx, "s2"); err != nil {
		t.Fatal(err)
	}
	if got := inner.getCount(); got != 3 {
		t.Fatalf("inner Get calls = %d, want 3 (RevokeOthers must clear cache)", got)
	}
}

func TestCachedSessionStoreTTLExpires(t *testing.T) {
	inner := newCountingSessionStore()
	ctx := context.Background()
	if err := inner.Save(ctx, SessionRecord{SessionID: "s1", UserID: "u1"}); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	cache := NewCachedSessionStore(inner, 5*time.Second, 16)
	cache.now = func() time.Time { return now }
	if _, _, err := cache.Get(ctx, "s1"); err != nil {
		t.Fatal(err)
	}
	now = now.Add(6 * time.Second)
	if _, _, err := cache.Get(ctx, "s1"); err != nil {
		t.Fatal(err)
	}
	if got := inner.getCount(); got != 2 {
		t.Fatalf("inner Get calls = %d, want 2 after TTL expiry", got)
	}
}

func TestCachedSessionStoreSaveAndTouchInvalidate(t *testing.T) {
	inner := newCountingSessionStore()
	ctx := context.Background()
	cache := NewCachedSessionStore(inner, time.Minute, 16)
	if err := cache.Save(ctx, SessionRecord{SessionID: "s1", UserID: "u1"}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := cache.Get(ctx, "s1"); err != nil {
		t.Fatal(err)
	}
	if err := cache.Touch(ctx, "s1"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := cache.Get(ctx, "s1"); err != nil {
		t.Fatal(err)
	}
	if got := inner.getCount(); got != 2 {
		t.Fatalf("inner Get calls = %d, want 2 (Touch invalidates)", got)
	}
}

func TestCachedSessionStoreBounded(t *testing.T) {
	inner := newCountingSessionStore()
	ctx := context.Background()
	cache := NewCachedSessionStore(inner, time.Minute, 4)
	for i := 0; i < 32; i++ {
		id := "s" + string(rune('a'+i%26)) + string(rune('0'+i/26))
		if err := inner.Save(ctx, SessionRecord{SessionID: id, UserID: "u1"}); err != nil {
			t.Fatal(err)
		}
		if _, _, err := cache.Get(ctx, id); err != nil {
			t.Fatal(err)
		}
	}
	cache.mu.Lock()
	size := len(cache.entries)
	cache.mu.Unlock()
	if size > 4 {
		t.Fatalf("cache entries = %d, want <= maxEntries 4", size)
	}
}

func TestCachedSessionStoreMissingNotCached(t *testing.T) {
	inner := newCountingSessionStore()
	cache := NewCachedSessionStore(inner, time.Minute, 16)
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		if _, ok, err := cache.Get(ctx, "missing"); ok || err != nil {
			t.Fatalf("missing session Get = (ok=%v, err=%v)", ok, err)
		}
	}
	if got := inner.getCount(); got != 3 {
		t.Fatalf("inner Get calls = %d, want 3 (misses not cached)", got)
	}
}

func TestCachedSessionStoreGetErrorPassthrough(t *testing.T) {
	inner := &errorSessionStore{err: errors.New("db down")}
	cache := NewCachedSessionStore(inner, time.Minute, 16)
	if _, _, err := cache.Get(context.Background(), "s1"); err == nil {
		t.Fatal("expected error passthrough")
	}
}

// errorSessionStore 只实现 Get 的失败注入，其余方法 panic（本测试不会触达）。
type errorSessionStore struct{ err error }

func (s *errorSessionStore) Save(context.Context, SessionRecord) error { return nil }
func (s *errorSessionStore) Touch(context.Context, string) error       { return nil }
func (s *errorSessionStore) ListActiveByUser(context.Context, string) ([]SessionRecord, error) {
	return nil, s.err
}
func (s *errorSessionStore) Revoke(context.Context, string) (bool, error) { return false, s.err }
func (s *errorSessionStore) RevokeOthers(context.Context, string, string) (int, error) {
	return 0, s.err
}
func (s *errorSessionStore) Get(context.Context, string) (SessionRecord, bool, error) {
	return SessionRecord{}, false, s.err
}

func TestCachedSessionStoreConcurrentAccess(t *testing.T) {
	inner := newCountingSessionStore()
	ctx := context.Background()
	if err := inner.Save(ctx, SessionRecord{SessionID: "s1", UserID: "u1"}); err != nil {
		t.Fatal(err)
	}
	cache := NewCachedSessionStore(inner, time.Minute, 16)
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			switch i % 5 {
			case 0:
				_, _ = cache.Revoke(ctx, "s1")
			case 1:
				_ = cache.Save(ctx, SessionRecord{SessionID: "s1", UserID: "u1"})
			default:
				if _, ok, err := cache.Get(ctx, "s1"); err != nil || !ok {
					t.Errorf("Get = (ok=%v, err=%v)", ok, err)
				}
			}
		}(i)
	}
	wg.Wait()
}
