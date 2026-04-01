package privatechats

import (
	"context"
	"sort"
	"time"

	private_messages "hotspot/backend-go/internal/modules/privatemessages"
	"hotspot/backend-go/internal/modules/users"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/observability"
)

type userReader interface {
	OpenTheDoor(context.Context, string) (users.UserItem, bool, error)
	GetMySettings(context.Context) (users.UserSettings, error)
}

type Service struct {
	repo     Repository
	messages private_messages.ReadService
	users    userReader
}

func NewService() *Service {
	userRepo := users.NewInMemoryRepository()
	return NewServiceWithDeps(NewInMemoryRepository(), private_messages.NewService(), users.NewServiceWithRepo(userRepo))
}
func NewServiceWithRepos(repo Repository, messages private_messages.ReadService) *Service {
	userRepo := users.NewInMemoryRepository()
	return NewServiceWithDeps(repo, messages, users.NewServiceWithRepo(userRepo))
}
func NewServiceWithDeps(repo Repository, messages private_messages.ReadService, userService userReader) *Service {
	return &Service{repo: repo, messages: messages, users: userService}
}
func (s *Service) OpenTheDoor(ctx context.Context, targetUserID string, recentLimit int) (PrivateChatOpenResult, error) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	createdAt, err := s.repo.GetOrCreate(ctx, principal.UserID, targetUserID)
	if err != nil {
		observability.L().Error("privatechats.open_the_door get_or_create failed", "error", err, "target_user_id", targetUserID)
		return PrivateChatOpenResult{}, err
	}
	state, err := s.repo.GetState(ctx, principal.UserID, targetUserID)
	if err != nil {
		observability.L().Error("privatechats.open_the_door get_state failed", "error", err, "target_user_id", targetUserID)
		return PrivateChatOpenResult{}, err
	}
	if state.CreatedAt.IsZero() {
		state.CreatedAt = createdAt
	}
	boundary := visibleBoundary(state)
	recent, err := s.messages.Recent(ctx, targetUserID, recentLimit, boundary)
	if err != nil {
		observability.L().Error("privatechats.open_the_door recent failed", "error", err, "target_user_id", targetUserID)
		return PrivateChatOpenResult{}, err
	}
	if state.DeletedAt != nil && len(recent.Items) == 0 {
		now := time.Now().UTC()
		if err := s.repo.RestoreForUser(ctx, principal.UserID, targetUserID, now); err != nil {
			observability.L().Error("privatechats.open_the_door restore failed", "error", err, "target_user_id", targetUserID)
			return PrivateChatOpenResult{}, err
		}
		state.DeletedAt = nil
		state.VisibleSince = &now
	}
	chat, err := s.buildChat(ctx, principal.UserID, state)
	if err != nil {
		observability.L().Error("privatechats.open_the_door build_chat failed", "error", err, "target_user_id", targetUserID)
		return PrivateChatOpenResult{}, err
	}
	return PrivateChatOpenResult{Chat: chat, RecentMessages: recent}, nil
}
func (s *Service) List(ctx context.Context, req MyPrivateChatsRequest) (PaginatedPrivateChats, error) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	states, _, err := s.repo.List(ctx, principal.UserID, ListOptions{})
	if err != nil {
		observability.L().Error("privatechats.list failed", "error", err)
		return PaginatedPrivateChats{}, err
	}
	items := make([]PrivateChatItem, 0, len(states))
	for _, state := range states {
		item, err := s.buildChat(ctx, principal.UserID, state)
		if err != nil {
			return PaginatedPrivateChats{}, err
		}
		if state.DeletedAt != nil && item.LastMessage == nil {
			continue
		}
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].IsPinned != items[j].IsPinned {
			return items[i].IsPinned
		}
		if items[i].LastActivityAt == items[j].LastActivityAt {
			return items[i].OtherUser.UserID < items[j].OtherUser.UserID
		}
		return items[i].LastActivityAt > items[j].LastActivityAt
	})
	cursor := ""
	if req.Pagination != nil {
		cursor = req.Pagination.Cursor
	}
	if cursor != "" {
		start := 0
		for start < len(items) && items[start].OtherUser.UserID != cursor {
			start++
		}
		if start == len(items) {
			return PaginatedPrivateChats{Items: []PrivateChatItem{}}, nil
		}
		items = items[start+1:]
	}
	limit := 20
	if req.Pagination != nil && req.Pagination.Limit > 0 {
		limit = req.Pagination.Limit
	}
	if len(items) <= limit {
		return PaginatedPrivateChats{Items: items}, nil
	}
	return PaginatedPrivateChats{Items: items[:limit], NextCursor: items[limit-1].OtherUser.UserID}, nil
}

