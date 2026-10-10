// Package hotcache 提供热数据读穿缓存：高频读路径（群成员关系、用户设置等）
// 不再逐请求打 DB。设计约束（2026-10-10 审计结论）：
//   - 配置了 Redis 则用 Redis（多实例就绪，失效 DEL 跨实例可见）；未配置或
//     Redis 故障时降级为进程内 TTL map——缓存永远 fail-open：任何错误都按
//     miss 处理，调用方回源 DB，绝不因缓存层故障放大延迟或阻断请求。
//   - 一致性敏感度高的数据（封禁等）禁止进本缓存；可容忍数据 TTL 仍应短
//     （60s 级），并在写路径主动 Delete。
package hotcache

import (
	"context"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

const localSweepInterval = time.Minute

type localEntry struct {
	value     string
	expiresAt time.Time
}

// Cache 是键值 TTL 缓存。零值不可用，经 New 创建；*Cache 为 nil 时所有方法
// 均为 no-op miss（调用方无需判空）。
type Cache struct {
	rdb   *redis.Client
	local sync.Map // key -> localEntry（仅 rdb == nil 时使用）
	stop  chan struct{}
}

// New 创建缓存：rdb 非 nil 用 Redis；否则用进程内 TTL map（带周期清扫）。
func New(rdb *redis.Client) *Cache {
	c := &Cache{rdb: rdb}
	if rdb == nil {
		c.stop = make(chan struct{})
		go func() {
			ticker := time.NewTicker(localSweepInterval)
			defer ticker.Stop()
			for {
				select {
				case <-c.stop:
					return
				case <-ticker.C:
					now := time.Now()
					c.local.Range(func(key, value any) bool {
						if e, ok := value.(localEntry); ok && now.After(e.expiresAt) {
							c.local.Delete(key)
						}
						return true
					})
				}
			}
		}()
	}
	return c
}

// Close 释放后台资源（本地模式清扫协程）。nil 安全。
func (c *Cache) Close() {
	if c != nil && c.stop != nil {
		close(c.stop)
	}
}

// Get 返回键值与命中标记。Redis 错误按 miss 处理（fail-open）。
func (c *Cache) Get(ctx context.Context, key string) (string, bool) {
	if c == nil {
		return "", false
	}
	if c.rdb != nil {
		v, err := c.rdb.Get(ctx, key).Result()
		if err != nil {
			return "", false
		}
		return v, true
	}
	if e, ok := c.local.Load(key); ok {
		if entry, ok := e.(localEntry); ok && time.Now().Before(entry.expiresAt) {
			return entry.value, true
		}
		c.local.Delete(key)
	}
	return "", false
}

// Set 写入键值（TTL 必须为正）。Redis 错误忽略（下次 Get 回源）。
func (c *Cache) Set(ctx context.Context, key, value string, ttl time.Duration) {
	if c == nil || ttl <= 0 {
		return
	}
	if c.rdb != nil {
		_ = c.rdb.Set(ctx, key, value, ttl).Err()
		return
	}
	c.local.Store(key, localEntry{value: value, expiresAt: time.Now().Add(ttl)})
}

// Delete 删除键（写路径失效入口）。Redis 错误忽略（TTL 兜底）。
func (c *Cache) Delete(ctx context.Context, keys ...string) {
	if c == nil || len(keys) == 0 {
		return
	}
	if c.rdb != nil {
		_ = c.rdb.Del(ctx, keys...).Err()
		return
	}
	for _, k := range keys {
		c.local.Delete(k)
	}
}

// ── 约定键族（跨模块共享同一格式，写失效才能命中同一键）──────────────────

// GroupMemberTTL 群成员关系缓存 TTL：踢人/退群写路径主动删键，TTL 仅兜底
// （成员变动低频、被移群后 ≤TTL 内发送会被拒——可容忍窗口）。
const GroupMemberTTL = time.Minute

// GroupMemberKey 群成员关系键；值为 "1"（成员）/"0"（非成员，防穿透）。
func GroupMemberKey(groupID, userID string) string {
	return "hot:grp:" + groupID + ":member:" + userID
}
