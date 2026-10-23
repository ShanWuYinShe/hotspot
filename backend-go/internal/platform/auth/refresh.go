package auth

import (
	"context"
	"errors"
	"time"

	"hotspot/backend-go/internal/platform/observability"
)

// RefreshRotationGrace 是旧 refresh token 轮换后的宽限期（spec
// api_spec_auth.md「并发刷新策略」）：期内旧 token 复用返回新链 token，
// 期后拒绝。此前 30s 在内存/PG 两个 store 各硬编码一次，统一到此处。
const RefreshRotationGrace = 30 * time.Second

// ErrRefreshRotationConflict 表示旧 token 已被并发请求轮换（UPDATE 的
// rotated_to IS NULL 守卫 0 行命中）。调用方应重读 store 走宽限期分支。
var ErrRefreshRotationConflict = errors.New("refresh token already rotated concurrently")

// token 生命周期内的预期业务拒绝 sentinel（消息与原 errors.New 字面量一致，
// 仅收敛为可 errors.Is 识别的导出值）：调用方据此区分业务失败（Warn 级
// 日志）与基础设施故障（Error 级），避免自然到期的刷新刷屏 Error 日志。
var (
	ErrInvalidRefreshToken   = errors.New("invalid refresh token")
	ErrExpiredRefreshToken   = errors.New("expired refresh token")
	ErrRevokedSession        = errors.New("revoked session")
	ErrRotationWindowExpired = errors.New("expired refresh rotation window")
)

// SessionIDForRefreshToken 只读解析 refresh token 对应的会话（不轮换、不产生
// 副作用），供设备绑定等预校验前置于 RefreshSession：不匹配的请求应在轮换前
// 被拒绝，否则伪造参数也能触发一次 token 轮换并把旧 token 推入宽限期。
// token 无效/过期或会话已吊销时返回 ok=false（err==nil）；查询故障返回 err。
func (s *Service) SessionIDForRefreshToken(ctx context.Context, refreshToken string) (sessionID string, ok bool, err error) {
	refreshKey := HashRefreshToken(refreshToken)
	session, found, err := s.refreshes.Get(ctx, refreshKey)
	if err != nil || !found {
		return "", false, err
	}
	if time.Now().UTC().After(session.ExpiresAt) {
		return "", false, nil
	}
	stored, found, err := s.sessions.Get(ctx, session.SessionID)
	if err != nil {
		return "", false, err
	}
	if !found || stored.RevokedAt != nil {
		return "", false, nil
	}
	return session.SessionID, true, nil
}

func (s *Service) RefreshSession(ctx context.Context, refreshToken string) (AuthResult, error) {
	// 存储键为 token 的 SHA-256（migration 053 起全表口径一致）；
	// 明文仅存在于客户端持有与宽限期托管密文中。
	refreshKey := HashRefreshToken(refreshToken)
	session, ok, err := s.refreshes.Get(ctx, refreshKey)
	if err != nil {
		return AuthResult{}, err
	}
	if !ok {
		return AuthResult{}, ErrInvalidRefreshToken
	}
	now := time.Now().UTC()
	if now.After(session.ExpiresAt) {
		return AuthResult{}, ErrExpiredRefreshToken
	}
	if storedSession, ok, err := s.sessions.Get(ctx, session.SessionID); err != nil {
		return AuthResult{}, err
	} else if !ok || storedSession.RevokedAt != nil {
		return AuthResult{}, ErrRevokedSession
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
		RefreshToken: HashRefreshToken(newRefreshToken),
		ExpiresAt:    now.Add(s.refreshTTL),
	}
	// 托管新链 token 明文：并发刷新败者在宽限期内凭密文换回明文
	//（resultFromRotated）。托管失败则本轮不产生托管——败者将走
	// 「宽限期内但无托管」兜底，安全侧失败。
	if s.escrow != nil {
		sealed, sealErr := s.escrow.Seal(newRefreshToken)
		if sealErr != nil {
			return AuthResult{}, sealErr
		}
		next.GraceTokenEnc = sealed
	}
	if err := s.refreshes.Rotate(ctx, refreshKey, session, next); err != nil {
		if errors.Is(err, ErrRefreshRotationConflict) {
			// 并发轮换败者：旧 token 刚被另一请求轮换（仍在宽限期内），
			// 重读 store 后走宽限期分支返回获胜链的 token——这正是轮换
			// 机制要兜住的「攻击者与真实客户端并发刷新」场景。
			rotated, ok, err := s.refreshes.Get(ctx, refreshKey)
			if err != nil {
				return AuthResult{}, err
			}
			if !ok || rotated.RotatedTo == "" {
				return AuthResult{}, ErrInvalidRefreshToken
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
// 期后视为 token 盗用信号——吊销整个会话并拒绝（P2-4）。同一会话的全部
// refresh 链共享 SessionID，会话吊销后旧链所有 token 在 RefreshSession
// 入口（revoked session 检查）与 access token 校验（AuthenticateRequest）
// 一并失效，即「检测到重用 → 吊销该会话链」。
func (s *Service) resultFromRotated(ctx context.Context, session RefreshSession, now time.Time) (AuthResult, error) {
	if now.After(session.GraceUntil) {
		if _, revokeErr := s.sessions.Revoke(ctx, session.SessionID); revokeErr != nil {
			observability.L().Error("refresh reuse: session revoke failed",
				"session_id", session.SessionID, "user_id", session.UserID, "error", revokeErr)
		}
		observability.L().Warn("refresh token reuse detected, session chain revoked",
			"user_id", session.UserID, "session_id", session.SessionID)
		return AuthResult{}, ErrRotationWindowExpired
	}
	rotated, ok, err := s.refreshes.Get(ctx, session.RotatedTo)
	if err != nil {
		return AuthResult{}, err
	}
	if s.escrow == nil || len(session.GraceTokenEnc) == 0 {
		// 无托管可解（理论不可达：轮换必写托管；迁移过渡期的存量宽限行
		// 除外）。安全侧失败：拒绝刷新，真实客户端重新登录。
		observability.L().Warn("refresh grace escrow missing",
			"user_id", session.UserID, "session_id", session.SessionID)
		return AuthResult{}, errors.New("invalid refresh token")
	}
	gracePlaintext, decErr := s.escrow.Open(session.GraceTokenEnc)
	if decErr != nil || gracePlaintext == "" {
		observability.L().Warn("refresh grace escrow decrypt failed",
			"user_id", session.UserID, "session_id", session.SessionID, "error", decErr)
		return AuthResult{}, ErrInvalidRefreshToken
	}
	if !ok {
		return AuthResult{}, ErrInvalidRefreshToken
	}
	accessToken, accessExp, err := s.tokens.MintAccessToken(rotated.UserID, rotated.SessionID, s.accessTTL)
	if err != nil {
		return AuthResult{}, err
	}
	principal := newPrincipal(rotated.UserID, rotated.SessionID, "")
	return AuthResult{
		Principal:      principal,
		AccessToken:    accessToken,
		RefreshToken:   gracePlaintext,
		AccessExpires:  accessExp,
		RefreshExpires: rotated.ExpiresAt,
	}, nil
}
