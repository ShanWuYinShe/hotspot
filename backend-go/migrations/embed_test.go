package migrations

import (
	"io/fs"
	"regexp"
	"strings"
	"testing"
)

func TestS2LeafCellMigrationAddsPartialIndexesForHotReadPaths(t *testing.T) {
	entries, err := fs.ReadDir(Files, ".")
	if err != nil {
		t.Fatalf("list migrations: %v", err)
	}
	var allSQL strings.Builder
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		data, err := Files.ReadFile(entry.Name())
		if err != nil {
			t.Fatalf("read migration %s: %v", entry.Name(), err)
		}
		allSQL.Write(data)
		allSQL.WriteString("\n")
	}
	sql := allSQL.String()
	// 归一化后断言（同时接受普通与 CONCURRENTLY 写法）。当前迁移架构经 pgx
	// 简单协议整文件单消息执行，多语句文件落入隐式事务块，CONCURRENTLY 以
	// 25001 拒绝（2026-09-20 实测）——实际只允许普通形式，见 AGENTS.md。
	normalized := strings.ReplaceAll(sql, " INDEX CONCURRENTLY ", " INDEX ")
	checks := []string{
		"CREATE INDEX IF NOT EXISTS idx_moments_public_s2_cell_id ON moments(s2_cell_id) WHERE privacy_level = 'public';",
		"CREATE INDEX IF NOT EXISTS idx_moments_non_private_s2_cell_id ON moments(s2_cell_id) WHERE privacy_level <> 'private';",
		"CREATE INDEX IF NOT EXISTS idx_groups_visible_s2_cell_id ON groups(s2_cell_id) WHERE deleted_at IS NULL AND is_searchable = TRUE;",
	}
	for _, want := range checks {
		if !strings.Contains(normalized, want) {
			t.Fatalf("migration set missing index statement %q\nSQL:\n%s", want, sql)
		}
	}
}

// ── 索引 DDL 静态纪律 ──────────────────────────────────────────────
// 迁移文件会在两种「非只跑一次」的场景下重执行：存量库基线登记后的增量执行，
// 以及「执行成功但版本登记前崩溃」后的重启重放（EnsureSchema 的崩溃窗口）。
// 因此索引 DDL 必须幂等、同名索引定义必须全库一致，否则新库（顺序执行）与
// 存量库（增量执行）会得到语义不同的索引。

var (
	// 跨行匹配单条 CREATE [UNIQUE] INDEX ... ;（调用前已去掉 -- 行注释）。
	sqlCreateIndexStatementPattern = regexp.MustCompile(`(?is)create\s+(?:unique\s+)?index\s+[^;]+;`)
	// 语句内是否带 IF NOT EXISTS。
	sqlCreateIndexIdempotentPattern = regexp.MustCompile(`(?i)^create\s+(?:unique\s+)?index\s+if\s+not\s+exists\s+`)
	// 语句中的索引名。
	sqlCreateIndexNamePattern = regexp.MustCompile(`(?i)^create\s+(?:unique\s+)?index\s+(?:if\s+not\s+exists\s+)?([a-z0-9_]+)`)
	// -- 行注释（迁移文件不用 /* */ 块注释）。
	sqlLineCommentPattern = regexp.MustCompile(`--[^\n]*`)
)

type createIndexStatement struct {
	file       string
	name       string
	normalized string
}

