package privatechats

import (
	"context"
	"errors"
	"fmt"
	"strings"
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

// TestPostgresIT_CountUnreadForUser 覆盖冷启动 unread_summary 私聊聚合的真实库
// 口径（第 81 轮遗留项）：按已读游标（last_read_sent_at，含同 sent_at 的
// message_id 边界）动态计算，排除本人发送与已撤回消息；会话可见边界
// （visible_since，删除会话后只算新消息）；对方已注销的会话不计（列表不可达）。
// 事务内自建数据、结束整体回滚（不残留、不污染共享库）。
func TestPostgresIT_CountUnreadForUser(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		q := txFactory.DB.(persistence.QueryableDB)
		viewer := testdb.InsertUser(t, txFactory, "it_pcu_viewer", "Viewer")
		peerA := testdb.InsertUser(t, txFactory, "it_pcu_a", "A")
		peerB := testdb.InsertUser(t, txFactory, "it_pcu_b", "B")
		peerC := testdb.InsertUser(t, txFactory, "it_pcu_c", "C")
		peerD := testdb.InsertUser(t, txFactory, "it_pcu_d", "D")
		peerE := testdb.InsertUser(t, txFactory, "it_pcu_e", "E")
		peerF := testdb.InsertUser(t, txFactory, "it_pcu_f", "F")

		base := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
		insertChat := func(peer string) string {
			chatKey := key(viewer, peer)
			if err := q.Exec(ctx, `INSERT INTO private_chats (chat_key, user_a_id, user_b_id, created_at) VALUES ($1,$2,$3,$4)`, chatKey, viewer, peer, base); err != nil {
				t.Fatalf("insert chat %s: %v", peer, err)
			}
			return chatKey
		}
		insertMessageWithID := func(chatKey, messageID, sender string, sentAt time.Time, recalled bool) {
			if err := q.Exec(ctx, `INSERT INTO private_messages (message_id, chat_key, sender_user_id, type, content, sent_at, is_recalled) VALUES ($1,$2,$3,'text','hi',$4,$5)`, messageID, chatKey, sender, sentAt, recalled); err != nil {
				t.Fatalf("insert message: %v", err)
			}
		}
		insertMessage := func(chatKey, sender string, sentAt time.Time, recalled bool) string {
			messageID := "it_pcu_msg_" + sentAt.Format("150405.000000000")
			insertMessageWithID(chatKey, messageID, sender, sentAt, recalled)
			return messageID
		}
		// 已读位点 upsert（同一会话可能多次推进，如「全部已读 -> 0」用例）。
		markRead := func(chatKey, messageID string, sentAt time.Time) {
			if err := q.Exec(ctx, `INSERT INTO private_chat_read_state (chat_key, user_id, last_read_message_id, last_read_sent_at) VALUES ($1,$2,$3,$4)
				ON CONFLICT (chat_key, user_id) DO UPDATE SET last_read_message_id = EXCLUDED.last_read_message_id, last_read_sent_at = EXCLUDED.last_read_sent_at`, chatKey, viewer, messageID, sentAt); err != nil {
				t.Fatalf("mark read: %v", err)
			}
		}

		// 会话 A：无已读游标 → 对方 3 条全部未读；本人发送与已撤回消息不计。
		chatA := insertChat(peerA)
		insertMessage(chatA, peerA, base.Add(1*time.Minute), false)
		insertMessage(chatA, peerA, base.Add(2*time.Minute), false)
		insertMessage(chatA, peerA, base.Add(3*time.Minute), false)
		insertMessage(chatA, viewer, base.Add(4*time.Minute), false)
		insertMessage(chatA, peerA, base.Add(5*time.Minute), true)

		// 会话 B：部分已读——已读位点落在首条，其后 1 条未读。
		chatB := insertChat(peerB)
		b1 := insertMessage(chatB, peerB, base.Add(10*time.Minute), false)
		insertMessage(chatB, peerB, base.Add(11*time.Minute), false)
		markRead(chatB, b1, base.Add(10*time.Minute))

		// 会话 C：对方已注销 → 会话在列表不可达，未读不计（否则角标无法清除）。
		chatC := insertChat(peerC)
		insertMessage(chatC, peerC, base.Add(20*time.Minute), false)
		if err := q.Exec(ctx, `UPDATE users SET is_deleted = TRUE WHERE user_id = $1`, peerC); err != nil {
			t.Fatalf("soft delete peer c: %v", err)
		}

		// 会话 D：本人删除会话（visible_since/deleted_at）后只计新消息。
		chatD := insertChat(peerD)
		insertMessage(chatD, peerD, base.Add(30*time.Minute), false)
		visibleSince := base.Add(31 * time.Minute)
		if err := q.Exec(ctx, `INSERT INTO private_chat_user_state (chat_key, user_id, visible_since, deleted_at) VALUES ($1,$2,$3,$3)`, chatD, viewer, visibleSince); err != nil {
			t.Fatalf("mark chat deleted: %v", err)
		}
		insertMessage(chatD, peerD, base.Add(32*time.Minute), false)

		// 会话 E：已读位点只剩 last_read_sent_at、last_read_message_id 为 NULL
		// （该列 ON DELETE SET NULL，被指向消息删除后即可到达）。同刻消息仍须
		// 按已读排除，只有更晚的 1 条计入——若把边界简化成
		// NOT(sent_at = last_read AND message_id = last_read)，这里会多算 1 条。
		chatE := insertChat(peerE)
		insertMessage(chatE, peerE, base.Add(40*time.Minute), false)
		insertMessage(chatE, peerE, base.Add(41*time.Minute), false)
		insertMessage(chatE, peerE, base.Add(42*time.Minute), false)
		if err := q.Exec(ctx, `INSERT INTO private_chat_read_state (chat_key, user_id, last_read_message_id, last_read_sent_at) VALUES ($1,$2,NULL,$3)`, chatE, viewer, base.Add(41*time.Minute)); err != nil {
			t.Fatalf("mark chat E read with null message id: %v", err)
		}

		// 会话 F：同一 sent_at 下两条对方消息、已读位点指向其中一条 message_id：
		// 另一条同刻消息按原 OR 语义计入（sent_at = last_read 的 message_id 分支），
		// 本人同刻消息不计。
		chatF := insertChat(peerF)
		sameAt := base.Add(50 * time.Minute)
		readMsgF := "it_pcu_msg_f_read"
		insertMessageWithID(chatF, readMsgF, peerF, sameAt, false)
		insertMessageWithID(chatF, "it_pcu_msg_f_other", peerF, sameAt, false)
		insertMessageWithID(chatF, "it_pcu_msg_f_self", viewer, sameAt, false)
		markRead(chatF, readMsgF, sameAt)

		repo := NewPostgresRepository(txFactory)
		// A=3 + B=1 + C=0 + D=1 + E=1 + F=1 = 7。
		count, err := repo.CountUnreadForUser(ctx, viewer)
		if err != nil {
			t.Fatalf("CountUnreadForUser: %v", err)
		}
		if count != 7 {
			t.Fatalf("unread count = %d, want 7 (A3+B1+C0+D1+E1+F1)", count)
		}

		// 全部会话已读（位点推进到各自最新消息）→ 0：游标口径不残留未读。
		markRead(chatA, insertMessage(chatA, viewer, base.Add(6*time.Minute), false), base.Add(6*time.Minute))
		markRead(chatB, insertMessage(chatB, viewer, base.Add(12*time.Minute), false), base.Add(12*time.Minute))
		markRead(chatD, insertMessage(chatD, viewer, base.Add(33*time.Minute), false), base.Add(33*time.Minute))
		markRead(chatE, insertMessage(chatE, viewer, base.Add(43*time.Minute), false), base.Add(43*time.Minute))
		markRead(chatF, insertMessage(chatF, viewer, sameAt.Add(time.Minute), false), sameAt.Add(time.Minute))
		count, err = repo.CountUnreadForUser(ctx, viewer)
		if err != nil {
			t.Fatalf("CountUnreadForUser after read: %v", err)
		}
		if count != 0 {
			t.Fatalf("unread count after read-all = %d, want 0", count)
		}

		// 无会话用户 → 0（不因空集返回错误）。
		stranger := testdb.InsertUser(t, txFactory, "it_pcu_stranger", "Stranger")
		count, err = repo.CountUnreadForUser(ctx, stranger)
		if err != nil || count != 0 {
			t.Fatalf("stranger count = %d err=%v, want 0", count, err)
		}
	})
}

