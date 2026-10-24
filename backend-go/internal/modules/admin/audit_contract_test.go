package admin

import (
	"context"
	"errors"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/testdb"
)

// adminAuditContractState 契约用例上下文：审计表无外键依赖，仅需 ctx。
type adminAuditContractState struct {
	ctx context.Context
}

// TestAdminAuditLogContract 接入双实现契约一致性基建：
// 审计日志的追加/keyset 分页/字段过滤/时间闭区间/includeIPAddress 开关 必须一致。
func TestAdminAuditLogContract(t *testing.T) {
	base := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)

	impls := []testdb.ContractImpl[Repository, adminAuditContractState]{
		{
			Name: "memory",
			New: func(t *testing.T) (Repository, adminAuditContractState, func()) {
				// 内存实现构造时会预置演示审计数据；契约用例需要干净状态。
				repo := NewInMemoryRepository()
				repo.auditLogs = make(map[string]AuditLogEntry)
				repo.auditOrder = nil
				return repo, adminAuditContractState{ctx: context.Background()}, nil
			},
		},
		{
			Name: "postgres",
			New: func(t *testing.T) (Repository, adminAuditContractState, func()) {
				testdb.SkipIfShort(t)
				testdb.Open(t)
				txFactory, cleanup := testdb.BeginScope(t)
				return NewPostgresRepository(txFactory), adminAuditContractState{ctx: context.Background()}, cleanup
			},
		},
	}

	entry := func(logID string, createdAt time.Time, action AuditAction, operatorID, targetType, targetID string) AuditLogEntry {
		return AuditLogEntry{
			LogID:      logID,
			Action:     action,
			Operator:   UserInfo{UserID: operatorID, Nickname: "op_" + operatorID},
			TargetType: AuditTargetType(targetType),
			TargetID:   targetID,
			TargetName: "目标",
			Details:    AuditDetails{"note": "value"},
			IPLocation: "CN",
			IPAddress:  "1.2.3.4",
			CreatedAt:  createdAt,
		}
	}

	appendEntry := func(t *testing.T, repo Repository, state adminAuditContractState, e AuditLogEntry) {
		t.Helper()
		if err := repo.AppendAuditLog(state.ctx, e); err != nil {
			t.Fatalf("append audit log %s: %v", e.LogID, err)
		}
	}

	testdb.RunContract(t, impls, []testdb.ContractCase[Repository, adminAuditContractState]{
		{
			Name: "list orders by created_at desc then log_id desc",
			Run: func(t *testing.T, repo Repository, state adminAuditContractState) {
				appendEntry(t, repo, state, entry("audit_t0", base, AuditActionAnnouncementCreate, "op1", "announcement", "a1"))
				appendEntry(t, repo, state, entry("audit_t1", base.Add(time.Minute), AuditActionAnnouncementUpdate, "op1", "announcement", "a1"))
				appendEntry(t, repo, state, entry("audit_a", base.Add(2*time.Minute), AuditActionRoleCreate, "op2", "role", "r1"))
				appendEntry(t, repo, state, entry("audit_b", base.Add(2*time.Minute), AuditActionRoleUpdate, "op2", "role", "r1"))
				appendEntry(t, repo, state, entry("audit_t3", base.Add(3*time.Minute), AuditActionConfigUpdate, "op3", "config", "c1"))

				page, err := repo.ListAuditLogs(state.ctx, AuditLogFilter{Pagination: PaginationParams{Limit: 10}}, true)
				if err != nil {
					t.Fatalf("list: %v", err)
				}
				want := []string{"audit_t3", "audit_b", "audit_a", "audit_t1", "audit_t0"}
				if len(page.Items) != len(want) {
					t.Fatalf("items = %d, want %d", len(page.Items), len(want))
				}
				for i := range want {
					if page.Items[i].LogID != want[i] {
						t.Fatalf("order = %v, want %v", idsOf(page.Items), want)
					}
				}
				if page.Items[0].CreatedAt == "" {
					t.Fatal("created_at must be non-empty")
				}
				if _, err := time.Parse(time.RFC3339, page.Items[0].CreatedAt); err != nil {
					t.Fatalf("created_at must be RFC3339: %q", page.Items[0].CreatedAt)
				}
			},
		},
		{
			Name: "filters narrow the result set",
			Run: func(t *testing.T, repo Repository, state adminAuditContractState) {
				appendEntry(t, repo, state, entry("audit_f1", base, AuditActionAnnouncementCreate, "op1", "announcement", "a1"))
				appendEntry(t, repo, state, entry("audit_f2", base.Add(time.Minute), AuditActionRoleCreate, "op2", "role", "r1"))
				appendEntry(t, repo, state, entry("audit_f3", base.Add(2*time.Minute), AuditActionRoleUpdate, "op2", "role", "r2"))

				cases := []struct {
					name   string
					filter AuditLogFilter
					want   []string
				}{
					{"action", AuditLogFilter{Action: AuditActionRoleCreate}, []string{"audit_f2"}},
					{"operator", AuditLogFilter{OperatorID: "op2"}, []string{"audit_f3", "audit_f2"}},
					{"target type", AuditLogFilter{TargetType: AuditTargetTypeRole}, []string{"audit_f3", "audit_f2"}},
					{"target id", AuditLogFilter{TargetID: "r1"}, []string{"audit_f2"}},
					{"combined", AuditLogFilter{OperatorID: "op2", TargetType: AuditTargetTypeRole, TargetID: "r2"}, []string{"audit_f3"}},
					{"no match", AuditLogFilter{Action: AuditActionGroupDismiss}, []string{}},
				}
				for _, tc := range cases {
					tc.filter.Pagination = PaginationParams{Limit: 50}
					page, err := repo.ListAuditLogs(state.ctx, tc.filter, true)
					if err != nil {
						t.Fatalf("%s: %v", tc.name, err)
					}
					if got := idsOf(page.Items); !equalIDs(got, tc.want) {
						t.Fatalf("%s ids = %v, want %v", tc.name, got, tc.want)
					}
				}
			},
		},
		{
			Name: "time range is inclusive on both ends",
			Run: func(t *testing.T, repo Repository, state adminAuditContractState) {
				appendEntry(t, repo, state, entry("audit_before", base.Add(-time.Minute), AuditActionAnnouncementCreate, "op1", "announcement", "a1"))
				appendEntry(t, repo, state, entry("audit_at_start", base, AuditActionAnnouncementUpdate, "op1", "announcement", "a1"))
				appendEntry(t, repo, state, entry("audit_mid", base.Add(time.Hour), AuditActionRoleCreate, "op2", "role", "r1"))
				appendEntry(t, repo, state, entry("audit_at_end", base.Add(2*time.Hour), AuditActionConfigUpdate, "op3", "config", "c1"))
				appendEntry(t, repo, state, entry("audit_after", base.Add(3*time.Hour), AuditActionTagSetBan, "op4", "tag", "t1"))

				start, end := base, base.Add(2*time.Hour)
				page, err := repo.ListAuditLogs(state.ctx, AuditLogFilter{StartTime: &start, EndTime: &end, Pagination: PaginationParams{Limit: 50}}, true)
				if err != nil {
					t.Fatalf("list: %v", err)
				}
				want := []string{"audit_at_end", "audit_mid", "audit_at_start"}
				if got := idsOf(page.Items); !equalIDs(got, want) {
					t.Fatalf("range ids = %v, want %v", got, want)
				}
			},
		},
		{
			Name: "pagination is stable and disjoint",
			Run: func(t *testing.T, repo Repository, state adminAuditContractState) {
				for i := 0; i < 5; i++ {
					appendEntry(t, repo, state, entry(
						[]string{"audit_p0", "audit_p1", "audit_p2", "audit_p3", "audit_p4"}[i],
						base.Add(time.Duration(i)*time.Minute),
						AuditActionAnnouncementCreate, "op1", "announcement", "a1"))
				}
				seen := map[string]bool{}
				cursor := ""
				pages := 0
				for {
					filter := AuditLogFilter{Pagination: PaginationParams{Limit: 2, Cursor: cursor}}
					page, err := repo.ListAuditLogs(state.ctx, filter, false)
					if err != nil {
						t.Fatalf("page %d: %v", pages, err)
					}
					if len(page.Items) > 2 {
						t.Fatalf("page %d size = %d", pages, len(page.Items))
					}
					for _, item := range page.Items {
						if seen[item.LogID] {
							t.Fatalf("duplicate audit log across pages: %s", item.LogID)
						}
						seen[item.LogID] = true
					}
					pages++
					if page.Next == nil {
						break
					}
					cursor = encodeAuditCursor(*page.Next)
					if pages > 6 {
						t.Fatal("pagination did not terminate")
					}
				}
				if len(seen) != 5 {
					t.Fatalf("seen %d audit logs, want 5 (no row loss/duplication)", len(seen))
				}
			},
		},
		{
			Name: "includeIPAddress gates only ip_address",
			Run: func(t *testing.T, repo Repository, state adminAuditContractState) {
				appendEntry(t, repo, state, entry("audit_ip", base, AuditActionAdminAward, "op1", "admin", "ad1"))

				withIP, err := repo.ListAuditLogs(state.ctx, AuditLogFilter{Pagination: PaginationParams{Limit: 10}}, true)
				if err != nil {
					t.Fatalf("list with ip: %v", err)
				}
				if len(withIP.Items) != 1 {
					t.Fatalf("items = %d", len(withIP.Items))
				}
				item := withIP.Items[0]
				if item.IPAddress == nil || *item.IPAddress != "1.2.3.4" {
					t.Fatalf("ip_address = %#v, want 1.2.3.4", item.IPAddress)
				}
				if item.IPLocation == nil || *item.IPLocation != "CN" {
					t.Fatalf("ip_location = %#v", item.IPLocation)
				}

				withoutIP, err := repo.ListAuditLogs(state.ctx, AuditLogFilter{Pagination: PaginationParams{Limit: 10}}, false)
				if err != nil {
					t.Fatalf("list without ip: %v", err)
				}
				if len(withoutIP.Items) != 1 || withoutIP.Items[0].IPAddress != nil {
					t.Fatalf("ip_address without flag = %#v, want nil", withoutIP.Items)
				}
				if withoutIP.Items[0].IPLocation == nil || *withoutIP.Items[0].IPLocation != "CN" {
					t.Fatalf("ip_location must remain visible: %#v", withoutIP.Items[0].IPLocation)
				}
			},
		},
		{
			Name: "details and nullable fields round-trip",
			Run: func(t *testing.T, repo Repository, state adminAuditContractState) {
				e := entry("audit_nullable", base, AuditActionTagSetBan, "op1", "tag", "t1")
				e.TargetName = ""
				e.IPLocation = ""
				e.DeviceSummary = ""
				e.IPAddress = ""
				appendEntry(t, repo, state, e)
				page, err := repo.ListAuditLogs(state.ctx, AuditLogFilter{Pagination: PaginationParams{Limit: 10}}, true)
				if err != nil {
					t.Fatalf("list: %v", err)
				}
				if len(page.Items) != 1 {
					t.Fatalf("items = %d", len(page.Items))
				}
				item := page.Items[0]
				if item.TargetName != nil {
					t.Fatalf("target_name = %#v, want nil", item.TargetName)
				}
				if item.IPAddress != nil {
					t.Fatalf("ip_address = %#v, want nil", item.IPAddress)
				}
				if item.Details["note"] != "value" {
					t.Fatalf("details = %#v", item.Details)
				}
			},
		},
		{
			Name: "invalid cursor is rejected as 10000",
			Run: func(t *testing.T, repo Repository, state adminAuditContractState) {
				_, err := repo.ListAuditLogs(state.ctx, AuditLogFilter{Pagination: PaginationParams{Limit: 10, Cursor: "not-a-cursor"}}, true)
				if err == nil {
					t.Fatal("expected invalid cursor error")
				}
				var apiErr *httpx.APIError
				if !errors.As(err, &apiErr) {
					t.Fatalf("err type = %T, want *httpx.APIError", err)
				}
				if apiErr.Code != httpx.CodeInvalidRequest {
					t.Fatalf("err code = %d, want %d", apiErr.Code, httpx.CodeInvalidRequest)
				}
			},
		},
	})
}

func idsOf(items []AuditLogItem) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, item.LogID)
	}
	return out
}

func equalIDs(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
