package migrate

import (
	"context"
	"fmt"
	"io/fs"
	"sort"
	"strings"
	"time"

	"hotspot/backend-go/internal/platform/persistence"
)

const advisoryLockKey int64 = 2026032301

var requiredTables = []string{
	"users",
	"user_settings",
	"user_relations",
	"groups",
	"auth_sessions",
	"auth_refresh_tokens",
	"moments",
	"comments",
	"private_chats",
	"private_messages",
	"group_messages",
	"group_message_read_state",
	"notifications",
	"shares",
	"reports",
	"auth_accounts",
	"auth_password_reset_requests",
	"admin_audit_logs",
	"app_configs",
}

func RequiredTables() []string {
	return append([]string(nil), requiredTables...)
}

func listMigrationsFS(fsys fs.FS) ([]string, error) {
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return nil, err
	}
	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.HasSuffix(entry.Name(), ".sql") {
			files = append(files, entry.Name())
		}
	}
	sort.Strings(files)
	return files, nil
}

func ApplyFS(ctx context.Context, db persistence.QueryableDB, fsys fs.FS) error {
	files, err := listMigrationsFS(fsys)
	if err != nil {
		return err
	}
	for _, name := range files {
		blob, err := fs.ReadFile(fsys, name)
		if err != nil {
			return err
		}
		if err := db.Exec(ctx, string(blob)); err != nil {
			return fmt.Errorf("apply migration %s: %w", name, err)
		}
	}
	return nil
}

func withSchemaLock(ctx context.Context, db persistence.QueryableDB, run func() error) error {
	lockCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := db.Exec(lockCtx, "SELECT pg_advisory_lock($1)", advisoryLockKey); err != nil {
		return fmt.Errorf("acquire schema lock: %w", err)
	}
	defer func() {
		unlockCtx, unlockCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer unlockCancel()
		_ = db.Exec(unlockCtx, "SELECT pg_advisory_unlock($1)", advisoryLockKey)
	}()
	return run()
}

func EnsureRequiredTables(ctx context.Context, db persistence.QueryableDB) error {
	missing := make([]string, 0)
	for _, table := range requiredTables {
		var present bool
		if err := db.QueryRow(ctx, `
SELECT EXISTS (
  SELECT 1
  FROM information_schema.tables
  WHERE table_schema = 'public' AND table_name = $1
)
`, table).Scan(&present); err != nil {
			return fmt.Errorf("check table %s: %w", table, err)
		}
		if !present {
			missing = append(missing, table)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required tables: %s", strings.Join(missing, ", "))
	}
	return nil
}

func EnsureSchema(ctx context.Context, db persistence.QueryableDB, fsys fs.FS) error {
	if err := db.Ping(ctx); err != nil {
		return fmt.Errorf("ping postgres: %w", err)
	}
	return withSchemaLock(ctx, db, func() error {
		if err := ApplyFS(ctx, db, fsys); err != nil {
			return err
		}
		if err := EnsureRequiredTables(ctx, db); err != nil {
			return err
		}
		return nil
	})
}
