package moments

import (
	"context"
	"errors"
	"testing"

	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/persistence"
)

// momentExecStub 提供 Exec + QueryRow stub。
type momentExecStub struct {
	execErr   error
	execs     int
	rowValues []any
}

func (s *momentExecStub) Exec(_ context.Context, _ string, _ ...any) error {
	s.execs++
	return s.execErr
}

func (s *momentExecStub) QueryRow(_ context.Context, _ string, _ ...any) persistence.Row {
	return momentRow{values: s.rowValues}
}

func (s *momentExecStub) Query(_ context.Context, _ string, _ ...any) (persistence.Rows, error) {
	return &persistence.InMemoryRows{}, nil
}

type momentRow struct {
	values []any
}

func (r momentRow) Scan(dest ...any) error {
	if r.values == nil {
		return errMomentNoRows
	}
	for i := range dest {
		switch d := dest[i].(type) {
		case *string:
			*d = r.values[i].(string)
		case *bool:
			*d = r.values[i].(bool)
		case *int:
			*d = r.values[i].(int)
		}
	}
	return nil
}

func newMomentFactory(backend persistence.Queryable) *persistence.Factory {
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	return factory
}

func TestPostgresMomentCountersAndFavorites(t *testing.T) {
	backend := &momentExecStub{}
	repo := NewPostgresRepository(newMomentFactory(backend))
	ctx := context.Background()

	if err := repo.IncrementCommentCount(ctx, "m1"); err != nil {
		t.Fatalf("increment comments: %v", err)
	}
	if err := repo.DecrementCommentCount(ctx, "m1"); err != nil {
		t.Fatalf("decrement comments: %v", err)
	}
	if err := repo.IncrementSharesCount(ctx, "m1"); err != nil {
		t.Fatalf("increment shares: %v", err)
	}
	if err := repo.Favorite(ctx, "u1", "m1"); err != nil {
		t.Fatalf("favorite: %v", err)
	}
	if err := repo.Unfavorite(ctx, "u1", "m1"); err != nil {
		t.Fatalf("unfavorite: %v", err)
	}
	if err := repo.Delete(ctx, "m1"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if backend.execs < 6 {
		t.Fatalf("expected writes, got %d", backend.execs)
	}

	// Exec 失败传播
	backend.execErr = errMomentFake
	if err := repo.Favorite(ctx, "u1", "m1"); !errors.Is(err, errMomentFake) {
		t.Fatalf("expected exec error propagation, got %v", err)
	}
}

func TestPostgresMomentNotificationContext(t *testing.T) {
	// 无 DSN 的 factory → Get 返回 ErrNotImplemented
	repo := NewPostgresRepository(persistence.NewFactory(config.Config{}))
	if _, _, err := repo.NotificationContext(context.Background(), "m1"); err != persistence.ErrNotImplemented {
		t.Fatalf("expected ErrNotImplemented, got %v", err)
	}
}

func TestPostgresMomentsNotConfigured(t *testing.T) {
	repo := NewPostgresRepository(persistence.NewFactory(config.Config{}))
	ctx := context.Background()
	if err := repo.IncrementCommentCount(ctx, "m1"); err != persistence.ErrNotImplemented {
		t.Fatalf("increment: %v", err)
	}
	if err := repo.IncrementSharesCount(ctx, "m1"); err != persistence.ErrNotImplemented {
		t.Fatalf("shares: %v", err)
	}
	if err := repo.Delete(ctx, "m1"); err != persistence.ErrNotImplemented {
		t.Fatalf("delete: %v", err)
	}
	if err := repo.Favorite(ctx, "u1", "m1"); err != persistence.ErrNotImplemented {
		t.Fatalf("favorite: %v", err)
	}
}

var errMomentFake = errors.New("fake moment db error")
var errMomentNoRows = errors.New("no rows")

func TestPostgresMomentUpdateBranches(t *testing.T) {
	backend := &momentExecStub{}
	repo := NewPostgresRepository(newMomentFactory(backend))
	ctx := context.Background()

	// 纯文本更新（无 tags/mentions/images）
	desc := "new"
	req := UpdateMomentRequest{MomentID: "m1", Description: &desc}
	if err := repo.Update(ctx, "u1", req); err != nil {
		t.Fatalf("update text: %v", err)
	}
	// 带 location
	loc := Location{LocationName: "上海", Point: GeoPoint{Latitude: 31.2, Longitude: 121.5}}
	req.Location = &loc
	if err := repo.Update(ctx, "u1", req); err != nil {
		t.Fatalf("update with location: %v", err)
	}
	// 带 tags + mentions + images
	req.Tags = []string{"美食"}
	req.Mentions = []string{"u2"}
	req.ImageFileIDs = []string{"f1"}
	if err := repo.Update(ctx, "u1", req); err != nil {
		t.Fatalf("update full: %v", err)
	}
	if backend.execs < 5 {
		t.Fatalf("expected multiple writes, got %d", backend.execs)
	}
}
