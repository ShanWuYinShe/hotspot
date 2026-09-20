package reactions

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"

	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/realtime"
	"hotspot/backend-go/internal/platform/types"
)

// stubTargetProvider serves a single target kind and reports existence by id.
type stubTargetProvider struct {
	kind   ReactTarget
	exists map[string]bool   // targetID -> exists
	owners map[string]string // targetID -> ownerUserID
}

func (s stubTargetProvider) ReactTarget() ReactTarget { return s.kind }

func (s stubTargetProvider) Resolve(_ context.Context, targetID string) (TargetContext, bool, error) {
	if !s.exists[targetID] {
		return TargetContext{}, false, nil
	}
	return TargetContext{OwnerUserID: s.owners[targetID], Summary: types.MomentSummary{MomentID: targetID}}, true, nil
}

// recordingNotifier captures Append* calls for assertion.
type recordingNotifier struct {
	mu                 sync.Mutex
	reactionAddedCalls []reactionAddedCall
}

type reactionAddedCall struct {
	targetUserID string
	targetType   string
	targetID     string
	emoji        string
}

func (n *recordingNotifier) AppendReactionAdded(_ context.Context, targetUserID string, _ types.UserInfo, targetType string, targetID string, emoji string) (string, error) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.reactionAddedCalls = append(n.reactionAddedCalls, reactionAddedCall{targetUserID, targetType, targetID, emoji})
	return "notif_1", nil
}

func ctxWithUser(userID string) context.Context {
	return platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: userID, SessionID: "s1"})
}

func TestSetReactAddsReaction(t *testing.T) {
	provider := stubTargetProvider{
		kind:   ReactTargetMoment,
		exists: map[string]bool{"m1": true},
		owners: map[string]string{"m1": "owner_alice"},
	}
	svc := NewServiceWithDeps(NewInMemoryRepository(), []TargetProvider{provider}, nil, nil)

	err := svc.Set(ctxWithUser("u_bob"), SetReactionRequest{
		TargetType: ReactTargetMoment, TargetID: "m1", Emoji: "👍", Active: true,
	})
	if err != nil {
		t.Fatalf("Set active=true: %v", err)
	}
	recs, err := svc.Aggregate(context.Background(), ReactTargetMoment, "m1")
	if err != nil {
		t.Fatalf("Aggregate: %v", err)
	}
	if len(recs) != 1 || recs[0].Emoji != "👍" || recs[0].Count != 1 {
		t.Fatalf("expected one 👍 reaction, got %+v", recs)
	}
}

func TestSetReactTargetNotFound(t *testing.T) {
	provider := stubTargetProvider{kind: ReactTargetMoment, exists: map[string]bool{}}
	svc := NewServiceWithDeps(NewInMemoryRepository(), []TargetProvider{provider}, nil, nil)

	err := svc.Set(ctxWithUser("u_bob"), SetReactionRequest{
		TargetType: ReactTargetMoment, TargetID: "missing", Emoji: "👍", Active: true,
	})
	if !errors.Is(err, ErrReactionTargetNotFound) {
		t.Fatalf("expected ErrReactionTargetNotFound, got %v", err)
	}
}

func TestSetReactSwitchingEmojiUpdatesInPlace(t *testing.T) {
	provider := stubTargetProvider{
		kind:   ReactTargetMoment,
		exists: map[string]bool{"m1": true},
		owners: map[string]string{"m1": "owner"},
	}
	svc := NewServiceWithDeps(NewInMemoryRepository(), []TargetProvider{provider}, nil, nil)
	ctx := ctxWithUser("u_bob")

	_ = svc.Set(ctx, SetReactionRequest{TargetType: ReactTargetMoment, TargetID: "m1", Emoji: "👍", Active: true})
	err := svc.Set(ctx, SetReactionRequest{TargetType: ReactTargetMoment, TargetID: "m1", Emoji: "❤️", Active: true})
	if err != nil {
		t.Fatalf("switch emoji: %v", err)
	}
	recs, _ := svc.Aggregate(context.Background(), ReactTargetMoment, "m1")
	if len(recs) != 1 || recs[0].Emoji != "❤️" || recs[0].Count != 1 {
		t.Fatalf("expected switched to ❤️ count=1, got %+v", recs)
	}
}

