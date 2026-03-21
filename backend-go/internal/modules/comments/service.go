package comments

import (
	"context"
	"fmt"

	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/realtime"
)

type Service struct {
	repo    Repository
	moments MomentCounter
}

func NewService(repo Repository, momentsStore MomentCounter) *Service {
	return &Service{repo: repo, moments: momentsStore}
}

func (s *Service) Create(ctx context.Context, req CreateCommentRequest) (CommentItem, error) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	item, err := s.repo.Create(ctx, principal.UserID, req)
	if err != nil {
		return CommentItem{}, err
	}
	if err := s.moments.IncrementCommentCount(ctx, req.MomentID); err != nil {
		if rollbackErr := s.repo.Delete(ctx, item.CommentID); rollbackErr != nil {
			return CommentItem{}, fmt.Errorf("increment comment count: %w; rollback delete comment: %v", err, rollbackErr)
		}
		return CommentItem{}, err
	}
	realtime.DefaultHub().BroadcastToUser(principal.UserID, realtime.Event{Type: "CommentReply", Payload: map[string]any{"comment_id": item.CommentID, "replied_comment_preview": item.Content, "reply_id": item.CommentID, "reply_content_preview": item.Content, "moment_id": req.MomentID, "moment_summary": map[string]any{"moment_id": req.MomentID, "description_preview": item.Content}}})
	return item, nil
}

func (s *Service) List(ctx context.Context, momentID string) (PaginatedComments, error) {
	return s.repo.List(ctx, momentID)
}
