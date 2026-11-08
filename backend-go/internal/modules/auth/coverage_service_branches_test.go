package auth

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/appconfig"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/httpx"
)

// newCovServiceWithAuth 用给定平台认证服务装配 auth Service。
func newCovServiceWithAuth(authSvc *platformauth.Service, repo Repository) *Service {
	cfg := testConfig()
	return NewServiceWithSource(context.Background(), authSvc, repo, cfg, appconfig.NewStaticSource(cfg))
}

// covFailingSessionStore 的 Save 恒失败，用于覆盖各签发会话失败分支。
type covFailingSessionStore struct{}

func (covFailingSessionStore) Save(context.Context, platformauth.SessionRecord) error {
	return errors.New("session store down")
}
func (covFailingSessionStore) Touch(context.Context, string) error {
	return errors.New("session store down")
}
func (covFailingSessionStore) ListActiveByUser(context.Context, string) ([]platformauth.SessionRecord, error) {
	return nil, errors.New("session store down")
}
func (covFailingSessionStore) Revoke(context.Context, string) (bool, error) {
	return false, errors.New("session store down")
}
func (covFailingSessionStore) RevokeOthers(context.Context, string, string) (int, error) {
	return 0, errors.New("session store down")
}
func (covFailingSessionStore) Get(context.Context, string) (platformauth.SessionRecord, bool, error) {
	return platformauth.SessionRecord{}, false, errors.New("session store down")
}

// covErrorRefreshStore 的 Get 恒失败，用于覆盖 refresh 基础设施错误分支。
type covErrorRefreshStore struct{}

func (covErrorRefreshStore) Save(context.Context, platformauth.RefreshSession) error {
	return errors.New("refresh store down")
}
func (covErrorRefreshStore) Get(context.Context, string) (platformauth.RefreshSession, bool, error) {
	return platformauth.RefreshSession{}, false, errors.New("refresh store down")
}
func (covErrorRefreshStore) Rotate(context.Context, string, platformauth.RefreshSession, platformauth.RefreshSession) error {
	return errors.New("refresh store down")
}

// covMissOnceRefreshStore 首次 Get 返回未命中（模拟 SessionIDForRefreshToken 预检与
// 实际轮换之间的时序窗口），其后委托真实存储。
type covMissOnceRefreshStore struct {
	inner    platformauth.RefreshTokenStore
	getCalls int
}

func (s *covMissOnceRefreshStore) Save(ctx context.Context, session platformauth.RefreshSession) error {
	return s.inner.Save(ctx, session)
}
func (s *covMissOnceRefreshStore) Get(ctx context.Context, token string) (platformauth.RefreshSession, bool, error) {
	s.getCalls++
	if s.getCalls == 1 {
		return platformauth.RefreshSession{}, false, nil
	}
	return s.inner.Get(ctx, token)
}
func (s *covMissOnceRefreshStore) Rotate(ctx context.Context, oldToken string, current, next platformauth.RefreshSession) error {
	return s.inner.Rotate(ctx, oldToken, current, next)
}

type covBanChecker struct{ banned bool }

func (c covBanChecker) IsBanned(context.Context, string) (bool, error) { return c.banned, nil }

// covConsumeFailRepo 的 ConsumeRegisterCode 恒失败，其余委托 stubAuthRepo。
type covConsumeFailRepo struct{ *stubAuthRepo }

func (r *covConsumeFailRepo) ConsumeRegisterCode(context.Context, string, string) error {
	return errors.New("consume down")
}

func TestNormalizeLoginTrackerLimitsDefaultsCov(t *testing.T) {
	limits := normalizeLoginTrackerLimits(0, 0)
	if limits.effectiveMaxFailures != defaultMaxFailures || limits.effectiveLockDuration != defaultLockDuration {
		t.Fatalf("unexpected defaults: %#v", limits)
	}
	limits = normalizeLoginTrackerLimits(3, time.Minute)
	if limits.effectiveMaxFailures != 3 || limits.effectiveLockDuration != time.Minute {
		t.Fatalf("explicit limits not honored: %#v", limits)
	}
}

func TestNewServiceWithSourcePanicsWithoutRepoCov(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic when repo is nil")
		}
	}()
	cfg := testConfig()
	NewServiceWithSource(context.Background(), platformauth.NewService(cfg), nil, cfg, appconfig.NewStaticSource(cfg))
}

