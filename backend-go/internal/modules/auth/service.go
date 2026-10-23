package auth

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"hotspot/backend-go/internal/platform/appconfig"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/observability"
	"hotspot/backend-go/internal/platform/realtime"
	"hotspot/backend-go/internal/platform/resourceguard"
	"hotspot/backend-go/internal/platform/types"
	"hotspot/backend-go/internal/platform/worker"
)

// FailedLoginTracker tracks per-account failed login attempts for brute-force protection.
// The default in-memory implementation is suitable for single-instance deployments;
// multi-instance deployments should use a Redis-backed implementation.
type FailedLoginTracker interface {
	// Check returns ErrAccountLocked if the account is currently locked.
	Check(ctx context.Context, loginID string) error
	// RecordFailed increments the failed-attempt counter for loginID.
	RecordFailed(ctx context.Context, loginID string)
	// Clear resets the counter for loginID (called on successful login).
	Clear(ctx context.Context, loginID string)
}

// loginLockEntry records the state of a locked account.
type loginLockEntry struct {
	failures  int
	lockedAt  time.Time
	lockUntil time.Time
	// lastFailedAt 最近一次计失败的时刻：未锁定条目的计数衰减依据
	//（见 loginCounterDecay），锁定条目不使用。
	lastFailedAt time.Time
}

const (
	defaultMaxFailures    = 5
	defaultLockDuration   = 15 * time.Minute
	defaultTrackerCleanup = 10 * time.Minute
	// loginCounterDecay 未锁定条目的失败计数衰减窗口：静默（无新失败）
	// 达该窗口后计数作废。窗口内连续失败会持续累计直至锁定——若按
	// sweeper 周期无差别清扫，攻击者以「每周期失败 max-1 次」的节奏可
	// 永不触发锁定（审查 P1-5），故只清「静默期已过」的条目。
	loginCounterDecay = defaultLockDuration
)

// inMemoryLoginTracker is a simple in-memory failed-login tracker.
// It should be replaced with a Redis-backed implementation in multi-instance deployments.
type inMemoryLoginTracker struct {
	mu       sync.Mutex
	entries  map[string]*loginLockEntry
	stopOnce sync.Once
	stopCh   chan struct{}
}

func newInMemoryLoginTracker(ctx context.Context) *inMemoryLoginTracker {
	t := &inMemoryLoginTracker{
		entries: make(map[string]*loginLockEntry),
		stopCh:  make(chan struct{}),
	}
	// Bind the sweeper to ctx so it exits when the app root context is
	// cancelled (graceful shutdown), and wrap it with worker.Go so a panic
	// inside the sweep loop is recovered instead of crashing the process.
	worker.GoRestartable("auth.login_tracker.sweeper", func() { t.sweep(ctx) })
	return t
}

func (t *inMemoryLoginTracker) Check(_ context.Context, loginID string) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	entry, ok := t.entries[loginID]
	if !ok {
		return nil
	}
	now := time.Now()
	if now.Before(entry.lockUntil) {
		return ErrAccountLocked
	}
	// Lock expired (lockUntil was set and is now in the past) — clean up.
	// Don't delete entries that haven't been locked yet (lockUntil is zero).
	if !entry.lockUntil.IsZero() && now.After(entry.lockUntil) {
		delete(t.entries, loginID)
	}
	return nil
}

func (t *inMemoryLoginTracker) RecordFailed(_ context.Context, loginID string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	now := time.Now()
	entry := t.entries[loginID]
	if entry == nil {
		entry = &loginLockEntry{}
		t.entries[loginID] = entry
	} else if now.Before(entry.lockUntil) {
		// Already locked — don't reset the timer
		return
	}
	entry.failures++
	entry.lastFailedAt = now
	if entry.failures >= defaultMaxFailures {
		entry.lockedAt = now
		entry.lockUntil = now.Add(defaultLockDuration)
	}
}

