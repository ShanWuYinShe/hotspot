package comments

import (
	"context"

	"hotspot/backend-go/internal/platform/persistence"
)

type PersistenceRepository struct{ impl Repository }

func NewPersistenceRepository(factory *persistence.Factory) *PersistenceRepository {
	return &PersistenceRepository{impl: NewPostgresRepository(factory)}
}

func (r *PersistenceRepository) Create(ctx context.Context, userID string, req CreateCommentRequest) (CommentItem, error) {
	return r.impl.Create(ctx, userID, req)
}

func (r *PersistenceRepository) SoftDelete(ctx context.Context, commentID string) error {
	return r.impl.SoftDelete(ctx, commentID)
}

func (r *PersistenceRepository) List(ctx context.Context, momentID string, limit int, cursor string) (PaginatedComments, error) {
	return r.impl.List(ctx, momentID, limit, cursor)
}

func (r *PersistenceRepository) Get(ctx context.Context, commentID string) (CommentItem, bool, error) {
	return r.impl.Get(ctx, commentID)
}

func (r *PersistenceRepository) ListReplies(ctx context.Context, commentID string, limit int, cursor string) (PaginatedComments, error) {
	return r.impl.ListReplies(ctx, commentID, limit, cursor)
}

func (r *PersistenceRepository) ListTopRepliesByParentIDs(ctx context.Context, parentIDs []string, limit int) (map[string][]CommentItem, error) {
	return r.impl.ListTopRepliesByParentIDs(ctx, parentIDs, limit)
}
