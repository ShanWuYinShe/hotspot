package users

import (
	"context"
	"errors"
	"testing"

	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/testdb"
)

type g3RowSequenceBackend struct {
	*g3Backend
	rowCalls int
}

func (b *g3RowSequenceBackend) QueryRow(ctx context.Context, query string, args ...any) persistence.Row {
	b.rowCalls++
	if b.rowCalls > 1 {
		return testdb.RowValues{Err: errors.New("second relation lookup failed")}
	}
	return b.g3Backend.QueryRow(ctx, query, args...)
}

func TestG3RepoRelationshipBranches(t *testing.T) {
	ctx := context.Background()

	backend := &g3Backend{
		queryRowByQuery: map[string]persistence.Row{
			queryGetUserRelation:     testdb.RowValues{Values: []any{true, false, "buddy", g3Time, nil, ""}},
			queryGetBlockedEitherWay: testdb.RowValues{Values: []any{true, false}},
		},
		rowsByQuery: map[string]persistence.Rows{},
		queryErrs:   map[string]error{},
		execErrs:    map[string]error{},
	}
	repo := g3RepoWith(backend)
	rel, err := repo.Relationship(ctx, "u1", "u2")
	if err != nil || !rel.IsFollowing || rel.IsBlocked {
		t.Fatalf("relationship = %#v err=%v", rel, err)
	}
	following, blocked, remark, followedAt, blockedAt, reason, err := repo.getRelationFields(ctx, "u1", "u2")
	if err != nil || !following || blocked || remark != "buddy" || followedAt == nil || blockedAt != nil || reason != "" {
		t.Fatalf("relation fields = %v %v %q %v %v %q %v", following, blocked, remark, followedAt, blockedAt, reason, err)
	}
	callerBlocked, targetBlocked, err := repo.BlockedEitherWay(ctx, "u1", "u2")
	if err != nil || !callerBlocked || targetBlocked {
		t.Fatalf("blocked either way = %v %v %v", callerBlocked, targetBlocked, err)
	}
	// 第二段关系查询失败（第一次成功、第二次报错）。
	seqRepo := g3RepoWith(&g3RowSequenceBackend{g3Backend: backend})
	if _, err := seqRepo.Relationship(ctx, "u1", "u2"); err == nil {
		t.Fatal("expected second relation lookup error")
	}
	// 首次查询即失败。
	errBackend := &g3Backend{queryRowByQuery: map[string]persistence.Row{}, rowsByQuery: map[string]persistence.Rows{}, queryErrs: map[string]error{}, execErrs: map[string]error{}}
	if _, err := g3RepoWith(errBackend).Relationship(ctx, "u1", "u2"); err == nil {
		t.Fatal("expected relationship error")
	}
	if _, _, err := g3RepoWith(errBackend).BlockedEitherWay(ctx, "u1", "u2"); err == nil {
		t.Fatal("expected blocked either way error")
	}
	if _, _, _, _, _, _, err := g3RepoWith(errBackend).getRelationFields(ctx, "u1", "u2"); err == nil {
		t.Fatal("expected relation fields error")
	}
}

