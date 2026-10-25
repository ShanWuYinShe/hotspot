package system

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/testdb"
)

// versionCacheBackend 统计 queryAnnouncementsVersion 的实际执行次数，用于验证读缓存。
type versionCacheBackend struct {
	mu      sync.Mutex
	queries int
	version string
	err     error
}

func (b *versionCacheBackend) Exec(context.Context, string, ...any) error { return nil }

func (b *versionCacheBackend) QueryRow(context.Context, string, ...any) persistence.Row {
	b.mu.Lock()
	b.queries++
	b.mu.Unlock()
	if b.err != nil {
		return testdb.RowValues{Err: b.err}
	}
	return testdb.RowValues{Values: []any{b.version}}
}

func (b *versionCacheBackend) Query(context.Context, string, ...any) (persistence.Rows, error) {
	return &persistence.InMemoryRows{}, nil
}

func (b *versionCacheBackend) count() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.queries
}

func newVersionCacheTestRepo(t *testing.T, backend *versionCacheBackend) *PostgresRepository {
	t.Helper()
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	db, ok := factory.DB.(*persistence.PostgresDB)
	if !ok {
		t.Fatalf("factory DB type = %T, want *persistence.PostgresDB", factory.DB)
	}
	db.Backend = backend
	return NewPostgresRepository(factory)
}

func TestAnnouncementsVersionCacheHitAndInvalidate(t *testing.T) {
	backend := &versionCacheBackend{version: "2026-10-25T00:00:00Z"}
	repo := newVersionCacheTestRepo(t, backend)
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		got, err := repo.AnnouncementsVersion(ctx)
		if err != nil || got != backend.version {
			t.Fatalf("AnnouncementsVersion #%d = (%q, %v), want %q", i, got, err, backend.version)
		}
	}
	if backend.count() != 1 {
		t.Fatalf("announcements version queries = %d, want 1 (cached)", backend.count())
	}
	repo.InvalidateAnnouncementsVersion()
	if _, err := repo.AnnouncementsVersion(ctx); err != nil {
		t.Fatal(err)
	}
	if backend.count() != 2 {
		t.Fatalf("announcements version queries = %d, want 2 after invalidate", backend.count())
	}
}

func TestAnnouncementsVersionCacheTTLExpiry(t *testing.T) {
	backend := &versionCacheBackend{version: "v1"}
	repo := newVersionCacheTestRepo(t, backend)
	ctx := context.Background()
	if _, err := repo.AnnouncementsVersion(ctx); err != nil {
		t.Fatal(err)
	}
	repo.annVersionMu.Lock()
	repo.annVersionCachedAt = time.Now().Add(-2 * annVersionCacheTTL)
	repo.annVersionMu.Unlock()
	if _, err := repo.AnnouncementsVersion(ctx); err != nil {
		t.Fatal(err)
	}
	if backend.count() != 2 {
		t.Fatalf("announcements version queries = %d, want 2 after TTL expiry", backend.count())
	}
}

func TestAnnouncementsVersionCacheDoesNotCacheError(t *testing.T) {
	backend := &versionCacheBackend{err: errors.New("db down")}
	repo := newVersionCacheTestRepo(t, backend)
	ctx := context.Background()
	for i := 0; i < 2; i++ {
		if _, err := repo.AnnouncementsVersion(ctx); err == nil {
			t.Fatalf("call #%d: expected error passthrough", i)
		}
	}
	if backend.count() != 2 {
		t.Fatalf("announcements version queries = %d, want 2 (errors not cached)", backend.count())
	}
}

func TestAnnouncementsVersionWriteInvalidatesCache(t *testing.T) {
	backend := &versionCacheBackend{version: "v1"}
	repo := newVersionCacheTestRepo(t, backend)
	ctx := context.Background()
	if _, err := repo.AnnouncementsVersion(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.CreateAnnouncement(ctx, AnnouncementCreateInput{
		Actor: AnnouncementActor{UserID: "admin"}, Title: "t", Content: "c", Severity: AnnounceLevelInfo,
	}); err != nil {
		t.Fatalf("CreateAnnouncement: %v", err)
	}
	if _, err := repo.AnnouncementsVersion(ctx); err != nil {
		t.Fatal(err)
	}
	if backend.count() != 2 {
		t.Fatalf("announcements version queries = %d, want 2 after write invalidation", backend.count())
	}
}
