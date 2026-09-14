package auth

import (
	"context"
	"testing"
	"time"
)

// 验证码防爆破（内存实现；PG 实现同逻辑，见 repo_postgres.go）：
// 最新有效码累计失败满 maxVerificationCodeFailures 次即静默作废，
// 对外仍返回原错误（重置 404 / 注册 10401 口径），不新增可探测口径。

func wrongCodeNotEqual(t *testing.T, real string) string {
	t.Helper()
	for _, candidate := range []string{"000000", "111111", "999999"} {
		if candidate != real {
			return candidate
		}
	}
	t.Fatal("cannot pick a wrong code distinct from the real one")
	return ""
}

func TestResetPasswordToleratesFewFailuresThenBurns(t *testing.T) {
	ctx := context.Background()
	newRepoWithAccount := func(loginID string) *InMemoryRepository {
		repo := NewInMemoryRepository()
		if _, err := repo.Register(ctx, RegisterParams{LoginID: loginID, Nickname: loginID, Password: "password1"}); err != nil {
			t.Fatalf("seed register: %v", err)
		}
		return repo
	}

	// 阈值内失败不影响正确码。
	repo := newRepoWithAccount("13800138001")
	if _, err := repo.CreatePasswordReset(ctx, "13800138001", time.Now().UTC().Add(10*time.Minute)); err != nil {
		t.Fatalf("create reset: %v", err)
	}
	code := repo.LastResetCode("13800138001")
	wrong := wrongCodeNotEqual(t, code)
	for i := 0; i < maxVerificationCodeFailures-1; i++ {
		if _, err := repo.ResetPassword(ctx, "13800138001", wrong, "newpass1"); err != ErrPasswordResetNotFound {
			t.Fatalf("attempt %d: expected NotFound, got %v", i+1, err)
		}
	}
	if _, err := repo.ResetPassword(ctx, "13800138001", code, "newpass1"); err != nil {
		t.Fatalf("correct code within tolerance must succeed, got %v", err)
	}

	// 满阈值失败后正确码同样失效（已烧码，须重新发送）。
	repo2 := newRepoWithAccount("13800138002")
	if _, err := repo2.CreatePasswordReset(ctx, "13800138002", time.Now().UTC().Add(10*time.Minute)); err != nil {
		t.Fatalf("create reset: %v", err)
	}
	code2 := repo2.LastResetCode("13800138002")
	wrong2 := wrongCodeNotEqual(t, code2)
	for i := 0; i < maxVerificationCodeFailures; i++ {
		if _, err := repo2.ResetPassword(ctx, "13800138002", wrong2, "newpass1"); err != ErrPasswordResetNotFound {
			t.Fatalf("attempt %d: expected NotFound, got %v", i+1, err)
		}
	}
	if _, err := repo2.ResetPassword(ctx, "13800138002", code2, "newpass1"); err != ErrPasswordResetNotFound {
		t.Fatalf("expected burned code to be rejected, got %v", err)
	}
}

func TestVerifyRegisterCodeToleratesFewFailuresThenBurns(t *testing.T) {
	ctx := context.Background()
	// 阈值内失败不影响正确码。
	repo := NewInMemoryRepository()
	if _, err := repo.CreateRegisterCode(ctx, "a@b.com", time.Now().UTC().Add(10*time.Minute)); err != nil {
		t.Fatalf("create code: %v", err)
	}
	code := repo.LastRegisterCode("a@b.com")
	wrong := wrongCodeNotEqual(t, code)
	for i := 0; i < maxVerificationCodeFailures-1; i++ {
		if err := repo.VerifyRegisterCode(ctx, "a@b.com", wrong); err != ErrInvalidRegisterCode {
			t.Fatalf("attempt %d: expected InvalidRegisterCode, got %v", i+1, err)
		}
	}
	if err := repo.VerifyRegisterCode(ctx, "a@b.com", code); err != nil {
		t.Fatalf("correct code within tolerance must verify, got %v", err)
	}

	// 满阈值失败后正确码同样失效。
	repo2 := NewInMemoryRepository()
	if _, err := repo2.CreateRegisterCode(ctx, "c@d.com", time.Now().UTC().Add(10*time.Minute)); err != nil {
		t.Fatalf("create code: %v", err)
	}
	code2 := repo2.LastRegisterCode("c@d.com")
	wrong2 := wrongCodeNotEqual(t, code2)
	for i := 0; i < maxVerificationCodeFailures; i++ {
		if err := repo2.VerifyRegisterCode(ctx, "c@d.com", wrong2); err != ErrInvalidRegisterCode {
			t.Fatalf("attempt %d: expected InvalidRegisterCode, got %v", i+1, err)
		}
	}
	if err := repo2.VerifyRegisterCode(ctx, "c@d.com", code2); err != ErrInvalidRegisterCode {
		t.Fatalf("expected burned code to be rejected, got %v", err)
	}
}
