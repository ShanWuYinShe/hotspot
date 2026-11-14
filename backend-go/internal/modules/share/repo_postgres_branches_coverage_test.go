package share

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/testdb"
)

func newShareTestRepo(backend persistence.PostgresQueryBackend) *PostgresRepository {
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://share-test"})
	db, ok := factory.DB.(*persistence.PostgresDB)
	if !ok {
		panic("factory.DB is not *persistence.PostgresDB")
	}
	db.Backend = backend
	return NewPostgresRepository(factory)
}

// sharePayloadJSONOf 把 map 编成仓储桩需要的 payload JSON，避免测试里手写转义。
func sharePayloadJSONOf(t *testing.T, value map[string]any) string {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	return string(data)
}

// shareSQLStateError 模拟 PostgreSQL 唯一约束冲突（SQLSTATE 23505）。
type shareSQLStateError struct{ code string }

func (e shareSQLStateError) Error() string    { return "sqlstate " + e.code }
func (e shareSQLStateError) SQLState() string { return e.code }

// shareCollidingBackend 让 Insert 前 N 次返回唯一约束冲突，用于覆盖分享码碰撞
// 重试与重试耗尽分支。
type shareCollidingBackend struct {
	*shareBackendStub
	collisions int
}

func (b *shareCollidingBackend) Exec(ctx context.Context, query string, args ...any) error {
	if query == queryInsertShare && b.collisions > 0 {
		b.collisions--
		return shareSQLStateError{code: "23505"}
	}
	return b.shareBackendStub.Exec(ctx, query, args...)
}

// shareListRowsBackend 覆写列表查询返回可控行集/错误。
type shareListRowsBackend struct {
	*shareBackendStub
	rows persistence.Rows
	err  error
}

func (b *shareListRowsBackend) Query(ctx context.Context, query string, args ...any) (persistence.Rows, error) {
	if query == queryListShares {
		return b.rows, b.err
	}
	return b.shareBackendStub.Query(ctx, query, args...)
}

// shareScanErrRows 行扫描必失败的行集。
type shareScanErrRows struct{ err error }

func (r shareScanErrRows) Next() bool        { return true }
func (r shareScanErrRows) Scan(...any) error { return r.err }
func (r shareScanErrRows) Close() error      { return nil }
func (r shareScanErrRows) Err() error        { return nil }

// TestSharePostgresRepoWithoutQueryable 覆盖无可用 Queryable（DB 未配置）时
// 五个仓储入口统一返回 ErrNotImplemented。
func TestSharePostgresRepoWithoutQueryable(t *testing.T) {
	repo := NewPostgresRepository(persistence.NewFactory(config.Config{}))
	ctx := context.Background()

	if _, err := repo.Create(ctx, createShareCommand{CreatorUserID: "u1", ShareType: ShareTypeLocation, Payload: validShareLocationPayload()}); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("create err = %v, want ErrNotImplemented", err)
	}
	if _, err := repo.Get(ctx, getShareQuery{ShareCode: "ABCDEFGHJKLM"}); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("get err = %v, want ErrNotImplemented", err)
	}
	if _, err := repo.List(ctx, listSharesQuery{CreatorUserID: "u1", Limit: 20}); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("list err = %v, want ErrNotImplemented", err)
	}
	if err := repo.Revoke(ctx, revokeShareCommand{CreatorUserID: "u1", ShareID: "share_1"}); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("revoke err = %v, want ErrNotImplemented", err)
	}
	if _, _, err := repo.ResolveShareIDByCode(ctx, "ABCDEFGHJKLM"); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("resolve err = %v, want ErrNotImplemented", err)
	}
}

// TestSharePostgresCreatePayloadMarshalFailure 覆盖 payload 序列化失败分支。
func TestSharePostgresCreatePayloadMarshalFailure(t *testing.T) {
	repo := newShareTestRepo(newShareBackendStub())
	_, err := repo.Create(context.Background(), createShareCommand{
		CreatorUserID: "user_alice",
		ShareType:     ShareTypeLocation,
		Payload:       map[string]any{"shared_location": make(chan int)},
	})
	if err == nil {
		t.Fatal("expected json marshal error")
	}
}

