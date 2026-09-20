package auth

import (
	"context"
	"strings"
	"sync"
	"time"

	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/observability"
)

// banCacheTTL 封禁结果的进程内缓存窗口。认证中间件每个请求都要过一次
// IsUserBanned（此前无缓存、每请求 +1 次 DB 查询，见 P2-3 注释），缓存把它
// 降为每 TTL 一次。撤销延迟 ≤TTL：封禁/解封动作经 InvalidateBanCache 主动
// 失效（bootstrap 接线到 admin 状态变更），只有绕过 admin 的直接改库才会
// 等满窗口——运维手册须注明。
const banCacheTTL = 30 * time.Second

type banCacheEntry struct {
	banned  bool
	expires time.Time
}

type Service struct {
	tokens     *TokenManager
	accessTTL  time.Duration
	refreshTTL time.Duration
	refreshes  RefreshTokenStore
	sessions   SessionStore
	banChecker BanChecker
	banCacheMu sync.Mutex
	banCache   map[string]banCacheEntry
	// escrow 宽限期托管（AES-GCM，密钥取自 AUTH_TOKEN_SECRET）：轮换时托管
	// 新链 token 明文，并发刷新败者在宽限期内凭此换回。nil 时（理论上不可达，
	// config.Validate 强制非空 secret）宽限期冲突路径按无效处理。
	escrow *TokenEscrow
}

// SetBanChecker injects a ban checker (typically the admin module's
// repository). Once wired, RequiredAuth rejects banned users with httpx.CodeAccountBanned.
// nil disables ban checking (dev/test default).
func (s *Service) SetBanChecker(bc BanChecker) { s.banChecker = bc }

// AccessTTL / RefreshTTL expose the service-configured token lifetimes so
// callers writing auth cookies use the same source of truth as token minting
// (no duplicated literal TTLs).
func (s *Service) AccessTTL() time.Duration  { return s.accessTTL }
func (s *Service) RefreshTTL() time.Duration { return s.refreshTTL }

// IsUserBanned reports whether the user is currently banned. Returns false
// when no ban checker is configured. The checker reads the persisted ban fact
// from Postgres (admin repository) behind a short TTL cache; admin 封禁/解封
// 会主动失效对应条目（InvalidateBanCache）。A checker error is treated as not
// banned (fail-open) so a transient DB issue does not lock out every request,
// and is deliberately NOT cached so recovery applies on the next request; the
// failure is logged so silent security degradations stay observable (P2-3).
func (s *Service) IsUserBanned(ctx context.Context, userID string) bool {
	if s.banChecker == nil {
		return false
	}
	if entry, ok := s.cachedBan(userID); ok {
		return entry
	}
	banned, err := s.banChecker.IsBanned(ctx, userID)
	if err != nil {
		observability.L().Warn("ban check failed, failing open",
			"user_id", userID, "error", err)
		return false
	}
	s.storeBan(userID, banned)
	return banned
}

// InvalidateBanCache drops cached ban facts for the given users (all users if
// none given). Bootstrap wires this to the admin module so 封禁/解封 生效不等待
// 缓存窗口自然过期。
func (s *Service) InvalidateBanCache(userIDs ...string) {
	s.banCacheMu.Lock()
	defer s.banCacheMu.Unlock()
	if s.banCache == nil {
		return
	}
	if len(userIDs) == 0 {
		s.banCache = nil
		return
	}
	for _, id := range userIDs {
		delete(s.banCache, id)
	}
}

func (s *Service) cachedBan(userID string) (bool, bool) {
	s.banCacheMu.Lock()
	defer s.banCacheMu.Unlock()
	entry, ok := s.banCache[userID]
	if !ok || time.Now().After(entry.expires) {
		return false, false
	}
	return entry.banned, true
}

func (s *Service) storeBan(userID string, banned bool) {
	s.banCacheMu.Lock()
	defer s.banCacheMu.Unlock()
	if s.banCache == nil {
		s.banCache = make(map[string]banCacheEntry)
	}
	s.banCache[userID] = banCacheEntry{banned: banned, expires: time.Now().Add(banCacheTTL)}
}

