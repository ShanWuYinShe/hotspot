package moments

import (
	"context"
	"fmt"
	"strings"
	"time"

	"hotspot/backend-go/internal/modules/comments"
	"hotspot/backend-go/internal/modules/notifications"
	"hotspot/backend-go/internal/platform/appconfig"
	"hotspot/backend-go/internal/platform/actor"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/observability"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/realtime"
	"hotspot/backend-go/internal/platform/resourceguard"
)

type Service struct {
	repo          Repository
	notifications momentNotificationWriter
	users         actor.Provider
	images        ImageResolver
	appConfig     appconfig.Source
	runInTx       persistence.TxRunner
}

type momentNotificationWriter interface {
	AppendMomentLiked(ctx context.Context, targetUserID string, actor map[string]any, momentID string, momentSummary map[string]any) error
	AppendMentioned(ctx context.Context, targetUserID string, actor map[string]any, targetType string, momentID string, momentSummary map[string]any, commentID string, commentPreview string) error
}

func NewService() *Service {
	return NewServiceWithDepsAndImagesAndConfig(NewInMemoryRepository(), notifications.NewService(), nil, nil, nil, nil)
}

func NewServiceWithRepo(repo Repository) *Service {
	return NewServiceWithDepsAndImagesAndConfig(repo, notifications.NewService(), nil, nil, nil, nil)
}

func NewServiceWithDeps(repo Repository, notificationWriter momentNotificationWriter, actorProvider actor.Provider) *Service {
	return NewServiceWithDepsAndImagesAndConfig(repo, notificationWriter, actorProvider, nil, nil, nil)
}

func NewServiceWithDepsAndImages(repo Repository, notificationWriter momentNotificationWriter, actorProvider actor.Provider, images ImageResolver) *Service {
	return NewServiceWithDepsAndImagesAndConfig(repo, notificationWriter, actorProvider, images, nil, nil)
}

// NewServiceWithDepsAndImagesAndConfig wires all dependencies including app
// config for moment image limits and a transaction runner for atomic updates.
func NewServiceWithDepsAndImagesAndConfig(repo Repository, notificationWriter momentNotificationWriter, actorProvider actor.Provider, images ImageResolver, cfg appconfig.Source, runInTx persistence.TxRunner) *Service {
	return &Service{repo: repo, notifications: notificationWriter, users: actorProvider, images: images, appConfig: cfg, runInTx: runInTx}
}

func (s *Service) Create(ctx context.Context, req CreateMomentRequest) (MomentItem, error) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	if err := s.validateImageOwnership(ctx, principal.UserID, req.ImageFileIDs); err != nil {
		return MomentItem{}, err
	}
		item, err := s.repo.Create(ctx, principal.UserID, req)
		if err != nil {
			observability.L().Error("moments.create failed", "error", err)
			return MomentItem{}, fmt.Errorf("moments.Service.Create: repo: %w", err)
		}
	resourceguard.RecordMoment(ctx, item.MomentID)
	s.notifyMomentMentions(ctx, principal.UserID, item)
	s.hydrateImages(ctx, &item)
	return item, nil
}

func (s *Service) OpenTheDoor(ctx context.Context, momentID string) (MomentItem, bool, error) {
	item, ok, err := s.repo.Get(ctx, momentID)
	if err != nil || !ok {
		return MomentItem{}, false, err
	}
	principal, _ := platformauth.PrincipalFromContext(ctx)
	isOwner := principal.UserID != "" && item.User.UserID == principal.UserID
	if item.PrivacyLevel == "private" && !isOwner {
		return MomentItem{}, false, nil
	}
	if !isOwner {
		item.PrivacyLevel = ""
	}
	if principal.UserID != "" {
		item.MyState = s.repo.LoadMyState(ctx, momentID, principal.UserID)
	}
	s.hydrateImages(ctx, &item)
	return item, true, nil
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

func (s *Service) ListMine(ctx context.Context, req MyMomentsRequest) (PaginatedMoments, error) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	limit := 20
	cursor := ""
	if req.Pagination != nil {
		if req.Pagination.Limit > 0 {
			limit = req.Pagination.Limit
		}
		if limit > 100 {
			limit = 100
		}
		cursor = req.Pagination.Cursor
	}
	result, err := s.repo.ListMine(ctx, principal.UserID, limit, cursor)
	if err != nil {
		return result, err
	}
	s.hydrateImagesBatch(ctx, result.Items)
	return result, nil
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
	result, err := s.repo.Search(ctx, req)
	if err != nil {
		return result, err
	}
	s.hydrateImagesBatch(ctx, result.Items)
	return result, nil
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
	if err := s.validateImageOwnership(ctx, principal.UserID, req.ImageFileIDs); err != nil {
		return MomentItem{}, err
	}
	updateFn := func(ctx context.Context) error {
		return s.repo.Update(ctx, principal.UserID, req)
	}
		if s.runInTx != nil {
			if err := s.runInTx(ctx, updateFn); err != nil {
				observability.L().Error("moments.update failed", "error", err, "moment_id", req.MomentID)
				return MomentItem{}, fmt.Errorf("moments.Service.Update: tx: %w", err)
			}
		} else {
			if err := updateFn(ctx); err != nil {
				observability.L().Error("moments.update failed", "error", err, "moment_id", req.MomentID)
				return MomentItem{}, fmt.Errorf("moments.Service.Update: repo: %w", err)
			}
		}
	updated, _, err := s.repo.Get(ctx, req.MomentID)
	if err != nil {
		observability.L().Error("moments.update reload failed", "error", err, "moment_id", req.MomentID)
		return MomentItem{}, err
	}
	s.notifyMomentMentions(ctx, principal.UserID, updated)
	s.hydrateImages(ctx, &updated)
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
		if err := s.repo.Delete(ctx, momentID); err != nil {
			return fmt.Errorf("moments.Service.Delete: repo: %w", err)
		}
		return nil
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
	actor, err := s.buildMomentActor(ctx, principal.UserID)
	if err != nil {
		observability.L().Error("moments.like build actor failed", "error", err, "user_id", principal.UserID)
		actor = fallbackActor(principal.UserID)
	}
	momentSummary := cloneMomentSummary(momentContext.Summary, momentID)
	if s.notifications != nil {
		if err := s.notifications.AppendMomentLiked(ctx, momentContext.UserID, actor, momentID, momentSummary); err != nil {
			observability.L().Error("moments.like append notification failed", "error", err, "moment_id", momentID, "target_user_id", momentContext.UserID)
		}
	}
	realtime.DefaultHub().BroadcastToUser(momentContext.UserID, realtime.Event{Type: realtime.EventTypeMomentLiked, Payload: map[string]any{
		"moment_id":      momentID,
		"moment_summary": cloneMomentSummary(momentSummary, momentID),
		"actor":          actor,
		"liked_at":       time.Now().UTC().Format(time.RFC3339),
	}})
	return nil
}

