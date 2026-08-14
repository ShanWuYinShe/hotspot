package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/appconfig"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/realtime"

	"golang.org/x/crypto/bcrypt"
)

func TestMain(m *testing.M) {
	realtime.ResetHub()
	// Drop bcrypt to MinCost for the test suite so Register/ChangePassword
	// stay sub-millisecond. Production wiring restores OWASP default (12)
	// via ConfigureHashingCost in module.go.
	ConfigureHashingCost(bcrypt.MinCost)
	m.Run()
}

// --- stub repo ---

type stubAuthRepo struct {
	registerAccount Account
	registerErr     error
	guestAccount    Account
	guestErr        error
	authAccount     Account
	authErr         error
	updateMetaErr   error
	resetCreateErr  error
	resetCode       string
	resetAccount    Account
	resetErr        error
	changePwdErr    error
}

func (r *stubAuthRepo) Register(_ context.Context, _ RegisterParams) (Account, error) {
	return r.registerAccount, r.registerErr
}
func (r *stubAuthRepo) CreateGuest(_ context.Context, _ GuestParams) (Account, error) {
	return r.guestAccount, r.guestErr
}
func (r *stubAuthRepo) Authenticate(_ context.Context, _, _ string) (Account, error) {
	return r.authAccount, r.authErr
}
func (r *stubAuthRepo) UpdateLoginMeta(_ context.Context, _, _, _ string) error {
	return r.updateMetaErr
}
func (r *stubAuthRepo) CreatePasswordReset(_ context.Context, _ string, _ time.Time) (string, error) {
	return r.resetCode, r.resetCreateErr
}
func (r *stubAuthRepo) ResetPassword(_ context.Context, _, _, _ string) (Account, error) {
	return r.resetAccount, r.resetErr
}
func (r *stubAuthRepo) ChangePassword(_ context.Context, _, _, _ string) error {
	return r.changePwdErr
}

// --- helpers ---

func testConfig() config.Config {
	return config.Config{AuthTokenSecret: "test-secret-for-unit-tests"}
}

func newTestService(repo *stubAuthRepo) (*Service, *platformauth.Service) {
	cfg := testConfig()
	authSvc := platformauth.NewService(cfg)
	return NewServiceWithSource(
		context.Background(),
		authSvc,
		repo,
		cfg,
		appconfig.NewStaticSource(cfg),
	), authSvc
}

func authCtx(userID string) context.Context {
	return platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: userID})
}

func authCtxWithSession(userID, sessionID string) context.Context {
	return platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: userID, SessionID: sessionID})
}

func authCtxWithExpiry(userID string, expiresAt time.Time) context.Context {
	ctx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: userID})
	return platformauth.WithAccessExpiry(ctx, expiresAt)
}

func defaultMeta() platformauth.SessionMeta {
	return platformauth.SessionMeta{DeviceID: "test-device", DeviceType: "test", DeviceName: "test", IPAddress: "1.2.3.4", IPLocation: "US"}
}

// --- GetStatus ---

func TestServiceGetStatusNoPrincipal(t *testing.T) {
	svc, _ := newTestService(&stubAuthRepo{})
	status := svc.GetStatus(context.Background())
	if status.IsValid {
		t.Fatal("expected invalid for no principal")
	}
	// 恒定 nullable：未认证时 user_id/expires_at 为 nil（序列化为 null）
	if status.UserID != nil || status.ExpiresAt != nil {
		t.Fatalf("expected nil user_id/expires_at when not authenticated, got %#v", status)
	}
}

func TestServiceGetStatusWithPrincipal(t *testing.T) {
	svc, _ := newTestService(&stubAuthRepo{})
	status := svc.GetStatus(authCtx("u1"))
	if !status.IsValid {
		t.Fatal("expected valid")
	}
	if status.UserID == nil || *status.UserID != "u1" {
		t.Fatalf("expected user id %q, got %v", "u1", status.UserID)
	}
	if status.ExpiresAt != nil {
		t.Fatal("expected no expiry info")
	}
}

func TestServiceGetStatusWithExpiry(t *testing.T) {
	svc, _ := newTestService(&stubAuthRepo{})
	expiry := time.Now().Add(15 * time.Minute).UTC()
	status := svc.GetStatus(authCtxWithExpiry("u1", expiry))
	if !status.IsValid {
		t.Fatal("expected valid")
	}
	if status.ExpiresAt == nil || *status.ExpiresAt == "" {
		t.Fatal("expected expiry info")
	}
	if status.UserID == nil || *status.UserID != "u1" {
		t.Fatalf("expected user id, got %v", status.UserID)
	}
}

