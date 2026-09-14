package share

import (
	"context"
	"encoding/json"
	"errors"
	"hotspot/backend-go/internal/platform/testdb"
	"sort"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/persistence"

	"github.com/jackc/pgx/v5"
)

type shareStubRecord struct {
	ShareID       string
	ShareCode     string
	ShareType     string
	CreatorUserID string
	PayloadJSON   string
	CreatedAt     string
	ExpiresAt     string
	IsRevoked     bool
}

type shareStubUser struct {
	Nickname string
	Avatar   string
	Deleted  bool // 模拟 users.is_deleted（已注销用户）
}

type shareBackendStub struct {
	shares       map[string]shareStubRecord
	byCode       map[string]string
	execErrs     map[string]error
	queryErrs    map[string]error
	queryRowErrs map[string]error
	users        map[string]shareStubUser
	groups       map[string][3]string
	// moments 值末位为可见性：public 之外的值（private/deleted）模拟
	// queryShareMomentExists 的 privacy_level='public' AND is_deleted=FALSE 过滤。
	moments map[string][5]string
}

func newShareBackendStub() *shareBackendStub {
	return &shareBackendStub{
		shares:       make(map[string]shareStubRecord),
		byCode:       make(map[string]string),
		execErrs:     make(map[string]error),
		queryErrs:    make(map[string]error),
		queryRowErrs: make(map[string]error),
		users: map[string]shareStubUser{
			"user_alice": {Nickname: "Alice", Avatar: "https://example.com/alice.png"},
			"user_bob":   {Nickname: "Bob", Avatar: "https://example.com/bob.png"},
		},
		groups: map[string][3]string{
			"group_1": {"Group One", "A nice group", "https://example.com/group.png"},
		},
		moments: map[string][5]string{
			"moment_1": {"hello world", "Shanghai", "Bob", "https://example.com/bob.png", "public"},
		},
	}
}

func (b *shareBackendStub) Exec(_ context.Context, query string, args ...any) error {
	if err := b.execErrs[query]; err != nil {
		return err
	}
	switch query {
	case queryInsertShare:
		expiresAt := ""
		if args[6] != nil {
			expiresAt = args[6].(string)
		}
		record := shareStubRecord{
			ShareID:       args[0].(string),
			ShareCode:     args[1].(string),
			CreatorUserID: args[2].(string),
			ShareType:     args[3].(string),
			PayloadJSON:   sharePayloadJSON(args[4]),
			CreatedAt:     args[5].(string),
			ExpiresAt:     expiresAt,
			IsRevoked:     args[7].(bool),
		}
		b.shares[record.ShareID] = record
		b.byCode[record.ShareCode] = record.ShareID
	case queryRevokeShare:
		shareID := args[0].(string)
		userID := args[1].(string)
		record, ok := b.shares[shareID]
		if ok && record.CreatorUserID == userID && !record.IsRevoked {
			record.IsRevoked = true
			b.shares[shareID] = record
		}
	}
	return nil
}

func (b *shareBackendStub) QueryRow(_ context.Context, query string, args ...any) persistence.Row {
	if err := b.queryRowErrs[query]; err != nil {
		return testdb.RowValues{Err: err}
	}
	switch query {
	case queryGetShare:
		shareID, ok := b.byCode[args[0].(string)]
		if !ok {
			return testdb.RowValues{Err: pgx.ErrNoRows}
		}
		record := b.shares[shareID]
		creator := b.users[record.CreatorUserID]
		return testdb.RowValues{Values: []any{record.ShareID, record.ShareCode, record.ShareType, record.CreatorUserID, creator.Nickname, creator.Avatar, record.PayloadJSON, record.CreatedAt, record.ExpiresAt, record.IsRevoked}}
	case queryGetShareByID:
		record, ok := b.shares[args[0].(string)]
		if !ok {
			return testdb.RowValues{Err: pgx.ErrNoRows}
		}
		return testdb.RowValues{Values: []any{record.ShareID, record.CreatorUserID, record.IsRevoked}}
	case queryShareUserExists:
		user, ok := b.users[args[0].(string)]
		// 对齐 SQL 过滤：已注销（is_deleted）用户按不存在处理。
		if !ok || user.Deleted {
			return testdb.RowValues{Err: pgx.ErrNoRows}
		}
		return testdb.RowValues{Values: []any{args[0].(string), user.Nickname, user.Avatar}}
	case queryShareGroupExists:
		group, ok := b.groups[args[0].(string)]
		if !ok {
			return testdb.RowValues{Err: pgx.ErrNoRows}
		}
		return testdb.RowValues{Values: []any{args[0].(string), group[0], group[1], group[2]}}
	case queryShareMomentExists:
		moment, ok := b.moments[args[0].(string)]
		// 对齐 SQL 过滤：privacy_level='public' 且未删除的瞬间才可见。
		if !ok || moment[4] != "public" {
			return testdb.RowValues{Err: pgx.ErrNoRows}
		}
		return testdb.RowValues{Values: []any{args[0].(string), moment[0], moment[1], moment[2], moment[3]}}
	default:
		return testdb.RowValues{Err: persistence.ErrNotImplemented}
	}
}

