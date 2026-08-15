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
	"group_members",
	"group_applications",
	"group_invites",
	"auth_sessions",
	"auth_refresh_tokens",
	"moments",
	"moment_mentions",
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

// withSchemaLock 用 pg_advisory_lock 串行化并发迁移进程。session 级 advisory
// lock 绑定在具体连接上：此前 lock/unlock 经 pgxpool 的普通 Exec 执行，每次
// 可能拿到不同连接——加锁在连接 A、解锁在连接 D（未持锁仅告警），真正的锁
// 挂在已归还池的连接 A 上滞留至其销毁，互斥完全不成立。现改为 Acquire 一条
// 专用连接持有锁的完整生命周期：lock 与 unlock 必然同一 session；迁移 SQL
// 仍走池（锁互斥的是「同时运行的迁移进程」，不要求 DDL 与锁同连接）。
func withSchemaLock(ctx context.Context, db persistence.QueryableDB, run func() error) error {
	pdb, ok := db.(*persistence.PostgresDB)
	if !ok || pdb.Backend != nil {
		// 非 pgxpool 后端（测试 stub / 内存）：无跨进程迁移竞争，直接执行。
		return run()
	}
	pool, err := pdb.EnsurePool(ctx)
	if err != nil {
		return fmt.Errorf("acquire pool for schema lock: %w", err)
	}
	if pool == nil {
		return run()
	}
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquire connection for schema lock: %w", err)
	}
	defer conn.Release()
	lockCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if _, err := conn.Exec(lockCtx, "SELECT pg_advisory_lock($1)", advisoryLockKey); err != nil {
		return fmt.Errorf("acquire schema lock: %w", err)
	}
	defer func() {
		unlockCtx, unlockCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer unlockCancel()
		_, _ = conn.Exec(unlockCtx, "SELECT pg_advisory_unlock($1)", advisoryLockKey)
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
