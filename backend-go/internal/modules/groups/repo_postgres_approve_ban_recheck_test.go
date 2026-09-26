package groups

import (
	"context"
	"errors"
	"testing"
)

// 审计 F-04：审批事务内的封禁复检分支。级联撤销（BanMember 同事务 revoke
// pending 申请）已覆盖主要竞态，此处的复检是「申请仍 pending 而申请人已被
// 封禁」这一异常状态下的最后防线（如未来级联回归）。桩后端使 queryIsGroupBanned
// 命中一行，验证审批整体失败、不触碰后续状态流转与成员插入。
func TestApproveApplicationBanRecheckInTx(t *testing.T) {
	stub := &groupFullStub{
		queryRow: map[string][]any{
			queryGetGroupApplication: {"a1", "group_1", "群名", "", 3, "u2", "申请人", "", "pending", gTestTime, nil, nil},
			queryGetGroup:            groupBaseRow(false),
			// queryIsGroupBanned 命中：申请人已被封禁
			queryIsGroupBanned: {1},
		},
	}
	repo := newGroupsFullRepo(stub)

	err := repo.ApproveApplication(context.Background(), "usr_owner", "a1")
	if !errors.Is(err, ErrBannedInGroup) {
		t.Fatalf("expected ErrBannedInGroup, got %v", err)
	}
}
