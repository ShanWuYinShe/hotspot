package admin

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	groupsmodule "hotspot/backend-go/internal/modules/groups"
	systemmodule "hotspot/backend-go/internal/modules/system"
	usersmodule "hotspot/backend-go/internal/modules/users"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/observability"
	"hotspot/backend-go/internal/platform/requestmeta"

	"hotspot/backend-go/internal/platform/persistence"
)

type Service struct {
	repo            Repository
	provider        Provider
	moderationStore ModerationStore
	pushService     announcementPushService
	// banCacheInvalidator 封禁状态变更后失效平台 auth 的进程内缓存（bootstrap
	// 接线 platformauth.Service.InvalidateBanCache）：保证解封/封禁即时生效，
	// 不等 banCacheTTL 窗口。nil 时跳过（单测/未接线场景）。
	banCacheInvalidator func(userIDs ...string)
	// momentModerator / commentModerator：举报裁决的内容删除走归属模块服务
	//（admin 不再直接 UPDATE moments/comments 表，架构审计 P0-4）。bootstrap
	// 注入缺失时裁决删除返回错误，绝不静默跳过。
	momentModerator  momentModerator
	commentModerator commentModerator
	// reportAdmin：举报裁决 UPDATE / 保留期 DELETE 归位 reports 模块（admin
	// 不再直写 reports 表，架构审计 P0-4）。bootstrap 注入缺失时裁决返回错误。
	reportAdmin             reportAdminStore
	groupMessageModerator   messageModerator
	privateMessageModerator messageModerator
	groupModerator          groupModerator
	accountStateWriter      accountStateWriter
	sessionAdmin            sessionAdmin
	// runInTx 让「跨模块编排写」（举报裁决：内容删除 + 封禁 + 举报终态 + 裁决
	// 审计；公告创建：落库 + 审计 + 广播入队）在同一事务内提交；nil 时退化为
	// 直跑（单测/内存装配）。通知/广播等外部副作用经 aftercommit 在 COMMIT 后
	// 执行。
	runInTx persistence.TxRunner
	// eventOutbox 是关键实时事件的事件生产者（bootstrap 注入通用 outbox
	// writer）：公告创建/举报裁决与事件入队同事务提交，进程在 COMMIT 后崩溃
	// 也能由后台 dispatcher 补投；nil 时退化为同步广播（单测/内存装配）。
	eventOutbox outboxWriter
}

// outboxWriter 是平台 outbox.Writer 的最小消费接口（bootstrap 注入）。
type outboxWriter interface {
	Enqueue(ctx context.Context, topic string, payload any) error
}

// SetEventOutbox 注入关键实时事件 outbox writer（bootstrap 装配）。
func (s *Service) SetEventOutbox(w outboxWriter) { s.eventOutbox = w }

// AnnouncementBroadcastPayload 是 outbox TopicSystemAnnouncement 的 payload
// （admin 生产、bootstrap dispatcher handler 消费）。
type AnnouncementBroadcastPayload struct {
	Item     systemmodule.SystemAnnouncementItem `json:"item"`
	NotifyID string                              `json:"notify_id"`
}

// ReportHandledBroadcastPayload 是 outbox TopicAdminReportHandled 的 payload：
// 举报人通知（notify 行已在裁决事务内落库）的 WS 广播补投。
type ReportHandledBroadcastPayload struct {
	NotifyID       string `json:"notify_id"`
	ReporterUserID string `json:"reporter_user_id"`
	ReportID       string `json:"report_id"`
	TargetType     string `json:"target_type"`
	TargetID       string `json:"target_id"`
	Note           string `json:"note,omitempty"`
	Resolved       bool   `json:"resolved"`
	OccurredAt     string `json:"occurred_at"`
}

// SetTxRunner wires the optional transaction runner（bootstrap 注入；nil 时直跑）。
func (s *Service) SetTxRunner(runInTx persistence.TxRunner) { s.runInTx = runInTx }

// errAdminDecisionAborted 是「业务错误需回滚已执行副作用」的内部哨兵：
// 包装事务时 APIError 不当作 Go error 上抛，但必须让事务回滚。
var errAdminDecisionAborted = errors.New("admin: decision aborted")

