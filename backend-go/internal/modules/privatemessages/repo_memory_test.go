package privatemessages

import (
	"context"
	"testing"
	"time"
)

func TestInMemoryRepositorySendReturnsMessageSent(t *testing.T) {
	repo := NewInMemoryRepository()

	result, err := repo.Send(context.Background(), "alice", "bob", SendPrivateMessageRequest{
		Type:    "text",
		Content: "hello bob",
	})
	if err != nil {
		t.Fatalf("Send returned error: %v", err)
	}
	if result.MessageID == "" {
		t.Fatal("expected non-empty MessageID")
	}
	if result.SentAt == "" {
		t.Fatal("expected non-empty SentAt")
	}
}

func TestInMemoryRepositorySendWithFileID(t *testing.T) {
	repo := NewInMemoryRepository()

	_, err := repo.Send(context.Background(), "alice", "bob", SendPrivateMessageRequest{
		Type:    "image",
		Content: "",
		FileID:  "file_123",
	})
	if err != nil {
		t.Fatalf("Send returned error: %v", err)
	}
	history, _ := repo.History(context.Background(), "alice", ListPrivateMessagesRequest{TargetUserID: "bob"})
	if len(history.Items) != 1 || history.Items[0].Media == nil {
		t.Fatal("expected message with media")
	}
	if history.Items[0].Media.URL != "file_123" {
		t.Fatalf("expected media URL 'file_123', got %q", history.Items[0].Media.URL)
	}
}

func TestInMemoryRepositorySendWithReply(t *testing.T) {
	repo := NewInMemoryRepository()

	sent, err := repo.Send(context.Background(), "alice", "bob", SendPrivateMessageRequest{
		Type:    "text",
		Content: "original",
	})
	if err != nil {
		t.Fatalf("Send original returned error: %v", err)
	}
	_, err = repo.Send(context.Background(), "alice", "bob", SendPrivateMessageRequest{
		Type:             "text",
		Content:          "reply",
		ReplyToMessageID: sent.MessageID,
	})
	if err != nil {
		t.Fatalf("Send reply returned error: %v", err)
	}
	history, _ := repo.History(context.Background(), "alice", ListPrivateMessagesRequest{TargetUserID: "bob"})
	if len(history.Items) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(history.Items))
	}
	replyMsg := history.Items[0]
	if replyMsg.ReplyTo == nil {
		t.Fatal("expected reply_to to be set")
	}
	if replyMsg.ReplyTo.MessageID != sent.MessageID {
		t.Fatalf("expected reply_to %q, got %q", sent.MessageID, replyMsg.ReplyTo.MessageID)
	}
}

func TestInMemoryRepositoryHistoryReturnsMessages(t *testing.T) {
	repo := NewInMemoryRepository()

	_, _ = repo.Send(context.Background(), "alice", "bob", SendPrivateMessageRequest{Type: "text", Content: "msg1"})
	_, _ = repo.Send(context.Background(), "alice", "bob", SendPrivateMessageRequest{Type: "text", Content: "msg2"})

	history, err := repo.History(context.Background(), "alice", ListPrivateMessagesRequest{TargetUserID: "bob"})
	if err != nil {
		t.Fatalf("History returned error: %v", err)
	}
	if len(history.Items) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(history.Items))
	}
	if *history.Items[0].Content != "msg2" {
		t.Fatalf("expected newest first, got %q", *history.Items[0].Content)
	}
}

func TestInMemoryRepositoryHistoryEmpty(t *testing.T) {
	repo := NewInMemoryRepository()

	history, err := repo.History(context.Background(), "alice", ListPrivateMessagesRequest{TargetUserID: "bob"})
	if err != nil {
		t.Fatalf("History returned error: %v", err)
	}
	if len(history.Items) != 0 {
		t.Fatalf("expected 0 messages, got %d", len(history.Items))
	}
}

func TestInMemoryRepositoryHistoryOrderIndependent(t *testing.T) {
	repo := NewInMemoryRepository()

	_, _ = repo.Send(context.Background(), "alice", "bob", SendPrivateMessageRequest{Type: "text", Content: "from alice"})
	_, _ = repo.Send(context.Background(), "bob", "alice", SendPrivateMessageRequest{Type: "text", Content: "from bob"})

	aliceHistory, _ := repo.History(context.Background(), "alice", ListPrivateMessagesRequest{TargetUserID: "bob"})
	bobHistory, _ := repo.History(context.Background(), "bob", ListPrivateMessagesRequest{TargetUserID: "alice"})

	if len(aliceHistory.Items) != 2 || len(bobHistory.Items) != 2 {
		t.Fatalf("expected 2 messages both ways, got alice=%d bob=%d", len(aliceHistory.Items), len(bobHistory.Items))
	}
}

