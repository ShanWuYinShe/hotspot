package auth

import (
	"context"
	"sync"
	"time"
)

// 会话校验缓存（认证路径性能优化）：AuthenticateRequest 每个请求都要用
// session_id 查一次会话（PostgresSessionStore.Get = 一次 DB 往返），是认证端点
// 的主要固定开销。CachedSessionStore 作为 SessionStore 装饰器缓存 Get 结果，
// 写路径（Save/Touch/Revoke/RevokeOthers）同步失效，保证同实例内撤销立即生效。
//
// 一致性边界（写清楚避免误用）：
//   - 同实例撤销（登出 / 踢设备 / 撤销其他会话 / 密码变更吊销）都经过
//     SessionStore.Revoke / RevokeOthers → 本装饰器立即失效对应条目；
//   - 跨实例或绕过服务直接改库的撤销，最多延迟 sessionCacheTTL 生效。当前部署为
//     单实例；若未来多实例，需要把失效广播到 Redis pub/sub 或换共享缓存。
//   - Get 的错误与「不存在」不缓存（避免故障期负缓存放大 + 无界增长）。
const (
	sessionCacheTTL        = 5 * time.Second
	sessionCacheMaxEntries = 4096
)

type sessionCacheEntry struct {
	record    SessionRecord
	expiresAt time.Time
}

// CachedSessionStore 是 SessionStore 的进程内读缓存装饰器；零值不可用，
// 必须经 NewCachedSessionStore 构造。
type CachedSessionStore struct {
	inner      SessionStore
	ttl        time.Duration
	maxEntries int
	now        func() time.Time

	mu      sync.Mutex
	entries map[string]sessionCacheEntry
}

// NewCachedSessionStore 包装 inner（nil 时用内存实现）；ttl/maxEntries 非正时取默认值。
func NewCachedSessionStore(inner SessionStore, ttl time.Duration, maxEntries int) *CachedSessionStore {
	if inner == nil {
		inner = NewInMemorySessionStore()
	}
	if ttl <= 0 {
		ttl = sessionCacheTTL
	}
	if maxEntries <= 0 {
		maxEntries = sessionCacheMaxEntries
	}
	return &CachedSessionStore{
		inner:      inner,
		ttl:        ttl,
		maxEntries: maxEntries,
		now:        time.Now,
		entries:    make(map[string]sessionCacheEntry, maxEntries/8),
	}
}

func (s *CachedSessionStore) Get(ctx context.Context, sessionID string) (SessionRecord, bool, error) {
	now := s.now()
	s.mu.Lock()
	if entry, ok := s.entries[sessionID]; ok {
		if now.Before(entry.expiresAt) {
			s.mu.Unlock()
			return entry.record, true, nil
		}
		delete(s.entries, sessionID)
	}
	s.mu.Unlock()

	record, ok, err := s.inner.Get(ctx, sessionID)
	if err != nil || !ok {
		return record, ok, err
	}
	s.mu.Lock()
	if len(s.entries) >= s.maxEntries {
		// 简单整体清空而不是精细淘汰：撤销/新会话是低频事件，稳态缓存命中后
		// 不常触达上限；清空保证内存有界且实现简单（无额外 LRU 维护成本）。
		s.entries = make(map[string]sessionCacheEntry, s.maxEntries/8)
	}
	s.entries[sessionID] = sessionCacheEntry{record: record, expiresAt: now.Add(s.ttl)}
	s.mu.Unlock()
	return record, true, nil
}

// InvalidateSessions 主动失效若干会话缓存。供绕过 SessionStore 直接改库的调用方
// （如 admin 封禁流程的逐条撤销会话）保持「撤销即时生效」，不依赖 TTL 兜底。
func (s *CachedSessionStore) InvalidateSessions(sessionIDs ...string) {
	for _, id := range sessionIDs {
		s.invalidate(id)
	}
}

func (s *CachedSessionStore) invalidate(sessionID string) {
	s.mu.Lock()
	delete(s.entries, sessionID)
	s.mu.Unlock()
}

func (s *CachedSessionStore) clear() {
	s.mu.Lock()
	s.entries = make(map[string]sessionCacheEntry, s.maxEntries/8)
	s.mu.Unlock()
}

func (s *CachedSessionStore) Save(ctx context.Context, session SessionRecord) error {
	if err := s.inner.Save(ctx, session); err != nil {
		return err
	}
	s.invalidate(session.SessionID)
	return nil
}

func (s *CachedSessionStore) Touch(ctx context.Context, sessionID string) error {
	if err := s.inner.Touch(ctx, sessionID); err != nil {
		return err
	}
	s.invalidate(sessionID)
	return nil
}

func (s *CachedSessionStore) Revoke(ctx context.Context, sessionID string) (bool, error) {
	revoked, err := s.inner.Revoke(ctx, sessionID)
	if revoked || err == nil {
		s.invalidate(sessionID)
	}
	return revoked, err
}

func (s *CachedSessionStore) RevokeOthers(ctx context.Context, userID, currentSessionID string) (int, error) {
	count, err := s.inner.RevokeOthers(ctx, userID, currentSessionID)
	// 无法从接口得知被撤销的 session_id 集合，直接清空（撤销他人会话是低频操作）。
	s.clear()
	return count, err
}

func (s *CachedSessionStore) ListActiveByUser(ctx context.Context, userID string) ([]SessionRecord, error) {
	return s.inner.ListActiveByUser(ctx, userID)
}
