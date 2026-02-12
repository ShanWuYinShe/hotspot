//! Redis 缓存工具
//!
//! 提供高级缓存功能：
//! - 缓存穿透保护（空值缓存）
//! - 缓存雪崩保护（随机过期时间）
//! - 缓存击穿保护（互斥锁）
//! - 批量操作优化

use redis::aio::ConnectionManager;
use redis::AsyncCommands;
use serde::{de::DeserializeOwned, Serialize};
use std::sync::atomic::{AtomicU64, Ordering};
use std::sync::Arc;
use std::time::Duration;

/// 空值标记
const NULL_MARKER: &str = "__NULL__";

/// 缓存指标
#[derive(Debug, Clone)]
pub struct CacheMetrics {
    pub hits: Arc<AtomicU64>,
    pub misses: Arc<AtomicU64>,
    pub errors: Arc<AtomicU64>,
}

impl Default for CacheMetrics {
    fn default() -> Self {
        Self {
            hits: Arc::new(AtomicU64::new(0)),
            misses: Arc::new(AtomicU64::new(0)),
            errors: Arc::new(AtomicU64::new(0)),
        }
    }
}

impl CacheMetrics {
    pub fn get_hit_rate(&self) -> f64 {
        let hits = self.hits.load(Ordering::Relaxed);
        let misses = self.misses.load(Ordering::Relaxed);
        let total = hits + misses;

        if total == 0 {
            0.0
        } else {
            hits as f64 / total as f64
        }
    }
}

/// 缓存配置
#[derive(Debug, Clone)]
pub struct CacheConfig {
    /// 基础过期时间（秒）
    pub base_ttl_secs: u64,
    /// 过期时间随机波动范围（秒），用于防止缓存雪崩
    pub ttl_jitter_secs: u64,
    /// 锁超时时间（秒），用于防止缓存击穿
    pub lock_timeout_secs: u64,
}

impl Default for CacheConfig {
    fn default() -> Self {
        Self {
            base_ttl_secs: 3600,  // 1 小时
            ttl_jitter_secs: 600, // 10 分钟随机波动
            lock_timeout_secs: 5, // 锁超时 5 秒
        }
    }
}

/// 缓存键前缀
pub mod cache_keys {
    /// 用户信息缓存
    pub const USER_INFO: &str = "cache:user:info";
    /// 群组信息缓存
    pub const GROUP_INFO: &str = "cache:group:info";
    /// 热门标签缓存
    pub const TRENDING_TAGS: &str = "cache:trending:tags";
    /// 系统配置缓存
    pub const SYSTEM_CONFIG: &str = "cache:system:config";
}

/// Redis 缓存管理器
#[derive(Clone)]
pub struct CacheManager {
    redis: ConnectionManager,
    config: CacheConfig,
    metrics: CacheMetrics,
}

impl CacheManager {
    pub fn new(redis: ConnectionManager) -> Self {
        Self::with_config(redis, CacheConfig::default())
    }

    pub fn with_config(redis: ConnectionManager, config: CacheConfig) -> Self {
        Self {
            redis,
            config,
            metrics: CacheMetrics::default(),
        }
    }

    /// 获取缓存指标
    pub fn metrics(&self) -> &CacheMetrics {
        &self.metrics
    }

    /// 生成带随机波动的过期时间
    fn get_ttl_with_jitter(&self) -> u64 {
        let jitter = rand::random::<u64>() % (self.config.ttl_jitter_secs + 1);
        self.config.base_ttl_secs + jitter
    }

    /// 生成指定 TTL 的随机波动过期时间
    fn get_ttl_with_jitter_for(&self, base_ttl_secs: u64) -> u64 {
        if base_ttl_secs == 0 {
            return 0;
        }
        let jitter_cap = self.config.ttl_jitter_secs.min(base_ttl_secs);
        let jitter = rand::random::<u64>() % (jitter_cap + 1);
        base_ttl_secs + jitter
    }

    /// 生成完整的缓存键
    fn make_key(&self, prefix: &str, id: &str) -> String {
        format!("{}:{}", prefix, id)
    }

