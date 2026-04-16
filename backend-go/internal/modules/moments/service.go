package moments

import (
	"context"
	"strings"
	"time"

	"hotspot/backend-go/internal/modules/comments"
	"hotspot/backend-go/internal/modules/notifications"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/observability"
	"hotspot/backend-go/internal/platform/realtime"
	"hotspot/backend-go/internal/platform/resourceguard"
)

type Service struct {
	repo          Repository
	notifications momentNotificationWriter
}

type momentNotificationWriter interface {
	AppendMomentLiked(ctx context.Context, targetUserID string, actor map[string]any, momentID string, momentSummary map[string]any) error
	AppendMentioned(ctx context.Context, targetUserID string, actor map[string]any, targetType string, momentID string, momentSummary map[string]any, commentID string, commentPreview string) error
}

func NewService() *Service {
	return NewServiceWithDeps(NewInMemoryRepository(), notifications.NewService())
}

func NewServiceWithRepo(repo Repository) *Service {
	return NewServiceWithDeps(repo, notifications.NewService())
}

func NewServiceWithDeps(repo Repository, notificationWriter momentNotificationWriter) *Service {
	return &Service{repo: repo, notifications: notificationWriter}
}

func (s *Service) Create(ctx context.Context, req CreateMomentRequest) (MomentItem, error) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	item, err := s.repo.Create(ctx, principal.UserID, req)
	if err != nil {
		observability.L().Error("moments.create failed", "error", err)
		return MomentItem{}, err
	}
	resourceguard.RecordMoment(ctx, item.MomentID)
	s.notifyMomentMentions(ctx, principal.UserID, item)
	return item, nil
}

func (s *Service) OpenTheDoor(ctx context.Context, momentID string) (MomentItem, bool, error) {
	return s.repo.Get(ctx, momentID)
}

func (s *Service) NotificationContext(ctx context.Context, momentID string) (comments.MomentNotificationContext, bool, error) {
	item, ok, err := s.repo.Get(ctx, momentID)
	if err != nil || !ok {
		return comments.MomentNotificationContext{}, ok, err
	}
	return comments.MomentNotificationContext{
		UserID: item.User.UserID,
		Summary: map[string]any{
			"moment_id":           item.MomentID,
			"description_preview": summarizeMomentDescription(item.Description),
		},
	}, true, nil
}

func (s *Service) ListMine(ctx context.Context) (PaginatedMoments, error) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	return s.repo.ListMine(ctx, principal.UserID)
}

func (s *Service) Exists(ctx context.Context, momentID string) (bool, error) {
	return s.repo.Exists(ctx, momentID)
}

func (s *Service) CheckCommentable(ctx context.Context, momentID string) (found bool, canComment bool, err error) {
	item, ok, err := s.repo.Get(ctx, momentID)
	if err != nil || !ok {
		return ok, false, err
	}
	principal, _ := platformauth.PrincipalFromContext(ctx)
	if item.PrivacyLevel == "private" && item.User.UserID != principal.UserID {
		return true, false, nil
	}
	return true, true, nil
}

func (s *Service) IncrementCommentCount(ctx context.Context, momentID string) error {
	return s.repo.IncrementCommentCount(ctx, momentID)
}

func (s *Service) DecrementCommentCount(ctx context.Context, momentID string) error {
	return s.repo.DecrementCommentCount(ctx, momentID)
}

func (s *Service) IncrementSharesCount(ctx context.Context, momentID string) error {
	return s.repo.IncrementSharesCount(ctx, momentID)
}

func (s *Service) Search(ctx context.Context, req SearchMomentsRequest) (PaginatedMoments, error) {
	return s.repo.Search(ctx, req)
}

func (s *Service) Update(ctx context.Context, req EditMomentRequest) (MomentItem, error) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	item, ok, err := s.repo.Get(ctx, req.MomentID)
	if !ok {
		if err != nil {
			return MomentItem{}, err
		}
		return MomentItem{}, ErrMomentNotFound
	}
	if item.User.UserID != principal.UserID {
		return MomentItem{}, ErrPermissionDenied
	}
	if err := s.repo.Update(ctx, principal.UserID, req); err != nil {
		observability.L().Error("moments.update failed", "error", err, "moment_id", req.MomentID)
		return MomentItem{}, err
	}
	updated, _, err := s.repo.Get(ctx, req.MomentID)
	if err != nil {
		observability.L().Error("moments.update reload failed", "error", err, "moment_id", req.MomentID)
		return MomentItem{}, err
	}
	s.notifyMomentMentions(ctx, principal.UserID, updated)
	return updated, nil
}

func (s *Service) Delete(ctx context.Context, momentID string) error {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	item, ok, err := s.repo.Get(ctx, momentID)
	if !ok {
		if err != nil {
			return err
		}
		return ErrMomentNotFound
	}
	if item.User.UserID != principal.UserID {
		return ErrPermissionDenied
	}
	return s.repo.Delete(ctx, momentID)
}