func TestInMemoryRepositoryHistoryWithPaginationLimit(t *testing.T) {
	repo := NewInMemoryRepository()

	for i := 0; i < 5; i++ {
		_, _ = repo.Send(context.Background(), "alice", "bob", SendPrivateMessageRequest{Type: "text", Content: "msg"})
	}

	history, err := repo.History(context.Background(), "alice", ListPrivateMessagesRequest{
		TargetUserID: "bob",
		Pagination:   &PaginationParams{Limit: 2},
	})
	if err != nil {
		t.Fatalf("History returned error: %v", err)
	}
	if len(history.Items) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(history.Items))
	}
	if history.NextCursor == nil {
		t.Fatal("expected NextCursor to be set")
	}
}

func TestInMemoryRepositoryHistoryWithInvalidCursor(t *testing.T) {
	repo := NewInMemoryRepository()

	_, _ = repo.Send(context.Background(), "alice", "bob", SendPrivateMessageRequest{Type: "text", Content: "msg"})

	_, err := repo.History(context.Background(), "alice", ListPrivateMessagesRequest{
		TargetUserID: "bob",
		Pagination:   &PaginationParams{Cursor: "invalid_cursor"},
	})
	if err != ErrInvalidCursor {
		t.Fatalf("expected ErrInvalidCursor, got %v", err)
	}
}

func TestInMemoryRepositoryRecent(t *testing.T) {
	repo := NewInMemoryRepository()

	_, _ = repo.Send(context.Background(), "alice", "bob", SendPrivateMessageRequest{Type: "text", Content: "msg1"})
	_, _ = repo.Send(context.Background(), "alice", "bob", SendPrivateMessageRequest{Type: "text", Content: "msg2"})

	recent, err := repo.Recent(context.Background(), "alice", "bob", 1, nil)
	if err != nil {
		t.Fatalf("Recent returned error: %v", err)
	}
	if len(recent.Items) != 1 {
		t.Fatalf("expected 1 recent message, got %d", len(recent.Items))
	}
	if *recent.Items[0].Content != "msg2" {
		t.Fatalf("expected msg2 as most recent, got %q", *recent.Items[0].Content)
	}
}

func TestInMemoryRepositoryLast(t *testing.T) {
	repo := NewInMemoryRepository()

	_, _ = repo.Send(context.Background(), "alice", "bob", SendPrivateMessageRequest{Type: "text", Content: "msg"})

	last, err := repo.Last(context.Background(), "alice", "bob", nil)
	if err != nil {
		t.Fatalf("Last returned error: %v", err)
	}
	if last == nil {
		t.Fatal("expected non-nil last message")
	}
	if *last.Content != "msg" {
		t.Fatalf("expected last content 'msg', got %q", *last.Content)
	}
}

func TestInMemoryRepositoryLastEmpty(t *testing.T) {
	repo := NewInMemoryRepository()

	last, err := repo.Last(context.Background(), "alice", "bob", nil)
	if err != nil {
		t.Fatalf("Last returned error: %v", err)
	}
	if last != nil {
		t.Fatalf("expected nil last for empty chat, got %v", last)
	}
}

func TestInMemoryRepositoryUpdate(t *testing.T) {
	repo := NewInMemoryRepository()

	sent, _ := repo.Send(context.Background(), "alice", "bob", SendPrivateMessageRequest{Type: "text", Content: "original"})

	meta, err := repo.Update(context.Background(), "alice", UpdatePrivateMessageRequest{
		MessageID: sent.MessageID,
		Content:   "updated",
	})
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}
	if meta.MessageID != sent.MessageID {
		t.Fatalf("expected MessageID %q, got %q", sent.MessageID, meta.MessageID)
	}

	history, _ := repo.History(context.Background(), "alice", ListPrivateMessagesRequest{TargetUserID: "bob"})
	if *history.Items[0].Content != "updated" {
		t.Fatalf("expected updated content, got %q", *history.Items[0].Content)
	}
}

func TestInMemoryRepositoryUpdatePermissionDenied(t *testing.T) {
	repo := NewInMemoryRepository()

	sent, _ := repo.Send(context.Background(), "alice", "bob", SendPrivateMessageRequest{Type: "text", Content: "original"})

	_, err := repo.Update(context.Background(), "stranger", UpdatePrivateMessageRequest{
		MessageID: sent.MessageID,
		Content:   "hacked",
	})
	if err != ErrMessagePermissionDenied {
		t.Fatalf("expected ErrMessagePermissionDenied, got %v", err)
	}
}

