package repository

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// TokenBlacklistRepository 负责存储被吊销的 access/refresh token。
//
// 设计上使用 Redis 的 Key 过期语义：
// - key: 原始 token 字符串（access 或 refresh）；
// - value: 固定占位（例如 "1"）；
// - TTL: 对应 token 的过期时间与当前时间之差。
type TokenBlacklistRepository interface {
	BlacklistToken(ctx context.Context, token string, expiresAt time.Time) error
	IsTokenBlacklisted(ctx context.Context, token string) (bool, error)
}

type tokenBlacklistRepository struct {
	client *redis.Client
}

// NewTokenBlacklistRepository 创建基于 Redis 的黑名单存储实现。
func NewTokenBlacklistRepository(client *redis.Client) TokenBlacklistRepository {
	return &tokenBlacklistRepository{client: client}
}

func (r *tokenBlacklistRepository) BlacklistToken(ctx context.Context, token string, expiresAt time.Time) error {
	if token == "" || r.client == nil {
		return nil
	}

	// 计算 TTL，若已过期则使用最小正 TTL 防止立即过期导致无效果。
	ttl := time.Until(expiresAt)
	if ttl <= 0 {
		ttl = time.Second
	}

	return r.client.Set(ctx, token, "1", ttl).Err()
}

func (r *tokenBlacklistRepository) IsTokenBlacklisted(ctx context.Context, token string) (bool, error) {
	if token == "" || r.client == nil {
		return false, nil
	}

	res := r.client.Exists(ctx, token)
	if err := res.Err(); err != nil {
		return false, err
	}
	return res.Val() > 0, nil
}
