package share

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"testing"

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

type shareFallbackRepo struct{ used bool }

func (r *shareFallbackRepo) Create(context.Context, string, CreateShareRequest) (ShareCreated, error) {
	r.used = true
	return ShareCreated{}, nil
}
func (r *shareFallbackRepo) Get(context.Context, string) (ShareDetail, bool, error) {
	r.used = true
	return ShareDetail{}, false, nil
}
func (r *shareFallbackRepo) List(context.Context, string) (PaginatedShares, error) {
	r.used = true
	return PaginatedShares{}, nil
}
func (r *shareFallbackRepo) Revoke(context.Context, string, string) (bool, error) {
	r.used = true
	return false, nil
}

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
}

func newShareBackendStub() *shareBackendStub {
	return &shareBackendStub{shares: make(map[string]shareStubRecord), byCode: make(map[string]string), execErrs: make(map[string]error), queryErrs: make(map[string]error), queryRowErrs: make(map[string]error)}
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
	if query != queryGetShare {
		return shareRowValues{err: persistence.ErrNotImplemented}
	}
	shareID, ok := b.byCode[args[0].(string)]
	if !ok {
		return shareRowValues{err: pgx.ErrNoRows}
	}
	record := b.shares[shareID]
	return shareRowValues{values: []any{record.ShareID, record.ShareCode, record.ShareType, record.CreatorUserID, record.PayloadJSON, record.CreatedAt, record.ExpiresAt, record.IsRevoked}}
}

func (b *shareBackendStub) Query(_ context.Context, query string, args ...any) (persistence.Rows, error) {
	if err := b.queryErrs[query]; err != nil {
		return nil, err
	}
	if query != queryListShares {
		return &persistence.InMemoryRows{}, nil
	}
	userID := args[0].(string)
	records := make([]shareStubRecord, 0)
	for _, record := range b.shares {
		if record.CreatorUserID == userID {
			records = append(records, record)
		}
	}
	sort.Slice(records, func(i, j int) bool {
		if records[i].CreatedAt == records[j].CreatedAt {
			return records[i].ShareID > records[j].ShareID
		}
		return records[i].CreatedAt > records[j].CreatedAt
	})
	rows := make([][]any, 0, len(records))
	for _, record := range records {
		rows = append(rows, []any{record.ShareID, record.ShareCode, record.ShareType, record.CreatedAt, record.ExpiresAt, record.IsRevoked})
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

func TestNewPersistenceRepositoryUsesPostgresWhenDSNConfigured(t *testing.T) {
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	repo := NewPersistenceRepository(factory)
	if _, ok := repo.impl.(*PostgresRepository); !ok {
		t.Fatalf("expected postgres impl, got %T", repo.impl)
	}
}

func TestPostgresRepositoryCreateGetListAndRevoke(t *testing.T) {
	backend := newShareBackendStub()
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	fallback := &shareFallbackRepo{}
	repo := NewPostgresRepository(factory, fallback)

	created, err := repo.Create(context.Background(), "user_alice", CreateShareRequest{ShareType: "user", Payload: map[string]any{"shared_user_id": "user_bob"}, ExpiresInSeconds: 60})
	if err != nil || created.ShareID == "" || created.ShareCode == "" || created.ExpiresAt == "" {
		t.Fatalf("unexpected created share: %#v err=%v", created, err)
	}

	detail, ok, err := repo.Get(context.Background(), created.ShareCode)
	if err != nil || !ok {
		t.Fatalf("unexpected get result: %#v ok=%v err=%v", detail, ok, err)
	}
	if detail.Share["share_id"] != created.ShareID || detail.Payload["shared_user_id"] != "user_bob" {
		t.Fatalf("unexpected share detail: %#v", detail)
	}

	list, err := repo.List(context.Background(), "user_alice")
	if err != nil || len(list.Items) != 1 || list.Items[0].ShareID != created.ShareID || list.Items[0].IsRevoked {
		t.Fatalf("unexpected list: %#v err=%v", list, err)
	}

	revoked, err := repo.Revoke(context.Background(), "user_alice", created.ShareID)
	if err != nil || !revoked {
		t.Fatalf("unexpected revoke result: revoked=%v err=%v", revoked, err)
	}

	_, ok, err = repo.Get(context.Background(), created.ShareCode)
	if err != nil || ok {
		t.Fatalf("expected revoked share to be unavailable, ok=%v err=%v", ok, err)
	}

	list, err = repo.List(context.Background(), "user_alice")
	if err != nil || len(list.Items) != 1 || !list.Items[0].IsRevoked {
		t.Fatalf("unexpected revoked list: %#v err=%v", list, err)
	}
	if fallback.used {
		t.Fatal("unexpected fallback usage")
	}
}

func TestPostgresRepositoryGetReturnsNotFoundOnNoRows(t *testing.T) {
	backend := newShareBackendStub()
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	fallback := &shareFallbackRepo{}
	repo := NewPostgresRepository(factory, fallback)

	_, ok, err := repo.Get(context.Background(), "MISSING")
	if err != nil || ok {
		t.Fatalf("expected not found, ok=%v err=%v", ok, err)
	}
	if fallback.used {
		t.Fatal("unexpected fallback usage")
	}
}

func TestPostgresRepositoryCreateReturnsErrorOnExecFailure(t *testing.T) {
	backend := newShareBackendStub()
	backend.execErrs[queryInsertShare] = errors.New("write failed")
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	fallback := &shareFallbackRepo{}
	repo := NewPostgresRepository(factory, fallback)

	_, err := repo.Create(context.Background(), "user_alice", CreateShareRequest{ShareType: "user", Payload: map[string]any{"shared_user_id": "user_bob"}})
	if err == nil {
		t.Fatal("expected create error")
	}
	if fallback.used {
		t.Fatal("unexpected fallback usage")
	}
}

func TestPostgresRepositoryListReturnsErrorOnQueryFailure(t *testing.T) {
	backend := newShareBackendStub()
	backend.queryErrs[queryListShares] = errors.New("query failed")
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	fallback := &shareFallbackRepo{}
	repo := NewPostgresRepository(factory, fallback)

	_, err := repo.List(context.Background(), "user_alice")
	if err == nil {
		t.Fatal("expected list error")
	}
	if fallback.used {
		t.Fatal("unexpected fallback usage")
	}
}
