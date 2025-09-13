/// 地理坐标值对象
/// 
/// 提供类型安全的地理坐标，包含验证和计算功能
use serde::{Deserialize, Serialize};
use std::fmt::{self, Display};
use sqlx::Type;

use crate::domain::DomainError;

/// 纬度值对象
#[derive(Debug, Clone, Copy, PartialEq, Serialize, Deserialize, Type)]
#[sqlx(transparent)]
pub struct Latitude(f64);

impl Latitude {
    /// 创建纬度
    pub fn new(value: f64) -> Result<Self, DomainError> {
        if value < -90.0 || value > 90.0 {
            return Err(DomainError::BusinessRuleViolation {
                rule: "纬度必须在-90到90之间".to_string(),
            });
        }
        Ok(Latitude(value))
    }

    /// 获取原始值
    pub fn value(&self) -> f64 {
        self.0
    }

    /// 转换为弧度
    pub fn to_radians(&self) -> f64 {
        self.0.to_radians()
    }
}

impl Display for Latitude {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "{:.6}°N", self.0.abs())
    }
}

/// 经度值对象
#[derive(Debug, Clone, Copy, PartialEq, Serialize, Deserialize, Type)]
#[sqlx(transparent)]
pub struct Longitude(f64);

impl Longitude {
    /// 创建经度
    pub fn new(value: f64) -> Result<Self, DomainError> {
        if value < -180.0 || value > 180.0 {
            return Err(DomainError::BusinessRuleViolation {
                rule: "经度必须在-180到180之间".to_string(),
            });
        }
        Ok(Longitude(value))
    }

    /// 获取原始值
    pub fn value(&self) -> f64 {
        self.0
    }

    /// 转换为弧度
    pub fn to_radians(&self) -> f64 {
        self.0.to_radians()
    }
}

impl Display for Longitude {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        let direction = if self.0 >= 0.0 { "E" } else { "W" };
        write!(f, "{:.6}°{}", self.0.abs(), direction)
    }
}

/// 地理坐标值对象
#[derive(Debug, Clone, Copy, PartialEq, Serialize, Deserialize)]
pub struct Coordinates {
    pub latitude: Latitude,
    pub longitude: Longitude,
}

impl Coordinates {
    /// 创建坐标
    pub fn new(latitude: f64, longitude: f64) -> Result<Self, DomainError> {
        Ok(Coordinates {
            latitude: Latitude::new(latitude)?,
            longitude: Longitude::new(longitude)?,
        })
    }

    /// 从纬度和经度对象创建
    pub fn from_lat_lon(latitude: Latitude, longitude: Longitude) -> Self {
        Coordinates { latitude, longitude }
    }

    /// 获取纬度值
    pub fn lat(&self) -> f64 {
        self.latitude.value()
    }

    /// 获取经度值
    pub fn lon(&self) -> f64 {
        self.longitude.value()
    }

    /// 计算到另一个坐标的距离（米）
    pub fn distance_to(&self, other: &Coordinates) -> Distance {
        const EARTH_RADIUS: f64 = 6371000.0; // 地球半径（米）
        
        let lat1_rad = self.latitude.to_radians();
        let lat2_rad = other.latitude.to_radians();
        let delta_lat = (other.lat() - self.lat()).to_radians();
        let delta_lon = (other.lon() - self.lon()).to_radians();
        
        let a = (delta_lat / 2.0).sin().powi(2)
            + lat1_rad.cos() * lat2_rad.cos() * (delta_lon / 2.0).sin().powi(2);
        let c = 2.0 * a.sqrt().atan2((1.0 - a).sqrt());
        
        Distance::from_meters(EARTH_RADIUS * c)
    }

    /// 检查是否在指定半径内
    pub fn is_within_radius(&self, center: &Coordinates, radius: &Distance) -> bool {
        self.distance_to(center).meters() <= radius.meters()
    }

    /// 获取坐标的边界框（用于数据库查询优化）
    pub fn bounding_box(&self, radius: &Distance) -> BoundingBox {
        // 简化计算，实际应用中可能需要更精确的算法
        let lat_delta = radius.meters() / 111320.0; // 大约每度纬度的米数
        let lon_delta = radius.meters() / (111320.0 * self.lat().to_radians().cos());
        
        let min_lat = (self.lat() - lat_delta).max(-90.0);
        let max_lat = (self.lat() + lat_delta).min(90.0);
        let min_lon = (self.lon() - lon_delta).max(-180.0);
        let max_lon = (self.lon() + lon_delta).min(180.0);
        
        BoundingBox {
            min_lat: Latitude::new(min_lat).unwrap(),
            max_lat: Latitude::new(max_lat).unwrap(),
            min_lon: Longitude::new(min_lon).unwrap(),
            max_lon: Longitude::new(max_lon).unwrap(),
        }
    }
}

