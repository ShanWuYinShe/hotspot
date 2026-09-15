package privatemessages

import (
	"context"
	"testing"
	"time"

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
		if *history.Items[0].Content != "hello bob" {
			t.Fatalf("content: got %q", *history.Items[0].Content)
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
		if len(history.Items) != 1 || *history.Items[0].Content != "edited" {
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

		_, err = repo.Recall(ctx, alice, 2*time.Minute, RecallPrivateMessageRequest{MessageID: sent.MessageID})
		if err != nil {
			t.Fatalf("Recall: %v", err)
		}

		history, _ := repo.History(ctx, alice, ListPrivateMessagesRequest{TargetUserID: bob})
		if len(history.Items) != 1 {
			t.Fatalf("expected 1 message after recall, got %d", len(history.Items))
		}
		if history.Items[0].RecalledAt == nil {
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

// TestPostgresIT_BatchLastAndUnreadByChatKeys 验证会话列表批量读（审计 perf
// H-1/H-2）：DISTINCT ON 取每会话最新一条、GROUP BY 批量未读、VALUES 边界
// 下推（visibleBoundary 过滤）三者的 SQL 语义。
func TestPostgresIT_BatchLastAndUnreadByChatKeys(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		alice := testdb.InsertUser(t, txFactory, "pm_bat_alice", "Alice")
		bob := testdb.InsertUser(t, txFactory, "pm_bat_bob", "Bob")
		carol := testdb.InsertUser(t, txFactory, "pm_bat_carol", "Carol")
		repo := NewPostgresRepository(txFactory)

		if _, err := repo.Send(ctx, bob, alice, SendPrivateMessageRequest{TargetUserID: alice, Type: "text", Content: "first"}); err != nil {
			t.Fatalf("Send first: %v", err)
		}
		if _, err := repo.Send(ctx, bob, alice, SendPrivateMessageRequest{TargetUserID: alice, Type: "text", Content: "second"}); err != nil {
			t.Fatalf("Send second: %v", err)
		}
		if _, err := repo.Send(ctx, carol, alice, SendPrivateMessageRequest{TargetUserID: alice, Type: "text", Content: "hi carol"}); err != nil {
			t.Fatalf("Send carol: %v", err)
		}

		keyBob := conversationKey(alice, bob)
		keyCarol := conversationKey(alice, carol)
		keys := []string{keyBob, keyCarol}

		lasts, err := repo.LastByChatKeys(ctx, alice, keys, nil)
		if err != nil {
			t.Fatalf("LastByChatKeys: %v", err)
		}
		if len(lasts) != 2 {
			t.Fatalf("expected 2 chats, got %d: %#v", len(lasts), lasts)
		}
		if got := *lasts[keyBob].Content; got != "second" {
			t.Fatalf("last of bob chat: got %q, want second", got)
		}
		if lasts[keyBob].Sender.UserID != bob {
			t.Fatalf("sender of bob chat: got %q", lasts[keyBob].Sender.UserID)
		}
		if got := *lasts[keyCarol].Content; got != "hi carol" {
			t.Fatalf("last of carol chat: got %q", got)
		}

		unreads, err := repo.UnreadCountsByChatKeys(ctx, alice, keys, nil)
		if err != nil {
			t.Fatalf("UnreadCountsByChatKeys: %v", err)
		}
		if unreads[keyBob] != 2 || unreads[keyCarol] != 1 {
			t.Fatalf("unread: bob=%d carol=%d, want 2/1", unreads[keyBob], unreads[keyCarol])
		}

		// 边界下推：未来时间作可见性下界 → 全部消息被过滤，两个会话都无末条。
		future := time.Now().UTC().Add(time.Hour)
		boundaries := map[string]*time.Time{keyBob: &future, keyCarol: &future}
		lasts, err = repo.LastByChatKeys(ctx, alice, keys, boundaries)
		if err != nil {
			t.Fatalf("LastByChatKeys(boundary): %v", err)
		}
		if len(lasts) != 0 {
			t.Fatalf("boundary 过滤后不应有末条, got %#v", lasts)
		}
		unreads, err = repo.UnreadCountsByChatKeys(ctx, alice, keys, boundaries)
		if err != nil {
			t.Fatalf("UnreadCountsByChatKeys(boundary): %v", err)
		}
		if len(unreads) != 0 {
			t.Fatalf("boundary 过滤后不应有未读, got %#v", unreads)
		}

		// Recent 的 limit 下推：只取 1 条且不为 history 默认页截断。
		recent, err := repo.Recent(ctx, alice, bob, 1, nil)
		if err != nil {
			t.Fatalf("Recent: %v", err)
		}
		if len(recent.Items) != 1 || *recent.Items[0].Content != "second" {
			t.Fatalf("Recent limit=1: %#v", recent.Items)
		}
	})
}
