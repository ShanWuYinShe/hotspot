package auth

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"hotspot/backend-go/internal/platform/appconfig"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/idempotency"
)

func newP2Service() (*Service, *InMemoryRepository) {
	cfg := testConfig()
	repo := NewInMemoryRepository()
	svc := NewServiceWithSource(context.Background(), platformauth.NewService(cfg), repo, cfg, appconfig.NewStaticSource(cfg))
	return svc, repo
}

// 审查 P2-3 防回归：验证码投递失败必须向客户端透传（noop 网关 → 10600），
// 不再吞错假成功——生产漏配 SMTP 时用户拿到「已发送」却永远等不到码。
func TestSendRegisterCodePropagatesDeliveryFailure(t *testing.T) {
	svc, _ := newP2Service()
	svc.codeSender = &noopVerificationSender{}

	err := svc.SendRegisterCode(context.Background(), SendRegisterCodeRequest{LoginID: "a@b.com", ClientType: ClientTypeApp})
	if err == nil {
		t.Fatal("expected delivery failure to propagate, got nil")
	}
	if !errors.Is(err, ErrVerificationServiceUnavailable) {
		t.Fatalf("expected ErrVerificationServiceUnavailable in chain, got %v", err)
	}
	if apiErr := apiErrorFrom(err); apiErr.Code != httpx.CodeInternalUnavailable {
		t.Fatalf("expected 10600 mapping, got %d", apiErr.Code)
	}
}

// 审查 P2-3 防回归：密码重置码投递失败同样透传（已存在账号才有投递环节；
// 防枚举权衡见 service 内注释——正常网关下两端同为成功）。节流路径仍静默。
func TestRequestPasswordResetPropagatesDeliveryFailure(t *testing.T) {
	svc, repo := newP2Service()
	if _, err := repo.Register(context.Background(), RegisterParams{LoginID: "alice", Nickname: "Alice", Password: "password1"}); err != nil {
		t.Fatalf("seed account: %v", err)
	}
	svc.codeSender = &noopVerificationSender{}

	if err := svc.RequestPasswordReset(context.Background(), RequestPasswordResetRequest{LoginID: "alice", ClientType: ClientTypeApp}); err == nil {
		t.Fatal("expected delivery failure to propagate for existing account")
	}
}

// 审查 P2-21 防回归：Register 昵称先 TrimSpace 再校验（对齐 Guest）——
// 纯空格昵称 10000；带空白昵称落库为去空白后的值。
func TestRegisterTrimsNicknameBeforeValidation(t *testing.T) {
	svc, repo := newP2Service()
	handler := NewHandler(svc)
	idempotency.SetStore(idempotency.NewInMemoryStore())

	// 纯空格：TrimSpace 后为空 → 10000。
	body1 := `{"login_id":"spacetest","nickname":"   ","password":"password1","client_type":"app","device_id":"dev-1"}`
	req := httptest.NewRequest("POST", "/register", strings.NewReader(body1))
	req.Header.Set("Idempotency-Key", "aaaa1234abcd1234abcd1234abcd1234")
	rr := httptest.NewRecorder()
	handler.Register(rr, req)
	if !strings.Contains(rr.Body.String(), `"code":10000`) {
		t.Fatalf("expected 10000 for whitespace-only nickname, got %s", rr.Body.String())
	}

	// 带空白：通过校验且落库为去空白后的值。
	idempotency.SetStore(idempotency.NewInMemoryStore())
	body2 := `{"login_id":"trimtest","nickname":"  Alice  ","password":"password1","client_type":"app","device_id":"dev-1"}`
	req2 := httptest.NewRequest("POST", "/register", strings.NewReader(body2))
	req2.Header.Set("Idempotency-Key", "bbbb1234abcd1234abcd1234abcd1234")
	rr2 := httptest.NewRecorder()
	handler.Register(rr2, req2)
	if !strings.Contains(rr2.Body.String(), `"code":0`) {
		t.Fatalf("expected success for padded nickname, got %s", rr2.Body.String())
	}
	account, err := repo.Authenticate(context.Background(), "trimtest", "password1")
	if err != nil {
		t.Fatalf("authenticate trimmed account: %v", err)
	}
	if account.Nickname != "Alice" {
		t.Fatalf("expected stored nickname trimmed to Alice, got %q", account.Nickname)
	}
}