// TestSharePostgresCreateRetriesOnCodeCollision 覆盖分享码唯一冲突后的重试成功。
func TestSharePostgresCreateRetriesOnCodeCollision(t *testing.T) {
	backend := &shareCollidingBackend{shareBackendStub: newShareBackendStub(), collisions: 1}
	repo := newShareTestRepo(backend)

	created, err := repo.Create(context.Background(), createShareCommand{
		CreatorUserID: "user_alice",
		ShareType:     ShareTypeUser,
		Payload:       map[string]any{"shared_user_id": "user_bob"},
	})
	if err != nil || created.ShareID == "" || !validShareCode(created.ShareCode) {
		t.Fatalf("create after collision: %#v err=%v", created, err)
	}
	if backend.collisions != 0 {
		t.Fatalf("expected one retry, remaining collisions = %d", backend.collisions)
	}
}

// TestSharePostgresCreateExhaustsCodeRetries 覆盖分享码连续冲突 5 次后的耗尽错误。
func TestSharePostgresCreateExhaustsCodeRetries(t *testing.T) {
	backend := &shareCollidingBackend{shareBackendStub: newShareBackendStub(), collisions: 10}
	repo := newShareTestRepo(backend)

	_, err := repo.Create(context.Background(), createShareCommand{
		CreatorUserID: "user_alice",
		ShareType:     ShareTypeUser,
		Payload:       map[string]any{"shared_user_id": "user_bob"},
	})
	if err == nil || !strings.Contains(err.Error(), "exhausted") {
		t.Fatalf("expected code generation exhausted error, got %v", err)
	}
}

// TestSharePostgresCreateTargetLookupErrors 覆盖三类目标资源查询的底层错误透传
// （非 ErrShareNotFound 的数据库错误必须原样返回）。
func TestSharePostgresCreateTargetLookupErrors(t *testing.T) {
	cases := []struct {
		name      string
		shareType string
		query     string
		payload   map[string]any
	}{
		{"user", ShareTypeUser, queryShareUserExists, map[string]any{"shared_user_id": "user_bob"}},
		{"group", ShareTypeGroup, queryShareGroupExists, map[string]any{"shared_group_id": "group_1"}},
		{"moment", ShareTypeMoment, queryShareMomentExists, map[string]any{"shared_moment_id": "moment_1"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			backend := newShareBackendStub()
			backend.queryRowErrs[tc.query] = errors.New("lookup boom")
			repo := newShareTestRepo(backend)
			_, err := repo.Create(context.Background(), createShareCommand{CreatorUserID: "user_alice", ShareType: tc.shareType, Payload: tc.payload})
			if err == nil || errors.Is(err, ErrShareNotFound) {
				t.Fatalf("lookup error must propagate, got %v", err)
			}
		})
	}
}

// TestSharePostgresRevokeErrorPaths 覆盖撤销路径的所有权查询失败与更新失败。
func TestSharePostgresRevokeErrorPaths(t *testing.T) {
	ctx := context.Background()

	backend := newShareBackendStub()
	backend.queryRowErrs[queryGetShareByID] = errors.New("ownership boom")
	if err := newShareTestRepo(backend).Revoke(ctx, revokeShareCommand{CreatorUserID: "user_alice", ShareID: "share_1"}); err == nil {
		t.Fatal("expected ownership lookup error")
	}

	backend2 := newShareBackendStub()
	backend2.shares["share_1"] = shareStubRecord{ShareID: "share_1", ShareCode: "ABCDEFGHJKLM", ShareType: ShareTypeUser, CreatorUserID: "user_alice", CreatedAt: "2026-01-01T00:00:00Z"}
	backend2.execErrs[queryRevokeShare] = errors.New("revoke boom")
	if err := newShareTestRepo(backend2).Revoke(ctx, revokeShareCommand{CreatorUserID: "user_alice", ShareID: "share_1"}); err == nil {
		t.Fatal("expected revoke update error")
	}
}

// TestSharePostgresResolveCodeError 覆盖按分享码解析的扫描错误透传。
func TestSharePostgresResolveCodeError(t *testing.T) {
	backend := newShareBackendStub()
	backend.queryRowErrs[queryResolveShareIDByCode] = errors.New("resolve boom")
	if _, _, err := newShareTestRepo(backend).ResolveShareIDByCode(context.Background(), "ABCDEFGHJKLM"); err == nil {
		t.Fatal("expected resolve error")
	}
}

