package bootstrap

import (
	"context"
	"testing"

	"hotspot/backend-go/internal/modules/users"
	"hotspot/backend-go/internal/platform/hotcache"
)

// 方向还原回归：拉黑/关注缓存的位掩码与 JSON 都按【归一化后】参数序存储，
// 实参顺序颠倒时读侧必须还原回实参方向——否则 u1 拉黑 u2 会被误判成 u2 拉黑 u1。
func TestRealtimeUserRepoBlockedEitherWayDirection(t *testing.T) {
	ctx := context.Background()
	repo := users.NewInMemoryRepository()
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
	repo := users.NewInMemoryRepository()
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
