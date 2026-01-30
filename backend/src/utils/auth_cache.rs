//! 认证缓存模块
//!
//! 使用内存缓存减少 Redis 查询频率，提升认证中间件性能。
//!
//! 设计要点：
//! - 使用 moka 作为缓存实现（支持 TTL、LRU、并发安全）
//! - 缓存已验证的 token，避免每次请求都查询 Redis
//! - 设置合理的 TTL（5分钟），平衡性能与安全性
//! - 提供缓存失效机制（用于登出、撤销会话等场景）

use moka::future::Cache;
use std::sync::Arc;
use std::time::Duration;

/// 认证缓存项
#[derive(Clone, Debug)]
pub struct AuthCacheEntry {
    pub user_id: i64,
    pub session_id: uuid::Uuid,
}

/// 认证缓存管理器
#[derive(Clone)]
pub struct AuthCache {
    cache: Arc<Cache<String, AuthCacheEntry>>,
}

impl AuthCache {
    /// 创建新的认证缓存实例
    ///
    /// # 参数
    /// - `max_capacity`: 最大缓存容量（默认 10000）
    /// - `ttl_seconds`: 缓存 TTL（默认 300 秒 = 5 分钟）
    pub fn new(max_capacity: u64, ttl_seconds: u64) -> Self {
        let cache = Cache::builder()
            .max_capacity(max_capacity)
            .time_to_live(Duration::from_secs(ttl_seconds))
            .build();

        Self {
            cache: Arc::new(cache),
        }
    }

    /// 获取缓存的认证信息
    pub async fn get(&self, token: &str) -> Option<AuthCacheEntry> {
        self.cache.get(token).await
    }

    /// 缓存认证信息
    pub async fn set(&self, token: String, entry: AuthCacheEntry) {
        self.cache.insert(token, entry).await;
    }

    /// 使指定 token 的缓存失效
    pub async fn invalidate(&self, token: &str) {
        self.cache.invalidate(token).await;
    }

    /// 使指定用户的所有缓存失效（用于登出、撤销所有会话等场景）
    ///
    /// 注意：这是一个代价较高的操作，需要遍历整个缓存
    pub async fn invalidate_by_user(&self, user_id: i64) {
        // invalidate_entries_if 的 predicate 必须返回 bool，且必须是 Fn 而非 FnMut
        self.cache
            .invalidate_entries_if(move |_key, entry| entry.user_id == user_id)
            .expect("Failed to invalidate cache entries by user");
    }

    /// 使指定会话的缓存失效（用于撤销单个会话）
    pub async fn invalidate_by_session(&self, session_id: uuid::Uuid) {
        self.cache
            .invalidate_entries_if(move |_key, entry| entry.session_id == session_id)
            .expect("Failed to invalidate cache entries by session");
    }

    /// 获取缓存统计信息（用于监控）
    pub fn stats(&self) -> CacheStats {
        let entry_count = self.cache.entry_count();
        let weighted_size = self.cache.weighted_size();

        CacheStats {
            entry_count,
            weighted_size,
        }
    }

    /// 清空所有缓存
    #[allow(dead_code)]
    pub async fn clear(&self) {
        self.cache.invalidate_all();
        // 等待后台清理完成
        self.cache.run_pending_tasks().await;
    }
}

/// 缓存统计信息
#[derive(Debug, Clone)]
pub struct CacheStats {
    pub entry_count: u64,
    pub weighted_size: u64,
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn test_auth_cache_basic() {
        let cache = AuthCache::new(100, 60);
        let token = "test_token".to_string();
        let entry = AuthCacheEntry {
            user_id: 123,
            session_id: uuid::Uuid::new_v4(),
        };

        // 测试设置和获取
        cache.set(token.clone(), entry.clone()).await;
        let result = cache.get(&token).await;
        assert!(result.is_some());
        assert_eq!(result.unwrap().user_id, 123);

        // 测试失效
        cache.invalidate(&token).await;
        let result = cache.get(&token).await;
        assert!(result.is_none());
    }

    #[tokio::test]
    async fn test_invalidate_by_user() {
        let cache = AuthCache::new(100, 60);
        let user_id = 456;
        let session_id_1 = uuid::Uuid::new_v4();
        let session_id_2 = uuid::Uuid::new_v4();

        // 为同一个用户添加多个 token
        cache
            .set(
                "token1".to_string(),
                AuthCacheEntry {
                    user_id,
                    session_id: session_id_1,
                },
            )
            .await;
        cache
            .set(
                "token2".to_string(),
                AuthCacheEntry {
                    user_id,
                    session_id: session_id_2,
                },
            )
            .await;

        // 失效该用户的所有缓存
        cache.invalidate_by_user(user_id).await;

        // 验证都被清除了
        assert!(cache.get("token1").await.is_none());
        assert!(cache.get("token2").await.is_none());
    }

    #[tokio::test]
    async fn test_invalidate_by_session() {
        let cache = AuthCache::new(100, 60);
        let session_id = uuid::Uuid::new_v4();
        let entry = AuthCacheEntry {
            user_id: 789,
            session_id,
        };

        cache.set("token".to_string(), entry).await;

        // 通过 session_id 失效缓存
        cache.invalidate_by_session(session_id).await;

        assert!(cache.get("token").await.is_none());
    }
}
