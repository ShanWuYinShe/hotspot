package reactions

import (
	"context"
	"errors"
	"sync"
	"testing"

	platformauth "hotspot/backend-go/internal/platform/auth"
)

// stubTargetProvider serves a single target kind and reports existence by id.
type stubTargetProvider struct {
	kind    ReactTarget
	exists  map[string]bool         // targetID -> exists
	owners  map[string]string       // targetID -> ownerUserID
}

func (s stubTargetProvider) ReactTarget() ReactTarget { return s.kind }

func (s stubTargetProvider) Resolve(_ context.Context, targetID string) (TargetContext, bool, error) {
	if !s.exists[targetID] {
		return TargetContext{}, false, nil
	}
	return TargetContext{OwnerUserID: s.owners[targetID], Summary: map[string]any{"moment_id": targetID}}, true, nil
}

// recordingNotifier captures Append* calls for assertion.
type recordingNotifier struct {
	mu                 sync.Mutex
	momentLiked        []string  // targetUserIDs
	commentLiked       []string
	reactionAddedCalls []reactionAddedCall
}

type reactionAddedCall struct {
	targetUserID string
	targetType   string
	targetID     string
	emoji        string
}

func (n *recordingNotifier) AppendMomentLiked(_ context.Context, targetUserID string, _ map[string]any, _ string, _ map[string]any) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.momentLiked = append(n.momentLiked, targetUserID)
	return nil
}

func (n *recordingNotifier) AppendCommentLiked(_ context.Context, targetUserID string, _ map[string]any, _ string, _ string, _ string, _ map[string]any) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.commentLiked = append(n.commentLiked, targetUserID)
	return nil
}

func (n *recordingNotifier) AppendReactionAdded(_ context.Context, targetUserID string, _ map[string]any, targetType string, targetID string, emoji string) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.reactionAddedCalls = append(n.reactionAddedCalls, reactionAddedCall{targetUserID, targetType, targetID, emoji})
	return nil
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
	// moment target -> AppendMomentLiked
	momentProv := stubTargetProvider{kind: ReactTargetMoment, exists: map[string]bool{"m1": true}, owners: map[string]string{"m1": "owner_m"}}
	commentProv := stubTargetProvider{kind: ReactTargetComment, exists: map[string]bool{"c1": true}, owners: map[string]string{"c1": "owner_c"}}
	svc := NewServiceWithDeps(NewInMemoryRepository(), []TargetProvider{momentProv, commentProv}, notif, nil)

	_ = svc.Set(ctxWithUser("u_bob"), SetReactionRequest{TargetType: ReactTargetMoment, TargetID: "m1", Emoji: "👍", Active: true})
	_ = svc.Set(ctxWithUser("u_bob"), SetReactionRequest{TargetType: ReactTargetComment, TargetID: "c1", Emoji: "👍", Active: true})

	notif.mu.Lock()
	defer notif.mu.Unlock()
	if len(notif.momentLiked) != 1 || notif.momentLiked[0] != "owner_m" {
		t.Fatalf("expected one moment_liked to owner_m, got %+v", notif.momentLiked)
	}
	if len(notif.commentLiked) != 1 || notif.commentLiked[0] != "owner_c" {
		t.Fatalf("expected one comment_liked to owner_c, got %+v", notif.commentLiked)
	}
}

func TestSetReactSkipsSelfNotification(t *testing.T) {
	notif := &recordingNotifier{}
	provider := stubTargetProvider{kind: ReactTargetMoment, exists: map[string]bool{"m1": true}, owners: map[string]string{"m1": "u_bob"}}
	svc := NewServiceWithDeps(NewInMemoryRepository(), []TargetProvider{provider}, notif, nil)

	_ = svc.Set(ctxWithUser("u_bob"), SetReactionRequest{TargetType: ReactTargetMoment, TargetID: "m1", Emoji: "👍", Active: true})
	notif.mu.Lock()
	defer notif.mu.Unlock()
	if len(notif.momentLiked) != 0 {
		t.Fatalf("self-reaction must not notify owner, got %+v", notif.momentLiked)
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
	if len(page.Items) != 2 || page.NextCursor == "" {
		t.Fatalf("expected 2 items + cursor, got %d items, cursor=%q", len(page.Items), page.NextCursor)
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
