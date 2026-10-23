package groups

import (
	"errors"
	"testing"

	"hotspot/backend-go/internal/platform/actor"
)

// TestServiceUpdateAnnouncementReturnsItemViaDirectLookup 防回归（审查
// P2-2）：更新后经 GetAnnouncement 归属双键直查取回完整对象，不再全量
// List 再线性扫描；公告不存在时仍返回 ErrAnnouncementNotFound。
func TestServiceUpdateAnnouncementReturnsItemViaDirectLookup(t *testing.T) {
	title := "新标题"
	repo := &stubGroupsRepo{
		getResult: ownerGroup(),
		getFound:  true,
		announceItems: []GroupAnnouncementItem{{
			AnnouncementID: "a1",
			GroupID:        "g1",
			Title:          title,
		}},
	}
	svc := NewServiceWithRepo(repo)
	item, err := svc.UpdateAnnouncement(grpCtx("u1"), UpdateGroupAnnouncementRequest{GroupID: "g1", AnnouncementID: "a1", Title: &title})
	if err != nil {
		t.Fatalf("UpdateAnnouncement: %v", err)
	}
	if item.AnnouncementID != "a1" || item.Title != title {
		t.Fatalf("unexpected item: %#v", item)
	}

	// 更新的公告不存在 → ErrAnnouncementNotFound（直查未命中）。
	_, err = svc.UpdateAnnouncement(grpCtx("u1"), UpdateGroupAnnouncementRequest{GroupID: "g1", AnnouncementID: "missing"})
	if !errors.Is(err, ErrAnnouncementNotFound) {
		t.Fatalf("expected ErrAnnouncementNotFound, got %v", err)
	}
}

// TestServiceCreateAnnouncementResolvesCreatedByNickname 防回归（审查
// P2-3）：创建响应的 created_by 昵称由 service 经 actor provider 解析，
// 不再透传 repo 侧占位的裸 user_id。
func TestServiceCreateAnnouncementResolvesCreatedByNickname(t *testing.T) {
	repo := &stubGroupsRepo{
		getResult: ownerGroup(),
		getFound:  true,
		createAnnItem: GroupAnnouncementItem{
			AnnouncementID: "a1",
			GroupID:        "g1",
			// repo 桩返回占位昵称（模拟旧行为的裸 user_id）。
			CreatedBy: UserInfo{UserID: "u1", Nickname: "user_raw_id"},
		},
	}
	svc := NewServiceWithDeps(repo, nil, actorProviderStub{info: actor.Info{UserID: "u1", Nickname: "群主昵称"}})
	item, err := svc.CreateAnnouncement(grpCtx("u1"), CreateGroupAnnouncementRequest{GroupID: "g1", Title: "t", Content: "c"})
	if err != nil {
		t.Fatalf("CreateAnnouncement: %v", err)
	}
	if item.CreatedBy.UserID != "u1" || item.CreatedBy.Nickname != "群主昵称" {
		t.Fatalf("expected resolved created_by, got %#v", item.CreatedBy)
	}
}