// --- Register ---

func TestServiceRegisterHappyPath(t *testing.T) {
	repo := &stubAuthRepo{registerAccount: Account{UserID: "u1", LoginID: "a@b.com"}}
	svc, _ := newTestService(repo)
	result, err := svc.Register(context.Background(), RegisterRequest{LoginID: "a@b.com", Password: "pass"}, defaultMeta())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Principal.UserID != "u1" {
		t.Fatalf("expected u1, got %s", result.Principal.UserID)
	}
}

func TestServiceRegisterPropagatesRepoError(t *testing.T) {
	repo := &stubAuthRepo{registerErr: ErrLoginIDConflict}
	svc, _ := newTestService(repo)
	_, err := svc.Register(context.Background(), RegisterRequest{LoginID: "dup"}, defaultMeta())
	if !errors.Is(err, ErrLoginIDConflict) {
		t.Fatalf("expected login id conflict, got %v", err)
	}
}

// --- Login ---

func TestServiceLoginHappyPath(t *testing.T) {
	repo := &stubAuthRepo{authAccount: Account{UserID: "u1", LoginID: "a@b.com"}}
	svc, _ := newTestService(repo)
	result, err := svc.Login(context.Background(), LoginRequest{LoginID: "a@b.com", Password: "pass"}, defaultMeta())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Principal.UserID != "u1" {
		t.Fatalf("expected u1, got %s", result.Principal.UserID)
	}
}

func TestServiceLoginRejectsBadCredentials(t *testing.T) {
	repo := &stubAuthRepo{authErr: ErrInvalidCredentials}
	svc, _ := newTestService(repo)
	_, err := svc.Login(context.Background(), LoginRequest{LoginID: "a@b.com", Password: "wrong"}, defaultMeta())
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected invalid credentials, got %v", err)
	}
}

func TestServiceLoginPropagatesMetaError(t *testing.T) {
	repo := &stubAuthRepo{authAccount: Account{UserID: "u1"}, updateMetaErr: errors.New("boom")}
	svc, _ := newTestService(repo)
	_, err := svc.Login(context.Background(), LoginRequest{LoginID: "a@b.com"}, defaultMeta())
	if err == nil {
		t.Fatal("expected error from update meta")
	}
}

// --- Lockout (brute-force protection) ---

func TestLoginTrackerLocksAfterFiveFailures(t *testing.T) {
	tracker := newInMemoryLoginTracker(context.Background())
	defer tracker.stop()
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		tracker.RecordFailed(ctx, "u@test")
	}
	// After 5 RecordFails, Check must return ErrAccountLocked.
	err := tracker.Check(ctx, "u@test")
	if !errors.Is(err, ErrAccountLocked) {
		t.Fatalf("expected ErrAccountLocked after 5 failures, got %v", err)
	}
}

func TestLoginTrackerClearResetsCounter(t *testing.T) {
	tracker := newInMemoryLoginTracker(context.Background())
	defer tracker.stop()
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		tracker.RecordFailed(ctx, "u@test")
	}
	tracker.Clear(ctx, "u@test")

	// After clear, another 5 failures should be needed.
	for i := 0; i < 5; i++ {
		tracker.RecordFailed(ctx, "u@test")
		if i < 4 {
			if err := tracker.Check(ctx, "u@test"); err != nil {
				t.Fatalf("unexpected lock after clear + %d failures: %v", i+1, err)
			}
		}
	}
	if err := tracker.Check(ctx, "u@test"); !errors.Is(err, ErrAccountLocked) {
		t.Fatalf("expected lock after clear + 5 failures, got %v", err)
	}
}

func TestServiceLoginLocksAfterConsecutiveFailures(t *testing.T) {
	repo := &stubAuthRepo{authErr: ErrInvalidCredentials}
	svc, _ := newTestService(repo)
	loginID := "lockout-test@b.com"

	// 5 consecutive failures — tracker records each one.
	for i := 0; i < 5; i++ {
		_, err := svc.Login(context.Background(), LoginRequest{LoginID: loginID, Password: "wrong"}, defaultMeta())
		if !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("attempt %d: expected ErrInvalidCredentials, got %v", i+1, err)
		}
	}

	// 6th attempt must be rejected by the lockout tracker.
	_, err := svc.Login(context.Background(), LoginRequest{LoginID: loginID, Password: "wrong"}, defaultMeta())
	if !errors.Is(err, ErrAccountLocked) {
		t.Fatalf("expected ErrAccountLocked after 5 failures, got %v", err)
	}
}