func TestServiceRegisterMissingVerificationCodeCov(t *testing.T) {
	repo := &stubAuthRepo{registerAccount: Account{UserID: "u1"}}
	svc, _ := newTestService(repo)
	_, err := svc.Register(context.Background(), RegisterRequest{LoginID: "a@b.com", Password: "password1"}, defaultMeta())
	if !errors.Is(err, ErrMissingRegisterCode) {
		t.Fatalf("expected ErrMissingRegisterCode, got %v", err)
	}
}

func TestServiceRegisterConsumeCodeFailureStillSucceedsCov(t *testing.T) {
	repo := &covConsumeFailRepo{stubAuthRepo: &stubAuthRepo{registerAccount: Account{UserID: "u1"}}}
	svc := newCovServiceWithAuth(platformauth.NewService(testConfig()), repo)
	code := "123456"
	result, err := svc.Register(context.Background(), RegisterRequest{
		LoginID: "a@b.com", Password: "password1", VerificationCode: &code,
	}, defaultMeta())
	if err != nil {
		t.Fatalf("consume failure must not fail registration, got %v", err)
	}
	if result.Principal.UserID != "u1" {
		t.Fatalf("unexpected principal: %#v", result.Principal)
	}
}

func TestServiceSendRegisterCodeErrorBranchesCov(t *testing.T) {
	throttled := &stubAuthRepo{registerCodeErr: ErrRegisterCodeThrottled}
	svc, _ := newTestService(throttled)
	err := svc.SendRegisterCode(context.Background(), SendRegisterCodeRequest{LoginID: "a@b.com", ClientType: ClientTypeApp})
	if !errors.Is(err, ErrRegisterCodeThrottled) {
		t.Fatalf("expected throttled error, got %v", err)
	}

	broken := &stubAuthRepo{registerCodeErr: errors.New("repo down")}
	svc, _ = newTestService(broken)
	err = svc.SendRegisterCode(context.Background(), SendRegisterCodeRequest{LoginID: "a@b.com", ClientType: ClientTypeApp})
	if err == nil || errors.Is(err, ErrRegisterCodeThrottled) {
		t.Fatalf("expected wrapped repo error, got %v", err)
	}
	if code := apiErrorFrom(err).Code; code != httpx.CodeInternalUnavailable {
		t.Fatalf("expected 10600, got %d", code)
	}
}

func TestServiceLoginGenericRepoErrorCov(t *testing.T) {
	repo := &stubAuthRepo{authErr: errors.New("db down")}
	svc, _ := newTestService(repo)
	_, err := svc.Login(context.Background(), LoginRequest{LoginID: "a@b.com", Password: "password1"}, defaultMeta())
	if err == nil || errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected infrastructure error, got %v", err)
	}
	if code := apiErrorFrom(err).Code; code != httpx.CodeInternalUnavailable {
		t.Fatalf("expected 10600, got %d", code)
	}
}

func TestServiceLoginBannedAccountCov(t *testing.T) {
	cfg := testConfig()
	authSvc := platformauth.NewService(cfg)
	authSvc.SetBanChecker(covBanChecker{banned: true})
	svc := newCovServiceWithAuth(authSvc, &stubAuthRepo{authAccount: Account{UserID: "u1"}})
	_, err := svc.Login(context.Background(), LoginRequest{LoginID: "a@b.com", Password: "password1"}, defaultMeta())
	if !errors.Is(err, ErrAccountBanned) {
		t.Fatalf("expected ErrAccountBanned, got %v", err)
	}
}

