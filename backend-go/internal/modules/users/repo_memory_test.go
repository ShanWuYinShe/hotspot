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
