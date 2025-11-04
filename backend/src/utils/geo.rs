use crate::error::{AppError, Result};

pub struct GeoUtil;

impl GeoUtil {
    /// 验证坐标合法性
    pub fn validate_coordinates(latitude: f64, longitude: f64) -> Result<()> {
        if !(-90.0..=90.0).contains(&latitude) {
            return Err(AppError::BadRequest(
                "纬度必须在 -90 到 90 之间".to_string(),
            ));
        }
        if !(-180.0..=180.0).contains(&longitude) {
            return Err(AppError::BadRequest(
                "经度必须在 -180 到 180 之间".to_string(),
            ));
        }
        Ok(())
    }
    
    /// 格式化距离（米 -> 人类可读）
    pub fn format_distance(meters: f64) -> String {
        if meters < 1000.0 {
            format!("{:.0}m", meters)
        } else {
            format!("{:.1}km", meters / 1000.0)
        }
    }
    
    /// 使用 Haversine 公式计算两点间的距离（米）
    pub fn haversine_distance(lat1: f64, lon1: f64, lat2: f64, lon2: f64) -> f64 {
        const EARTH_RADIUS_M: f64 = 6371000.0; // 地球半径（米）
        
        let lat1_rad = lat1.to_radians();
        let lat2_rad = lat2.to_radians();
        let delta_lat = (lat2 - lat1).to_radians();
        let delta_lon = (lon2 - lon1).to_radians();
        
        let a = (delta_lat / 2.0).sin().powi(2)
            + lat1_rad.cos() * lat2_rad.cos() * (delta_lon / 2.0).sin().powi(2);
        let c = 2.0 * a.sqrt().atan2((1.0 - a).sqrt());
        
        EARTH_RADIUS_M * c
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_valid_coordinates() {
        assert!(GeoUtil::validate_coordinates(39.9042, 116.4074).is_ok());
    }

    #[test]
    fn test_invalid_latitude() {
        assert!(GeoUtil::validate_coordinates(91.0, 116.4074).is_err());
        assert!(GeoUtil::validate_coordinates(-91.0, 116.4074).is_err());
    }

    #[test]
    fn test_invalid_longitude() {
        assert!(GeoUtil::validate_coordinates(39.9042, 181.0).is_err());
        assert!(GeoUtil::validate_coordinates(39.9042, -181.0).is_err());
    }

    #[test]
    fn test_format_distance() {
        assert_eq!(GeoUtil::format_distance(500.0), "500m");
        assert_eq!(GeoUtil::format_distance(1500.0), "1.5km");
    }
}

