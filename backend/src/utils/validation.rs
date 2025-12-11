use uuid::Uuid;
use crate::error::{AppError, Result};
use crate::constants::DEFAULT_PAGE_LIMIT;

pub struct Validator;

impl Validator {
    /// 验证分页参数
    /// - None -> 默认值 20
    /// - 超出 1..=max_limit 范围 -> 返回错误
    pub fn validate_limit(limit: Option<i32>, max_limit: i32) -> Result<i32> {
        let limit = limit.unwrap_or(DEFAULT_PAGE_LIMIT as i32);
        if !(1..=max_limit).contains(&limit) {
            return Err(AppError::BadRequest(format!(
                "limit 必须在 1 到 {} 之间",
                max_limit
            )));
        }
        Ok(limit)
    }

    /// 验证可选搜索半径
    /// - None -> 默认值 5000
    /// - 超出 1..=max_radius 范围 -> 返回错误
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

    /// 验证必填搜索半径 (f64)
    pub fn validate_radius_required(radius: f64, max_radius: f64) -> Result<f64> {
        if !(1.0..=max_radius).contains(&radius) {
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

    /// 验证地理坐标合法性
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

    /// 解析 UUID 游标
    /// - None -> Ok(None)
    /// - 有效 UUID -> Ok(Some(uuid))
    /// - 无效格式 -> 返回错误
    pub fn parse_cursor(cursor: Option<&str>) -> Result<Option<Uuid>> {
        cursor
            .map(|c| Uuid::parse_str(c).map_err(|_| AppError::BadRequest("无效的游标".to_string())))
            .transpose()
    }

    /// 解析 offset 游标
    /// - None -> 0
    /// - 有效数字 -> offset 值
    /// - 无效格式 -> 返回错误
    pub fn parse_offset(cursor: Option<&str>) -> Result<i64> {
        match cursor {
            Some(c) => c.parse().map_err(|_| AppError::BadRequest("无效的游标".to_string())),
            None => Ok(0),
        }
    }
}
