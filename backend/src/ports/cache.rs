//! 缓存端口定义
//! 
//! 定义缓存操作的抽象接口，支持不同的缓存实现（Redis、内存缓存等）

use std::future::Future;
use std::pin::Pin;
use std::time::Duration;
use thiserror::Error;
use serde::{Serialize, Deserialize};

// =============== 类型别名定义 ===============

/// 批量获取结果类型
#[allow(dead_code)]
type MgetResult<'a> = Pin<Box<dyn Future<Output = CachePortResult<Vec<Option<String>>>> + Send + 'a>>;

/// 缓存端口错误类型
#[derive(Debug, Error)]
pub enum CachePortError {
    #[error("connection error: {0}")]
    Connection(String),
    #[error("serialization error: {0}")]
    Serialization(String),
    #[error("key not found: {key}")]
    KeyNotFound { key: String },
    #[error("cache full: {0}")]
    CacheFull(String),
    #[error("invalid ttl: {0}")]
    InvalidTtl(String),
    #[error("other error: {0}")]
    Other(String),
}

pub type CachePortResult<T> = Result<T, CachePortError>;

/// 缓存端口 - 定义缓存操作的抽象接口
pub trait CachePort: Send + Sync {
    /// 获取端口名称
    fn name(&self) -> &str;

    /// 获取缓存值
    fn get<'a>(
        &'a self,
        key: &'a str,
    ) -> Pin<Box<dyn Future<Output = CachePortResult<Option<String>>> + Send + 'a>>;

    /// 设置缓存值
    fn set<'a>(
        &'a self,
        key: &'a str,
        value: &'a str,
        ttl: Option<Duration>,
    ) -> Pin<Box<dyn Future<Output = CachePortResult<()>> + Send + 'a>>;

    /// 删除缓存值
    fn delete<'a>(
        &'a self,
        key: &'a str,
    ) -> Pin<Box<dyn Future<Output = CachePortResult<bool>> + Send + 'a>>;

    /// 检查键是否存在
    fn exists<'a>(
        &'a self,
        key: &'a str,
    ) -> Pin<Box<dyn Future<Output = CachePortResult<bool>> + Send + 'a>>;

    /// 设置过期时间
    fn expire<'a>(
        &'a self,
        key: &'a str,
        ttl: Duration,
    ) -> Pin<Box<dyn Future<Output = CachePortResult<bool>> + Send + 'a>>;

    /// 获取剩余过期时间
    fn ttl<'a>(
        &'a self,
        key: &'a str,
    ) -> Pin<Box<dyn Future<Output = CachePortResult<Option<Duration>>> + Send + 'a>>;

    /// 批量获取
    fn mget<'a>(
        &'a self,
        keys: &'a [String],
    ) -> MgetResult<'a>;

    /// 批量设置
    fn mset<'a>(
        &'a self,
        pairs: &'a [(String, String)],
        ttl: Option<Duration>,
    ) -> Pin<Box<dyn Future<Output = CachePortResult<()>> + Send + 'a>>;

    /// 原子递增
    fn incr<'a>(
        &'a self,
        key: &'a str,
        delta: i64,
    ) -> Pin<Box<dyn Future<Output = CachePortResult<i64>> + Send + 'a>>;

    /// 原子递减
    fn decr<'a>(
        &'a self,
        key: &'a str,
        delta: i64,
    ) -> Pin<Box<dyn Future<Output = CachePortResult<i64>> + Send + 'a>>;

    /// 清空所有缓存
    fn flush_all<'a>(
        &'a self,
    ) -> Pin<Box<dyn Future<Output = CachePortResult<()>> + Send + 'a>>;

    /// 获取缓存统计信息
    fn stats<'a>(
        &'a self,
    ) -> Pin<Box<dyn Future<Output = CachePortResult<CacheStats>> + Send + 'a>>;
}

