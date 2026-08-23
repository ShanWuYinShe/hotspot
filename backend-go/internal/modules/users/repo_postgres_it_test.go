package users

import (
	"context"
	"testing"

	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/testdb"
)

func TestPostgresIT_PutAndGetProfile(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		repo := NewPostgresRepository(txFactory)
		userID := testdb.InsertUser(t, txFactory, "u_profile_test", "TestUser")
		nowTS := nowRFC3339()

		profile := MyProfile{
			UserID:     userID,
			Nickname:   "TestUser",
			UserType:   UserTypeNormal,
			Location:   Location{LocationName: "Beijing", Point: GeoPoint{Latitude: 39.9, Longitude: 116.4}},
			LastOnline: nowTS,
			CreatedAt:  nowTS,
		}
		if err := repo.PutProfile(ctx, profile); err != nil {
			t.Fatalf("PutProfile: %v", err)
		}

		got, found, err := repo.GetProfile(ctx, userID)
		if err != nil {
			t.Fatalf("GetProfile: %v", err)
		}
		if !found {
			t.Fatal("expected found=true")
		}
		if got.UserID != userID {
			t.Fatalf("user_id: got %q, want %q", got.UserID, userID)
		}
		if got.Nickname != "TestUser" {
			t.Fatalf("nickname: got %q", got.Nickname)
		}
	})
}

func TestPostgresIT_GetProfileNotFound(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		repo := NewPostgresRepository(txFactory)
		_, found, err := repo.GetProfile(ctx, "nonexistent_user")
		if err != nil {
			t.Fatalf("GetProfile: %v", err)
		}
		if found {
			t.Fatal("expected found=false for nonexistent user")
		}
	})
}

func TestPostgresIT_PutAndGetSettings(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		repo := NewPostgresRepository(txFactory)
		userID := testdb.InsertUser(t, txFactory, "u_settings_test", "SettingsUser")

		settings := UserSettings{
			PrivacyLevel:     "public",
			LocationVisible:  true,
			PresenceVisible:  false,
			AllowFollow:      true,
			AllowPrivateChat: ChatPermFriends,
			Searchable:       true,
		}
		if err := repo.PutSettings(ctx, userID, settings); err != nil {
			t.Fatalf("PutSettings: %v", err)
		}

		got, found, err := repo.GetSettings(ctx, userID)
		if err != nil {
			t.Fatalf("GetSettings: %v", err)
		}
		if !found {
			t.Fatal("expected found=true")
		}
		if got.AllowPrivateChat != ChatPermFriends {
			t.Fatalf("allow_private_chat: got %q, want %q", got.AllowPrivateChat, ChatPermFriends)
		}
		if got.PresenceVisible {
			t.Fatal("expected presence_visible=false")
		}
	})
}

func TestPostgresIT_FollowAndRelationship(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		repo := NewPostgresRepository(txFactory)
		alice := testdb.InsertUser(t, txFactory, "u_follow_alice", "Alice")
		bob := testdb.InsertUser(t, txFactory, "u_follow_bob", "Bob")

		if err := repo.Follow(ctx, alice, bob, FollowSourceProfile, ""); err != nil {
			t.Fatalf("Follow: %v", err)
		}

		rel, err := repo.Relationship(ctx, alice, bob)
		if err != nil {
			t.Fatalf("Relationship: %v", err)
		}
		if !rel.IsFollowing {
			t.Fatal("expected is_following=true")
		}
		if rel.IsFollowedBy {
			t.Fatal("expected is_followed_by=false")
		}

		if err := repo.Unfollow(ctx, alice, bob); err != nil {
			t.Fatalf("Unfollow: %v", err)
		}
		rel, err = repo.Relationship(ctx, alice, bob)
		if err != nil {
			t.Fatalf("Relationship after unfollow: %v", err)
		}
		if rel.IsFollowing {
			t.Fatal("expected is_following=false after unfollow")
		}
	})
}

func TestPostgresIT_BlockAndUnblock(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		repo := NewPostgresRepository(txFactory)
		alice := testdb.InsertUser(t, txFactory, "u_block_alice", "Alice")
		bob := testdb.InsertUser(t, txFactory, "u_block_bob", "Bob")

		if err := repo.Block(ctx, alice, bob, "spam"); err != nil {
			t.Fatalf("Block: %v", err)
		}

		rel, err := repo.Relationship(ctx, alice, bob)
		if err != nil {
			t.Fatalf("Relationship: %v", err)
		}
		if !rel.IsBlocked {
			t.Fatal("expected is_blocked=true")
		}

		if err := repo.Unblock(ctx, alice, bob); err != nil {
			t.Fatalf("Unblock: %v", err)
		}
		rel, err = repo.Relationship(ctx, alice, bob)
		if err != nil {
			t.Fatalf("Relationship after unblock: %v", err)
		}
		if rel.IsBlocked {
			t.Fatal("expected is_blocked=false after unblock")
		}
	})
}

