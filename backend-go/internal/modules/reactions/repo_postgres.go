package reactions

import (
	"context"
	"errors"
	"fmt"
	"time"

	"hotspot/backend-go/internal/platform/cursor"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/types"
)

// PostgresRepository implements Repository over the configured postgres pool.
type PostgresRepository struct {
	factory *persistence.Factory
}

func NewPostgresRepository(factory *persistence.Factory) *PostgresRepository {
	return &PostgresRepository{factory: factory}
}

func (r *PostgresRepository) React(ctx context.Context, targetType ReactTarget, targetID, userID, emoji string) (bool, error) {
	q, ok := queryable(ctx, r.factory)
	if !ok {
		return false, persistence.ErrNotImplemented
	}
	now := time.Now().UTC()
	var changed bool
	if err := q.QueryRow(ctx, queryUpsertReaction, string(targetType), targetID, userID, emoji, now).Scan(&changed); err != nil {
		return false, fmt.Errorf("reactions.React: %w", err)
	}
	return changed, nil
}

func (r *PostgresRepository) Unreact(ctx context.Context, targetType ReactTarget, targetID, userID string) (bool, error) {
	q, ok := queryable(ctx, r.factory)
	if !ok {
		return false, persistence.ErrNotImplemented
	}
	// Use a CTE that RETURNINGs a row so we learn whether anything was
	// actually removed (pgx's Exec here does not report rows-affected).
	var deleted bool
	err := q.QueryRow(ctx, queryDeleteReactionWithFlag, string(targetType), targetID, userID).Scan(&deleted)
	if err != nil {
		return false, fmt.Errorf("reactions.Unreact: %w", err)
	}
	return deleted, nil
}

func (r *PostgresRepository) ListReactions(ctx context.Context, targetType ReactTarget, targetID, emoji string, limit int, cursor string) ([]ReactionItem, string, error) {
	q, ok := queryable(ctx, r.factory)
	if !ok {
		return nil, "", persistence.ErrNotImplemented
	}
	limit = normalizeReactionLimit(limit)
	cursorAt, cursorUser, err := decodeListCursor(cursor)
	if err != nil {
		return nil, "", err
	}
	emojiFilter := ""
	if emoji != "" {
		emojiFilter = emoji
	}
	rows, err := q.Query(ctx, queryListReactions, string(targetType), targetID, nullableString(emojiFilter), cursorAt, cursorUser, limit+1)
	if err != nil {
		return nil, "", fmt.Errorf("reactions.ListReactions: %w", err)
	}
	defer rows.Close()
	items := make([]ReactionItem, 0, limit+1)
	for rows.Next() {
		var item ReactionItem
		var reactedAt time.Time
		if err := rows.Scan(&item.User.UserID, &item.User.Nickname, &item.Emoji, &reactedAt); err != nil {
			return nil, "", fmt.Errorf("reactions.ListReactions scan: %w", err)
		}
		// 纳秒精度：秒级截断会让同秒内更晚的行在下一页被谓词排除（分页丢行）。
		item.ReactedAt = reactedAt.UTC().Format(time.RFC3339Nano)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("reactions.ListReactions rows: %w", err)
	}
	if len(items) > limit {
		last := items[limit-1]
		return items[:limit], encodeListCursor(last.ReactedAt, last.User.UserID), nil
	}
	return items, "", nil
}

