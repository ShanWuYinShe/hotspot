/// 附近通知策略服务
/// 
/// 提供智能的地理位置通知功能，包括距离计算、智能推送、用户偏好等

use async_trait::async_trait;
use std::sync::Arc;
use std::collections::HashMap;
use tracing::{debug, info, warn};
use serde::{Serialize, Deserialize};
use chrono::{DateTime, Utc, Duration as ChronoDuration, Timelike};

use crate::domain::{DomainResult, DomainError};
use crate::ports::{
    UserRepositoryPort, 
    NotificationPolicyPort, 
    NotificationPolicy, 
    NotificationContext,
    LocationContext,
};
use crate::domain::constants::location_constants::{
    DEFAULT_LATITUDE, 
    DEFAULT_LONGITUDE,
    notification::{
        DEFAULT_NOTIFICATION_RADIUS_KM, 
        MAX_NOTIFICATION_RADIUS_KM, 
        MIN_NOTIFICATION_RADIUS_KM
    }
};

/// 附近通知配置
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NearbyNotificationConfig {
    /// 默认通知半径（公里）
    pub default_radius_km: f64,
    /// 最大通知半径（公里）
    pub max_radius_km: f64,
    /// 最小通知半径（公里）
    pub min_radius_km: f64,
    /// 最大通知用户数
    pub max_notification_users: usize,
    /// 通知频率限制（分钟）
    pub notification_cooldown_minutes: i64,
    /// 是否启用智能半径调整
    pub enable_smart_radius: bool,
    /// 是否启用用户偏好过滤
    pub enable_user_preference_filter: bool,
}

impl Default for NearbyNotificationConfig {
    fn default() -> Self {
        Self {
            default_radius_km: DEFAULT_NOTIFICATION_RADIUS_KM,
            max_radius_km: MAX_NOTIFICATION_RADIUS_KM,
            min_radius_km: MIN_NOTIFICATION_RADIUS_KM,
            max_notification_users: 100,
            notification_cooldown_minutes: 5,
            enable_smart_radius: true,
            enable_user_preference_filter: true,
        }
    }
}

/// 用户通知偏好
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UserNotificationPreference {
    pub user_id: i64,
    /// 是否接收附近通知
    pub receive_nearby_notifications: bool,
    /// 接收通知的最大距离（公里）
    pub max_notification_distance_km: f64,
    /// 通知时间段限制
    pub notification_hours: Option<(u8, u8)>, // (开始小时, 结束小时)
    /// 最后接收通知时间
    pub last_notification_at: Option<DateTime<Utc>>,
}

impl Default for UserNotificationPreference {
    fn default() -> Self {
        Self {
            user_id: 0,
            receive_nearby_notifications: true,
            max_notification_distance_km: DEFAULT_NOTIFICATION_RADIUS_KM,
            notification_hours: Some((8, 22)), // 8:00 - 22:00
            last_notification_at: None,
        }
    }
}

/// 通知距离信息
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct NotificationDistance {
    pub user_id: i64,
    pub distance_km: f64,
    pub bearing_degrees: f64, // 方位角
    pub estimated_travel_time_minutes: Option<u32>,
}

/// 附近通知策略服务
pub struct NearbyNotificationService<U>
where
    U: UserRepositoryPort,
{
    user_repository: Arc<U>,
    config: NearbyNotificationConfig,
    user_preferences: HashMap<i64, UserNotificationPreference>,
    notification_history: HashMap<i64, DateTime<Utc>>,
}