func (b *shareBackendStub) Query(_ context.Context, query string, args ...any) (persistence.Rows, error) {
	if err := b.queryErrs[query]; err != nil {
		return nil, err
	}
	if query != queryListShares {
		return &persistence.InMemoryRows{}, nil
	}
	userID := args[0].(string)
	shareType, _ := args[1].(string)
	cursorCreatedAt, _ := args[2].(string)
	cursorShareID := args[3].(string)
	records := make([]shareStubRecord, 0)
	for _, record := range b.shares {
		if record.CreatorUserID != userID {
			continue
		}
		if shareType != "" && record.ShareType != shareType {
			continue
		}
		if cursorCreatedAt != "" {
			if !(record.CreatedAt < cursorCreatedAt || (record.CreatedAt == cursorCreatedAt && record.ShareID < cursorShareID)) {
				continue
			}
		}
		records = append(records, record)
	}
	sort.Slice(records, func(i, j int) bool {
		if records[i].CreatedAt == records[j].CreatedAt {
			return records[i].ShareID > records[j].ShareID
		}
		return records[i].CreatedAt > records[j].CreatedAt
	})
	limit := args[4].(int)
	if len(records) > limit {
		records = records[:limit]
	}
	rows := make([][]any, 0, len(records))
	for _, record := range records {
		rows = append(rows, []any{record.ShareID, record.ShareCode, record.ShareType, record.CreatedAt, record.ExpiresAt, record.IsRevoked, record.PayloadJSON})
	}
	return &testdb.RowsValues{Rows: rows}, nil
}

func sharePayloadJSON(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return v
	default:
		data, _ := json.Marshal(v)
		return string(data)
	}
}

func TestPostgresRepositoryCreateGetListAndRevoke(t *testing.T) {
	backend := newShareBackendStub()
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	repo := NewPostgresRepository(factory)
	expiresAt := time.Now().UTC().Add(time.Minute)

	created, err := repo.Create(context.Background(), createShareCommand{CreatorUserID: "user_alice", ShareType: ShareTypeUser, Payload: map[string]any{"shared_user_id": "user_bob"}, ExpiresAt: &expiresAt})
	if err != nil || created.ShareID == "" || created.ShareCode == "" || created.ExpiresAt == nil || !validShareCode(created.ShareCode) {
		t.Fatalf("unexpected created share: %#v err=%v", created, err)
	}

	detail, err := repo.Get(context.Background(), getShareQuery{ShareCode: created.ShareCode})
	if err != nil {
		t.Fatalf("unexpected get error: %v", err)
	}
	if detail.Share.ShareID != created.ShareID || detail.Payload["shared_user_id"] != "user_bob" || detail.Share.Creator.Nickname != "Alice" {
		t.Fatalf("unexpected share detail: %#v", detail)
	}

	list, err := repo.List(context.Background(), listSharesQuery{CreatorUserID: "user_alice", Limit: 20})
	if err != nil || len(list.Items) != 1 || list.Items[0].ShareID != created.ShareID || list.Items[0].IsRevoked {
		t.Fatalf("unexpected list: %#v err=%v", list, err)
	}

	if err := repo.Revoke(context.Background(), revokeShareCommand{CreatorUserID: "user_alice", ShareID: created.ShareID}); err != nil {
		t.Fatalf("unexpected revoke error: %v", err)
	}

	if _, err := repo.Get(context.Background(), getShareQuery{ShareCode: created.ShareCode}); !errors.Is(err, ErrShareNotFound) {
		t.Fatalf("expected revoked share to be unavailable, got %v", err)
	}

	list, err = repo.List(context.Background(), listSharesQuery{CreatorUserID: "user_alice", Limit: 20})
	if err != nil || len(list.Items) != 1 || !list.Items[0].IsRevoked {
		t.Fatalf("unexpected revoked list: %#v err=%v", list, err)
	}
}

