package comments

import (
	"context"

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

func (s *Service) Create(ctx context.Context, req CreateCommentRequest) CommentItem {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	item := s.repo.Create(ctx, principal.UserID, req)
	s.moments.IncrementCommentCount(req.MomentID)
	realtime.DefaultHub().BroadcastToUser(principal.UserID, realtime.Event{Type: "CommentReply", Payload: map[string]any{"comment_id": item.CommentID, "replied_comment_preview": item.Content, "reply_id": item.CommentID, "reply_content_preview": item.Content, "moment_id": req.MomentID, "moment_summary": map[string]any{"moment_id": req.MomentID, "description_preview": item.Content}}})
	return item
}

func (s *Service) List(ctx context.Context, momentID string) PaginatedComments {
	return s.repo.List(ctx, momentID)
}
