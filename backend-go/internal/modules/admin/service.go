package admin

import (
	"context"
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
}

type announcementPushService interface {
	AppendSystemAnnouncement(ctx context.Context, userIDs []string, item systemmodule.SystemAnnouncementItem) error
	PushSystemAnnouncement(item systemmodule.SystemAnnouncementItem)
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

func (s *Service) actorFromContext(ctx context.Context) (AdminActor, bool, error) {
	principal, ok := platformauth.PrincipalFromContext(ctx)
	if !ok || principal.UserID == "" {
		return AdminActor{}, false, &httpx.APIError{Code: 10100, Message: "Access Token 无效或过期"}
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
		if apiErr, ok := err.(*httpx.APIError); ok {
			return AdminActor{}, apiErr, nil
		}
		return AdminActor{}, nil, err
	}
	if !found {
		return AdminActor{}, &httpx.APIError{Code: 10200, Message: "非系统管理员"}, nil
	}
	if !actor.HasPermission(permission) {
		return AdminActor{}, &httpx.APIError{Code: 10200, Message: "系统管理员权限不足: " + permission}, nil
	}
	return actor, nil, nil
}

func formatAuditLogID(action, targetID string, ts time.Time) string {
	return fmt.Sprintf("audit_%s_%s_%d", action, targetID, ts.UnixNano())
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
		entry.Operator = UserInfo{UserID: actor.UserID, Nickname: actor.UserID}
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
