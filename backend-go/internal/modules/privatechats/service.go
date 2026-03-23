package privatechats

import (
	"context"
	"sort"
	"strings"
	"time"

	private_messages "hotspot/backend-go/internal/modules/privatemessages"
	platformauth "hotspot/backend-go/internal/platform/auth"
)

type Service struct {
	repo     Repository
	messages private_messages.ReadService
}

func NewService() *Service {
	return NewServiceWithRepos(NewInMemoryRepository(), private_messages.NewService())
}
func NewServiceWithRepos(repo Repository, messages private_messages.ReadService) *Service {
	return &Service{repo: repo, messages: messages}
}
func (s *Service) OpenTheDoor(ctx context.Context, targetUserID string, recentLimit int) (PrivateChatOpenResult, error) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	createdAt, err := s.repo.GetOrCreate(ctx, principal.UserID, targetUserID)
	if err != nil {
		return PrivateChatOpenResult{}, err
	}
	chat, err := s.buildChat(ctx, principal.UserID, targetUserID, createdAt)
	if err != nil {
		return PrivateChatOpenResult{}, err
	}
	recent, err := s.messages.Recent(ctx, targetUserID, recentLimit)
	if err != nil {
		return PrivateChatOpenResult{}, err
	}
	return PrivateChatOpenResult{Chat: chat, RecentMessages: recent}, nil
}
func (s *Service) List(ctx context.Context) (PaginatedPrivateChats, error) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	created, err := s.repo.List(ctx, principal.UserID)
	if err != nil {
		return PaginatedPrivateChats{}, err
	}
	items := make([]PrivateChatItem, 0)
	for k, createdAt := range created {
		parts := strings.Split(k, ":")
		if len(parts) != 2 {
			continue
		}
		var target string
		switch {
		case parts[0] == principal.UserID:
			target = parts[1]
		case parts[1] == principal.UserID:
			target = parts[0]
		default:
			continue
		}
		item, err := s.buildChat(ctx, principal.UserID, target, createdAt)
		if err != nil {
			return PaginatedPrivateChats{}, err
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
	return PaginatedPrivateChats{Items: items}, nil
}
func (s *Service) buildChat(ctx context.Context, userID, targetUserID string, createdAt time.Time) (PrivateChatItem, error) {
	last, err := s.messages.Last(ctx, targetUserID)
	if err != nil {
		return PrivateChatItem{}, err
	}
	item := PrivateChatItem{OtherUser: UserInfo{UserID: targetUserID, Nickname: targetUserID}, UnreadCount: 0, IsMuted: false, IsPinned: false, LastActivityAt: createdAt.UTC().Format(time.RFC3339), CreatedAt: createdAt.UTC().Format(time.RFC3339), CanMessage: true}
	if last != nil {
		item.LastActivityAt = last.SentAt
		item.LastMessage = &LastMessage{MessageID: last.MessageID, Content: last.Content, Type: last.Type, ShareCode: last.ShareCode, Sender: UserInfo{UserID: last.Sender.UserID, Nickname: last.Sender.Nickname}, SentAt: last.SentAt}
	}
	_ = userID
	return item, nil
}