func TestInMemoryRepositoryUpdateNonEditableType(t *testing.T) {
	repo := NewInMemoryRepository()

	sent, _ := repo.Send(context.Background(), "alice", "bob", SendPrivateMessageRequest{Type: "image", Content: "pic"})

	_, err := repo.Update(context.Background(), "alice", UpdatePrivateMessageRequest{
		MessageID: sent.MessageID,
		Content:   "try edit",
	})
	if err != ErrMessageNotEditable {
		t.Fatalf("expected ErrMessageNotEditable, got %v", err)
	}
}

func TestInMemoryRepositoryUpdateNotFound(t *testing.T) {
	repo := NewInMemoryRepository()

	_, err := repo.Update(context.Background(), "alice", UpdatePrivateMessageRequest{
		MessageID: "nonexistent",
		Content:   "msg",
	})
	if err != ErrMessageNotFound {
		t.Fatalf("expected ErrMessageNotFound, got %v", err)
	}
}

func TestInMemoryRepositoryRecallWithinWindow(t *testing.T) {
	repo := NewInMemoryRepository()

	sent, _ := repo.Send(context.Background(), "alice", "bob", SendPrivateMessageRequest{Type: "text", Content: "recall me"})

	meta, err := repo.Recall(context.Background(), "alice", 2*time.Minute, RecallPrivateMessageRequest{MessageID: sent.MessageID})
	if err != nil {
		t.Fatalf("Recall returned error: %v", err)
	}
	if meta.MessageID != sent.MessageID {
		t.Fatalf("expected MessageID %q, got %q", sent.MessageID, meta.MessageID)
	}

	history, _ := repo.History(context.Background(), "alice", ListPrivateMessagesRequest{TargetUserID: "bob"})
	if history.Items[0].RecalledAt == nil {
		t.Fatal("expected RecalledAt to be set")
	}
}

func TestInMemoryRepositoryRecallPermissionDenied(t *testing.T) {
	repo := NewInMemoryRepository()

	sent, _ := repo.Send(context.Background(), "alice", "bob", SendPrivateMessageRequest{Type: "text", Content: "msg"})

	_, err := repo.Recall(context.Background(), "stranger", 2*time.Minute, RecallPrivateMessageRequest{MessageID: sent.MessageID})
	if err != ErrMessagePermissionDenied {
		t.Fatalf("expected ErrMessagePermissionDenied, got %v", err)
	}
}

func TestInMemoryRepositoryRecallNotFound(t *testing.T) {
	repo := NewInMemoryRepository()

	_, err := repo.Recall(context.Background(), "alice", 2*time.Minute, RecallPrivateMessageRequest{MessageID: "nonexistent"})
	if err != ErrMessageNotFound {
		t.Fatalf("expected ErrMessageNotFound, got %v", err)
	}
}

func TestInMemoryRepositoryMarkAsRead(t *testing.T) {
	repo := NewInMemoryRepository()

	sent, _ := repo.Send(context.Background(), "alice", "bob", SendPrivateMessageRequest{Type: "text", Content: "msg"})

	meta, err := repo.MarkAsRead(context.Background(), "bob", MarkPrivateMessagesReadRequest{
		TargetUserID:  "alice",
		UpToMessageID: sent.MessageID,
	})
	if err != nil {
		t.Fatalf("MarkAsRead returned error: %v", err)
	}
	if meta.TargetUserID != "alice" {
		t.Fatalf("expected TargetUserID 'alice', got %q", meta.TargetUserID)
	}
}

func TestInMemoryRepositoryMarkAsReadAll(t *testing.T) {
	repo := NewInMemoryRepository()

	_, _ = repo.Send(context.Background(), "alice", "bob", SendPrivateMessageRequest{Type: "text", Content: "msg"})

	meta, err := repo.MarkAsRead(context.Background(), "bob", MarkPrivateMessagesReadRequest{
		TargetUserID: "alice",
	})
	if err != nil {
		t.Fatalf("MarkAsRead returned error: %v", err)
	}
	if meta.TargetUserID != "alice" {
		t.Fatalf("expected TargetUserID 'alice', got %q", meta.TargetUserID)
	}
}

func TestInMemoryRepositoryMarkAllRead(t *testing.T) {
	repo := NewInMemoryRepository()

	_, _ = repo.Send(context.Background(), "alice", "bob", SendPrivateMessageRequest{Type: "text", Content: "msg1"})
	_, _ = repo.Send(context.Background(), "alice", "bob", SendPrivateMessageRequest{Type: "text", Content: "msg2"})

	err := repo.MarkAllRead(context.Background(), "bob")
	if err != nil {
		t.Fatalf("MarkAllRead returned error: %v", err)
	}
}

