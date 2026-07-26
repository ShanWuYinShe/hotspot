package reactions

import (
	"context"
	"strings"
	"time"

	"hotspot/backend-go/internal/platform/actor"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/observability"
	"hotspot/backend-go/internal/platform/realtime"
)

// Service orchestrates reaction reads/writes and the side effects (owner
// notification, realtime broadcast) that follow a reaction being added.
type Service struct {
	repo          Repository
	providers     map[ReactTarget]TargetProvider
	notifications reactionNotificationWriter
	users         actor.Provider
}

// NewService returns a Service suitable for tests (in-memory repo, no
// providers, no notifications).
func NewService() *Service {
	return NewServiceWithDeps(NewInMemoryRepository(), nil, nil, nil)
}

// NewServiceWithRepo wires a custom repo (e.g. the persistence-backed one)
// while leaving providers/notifications nil — call sites that need side
// effects should use NewServiceWithDeps instead.
func NewServiceWithRepo(repo Repository) *Service {
	return NewServiceWithDeps(repo, nil, nil, nil)
}

// NewServiceWithDeps is the full dependency-injecting constructor.
//   - providers maps each supported target kind to its resolver; may be nil.
//   - notificationWriter receives AppendMomentLiked/AppendCommentLiked/
//     AppendReactionAdded calls; nil disables notifications.
//   - actorProvider resolves nickname/avatar for the reacting user.
func NewServiceWithDeps(repo Repository, providers []TargetProvider, notificationWriter reactionNotificationWriter, actorProvider actor.Provider) *Service {
	byKind := make(map[ReactTarget]TargetProvider, len(providers))
	for _, p := range providers {
		if p != nil {
			byKind[p.ReactTarget()] = p
		}
	}
	return &Service{repo: repo, providers: byKind, notifications: notificationWriter, users: actorProvider}
}

// Set applies a reaction (active=true adds/switches the emoji; active=false
// removes). Returns nil on success; ErrReactionTargetNotFound when the target
// resource does not exist.
func (s *Service) Set(ctx context.Context, req SetReactionRequest) error {
	if !ValidReactTarget(req.TargetType) {
		return ErrReactionTargetNotFound
	}
	if !req.Active {
		return s.unreact(ctx, req)
	}
	return s.react(ctx, req)
}

func (s *Service) react(ctx context.Context, req SetReactionRequest) error {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	owner, targetExists, err := s.resolveTarget(ctx, req.TargetType, req.TargetID)
	if err != nil {
		observability.L().Error("reactions.react resolve target failed", "error", err, "target_type", req.TargetType, "target_id", req.TargetID)
		return err
	}
	if !targetExists {
		return ErrReactionTargetNotFound
	}
	changed, err := s.repo.React(ctx, req.TargetType, req.TargetID, principal.UserID, req.Emoji)
	if err != nil {
		observability.L().Error("reactions.react repo failed", "error", err, "target_type", req.TargetType, "target_id", req.TargetID)
		return err
	}
	if !changed {
		// Emoji unchanged: no-op per spec idempotency.
		return nil
	}
	s.notifyOwner(ctx, req, principal.UserID, owner, true)
	return nil
}

func (s *Service) unreact(ctx context.Context, req SetReactionRequest) error {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	// We do not require the target to exist for unreact — removing a reaction
	// on a deleted target is a legitimate no-op and spec marks it idempotent.
	changed, err := s.repo.Unreact(ctx, req.TargetType, req.TargetID, principal.UserID)
	if err != nil {
		observability.L().Error("reactions.unreact repo failed", "error", err, "target_type", req.TargetType, "target_id", req.TargetID)
		return err
	}
	if !changed {
		return nil
	}
	// For removal we still broadcast ReactionRemoved but skip owner
	// notification (removal isn't worth notifying about).
	realtime.DefaultHub().BroadcastToUser(principal.UserID, realtime.Event{
		Type: EventTypeReactionRemoved,
		Payload: map[string]any{
			"target_type": string(req.TargetType),
			"target_id":   req.TargetID,
			"emoji":       req.Emoji,
			"actor":       s.buildActor(ctx, principal.UserID),
		},
	})
	return nil
}

// List returns the paginated reaction rows for one target.
func (s *Service) List(ctx context.Context, req ListReactionsRequest) (PaginatedReactions, error) {
	if !ValidReactTarget(req.TargetType) {
		return PaginatedReactions{}, ErrReactionTargetNotFound
	}
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
	items, nextCursor, err := s.repo.ListReactions(ctx, req.TargetType, req.TargetID, req.Emoji, limit, cursor)
	if err != nil {
		return PaginatedReactions{}, err
	}
	return PaginatedReactions{Items: items, NextCursor: nextCursor}, nil
}

// Aggregate returns the per-emoji Reaction[] for one target (backs
// MomentItem.reactions / CommentItem.reactions).
func (s *Service) Aggregate(ctx context.Context, targetType ReactTarget, targetID string) ([]Reaction, error) {
	viewer := viewerUserID(ctx)
	return s.repo.Aggregate(ctx, targetType, targetID, viewer)
}