func TestG3RepoMutualFriendsBranches(t *testing.T) {
	ctx := context.Background()
	backend := &g3Backend{
		queryRowByQuery: map[string]persistence.Row{},
		rowsByQuery: map[string]persistence.Rows{
			queryMutualFriends: &g3Rows{rows: [][]any{
				{"u3", "Three", "", 2}, {"u4", "Four", "", 2},
			}},
		},
		queryErrs: map[string]error{},
		execErrs:  map[string]error{},
	}
	repo := g3RepoWith(backend)
	total, preview, err := repo.MutualFriends(ctx, "u1", "u2", 3)
	if err != nil || total != 2 || len(preview) != 2 {
		t.Fatalf("mutual friends = %d %#v err=%v", total, preview, err)
	}
	// previewLimit=0 → 仍取 1 行读总数，预览为空。
	backend.rowsByQuery[queryMutualFriends] = &g3Rows{rows: [][]any{{"u3", "Three", "", 2}, {"u4", "Four", "", 2}}}
	if total, preview, err := repo.MutualFriends(ctx, "u1", "u2", 0); err != nil || total != 2 || len(preview) != 0 {
		t.Fatalf("mutual friends limit0 = %d %#v err=%v", total, preview, err)
	}
	// 负数 previewLimit 归一为 0。
	backend.rowsByQuery[queryMutualFriends] = &g3Rows{rows: [][]any{{"u3", "Three", "", 2}, {"u4", "Four", "", 2}}}
	if _, preview, err := repo.MutualFriends(ctx, "u1", "u2", -5); err != nil || len(preview) != 0 {
		t.Fatalf("mutual friends negative = %#v err=%v", preview, err)
	}
	backend.queryErrs[queryMutualFriends] = errors.New("boom")
	if _, _, err := repo.MutualFriends(ctx, "u1", "u2", 3); err == nil {
		t.Fatal("expected mutual friends query error")
	}
	backend.queryErrs = map[string]error{}
	backend.rowsByQuery[queryMutualFriends] = &g3Rows{rows: [][]any{{"u3", "Three", "", 2}}, scanErr: errors.New("scan")}
	if _, _, err := repo.MutualFriends(ctx, "u1", "u2", 3); err == nil {
		t.Fatal("expected mutual friends scan error")
	}
	backend.rowsByQuery[queryMutualFriends] = &g3Rows{rows: nil, tailErr: errors.New("tail")}
	if _, _, err := repo.MutualFriends(ctx, "u1", "u2", 3); err == nil {
		t.Fatal("expected mutual friends rows error")
	}
}

func TestG3RepoRelationshipBatchBranches(t *testing.T) {
	ctx := context.Background()
	backend := &g3Backend{
		queryRowByQuery: map[string]persistence.Row{},
		rowsByQuery: map[string]persistence.Rows{
			queryListUserRelationsForward: &g3Rows{rows: [][]any{{"u2", true, false}}},
			queryListUserRelationsReverse: &g3Rows{rows: [][]any{{"u2", true, true}}},
		},
		queryErrs: map[string]error{},
		execErrs:  map[string]error{},
	}
	repo := g3RepoWith(backend)
	result, err := repo.RelationshipBatch(ctx, "u1", []string{"u2", "u3"})
	if err != nil {
		t.Fatalf("relationship batch: %v", err)
	}
	if len(result) != 2 || !result["u2"].IsFollowing || !result["u2"].IsFollowedBy || !result["u2"].TargetBlocked {
		t.Fatalf("relationship batch = %#v", result)
	}
	if _, ok := result["u3"]; !ok {
		t.Fatal("missing target must still have zero relationship entry")
	}
	// 空目标列表直接返回。
	if result, err := repo.RelationshipBatch(ctx, "u1", nil); err != nil || len(result) != 0 {
		t.Fatalf("empty batch = %#v err=%v", result, err)
	}
	// 前向查询错误 / 前向 scan / 前向 rows.Err。
	backend.queryErrs = map[string]error{queryListUserRelationsForward: errors.New("boom")}
	if _, err := repo.RelationshipBatch(ctx, "u1", []string{"u2"}); err == nil {
		t.Fatal("expected forward query error")
	}
	backend.queryErrs = map[string]error{}
	backend.rowsByQuery[queryListUserRelationsForward] = &g3Rows{rows: [][]any{{"u2", true, false}}, scanErr: errors.New("scan")}
	if _, err := repo.RelationshipBatch(ctx, "u1", []string{"u2"}); err == nil {
		t.Fatal("expected forward scan error")
	}
	backend.rowsByQuery[queryListUserRelationsForward] = &g3Rows{rows: [][]any{{"u2", true, false}}, tailErr: errors.New("tail")}
	if _, err := repo.RelationshipBatch(ctx, "u1", []string{"u2"}); err == nil {
		t.Fatal("expected forward rows error")
	}
	// 反向查询错误 / 反向 scan / 反向 rows.Err。
	backend.rowsByQuery[queryListUserRelationsForward] = &g3Rows{rows: [][]any{{"u2", true, false}}}
	backend.queryErrs = map[string]error{queryListUserRelationsReverse: errors.New("boom")}
	if _, err := repo.RelationshipBatch(ctx, "u1", []string{"u2"}); err == nil {
		t.Fatal("expected reverse query error")
	}
	backend.queryErrs = map[string]error{}
	backend.rowsByQuery[queryListUserRelationsReverse] = &g3Rows{rows: [][]any{{"u2", true, true}}, scanErr: errors.New("scan")}
	if _, err := repo.RelationshipBatch(ctx, "u1", []string{"u2"}); err == nil {
		t.Fatal("expected reverse scan error")
	}
	backend.rowsByQuery[queryListUserRelationsReverse] = &g3Rows{rows: [][]any{{"u2", true, true}}, tailErr: errors.New("tail")}
	if _, err := repo.RelationshipBatch(ctx, "u1", []string{"u2"}); err == nil {
		t.Fatal("expected reverse rows error")
	}
}

