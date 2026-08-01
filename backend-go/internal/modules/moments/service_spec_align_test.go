package moments

import (
	"context"
	"errors"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/actor"
	"hotspot/backend-go/internal/platform/appconfig"
	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/geo"
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

// momentsLocationProvider serves the reference_point fallback level 2.
type momentsLocationProvider struct {
	point geo.Point
	found bool
}

func (p momentsLocationProvider) RecentLocation(_ context.Context, _ string) (geo.Point, bool) {
	return p.point, p.found
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

// --- resolveReferencePoint 回退链(用户位置 → default_location → geo 常量) ---

func TestMomentsResolveReferencePointUserRecentLocation(t *testing.T) {
	svc := NewServiceWithRepo(&stubMomentRepo{})
	svc.SetLocationProvider(momentsLocationProvider{point: geo.Point{Latitude: 3.5, Longitude: 4.5}, found: true})

	pt := svc.resolveReferencePoint(momentCtx("u1"))
	if pt.Latitude != 3.5 || pt.Longitude != 4.5 {
		t.Fatalf("expected user recent location, got %+v", pt)
	}
}

func TestMomentsResolveReferencePointDefaultLocation(t *testing.T) {
	source := appconfig.NewStaticSource(config.Config{})
	_, _ = source.Update(context.Background(), func(cfg appconfig.AppConfig) appconfig.AppConfig {
		cfg.Defaults.DefaultLocation.Point = appconfig.GeoPoint{Latitude: 5.5, Longitude: 6.5}
		return cfg
	})
	svc := NewServiceWithDepsAndImagesAndConfig(&stubMomentRepo{}, nil, nil, nil, source, nil)

	pt := svc.resolveReferencePoint(momentCtx("u1"))
	if pt.Latitude != 5.5 || pt.Longitude != 6.5 {
		t.Fatalf("expected appConfig default_location, got %+v", pt)
	}
}

func TestMomentsResolveReferencePointGeoFallback(t *testing.T) {
	svc := NewServiceWithRepo(&stubMomentRepo{}) // 无 appConfig

	pt := svc.resolveReferencePoint(momentCtx("u1"))
	if pt.Latitude != geo.DefaultFallbackLatitude || pt.Longitude != geo.DefaultFallbackLongitude {
		t.Fatalf("expected geo default fallback, got %+v", pt)
	}
}

// --- validateImageOwnership(Create 路径) ---

func TestServiceCreateValidatesImageOwnership(t *testing.T) {
	resolver := momentsImageResolver{assets: map[string]ImageAsset{
		"f1": {FileID: "f1", StorageKey: "sk/1"},
	}}
	svc := NewServiceWithDepsAndImagesAndConfig(&stubMomentRepo{}, nil, nil, resolver, nil, nil)

	_, err := svc.Create(momentCtx("u1"), CreateMomentRequest{Description: "hi", ImageFileIDs: []string{"f1"}})
	if err != nil {
		t.Fatalf("owned image must pass ownership validation: %v", err)
	}
}

func TestServiceCreateRejectsUnownedImages(t *testing.T) {
	resolver := momentsImageResolver{err: errors.New("not owned")}
	svc := NewServiceWithDepsAndImagesAndConfig(&stubMomentRepo{}, nil, nil, resolver, nil, nil)

	_, err := svc.Create(momentCtx("u1"), CreateMomentRequest{Description: "hi", ImageFileIDs: []string{"f1"}})
	if !errors.Is(err, ErrImagesNotOwned) {
		t.Fatalf("expected ErrImagesNotOwned, got %v", err)
	}
}

func TestServiceCreateAcceptsEmptyImageList(t *testing.T) {
	svc := NewServiceWithRepo(&stubMomentRepo{})
	_, err := svc.Create(momentCtx("u1"), CreateMomentRequest{Description: "no images"})
	if err != nil {
		t.Fatalf("empty image list must pass: %v", err)
	}
}

func TestServiceCreateRejectsTooManyImages(t *testing.T) {
	source := appconfig.NewStaticSource(config.Config{})
	_, _ = source.Update(context.Background(), func(cfg appconfig.AppConfig) appconfig.AppConfig {
		cfg.Limits.MaxMomentImages = 1
		return cfg
	})
	svc := NewServiceWithDepsAndImagesAndConfig(&stubMomentRepo{}, nil, nil, nil, source, nil)

	_, err := svc.Create(momentCtx("u1"), CreateMomentRequest{Description: "hi", ImageFileIDs: []string{"f1", "f2"}})
	if !errors.Is(err, ErrTooManyImages) {
		t.Fatalf("expected ErrTooManyImages, got %v", err)
	}
}

// --- buildMomentActor / actorFromInfo ---

type stubActorProvider struct {
	infos map[string]actor.Info
	err   error
}

func (s stubActorProvider) GetActorInfo(_ context.Context, userID string) (actor.Info, error) {
	if s.err != nil {
		return actor.Info{}, s.err
	}
	if info, ok := s.infos[userID]; ok {
		return info, nil
	}
	return actor.Info{UserID: userID, Nickname: userID}, nil
}

func TestServiceBuildMomentActorWithProvider(t *testing.T) {
	svc := NewServiceWithDeps(&stubMomentRepo{}, nil, stubActorProvider{infos: map[string]actor.Info{
		"u1": {UserID: "u1", Nickname: "Alice", AvatarURL: "https://a.example/alice.png"},
	}})

	a, err := svc.buildMomentActor(context.Background(), "u1")
	if err != nil {
		t.Fatalf("buildMomentActor: %v", err)
	}
	if a["user_id"] != "u1" || a["nickname"] != "Alice" || a["avatar_url"] != "https://a.example/alice.png" {
		t.Fatalf("unexpected actor: %+v", a)
	}
}

func TestServiceBuildMomentActorProviderError(t *testing.T) {
	svc := NewServiceWithDeps(&stubMomentRepo{}, nil, stubActorProvider{err: errors.New("boom")})

	_, err := svc.buildMomentActor(context.Background(), "u1")
	if err == nil {
		t.Fatal("expected error from provider")
	}
}

func TestServiceBuildMomentActorFallback(t *testing.T) {
	svc := NewServiceWithRepo(&stubMomentRepo{}) // users == nil

	a, err := svc.buildMomentActor(context.Background(), "u1")
	if err != nil {
		t.Fatalf("fallback actor must not error: %v", err)
	}
	if a["user_id"] != "u1" || a["nickname"] != "u1" {
		t.Fatalf("expected fallback actor, got %+v", a)
	}
}

func TestActorFromInfo(t *testing.T) {
	withAvatar := actorFromInfo(actor.Info{UserID: "u1", Nickname: "Alice", AvatarURL: "https://a"})
	if withAvatar["avatar_url"] != "https://a" {
		t.Fatalf("avatar_url must be present: %+v", withAvatar)
	}
	withoutAvatar := actorFromInfo(actor.Info{UserID: "u2", Nickname: "Bob"})
	if _, has := withoutAvatar["avatar_url"]; has {
		t.Fatalf("avatar_url must be omitted when empty: %+v", withoutAvatar)
	}
	if withoutAvatar["user_id"] != "u2" || withoutAvatar["nickname"] != "Bob" {
		t.Fatalf("unexpected actor: %+v", withoutAvatar)
	}
}