type AuthResult struct {
	Principal      Principal
	AccessToken    string
	RefreshToken   string
	AccessExpires  time.Time
	RefreshExpires time.Time
}

// SessionMeta carries request-derived session metadata for auditing and UX.
type SessionMeta struct {
	DeviceID   string
	DeviceType string
	DeviceName string
	IPAddress  string
	IPLocation string
}

func NewService(cfg config.Config) *Service {
	escrow, _ := NewTokenEscrow(cfg.AuthTokenSecret)
	return &Service{
		tokens:     NewTokenManager(cfg.AuthTokenSecret),
		accessTTL:  15 * time.Minute,
		refreshTTL: 30 * 24 * time.Hour,
		refreshes:  NewInMemoryRefreshStore(),
		sessions:   NewInMemorySessionStore(),
		escrow:     escrow,
	}
}

func NewServiceWithStores(cfg config.Config, sessions SessionStore, refreshes RefreshTokenStore) *Service {
	if sessions == nil {
		sessions = NewInMemorySessionStore()
	}
	if refreshes == nil {
		refreshes = NewInMemoryRefreshStore()
	}
	escrow, _ := NewTokenEscrow(cfg.AuthTokenSecret)
	return &Service{
		tokens:     NewTokenManager(cfg.AuthTokenSecret),
		accessTTL:  15 * time.Minute,
		refreshTTL: 30 * 24 * time.Hour,
		refreshes:  refreshes,
		sessions:   sessions,
		escrow:     escrow,
	}
}

func (s *Service) AuthenticateRequest(rctx context.Context, authHeader string, accessCookie string) (Principal, time.Time, bool) {
	token := bearerToken(authHeader)
	if token == "" {
		token = accessCookie
	}
	if token == "" {
		return Principal{}, time.Time{}, false
	}
	claims, err := s.tokens.ParseAccessToken(token)
	if err != nil {
		return Principal{}, time.Time{}, false
	}
	session, ok, err := s.sessions.Get(rctx, claims.SessionID)
	if err != nil || !ok || session.UserID != claims.UserID || session.RevokedAt != nil {
		return Principal{}, time.Time{}, false
	}
	return newPrincipal(claims.UserID, claims.SessionID, platformFromContext(rctx)), claims.ExpiresAt, true
}

func (s *Service) IssueSession(ctx context.Context, userID string) (AuthResult, error) {
	return s.IssueSessionWithMeta(ctx, userID, SessionMeta{})
}

func (s *Service) IssueSessionWithMeta(ctx context.Context, userID string, meta SessionMeta) (AuthResult, error) {
	sessionID, err := NewSessionID()
	if err != nil {
		return AuthResult{}, err
	}
	createdAt := time.Now().UTC()
	accessToken, accessExp, err := s.tokens.MintAccessToken(userID, sessionID, s.accessTTL)
	if err != nil {
		return AuthResult{}, err
	}
	refreshToken, err := GenerateOpaqueToken()
	if err != nil {
		return AuthResult{}, err
	}
	refreshExpires := time.Now().UTC().Add(s.refreshTTL)
	if err := s.sessions.Save(ctx, SessionRecord{
		SessionID:    sessionID,
		UserID:       userID,
		DeviceID:     meta.DeviceID,
		CreatedAt:    createdAt,
		LastActiveAt: createdAt,
		DeviceType:   meta.DeviceType,
		DeviceName:   meta.DeviceName,
		IPAddress:    meta.IPAddress,
		IPLocation:   meta.IPLocation,
	}); err != nil {
		return AuthResult{}, err
	}
	// 存储只落哈希（migration 053 起全表口径一致），明文仅随 AuthResult 返还客户端。
	if err := s.refreshes.Save(ctx, RefreshSession{UserID: userID, SessionID: sessionID, RefreshToken: HashRefreshToken(refreshToken), ExpiresAt: refreshExpires}); err != nil {
		return AuthResult{}, err
	}
	return AuthResult{Principal: newPrincipal(userID, sessionID, ""), AccessToken: accessToken, RefreshToken: refreshToken, AccessExpires: accessExp, RefreshExpires: refreshExpires}, nil
}

func bearerToken(header string) string {
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(header, prefix))
}
