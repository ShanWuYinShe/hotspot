package reactions

import (
	"context"
	"strings"
	"time"

	"hotspot/backend-go/internal/platform/actor"
	"hotspot/backend-go/internal/platform/appconfig"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/observability"
	"hotspot/backend-go/internal/platform/realtime"
	"hotspot/backend-go/internal/platform/types"
)

// Service orchestrates reaction reads/writes and the side effects (owner
// notification, realtime broadcast) that follow a reaction being added.
type Service struct {
	repo            Repository
	providers       map[ReactTarget]TargetProvider
	notifications   reactionNotificationWriter
	users           actor.Provider
	appConfig       appconfig.Source // optional; nil => emoji whitelist check skipped
	reactionCounter ReactionCounter  // optional; nil => 不维护目标计数冗余列
}

// ReactionCounter 是 reactions 成功变更后维护目标侧计数冗余列的最小接口
// （bootstrap 把 moments.Service 注入进来，消除 moments/search popular 排序
// 的每行 reactions 聚合子查询）。delta 可负；实现自行钳制下限。
type ReactionCounter interface {
	AdjustReactionCount(ctx context.Context, momentID string, delta int)
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
//   - notificationWriter receives AppendReactionAdded calls; nil disables
//     notifications.
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

// SetReactionCounter wires the target-side reaction count maintainer (optional;
// nil keeps read-side aggregation as before). Failures inside the counter are
// logged by the implementation, not surfaced here.
func (s *Service) SetReactionCounter(c ReactionCounter) { s.reactionCounter = c }

// SetAppConfig wires the app config source whose Limits.ReactionEmojis is the
// server-side emoji whitelist (spec api_spec_reaction.md: emoji 不在白名单返回
// httpx.CodeInvalidRequest). Optional: nil skips the check (tests / degraded mode).
func (s *Service) SetAppConfig(source appconfig.Source) { s.appConfig = source }

// emojiAllowed reports whether the emoji is on the AppConfig whitelist.
// Without a wired source every emoji is accepted (degraded mode).
func (s *Service) emojiAllowed(ctx context.Context, emoji string) bool {
	if s.appConfig == nil {
		return true
	}
	for _, allowed := range s.appConfig.Get(ctx).Limits.ReactionEmojis {
		if allowed == emoji {
			return true
		}
	}
	return false
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
	if !s.emojiAllowed(ctx, req.Emoji) {
		// Spec api_spec_reaction.md: emoji 不在白名单返回 httpx.CodeInvalidRequest。
		return ErrInvalidEmoji
	}
	owner, targetExists, err := s.resolveTarget(ctx, req.TargetType, req.TargetID)
	if err != nil {
		observability.L().Error("reactions.react resolve target failed", "error", err, "target_type", req.TargetType, "target_id", req.TargetID)
		return err
	}
	if !targetExists {
		return ErrReactionTargetNotFound
	}
	outcome, err := s.repo.React(ctx, req.TargetType, req.TargetID, principal.UserID, req.Emoji)
	if err != nil {
		observability.L().Error("reactions.react repo failed", "error", err, "target_type", req.TargetType, "target_id", req.TargetID)
		return err
	}
	switch outcome {
	case ReactionUnchanged:
		// Emoji unchanged: no-op per spec idempotency.
		return nil
	case ReactionInserted:
		// 新增行：总数口径的计数列 +1。切换 emoji 不改变总数，不调计数
		//（此前 changed bool 无法区分二者，切换也 +1，计数列可被无限虚增）。
		s.adjustReactionCount(ctx, req, +1)
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
	s.adjustReactionCount(ctx, req, -1)
	// ReactionRemoved goes to the target resource's author (spec
	// api_spec_websocket.md). Resolve the owner for the broadcast; a deleted
	// or unresolvable target has no author to push to, and unreact itself is
	// an idempotent no-op, so we skip the broadcast in that case.
	owner, targetExists, err := s.resolveTarget(ctx, req.TargetType, req.TargetID)
	if err != nil || !targetExists {
		return nil
	}
	ownerID := strings.TrimSpace(owner.OwnerUserID)
	if ownerID == "" || ownerID == principal.UserID {
		// No owner (message targets / degraded mode) or self-removal: nothing
		// to broadcast (mirrors notifyOwner's self-reaction skip).
		return nil
	}
	// Per spec ReactionRemovedPayload: no notification_id (removal does not
	// produce a notification).
	realtime.DefaultHub().BroadcastToUser(ownerID, realtime.Event{
		Type: EventTypeReactionRemoved,
		Payload: ReactionRemovedPayload{
			TargetType: string(req.TargetType),
			TargetID:   req.TargetID,
			Emoji:      req.Emoji,
			Actor:      s.buildActor(ctx, principal.UserID),
			RemovedAt:  time.Now().UTC().Format(time.RFC3339),
		},
	})
	return nil
}

// adjustReactionCount 维护目标侧计数冗余列（当前仅 moment 目标有消费方）。
// 失败由实现记 Warn：列与 reactions 表的最终一致由启动期差异回填自愈，
// 不阻断已成功的反应主流程。
func (s *Service) adjustReactionCount(ctx context.Context, req SetReactionRequest, delta int) {
	if s.reactionCounter == nil || req.TargetType != ReactTargetMoment {
		return
	}
	s.reactionCounter.AdjustReactionCount(ctx, req.TargetID, delta)
}

// List returns the paginated reaction rows for one target.
func (s *Service) List(ctx context.Context, req ListReactionsRequest) (PaginatedReactions, error) {
	if !ValidReactTarget(req.TargetType) {
		return PaginatedReactions{}, ErrReactionTargetNotFound
	}
	// 目标可见性（spec reaction.md → reactions/fetch）：存在但调用方不可见按不存在处理
	if _, targetExists, err := s.resolveTarget(ctx, req.TargetType, req.TargetID); err != nil {
		return PaginatedReactions{}, err
	} else if !targetExists {
		return PaginatedReactions{}, ErrReactionTargetNotFound
	}
	limit := 20
	cursor := ""
	if req.Pagination != nil {
		if req.Pagination.Limit > 0 {
			limit = req.Pagination.Limit
		}
		if limit > types.MaxPageSize {
			limit = types.MaxPageSize
		}
		cursor = req.Pagination.Cursor
	}
	items, nextCursor, err := s.repo.ListReactions(ctx, req.TargetType, req.TargetID, req.Emoji, limit, cursor)
	if err != nil {
		return PaginatedReactions{}, err
	}
	var next *string
	if nextCursor != "" {
		next = &nextCursor
	}
	return PaginatedReactions{Items: items, NextCursor: next}, nil
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
// Backs users/fetch_reacted_moments (target_type='moment').
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

// notifyOwner dispatches the realtime ReactionAdded event + reaction_added
// notification after a reaction is added. Per spec api_spec_types.md NotifyType
// and api_spec_websocket.md ReactionAddedPayload, ALL target kinds
// (moment/comment/message) use the unified reaction_added notification (the
// legacy moment_liked/comment_liked types were removed from the contract), and
// the realtime payload carries notification_id when the reactor is not the
// target author (self-reactions are not notified).
func (s *Service) notifyOwner(ctx context.Context, req SetReactionRequest, actorUserID string, owner TargetContext, added bool) {
	actor := s.buildActor(ctx, actorUserID)
	reactedAt := time.Now().UTC().Format(time.RFC3339)
	ownerID := strings.TrimSpace(owner.OwnerUserID)
	// Realtime broadcast goes to the target owner only (their client updates the
	// reactions badge / list); subscribers to the resource read the aggregate
	// via their own list calls. Self-reactions skip both notification and event.
	if ownerID == "" || ownerID == actorUserID {
		return
	}
	// Persist the notification first so the event payload can carry the
	// notification_id the client needs to deep-link.
	notificationID := ""
	if s.notifications != nil {
		id, err := s.notifications.AppendReactionAdded(ctx, ownerID, actor, string(req.TargetType), req.TargetID, req.Emoji)
		if err != nil {
			observability.L().Error("reactions.notify AppendReactionAdded failed", "error", err, "target_type", req.TargetType, "target_id", req.TargetID)
		} else {
			notificationID = id
		}
	}
	eventPayload := map[string]any{
		"target_type": string(req.TargetType),
		"target_id":   req.TargetID,
		"emoji":       req.Emoji,
		"actor":       actor,
		"reacted_at":  reactedAt,
	}
	if notificationID != "" {
		eventPayload["notify_id"] = notificationID
	}
	realtime.DefaultHub().BroadcastToUser(ownerID, realtime.Event{
		Type:    EventTypeReactionAdded,
		Payload: eventPayload,
	})
}

func (s *Service) buildActor(ctx context.Context, userID string) types.UserInfo {
	if s.users != nil {
		info, err := s.users.GetActorInfo(ctx, userID)
		if err == nil {
			return types.UserInfo{UserID: info.UserID, Nickname: info.Nickname, AvatarURL: types.StrPtr(info.AvatarURL)}
		}
		observability.L().Error("reactions.build actor failed", "error", err, "user_id", userID)
	}
	return types.UserInfo{UserID: userID, Nickname: userID}
}
