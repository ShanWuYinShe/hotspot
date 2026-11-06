package integration

import (
	"context"
	"testing"

	"hotspot/backend-go/internal/modules/auth"
	"hotspot/backend-go/internal/modules/groups"
	"hotspot/backend-go/internal/modules/media"
	"hotspot/backend-go/internal/modules/moments"
	"hotspot/backend-go/internal/modules/notify"
	"hotspot/backend-go/internal/modules/privatechats"
	"hotspot/backend-go/internal/modules/privatemessages"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/testdb"
)

// TestAccountPurgeModuleRepositories：账户注销级联的每段清理必须删「归属本模块」
// 的行且不触碰他人行（架构审计 P0-4）。用户编排顺序在 users.Service；这里逐模块
// 验证真实库 SQL 归属与幂等性。
func TestAccountPurgeModuleRepositories(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		target := testdb.InsertUser(t, txFactory, "u_purge_target", "Target")
		peer := testdb.InsertUser(t, txFactory, "u_purge_peer", "Peer")

		exec := func(sql string, args ...any) {
			t.Helper()
			qdb, ok := txFactory.DB.(persistence.QueryableDB)
			if !ok {
				t.Fatal("factory.DB is not QueryableDB")
			}
			if err := qdb.Exec(ctx, sql, args...); err != nil {
				t.Fatalf("seed %q: %v", sql, err)
			}
		}
		count := func(sql string, args ...any) int {
			t.Helper()
			qdb, ok := txFactory.DB.(persistence.QueryableDB)
			if !ok {
				t.Fatal("factory.DB is not QueryableDB")
			}
			var n int
			if err := qdb.QueryRow(ctx, sql, args...).Scan(&n); err != nil {
				t.Fatalf("count %q: %v", sql, err)
			}
			return n
		}

		// 各归属模块的用户数据（目标用户各一行；peer 对照行）。
		exec(`INSERT INTO auth_accounts (login_id, user_id, password_hash) VALUES ('purge_login', $1, 'x'), ('purge_peer_login', $2, 'x')`, target, peer)
		exec(`INSERT INTO auth_sessions (session_id, user_id) VALUES ('purge_sess', $1), ('purge_peer_sess', $2)`, target, peer)
		exec(`INSERT INTO notifications (notification_id, user_id, type) VALUES ('purge_notif', $1, 'system_announcement'), ('purge_peer_notif', $2, 'system_announcement')`, target, peer)
		exec(`INSERT INTO media_upload_sessions (upload_id, user_id, storage_key, file_name, declared_file_size, declared_mime_type, category, status, created_at) VALUES ('purge_upload', $1, 'k1', 'a.png', 1, 'image/png', 'moment_image', 'pending', NOW()), ('purge_peer_upload', $2, 'k2', 'b.png', 1, 'image/png', 'moment_image', 'pending', NOW())`, target, peer)
		exec(`INSERT INTO moments (moment_id, user_id, description, location_name, latitude, longitude, privacy_level) VALUES ('purge_moment', $1, 'm', '', 0, 0, 'public')`, peer)
		exec(`INSERT INTO moment_favorites (user_id, moment_id) VALUES ($1, 'purge_moment'), ($2, 'purge_moment')`, target, peer)
		exec(`INSERT INTO private_chats (chat_key, user_a_id, user_b_id) VALUES ('purge_chat', $1, $2)`, target, peer)
		exec(`INSERT INTO private_chat_user_state (chat_key, user_id) VALUES ('purge_chat', $1), ('purge_chat', $2)`, target, peer)
		exec(`INSERT INTO private_chat_read_state (chat_key, user_id) VALUES ('purge_chat', $1), ('purge_chat', $2)`, target, peer)
		exec(`INSERT INTO groups (group_id, name, location_name, latitude, longitude, owner_id) VALUES ('purge_group', 'g', '', 0, 0, $1)`, peer)
		exec(`INSERT INTO group_members (group_id, user_id) VALUES ('purge_group', $1), ('purge_group', $2)`, target, peer)

		purgers := []struct {
			name  string
			purge func(context.Context, string) error
		}{
			{"auth", auth.NewPostgresRepository(txFactory).PurgeUserData},
			{"groups", groups.NewPostgresRepository(txFactory).PurgeUserData},
			{"media", media.NewPostgresRepository(txFactory).PurgeUserData},
			{"moments", moments.NewPostgresRepository(txFactory).PurgeUserData},
			{"notify", notify.NewPostgresRepository(txFactory).PurgeUserData},
			{"privatechats", privatechats.NewPostgresRepository(txFactory).PurgeUserData},
			{"privatemessages", privatemessages.NewPostgresRepository(txFactory).PurgeUserData},
		}
		for _, p := range purgers {
			if err := p.purge(ctx, target); err != nil {
				t.Fatalf("%s purge: %v", p.name, err)
			}
			// 幂等：重复执行不报错。
			if err := p.purge(ctx, target); err != nil {
				t.Fatalf("%s purge (repeat): %v", p.name, err)
			}
		}

		for _, tc := range []struct {
			name string
			sql  string
		}{
			{"auth_accounts", `SELECT COUNT(*) FROM auth_accounts WHERE user_id = $1`},
			{"auth_sessions", `SELECT COUNT(*) FROM auth_sessions WHERE user_id = $1`},
			{"notifications", `SELECT COUNT(*) FROM notifications WHERE user_id = $1`},
			{"media_upload_sessions", `SELECT COUNT(*) FROM media_upload_sessions WHERE user_id = $1`},
			{"moment_favorites", `SELECT COUNT(*) FROM moment_favorites WHERE user_id = $1`},
			{"private_chat_user_state", `SELECT COUNT(*) FROM private_chat_user_state WHERE user_id = $1`},
			{"private_chat_read_state", `SELECT COUNT(*) FROM private_chat_read_state WHERE user_id = $1`},
			{"group_members", `SELECT COUNT(*) FROM group_members WHERE user_id = $1`},
		} {
			if got := count(tc.sql, target); got != 0 {
				t.Fatalf("%s target rows = %d, want 0", tc.name, got)
			}
			if got := count(tc.sql, peer); got != 1 {
				t.Fatalf("%s peer rows = %d, want 1", tc.name, got)
			}
		}
	})
}