func TestG3RepoListPostgresBranches(t *testing.T) {
	ctx := context.Background()
	blockedRow := []any{"u2", "Two", "", g3Time, "spam"}
	friendRow := []any{"u2", "Two", "", "online", g3Time, "buddy", g3Time, g3Time}
	followingRow := []any{"u2", "Two", "", g3Time, "online", g3Time, "buddy"}
	followerRow := []any{"u2", "Two", "", g3Time, "online", g3Time}
	backend := &g3Backend{
		queryRowByQuery: map[string]persistence.Row{},
		rowsByQuery: map[string]persistence.Rows{
			queryListBlockedUsers: &g3Rows{rows: [][]any{blockedRow, blockedRow, blockedRow}},
			queryListFriends:      &g3Rows{rows: [][]any{friendRow, friendRow}},
			queryListFollowing:    &g3Rows{rows: [][]any{followingRow, followingRow}},
			queryListFollowers:    &g3Rows{rows: [][]any{followerRow}},
		},
		queryErrs: map[string]error{},
		execErrs:  map[string]error{},
	}
	repo := g3RepoWith(backend)

	blocked, err := repo.ListBlocked(ctx, "u1", 2, "")
	if err != nil || len(blocked.Items) != 2 || blocked.NextCursor == nil {
		t.Fatalf("ListBlocked = %#v err=%v", blocked, err)
	}
	if blocked.Items[0].BlockedAt == "" || blocked.Items[0].Reason == nil {
		t.Fatalf("blocked item = %#v", blocked.Items[0])
	}
	friends, err := repo.ListFriends(ctx, "u1", 1, "")
	if err != nil || len(friends.Items) != 1 || friends.NextCursor == nil || friends.Items[0].Remark == nil {
		t.Fatalf("ListFriends = %#v err=%v", friends, err)
	}
	following, err := repo.ListFollowing(ctx, "u1", 5, "")
	if err != nil || len(following.Items) != 2 || following.Items[0].User.LastOnline == nil {
		t.Fatalf("ListFollowing = %#v err=%v", following, err)
	}
	followers, err := repo.ListFollowers(ctx, "u1", 5, "")
	if err != nil || len(followers.Items) != 1 || followers.Items[0].User.PresenceStatus == nil {
		t.Fatalf("ListFollowers = %#v err=%v", followers, err)
	}

	// 查询错误。
	for _, q := range []string{queryListBlockedUsers, queryListFriends, queryListFollowing, queryListFollowers} {
		backend.queryErrs = map[string]error{q: errors.New("boom")}
		switch q {
		case queryListBlockedUsers:
			_, err = repo.ListBlocked(ctx, "u1", 10, "")
		case queryListFriends:
			_, err = repo.ListFriends(ctx, "u1", 10, "")
		case queryListFollowing:
			_, err = repo.ListFollowing(ctx, "u1", 10, "")
		default:
			_, err = repo.ListFollowers(ctx, "u1", 10, "")
		}
		if err == nil {
			t.Fatalf("expected query error for %s", q)
		}
	}
	backend.queryErrs = map[string]error{}

	// scan / rows.Err。
	backend.rowsByQuery[queryListBlockedUsers] = &g3Rows{rows: [][]any{blockedRow}, scanErr: errors.New("scan")}
	if _, err := repo.ListBlocked(ctx, "u1", 10, ""); err == nil {
		t.Fatal("expected ListBlocked scan error")
	}
	backend.rowsByQuery[queryListBlockedUsers] = &g3Rows{rows: nil, tailErr: errors.New("tail")}
	if _, err := repo.ListBlocked(ctx, "u1", 10, ""); err == nil {
		t.Fatal("expected ListBlocked rows error")
	}
	backend.rowsByQuery[queryListFriends] = &g3Rows{rows: [][]any{friendRow}, scanErr: errors.New("scan")}
	if _, err := repo.ListFriends(ctx, "u1", 10, ""); err == nil {
		t.Fatal("expected ListFriends scan error")
	}
	backend.rowsByQuery[queryListFriends] = &g3Rows{rows: nil, tailErr: errors.New("tail")}
	if _, err := repo.ListFriends(ctx, "u1", 10, ""); err == nil {
		t.Fatal("expected ListFriends rows error")
	}
	backend.rowsByQuery[queryListFollowing] = &g3Rows{rows: [][]any{followingRow}, scanErr: errors.New("scan")}
	if _, err := repo.ListFollowing(ctx, "u1", 10, ""); err == nil {
		t.Fatal("expected ListFollowing scan error")
	}
	backend.rowsByQuery[queryListFollowing] = &g3Rows{rows: nil, tailErr: errors.New("tail")}
	if _, err := repo.ListFollowing(ctx, "u1", 10, ""); err == nil {
		t.Fatal("expected ListFollowing rows error")
	}
	backend.rowsByQuery[queryListFollowers] = &g3Rows{rows: [][]any{followerRow}, scanErr: errors.New("scan")}
	if _, err := repo.ListFollowers(ctx, "u1", 10, ""); err == nil {
		t.Fatal("expected ListFollowers scan error")
	}
	backend.rowsByQuery[queryListFollowers] = &g3Rows{rows: nil, tailErr: errors.New("tail")}
	if _, err := repo.ListFollowers(ctx, "u1", 10, ""); err == nil {
		t.Fatal("expected ListFollowers rows error")
	}

	// 无 PG 时（package 级函数）返回 ErrNotImplemented。
	emptyFactory := persistence.NewFactory(config.Config{})
	if _, err := listBlockedUsersPostgres(ctx, emptyFactory, "u1", 10, ""); err == nil {
		t.Fatal("expected listBlockedUsersPostgres not-implemented error")
	}
	if _, err := listFriendsPostgres(ctx, emptyFactory, "u1", 10, ""); err == nil {
		t.Fatal("expected listFriendsPostgres not-implemented error")
	}
	if _, err := listFollowingPostgres(ctx, emptyFactory, "u1", 10, ""); err == nil {
		t.Fatal("expected listFollowingPostgres not-implemented error")
	}
	if _, err := listFollowersPostgres(ctx, emptyFactory, "u1", 10, ""); err == nil {
		t.Fatal("expected listFollowersPostgres not-implemented error")
	}
}

