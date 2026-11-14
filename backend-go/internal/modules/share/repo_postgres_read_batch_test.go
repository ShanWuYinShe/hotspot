package share

import (
	"context"
	"errors"
	"testing"

	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/testdb"
)

// 本文件锁定第 81 轮审计的 share 列表 preview 批量解析（fetch_created N+1）：
// 一页内每类目标（用户/群组/瞬间）只查一次，单条目标查询在列表路径不再发生；
// preview 结构与「目标不可见时降级为 nil」的语义与单条 resolveSharePreview 一致。

// shareBatchRowsBackend 覆写某一条批量查询的返回行集/错误，用于覆盖批量
// 解析的扫描错误与 rows.Err 尾部错误分支。
type shareBatchRowsBackend struct {
	*shareBackendStub
	query string
	rows  persistence.Rows
	err   error
}

func (b *shareBatchRowsBackend) Query(ctx context.Context, sql string, args ...any) (persistence.Rows, error) {
	if sql == b.query {
		return b.rows, b.err
	}
	return b.shareBackendStub.Query(ctx, sql, args...)
}

func addListShare(backend *shareBackendStub, shareID, shareType string, payload map[string]any, createdAt string) {
	backend.shares[shareID] = shareStubRecord{
		ShareID:       shareID,
		ShareCode:     shareID,
		ShareType:     shareType,
		CreatorUserID: "user_alice",
		PayloadJSON:   sharePayloadJSON(payload),
		CreatedAt:     createdAt,
	}
}

func TestPostgresRepositoryListPreviewsAreBatched(t *testing.T) {
	backend := newShareBackendStub()
	backend.users["user_carol"] = shareStubUser{Nickname: "Carol", Avatar: "https://example.com/carol.png"}
	backend.users["user_ghost"] = shareStubUser{Nickname: "Ghost", Deleted: true}
	backend.groups["group_2"] = [3]string{"Group Two", "", ""}
	backend.moments["moment_no_desc"] = [5]string{"", "Shanghai", "Bob", "", "public"}

	addListShare(backend, "share_p_user_bob", ShareTypeUser, map[string]any{"shared_user_id": "user_bob"}, "2026-01-10T00:00:00Z")
	addListShare(backend, "share_p_user_carol", ShareTypeUser, map[string]any{"shared_user_id": "user_carol"}, "2026-01-09T00:00:00Z")
	addListShare(backend, "share_p_user_missing", ShareTypeUser, map[string]any{"shared_user_id": "user_missing"}, "2026-01-08T00:00:00Z")
	addListShare(backend, "share_p_user_deleted", ShareTypeUser, map[string]any{"shared_user_id": "user_ghost"}, "2026-01-07T00:00:00Z")
	addListShare(backend, "share_p_group", ShareTypeGroup, map[string]any{"shared_group_id": "group_1"}, "2026-01-06T00:00:00Z")
	addListShare(backend, "share_p_group_missing", ShareTypeGroup, map[string]any{"shared_group_id": "group_missing"}, "2026-01-05T00:00:00Z")
	addListShare(backend, "share_p_moment", ShareTypeMoment, map[string]any{"shared_moment_id": "moment_1"}, "2026-01-04T00:00:00Z")
	addListShare(backend, "share_p_moment_fallback", ShareTypeMoment, map[string]any{"shared_moment_id": "moment_no_desc"}, "2026-01-03T00:00:00Z")
	addListShare(backend, "share_p_moment_missing", ShareTypeMoment, map[string]any{"shared_moment_id": "moment_gone"}, "2026-01-02T00:00:00Z")
	addListShare(backend, "share_p_location", ShareTypeLocation, map[string]any{"shared_location": map[string]any{"location_name": "Beach"}}, "2026-01-01T00:00:00Z")

	page, err := newShareTestRepo(backend).List(context.Background(), listSharesQuery{CreatorUserID: "user_alice", Limit: 20})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(page.Items) != 10 {
		t.Fatalf("items = %d，期望 10", len(page.Items))
	}
	byID := make(map[string]ShareListItem, len(page.Items))
	for _, item := range page.Items {
		byID[item.ShareID] = item
	}
	if item := byID["share_p_user_bob"]; item.Preview == nil || item.Preview.Title != "Bob" || item.Preview.Subtitle == nil || *item.Preview.Subtitle != "用户" || item.Preview.CoverURL == nil || *item.Preview.CoverURL != "https://example.com/bob.png" {
		t.Fatalf("user preview = %#v", item.Preview)
	}
	if item := byID["share_p_user_carol"]; item.Preview == nil || item.Preview.Title != "Carol" {
		t.Fatalf("user preview (carol) = %#v", item.Preview)
	}
	for _, shareID := range []string{"share_p_user_missing", "share_p_user_deleted", "share_p_group_missing", "share_p_moment_missing"} {
		if item := byID[shareID]; item.Preview != nil {
			t.Fatalf("%s 的不可见目标 preview 必须为 nil，实际 %#v", shareID, item.Preview)
		}
	}
	if item := byID["share_p_group"]; item.Preview == nil || item.Preview.Title != "Group One" || item.Preview.Subtitle == nil || *item.Preview.Subtitle != "A nice group" || item.Preview.CoverURL == nil || *item.Preview.CoverURL != "https://example.com/group.png" {
		t.Fatalf("group preview = %#v", item.Preview)
	}
	if item := byID["share_p_moment"]; item.Preview == nil || item.Preview.Title != "hello world" || item.Preview.Subtitle == nil || *item.Preview.Subtitle != "Bob" {
		t.Fatalf("moment preview = %#v", item.Preview)
	}
	if item := byID["share_p_moment_fallback"]; item.Preview == nil || item.Preview.Title != "Shanghai" {
		t.Fatalf("moment 空描述 preview 应回退 location_name: %#v", item.Preview)
	}
	if item := byID["share_p_location"]; item.Preview == nil || item.Preview.Title != "Beach" || item.Preview.Subtitle == nil || *item.Preview.Subtitle != "位置" {
		t.Fatalf("location preview = %#v", item.Preview)
	}

	// 批量口径：三类目标各一次查询（10 条分享中 4 user + 2 group + 3 moment），
	// 且列表路径不再调用单条目标查询（否则 N+1 回归）。
	for query, want := range map[string]int{
		queryShareUsersByIDs:   1,
		queryShareGroupsByIDs:  1,
		queryShareMomentsByIDs: 1,
	} {
		if got := backend.queryCounts[query]; got != want {
			t.Errorf("批量查询 %s 调用 %d 次，期望 %d（N+1 回归）", query, got, want)
		}
	}
	for _, query := range []string{queryShareUserExists, queryShareGroupExists, queryShareMomentExists} {
		if got := backend.queryRowCounts[query]; got != 0 {
			t.Errorf("列表路径不应再调用单条目标查询 %s（%d 次）", query, got)
		}
	}
}

