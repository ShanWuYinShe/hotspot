package reactions

import "context"

// Repository is the persistence surface for reactions.
type Repository interface {
	// React upserts the caller's emoji for a target. Returns changed=true when
	// a row was inserted or its emoji was updated (caller-side semantics: a
	// no-op toggle yields changed=false so notifications aren't re-sent).
	React(ctx context.Context, targetType ReactTarget, targetID, userID, emoji string) (changed bool, err error)
	// Unreact removes the caller's reaction row. Removing a non-existent row is
	// a no-op (returns changed=false, no error).
	Unreact(ctx context.Context, targetType ReactTarget, targetID, userID string) (changed bool, err error)
	// ListReactions returns the reaction rows for one target, newest first,
	// optionally filtered by emoji.
	ListReactions(ctx context.Context, targetType ReactTarget, targetID, emoji string, limit int, cursor string) ([]ReactionItem, string, error)
	// Aggregate returns per-emoji counts for one target, with whether the
	// viewer reacted. Backs MomentItem/CommentItem.reactions[].
	Aggregate(ctx context.Context, targetType ReactTarget, targetID, viewerID string) ([]Reaction, error)
	// AggregateBatch is the multi-target form of Aggregate for list/feed
	// scenarios (avoids N+1). Returns target_id -> reactions.
	AggregateBatch(ctx context.Context, targetType ReactTarget, targetIDs []string, viewerID string) (map[string][]Reaction, error)
	// ListReactedTargets lists target_ids the user reacted to, for a given
	// target kind (backs users/fetch_reacted_moments). Cursor-paginated by
	// reacted_at DESC, target_id DESC tie-break.
	ListReactedTargets(ctx context.Context, targetType ReactTarget, userID string, limit int, cursor string) ([]string, string, error)
}