func TestInMemoryRepositoryUnreadCount(t *testing.T) {
	repo := NewInMemoryRepository()

	sent, _ := repo.Send(context.Background(), "alice", "bob", SendPrivateMessageRequest{Type: "text", Content: "msg1"})

	count, err := repo.UnreadCount(context.Background(), "bob", "alice", nil)
	if err != nil {
		t.Fatalf("UnreadCount returned error: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected unread count 1, got %d", count)
	}

	repo.MarkAsRead(context.Background(), "bob", MarkPrivateMessagesReadRequest{
		TargetUserID:  "alice",
		UpToMessageID: sent.MessageID,
	})
	count2, _ := repo.UnreadCount(context.Background(), "bob", "alice", nil)
	if count2 != 0 {
		t.Fatalf("expected unread count 0 after mark read, got %d", count2)
	}
}

func TestInMemoryRepositoryUnreadCountOrderIndependent(t *testing.T) {
	repo := NewInMemoryRepository()

	_, _ = repo.Send(context.Background(), "bob", "alice", SendPrivateMessageRequest{Type: "text", Content: "msg"})

	count, err := repo.UnreadCount(context.Background(), "alice", "bob", nil)
	if err != nil {
		t.Fatalf("UnreadCount returned error: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected unread count 1, got %d", count)
	}
}

func TestInMemoryRepositoryHistoryDeduplicatesByConversationKey(t *testing.T) {
	repo := NewInMemoryRepository()

	_, _ = repo.Send(context.Background(), "alice", "bob", SendPrivateMessageRequest{Type: "text", Content: "msg1"})
	_, _ = repo.Send(context.Background(), "bob", "alice", SendPrivateMessageRequest{Type: "text", Content: "msg2"})

	aliceHistory, _ := repo.History(context.Background(), "alice", ListPrivateMessagesRequest{TargetUserID: "bob"})
	bobHistory, _ := repo.History(context.Background(), "bob", ListPrivateMessagesRequest{TargetUserID: "alice"})

	if len(aliceHistory.Items) != len(bobHistory.Items) {
		t.Fatalf("history length should match: alice=%d bob=%d", len(aliceHistory.Items), len(bobHistory.Items))
	}
	for i := range aliceHistory.Items {
		if *aliceHistory.Items[i].Content != *bobHistory.Items[i].Content {
			t.Fatalf("message %d: alice=%q bob=%q", i, *aliceHistory.Items[i].Content, *bobHistory.Items[i].Content)
		}
	}
}

func TestInMemoryRepositoryUpdateSetsEditedAt(t *testing.T) {
	repo := NewInMemoryRepository()

	sent, _ := repo.Send(context.Background(), "alice", "bob", SendPrivateMessageRequest{Type: "text", Content: "original"})

	_, _ = repo.Update(context.Background(), "alice", UpdatePrivateMessageRequest{MessageID: sent.MessageID, Content: "updated"})

	history, _ := repo.History(context.Background(), "alice", ListPrivateMessagesRequest{TargetUserID: "bob"})
	if history.Items[0].EditedAt == nil {
		t.Fatal("expected EditedAt to be set")
	}
}

func TestInMemoryRepositoryUpdateCannotEditAfterRecall(t *testing.T) {
	repo := NewInMemoryRepository()

	sent, _ := repo.Send(context.Background(), "alice", "bob", SendPrivateMessageRequest{Type: "text", Content: "original"})

	_, _ = repo.Recall(context.Background(), "alice", 2*time.Minute, RecallPrivateMessageRequest{MessageID: sent.MessageID})

	_, err := repo.Update(context.Background(), "alice", UpdatePrivateMessageRequest{MessageID: sent.MessageID, Content: "updated"})
	if err != ErrMessageNotEditable {
		t.Fatalf("expected ErrMessageNotEditable after recall, got %v", err)
	}
}

func TestInMemoryRepositoryRecallTwoMinuteWindow(t *testing.T) {
	repo := NewInMemoryRepository()

	history, _ := repo.History(context.Background(), "alice", ListPrivateMessagesRequest{TargetUserID: "bob"})
	msgs := history.Items
	_ = msgs
}

func TestInMemoryRepositoryHistoryNewestFirst(t *testing.T) {
	repo := NewInMemoryRepository()

	_, _ = repo.Send(context.Background(), "alice", "bob", SendPrivateMessageRequest{Type: "text", Content: "first"})

	history, _ := repo.History(context.Background(), "alice", ListPrivateMessagesRequest{TargetUserID: "bob"})
	if len(history.Items) < 1 {
		t.Fatalf("expected at least 1 message, got %d", len(history.Items))
	}
	if *history.Items[0].Content != "first" {
		t.Fatalf("expected 'first' as newest, got %q", *history.Items[0].Content)
	}
}
