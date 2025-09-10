/// 用户位置领域实体
/// 
/// 管理用户的地理位置信息
/// 用户位置领域实体
/// 
/// 功能职责：
/// - 定义用户地理位置的核心业务实体
/// - 实现位置更新、验证和隐私控制
/// - 管理位置历史和轨迹记录
/// - 提供基于位置的距离计算和搜索逻辑
/// 
/// 业务规则：
/// - 位置坐标的精度和有效性验证
/// - 位置更新频率的限制和控制
/// - 位置隐私设置和可见性管理
/// - 位置数据的加密和安全保护
/// - 位置历史的保存时间和清理策略
use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};

/// 用户位置实体
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct UserLocation {
    /// 数据库自增主键
    pub id: i64,
    /// 关联的用户ID
    pub user_id: i64,
    /// 纬度
    pub latitude: f64,
    /// 经度
    pub longitude: f64,
    /// 位置名称/地址
    pub location_name: String,
    /// 更新时间
    pub updated_at: DateTime<Utc>,
}

impl UserLocation {
    /// 创建新的用户位置
    pub fn new(user_id: i64, latitude: f64, longitude: f64, location_name: String) -> Self {
        Self {
            id: 0, // 将由数据库生成
            user_id,
            latitude,
            longitude,
            location_name,
            updated_at: Utc::now(),
        }
    }
    
    /// 更新位置信息
    pub fn update_location(&mut self, latitude: f64, longitude: f64, location_name: String) {
        self.latitude = latitude;
        self.longitude = longitude;
        self.location_name = location_name;
        self.updated_at = Utc::now();
    }
    
    /// 验证位置数据的有效性
    pub fn validate(&self) -> Result<(), Vec<String>> {
        let mut errors = Vec::new();
        
        if self.latitude < -90.0 || self.latitude > 90.0 {
            errors.push("纬度必须在-90到90之间".to_string());
        }
        
        if self.longitude < -180.0 || self.longitude > 180.0 {
            errors.push("经度必须在-180到180之间".to_string());
        }
        
        if self.location_name.is_empty() {
            errors.push("位置名称不能为空".to_string());
        }
        
        if self.location_name.len() > 255 {
            errors.push("位置名称长度不能超过255个字符".to_string());
        }
        
        if errors.is_empty() {
            Ok(())
        } else {
            Err(errors)
        }
    }
    
    /// 计算与另一个位置的距离（使用Haversine公式，返回米）
    pub fn distance_to(&self, other: &UserLocation) -> f64 {
        self.distance_to_coordinates(other.latitude, other.longitude)
    }
    
    /// 计算与指定坐标的距离（使用Haversine公式，返回米）
    pub fn distance_to_coordinates(&self, lat: f64, lon: f64) -> f64 {
        const EARTH_RADIUS: f64 = 6371000.0; // 地球半径（米）
        
        let lat1_rad = self.latitude.to_radians();
        let lat2_rad = lat.to_radians();
        let delta_lat = (lat - self.latitude).to_radians();
        let delta_lon = (lon - self.longitude).to_radians();
        
        let a = (delta_lat / 2.0).sin().powi(2) +
                lat1_rad.cos() * lat2_rad.cos() * (delta_lon / 2.0).sin().powi(2);
        let c = 2.0 * a.sqrt().atan2((1.0 - a).sqrt());
        
        EARTH_RADIUS * c
    }
}

