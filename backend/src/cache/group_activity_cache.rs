//! Redis实现的群组活动缓存
use std::sync::Arc;
use uuid::Uuid;

use crate::cache::{Cache, GroupActivityCache, CacheError};

/// Redis实现的群组活动缓存
pub struct RedisGroupActivityCache {
    /// Redis连接管理器
    redis: Option<Arc<redis::aio::ConnectionManager>>,
}

impl RedisGroupActivityCache {
    /// 创建新的Redis群组活动缓存
    pub fn new(redis: Option<Arc<redis::aio::ConnectionManager>>) -> Self {
        Self { redis }
    }
}

impl Cache for RedisGroupActivityCache {
    fn name(&self) -> &str {
        "redis_group_activity_cache"
    }
}

impl GroupActivityCache for RedisGroupActivityCache {
    async fn record_user_group_activity(
        &self, 
        group_id: Uuid, 
        user_id: i64, 
        activity_type: &str,
        timestamp: i64,
        score: i32
    ) -> Result<(), CacheError> {
        let redis = match &self.redis {
            Some(redis) => redis,
            None => return Err(CacheError::ConnectionUnavailable),
        };
        
        let mut conn = (**redis).clone();
        
        // 1. 记录活动到用户活动列表
        let activity_json = serde_json::json!({
            "type": activity_type,
            "timestamp": timestamp,
            "group_id": group_id.to_string()
        });
        
        // 使用Redis存储最近的活动
        let _: () = redis::cmd("LPUSH")
            .arg(format!("group:{}:user:{}:activities", group_id, user_id))
            .arg(activity_json.to_string())
            .query_async(&mut conn)
            .await?;
        
        // 限制活动列表长度，只保留最近的50条记录
        let _: () = redis::cmd("LTRIM")
            .arg(format!("group:{}:user:{}:activities", group_id, user_id))
            .arg(0)
            .arg(49)
            .query_async(&mut conn)
            .await?;
            
        // 2. 更新群组成员活跃度排名
        let _: () = redis::cmd("ZINCRBY")
            .arg(format!("group:{}:activity", group_id))
            .arg(score)
            .arg(user_id.to_string())
            .query_async(&mut conn)
            .await?;
        
        // 3. 设置过期时间 (90天)
        let _: () = redis::cmd("EXPIRE")
            .arg(format!("group:{}:activity", group_id))
            .arg(86400 * 90)
            .query_async(&mut conn)
            .await?;
            
        Ok(())
    }
    
    async fn update_user_last_active_in_group(
        &self, 
        user_id: i64, 
        group_id: Uuid, 
        timestamp: i64
    ) -> Result<(), CacheError> {
        let redis = match &self.redis {
            Some(redis) => redis,
            None => return Err(CacheError::ConnectionUnavailable),
        };
        
        let mut conn = (**redis).clone();
        
        // 更新用户的"最后在群组中活跃"时间
        let _: () = redis::cmd("HSET")
            .arg(format!("user:{}:group_activity", user_id))
            .arg(group_id.to_string())
            .arg(timestamp.to_string())
            .query_async(&mut conn)
            .await?;
            
        Ok(())
    }
    
    async fn get_group_activity_ranking(
        &self, 
        group_id: Uuid,
        limit: usize
    ) -> Result<Vec<(i64, f64)>, CacheError> {
        let redis = match &self.redis {
            Some(redis) => redis,
            None => return Err(CacheError::ConnectionUnavailable),
        };
        
        let mut conn = (**redis).clone();
        
        // 获取群组活跃度排行榜
        let result: Vec<(String, f64)> = redis::cmd("ZREVRANGE")
            .arg(format!("group:{}:activity", group_id))
            .arg(0)
            .arg(limit as isize - 1)
            .arg("WITHSCORES")
            .query_async(&mut conn)
            .await?;
            
        // 转换用户ID格式
        let rankings = result.into_iter()
            .filter_map(|(user_id_str, score)| {
                user_id_str.parse::<i64>().ok().map(|user_id| (user_id, score))
            })
            .collect();
            
        Ok(rankings)
    }
}