// TestPostgresIT_CountUnreadForUserUsesIndexRange 锁死聚合 SQL 的计划形态（第 82
// 轮审计 C-P1-1 性能回归）：未读区间必须能下推为消息索引的 Index Cond，而不是
// 把整张 private_messages 读出来再过滤。enable_seqscan=off 只用于在小数据集的
// 测试事务里暴露「谓词能否作为索引条件」（真实数据量下规划器本就会选索引），
// 不改变 SQL 语义；断言只要求「目标索引 + Index Cond 含 sent_at」，索引扫描与
// 位图索引扫描均可通过，不锁定具体节点形态。
func TestPostgresIT_CountUnreadForUserUsesIndexRange(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		q := txFactory.DB.(persistence.QueryableDB)
		viewer := testdb.InsertUser(t, txFactory, "it_pcu_plan_viewer", "Viewer")
		base := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
		for i := 0; i < 30; i++ {
			peer := testdb.InsertUser(t, txFactory, fmt.Sprintf("it_pcu_plan_peer_%02d", i), "Peer")
			if err := q.Exec(ctx, `INSERT INTO private_chats (chat_key, user_a_id, user_b_id, created_at) VALUES ($1,$2,$3,$4)`,
				key(viewer, peer), viewer, peer, base); err != nil {
				t.Fatalf("insert chat: %v", err)
			}
		}
		// 每个会话 100 条消息、已读游标停在 60：未读只占 40%，规划器才有依据
		// 选择 (chat_key, sent_at) 的范围扫描，而不是读完整个会话再过滤；数据量
		// 需要过 ANALYZE，否则统计缺失时索引选择退化为页大小比较（会选中
		// chat_key 前缀相同但无 sent_at 列的索引，断言失去区分力）。
		if err := q.Exec(ctx, `INSERT INTO private_messages (message_id, chat_key, sender_user_id, type, content, sent_at, is_recalled)
SELECT pc.chat_key || ':' || j, pc.chat_key,
       CASE WHEN j % 2 = 0 THEN $1::text ELSE CASE WHEN pc.user_a_id = $1 THEN pc.user_b_id ELSE pc.user_a_id END END,
       'text', 'hi', $2::timestamptz + j * INTERVAL '1 minute', FALSE
FROM private_chats pc, generate_series(1, 100) j
WHERE pc.user_a_id = $1 OR pc.user_b_id = $1`, viewer, base); err != nil {
			t.Fatalf("insert messages: %v", err)
		}
		if err := q.Exec(ctx, `INSERT INTO private_chat_read_state (chat_key, user_id, last_read_message_id, last_read_sent_at)
SELECT pc.chat_key, $1, pc.chat_key || ':60', $2::timestamptz + 60 * INTERVAL '1 minute'
FROM private_chats pc WHERE pc.user_a_id = $1 OR pc.user_b_id = $1`, viewer, base); err != nil {
			t.Fatalf("insert read state: %v", err)
		}
		for _, table := range []string{"private_messages", "private_chats", "private_chat_read_state", "private_chat_user_state"} {
			if err := q.Exec(ctx, "ANALYZE "+table); err != nil {
				t.Fatalf("analyze %s: %v", table, err)
			}
		}
		if _, err := NewPostgresRepository(txFactory).CountUnreadForUser(ctx, viewer); err != nil {
			t.Fatalf("CountUnreadForUser: %v", err)
		}
		if err := q.Exec(ctx, `SET LOCAL enable_seqscan = off`); err != nil {
			t.Fatalf("set local enable_seqscan: %v", err)
		}
		plan := explainPlanText(t, ctx, q, queryCountUnreadPrivateMessagesForUser, viewer)
		assertUnreadPlanUsesIndexRange(t, plan, "private_messages", "idx_private_messages_chat_key_sent_at")
	})
}

