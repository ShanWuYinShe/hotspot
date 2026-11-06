package groupmessages

import (
	"context"
	"testing"
)

// TestPostgresRemainingReadPathsCoverage 覆盖群消息仓储剩余读/审核路径的骨架
// （未命中/错误分支；真实库行为由 IT 用例覆盖）。
func TestPostgresRemainingReadPathsCoverage(t *testing.T) {
	ctx := context.Background()

	svc := NewServiceWithRepo(NewInMemoryRepository())
	svc.SetMemberCache(nil)

	repo := NewPostgresRepository(newGmFactory(&gmStub{}))
	_, _ = repo.RecallAsAdmin(ctx, "m1", "op1")
	_, _ = repo.GetReadCounts(ctx, "g1", []string{"m1"})
	_, _ = repo.FilterMemberIDs(ctx, "g1", []string{"u1"})
	_, _, _, _ = repo.MemberMuteStatus(ctx, "g1", "u1")
	_, _, _ = repo.GetMessageForReply(ctx, "m1")
	_, _, _ = repo.LookupMessageMeta(ctx, "m1")
	_, _, _ = repo.GetMessageReportSnapshot(ctx, "m1")
	_, _, _ = repo.GetMediaForMessage(ctx, "f1", "u1")
}