func TestSetReactIdempotentSameEmoji(t *testing.T) {
	provider := stubTargetProvider{kind: ReactTargetMoment, exists: map[string]bool{"m1": true}, owners: map[string]string{"m1": "o"}}
	svc := NewServiceWithDeps(NewInMemoryRepository(), []TargetProvider{provider}, nil, nil)
	ctx := ctxWithUser("u_bob")

	_ = svc.Set(ctx, SetReactionRequest{TargetType: ReactTargetMoment, TargetID: "m1", Emoji: "👍", Active: true})
	err := svc.Set(ctx, SetReactionRequest{TargetType: ReactTargetMoment, TargetID: "m1", Emoji: "👍", Active: true})
	if err != nil {
		t.Fatalf("idempotent Set: %v", err)
	}
	recs, _ := svc.Aggregate(context.Background(), ReactTargetMoment, "m1")
	if recs[0].Count != 1 {
		t.Fatalf("idempotent Set should keep count=1, got %d", recs[0].Count)
	}
}

func TestSetUnreactRemovesAndIsIdempotent(t *testing.T) {
	provider := stubTargetProvider{kind: ReactTargetMoment, exists: map[string]bool{"m1": true}, owners: map[string]string{"m1": "o"}}
	svc := NewServiceWithDeps(NewInMemoryRepository(), []TargetProvider{provider}, nil, nil)
	ctx := ctxWithUser("u_bob")

	_ = svc.Set(ctx, SetReactionRequest{TargetType: ReactTargetMoment, TargetID: "m1", Emoji: "👍", Active: true})
	if err := svc.Set(ctx, SetReactionRequest{TargetType: ReactTargetMoment, TargetID: "m1", Emoji: "👍", Active: false}); err != nil {
		t.Fatalf("unreact: %v", err)
	}
	recs, _ := svc.Aggregate(context.Background(), ReactTargetMoment, "m1")
	if len(recs) != 0 {
		t.Fatalf("expected no reactions after unreact, got %+v", recs)
	}
	// Second unreact is a no-op success.
	if err := svc.Set(ctx, SetReactionRequest{TargetType: ReactTargetMoment, TargetID: "m1", Emoji: "👍", Active: false}); err != nil {
		t.Fatalf("idempotent unreact: %v", err)
	}
}

func TestSetReactNotifiesOwnerByTargetKind(t *testing.T) {
	notif := &recordingNotifier{}
	// All target kinds (moment/comment) dispatch the unified reaction_added
	// notification (legacy moment_liked/comment_liked removed per contract).
	momentProv := stubTargetProvider{kind: ReactTargetMoment, exists: map[string]bool{"m1": true}, owners: map[string]string{"m1": "owner_m"}}
	commentProv := stubTargetProvider{kind: ReactTargetComment, exists: map[string]bool{"c1": true}, owners: map[string]string{"c1": "owner_c"}}
	svc := NewServiceWithDeps(NewInMemoryRepository(), []TargetProvider{momentProv, commentProv}, notif, nil)

	_ = svc.Set(ctxWithUser("u_bob"), SetReactionRequest{TargetType: ReactTargetMoment, TargetID: "m1", Emoji: "👍", Active: true})
	_ = svc.Set(ctxWithUser("u_bob"), SetReactionRequest{TargetType: ReactTargetComment, TargetID: "c1", Emoji: "👍", Active: true})

	notif.mu.Lock()
	defer notif.mu.Unlock()
	if len(notif.reactionAddedCalls) != 2 {
		t.Fatalf("expected two reaction_added notifications, got %+v", notif.reactionAddedCalls)
	}
	if notif.reactionAddedCalls[0].targetUserID != "owner_m" || notif.reactionAddedCalls[0].targetType != "moment" {
		t.Fatalf("unexpected first call: %+v", notif.reactionAddedCalls[0])
	}
	if notif.reactionAddedCalls[1].targetUserID != "owner_c" || notif.reactionAddedCalls[1].targetType != "comment" {
		t.Fatalf("unexpected second call: %+v", notif.reactionAddedCalls[1])
	}
}

