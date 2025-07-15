use crate::utils::response::AppError;
use crate::models::constants::group_roles;

/// 验证群组创建请求参数
pub fn validate_group_creation_request(description: Option<&String>) -> Result<(), AppError> {
    // 检查描述长度
    if let Some(desc) = description {
        if desc.len() > 1000 {
            return Err(AppError::BadRequest("群组描述不能超过1000个字符".to_string()));
        }
    }
    
    // 检查经纬度范围（交给common_validator处理）
    
    Ok(())
}

/// 验证群组名称
pub fn validate_group_name(name: &str) -> Result<(), AppError> {
    if name.len() < 2 || name.len() > 50 {
        return Err(AppError::BadRequest("群组名称长度应在2-50个字符之间".to_string()));
    }
    Ok(())
}

/// 验证群组密码
pub fn validate_group_password(password: Option<&str>) -> Result<(), AppError> {
    if let Some(pwd) = password {
        if pwd.len() < 4 || pwd.len() > 20 {
            return Err(AppError::BadRequest("密码长度应在4-20个字符之间".to_string()));
        }
    }
    Ok(())
}

/// 验证搜索半径
pub fn validate_search_radius(radius: f64) -> Result<(), AppError> {
    if radius <= 0.0 || radius > 50000.0 {
        return Err(AppError::BadRequest("搜索半径必须在0-50000米之间".to_string()));
    }
    Ok(())
}

/// 验证群组角色是否合法
pub fn validate_group_role(role: &str) -> Result<(), AppError> {
    if !group_roles::is_valid_role(role) {
        return Err(AppError::BadRequest(format!("无效的群组角色: {}", role)));
    }
    Ok(())
} 