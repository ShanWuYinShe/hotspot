package share

import (
	"context"
	"errors"
	"testing"

	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/persistence"
)

func TestPostgresCreateGroupAndMomentShares(t *testing.T) {
	backend := newShareBackendStub()
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	repo := NewPostgresRepository(factory)

	group, err := repo.Create(context.Background(), createShareCommand{CreatorUserID: "user_alice", ShareType: ShareTypeGroup, Payload: map[string]any{"shared_group_id": "group_1"}})
	if err != nil || group.ShareID == "" {
		t.Fatalf("group share create: %#v err=%v", group, err)
	}
	moment, err := repo.Create(context.Background(), createShareCommand{CreatorUserID: "user_alice", ShareType: ShareTypeMoment, Payload: map[string]any{"shared_moment_id": "moment_1"}})
	if err != nil || moment.ShareID == "" {
		t.Fatalf("moment share create: %#v err=%v", moment, err)
	}
	// 未知 share type → invalid
	if _, err := repo.Create(context.Background(), createShareCommand{CreatorUserID: "user_alice", ShareType: "bogus", Payload: map[string]any{}}); !errors.Is(err, ErrInvalidShareRequest) {
		t.Fatalf("bogus type must be invalid, got %v", err)
	}
	// 目标不存在 → not found
	if _, err := repo.Create(context.Background(), createShareCommand{CreatorUserID: "user_alice", ShareType: ShareTypeGroup, Payload: map[string]any{"shared_group_id": "missing"}}); !errors.Is(err, ErrShareNotFound) {
		t.Fatalf("missing group must be not-found, got %v", err)
	}
}
