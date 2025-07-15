use crate::utils::response::AppError;

/// 验证签到内容
pub fn validate_checkin_content(description: &str) -> Result<(), AppError> {
    if description.is_empty() || description.len() > 1000 {
        return Err(AppError::BadRequest("签到内容长度应在1-1000个字符之间".to_string()));
    }
    Ok(())
}

/// 验证签到标签
pub fn validate_checkin_tags(tags: &[String]) -> Result<(), AppError> {
    // 验证标签
    if tags.len() > 10 {
        return Err(AppError::BadRequest("标签数量不能超过10个".to_string()));
    }
    
    for tag in tags {
        if tag.is_empty() || tag.len() > 20 {
            return Err(AppError::BadRequest("单个标签长度应在1-20个字符之间".to_string()));
        }
        
        // 验证标签格式
        if !tag.chars().all(|c| c.is_alphanumeric() || c.is_whitespace() || c == '_' || c == '-' || c == '#') {
            return Err(AppError::BadRequest("标签只能包含字母、数字、空格和_-#符号".to_string()));
        }
    }
    
    Ok(())
}

/// 验证签到创建请求的所有参数
pub fn validate_checkin_creation_request(description: &str, latitude: f64, longitude: f64, location_name: &str, tags: &[String]) -> Result<(), AppError> {
    // 组合各个验证函数
    validate_checkin_content(description)?;
    
    // 位置验证由common_validator处理
    super::common_validator::validate_location_params(latitude, longitude)?;
    
    // 位置名称验证由common_validator处理
    super::common_validator::validate_location_name(location_name)?;
    
    validate_checkin_tags(tags)?;
    
    Ok(())
}

/// 验证搜索半径
/// 注意：通常应优先使用 common_validator::validate_search_radius
pub fn validate_search_radius(radius: f64) -> Result<(), AppError> {
    // 使用统一的最大半径50000米
    if radius <= 0.0 || radius > 50000.0 {
        return Err(AppError::BadRequest("搜索半径必须在0-50000米之间".to_string()));
    }
    Ok(())
} 