// TestSharePostgresGetErrorPaths 覆盖读取路径：查询失败/payload 非法 JSON/
// preview 解析失败/expires_at 非法格式（isExpired 解析失败按未过期处理）。
func TestSharePostgresGetErrorPaths(t *testing.T) {
	ctx := context.Background()

	backend := newShareBackendStub()
	backend.queryRowErrs[queryGetShare] = errors.New("get boom")
	if _, err := newShareTestRepo(backend).Get(ctx, getShareQuery{ShareCode: "ABCDEFGHJKLM"}); err == nil {
		t.Fatal("expected get query error")
	}

	backend2 := newShareBackendStub()
	backend2.byCode["ABCDEFGHJKLM"] = "share_bad_payload"
	backend2.shares["share_bad_payload"] = shareStubRecord{ShareID: "share_bad_payload", ShareCode: "ABCDEFGHJKLM", ShareType: ShareTypeLocation, CreatorUserID: "user_alice", PayloadJSON: "not-json"}
	if _, err := newShareTestRepo(backend2).Get(ctx, getShareQuery{ShareCode: "ABCDEFGHJKLM"}); err == nil {
		t.Fatal("expected payload unmarshal error")
	}

	backend3 := newShareBackendStub()
	backend3.byCode["ABCDEFGHJKLM"] = "share_preview_err"
	backend3.shares["share_preview_err"] = shareStubRecord{ShareID: "share_preview_err", ShareCode: "ABCDEFGHJKLM", ShareType: ShareTypeUser, CreatorUserID: "user_alice", PayloadJSON: sharePayloadJSONOf(t, map[string]any{"shared_user_id": "user_bob"})}
	backend3.queryRowErrs[queryShareUserExists] = errors.New("preview boom")
	if _, err := newShareTestRepo(backend3).Get(ctx, getShareQuery{ShareCode: "ABCDEFGHJKLM"}); err == nil {
		t.Fatal("expected preview resolve error")
	}

	backend4 := newShareBackendStub()
	backend4.byCode["ABCDEFGHJKLM"] = "share_bad_expiry"
	backend4.shares["share_bad_expiry"] = shareStubRecord{ShareID: "share_bad_expiry", ShareCode: "ABCDEFGHJKLM", ShareType: ShareTypeLocation, CreatorUserID: "user_alice", PayloadJSON: sharePayloadJSONOf(t, map[string]any{"shared_location": map[string]any{"location_name": "x"}}), ExpiresAt: "not-a-timestamp"}
	detail, err := newShareTestRepo(backend4).Get(ctx, getShareQuery{ShareCode: "ABCDEFGHJKLM"})
	if err != nil || detail.Share.ShareID != "share_bad_expiry" {
		t.Fatalf("invalid expires_at must be treated as not expired: %#v err=%v", detail, err)
	}
}

