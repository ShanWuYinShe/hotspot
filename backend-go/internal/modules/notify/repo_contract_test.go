package notify

import (
	"context"
	"errors"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/testdb"
)

// notifyContractState 契约用例上下文：内存实现无需预置；真实库实现需先插入
// users 行（notifications.user_id 外键）。
type notifyContractState struct {
	ctx         context.Context
	userID      string
	otherUserID string
}

func notifyTime(hour int) string {
	return time.Date(2026, 5, 1, hour, 0, 0, 0, time.UTC).Format(time.RFC3339)
}

// TestNotifyRepositoryContract 双实现契约一致性（审查遗留项：契约测试基建
// 首个私有模块之外接入 notify）：同一组用例跑内存实现与真实库实现，覆盖
// 列表排序/keyset 分页/过滤/未读分类计数/已读语义/非法游标。
func TestNotifyRepositoryContract(t *testing.T) {
	userID := "it_notify_contract_u1"
	otherUserID := "it_notify_contract_u2"

	impls := []testdb.ContractImpl[Repository, notifyContractState]{
		{
			Name: "memory",
			New: func(t *testing.T) (Repository, notifyContractState, func()) {
				return NewInMemoryRepository(), notifyContractState{ctx: context.Background(), userID: userID, otherUserID: otherUserID}, nil
			},
		},
		{
			Name: "postgres",
			New: func(t *testing.T) (Repository, notifyContractState, func()) {
				testdb.SkipIfShort(t)
				testdb.Open(t)
				txFactory, cleanup := testdb.BeginScope(t)
				testdb.InsertUser(t, txFactory, userID, "NotifyContract1")
				testdb.InsertUser(t, txFactory, otherUserID, "NotifyContract2")
				return NewPostgresRepository(txFactory), notifyContractState{ctx: context.Background(), userID: userID, otherUserID: otherUserID}, cleanup
			},
		},
	}

	appendItem := func(t *testing.T, repo Repository, userID string, item NotifyItem) {
		t.Helper()
		if err := repo.Append(context.Background(), userID, item); err != nil {
			t.Fatalf("append %s: %v", item.NotifyID, err)
		}
	}

	testdb.RunContract(t, impls, []testdb.ContractCase[Repository, notifyContractState]{
		{
			Name: "append and list normalizes payload",
			Run: func(t *testing.T, repo Repository, state notifyContractState) {
				appendItem(t, repo, state.userID, NotifyItem{Type: "new_follower", Payload: map[string]any{"follower_id": "u2"}})
				page, err := repo.List(state.ctx, state.userID, NotifyListRequest{})
				if err != nil {
					t.Fatalf("list: %v", err)
				}
				if len(page.Items) != 1 {
					t.Fatalf("items = %#v", page.Items)
				}
				item := page.Items[0]
				if item.Type != "new_follower" || item.IsRead || item.Payload["type"] != "new_follower" || item.Payload["follower_id"] != "u2" {
					t.Fatalf("item = %#v", item)
				}
				if _, err := time.Parse(time.RFC3339, item.CreatedAt); err != nil {
					t.Fatalf("created_at must be RFC3339, got %q: %v", item.CreatedAt, err)
				}
			},
		},
		{
			Name: "list order is created_at desc then notify_id desc",
			Run: func(t *testing.T, repo Repository, state notifyContractState) {
				for _, item := range []NotifyItem{
					{NotifyID: "notif_contract_c", Type: "new_follower", CreatedAt: notifyTime(0)},
					{NotifyID: "notif_contract_d", Type: "new_follower", CreatedAt: notifyTime(0)},
					{NotifyID: "notif_contract_e", Type: "new_follower", CreatedAt: notifyTime(1)},
				} {
					appendItem(t, repo, state.userID, item)
				}
				page, err := repo.List(state.ctx, state.userID, NotifyListRequest{})
				if err != nil {
					t.Fatalf("list: %v", err)
				}
				got := make([]string, 0, len(page.Items))
				for _, item := range page.Items {
					got = append(got, item.NotifyID)
				}
				want := []string{"notif_contract_e", "notif_contract_d", "notif_contract_c"}
				if len(got) != len(want) {
					t.Fatalf("order = %v want %v", got, want)
				}
				for i := range want {
					if got[i] != want[i] {
						t.Fatalf("order = %v want %v", got, want)
					}
				}
			},
		},
		{
			Name: "keyset pagination is stable and disjoint",
			Run: func(t *testing.T, repo Repository, state notifyContractState) {
				for i := 0; i < 5; i++ {
					appendItem(t, repo, state.userID, NotifyItem{
						NotifyID:  "notif_contract_page_" + string(rune('a'+i)),
						Type:      "moment_commented",
						CreatedAt: notifyTime(i),
					})
				}
				seen := map[string]bool{}
				cursor := ""
				for page := 0; page < 3; page++ {
					req := NotifyListRequest{Pagination: &PaginationParams{Limit: 2}}
					if cursor != "" {
						req.Pagination.Cursor = cursor
					}
					result, err := repo.List(state.ctx, state.userID, req)
					if err != nil {
						t.Fatalf("page %d: %v", page, err)
					}
					for _, item := range result.Items {
						if seen[item.NotifyID] {
							t.Fatalf("duplicate across pages: %s", item.NotifyID)
						}
						seen[item.NotifyID] = true
					}
					if page < 2 && (result.NextCursor == nil || len(result.Items) != 2) {
						t.Fatalf("page %d = %#v", page, result)
					}
					if page == 2 && (result.NextCursor != nil || len(result.Items) != 1) {
						t.Fatalf("last page = %#v", result)
					}
					if result.NextCursor != nil {
						cursor = *result.NextCursor
					}
				}
				if len(seen) != 5 {
					t.Fatalf("seen %d want 5", len(seen))
				}
			},
		},
		{
			Name: "filters by type and read state",
			Run: func(t *testing.T, repo Repository, state notifyContractState) {
				appendItem(t, repo, state.userID, NotifyItem{NotifyID: "notif_contract_social", Type: "new_follower", CreatedAt: notifyTime(0)})
				appendItem(t, repo, state.userID, NotifyItem{NotifyID: "notif_contract_report", Type: "report_resolved", CreatedAt: notifyTime(1)})
				if err := repo.MarkAsRead(state.ctx, state.userID, []string{"notif_contract_report"}); err != nil {
					t.Fatalf("mark read: %v", err)
				}
				byType, err := repo.List(state.ctx, state.userID, NotifyListRequest{Type: "report_resolved"})
				if err != nil {
					t.Fatalf("list by type: %v", err)
				}
				if len(byType.Items) != 1 || byType.Items[0].NotifyID != "notif_contract_report" {
					t.Fatalf("by type = %#v", byType.Items)
				}
				readFilter := true
				readList, err := repo.List(state.ctx, state.userID, NotifyListRequest{IsRead: &readFilter})
				if err != nil {
					t.Fatalf("list read: %v", err)
				}
				if len(readList.Items) != 1 || readList.Items[0].NotifyID != "notif_contract_report" {
					t.Fatalf("read list = %#v", readList.Items)
				}
				unreadFilter := false
				unreadList, err := repo.List(state.ctx, state.userID, NotifyListRequest{IsRead: &unreadFilter})
				if err != nil {
					t.Fatalf("list unread: %v", err)
				}
				if len(unreadList.Items) != 1 || unreadList.Items[0].NotifyID != "notif_contract_social" {
					t.Fatalf("unread list = %#v", unreadList.Items)
				}
			},
		},
		{
			Name: "unread counts bucket all six categories",
			Run: func(t *testing.T, repo Repository, state notifyContractState) {
				types := []string{"new_follower", "moment_commented", "group_invite_received", "report_resolved", "nearby_new_moment", "system_announcement"}
				for i, notifyType := range types {
					appendItem(t, repo, state.userID, NotifyItem{
						NotifyID:  "notif_contract_bucket_" + string(rune('a'+i)),
						Type:      notifyType,
						CreatedAt: notifyTime(i),
					})
				}
				summary, err := repo.UnreadCount(state.ctx, state.userID)
				if err != nil {
					t.Fatalf("unread count: %v", err)
				}
				if summary.Total != 6 {
					t.Fatalf("total = %d want 6", summary.Total)
				}
				want := UnreadCountByType{Social: 1, Interaction: 1, Group: 1, Report: 1, Nearby: 1, System: 1}
				if summary.ByType != want {
					t.Fatalf("by_type = %#v want %#v", summary.ByType, want)
				}
			},
		},
		{
			Name: "mark read semantics and cross-user isolation",
			Run: func(t *testing.T, repo Repository, state notifyContractState) {
				appendItem(t, repo, state.userID, NotifyItem{NotifyID: "notif_contract_m1", Type: "new_follower", CreatedAt: notifyTime(0)})
				appendItem(t, repo, state.userID, NotifyItem{NotifyID: "notif_contract_m2", Type: "new_follower", CreatedAt: notifyTime(1)})
				appendItem(t, repo, state.otherUserID, NotifyItem{NotifyID: "notif_contract_other", Type: "new_follower", CreatedAt: notifyTime(2)})

				// 他人 id 静默忽略，不报错、不影响他人未读
				if err := repo.MarkAsRead(state.ctx, state.userID, []string{"notif_contract_m1", "notif_contract_other"}); err != nil {
					t.Fatalf("mark read: %v", err)
				}
				mine, err := repo.UnreadCount(state.ctx, state.userID)
				if err != nil {
					t.Fatalf("unread mine: %v", err)
				}
				if mine.Total != 1 {
					t.Fatalf("my unread = %d want 1", mine.Total)
				}
				other, err := repo.UnreadCount(state.ctx, state.otherUserID)
				if err != nil {
					t.Fatalf("unread other: %v", err)
				}
				if other.Total != 1 {
					t.Fatalf("other unread = %d want 1", other.Total)
				}

				if err := repo.MarkAllRead(state.ctx, state.userID); err != nil {
					t.Fatalf("mark all read: %v", err)
				}
				mine, err = repo.UnreadCount(state.ctx, state.userID)
				if err != nil {
					t.Fatalf("unread after mark all: %v", err)
				}
				if mine.Total != 0 {
					t.Fatalf("my unread after mark all = %d want 0", mine.Total)
				}
				other, err = repo.UnreadCount(state.ctx, state.otherUserID)
				if err != nil {
					t.Fatalf("other unread after my mark all: %v", err)
				}
				if other.Total != 1 {
					t.Fatalf("other unread after my mark all = %d want 1", other.Total)
				}
			},
		},
		{
			Name: "invalid cursor is rejected",
			Run: func(t *testing.T, repo Repository, state notifyContractState) {
				if _, err := repo.List(state.ctx, state.userID, NotifyListRequest{Pagination: &PaginationParams{Cursor: "not-a-cursor"}}); !errors.Is(err, ErrInvalidNotifyRequest) {
					t.Fatalf("invalid cursor err = %v want ErrInvalidNotifyRequest", err)
				}
			},
		},
	})
}

var _ = persistence.ErrNotImplemented
