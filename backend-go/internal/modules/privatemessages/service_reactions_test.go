package privatemessages

import (
	"context"
	"errors"
	"testing"

	"hotspot/backend-go/internal/modules/users"
	platformauth "hotspot/backend-go/internal/platform/auth"
)

type stubPrivateReactionsSource struct {
	agg   map[string][]Reaction
	err   error
	calls int
}

func (s *stubPrivateReactionsSource) AggregatedMessageReactions(_ context.Context, _ []string) (map[string][]Reaction, error) {
	s.calls++
	return s.agg, s.err
}

func privateReactionsTestService() (*Service, *InMemoryRepository) {
	userStub := &privateMessageUserStub{
		item: users.UserItem{Relation: &users.Relationship{}},
		ok:   true,
		settings: users.UserSettings{
			AllowPrivateChat: users.ChatPermEveryone,
		},
	}
	repo := NewInMemoryRepository()
	return NewServiceWithDeps(repo, userStub), repo
}

func privateReactionsCtx(userID string) context.Context {
	return platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: userID, SessionID: "sess_1"})
}

// 回归（spec api_spec_chat.md PrivateMessageItem.reactions 恒在数组）：
// private_messages/fetch 与 open_the_door.recent_messages（Recent）的消息项
// 必须带 reactions——无数据源空数组兜底、有数据源按查看者回填、失败降级。
func TestServiceHistoryAttachesReactions(t *testing.T) {
	svc, _ := privateReactionsTestService()
	ctx := privateReactionsCtx("user_alice")
	if _, err := svc.Send(ctx, SendPrivateMessageRequest{TargetUserID: "user_bob", Type: "text", Content: "hello"}); err != nil {
		t.Fatalf("send: %v", err)
	}

	result, err := svc.History(ctx, ListPrivateMessagesRequest{TargetUserID: "user_bob"})
	if err != nil {
		t.Fatalf("history: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(result.Items))
	}
	if result.Items[0].Reactions == nil || len(result.Items[0].Reactions) != 0 {
		t.Fatalf("expected empty non-nil reactions without source, got %#v", result.Items[0].Reactions)
	}

	src := &stubPrivateReactionsSource{agg: map[string][]Reaction{
		result.Items[0].MessageID: {{Emoji: "❤️", Count: 2, ReactedByMe: true}},
	}}
	svc.SetReactionsSource(src)
	result, err = svc.History(ctx, ListPrivateMessagesRequest{TargetUserID: "user_bob"})
	if err != nil {
		t.Fatalf("history with source: %v", err)
	}
	if src.calls != 1 {
		t.Fatalf("expected reactions source called once, got %d", src.calls)
	}
	if len(result.Items[0].Reactions) != 1 || result.Items[0].Reactions[0].Emoji != "❤️" || result.Items[0].Reactions[0].Count != 2 || !result.Items[0].Reactions[0].ReactedByMe {
		t.Fatalf("unexpected reactions: %#v", result.Items[0].Reactions)
	}

	svc.SetReactionsSource(&stubPrivateReactionsSource{err: errors.New("boom")})
	result, err = svc.History(ctx, ListPrivateMessagesRequest{TargetUserID: "user_bob"})
	if err != nil {
		t.Fatalf("history with failing source must degrade, got %v", err)
	}
	if result.Items[0].Reactions == nil || len(result.Items[0].Reactions) != 0 {
		t.Fatalf("expected degraded empty reactions, got %#v", result.Items[0].Reactions)
	}
}

func TestServiceRecentAttachesReactions(t *testing.T) {
	svc, _ := privateReactionsTestService()
	ctx := privateReactionsCtx("user_alice")
	sent, err := svc.Send(ctx, SendPrivateMessageRequest{TargetUserID: "user_bob", Type: "text", Content: "hi"})
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	svc.SetReactionsSource(&stubPrivateReactionsSource{agg: map[string][]Reaction{
		sent.MessageID: {{Emoji: "👍", Count: 1}},
	}})
	result, err := svc.Recent(ctx, "user_bob", 20, nil)
	if err != nil {
		t.Fatalf("recent: %v", err)
	}
	if len(result.Items) != 1 || len(result.Items[0].Reactions) != 1 || result.Items[0].Reactions[0].Emoji != "👍" {
		t.Fatalf("unexpected recent items: %#v", result.Items)
	}
}
