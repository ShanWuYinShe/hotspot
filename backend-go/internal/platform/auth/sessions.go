package auth

import (
	"context"

	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/observability"
)

func (s *Service) GetSession(ctx context.Context, sessionID string) (SessionRecord, bool, error) {
	return s.sessions.Get(ctx, sessionID)
}

func (s *Service) RevokeByUserDevice(ctx context.Context, userID, deviceID string) (int, error) {
	sessions, err := s.sessions.ListActiveByUser(ctx, userID)
	if err != nil {
		return 0, err
	}
	count := 0
	for _, session := range sessions {
		if session.DeviceID == deviceID && deviceID != "" && session.RevokedAt == nil {
			if revoked, err := s.sessions.Revoke(ctx, session.SessionID); err != nil {
				return count, err
			} else if revoked {
				count++
			}
		}
	}
	return count, nil
}

// ListSessions 会话列表查询失败时降级为空列表返回（接口契约不定义内部错误），
// 但必须告警：静默吞掉会让用户在 store 故障时误判「无其他登录会话」，
// 登出全设备的撤销计数也会失真且不可观测。
func (s *Service) ListSessions(ctx context.Context, userID, currentSessionID string) []SessionRecord {
	sessions, err := s.sessions.ListActiveByUser(ctx, userID)
	if err != nil {
		observability.L().Warn("auth.ListSessions: list active sessions failed, degrading to empty", "error", err, "user_id", userID)
		return []SessionRecord{}
	}
	if currentSessionID == "" {
		return sessions
	}
	for i := range sessions {
		if sessions[i].SessionID == currentSessionID {
			if err := s.sessions.Touch(ctx, currentSessionID); err != nil {
				observability.L().Warn("auth.ListSessions: touch session failed", "error", err, "session_id", currentSessionID)
			}
			break
		}
	}
	updated, err := s.sessions.ListActiveByUser(ctx, userID)
	if err != nil {
		observability.L().Warn("auth.ListSessions: re-list after touch failed, returning pre-touch snapshot", "error", err, "user_id", userID)
		return sessions
	}
	return updated
}

// RevokeOtherSessions 撤销失败时返回 0（调用方按撤销数下发响应，契约不定义
// 内部错误），但必须告警：静默返回会让「登出其他设备」故障被当成功。
func (s *Service) RevokeOtherSessions(ctx context.Context, userID, currentSessionID string) int {
	count, err := s.sessions.RevokeOthers(ctx, userID, currentSessionID)
	if err != nil {
		observability.L().Warn("auth.RevokeOtherSessions: revoke others failed", "error", err, "user_id", userID)
		return 0
	}
	return count
}

func (s *Service) RevokeSession(ctx context.Context, userID, _ string, targetSessionID string) error {
	target, ok, err := s.sessions.Get(ctx, targetSessionID)
	if err != nil {
		return err
	}
	if !ok || target.RevokedAt != nil {
		return &httpx.APIError{Code: httpx.CodeNotFound, Message: "会话不存在"}
	}
	if target.UserID != userID {
		return httpx.Forbidden(httpx.ReasonNotResourceOwner, "无权撤销该会话")
	}
	if _, err := s.sessions.Revoke(ctx, targetSessionID); err != nil {
		return err
	}
	return nil
}
