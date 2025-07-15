use crate::utils::response::AppError;
use uuid::Uuid;

/// 验证UUID格式的参数
pub fn validate_uuid(uuid_str: &str, param_name: &str) -> Result<Uuid, AppError> {
    Uuid::parse_str(uuid_str).map_err(|_| AppError::BadRequest(format!("无效的{}格式", param_name)))
}

/// 验证位置参数（经纬度）
pub fn validate_location_params(latitude: f64, longitude: f64) -> Result<(), AppError> {
    if latitude < -90.0 || latitude > 90.0 {
        return Err(AppError::BadRequest("纬度必须在-90到90度之间".to_string()));
    }

    if longitude < -180.0 || longitude > 180.0 {
        return Err(AppError::BadRequest(
            "经度必须在-180到180度之间".to_string(),
        ));
    }

    Ok(())
}

/// 验证分页参数
pub fn validate_pagination(cursor: i64, limit: i64, max_limit: i64) -> Result<(), AppError> {
    if cursor < 0 {
        return Err(AppError::BadRequest("分页游标不能为负数".to_string()));
    }

    if limit <= 0 || limit > max_limit {
        return Err(AppError::BadRequest(format!(
            "每页记录数必须在1-{}之间",
            max_limit
        )));
    }

    Ok(())
}

/// 验证位置名称
pub fn validate_location_name(location_name: &str) -> Result<(), AppError> {
    if location_name.is_empty() || location_name.len() > 100 {
        return Err(AppError::BadRequest(
            "位置名称不能为空且不超过100个字符".to_string(),
        ));
    }
    Ok(())
}

/// 验证搜索半径 - 参数为搜索半径和允许的最大半径
pub fn validate_search_radius(radius: f64, max_radius: f64) -> Result<(), AppError> {
    if radius <= 0.0 || radius > max_radius {
        return Err(AppError::BadRequest(format!(
            "搜索半径必须在0-{}米之间",
            max_radius
        )));
    }
    Ok(())
}

/// 验证字符串长度
pub fn validate_string_length(
    value: &str,
    name: &str,
    min: usize,
    max: usize,
) -> Result<(), AppError> {
    if value.len() < min || value.len() > max {
        return Err(AppError::BadRequest(format!(
            "{}长度必须在{}-{}个字符之间",
            name, min, max
        )));
    }
    Ok(())
}
