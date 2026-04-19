package private_messages

import (
	"context"
	"time"

	"hotspot/backend-go/internal/modules/users"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/observability"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/realtime"
)

type ReadService interface {
	Recent(context.Context, string, int, *time.Time) (PaginatedMessages, error)
	Last(context.Context, string, *time.Time) (*PrivateMessageItem, error)
	UnreadCount(context.Context, string, *time.Time) (int, error)
}

type Runtime struct {
	handler     *Handler
	readService ReadService
}

type userReader interface {
	OpenTheDoor(context.Context, string) (users.UserItem, bool, error)
	GetMySettings(context.Context) (users.UserSettings, error)
}

type Service struct {
	repo  Repository
	users userReader
}

func NewService() *Service {
	userRepo := users.NewInMemoryRepository()
	return NewServiceWithDeps(NewInMemoryRepository(), users.NewServiceWithRepo(userRepo))
}

func NewServiceWithRepo(repo Repository) *Service {
	userRepo := users.NewInMemoryRepository()
	return NewServiceWithDeps(repo, users.NewServiceWithRepo(userRepo))
}

func NewServiceWithDeps(repo Repository, userService userReader) *Service {
	return &Service{repo: repo, users: userService}
}

func NewRuntime(factory *persistence.Factory) *Runtime {
	return NewRuntimeWithRepo(NewPostgresRepository(factory))
}

func NewRuntimeWithRepo(repo Repository) *Runtime {
	service := NewServiceWithRepo(repo)
	return &Runtime{
		handler:     NewHandler(service),
		readService: service,
	}
}

func (r *Runtime) Handler() *Handler {
	return r.handler
}

func (r *Runtime) ReadService() ReadService {
	return r.readService
}

func (s *Service) Send(ctx context.Context, req SendPrivateMessageRequest) (MessageSent, error) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	if err := s.ensureCanSend(ctx, principal.UserID, req.TargetUserID); err != nil {
		return MessageSent{}, err
	}
	msg, err := s.repo.Send(ctx, principal.UserID, req.TargetUserID, req)
	if err != nil {
		observability.L().Error("privatemessages.send failed", "error", err, "target_user_id", req.TargetUserID)
		return MessageSent{}, err
	}
	realtime.DefaultHub().BroadcastToUser(req.TargetUserID, realtime.Event{Type: "PrivateMessageNew", Payload: map[string]any{"message_id": msg.MessageID, "sent_at": msg.SentAt}})
	return msg, nil
}

func (s *Service) History(ctx context.Context, req GetHistoryRequest) (PaginatedMessages, error) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	return s.repo.History(ctx, principal.UserID, req)
}

func (s *Service) Recent(ctx context.Context, targetUserID string, limit int, deletedAfter *time.Time) (PaginatedMessages, error) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	return s.repo.Recent(ctx, principal.UserID, targetUserID, limit, deletedAfter)
}

func (s *Service) Last(ctx context.Context, targetUserID string, deletedAfter *time.Time) (*PrivateMessageItem, error) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	return s.repo.Last(ctx, principal.UserID, targetUserID, deletedAfter)
}

func (s *Service) UnreadCount(ctx context.Context, targetUserID string, deletedAfter *time.Time) (int, error) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	return s.repo.UnreadCount(ctx, principal.UserID, targetUserID, deletedAfter)
}

func (s *Service) Update(ctx context.Context, req UpdatePrivateMessageRequest) error {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	meta, err := s.repo.Update(ctx, principal.UserID, req)
	if err != nil {
		observability.L().Error("privatemessages.update failed", "error", err, "message_id", req.MessageID)
		return err
	}
	event := realtime.Event{Type: "PrivateMessageEdited", Payload: map[string]any{"message_id": req.MessageID, "target_user_id": meta.TargetUserID}}
	realtime.DefaultHub().BroadcastToUser(meta.TargetUserID, event)
	realtime.DefaultHub().BroadcastToUser(principal.UserID, event)
	return nil
}

func (s *Service) Recall(ctx context.Context, req RecallPrivateMessageRequest) error {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	meta, err := s.repo.Recall(ctx, principal.UserID, req)
	if err != nil {
		observability.L().Error("privatemessages.recall failed", "error", err, "message_id", req.MessageID)
		return err
	}
	event := realtime.Event{Type: "PrivateMessageRecalled", Payload: map[string]any{"message_id": req.MessageID, "target_user_id": meta.TargetUserID}}
	realtime.DefaultHub().BroadcastToUser(meta.TargetUserID, event)
	realtime.DefaultHub().BroadcastToUser(principal.UserID, event)
	return nil
}

func (s *Service) MarkAsRead(ctx context.Context, req MarkPrivateMessagesReadRequest) error {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	meta, err := s.repo.MarkAsRead(ctx, principal.UserID, req)
	if err != nil {
		observability.L().Error("privatemessages.mark_as_read failed", "error", err, "target_user_id", req.TargetUserID, "up_to_message_id", req.UpToMessageID)
		return err
	}
	event := realtime.Event{Type: "PrivateMessageRead", Payload: map[string]any{"target_user_id": meta.TargetUserID, "message_id": meta.MessageID, "read_at": time.Now().UTC().Format(time.RFC3339Nano)}}
	realtime.DefaultHub().BroadcastToUser(meta.TargetUserID, event)
	if principal.UserID != meta.TargetUserID {
		realtime.DefaultHub().BroadcastToUser(principal.UserID, event)
	}
	return nil
}

func (s *Service) MarkAllRead(ctx context.Context) error {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	if err := s.repo.MarkAllRead(ctx, principal.UserID); err != nil {
		observability.L().Error("privatemessages.mark_all_read failed", "error", err)
		return err
	}
	return nil
}

func (s *Service) ensureCanSend(ctx context.Context, userID, targetUserID string) error {
	if s.users == nil {
		return nil
	}
	item, ok, err := s.users.OpenTheDoor(ctx, targetUserID)
	if err != nil {
		return err
	}
	if !ok {
		return users.ErrUserNotFound
	}
	if userID == targetUserID {
		return nil
	}
	if item.Relation != nil && item.Relation.IsBlocked {
		return ErrMessagePermissionDenied
	}
	settings, settingsErr := s.users.GetMySettings(platformauth.WithPrincipal(ctx, platformauth.Principal{UserID: targetUserID, SessionID: "private-message-probe"}))
	if settingsErr != nil {
		return settingsErr
	}
	switch settings.AllowPrivateChat {
	case users.PrivateChatNobody:
		return ErrMessagePermissionDenied
	case users.PrivateChatFriends:
		if item.Relation == nil || !(item.Relation.IsFollowing && item.Relation.IsFollowedBy) {
			return ErrMessagePermissionDenied
		}
	}
	return nil
}
