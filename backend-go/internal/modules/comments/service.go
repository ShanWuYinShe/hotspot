package comments

import (
	"context"
	"errors"
	"fmt"

	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/realtime"
)

var ErrMomentNotFound = errors.New("comments: moment not found")

type Service struct {
	repo    Repository
	moments MomentProvider
}

func NewService(repo Repository, moments MomentProvider) *Service {
	return &Service{repo: repo, moments: moments}
}

func (s *Service) Create(ctx context.Context, req CreateCommentRequest) (CommentItem, error) {
	if err := s.ensureMomentExists(ctx, req.MomentID); err != nil {
		return CommentItem{}, err
	}
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
	if err := s.ensureMomentExists(ctx, momentID); err != nil {
		return PaginatedComments{}, err
	}
	return s.repo.List(ctx, momentID)
}

func (s *Service) ensureMomentExists(ctx context.Context, momentID string) error {
	exists, err := s.moments.Exists(ctx, momentID)
	if err != nil {
		return err
	}
	if !exists {
		return ErrMomentNotFound
	}
	return nil
}
