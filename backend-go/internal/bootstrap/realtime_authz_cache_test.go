package bootstrap

import (
	"context"
	"sync"
	"testing"

	"hotspot/backend-go/internal/modules/users"
	"hotspot/backend-go/internal/platform/hotcache"
)

// authzUsersRepo 是 users.Repository 的最小测试桩（users 内存仓储已迁入其
// 模块测试构建，P0-2）：只实现缓存方向还原用例需要的关注/拉黑读写。
type authzUsersRepo struct {
	users.Repository
	mu      sync.Mutex
	follows map[[2]string]bool
	blocks  map[[2]string]bool
}

func newAuthzUsersRepo() *authzUsersRepo {
	return &authzUsersRepo{follows: make(map[[2]string]bool), blocks: make(map[[2]string]bool)}
}

func (r *authzUsersRepo) Follow(_ context.Context, userID, targetUserID string, _ users.FollowSource, _ string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.follows[[2]string{userID, targetUserID}] = true
	return true, nil
}

func (r *authzUsersRepo) Block(_ context.Context, userID, targetUserID, _ string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.blocks[[2]string{userID, targetUserID}] = true
	return nil
}

func (r *authzUsersRepo) Relationship(_ context.Context, userID, targetUserID string) (users.Relationship, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return users.Relationship{
		IsFollowing:  r.follows[[2]string{userID, targetUserID}],
		IsFollowedBy: r.follows[[2]string{targetUserID, userID}],
		IsBlocked:    r.blocks[[2]string{userID, targetUserID}],
	}, nil
}

func (r *authzUsersRepo) BlockedEitherWay(_ context.Context, aUserID, bUserID string) (bool, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.blocks[[2]string{aUserID, bUserID}], r.blocks[[2]string{bUserID, aUserID}], nil
}

// 方向还原回归：拉黑/关注缓存的位掩码与 JSON 都按【归一化后】参数序存储，
// 实参顺序颠倒时读侧必须还原回实参方向——否则 u1 拉黑 u2 会被误判成 u2 拉黑 u1。
func TestRealtimeUserRepoBlockedEitherWayDirection(t *testing.T) {
	ctx := context.Background()
	repo := newAuthzUsersRepo()
	// u1 拉黑 u2（实参顺序：u1 在前）
	if err := repo.Block(ctx, "u1", "u2", "test"); err != nil {
		t.Fatal(err)
	}
	adapter := realtimeUserRepo{repo: repo, cache: hotcache.New(nil)}
	defer adapter.cache.Close()

	// 先以 (u1, u2) 顺序回源并填充缓存
	caller, target, err := adapter.BlockedEitherWay(ctx, "u1", "u2")
	if err != nil || !caller || target {
		t.Fatalf("u1 拉黑 u2：期望 caller=true target=false，得到 %v/%v err=%v", caller, target, err)
	}
	// 再以交换顺序命中缓存：方向必须还原（u2 没拉黑 u1，u1 仍拉黑 u2）
	caller, target, err = adapter.BlockedEitherWay(ctx, "u2", "u1")
	if err != nil || caller || !target {
		t.Fatalf("交换顺序读缓存：期望 caller=false target=true，得到 %v/%v err=%v", caller, target, err)
	}
}

func TestRealtimeUserRepoRelationshipDirection(t *testing.T) {
	ctx := context.Background()
	repo := newAuthzUsersRepo()
	// u1 关注 u3（实参顺序：u1 在前）
	if _, err := repo.Follow(ctx, "u1", "u3", users.FollowSourceMoment, ""); err != nil {
		t.Fatal(err)
	}
	adapter := realtimeUserRepo{repo: repo, cache: hotcache.New(nil)}
	defer adapter.cache.Close()

	following, followedBy, _, err := adapter.Relationship(ctx, "u1", "u3")
	if err != nil || !following || followedBy {
		t.Fatalf("u1 关注 u3：期望 following=true followedBy=false，得到 %v/%v err=%v", following, followedBy, err)
	}
	// 交换顺序命中缓存：u3 视角应为 following=false followedBy=true
	following, followedBy, _, err = adapter.Relationship(ctx, "u3", "u1")
	if err != nil || following || !followedBy {
		t.Fatalf("交换顺序读缓存：期望 following=false followedBy=true，得到 %v/%v err=%v", following, followedBy, err)
	}
}