    /// 获取缓存值
    ///
    /// 返回值：
    /// - Some(Some(value)): 缓存命中，有值
    /// - Some(None): 缓存命中，但值为空（空值缓存）
    /// - None: 缓存未命中
    pub async fn get<T: DeserializeOwned>(
        &self,
        prefix: &str,
        id: &str,
    ) -> Result<Option<Option<T>>, redis::RedisError> {
        let key = self.make_key(prefix, id);
        let mut conn = self.redis.clone();

        let result = conn.get(&key).await;

        // 记录指标
        match &result {
            Ok(Some(_)) => {
                self.metrics.hits.fetch_add(1, Ordering::Relaxed);
            }
            Ok(None) => {
                self.metrics.misses.fetch_add(1, Ordering::Relaxed);
            }
            Err(_) => {
                self.metrics.errors.fetch_add(1, Ordering::Relaxed);
            }
        }

        let value: Option<String> = result?;

        match value {
            Some(v) if v == NULL_MARKER => Ok(Some(None)), // 空值缓存
            Some(v) => {
                match serde_json::from_str(&v) {
                    Ok(parsed) => Ok(Some(Some(parsed))),
                    Err(_) => Ok(None), // 解析失败当作缓存未命中
                }
            }
            None => Ok(None),
        }
    }

    /// 设置缓存值
    pub async fn set<T: Serialize>(
        &self,
        prefix: &str,
        id: &str,
        value: &T,
    ) -> Result<(), redis::RedisError> {
        let key = self.make_key(prefix, id);
        let ttl = self.get_ttl_with_jitter();
        let serialized = serde_json::to_string(value).unwrap_or_default();

        let mut conn = self.redis.clone();
        conn.set_ex::<_, _, ()>(&key, serialized, ttl).await?;
        Ok(())
    }

    /// 设置缓存值（自定义 TTL）
    pub async fn set_with_ttl<T: Serialize>(
        &self,
        prefix: &str,
        id: &str,
        value: &T,
        base_ttl_secs: u64,
    ) -> Result<(), redis::RedisError> {
        let key = self.make_key(prefix, id);
        let ttl = self.get_ttl_with_jitter_for(base_ttl_secs);
        let serialized = serde_json::to_string(value).unwrap_or_default();
        let mut conn = self.redis.clone();
        conn.set_ex::<_, _, ()>(&key, serialized, ttl).await?;
        Ok(())
    }

    /// 设置空值缓存（自定义 TTL）
    pub async fn set_null_with_ttl(
        &self,
        prefix: &str,
        id: &str,
        null_ttl_secs: u64,
    ) -> Result<(), redis::RedisError> {
        let key = self.make_key(prefix, id);
        let mut conn = self.redis.clone();
        conn.set_ex::<_, _, ()>(&key, NULL_MARKER, null_ttl_secs)
            .await?;
        Ok(())
    }

    /// 删除缓存
    pub async fn delete(&self, prefix: &str, id: &str) -> Result<(), redis::RedisError> {
        let key = self.make_key(prefix, id);
        let mut conn = self.redis.clone();
        conn.del::<_, ()>(&key).await?;
        Ok(())
    }

    /// 尝试获取锁（用于防止缓存击穿）
    pub async fn try_lock(&self, lock_name: &str) -> Result<bool, redis::RedisError> {
        let key = format!("lock:{}", lock_name);
        let mut conn = self.redis.clone();

        // SET key value NX EX seconds
        let result: Option<String> = redis::cmd("SET")
            .arg(&key)
            .arg("1")
            .arg("NX")
            .arg("EX")
            .arg(self.config.lock_timeout_secs)
            .query_async(&mut conn)
            .await?;

        Ok(result.is_some())
    }

    /// 释放锁
    pub async fn unlock(&self, lock_name: &str) -> Result<(), redis::RedisError> {
        let key = format!("lock:{}", lock_name);
        let mut conn = self.redis.clone();
        conn.del::<_, ()>(&key).await?;
        Ok(())
    }

