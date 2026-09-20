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
	// 建索引一律 CONCURRENTLY（迁移纪律：无停机替换下普通 CREATE INDEX 的
	// SHARE 锁阻塞读写；见 backend-go/AGENTS.md「数据库迁移」）。
	checks := []string{
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_moments_public_s2_cell_id ON moments(s2_cell_id) WHERE privacy_level = 'public';",
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_moments_non_private_s2_cell_id ON moments(s2_cell_id) WHERE privacy_level <> 'private';",
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_groups_visible_s2_cell_id ON groups(s2_cell_id) WHERE deleted_at IS NULL AND is_searchable = TRUE;",
	}
	for _, want := range checks {
		if !strings.Contains(sql, want) {
			t.Fatalf("migration set missing index statement %q\nSQL:\n%s", want, sql)
		}
	}
}