// TestSharePostgresResolvePreviewTypes 覆盖 group/moment preview、moment 描述为空
// 回退位置、未知类型拒绝（user/location 由既有用例覆盖）。
func TestSharePostgresResolvePreviewTypes(t *testing.T) {
	ctx := context.Background()
	backend := newShareBackendStub()

	backend.shares["share_group_preview"] = shareStubRecord{ShareID: "share_group_preview", ShareCode: "ABCDEFGHJKLM", ShareType: ShareTypeGroup, CreatorUserID: "user_alice", PayloadJSON: sharePayloadJSONOf(t, map[string]any{"shared_group_id": "group_1"})}
	backend.byCode["ABCDEFGHJKLM"] = "share_group_preview"
	detail, err := newShareTestRepo(backend).Get(ctx, getShareQuery{ShareCode: "ABCDEFGHJKLM"})
	if err != nil {
		t.Fatalf("get group share: %v", err)
	}
	if detail.Preview == nil || detail.Preview.Title != "Group One" || detail.Preview.Subtitle == nil || *detail.Preview.Subtitle != "A nice group" || detail.Preview.CoverURL == nil || *detail.Preview.CoverURL != "https://example.com/group.png" {
		t.Fatalf("group preview = %#v", detail.Preview)
	}

	backend.shares["share_moment_preview"] = shareStubRecord{ShareID: "share_moment_preview", ShareCode: "ABCDEFGHJKMN", ShareType: ShareTypeMoment, CreatorUserID: "user_alice", PayloadJSON: sharePayloadJSONOf(t, map[string]any{"shared_moment_id": "moment_1"})}
	backend.byCode["ABCDEFGHJKMN"] = "share_moment_preview"
	detail, err = newShareTestRepo(backend).Get(ctx, getShareQuery{ShareCode: "ABCDEFGHJKMN"})
	if err != nil {
		t.Fatalf("get moment share: %v", err)
	}
	if detail.Preview == nil || detail.Preview.Title != "hello world" || detail.Preview.Subtitle == nil || *detail.Preview.Subtitle != "Bob" || detail.Preview.CoverURL == nil || *detail.Preview.CoverURL != "https://example.com/bob.png" {
		t.Fatalf("moment preview = %#v", detail.Preview)
	}

	// moment 描述为空 → 标题回退 location_name
	backend.moments["moment_no_desc"] = [5]string{"", "Shanghai", "Bob", "", "public"}
	backend.shares["share_moment_fallback"] = shareStubRecord{ShareID: "share_moment_fallback", ShareCode: "ABCDEFGHJKMP", ShareType: ShareTypeMoment, CreatorUserID: "user_alice", PayloadJSON: sharePayloadJSONOf(t, map[string]any{"shared_moment_id": "moment_no_desc"})}
	backend.byCode["ABCDEFGHJKMP"] = "share_moment_fallback"
	detail, err = newShareTestRepo(backend).Get(ctx, getShareQuery{ShareCode: "ABCDEFGHJKMP"})
	if err != nil {
		t.Fatalf("get moment fallback share: %v", err)
	}
	if detail.Preview == nil || detail.Preview.Title != "Shanghai" {
		t.Fatalf("moment fallback preview = %#v", detail.Preview)
	}

	if _, err := resolveSharePreview(ctx, backend, "bogus", map[string]any{}); !errors.Is(err, ErrInvalidShareRequest) {
		t.Fatalf("unknown share type preview err = %v, want ErrInvalidShareRequest", err)
	}
}

