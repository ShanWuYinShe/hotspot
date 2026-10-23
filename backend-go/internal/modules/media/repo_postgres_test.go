package media

import (
	"context"
	"errors"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/persistence"

	"github.com/jackc/pgx/v5"
)

type mediaRowValues struct {
	values []any
	err    error
}

func (r mediaRowValues) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	for i := range dest {
		switch d := dest[i].(type) {
		case *string:
			*d = r.values[i].(string)
		case *int:
			*d = r.values[i].(int)
		case *MediaCategory:
			*d = r.values[i].(MediaCategory)
		case *time.Time:
			*d = r.values[i].(time.Time)
		case **time.Time:
			if r.values[i] == nil {
				*d = nil
			} else {
				v := r.values[i].(time.Time)
				*d = &v
			}
		default:
			return errors.New("unsupported scan dest")
		}
	}
	return nil
}

type mediaBackendStub struct {
	execErrs   map[string]error
	queryRows  map[string]mediaRowValues
	execCalls  []string
	queryCalls []string
}

func (b *mediaBackendStub) Ping(context.Context) error { return nil }
func (b *mediaBackendStub) Close() error               { return nil }
func (b *mediaBackendStub) Exec(_ context.Context, query string, _ ...any) error {
	b.execCalls = append(b.execCalls, query)
	return b.execErrs[query]
}
func (b *mediaBackendStub) QueryRow(_ context.Context, query string, _ ...any) persistence.Row {
	b.queryCalls = append(b.queryCalls, query)
	if row, ok := b.queryRows[query]; ok {
		return row
	}
	return mediaRowValues{err: persistence.ErrNotImplemented}
}
func (b *mediaBackendStub) Query(context.Context, string, ...any) (persistence.Rows, error) {
	return &persistence.InMemoryRows{}, nil
}

func newMediaFactory(backend persistence.PostgresQueryBackend) *persistence.Factory {
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	return factory
}

func TestPostgresRepositoryReturnsNotImplementedWithoutDatabase(t *testing.T) {
	repo := NewPostgresRepository(nil)
	if err := repo.Save(context.Background(), pendingUpload{UploadID: "upload_1"}); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("expected ErrNotImplemented, got %v", err)
	}
}

func TestPostgresRepositoryReturnsNotImplementedWithoutQueryable(t *testing.T) {
	factory := &persistence.Factory{Config: persistence.Config{PostgresDSN: "postgres://example"}, DB: persistence.NewInMemoryDB()}
	repo := NewPostgresRepository(factory)
	if err := repo.Save(context.Background(), pendingUpload{UploadID: "upload_1"}); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("expected ErrNotImplemented, got %v", err)
	}
}

func TestPostgresRepositoryGetAndCompleteUpload(t *testing.T) {
	createdAt := time.Date(2026, 3, 30, 0, 0, 0, 0, time.UTC)
	completedAt := time.Date(2026, 3, 30, 1, 0, 0, 0, time.UTC)
	backend := &mediaBackendStub{queryRows: map[string]mediaRowValues{
		queryGetUploadSession: {values: []any{"upload_1", "u1", "uploads/u1/upload_1.png", "a.png", 100, "image/png", CategoryAvatar, "", "completed", "file_1", 100, "image/png", "", 0, 0, 0, "key_1", createdAt, completedAt}},
		// Complete 改为条件 UPDATE ... RETURNING upload_id 单往返（审查 P2-19）。
		queryCompleteUploadSession: {values: []any{"upload_1"}},
	}}
	repo := NewPostgresRepository(newMediaFactory(backend))

	if err := repo.Save(context.Background(), pendingUpload{UploadID: "upload_1", UserID: "u1", StorageKey: "uploads/u1/upload_1.png", FileName: "a.png", DeclaredFileSize: 100, DeclaredMimeType: "image/png", Category: CategoryAvatar, CreatedAt: createdAt.Format(time.RFC3339)}); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}
	upload, ok, err := repo.Get(context.Background(), "upload_1")
	if err != nil || !ok {
		t.Fatalf("Get returned ok=%v err=%v", ok, err)
	}
	if !upload.Completed || upload.File.FileID != "file_1" || upload.CompletedIdempotencyKey != "key_1" {
		t.Fatalf("unexpected upload: %#v", upload)
	}
	ok, err = repo.Complete(context.Background(), "upload_1", UploadedMedia{FileID: "file_1", FileSize: 100, MimeType: "image/png"}, "key_1")
	if err != nil || !ok {
		t.Fatalf("Complete returned ok=%v err=%v", ok, err)
	}
	if len(backend.queryCalls) == 0 || backend.queryCalls[len(backend.queryCalls)-1] != queryCompleteUploadSession {
		t.Fatalf("Complete must use the conditional RETURNING update, calls=%v", backend.queryCalls)
	}
}

func TestPostgresRepositoryGetReturnsFalseOnNoRows(t *testing.T) {
	backend := &mediaBackendStub{queryRows: map[string]mediaRowValues{queryGetUploadSession: {err: pgx.ErrNoRows}}}
	repo := NewPostgresRepository(newMediaFactory(backend))
	_, ok, err := repo.Get(context.Background(), "missing")
	if err != nil || ok {
		t.Fatalf("expected ok=false nil error, got ok=%v err=%v", ok, err)
	}
}

func TestPostgresRepositoryPropagatesExecErrors(t *testing.T) {
	backend := &mediaBackendStub{
		execErrs:   map[string]error{queryInsertUploadSession: errors.New("insert boom")},
		queryRows:  map[string]mediaRowValues{queryCompleteUploadSession: {err: errors.New("complete boom")}},
		queryCalls: nil,
	}
	repo := NewPostgresRepository(newMediaFactory(backend))
	if err := repo.Save(context.Background(), pendingUpload{UploadID: "upload_1"}); err == nil {
		t.Fatal("expected insert error")
	}
	if _, err := repo.Complete(context.Background(), "upload_1", UploadedMedia{FileID: "file_1", FileSize: 100, MimeType: "image/png"}, "key_1"); err == nil {
		t.Fatal("expected complete error")
	}
}