// runDecisionInTx 把跨模块编排写（内容删除/封禁/会话撤销/举报终态）放入同一
// 事务；未注入 TxRunner（单测/内存装配）时直跑，语义不变。业务错误（含并发
// 败者）通过哨兵回滚已执行副作用，再原样返回 APIError。
func (s *Service) runDecisionInTx(ctx context.Context, fn func(context.Context) (*httpx.APIError, error)) (*httpx.APIError, error) {
	if s.runInTx == nil {
		return fn(ctx)
	}
	var apiErr *httpx.APIError
	var opErr error
	txErr := s.runInTx(ctx, func(txCtx context.Context) error {
		apiErr, opErr = fn(txCtx)
		if apiErr != nil {
			return errAdminDecisionAborted
		}
		return opErr
	})
	if errors.Is(txErr, errAdminDecisionAborted) {
		return apiErr, nil
	}
	if opErr != nil {
		return nil, opErr
	}
	if txErr != nil {
		return nil, txErr
	}
	return nil, nil
}

// momentModerator / commentModerator 是 admin 消费的内容审核删除能力最小接口
// （由 moments/comments 模块 Service 实现）。
type momentModerator interface {
	DeleteAsAdmin(ctx context.Context, momentID string) error
}

type commentModerator interface {
	DeleteAsAdmin(ctx context.Context, commentID string) error
}

// reportAdminStore 是 reports 模块的管理端能力（裁决/保留期清理）。
type reportAdminStore interface {
	MarkHandledAsAdmin(ctx context.Context, reportID, status string, handledAt time.Time, handledByID *string, note *string) (bool, error)
	DeleteHandledBefore(ctx context.Context, before time.Time, limit int) (int64, error)
}

// accountStateWriter 写入账户封禁状态（users 模块 Service 实现；users 表归属
// users 模块，admin 不再直写）。
type accountStateWriter interface {
	SetAccountBanState(ctx context.Context, userID string, state usersmodule.AccountBanState) error
}

// sessionAdmin 是平台认证服务的管理端会话能力（列出/撤销任意用户会话；
// auth_sessions 归属 platform/auth，admin 不再直写）。
type sessionAdmin interface {
	ListActiveSessionIDsAsAdmin(ctx context.Context, userID string) ([]string, error)
	RevokeSessionAsAdmin(ctx context.Context, sessionID string) (bool, error)
}

// groupModerator 是平台审核解散群能力（groups 模块 Service 实现）。返回群信息
// 供审计/日志使用；群不存在返回 groups.ErrGroupNotFound。
type groupModerator interface {
	DismissAsAdmin(ctx context.Context, operatorID, groupID, reason string) (groupsmodule.GroupInfo, error)
}

// messageModerator 是按 message_id 撤回消息的审核能力（groupmessages 与
// privatemessages Service 均实现相同签名；返回是否命中，供裁决侧先群后私）。
type messageModerator interface {
	RecallMessageByIDAsAdmin(ctx context.Context, operatorID, messageID string) (bool, error)
}

type announcementPushService interface {
	PushSystemAnnouncement(ctx context.Context, item systemmodule.SystemAnnouncementItem, notificationID string)
	// Report-outcome notifications (resolve/dismiss), returning the generated
	// notification_id for the realtime payload.
	AppendReportResolved(ctx context.Context, targetUserID string, reportID string, targetType string, targetID string, note string) (string, error)
	AppendReportDismissed(ctx context.Context, targetUserID string, reportID string, targetType string, targetID string, note string) (string, error)
}

// NewService 要求显式注入 ModerationStore（bootstrap 装配）：生产不允许回退
// 内存实现（架构审计 C1「生产代码 fail-fast」），缺失即编程错误。
func NewService(repo Repository, provider Provider, moderationStore ModerationStore, pushService announcementPushService) *Service {
	if moderationStore == nil {
		panic("admin.NewService: moderationStore is required")
	}
	return &Service{repo: repo, provider: provider, moderationStore: moderationStore, pushService: pushService}
}

// SetBanCacheInvalidator wires the platform auth ban-cache invalidation hook
// (bootstrap 接线；userIDs 传空表示全量失效)。
func (s *Service) SetBanCacheInvalidator(fn func(userIDs ...string)) {
	s.banCacheInvalidator = fn
}

// SetMomentModerator wires the moments module's admin delete capability.
func (s *Service) SetMomentModerator(m momentModerator) { s.momentModerator = m }

// SetCommentModerator wires the comments module's admin delete capability.
func (s *Service) SetCommentModerator(c commentModerator) { s.commentModerator = c }

// SetReportAdminStore wires the reports module's admin handling capability.
func (s *Service) SetReportAdminStore(store reportAdminStore) { s.reportAdmin = store }

// SetAccountStateWriter wires the users module's ban-state writer.
func (s *Service) SetAccountStateWriter(w accountStateWriter) { s.accountStateWriter = w }

// SetSessionAdmin wires the platform auth admin session capability.
func (s *Service) SetSessionAdmin(a sessionAdmin) { s.sessionAdmin = a }

