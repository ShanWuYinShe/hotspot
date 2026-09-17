package users

import (
	"context"
	"testing"

	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/testdb"
)

// mustExecIT IT 测试内的裸 SQL 执行辅助（跨模块表插桩/断言专用）。
func mustExecIT(t *testing.T, factory *persistence.Factory, sql string, args ...any) {
	t.Helper()
	qdb, ok := factory.DB.(persistence.QueryableDB)
	if !ok {
		t.Fatal("factory.DB is not QueryableDB")
	}
	if err := qdb.Exec(context.Background(), sql, args...); err != nil {
		t.Fatalf("exec %q: %v", sql, err)
	}
}

func countIT(t *testing.T, factory *persistence.Factory, sql string, args ...any) int {
	t.Helper()
	qdb, ok := factory.DB.(persistence.QueryableDB)
	if !ok {
		t.Fatal("factory.DB is not QueryableDB")
	}
	var n int
	if err := qdb.QueryRow(context.Background(), sql, args...).Scan(&n); err != nil {
		t.Fatalf("query %q: %v", sql, err)
	}
	return n
}

// TestPostgresIT_DeleteProfilePreservesPeerChatAndClearsPII 注销语义双断言
// （spec api_spec_user.md「数据保留策略」）：
//  1. 对话完整性：注销方 peer 的私聊消息与会话状态保留（此前 DELETE
//     private_chats 级联摧毁双方全部记录，属契约违例）；
//  2. 删除权：注销方 users 行的 IP/定位 PII 清除。
func TestPostgresIT_DeleteProfilePreservesPeerChatAndClearsPII(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		repo := NewPostgresRepository(txFactory)

		alice := testdb.InsertUser(t, txFactory, "u_del_alice", "Alice")
		bob := testdb.InsertUser(t, txFactory, "u_del_bob", "Bob")

		// 带注册 IP 的资料（匿名化后须清除）。
		mustExecIT(t, txFactory, `UPDATE users SET register_ip = '10.1.2.3', register_location = 'Shanghai', last_login_ip = '10.1.2.4', latitude = 31.2, longitude = 121.5 WHERE user_id = $1`, alice)

		// Alice↔Bob 私聊：一条双方共享的消息 + 双方各自的会话状态行。
		mustExecIT(t, txFactory, `INSERT INTO private_chats (chat_key, user_a_id, user_b_id, created_at) VALUES ('chat_del_it', $1, $2, NOW())`, alice, bob)
		mustExecIT(t, txFactory, `INSERT INTO private_messages (message_id, chat_key, sender_user_id, type, content, sent_at) VALUES ('msg_del_it_1', 'chat_del_it', $1, 'text', '你好', NOW())`, bob)
		mustExecIT(t, txFactory, `INSERT INTO private_chat_user_state (chat_key, user_id, updated_at) VALUES ('chat_del_it', $1, NOW()), ('chat_del_it', $2, NOW())`, alice, bob)
		mustExecIT(t, txFactory, `INSERT INTO private_chat_read_state (chat_key, user_id, updated_at) VALUES ('chat_del_it', $1, NOW()), ('chat_del_it', $2, NOW())`, alice, bob)

		// Alice 在一个 Bob 拥有的群里。
		mustExecIT(t, txFactory, `INSERT INTO groups (group_id, name, location_name, latitude, longitude, owner_id) VALUES ('g_del_it', 'IT群', '', 0, 0, $1)`, bob)
		mustExecIT(t, txFactory, `INSERT INTO group_members (group_id, user_id) VALUES ('g_del_it', $1), ('g_del_it', $2)`, bob, alice)

		if err := repo.DeleteProfile(ctx, alice); err != nil {
			t.Fatalf("DeleteProfile: %v", err)
		}

		// 对话完整性：消息本体与对方的会话/已读状态行保留。
		if got := countIT(t, txFactory, `SELECT COUNT(*) FROM private_messages WHERE chat_key = 'chat_del_it'`); got != 1 {
			t.Fatalf("peer messages survived = %d, want 1", got)
		}
		if got := countIT(t, txFactory, `SELECT COUNT(*) FROM private_chat_user_state WHERE user_id = $1`, bob); got != 1 {
			t.Fatalf("peer chat state survived = %d, want 1", got)
		}
		if got := countIT(t, txFactory, `SELECT COUNT(*) FROM private_chat_read_state WHERE user_id = $1`, bob); got != 1 {
			t.Fatalf("peer read state survived = %d, want 1", got)
		}
		// 注销方视角数据清除。
		if got := countIT(t, txFactory, `SELECT COUNT(*) FROM private_chat_user_state WHERE user_id = $1`, alice); got != 0 {
			t.Fatalf("deleted user chat state rows = %d, want 0", got)
		}
		// 群成员身份清除，群与其余成员不受影响。
		if got := countIT(t, txFactory, `SELECT COUNT(*) FROM group_members WHERE user_id = $1`, alice); got != 0 {
			t.Fatalf("deleted user group memberships = %d, want 0", got)
		}
		if got := countIT(t, txFactory, `SELECT COUNT(*) FROM group_members WHERE user_id = $1`, bob); got != 1 {
			t.Fatalf("peer group membership = %d, want 1", got)
		}

		// 删除权：IP/定位 PII 清除 + 匿名化占位。
		var nickname, regIP, lastIP string
		var lat float64
		if err := txFactory.DB.(persistence.QueryableDB).QueryRow(context.Background(),
			`SELECT nickname, COALESCE(register_ip, ''), COALESCE(last_login_ip, ''), latitude FROM users WHERE user_id = $1`, alice,
		).Scan(&nickname, &regIP, &lastIP, &lat); err != nil {
			t.Fatalf("query anonymized user: %v", err)
		}
		if nickname != "已注销用户" {
			t.Fatalf("nickname = %q, want 已注销用户", nickname)
		}
		if regIP != "" || lastIP != "" {
			t.Fatalf("PII not cleared: register_ip=%q last_login_ip=%q", regIP, lastIP)
		}
		if lat != 0 {
			t.Fatalf("latitude = %v, want 0", lat)
		}
	})
}
