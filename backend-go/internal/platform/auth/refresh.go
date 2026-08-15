package auth

import (
	"context"
	"errors"
	"time"
)

// RefreshRotationGrace 是旧 refresh token 轮换后的宽限期（spec
// api_spec_auth.md「并发刷新策略」）：期内旧 token 复用返回新链 token，
// 期后拒绝。此前 30s 在内存/PG 两个 store 各硬编码一次，统一到此处。
const RefreshRotationGrace = 30 * time.Second

// ErrRefreshRotationConflict 表示旧 token 已被并发请求轮换（UPDATE 的
// rotated_to IS NULL 守卫 0 行命中）。调用方应重读 store 走宽限期分支。
var ErrRefreshRotationConflict = errors.New("refresh token already rotated concurrently")

func (s *Service) RefreshSession(ctx context.Context, refreshToken string) (AuthResult, error) {
	session, ok, err := s.refreshes.Get(ctx, refreshToken)
	if err != nil {
		return AuthResult{}, err
	}
	if !ok {
		return AuthResult{}, errors.New("invalid refresh token")
	}
	now := time.Now().UTC()
	if now.After(session.ExpiresAt) {
		return AuthResult{}, errors.New("expired refresh token")
	}
	if storedSession, ok, err := s.sessions.Get(ctx, session.SessionID); err != nil {
		return AuthResult{}, err
	} else if !ok || storedSession.RevokedAt != nil {
		return AuthResult{}, errors.New("revoked session")
	}
	if session.RotatedTo != "" {
		return s.resultFromRotated(ctx, session, now)
	}
	newRefreshToken, err := GenerateOpaqueToken()
	if err != nil {
		return AuthResult{}, err
	}
	newAccessToken, accessExp, err := s.tokens.MintAccessToken(session.UserID, session.SessionID, s.accessTTL)
	if err != nil {
		return AuthResult{}, err
	}
	next := RefreshSession{
		UserID:       session.UserID,
		SessionID:    session.SessionID,
		RefreshToken: newRefreshToken,
		ExpiresAt:    now.Add(s.refreshTTL),
	}
	if err := s.refreshes.Rotate(ctx, refreshToken, session, next); err != nil {
		if errors.Is(err, ErrRefreshRotationConflict) {
			// 并发轮换败者：旧 token 刚被另一请求轮换（仍在宽限期内），
			// 重读 store 后走宽限期分支返回获胜链的 token——这正是轮换
			// 机制要兜住的「攻击者与真实客户端并发刷新」场景。
			rotated, ok, err := s.refreshes.Get(ctx, refreshToken)
			if err != nil {
				return AuthResult{}, err
			}
			if !ok || rotated.RotatedTo == "" {
				return AuthResult{}, errors.New("invalid refresh token")
			}
			return s.resultFromRotated(ctx, rotated, now)
		}
		return AuthResult{}, err
	}
	principal := newPrincipal(session.UserID, session.SessionID, "")
	return AuthResult{
		Principal:      principal,
		AccessToken:    newAccessToken,
		RefreshToken:   newRefreshToken,
		AccessExpires:  accessExp,
		RefreshExpires: next.ExpiresAt,
	}, nil
}

// resultFromRotated 处理已轮换 token 的宽限期复用：期内返回新链 token，
// 期后拒绝。
func (s *Service) resultFromRotated(ctx context.Context, session RefreshSession, now time.Time) (AuthResult, error) {
	if now.After(session.GraceUntil) {
		return AuthResult{}, errors.New("expired refresh rotation window")
	}
	rotated, ok, err := s.refreshes.Get(ctx, session.RotatedTo)
	if err != nil {
		return AuthResult{}, err
	}
	if !ok {
		return AuthResult{}, errors.New("invalid refresh token")
	}
	accessToken, accessExp, err := s.tokens.MintAccessToken(rotated.UserID, rotated.SessionID, s.accessTTL)
	if err != nil {
		return AuthResult{}, err
	}
	principal := newPrincipal(rotated.UserID, rotated.SessionID, "")
	return AuthResult{
		Principal:      principal,
		AccessToken:    accessToken,
		RefreshToken:   rotated.RefreshToken,
		AccessExpires:  accessExp,
		RefreshExpires: rotated.ExpiresAt,
	}, nil
}
