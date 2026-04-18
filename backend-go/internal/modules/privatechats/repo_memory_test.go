package privatechats

import (
	"context"
	"testing"
	"time"
)

func TestInMemoryRepositoryGetOrCreateCreatesNewChat(t *testing.T) {
	repo := NewInMemoryRepository()

	createdAt, err := repo.GetOrCreate(context.Background(), "alice", "bob")
	if err != nil {
		t.Fatalf("GetOrCreate returned error: %v", err)
	}
	if createdAt.IsZero() {
		t.Fatal("expected non-zero createdAt")
	}
}

func TestInMemoryRepositoryGetOrCreateReturnsSameChat(t *testing.T) {
	repo := NewInMemoryRepository()

	first, err := repo.GetOrCreate(context.Background(), "alice", "bob")
	if err != nil {
		t.Fatalf("first GetOrCreate returned error: %v", err)
	}
	second, err := repo.GetOrCreate(context.Background(), "alice", "bob")
	if err != nil {
		t.Fatalf("second GetOrCreate returned error: %v", err)
	}
	if !first.Equal(second) {
		t.Fatalf("expected same createdAt, got %v vs %v", first, second)
	}
}

func TestInMemoryRepositoryGetOrCreateOrderIndependent(t *testing.T) {
	repo := NewInMemoryRepository()

	ab, err := repo.GetOrCreate(context.Background(), "alice", "bob")
	if err != nil {
		t.Fatalf("GetOrCreate(alice,bob) returned error: %v", err)
	}
	ba, err := repo.GetOrCreate(context.Background(), "bob", "alice")
	if err != nil {
		t.Fatalf("GetOrCreate(bob,alice) returned error: %v", err)
	}
	if !ab.Equal(ba) {
		t.Fatalf("expected same createdAt regardless of order, got %v vs %v", ab, ba)
	}
}

func TestInMemoryRepositoryListReturnsChatsForUser(t *testing.T) {
	repo := NewInMemoryRepository()

	_, _ = repo.GetOrCreate(context.Background(), "alice", "bob")
	_, _ = repo.GetOrCreate(context.Background(), "alice", "carol")

	chats, _, err := repo.List(context.Background(), "alice", ListOptions{})
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(chats) != 2 {
		t.Fatalf("expected 2 chats for alice, got %d", len(chats))
	}
}

func TestInMemoryRepositoryListReturnsEmptyForUserWithNoChats(t *testing.T) {
	repo := NewInMemoryRepository()

	chats, _, err := repo.List(context.Background(), "stranger", ListOptions{})
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(chats) != 0 {
		t.Fatalf("expected 0 chats for stranger, got %d", len(chats))
	}
}

func TestInMemoryRepositoryGetStateReturnsExistingState(t *testing.T) {
	repo := NewInMemoryRepository()

	_, _ = repo.GetOrCreate(context.Background(), "alice", "bob")

	state, err := repo.GetState(context.Background(), "alice", "bob")
	if err != nil {
		t.Fatalf("GetState returned error: %v", err)
	}
	if state.ChatKey == "" {
		t.Fatal("expected non-empty ChatKey")
	}
	if state.TargetID != "bob" {
		t.Fatalf("expected TargetID 'bob', got %q", state.TargetID)
	}
}

func TestInMemoryRepositoryGetStateReturnsEmptyForUnknownPair(t *testing.T) {
	repo := NewInMemoryRepository()

	state, err := repo.GetState(context.Background(), "alice", "bob")
	if err != nil {
		t.Fatalf("GetState returned error: %v", err)
	}
	if state.ChatKey == "" {
		t.Fatal("expected non-empty ChatKey for unknown pair (creates on demand)")
	}
}

func TestInMemoryRepositoryUpdateSettingsMute(t *testing.T) {
	repo := NewInMemoryRepository()
	_, _ = repo.GetOrCreate(context.Background(), "alice", "bob")

	mute := true
	err := repo.UpdateSettings(context.Background(), "alice", "bob", UpdateChatSettings{IsMuted: &mute})
	if err != nil {
		t.Fatalf("UpdateSettings returned error: %v", err)
	}
	state, _ := repo.GetState(context.Background(), "alice", "bob")
	if !state.IsMuted {
		t.Fatal("expected IsMuted=true")
	}
}

