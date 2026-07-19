package comments

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"time"

	"hotspot/backend-go/internal/platform/idgen"
	"hotspot/backend-go/internal/platform/persistence"

	"github.com/jackc/pgx/v5"
)

type PostgresRepository struct {
	factory *persistence.Factory
}

func NewPostgresRepository(factory *persistence.Factory) *PostgresRepository {
	return &PostgresRepository{factory: factory}
}

func (r *PostgresRepository) Create(ctx context.Context, userID string, req CreateCommentRequest) (CommentItem, error) {
	queryable, ok := queryable(ctx, r.factory)
	if !ok {
		return CommentItem{}, persistence.ErrNotImplemented
	}
	now := time.Now().UTC()
	createdAt := now.Format(time.RFC3339)
	commentID, err := idgen.WithPrefix("comment_")
	if err != nil {
		return CommentItem{}, err
	}
	var parentID any
	if req.ParentID != "" {
		parentID = req.ParentID
	}
	if err := queryable.Exec(ctx, queryInsertComment, commentID, req.MomentID, userID, parentID, req.Content, createdAt, 0, 0); err != nil {
		return CommentItem{}, err
	}
	var nickname string
	_ = queryable.QueryRow(ctx, queryGetNickname, userID).Scan(&nickname)
	if nickname == "" {
		nickname = userID
	}
	return CommentItem{
		CommentID:    commentID,
		MomentID:     req.MomentID,
		User:         UserInfo{UserID: userID, Nickname: nickname},
		Content:      req.Content,
		CreatedAt:    createdAt,
		LikesCount:   0,
		RepliesCount: 0,
		MyState:      &MyCommentState{IsLiked: false},
	}, nil
}

func (r *PostgresRepository) SoftDelete(ctx context.Context, commentID string) error {
	queryable, ok := queryable(ctx, r.factory)
	if !ok {
		return persistence.ErrNotImplemented
	}
	return queryable.Exec(ctx, querySoftDeleteComment, commentID)
}

func (r *PostgresRepository) List(ctx context.Context, momentID string, limit int, cursor string) (PaginatedComments, error) {
	queryable, ok := queryable(ctx, r.factory)
	if !ok {
		return PaginatedComments{}, persistence.ErrNotImplemented
	}
	limit = normalizeCommentLimit(limit)
	cursorCreatedAt, cursorCommentID, err := decodeCommentCursor(cursor)
	if err != nil {
		return PaginatedComments{}, err
	}
	rows, err := queryable.Query(ctx, queryListComments, momentID, cursorCreatedAt, cursorCommentID, limit+1)
	if err != nil {
		return PaginatedComments{}, err
	}
	defer rows.Close()
	items := make([]CommentItem, 0, limit+1)
	for rows.Next() {
		var item CommentItem
		var userID, nickname, parentID string
		var createdAt time.Time
		if err := rows.Scan(&item.CommentID, &userID, &nickname, &parentID, &item.Content, &createdAt, &item.LikesCount, &item.RepliesCount, &item.IsDeleted); err != nil {
			return PaginatedComments{}, err
		}
		item.User = UserInfo{UserID: userID, Nickname: nickname}
		item.ParentID = parentID
		item.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		item.MyState = &MyCommentState{IsLiked: false}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return PaginatedComments{}, err
	}
	if len(items) > limit {
		return PaginatedComments{Items: items[:limit], NextCursor: encodeCommentCursor(items[limit-1].CreatedAt, items[limit-1].CommentID)}, nil
	}
	return PaginatedComments{Items: items}, nil
}

func (r *PostgresRepository) Get(ctx context.Context, commentID string) (CommentItem, bool, error) {
	queryable, ok := queryable(ctx, r.factory)
	if !ok {
		return CommentItem{}, false, persistence.ErrNotImplemented
	}
	row := queryable.QueryRow(ctx, queryGetComment, commentID)
	var item CommentItem
	var userID, nickname, parentID string
	var createdAt time.Time
	if err := row.Scan(&item.CommentID, &item.MomentID, &userID, &nickname, &parentID, &item.Content, &createdAt, &item.LikesCount, &item.RepliesCount, &item.IsDeleted); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return CommentItem{}, false, nil
		}
		return CommentItem{}, false, err
	}
	item.User = UserInfo{UserID: userID, Nickname: nickname}
	item.ParentID = parentID
	item.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	item.MyState = &MyCommentState{IsLiked: false}
	return item, true, nil
}

func (r *PostgresRepository) ListReplies(ctx context.Context, commentID string, limit int, cursor string) (PaginatedComments, error) {
	queryable, ok := queryable(ctx, r.factory)
	if !ok {
		return PaginatedComments{}, persistence.ErrNotImplemented
	}
	limit = normalizeCommentLimit(limit)
	cursorCreatedAt, cursorCommentID, err := decodeCommentCursor(cursor)
	if err != nil {
		return PaginatedComments{}, err
	}
	rows, err := queryable.Query(ctx, queryListCommentReplies, commentID, cursorCreatedAt, cursorCommentID, limit+1)
	if err != nil {
		return PaginatedComments{}, err
	}
	defer rows.Close()
	items := make([]CommentItem, 0, limit+1)
	for rows.Next() {
		var item CommentItem
		var userID, nickname, parentID string
		var createdAt time.Time
		if err := rows.Scan(&item.CommentID, &userID, &nickname, &parentID, &item.Content, &createdAt, &item.LikesCount, &item.RepliesCount, &item.IsDeleted); err != nil {
			return PaginatedComments{}, err
		}
		item.User = UserInfo{UserID: userID, Nickname: nickname}
		item.ParentID = parentID
		item.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		item.MyState = &MyCommentState{IsLiked: false}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return PaginatedComments{}, err
	}
	if len(items) > limit {
		return PaginatedComments{Items: items[:limit], NextCursor: encodeCommentCursor(items[limit-1].CreatedAt, items[limit-1].CommentID)}, nil
	}
	return PaginatedComments{Items: items}, nil
}

