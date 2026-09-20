package migrations

import (
	"io/fs"
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
