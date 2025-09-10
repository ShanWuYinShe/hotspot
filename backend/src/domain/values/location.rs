/// 位置值对象
/// 
/// 封装地理位置相关的逻辑和验证
/// 位置值对象
/// 
/// 功能职责：
/// - 定义地理位置的值对象和坐标系统
/// - 实现位置计算和距离测量算法
/// - 提供位置验证和格式化功能
/// - 管理位置精度和误差处理
/// 
/// 设计原则：
/// - 不可变性：值对象创建后不可修改
/// - 自我验证：内部包含数据有效性检查
/// - 值语义：基于内容而非身份的相等性
/// - 无副作用：操作不会改变外部状态
use serde::{Deserialize, Serialize};
use super::coordinates::Coordinates;


/// 完整的位置信息
#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct Location {
    pub coordinates: Coordinates,
    pub address: Option<String>,
}

impl Location {
    /// 创建新的位置
    pub fn new(latitude: f64, longitude: f64, address: Option<String>) -> Result<Self, String> {
        let coordinates = Coordinates::new(latitude, longitude)
            .map_err(|e| format!("坐标错误: {}", e))?;
        
        if let Some(ref addr) = address {
            if addr.len() > 200 {
                return Err("地址长度不能超过200个字符".to_string());
            }
        }
        
        Ok(Self {
            coordinates,
            address,
        })
    }
    
    /// 从坐标创建位置
    pub fn from_coordinates(coordinates: Coordinates) -> Self {
        Self {
            coordinates,
            address: None,
        }
    }
    
    /// 获取纬度
    pub fn latitude(&self) -> f64 {
        self.coordinates.lat()
    }
    
    /// 获取经度
    pub fn longitude(&self) -> f64 {
        self.coordinates.lon()
    }
    
    /// 获取地址
    pub fn address(&self) -> Option<&str> {
        self.address.as_deref()
    }
    
    /// 更新地址
    pub fn update_address(&mut self, address: Option<String>) -> Result<(), String> {
        if let Some(ref addr) = address {
            if addr.len() > 200 {
                return Err("地址长度不能超过200个字符".to_string());
            }
        }
        self.address = address;
        Ok(())
    }
    
    /// 计算与另一个位置的距离（米）
    pub fn distance_to(&self, other: &Location) -> f64 {
        self.coordinates.distance_to(&other.coordinates).meters()
    }
    
    /// 计算与另一个位置的距离（公里）
    pub fn distance_to_km(&self, other: &Location) -> f64 {
        self.coordinates.distance_to(&other.coordinates).kilometers()
    }
    
    /// 检查是否在指定范围内
    pub fn is_within_radius(&self, other: &Location, radius_meters: f64) -> bool {
        use super::coordinates::Distance;
        let radius = Distance::from_meters(radius_meters);
        self.coordinates.is_within_radius(&other.coordinates, &radius)
    }
    
    /// 检查是否在指定范围内（公里）
    pub fn is_within_radius_km(&self, other: &Location, radius_km: f64) -> bool {
        use super::coordinates::Distance;
        let radius = Distance::from_kilometers(radius_km);
        self.coordinates.is_within_radius(&other.coordinates, &radius)
    }
}

/// 默认的搜索半径常量
pub mod radius {
    /// 附近用户搜索默认半径（米）
    pub const DEFAULT_NEARBY_RADIUS_METERS: f64 = 5000.0; // 5公里
    
    /// 附近群组搜索默认半径（米）
    pub const DEFAULT_GROUP_RADIUS_METERS: f64 = 10000.0; // 10公里
    
    /// 最大搜索半径（米）
    pub const MAX_SEARCH_RADIUS_METERS: f64 = 50000.0; // 50公里
    
    /// 附近用户搜索默认半径（公里）
    pub const DEFAULT_NEARBY_RADIUS_KM: f64 = DEFAULT_NEARBY_RADIUS_METERS / 1000.0;
    
    /// 附近群组搜索默认半径（公里）
    pub const DEFAULT_GROUP_RADIUS_KM: f64 = DEFAULT_GROUP_RADIUS_METERS / 1000.0;
    
    /// 最大搜索半径（公里）
    pub const MAX_SEARCH_RADIUS_KM: f64 = MAX_SEARCH_RADIUS_METERS / 1000.0;
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_location_creation() {
        // 有效位置
        assert!(Location::new(40.7128, -74.0060, Some("New York".to_string())).is_ok());
        
        // 无效纬度
        assert!(Location::new(91.0, -74.0060, None).is_err());
        assert!(Location::new(-91.0, -74.0060, None).is_err());
        
        // 无效经度
        assert!(Location::new(40.7128, 181.0, None).is_err());
        assert!(Location::new(40.7128, -181.0, None).is_err());
    }

    #[test]
    fn test_location_distance_calculation() {
        let nyc = Location::new(40.7128, -74.0060, Some("New York".to_string())).unwrap();
        let la = Location::new(34.0522, -118.2437, Some("Los Angeles".to_string())).unwrap();
        
        let distance_km = nyc.distance_to_km(&la);
        
        // 纽约到洛杉矶的距离大约是3944公里
        assert!((distance_km - 3944.0).abs() < 100.0);
    }

    #[test]
    fn test_location_within_radius() {
        let center = Location::new(40.7128, -74.0060, None).unwrap();
        let nearby = Location::new(40.7228, -74.0160, None).unwrap(); // 大约1.4公里
        let far = Location::new(41.7128, -74.0060, None).unwrap(); // 大约111公里
        
        assert!(center.is_within_radius_km(&nearby, 2.0));
        assert!(!center.is_within_radius_km(&far, 100.0));
    }
}