    /// 带锁的缓存获取/设置（自定义 TTL）
    pub async fn get_or_set_with_lock_with_ttl<T, F, Fut>(
        &self,
        prefix: &str,
        id: &str,
        base_ttl_secs: u64,
        null_ttl_secs: u64,
        loader: F,
    ) -> Result<Option<T>, Box<dyn std::error::Error + Send + Sync>>
    where
        T: Serialize + DeserializeOwned + Clone,
        F: FnOnce() -> Fut,
        Fut: std::future::Future<
            Output = Result<Option<T>, Box<dyn std::error::Error + Send + Sync>>,
        >,
    {
        if let Some(cached) = self.get::<T>(prefix, id).await? {
            return Ok(cached);
        }

        let lock_name = format!("{}:{}", prefix, id);
        let got_lock = self.try_lock(&lock_name).await?;

        if got_lock {
            if let Some(cached) = self.get::<T>(prefix, id).await? {
                let _ = self.unlock(&lock_name).await;
                return Ok(cached);
            }

            let result = loader().await;
            match &result {
                Ok(Some(value)) => {
                    let _ = self.set_with_ttl(prefix, id, value, base_ttl_secs).await;
                }
                Ok(None) => {
                    let _ = self.set_null_with_ttl(prefix, id, null_ttl_secs).await;
                }
                Err(_) => {}
            }
            let _ = self.unlock(&lock_name).await;
            result
        } else {
            tokio::time::sleep(Duration::from_millis(100)).await;
            if let Some(cached) = self.get::<T>(prefix, id).await? {
                return Ok(cached);
            }
            loader().await
        }
    }

    /// 增加计数器（用于统计）
    pub async fn incr(&self, key: &str, delta: i64) -> Result<i64, redis::RedisError> {
        let mut conn = self.redis.clone();
        conn.incr(key, delta).await
    }

    /// 设置计数器过期时间
    pub async fn expire(&self, key: &str, ttl_secs: u64) -> Result<(), redis::RedisError> {
        let mut conn = self.redis.clone();
        conn.expire::<_, ()>(key, ttl_secs as i64).await?;
        Ok(())
    }

    /// 执行 Lua 脚本
    ///
    /// 用于需要原子性操作的场景（如限流、分布式锁等）
    pub async fn eval_script<T>(
        &self,
        script: &str,
        keys: &[&str],
        args: &[&str],
    ) -> Result<T, redis::RedisError>
    where
        T: redis::FromRedisValue,
    {
        let mut conn = self.redis.clone();

        redis::Script::new(script)
            .key(keys)
            .arg(args)
            .invoke_async(&mut conn)
            .await
    }
}

/// 速率限制器
#[derive(Clone)]
pub struct RateLimiter {
    redis: ConnectionManager,
}

impl RateLimiter {
    pub fn new(redis: ConnectionManager) -> Self {
        Self { redis }
    }

    /// 检查是否超过速率限制
    ///
    /// 使用滑动窗口算法
    pub async fn is_rate_limited(
        &self,
        key: &str,
        max_requests: i64,
        window_secs: i64,
    ) -> Result<bool, redis::RedisError> {
        let full_key = format!("ratelimit:{}", key);
        let now = chrono::Utc::now().timestamp();
        let window_start = now - window_secs;

        let mut conn = self.redis.clone();

        // 使用 Lua 脚本保证原子性
        let script = redis::Script::new(
            r#"
            local key = KEYS[1]
            local now = tonumber(ARGV[1])
            local window_start = tonumber(ARGV[2])
            local max_requests = tonumber(ARGV[3])
            local window_secs = tonumber(ARGV[4])
            
            -- 移除窗口外的记录
            redis.call('ZREMRANGEBYSCORE', key, '-inf', window_start)
            
            -- 获取当前窗口内的请求数
            local current_count = redis.call('ZCARD', key)
            
            if current_count >= max_requests then
                return 1  -- 超过限制
            end
            
            -- 添加当前请求
            redis.call('ZADD', key, now, now .. '-' .. math.random())
            redis.call('EXPIRE', key, window_secs)
            
            return 0  -- 未超过限制
            "#,
        );

        let result: i32 = script
            .key(&full_key)
            .arg(now)
            .arg(window_start)
            .arg(max_requests)
            .arg(window_secs)
            .invoke_async(&mut conn)
            .await?;

        Ok(result == 1)
    }

    /// 获取剩余请求次数
    pub async fn get_remaining(
        &self,
        key: &str,
        max_requests: i64,
        window_secs: i64,
    ) -> Result<i64, redis::RedisError> {
        let full_key = format!("ratelimit:{}", key);
        let now = chrono::Utc::now().timestamp();
        let window_start = now - window_secs;

        let mut conn = self.redis.clone();

        // 移除窗口外的记录并获取计数
        let _: () = conn.zrembyscore(&full_key, "-inf", window_start).await?;
        let count: i64 = conn.zcard(&full_key).await?;

        Ok((max_requests - count).max(0))
    }
}