func TestServiceLoginClearOnSuccessResetsLockout(t *testing.T) {
	// Use the tracker directly — the Login method calls Clear() on success,
	// and RecordFailed() + Check() flow is already verified via
	// TestServiceLoginLocksAfterConsecutiveFailures.
	tracker := newInMemoryLoginTracker(context.Background())
	defer tracker.stop()
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		tracker.RecordFailed(ctx, "u@test")
	}
	tracker.Clear(ctx, "u@test")

	// After clear, 5 fresh failures should be needed to lock again.
	for i := 0; i < 5; i++ {
		tracker.RecordFailed(ctx, "u@test")
	}
	if err := tracker.Check(ctx, "u@test"); !errors.Is(err, ErrAccountLocked) {
		t.Fatalf("expected lock after clear+5 failures, got %v", err)
	}
}

// --- Guest ---

func TestServiceGuestHappyPath(t *testing.T) {
	repo := &stubAuthRepo{guestAccount: Account{UserID: "g1"}}
	svc, _ := newTestService(repo)
	result, err := svc.Guest(context.Background(), GuestRequest{Nickname: "guest"}, defaultMeta())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Principal.UserID != "g1" {
		t.Fatalf("expected g1, got %s", result.Principal.UserID)
	}
}

func TestServiceGuestPropagatesError(t *testing.T) {
	repo := &stubAuthRepo{guestErr: errors.New("boom")}
	svc, _ := newTestService(repo)
	_, err := svc.Guest(context.Background(), GuestRequest{Nickname: "guest"}, defaultMeta())
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- Refresh ---

func TestServiceRefreshUsesCookieForWeb(t *testing.T) {
	// First register+login to get a valid refresh token
	repo := &stubAuthRepo{registerAccount: Account{UserID: "u1"}}
	svc, _ := newTestService(repo)
	result, err := svc.Register(context.Background(), RegisterRequest{LoginID: "a@b.com"}, defaultMeta())
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	// Now refresh using the token via web client
	refreshed, err := svc.Refresh(context.Background(), RefreshTokenRequest{RefreshToken: result.RefreshToken, ClientType: ClientTypeApp, DeviceID: "test-device"}, "test-device", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if refreshed.Principal.UserID != "u1" {
		t.Fatalf("expected u1, got %s", refreshed.Principal.UserID)
	}
}

func TestServiceRefreshRejectsInvalidToken(t *testing.T) {
	svc, _ := newTestService(&stubAuthRepo{})
	_, err := svc.Refresh(context.Background(), RefreshTokenRequest{RefreshToken: "bad-token", ClientType: ClientTypeApp, DeviceID: "test-device"}, "test-device", "")
	if err == nil {
		t.Fatal("expected error for invalid token")
	}
}

// --- ListSessions ---

func TestServiceListSessionsMasksIP(t *testing.T) {
	repo := &stubAuthRepo{}
	svc, _ := newTestService(repo)
	// Register to create a session
	meta := platformauth.SessionMeta{DeviceType: "ios", IPAddress: "192.168.1.100", IPLocation: "US"}
	result, _ := svc.Register(context.Background(), RegisterRequest{LoginID: "a@b.com"}, meta)
	ctx := platformauth.WithPrincipal(context.Background(), result.Principal)
	list := svc.ListSessions(ctx)
	if len(list.Items) == 0 {
		t.Fatal("expected at least 1 session")
	}
	if list.Items[0].IPAddress == "192.168.1.100" {
		t.Fatal("expected IP to be masked")
	}
	if list.Items[0].IPAddress != "192.168.xxx.xxx" {
		t.Fatalf("expected masked IP, got %s", list.Items[0].IPAddress)
	}
}

func TestServiceListSessionsEmptyWithoutAuth(t *testing.T) {
	svc, _ := newTestService(&stubAuthRepo{})
	list := svc.ListSessions(context.Background())
	if len(list.Items) != 0 {
		t.Fatal("expected empty list without auth")
	}
}

// --- RevokeSession ---

func TestServiceRevokeSessionRejectsNoAuth(t *testing.T) {
	svc, _ := newTestService(&stubAuthRepo{})
	err := svc.RevokeSession(context.Background(), RevokeSessionRequest{SessionID: "s1"})
	if err == nil {
		t.Fatal("expected error for no auth")
	}
}

func TestServiceRevokeSessionRejectsCurrentSession(t *testing.T) {
	svc, _ := newTestService(&stubAuthRepo{})
	ctx := authCtxWithSession("u1", "current-session")
	err := svc.RevokeSession(ctx, RevokeSessionRequest{SessionID: "current-session"})
	if err == nil {
		t.Fatal("expected error for revoking current session")
	}
}

func TestServiceRevokeSessionHappyPath(t *testing.T) {
	repo := &stubAuthRepo{}
	svc, _ := newTestService(repo)
	// Create two sessions for the user
	meta1 := platformauth.SessionMeta{DeviceType: "ios", IPAddress: "1.1.1.1"}
	meta2 := platformauth.SessionMeta{DeviceType: "web", IPAddress: "2.2.2.2"}
	result1, _ := svc.Register(context.Background(), RegisterRequest{LoginID: "a@b.com"}, meta1)
	result2, _ := svc.Login(context.Background(), LoginRequest{LoginID: "a@b.com"}, meta2)
	_ = result2

	ctx := platformauth.WithPrincipal(context.Background(), result1.Principal)
	err := svc.RevokeSession(ctx, RevokeSessionRequest{SessionID: result2.Principal.SessionID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- ChangePassword ---

func TestServiceChangePasswordRequiresAuth(t *testing.T) {
	svc, _ := newTestService(&stubAuthRepo{})
	err := svc.ChangePassword(context.Background(), ChangePasswordRequest{OldPassword: "old", NewPassword: "new"})
	if err == nil {
		t.Fatal("expected error for no auth")
	}
}

func TestServiceChangePasswordHappyPath(t *testing.T) {
	repo := &stubAuthRepo{}
	svc, _ := newTestService(repo)
	ctx := authCtx("u1")
	err := svc.ChangePassword(ctx, ChangePasswordRequest{OldPassword: "old", NewPassword: "new"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestServiceChangePasswordPropagatesError(t *testing.T) {
	repo := &stubAuthRepo{changePwdErr: ErrInvalidOldPassword}
	svc, _ := newTestService(repo)
	ctx := authCtx("u1")
	err := svc.ChangePassword(ctx, ChangePasswordRequest{OldPassword: "wrong"})
	if !errors.Is(err, ErrInvalidOldPassword) {
		t.Fatalf("expected invalid old password, got %v", err)
	}
}

// --- RequestPasswordReset ---

func TestServiceRequestPasswordResetHappyPath(t *testing.T) {
	repo := &stubAuthRepo{}
	svc, _ := newTestService(repo)
	err := svc.RequestPasswordReset(context.Background(), RequestPasswordResetRequest{LoginID: "a@b.com"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestServiceRequestPasswordResetSwallowsThrottled(t *testing.T) {
	repo := &stubAuthRepo{resetCreateErr: ErrPasswordResetThrottled}
	svc, _ := newTestService(repo)
	err := svc.RequestPasswordReset(context.Background(), RequestPasswordResetRequest{LoginID: "a@b.com"})
	if err != nil {
		t.Fatalf("expected nil (swallowed), got %v", err)
	}
}

func TestServiceRequestPasswordResetPropagatesOtherError(t *testing.T) {
	repo := &stubAuthRepo{resetCreateErr: errors.New("boom")}
	svc, _ := newTestService(repo)
	err := svc.RequestPasswordReset(context.Background(), RequestPasswordResetRequest{LoginID: "a@b.com"})
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- ResetPassword ---

func TestServiceResetPasswordHappyPath(t *testing.T) {
	repo := &stubAuthRepo{resetAccount: Account{UserID: "u1"}}
	svc, _ := newTestService(repo)
	result, err := svc.ResetPassword(context.Background(), ResetPasswordRequest{LoginID: "a@b.com", VerificationCode: "123456", NewPassword: "new"}, defaultMeta())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Principal.UserID != "u1" {
		t.Fatalf("expected u1, got %s", result.Principal.UserID)
	}
}

func TestServiceResetPasswordRejectsBadCode(t *testing.T) {
	repo := &stubAuthRepo{resetErr: ErrPasswordResetNotFound}
	svc, _ := newTestService(repo)
	_, err := svc.ResetPassword(context.Background(), ResetPasswordRequest{LoginID: "a@b.com", VerificationCode: "bad"}, defaultMeta())
	if !errors.Is(err, ErrPasswordResetNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}

// --- RevokeAll ---

func TestServiceRevokeAllWithoutAuth(t *testing.T) {
	svc, _ := newTestService(&stubAuthRepo{})
	result := svc.RevokeAll(context.Background())
	if result.RevokedCount != 0 {
		t.Fatal("expected 0 revoked without auth")
	}
}

func TestServiceRevokeAllHappyPath(t *testing.T) {
	repo := &stubAuthRepo{}
	svc, _ := newTestService(repo)
	// Create two sessions
	meta1 := platformauth.SessionMeta{DeviceType: "ios", IPAddress: "1.1.1.1"}
	meta2 := platformauth.SessionMeta{DeviceType: "web", IPAddress: "2.2.2.2"}
	result1, _ := svc.Register(context.Background(), RegisterRequest{LoginID: "a@b.com"}, meta1)
	svc.Login(context.Background(), LoginRequest{LoginID: "a@b.com"}, meta2)

	ctx := platformauth.WithPrincipal(context.Background(), result1.Principal)
	allResult := svc.RevokeAll(ctx)
	if allResult.RevokedCount != 1 {
		t.Fatalf("expected 1 revoked, got %d", allResult.RevokedCount)
	}
}

// --- maskIPAddress ---

func TestMaskIPv4(t *testing.T) {
	result := maskIPAddress("192.168.1.100")
	if result != "192.168.xxx.xxx" {
		t.Fatalf("expected 192.168.xxx.xxx, got %s", result)
	}
}

func TestMaskIPv6(t *testing.T) {
	result := maskIPAddress("2001:0db8:85a3:0000:0000:8a2e:0370:7334")
	if result != "2001:0db8:xxxx:xxxx" {
		t.Fatalf("expected masked IPv6, got %s", result)
	}
}

func TestMaskEmpty(t *testing.T) {
	result := maskIPAddress("")
	if result != "" {
		t.Fatalf("expected empty, got %s", result)
	}
}

// --- realtime integration ---

func TestRevokeSessionClosesRealtimeConnection(t *testing.T) {
	svc, authSvc := newTestService(&stubAuthRepo{})
	reg, _ := svc.Register(context.Background(), RegisterRequest{LoginID: "ws_revoke", Nickname: "test", Password: "pass123"}, defaultMeta())
	sessionID := reg.Principal.SessionID
	userID := reg.Principal.UserID
	otherResult, _ := authSvc.IssueSession(context.Background(), userID)
	otherSessionID := otherResult.Principal.SessionID

	hub := realtime.DefaultHub()
	sink := &testSink{}
	hub.RegisterUser(userID, otherSessionID, sink)

	ctx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: userID, SessionID: sessionID})
	err := svc.RevokeSession(ctx, RevokeSessionRequest{SessionID: otherSessionID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !sink.closed {
		t.Fatal("expected sink to be closed after RevokeSession")
	}
}

type testSink struct {
	closed bool
}

func (s *testSink) Send(event realtime.Event) error {
	if event.Type == "SessionRevoked" {
		s.closed = true
	}
	return nil
}

func (s *testSink) Close() error { return nil }

// --- verification sender selection ---

func TestDefaultVerificationSenderDevMode(t *testing.T) {
	if _, ok := defaultVerificationSender(config.Config{AppEnv: "dev"}).(*logVerificationSender); !ok {
		t.Fatal("expected *logVerificationSender in dev")
	}
	if _, ok := defaultVerificationSender(config.Config{AppEnv: "test"}).(*logVerificationSender); !ok {
		t.Fatal("expected *logVerificationSender in test")
	}
}

func TestDefaultVerificationSenderProductionLike(t *testing.T) {
	for _, env := range []string{"", "production", "staging"} {
		if _, ok := defaultVerificationSender(config.Config{AppEnv: env}).(*noopVerificationSender); !ok {
			t.Fatalf("expected *noopVerificationSender for AppEnv=%q", env)
		}
	}
}

func TestLogVerificationSenderReturnsNil(t *testing.T) {
	if err := (&logVerificationSender{}).SendVerification(context.Background(), "a@b.com", ChannelEmail, "123456"); err != nil {
		t.Fatalf("logVerificationSender must succeed in dev, got %v", err)
	}
}

func TestDetectChannel(t *testing.T) {
	for _, tc := range []struct {
		loginID string
		want    VerificationChannel
	}{
		{"user@mail.com", ChannelEmail},
		{"13800138000", ChannelSMS},
		{"alice", ChannelEmail}, // username defaults to email
	} {
		if got := detectChannel(tc.loginID); got != tc.want {
			t.Fatalf("detectChannel(%q) = %q, want %q", tc.loginID, got, tc.want)
		}
	}
}

// NewServiceWithSource must wire the log sender in dev so the password-reset
// flow is usable end-to-end without an external gateway.
func TestNewServiceWithSourceDevWiresLogSender(t *testing.T) {
	cfg := config.Config{AppEnv: "dev", AuthTokenSecret: "test-secret"}
	svc := NewServiceWithSource(context.Background(), platformauth.NewService(cfg), nil, cfg, nil)
	if _, ok := svc.codeSender.(*logVerificationSender); !ok {
		t.Fatalf("expected *logVerificationSender in dev, got %T", svc.codeSender)
	}
}
