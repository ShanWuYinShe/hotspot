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
	// runInTx 让「跨模块编排写」（举报裁决：内容删除 + 封禁 + 举报终态）在
	// 同一事务内提交；nil 时退化为直跑（单测/内存装配）。审计/通知等副作用
	// 经 aftercommit 在 COMMIT 后执行。
	runInTx persistence.TxRunner
}

// SetTxRunner wires the optional transaction runner（bootstrap 注入；nil 时直跑）。
func (s *Service) SetTxRunner(runInTx persistence.TxRunner) { s.runInTx = runInTx }

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
	DeleteHandledBefore(ctx context.Context, before time.Time) (int64, error)
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
	AppendSystemAnnouncement(ctx context.Context, userIDs []string, item systemmodule.SystemAnnouncementItem) (string, error)
	// NewAnnouncementNotifyID 预生成整条公告共用的 notify_id；
	// AppendSystemAnnouncementWithID 用该 id 落库某一批收件人——全量 fanout
	// 会分多批，但契约要求整条公告对应同一个 WS notify_id（见 notify.Service）。
	NewAnnouncementNotifyID() string
	AppendSystemAnnouncementWithID(ctx context.Context, id string, userIDs []string, item systemmodule.SystemAnnouncementItem) (string, error)
	PushSystemAnnouncement(item systemmodule.SystemAnnouncementItem, notificationID string)
	// Report-outcome notifications (resolve/dismiss), returning the generated
	// notification_id for the realtime payload.
	AppendReportResolved(ctx context.Context, targetUserID string, reportID string, targetType string, targetID string, note string) (string, error)
	AppendReportDismissed(ctx context.Context, targetUserID string, reportID string, targetType string, targetID string, note string) (string, error)
}

func NewService(repo Repository, provider Provider, moderationStore ModerationStore, pushService announcementPushService) *Service {
	if moderationStore == nil {
		moderationStore = NewInMemoryModerationStore()
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
