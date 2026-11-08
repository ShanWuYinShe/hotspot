package migrate

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"sort"
	"strings"
	"time"

	"hotspot/backend-go/internal/platform/observability"
	"hotspot/backend-go/internal/platform/persistence"
)

const advisoryLockKey int64 = 2026032301

// schemaMigrationsTable 记录已应用迁移：文件名 + 文件内容 checksum + 执行耗时。
// 不变量：
//   - 迁移只从空库按文件顺序执行；已有业务表但版本表为空时直接失败，不做基线/接管；
//   - 每个已应用文件的 sha256 入库，历史文件被改动/删除会在启动期 fail-fast；
//   - 每个迁移文件与其版本登记在同一事务内提交，消除「执行成功但登记前崩溃」窗口。
const schemaMigrationsDDL = `CREATE TABLE IF NOT EXISTS schema_migrations (
  name         text PRIMARY KEY,
  checksum     text,
  execution_ms bigint,
  applied_at   timestamptz NOT NULL DEFAULT now()
);
ALTER TABLE schema_migrations ADD COLUMN IF NOT EXISTS checksum text;
ALTER TABLE schema_migrations ADD COLUMN IF NOT EXISTS execution_ms bigint;
`

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

// ApplyFS 全量执行迁移文件（无版本表语义）。仅用于测试 stub / 内存后端；
// 生产 Postgres 路径走 EnsureSchema 的版本表 + checksum + manifest 校验。
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
// lock 绑定在具体连接上：lock/unlock 必须在同一 session 上执行，故 Acquire
// 一条专用连接持有锁的完整生命周期。
func withSchemaLock(ctx context.Context, db persistence.QueryableDB, run func() error) error {
	pdb, ok := db.(*persistence.PostgresDB)
	if !ok || pdb.Backend != nil {
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

// isPostgres 判断是否为可执行版本表语义的 Postgres 连接池。测试 stub
// （PostgresDB.Backend != nil）与内存后端走全量执行路径。
func isPostgres(db persistence.QueryableDB) (*persistence.PostgresDB, bool) {
	pdb, ok := db.(*persistence.PostgresDB)
	if !ok || pdb.Backend != nil {
		return nil, false
	}
	return pdb, true
}

// EnsureSchema 是应用启动期的 schema 入口：确保版本表、校验历史 checksum、
// 增量执行 pending 迁移（每文件与登记同事务），最后用 schema manifest 校验
// 「期望对象是否都存在」。任何不一致都返回错误（启动失败），不静默继续。
func EnsureSchema(ctx context.Context, db persistence.QueryableDB, fsys fs.FS) error {
	return ensureSchema(ctx, db, fsys, true)
}

// EnsureSchemaForManifest 与 EnsureSchema 相同，但跳过末尾的 schema manifest 校验。
// 仅供 `cmd/migrate -write-manifest` 重建 manifest 时使用：此时嵌入的是旧 manifest，
// 新迁移刚改变了期望 schema，先应用再生成新 manifest，再用新 manifest 校验。
// 生产启动路径不得使用（会绕过漂移检测）。
func EnsureSchemaForManifest(ctx context.Context, db persistence.QueryableDB, fsys fs.FS) error {
	return ensureSchema(ctx, db, fsys, false)
}

func ensureSchema(ctx context.Context, db persistence.QueryableDB, fsys fs.FS, verifyManifest bool) error {
	if err := db.Ping(ctx); err != nil {
		return fmt.Errorf("ping postgres: %w", err)
	}
	if _, ok := isPostgres(db); !ok {
		// 测试 stub / 内存后端：无版本表与 catalog 语义，退回全量执行。
		return ApplyFS(ctx, db, fsys)
	}
	return withSchemaLock(ctx, db, func() error {
		if err := ensureVersionTable(ctx, db); err != nil {
			return err
		}
		files, err := listMigrationsFS(fsys)
		if err != nil {
			return err
		}
		checksums, err := migrationChecksums(fsys, files)
		if err != nil {
			return err
		}
		applied, err := loadAppliedMigrations(ctx, db)
		if err != nil {
			return err
		}
		if len(applied) == 0 && len(files) > 0 {
			hasTables, err := databaseHasApplicationTables(ctx, db)
			if err != nil {
				return err
			}
			if hasTables {
				return fmt.Errorf("database already contains application tables but schema_migrations is empty; this runner only supports initializing an empty database — create a fresh database and run the full migration set")
			}
		}
		pending, err := planMigrations(files, applied, checksums)
		if err != nil {
			return err
		}
		for _, name := range pending {
			blob, err := fs.ReadFile(fsys, name)
			if err != nil {
				return fmt.Errorf("read migration %s: %w", name, err)
			}
			if err := applyMigration(ctx, db, name, string(blob), checksums[name]); err != nil {
				return err
			}
			observability.L().Info("schema_migrations: applied", "file", name)
		}
		if !verifyManifest {
			return nil
		}
		return verifySchemaManifest(ctx, db, fsys)
	})
}

// VerifySchema 执行只读校验：已应用文件的 checksum、pending 是否为空、schema
// manifest 期望对象是否齐全。CI/部署门禁使用（不写库、不执行迁移）。
func VerifySchema(ctx context.Context, db persistence.QueryableDB, fsys fs.FS) error {
	if _, ok := isPostgres(db); !ok {
		return nil
	}
	files, err := listMigrationsFS(fsys)
	if err != nil {
		return err
	}
	checksums, err := migrationChecksums(fsys, files)
	if err != nil {
		return err
	}
	applied, err := loadAppliedMigrations(ctx, db)
	if err != nil {
		return err
	}
	pending, err := planMigrations(files, applied, checksums)
	if err != nil {
		return err
	}
	if len(pending) > 0 {
		return fmt.Errorf("schema has %d pending migrations: %s", len(pending), strings.Join(limitStrings(pending, 10), ", "))
	}
	for name, sum := range applied {
		if sum == "" {
			return fmt.Errorf("migration %q has no recorded checksum; schema_migrations must be written by this runner", name)
		}
	}
	return verifySchemaManifest(ctx, db, fsys)
}

func ensureVersionTable(ctx context.Context, db persistence.QueryableDB) error {
	if err := db.Exec(ctx, schemaMigrationsDDL); err != nil {
		return fmt.Errorf("ensure schema_migrations: %w", err)
	}
	return nil
}

// migrationChecksums 计算全部迁移文件的内容哈希（sha256 hex）。
func migrationChecksums(fsys fs.FS, files []string) (map[string]string, error) {
	out := make(map[string]string, len(files))
	for _, name := range files {
		blob, err := fs.ReadFile(fsys, name)
		if err != nil {
			return nil, fmt.Errorf("read migration %s: %w", name, err)
		}
		out[name] = migrationChecksum(blob)
	}
	return out, nil
}

func migrationChecksum(blob []byte) string {
	sum := sha256.Sum256(blob)
	return hex.EncodeToString(sum[:])
}

// planMigrations 校验已应用集合与当前文件集合的一致性，返回待执行文件列表。
// 两类失败必须 fail-fast：
//   - 已登记的文件在仓库中不存在（历史被改写/删除）；
//   - 已登记文件的当前内容 checksum 与记录不一致（迁移内容被修改）。
func planMigrations(files []string, applied map[string]string, checksums map[string]string) ([]string, error) {
	fileSet := make(map[string]struct{}, len(files))
	for _, name := range files {
		fileSet[name] = struct{}{}
	}
	appliedNames := make([]string, 0, len(applied))
	for name := range applied {
		appliedNames = append(appliedNames, name)
	}
	sort.Strings(appliedNames)
	for _, name := range appliedNames {
		if _, ok := fileSet[name]; !ok {
			return nil, fmt.Errorf("schema_migrations records %q but no such migration file exists (migration history was rewritten or a file was deleted)", name)
		}
		recorded := applied[name]
		if recorded != "" && checksums[name] != recorded {
			return nil, fmt.Errorf("migration %q checksum mismatch: recorded %s but current file is %s (historical migration content must not change; add a new migration instead)", name, recorded, checksums[name])
		}
	}
	pending := make([]string, 0, len(files))
	for _, name := range files {
		if _, ok := applied[name]; !ok {
			pending = append(pending, name)
		}
	}
	return pending, nil
}

func loadAppliedMigrations(ctx context.Context, db persistence.QueryableDB) (map[string]string, error) {
	rows, err := db.Query(ctx, "SELECT name, COALESCE(checksum, '') FROM schema_migrations")
	if err != nil {
		return nil, fmt.Errorf("load schema_migrations: %w", err)
	}
	defer rows.Close()
	applied := make(map[string]string)
	for rows.Next() {
		var name, checksum string
		if err := rows.Scan(&name, &checksum); err != nil {
			return nil, fmt.Errorf("scan schema_migrations: %w", err)
		}
		applied[name] = checksum
	}
	return applied, rows.Err()
}

// databaseHasApplicationTables 判断库内是否已有业务表（忽略迁移簿记表）。
func databaseHasApplicationTables(ctx context.Context, db persistence.QueryableDB) (bool, error) {
	var exists bool
	err := db.QueryRow(ctx, `
SELECT EXISTS (
  SELECT 1
  FROM information_schema.tables
  WHERE table_schema = 'public'
    AND table_type = 'BASE TABLE'
    AND table_name NOT IN ('schema_migrations', 'data_migrations')
)`).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check existing tables: %w", err)
	}
	return exists, nil
}

// applyMigration 在单个事务内执行迁移文件并登记版本（原子）：文件执行成功而
// 登记失败/进程崩溃时整体回滚，重启后重放该文件，不会出现「已执行但未登记」
// 的重放窗口，也不会把失败文件标记为已应用。
func applyMigration(ctx context.Context, db persistence.QueryableDB, name, sql, checksum string) error {
	pdb, ok := db.(*persistence.PostgresDB)
	if !ok || pdb.Backend != nil || pdb.Pool == nil {
		return execMigrationSQL(ctx, db, sql)
	}
	pool, err := pdb.EnsurePool(ctx)
	if err != nil {
		return fmt.Errorf("apply migration %s: %w", name, err)
	}
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("apply migration %s: acquire conn: %w", name, err)
	}
	defer conn.Release()
	tx, err := conn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("apply migration %s: begin: %w", name, err)
	}
	started := time.Now()
	// 无参数 SQL 会走 pgx 简单协议，支持整文件多语句（与历史 execMigrationSQL
	// 语义一致）；迁移文件整体在显式事务内，仍禁止 CREATE INDEX CONCURRENTLY。
	if _, err := tx.Exec(ctx, sql); err != nil {
		_ = tx.Rollback(ctx)
		return fmt.Errorf("apply migration %s: %w", name, err)
	}
	if _, err := tx.Exec(ctx,
		"INSERT INTO schema_migrations (name, checksum, execution_ms) VALUES ($1, $2, $3) ON CONFLICT (name) DO UPDATE SET checksum = EXCLUDED.checksum, execution_ms = EXCLUDED.execution_ms, applied_at = now()",
		name, checksum, time.Since(started).Milliseconds()); err != nil {
		_ = tx.Rollback(ctx)
		return fmt.Errorf("record migration %s: %w", name, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit migration %s: %w", name, err)
	}
	return nil
}

func limitStrings(items []string, max int) []string {
	if len(items) <= max {
		return items
	}
	out := append([]string(nil), items[:max]...)
	out = append(out, fmt.Sprintf("...(and %d more)", len(items)-max))
	return out
}