func TestServiceIssueSessionFailureBranchesCov(t *testing.T) {
	cfg := testConfig()
	newService := func(repo Repository) *Service {
		authSvc := platformauth.NewServiceWithStores(cfg, covFailingSessionStore{}, platformauth.NewInMemoryRefreshStore())
		return newCovServiceWithAuth(authSvc, repo)
	}

	t.Run("register", func(t *testing.T) {
		svc := newService(&stubAuthRepo{registerAccount: Account{UserID: "u1"}})
		_, err := svc.Register(context.Background(), RegisterRequest{LoginID: "alice", Password: "password1"}, defaultMeta())
		if err == nil || !strings.Contains(err.Error(), "issue session") {
			t.Fatalf("expected issue session error, got %v", err)
		}
	})

	t.Run("login", func(t *testing.T) {
		svc := newService(&stubAuthRepo{authAccount: Account{UserID: "u1"}})
		_, err := svc.Login(context.Background(), LoginRequest{LoginID: "alice", Password: "password1"}, defaultMeta())
		if err == nil || !strings.Contains(err.Error(), "issue session") {
			t.Fatalf("expected issue session error, got %v", err)
		}
	})

	t.Run("guest", func(t *testing.T) {
		svc := newService(&stubAuthRepo{guestAccount: Account{UserID: "g1"}})
		_, err := svc.Guest(context.Background(), GuestRequest{}, defaultMeta())
		if err == nil || !strings.Contains(err.Error(), "issue session") {
			t.Fatalf("expected issue session error, got %v", err)
		}
	})

	t.Run("reset password", func(t *testing.T) {
		svc := newService(&stubAuthRepo{resetAccount: Account{UserID: "u1"}})
		_, err := svc.ResetPassword(context.Background(), ResetPasswordRequest{
			LoginID: "alice", VerificationCode: "123456", NewPassword: "password1",
		}, defaultMeta())
		if err == nil || !strings.Contains(err.Error(), "issue session") {
			t.Fatalf("expected issue session error, got %v", err)
		}
	})
}

func TestServiceRefreshUsesCookieForWebCov(t *testing.T) {
	cfg := testConfig()
	authSvc := platformauth.NewService(cfg)
	svc := newCovServiceWithAuth(authSvc, &stubAuthRepo{})
	issued, err := authSvc.IssueSessionWithMeta(context.Background(), "u1", platformauth.SessionMeta{DeviceID: "dev-1"})
	if err != nil {
		t.Fatalf("issue session: %v", err)
	}
	result, err := svc.Refresh(context.Background(), RefreshTokenRequest{ClientType: ClientTypeWeb, RefreshToken: "ignored-body-token"}, "dev-1", issued.RefreshToken)
	if err != nil {
		t.Fatalf("web refresh with cookie: %v", err)
	}
	if result.Principal.SessionID != issued.Principal.SessionID || result.RefreshToken == "" {
		t.Fatalf("unexpected refresh result: %#v", result.Principal)
	}
}

func TestServiceRefreshDeviceMismatchCov(t *testing.T) {
	cfg := testConfig()
	authSvc := platformauth.NewService(cfg)
	svc := newCovServiceWithAuth(authSvc, &stubAuthRepo{})
	issued, err := authSvc.IssueSessionWithMeta(context.Background(), "u1", platformauth.SessionMeta{DeviceID: "real-device"})
	if err != nil {
		t.Fatalf("issue session: %v", err)
	}
	_, err = svc.Refresh(context.Background(), RefreshTokenRequest{ClientType: ClientTypeApp, RefreshToken: issued.RefreshToken}, "claimed-device", "")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected device mismatch rejection, got %v", err)
	}
}

func TestServiceRefreshInfraErrorCov(t *testing.T) {
	cfg := testConfig()
	authSvc := platformauth.NewServiceWithStores(cfg, platformauth.NewInMemorySessionStore(), covErrorRefreshStore{})
	svc := newCovServiceWithAuth(authSvc, &stubAuthRepo{})
	_, err := svc.Refresh(context.Background(), RefreshTokenRequest{ClientType: ClientTypeApp, RefreshToken: "token"}, "", "")
	if err == nil {
		t.Fatal("expected refresh store error")
	}
	if !strings.Contains(err.Error(), "refresh store down") {
		t.Fatalf("expected wrapped refresh store error, got %v", err)
	}
}

func TestServiceRefreshPostRotationDeviceMismatchCov(t *testing.T) {
	cfg := testConfig()
	refreshStore := &covMissOnceRefreshStore{inner: platformauth.NewInMemoryRefreshStore()}
	authSvc := platformauth.NewServiceWithStores(cfg, platformauth.NewInMemorySessionStore(), refreshStore)
	svc := newCovServiceWithAuth(authSvc, &stubAuthRepo{})
	issued, err := authSvc.IssueSessionWithMeta(context.Background(), "u1", platformauth.SessionMeta{DeviceID: "real-device"})
	if err != nil {
		t.Fatalf("issue session: %v", err)
	}
	_, err = svc.Refresh(context.Background(), RefreshTokenRequest{ClientType: ClientTypeApp, RefreshToken: issued.RefreshToken}, "claimed-device", "")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected post-rotation device mismatch rejection, got %v", err)
	}
}

