package migrate

import (
	"strings"
	"testing"
	"testing/fstest"
)

func TestListMigrationsSortsByVersion(t *testing.T) {
	files := []string{"003_moments_comments.sql", "001_users_core.sql", "002_auth_sessions.sql"}
	ordered := order(files)
	if ordered[0] != "001_users_core.sql" || ordered[2] != "003_moments_comments.sql" {
		t.Fatalf("unexpected order: %#v", ordered)
	}
}

func TestListMigrationsFSSortsSQLFiles(t *testing.T) {
	fsys := fstest.MapFS{
		"002_auth_sessions.sql":   &fstest.MapFile{Data: []byte("select 2")},
		"001_users_core.sql":      &fstest.MapFile{Data: []byte("select 1")},
		"README.txt":              &fstest.MapFile{Data: []byte("ignore")},
		"nested/003_comments.sql": &fstest.MapFile{Data: []byte("ignore nested")},
	}
	files, err := listMigrationsFS(fsys)
	if err != nil {
		t.Fatalf("list migrations fs: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("unexpected file count: %#v", files)
	}
	if files[0] != "001_users_core.sql" || files[1] != "002_auth_sessions.sql" {
		t.Fatalf("unexpected files: %#v", files)
	}
}

func TestMigrationChecksumIsStableAndContentSensitive(t *testing.T) {
	a := migrationChecksum([]byte("CREATE TABLE t (id text);"))
	b := migrationChecksum([]byte("CREATE TABLE t (id text);"))
	c := migrationChecksum([]byte("CREATE TABLE t (id text, name text);"))
	if a != b {
		t.Fatalf("checksum must be deterministic: %s vs %s", a, b)
	}
	if a == c {
		t.Fatal("checksum must change when file content changes")
	}
	if len(a) != 64 {
		t.Fatalf("want sha256 hex (64 chars), got %d", len(a))
	}
}

func TestPlanMigrationsVerifiesAppliedHistory(t *testing.T) {
	files := []string{"001_a.sql", "002_b.sql", "003_c.sql"}
	checksums := map[string]string{
		"001_a.sql": "h1",
		"002_b.sql": "h2",
		"003_c.sql": "h3",
	}
	t.Run("pending excludes applied and keeps order", func(t *testing.T) {
		pending, err := planMigrations(files, map[string]string{"001_a.sql": "h1", "002_b.sql": "h2"}, checksums)
		if err != nil {
			t.Fatalf("plan: %v", err)
		}
		if len(pending) != 1 || pending[0] != "003_c.sql" {
			t.Fatalf("unexpected pending: %#v", pending)
		}
	})
	t.Run("empty recorded checksum row stays applied (strict verification is a separate check)", func(t *testing.T) {
		pending, err := planMigrations(files, map[string]string{"001_a.sql": ""}, checksums)
		if err != nil {
			t.Fatalf("plan: %v", err)
		}
		if len(pending) != 2 || pending[0] != "002_b.sql" || pending[1] != "003_c.sql" {
			t.Fatalf("unexpected pending: %#v", pending)
		}
	})
	t.Run("applied file removed from repository fails fast", func(t *testing.T) {
		_, err := planMigrations(files, map[string]string{"999_deleted.sql": "h9"}, checksums)
		if err == nil || !strings.Contains(err.Error(), "999_deleted.sql") {
			t.Fatalf("want missing-file error, got %v", err)
		}
	})
	t.Run("changed historical migration fails fast", func(t *testing.T) {
		_, err := planMigrations(files, map[string]string{"001_a.sql": "tampered"}, checksums)
		if err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
			t.Fatalf("want checksum mismatch error, got %v", err)
		}
	})
}
