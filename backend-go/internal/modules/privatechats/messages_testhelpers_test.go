package privatechats

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"hotspot/backend-go/internal/modules/privatemessages"
	platformauth "hotspot/backend-go/internal/platform/auth"
)

// fakeMessageStore 是私聊消息的最小测试存储：privatemessages 内存实现已迁入其
// 模块测试构建（P0-2），privatechats 跨模块用例只需要「发消息后可读历史/最近」
// 的行为。同一份存储同时驱动消息 handler（Repository）与会话服务（ReadService）。
type fakeMessageStore struct {
	mu   sync.Mutex
	seq  int
	msgs []fakeStoredMessage
}

type fakeStoredMessage struct {
	id             string
	sender, target string
	content        string
	sentAt         string
}

func newFakeMessageStore() *fakeMessageStore { return &fakeMessageStore{} }

func (s *fakeMessageStore) add(sender, target, content string) (string, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	now := time.Now().UTC()
	id := fmt.Sprintf("pm_test_%d_%d", now.UnixNano(), s.seq)
	sentAt := now.Format(time.RFC3339Nano)
	s.msgs = append(s.msgs, fakeStoredMessage{id: id, sender: sender, target: target, content: content, sentAt: sentAt})
	return id, sentAt
}

func (s *fakeMessageStore) items(userID, targetUserID string, limit int, descending bool, after *time.Time) []privatemessages.PrivateMessageItem {
	s.mu.Lock()
	defer s.mu.Unlock()
	items := make([]privatemessages.PrivateMessageItem, 0)
	for _, m := range s.msgs {
		if !((m.sender == userID && m.target == targetUserID) || (m.sender == targetUserID && m.target == userID)) {
			continue
		}
		if after != nil {
			sent, err := time.Parse(time.RFC3339Nano, m.sentAt)
			if err != nil || !sent.After(*after) {
				continue
			}
		}
		content := m.content
		items = append(items, privatemessages.PrivateMessageItem{
			MessageID: m.id,
			Type:      "text",
			Content:   &content,
			Sender:    privatemessages.UserInfo{UserID: m.sender, Nickname: m.sender},
			SentAt:    m.sentAt,
		})
	}
	sort.SliceStable(items, func(i, j int) bool {
		if descending {
			return items[i].SentAt > items[j].SentAt
		}
		return items[i].SentAt < items[j].SentAt
	})
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	return items
}

func (s *fakeMessageStore) lastByChatKey(chatKeys map[string]struct{}, boundaries map[string]*time.Time) map[string]privatemessages.PrivateMessageItem {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[string]privatemessages.PrivateMessageItem)
	for _, m := range s.msgs {
		ck := key(m.sender, m.target)
		if _, want := chatKeys[ck]; !want {
			continue
		}
		if after := boundaries[ck]; after != nil {
			sent, err := time.Parse(time.RFC3339Nano, m.sentAt)
			if err != nil || !sent.After(*after) {
				continue
			}
		}
		content := m.content
		out[ck] = privatemessages.PrivateMessageItem{
			MessageID: m.id, Type: "text", Content: &content,
			Sender: privatemessages.UserInfo{UserID: m.sender, Nickname: m.sender}, SentAt: m.sentAt,
		}
	}
	return out
}

// fakeMessageRepo 实现 privatemessages.Repository 中被消息 handler 使用的方法。
type fakeMessageRepo struct {
	privatemessages.Repository
	store *fakeMessageStore
}

func newFakeMessageRepo(store *fakeMessageStore) *fakeMessageRepo {
	return &fakeMessageRepo{store: store}
}

func (r *fakeMessageRepo) Send(_ context.Context, senderID, targetUserID string, req privatemessages.SendPrivateMessageRequest) (privatemessages.MessageSentResult, error) {
	id, sentAt := r.store.add(senderID, targetUserID, req.Content)
	return privatemessages.MessageSentResult{MessageID: id, SentAt: sentAt}, nil
}

func (r *fakeMessageRepo) History(_ context.Context, userID string, req privatemessages.ListPrivateMessagesRequest) (privatemessages.PaginatedMessages, error) {
	limit := 20
	if req.Pagination != nil && req.Pagination.Limit > 0 {
		limit = req.Pagination.Limit
	}
	return privatemessages.PaginatedMessages{Items: r.store.items(userID, req.TargetUserID, limit, true, nil)}, nil
}

// LockStrangerQuota 幂等 no-op（测试不涉及配额事务语义）。
func (r *fakeMessageRepo) LockStrangerQuota(context.Context, string, string) error { return nil }

// CountMessagesByTarget 无陌生人配额限制。
func (r *fakeMessageRepo) CountMessagesByTarget(context.Context, string, time.Time) (map[string]int, error) {
	return map[string]int{}, nil
}

// fakeReadService 实现 privatemessages.ReadService（会话列表/打开会话读取）。
type fakeReadService struct {
	privatemessages.ReadService
	store *fakeMessageStore
}

func newFakeReadService(store *fakeMessageStore) *fakeReadService {
	return &fakeReadService{store: store}
}

func (s *fakeReadService) Recent(ctx context.Context, targetUserID string, limit int, deletedAfter *time.Time) (privatemessages.PaginatedMessages, error) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	return privatemessages.PaginatedMessages{Items: s.store.items(principal.UserID, targetUserID, limit, true, deletedAfter)}, nil
}

func (s *fakeReadService) Last(ctx context.Context, targetUserID string, deletedAfter *time.Time) (*privatemessages.PrivateMessageItem, error) {
	principal, _ := platformauth.PrincipalFromContext(ctx)
	items := s.store.items(principal.UserID, targetUserID, 1, true, deletedAfter)
	if len(items) == 0 {
		return nil, nil
	}
	return &items[0], nil
}

func (s *fakeReadService) UnreadCount(context.Context, string, *time.Time) (int, error) {
	return 0, nil
}

func (s *fakeReadService) LastByChatKeys(_ context.Context, chatKeys []string, boundaries map[string]*time.Time) (map[string]privatemessages.PrivateMessageItem, error) {
	wanted := make(map[string]struct{}, len(chatKeys))
	for _, k := range chatKeys {
		wanted[k] = struct{}{}
	}
	return s.store.lastByChatKey(wanted, boundaries), nil
}

func (s *fakeReadService) UnreadCountsByChatKeys(context.Context, []string, map[string]*time.Time) (map[string]int, error) {
	return map[string]int{}, nil
}

func (s *fakeReadService) LookupMessage(context.Context, string) (bool, string, error) {
	return false, "", nil
}

func (s *fakeReadService) IsReactionTargetVisible(context.Context, string) (bool, bool, error) {
	return true, true, nil
}

func (s *fakeReadService) GetMessageReportSnapshot(context.Context, string) (privatemessages.MessageReportSnapshot, bool, error) {
	return privatemessages.MessageReportSnapshot{}, false, nil
}