func (t *inMemoryLoginTracker) Clear(_ context.Context, loginID string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.entries, loginID)
}

func (t *inMemoryLoginTracker) sweep(ctx context.Context) {
	ticker := time.NewTicker(defaultTrackerCleanup)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.stopCh:
			return
		case <-ticker.C:
			t.sweepOnce(time.Now())
		}
	}
}

// sweepOnce 清扫过期条目（与 Check 的惰性清理同口径，供周期 sweeper 与测试复用）：
//   - 已锁定且锁期已过：锁定解除，计数随之作废（下次失败从零计）；
//   - 从未锁定且静默已达 loginCounterDecay：计数衰减作废；
//   - 锁定中的条目一律保留（清扫周期短于锁期时不得提前释放）。
//
// 此前条件是裸的 now.After(entry.lockUntil)，未锁定条目的零值 lockUntil
// 恒满足，等于每轮清光所有部分计数——「每 10 分钟失败 4 次」即可永不锁定。
func (t *inMemoryLoginTracker) sweepOnce(now time.Time) {
	t.mu.Lock()
	defer t.mu.Unlock()
	for id, entry := range t.entries {
		if !entry.lockUntil.IsZero() {
			if now.After(entry.lockUntil) {
				delete(t.entries, id)
			}
			continue
		}
		if !entry.lastFailedAt.IsZero() && now.Sub(entry.lastFailedAt) >= loginCounterDecay {
			delete(t.entries, id)
		}
	}
}

func (t *inMemoryLoginTracker) stop() {
	t.stopOnce.Do(func() { close(t.stopCh) })
}

type Service struct {
	auth         *platformauth.Service
	repo         Repository
	cfg          config.Config
	appConfig    appconfig.Source
	codeSender   verificationSender
	loginTracker FailedLoginTracker
}

func NewService(auth *platformauth.Service, repo Repository, cfg config.Config) *Service {
	return NewServiceWithSource(context.Background(), auth, repo, cfg, appconfig.NewStaticSource(cfg))
}

// NewServiceWithSource builds a Service. ctx governs the lifetime of the
// in-memory login-tracker sweeper goroutine; pass the application root context
// from bootstrap so the sweeper exits on graceful shutdown. Tests can pass
// context.Background() (the goroutine is cleaned up when the test process
// exits) or a cancellable context via t.Cleanup.
func NewServiceWithSource(ctx context.Context, auth *platformauth.Service, repo Repository, cfg config.Config, source appconfig.Source) *Service {
	if repo == nil {
		repo = NewInMemoryRepository()
	}
	if source == nil {
		source = appconfig.NewStaticSource(cfg)
	}
	return &Service{auth: auth, repo: repo, cfg: cfg, appConfig: source, codeSender: defaultVerificationSender(cfg), loginTracker: newInMemoryLoginTracker(ctx)}
}

// AttachCredentials 为既有用户写入登录凭证（spec api_spec_user.md → upgrade
// 契约级语义：升级成功后 login_id+password 落 auth_accounts，此后可用于
// /public/v1/auth/signin）。复用注册路径的 bcrypt 口径；login_id 已被占用
// 返回 ErrLoginIDConflict。ctx 携带 users 模块开启的事务时自动加入该事务，
// 使凭证与 user_type 同事务落库（见 persistence.RunInTransaction 嵌套语义）。
func (s *Service) AttachCredentials(ctx context.Context, userID, loginID, password string) error {
	if err := s.repo.AttachCredential(ctx, userID, loginID, password); err != nil {
		return fmt.Errorf("auth.Service.AttachCredentials: %w", err)
	}
	return nil
}

