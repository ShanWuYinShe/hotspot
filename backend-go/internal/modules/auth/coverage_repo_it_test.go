package auth

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	usersmodule "hotspot/backend-go/internal/modules/users"
	"hotspot/backend-go/internal/platform/appconfig"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/testdb"
)

func covITLoginID(name string) string {
	return fmt.Sprintf("auth_cov_g3_%s_%d@example.com", name, time.Now().UnixNano())
}

func covITUsername(name string) string {
	return fmt.Sprintf("auth_cov_g3_%s_%d", name, time.Now().UnixNano())
}

func covITUserID(name string) string {
	return fmt.Sprintf("user_auth_cov_g3_%s_%d", name, time.Now().UnixNano())
}

func covRegisterParams(loginID, password string) RegisterParams {
	return RegisterParams{
		LoginID:          loginID,
		Nickname:         "CovUser",
		Password:         password,
		Defaults:         ProfileDefaults{LocationName: "Test City", Latitude: 35.0, Longitude: 112.0, CreatedAt: time.Now().UTC()},
		RegisterIP:       "203.0.113.10",
		RegisterLocation: "US",
	}
}

func TestPostgresIT_CovFailClosedGuards(t *testing.T) {
	factory := &persistence.Factory{
		Config: persistence.Config{PostgresDSN: "postgres://example"},
		DB:     persistence.NewInMemoryDB(),
	}
	repo := NewPostgresRepository(factory)
	repo.SetProfileWriter(usersmodule.NewPostgresRepository(factory))
	ctx := context.Background()
	now := time.Now().UTC()
	cases := []struct {
		name string
		fn   func() error
	}{
		{"Register", func() error {
			_, err := repo.Register(ctx, RegisterParams{LoginID: "alice", Password: "password1"})
			return err
		}},
		{"AttachCredential", func() error { return repo.AttachCredential(ctx, "u1", "alice", "password1") }},
		{"Authenticate", func() error { _, err := repo.Authenticate(ctx, "alice", "password1"); return err }},
		{"CreatePasswordReset", func() error { _, err := repo.CreatePasswordReset(ctx, "alice", now); return err }},
		{"ResetPassword", func() error { _, err := repo.ResetPassword(ctx, "alice", "123456", "password1"); return err }},
		{"ChangePassword", func() error { return repo.ChangePassword(ctx, "u1", "old", "password1") }},
		{"CreateRegisterCode", func() error { _, err := repo.CreateRegisterCode(ctx, "alice", now); return err }},
		{"VerifyRegisterCode", func() error { return repo.VerifyRegisterCode(ctx, "alice", "123456") }},
		{"ConsumeRegisterCode", func() error { return repo.ConsumeRegisterCode(ctx, "alice", "123456") }},
		{"PurgeUserData", func() error { return repo.PurgeUserData(ctx, "u1") }},
		{"SweepRetention", func() error { _, err := repo.SweepRetention(ctx, now, time.Hour, time.Hour); return err }},
	}
	for _, tc := range cases {
		if err := tc.fn(); !errors.Is(err, persistence.ErrNotImplemented) {
			t.Fatalf("%s: expected ErrNotImplemented, got %v", tc.name, err)
		}
	}
}

func TestPostgresIT_CovRegisterCodeLifecycle(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		repo := newITRepo(txFactory)
		q, ok := persistence.RepoQueryable(ctx, txFactory)
		if !ok {
			t.Fatal("no tx queryable")
		}
		loginID := covITLoginID("regcode")
		code, err := repo.CreateRegisterCode(ctx, loginID, time.Now().UTC().Add(10*time.Minute))
		if err != nil || code == "" {
			t.Fatalf("create register code: code=%q err=%v", code, err)
		}
		if _, err := repo.CreateRegisterCode(ctx, loginID, time.Now().UTC().Add(10*time.Minute)); !errors.Is(err, ErrRegisterCodeThrottled) {
			t.Fatalf("expected throttle on immediate resend, got %v", err)
		}
		latest, err := repo.latestRegisterCodeCreatedAt(ctx, q, loginID)
		if err != nil || latest == nil {
			t.Fatalf("latestRegisterCodeCreatedAt: %v %v", latest, err)
		}
		if err := repo.VerifyRegisterCode(ctx, loginID, code); err != nil {
			t.Fatalf("verify valid code: %v", err)
		}
		if err := repo.ConsumeRegisterCode(ctx, loginID, code); err != nil {
			t.Fatalf("consume code: %v", err)
		}
		if err := repo.ConsumeRegisterCode(ctx, loginID, code); err != nil {
			t.Fatalf("consume must be idempotent, got %v", err)
		}
		if err := repo.VerifyRegisterCode(ctx, loginID, code); !errors.Is(err, ErrInvalidRegisterCode) {
			t.Fatalf("consumed code must be rejected, got %v", err)
		}
	})
}

