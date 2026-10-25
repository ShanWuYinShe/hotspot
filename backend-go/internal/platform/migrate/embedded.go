package migrate

import (
	"context"
	"fmt"
	"io/fs"
	"sort"
	"strings"
	"time"

	"hotspot/backend-go/internal/platform/observability"
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
		if err := execMigrationSQL(ctx, db, string(blob)); err != nil {
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

// schemaMigrationsTable 记录已应用的迁移文件名（name 为主键）。落地后：
//   - 新库按字典序逐个执行未登记文件并登记（替代旧「全量重跑」）；
//   - 存量库（升级前已有全部表、无版本记录）做一次性基线登记，不重放 55+ 个
//     历史文件——历史文件虽按纪律幂等，但全量重放拖慢启动且与未来
//     CREATE INDEX CONCURRENTLY 拆单语句迁移不兼容（见 check-migrations.sh）。
//   - 非 Postgres 后端（测试 stub / 内存）走旧 ApplyFS 全量路径，行为不变。
//
// 崩溃窗口：文件执行与登记非同一事务，中断后重启会重放该文件——幂等纪律是防线
// （与旧行为一致，未变差）。
const schemaMigrationsDDL = `CREATE TABLE IF NOT EXISTS schema_migrations (
  name       text PRIMARY KEY,
  applied_at timestamptz NOT NULL DEFAULT now()
)`

func EnsureSchema(ctx context.Context, db persistence.QueryableDB, fsys fs.FS) error {
	if err := db.Ping(ctx); err != nil {
		return fmt.Errorf("ping postgres: %w", err)
	}
	return withSchemaLock(ctx, db, func() error {
		if err := applyWithVersionTable(ctx, db, fsys); err != nil {
			return err
		}
		if err := EnsureRequiredTables(ctx, db); err != nil {
			return err
		}
		return nil
	})
}

// applyWithVersionTable 版本表驱动的应用逻辑。非 Postgres 后端退回全量重放。
func applyWithVersionTable(ctx context.Context, db persistence.QueryableDB, fsys fs.FS) error {
	pgdb, isPG := db.(*persistence.PostgresDB)
	if !isPG || pgdb.Backend != nil {
		return ApplyFS(ctx, db, fsys)
	}
	files, err := listMigrationsFS(fsys)
	if err != nil {
		return err
	}
	if err := db.Exec(ctx, schemaMigrationsDDL); err != nil {
		return fmt.Errorf("ensure schema_migrations: %w", err)
	}
	applied, err := loadAppliedMigrations(ctx, db)
	if err != nil {
		return err
	}
	pending := make([]string, 0, len(files))
	for _, name := range files {
		if !applied[name] {
			pending = append(pending, name)
		}
	}
	// 存量库基线：从未登记过且核心表已齐 → 全部现存文件只登记不重放。
	// 判定用 requiredTables 全量在位（半迁移状态的库不满足，落到正常执行路径
	// 逐个补齐；EnsureRequiredTables 兜底失败会拒绝启动，交由人工处理）。
	if len(applied) == 0 && len(pending) == len(files) {
		if err := EnsureRequiredTables(ctx, db); err == nil {
			for _, name := range files {
				if err := recordMigration(ctx, db, name); err != nil {
					return err
				}
			}
			observability.L().Info("schema_migrations: legacy database baselined",
				"files", len(files))
			return nil
		}
		// required tables 不齐：视为全新/半迁移库，走正常逐文件执行。
	}
	for _, name := range pending {
		blob, err := fs.ReadFile(fsys, name)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}
		if err := execMigrationSQL(ctx, db, string(blob)); err != nil {
			return fmt.Errorf("apply migration %s: %w", name, err)
		}
		if err := recordMigration(ctx, db, name); err != nil {
			return err
		}
		observability.L().Info("schema_migrations: applied", "file", name)
	}
	return nil
}

func loadAppliedMigrations(ctx context.Context, db persistence.QueryableDB) (map[string]bool, error) {
	rows, err := db.Query(ctx, "SELECT name FROM schema_migrations")
	if err != nil {
		return nil, fmt.Errorf("load schema_migrations: %w", err)
	}
	defer rows.Close()
	applied := make(map[string]bool)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scan schema_migrations: %w", err)
		}
		applied[name] = true
	}
	return applied, rows.Err()
}

func recordMigration(ctx context.Context, db persistence.QueryableDB, name string) error {
	if err := db.Exec(ctx, "INSERT INTO schema_migrations (name) VALUES ($1) ON CONFLICT (name) DO NOTHING", name); err != nil {
		return fmt.Errorf("record migration %s: %w", name, err)
	}
	return nil
}
