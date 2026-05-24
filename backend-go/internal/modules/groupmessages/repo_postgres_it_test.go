package groupmessages

import (
	"context"
	"testing"

	groups "hotspot/backend-go/internal/modules/groups"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/testdb"
)

func createTestGroup(t *testing.T, ctx context.Context, txFactory *persistence.Factory, ownerID, name string) string {
	t.Helper()
	groupRepo := groups.NewPostgresRepository(txFactory)
	group, err := groupRepo.Create(ctx, ownerID, groups.CreateGroupRequest{
		Name:       name,
		JoinMethod: groups.GroupJoinMethodOpen,
		Location: groups.Location{LocationName: "Beijing", Point: struct {
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
		}{Latitude: 39.9, Longitude: 116.4}},
	})
	if err != nil {
		t.Fatalf("create group: %v", err)
	}
	return group.Group.GroupID
}

func joinTestGroup(t *testing.T, ctx context.Context, txFactory *persistence.Factory, userID, groupID string) {
	t.Helper()
	groupRepo := groups.NewPostgresRepository(txFactory)
	if _, err := groupRepo.JoinGroup(ctx, userID, groupID, nil, nil); err != nil {
		t.Fatalf("join group: %v", err)
	}
}

func TestPostgresIT_SendAndGetHistory(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		owner := testdb.InsertUser(t, txFactory, "gm_owner_1", "Owner")
		groupID := createTestGroup(t, ctx, txFactory, owner, "GM Test Group")
		repo := NewPostgresRepository(txFactory)

		sent, err := repo.Send(ctx, SendGroupMessageRequest{
			GroupID:      groupID,
			SenderUserID: owner,
			Type:         "text",
			Content:      "hello group",
		})
		if err != nil {
			t.Fatalf("Send: %v", err)
		}
		if sent.MessageID == "" {
			t.Fatal("expected non-empty message_id")
		}

		history, err := repo.History(ctx, groupID, nil)
		if err != nil {
			t.Fatalf("History: %v", err)
		}
		if len(history.Items) != 1 {
			t.Fatalf("expected 1 message, got %d", len(history.Items))
		}
		if history.Items[0].Content != "hello group" {
			t.Fatalf("content: got %q", history.Items[0].Content)
		}
	})
}

func TestPostgresIT_UpdateMessage(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		owner := testdb.InsertUser(t, txFactory, "gm_update_owner", "Owner")
		groupID := createTestGroup(t, ctx, txFactory, owner, "GM Update Group")
		repo := NewPostgresRepository(txFactory)

		sent, _ := repo.Send(ctx, SendGroupMessageRequest{
			GroupID:      groupID,
			SenderUserID: owner,
			Type:         "text",
			Content:      "original",
		})

		_, err := repo.Update(ctx, UpdateGroupMessageRequest{
			MessageID: sent.MessageID,
			UserID:    owner,
			Content:   "edited",
		})
		if err != nil {
			t.Fatalf("Update: %v", err)
		}

		history, _ := repo.History(ctx, groupID, nil)
		if len(history.Items) != 1 || history.Items[0].Content != "edited" {
			t.Fatalf("expected edited content, got %#v", history.Items)
		}
	})
}

func TestPostgresIT_RecallMessage(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		owner := testdb.InsertUser(t, txFactory, "gm_recall_owner", "Owner")
		groupID := createTestGroup(t, ctx, txFactory, owner, "GM Recall Group")
		repo := NewPostgresRepository(txFactory)

		sent, _ := repo.Send(ctx, SendGroupMessageRequest{
			GroupID:      groupID,
			SenderUserID: owner,
			Type:         "text",
			Content:      "to recall",
		})

		_, err := repo.Recall(ctx, RecallGroupMessageRequest{
			MessageID: sent.MessageID,
			UserID:    owner,
		})
		if err != nil {
			t.Fatalf("Recall: %v", err)
		}

		history, _ := repo.History(ctx, groupID, nil)
		if len(history.Items) != 1 {
			t.Fatalf("expected 1 message, got %d", len(history.Items))
		}
		if history.Items[0].RecalledAt == "" {
			t.Fatal("expected recalled_at to be set")
		}
	})
}

