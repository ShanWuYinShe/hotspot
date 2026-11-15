package share

import (
	"context"
	"testing"
)

// 本文件锁定仓储层的 moment 封面来源：preview.coverStorageKey 必须是该瞬间
// 第一张「已完成」媒体（按 position），CoverURL 仍先填作者头像兜底；单条
// resolveSharePreview 与列表批量 resolveSharePreviews 两条路径同口径。

func TestSharePostgresPreviewCarriesMomentCoverStorageKey(t *testing.T) {
	const (
		coverKey    = "uploads/user_bob/first.jpg"
		avatarURL   = "https://example.com/bob.png"
		textMoment  = "moment_text_only"
		coverMoment = "moment_with_cover"
	)
	backend := newShareBackendStub()
	backend.moments[coverMoment] = shareStubMoment{
		Description:     "带图瞬间",
		LocationName:    "Shanghai",
		AuthorNickname:  "Bob",
		AuthorAvatarURL: avatarURL,
		Visibility:      "public",
		CoverStorageKey: coverKey,
	}
	backend.moments[textMoment] = shareStubMoment{
		Description:     "纯文字瞬间",
		AuthorNickname:  "Bob",
		AuthorAvatarURL: avatarURL,
		Visibility:      "public",
	}
	repo := newShareTestRepo(backend)
	ctx := context.Background()

	coverShare, err := repo.Create(ctx, createShareCommand{CreatorUserID: "user_alice", ShareType: ShareTypeMoment, Payload: map[string]any{"shared_moment_id": coverMoment}})
	if err != nil {
		t.Fatalf("create cover share: %v", err)
	}
	textShare, err := repo.Create(ctx, createShareCommand{CreatorUserID: "user_alice", ShareType: ShareTypeMoment, Payload: map[string]any{"shared_moment_id": textMoment}})
	if err != nil {
		t.Fatalf("create text share: %v", err)
	}

	// 单条路径
	detail, err := repo.Get(ctx, getShareQuery{ShareCode: coverShare.ShareCode})
	if err != nil {
		t.Fatalf("get cover share: %v", err)
	}
	if detail.Preview == nil || detail.Preview.coverStorageKey != coverKey {
		t.Fatalf("单条预览封面 key = %#v，期望 %q", detail.Preview, coverKey)
	}
	if detail.Preview.CoverURL == nil || *detail.Preview.CoverURL != avatarURL {
		t.Fatalf("仓储层封面仍应是作者头像兜底: %#v", detail.Preview.CoverURL)
	}
	textDetail, err := repo.Get(ctx, getShareQuery{ShareCode: textShare.ShareCode})
	if err != nil {
		t.Fatalf("get text share: %v", err)
	}
	if textDetail.Preview == nil || textDetail.Preview.coverStorageKey != "" {
		t.Fatalf("纯文字瞬间不得带封面 key: %#v", textDetail.Preview)
	}

	// 列表批量路径（与单条同口径）
	page, err := repo.List(ctx, listSharesQuery{CreatorUserID: "user_alice", Limit: 20})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	byID := make(map[string]ShareListItem, len(page.Items))
	for _, item := range page.Items {
		byID[item.ShareID] = item
	}
	if item := byID[coverShare.ShareID]; item.Preview == nil || item.Preview.coverStorageKey != coverKey {
		t.Fatalf("列表带图瞬间封面 key = %#v，期望 %q", item.Preview, coverKey)
	}
	if item := byID[textShare.ShareID]; item.Preview == nil || item.Preview.coverStorageKey != "" {
		t.Fatalf("列表纯文字瞬间不得带封面 key: %#v", item.Preview)
	}
}