func TestPostgresIT_UserExists(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		repo := NewPostgresRepository(txFactory)
		userID := testdb.InsertUser(t, txFactory, "u_exists_test", "ExistsUser")

		exists, err := repo.UserExists(ctx, userID)
		if err != nil {
			t.Fatalf("UserExists: %v", err)
		}
		if !exists {
			t.Fatal("expected exists=true")
		}

		exists, err = repo.UserExists(ctx, "nonexistent_user")
		if err != nil {
			t.Fatalf("UserExists nonexistent: %v", err)
		}
		if exists {
			t.Fatal("expected exists=false for nonexistent user")
		}
	})
}

func TestPostgresIT_ListFriendsReturnsMutualFollows(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		repo := NewPostgresRepository(txFactory)
		alice := testdb.InsertUser(t, txFactory, "u_lf_alice", "Alice")
		bob := testdb.InsertUser(t, txFactory, "u_lf_bob", "Bob")
		carol := testdb.InsertUser(t, txFactory, "u_lf_carol", "Carol")

		// Alice follows Bob, Bob follows Alice → mutual (friend)
		if err := repo.Follow(ctx, alice, bob, FollowSourceProfile, ""); err != nil {
			t.Fatalf("Follow alice->bob: %v", err)
		}
		if err := repo.Follow(ctx, bob, alice, FollowSourceProfile, ""); err != nil {
			t.Fatalf("Follow bob->alice: %v", err)
		}
		// Alice follows Carol but Carol does not follow back → NOT a friend
		if err := repo.Follow(ctx, alice, carol, FollowSourceProfile, ""); err != nil {
			t.Fatalf("Follow alice->carol: %v", err)
		}

		friends, err := repo.ListFriends(ctx, alice, 10, "")
		if err != nil {
			t.Fatalf("ListFriends: %v", err)
		}
		if len(friends.Items) != 1 {
			t.Fatalf("expected 1 friend (mutual only), got %d", len(friends.Items))
		}
		if friends.Items[0].User.User.UserID != bob {
			t.Fatalf("expected friend bob, got %q", friends.Items[0].User.User.UserID)
		}
		if friends.Items[0].Remark == nil || *friends.Items[0].Remark != "" {
			t.Fatalf("expected empty remark, got %q", *friends.Items[0].Remark)
		}
	})
}

func TestPostgresIT_ListFollowingAndFollowers(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		repo := NewPostgresRepository(txFactory)
		alice := testdb.InsertUser(t, txFactory, "u_lfol_alice", "Alice")
		bob := testdb.InsertUser(t, txFactory, "u_lfol_bob", "Bob")

		if err := repo.Follow(ctx, alice, bob, FollowSourceProfile, ""); err != nil {
			t.Fatalf("Follow: %v", err)
		}

		following, err := repo.ListFollowing(ctx, alice, 10, "")
		if err != nil {
			t.Fatalf("ListFollowing: %v", err)
		}
		if len(following.Items) != 1 {
			t.Fatalf("expected 1 following, got %d", len(following.Items))
		}
		if following.Items[0].User.User.UserID != bob {
			t.Fatalf("expected following bob, got %q", following.Items[0].User.User.UserID)
		}

		followers, err := repo.ListFollowers(ctx, bob, 10, "")
		if err != nil {
			t.Fatalf("ListFollowers: %v", err)
		}
		if len(followers.Items) != 1 {
			t.Fatalf("expected 1 follower, got %d", len(followers.Items))
		}
		if followers.Items[0].User.User.UserID != alice {
			t.Fatalf("expected follower alice, got %q", followers.Items[0].User.User.UserID)
		}
	})
}

func TestPostgresIT_ListFollowingMutualFlag(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		repo := NewPostgresRepository(txFactory)
		alice := testdb.InsertUser(t, txFactory, "u_mutual_alice", "Alice")
		bob := testdb.InsertUser(t, txFactory, "u_mutual_bob", "Bob")

		if err := repo.Follow(ctx, alice, bob, FollowSourceProfile, ""); err != nil {
			t.Fatalf("Follow alice->bob: %v", err)
		}
		if err := repo.Follow(ctx, bob, alice, FollowSourceProfile, ""); err != nil {
			t.Fatalf("Follow bob->alice: %v", err)
		}

		// RelatedUserItem 已删除 is_mutual（契约变更）：互相关注场景仍应正常返回列表
		following, err := repo.ListFollowing(ctx, alice, 10, "")
		if err != nil {
			t.Fatalf("ListFollowing: %v", err)
		}
		if len(following.Items) != 1 {
			t.Fatalf("expected 1 following, got %d", len(following.Items))
		}
	})
}

