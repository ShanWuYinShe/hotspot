//! 用户在线状态服务
//!
//! 使用 Redis 管理用户实时在线状态和位置

#![allow(dead_code)]

use redis::aio::ConnectionManager as RedisConnectionManager;
use redis::AsyncCommands;
use chrono::Utc;

use crate::constants::{
    REDIS_KEY_PRESENCE_ONLINE, REDIS_KEY_PRESENCE_LOCATION, REDIS_TTL_ONLINE_STATUS,
};
use crate::error::{AppError, Result};

pub struct PresenceService {
    redis: RedisConnectionManager,
}

impl PresenceService {
    pub fn new(redis: RedisConnectionManager) -> Self {
        Self { redis }
    }

    /// 标记用户上线
    pub async fn mark_user_online(&mut self, user_id: i64) -> Result<()> {
        let mut conn = self.redis.clone();
        let timestamp = Utc::now().timestamp();
        
        let _: () = conn.zadd(REDIS_KEY_PRESENCE_ONLINE, user_id, timestamp)
            .await
            .map_err(|e| AppError::InternalError(format!("Redis错误: {}", e)))?;
        
        tracing::debug!("用户上线: user_id={}", user_id);
        Ok(())
    }

    /// 标记用户离线
    pub async fn mark_user_offline(&mut self, user_id: i64) -> Result<()> {
        let mut conn = self.redis.clone();
        
        let _: () = conn.zrem(REDIS_KEY_PRESENCE_ONLINE, user_id)
            .await
            .map_err(|e| AppError::InternalError(format!("Redis错误: {}", e)))?;
        
        // 删除位置信息
        let location_key = format!("{}{}", REDIS_KEY_PRESENCE_LOCATION, user_id);
        let _: () = conn.del(&location_key)
            .await
            .map_err(|e| AppError::InternalError(format!("Redis错误: {}", e)))?;
        
        tracing::debug!("用户离线: user_id={}", user_id);
        Ok(())
    }

    /// 检查用户是否在线
    pub async fn is_user_online(&mut self, user_id: i64) -> Result<bool> {
        let mut conn = self.redis.clone();
        
        let score: Option<i64> = conn.zscore(REDIS_KEY_PRESENCE_ONLINE, user_id)
            .await
            .map_err(|e| AppError::InternalError(format!("Redis错误: {}", e)))?;
        
        Ok(score.is_some())
    }

    /// 获取在线用户列表
    pub async fn get_online_users(&mut self) -> Result<Vec<i64>> {
        let mut conn = self.redis.clone();
        
        let users: Vec<i64> = conn.zrange(REDIS_KEY_PRESENCE_ONLINE, 0, -1)
            .await
            .map_err(|e| AppError::InternalError(format!("Redis错误: {}", e)))?;
        
        Ok(users)
    }

    /// 获取在线用户数量
    pub async fn get_online_count(&mut self) -> Result<usize> {
        let mut conn = self.redis.clone();
        
        let count: usize = conn.zcard(REDIS_KEY_PRESENCE_ONLINE)
            .await
            .map_err(|e| AppError::InternalError(format!("Redis错误: {}", e)))?;
        
        Ok(count)
    }

    /// 更新用户位置
    pub async fn update_user_location(
        &mut self,
        user_id: i64,
        latitude: f64,
        longitude: f64,
        location_name: String,
    ) -> Result<()> {
        let mut conn = self.redis.clone();
        let location_key = format!("{}{}", REDIS_KEY_PRESENCE_LOCATION, user_id);
        
        let location_data = serde_json::json!({
            "latitude": latitude,
            "longitude": longitude,
            "location_name": location_name,
            "updated_at": Utc::now().to_rfc3339(),
        });
        
        let location_str = serde_json::to_string(&location_data)
            .map_err(|e| AppError::InternalError(format!("序列化错误: {}", e)))?;
        
        let _: () = conn.set_ex(&location_key, location_str, REDIS_TTL_ONLINE_STATUS)
            .await
            .map_err(|e| AppError::InternalError(format!("Redis错误: {}", e)))?;
        
        tracing::debug!("用户位置更新: user_id={}, location={}", user_id, location_name);
        Ok(())
    }

    /// 获取用户位置
    pub async fn get_user_location(&mut self, user_id: i64) -> Result<Option<UserLocation>> {
        let mut conn = self.redis.clone();
        let location_key = format!("{}{}", REDIS_KEY_PRESENCE_LOCATION, user_id);
        
        let location_str: Option<String> = conn.get(&location_key)
            .await
            .map_err(|e| AppError::InternalError(format!("Redis错误: {}", e)))?;
        
        match location_str {
            Some(s) => {
                let location: UserLocation = serde_json::from_str(&s)
                    .map_err(|e| AppError::InternalError(format!("反序列化错误: {}", e)))?;
                Ok(Some(location))
            }
            None => Ok(None),
        }
    }

    /// 清理离线用户（超过TTL的用户）
    pub async fn cleanup_offline_users(&mut self) -> Result<usize> {
        let mut conn = self.redis.clone();
        let cutoff_time = Utc::now().timestamp() - (REDIS_TTL_ONLINE_STATUS as i64);
        
        let removed: usize = conn.zrembyscore(REDIS_KEY_PRESENCE_ONLINE, "-inf", cutoff_time)
            .await
            .map_err(|e| AppError::InternalError(format!("Redis错误: {}", e)))?;
        
        if removed > 0 {
            tracing::info!("清理离线用户: 数量={}", removed);
        }
        
        Ok(removed)
    }
}

#[derive(Debug, serde::Serialize, serde::Deserialize)]
pub struct UserLocation {
    pub latitude: f64,
    pub longitude: f64,
    pub location_name: String,
    pub updated_at: String,
}






