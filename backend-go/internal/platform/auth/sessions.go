package auth

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"

	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/observability"
)

// deviceIssueLockIndex 把 (userID, deviceID) 映射到分片锁（FNV-1a，无需
// 全局 map 与额外锁；同键必然同分片）。
func deviceIssueLockIndex(userID, deviceID string) uint32 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(userID))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(deviceID))
	return h.Sum32() % deviceIssueLockStripes
}

// IssueSessionEvictingDevice 为新登录签发会话，并保证「同一设备仅一个活跃
// 会话」（审计未修限制 1 的 TOCTOU 修复）。
//
// 旧实现是「先 RevokeByUserDevice 再 IssueSession」两步：两个并发登录请求可
// 双双通过撤销检查后各自插入会话，同一设备留下两个活跃会话，旧会话的 refresh
// token 仍可轮转。现在改为「先插入（由迁移 057 的部分唯一索引兜底），冲突则
// 撤销该设备旧会话后重试」，数据库约束保证最终至多一个活跃会话。
func (s *Service) IssueSessionEvictingDevice(ctx context.Context, userID string, meta SessionMeta) (AuthResult, error) {
	// 同进程内按 (user_id, device_id) 串行化：唯一索引负责跨实例正确性，这把
	// 分片锁消除同实例内多个登录请求「撤销→重试」互相踩踏导致的空闲窗口
	// （极端情况下重试耗尽会留下 0 个活跃会话）。仅逐出路径加锁，普通并发
	// 登录（不同设备）不受影响。
	lock := &s.deviceIssueLocks[deviceIssueLockIndex(userID, meta.DeviceID)]
	lock.Lock()
	defer lock.Unlock()

	const maxAttempts = 2
	for attempt := 0; attempt < maxAttempts; attempt++ {
		result, err := s.IssueSessionWithMeta(ctx, userID, meta)
		if err == nil {
			return result, nil
		}
		if !errors.Is(err, ErrDeviceSessionConflict) {
			return AuthResult{}, err
		}
		// 本设备已有活跃会话：撤销后重试。并发对手可能刚插入了新会话，
		// 第二轮同样会被索引拦截并再次撤销，最坏情况下返回冲突由调用方报错。
		if _, revokeErr := s.RevokeByUserDevice(ctx, userID, meta.DeviceID); revokeErr != nil {
			return AuthResult{}, fmt.Errorf("auth: evict device sessions: %w", revokeErr)
		}
	}
	return AuthResult{}, ErrDeviceSessionConflict
}

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