func (r *PostgresRepository) Aggregate(ctx context.Context, targetType ReactTarget, targetID, viewerID string) ([]Reaction, error) {
	q, ok := queryable(ctx, r.factory)
	if !ok {
		return nil, persistence.ErrNotImplemented
	}
	rows, err := q.Query(ctx, queryAggregateReactions, string(targetType), targetID, viewerID)
	if err != nil {
		return nil, fmt.Errorf("reactions.Aggregate: %w", err)
	}
	defer rows.Close()
	out := make([]Reaction, 0, 4)
	for rows.Next() {
		var rec Reaction
		if err := rows.Scan(&rec.Emoji, &rec.Count, &rec.ReactedByMe); err != nil {
			return nil, fmt.Errorf("reactions.Aggregate scan: %w", err)
		}
		out = append(out, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reactions.Aggregate rows: %w", err)
	}
	return out, nil
}

func (r *PostgresRepository) AggregateBatch(ctx context.Context, targetType ReactTarget, targetIDs []string, viewerID string) (map[string][]Reaction, error) {
	q, ok := queryable(ctx, r.factory)
	if !ok {
		return nil, persistence.ErrNotImplemented
	}
	result := make(map[string][]Reaction, len(targetIDs))
	if len(targetIDs) == 0 {
		return result, nil
	}
	rows, err := q.Query(ctx, queryAggregateReactionsBatch, string(targetType), targetIDs, viewerID)
	if err != nil {
		return nil, fmt.Errorf("reactions.AggregateBatch: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var (
			targetID    string
			rec         Reaction
			reactedByMe bool
		)
		if err := rows.Scan(&targetID, &rec.Emoji, &rec.Count, &reactedByMe); err != nil {
			return nil, fmt.Errorf("reactions.AggregateBatch scan: %w", err)
		}
		rec.ReactedByMe = reactedByMe
		result[targetID] = append(result[targetID], rec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reactions.AggregateBatch rows: %w", err)
	}
	return result, nil
}

func (r *PostgresRepository) ListReactedTargets(ctx context.Context, targetType ReactTarget, userID string, limit int, cursor string) ([]string, string, error) {
	q, ok := queryable(ctx, r.factory)
	if !ok {
		return nil, "", persistence.ErrNotImplemented
	}
	limit = normalizeReactionLimit(limit)
	cursorAt, cursorID, err := decodeTargetCursor(cursor)
	if err != nil {
		return nil, "", err
	}
	rows, err := q.Query(ctx, queryListReactedTargets, string(targetType), userID, cursorAt, cursorID, limit+1)
	if err != nil {
		return nil, "", fmt.Errorf("reactions.ListReactedTargets: %w", err)
	}
	defer rows.Close()
	ids := make([]string, 0, limit+1)
	var lastAt time.Time
	var lastID string
	for rows.Next() {
		var id string
		var at time.Time
		if err := rows.Scan(&id, &at); err != nil {
			return nil, "", fmt.Errorf("reactions.ListReactedTargets scan: %w", err)
		}
		ids = append(ids, id)
		lastAt, lastID = at, id
	}
	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("reactions.ListReactedTargets rows: %w", err)
	}
	if len(ids) > limit {
		return ids[:limit], encodeTargetCursor(lastAt, lastID), nil
	}
	return ids, "", nil
}

// queryable resolves the Queryable to use for this request: an in-flight
// transaction bound to ctx when present, otherwise the shared pool. Returns
// ok=false when persistence is not configured (in-memory mode).
func queryable(ctx context.Context, factory *persistence.Factory) (persistence.Queryable, bool) {
	if q, ok := persistence.QueryableFromContext(ctx); ok {
		return q, true
	}
	if factory == nil || factory.DB == nil || factory.Config.PostgresDSN == "" {
		return nil, false
	}
	q, ok := factory.DB.(persistence.QueryableDB)
	return q, ok
}

func normalizeReactionLimit(limit int) int {
	switch {
	case limit <= 0:
		return 20
	case limit > types.MaxPageSize:
		return types.MaxPageSize
	default:
		return limit
	}
}

// nullableString returns nil for an empty string so it binds to a nullable SQL
// parameter (used for optional filters via $n::text IS NULL semantics).
func nullableString(s string) any {
	if s == "" {
		return nil
	}
	return s
}

type listCursor struct {
	At string `json:"at"`
	ID string `json:"id"`
}

// reactionsListCursorNamespace / reactionsTargetCursorNamespace 分别是
// 点赞列表与点赞目标列表游标的加密命名空间。
const (
	reactionsListCursorNamespace   = "reactions.list"
	reactionsTargetCursorNamespace = "reactions.targets"
)

func encodeListCursor(at, id string) string {
	value, err := cursor.Encode(reactionsListCursorNamespace, listCursor{At: at, ID: id})
	if err != nil {
		return ""
	}
	return value
}

func decodeListCursor(cursorValue string) (any, string, error) {
	if cursorValue == "" {
		return nil, "", nil
	}
	var c listCursor
	if err := cursor.Decode(reactionsListCursorNamespace, cursorValue, &c); err != nil || c.At == "" || c.ID == "" {
		return nil, "", ErrInvalidCursor
	}
	at, err := time.Parse(time.RFC3339Nano, c.At)
	if err != nil {
		return nil, "", ErrInvalidCursor
	}
	return at.UTC(), c.ID, nil
}

func encodeTargetCursor(at time.Time, id string) string {
	value, err := cursor.Encode(reactionsTargetCursorNamespace, listCursor{At: at.UTC().Format(time.RFC3339Nano), ID: id})
	if err != nil {
		return ""
	}
	return value
}

func decodeTargetCursor(cursorValue string) (any, string, error) {
	if cursorValue == "" {
		return nil, "", nil
	}
	var c listCursor
	if err := cursor.Decode(reactionsTargetCursorNamespace, cursorValue, &c); err != nil || c.At == "" || c.ID == "" {
		return nil, "", ErrInvalidCursor
	}
	at, err := time.Parse(time.RFC3339Nano, c.At)
	if err != nil {
		return nil, "", ErrInvalidCursor
	}
	return at.UTC(), c.ID, nil
}

// errRepoNotConfigured is returned by helpers when a code path expects a repo
// that wasn't wired (kept for symmetry with other modules).
var errRepoNotConfigured = errors.New("reactions: repository not configured")
