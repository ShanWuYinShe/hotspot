package share

import (
	"context"
	"testing"
)

// TestSharePostgresGetPreviewUnavailableForDeletedTargets 覆盖 group/moment 目标
// 不可用时的 preview 降级：preview 置空但分享本身仍可读（Get 不因 preview 缺失失败）。
func TestSharePostgresGetPreviewUnavailableForDeletedTargets(t *testing.T) {
	ctx := context.Background()

	backend := newShareBackendStub()
	backend.byCode["ABCDEFGHJKLM"] = "share_group_gone"
	backend.shares["share_group_gone"] = shareStubRecord{
		ShareID: "share_group_gone", ShareCode: "ABCDEFGHJKLM", ShareType: ShareTypeGroup, CreatorUserID: "user_alice",
		PayloadJSON: sharePayloadJSONOf(t, map[string]any{"shared_group_id": "group_missing"}),
	}
	detail, err := newShareTestRepo(backend).Get(ctx, getShareQuery{ShareCode: "ABCDEFGHJKLM"})
	if err != nil {
		t.Fatalf("get share with missing group: %v", err)
	}
	if detail.Preview != nil {
		t.Fatalf("missing group preview must be nil, got %#v", detail.Preview)
	}

	backend.byCode["ABCDEFGHJKMN"] = "share_moment_gone"
	backend.shares["share_moment_gone"] = shareStubRecord{
		ShareID: "share_moment_gone", ShareCode: "ABCDEFGHJKMN", ShareType: ShareTypeMoment, CreatorUserID: "user_alice",
		PayloadJSON: sharePayloadJSONOf(t, map[string]any{"shared_moment_id": "moment_missing"}),
	}
	detail, err = newShareTestRepo(backend).Get(ctx, getShareQuery{ShareCode: "ABCDEFGHJKMN"})
	if err != nil {
		t.Fatalf("get share with missing moment: %v", err)
	}
	if detail.Preview != nil {
		t.Fatalf("missing moment preview must be nil, got %#v", detail.Preview)
	}
}
