package users

import (
	"context"
	"testing"
)

// validProfile is a minimal MyProfile used by the in-memory repo tests.
func validProfile(userID string) MyProfile {
	return MyProfile{
		UserID:   userID,
		Nickname: "nick-" + userID,
		Tags:     []string{"go", "music"},
		Location: Location{Point: GeoPoint{Latitude: 35.0, Longitude: 112.0}, LocationName: "here"},
	}
}

// TestInMemoryPutProfile_StoresProfileAndTags verifies the in-memory PutProfile
// keeps the profile and its tags together. The postgres path now wraps the
// upsert + tag delete + batch tag insert in a single transaction (so a mid-step
// failure can't wipe tags); this guards the tx-transparent in-memory mirror.
func TestInMemoryPutProfile_StoresProfileAndTags(t *testing.T) {
	repo := NewInMemoryRepository()
	if err := repo.PutProfile(context.Background(), validProfile("u1")); err != nil {
		t.Fatalf("PutProfile failed: %v", err)
	}
	got, ok, err := repo.GetProfile(context.Background(), "u1")
	if err != nil || !ok {
		t.Fatalf("GetProfile: ok=%v err=%v", ok, err)
	}
	if got.Nickname != "nick-u1" {
		t.Fatalf("expected nickname nick-u1, got %s", got.Nickname)
	}
	if len(got.Tags) != 2 || got.Tags[0] != "go" || got.Tags[1] != "music" {
		t.Fatalf("tags not stored with profile, got %#v", got.Tags)
	}
}

// TestInMemoryPutProfile_ReplacesTagsWholesale verifies that a second
// PutProfile with a different tag set fully replaces the old tags (the
// postgres path does delete-then-insert inside its transaction for the same
// effect).
func TestInMemoryPutProfile_ReplacesTagsWholesale(t *testing.T) {
	repo := NewInMemoryRepository()
	if err := repo.PutProfile(context.Background(), validProfile("u2")); err != nil {
		t.Fatalf("first PutProfile: %v", err)
	}
	updated := validProfile("u2")
	updated.Tags = []string{"rust", "hiking", "coffee"}
	if err := repo.PutProfile(context.Background(), updated); err != nil {
		t.Fatalf("second PutProfile: %v", err)
	}
	got, _, _ := repo.GetProfile(context.Background(), "u2")
	if len(got.Tags) != 3 {
		t.Fatalf("expected 3 tags after replace, got %#v", got.Tags)
	}
	for _, old := range []string{"go", "music"} {
		for _, tag := range got.Tags {
			if tag == old {
				t.Fatalf("stale tag %q still present after replace", old)
			}
		}
	}
}

// TestInMemoryBlockCascade_ClearsBothFollows verifies BlockCascade atomically
// blocks and clears follows in both directions (spec api_spec_user.md block），
// guarding the tx-wrapped postgres mirror.
func TestInMemoryBlockCascade_ClearsBothFollows(t *testing.T) {
	repo := NewInMemoryRepository()
	ctx := context.Background()
	if _, err := repo.Follow(ctx, "user_a", "user_b", FollowSourceProfile, ""); err != nil {
		t.Fatalf("follow a->b: %v", err)
	}
	if _, err := repo.Follow(ctx, "user_b", "user_a", FollowSourceProfile, ""); err != nil {
		t.Fatalf("follow b->a: %v", err)
	}
	if err := repo.BlockCascade(ctx, "user_a", "user_b", "spam"); err != nil {
		t.Fatalf("block cascade: %v", err)
	}
	relAB, err := repo.Relationship(ctx, "user_a", "user_b")
	if err != nil {
		t.Fatalf("relationship a->b: %v", err)
	}
	if !relAB.IsBlocked || relAB.IsFollowing {
		t.Fatalf("expected blocked and unfollowed a->b: %#v", relAB)
	}
	relBA, err := repo.Relationship(ctx, "user_b", "user_a")
	if err != nil {
		t.Fatalf("relationship b->a: %v", err)
	}
	if relBA.IsBlocked || relBA.IsFollowing {
		t.Fatalf("expected b->a not blocked but unfollowed: %#v", relBA)
	}
}

func TestInMemoryMutualFriends(t *testing.T) {
	repo := NewInMemoryRepository()
	ctx := context.Background()
	mutual := func(a, b string) {
		t.Helper()
		if _, err := repo.Follow(ctx, a, b, FollowSourceProfile, ""); err != nil {
			t.Fatalf("follow %s->%s: %v", a, b, err)
		}
		if _, err := repo.Follow(ctx, b, a, FollowSourceProfile, ""); err != nil {
			t.Fatalf("follow %s->%s: %v", b, a, err)
		}
	}
	mutual("viewer", "m1")
	mutual("viewer", "m2")
	mutual("target", "m1")
	mutual("target", "m2")
	// 单向关注不成共同好友。
	if _, err := repo.Follow(ctx, "viewer", "stranger", FollowSourceProfile, ""); err != nil {
		t.Fatalf("follow stranger: %v", err)
	}
	total, preview, err := repo.MutualFriends(ctx, "viewer", "target", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 2 {
		t.Fatalf("expected total=2, got %d", total)
	}
	if len(preview) != 1 {
		t.Fatalf("expected 1 preview, got %d", len(preview))
	}
	if preview[0].UserID != "m1" && preview[0].UserID != "m2" {
		t.Fatalf("unexpected preview: %+v", preview[0])
	}
}

func TestRepoNearbyMomentNotifyTargetsFilters(t *testing.T) {
	repo := NewInMemoryRepository()
	ctx := context.Background()

	// u1：作者（排除）；u2：圈内开启通知；u3：圈内未开启；u4：圈外开启；u5：已注销
	usersIn := map[string]struct {
		lat, lng float64
		notify   bool
		deleted  bool
	}{
		"u1": {35.0976, 112.6124, false, false},
		"u2": {35.0980, 112.6130, true, false},
		"u3": {35.0980, 112.6130, false, false},
		"u4": {40.0000, 116.0000, true, false},
		"u5": {35.0980, 112.6130, true, true},
	}
	for uid, u := range usersIn {
		p := MyProfile{UserID: uid, IsDeleted: u.deleted}
		p.Location.Point.Latitude = u.lat
		p.Location.Point.Longitude = u.lng
		if err := repo.PutProfile(ctx, p); err != nil {
			t.Fatalf("put profile %s: %v", uid, err)
		}
		if err := repo.PutSettings(ctx, uid, UserSettings{NearbyMomentNotify: u.notify}); err != nil {
			t.Fatalf("put settings %s: %v", uid, err)
		}
	}

	targets, err := repo.NearbyMomentNotifyTargets(ctx, 35.0976, 112.6124, 5000, "u1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	gots := map[string]bool{}
	for _, t := range targets {
		gots[t.UserID] = true
	}
	if len(targets) != 1 || !gots["u2"] {
		t.Fatalf("应只返回 u2, got %#v", gots)
	}
}
