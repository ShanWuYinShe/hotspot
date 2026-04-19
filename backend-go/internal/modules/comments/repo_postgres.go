package comments

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"

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
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return CommentItem{}, persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return CommentItem{}, persistence.ErrNotImplemented
	}
	now := time.Now().UTC()
	createdAt := now.Format(time.RFC3339)
	commentID := fmt.Sprintf("comment_%d", now.UnixNano())
	var parentID any
	if req.ParentID != "" {
		parentID = req.ParentID
	}
	if err := queryable.Exec(ctx, queryInsertComment, commentID, req.MomentID, userID, parentID, req.Content, createdAt, 0, 0); err != nil {
		return CommentItem{}, err
	}
	return CommentItem{
		CommentID:    commentID,
		MomentID:     req.MomentID,
		User:         UserInfo{UserID: userID, Nickname: userID},
		Content:      req.Content,
		CreatedAt:    createdAt,
		LikesCount:   0,
		RepliesCount: 0,
		MyState:      &MyCommentState{IsLiked: false},
	}, nil
}

func (r *PostgresRepository) Delete(ctx context.Context, commentID string) error {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return persistence.ErrNotImplemented
	}
	return queryable.Exec(ctx, queryDeleteComment, commentID)
}

func (r *PostgresRepository) List(ctx context.Context, momentID string) (PaginatedComments, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return PaginatedComments{}, persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return PaginatedComments{}, persistence.ErrNotImplemented
	}
	rows, err := queryable.Query(ctx, queryListComments, momentID)
	if err != nil {
		return PaginatedComments{}, err
	}
	defer rows.Close()
	items := []CommentItem{}
	for rows.Next() {
		var item CommentItem
		var userID, parentID, createdAt string
		if err := rows.Scan(&item.CommentID, &userID, &parentID, &item.Content, &createdAt, &item.LikesCount, &item.RepliesCount); err != nil {
			return PaginatedComments{}, err
		}
		item.User = UserInfo{UserID: userID, Nickname: userID}
		item.ParentID = parentID
		item.CreatedAt = createdAt
		item.MyState = &MyCommentState{IsLiked: false}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return PaginatedComments{}, err
	}
	return PaginatedComments{Items: items}, nil
}

func (r *PostgresRepository) Get(ctx context.Context, commentID string) (CommentItem, bool, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return CommentItem{}, false, persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return CommentItem{}, false, persistence.ErrNotImplemented
	}
	row := queryable.QueryRow(ctx, queryGetComment, commentID)
	var item CommentItem
	var userID, parentID, createdAt string
	if err := row.Scan(&item.CommentID, &item.MomentID, &userID, &parentID, &item.Content, &createdAt, &item.LikesCount, &item.RepliesCount); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return CommentItem{}, false, nil
		}
		return CommentItem{}, false, err
	}
	item.User = UserInfo{UserID: userID, Nickname: userID}
	item.ParentID = parentID
	item.CreatedAt = createdAt
	item.MyState = &MyCommentState{IsLiked: false}
	return item, true, nil
}

func (r *PostgresRepository) ListReplies(ctx context.Context, commentID string, limit int, cursor string) (PaginatedComments, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return PaginatedComments{}, persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
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
		var userID, parentID, createdAt string
		if err := rows.Scan(&item.CommentID, &userID, &parentID, &item.Content, &createdAt, &item.LikesCount, &item.RepliesCount); err != nil {
			return PaginatedComments{}, err
		}
		item.User = UserInfo{UserID: userID, Nickname: userID}
		item.ParentID = parentID
		item.CreatedAt = createdAt
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

func (r *PostgresRepository) Like(ctx context.Context, commentID string, userID string) (bool, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return false, persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return false, persistence.ErrNotImplemented
	}
	var exists int
	if err := queryable.QueryRow(ctx, queryCheckCommentLike, commentID, userID).Scan(&exists); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return false, err
	} else if err == nil {
		return false, nil
	}
	now := time.Now().UTC()
	if err := queryable.Exec(ctx, queryInsertCommentLike, commentID, userID, now); err != nil {
		return false, err
	}
	if err := queryable.Exec(ctx, queryIncrementCommentLikeCount, commentID); err != nil {
		return false, err
	}
	return true, nil
}

func (r *PostgresRepository) Unlike(ctx context.Context, commentID string, userID string) error {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return persistence.ErrNotImplemented
	}
	if err := queryable.Exec(ctx, queryDeleteCommentLike, commentID, userID); err != nil {
		return err
	}
	return queryable.Exec(ctx, queryDecrementCommentLikeCount, commentID)
}

func (r *PostgresRepository) ListLikes(ctx context.Context, commentID string, limit int, cursor string) ([]CommentLikeItem, string, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return nil, "", persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
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
		if err := rows.Scan(&item.User.UserID, &item.User.Nickname, &item.LikedAt); err != nil {
			return nil, "", err
		}
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
		return nil, "", persistence.ErrNotImplemented
	}
	var decoded commentCursor
	if err := json.Unmarshal(blob, &decoded); err != nil || decoded.At == "" || decoded.ID == "" {
		return nil, "", persistence.ErrNotImplemented
	}
	at, err := time.Parse(time.RFC3339, decoded.At)
	if err != nil {
		return nil, "", persistence.ErrNotImplemented
	}
	return at.UTC(), decoded.ID, nil
}
