package moments

import (
	"context"
	"testing"

	"hotspot/backend-go/internal/platform/actor"
	platformauth "hotspot/backend-go/internal/platform/auth"
)

func TestServiceCoverageHelpers(t *testing.T) {
	// NewService 系列构造
	if svc := NewService(); svc == nil || svc.repo == nil {
		t.Fatal("NewService must wire repo")
	}
	if svc := NewServiceWithRepo(NewInMemoryRepository()); svc == nil || svc.repo == nil {
		t.Fatal("NewServiceWithRepo must wire repo")
	}
	if svc := NewServiceWithDeps(NewInMemoryRepository(), nil, nil); svc == nil || svc.repo == nil {
		t.Fatal("NewServiceWithDeps must wire repo")
	}
	if svc := NewServiceWithDepsAndImages(NewInMemoryRepository(), nil, nil, nil); svc == nil || svc.repo == nil {
		t.Fatal("NewServiceWithDepsAndImages must wire repo")
	}
	if svc := NewServiceWithDepsAndImagesAndConfig(NewInMemoryRepository(), nil, nil, nil, nil, nil); svc == nil || svc.repo == nil {
		t.Fatal("NewServiceWithDepsAndImagesAndConfig must wire repo")
	}

	// containsTag
	if !containsTag([]string{"a", "b"}, "a") {
		t.Fatal("existing tag must match")
	}
	if containsTag([]string{"a", "b"}, "c") {
		t.Fatal("missing tag must not match")
	}
	if containsTag(nil, "c") {
		t.Fatal("nil tags must not match")
	}

	// ensureMomentArrays / ensureOwnerMomentArrays：nil 数组填充为空切片
	item := &MomentItem{}
	ensureMomentArrays(item)
	if item.Images == nil || item.Tags == nil || item.Mentions == nil {
		t.Fatalf("arrays must be non-nil: %#v", item)
	}
	owner := &OwnerMomentItem{}
	ensureOwnerMomentArrays(owner)
	if owner.Images == nil || owner.Tags == nil || owner.Mentions == nil {
		t.Fatalf("owner arrays must be non-nil: %#v", owner)
	}

	// actorFromInfo / fallbackActor
	info := actor.Info{UserID: "u1", Nickname: "n1", AvatarURL: "http://a"}
	actor := actorFromInfo(info)
	if actor.UserID != "u1" || actor.Nickname != "n1" || actor.AvatarURL == nil || *actor.AvatarURL != "http://a" {
		t.Fatalf("actorFromInfo: %#v", actor)
	}
	fallback := fallbackActor("  u2  ")
	if fallback.UserID != "u2" || fallback.Nickname != "u2" {
		t.Fatalf("fallbackActor: %#v", fallback)
	}
	empty := fallbackActor("")
	if empty.UserID != "" || empty.Nickname != "" {
		t.Fatalf("empty fallbackActor: %#v", empty)
	}

	// emptyReactions / reactionsFor
	if len(emptyReactions()) != 0 {
		t.Fatal("emptyReactions must be empty")
	}
	if len(reactionsFor(nil)) != 0 {
		t.Fatal("nil reactions must be empty")
	}
	if got := reactionsFor([]Reaction{{Emoji: "👍"}}); len(got) != 1 {
		t.Fatalf("reactionsFor: %#v", got)
	}
}

func TestMemoryFetchMomentImageRows(t *testing.T) {
	repo := NewInMemoryRepository()
	rows, err := repo.FetchMomentImageRows(context.Background(), []string{"m1"})
	if err != nil || rows == nil || len(rows) != 0 {
		t.Fatalf("fetch rows: %#v err=%v", rows, err)
	}
}

func TestViewerUserIDHelper(t *testing.T) {
	if got := viewerUserID(context.Background()); got != "" {
		t.Fatalf("anonymous: %q", got)
	}
	ctx := withTestPrincipal(context.Background(), "u1")
	if got := viewerUserID(ctx); got != "u1" {
		t.Fatalf("principal: %q", got)
	}
}

func withTestPrincipal(ctx context.Context, userID string) context.Context {
	return platformauth.WithPrincipal(ctx, platformauth.Principal{UserID: userID, SessionID: "sess_" + userID})
}

func TestServiceUserListsAndDiscoveryHelpers(t *testing.T) {
	ctx := context.Background()
	repo := NewInMemoryRepository()
	svc := NewServiceWithRepo(repo)

	// 1. SetLocationProvider
	svc.SetLocationProvider(nil)

	// 2. SeedMomentsByUser & ListMomentsByAuthor
	repo.SeedMomentsByUser("u1", []MomentItem{
		{MomentID: "m1", User: UserInfo{UserID: "u1"}, PrivacyLevel: "public"},
		{MomentID: "m2", User: UserInfo{UserID: "u1"}, PrivacyLevel: "private"},
	})
	publicList, err := svc.ListMomentsByAuthor(ctx, "u1", 10, "", false)
	if err != nil || len(publicList.Items) != 1 {
		t.Fatalf("ListMomentsByAuthor public: %v len=%d", err, len(publicList.Items))
	}
	allList, err := svc.ListMomentsByAuthor(ctx, "u1", 10, "", true)
	if err != nil || len(allList.Items) != 2 {
		t.Fatalf("ListMomentsByAuthor all: %v len=%d", err, len(allList.Items))
	}

	// 3. SeedReactedMoment & ListReactedByUser
	repo.SeedReactedMoment("u2", MomentItem{
		MomentID:     "m3",
		User:         UserInfo{UserID: "u1"},
		PrivacyLevel: "public",
	})
	reactedList, err := svc.ListReactedByUser(ctx, "u2", "u2", 10, "")
	if err != nil || len(reactedList.Items) != 1 {
		t.Fatalf("ListReactedByUser: %v len=%d", err, len(reactedList.Items))
	}
}
