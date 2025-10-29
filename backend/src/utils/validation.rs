use crate::error::{AppError, Result};

pub struct Validator;

impl Validator {
    /// 验证分页参数
    pub fn validate_pagination(limit: Option<i32>, max_limit: i32) -> Result<i32> {
        let limit = limit.unwrap_or(20);
        if !(1..=max_limit).contains(&limit) {
            return Err(AppError::BadRequest(format!(
                "limit 必须在 1 到 {} 之间",
                max_limit
            )));
        }
        Ok(limit)
    }

    /// 验证搜索半径
    pub fn validate_radius(radius: Option<i32>, max_radius: i32) -> Result<i32> {
        let radius = radius.unwrap_or(5000);
        if !(1..=max_radius).contains(&radius) {
            return Err(AppError::BadRequest(format!(
                "radius 必须在 1 到 {} 之间",
                max_radius
            )));
        }
        Ok(radius)
    }

    /// 验证非空字符串
    pub fn validate_not_empty(value: &str, field_name: &str) -> Result<()> {
        if value.trim().is_empty() {
            return Err(AppError::BadRequest(format!("{} 不能为空", field_name)));
        }
        Ok(())
    }

    /// 验证字符串长度
    pub fn validate_length(
        value: &str,
        field_name: &str,
        min: usize,
        max: usize,
    ) -> Result<()> {
        let len = value.len();
        if len < min || len > max {
            return Err(AppError::BadRequest(format!(
                "{} 长度必须在 {} 到 {} 之间",
                field_name, min, max
            )));
        }
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_validate_pagination() {
        assert_eq!(Validator::validate_pagination(Some(50), 100).unwrap(), 50);
        assert_eq!(Validator::validate_pagination(None, 100).unwrap(), 20);
        assert!(Validator::validate_pagination(Some(0), 100).is_err());
        assert!(Validator::validate_pagination(Some(101), 100).is_err());
    }

    #[test]
    fn test_validate_radius() {
        assert_eq!(Validator::validate_radius(Some(1000), 50000).unwrap(), 1000);
        assert_eq!(Validator::validate_radius(None, 50000).unwrap(), 5000);
    }

    #[test]
    fn test_validate_not_empty() {
        assert!(Validator::validate_not_empty("test", "field").is_ok());
        assert!(Validator::validate_not_empty("", "field").is_err());
        assert!(Validator::validate_not_empty("   ", "field").is_err());
    }
}

