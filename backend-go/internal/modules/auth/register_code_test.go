package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/appconfig"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/testdb"
	"hotspot/backend-go/internal/platform/types"
)

// --- login_id 格式三分 ---

func TestClassifyLoginID(t *testing.T) {
	cases := []struct {
		loginID string
		kind    loginIDKind
	}{
		{"13800138000", loginIDKindPhone},
		{"12345", loginIDKindPhone},
		{"a@b.com", loginIDKindEmail},
		{"alice", loginIDKindUsername},
		{"user_01", loginIDKindUsername},
		{"a@b", loginIDKindInvalid},    // 缺顶级域名
		{"@b.com", loginIDKindInvalid}, // 缺 local 部分
	}
	for _, tc := range cases {
		if got := classifyLoginID(tc.loginID); got != tc.kind {
			t.Fatalf("classifyLoginID(%q) = %d, want %d", tc.loginID, got, tc.kind)
		}
	}
}

func TestIsValidAuthLoginID(t *testing.T) {
	for _, id := range []string{"alice", "13800138000", "a@b.com", "user_01", strings.Repeat("a", 64)} {
		if !isValidAuthLoginID(id) {
			t.Fatalf("expected %q valid", id)
		}
	}
	for _, id := range []string{"", "ab", strings.Repeat("a", 65), "bad@", "@b.com", "a@b"} {
		if isValidAuthLoginID(id) {
			t.Fatalf("expected %q invalid", id)
		}
	}
}

// --- send_register_code ---

func TestHandlerSendRegisterCodeHappyPath(t *testing.T) {
	handler := newTestHandler()
	// P2-3 后投递失败不再被吞：testConfig 的 AppEnv 为空 → production-like
	// → 默认 noop 网关（恒失败）。happy-path 用例须显式注入可用投递通道。
	handler.service.codeSender = &logVerificationSender{}
	req := httptest.NewRequest(http.MethodPost, "/send_register_code", strings.NewReader(`{"login_id":"13800138000","client_type":"app"}`))
	rr := httptest.NewRecorder()
	handler.SendRegisterCode(rr, req)
	if code := testdb.DecodeAPICode(t, rr); code != 0 {
		t.Fatalf("expected success, got %d", code)
	}
}

// 用户名型 login_id 返回 10000（无第三方所有权，无需验证码）。
func TestHandlerSendRegisterCodeRejectsUsernameLoginID(t *testing.T) {
	handler := newTestHandler()
	for _, body := range []string{
		`{"login_id":"alice","client_type":"app"}`,
		`{"login_id":"bad@@example","client_type":"app"}`,
		`{"login_id":"","client_type":"app"}`,
		`{"login_id":"13800138000","client_type":"bogus"}`,
	} {
		req := httptest.NewRequest(http.MethodPost, "/send_register_code", strings.NewReader(body))
		rr := httptest.NewRecorder()
		handler.SendRegisterCode(rr, req)
		if code := testdb.DecodeAPICode(t, rr); code != httpx.CodeInvalidRequest {
			t.Fatalf("body %s: expected 10000, got %d", body, code)
		}
	}
}

// 防枚举：已注册与未注册的 login_id 均成功（内存 repo 落码不查账号，
// 与 CreatePasswordReset 的「账号不存在不落码」形成对照）。
func TestServiceSendRegisterCodeAntiEnumeration(t *testing.T) {
	repo := NewInMemoryRepository()
	if _, err := repo.Register(context.Background(), RegisterParams{LoginID: "13800138000", Nickname: "已注册", Password: "password1"}); err != nil {
		t.Fatalf("seed register: %v", err)
	}
	cfg := testConfig()
	svc := NewServiceWithSource(context.Background(), platformauth.NewService(cfg), repo, cfg, appconfig.NewStaticSource(cfg))
	// P2-3 后 noop 网关（production-like 默认）投递失败会透传，注入可用通道。
	svc.codeSender = &logVerificationSender{}

	for _, loginID := range []string{"13800138000", "13900139000"} {
		if err := svc.SendRegisterCode(context.Background(), SendRegisterCodeRequest{LoginID: loginID, ClientType: ClientTypeApp}); err != nil {
			t.Fatalf("send code for %s: %v", loginID, err)
		}
		if repo.LastRegisterCode(loginID) == "" {
			t.Fatalf("expected code stored for %s", loginID)
		}
	}
}

