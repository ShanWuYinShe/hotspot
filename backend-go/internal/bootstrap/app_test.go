package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/resourceguard"
)

type ensureSchemaStub struct {
	pingErr error
	execErr error
	queries []string
}

func (s *ensureSchemaStub) Ping(context.Context) error {
	return s.pingErr
}

func (s *ensureSchemaStub) Close() error {
	return nil
}

func (s *ensureSchemaStub) Exec(_ context.Context, query string, _ ...any) error {
	s.queries = append(s.queries, query)
	if s.execErr != nil && query != "SELECT pg_advisory_lock($1)" && query != "SELECT pg_advisory_unlock($1)" {
		return s.execErr
	}
	return nil
}

func (s *ensureSchemaStub) QueryRow(_ context.Context, query string, args ...any) persistence.Row {
	s.queries = append(s.queries, query)
	table := ""
	if len(args) > 0 {
		if value, ok := args[0].(string); ok {
			table = value
		}
	}
	if query == `
SELECT EXISTS (
  SELECT 1
  FROM information_schema.tables
  WHERE table_schema = 'public' AND table_name = $1
)
` {
		return boolRow{value: table == "users"}
	}
	return boolRow{value: true}
}

func (s *ensureSchemaStub) Query(_ context.Context, _ string, _ ...any) (persistence.Rows, error) {
	return &persistence.InMemoryRows{}, nil
}

type boolRow struct {
	value bool
	err   error
}

func (r boolRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	if len(dest) == 1 {
		if target, ok := dest[0].(*bool); ok {
			*target = r.value
			return nil
		}
	}
	return persistence.ErrNotImplemented
}

func TestEnsureSchemaSkipsWithoutPostgresDSN(t *testing.T) {
	if err := ensureSchema(config.Config{}, persistence.NewFactory(config.Config{})); err != nil {
		t.Fatalf("ensureSchema should skip without PostgresDSN: %v", err)
	}
}

func TestEnsureSchemaReturnsMigrationError(t *testing.T) {
	stub := &ensureSchemaStub{execErr: errors.New("boom")}
	factory := &persistence.Factory{
		Config: persistence.Config{PostgresDSN: "postgres://example"},
		DB:     stub,
	}
	cfg := config.Config{PostgresDSN: "postgres://example"}

	err := ensureSchema(cfg, factory)
	if err == nil {
		t.Fatal("expected ensureSchema to fail when migration exec fails")
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Fatalf("expected migration error, got: %v", err)
	}
}

func TestEnsureSchemaReturnsMissingTableError(t *testing.T) {
	stub := &ensureSchemaStub{}
	factory := &persistence.Factory{
		Config: persistence.Config{PostgresDSN: "postgres://example"},
		DB:     stub,
	}
	cfg := config.Config{PostgresDSN: "postgres://example"}

	err := ensureSchema(cfg, factory)
	if err == nil {
		t.Fatal("expected ensureSchema to fail when required tables are missing")
	}
	if !strings.Contains(err.Error(), "missing required tables") {
		t.Fatalf("expected missing table error, got: %v", err)
	}
}

func TestNewResourceGuardCheckerUsesInMemoryQueryableDB(t *testing.T) {
	checker := newResourceGuardChecker(&persistence.Factory{DB: persistence.NewInMemoryDB()})
	if _, ok := checker.(*resourceguard.DBChecker); !ok {
		t.Fatalf("expected *resourceguard.DBChecker, got %T", checker)
	}
}

func TestNewResourceGuardCheckerUsesCustomQueryableDB(t *testing.T) {
	stub := &ensureSchemaStub{}
	checker := newResourceGuardChecker(&persistence.Factory{DB: stub})
	if _, ok := checker.(*resourceguard.DBChecker); !ok {
		t.Fatalf("expected *resourceguard.DBChecker, got %T", checker)
	}
}

func TestAppRouterServesWSV1(t *testing.T) {
	cfg := config.Config{
		AppEnv:           "test",
		LogLevel:         "info",
		S3Endpoint:       "http://localhost:8333",
		S3Region:         "us-east-1",
		S3Bucket:         "hotspot-media",
		S3AccessKey:      "test-access-key",
		S3SecretKey:      "test-secret-key",
		S3ForcePathStyle: true,
	}
	app, err := NewApp(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Use HTTP request to /ws/v1 - WebSocket upgrade requires specific headers,
	// so we just verify the route exists and does not return 404.
	req := httptest.NewRequest(http.MethodGet, "/ws/v1", nil)
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Sec-WebSocket-Version", "13")
	req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
	rr := httptest.NewRecorder()
	app.Router().ServeHTTP(rr, req)
	// Route should exist - not 404
	if rr.Code == http.StatusNotFound {
		t.Fatal("/ws/v1 route not found - got 404")
	}
}