// GetStatus 返回当前会话状态（spec auth.md → auth/session）。
// 该端点挂在必须认证节点下；无 principal 属编程/测试错误，按 10100 处理。
func (s *Service) GetStatus(ctx context.Context) (SessionStatus, error) {
	principal, ok := platformauth.PrincipalFromContext(ctx)
	if !ok || principal.UserID == "" {
		return SessionStatus{}, ErrUnauthenticated
	}
	expiresAt, ok := platformauth.AccessExpiryFromContext(ctx)
	if !ok {
		return SessionStatus{}, ErrUnauthenticated
	}
	return SessionStatus{
		UserID:    principal.UserID,
		ExpiresAt: expiresAt.UTC().Format(time.RFC3339),
	}, nil
}

// AccessTTL / RefreshTTL 透传 platformauth.Service 的令牌有效期（单一事实
// 来源），供 Handler 写 auth Cookie 时引用，消除内联字面量 TTL。
func (s *Service) AccessTTL() time.Duration  { return s.auth.AccessTTL() }
func (s *Service) RefreshTTL() time.Duration { return s.auth.RefreshTTL() }

func (s *Service) Register(ctx context.Context, req RegisterRequest, meta platformauth.SessionMeta) (platformauth.AuthResult, error) {
	startedAt := time.Now()
	// 平台开关服务端强制（features.enable_public_registration）：管理后台关闭
	// 注册后，绕过客户端开关的直接 API 调用同样被拒（10400）。
	if !s.appConfig.Get(ctx).Features.EnablePublicRegistration {
		return platformauth.AuthResult{}, ErrRegistrationClosed
	}
	// 手机号/邮箱型 login_id 必须先验证所有权（spec api_spec_public.md →
	// register）：验证码错误/过期 → 10401；已占用的 login_id 由 repo 的唯一
	// 约束拒绝（10400）。
	if isPhoneOrEmailLoginID(req.LoginID) {
		if req.VerificationCode == nil {
			return platformauth.AuthResult{}, ErrMissingRegisterCode
		}
		if err := s.repo.VerifyRegisterCode(ctx, req.LoginID, *req.VerificationCode); err != nil {
			return platformauth.AuthResult{}, err
		}
	}
	registerEmail, registerPhone := loginIDContact(req.LoginID)
	account, err := s.repo.Register(ctx, RegisterParams{
		LoginID:          req.LoginID,
		Nickname:         req.Nickname,
		Password:         req.Password,
		Defaults:         s.profileDefaults(ctx),
		RegisterIP:       meta.IPAddress,
		RegisterLocation: meta.IPLocation,
		Email:            registerEmail,
		Phone:            registerPhone,
	})
	if err != nil {
		observability.L().Error("service error",
			"method", "Register",
			"error", err,
			"elapsed_ms", time.Since(startedAt).Milliseconds(),
		)
		return platformauth.AuthResult{}, fmt.Errorf("auth.Service.Register: repo: %w", err)
	}
	// 验证码一次性：注册成功后消费（失败不烧码）。login_id 唯一约束保证
	// 同一验证码最多促成一次成功注册；消费失败仅告警（注册已成功，不能回滚）。
	if isPhoneOrEmailLoginID(req.LoginID) && req.VerificationCode != nil {
		if err := s.repo.ConsumeRegisterCode(ctx, req.LoginID, *req.VerificationCode); err != nil {
			observability.L().Warn("register code consume failed",
				"method", "Register",
				"login_id", observability.Redact(req.LoginID),
				"error", err,
			)
		}
	}
	result, err := s.auth.IssueSessionWithMeta(ctx, account.UserID, meta)
	if err != nil {
		observability.L().Error("service error",
			"method", "Register",
			"error", err,
			"elapsed_ms", time.Since(startedAt).Milliseconds(),
		)
		return platformauth.AuthResult{}, fmt.Errorf("auth.Service.Register: issue session: %w", err)
	}
	resourceguard.RecordUser(ctx, account.UserID)
	return result, nil
}