/// 缓存统计信息
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CacheStats {
    /// 总键数
    pub total_keys: usize,
    /// 命中次数
    pub hits: u64,
    /// 未命中次数
    pub misses: u64,
    /// 命中率
    pub hit_rate: f64,
    /// 内存使用量（字节）
    pub memory_usage: usize,
    /// 最大内存限制（字节）
    pub max_memory: Option<usize>,
    /// 过期键数
    pub expired_keys: u64,
    /// 驱逐键数
    pub evicted_keys: u64,
}

impl CacheStats {
    /// 创建默认统计信息
    pub fn new() -> Self {
        Self {
            total_keys: 0,
            hits: 0,
            misses: 0,
            hit_rate: 0.0,
            memory_usage: 0,
            max_memory: None,
            expired_keys: 0,
            evicted_keys: 0,
        }
    }

    /// 计算命中率
    pub fn calculate_hit_rate(&mut self) {
        let total = self.hits + self.misses;
        if total > 0 {
            self.hit_rate = self.hits as f64 / total as f64;
        } else {
            self.hit_rate = 0.0;
        }
    }
}

impl CachePortError {
    /// 从连接错误创建
    pub fn from_connection_error(err: impl std::fmt::Display) -> Self {
        Self::Connection(err.to_string())
    }

    /// 从序列化错误创建
    pub fn from_serialization_error(err: impl std::fmt::Display) -> Self {
        Self::Serialization(err.to_string())
    }

    /// 创建键未找到错误
    pub fn key_not_found(key: impl Into<String>) -> Self {
        Self::KeyNotFound { key: key.into() }
    }

    /// 创建缓存已满错误
    pub fn cache_full(msg: impl Into<String>) -> Self {
        Self::CacheFull(msg.into())
    }

    /// 创建无效TTL错误
    pub fn invalid_ttl(msg: impl Into<String>) -> Self {
        Self::InvalidTtl(msg.into())
    }

    /// 从缓存管理器错误转换
    pub fn from_cache_error(err: crate::cache::CacheError) -> Self {
        match err {
            crate::cache::CacheError::Redis(e) => Self::Connection(e.to_string()),
            crate::cache::CacheError::Serialization(e) => Self::Serialization(e.to_string()),
            crate::cache::CacheError::ConnectionUnavailable => Self::Connection("Cache connection unavailable".to_string()),
            crate::cache::CacheError::KeyNotFound => Self::KeyNotFound { key: "unknown".to_string() },
            crate::cache::CacheError::Other(msg) => Self::Other(msg),
        }
    }
}

impl Default for CacheStats {
    fn default() -> Self {
        Self::new()
    }
}

/// 缓存键构建器 - 提供统一的缓存键命名规范
pub struct CacheKeyBuilder;

impl CacheKeyBuilder {
    /// 用户相关缓存键
    pub fn user_key(user_id: i64, suffix: &str) -> String {
        format!("user:{}:{}", user_id, suffix)
    }

    /// 群组相关缓存键
    pub fn group_key(group_id: &str, suffix: &str) -> String {
        format!("group:{}:{}", group_id, suffix)
    }

    /// 消息相关缓存键
    pub fn message_key(conversation_id: &str, suffix: &str) -> String {
        format!("message:{}:{}", conversation_id, suffix)
    }

    /// 动态相关缓存键
    pub fn moment_key(moment_id: &str, suffix: &str) -> String {
        format!("moment:{}:{}", moment_id, suffix)
    }

    /// 会话相关缓存键
    pub fn session_key(session_id: &str) -> String {
        format!("session:{}", session_id)
    }

    /// 权限相关缓存键
    pub fn permission_key(user_id: i64, resource: &str) -> String {
        format!("permission:{}:{}", user_id, resource)
    }

    /// 统计相关缓存键
    pub fn stats_key(category: &str, period: &str) -> String {
        format!("stats:{}:{}", category, period)
    }

    /// 临时数据缓存键
    pub fn temp_key(category: &str, id: &str) -> String {
        format!("temp:{}:{}", category, id)
    }
}