// TestPostgresRepositoryListLocationOnlyPreviewSkipsTargetQueries：纯 location
// 分享的列表不需要任何目标表查询（批量解析按需查询）。
func TestPostgresRepositoryListLocationOnlyPreviewSkipsTargetQueries(t *testing.T) {
	backend := newShareBackendStub()
	addListShare(backend, "share_loc_1", ShareTypeLocation, map[string]any{"shared_location": map[string]any{"location_name": "One"}}, "2026-01-02T00:00:00Z")
	addListShare(backend, "share_loc_2", ShareTypeLocation, map[string]any{"shared_location": map[string]any{"location_name": "Two"}}, "2026-01-01T00:00:00Z")
	page, err := newShareTestRepo(backend).List(context.Background(), listSharesQuery{CreatorUserID: "user_alice", Limit: 20})
	if err != nil || len(page.Items) != 2 {
		t.Fatalf("list = %#v err=%v", page, err)
	}
	for _, item := range page.Items {
		if item.Preview == nil || item.Preview.Subtitle == nil || *item.Preview.Subtitle != "位置" {
			t.Fatalf("location preview = %#v", item.Preview)
		}
	}
	for _, query := range []string{queryShareUsersByIDs, queryShareGroupsByIDs, queryShareMomentsByIDs, queryShareUserExists, queryShareGroupExists, queryShareMomentExists} {
		if got := backend.queryCounts[query] + backend.queryRowCounts[query]; got != 0 {
			t.Errorf("纯 location 列表不应查询目标表 %s（%d 次）", query, got)
		}
	}
}

// TestPostgresRepositoryListRejectsUnknownPreviewType：未知分享类型与单条路径
// 一致整体拒绝（不能静默降级为空 preview）。
func TestPostgresRepositoryListRejectsUnknownPreviewType(t *testing.T) {
	backend := newShareBackendStub()
	addListShare(backend, "share_bogus", "bogus", map[string]any{"x": 1}, "2026-01-01T00:00:00Z")
	if _, err := newShareTestRepo(backend).List(context.Background(), listSharesQuery{CreatorUserID: "user_alice", Limit: 20}); !errors.Is(err, ErrInvalidShareRequest) {
		t.Fatalf("未知分享类型应返回 ErrInvalidShareRequest，got %v", err)
	}
}

