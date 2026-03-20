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
	messages *private_messages.Service
}

func NewService() *Service {
	return NewServiceWithRepos(NewInMemoryRepository(), private_messages.NewService())
}
func NewServiceWithRepos(repo Repository, messages *private_messages.Service) *Service {
	return &Service{repo: repo, messages: messages}
}
func (s *Service) OpenTheDoor(ctx context.Context, targetUserID string, recentLimit int) PrivateChatOpenResult {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	createdAt := s.repo.GetOrCreate(ctx, principal.UserID, targetUserID)
	return PrivateChatOpenResult{Chat: s.buildChat(ctx, principal.UserID, targetUserID, createdAt), RecentMessages: s.messages.Recent(ctx, targetUserID, recentLimit)}
}
func (s *Service) List(ctx context.Context) PaginatedPrivateChats {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	created := s.repo.List(ctx, principal.UserID)
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
		items = append(items, s.buildChat(ctx, principal.UserID, target, createdAt))
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
	return PaginatedPrivateChats{Items: items}
}
func (s *Service) buildChat(ctx context.Context, userID, targetUserID string, createdAt time.Time) PrivateChatItem {
	last := s.messages.Last(ctx, targetUserID)
	item := PrivateChatItem{OtherUser: UserInfo{UserID: targetUserID, Nickname: targetUserID}, UnreadCount: 0, IsMuted: false, IsPinned: false, LastActivityAt: createdAt.UTC().Format(time.RFC3339), CreatedAt: createdAt.UTC().Format(time.RFC3339), CanMessage: true}
	if last != nil {
		item.LastActivityAt = last.SentAt
		item.LastMessage = &LastMessage{MessageID: last.MessageID, Content: last.Content, Type: last.Type, ShareCode: last.ShareCode, Sender: UserInfo{UserID: last.Sender.UserID, Nickname: last.Sender.Nickname}, SentAt: last.SentAt}
	}
	return item
}