func TestServiceListSessionsOrderingAndMaskFallbackCov(t *testing.T) {
	cfg := testConfig()
	sessions := platformauth.NewInMemorySessionStore()
	now := time.Now().UTC()
	for _, record := range []platformauth.SessionRecord{
		{SessionID: "cov_sess_old", UserID: "u1", LastActiveAt: now.Add(-2 * time.Hour), IPAddress: "203.0.113.9"},
		{SessionID: "cov_sess_mid", UserID: "u1", LastActiveAt: now.Add(-time.Hour), IPAddress: "unparseable-ip"},
		{SessionID: "cov_sess_current", UserID: "u1", LastActiveAt: now.Add(-3 * time.Hour), IPAddress: "198.51.100.7"},
	} {
		if err := sessions.Save(context.Background(), record); err != nil {
			t.Fatalf("seed session: %v", err)
		}
	}
	authSvc := platformauth.NewServiceWithStores(cfg, sessions, platformauth.NewInMemoryRefreshStore())
	svc := newCovServiceWithAuth(authSvc, &stubAuthRepo{})
	list := svc.ListSessions(authCtxWithSession("u1", "cov_sess_current"))
	if len(list.Items) != 3 {
		t.Fatalf("expected 3 sessions, got %d", len(list.Items))
	}
	if list.Items[0].SessionID != "cov_sess_current" || !list.Items[0].IsCurrent {
		t.Fatalf("current session must be first, got %#v", list.Items[0])
	}
	if list.Items[1].SessionID != "cov_sess_mid" || list.Items[2].SessionID != "cov_sess_old" {
		t.Fatalf("unexpected ordering: %#v", list.Items)
	}
	if list.Items[1].IPAddress == nil || *list.Items[1].IPAddress != "unparseable-ip" {
		t.Fatalf("unparseable IP must fall back to raw value, got %#v", list.Items[1].IPAddress)
	}
}

func TestServiceRequestPasswordResetEmptyCodeNoopCov(t *testing.T) {
	repo := &stubAuthRepo{resetCode: ""}
	svc, _ := newTestService(repo)
	if err := svc.RequestPasswordReset(context.Background(), RequestPasswordResetRequest{LoginID: "missing", ClientType: ClientTypeApp}); err != nil {
		t.Fatalf("empty code (unknown account) must be a no-op, got %v", err)
	}
}

type covPurgeRepo struct {
	*InMemoryRepository
	purged []string
	err    error
}

func (r *covPurgeRepo) PurgeUserData(_ context.Context, userID string) error {
	r.purged = append(r.purged, userID)
	return r.err
}

func TestServicePurgeUserDataDelegatesCov(t *testing.T) {
	repo := &covPurgeRepo{InMemoryRepository: NewInMemoryRepository()}
	svc := newCovServiceWithAuth(platformauth.NewService(testConfig()), repo)
	if err := svc.PurgeUserData(context.Background(), "u1"); err != nil {
		t.Fatalf("purge: %v", err)
	}
	if len(repo.purged) != 1 || repo.purged[0] != "u1" {
		t.Fatalf("expected delegate call, got %#v", repo.purged)
	}

	repo.err = errors.New("purge down")
	if err := svc.PurgeUserData(context.Background(), "u1"); err == nil {
		t.Fatal("expected purge error to propagate")
	}
}

func TestServicePurgeUserDataNoopForNonPurgerCov(t *testing.T) {
	svc := newCovServiceWithAuth(platformauth.NewService(testConfig()), NewInMemoryRepository())
	if err := svc.PurgeUserData(context.Background(), "u1"); err != nil {
		t.Fatalf("non-purger repo must return nil, got %v", err)
	}
}

func TestServiceRegisterUsesLoginIDContactCov(t *testing.T) {
	if email, phone := loginIDContact("13800138000"); email != "" || phone != "13800138000" {
		t.Fatalf("phone contact: %q/%q", email, phone)
	}
	if email, phone := loginIDContact("alice"); email != "" || phone != "" {
		t.Fatalf("username contact: %q/%q", email, phone)
	}
}