// SendRegisterCode 发送注册验证码（spec api_spec_public.md →
// auth/send_register_code）。防枚举：无论 login_id 是否已被注册均返回成功；
// 已占用的 login_id 对应验证码不生效（注册时先撞唯一约束 10400）。
func (s *Service) SendRegisterCode(ctx context.Context, req SendRegisterCodeRequest) error {
	code, err := s.repo.CreateRegisterCode(ctx, req.LoginID, time.Now().UTC().Add(10*time.Minute))
	if err != nil {
		// 发送节流是预期业务限流：直接透传（apiErrorFrom 映射 10500 + Retry-After），
		// 不与基础设施错误混记 Error。
		if errors.Is(err, ErrRegisterCodeThrottled) {
			observability.L().Warn("verification code send throttled for register",
				"method", "SendRegisterCode",
				"login_id", observability.Redact(req.LoginID),
			)
			return err
		}
		observability.L().Error("service error",
			"method", "SendRegisterCode",
			"error", err,
		)
		return fmt.Errorf("auth.Service.SendRegisterCode: repo: %w", err)
	}
	if sendErr := s.codeSender.SendVerification(ctx, req.LoginID, detectChannel(req.LoginID), code); sendErr != nil {
		// 投递失败必须向客户端透传（审查 P2-3）：生产未配 SMTP/SMS 网关时
		// 此前只 Warn 后仍返回成功——用户收到「已发送」却永远等不到验证码，
		// ErrVerificationServiceUnavailable→10600 的映射成为不可达死路径。
		// 防枚举不受影响：投递能力对同一时刻的所有 login_id 一致（网关级
		// 状态，与账号是否存在无关），统一成功/失败无存在性侧信道。
		observability.L().Warn("verification code delivery failed for register",
			"method", "SendRegisterCode",
			"login_id", observability.Redact(req.LoginID),
			"error", sendErr,
		)
		return fmt.Errorf("auth.Service.SendRegisterCode: deliver: %w", sendErr)
	}
	return nil
}

func (s *Service) Login(ctx context.Context, req LoginRequest, meta platformauth.SessionMeta) (platformauth.AuthResult, error) {
	startedAt := time.Now()

	// Check brute-force lockout before even hashing the password.
	if err := s.loginTracker.Check(ctx, req.LoginID); err != nil {
		observability.L().Warn("login blocked by lockout",
			"method", "Login",
			"login_id", observability.Redact(req.LoginID),
		)
		return platformauth.AuthResult{}, err
	}

	account, err := s.repo.Authenticate(ctx, req.LoginID, req.Password)
	if err != nil {
		// ErrInvalidCredentials 是预期业务错误（密码错误/账号不存在，防枚举下
		// 两者同响应）：按约定走 Warn，不与基础设施错误混记 Error。
		if errors.Is(err, ErrInvalidCredentials) {
			s.loginTracker.RecordFailed(ctx, req.LoginID)
			observability.L().Warn("login failed: invalid credentials",
				"method", "Login",
				"login_id", observability.Redact(req.LoginID),
				"elapsed_ms", time.Since(startedAt).Milliseconds(),
			)
			return platformauth.AuthResult{}, fmt.Errorf("auth.Service.Login: authenticate: %w", err)
		}
		observability.L().Error("service error",
			"method", "Login",
			"error", err,
			"elapsed_ms", time.Since(startedAt).Milliseconds(),
		)
		return platformauth.AuthResult{}, fmt.Errorf("auth.Service.Login: authenticate: %w", err)
	}

	// Successful login — reset the failed-attempt counter.
	s.loginTracker.Clear(ctx, req.LoginID)

	// 封禁账号拒绝登录（spec api_spec_errors.md → 10201）：否则被封禁用户可反复
	// 登录换取新 session；HTTP 侧虽被 RequiredAuth 拦截，但 WS 握手等非中间件
	// 路径会绕过封禁。检查放在凭据校验之后，避免向封禁账号泄露账号存在性。
	if s.auth.IsUserBanned(ctx, account.UserID) {
		observability.L().Warn("login blocked: account banned",
			"method", "Login",
			"user_id", account.UserID,
		)
		return platformauth.AuthResult{}, ErrAccountBanned
	}

	if err := s.repo.UpdateLoginMeta(ctx, account.UserID, meta.IPAddress, meta.IPLocation); err != nil {
		observability.L().Error("service error",
			"method", "Login",
			"error", err,
			"elapsed_ms", time.Since(startedAt).Milliseconds(),
		)
		return platformauth.AuthResult{}, fmt.Errorf("auth.Service.Login: update meta: %w", err)
	}
	// 设备逐出改为「先插新会话，冲突则由平台层撤销旧会话重试」（TOCTOU 修复，
	// 见 IssueSessionEvictingDevice 注释）。撤销后旧会话不再出现在活跃列表，
	// 故先快照需要关闭的 WS 连接，成功后再逐个关闭。
	evictedSessionIDs := make([]string, 0, 1)
	for _, session := range s.auth.ListSessions(ctx, account.UserID, "") {
		if session.DeviceID == meta.DeviceID && meta.DeviceID != "" {
			evictedSessionIDs = append(evictedSessionIDs, session.SessionID)
		}
	}
	result, err := s.auth.IssueSessionEvictingDevice(ctx, account.UserID, meta)
	if err != nil {
		observability.L().Error("service error",
			"method", "Login",
			"error", err,
			"elapsed_ms", time.Since(startedAt).Milliseconds(),
		)
		return platformauth.AuthResult{}, fmt.Errorf("auth.Service.Login: issue session: %w", err)
	}
	for _, sessionID := range evictedSessionIDs {
		if sessionID != result.Principal.SessionID {
			realtime.DefaultHub().CloseSession(sessionID, sessionRevokedEvent("device_evicted"))
		}
	}
	return result, nil
}

