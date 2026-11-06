package privatemessages

import (
	"context"

	"hotspot/backend-go/internal/platform/persistence"
)

// queryPurgeUserChatReadState 账户注销级联：删除本模块归属的私聊已读位点（private_chat_read_state；对方状态保留）。
const queryPurgeUserChatReadState = `DELETE FROM private_chat_read_state WHERE user_id = $1`

// PurgeUserData 账户注销级联（users 模块经 bootstrap 编排调用）：删除本模块
// 归属的该用户数据；幂等，失败由编排层中止并允许整体重试。
func (r *PostgresRepository) PurgeUserData(ctx context.Context, userID string) error {
	q, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return persistence.ErrNotImplemented
	}
	return q.Exec(ctx, queryPurgeUserChatReadState, userID)
}

// PurgeUserData 账户注销级联入口：实现由 Postgres 仓储提供；内存/测试仓储
// 无跨模块行可清时返回 nil。
func (s *Service) PurgeUserData(ctx context.Context, userID string) error {
	if p, ok := s.repo.(userDataPurger); ok {
		return p.PurgeUserData(ctx, userID)
	}
	return nil
}

type userDataPurger interface {
	PurgeUserData(ctx context.Context, userID string) error
}