func (s *Service) Like(ctx context.Context, momentID string) error {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	exists, err := s.repo.Exists(ctx, momentID)
	if err != nil {
		observability.L().Error("moments.like failed", "error", err, "moment_id", momentID)
		return err
	}
	if !exists {
		return ErrMomentNotFound
	}
	newlyLiked, err := s.repo.Like(ctx, principal.UserID, momentID)
	if err != nil {
		observability.L().Error("moments.like failed", "error", err, "moment_id", momentID, "user_id", principal.UserID)
		return err
	}
	if !newlyLiked {
		return nil
	}
	momentContext, ok, err := s.NotificationContext(ctx, momentID)
	if err != nil {
		observability.L().Error("moments.like load notification context failed", "error", err, "moment_id", momentID)
		return nil
	}
	if !ok || strings.TrimSpace(momentContext.UserID) == "" || momentContext.UserID == principal.UserID {
		return nil
	}
	actor := buildMomentActor(principal.UserID)
	momentSummary := cloneMomentSummary(momentContext.Summary, momentID)
	if s.notifications != nil {
		if err := s.notifications.AppendMomentLiked(ctx, momentContext.UserID, actor, momentID, momentSummary); err != nil {
			observability.L().Error("moments.like append notification failed", "error", err, "moment_id", momentID, "target_user_id", momentContext.UserID)
		}
	}
	realtime.DefaultHub().BroadcastToUser(momentContext.UserID, realtime.Event{Type: "MomentLiked", Payload: map[string]any{
		"moment_id":      momentID,
		"moment_summary": cloneMomentSummary(momentSummary, momentID),
		"actor":          actor,
		"liked_at":       time.Now().UTC().Format(time.RFC3339),
	}})
	return nil
}

func (s *Service) Unlike(ctx context.Context, momentID string) error {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	return s.repo.Unlike(ctx, principal.UserID, momentID)
}

func (s *Service) Favorite(ctx context.Context, momentID string) error {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	exists, err := s.repo.Exists(ctx, momentID)
	if err != nil {
		observability.L().Error("moments.favorite failed", "error", err, "moment_id", momentID)
		return err
	}
	if !exists {
		return ErrMomentNotFound
	}
	return s.repo.Favorite(ctx, principal.UserID, momentID)
}

func (s *Service) Unfavorite(ctx context.Context, momentID string) error {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	return s.repo.Unfavorite(ctx, principal.UserID, momentID)
}

func (s *Service) ListLikes(ctx context.Context, req ListLikesRequest) ([]LikeItem, string, error) {
	limit := 20
	if req.Pagination != nil && req.Pagination.Limit > 0 {
		limit = req.Pagination.Limit
	}
	if limit > 100 {
		limit = 100
	}
	cursor := ""
	if req.Pagination != nil {
		cursor = req.Pagination.Cursor
	}
	return s.repo.ListLikes(ctx, req.MomentID, limit, cursor)
}

func (s *Service) ListFavorites(ctx context.Context, req MyFavoritesRequest) (PaginatedMoments, error) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	limit := 20
	if req.Pagination != nil && req.Pagination.Limit > 0 {
		limit = req.Pagination.Limit
	}
	if limit > 100 {
		limit = 100
	}
	cursor := ""
	if req.Pagination != nil {
		cursor = req.Pagination.Cursor
	}
	return s.repo.ListFavorites(ctx, principal.UserID, limit, cursor)
}

func buildMomentActor(userID string) map[string]any {
	userID = strings.TrimSpace(userID)
	return map[string]any{"user_id": userID, "nickname": userID}
}

func cloneMomentSummary(summary map[string]any, momentID string) map[string]any {
	cloned := make(map[string]any, len(summary)+1)
	for key, value := range summary {
		cloned[key] = value
	}
	if len(cloned) == 0 {
		cloned["moment_id"] = momentID
	}
	return cloned
}

func summarizeMomentDescription(description string) string {
	value := strings.TrimSpace(description)
	if len(value) > 80 {
		return value[:80]
	}
	return value
}

func (s *Service) notifyMomentMentions(ctx context.Context, actorUserID string, item MomentItem) {
	if s.notifications == nil {
		return
	}
	momentSummary := cloneMomentSummary(map[string]any{"moment_id": item.MomentID, "description_preview": summarizeMomentDescription(item.Description)}, item.MomentID)
	actor := buildMomentActor(actorUserID)
	for _, mentionedUserID := range normalizeMomentMentions(item.Mentions) {
		if mentionedUserID == actorUserID {
			continue
		}
		if err := s.notifications.AppendMentioned(ctx, mentionedUserID, actor, "moment", item.MomentID, momentSummary, "", ""); err != nil {
			observability.L().Error("moments.mentions append notification failed", "error", err, "moment_id", item.MomentID, "mentioned_user_id", mentionedUserID)
		}
		realtime.DefaultHub().BroadcastToUser(mentionedUserID, realtime.Event{Type: "MessageMentioned", Payload: map[string]any{
			"target_type":    "moment",
			"moment_id":      item.MomentID,
			"moment_summary": cloneMomentSummary(momentSummary, item.MomentID),
			"actor":          actor,
			"mentioned_at":   item.CreatedAt,
		}})
	}
}
