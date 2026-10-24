package system

import (
	"context"
	"errors"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/testdb"
)

// systemAnnouncementContractState 契约用例上下文：PG 侧 system_announcements.created_by_user_id
// 有 users 外键，需预置操作人；内存实现无需。
type systemAnnouncementContractState struct {
	ctx   context.Context
	actor AnnouncementActor
}

// TestSystemAnnouncementContract 接入双实现契约一致性基建：
// 公告创建/读取/公开列表、后台检索（severity 过滤 + keyset 分页不重不漏 + 含已删）、
// 编辑 patch/删除幂等、AnnouncementsVersion 变更指纹。
func TestSystemAnnouncementContract(t *testing.T) {
	actorID := "it_system_ann_actor"

	impls := []testdb.ContractImpl[Repository, systemAnnouncementContractState]{
		{
			Name: "memory",
			New: func(t *testing.T) (Repository, systemAnnouncementContractState, func()) {
				actor := AnnouncementActor{UserID: actorID, Nickname: "nick_" + actorID}
				return NewInMemoryRepository(), systemAnnouncementContractState{ctx: context.Background(), actor: actor}, nil
			},
		},
		{
			Name: "postgres",
			New: func(t *testing.T) (Repository, systemAnnouncementContractState, func()) {
				testdb.SkipIfShort(t)
				testdb.Open(t)
				txFactory, cleanup := testdb.BeginScope(t)
				testdb.InsertUser(t, txFactory, actorID, "nick_"+actorID)
				actor := AnnouncementActor{UserID: actorID, Nickname: "nick_" + actorID}
				return NewPostgresRepository(txFactory), systemAnnouncementContractState{ctx: context.Background(), actor: actor}, cleanup
			},
		},
	}

	create := func(t *testing.T, repo Repository, state systemAnnouncementContractState, title string, severity AnnounceSeverity) AnnouncementRecord {
		t.Helper()
		item, err := repo.CreateAnnouncement(state.ctx, AnnouncementCreateInput{
			Actor: state.actor,
			// 裁剪属于 admin service 职责（service 已 TrimSpace）；repo 契约只验证持久化原样。
			Title:    title,
			Content:  "content of " + title,
			Severity: severity,
		})
		if err != nil {
			t.Fatalf("create announcement %q: %v", title, err)
		}
		if item.AnnouncementID == "" {
			t.Fatal("created announcement id must be non-empty")
		}
		return item
	}

	testdb.RunContract(t, impls, []testdb.ContractCase[Repository, systemAnnouncementContractState]{
		{
			Name: "create and get round-trips fields",
			Run: func(t *testing.T, repo Repository, state systemAnnouncementContractState) {
				created := create(t, repo, state, "Welcome", AnnounceLevelInfo)
				got, found, err := repo.GetAnnouncement(state.ctx, created.AnnouncementID)
				if err != nil || !found {
					t.Fatalf("get: found=%v err=%v", found, err)
				}
				if got.Title != "Welcome" || got.Content != "content of Welcome" {
					t.Fatalf("trimmed fields = %#v", got)
				}
				if got.Severity != AnnounceLevelInfo {
					t.Fatalf("severity = %q", got.Severity)
				}
				if got.CreatedBy.UserID != state.actor.UserID || got.CreatedBy.Nickname != state.actor.Nickname {
					t.Fatalf("created_by = %#v", got.CreatedBy)
				}
				if got.IsDeleted() {
					t.Fatalf("new announcement must not be deleted: %#v", got)
				}
				if _, err := time.Parse(time.RFC3339, got.CreatedAt.UTC().Format(time.RFC3339)); err != nil {
					t.Fatalf("created_at = %v", got.CreatedAt)
				}
				if _, found, err := repo.GetAnnouncement(state.ctx, "ann_missing"); err != nil || found {
					t.Fatalf("get missing = found:%v err:%v", found, err)
				}
			},
		},
		{
			Name: "public list is newest-first and hides deleted",
			Run: func(t *testing.T, repo Repository, state systemAnnouncementContractState) {
				first := create(t, repo, state, "First", AnnounceLevelInfo)
				time.Sleep(2 * time.Millisecond)
				second := create(t, repo, state, "Second", AnnounceLevelWarning)
				time.Sleep(2 * time.Millisecond)
				third := create(t, repo, state, "Third", AnnounceLevelInfo)

				list, err := repo.ListAnnouncements(state.ctx, 10)
				if err != nil {
					t.Fatalf("list: %v", err)
				}
				if len(list) != 3 {
					t.Fatalf("list size = %d, want 3", len(list))
				}
				wantOrder := []string{third.AnnouncementID, second.AnnouncementID, first.AnnouncementID}
				for i := range wantOrder {
					if list[i].AnnouncementID != wantOrder[i] {
						t.Fatalf("order = %v, want %v", announcementIDs(list), wantOrder)
					}
				}
				limited, err := repo.ListAnnouncements(state.ctx, 2)
				if err != nil {
					t.Fatalf("list limited: %v", err)
				}
				if len(limited) != 2 {
					t.Fatalf("limited list size = %d, want 2", len(limited))
				}
				if err := repo.DeleteAnnouncement(state.ctx, second.AnnouncementID); err != nil {
					t.Fatalf("delete: %v", err)
				}
				list, err = repo.ListAnnouncements(state.ctx, 10)
				if err != nil {
					t.Fatalf("list after delete: %v", err)
				}
				if len(list) != 2 {
					t.Fatalf("list after delete size = %d, want 2", len(list))
				}
				for _, item := range list {
					if item.AnnouncementID == second.AnnouncementID {
						t.Fatalf("deleted announcement leaked into public list: %#v", list)
					}
				}
			},
		},
		{
			Name: "admin search filters severity and enumerates all rows",
			Run: func(t *testing.T, repo Repository, state systemAnnouncementContractState) {
				for i := 0; i < 5; i++ {
					create(t, repo, state, []string{"A0", "A1", "A2", "A3", "A4"}[i], AnnounceLevelInfo)
					time.Sleep(2 * time.Millisecond)
				}
				create(t, repo, state, "W1", AnnounceLevelWarning)

				all, err := repo.SearchAnnouncements(state.ctx, AnnouncementSearchFilter{Limit: 100})
				if err != nil {
					t.Fatalf("search all: %v", err)
				}
				if len(all.Items) != 6 || all.Next != nil {
					t.Fatalf("search all = %d items next=%v", len(all.Items), all.Next)
				}
				warnings, err := repo.SearchAnnouncements(state.ctx, AnnouncementSearchFilter{Severity: AnnounceLevelWarning, Limit: 100})
				if err != nil {
					t.Fatalf("search warning: %v", err)
				}
				if len(warnings.Items) != 1 || warnings.Items[0].Title != "W1" {
					t.Fatalf("severity filter = %#v", warnings.Items)
				}

				// keyset 分页必须不重不漏：每页边界丢行会让总数少于创建数。
				seen := map[string]bool{}
				var cursor *AnnouncementCursor
				pages := 0
				for {
					page, err := repo.SearchAnnouncements(state.ctx, AnnouncementSearchFilter{Limit: 2, Cursor: cursor})
					if err != nil {
						t.Fatalf("page %d: %v", pages, err)
					}
					if len(page.Items) > 2 {
						t.Fatalf("page %d size = %d", pages, len(page.Items))
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
					if pages > 6 {
						t.Fatal("pagination did not terminate")
					}
					cursor = page.Next
				}
				if len(seen) != 6 {
					t.Fatalf("seen %d announcements, want 6 (page boundary must not drop rows)", len(seen))
				}
			},
		},
		{
			Name: "admin search includes deleted records",
			Run: func(t *testing.T, repo Repository, state systemAnnouncementContractState) {
				item := create(t, repo, state, "Gone", AnnounceLevelError)
				if err := repo.DeleteAnnouncement(state.ctx, item.AnnouncementID); err != nil {
					t.Fatalf("delete: %v", err)
				}
				page, err := repo.SearchAnnouncements(state.ctx, AnnouncementSearchFilter{Limit: 10})
				if err != nil {
					t.Fatalf("search: %v", err)
				}
				if len(page.Items) != 1 {
					t.Fatalf("search items = %d, want 1", len(page.Items))
				}
				if !page.Items[0].IsDeleted() || page.Items[0].DeletedAt == nil {
					t.Fatalf("deleted_at = %#v", page.Items[0])
				}
			},
		},
		{
			Name: "update applies patch semantics and rejects deleted/missing",
			Run: func(t *testing.T, repo Repository, state systemAnnouncementContractState) {
				item := create(t, repo, state, "Before", AnnounceLevelInfo)
				title := "After"
				severity := AnnounceLevelError
				updated, err := repo.UpdateAnnouncement(state.ctx, AnnouncementUpdateInput{
					AnnouncementID: item.AnnouncementID,
					Title:          &title,
					Severity:       &severity,
				})
				if err != nil {
					t.Fatalf("update: %v", err)
				}
				if updated.Title != "After" || updated.Severity != AnnounceLevelError {
					t.Fatalf("updated = %#v", updated)
				}
				got, _, err := repo.GetAnnouncement(state.ctx, item.AnnouncementID)
				if err != nil {
					t.Fatalf("get after update: %v", err)
				}
				if got.Title != "After" || got.Content != "content of Before" {
					t.Fatalf("content must keep with nil patch: %#v", got)
				}
				if err := repo.DeleteAnnouncement(state.ctx, item.AnnouncementID); err != nil {
					t.Fatalf("delete: %v", err)
				}
				if _, err := repo.UpdateAnnouncement(state.ctx, AnnouncementUpdateInput{AnnouncementID: item.AnnouncementID, Title: &title}); !errors.Is(err, ErrAnnouncementNotFound()) {
					t.Fatalf("update deleted err = %v, want ErrAnnouncementNotFound", err)
				}
				if _, err := repo.UpdateAnnouncement(state.ctx, AnnouncementUpdateInput{AnnouncementID: "ann_missing", Title: &title}); !errors.Is(err, ErrAnnouncementNotFound()) {
					t.Fatalf("update missing err = %v, want ErrAnnouncementNotFound", err)
				}
			},
		},
		{
			Name: "delete is idempotent and reports missing",
			Run: func(t *testing.T, repo Repository, state systemAnnouncementContractState) {
				item := create(t, repo, state, "Delete Me", AnnounceLevelInfo)
				if err := repo.DeleteAnnouncement(state.ctx, item.AnnouncementID); err != nil {
					t.Fatalf("delete: %v", err)
				}
				if err := repo.DeleteAnnouncement(state.ctx, item.AnnouncementID); err != nil {
					t.Fatalf("second delete must be idempotent: %v", err)
				}
				if err := repo.DeleteAnnouncement(state.ctx, "ann_missing"); !errors.Is(err, ErrAnnouncementNotFound()) {
					t.Fatalf("delete missing err = %v, want ErrAnnouncementNotFound", err)
				}
				got, found, err := repo.GetAnnouncement(state.ctx, item.AnnouncementID)
				if err != nil || !found {
					t.Fatalf("get deleted: found=%v err=%v", found, err)
				}
				if !got.IsDeleted() {
					t.Fatalf("deleted record = %#v", got)
				}
			},
		},
		{
			Name: "announcements version changes on every mutation",
			Run: func(t *testing.T, repo Repository, state systemAnnouncementContractState) {
				baseline, err := repo.AnnouncementsVersion(state.ctx)
				if err != nil {
					t.Fatalf("version baseline: %v", err)
				}
				item := create(t, repo, state, "Versioned", AnnounceLevelInfo)
				afterCreate, err := repo.AnnouncementsVersion(state.ctx)
				if err != nil {
					t.Fatalf("version after create: %v", err)
				}
				if afterCreate == baseline {
					t.Fatalf("version unchanged after create: %q", afterCreate)
				}
				time.Sleep(5 * time.Millisecond)
				title := "Versioned 2"
				if _, err := repo.UpdateAnnouncement(state.ctx, AnnouncementUpdateInput{AnnouncementID: item.AnnouncementID, Title: &title}); err != nil {
					t.Fatalf("update: %v", err)
				}
				afterUpdate, err := repo.AnnouncementsVersion(state.ctx)
				if err != nil {
					t.Fatalf("version after update: %v", err)
				}
				if afterUpdate == afterCreate {
					t.Fatalf("version unchanged after update: %q", afterUpdate)
				}
				time.Sleep(5 * time.Millisecond)
				if err := repo.DeleteAnnouncement(state.ctx, item.AnnouncementID); err != nil {
					t.Fatalf("delete: %v", err)
				}
				afterDelete, err := repo.AnnouncementsVersion(state.ctx)
				if err != nil {
					t.Fatalf("version after delete: %v", err)
				}
				if afterDelete == afterUpdate {
					t.Fatalf("version unchanged after delete: %q", afterDelete)
				}
			},
		},
	})
}

func announcementIDs(items []AnnouncementRecord) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, item.AnnouncementID)
	}
	return out
}