// AggregateBatch is the multi-target form for list/feed scenarios.
func (s *Service) AggregateBatch(ctx context.Context, targetType ReactTarget, targetIDs []string) (map[string][]Reaction, error) {
	viewer := viewerUserID(ctx)
	return s.repo.AggregateBatch(ctx, targetType, targetIDs, viewer)
}

// viewerUserID returns the authenticated user id, or "" for anonymous callers.
func viewerUserID(ctx context.Context) string {
	principal, ok := platformauth.PrincipalFromContext(ctx)
	if !ok {
		return ""
	}
	return principal.UserID
}

// ListReactedTargets returns the target_ids a user reacted to, for one kind.
// Backs users/list_reacted_moments (target_type='moment').
func (s *Service) ListReactedTargets(ctx context.Context, targetType ReactTarget, limit int, cursor string) ([]string, string, error) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	return s.repo.ListReactedTargets(ctx, targetType, principal.UserID, limit, cursor)
}

// resolveTarget looks up the provider for the given kind and asks it for the
// target context (existence + owner). Returns found=false (no error) when the
// provider has no entry for the target.
func (s *Service) resolveTarget(ctx context.Context, kind ReactTarget, targetID string) (TargetContext, bool, error) {
	if s.providers == nil {
		// No providers wired: assume target exists but owner is unknown so we
		// still write the reaction without notifications. This path is used
		// only in tests / degraded single-instance mode.
		return TargetContext{}, true, nil
	}
	provider, ok := s.providers[kind]
	if !ok {
		return TargetContext{}, true, nil
	}
	return provider.Resolve(ctx, targetID)
}

// notifyOwner dispatches the per-target-kind notification + realtime event
// after a reaction is added. Per spec api_spec_types.md NotifyType, the
// notification kind follows the target: moment→moment_liked,
// comment→comment_liked, message→reaction_added. We never notify the author
// about their own reaction (self short-circuit).
func (s *Service) notifyOwner(ctx context.Context, req SetReactionRequest, actorUserID string, owner TargetContext, added bool) {
	actor := s.buildActor(ctx, actorUserID)
	reactedAt := time.Now().UTC().Format(time.RFC3339)
	ownerID := strings.TrimSpace(owner.OwnerUserID)
	// Realtime broadcast goes to the target owner only (their client updates the
	// reactions badge / list); subscribers to the resource read the aggregate
	// via their own list calls.
	if ownerID != "" && ownerID != actorUserID {
		eventPayload := map[string]any{
			"target_type": string(req.TargetType),
			"target_id":   req.TargetID,
			"emoji":       req.Emoji,
			"actor":       actor,
			"reacted_at":  reactedAt,
		}
		// Merge target-specific summary fields if present (e.g. moment_summary).
		for k, v := range owner.Summary {
			eventPayload[k] = v
		}
		realtime.DefaultHub().BroadcastToUser(ownerID, realtime.Event{
			Type:    EventTypeReactionAdded,
			Payload: eventPayload,
		})
	}
	if s.notifications == nil || ownerID == "" || ownerID == actorUserID {
		return
	}
	switch req.TargetType {
	case ReactTargetMoment:
		momentID := req.TargetID
		momentSummary := owner.Summary
		if momentSummary == nil {
			momentSummary = map[string]any{"moment_id": momentID}
		}
		if err := s.notifications.AppendMomentLiked(ctx, ownerID, actor, momentID, momentSummary); err != nil {
			observability.L().Error("reactions.notify AppendMomentLiked failed", "error", err, "moment_id", momentID)
		}
	case ReactTargetComment:
		commentID := req.TargetID
		momentID, _ := owner.Summary["moment_id"].(string)
		preview, _ := owner.Summary["comment_preview"].(string)
		if err := s.notifications.AppendCommentLiked(ctx, ownerID, actor, commentID, preview, momentID, owner.Summary); err != nil {
			observability.L().Error("reactions.notify AppendCommentLiked failed", "error", err, "comment_id", commentID)
		}
	default:
		// message (and any future kind): generic reaction_added notification.
		if err := s.notifications.AppendReactionAdded(ctx, ownerID, actor, string(req.TargetType), req.TargetID, req.Emoji); err != nil {
			observability.L().Error("reactions.notify AppendReactionAdded failed", "error", err, "target_type", req.TargetType, "target_id", req.TargetID)
		}
	}
}

func (s *Service) buildActor(ctx context.Context, userID string) map[string]any {
	if s.users != nil {
		info, err := s.users.GetActorInfo(ctx, userID)
		if err == nil {
			a := map[string]any{"user_id": info.UserID, "nickname": info.Nickname}
			if info.AvatarURL != "" {
				a["avatar_url"] = info.AvatarURL
			}
			return a
		}
		observability.L().Error("reactions.build actor failed", "error", err, "user_id", userID)
	}
	return map[string]any{"user_id": userID, "nickname": userID}
}