func TestPostgresIT_IsMember(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		owner := testdb.InsertUser(t, txFactory, "gm_member_owner", "Owner")
		outsider := testdb.InsertUser(t, txFactory, "gm_member_outside", "Outsider")
		groupID := createTestGroup(t, ctx, txFactory, owner, "GM Member Group")
		repo := NewPostgresRepository(txFactory)

		isMember, err := repo.IsMember(ctx, groupID, owner)
		if err != nil {
			t.Fatalf("IsMember owner: %v", err)
		}
		if !isMember {
			t.Fatal("expected owner to be member")
		}

		isMember, err = repo.IsMember(ctx, groupID, outsider)
		if err != nil {
			t.Fatalf("IsMember outsider: %v", err)
		}
		if isMember {
			t.Fatal("expected outsider to not be member")
		}
	})
}

func TestPostgresIT_MarkAsReadToLatest(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		owner := testdb.InsertUser(t, txFactory, "gm_read_owner", "Owner")
		member := testdb.InsertUser(t, txFactory, "gm_read_member", "Member")
		groupID := createTestGroup(t, ctx, txFactory, owner, "GM Read Group")
		joinTestGroup(t, ctx, txFactory, member, groupID)
		repo := NewPostgresRepository(txFactory)

		// Owner sends two messages
		_, err := repo.Send(ctx, SendGroupMessageRequest{
			GroupID: groupID, SenderUserID: owner, Type: "text", Content: "msg1",
		})
		if err != nil {
			t.Fatalf("Send msg1: %v", err)
		}
		_, err = repo.Send(ctx, SendGroupMessageRequest{
			GroupID: groupID, SenderUserID: owner, Type: "text", Content: "msg2",
		})
		if err != nil {
			t.Fatalf("Send msg2: %v", err)
		}

		// Member marks all as read (no UpToMessageID → marks to latest)
		if err := repo.MarkAsRead(ctx, member, MarkGroupMessagesReadRequest{GroupID: groupID}); err != nil {
			t.Fatalf("MarkAsRead to latest: %v", err)
		}
	})
}

func TestPostgresIT_MarkAsReadUpToMessage(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		owner := testdb.InsertUser(t, txFactory, "gm_readup_owner", "Owner")
		member := testdb.InsertUser(t, txFactory, "gm_readup_member", "Member")
		groupID := createTestGroup(t, ctx, txFactory, owner, "GM ReadUp Group")
		joinTestGroup(t, ctx, txFactory, member, groupID)
		repo := NewPostgresRepository(txFactory)

		sent, err := repo.Send(ctx, SendGroupMessageRequest{
			GroupID: groupID, SenderUserID: owner, Type: "text", Content: "target msg",
		})
		if err != nil {
			t.Fatalf("Send: %v", err)
		}

		if err := repo.MarkAsRead(ctx, member, MarkGroupMessagesReadRequest{
			GroupID:       groupID,
			UpToMessageID: sent.MessageID,
		}); err != nil {
			t.Fatalf("MarkAsRead up to message: %v", err)
		}
	})
}

func TestPostgresIT_MarkAllReadAcrossGroups(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		owner := testdb.InsertUser(t, txFactory, "gm_mall_owner", "Owner")
		member := testdb.InsertUser(t, txFactory, "gm_mall_member", "Member")
		group1 := createTestGroup(t, ctx, txFactory, owner, "GM All Group 1")
		group2 := createTestGroup(t, ctx, txFactory, owner, "GM All Group 2")
		joinTestGroup(t, ctx, txFactory, member, group1)
		joinTestGroup(t, ctx, txFactory, member, group2)
		repo := NewPostgresRepository(txFactory)

		_, err := repo.Send(ctx, SendGroupMessageRequest{
			GroupID: group1, SenderUserID: owner, Type: "text", Content: "g1 msg",
		})
		if err != nil {
			t.Fatalf("Send to g1: %v", err)
		}
		_, err = repo.Send(ctx, SendGroupMessageRequest{
			GroupID: group2, SenderUserID: owner, Type: "text", Content: "g2 msg",
		})
		if err != nil {
			t.Fatalf("Send to g2: %v", err)
		}

		if err := repo.MarkAllRead(ctx, member); err != nil {
			t.Fatalf("MarkAllRead: %v", err)
		}
	})
}