// collectCreateIndexStatements 按文件名顺序收集全部迁移里的 CREATE INDEX 语句，
// 语句归一化为小写 + 单空格（忽略空白/大小写差异）。
func collectCreateIndexStatements(t *testing.T) []createIndexStatement {
	t.Helper()
	entries, err := fs.ReadDir(Files, ".")
	if err != nil {
		t.Fatalf("list migrations: %v", err)
	}
	statements := make([]createIndexStatement, 0, 96)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		data, err := Files.ReadFile(entry.Name())
		if err != nil {
			t.Fatalf("read migration %s: %v", entry.Name(), err)
		}
		cleaned := sqlLineCommentPattern.ReplaceAllString(string(data), "")
		for _, stmt := range sqlCreateIndexStatementPattern.FindAllString(cleaned, -1) {
			match := sqlCreateIndexNamePattern.FindStringSubmatch(stmt)
			if match == nil {
				t.Fatalf("无法解析 CREATE INDEX 语句的索引名（%s）：%s", entry.Name(), stmt)
			}
			statements = append(statements, createIndexStatement{
				file:       entry.Name(),
				name:       match[1],
				normalized: normalizeDDL(stmt),
			})
		}
	}
	return statements
}

func normalizeDDL(stmt string) string {
	return strings.Join(strings.Fields(strings.ToLower(stmt)), " ")
}

// TestCreateIndexStatementsAreIdempotent 锁定 skill hotspot-backend-migrations
// 的幂等纪律：迁移里的 CREATE INDEX 一律带 IF NOT EXISTS。
func TestCreateIndexStatementsAreIdempotent(t *testing.T) {
	statements := collectCreateIndexStatements(t)
	if len(statements) == 0 {
		t.Fatal("未解析到任何 CREATE INDEX 语句，正则或 embed 可能失效")
	}
	for _, stmt := range statements {
		if !sqlCreateIndexIdempotentPattern.MatchString(stmt.normalized) {
			t.Errorf("%s: CREATE INDEX 缺少 IF NOT EXISTS（存量库基线/崩溃重放会重复执行该文件）：%s", stmt.file, stmt.normalized)
		}
	}
}

// TestDuplicateIndexDeclarationsShareDefinition：同名索引允许在多个迁移中声明
// （056 建索引 + 058 为存量库补建），但定义（表/列/排序/WHERE 谓词）必须完全
// 一致，否则新库（056 执行、058 no-op）与存量库（056 被基线跳过、058 执行）
// 会得到语义不同的索引。
func TestDuplicateIndexDeclarationsShareDefinition(t *testing.T) {
	seen := make(map[string]createIndexStatement)
	for _, stmt := range collectCreateIndexStatements(t) {
		prev, ok := seen[stmt.name]
		if !ok {
			seen[stmt.name] = stmt
			continue
		}
		if prev.normalized != stmt.normalized {
			t.Errorf("索引 %s 在 %s 与 %s 中定义不一致：\n  前者：%s\n  后者：%s", stmt.name, prev.file, stmt.file, prev.normalized, stmt.normalized)
		}
	}
}

// TestMomentsKeysetIndexBackfilledForLegacyBaseline：线上库 2026-10-08 做过
// 「存量库基线」（schema_migrations 只登记不执行），056 声明的
// idx_moments_keyset 在基线的数据快照里不存在，于是被永久标记为已应用、索引
// 永远不会补建。修法是 056 之后的新迁移（058）重放同一索引定义——直接改 056
// 对已基线库无效。本测试防止补建迁移被误删。
func TestMomentsKeysetIndexBackfilledForLegacyBaseline(t *testing.T) {
	const indexName = "idx_moments_keyset"
	const originFile = "056_moments_keyset_index.sql"
	declaringFiles := make([]string, 0, 2)
	for _, stmt := range collectCreateIndexStatements(t) {
		if stmt.name == indexName {
			declaringFiles = append(declaringFiles, stmt.file)
		}
	}
	if len(declaringFiles) < 2 {
		t.Fatalf("索引 %s 只在 %v 中声明：存量库基线会跳过 056 而永远缺该索引，需要 056 之后的新迁移重放定义", indexName, declaringFiles)
	}
	backfillFound := false
	for _, file := range declaringFiles {
		if file > originFile {
			backfillFound = true
		}
	}
	if !backfillFound {
		t.Fatalf("索引 %s 的声明文件 %v 中没有位于 %s 之后的新迁移（已基线库只会执行新文件）", indexName, declaringFiles, originFile)
	}
}
