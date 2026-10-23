package privatechats

import (
	"context"
	"errors"
	"testing"
	"time"

	"hotspot/backend-go/internal/modules/privatemessages"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/testdb"
)

// TestPostgresIT_ListPrivateChatsKeysetPagination 防回归（审查 P2-7）：列表分页
// 下推到 SQL——排序键为 is_pinned DESC, last_activity_at DESC, target_user_id ASC，
// last_activity 取最近可见消息时间（无消息回退 chat created_at）；已删除且窗口内
// 无消息的会话必须被排除，非法游标返回 ErrInvalidCursor。
func TestPostgresIT_ListPrivateChatsKeysetPagination(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		q := txFactory.DB.(persistence.QueryableDB)
		viewer := testdb.InsertUser(t, txFactory, "it_pc_viewer", "Viewer")
		targetA := testdb.InsertUser(t, txFactory, "it_pc_a", "A")
		targetB := testdb.InsertUser(t, txFactory, "it_pc_b", "B")
		targetC := testdb.InsertUser(t, txFactory, "it_pc_c", "C")
		targetD := testdb.InsertUser(t, txFactory, "it_pc_d", "D")

		t0 := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
		insertChat := func(target string) string {
			chatKey := key(viewer, target)
			if err := q.Exec(ctx, `INSERT INTO private_chats (chat_key, user_a_id, user_b_id, created_at) VALUES ($1,$2,$3,$4)`, chatKey, viewer, target, t0); err != nil {
				t.Fatalf("insert chat %s: %v", target, err)
			}
			return chatKey
		}
		chatA := insertChat(targetA)
		chatB := insertChat(targetB)
		chatC := insertChat(targetC)
		chatD := insertChat(targetD)

		insertMessage := func(chatKey, sender string, sentAt time.Time) {
			if err := q.Exec(ctx, `INSERT INTO private_messages (message_id, chat_key, sender_user_id, type, content, sent_at) VALUES ($1,$2,$3,'text','hi',$4)`, "msg_"+chatKey+"_"+sentAt.Format("150405"), chatKey, sender, sentAt); err != nil {
				t.Fatalf("insert message: %v", err)
			}
		}
		// A 最近活跃 t3，B 为 t2，C 无消息（回退 created_at = t0）。
		insertMessage(chatA, targetA, t0.Add(3*time.Hour))
		insertMessage(chatB, targetB, t0.Add(2*time.Hour))

		repo := NewPostgresRepository(txFactory)

		page1, next1, err := repo.List(ctx, viewer, ListOptions{Limit: 2})
		if err != nil {
			t.Fatalf("page1: %v", err)
		}
		if len(page1) != 2 || page1[0].TargetID != targetA || page1[1].TargetID != targetB {
			t.Fatalf("page1 order: %+v", targetIDs(page1))
		}
		if !page1[0].LastActivityAt.Equal(t0.Add(3 * time.Hour)) {
			t.Fatalf("last activity must come from last message: %v", page1[0].LastActivityAt)
		}
		if next1 == "" {
			t.Fatal("expected next cursor")
		}

		// 第二页：C、D 均无消息（activity = created_at = t0），同刻按 target_id ASC。
		page2, next2, err := repo.List(ctx, viewer, ListOptions{Limit: 2, Cursor: next1})
		if err != nil {
			t.Fatalf("page2: %v", err)
		}
		if len(page2) != 2 || page2[0].TargetID != targetC || page2[1].TargetID != targetD {
			t.Fatalf("page2: %+v", targetIDs(page2))
		}
		if !page2[0].LastActivityAt.Equal(t0) {
			t.Fatalf("no-message chat must fall back to created_at: %v", page2[0].LastActivityAt)
		}
		if next2 != "" {
			t.Fatalf("expected no next cursor, got %q", next2)
		}

		// 置顶维度：C 置顶后排最前（pinned DESC 优先于 activity）。
		if err := q.Exec(ctx, `INSERT INTO private_chat_user_state (chat_key, user_id, is_pinned) VALUES ($1,$2,TRUE)`, chatC, viewer); err != nil {
			t.Fatalf("pin chat: %v", err)
		}
		pinned, _, err := repo.List(ctx, viewer, ListOptions{Limit: 2})
		if err != nil {
			t.Fatalf("pinned page: %v", err)
		}
		if len(pinned) != 2 || pinned[0].TargetID != targetC || pinned[1].TargetID != targetA {
			t.Fatalf("pinned order: %+v", targetIDs(pinned))
		}

		// 已删除且无可见消息 → 排除（D 无消息）；有消息的删除会话保留（B，restore 语义）。
		if err := q.Exec(ctx, `UPDATE private_chat_user_state SET deleted_at = NOW() WHERE chat_key = $1 AND user_id = $2`, chatB, viewer); err != nil {
			t.Fatalf("delete chat b: %v", err)
		}
		if err := q.Exec(ctx, `INSERT INTO private_chat_user_state (chat_key, user_id, deleted_at) VALUES ($1,$2,NOW())`, chatD, viewer); err != nil {
			t.Fatalf("delete chat d: %v", err)
		}
		all, _, err := repo.List(ctx, viewer, ListOptions{Limit: 10})
		if err != nil {
			t.Fatalf("after delete: %v", err)
		}
		if len(all) != 3 || all[0].TargetID != targetC || all[1].TargetID != targetA || all[2].TargetID != targetB {
			t.Fatalf("deleted-chat filtering: %+v", targetIDs(all))
		}

		// 非法游标 → 模块错误（handler 映射 10400）。
		if _, _, err := repo.List(ctx, viewer, ListOptions{Limit: 2, Cursor: "not-a-cursor"}); !errors.Is(err, privatemessages.ErrInvalidCursor) {
			t.Fatalf("invalid cursor err=%v want ErrInvalidCursor", err)
		}
	})
}

func targetIDs(states []ChatState) []string {
	ids := make([]string, 0, len(states))
	for _, state := range states {
		ids = append(ids, state.TargetID)
	}
	return ids
}
