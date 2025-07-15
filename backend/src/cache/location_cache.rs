//! 位置缓存实现
use redis::{AsyncCommands, Client};
use std::sync::Arc;
use std::future::Future;
use serde_json::{json, Value};

use crate::cache::{Cache, CacheError};

/// 位置缓存接口
pub trait LocationCache: Cache {
    /// 存储用户位置
    #[allow(dead_code)]
    fn store_user_location(
        &self,
        user_id: i64,
        latitude: f64,
        longitude: f64,
        expire_seconds: u32
    ) -> impl Future<Output = Result<(), CacheError>> + Send;
    
    /// 获取附近的用户
    #[allow(dead_code)]
    fn find_nearby_users(
        &self,
        latitude: f64,
        longitude: f64,
        radius_meters: f64
    ) -> impl Future<Output = Result<Vec<(i64, f64, f64)>, CacheError>> + Send;
    
    /// 存储签到位置
    fn store_checkin_location(
        &self,
        checkin_id: String,
        user_id: i64,
        latitude: f64,
        longitude: f64,
        expire_days: u32
    ) -> impl Future<Output = Result<(), CacheError>> + Send;
}

/// Redis实现的位置缓存
pub struct RedisLocationCache {
    /// Redis客户端
    client: Arc<Client>,
}

impl RedisLocationCache {
    /// 创建新的Redis位置缓存
    pub fn new(client: Arc<Client>) -> Self {
        Self { client }
    }
}

impl Cache for RedisLocationCache {
    fn name(&self) -> &str {
        "RedisLocationCache"
    }
}

impl LocationCache for RedisLocationCache {
    async fn store_user_location(
        &self,
        user_id: i64,
        latitude: f64,
        longitude: f64,
        expire_seconds: u32
    ) -> Result<(), CacheError> {
        let key = format!("user:location:{}", user_id);
        let mut conn = self.client.get_multiplexed_async_connection().await
            .map_err(CacheError::Redis)?;
        
        // 使用Redis GEO添加或更新用户位置
        // 使用"users_geo"作为地理空间集合名称
        let _: () = redis::cmd("GEOADD")
            .arg("users_geo")
            .arg(longitude)
            .arg(latitude)
            .arg(user_id.to_string())
            .query_async(&mut conn)
            .await
            .map_err(CacheError::Redis)?;
        
        // 同时保存详细信息
        let json_data = json!({
            "user_id": user_id,
            "latitude": latitude,
            "longitude": longitude,
            "updated_at": chrono::Utc::now().timestamp(),
        });
        
        let _: () = redis::cmd("SET")
            .arg(&key)
            .arg(json_data.to_string())
            .query_async(&mut conn)
            .await
            .map_err(CacheError::Redis)?;
            
        let _: () = redis::cmd("EXPIRE")
            .arg(&key)
            .arg(expire_seconds as i64)
            .query_async(&mut conn)
            .await
            .map_err(CacheError::Redis)?;
            
        Ok(())
    }
    
    async fn find_nearby_users(
        &self,
        latitude: f64,
        longitude: f64,
        radius_meters: f64
    ) -> Result<Vec<(i64, f64, f64)>, CacheError> {
        let mut conn = self.client.get_multiplexed_async_connection().await
            .map_err(CacheError::Redis)?;
        
        // 使用Redis GEORADIUS命令查找附近用户
        // 将半径转换为千米单位
        let radius_km = radius_meters / 1000.0;
        
        // 更优雅地解析结果为 Vec<(String, String)>
        let results: Vec<(String, String)> = redis::cmd("GEORADIUS")
            .arg("users_geo")
            .arg(longitude)
            .arg(latitude)
            .arg(radius_km)
            .arg("km")
            .arg("WITHDIST")
            .query_async(&mut conn)
            .await
            .map_err(CacheError::Redis)?;
        
        let mut nearby_users = Vec::new();
        for (user_id_str, _distance_str) in results {
            if let Ok(user_id) = user_id_str.parse::<i64>() {
                let key = format!("user:location:{}", user_id);
                let location_data: Option<String> = conn.get(&key).await
                    .map_err(CacheError::Redis)?;
                if let Some(location_json) = location_data {
                    let location: Value = serde_json::from_str(&location_json)
                        .map_err(CacheError::Serialization)?;
                    let lat = location["latitude"].as_f64()
                        .ok_or_else(|| CacheError::Other("Invalid latitude value".to_string()))?;
                    let lng = location["longitude"].as_f64()
                        .ok_or_else(|| CacheError::Other("Invalid longitude value".to_string()))?;
                    nearby_users.push((user_id, lat, lng));
                }
            }
        }
        Ok(nearby_users)
    }
    
    async fn store_checkin_location(
        &self,
        checkin_id: String,
        user_id: i64,
        latitude: f64,
        longitude: f64,
        expire_days: u32
    ) -> Result<(), CacheError> {
        let key = format!("checkin:location:{}", checkin_id);
        let mut conn = self.client.get_multiplexed_async_connection().await
            .map_err(CacheError::Redis)?;
        
        // 使用Redis GEO添加签到位置
        // 使用"checkins_geo"作为签到地理空间集合名称
        let _: () = redis::cmd("GEOADD")
            .arg("checkins_geo")
            .arg(longitude)
            .arg(latitude)
            .arg(&checkin_id)
            .query_async(&mut conn)
            .await
            .map_err(CacheError::Redis)?;
        
        // 同时保存详细信息
        let json_data = json!({
            "checkin_id": checkin_id,
            "user_id": user_id,
            "latitude": latitude,
            "longitude": longitude,
            "created_at": chrono::Utc::now().timestamp(),
        });
        
        let _: () = redis::cmd("SET")
            .arg(&key)
            .arg(json_data.to_string())
            .query_async(&mut conn)
            .await
            .map_err(CacheError::Redis)?;
        
        // 设置过期时间（天为单位）
        let expire_seconds = expire_days as i64 * 24 * 60 * 60;
        let _: () = redis::cmd("EXPIRE")
            .arg(&key)
            .arg(expire_seconds)
            .query_async(&mut conn)
            .await
            .map_err(CacheError::Redis)?;
        
        Ok(())
    }
}
