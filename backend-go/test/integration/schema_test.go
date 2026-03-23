package integration

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	platformmigrate "hotspot/backend-go/internal/platform/migrate"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestRequiredTablesExist(t *testing.T) {
	requireIntegrationEnv(t)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, getenvOrFail(t, "POSTGRES_DSN"))
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}
	defer pool.Close()

	missing := make([]string, 0)
	for _, table := range platformmigrate.RequiredTables() {
		var present bool
		if err := pool.QueryRow(ctx, `
SELECT EXISTS (
  SELECT 1
  FROM information_schema.tables
  WHERE table_schema = 'public' AND table_name = $1
)
`, table).Scan(&present); err != nil {
			t.Fatalf("check table %s: %v", table, err)
		}
		if !present {
			missing = append(missing, table)
		}
	}
	if len(missing) > 0 {
		t.Fatalf("missing required tables: %s", strings.Join(missing, ", "))
	}
}

func getenvOrFail(t *testing.T, key string) string {
	t.Helper()
	value := os.Getenv(key)
	if value == "" {
		t.Fatalf("%s not set", key)
	}
	return value
}
