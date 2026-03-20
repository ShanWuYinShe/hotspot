package migrate

import "testing"

func TestListMigrationsSortsByVersion(t *testing.T) {
	files := []string{"003_moments_comments.sql", "001_users_core.sql", "002_auth_sessions.sql"}
	ordered := order(files)
	if ordered[0] != "001_users_core.sql" || ordered[2] != "003_moments_comments.sql" {
		t.Fatalf("unexpected order: %#v", ordered)
	}
}