impl Display for Coordinates {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "{}, {}", self.latitude, self.longitude)
    }
}

/// 距离值对象
#[derive(Debug, Clone, Copy, PartialEq, PartialOrd, Serialize, Deserialize)]
pub struct Distance(f64); // 以米为单位

impl Distance {
    /// 从米创建距离
    pub fn from_meters(meters: f64) -> Self {
        Distance(meters.max(0.0))
    }

    /// 从千米创建距离
    pub fn from_kilometers(kilometers: f64) -> Self {
        Distance((kilometers * 1000.0).max(0.0))
    }

    /// 获取米数
    pub fn meters(&self) -> f64 {
        self.0
    }

    /// 获取千米数
    pub fn kilometers(&self) -> f64 {
        self.0 / 1000.0
    }

    /// 验证距离是否在合理范围内
    pub fn validate(&self, max_distance: f64) -> Result<(), DomainError> {
        if self.0 <= 0.0 {
            return Err(DomainError::BusinessRuleViolation {
                rule: "距离必须大于0".to_string(),
            });
        }
        
        if self.0 > max_distance {
            return Err(DomainError::BusinessRuleViolation {
                rule: format!("距离不能超过{}米", max_distance),
            });
        }
        
        Ok(())
    }
}

impl Display for Distance {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        if self.0 >= 1000.0 {
            write!(f, "{:.2} km", self.kilometers())
        } else {
            write!(f, "{:.0} m", self.0)
        }
    }
}

/// 边界框
#[derive(Debug, Clone, Copy, PartialEq, Serialize, Deserialize)]
pub struct BoundingBox {
    pub min_lat: Latitude,
    pub max_lat: Latitude,
    pub min_lon: Longitude,
    pub max_lon: Longitude,
}

impl BoundingBox {
    /// 检查坐标是否在边界框内
    pub fn contains(&self, coordinates: &Coordinates) -> bool {
        coordinates.lat() >= self.min_lat.value()
            && coordinates.lat() <= self.max_lat.value()
            && coordinates.lon() >= self.min_lon.value()
            && coordinates.lon() <= self.max_lon.value()
    }

    /// 获取边界框的中心点
    pub fn center(&self) -> Coordinates {
        let center_lat = (self.min_lat.value() + self.max_lat.value()) / 2.0;
        let center_lon = (self.min_lon.value() + self.max_lon.value()) / 2.0;
        
        Coordinates::new(center_lat, center_lon).unwrap()
    }
}

/// 地理区域值对象
#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct GeoRegion {
    pub center: Coordinates,
    pub radius: Distance,
}

impl GeoRegion {
    /// 创建地理区域
    pub fn new(center: Coordinates, radius: Distance) -> Result<Self, DomainError> {
        radius.validate(100000.0)?; // 最大100公里
        Ok(GeoRegion { center, radius })
    }

    /// 检查坐标是否在区域内
    pub fn contains(&self, coordinates: &Coordinates) -> bool {
        self.center.is_within_radius(coordinates, &self.radius)
    }

    /// 获取边界框
    pub fn bounding_box(&self) -> BoundingBox {
        self.center.bounding_box(&self.radius)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_coordinates_creation() {
        // 有效坐标
        let coords = Coordinates::new(39.9042, 116.4074).unwrap();
        assert_eq!(coords.lat(), 39.9042);
        assert_eq!(coords.lon(), 116.4074);

        // 无效坐标
        assert!(Coordinates::new(91.0, 116.4074).is_err());
        assert!(Coordinates::new(39.9042, 181.0).is_err());
    }

    #[test]
    fn test_distance_calculation() {
        // 北京天安门
        let tiananmen = Coordinates::new(39.9042, 116.4074).unwrap();
        // 北京故宫
        let forbidden_city = Coordinates::new(39.9163, 116.3972).unwrap();
        
        let distance = tiananmen.distance_to(&forbidden_city);
        
        // 距离应该在900-1100米之间
        assert!(distance.meters() > 900.0 && distance.meters() < 1100.0);
    }

    #[test]
    fn test_distance_formatting() {
        let short_distance = Distance::from_meters(500.0);
        let long_distance = Distance::from_kilometers(2.5);
        
        assert_eq!(format!("{}", short_distance), "500 m");
        assert_eq!(format!("{}", long_distance), "2.50 km");
    }

    #[test]
    fn test_geo_region() {
        let center = Coordinates::new(39.9042, 116.4074).unwrap();
        let radius = Distance::from_kilometers(1.0);
        let region = GeoRegion::new(center, radius).unwrap();
        
        // 测试点在区域内
        let nearby_point = Coordinates::new(39.9050, 116.4080).unwrap();
        assert!(region.contains(&nearby_point));
        
        // 测试点在区域外
        let far_point = Coordinates::new(40.0000, 117.0000).unwrap();
        assert!(!region.contains(&far_point));
    }
}