func TestPostgresIT_CovRegisterCodeFailureBurnsCode(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		repo := newITRepo(txFactory)
		loginID := covITLoginID("regburn")
		code, err := repo.CreateRegisterCode(ctx, loginID, time.Now().UTC().Add(10*time.Minute))
		if err != nil || code == "" {
			t.Fatalf("create register code: code=%q err=%v", code, err)
		}
		wrong := wrongCodeNotEqual(t, code)
		for i := 0; i < maxVerificationCodeFailures; i++ {
			if err := repo.VerifyRegisterCode(ctx, loginID, wrong); !errors.Is(err, ErrInvalidRegisterCode) {
				t.Fatalf("attempt %d: expected invalid register code, got %v", i+1, err)
			}
		}
		if err := repo.VerifyRegisterCode(ctx, loginID, code); !errors.Is(err, ErrInvalidRegisterCode) {
			t.Fatalf("burned code must be rejected, got %v", err)
		}
	})
}

func TestPostgresIT_CovRegisterCodeExpiredAndUnknown(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		repo := newITRepo(txFactory)
		loginID := covITLoginID("regexpired")
		code, err := repo.CreateRegisterCode(ctx, loginID, time.Now().UTC().Add(-time.Minute))
		if err != nil || code == "" {
			t.Fatalf("create expired register code: code=%q err=%v", code, err)
		}
		if err := repo.VerifyRegisterCode(ctx, loginID, code); !errors.Is(err, ErrInvalidRegisterCode) {
			t.Fatalf("expired code must be rejected, got %v", err)
		}
		if err := repo.VerifyRegisterCode(ctx, covITLoginID("regnone"), "123456"); !errors.Is(err, ErrInvalidRegisterCode) {
			t.Fatalf("unknown login must be rejected, got %v", err)
		}
	})
}

func TestPostgresIT_CovAttachCredential(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		repo := newITRepo(txFactory)
		userID := covITUserID("attach")
		testdb.InsertUser(t, txFactory, userID, "AttachUser")
		loginID := covITUsername("attach")
		if err := repo.AttachCredential(ctx, userID, loginID, "password1"); err != nil {
			t.Fatalf("attach credential: %v", err)
		}
		account, err := repo.Authenticate(ctx, loginID, "password1")
		if err != nil || account.UserID != userID {
			t.Fatalf("authenticate attached credential: %#v err=%v", account, err)
		}
		otherUserID := covITUserID("attach2")
		testdb.InsertUser(t, txFactory, otherUserID, "AttachUser2")
		if err := repo.AttachCredential(ctx, otherUserID, loginID, "password1"); !errors.Is(err, ErrLoginIDConflict) {
			t.Fatalf("duplicate login must conflict, got %v", err)
		}
	})
}

func TestPostgresIT_CovChangePassword(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		repo := newITRepo(txFactory)
		loginID := covITLoginID("changepwd")
		account, err := repo.Register(ctx, covRegisterParams(loginID, "oldpassword1"))
		if err != nil {
			t.Fatalf("register: %v", err)
		}
		if err := repo.ChangePassword(ctx, account.UserID, "oldpassword1", "newpassword1"); err != nil {
			t.Fatalf("change password: %v", err)
		}
		if _, err := repo.Authenticate(ctx, loginID, "newpassword1"); err != nil {
			t.Fatalf("authenticate with new password: %v", err)
		}
		if _, err := repo.Authenticate(ctx, loginID, "oldpassword1"); !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("old password must be rejected, got %v", err)
		}
		if err := repo.ChangePassword(ctx, account.UserID, "wrong-old", "newpassword2"); !errors.Is(err, ErrInvalidOldPassword) {
			t.Fatalf("wrong old password must be rejected, got %v", err)
		}
		if err := repo.ChangePassword(ctx, covITUserID("missing"), "oldpassword1", "newpassword1"); !errors.Is(err, ErrInvalidOldPassword) {
			t.Fatalf("missing account must be rejected, got %v", err)
		}
	})
}

