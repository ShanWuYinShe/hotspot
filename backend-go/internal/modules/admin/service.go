package admin

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	systemmodule "hotspot/backend-go/internal/modules/system"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/observability"
	"hotspot/backend-go/internal/platform/requestmeta"
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
	// sessionCacheInvalidator 封禁逐条撤销会话后失效平台 auth 的会话校验缓存：
	// admin 仓库直接 UPDATE auth_sessions（不经 platform auth SessionStore 装饰器），
	// 不主动失效会让被撤销会话在缓存 TTL 内继续可用。bootstrap 接线
	// platformauth.Service.InvalidateSessionCache；nil 时跳过（单测/未接线场景）。
	sessionCacheInvalidator func(sessionIDs ...string)
	// announcementsVersionInvalidator 公告 create/update/delete 成功后失效 system
	// 模块的公告版本缓存（bootstrap ETag）。admin 通过自己的 systemRepo 实例写库，
	// 与 system 模块的缓存互不可见，必须显式失效；bootstrap 接线 system.Service 的
	// InvalidateAnnouncementsVersion；nil 时跳过（单测/未接线场景）。
	announcementsVersionInvalidator func()
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

// SetSessionCacheInvalidator wires the platform auth session-cache invalidation hook.
func (s *Service) SetSessionCacheInvalidator(fn func(sessionIDs ...string)) {
	s.sessionCacheInvalidator = fn
}

// SetAnnouncementsVersionInvalidator wires the system announcement-version cache hook.
func (s *Service) SetAnnouncementsVersionInvalidator(fn func()) {
	s.announcementsVersionInvalidator = fn
}

func (s *Service) invalidateAnnouncementsVersion() {
	if s.announcementsVersionInvalidator != nil {
		s.announcementsVersionInvalidator()
	}
}

func (s *Service) invalidateSessionCache(sessionIDs ...string) {
	if s.sessionCacheInvalidator != nil {
		s.sessionCacheInvalidator(sessionIDs...)
	}
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