func TestPostgresIT_ListBlockedUsers(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		repo := NewPostgresRepository(txFactory)
		alice := testdb.InsertUser(t, txFactory, "u_blk_alice", "Alice")
		bob := testdb.InsertUser(t, txFactory, "u_blk_bob", "Bob")

		if err := repo.Block(ctx, alice, bob, "harassment"); err != nil {
			t.Fatalf("Block: %v", err)
		}

		blocked, err := repo.ListBlocked(ctx, alice, 10, "")
		if err != nil {
			t.Fatalf("ListBlocked: %v", err)
		}
		if len(blocked.Items) != 1 {
			t.Fatalf("expected 1 blocked user, got %d", len(blocked.Items))
		}
		if blocked.Items[0].User.UserID != bob {
			t.Fatalf("expected blocked user bob, got %q", blocked.Items[0].User.UserID)
		}
		if blocked.Items[0].Reason == nil || *blocked.Items[0].Reason != "harassment" {
			t.Fatalf("expected reason 'harassment', got %v", blocked.Items[0].Reason)
		}
	})
}

func TestPostgresIT_CountUserStatsViaGetProfile(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		repo := NewPostgresRepository(txFactory)
		alice := testdb.InsertUser(t, txFactory, "u_stats_alice", "Alice")
		bob := testdb.InsertUser(t, txFactory, "u_stats_bob", "Bob")
		carol := testdb.InsertUser(t, txFactory, "u_stats_carol", "Carol")

		// Create friendships: alice<->bob (mutual), alice->carol (one-way)
		repo.Follow(ctx, alice, bob, FollowSourceProfile, "")
		repo.Follow(ctx, bob, alice, FollowSourceProfile, "")
		repo.Follow(ctx, alice, carol, FollowSourceProfile, "")

		profile, found, err := repo.GetProfile(ctx, alice)
		if err != nil {
			t.Fatalf("GetProfile: %v", err)
		}
		if !found {
			t.Fatal("expected found=true")
		}

		// FriendsCount: mutual only → 1 (bob)
		if profile.Stats.FriendsCount != 1 {
			t.Fatalf("friends_count: got %d, want 1", profile.Stats.FriendsCount)
		}
		// FollowingCount: alice follows bob + carol → 2
		if profile.Stats.FollowingCount != 2 {
			t.Fatalf("following_count: got %d, want 2", profile.Stats.FollowingCount)
		}
		// FollowersCount: bob follows alice → 1
		if profile.Stats.FollowersCount != 1 {
			t.Fatalf("followers_count: got %d, want 1", profile.Stats.FollowersCount)
		}
	})
}

func TestPostgresIT_ListFollowingPagination(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		repo := NewPostgresRepository(txFactory)
		alice := testdb.InsertUser(t, txFactory, "u_page_alice", "Alice")
		for i := 0; i < 3; i++ {
			testdb.InsertUser(t, txFactory, "u_page_target_"+string(rune('A'+i)), "User"+string(rune('A'+i)))
		}

		// Follow all 3 targets
		for i := 0; i < 3; i++ {
			target := "u_page_target_" + string(rune('A'+i))
			if err := repo.Follow(ctx, alice, target, FollowSourceProfile, ""); err != nil {
				t.Fatalf("Follow %s: %v", target, err)
			}
		}

		// Page 1: limit 2
		page1, err := repo.ListFollowing(ctx, alice, 2, "")
		if err != nil {
			t.Fatalf("ListFollowing page1: %v", err)
		}
		if len(page1.Items) != 2 {
			t.Fatalf("page1: expected 2 items, got %d", len(page1.Items))
		}
		if page1.NextCursor == nil {
			t.Fatal("page1: expected non-empty next_cursor")
		}

		// Page 2: should have 1 item
		page2, err := repo.ListFollowing(ctx, alice, 2, *page1.NextCursor)
		if err != nil {
			t.Fatalf("ListFollowing page2: %v", err)
		}
		if len(page2.Items) != 1 {
			t.Fatalf("page2: expected 1 item, got %d", len(page2.Items))
		}
		if page2.NextCursor != nil {
			t.Fatalf("page2: expected empty next_cursor (last page), got %q", *page2.NextCursor)
		}
	})
}

func TestPostgresIT_CommonGroupCount(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		repo := NewPostgresRepository(txFactory)
		alice := testdb.InsertUser(t, txFactory, "u_cgc_alice", "Alice")
		bob := testdb.InsertUser(t, txFactory, "u_cgc_bob", "Bob")

		count, err := repo.CommonGroupCount(ctx, alice, bob)
		if err != nil {
			t.Fatalf("CommonGroupCount: %v", err)
		}
		if count != 0 {
			t.Fatalf("expected 0 common groups, got %d", count)
		}
	})
}
