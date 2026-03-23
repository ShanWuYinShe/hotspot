package migrate

import (
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