func (s *Service) Guest(ctx context.Context, req GuestRequest, meta platformauth.SessionMeta) (platformauth.AuthResult, error) {
	startedAt := time.Now()
	// 平台开关服务端强制（features.enable_guest_register），同 Register。
	if !s.appConfig.Get(ctx).Features.EnableGuestRegister {
		return platformauth.AuthResult{}, ErrGuestDisabled
	}
	// Nickname 可选（spec api_spec_public.md → guest）：null/空时服务端生成
	// 默认昵称「游客·xxxx」（4 位小写字母数字随机），进入后引导修改。
	nickname := ""
	if req.Nickname != nil {
		nickname = strings.TrimSpace(*req.Nickname)
	}
	if nickname == "" {
		generated, err := generateGuestNickname()
		if err != nil {
			return platformauth.AuthResult{}, err
		}
		nickname = generated
	}
	account, err := s.repo.CreateGuest(ctx, GuestParams{Nickname: nickname, Defaults: s.profileDefaults(ctx), RegisterIP: meta.IPAddress, RegisterLocation: meta.IPLocation})
	if err != nil {
		observability.L().Error("service error",
			"method", "Guest",
			"error", err,
			"elapsed_ms", time.Since(startedAt).Milliseconds(),
		)
		return platformauth.AuthResult{}, fmt.Errorf("auth.Service.Guest: repo: %w", err)
	}
	// 同 Login：先快照旧会话用于关 WS，再插入新会话（冲突时自动逐出后重试）。
	evictedSessionIDs := make([]string, 0, 1)
	for _, session := range s.auth.ListSessions(ctx, account.UserID, "") {
		if session.DeviceID == meta.DeviceID && meta.DeviceID != "" {
			evictedSessionIDs = append(evictedSessionIDs, session.SessionID)
		}
	}
	result, err := s.auth.IssueSessionEvictingDevice(ctx, account.UserID, meta)
	if err != nil {
		observability.L().Error("service error",
			"method", "Guest",
			"error", err,
			"elapsed_ms", time.Since(startedAt).Milliseconds(),
		)
		return platformauth.AuthResult{}, fmt.Errorf("auth.Service.Guest: issue session: %w", err)
	}
	for _, sessionID := range evictedSessionIDs {
		if sessionID != result.Principal.SessionID {
			realtime.DefaultHub().CloseSession(sessionID, sessionRevokedEvent("device_evicted"))
		}
	}
	resourceguard.RecordUser(ctx, account.UserID)
	return result, nil
}