// 内存 repo 验证码流程：生成 → 校验 → 消费 → 一次性。
func TestInMemoryRegisterCodeLifecycle(t *testing.T) {
	repo := NewInMemoryRepository()
	ctx := context.Background()
	if _, err := repo.CreateRegisterCode(ctx, "a@b.com", time.Now().UTC().Add(10*time.Minute)); err != nil {
		t.Fatalf("create code: %v", err)
	}
	code := repo.LastRegisterCode("a@b.com")
	if code == "" {
		t.Fatal("expected code to be stored")
	}
	if err := repo.VerifyRegisterCode(ctx, "a@b.com", code); err != nil {
		t.Fatalf("verify: %v", err)
	}
	if err := repo.ConsumeRegisterCode(ctx, "a@b.com", code); err != nil {
		t.Fatalf("consume: %v", err)
	}
	if err := repo.VerifyRegisterCode(ctx, "a@b.com", code); err != ErrInvalidRegisterCode {
		t.Fatalf("expected consumed code to be rejected, got %v", err)
	}
	// 过期码无效
	if _, err := repo.CreateRegisterCode(ctx, "13800138000", time.Now().UTC().Add(-time.Minute)); err != nil {
		t.Fatalf("create expired code: %v", err)
	}
	expired := repo.LastRegisterCode("13800138000")
	if err := repo.VerifyRegisterCode(ctx, "13800138000", expired); err != ErrInvalidRegisterCode {
		t.Fatalf("expected expired code to be rejected, got %v", err)
	}
}

// --- register 验证码校验 ---

// 手机号/邮箱型注册：缺码 10000、错码 10401、用户名型带码 10000。
func TestHandlerRegisterVerificationCodeRules(t *testing.T) {
	handler := newTestHandler()
	cases := []struct {
		name string
		body string
		code httpx.Code
	}{
		{"missing code for phone", `{"login_id":"13800138000","nickname":"A","password":"password1","client_type":"app","device_id":"d1"}`, httpx.CodeInvalidRequest},
		{"missing code for email", `{"login_id":"a@b.com","nickname":"A","password":"password1","client_type":"app","device_id":"d1"}`, httpx.CodeInvalidRequest},
		{"username with code", `{"login_id":"alice","nickname":"A","password":"password1","verification_code":"123456","client_type":"app","device_id":"d1"}`, httpx.CodeInvalidRequest},
		{"bad login_id format", `{"login_id":"a@b","nickname":"A","password":"password1","client_type":"app","device_id":"d1"}`, httpx.CodeInvalidRequest},
		{"login_id too short", `{"login_id":"ab","nickname":"A","password":"password1","client_type":"app","device_id":"d1"}`, httpx.CodeInvalidRequest},
	}
	for i, tc := range cases {
		req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(tc.body))
		req.Header.Set("Idempotency-Key", strings.Repeat("0", 31)+string(rune('a'+i)))
		rr := httptest.NewRecorder()
		handler.Register(rr, req)
		if code := testdb.DecodeAPICode(t, rr); code != tc.code {
			t.Fatalf("%s: expected %d, got %d", tc.name, tc.code, code)
		}
	}
}

// 错误/过期验证码 → 10401（CodeInvalidOldCredential）。
func TestServiceRegisterRejectsInvalidCode(t *testing.T) {
	repo := &stubAuthRepo{verifyCodeErr: ErrInvalidRegisterCode, registerAccount: Account{UserID: "u1"}}
	svc, _ := newTestService(repo)
	_, err := svc.Register(context.Background(), RegisterRequest{LoginID: "a@b.com", Password: "p", VerificationCode: types.StrPtr("000000")}, defaultMeta())
	if err == nil || !strings.Contains(err.Error(), ErrInvalidRegisterCode.Error()) {
		t.Fatalf("expected invalid register code error, got %v", err)
	}
	if apiErr := apiErrorFrom(err); apiErr.Code != httpx.CodeInvalidOldCredential {
		t.Fatalf("expected 10401 mapping, got %#v", apiErr)
	}
}

