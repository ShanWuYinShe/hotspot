use crate::utils::response::AppError;

/// 常用验证函数
pub mod validators {
    use super::*;
    use crate::validators::common_validator;
    use uuid::Uuid;
    
    /// 验证可选的分页参数
    pub fn validate_pagination_if_provided(cursor: Option<i64>, limit: Option<i64>, max_limit: i64) -> Result<(), AppError> {
        if let (Some(c), Some(l)) = (cursor, limit) {
            common_validator::validate_pagination(c, l, max_limit)
        } else {
            Ok(())
        }
    }
    
    /// 验证并提取分页参数，提供默认值
    pub fn extract_pagination(cursor: Option<i64>, limit: Option<i64>, default_limit: i64, max_limit: i64) -> Result<(i64, i64), AppError> {
        let cursor = cursor.unwrap_or(0);
        let limit = limit.unwrap_or(default_limit);
        
        common_validator::validate_pagination(cursor, limit, max_limit)?;
        Ok((cursor, limit))
    }
    
    /// 验证UUID并返回解析后的UUID
    pub fn validate_uuid(uuid_str: &str, param_name: &str) -> Result<Uuid, AppError> {
        common_validator::validate_uuid(uuid_str, param_name)
    }
    
    /// 验证位置参数的便捷函数
    pub fn validate_location(latitude: f64, longitude: f64) -> Result<(), AppError> {
        common_validator::validate_location_params(latitude, longitude)
    }
    
    /// 验证搜索关键词
    pub fn validate_search_keyword(keyword: &str, max_length: usize) -> Result<(), AppError> {
        common_validator::validate_string_length(keyword, "搜索关键词", 1, max_length)
    }
    
    /// 验证位置搜索请求的组合验证函数
    pub fn validate_location_search(
        latitude: f64, 
        longitude: f64, 
        radius: f64, 
        max_radius: f64,
        cursor: Option<i64>, 
        limit: Option<i64>, 
        default_limit: i64, 
        max_limit: i64
    ) -> Result<(i64, i64), AppError> {
        // 验证位置参数
        validate_location(latitude, longitude)?;
        
        // 验证搜索半径
        common_validator::validate_search_radius(radius, max_radius)?;
        
        // 验证并提取分页参数
        extract_pagination(cursor, limit, default_limit, max_limit)
    }
    
    /// 验证关键词搜索请求的组合验证函数
    pub fn validate_keyword_search(
        keyword: &str, 
        max_keyword_length: usize,
        cursor: Option<i64>, 
        limit: Option<i64>, 
        default_limit: i64, 
        max_limit: i64
    ) -> Result<(i64, i64), AppError> {
        // 验证搜索关键词
        validate_search_keyword(keyword, max_keyword_length)?;
        
        // 验证并提取分页参数
        extract_pagination(cursor, limit, default_limit, max_limit)
    }
}