func TestPostgresIT_CovPasswordResetFailures(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		repo := newITRepo(txFactory)

		loginID := covITLoginID("resetburn")
		if _, err := repo.Register(ctx, covRegisterParams(loginID, "password1")); err != nil {
			t.Fatalf("register: %v", err)
		}
		code, err := repo.CreatePasswordReset(ctx, loginID, time.Now().UTC().Add(10*time.Minute))
		if err != nil || code == "" {
			t.Fatalf("create password reset: code=%q err=%v", code, err)
		}
		if _, err := repo.CreatePasswordReset(ctx, loginID, time.Now().UTC().Add(10*time.Minute)); !errors.Is(err, ErrPasswordResetThrottled) {
			t.Fatalf("expected reset throttle, got %v", err)
		}
		wrong := wrongCodeNotEqual(t, code)
		for i := 0; i < maxVerificationCodeFailures; i++ {
			if _, err := repo.ResetPassword(ctx, loginID, wrong, "newpassword1"); !errors.Is(err, ErrPasswordResetNotFound) {
				t.Fatalf("attempt %d: expected reset not found, got %v", i+1, err)
			}
		}
		if _, err := repo.ResetPassword(ctx, loginID, code, "newpassword1"); !errors.Is(err, ErrPasswordResetNotFound) {
			t.Fatalf("burned reset code must be rejected, got %v", err)
		}

		// 成功使用后不可重放。
		loginID2 := covITLoginID("resetreuse")
		if _, err := repo.Register(ctx, covRegisterParams(loginID2, "password1")); err != nil {
			t.Fatalf("register2: %v", err)
		}
		code2, err := repo.CreatePasswordReset(ctx, loginID2, time.Now().UTC().Add(10*time.Minute))
		if err != nil || code2 == "" {
			t.Fatalf("create reset2: code=%q err=%v", code2, err)
		}
		if _, err := repo.ResetPassword(ctx, loginID2, code2, "newpassword1"); err != nil {
			t.Fatalf("reset password success path: %v", err)
		}
		if _, err := repo.ResetPassword(ctx, loginID2, code2, "newpassword2"); !errors.Is(err, ErrPasswordResetNotFound) {
			t.Fatalf("used reset code must be rejected, got %v", err)
		}

		// 过期重置码被拒绝。
		loginID3 := covITLoginID("resetexpired")
		if _, err := repo.Register(ctx, covRegisterParams(loginID3, "password1")); err != nil {
			t.Fatalf("register3: %v", err)
		}
		code3, err := repo.CreatePasswordReset(ctx, loginID3, time.Now().UTC().Add(-time.Minute))
		if err != nil || code3 == "" {
			t.Fatalf("create expired reset: code=%q err=%v", code3, err)
		}
		if _, err := repo.ResetPassword(ctx, loginID3, code3, "newpassword1"); !errors.Is(err, ErrPasswordResetNotFound) {
			t.Fatalf("expired reset code must be rejected, got %v", err)
		}

		// 未知账号 → not found（账号不存在分支）。
		if _, err := repo.ResetPassword(ctx, covITLoginID("resetmissing"), "123456", "newpassword1"); !errors.Is(err, ErrPasswordResetNotFound) {
			t.Fatalf("unknown account must be rejected, got %v", err)
		}
	})
}

func TestPostgresIT_CovPurgeUserData(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		repo := newITRepo(txFactory)
		q, ok := persistence.RepoQueryable(ctx, txFactory)
		if !ok {
			t.Fatal("no tx queryable")
		}
		loginID := covITLoginID("purge")
		account, err := repo.Register(ctx, covRegisterParams(loginID, "password1"))
		if err != nil {
			t.Fatalf("register: %v", err)
		}
		sessionID := fmt.Sprintf("cov_sess_%d", time.Now().UnixNano())
		if err := q.Exec(ctx, `INSERT INTO auth_sessions (session_id, user_id) VALUES ($1, $2)`, sessionID, account.UserID); err != nil {
			t.Fatalf("seed session: %v", err)
		}
		svc := NewServiceWithSource(ctx, platformauth.NewService(testConfig()), repo, testConfig(), appconfig.NewStaticSource(testConfig()))
		if err := svc.PurgeUserData(ctx, account.UserID); err != nil {
			t.Fatalf("service purge: %v", err)
		}
		count := func(sql string) int {
			var n int
			if err := q.QueryRow(ctx, sql, account.UserID).Scan(&n); err != nil {
				t.Fatalf("count: %v", err)
			}
			return n
		}
		if n := count(`SELECT COUNT(*) FROM auth_accounts WHERE user_id = $1`); n != 0 {
			t.Fatalf("auth_accounts not purged: %d", n)
		}
		if n := count(`SELECT COUNT(*) FROM auth_sessions WHERE user_id = $1`); n != 0 {
			t.Fatalf("auth_sessions not purged: %d", n)
		}
		if n := count(`SELECT COUNT(*) FROM users WHERE user_id = $1`); n != 1 {
			t.Fatalf("users row must remain (owned by users module): %d", n)
		}
		if err := svc.PurgeUserData(ctx, account.UserID); err != nil {
			t.Fatalf("purge must be idempotent, got %v", err)
		}
	})
}

func TestPostgresIT_CovModuleSweepRetention(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		q, ok := persistence.RepoQueryable(ctx, txFactory)
		if !ok {
			t.Fatal("no tx queryable")
		}
		loginID := covITLoginID("sweep")
		requestID := fmt.Sprintf("cov_sweep_%d", time.Now().UnixNano())
		if err := q.Exec(ctx, `INSERT INTO auth_register_verification_codes (request_id, login_id, verification_code, expires_at) VALUES ($1, $2, $3, $4)`,
			requestID, loginID, "123456", time.Now().UTC().AddDate(0, 0, -8)); err != nil {
			t.Fatalf("seed expired register code: %v", err)
		}
		m := NewModule(ctx, testConfig(), txFactory, appconfig.NewStaticSource(testConfig()), nil)
		m.sweepRetention(ctx)
		var n int
		if err := q.QueryRow(ctx, `SELECT COUNT(*) FROM auth_register_verification_codes WHERE request_id = $1`, requestID).Scan(&n); err != nil {
			t.Fatalf("count swept row: %v", err)
		}
		if n != 0 {
			t.Fatalf("expired register code was not swept: %d", n)
		}
	})
}
