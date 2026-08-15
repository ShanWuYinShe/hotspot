package reactions

const (
	// queryUpsertReaction inserts a reaction row or, when the caller already has
	// one for this target, updates the emoji/reacted_at in place (spec set
	// semantics: switching emoji is an update, not an insert). We learn whether
	// the row materially changed via the returned changed flag.
	queryUpsertReaction = `
WITH existing AS (
  SELECT emoji FROM reactions
  WHERE target_type = $1 AND target_id = $2 AND user_id = $3
),
ins AS (
  INSERT INTO reactions (target_type, target_id, user_id, emoji, reacted_at)
  SELECT $1, $2, $3, $4, $5
  WHERE NOT EXISTS (SELECT 1 FROM existing)
  ON CONFLICT (target_type, target_id, user_id) DO NOTHING
  RETURNING emoji
)
SELECT
  -- changed=true when a brand-new row was inserted, or when the existing
  -- emoji differs from the one we are setting.
  (EXISTS (SELECT 1 FROM ins))
  OR (NOT EXISTS (SELECT 1 FROM ins) AND (SELECT emoji FROM existing) IS DISTINCT FROM $4)
  AS changed`

	// queryDeleteReactionWithFlag removes the caller's row and reports whether a
	// row was actually deleted. deleted=false means there was nothing to remove
	// (no-op), so the caller can skip side effects.
	queryDeleteReactionWithFlag = `
WITH del AS (
  DELETE FROM reactions
  WHERE target_type = $1 AND target_id = $2 AND user_id = $3
  RETURNING emoji
)
SELECT EXISTS (SELECT 1 FROM del) AS deleted`

	// queryListReactions lists reaction rows for one target. reacted_at DESC is
	// the primary order, user_id DESC the stable tie-breaker so the row-wise
	// (<) cursor predicate matches the ordering. An optional emoji filter is
	// applied via $3::text NULL.
	queryListReactions = `
SELECT r.user_id, COALESCE(u.nickname, r.user_id), r.emoji, r.reacted_at
FROM reactions r
LEFT JOIN users u ON u.user_id = r.user_id
WHERE r.target_type = $1
  AND r.target_id = $2
  AND ($3::text IS NULL OR r.emoji = $3::text)
  AND ($4::timestamptz IS NULL OR (r.reacted_at, r.user_id) < ($4::timestamptz, $5))
ORDER BY r.reacted_at DESC, r.user_id DESC
LIMIT $6`

	// queryAggregateReactions computes per-emoji counts + reacted_by_me for a
	// single target. BOOL_OR(user_id = $3) is false when the viewer is
	// anonymous (empty viewerID never matches) or hasn't reacted.
	queryAggregateReactions = `
SELECT r.emoji, COUNT(*) AS cnt, BOOL_OR(r.user_id = $3) AS reacted_by_me
FROM reactions r
WHERE r.target_type = $1 AND r.target_id = $2
GROUP BY r.emoji
ORDER BY cnt DESC, r.emoji ASC`

	// queryAggregateReactionsBatch is the multi-target form, used by feed/list
	// to avoid N+1. Returns one row per (target_id, emoji).
	queryAggregateReactionsBatch = `
SELECT r.target_id, r.emoji, COUNT(*) AS cnt, BOOL_OR(r.user_id = $3) AS reacted_by_me
FROM reactions r
WHERE r.target_type = $1 AND r.target_id = ANY($2::text[])
GROUP BY r.target_id, r.emoji`

	// queryListReactedTargets returns target_ids the user reacted to, newest
	// first. Backed by idx_reactions_user_reacted_at. Cursor is
	// (reacted_at, target_id).
	queryListReactedTargets = `
SELECT target_id, reacted_at
FROM reactions
WHERE target_type = $1 AND user_id = $2
  AND ($3::timestamptz IS NULL OR (reacted_at, target_id) < ($3::timestamptz, $4))
ORDER BY reacted_at DESC, target_id DESC
LIMIT $5`
)
