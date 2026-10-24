package users

import (
	"context"
	"errors"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/testdb"
)

// usersContractState 契约用例上下文：两实现都需要预置用户行（PG 侧为 users 表
// 外键，内存侧为画像 map），准备逻辑放在各 ContractImpl.New 内。
type usersContractState struct {
	ctx     context.Context
	viewer  string
	targets []string
}

// TestUsersRelationshipContract 接入双实现契约一致性基建（users 模块）：
// 关注/取关/拉黑/级联/好友/粉丝列表与 keyset 分页的行为必须与真实库实现一致。
// 排列组合覆盖状态机与列表排序；游标分页只断言「不重不漏 + 页大小」，不绑定
// 同一秒内的时间并列顺序（两实现 tie-breaker 均为 user_id ASC）。
func TestUsersRelationshipContract(t *testing.T) {
	viewer := "it_users_contract_viewer"
	targets := []string{"it_users_contract_t01", "it_users_contract_t02", "it_users_contract_t03", "it_users_contract_t04"}

	impls := []testdb.ContractImpl[Repository, usersContractState]{
		{
			Name: "memory",
			New: func(t *testing.T) (Repository, usersContractState, func()) {
				repo := NewInMemoryRepository()
				ctx := context.Background()
				lastOnline := time.Now().UTC().Format(time.RFC3339)
				for _, uid := range append([]string{viewer}, targets...) {
					if err := repo.PutProfile(ctx, MyProfile{
						UserID:         uid,
						Nickname:       "nick_" + uid,
						AvatarURL:      "avatar_" + uid,
						PresenceStatus: UserPresenceStatus("online"),
						LastOnline:     lastOnline,
					}); err != nil {
						t.Fatalf("put profile %s: %v", uid, err)
					}
				}
				return repo, usersContractState{ctx: ctx, viewer: viewer, targets: targets}, nil
			},
		},
		{
			Name: "postgres",
			New: func(t *testing.T) (Repository, usersContractState, func()) {
				testdb.SkipIfShort(t)
				testdb.Open(t)
				txFactory, cleanup := testdb.BeginScope(t)
				qdb, ok := txFactory.DB.(persistence.QueryableDB)
				if !ok {
					t.Fatal("tx factory DB is not QueryableDB")
				}
				for _, uid := range append([]string{viewer}, targets...) {
					testdb.InsertUser(t, txFactory, uid, "nick_"+uid)
					if err := qdb.Exec(context.Background(), "UPDATE users SET avatar_url = $1 WHERE user_id = $2", "avatar_"+uid, uid); err != nil {
						t.Fatalf("set avatar %s: %v", uid, err)
					}
				}
				return NewPostgresRepository(txFactory), usersContractState{ctx: context.Background(), viewer: viewer, targets: targets}, cleanup
			},
		},
	}

	follow := func(t *testing.T, repo Repository, state usersContractState, target string) bool {
		t.Helper()
		changed, err := repo.Follow(state.ctx, state.viewer, target, FollowSourceProfile, "")
		if err != nil {
			t.Fatalf("follow %s: %v", target, err)
		}
		return changed
	}

	testdb.RunContract(t, impls, []testdb.ContractCase[Repository, usersContractState]{
		{
			Name: "follow is idempotent and reflected in following list",
			Run: func(t *testing.T, repo Repository, state usersContractState) {
				if !follow(t, repo, state, state.targets[0]) {
					t.Fatal("first follow must report changed")
				}
				if follow(t, repo, state, state.targets[0]) {
					t.Fatal("repeated follow must be a no-op (changed=false)")
				}
				page, err := repo.ListFollowing(state.ctx, state.viewer, 10, "")
				if err != nil {
					t.Fatalf("list following: %v", err)
				}
				if len(page.Items) != 1 || page.Items[0].User.User.UserID != state.targets[0] {
					t.Fatalf("following = %#v", page.Items)
				}
				item := page.Items[0]
				if item.User.User.Nickname != "nick_"+state.targets[0] {
					t.Fatalf("nickname = %q", item.User.User.Nickname)
				}
				if _, err := time.Parse(time.RFC3339, item.FollowedAt); err != nil {
					t.Fatalf("followed_at must be RFC3339, got %q", item.FollowedAt)
				}
				if item.User.PresenceStatus == nil || *item.User.PresenceStatus != "online" {
					t.Fatalf("presence_status = %#v", item.User.PresenceStatus)
				}
				if item.User.User.AvatarURL == nil || *item.User.User.AvatarURL != "avatar_"+state.targets[0] {
					t.Fatalf("avatar_url = %#v, want avatar_%s", item.User.User.AvatarURL, state.targets[0])
				}
				followers, err := repo.ListFollowers(state.ctx, state.viewer, 10, "")
				if err != nil {
					t.Fatalf("list followers: %v", err)
				}
				if len(followers.Items) != 0 {
					t.Fatalf("followers = %#v, want empty", followers.Items)
				}
				rel, err := repo.Relationship(state.ctx, state.viewer, state.targets[0])
				if err != nil {
					t.Fatalf("relationship: %v", err)
				}
				if !rel.IsFollowing || rel.IsFollowedBy || rel.IsBlocked {
					t.Fatalf("relationship = %#v", rel)
				}
			},
		},
		{
			Name: "mutual follow yields friend with followed_back_at",
			Run: func(t *testing.T, repo Repository, state usersContractState) {
				follow(t, repo, state, state.targets[0])
				if _, err := repo.Follow(state.ctx, state.targets[0], state.viewer, FollowSourceProfile, ""); err != nil {
					t.Fatalf("reverse follow: %v", err)
				}
				friends, err := repo.ListFriends(state.ctx, state.viewer, 10, "")
				if err != nil {
					t.Fatalf("list friends: %v", err)
				}
				if len(friends.Items) != 1 || friends.Items[0].User.User.UserID != state.targets[0] {
					t.Fatalf("friends = %#v", friends.Items)
				}
				if friends.Items[0].FollowedBackAt == nil {
					t.Fatal("followed_back_at must be set for mutual follow")
				}
				if _, err := time.Parse(time.RFC3339, *friends.Items[0].FollowedBackAt); err != nil {
					t.Fatalf("followed_back_at must be RFC3339, got %q", *friends.Items[0].FollowedBackAt)
				}
				rel, err := repo.Relationship(state.ctx, state.viewer, state.targets[0])
				if err != nil {
					t.Fatalf("relationship: %v", err)
				}
				if !rel.IsFollowing || !rel.IsFollowedBy {
					t.Fatalf("relationship = %#v", rel)
				}
			},
		},
		{
			Name: "unfollow clears following and friend edges",
			Run: func(t *testing.T, repo Repository, state usersContractState) {
				follow(t, repo, state, state.targets[0])
				if _, err := repo.Follow(state.ctx, state.targets[0], state.viewer, FollowSourceProfile, ""); err != nil {
					t.Fatalf("reverse follow: %v", err)
				}
				if err := repo.Unfollow(state.ctx, state.viewer, state.targets[0]); err != nil {
					t.Fatalf("unfollow: %v", err)
				}
				following, err := repo.ListFollowing(state.ctx, state.viewer, 10, "")
				if err != nil {
					t.Fatalf("list following: %v", err)
				}
				if len(following.Items) != 0 {
					t.Fatalf("following = %#v, want empty", following.Items)
				}
				friends, err := repo.ListFriends(state.ctx, state.viewer, 10, "")
				if err != nil {
					t.Fatalf("list friends: %v", err)
				}
				if len(friends.Items) != 0 {
					t.Fatalf("friends = %#v, want empty (mutual edge broken)", friends.Items)
				}
				rel, err := repo.Relationship(state.ctx, state.viewer, state.targets[0])
				if err != nil {
					t.Fatalf("relationship: %v", err)
				}
				if rel.IsFollowing {
					t.Fatalf("relationship = %#v", rel)
				}
				if !follow(t, repo, state, state.targets[0]) {
					t.Fatal("follow after unfollow must report changed")
				}
			},
		},
		{
			Name: "block cascade clears follows and exposes blocked list fields",
			Run: func(t *testing.T, repo Repository, state usersContractState) {
				follow(t, repo, state, state.targets[1])
				if _, err := repo.Follow(state.ctx, state.targets[1], state.viewer, FollowSourceProfile, ""); err != nil {
					t.Fatalf("reverse follow: %v", err)
				}
				if err := repo.BlockCascade(state.ctx, state.viewer, state.targets[1], "spam"); err != nil {
					t.Fatalf("block cascade: %v", err)
				}
				rel, err := repo.Relationship(state.ctx, state.viewer, state.targets[1])
				if err != nil {
					t.Fatalf("relationship: %v", err)
				}
				if !rel.IsBlocked || rel.IsFollowing || rel.IsFollowedBy {
					t.Fatalf("relationship = %#v", rel)
				}
				reverse, err := repo.Relationship(state.ctx, state.targets[1], state.viewer)
				if err != nil {
					t.Fatalf("reverse relationship: %v", err)
				}
				if reverse.IsFollowing {
					t.Fatalf("reverse relationship = %#v (cascade must clear both directions)", reverse)
				}
				blocked, err := repo.ListBlocked(state.ctx, state.viewer, 10, "")
				if err != nil {
					t.Fatalf("list blocked: %v", err)
				}
				if len(blocked.Items) != 1 || blocked.Items[0].User.UserID != state.targets[1] {
					t.Fatalf("blocked = %#v", blocked.Items)
				}
				blockedItem := blocked.Items[0]
				if blockedItem.Reason == nil || *blockedItem.Reason != "spam" {
					t.Fatalf("block reason = %#v", blockedItem.Reason)
				}
				if _, err := time.Parse(time.RFC3339, blockedItem.BlockedAt); err != nil {
					t.Fatalf("blocked_at must be RFC3339, got %q", blockedItem.BlockedAt)
				}
				if blockedItem.User.AvatarURL == nil || *blockedItem.User.AvatarURL != "avatar_"+state.targets[1] {
					t.Fatalf("blocked user avatar_url = %#v, want avatar_%s", blockedItem.User.AvatarURL, state.targets[1])
				}
				callerBlocked, targetBlocked, err := repo.BlockedEitherWay(state.ctx, state.viewer, state.targets[1])
				if err != nil {
					t.Fatalf("blocked either way: %v", err)
				}
				if !callerBlocked || targetBlocked {
					t.Fatalf("blocked either way = %v,%v", callerBlocked, targetBlocked)
				}
				if err := repo.Unblock(state.ctx, state.viewer, state.targets[1]); err != nil {
					t.Fatalf("unblock: %v", err)
				}
				blocked, err = repo.ListBlocked(state.ctx, state.viewer, 10, "")
				if err != nil {
					t.Fatalf("list blocked after unblock: %v", err)
				}
				if len(blocked.Items) != 0 {
					t.Fatalf("blocked after unblock = %#v", blocked.Items)
				}
			},
		},
		{
			Name: "following pagination is stable and disjoint",
			Run: func(t *testing.T, repo Repository, state usersContractState) {
				for _, target := range state.targets {
					follow(t, repo, state, target)
				}
				seen := map[string]bool{}
				cursor := ""
				pages := 0
				for {
					page, err := repo.ListFollowing(state.ctx, state.viewer, 2, cursor)
					if err != nil {
						t.Fatalf("page %d: %v", pages, err)
					}
					if len(page.Items) > 2 {
						t.Fatalf("page %d size = %d", pages, len(page.Items))
					}
					for _, item := range page.Items {
						id := item.User.User.UserID
						if seen[id] {
							t.Fatalf("duplicate %s across pages", id)
						}
						seen[id] = true
					}
					pages++
					if page.NextCursor == nil {
						break
					}
					cursor = *page.NextCursor
					if pages > 5 {
						t.Fatal("pagination did not terminate")
					}
				}
				if len(seen) != len(state.targets) {
					t.Fatalf("seen %d users, want %d", len(seen), len(state.targets))
				}
			},
		},
		{
			Name: "relationship batch matches single lookups",
			Run: func(t *testing.T, repo Repository, state usersContractState) {
				follow(t, repo, state, state.targets[0])
				batch, err := repo.RelationshipBatch(state.ctx, state.viewer, state.targets)
				if err != nil {
					t.Fatalf("relationship batch: %v", err)
				}
				if len(batch) != len(state.targets) {
					t.Fatalf("batch size = %d", len(batch))
				}
				single, err := repo.Relationship(state.ctx, state.viewer, state.targets[0])
				if err != nil {
					t.Fatalf("relationship: %v", err)
				}
				if batch[state.targets[0]] != single {
					t.Fatalf("batch[0] = %#v, single = %#v", batch[state.targets[0]], single)
				}
				for _, target := range state.targets[1:] {
					if batch[target].IsFollowing {
						t.Fatalf("unexpected following edge for %s", target)
					}
				}
			},
		},
		{
			Name: "invalid cursor is rejected",
			Run: func(t *testing.T, repo Repository, state usersContractState) {
				if _, err := repo.ListFollowing(state.ctx, state.viewer, 10, "not-a-cursor"); !errors.Is(err, ErrInvalidCursor) {
					t.Fatalf("invalid cursor err = %v, want ErrInvalidCursor", err)
				}
			},
		},
	})
}
