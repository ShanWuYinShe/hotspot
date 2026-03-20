package migrate

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"hotspot/backend-go/internal/platform/persistence"
)

func order(files []string) []string {
	out := append([]string(nil), files...)
	sort.Strings(out)
	return out
}

func ListMigrations(dir string) ([]string, error) {
	entries := []string{}
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(d.Name(), ".sql") {
			entries = append(entries, d.Name())
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return order(entries), nil
}

func ApplyDir(ctx context.Context, db persistence.QueryableDB, dir string) error {
	files, err := ListMigrations(dir)
	if err != nil {
		return err
	}
	for _, name := range files {
		blob, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return err
		}
		if err := db.Exec(ctx, string(blob)); err != nil {
			return err
		}
	}
	return nil
}