func TestG3RepoRelationshipMopUp(t *testing.T) {
	ctx := context.Background()
	empty := g3EmptyRepo()
	if _, _, _, _, _, _, err := empty.getRelationFields(ctx, "u1", "u2"); err == nil {
		t.Fatal("expected getRelationFields not-implemented error")
	}
	if _, err := g3RepoWith(&g3Backend{queryRowByQuery: map[string]persistence.Row{}, rowsByQuery: map[string]persistence.Rows{}, queryErrs: map[string]error{}, execErrs: map[string]error{}}).ListBlocked(ctx, "u1", 10, "not-a-cursor"); !errors.Is(err, ErrInvalidCursor) {
		t.Fatalf("expected invalid cursor, got %v", err)
	}
	// 关注者分页：两行 + limit=1 → NextCursor。
	backend := &g3Backend{
		queryRowByQuery: map[string]persistence.Row{},
		rowsByQuery: map[string]persistence.Rows{
			queryListFollowers: &g3Rows{rows: [][]any{
				{"u2", "Two", "", g3Time, "online", g3Time},
				{"u3", "Three", "", g3Time, "online", g3Time},
			}},
		},
		queryErrs: map[string]error{},
		execErrs:  map[string]error{},
	}
	followers, err := g3RepoWith(backend).ListFollowers(ctx, "u1", 1, "")
	if err != nil || len(followers.Items) != 1 || followers.NextCursor == nil {
		t.Fatalf("followers pagination = %#v err=%v", followers, err)
	}
}