func TestSetReactSkipsSelfNotification(t *testing.T) {
	notif := &recordingNotifier{}
	provider := stubTargetProvider{kind: ReactTargetMoment, exists: map[string]bool{"m1": true}, owners: map[string]string{"m1": "u_bob"}}
	svc := NewServiceWithDeps(NewInMemoryRepository(), []TargetProvider{provider}, notif, nil)

	_ = svc.Set(ctxWithUser("u_bob"), SetReactionRequest{TargetType: ReactTargetMoment, TargetID: "m1", Emoji: "👍", Active: true})
	notif.mu.Lock()
	defer notif.mu.Unlock()
	if len(notif.reactionAddedCalls) != 0 {
		t.Fatalf("self-reaction must not notify owner, got %+v", notif.reactionAddedCalls)
	}
}

func TestListPaginatesAndFiltersByEmoji(t *testing.T) {
	svc := NewServiceWithDeps(NewInMemoryRepository(), nil, nil, nil)
	// Add multiple reactions from several users.
	for _, u := range []string{"u1", "u2", "u3"} {
		if _, err := svc.repo.React(context.Background(), ReactTargetMoment, "m1", u, "👍"); err != nil {
			t.Fatalf("seed react %s: %v", u, err)
		}
	}
	if _, err := svc.repo.React(context.Background(), ReactTargetMoment, "m1", "u4", "❤️"); err != nil {
		t.Fatalf("seed react u4: %v", err)
	}

	page, err := svc.List(context.Background(), ListReactionsRequest{
		TargetType: ReactTargetMoment, TargetID: "m1", Emoji: "👍", Pagination: &PaginationParams{Limit: 2},
	})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(page.Items) != 2 || page.NextCursor == nil {
		t.Fatalf("expected 2 items + cursor, got %d items, cursor=%v", len(page.Items), page.NextCursor)
	}
	// Emoji filter excludes the ❤️ reaction.
	for _, it := range page.Items {
		if it.Emoji != "👍" {
			t.Fatalf("emoji filter leaked %q", it.Emoji)
		}
	}
}

