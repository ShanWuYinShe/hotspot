package comments

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"hotspot/backend-go/internal/modules/notifications"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/observability"
	"hotspot/backend-go/internal/platform/realtime"
)

var ErrMomentNotFound = errors.New("comments: moment not found")

type Service struct {
	repo          Repository
	moments       MomentProvider
	notifications commentNotificationWriter
}

type commentNotificationWriter interface {
	AppendCommentReply(ctx context.Context, targetUserID string, actor map[string]any, momentID string, momentSummary map[string]any, parentCommentID string, parentPreview string, replyID string, replyPreview string) error
}

func NewService(repo Repository, moments MomentProvider) *Service {
	return NewServiceWithDeps(repo, moments, notifications.NewService())
}

func NewServiceWithDeps(repo Repository, moments MomentProvider, notificationWriter commentNotificationWriter) *Service {
	return &Service{repo: repo, moments: moments, notifications: notificationWriter}
}

func (s *Service) Create(ctx context.Context, req CreateCommentRequest) (CommentItem, error) {
	if err := s.ensureMomentExists(ctx, req.MomentID); err != nil {
		return CommentItem{}, err
	}
	principal, _ := platformauth.PrincipalFromContext(ctx)
	item, err := s.repo.Create(ctx, principal.UserID, req)
	if err != nil {
		observability.L().Error("comments.create repo create failed", "error", err)
		return CommentItem{}, err
	}
	if err := s.moments.IncrementCommentCount(ctx, req.MomentID); err != nil {
		observability.L().Error("comments.create increment comment count failed", "error", err, "moment_id", req.MomentID, "comment_id", item.CommentID)
		if rollbackErr := s.repo.Delete(ctx, item.CommentID); rollbackErr != nil {
			observability.L().Error("comments.create rollback delete failed", "error", rollbackErr, "comment_id", item.CommentID)
			return CommentItem{}, fmt.Errorf("increment comment count: %w; rollback delete comment: %v", err, rollbackErr)
		}
		return CommentItem{}, err
	}
	if req.ParentID == "" {
		return item, nil
	}
	parent, found, err := s.repo.Get(ctx, req.ParentID)
	if err != nil {
		return CommentItem{}, err
	}
	if !found {
		return CommentItem{}, ErrCommentNotFound
	}
	targetUserID := parent.User.UserID
	if targetUserID == "" || targetUserID == principal.UserID {
		return item, nil
	}
	actor := map[string]any{"user_id": principal.UserID, "nickname": principal.UserID}
	momentSummary := map[string]any{"moment_id": req.MomentID, "description_preview": summarizeCommentText(parent.Content, item.Content)}
	payload := map[string]any{
		"moment_id":               req.MomentID,
		"moment_summary":          momentSummary,
		"comment_id":              parent.CommentID,
		"replied_comment_preview": summarizeCommentText(parent.Content, ""),
		"reply_id":                item.CommentID,
		"actor":                   actor,
		"content":                 item.Content,
		"replied_at":              time.Now().UTC().Format(time.RFC3339),
	}
	if s.notifications != nil {
		if err := s.notifications.AppendCommentReply(ctx, targetUserID, actor, req.MomentID, momentSummary, parent.CommentID, summarizeCommentText(parent.Content, ""), item.CommentID, summarizeCommentText(item.Content, "")); err != nil {
			return CommentItem{}, err
		}
	}
	realtime.DefaultHub().BroadcastToUser(targetUserID, realtime.Event{Type: "CommentReply", Payload: payload})
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

func summarizeCommentText(primary string, fallback string) string {
	value := strings.TrimSpace(primary)
	if value == "" {
		value = strings.TrimSpace(fallback)
	}
	if len(value) > 80 {
		return value[:80]
	}
	return value
}

func (s *Service) ensureMomentExists(ctx context.Context, momentID string) error {
	exists, err := s.moments.Exists(ctx, momentID)
	if err != nil {
		observability.L().Error("comments.ensure_moment_exists failed", "error", err, "moment_id", momentID)
		return err
	}
	if !exists {
		return ErrMomentNotFound
	}
	return nil
}
