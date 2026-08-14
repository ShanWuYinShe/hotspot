package privatemessages

import (
	"context"
	"testing"

	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/testdb"
)

func TestPostgresIT_SendAndGetHistory(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		alice := testdb.InsertUser(t, txFactory, "pm_alice", "Alice")
		bob := testdb.InsertUser(t, txFactory, "pm_bob", "Bob")
		repo := NewPostgresRepository(txFactory)

		sent, err := repo.Send(ctx, alice, bob, SendPrivateMessageRequest{
			TargetUserID: bob,
			Type:         "text",
			Content:      "hello bob",
		})
		if err != nil {
			t.Fatalf("Send: %v", err)
		}
		if sent.MessageID == "" {
			t.Fatal("expected non-empty message_id")
		}

		history, err := repo.History(ctx, alice, ListPrivateMessagesRequest{TargetUserID: bob})
		if err != nil {
			t.Fatalf("History: %v", err)
		}
		if len(history.Items) != 1 {
			t.Fatalf("expected 1 message, got %d", len(history.Items))
		}
		if history.Items[0].Content != "hello bob" {
			t.Fatalf("content: got %q", history.Items[0].Content)
		}
	})
}

func TestPostgresIT_UpdateMessage(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		alice := testdb.InsertUser(t, txFactory, "pm_update_alice", "Alice")
		bob := testdb.InsertUser(t, txFactory, "pm_update_bob", "Bob")
		repo := NewPostgresRepository(txFactory)

		sent, err := repo.Send(ctx, alice, bob, SendPrivateMessageRequest{
			TargetUserID: bob,
			Type:         "text",
			Content:      "original",
		})
		if err != nil {
			t.Fatalf("Send: %v", err)
		}

		meta, err := repo.Update(ctx, alice, UpdatePrivateMessageRequest{
			MessageID: sent.MessageID,
			Content:   "edited",
		})
		if err != nil {
			t.Fatalf("Update: %v", err)
		}
		if meta.MessageID != sent.MessageID {
			t.Fatalf("message_id: got %q, want %q", meta.MessageID, sent.MessageID)
		}

		history, _ := repo.History(ctx, alice, ListPrivateMessagesRequest{TargetUserID: bob})
		if len(history.Items) != 1 || history.Items[0].Content != "edited" {
			t.Fatalf("expected edited content, got %#v", history.Items)
		}
	})
}

func TestPostgresIT_RecallMessage(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		alice := testdb.InsertUser(t, txFactory, "pm_recall_alice", "Alice")
		bob := testdb.InsertUser(t, txFactory, "pm_recall_bob", "Bob")
		repo := NewPostgresRepository(txFactory)

		sent, err := repo.Send(ctx, alice, bob, SendPrivateMessageRequest{
			TargetUserID: bob,
			Type:         "text",
			Content:      "to recall",
		})
		if err != nil {
			t.Fatalf("Send: %v", err)
		}

		_, err = repo.Recall(ctx, alice, RecallPrivateMessageRequest{MessageID: sent.MessageID})
		if err != nil {
			t.Fatalf("Recall: %v", err)
		}

		history, _ := repo.History(ctx, alice, ListPrivateMessagesRequest{TargetUserID: bob})
		if len(history.Items) != 1 {
			t.Fatalf("expected 1 message after recall, got %d", len(history.Items))
		}
		if history.Items[0].RecalledAt == "" {
			t.Fatal("expected recalled_at to be set")
		}
	})
}

func TestPostgresIT_MarkAsRead(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		alice := testdb.InsertUser(t, txFactory, "pm_read_alice", "Alice")
		bob := testdb.InsertUser(t, txFactory, "pm_read_bob", "Bob")
		repo := NewPostgresRepository(txFactory)

		sent, err := repo.Send(ctx, alice, bob, SendPrivateMessageRequest{
			TargetUserID: bob,
			Type:         "text",
			Content:      "hello",
		})
		if err != nil {
			t.Fatalf("Send: %v", err)
		}

		_, err = repo.MarkAsRead(ctx, bob, MarkPrivateMessagesReadRequest{
			TargetUserID:  alice,
			UpToMessageID: sent.MessageID,
		})
		if err != nil {
			t.Fatalf("MarkAsRead: %v", err)
		}
	})
}

func TestPostgresIT_MarkAsReadWithoutMessageID(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		alice := testdb.InsertUser(t, txFactory, "pm_readall_alice", "Alice")
		bob := testdb.InsertUser(t, txFactory, "pm_readall_bob", "Bob")
		repo := NewPostgresRepository(txFactory)

		_, err := repo.Send(ctx, alice, bob, SendPrivateMessageRequest{
			TargetUserID: bob,
			Type:         "text",
			Content:      "msg1",
		})
		if err != nil {
			t.Fatalf("Send msg1: %v", err)
		}
		_, err = repo.Send(ctx, alice, bob, SendPrivateMessageRequest{
			TargetUserID: bob,
			Type:         "text",
			Content:      "msg2",
		})
		if err != nil {
			t.Fatalf("Send msg2: %v", err)
		}

		// MarkAsRead without UpToMessageID → marks all in this chat as read
		_, err = repo.MarkAsRead(ctx, bob, MarkPrivateMessagesReadRequest{
			TargetUserID: alice,
		})
		if err != nil {
			t.Fatalf("MarkAsRead (all in chat): %v", err)
		}
	})
}

func TestPostgresIT_MarkAllReadAcrossChats(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		alice := testdb.InsertUser(t, txFactory, "pm_mall_alice", "Alice")
		bob := testdb.InsertUser(t, txFactory, "pm_mall_bob", "Bob")
		carol := testdb.InsertUser(t, txFactory, "pm_mall_carol", "Carol")
		repo := NewPostgresRepository(txFactory)

		// Two separate chats: alice-bob and alice-carol
		_, err := repo.Send(ctx, alice, bob, SendPrivateMessageRequest{
			TargetUserID: bob,
			Type:         "text",
			Content:      "hi bob",
		})
		if err != nil {
			t.Fatalf("Send to bob: %v", err)
		}
		_, err = repo.Send(ctx, alice, carol, SendPrivateMessageRequest{
			TargetUserID: carol,
			Type:         "text",
			Content:      "hi carol",
		})
		if err != nil {
			t.Fatalf("Send to carol: %v", err)
		}

		// MarkAllRead across all chats for alice
		if err := repo.MarkAllRead(ctx, alice); err != nil {
			t.Fatalf("MarkAllRead: %v", err)
		}
	})
}