func (s *Service) Refresh(ctx context.Context, req RefreshTokenRequest, deviceID string, refreshCookie string) (platformauth.AuthResult, error) {
	startedAt := time.Now()
	refreshToken := req.RefreshToken
	if req.ClientType == ClientTypeWeb {
		refreshToken = refreshCookie
	}
	// 设备绑定预校验前置于轮换：不匹配的请求在轮换前拒绝，避免伪造 device_id
	// 也触发一次 token 轮换并把旧 token 推入宽限期。预解析失败（无效/过期
	// token）不在此报错，统一交给 RefreshSession 返回标准错误口径。
	if deviceID != "" {
		if sessionID, ok, resolveErr := s.auth.SessionIDForRefreshToken(ctx, refreshToken); resolveErr == nil && ok {
			session, found, getErr := s.auth.GetSession(ctx, sessionID)
			if getErr != nil || !found || session.DeviceID != deviceID {
				observability.L().Warn("refresh device_id mismatch",
					"method", "Refresh",
					"session_id", sessionID,
					"claimed_device_id", deviceID,
				)
				return platformauth.AuthResult{}, ErrInvalidCredentials
			}
		}
	}
	result, err := s.auth.RefreshSession(ctx, refreshToken)
	if err != nil {
		// token 生命周期内的预期拒绝（无效/过期/会话已撤销/宽限窗外复用）
		// 是正常业务态——30 天到期换新、他端登出后刷新都会走到这里，按日志
		// 约定降为 Warn；仅基础设施故障（DB/存储/加密错误）保留 Error
		//（审查 P2-2：此前统一 Error，Error 日志被自然到期的刷新刷屏，淹没
		// 真实故障告警）。客户端口径不变：handler 统一映射 10100。
		if errors.Is(err, platformauth.ErrInvalidRefreshToken) ||
			errors.Is(err, platformauth.ErrExpiredRefreshToken) ||
			errors.Is(err, platformauth.ErrRevokedSession) ||
			errors.Is(err, platformauth.ErrRotationWindowExpired) {
			observability.L().Warn("refresh rejected: token lifecycle",
				"method", "Refresh",
				"error", err,
			)
		} else {
			observability.L().Error("service error",
				"method", "Refresh",
				"error", err,
				"elapsed_ms", time.Since(startedAt).Milliseconds(),
			)
		}
		return platformauth.AuthResult{}, fmt.Errorf("auth.Service.Refresh: %w", err)
	}
	if deviceID != "" {
		session, ok, getErr := s.auth.GetSession(ctx, result.Principal.SessionID)
		if getErr != nil || !ok || session.DeviceID != deviceID {
			observability.L().Warn("refresh device_id mismatch",
				"method", "Refresh",
				"session_id", result.Principal.SessionID,
				"claimed_device_id", deviceID,
				"session_device_id", session.DeviceID,
			)
			return platformauth.AuthResult{}, ErrInvalidCredentials
		}
	}
	return result, nil
}

