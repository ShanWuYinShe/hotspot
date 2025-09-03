/// 缓存端口定义
/// 
/// 定义缓存相关的抽象接口

use async_trait::async_trait;
use crate::domain::DomainResult;
use std::time::Duration;

/// 缓存端口
#[async_trait]
pub trait CachePort: Send + Sync {
    /// 获取缓存值
    async fn get<T>(&self, key: &str) -> DomainResult<Option<T>>
    where
        T: serde::de::DeserializeOwned + Send;
    
    /// 设置缓存值
    async fn set<T>(&self, key: &str, value: &T, ttl: Option<Duration>) -> DomainResult<()>
    where
        T: serde::Serialize + Send + Sync;
    
    /// 删除缓存值
    async fn delete(&self, key: &str) -> DomainResult<()>;
    
    /// 批量删除缓存值
    async fn delete_batch(&self, keys: &[&str]) -> DomainResult<()>;
    
    /// 检查缓存键是否存在
    async fn exists(&self, key: &str) -> DomainResult<bool>;
    
    /// 设置缓存过期时间
    async fn expire(&self, key: &str, ttl: Duration) -> DomainResult<bool>;
    
    /// 清空所有缓存
    async fn clear_all(&self) -> DomainResult<()>;
    
    /// 获取缓存统计信息
    async fn get_stats(&self) -> DomainResult<CacheStats>;
}

/// 缓存统计信息
#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct CacheStats {
    /// 缓存命中次数
    pub hits: u64,
    /// 缓存未命中次数
    pub misses: u64,
    /// 当前缓存条目数量
    pub entries: u64,
    /// 内存使用量（字节）
    pub memory_usage: u64,
}

impl CacheStats {
    /// 计算命中率
    pub fn hit_rate(&self) -> f64 {
        let total = self.hits + self.misses;
        if total == 0 {
            0.0
        } else {
            self.hits as f64 / total as f64
        }
    }
}

/// 缓存键构建器
pub struct CacheKeyBuilder;

impl CacheKeyBuilder {
    /// 用户相关的缓存键前缀
    pub const USER_PREFIX: &'static str = "user:";
    /// 群组相关的缓存键前缀
    pub const GROUP_PREFIX: &'static str = "group:";
    /// 消息相关的缓存键前缀
    pub const MESSAGE_PREFIX: &'static str = "message:";
    /// 瞬间相关的缓存键前缀
    pub const MOMENT_PREFIX: &'static str = "moment:";
    /// 会话相关的缓存键前缀
    pub const SESSION_PREFIX: &'static str = "session:";
    
    /// 构建用户缓存键
    pub fn user_key(user_id: i64) -> String {
        format!("{}profile:{}", Self::USER_PREFIX, user_id)
    }
    
    /// 构建用户登录ID索引键
    pub fn user_login_id_key(login_id: &str) -> String {
        format!("{}login_id:{}", Self::USER_PREFIX, login_id)
    }
    
    /// 构建群组缓存键
    pub fn group_key(group_id: i64) -> String {
        format!("{}profile:{}", Self::GROUP_PREFIX, group_id)
    }
    
    /// 构建消息缓存键
    pub fn message_key(message_id: i64) -> String {
        format!("{}content:{}", Self::MESSAGE_PREFIX, message_id)
    }
    
    /// 构建群组消息列表键
    pub fn group_messages_key(group_id: i64, cursor: i64, limit: i32) -> String {
        format!("{}list:{}:{}:{}", Self::MESSAGE_PREFIX, group_id, cursor, limit)
    }
    
    /// 构建瞬间缓存键
    pub fn moment_key(moment_id: i64) -> String {
        format!("{}content:{}", Self::MOMENT_PREFIX, moment_id)
    }
    
    /// 构建附近用户搜索结果键
    pub fn nearby_users_key(latitude: f64, longitude: f64, radius_km: f64) -> String {
        format!("{}nearby:{:.6}:{:.6}:{:.2}", Self::USER_PREFIX, latitude, longitude, radius_km)
    }
    
    /// 构建附近群组搜索结果键
    pub fn nearby_groups_key(latitude: f64, longitude: f64, radius_km: f64) -> String {
        format!("{}nearby:{:.6}:{:.6}:{:.2}", Self::GROUP_PREFIX, latitude, longitude, radius_km)
    }
    
    /// 构建会话缓存键
    pub fn session_key(user_id: i64) -> String {
        format!("{}user:{}", Self::SESSION_PREFIX, user_id)
    }
}

