package auth

import (
	"context"
	"sync"
	"time"
)

type RefreshSession struct {
	UserID       string
	SessionID    string
	RefreshToken string
	RotatedTo    string
	GraceUntil   time.Time
	ExpiresAt    time.Time
	// GraceTokenEnc 宽限期托管：轮换后新链 token 的 AES-GCM 密文（密钥
	// 不在 DB），供并发刷新败者在宽限期内换回明文。仅被轮换的旧行携带。
	GraceTokenEnc []byte
}

type RefreshStore struct {
	mu    sync.RWMutex
	items map[string]RefreshSession
}

func NewRefreshStore() *RefreshStore {
	return &RefreshStore{items: make(map[string]RefreshSession)}
}

func (s *RefreshStore) Save(_ context.Context, session RefreshSession) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[session.RefreshToken] = session
	return nil
}

func (s *RefreshStore) Get(_ context.Context, token string) (RefreshSession, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	session, ok := s.items[token]
	return session, ok, nil
}

func (s *RefreshStore) Rotate(_ context.Context, oldToken string, current RefreshSession, next RefreshSession) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	current.RotatedTo = next.RefreshToken
	current.GraceUntil = time.Now().UTC().Add(RefreshRotationGrace)
	current.GraceTokenEnc = next.GraceTokenEnc
	s.items[oldToken] = current
	s.items[next.RefreshToken] = next
	return nil
}
