package private_messages

import (
	"context"

	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/realtime"
)

type ReadService interface {
	Recent(context.Context, string, int) (PaginatedMessages, error)
	Last(context.Context, string) (*PrivateMessageItem, error)
}

type Runtime struct {
	handler     *Handler
	readService ReadService
}

type Service struct {
	repo Repository
}

func NewService() *Service {
	return NewServiceWithRepo(NewInMemoryRepository())
}

func NewServiceWithRepo(repo Repository) *Service {
	return &Service{repo: repo}
}

func NewRuntime(factory *persistence.Factory) *Runtime {
	return NewRuntimeWithRepo(NewPostgresRepository(factory, nil))
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
	msg, err := s.repo.Send(ctx, principal.UserID, req.TargetUserID, req)
	if err != nil {
		return MessageSent{}, err
	}
	realtime.DefaultHub().BroadcastToUser(req.TargetUserID, realtime.Event{Type: "PrivateMessageNew", Payload: map[string]any{"message_id": msg.MessageID, "sent_at": msg.SentAt}})
	return msg, nil
}

func (s *Service) History(ctx context.Context, targetUserID string) (PaginatedMessages, error) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	return s.repo.History(ctx, principal.UserID, targetUserID)
}

func (s *Service) Recent(ctx context.Context, targetUserID string, limit int) (PaginatedMessages, error) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	return s.repo.Recent(ctx, principal.UserID, targetUserID, limit)
}

func (s *Service) Last(ctx context.Context, targetUserID string) (*PrivateMessageItem, error) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	return s.repo.Last(ctx, principal.UserID, targetUserID)
}

func (s *Service) Update(ctx context.Context, req UpdatePrivateMessageRequest) error {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	return s.repo.Update(ctx, principal.UserID, req)
}

func (s *Service) Recall(ctx context.Context, req RecallPrivateMessageRequest) error {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	return s.repo.Recall(ctx, principal.UserID, req)
}

func (s *Service) MarkAsRead(ctx context.Context, req MarkPrivateMessagesReadRequest) error {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	return s.repo.MarkAsRead(ctx, principal.UserID, req)
}

func (s *Service) MarkAllRead(ctx context.Context) error {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	return s.repo.MarkAllRead(ctx, principal.UserID)
}