impl<U> NearbyNotificationService<U>
where
    U: UserRepositoryPort,
{
    /// 创建新的附近通知服务
    pub fn new(user_repository: Arc<U>, config: Option<NearbyNotificationConfig>) -> Self {
        Self {
            user_repository,
            config: config.unwrap_or_default(),
            user_preferences: HashMap::new(),
            notification_history: HashMap::new(),
        }
    }

    /// 获取附近用户并计算距离
    pub async fn get_nearby_users_with_distance(
        &self,
        center_lat: f64,
        center_lng: f64,
        radius_km: f64,
    ) -> DomainResult<Vec<NotificationDistance>> {
        debug!(
            "查询附近用户: 中心点({}, {}), 半径{}km",
            center_lat, center_lng, radius_km
        );

        // 验证半径范围
        let validated_radius = self.validate_radius(radius_km);
        
        // 查询附近用户
        let nearby_users = self.user_repository
            .find_nearby_users(center_lat, center_lng, validated_radius, 0, self.config.max_notification_users as i32)
            .await?;

        let mut distances = Vec::new();
        
        for user in nearby_users {
            // 获取用户位置
            if let Some(user_location) = self.get_user_location(user.id).await? {
                let distance = self.calculate_distance(
                    center_lat, center_lng,
                    user_location.latitude, user_location.longitude,
                );
                
                let bearing = self.calculate_bearing(
                    center_lat, center_lng,
                    user_location.latitude, user_location.longitude,
                );

                distances.push(NotificationDistance {
                    user_id: user.id,
                    distance_km: distance,
                    bearing_degrees: bearing,
                    estimated_travel_time_minutes: self.estimate_travel_time(distance),
                });
            }
        }

        // 按距离排序
        distances.sort_by(|a, b| a.distance_km.partial_cmp(&b.distance_km).unwrap());

        debug!("找到{}个附近用户", distances.len());
        Ok(distances)
    }

    /// 智能附近通知推送
    pub async fn smart_nearby_notification(
        &mut self,
        trigger_user_id: i64,
        center_lat: f64,
        center_lng: f64,
        notification_type: &str,
        _content: &str,
    ) -> DomainResult<Vec<i64>> {
        info!(
            "智能附近通知: 用户{} 在({}, {}) 发送通知",
            trigger_user_id, center_lat, center_lng
        );

        // 检查通知频率限制
        if !self.check_notification_cooldown(trigger_user_id) {
            warn!("用户{}通知频率过高，跳过此次通知", trigger_user_id);
            return Ok(vec![]);
        }

        // 智能半径调整
        let optimal_radius = if self.config.enable_smart_radius {
            self.calculate_optimal_radius(center_lat, center_lng, notification_type).await?
        } else {
            self.config.default_radius_km
        };

        // 获取附近用户
        let nearby_distances = self.get_nearby_users_with_distance(
            center_lat, center_lng, optimal_radius
        ).await?;

        // 应用用户偏好过滤
        let filtered_users = if self.config.enable_user_preference_filter {
            self.filter_by_user_preferences(&nearby_distances, notification_type).await?
        } else {
            nearby_distances.into_iter().map(|d| d.user_id).collect()
        };

        // 排除触发用户自己
        let target_users: Vec<i64> = filtered_users
            .into_iter()
            .filter(|&user_id| user_id != trigger_user_id)
            .collect();

        // 更新通知历史
        self.update_notification_history(trigger_user_id);

        info!(
            "智能附近通知完成: 半径{}km, 目标用户{}个",
            optimal_radius, target_users.len()
        );

        Ok(target_users)
    }

    /// 计算最优通知半径
    async fn calculate_optimal_radius(
        &self,
        center_lat: f64,
        center_lng: f64,
        notification_type: &str,
    ) -> DomainResult<f64> {
        // 根据通知类型调整基础半径
        let base_radius = match notification_type {
            "urgent" | "emergency" => self.config.max_radius_km * 0.8, // 紧急通知扩大范围
            "social" | "moment" => self.config.default_radius_km * 0.6, // 社交通知缩小范围
            "location" | "checkin" => self.config.default_radius_km * 1.2, // 位置通知适度扩大
            _ => self.config.default_radius_km,
        };

        // 根据用户密度动态调整
        let user_density = self.calculate_user_density(center_lat, center_lng).await?;
        let density_factor = if user_density > 50.0 {
            0.8 // 高密度区域缩小半径
        } else if user_density < 10.0 {
            1.5 // 低密度区域扩大半径
        } else {
            1.0 // 中等密度保持不变
        };

        let optimal_radius = (base_radius * density_factor)
            .max(self.config.min_radius_km)
            .min(self.config.max_radius_km);

        debug!(
            "计算最优半径: 类型={}, 基础半径={:.2}km, 密度因子={:.2}, 最终半径={:.2}km",
            notification_type, base_radius, density_factor, optimal_radius
        );

        Ok(optimal_radius)
    }

    /// 计算用户密度（每平方公里用户数）
    async fn calculate_user_density(&self, lat: f64, lng: f64) -> DomainResult<f64> {
        let sample_radius = 2.0; // 2公里采样半径
        let nearby_users = self.user_repository
            .find_nearby_users(lat, lng, sample_radius, 0, 200)
            .await?;

        let area_km2 = std::f64::consts::PI * sample_radius * sample_radius;
        let density = nearby_users.len() as f64 / area_km2;

        debug!("用户密度: {}用户/km²", density);
        Ok(density)
    }

    /// 根据用户偏好过滤通知目标
    async fn filter_by_user_preferences(
        &self,
        distances: &[NotificationDistance],
        _notification_type: &str,
    ) -> DomainResult<Vec<i64>> {
        let mut filtered_users = Vec::new();

        for distance in distances {
            let preference = self.get_user_preference(distance.user_id).await;
            
            // 检查是否接收附近通知
            if !preference.receive_nearby_notifications {
                continue;
            }

            // 检查距离限制
            if distance.distance_km > preference.max_notification_distance_km {
                continue;
            }

            // 检查时间限制
            if let Some((start_hour, end_hour)) = preference.notification_hours {
                let current_hour = Utc::now().hour() as u8;
                if current_hour < start_hour || current_hour > end_hour {
                    continue;
                }
            }

            // 检查通知频率限制
            if let Some(last_notification) = preference.last_notification_at {
                let cooldown = ChronoDuration::minutes(self.config.notification_cooldown_minutes);
                if Utc::now() - last_notification < cooldown {
                    continue;
                }
            }

            filtered_users.push(distance.user_id);
        }

        debug!(
            "用户偏好过滤: 原始{}个用户, 过滤后{}个用户",
            distances.len(), filtered_users.len()
        );

        Ok(filtered_users)
    }

    /// 获取用户位置信息
    async fn get_user_location(&self, _user_id: i64) -> DomainResult<Option<LocationContext>> {
        // 这里应该查询用户的当前位置
        // 简化实现，返回默认位置
        Ok(Some(LocationContext {
            latitude: DEFAULT_LATITUDE,
            longitude: DEFAULT_LONGITUDE,
            address: Some("默认位置".to_string()),
        }))
    }

    /// 获取用户通知偏好
    async fn get_user_preference(&self, user_id: i64) -> UserNotificationPreference {
        self.user_preferences.get(&user_id)
            .cloned()
            .unwrap_or_else(|| {
                let mut pref = UserNotificationPreference::default();
                pref.user_id = user_id;
                pref
            })
    }

    /// 验证半径范围
    fn validate_radius(&self, radius_km: f64) -> f64 {
        radius_km
            .max(self.config.min_radius_km)
            .min(self.config.max_radius_km)
    }

    /// 检查通知冷却时间
    fn check_notification_cooldown(&self, user_id: i64) -> bool {
        if let Some(last_notification) = self.notification_history.get(&user_id) {
            let cooldown = ChronoDuration::minutes(self.config.notification_cooldown_minutes);
            Utc::now() - *last_notification >= cooldown
        } else {
            true
        }
    }

    /// 更新通知历史
    fn update_notification_history(&mut self, user_id: i64) {
        self.notification_history.insert(user_id, Utc::now());
    }

    /// 计算两点间距离（公里）
    fn calculate_distance(&self, lat1: f64, lng1: f64, lat2: f64, lng2: f64) -> f64 {
        const EARTH_RADIUS_KM: f64 = 6371.0;

        let lat1_rad = lat1.to_radians();
        let lat2_rad = lat2.to_radians();
        let delta_lat = (lat2 - lat1).to_radians();
        let delta_lng = (lng2 - lng1).to_radians();

        let a = (delta_lat / 2.0).sin().powi(2) +
            lat1_rad.cos() * lat2_rad.cos() * (delta_lng / 2.0).sin().powi(2);
        let c = 2.0 * a.sqrt().atan2((1.0 - a).sqrt());

        EARTH_RADIUS_KM * c
    }

    /// 计算方位角（度）
    fn calculate_bearing(&self, lat1: f64, lng1: f64, lat2: f64, lng2: f64) -> f64 {
        let lat1_rad = lat1.to_radians();
        let lat2_rad = lat2.to_radians();
        let delta_lng = (lng2 - lng1).to_radians();

        let y = delta_lng.sin() * lat2_rad.cos();
        let x = lat1_rad.cos() * lat2_rad.sin() - 
                lat1_rad.sin() * lat2_rad.cos() * delta_lng.cos();

        let bearing_rad = y.atan2(x);
        (bearing_rad.to_degrees() + 360.0) % 360.0
    }

    /// 估算步行时间（分钟）
    fn estimate_travel_time(&self, distance_km: f64) -> Option<u32> {
        const WALKING_SPEED_KMH: f64 = 5.0; // 步行速度 5km/h
        
        if distance_km > 10.0 {
            None // 距离太远，不提供步行时间估算
        } else {
            Some((distance_km / WALKING_SPEED_KMH * 60.0) as u32)
        }
    }

    /// 设置用户通知偏好
    pub fn set_user_preference(&mut self, preference: UserNotificationPreference) {
        self.user_preferences.insert(preference.user_id, preference);
    }

    /// 获取服务统计信息
    pub fn get_service_stats(&self) -> HashMap<String, serde_json::Value> {
        let mut stats = HashMap::new();
        
        stats.insert("total_users_with_preferences".to_string(), 
                    serde_json::Value::Number(self.user_preferences.len().into()));
        
        stats.insert("notification_history_size".to_string(),
                    serde_json::Value::Number(self.notification_history.len().into()));
        
        stats.insert("config".to_string(),
                    serde_json::to_value(&self.config).unwrap_or_default());

        stats
    }
}