// TestPostgresRepositoryListBatchPreviewQueryErrors：三类批量查询的底层错误
// 必须原样向上返回（不可静默吞成空 preview）。
func TestPostgresRepositoryListBatchPreviewQueryErrors(t *testing.T) {
	cases := []struct {
		name      string
		query     string
		shareType string
		payload   map[string]any
	}{
		{"user", queryShareUsersByIDs, ShareTypeUser, map[string]any{"shared_user_id": "user_bob"}},
		{"group", queryShareGroupsByIDs, ShareTypeGroup, map[string]any{"shared_group_id": "group_1"}},
		{"moment", queryShareMomentsByIDs, ShareTypeMoment, map[string]any{"shared_moment_id": "moment_1"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			backend := newShareBackendStub()
			addListShare(backend, "share_batch_err", tc.shareType, tc.payload, "2026-01-01T00:00:00Z")
			batchErr := errors.New("batch query boom")
			backend.queryErrs[tc.query] = batchErr
			if _, err := newShareTestRepo(backend).List(context.Background(), listSharesQuery{CreatorUserID: "user_alice", Limit: 20}); !errors.Is(err, batchErr) {
				t.Fatalf("批量查询错误必须透传，got %v", err)
			}
		})
	}
}

// TestPostgresRepositoryListBatchPreviewScanAndTailErrors：批量查询的行扫描
// 错误与 rows.Err 尾部错误都必须透传。
func TestPostgresRepositoryListBatchPreviewScanAndTailErrors(t *testing.T) {
	cases := []struct {
		name      string
		query     string
		shareType string
		payload   map[string]any
	}{
		{"user", queryShareUsersByIDs, ShareTypeUser, map[string]any{"shared_user_id": "user_bob"}},
		{"group", queryShareGroupsByIDs, ShareTypeGroup, map[string]any{"shared_group_id": "group_1"}},
		{"moment", queryShareMomentsByIDs, ShareTypeMoment, map[string]any{"shared_moment_id": "moment_1"}},
	}
	for _, tc := range cases {
		t.Run(tc.name+"/scan", func(t *testing.T) {
			stub := newShareBackendStub()
			addListShare(stub, "share_batch_scan", tc.shareType, tc.payload, "2026-01-01T00:00:00Z")
			scanErr := errors.New("scan boom")
			backend := &shareBatchRowsBackend{shareBackendStub: stub, query: tc.query, rows: shareScanErrRows{err: scanErr}}
			if _, err := newShareTestRepo(backend).List(context.Background(), listSharesQuery{CreatorUserID: "user_alice", Limit: 20}); !errors.Is(err, scanErr) {
				t.Fatalf("扫描错误必须透传，got %v", err)
			}
		})
		t.Run(tc.name+"/tail", func(t *testing.T) {
			stub := newShareBackendStub()
			addListShare(stub, "share_batch_tail", tc.shareType, tc.payload, "2026-01-01T00:00:00Z")
			tailErr := errors.New("tail boom")
			backend := &shareBatchRowsBackend{shareBackendStub: stub, query: tc.query, rows: &testdb.RowsValues{TailErr: tailErr}}
			if _, err := newShareTestRepo(backend).List(context.Background(), listSharesQuery{CreatorUserID: "user_alice", Limit: 20}); !errors.Is(err, tailErr) {
				t.Fatalf("尾部错误必须透传，got %v", err)
			}
		})
	}
}

// TestPostgresRepositoryListPreviewBatchKeepsRowOrder 断言批量解析后 preview
// 仍与分享行按位对应（游标分页依赖行序，错位会把 A 的 preview 挂到 B 上）。
func TestPostgresRepositoryListPreviewBatchKeepsRowOrder(t *testing.T) {
	backend := newShareBackendStub()
	addListShare(backend, "share_order_user", ShareTypeUser, map[string]any{"shared_user_id": "user_bob"}, "2026-01-03T00:00:00Z")
	addListShare(backend, "share_order_location", ShareTypeLocation, map[string]any{"shared_location": map[string]any{"location_name": "Order Place"}}, "2026-01-02T00:00:00Z")
	addListShare(backend, "share_order_group", ShareTypeGroup, map[string]any{"shared_group_id": "group_1"}, "2026-01-01T00:00:00Z")
	page, err := newShareTestRepo(backend).List(context.Background(), listSharesQuery{CreatorUserID: "user_alice", Limit: 20})
	if err != nil || len(page.Items) != 3 {
		t.Fatalf("list = %#v err=%v", page, err)
	}
	want := []struct{ shareID, title string }{
		{"share_order_user", "Bob"},
		{"share_order_location", "Order Place"},
		{"share_order_group", "Group One"},
	}
	for i, expected := range want {
		item := page.Items[i]
		if item.ShareID != expected.shareID || item.Preview == nil || item.Preview.Title != expected.title {
			t.Fatalf("item[%d] = %#v, preview = %#v，期望 %s/%s", i, item, item.Preview, expected.shareID, expected.title)
		}
	}
}
