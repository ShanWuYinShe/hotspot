package reports

import (
	"context"
	"errors"
	"testing"

	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/testdb"
)

// reportsContractState 契约用例上下文：TargetOwner 在 PG 侧读真实 users/moments 表，
// 内存实现用 SeedTargetOwner 预置等价目标；两边都只覆盖 user/moment 两类目标
// （comment/message/group 需要更多跨模块种子，另由模块单测覆盖）。
type reportsContractState struct {
	ctx          context.Context
	reporter     string
	other        string
	owner        string
	userTarget   string
	momentTarget string
}

func TestReportsRepositoryContract(t *testing.T) {
	reporter := "it_reports_contract_reporter"
	other := "it_reports_contract_other"
	owner := "it_reports_contract_owner"
	userTarget := "it_reports_contract_target_user"
	momentTarget := "it_reports_contract_target_moment"

	impls := []testdb.ContractImpl[Repository, reportsContractState]{
		{
			Name: "memory",
			New: func(t *testing.T) (Repository, reportsContractState, func()) {
				repo := NewInMemoryRepository()
				repo.SeedTargetOwner(ReportTargetTypeUser, userTarget, userTarget)
				repo.SeedTargetOwner(ReportTargetTypeMoment, momentTarget, owner)
				state := reportsContractState{
					ctx: context.Background(), reporter: reporter, other: other, owner: owner,
					userTarget: userTarget, momentTarget: momentTarget,
				}
				return repo, state, nil
			},
		},
		{
			Name: "postgres",
			New: func(t *testing.T) (Repository, reportsContractState, func()) {
				testdb.SkipIfShort(t)
				testdb.Open(t)
				txFactory, cleanup := testdb.BeginScope(t)
				testdb.InsertUser(t, txFactory, reporter, "nick_"+reporter)
				testdb.InsertUser(t, txFactory, other, "nick_"+other)
				testdb.InsertUser(t, txFactory, owner, "nick_"+owner)
				testdb.InsertUser(t, txFactory, userTarget, "nick_"+userTarget)
				testdb.InsertMoment(t, txFactory, momentTarget, owner)
				state := reportsContractState{
					ctx: context.Background(), reporter: reporter, other: other, owner: owner,
					userTarget: userTarget, momentTarget: momentTarget,
				}
				return NewPostgresRepository(txFactory), state, cleanup
			},
		},
	}

	description := "spam details"

	create := func(t *testing.T, repo Repository, state reportsContractState, targetType ReportTargetType, targetID string, snapshot *ReportSnapshot) ReportCreated {
		t.Helper()
		created, err := repo.Create(state.ctx, state.reporter, CreateReportRequest{
			TargetType:  targetType,
			TargetID:    targetID,
			Reason:      "spam",
			Description: &description,
		}, snapshot)
		if err != nil {
			t.Fatalf("create report target %s/%s: %v", targetType, targetID, err)
		}
		if created.ReportID == "" {
			t.Fatal("created report id must be non-empty")
		}
		return created
	}

	testdb.RunContract(t, impls, []testdb.ContractCase[Repository, reportsContractState]{
		{
			Name: "create and list round-trips fields including message snapshot",
			Run: func(t *testing.T, repo Repository, state reportsContractState) {
				userReport := create(t, repo, state, ReportTargetTypeUser, state.userTarget, nil)
				snapshotContent := "snapshot content"
				snapshot := &ReportSnapshot{
					ChatType:        "private",
					TargetChatID:    "it_reports_chat",
					MessageType:     "text",
					SnapshotContent: &snapshotContent,
					SentAt:          "2026-05-01T00:00:00Z",
				}
				messageReport := create(t, repo, state, ReportTargetTypeMessage, "it_reports_msg", snapshot)

				page, err := repo.List(state.ctx, state.reporter, ListReportsRequest{Pagination: &PaginationParams{Limit: 10}})
				if err != nil {
					t.Fatalf("list: %v", err)
				}
				if len(page.Items) != 2 || page.NextCursor != nil {
					t.Fatalf("list = %d items next=%v", len(page.Items), page.NextCursor)
				}
				byID := map[string]ReportItem{}
				for _, item := range page.Items {
					byID[item.ReportID] = item
				}
				userItem, ok := byID[userReport.ReportID]
				if !ok {
					t.Fatalf("user report missing: %#v", page.Items)
				}
				if userItem.TargetType != ReportTargetTypeUser || userItem.TargetID != state.userTarget {
					t.Fatalf("user report target = %#v", userItem)
				}
				if userItem.Reason != "spam" || userItem.Description == nil || *userItem.Description != description {
					t.Fatalf("user report fields = %#v", userItem)
				}
				if userItem.Status != ReportStatusPending || userItem.HandledAt != nil || userItem.Resolution != nil {
					t.Fatalf("pending report = %#v", userItem)
				}
				if userItem.MessageSnapshot != nil {
					t.Fatalf("non-message report snapshot = %#v", userItem.MessageSnapshot)
				}
				messageItem, ok := byID[messageReport.ReportID]
				if !ok {
					t.Fatalf("message report missing: %#v", page.Items)
				}
				if messageItem.MessageSnapshot == nil {
					t.Fatal("message report snapshot must be stored")
				}
				if messageItem.MessageSnapshot.ChatType != "private" || messageItem.MessageSnapshot.TargetChatID != "it_reports_chat" {
					t.Fatalf("message snapshot = %#v", messageItem.MessageSnapshot)
				}
				if messageItem.MessageSnapshot.SnapshotContent == nil || *messageItem.MessageSnapshot.SnapshotContent != snapshotContent {
					t.Fatalf("message snapshot content = %#v", messageItem.MessageSnapshot.SnapshotContent)
				}
				otherPage, err := repo.List(state.ctx, state.other, ListReportsRequest{Pagination: &PaginationParams{Limit: 10}})
				if err != nil {
					t.Fatalf("list other: %v", err)
				}
				if len(otherPage.Items) != 0 {
					t.Fatalf("other reporter sees reports: %#v", otherPage.Items)
				}
			},
		},
		{
			Name: "list filters status and target type",
			Run: func(t *testing.T, repo Repository, state reportsContractState) {
				create(t, repo, state, ReportTargetTypeUser, state.userTarget, nil)
				create(t, repo, state, ReportTargetTypeMoment, state.momentTarget, nil)

				pending, err := repo.List(state.ctx, state.reporter, ListReportsRequest{Status: ReportStatusPending, Pagination: &PaginationParams{Limit: 10}})
				if err != nil {
					t.Fatalf("list pending: %v", err)
				}
				if len(pending.Items) != 2 {
					t.Fatalf("pending = %d items, want 2", len(pending.Items))
				}
				resolved, err := repo.List(state.ctx, state.reporter, ListReportsRequest{Status: ReportStatusResolved, Pagination: &PaginationParams{Limit: 10}})
				if err != nil {
					t.Fatalf("list resolved: %v", err)
				}
				if len(resolved.Items) != 0 {
					t.Fatalf("resolved = %#v, want empty", resolved.Items)
				}
				byType, err := repo.List(state.ctx, state.reporter, ListReportsRequest{TargetType: ReportTargetTypeMoment, Pagination: &PaginationParams{Limit: 10}})
				if err != nil {
					t.Fatalf("list by target type: %v", err)
				}
				if len(byType.Items) != 1 || byType.Items[0].TargetType != ReportTargetTypeMoment {
					t.Fatalf("target type filter = %#v", byType.Items)
				}
			},
		},
		{
			Name: "list pagination enumerates all rows and rejects invalid cursor",
			Run: func(t *testing.T, repo Repository, state reportsContractState) {
				for i := 0; i < 5; i++ {
					create(t, repo, state, ReportTargetTypeComment, []string{"it_comment_0", "it_comment_1", "it_comment_2", "it_comment_3", "it_comment_4"}[i], nil)
				}
				seen := map[string]bool{}
				cursor := ""
				pages := 0
				for {
					page, err := repo.List(state.ctx, state.reporter, ListReportsRequest{Pagination: &PaginationParams{Limit: 2, Cursor: cursor}})
					if err != nil {
						t.Fatalf("page %d: %v", pages, err)
					}
					if len(page.Items) > 2 {
						t.Fatalf("page %d size = %d", pages, len(page.Items))
					}
					for _, item := range page.Items {
						if seen[item.ReportID] {
							t.Fatalf("duplicate report across pages: %s", item.ReportID)
						}
						seen[item.ReportID] = true
					}
					pages++
					if page.NextCursor == nil {
						break
					}
					cursor = *page.NextCursor
					if pages > 6 {
						t.Fatal("pagination did not terminate")
					}
				}
				if len(seen) != 5 {
					t.Fatalf("seen %d reports, want 5 (page boundary must not drop rows)", len(seen))
				}
				_, err := repo.List(state.ctx, state.reporter, ListReportsRequest{Pagination: &PaginationParams{Limit: 2, Cursor: "not-a-cursor"}})
				if err == nil {
					t.Fatal("expected invalid cursor error")
				}
				var apiErr *httpx.APIError
				if !errors.As(err, &apiErr) || apiErr.Code != httpx.CodeInvalidRequest {
					t.Fatalf("invalid cursor err = %v, want *httpx.APIError 10000", err)
				}
			},
		},
		{
			Name: "find existing report is scoped to reporter and target",
			Run: func(t *testing.T, repo Repository, state reportsContractState) {
				if id, found, err := repo.FindExistingReport(state.ctx, state.reporter, ReportTargetTypeUser, state.userTarget); err != nil || found {
					t.Fatalf("before create = %q found:%v err:%v", id, found, err)
				}
				created := create(t, repo, state, ReportTargetTypeUser, state.userTarget, nil)
				id, found, err := repo.FindExistingReport(state.ctx, state.reporter, ReportTargetTypeUser, state.userTarget)
				if err != nil || !found || id != created.ReportID {
					t.Fatalf("existing = %q found:%v err:%v, want %s", id, found, err, created.ReportID)
				}
				if id, found, err := repo.FindExistingReport(state.ctx, state.other, ReportTargetTypeUser, state.userTarget); err != nil || found {
					t.Fatalf("other reporter must not match: %q found:%v err:%v", id, found, err)
				}
				if id, found, err := repo.FindExistingReport(state.ctx, state.reporter, ReportTargetTypeUser, "it_other_target"); err != nil || found {
					t.Fatalf("other target must not match: %q found:%v err:%v", id, found, err)
				}
				if id, found, err := repo.FindExistingReport(state.ctx, state.reporter, ReportTargetTypeMoment, state.userTarget); err != nil || found {
					t.Fatalf("other target type must not match: %q found:%v err:%v", id, found, err)
				}
			},
		},
		{
			Name: "target owner resolves real targets",
			Run: func(t *testing.T, repo Repository, state reportsContractState) {
				if owner, found, err := repo.TargetOwner(state.ctx, ReportTargetTypeUser, state.userTarget); err != nil || !found || owner != state.userTarget {
					t.Fatalf("user target owner = %q found:%v err:%v", owner, found, err)
				}
				if owner, found, err := repo.TargetOwner(state.ctx, ReportTargetTypeMoment, state.momentTarget); err != nil || !found || owner != state.owner {
					t.Fatalf("moment target owner = %q found:%v err:%v", owner, found, err)
				}
				if owner, found, err := repo.TargetOwner(state.ctx, ReportTargetTypeUser, "it_missing_target"); err != nil || found || owner != "" {
					t.Fatalf("missing target = %q found:%v err:%v", owner, found, err)
				}
				if owner, found, err := repo.TargetOwner(state.ctx, ReportTargetType("unknown_type"), state.userTarget); err != nil || found || owner != "" {
					t.Fatalf("unsupported target type = %q found:%v err:%v", owner, found, err)
				}
			},
		},
	})
}