func TestPostgresRepositoryGetReturnsNotFoundOnNoRows(t *testing.T) {
	backend := newShareBackendStub()
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	repo := NewPostgresRepository(factory)

	_, err := repo.Get(context.Background(), getShareQuery{ShareCode: "ABCDEFGHJKLM"})
	if !errors.Is(err, ErrShareNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestPostgresRepositoryCreateReturnsErrorOnExecFailure(t *testing.T) {
	backend := newShareBackendStub()
	backend.execErrs[queryInsertShare] = errors.New("write failed")
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	repo := NewPostgresRepository(factory)

	_, err := repo.Create(context.Background(), createShareCommand{CreatorUserID: "user_alice", ShareType: ShareTypeUser, Payload: map[string]any{"shared_user_id": "user_bob"}})
	if err == nil {
		t.Fatal("expected create error")
	}
}

func TestPostgresRepositoryListReturnsErrorOnQueryFailure(t *testing.T) {
	backend := newShareBackendStub()
	backend.queryErrs[queryListShares] = errors.New("query failed")
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	repo := NewPostgresRepository(factory)

	_, err := repo.List(context.Background(), listSharesQuery{CreatorUserID: "user_alice", Limit: 20})
	if err == nil {
		t.Fatal("expected list error")
	}
}

func TestPostgresRepositoryRejectsMissingTarget(t *testing.T) {
	backend := newShareBackendStub()
	delete(backend.users, "user_bob")
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	repo := NewPostgresRepository(factory)

	_, err := repo.Create(context.Background(), createShareCommand{CreatorUserID: "user_alice", ShareType: ShareTypeUser, Payload: map[string]any{"shared_user_id": "user_bob"}})
	if !errors.Is(err, ErrShareNotFound) {
		t.Fatalf("expected ErrShareNotFound, got %v", err)
	}
}

// 目标可见性（spec api_spec_share.md：目标资源必须存在且可访问，不可访问统一
// 按不存在返回 10300 防枚举）：已注销用户、私密动态、已删动态均拒绝创建分享。
func TestPostgresRepositoryRejectsInvisibleTargets(t *testing.T) {
	backend := newShareBackendStub()
	backend.users["user_ghost"] = shareStubUser{Nickname: "Ghost", Deleted: true}
	backend.moments["moment_private"] = [5]string{"secret", "", "Bob", "", "private"}
	backend.moments["moment_deleted"] = [5]string{"gone", "", "Bob", "", "deleted"}
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	repo := NewPostgresRepository(factory)

	cases := []struct {
		name      string
		shareType string
		payload   map[string]any
	}{
		{"deleted user", ShareTypeUser, map[string]any{"shared_user_id": "user_ghost"}},
		{"private moment", ShareTypeMoment, map[string]any{"shared_moment_id": "moment_private"}},
		{"deleted moment", ShareTypeMoment, map[string]any{"shared_moment_id": "moment_deleted"}},
	}
	for _, tc := range cases {
		_, err := repo.Create(context.Background(), createShareCommand{CreatorUserID: "user_alice", ShareType: tc.shareType, Payload: tc.payload})
		if !errors.Is(err, ErrShareNotFound) {
			t.Fatalf("%s: expected ErrShareNotFound, got %v", tc.name, err)
		}
	}
}

func TestPostgresRepositoryGetOmitsPreviewWhenTargetBecomesUnavailable(t *testing.T) {
	backend := newShareBackendStub()
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	repo := NewPostgresRepository(factory)

	created, err := repo.Create(context.Background(), createShareCommand{CreatorUserID: "user_alice", ShareType: ShareTypeUser, Payload: map[string]any{"shared_user_id": "user_bob"}})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	delete(backend.users, "user_bob")

	detail, err := repo.Get(context.Background(), getShareQuery{ShareCode: created.ShareCode})
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if detail.Preview != nil {
		t.Fatalf("expected nil preview, got %#v", detail.Preview)
	}
}

func TestPostgresRepositoryListOmitsPreviewForRevokedOrUnavailableItems(t *testing.T) {
	backend := newShareBackendStub()
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	repo := NewPostgresRepository(factory)

	revoked, err := repo.Create(context.Background(), createShareCommand{CreatorUserID: "user_alice", ShareType: ShareTypeUser, Payload: map[string]any{"shared_user_id": "user_bob"}})
	if err != nil {
		t.Fatalf("create revoked: %v", err)
	}
	if err := repo.Revoke(context.Background(), revokeShareCommand{CreatorUserID: "user_alice", ShareID: revoked.ShareID}); err != nil {
		t.Fatalf("revoke: %v", err)
	}

	active, err := repo.Create(context.Background(), createShareCommand{CreatorUserID: "user_alice", ShareType: ShareTypeUser, Payload: map[string]any{"shared_user_id": "user_bob"}})
	if err != nil {
		t.Fatalf("create active: %v", err)
	}
	_ = active
	delete(backend.users, "user_bob")

	list, err := repo.List(context.Background(), listSharesQuery{CreatorUserID: "user_alice", Limit: 20})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list.Items) != 2 {
		t.Fatalf("expected 2 items, got %#v", list)
	}
	for _, item := range list.Items {
		if item.Preview != nil {
			t.Fatalf("expected nil preview for item %#v", item)
		}
	}
}