// ListTopRepliesByParentIDs fetches up to `limit` newest replies for each of the
// given parent comment IDs in a single query, returning a parentID->replies
// map. It replaces the N+1 of calling ListReplies once per top-level comment.
func (r *PostgresRepository) ListTopRepliesByParentIDs(ctx context.Context, parentIDs []string, limit int) (map[string][]CommentItem, error) {
	result := make(map[string][]CommentItem, len(parentIDs))
	if len(parentIDs) == 0 || limit <= 0 {
		return result, nil
	}
	queryable, ok := queryable(ctx, r.factory)
	if !ok {
		return result, persistence.ErrNotImplemented
	}
	rows, err := queryable.Query(ctx, queryListTopRepliesByParentIDs, parentIDs, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var item CommentItem
		var userID, nickname, parentID string
		var createdAt time.Time
		if err := rows.Scan(&item.CommentID, &userID, &nickname, &parentID, &item.Content, &createdAt, &item.LikesCount, &item.RepliesCount, &item.IsDeleted); err != nil {
			return nil, err
		}
		item.User = UserInfo{UserID: userID, Nickname: nickname}
		item.ParentID = parentID
		item.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		item.MyState = &MyCommentState{IsLiked: false}
		result[parentID] = append(result[parentID], item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *PostgresRepository) Like(ctx context.Context, commentID string, userID string) (bool, error) {
	queryable, ok := queryable(ctx, r.factory)
	if !ok {
		return false, persistence.ErrNotImplemented
	}
	now := time.Now().UTC()
	var inserted bool
	if err := queryable.QueryRow(ctx, queryInsertCommentLikeAndCount, commentID, userID, now).Scan(&inserted); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return inserted, nil
}

func (r *PostgresRepository) Unlike(ctx context.Context, commentID string, userID string) error {
	queryable, ok := queryable(ctx, r.factory)
	if !ok {
		return persistence.ErrNotImplemented
	}
	var deleted bool
	if err := queryable.QueryRow(ctx, queryDeleteCommentLikeAndCount, commentID, userID).Scan(&deleted); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return err
	}
	return nil
}

func (r *PostgresRepository) ListLikes(ctx context.Context, commentID string, limit int, cursor string) ([]CommentLikeItem, string, error) {
	queryable, ok := queryable(ctx, r.factory)
	if !ok {
		return nil, "", persistence.ErrNotImplemented
	}
	limit = normalizeCommentLimit(limit)
	cursorLikedAt, cursorUserID, err := decodeCommentCursor(cursor)
	if err != nil {
		return nil, "", err
	}
	rows, err := queryable.Query(ctx, queryListCommentLikes, commentID, cursorLikedAt, cursorUserID, limit+1)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	items := make([]CommentLikeItem, 0, limit+1)
	for rows.Next() {
		var item CommentLikeItem
		var likedAt time.Time
		if err := rows.Scan(&item.User.UserID, &item.User.Nickname, &likedAt); err != nil {
			return nil, "", err
		}
		item.LikedAt = likedAt.UTC().Format(time.RFC3339)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}
	if len(items) > limit {
		return items[:limit], encodeCommentCursor(items[limit-1].LikedAt, items[limit-1].User.UserID), nil
	}
	return items, "", nil
}

type commentCursor struct {
	At string `json:"at"`
	ID string `json:"id"`
}

// queryable resolves the Queryable to use for this request: an in-flight
// transaction bound to ctx when present, otherwise the shared pool. Returns
// ok=false when persistence is not configured (in-memory mode), in which case
// callers surface ErrNotImplemented.
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

func normalizeCommentLimit(limit int) int {
	switch {
	case limit <= 0:
		return 20
	case limit > 100:
		return 100
	default:
		return limit
	}
}

func encodeCommentCursor(at, id string) string {
	blob, err := json.Marshal(commentCursor{At: at, ID: id})
	if err != nil {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(blob)
}

func decodeCommentCursor(cursor string) (any, string, error) {
	if cursor == "" {
		return nil, "", nil
	}
	blob, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return nil, "", ErrInvalidCursor
	}
	var decoded commentCursor
	if err := json.Unmarshal(blob, &decoded); err != nil || decoded.At == "" || decoded.ID == "" {
		return nil, "", ErrInvalidCursor
	}
	at, err := time.Parse(time.RFC3339, decoded.At)
	if err != nil {
		return nil, "", ErrInvalidCursor
	}
	return at.UTC(), decoded.ID, nil
}