/// 为附近通知服务实现通知策略端口
#[async_trait]
impl<U> NotificationPolicyPort for NearbyNotificationService<U>
where
    U: UserRepositoryPort,
{
    async fn get_target_users(
        &self,
        policy: NotificationPolicy,
        _context: NotificationContext,
    ) -> DomainResult<Vec<i64>> {
        match policy {
            NotificationPolicy::Nearby { latitude, longitude, radius_km } => {
                let distances = self.get_nearby_users_with_distance(
                    latitude, longitude, radius_km
                ).await?;
                
                Ok(distances.into_iter().map(|d| d.user_id).collect())
            }
            _ => {
                // 其他策略不在此服务处理
                Err(DomainError::BusinessRuleViolation {
                    rule: "NearbyNotificationService只处理Nearby策略".to_string()
                })
            }
        }
    }

    async fn should_notify_user(
        &self,
        user_id: i64,
        _notification_type: crate::ports::NotificationType,
        context: &NotificationContext,
    ) -> DomainResult<bool> {
        let preference = self.get_user_preference(user_id).await;
        
        // 基础检查
        if !preference.receive_nearby_notifications {
            return Ok(false);
        }

        // 检查时间限制
        if let Some((start_hour, end_hour)) = preference.notification_hours {
            let current_hour = Utc::now().hour() as u8;
            if current_hour < start_hour || current_hour > end_hour {
                return Ok(false);
            }
        }

        // 检查距离限制（如果有位置上下文）
        if let Some(location) = &context.location {
            if let Some(user_location) = self.get_user_location(user_id).await? {
                let distance = self.calculate_distance(
                    location.latitude, location.longitude,
                    user_location.latitude, user_location.longitude,
                );
                
                if distance > preference.max_notification_distance_km {
                    return Ok(false);
                }
            }
        }

        Ok(true)
    }

    async fn get_policy_stats(&self) -> DomainResult<crate::ports::PolicyStats> {
        let mut policy_usage = HashMap::new();
        policy_usage.insert("nearby".to_string(), self.notification_history.len() as u64);

        Ok(crate::ports::PolicyStats {
            policy_usage,
            avg_target_users: 10.0, // 简化统计
            avg_evaluation_time_ms: 50, // 简化统计
        })
    }
}