// TestSharePostgresListPreviewAndErrors 覆盖列表读取路径：扫描失败/尾部错误/
// payload 非法 JSON/preview 解析失败/各类型 preview 与过期标记。
func TestSharePostgresListPreviewAndErrors(t *testing.T) {
	ctx := context.Background()

	scanBackend := &shareListRowsBackend{shareBackendStub: newShareBackendStub(), rows: shareScanErrRows{err: errors.New("scan boom")}}
	if _, err := newShareTestRepo(scanBackend).List(ctx, listSharesQuery{CreatorUserID: "user_alice", Limit: 20}); err == nil {
		t.Fatal("expected row scan error")
	}

	tailBackend := &shareListRowsBackend{shareBackendStub: newShareBackendStub(), rows: &testdb.RowsValues{TailErr: errors.New("tail boom")}}
	if _, err := newShareTestRepo(tailBackend).List(ctx, listSharesQuery{CreatorUserID: "user_alice", Limit: 20}); err == nil {
		t.Fatal("expected rows tail error")
	}

	badPayloadBackend := newShareBackendStub()
	badPayloadBackend.shares["share_bad_json"] = shareStubRecord{ShareID: "share_bad_json", ShareCode: "ABCDEFGHJKLM", ShareType: ShareTypeLocation, CreatorUserID: "user_alice", PayloadJSON: "{bad", CreatedAt: "2026-01-01T00:00:00Z"}
	if _, err := newShareTestRepo(badPayloadBackend).List(ctx, listSharesQuery{CreatorUserID: "user_alice", Limit: 20}); err == nil {
		t.Fatal("expected list payload unmarshal error")
	}

	previewErrBackend := newShareBackendStub()
	previewErrBackend.shares["share_preview_fail"] = shareStubRecord{ShareID: "share_preview_fail", ShareCode: "ABCDEFGHJKLM", ShareType: ShareTypeUser, CreatorUserID: "user_alice", PayloadJSON: sharePayloadJSONOf(t, map[string]any{"shared_user_id": "user_bob"}), CreatedAt: "2026-01-01T00:00:00Z"}
	previewErrBackend.queryErrs[queryShareUsersByIDs] = errors.New("preview boom")
	if _, err := newShareTestRepo(previewErrBackend).List(ctx, listSharesQuery{CreatorUserID: "user_alice", Limit: 20}); err == nil {
		t.Fatal("expected list preview error")
	}

	// 正常列表：group/moment/location preview、过期项标记不可用、活动项保留 preview
	backend := newShareBackendStub()
	past := time.Now().UTC().Add(-time.Hour).Format(time.RFC3339)
	backend.shares["share_list_group"] = shareStubRecord{ShareID: "share_list_group", ShareCode: "ABCDEFGHJKLM", ShareType: ShareTypeGroup, CreatorUserID: "user_alice", PayloadJSON: sharePayloadJSONOf(t, map[string]any{"shared_group_id": "group_1"}), CreatedAt: "2026-01-03T00:00:00Z"}
	backend.shares["share_list_moment"] = shareStubRecord{ShareID: "share_list_moment", ShareCode: "ABCDEFGHJKMN", ShareType: ShareTypeMoment, CreatorUserID: "user_alice", PayloadJSON: sharePayloadJSONOf(t, map[string]any{"shared_moment_id": "moment_1"}), CreatedAt: "2026-01-02T00:00:00Z"}
	backend.shares["share_list_location"] = shareStubRecord{ShareID: "share_list_location", ShareCode: "ABCDEFGHJKMP", ShareType: ShareTypeLocation, CreatorUserID: "user_alice", PayloadJSON: sharePayloadJSONOf(t, map[string]any{"shared_location": map[string]any{"location_name": "Beach"}}), CreatedAt: "2026-01-01T00:00:00Z"}
	backend.shares["share_list_expired"] = shareStubRecord{ShareID: "share_list_expired", ShareCode: "ABCDEFGHJKMR", ShareType: ShareTypeLocation, CreatorUserID: "user_alice", PayloadJSON: sharePayloadJSONOf(t, map[string]any{"shared_location": map[string]any{"location_name": "Old"}}), CreatedAt: "2025-12-31T00:00:00Z", ExpiresAt: past}

	page, err := newShareTestRepo(backend).List(ctx, listSharesQuery{CreatorUserID: "user_alice", Limit: 20})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	byID := make(map[string]ShareListItem, len(page.Items))
	for _, item := range page.Items {
		byID[item.ShareID] = item
	}
	if item := byID["share_list_group"]; item.Preview == nil || item.Preview.Title != "Group One" {
		t.Fatalf("group list item = %#v", item)
	}
	if item := byID["share_list_moment"]; item.Preview == nil || item.Preview.Title != "hello world" {
		t.Fatalf("moment list item = %#v", item)
	}
	if item := byID["share_list_location"]; item.Preview == nil || item.Preview.Title != "Beach" || item.Preview.Subtitle == nil || *item.Preview.Subtitle != "位置" {
		t.Fatalf("location list item = %#v", item)
	}
	if item := byID["share_list_expired"]; !item.IsRevoked || item.Preview != nil {
		t.Fatalf("expired list item = %#v", item)
	}
}

// TestSharePostgresListPaginationCursor 覆盖列表多于 limit 时生成 next_cursor。
func TestSharePostgresListPaginationCursor(t *testing.T) {
	ctx := context.Background()
	repo := newShareTestRepo(newShareBackendStub())
	for i := 0; i < 3; i++ {
		if _, err := repo.Create(ctx, createShareCommand{CreatorUserID: "user_alice", ShareType: ShareTypeLocation, Payload: validShareLocationPayload()}); err != nil {
			t.Fatalf("create share %d: %v", i, err)
		}
	}
	page, err := repo.List(ctx, listSharesQuery{CreatorUserID: "user_alice", Limit: 2})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(page.Items) != 2 || page.NextCursor == nil || *page.NextCursor == "" {
		t.Fatalf("expected 2 items with next_cursor, got %#v", page)
	}
}
