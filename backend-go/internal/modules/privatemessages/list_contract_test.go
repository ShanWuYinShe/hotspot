package privatemessages

import (
	"context"
	"errors"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/testdb"
)

// pmContractState 契约用例上下文：PG 侧 private_messages/private_chats 有 users
// 外键，需预置用户行；内存实现无需。
type pmContractState struct {
	ctx context.Context
	a   string // 发送方
	b   string // 接收方
	c   string // 第三方（跨会话回复/计数用例）
}

// TestPrivateMessagesRepositoryContract 接入双实现契约一致性基建：
// 发送/历史/最近一条/未读数/已读状态机/撤回/批量读/配额计数 的语义必须与真实库
// 实现一致。媒体回填（file_id→storage_key）依赖 media 表与存储侧数据，此处不覆盖。
func TestPrivateMessagesRepositoryContract(t *testing.T) {
	a, b, c := "it_pm_contract_a", "it_pm_contract_b", "it_pm_contract_c"

	impls := []testdb.ContractImpl[Repository, pmContractState]{
		{
			Name: "memory",
			New: func(t *testing.T) (Repository, pmContractState, func()) {
				return NewInMemoryRepository(), pmContractState{ctx: context.Background(), a: a, b: b, c: c}, nil
			},
		},
		{
			Name: "postgres",
			New: func(t *testing.T) (Repository, pmContractState, func()) {
				testdb.SkipIfShort(t)
				testdb.Open(t)
				txFactory, cleanup := testdb.BeginScope(t)
				for _, uid := range []string{a, b, c} {
					testdb.InsertUser(t, txFactory, uid, "nick_"+uid)
				}
				return NewPostgresRepository(txFactory), pmContractState{ctx: context.Background(), a: a, b: b, c: c}, cleanup
			},
		},
	}

	send := func(t *testing.T, repo Repository, from, to, content string) MessageSentResult {
		t.Helper()
		result, err := repo.Send(context.Background(), from, to, SendPrivateMessageRequest{TargetUserID: to, Type: "text", Content: content})
		if err != nil {
			t.Fatalf("send %s->%s: %v", from, to, err)
		}
		if result.MessageID == "" {
			t.Fatal("sent message id must be non-empty")
		}
		return result
	}

	testdb.RunContract(t, impls, []testdb.ContractCase[Repository, pmContractState]{
		{
			Name: "send and history reflect both views",
			Run: func(t *testing.T, repo Repository, state pmContractState) {
				first := send(t, repo, state.a, state.b, "hello 1")
				second := send(t, repo, state.a, state.b, "hello 2")

				for _, viewer := range []string{state.a, state.b} {
					page, err := repo.History(state.ctx, viewer, ListPrivateMessagesRequest{TargetUserID: otherOf(viewer, state.a, state.b)})
					if err != nil {
						t.Fatalf("history for %s: %v", viewer, err)
					}
					if len(page.Items) != 2 {
						t.Fatalf("history for %s = %d items", viewer, len(page.Items))
					}
					if page.Items[0].MessageID != second.MessageID || page.Items[1].MessageID != first.MessageID {
						t.Fatalf("history order for %s = %#v", viewer, page.Items)
					}
					if page.NextCursor != nil {
						t.Fatalf("unexpected next cursor for %s", viewer)
					}
					for _, item := range page.Items {
						if item.Type != "text" {
							t.Fatalf("type = %q", item.Type)
						}
						if item.Sender.UserID != state.a {
							t.Fatalf("sender = %#v", item.Sender)
						}
						if _, err := time.Parse(time.RFC3339Nano, item.SentAt); err != nil {
							t.Fatalf("sent_at must be RFC3339Nano: %q", item.SentAt)
						}
					}
					if page.Items[0].Content == nil || *page.Items[0].Content != "hello 2" {
						t.Fatalf("content = %#v", page.Items[0].Content)
					}
				}
			},
		},
		{
			Name: "history pagination is stable and disjoint",
			Run: func(t *testing.T, repo Repository, state pmContractState) {
				for i := 0; i < 5; i++ {
					send(t, repo, state.a, state.b, "page message")
				}
				seen := map[string]bool{}
				cursor := ""
				pages := 0
				for {
					page, err := repo.History(state.ctx, state.b, ListPrivateMessagesRequest{
						TargetUserID: state.a,
						Pagination:   &PaginationParams{Limit: 2, Cursor: cursor},
					})
					if err != nil {
						t.Fatalf("page %d: %v", pages, err)
					}
					if len(page.Items) > 2 {
						t.Fatalf("page %d size = %d", pages, len(page.Items))
					}
					for _, item := range page.Items {
						if seen[item.MessageID] {
							t.Fatalf("duplicate message across pages: %s", item.MessageID)
						}
						seen[item.MessageID] = true
					}
					pages++
					if page.NextCursor == nil {
						break
					}
					cursor = *page.NextCursor
					if pages > 6 {
						t.Fatal("pagination did not terminate")
					}
				}
				if len(seen) != 5 {
					t.Fatalf("seen %d messages, want 5", len(seen))
				}
				if _, err := repo.History(state.ctx, state.b, ListPrivateMessagesRequest{TargetUserID: state.a, Pagination: &PaginationParams{Cursor: "not-a-cursor"}}); !errors.Is(err, ErrInvalidCursor) {
					t.Fatalf("invalid cursor err = %v, want ErrInvalidCursor", err)
				}
			},
		},
		{
			Name: "recent honors limit and last returns newest",
			Run: func(t *testing.T, repo Repository, state pmContractState) {
				var newest MessageSentResult
				for i := 0; i < 22; i++ {
					newest = send(t, repo, state.a, state.b, "bulk message")
				}
				recent, err := repo.Recent(state.ctx, state.b, state.a, 21, nil)
				if err != nil {
					t.Fatalf("recent: %v", err)
				}
				if len(recent.Items) != 21 {
					t.Fatalf("recent = %d items, want 21 (limit must be pushed down)", len(recent.Items))
				}
				last, err := repo.Last(state.ctx, state.b, state.a, nil)
				if err != nil {
					t.Fatalf("last: %v", err)
				}
				if last == nil || last.MessageID != newest.MessageID {
					t.Fatalf("last = %#v, want %s", last, newest.MessageID)
				}
			},
		},
		{
			Name: "unread count and mark read advance semantics",
			Run: func(t *testing.T, repo Repository, state pmContractState) {
				oldest := send(t, repo, state.a, state.b, "m1")
				send(t, repo, state.a, state.b, "m2")
				newest := send(t, repo, state.a, state.b, "m3")

				unread, err := repo.UnreadCount(state.ctx, state.b, state.a, nil)
				if err != nil {
					t.Fatalf("unread: %v", err)
				}
				if unread != 3 {
					t.Fatalf("unread = %d, want 3", unread)
				}
				if senderUnread, err := repo.UnreadCount(state.ctx, state.a, state.b, nil); err != nil || senderUnread != 0 {
					t.Fatalf("sender unread = %d err=%v, want 0", senderUnread, err)
				}

				meta, err := repo.MarkAsRead(state.ctx, state.b, MarkPrivateMessagesReadRequest{TargetUserID: state.a, UpToMessageID: oldest.MessageID})
				if err != nil {
					t.Fatalf("mark up to oldest: %v", err)
				}
				if meta.MessageID != oldest.MessageID {
					t.Fatalf("mark meta = %#v", meta)
				}
				if unread, err = repo.UnreadCount(state.ctx, state.b, state.a, nil); err != nil || unread != 2 {
					t.Fatalf("unread after marking oldest = %d err=%v, want 2", unread, err)
				}

				if _, err := repo.MarkAsRead(state.ctx, state.b, MarkPrivateMessagesReadRequest{TargetUserID: state.a, UpToMessageID: newest.MessageID}); err != nil {
					t.Fatalf("mark up to newest: %v", err)
				}
				if unread, err = repo.UnreadCount(state.ctx, state.b, state.a, nil); err != nil || unread != 0 {
					t.Fatalf("unread after marking newest = %d err=%v, want 0", unread, err)
				}
				// 已读游标只前进：再次标记更早的消息不得回退游标/增加未读
				if _, err := repo.MarkAsRead(state.ctx, state.b, MarkPrivateMessagesReadRequest{TargetUserID: state.a, UpToMessageID: oldest.MessageID}); err != nil {
					t.Fatalf("mark older again: %v", err)
				}
				if unread, err = repo.UnreadCount(state.ctx, state.b, state.a, nil); err != nil || unread != 0 {
					t.Fatalf("unread after regressing mark = %d err=%v, want 0 (cursor is forward-only)", unread, err)
				}

				// 非目标会话的消息 id → 10300
				foreign := send(t, repo, state.a, state.c, "other chat")
				if _, err := repo.MarkAsRead(state.ctx, state.b, MarkPrivateMessagesReadRequest{TargetUserID: state.a, UpToMessageID: foreign.MessageID}); !errors.Is(err, ErrMessageNotFound) {
					t.Fatalf("foreign up_to err = %v, want ErrMessageNotFound", err)
				}
			},
		},
		{
			Name: "mark read without up_to returns latest message",
			Run: func(t *testing.T, repo Repository, state pmContractState) {
				send(t, repo, state.a, state.b, "one")
				newest := send(t, repo, state.a, state.b, "two")
				meta, err := repo.MarkAsRead(state.ctx, state.b, MarkPrivateMessagesReadRequest{TargetUserID: state.a})
				if err != nil {
					t.Fatalf("mark all in session: %v", err)
				}
				if meta.MessageID != newest.MessageID {
					t.Fatalf("meta.message_id = %q, want latest message %q (WS PrivateMessageRead.up_to_message_id 必填)", meta.MessageID, newest.MessageID)
				}
				if _, err := time.Parse(time.RFC3339Nano, meta.SentAt); err != nil {
					t.Fatalf("meta.sent_at must be RFC3339Nano: %q", meta.SentAt)
				}
				if unread, err := repo.UnreadCount(state.ctx, state.b, state.a, nil); err != nil || unread != 0 {
					t.Fatalf("unread after mark all in session = %d err=%v, want 0", unread, err)
				}
			},
		},
		{
			Name: "history is_read reflects receiver read cursor",
			Run: func(t *testing.T, repo Repository, state pmContractState) {
				send(t, repo, state.a, state.b, "r1")
				newest := send(t, repo, state.a, state.b, "r2")
				senderView, err := repo.History(state.ctx, state.a, ListPrivateMessagesRequest{TargetUserID: state.b})
				if err != nil {
					t.Fatalf("sender history: %v", err)
				}
				for _, item := range senderView.Items {
					if item.IsRead {
						t.Fatalf("sender view must be unread before receiver reads: %#v", item)
					}
				}
				if _, err := repo.MarkAsRead(state.ctx, state.b, MarkPrivateMessagesReadRequest{TargetUserID: state.a, UpToMessageID: newest.MessageID}); err != nil {
					t.Fatalf("mark read: %v", err)
				}
				senderView, err = repo.History(state.ctx, state.a, ListPrivateMessagesRequest{TargetUserID: state.b})
				if err != nil {
					t.Fatalf("sender history after read: %v", err)
				}
				for _, item := range senderView.Items {
					if !item.IsRead {
						t.Fatalf("sender view must be read after receiver read cursor advanced: %#v", item)
					}
				}
				receiverView, err := repo.History(state.ctx, state.b, ListPrivateMessagesRequest{TargetUserID: state.a})
				if err != nil {
					t.Fatalf("receiver history: %v", err)
				}
				for _, item := range receiverView.Items {
					if !item.IsRead {
						t.Fatalf("received messages are read for viewer: %#v", item)
					}
				}
			},
		},
		{
			Name: "recalled messages are not unread",
			Run: func(t *testing.T, repo Repository, state pmContractState) {
				send(t, repo, state.a, state.b, "keep")
				recalled := send(t, repo, state.a, state.b, "recall me")
				if _, err := repo.Recall(state.ctx, state.a, time.Hour, RecallPrivateMessageRequest{MessageID: recalled.MessageID}); err != nil {
					t.Fatalf("recall: %v", err)
				}
				unread, err := repo.UnreadCount(state.ctx, state.b, state.a, nil)
				if err != nil {
					t.Fatalf("unread: %v", err)
				}
				if unread != 1 {
					t.Fatalf("unread = %d, want 1 (recalled message must not count)", unread)
				}
			},
		},
		{
			Name: "invalid reply target is rejected",
			Run: func(t *testing.T, repo Repository, state pmContractState) {
				send(t, repo, state.a, state.b, "base")
				_, err := repo.Send(state.ctx, state.b, state.a, SendPrivateMessageRequest{TargetUserID: state.a, Type: "text", Content: "reply", ReplyToMessageID: "pm_does_not_exist"})
				if !errors.Is(err, ErrMessageNotFound) {
					t.Fatalf("invalid reply target err = %v, want ErrMessageNotFound", err)
				}
			},
		},
		{
			Name: "batch reads match single lookups",
			Run: func(t *testing.T, repo Repository, state pmContractState) {
				send(t, repo, state.a, state.b, "to b 1")
				send(t, repo, state.a, state.b, "to b 2")
				send(t, repo, state.a, state.c, "to c 1")
				keyB := conversationKey(state.b, state.a)
				keyC := conversationKey(state.c, state.a)
				keys := []string{keyB, keyC}

				lastByKey, err := repo.LastByChatKeys(state.ctx, state.b, keys, nil)
				if err != nil {
					t.Fatalf("last by chat keys: %v", err)
				}
				last, err := repo.Last(state.ctx, state.b, state.a, nil)
				if err != nil || last == nil {
					t.Fatalf("last: %#v err=%v", last, err)
				}
				if item, ok := lastByKey[keyB]; !ok || item.MessageID != last.MessageID {
					t.Fatalf("batch last = %#v, single = %s", lastByKey, last.MessageID)
				}

				unreadByKey, err := repo.UnreadCountsByChatKeys(state.ctx, state.b, keys, nil)
				if err != nil {
					t.Fatalf("unread by chat keys: %v", err)
				}
				single, err := repo.UnreadCount(state.ctx, state.b, state.a, nil)
				if err != nil {
					t.Fatalf("single unread: %v", err)
				}
				if unreadByKey[keyB] != single {
					t.Fatalf("batch unread = %d, single = %d", unreadByKey[keyB], single)
				}
			},
		},
		{
			Name: "count messages by target aggregates per conversation",
			Run: func(t *testing.T, repo Repository, state pmContractState) {
				send(t, repo, state.a, state.b, "b1")
				send(t, repo, state.a, state.b, "b2")
				send(t, repo, state.a, state.c, "c1")
				counts, err := repo.CountMessagesByTarget(state.ctx, state.a, time.Now().UTC().Add(-time.Hour))
				if err != nil {
					t.Fatalf("count messages by target: %v", err)
				}
				all, err := repo.CountMessagesByTarget(state.ctx, state.a, time.Now().UTC().Add(time.Hour))
				if err != nil {
					t.Fatalf("count messages by target (future): %v", err)
				}
				if len(all) != 0 {
					t.Fatalf("future window counts = %#v, want empty", all)
				}
				if counts[state.b] != 2 || counts[state.c] != 1 {
					t.Fatalf("counts = %#v, want {%s:2, %s:1}", counts, state.b, state.c)
				}
			},
		},
	})
}

func otherOf(viewer, a, b string) string {
	if viewer == a {
		return b
	}
	return a
}
