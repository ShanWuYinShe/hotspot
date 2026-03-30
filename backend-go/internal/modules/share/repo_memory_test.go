package share

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestInMemoryRepositoryCreateGetListAndRevoke(t *testing.T) {
	repo := NewInMemoryRepository()
	expiresAt := time.Now().UTC().Add(time.Minute)
	created, err := repo.Create(context.Background(), createShareCommand{CreatorUserID: "u1", ShareType: ShareTypeUser, Payload: map[string]any{"shared_user_id": "u2"}, ExpiresAt: &expiresAt})
	if err != nil || created.ShareID == "" || !validShareCode(created.ShareCode) {
		t.Fatalf("unexpected created share: %#v err=%v", created, err)
	}
	detail, err := repo.Get(context.Background(), getShareQuery{ShareCode: created.ShareCode})
	if err != nil || detail.Share.ShareID != created.ShareID {
		t.Fatalf("unexpected share detail: %#v err=%v", detail, err)
	}
	list, err := repo.List(context.Background(), listSharesQuery{CreatorUserID: "u1", Limit: 20})
	if err != nil || len(list.Items) != 1 {
		t.Fatalf("unexpected share list: %#v err=%v", list, err)
	}
	if err := repo.Revoke(context.Background(), revokeShareCommand{CreatorUserID: "u1", ShareID: created.ShareID}); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	if _, err := repo.Get(context.Background(), getShareQuery{ShareCode: created.ShareCode}); !errors.Is(err, ErrShareNotFound) {
		t.Fatalf("expected not found after revoke, got %v", err)
	}
}

func TestInMemoryRepositoryRejectsForeignRevoke(t *testing.T) {
	repo := NewInMemoryRepository()
	created, _ := repo.Create(context.Background(), createShareCommand{CreatorUserID: "u1", ShareType: ShareTypeUser, Payload: map[string]any{"shared_user_id": "u2"}})
	if err := repo.Revoke(context.Background(), revokeShareCommand{CreatorUserID: "u3", ShareID: created.ShareID}); !errors.Is(err, ErrSharePermission) {
		t.Fatalf("expected ErrSharePermission, got %v", err)
	}
}

func TestInMemoryRepositoryGetOmitsPreviewWhenTargetBecomesUnavailable(t *testing.T) {
	repo := NewInMemoryRepository()
	created, err := repo.Create(context.Background(), createShareCommand{CreatorUserID: "u1", ShareType: ShareTypeUser, Payload: map[string]any{"shared_user_id": "u2"}})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	repo.mu.Lock()
	record := repo.shares[created.ShareID]
	record.Payload = map[string]any{"shared_user_id": ""}
	repo.shares[created.ShareID] = record
	repo.mu.Unlock()

	detail, err := repo.Get(context.Background(), getShareQuery{ShareCode: created.ShareCode})
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if detail.Preview != nil {
		t.Fatalf("expected nil preview, got %#v", detail.Preview)
	}
}

func TestInMemoryRepositoryListOmitsPreviewForRevokedItems(t *testing.T) {
	repo := NewInMemoryRepository()
	created, err := repo.Create(context.Background(), createShareCommand{CreatorUserID: "u1", ShareType: ShareTypeUser, Payload: map[string]any{"shared_user_id": "u2"}})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := repo.Revoke(context.Background(), revokeShareCommand{CreatorUserID: "u1", ShareID: created.ShareID}); err != nil {
		t.Fatalf("revoke: %v", err)
	}

	list, err := repo.List(context.Background(), listSharesQuery{CreatorUserID: "u1", Limit: 20})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list.Items) != 1 {
		t.Fatalf("unexpected list: %#v", list)
	}
	if !list.Items[0].IsRevoked {
		t.Fatalf("expected revoked item, got %#v", list.Items[0])
	}
	if list.Items[0].Preview != nil {
		t.Fatalf("expected nil preview, got %#v", list.Items[0].Preview)
	}
}