func (s *Service) Unlike(ctx context.Context, momentID string) error {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	exists, err := s.repo.Exists(ctx, momentID)
	if err != nil {
		observability.L().Error("moments.unlike failed", "error", err, "moment_id", momentID)
		return err
	}
	if !exists {
		return ErrMomentNotFound
	}
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
	exists, err := s.repo.Exists(ctx, momentID)
	if err != nil {
		observability.L().Error("moments.unfavorite failed", "error", err, "moment_id", momentID)
		return err
	}
	if !exists {
		return ErrMomentNotFound
	}
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
	result, err := s.repo.ListFavorites(ctx, principal.UserID, limit, cursor)
	if err != nil {
		return result, err
	}
	s.hydrateImagesBatch(ctx, result.Items)
	return result, nil
}

// validateImageOwnership ensures every file_id in the list belongs to userID,
// has category 'moment_image' and is completed, and that the count does not
// exceed the configured max. A nil slice (no update) and an empty slice
// (clear) are both valid. Returns ErrImagesNotOwned on mismatch.
func (s *Service) validateImageOwnership(ctx context.Context, userID string, fileIDs []string) error {
	if len(fileIDs) == 0 {
		return nil
	}
	if s.appConfig != nil {
		maxImages := s.appConfig.Get(ctx).Limits.MaxMomentImages
		if maxImages > 0 && len(fileIDs) > maxImages {
			return ErrTooManyImages
		}
	}
	if s.images == nil {
		return nil
	}
	if _, err := s.images.ResolveMomentImages(ctx, fileIDs, userID); err != nil {
		observability.L().Warn("moments.image_ownership denied", "error", err, "user_id", userID)
		return ErrImagesNotOwned
	}
	return nil
}

// hydrateImages fills a single moment's images in place via the shared helper.
func (s *Service) hydrateImages(ctx context.Context, item *MomentItem) {
	if s.images == nil || item == nil {
		return
	}
	HydrateSingleMoment(ctx, s.repo, s.images, item)
}

// hydrateImagesBatch fills a slice of moments' images in place via one query.
func (s *Service) hydrateImagesBatch(ctx context.Context, items []MomentItem) {
	if s.images == nil || len(items) == 0 {
		return
	}
	HydrateMomentImages(ctx, s.repo, s.images, items)
}

func (s *Service) buildMomentActor(ctx context.Context, userID string) (map[string]any, error) {
	if s.users != nil {
		info, err := s.users.GetActorInfo(ctx, userID)
		if err != nil {
			return nil, err
		}
		return actorFromInfo(info), nil
	}
	return fallbackActor(userID), nil
}

func fallbackActor(userID string) map[string]any {
	userID = strings.TrimSpace(userID)
	return map[string]any{"user_id": userID, "nickname": userID}
}

func actorFromInfo(info actor.Info) map[string]any {
	actor := map[string]any{"user_id": info.UserID, "nickname": info.Nickname}
	if info.AvatarURL != "" {
		actor["avatar_url"] = info.AvatarURL
	}
	return actor
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
	actor, err := s.buildMomentActor(ctx, actorUserID)
	if err != nil {
		observability.L().Error("moments.mentions build actor failed", "error", err, "user_id", actorUserID)
		actor = fallbackActor(actorUserID)
	}
	for _, mentionedUserID := range normalizeMomentMentions(item.Mentions) {
		if mentionedUserID == actorUserID {
			continue
		}
		if err := s.notifications.AppendMentioned(ctx, mentionedUserID, actor, "moment", item.MomentID, momentSummary, "", ""); err != nil {
			observability.L().Error("moments.mentions append notification failed", "error", err, "moment_id", item.MomentID, "mentioned_user_id", mentionedUserID)
		}
		realtime.DefaultHub().BroadcastToUser(mentionedUserID, realtime.Event{Type: realtime.EventTypeMessageMentioned, Payload: map[string]any{
			"target_type":    "moment",
			"moment_id":      item.MomentID,
			"moment_summary": cloneMomentSummary(momentSummary, item.MomentID),
			"actor":          actor,
			"mentioned_at":   item.CreatedAt,
		}})
	}
}
