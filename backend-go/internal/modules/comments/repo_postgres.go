package comments

import (
	"context"
	"fmt"
	"time"

	"hotspot/backend-go/internal/platform/persistence"
)

type PostgresRepository struct {
	factory  *persistence.Factory
	fallback Repository
}

func NewPostgresRepository(factory *persistence.Factory, fallback Repository) *PostgresRepository {
	if fallback == nil {
		fallback = NewInMemoryRepository()
	}
	return &PostgresRepository{factory: factory, fallback: fallback}
}

func (r *PostgresRepository) Create(ctx context.Context, userID string, req CreateCommentRequest) (CommentItem, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return r.fallback.Create(ctx, userID, req)
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
		return r.fallback.Delete(ctx, commentID)
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return persistence.ErrNotImplemented
	}
	return queryable.Exec(ctx, queryDeleteComment, commentID)
}

func (r *PostgresRepository) List(ctx context.Context, momentID string) (PaginatedComments, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return r.fallback.List(ctx, momentID)
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