// 完整流程：send → register（码正确）→ 码被消费不可重放。
func TestServiceRegisterWithCodeEndToEnd(t *testing.T) {
	repo := NewInMemoryRepository()
	cfg := testConfig()
	svc := NewServiceWithSource(context.Background(), platformauth.NewService(cfg), repo, cfg, appconfig.NewStaticSource(cfg))
	// P2-3 后 noop 网关（production-like 默认）投递失败会透传，注入可用通道。
	svc.codeSender = &logVerificationSender{}
	ctx := context.Background()
	if err := svc.SendRegisterCode(ctx, SendRegisterCodeRequest{LoginID: "a@b.com", ClientType: ClientTypeApp}); err != nil {
		t.Fatalf("send code: %v", err)
	}
	code := repo.LastRegisterCode("a@b.com")
	if _, err := svc.Register(ctx, RegisterRequest{LoginID: "a@b.com", Nickname: "Alice", Password: "password1", VerificationCode: &code, ClientType: ClientTypeApp, DeviceID: "d1"}, defaultMeta()); err != nil {
		t.Fatalf("register with code: %v", err)
	}
	// 一次性：已消费的码再校验失败
	if err := repo.VerifyRegisterCode(ctx, "a@b.com", code); err != ErrInvalidRegisterCode {
		t.Fatalf("expected consumed code to fail replay, got %v", err)
	}
}

// --- guest 默认昵称 ---

func TestGenerateGuestNickname(t *testing.T) {
	nickname, err := generateGuestNickname()
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if !strings.HasPrefix(nickname, "游客·") || len(nickname) != len("游客·")+4 {
		t.Fatalf("unexpected nickname format: %q", nickname)
	}
	for _, r := range nickname[len("游客·"):] {
		if (r < '0' || r > '9') && (r < 'a' || r > 'z') {
			t.Fatalf("suffix must be [0-9a-z], got %q", nickname)
		}
	}
	// 随机熵：两次生成应几乎必然不同
	other, _ := generateGuestNickname()
	if nickname == other {
		t.Fatalf("expected distinct nicknames, got %q twice", nickname)
	}
}

// Nickname null → 服务端生成默认昵称落库。
func TestServiceGuestGeneratesNicknameWhenNil(t *testing.T) {
	repo := &stubAuthRepo{guestAccount: Account{UserID: "g1"}}
	svc, _ := newTestService(repo)
	result, err := svc.Guest(context.Background(), GuestRequest{}, defaultMeta())
	if err != nil {
		t.Fatalf("guest: %v", err)
	}
	_ = result
}

// handler：nickname 缺省（null）合法；显式空串仍 10000。
func TestHandlerGuestNicknameOptional(t *testing.T) {
	handler := newTestHandler()
	req := httptest.NewRequest(http.MethodPost, "/guest", strings.NewReader(`{"client_type":"app","device_id":"d1"}`))
	req.Header.Set("Idempotency-Key", "guest_no_nick_0000000000000000000000000")
	rr := httptest.NewRecorder()
	handler.Guest(rr, req)
	if code := testdb.DecodeAPICode(t, rr); code != 0 {
		t.Fatalf("expected success without nickname, got %d", code)
	}

	emptyReq := httptest.NewRequest(http.MethodPost, "/guest", strings.NewReader(`{"nickname":"","client_type":"app","device_id":"d1"}`))
	emptyReq.Header.Set("Idempotency-Key", "guest_empty_nick_00000000000000000000000")
	emptyRR := httptest.NewRecorder()
	handler.Guest(emptyRR, emptyReq)
	if code := testdb.DecodeAPICode(t, emptyRR); code != httpx.CodeInvalidRequest {
		t.Fatalf("expected 10000 for empty nickname, got %d", code)
	}
}

// --- signin login_id 三分校验 ---

func TestHandlerLoginValidatesLoginIDFormat(t *testing.T) {
	handler := newTestHandler()
	for _, body := range []string{
		`{"login_id":"ab","password":"x","client_type":"app","device_id":"d"}`,
		`{"login_id":"a@b","password":"x","client_type":"app","device_id":"d"}`,
	} {
		req := httptest.NewRequest(http.MethodPost, "/signin", strings.NewReader(body))
		rr := httptest.NewRecorder()
		handler.Login(rr, req)
		if code := testdb.DecodeAPICode(t, rr); code != httpx.CodeInvalidRequest {
			t.Fatalf("body %s: expected 10000, got %d", body, code)
		}
	}
}
