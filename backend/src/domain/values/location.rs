/// 位置值对象
/// 
/// 封装地理位置相关的逻辑和验证

use serde::{Deserialize, Serialize};

/// 地理位置坐标
#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct Coordinates {
    pub latitude: f64,
    pub longitude: f64,
}

impl Coordinates {
    /// 创建新的坐标
    pub fn new(latitude: f64, longitude: f64) -> Result<Self, String> {
        let coords = Self { latitude, longitude };
        coords.validate()?;
        Ok(coords)
    }
    
    /// 验证坐标有效性
    pub fn validate(&self) -> Result<(), String> {
        if self.latitude < -90.0 || self.latitude > 90.0 {
            return Err("纬度必须在-90到90之间".to_string());
        }
        
        if self.longitude < -180.0 || self.longitude > 180.0 {
            return Err("经度必须在-180到180之间".to_string());
        }
        
        Ok(())
    }
    
    /// 计算与另一个坐标点的距离（使用Haversine公式，返回米）
    pub fn distance_to(&self, other: &Coordinates) -> f64 {
        let earth_radius = 6371000.0; // 地球半径，单位：米
        
        let lat1_rad = self.latitude.to_radians();
        let lat2_rad = other.latitude.to_radians();
        let delta_lat = (other.latitude - self.latitude).to_radians();
        let delta_lon = (other.longitude - self.longitude).to_radians();
        
        let a = (delta_lat / 2.0).sin().powi(2)
            + lat1_rad.cos() * lat2_rad.cos() * (delta_lon / 2.0).sin().powi(2);
        let c = 2.0 * a.sqrt().atan2((1.0 - a).sqrt());
        
        earth_radius * c
    }
    
    /// 计算与另一个坐标点的距离（返回公里）
    pub fn distance_to_km(&self, other: &Coordinates) -> f64 {
        self.distance_to(other) / 1000.0
    }
    
    /// 检查是否在指定半径范围内（米）
    pub fn is_within_radius(&self, other: &Coordinates, radius_meters: f64) -> bool {
        self.distance_to(other) <= radius_meters
    }
    
    /// 检查是否在指定半径范围内（公里）
    pub fn is_within_radius_km(&self, other: &Coordinates, radius_km: f64) -> bool {
        self.distance_to_km(other) <= radius_km
    }
}

/// 完整的位置信息
#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct Location {
    pub coordinates: Coordinates,
    pub address: Option<String>,
}

impl Location {
    /// 创建新的位置
    pub fn new(latitude: f64, longitude: f64, address: Option<String>) -> Result<Self, String> {
        let coordinates = Coordinates::new(latitude, longitude)?;
        
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
        self.coordinates.latitude
    }
    
    /// 获取经度
    pub fn longitude(&self) -> f64 {
        self.coordinates.longitude
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
        self.coordinates.distance_to(&other.coordinates)
    }
    
    /// 计算与另一个位置的距离（公里）
    pub fn distance_to_km(&self, other: &Location) -> f64 {
        self.coordinates.distance_to_km(&other.coordinates)
    }
    
    /// 检查是否在指定范围内
    pub fn is_within_radius(&self, other: &Location, radius_meters: f64) -> bool {
        self.coordinates.is_within_radius(&other.coordinates, radius_meters)
    }
    
    /// 检查是否在指定范围内（公里）
    pub fn is_within_radius_km(&self, other: &Location, radius_km: f64) -> bool {
        self.coordinates.is_within_radius_km(&other.coordinates, radius_km)
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
    fn test_coordinates_validation() {
        // 有效坐标
        assert!(Coordinates::new(40.7128, -74.0060).is_ok());
        
        // 无效纬度
        assert!(Coordinates::new(91.0, -74.0060).is_err());
        assert!(Coordinates::new(-91.0, -74.0060).is_err());
        
        // 无效经度
        assert!(Coordinates::new(40.7128, 181.0).is_err());
        assert!(Coordinates::new(40.7128, -181.0).is_err());
    }

    #[test]
    fn test_distance_calculation() {
        let nyc = Coordinates::new(40.7128, -74.0060).unwrap();
        let la = Coordinates::new(34.0522, -118.2437).unwrap();
        
        let distance_km = nyc.distance_to_km(&la);
        
        // 纽约到洛杉矶的距离大约是3944公里
        assert!((distance_km - 3944.0).abs() < 100.0);
    }

    #[test]
    fn test_within_radius() {
        let center = Coordinates::new(40.7128, -74.0060).unwrap();
        let nearby = Coordinates::new(40.7228, -74.0160).unwrap(); // 大约1.4公里
        let far = Coordinates::new(41.7128, -74.0060).unwrap(); // 大约111公里
        
        assert!(center.is_within_radius_km(&nearby, 2.0));
        assert!(!center.is_within_radius_km(&far, 100.0));
    }
}