func (s *Service) ListSessions(ctx context.Context) SessionList {
	principal, ok := platformauth.PrincipalFromContext(ctx)
	if !ok {
		return SessionList{Items: []SessionItem{}}
	}
	sessions := s.auth.ListSessions(ctx, principal.UserID, principal.SessionID)
	// 排序（spec auth.md → list_sessions）：当前会话置顶，其余按 last_active_at 倒序
	sort.SliceStable(sessions, func(i, j int) bool {
		if sessions[i].SessionID == principal.SessionID {
			return true
		}
		if sessions[j].SessionID == principal.SessionID {
			return false
		}
		return sessions[i].LastActiveAt.After(sessions[j].LastActiveAt)
	})
	items := make([]SessionItem, 0, len(sessions))
	for _, session := range sessions {
		items = append(items, SessionItem{
			SessionID:    session.SessionID,
			DeviceID:     session.DeviceID,
			DeviceType:   types.StrPtr(session.DeviceType),
			DeviceName:   types.StrPtr(session.DeviceName),
			IPAddress:    types.StrPtr(maskIPAddress(session.IPAddress)),
			IPLocation:   types.StrPtr(session.IPLocation),
			CreatedAt:    session.CreatedAt.UTC().Format(time.RFC3339),
			LastActiveAt: session.LastActiveAt.UTC().Format(time.RFC3339),
			IsCurrent:    session.SessionID == principal.SessionID,
		})
	}
	return SessionList{Items: items}
}

func (s *Service) RevokeSession(ctx context.Context, req RevokeSessionRequest) error {
	principal, ok := platformauth.PrincipalFromContext(ctx)
	if !ok {
		return platformauth.Unauthorized("Access Token 无效或过期")
	}
	if req.SessionID == principal.SessionID {
		// spec auth.md → revoke_session：不能撤销当前会话返回 10400（业务冲突），非 10200。
		return &httpx.APIError{Code: httpx.CodeConflict, Message: "不允许撤销当前会话"}
	}
	if err := s.auth.RevokeSession(ctx, principal.UserID, principal.SessionID, req.SessionID); err != nil {
		observability.L().Error("service error",
			"method", "RevokeSession",
			"error", err,
		)
		return fmt.Errorf("auth.Service.RevokeSession: %w", err)
	}
	realtime.DefaultHub().CloseSession(req.SessionID, sessionRevokedEvent("revoked_by_user"))
	return nil
}

func (s *Service) RequestPasswordReset(ctx context.Context, req RequestPasswordResetRequest) error {
	code, err := s.repo.CreatePasswordReset(ctx, req.LoginID, time.Now().UTC().Add(10*time.Minute))
	if errors.Is(err, ErrPasswordResetThrottled) {
		return nil
	}
	if err != nil {
		observability.L().Error("service error",
			"method", "RequestPasswordReset",
			"error", err,
		)
		return fmt.Errorf("auth.Service.RequestPasswordReset: repo: %w", err)
	}
	if code == "" {
		return nil
	}
	if sendErr := s.codeSender.SendVerification(ctx, req.LoginID, detectChannel(req.LoginID), code); sendErr != nil {
		// 投递失败透传（审查 P2-3，同 SendRegisterCode）。防枚举权衡：
		// 本端点只有已存在的账号会走到投递，网关故障期确实形成「存在→
		// 10600 / 不存在→成功」的差异——但该差异仅在网关整体故障（全局
		// outage、运维可观测，非攻击者可控状态）下出现，且此时静默假成功
		// 更糟：所有找回密码的用户无任何信号地卡死。正常网关下两端同为
		// 成功，无存在性侧信道。
		observability.L().Warn("verification code delivery failed for password reset",
			"method", "RequestPasswordReset",
			"login_id", observability.Redact(req.LoginID),
			"error", sendErr,
		)
		return fmt.Errorf("auth.Service.RequestPasswordReset: deliver: %w", sendErr)
	}
	return nil
}

func (s *Service) ResetPassword(ctx context.Context, req ResetPasswordRequest, meta platformauth.SessionMeta) (platformauth.AuthResult, error) {
	startedAt := time.Now()
	account, err := s.repo.ResetPassword(ctx, req.LoginID, req.VerificationCode, req.NewPassword)
	if err != nil {
		observability.L().Error("service error",
			"method", "ResetPassword",
			"error", err,
			"elapsed_ms", time.Since(startedAt).Milliseconds(),
		)
		return platformauth.AuthResult{}, fmt.Errorf("auth.Service.ResetPassword: repo: %w", err)
	}
	s.closeAllSessions(ctx, account.UserID, "password_changed")
	s.auth.RevokeOtherSessions(ctx, account.UserID, "")
	result, err := s.auth.IssueSessionWithMeta(ctx, account.UserID, meta)
	if err != nil {
		observability.L().Error("service error",
			"method", "ResetPassword",
			"error", err,
			"elapsed_ms", time.Since(startedAt).Milliseconds(),
		)
		return platformauth.AuthResult{}, fmt.Errorf("auth.Service.ResetPassword: issue session: %w", err)
	}
	return result, nil
}

