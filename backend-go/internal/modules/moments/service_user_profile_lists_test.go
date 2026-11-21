package moments

import (
	"context"
	"testing"
)

// users/open_the_door（include_recent_moments）与 users/fetch_moments 通过
// ListMomentsByAuthor 消费数据；spec 要求 MomentItem.images/reactions 恒定数组，
// 个人主页「最近瞬间」画廊还依赖真实图片 URL。此处锁住装配步骤不回退。
func TestListMomentsByAuthorHydratesImagesAndArrays(t *testing.T) {
	repo := &stubMomentRepo{
		authorMoments: PaginatedMoments{Items: []MomentItem{{MomentID: "m1", User: UserInfo{UserID: "u1"}}}},
		imageRows: map[string][]momentImageRow{
			"m1": {{StorageKey: "key1", ThumbnailURL: "thumb1", Width: 100, Height: 200, MimeType: "image/jpeg"}},
		},
	}
	svc := NewServiceWithDepsAndImagesAndConfig(repo, nil, nil, stubImageResolver{urls: map[string]string{"thumb1": "https://cdn.example.com/thumb1"}}, nil, nil)
	agg := &momentsReactionAggregator{result: map[string][]Reaction{}}
	svc.SetReactionAggregator(agg)

	page, err := svc.ListMomentsByAuthor(context.Background(), "u1", 20, "", false)
	if err != nil {
		t.Fatalf("ListMomentsByAuthor error: %v", err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("items = %d, want 1", len(page.Items))
	}
	item := page.Items[0]
	if len(item.Images) != 1 {
		t.Fatalf("images = %#v, want 1（主页画廊依赖图片列装载）", item.Images)
	}
	if item.Images[0].URL != "https://cdn.example.com/key1" {
		t.Fatalf("image url = %q", item.Images[0].URL)
	}
	if item.Reactions == nil || len(item.Reactions) != 0 {
		t.Fatalf("reactions = %#v, want 非 nil 空数组（spec 恒定形状）", item.Reactions)
	}
	if item.Tags == nil || item.Mentions == nil {
		t.Fatalf("tags/mentions 必须为数组：tags=%#v mentions=%#v", item.Tags, item.Mentions)
	}
	if len(agg.calls) == 0 {
		t.Fatal("reactions aggregator 未被调用")
	}
}

func TestListReactedByUserHydratesImagesAndArrays(t *testing.T) {
	repo := &stubMomentRepo{
		reactedMoments: PaginatedMoments{Items: []MomentItem{{MomentID: "m2", User: UserInfo{UserID: "u2"}}}},
		imageRows: map[string][]momentImageRow{
			"m2": {{StorageKey: "key2"}},
		},
	}
	svc := NewServiceWithDepsAndImagesAndConfig(repo, nil, nil, stubImageResolver{}, nil, nil)
	svc.SetReactionAggregator(&momentsReactionAggregator{result: map[string][]Reaction{}})

	page, err := svc.ListReactedByUser(context.Background(), "u2", "viewer1", 20, "")
	if err != nil {
		t.Fatalf("ListReactedByUser error: %v", err)
	}
	if len(page.Items) != 1 || len(page.Items[0].Images) != 1 {
		t.Fatalf("items/images = %#v", page.Items)
	}
	if page.Items[0].Reactions == nil {
		t.Fatal("reactions 必须为非 nil 数组")
	}
}
