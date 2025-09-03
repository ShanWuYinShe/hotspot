/// 通用验证器
/// 
/// 迁移自传统validators/common_validator.rs，适配新架构
/// 提供通用的参数验证功能

use uuid::Uuid;

/// 验证错误类型
#[derive(Debug, thiserror::Error)]
pub enum ValidationError {
    #[error("参数无效: {message}")]
    InvalidParameter { message: String },
    
    #[error("UUID格式无效: {param_name}")]
    InvalidUuid { param_name: String },
    
    #[error("位置参数无效: {message}")]
    InvalidLocation { message: String },
    
    #[error("分页参数无效: {message}")]
    InvalidPagination { message: String },
    
    #[error("字符串长度无效: {field} 长度应在 {min}-{max} 之间")]
    InvalidStringLength { field: String, min: usize, max: usize },
}

/// 验证结果类型
pub type ValidationResult<T> = Result<T, ValidationError>;

/// 验证UUID格式的参数
pub fn validate_uuid(uuid_str: &str, param_name: &str) -> ValidationResult<Uuid> {
    Uuid::parse_str(uuid_str).map_err(|_| ValidationError::InvalidUuid {
        param_name: param_name.to_string(),
    })
}

/// 验证位置参数（经纬度）
pub fn validate_location_params(latitude: f64, longitude: f64) -> ValidationResult<()> {
    if !(-90.0..=90.0).contains(&latitude) {
        return Err(ValidationError::InvalidLocation {
            message: "纬度必须在-90到90度之间".to_string(),
        });
    }

    if !(-180.0..=180.0).contains(&longitude) {
        return Err(ValidationError::InvalidLocation {
            message: "经度必须在-180到180度之间".to_string(),
        });
    }

    Ok(())
}

/// 验证分页参数
pub fn validate_pagination(cursor: i64, limit: i64, max_limit: i64) -> ValidationResult<()> {
    if cursor < 0 {
        return Err(ValidationError::InvalidPagination {
            message: "分页游标不能为负数".to_string(),
        });
    }

    if limit <= 0 || limit > max_limit {
        return Err(ValidationError::InvalidPagination {
            message: format!("每页记录数必须在1-{}之间", max_limit),
        });
    }

    Ok(())
}

/// 验证位置名称
pub fn validate_location_name(location_name: &str) -> ValidationResult<()> {
    if location_name.is_empty() || location_name.len() > 100 {
        return Err(ValidationError::InvalidParameter {
            message: "位置名称不能为空且不超过100个字符".to_string(),
        });
    }
    Ok(())
}

/// 验证搜索半径
pub fn validate_search_radius(radius: f64, max_radius: f64) -> ValidationResult<()> {
    if radius <= 0.0 || radius > max_radius {
        return Err(ValidationError::InvalidParameter {
            message: format!("搜索半径必须在0-{}米之间", max_radius),
        });
    }
    Ok(())
}

/// 验证字符串长度
pub fn validate_string_length(
    value: &str,
    field_name: &str,
    min: usize,
    max: usize,
) -> ValidationResult<()> {
    if value.len() < min || value.len() > max {
        return Err(ValidationError::InvalidStringLength {
            field: field_name.to_string(),
            min,
            max,
        });
    }
    Ok(())
}

/// 验证非空字符串
pub fn validate_non_empty_string(value: &str, field_name: &str) -> ValidationResult<()> {
    if value.trim().is_empty() {
        return Err(ValidationError::InvalidParameter {
            message: format!("{} 不能为空", field_name),
        });
    }
    Ok(())
}

/// 验证数字范围
pub fn validate_number_range<T>(value: T, field_name: &str, min: T, max: T) -> ValidationResult<()>
where
    T: PartialOrd + std::fmt::Display + Copy,
{
    if value < min || value > max {
        return Err(ValidationError::InvalidParameter {
            message: format!("{} 必须在 {} 到 {} 之间", field_name, min, max),
        });
    }
    Ok(())
}

/// 验证邮箱格式（简单验证）
pub fn validate_email_format(email: &str) -> ValidationResult<()> {
    if !email.contains('@') || !email.contains('.') || email.len() < 5 {
        return Err(ValidationError::InvalidParameter {
            message: "邮箱格式无效".to_string(),
        });
    }
    Ok(())
}
