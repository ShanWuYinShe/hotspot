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

func (s *Service) Delete(ctx context.Context, req DeleteCommentRequest) error {
	comment, found, err := s.repo.Get(ctx, req.CommentID)
	if err != nil {
		return err
	}
	if !found {
		return ErrCommentNotFound
	}
	principal, _ := platformauth.PrincipalFromContext(ctx)
	if comment.User.UserID != principal.UserID {
		return ErrPermissionDenied
	}
	if err := s.repo.Delete(ctx, req.CommentID); err != nil {
		return err
	}
	_ = s.moments.DecrementCommentCount(ctx, comment.MomentID)
	return nil
}

func (s *Service) ListReplies(ctx context.Context, req CommentRepliesRequest) (PaginatedComments, error) {
	_, found, err := s.repo.Get(ctx, req.CommentID)
	if err != nil {
		return PaginatedComments{}, err
	}
	if !found {
		return PaginatedComments{}, ErrCommentNotFound
	}
	limit := 20
	cursor := ""
	if req.Pagination != nil {
		if req.Pagination.Limit > 0 {
			limit = req.Pagination.Limit
		}
		if limit > 100 {
			limit = 100
		}
		cursor = req.Pagination.Cursor
	}
	return s.repo.ListReplies(ctx, req.CommentID, limit, cursor)
}

func (s *Service) Like(ctx context.Context, req LikeCommentRequest) error {
	_, found, err := s.repo.Get(ctx, req.CommentID)
	if err != nil {
		return err
	}
	if !found {
		return ErrCommentNotFound
	}
	principal, _ := platformauth.PrincipalFromContext(ctx)
	return s.repo.Like(ctx, req.CommentID, principal.UserID)
}

func (s *Service) Unlike(ctx context.Context, req UnlikeCommentRequest) error {
	_, found, err := s.repo.Get(ctx, req.CommentID)
	if err != nil {
		return err
	}
	if !found {
		return ErrCommentNotFound
	}
	principal, _ := platformauth.PrincipalFromContext(ctx)
	return s.repo.Unlike(ctx, req.CommentID, principal.UserID)
}

func (s *Service) ListLikes(ctx context.Context, req CommentLikesRequest) (PaginatedCommentLikes, error) {
	_, found, err := s.repo.Get(ctx, req.CommentID)
	if err != nil {
		return PaginatedCommentLikes{}, err
	}
	if !found {
		return PaginatedCommentLikes{}, ErrCommentNotFound
	}
	limit := 20
	cursor := ""
	if req.Pagination != nil {
		if req.Pagination.Limit > 0 {
			limit = req.Pagination.Limit
		}
		if limit > 100 {
			limit = 100
		}
		cursor = req.Pagination.Cursor
	}
	items, nextCursor, err := s.repo.ListLikes(ctx, req.CommentID, limit, cursor)
	if err != nil {
		return PaginatedCommentLikes{}, err
	}
	return PaginatedCommentLikes{Items: items, NextCursor: nextCursor}, nil
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