// SetGroupModerator wires the groups module's admin dismiss capability.
func (s *Service) SetGroupModerator(g groupModerator) { s.groupModerator = g }

// SetMessageModerators wires group/private message admin recall capabilities.
func (s *Service) SetMessageModerators(groupModerator, privateModerator messageModerator) {
	s.groupMessageModerator = groupModerator
	s.privateMessageModerator = privateModerator
}

func (s *Service) invalidateBanCache(userID string) {
	if s.banCacheInvalidator != nil {
		s.banCacheInvalidator(userID)
	}
}

func (s *Service) actorFromContext(ctx context.Context) (AdminActor, bool, error) {
	principal, ok := platformauth.PrincipalFromContext(ctx)
	if !ok || principal.UserID == "" {
		return AdminActor{}, false, &httpx.APIError{Code: httpx.CodeUnauthorized, Message: "Access Token 无效或过期"}
	}
	actor, found, err := s.repo.GetActor(ctx, principal.UserID)
	if err != nil {
		return AdminActor{}, false, err
	}
	return actor, found, nil
}

func (s *Service) requirePermission(ctx context.Context, permission string) (AdminActor, *httpx.APIError, error) {
	actor, found, err := s.actorFromContext(ctx)
	if err != nil {
		if apiErr := asAPIError(err); apiErr != nil {
			return AdminActor{}, apiErr, nil
		}
		return AdminActor{}, nil, err
	}
	if !found {
		return AdminActor{}, httpx.Forbidden(httpx.ReasonNotAdmin, "非平台管理员"), nil
	}
	if !actor.HasPermission(permission) {
		return AdminActor{}, httpx.Forbidden(httpx.ReasonInsufficientAdminPermission, "平台管理员权限不足: "+permission), nil
	}
	return actor, nil, nil
}

func formatAuditLogID(action, targetID string, ts time.Time) string {
	return fmt.Sprintf("audit_%s_%s_%d", action, targetID, ts.UnixNano())
}

// asAPIError 从错误链中提取 *httpx.APIError（errors.As 语义，兼容 %w 包装），
// 提取不到返回 nil。admin 模块的业务错误统一经此出口识别，禁止直接类型断言。
func asAPIError(err error) *httpx.APIError {
	var apiErr *httpx.APIError
	if errors.As(err, &apiErr) {
		return apiErr
	}
	return nil
}

func applyPositiveIntPatch(target *int, patch *int) *httpx.APIError {
	if patch == nil {
		return nil
	}
	if *patch <= 0 {
		return httpx.InvalidRequest("参数错误")
	}
	*target = *patch
	return nil
}

func newAuditEntry(ctx context.Context, actor AdminActor, entry AuditLogEntry) AuditLogEntry {
	if entry.Operator.UserID == "" {
		nickname := actor.Nickname
		if nickname == "" {
			nickname = actor.UserID
		}
		entry.Operator = UserInfo{UserID: actor.UserID, Nickname: nickname}
	}
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now().UTC()
	}
	if entry.LogID == "" {
		if requestID := observability.RequestIDFromContext(ctx); requestID != "" {
			entry.LogID = "audit_" + requestID
		} else {
			entry.LogID = formatAuditLogID(string(entry.Action), entry.TargetID, entry.CreatedAt)
		}
	}
	if entry.IPAddress == "" {
		entry.IPAddress = requestIPFromContext(ctx)
	}
	if entry.DeviceSummary == "" {
		entry.DeviceSummary = deviceSummaryFromContext(ctx)
	}
	if entry.IPLocation == "" && entry.IPAddress != "" {
		if req, ok := requestFromContext(ctx); ok {
			entry.IPLocation = requestmeta.ClientCountry(req)
		}
		if entry.IPLocation == "" {
			entry.IPLocation = "unknown"
		}
	}
	return entry
}

func requestIPFromContext(ctx context.Context) string {
	req, _ := requestFromContext(ctx)
	return requestmeta.ClientIP(req)
}

func deviceSummaryFromContext(ctx context.Context) string {
	req, _ := requestFromContext(ctx)
	if req == nil {
		return ""
	}
	ua := strings.TrimSpace(req.Header.Get("User-Agent"))
	if ua == "" {
		return ""
	}
	if len(ua) > 120 {
		return ua[:120]
	}
	return ua
}

type requestContextKey struct{}

func withRequest(ctx context.Context, req *http.Request) context.Context {
	return context.WithValue(ctx, requestContextKey{}, req)
}

func requestFromContext(ctx context.Context) (*http.Request, bool) {
	req, ok := ctx.Value(requestContextKey{}).(*http.Request)
	return req, ok
}
