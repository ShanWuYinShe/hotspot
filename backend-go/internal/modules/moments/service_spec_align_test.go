package moments

import (
	"context"
	"errors"
	"testing"
	"time"
)

// momentsReactionAggregator records AggregateMoments calls and returns canned
// per-moment reaction aggregates.
type momentsReactionAggregator struct {
	calls   [][]string
	viewers []string
	result  map[string][]Reaction
	err     error
}

func (a *momentsReactionAggregator) AggregateMoments(_ context.Context, momentIDs []string, viewerID string) (map[string][]Reaction, error) {
	a.calls = append(a.calls, append([]string(nil), momentIDs...))
	a.viewers = append(a.viewers, viewerID)
	if a.err != nil {
		return nil, a.err
	}
	return a.result, nil
}

// momentsImageResolver resolves file_ids to assets and presigns keys.
type momentsImageResolver struct {
	assets map[string]ImageAsset
	err    error
}

func (r momentsImageResolver) ResolveMomentImages(_ context.Context, fileIDs []string, _ string) (map[string]ImageAsset, error) {
	if r.err != nil {
		return nil, r.err
	}
	out := make(map[string]ImageAsset, len(fileIDs))
	for _, id := range fileIDs {
		if a, ok := r.assets[id]; ok {
			out[id] = a
		}
	}
	return out, nil
}

func (r momentsImageResolver) PresignGet(_ context.Context, key string, _ time.Duration) (string, error) {
	return "https://cdn.example/" + key, nil
}

// --- fillReactions / fillReactionsBatch ---

func TestServiceFillReactionsFillsItem(t *testing.T) {
	agg := &momentsReactionAggregator{result: map[string][]Reaction{
		"m1": {{Emoji: "👍", Count: 3, ReactedByMe: true}},
	}}
	svc := NewServiceWithRepo(&stubMomentRepo{})
	svc.SetReactionAggregator(agg)

	item := MomentItem{MomentID: "m1"}
	svc.fillReactions(context.Background(), &item, "u1")
	if len(item.Reactions) != 1 || item.Reactions[0].Emoji != "👍" || item.Reactions[0].Count != 3 || !item.Reactions[0].ReactedByMe {
		t.Fatalf("expected filled reactions, got %+v", item.Reactions)
	}
	if len(agg.calls) != 1 || len(agg.calls[0]) != 1 || agg.calls[0][0] != "m1" || agg.viewers[0] != "u1" {
		t.Fatalf("unexpected aggregate call: %+v viewers=%+v", agg.calls, agg.viewers)
	}
}

func TestServiceFillReactionsSkipsWhenNoReactions(t *testing.T) {
	agg := &momentsReactionAggregator{result: map[string][]Reaction{}}
	svc := NewServiceWithRepo(&stubMomentRepo{})
	svc.SetReactionAggregator(agg)

	item := MomentItem{MomentID: "m1"}
	svc.fillReactions(context.Background(), &item, "u1")
	if len(item.Reactions) != 0 {
		t.Fatalf("expected no reactions, got %+v", item.Reactions)
	}
}

func TestServiceFillReactionsAggregatorErrorIsSwallowed(t *testing.T) {
	agg := &momentsReactionAggregator{err: errors.New("boom")}
	svc := NewServiceWithRepo(&stubMomentRepo{})
	svc.SetReactionAggregator(agg)

	item := MomentItem{MomentID: "m1"}
	svc.fillReactions(context.Background(), &item, "u1") // 不得 panic
	if len(item.Reactions) != 0 {
		t.Fatalf("expected no reactions on error, got %+v", item.Reactions)
	}
}

func TestServiceFillReactionsBatchFillsItems(t *testing.T) {
	agg := &momentsReactionAggregator{result: map[string][]Reaction{
		"m1": {{Emoji: "👍", Count: 1}},
		"m2": {{Emoji: "❤️", Count: 2}},
	}}
	svc := NewServiceWithRepo(&stubMomentRepo{})
	svc.SetReactionAggregator(agg)

	items := []MomentItem{{MomentID: "m1"}, {MomentID: "m2"}, {MomentID: "m3"}}
	svc.fillReactionsBatch(context.Background(), items, "u9")
	if len(items[0].Reactions) != 1 || items[0].Reactions[0].Emoji != "👍" {
		t.Fatalf("m1 reactions: %+v", items[0].Reactions)
	}
	if len(items[1].Reactions) != 1 || items[1].Reactions[0].Emoji != "❤️" {
		t.Fatalf("m2 reactions: %+v", items[1].Reactions)
	}
	if len(items[2].Reactions) != 0 {
		t.Fatalf("m3 has no reactions, got %+v", items[2].Reactions)
	}
	// 一次批量调用,ids 为全部 moment 列表。
	if len(agg.calls) != 1 || len(agg.calls[0]) != 3 || agg.calls[0][2] != "m3" || agg.viewers[0] != "u9" {
		t.Fatalf("unexpected batch call: %+v viewers=%+v", agg.calls, agg.viewers)
	}
}

func TestServiceFillReactionsBatchAggregatorErrorIsSwallowed(t *testing.T) {
	agg := &momentsReactionAggregator{err: errors.New("boom")}
	svc := NewServiceWithRepo(&stubMomentRepo{})
	svc.SetReactionAggregator(agg)

	items := []MomentItem{{MomentID: "m1"}}
	svc.fillReactionsBatch(context.Background(), items, "u1")
	if len(items[0].Reactions) != 0 {
		t.Fatalf("expected no reactions on error, got %+v", items[0].Reactions)
	}
}

func TestServiceFillReactionsNilAggregatorIsNoop(t *testing.T) {
	svc := NewServiceWithRepo(&stubMomentRepo{}) // reactions == nil
	item := MomentItem{MomentID: "m1"}
	svc.fillReactions(context.Background(), &item, "u1")
	items := []MomentItem{{MomentID: "m2"}}
	svc.fillReactionsBatch(context.Background(), items, "u1")
	if len(item.Reactions) != 0 || len(items[0].Reactions) != 0 {
		t.Fatalf("nil aggregator must be a no-op")
	}
}