// explainPlanText 取查询的文本计划（裸 EXPLAIN，不执行），供计划形态回归断言。
func explainPlanText(t *testing.T, ctx context.Context, q persistence.QueryableDB, query string, args ...any) string {
	t.Helper()
	rows, err := q.Query(ctx, "EXPLAIN "+query, args...)
	if err != nil {
		t.Fatalf("explain: %v", err)
	}
	defer rows.Close()
	var lines []string
	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err != nil {
			t.Fatalf("scan explain: %v", err)
		}
		lines = append(lines, line)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("explain rows: %v", err)
	}
	return strings.Join(lines, "\n")
}

// assertUnreadPlanUsesIndexRange 断言未读聚合在消息表上走索引范围扫描：计划里
// 出现目标索引名、且某条 Index Cond 含 sent_at（单次比较下推成功），同时不得
// 出现对消息表的全表 Seq Scan（否则成本重新随聊天历史总量增长）。
func assertUnreadPlanUsesIndexRange(t *testing.T, plan, table, index string) {
	t.Helper()
	if !strings.Contains(plan, index) {
		t.Fatalf("plan does not use %s:\n%s", index, plan)
	}
	hasRangeCond := false
	for _, line := range strings.Split(plan, "\n") {
		if strings.Contains(line, "Seq Scan on "+table) {
			t.Fatalf("unread aggregation still scans %s sequentially:\n%s", table, plan)
		}
		if strings.Contains(line, "Index Cond") && strings.Contains(line, "sent_at") {
			hasRangeCond = true
		}
	}
	if !hasRangeCond {
		t.Fatalf("sent_at is not an index range condition on %s:\n%s", table, plan)
	}
}

func targetIDs(states []ChatState) []string {
	ids := make([]string, 0, len(states))
	for _, state := range states {
		ids = append(ids, state.TargetID)
	}
	return ids
}
