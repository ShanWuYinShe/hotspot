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

