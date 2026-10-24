package admin

import (
	"context"
	"testing"
	"time"

	systemmodule "hotspot/backend-go/internal/modules/system"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/testdb"
)

// 本文件锁定管理后台各列表的 keyset 分页不变量：翻页必须能枚举全部行、
// 不重不漏。历史缺陷（2026-10-24 审计）：多个列表把「第 limit+1 条（下一页
// 首条）」当游标，而谓词是严格排他 → 每个分页边界静默丢一条。

func TestInMemoryModerationStoreTagsPaginationEnumeratesAll(t *testing.T) {
	store := NewInMemoryModerationStore()
	now := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	store.tags = map[string]TagRecord{}
	for i, name := range []string{"t0", "t1", "t2", "t3", "t4"} {
		store.tags[name] = TagRecord{Name: name, UsageCount: 5 - i, CreatedAt: now}
	}
	seen := map[string]bool{}
	var cursor *tagCursor
	pages := 0
	for {
		page, err := store.SearchTags(context.Background(), TagSearchFilter{Limit: 2, Cursor: cursor})
		if err != nil {
			t.Fatalf("page %d: %v", pages, err)
		}
		if len(page.Items) > 2 {
			t.Fatalf("page %d size = %d", pages, len(page.Items))
		}
		for _, item := range page.Items {
			if seen[item.Name] {
				t.Fatalf("duplicate tag across pages: %s", item.Name)
			}
			seen[item.Name] = true
		}
		pages++
		if page.Next == nil {
			break
		}
		cursor = page.Next
		if pages > 5 {
			t.Fatal("pagination did not terminate")
		}
	}
	if len(seen) != 5 {
		t.Fatalf("seen %d tags, want 5 (page boundary must not drop rows)", len(seen))
	}
}

func TestInMemoryModerationStoreAnnouncementsPaginationEnumeratesAll(t *testing.T) {
	store := NewInMemoryModerationStore()
	now := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	store.announcements = map[string]systemmodule.AnnouncementRecord{}
	for i := 0; i < 5; i++ {
		id := []string{"ann_0", "ann_1", "ann_2", "ann_3", "ann_4"}[i]
		store.announcements[id] = systemmodule.AnnouncementRecord{
			AnnouncementID: id,
			Title:          id,
			CreatedAt:      now.Add(time.Duration(i) * time.Minute),
		}
	}
	seen := map[string]bool{}
	var cursor *systemmodule.AnnouncementCursor
	pages := 0
	for {
		page, err := store.SearchAnnouncements(context.Background(), systemmodule.AnnouncementSearchFilter{Limit: 2, Cursor: cursor})
		if err != nil {
			t.Fatalf("page %d: %v", pages, err)
		}
		for _, item := range page.Items {
			if seen[item.AnnouncementID] {
				t.Fatalf("duplicate announcement across pages: %s", item.AnnouncementID)
			}
			seen[item.AnnouncementID] = true
		}
		pages++
		if page.Next == nil {
			break
		}
		cursor = page.Next
		if pages > 5 {
			t.Fatal("pagination did not terminate")
		}
	}
	if len(seen) != 5 {
		t.Fatalf("seen %d announcements, want 5 (page boundary must not drop rows)", len(seen))
	}
}

func TestInMemoryRBACListAdminsPaginationEnumeratesAll(t *testing.T) {
	repo := NewInMemoryRepository()
	repo.users = map[string]UserInfo{}
	repo.admins = map[string]adminGrantRecord{}
	grantedAt := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 5; i++ {
		userID := []string{"it_rbac_0", "it_rbac_1", "it_rbac_2", "it_rbac_3", "it_rbac_4"}[i]
		repo.users[userID] = UserInfo{UserID: userID, Nickname: userID}
		repo.admins[userID] = adminGrantRecord{
			userID:              userID,
			grantAllPermissions: true,
			grantedAt:           grantedAt.Add(time.Duration(i) * time.Minute),
		}
	}
	seen := map[string]bool{}
	cursor := ""
	pages := 0
	for {
		page, err := repo.ListAdmins(context.Background(), PaginationParams{Limit: 2, Cursor: cursor})
		if err != nil {
			t.Fatalf("page %d: %v", pages, err)
		}
		if len(page.Items) > 2 {
			t.Fatalf("page %d size = %d", pages, len(page.Items))
		}
		for _, item := range page.Items {
			if seen[item.User.UserID] {
				t.Fatalf("duplicate admin across pages: %s", item.User.UserID)
			}
			seen[item.User.UserID] = true
		}
		pages++
		if page.Next == nil {
			break
		}
		cursor = encodeAdminCursor(*page.Next)
		if pages > 5 {
			t.Fatal("pagination did not terminate")
		}
	}
	if len(seen) != 5 {
		t.Fatalf("seen %d admins, want 5 (page boundary must not drop rows)", len(seen))
	}
}

func TestPostgresModerationStoreTagsPaginationEnumeratesAll(t *testing.T) {
	testdb.SkipIfShort(t)
	testdb.Open(t)
	txFactory, cleanup := testdb.BeginScope(t)
	defer cleanup()
	qdb, ok := txFactory.DB.(persistence.QueryableDB)
	if !ok {
		t.Fatal("factory.DB is not QueryableDB")
	}
	ctx := context.Background()
	userID := "it_admin_tags_user"
	momentID := "it_admin_tags_moment"
	testdb.InsertUser(t, txFactory, userID, "nick_"+userID)
	testdb.InsertMoment(t, txFactory, momentID, userID)
	now := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	for i, name := range []string{"p0", "p1", "p2", "p3", "p4"} {
		if err := qdb.Exec(ctx, "INSERT INTO moment_tags (moment_id, tag_name, created_at) VALUES ($1,$2,$3)", momentID, name, now.Add(time.Duration(i)*time.Minute)); err != nil {
			t.Fatalf("insert tag %s: %v", name, err)
		}
	}
	store := NewPersistenceModerationStore(txFactory, nil)
	seen := map[string]bool{}
	var cursor *tagCursor
	pages := 0
	for {
		page, err := store.SearchTags(ctx, TagSearchFilter{Limit: 2, Cursor: cursor})
		if err != nil {
			t.Fatalf("page %d: %v", pages, err)
		}
		if len(page.Items) > 2 {
			t.Fatalf("page %d size = %d", pages, len(page.Items))
		}
		for _, item := range page.Items {
			if seen[item.Name] {
				t.Fatalf("duplicate tag across pages: %s", item.Name)
			}
			seen[item.Name] = true
		}
		pages++
		if page.Next == nil {
			break
		}
		cursor = page.Next
		if pages > 6 {
			t.Fatal("pagination did not terminate")
		}
	}
	if len(seen) != 5 {
		t.Fatalf("seen %d tags, want 5 (page boundary must not drop rows): %v", len(seen), seen)
	}
}