func (s *Service) ChangePassword(ctx context.Context, req ChangePasswordRequest) error {
	principal, ok := platformauth.PrincipalFromContext(ctx)
	if !ok || principal.UserID == "" {
		return platformauth.Unauthorized("Access Token 无效或过期")
	}
	if err := s.repo.ChangePassword(ctx, principal.UserID, req.OldPassword, req.NewPassword); err != nil {
		return fmt.Errorf("auth.Service.ChangePassword: repo: %w", err)
	}
	s.closeOtherSessions(ctx, principal.UserID, principal.SessionID, "password_changed")
	s.auth.RevokeOtherSessions(ctx, principal.UserID, principal.SessionID)
	return nil
}

func (s *Service) RevokeAll(ctx context.Context) RevokeOtherSessionsResult {
	principal, ok := platformauth.PrincipalFromContext(ctx)
	if !ok {
		return RevokeOtherSessionsResult{}
	}
	s.closeOtherSessions(ctx, principal.UserID, principal.SessionID, "revoked_by_user")
	return RevokeOtherSessionsResult{RevokedCount: s.auth.RevokeOtherSessions(ctx, principal.UserID, principal.SessionID)}
}

func (s *Service) profileDefaults(ctx context.Context) ProfileDefaults {
	now := time.Now().UTC()
	defaults := s.appConfig.Get(ctx).Defaults
	return ProfileDefaults{
		AvatarURL:    defaults.DefaultUserAvatar,
		LocationName: defaults.DefaultLocation.LocationName,
		Latitude:     defaults.DefaultLocation.Point.Latitude,
		Longitude:    defaults.DefaultLocation.Point.Longitude,
		CreatedAt:    now,
	}
}

func (s *Service) closeAllSessions(ctx context.Context, userID, reason string) {
	sessions := s.auth.ListSessions(ctx, userID, "")
	for _, session := range sessions {
		realtime.DefaultHub().CloseSession(session.SessionID, sessionRevokedEvent(reason))
	}
}

func (s *Service) closeOtherSessions(ctx context.Context, userID, currentSessionID, reason string) {
	sessions := s.auth.ListSessions(ctx, userID, currentSessionID)
	for _, session := range sessions {
		if session.SessionID == currentSessionID {
			continue
		}
		realtime.DefaultHub().CloseSession(session.SessionID, sessionRevokedEvent(reason))
	}
}

// sessionRevokedEvent builds the SessionRevoked event (spec api_spec_websocket.md)
// sent before the connection is closed. reason is one of device_evicted /
// revoked_by_user / password_changed.
func sessionRevokedEvent(reason string) realtime.Event {
	return realtime.Event{
		Type: realtime.EventTypeSessionRevoked,
		Payload: realtime.SessionRevokedPayload{
			Reason:    reason,
			RevokedAt: time.Now().UTC().Format(time.RFC3339),
		},
	}
}

func maskIPAddress(raw string) string {
	if raw == "" {
		return ""
	}
	if strings.Count(raw, ".") == 3 {
		parts := strings.Split(raw, ".")
		return parts[0] + "." + parts[1] + ".xxx.xxx"
	}
	if strings.Contains(raw, ":") {
		parts := strings.Split(raw, ":")
		if len(parts) > 2 {
			return strings.Join(parts[:2], ":") + ":xxxx:xxxx"
		}
	}
	return raw
}