func TestInMemoryRepositoryUpdateSettingsPin(t *testing.T) {
	repo := NewInMemoryRepository()
	_, _ = repo.GetOrCreate(context.Background(), "alice", "bob")

	pin := true
	err := repo.UpdateSettings(context.Background(), "alice", "bob", UpdateChatSettings{IsPinned: &pin})
	if err != nil {
		t.Fatalf("UpdateSettings returned error: %v", err)
	}
	state, _ := repo.GetState(context.Background(), "alice", "bob")
	if !state.IsPinned {
		t.Fatal("expected IsPinned=true")
	}
}

func TestInMemoryRepositoryUpdateSettingsMultiple(t *testing.T) {
	repo := NewInMemoryRepository()
	_, _ = repo.GetOrCreate(context.Background(), "alice", "bob")

	mute := true
	pin := true
	err := repo.UpdateSettings(context.Background(), "alice", "bob", UpdateChatSettings{IsMuted: &mute, IsPinned: &pin})
	if err != nil {
		t.Fatalf("UpdateSettings returned error: %v", err)
	}
	state, _ := repo.GetState(context.Background(), "alice", "bob")
	if !state.IsMuted || !state.IsPinned {
		t.Fatal("expected IsMuted=true and IsPinned=true")
	}
}

func TestInMemoryRepositoryDeleteForUser(t *testing.T) {
	repo := NewInMemoryRepository()
	_, _ = repo.GetOrCreate(context.Background(), "alice", "bob")

	deletedAt := time.Now().UTC()
	err := repo.DeleteForUser(context.Background(), "alice", "bob", &deletedAt)
	if err != nil {
		t.Fatalf("DeleteForUser returned error: %v", err)
	}
	state, _ := repo.GetState(context.Background(), "alice", "bob")
	if state.DeletedAt == nil {
		t.Fatal("expected DeletedAt to be set")
	}
}

func TestInMemoryRepositoryRestoreForUser(t *testing.T) {
	repo := NewInMemoryRepository()
	_, _ = repo.GetOrCreate(context.Background(), "alice", "bob")

	deletedAt := time.Now().UTC()
	_ = repo.DeleteForUser(context.Background(), "alice", "bob", &deletedAt)

	restoreAt := time.Now().UTC()
	err := repo.RestoreForUser(context.Background(), "alice", "bob", restoreAt)
	if err != nil {
		t.Fatalf("RestoreForUser returned error: %v", err)
	}
	state, _ := repo.GetState(context.Background(), "alice", "bob")
	if state.DeletedAt != nil {
		t.Fatalf("expected DeletedAt=nil after restore, got %v", state.DeletedAt)
	}
	if state.VisibleSince == nil {
		t.Fatal("expected VisibleSince to be set")
	}
}

func TestInMemoryRepositoryDeleteForUserWithoutTimestamp(t *testing.T) {
	repo := NewInMemoryRepository()
	_, _ = repo.GetOrCreate(context.Background(), "alice", "bob")

	err := repo.DeleteForUser(context.Background(), "alice", "bob", nil)
	if err != nil {
		t.Fatalf("DeleteForUser(nil) returned error: %v", err)
	}
	state, _ := repo.GetState(context.Background(), "alice", "bob")
	if state.DeletedAt != nil {
		t.Fatal("expected DeletedAt=nil when deleted with nil timestamp")
	}
}

func TestInMemoryRepositoryUpdateSettingsOnNewChat(t *testing.T) {
	repo := NewInMemoryRepository()

	pin := true
	err := repo.UpdateSettings(context.Background(), "alice", "bob", UpdateChatSettings{IsPinned: &pin})
	if err != nil {
		t.Fatalf("UpdateSettings on new chat returned error: %v", err)
	}
	state, _ := repo.GetState(context.Background(), "alice", "bob")
	if !state.IsPinned {
		t.Fatal("expected IsPinned=true on newly created chat")
	}
}
