package auth

import (
	"context"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/testdb"
)

// SweepRetention 数据保留期清理（含 IP/PII 与验证码明文）此前无真实库测试。
// 锁定：只删「已撤销且超过保留期」的会话、「过期且超过保留期」的验证码/重置码；
// 活跃会话与未超保留期的行必须保留；清理幂等（第二轮删除 0 行）。
func TestPostgresIT_SweepRetention(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		repo := NewPostgresRepository(txFactory)
		q, ok := persistence.RepoQueryable(ctx, txFactory)
		if !ok {
			t.Fatal("no tx queryable")
		}

		account, err := repo.Register(ctx, RegisterParams{
			LoginID:          "it_retention@example.com",
			Nickname:         "RetentionIT",
			Password:         "secret123",
			Defaults:         ProfileDefaults{LocationName: "Test City", Latitude: 35.0, Longitude: 112.0, CreatedAt: time.Now().UTC()},
			RegisterIP:       "203.0.113.10",
			RegisterLocation: "US",
		})
		if err != nil {
			t.Fatalf("Register: %v", err)
		}

		now := time.Now().UTC()
		exec := func(sql string, args ...any) {
			t.Helper()
			if err := q.Exec(ctx, sql, args...); err != nil {
				t.Fatalf("seed: %v", err)
			}
		}
		// 会话：过期撤销（应删）、近期撤销（保留）、活跃（保留）。
		exec(`INSERT INTO auth_sessions (session_id, user_id, revoked_at) VALUES
			('it_sess_old', $1, $2), ('it_sess_recent', $1, $3), ('it_sess_active', $1, NULL)`,
			account.UserID, now.AddDate(0, 0, -31), now.AddDate(0, 0, -1))
		// 注册验证码：过期 8 天（应删）、过期 1 天（保留）。
		exec(`INSERT INTO auth_register_verification_codes (request_id, login_id, verification_code, expires_at) VALUES
			('it_reg_old', 'nobody@example.com', '111111', $1),
			('it_reg_recent', 'nobody@example.com', '222222', $2)`,
			now.AddDate(0, 0, -8), now.AddDate(0, 0, -1))
		// 密码重置请求：过期 8 天（应删）、过期 1 天（保留）。
		exec(`INSERT INTO auth_password_reset_requests (request_id, login_id, verification_code, expires_at) VALUES
			('it_reset_old', $1, '333333', $2),
			('it_reset_recent', $1, '444444', $3)`,
			account.LoginID, now.AddDate(0, 0, -8), now.AddDate(0, 0, -1))

		count := func(sql string, args ...any) int {
			t.Helper()
			var n int
			if err := q.QueryRow(ctx, sql, args...).Scan(&n); err != nil {
				t.Fatalf("count: %v", err)
			}
			return n
		}

		const sessionRetention = 30 * 24 * time.Hour
		const codeRetention = 7 * 24 * time.Hour
		deleted, err := repo.SweepRetention(ctx, now, sessionRetention, codeRetention)
		if err != nil {
			t.Fatalf("SweepRetention: %v", err)
		}
		if deleted != 3 {
			t.Fatalf("deleted = %d, want 3 (old session + old register code + old reset)", deleted)
		}

		if n := count("SELECT COUNT(*) FROM auth_sessions WHERE session_id = 'it_sess_old'"); n != 0 {
			t.Fatalf("expired revoked session not deleted: %d", n)
		}
		if n := count("SELECT COUNT(*) FROM auth_sessions WHERE session_id IN ('it_sess_recent','it_sess_active')"); n != 2 {
			t.Fatalf("recent/active sessions wrongly deleted: %d", n)
		}
		if n := count("SELECT COUNT(*) FROM auth_register_verification_codes WHERE request_id = 'it_reg_old'"); n != 0 {
			t.Fatalf("expired register code not deleted: %d", n)
		}
		if n := count("SELECT COUNT(*) FROM auth_register_verification_codes WHERE request_id = 'it_reg_recent'"); n != 1 {
			t.Fatalf("recent register code wrongly deleted: %d", n)
		}
		if n := count("SELECT COUNT(*) FROM auth_password_reset_requests WHERE request_id = 'it_reset_old'"); n != 0 {
			t.Fatalf("expired reset request not deleted: %d", n)
		}
		if n := count("SELECT COUNT(*) FROM auth_password_reset_requests WHERE request_id = 'it_reset_recent'"); n != 1 {
			t.Fatalf("recent reset request wrongly deleted: %d", n)
		}

		// 幂等：第二轮已无可删行。
		if deleted, err := repo.SweepRetention(ctx, now, sessionRetention, codeRetention); err != nil || deleted != 0 {
			t.Fatalf("second sweep deleted=%d err=%v, want 0 nil", deleted, err)
		}
	})
}
