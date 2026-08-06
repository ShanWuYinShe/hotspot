package share

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/persistence"

	"github.com/jackc/pgx/v5"
)

type shareRowValues struct {
	values []any
	err    error
}

func (r shareRowValues) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	for i := range dest {
		switch d := dest[i].(type) {
		case *string:
			*d = r.values[i].(string)
		case *bool:
			*d = r.values[i].(bool)
		}
	}
	return nil
}

type shareRowsValues struct {
	rows [][]any
	idx  int
	err  error
}

func (r *shareRowsValues) Next() bool { return r.idx < len(r.rows) }
func (r *shareRowsValues) Scan(dest ...any) error {
	row := shareRowValues{values: r.rows[r.idx]}
	r.idx++
	return row.Scan(dest...)
}
func (r *shareRowsValues) Close() error { return nil }
func (r *shareRowsValues) Err() error   { return r.err }

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

type shareBackendStub struct {
	shares       map[string]shareStubRecord
	byCode       map[string]string
	execErrs     map[string]error
	queryErrs    map[string]error
	queryRowErrs map[string]error
	users        map[string][2]string
	groups       map[string][3]string
	moments      map[string][4]string
}

func newShareBackendStub() *shareBackendStub {
	return &shareBackendStub{
		shares:       make(map[string]shareStubRecord),
		byCode:       make(map[string]string),
		execErrs:     make(map[string]error),
		queryErrs:    make(map[string]error),
		queryRowErrs: make(map[string]error),
		users: map[string][2]string{
			"user_alice": {"Alice", "https://example.com/alice.png"},
			"user_bob":   {"Bob", "https://example.com/bob.png"},
		},
		groups: map[string][3]string{
			"group_1": {"Group One", "A nice group", "https://example.com/group.png"},
		},
		moments: map[string][4]string{
			"moment_1": {"hello world", "Shanghai", "Bob", "https://example.com/bob.png"},
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
		return shareRowValues{err: err}
	}
	switch query {
	case queryGetShare:
		shareID, ok := b.byCode[args[0].(string)]
		if !ok {
			return shareRowValues{err: pgx.ErrNoRows}
		}
		record := b.shares[shareID]
		creator := b.users[record.CreatorUserID]
		return shareRowValues{values: []any{record.ShareID, record.ShareCode, record.ShareType, record.CreatorUserID, creator[0], creator[1], record.PayloadJSON, record.CreatedAt, record.ExpiresAt, record.IsRevoked}}
	case queryGetShareByID:
		record, ok := b.shares[args[0].(string)]
		if !ok {
			return shareRowValues{err: pgx.ErrNoRows}
		}
		return shareRowValues{values: []any{record.ShareID, record.CreatorUserID, record.IsRevoked}}
	case queryShareUserExists:
		user, ok := b.users[args[0].(string)]
		if !ok {
			return shareRowValues{err: pgx.ErrNoRows}
		}
		return shareRowValues{values: []any{args[0].(string), user[0], user[1]}}
	case queryShareGroupExists:
		group, ok := b.groups[args[0].(string)]
		if !ok {
			return shareRowValues{err: pgx.ErrNoRows}
		}
		return shareRowValues{values: []any{args[0].(string), group[0], group[1], group[2]}}
	case queryShareMomentExists:
		moment, ok := b.moments[args[0].(string)]
		if !ok {
			return shareRowValues{err: pgx.ErrNoRows}
		}
		return shareRowValues{values: []any{args[0].(string), moment[0], moment[1], moment[2], moment[3]}}
	default:
		return shareRowValues{err: persistence.ErrNotImplemented}
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
	return &shareRowsValues{rows: rows}, nil
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
	if err != nil || created.ShareID == "" || created.ShareCode == "" || created.ExpiresAt == "" || !validShareCode(created.ShareCode) {
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
