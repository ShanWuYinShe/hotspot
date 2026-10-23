package reactions

import "context"

// ReactionOutcome 区分一次 React 的实际效果：
//   - ReactionInserted：新增 reaction 行（目标计数 +1、发通知）；
//   - ReactionSwitched：已有行切换 emoji（计数不变——总数口径的 likes_count
//     不受切换影响；仍发通知，owner 感知新 emoji）；
//   - ReactionUnchanged：同 emoji 幂等重放（无副作用）。
type ReactionOutcome int

const (
	ReactionUnchanged ReactionOutcome = iota
	ReactionInserted
	ReactionSwitched
)

// Repository is the persistence surface for reactions.
type Repository interface {
	// React upserts the caller's emoji for a target and报告本次实际效果
	//（inserted/switched/unchanged）。此前仅返回 changed bool：PG 实现的 CTE
	// 缺 UPDATE 分支，切换 emoji 不落库却报 changed=true，导致计数列虚增、
	// 通知 emoji 与库内实态不符——三态返回值让 Service 按效果决定副作用。
	React(ctx context.Context, targetType ReactTarget, targetID, userID, emoji string) (ReactionOutcome, error)
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