func (s *Service) UpdateSettings(ctx context.Context, req UpdatePrivateChatSettingsRequest) error {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	if _, err := s.repo.GetOrCreate(ctx, principal.UserID, req.TargetUserID); err != nil {
		return err
	}
	return s.repo.UpdateSettings(ctx, principal.UserID, req.TargetUserID, UpdateChatSettings{IsMuted: req.IsMuted, IsPinned: req.IsPinned})
}

func (s *Service) Delete(ctx context.Context, targetUserID string) error {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	now := time.Now().UTC()
	return s.repo.DeleteForUser(ctx, principal.UserID, targetUserID, &now)
}

func (s *Service) buildChat(ctx context.Context, userID string, state ChatState) (PrivateChatItem, error) {
	userItem, ok, err := s.users.OpenTheDoor(ctx, state.TargetID)
	if err != nil {
		return PrivateChatItem{}, err
	}
	if !ok {
		return PrivateChatItem{}, users.ErrUserNotFound
	}
	boundary := visibleBoundary(state)
	last, err := s.messages.Last(ctx, state.TargetID, boundary)
	if err != nil {
		observability.L().Error("privatechats.build_chat last message failed", "error", err, "target_user_id", state.TargetID)
		return PrivateChatItem{}, err
	}
	unreadCount, err := s.messages.UnreadCount(ctx, state.TargetID, boundary)
	if err != nil {
		return PrivateChatItem{}, err
	}
	item := PrivateChatItem{
		OtherUser:      UserInfo{UserID: userItem.User.UserID, Nickname: userItem.User.Nickname, AvatarURL: userItem.User.AvatarURL},
		UnreadCount:    unreadCount,
		IsMuted:        state.IsMuted,
		IsPinned:       state.IsPinned,
		LastActivityAt: state.CreatedAt.UTC().Format(time.RFC3339Nano),
		CreatedAt:      state.CreatedAt.UTC().Format(time.RFC3339Nano),
		CanMessage:     true,
	}
	if userItem.PresenceStatus != "" {
		item.PresenceStatus = string(userItem.PresenceStatus)
	}
	if userItem.Relation != nil && userItem.Relation.IsBlocked {
		item.CanMessage = false
		item.MessageBlockedReason = "privacy"
	}
	settings, settingsErr := s.users.GetMySettings(platformauth.WithPrincipal(ctx, platformauth.Principal{UserID: state.TargetID, SessionID: "private-chat-probe"}))
	if settingsErr == nil {
		switch settings.AllowPrivateChat {
		case users.PrivateChatNobody:
			item.CanMessage = false
			item.MessageBlockedReason = "privacy"
		case users.PrivateChatFriends:
			if userItem.Relation == nil || !(userItem.Relation.IsFollowing && userItem.Relation.IsFollowedBy) {
				item.CanMessage = false
				item.MessageBlockedReason = "privacy"
			}
		}
	}
	if userID == state.TargetID {
		item.CanMessage = true
		item.MessageBlockedReason = ""
	}
	if last != nil {
		item.LastActivityAt = last.SentAt
		item.LastMessage = &LastMessage{MessageID: last.MessageID, Content: last.Content, Type: last.Type, ShareCode: last.ShareCode, Sender: UserInfo{UserID: last.Sender.UserID, Nickname: last.Sender.Nickname, AvatarURL: last.Sender.AvatarURL}, SentAt: last.SentAt}
	}
	return item, nil
}

func visibleBoundary(state ChatState) *time.Time {
	if state.VisibleSince != nil {
		return state.VisibleSince
	}
	return state.DeletedAt
}