func TestAggregateMarksReactedByMe(t *testing.T) {
	svc := NewServiceWithDeps(NewInMemoryRepository(), nil, nil, nil)
	if _, err := svc.repo.React(context.Background(), ReactTargetMoment, "m1", "u1", "👍"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.repo.React(context.Background(), ReactTargetMoment, "m1", "u2", "👍"); err != nil {
		t.Fatal(err)
	}

	// Anonymous viewer: reacted_by_me=false everywhere.
	anon, _ := svc.Aggregate(context.Background(), ReactTargetMoment, "m1")
	if anon[0].ReactedByMe {
		t.Fatalf("anonymous viewer must have reacted_by_me=false")
	}
	// u1 views: reacted_by_me=true on 👍.
	known, _ := svc.Aggregate(ctxWithUser("u1"), ReactTargetMoment, "m1")
	if !known[0].ReactedByMe {
		t.Fatalf("u1 should see reacted_by_me=true")
	}
}

func TestSetRejectsUnknownTargetType(t *testing.T) {
	svc := NewServiceWithDeps(NewInMemoryRepository(), nil, nil, nil)
	err := svc.Set(ctxWithUser("u"), SetReactionRequest{TargetType: "unknown", TargetID: "x", Emoji: "👍", Active: true})
	if !errors.Is(err, ErrReactionTargetNotFound) {
		t.Fatalf("expected ErrReactionTargetNotFound for bad target_type, got %v", err)
	}
}

func TestAggregateBatchHandlesEmpty(t *testing.T) {
	svc := NewServiceWithDeps(NewInMemoryRepository(), nil, nil, nil)
	out, err := svc.AggregateBatch(context.Background(), ReactTargetMoment, nil)
	if err != nil {
		t.Fatalf("AggregateBatch empty: %v", err)
	}
	if len(out) != 0 {
		t.Fatalf("expected empty map, got %+v", out)
	}
}

// realtimeSinkStub captures BroadcastToUser deliveries for hub-based tests.
type realtimeSinkStub struct{ events []realtime.Event }

func (s *realtimeSinkStub) Send(e realtime.Event) error { s.events = append(s.events, e); return nil }
func (s *realtimeSinkStub) Close() error                { return nil }

func TestUnreactBroadcastsToTargetOwner(t *testing.T) {
	hub := realtime.NewHub()
	realtime.SetDefaultHub(hub)
	defer realtime.ResetHub()
	ownerSink := &realtimeSinkStub{}
	actorSink := &realtimeSinkStub{}
	hub.RegisterUser("owner_alice", "sess_owner", ownerSink)
	hub.RegisterUser("u_bob", "sess_bob", actorSink)

	provider := stubTargetProvider{
		kind:   ReactTargetMoment,
		exists: map[string]bool{"m1": true},
		owners: map[string]string{"m1": "owner_alice"},
	}
	svc := NewServiceWithDeps(NewInMemoryRepository(), []TargetProvider{provider}, nil, nil)
	if _, err := svc.repo.React(context.Background(), ReactTargetMoment, "m1", "u_bob", "👍"); err != nil {
		t.Fatalf("seed react: %v", err)
	}
	if err := svc.Set(ctxWithUser("u_bob"), SetReactionRequest{TargetType: ReactTargetMoment, TargetID: "m1", Emoji: "👍", Active: false}); err != nil {
		t.Fatalf("unreact: %v", err)
	}
	// 契约:ReactionRemoved 推送给被反应资源作者,而非操作者本人。
	if len(ownerSink.events) != 1 || ownerSink.events[0].Type != EventTypeReactionRemoved {
		t.Fatalf("expected ReactionRemoved on owner sink, got %+v", ownerSink.events)
	}
	if len(actorSink.events) != 0 {
		t.Fatalf("ReactionRemoved must not be delivered to the actor, got %+v", actorSink.events)
	}
	payload := decodePayload(t, ownerSink.events[0].Payload)
	// 契约:反应移除不产生通知,payload 不含 notification_id。
	if _, has := payload["notify_id"]; has {
		t.Fatalf("ReactionRemoved payload must not carry notification_id, got %+v", payload)
	}
	if payload["target_type"] != "moment" || payload["target_id"] != "m1" || payload["emoji"] != "👍" {
		t.Fatalf("unexpected payload: %+v", payload)
	}
}

func TestUnreactSkipsBroadcastWhenTargetGone(t *testing.T) {
	hub := realtime.NewHub()
	realtime.SetDefaultHub(hub)
	defer realtime.ResetHub()
	ownerSink := &realtimeSinkStub{}
	hub.RegisterUser("owner_alice", "sess_owner", ownerSink)

	// 目标已删除:unreact 是幂等 no-op,不产生广播(没有作者可推)。
	provider := stubTargetProvider{kind: ReactTargetMoment, exists: map[string]bool{"m1": false}}
	svc := NewServiceWithDeps(NewInMemoryRepository(), []TargetProvider{provider}, nil, nil)
	if _, err := svc.repo.React(context.Background(), ReactTargetMoment, "m1", "u_bob", "👍"); err != nil {
		t.Fatalf("seed react: %v", err)
	}
	if err := svc.Set(ctxWithUser("u_bob"), SetReactionRequest{TargetType: ReactTargetMoment, TargetID: "m1", Emoji: "👍", Active: false}); err != nil {
		t.Fatalf("unreact on deleted target must succeed: %v", err)
	}
	if len(ownerSink.events) != 0 {
		t.Fatalf("no broadcast expected for deleted target, got %+v", ownerSink.events)
	}
}

// decodePayload 把 WS payload（现为类型化 struct）经 JSON round-trip 转回
// map，让既有键级断言继续验证「序列化后的形状」——那才是真正的线上契约。
// 注意数字经 JSON 解码为 float64。
func decodePayload(t *testing.T, p any) map[string]any {
	t.Helper()
	b, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("payload is not an object: %v (%#v)", err, p)
	}
	return m
}
