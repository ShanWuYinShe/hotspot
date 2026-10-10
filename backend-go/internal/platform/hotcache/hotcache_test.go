package hotcache

import (
	"context"
	"testing"
	"time"
)

func TestLocalCacheSetGetDelete(t *testing.T) {
	c := New(nil) // 无 Redis：进程内 TTL map
	defer c.Close()
	ctx := context.Background()

	if _, ok := c.Get(ctx, "k"); ok {
		t.Fatal("空缓存不应命中")
	}
	c.Set(ctx, "k", "v", time.Minute)
	if v, ok := c.Get(ctx, "k"); !ok || v != "v" {
		t.Fatalf("期望命中 v，得到 %q ok=%v", v, ok)
	}
	c.Delete(ctx, "k")
	if _, ok := c.Get(ctx, "k"); ok {
		t.Fatal("删除后不应命中")
	}
}

func TestLocalCacheExpiry(t *testing.T) {
	c := New(nil)
	defer c.Close()
	ctx := context.Background()
	c.Set(ctx, "k", "v", 5*time.Millisecond)
	time.Sleep(15 * time.Millisecond)
	if _, ok := c.Get(ctx, "k"); ok {
		t.Fatal("过期后不应命中")
	}
}

func TestNilCacheIsNoOp(t *testing.T) {
	var c *Cache
	ctx := context.Background()
	c.Set(ctx, "k", "v", time.Minute)
	if _, ok := c.Get(ctx, "k"); ok {
		t.Fatal("nil 缓存不应命中")
	}
	c.Delete(ctx, "k") // 不应 panic
	c.Close()
}

func TestGroupMemberKey(t *testing.T) {
	got := GroupMemberKey("g1", "u1")
	want := "hot:grp:g1:member:u1"
	if got != want {
		t.Fatalf("键格式漂移：got %q want %q（写失效两端依赖同一格式）", got, want)
	}
}
