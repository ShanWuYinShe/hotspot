/// 转换辅助工具
/// 
/// 提供便捷的转换函数，简化控制器和服务层中的类型转换
use crate::utils::js_safe_types::JsSafeI64;
use crate::utils::error::AppError;

/// 分页参数转换辅助工具
pub mod pagination {
    use super::*;
    use crate::presentation::http::models::Pagination;
    
    /// 从前端分页参数中提取数据库查询参数
    pub fn extract_db_params(pagination: &Pagination) -> Result<(i64, i64), AppError> {
        let cursor = pagination.cursor.map(|c| c.value()).unwrap_or(0);
        let limit = pagination.limit.map(|l| l.value()).unwrap_or(20);
        
        // 验证参数范围
        if limit <= 0 || limit > 100 {
            return Err(AppError::validation_error("limit", "must be between 1 and 100"));
        }
        
        Ok((cursor, limit))
    }
    
    /// 创建JavaScript安全的分页元信息
    pub fn create_js_safe_meta(has_more: bool, next_cursor: Option<i64>) -> crate::presentation::http::models::PaginationMeta {
        crate::presentation::http::models::PaginationMeta {
            has_more,
            next_cursor: next_cursor.map(JsSafeI64::new),
        }
    }
}

/// 用户ID转换辅助工具
pub mod user_id {
    use super::*;
    
    /// 从JavaScript安全类型转换为数据库类型
    pub fn to_db(js_safe_id: JsSafeI64) -> i64 {
        js_safe_id.value()
    }
    
    /// 从数据库类型转换为JavaScript安全类型
    pub fn from_db(db_id: i64) -> JsSafeI64 {
        JsSafeI64::new(db_id)
    }
    
    /// 从可选的JavaScript安全类型转换为数据库类型
    pub fn optional_to_db(js_safe_id: Option<JsSafeI64>) -> Option<i64> {
        js_safe_id.map(|id| id.value())
    }
    
    /// 从可选的数据库类型转换为JavaScript安全类型
    pub fn optional_from_db(db_id: Option<i64>) -> Option<JsSafeI64> {
        db_id.map(JsSafeI64::new)
    }
    
    /// 批量转换用户ID列表
    pub fn vec_to_db(js_safe_ids: Vec<JsSafeI64>) -> Vec<i64> {
        js_safe_ids.into_iter().map(|id| id.value()).collect()
    }
    
    /// 批量转换用户ID列表
    pub fn vec_from_db(db_ids: Vec<i64>) -> Vec<JsSafeI64> {
        db_ids.into_iter().map(JsSafeI64::new).collect()
    }
}

/// 通用ID转换辅助工具（适用于所有64位整数ID）
pub mod id_conversion {
    use super::*;
    
    /// 通用转换：JavaScript安全类型 -> 数据库类型
    pub fn js_to_db(js_safe_id: JsSafeI64) -> i64 {
        js_safe_id.value()
    }
    
    /// 通用转换：数据库类型 -> JavaScript安全类型
    pub fn db_to_js(db_id: i64) -> JsSafeI64 {
        JsSafeI64::new(db_id)
    }
    
    /// 通用转换：可选JavaScript安全类型 -> 可选数据库类型
    pub fn optional_js_to_db(js_safe_id: Option<JsSafeI64>) -> Option<i64> {
        js_safe_id.map(|id| id.value())
    }
    
    /// 通用转换：可选数据库类型 -> 可选JavaScript安全类型
    pub fn optional_db_to_js(db_id: Option<i64>) -> Option<JsSafeI64> {
        db_id.map(JsSafeI64::new)
    }
}

/// 请求参数验证和转换
pub mod request_validation {
    use super::*;
    
    /// 验证并转换用户ID
    pub fn validate_user_id(js_safe_id: JsSafeI64) -> Result<i64, AppError> {
        let id = js_safe_id.value();
        if id <= 0 {
            return Err(AppError::validation_error("user_id", "must be positive"));
        }
        Ok(id)
    }
    
    /// 验证并转换群组ID
    pub fn validate_group_id(js_safe_id: JsSafeI64) -> Result<i64, AppError> {
        let id = js_safe_id.value();
        if id <= 0 {
            return Err(AppError::validation_error("group_id", "must be positive"));
        }
        Ok(id)
    }
    
    /// 验证并转换消息ID
    pub fn validate_message_id(js_safe_id: JsSafeI64) -> Result<i64, AppError> {
        let id = js_safe_id.value();
        if id <= 0 {
            return Err(AppError::validation_error("message_id", "must be positive"));
        }
        Ok(id)
    }
    
    /// 验证并转换分页限制
    pub fn validate_limit(js_safe_limit: Option<JsSafeI64>) -> Result<i64, AppError> {
        let limit = js_safe_limit.map(|l| l.value()).unwrap_or(20);
        if limit <= 0 || limit > 100 {
            return Err(AppError::validation_error("limit", "must be between 1 and 100"));
        }
        Ok(limit)
    }
    
    /// 验证并转换游标
    pub fn validate_cursor(js_safe_cursor: Option<JsSafeI64>) -> Result<i64, AppError> {
        let cursor = js_safe_cursor.map(|c| c.value()).unwrap_or(0);
        if cursor < 0 {
            return Err(AppError::validation_error("cursor", "must be non-negative"));
        }
        Ok(cursor)
    }
}

/// 响应数据转换
pub mod response_conversion {
    use super::*;
    use crate::presentation::http::models::{PaginatedResponse, PaginationMeta};
    
    /// 创建JavaScript安全的分页响应
    pub fn create_paginated_response<T>(
        items: Vec<T>,
        has_more: bool,
        next_cursor: Option<i64>,
    ) -> PaginatedResponse<T> {
        PaginatedResponse {
            items,
            pagination: PaginationMeta {
                has_more,
                next_cursor: next_cursor.map(JsSafeI64::new),
            },
        }
    }
}

/// 宏定义，简化转换操作
#[macro_export]
macro_rules! js_to_db {
    ($js_safe:expr) => {
        $js_safe.value()
    };
}

#[macro_export]
macro_rules! db_to_js {
    ($db_value:expr) => {
        $crate::utils::JsSafeI64::new($db_value)
    };
}

#[macro_export]
macro_rules! optional_js_to_db {
    ($js_safe_opt:expr) => {
        $js_safe_opt.map(|x| x.value())
    };
}

#[macro_export]
macro_rules! optional_db_to_js {
    ($db_opt:expr) => {
        $db_opt.map($crate::utils::JsSafeI64::new)
    };
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::presentation::http::models::Pagination;
    
    #[test]
    fn test_pagination_extraction() {
        let pagination = Pagination {
            cursor: Some(JsSafeI64::new(100)),
            limit: Some(JsSafeI64::new(20)),
        };
        
        let (cursor, limit) = pagination::extract_db_params(&pagination).unwrap();
        assert_eq!(cursor, 100);
        assert_eq!(limit, 20);
    }
    
    #[test]
    fn test_pagination_defaults() {
        let pagination = Pagination {
            cursor: None,
            limit: None,
        };
        
        let (cursor, limit) = pagination::extract_db_params(&pagination).unwrap();
        assert_eq!(cursor, 0);
        assert_eq!(limit, 20);
    }
    
    #[test]
    fn test_user_id_conversion() {
        let db_id = 12345i64;
        let js_safe_id = user_id::from_db(db_id);
        let converted_back = user_id::to_db(js_safe_id);
        assert_eq!(db_id, converted_back);
    }
    
    #[test]
    fn test_validation_positive_id() {
        let valid_id = JsSafeI64::new(123);
        assert!(request_validation::validate_user_id(valid_id).is_ok());
        
        let invalid_id = JsSafeI64::new(-1);
        assert!(request_validation::validate_user_id(invalid_id).is_err());
        
        let zero_id = JsSafeI64::new(0);
        assert!(request_validation::validate_user_id(zero_id).is_err());
    }
    
    #[test]
    fn test_limit_validation() {
        // 有效的限制
        assert!(request_validation::validate_limit(Some(JsSafeI64::new(20))).is_ok());
        assert!(request_validation::validate_limit(Some(JsSafeI64::new(1))).is_ok());
        assert!(request_validation::validate_limit(Some(JsSafeI64::new(100))).is_ok());
        
        // 无效的限制
        assert!(request_validation::validate_limit(Some(JsSafeI64::new(0))).is_err());
        assert!(request_validation::validate_limit(Some(JsSafeI64::new(-1))).is_err());
        assert!(request_validation::validate_limit(Some(JsSafeI64::new(101))).is_err());
        
        // 默认值
        assert_eq!(request_validation::validate_limit(None).unwrap(), 20);
    }
    
    #[test]
    fn test_macros() {
        let db_value = 12345i64;
        let js_safe = db_to_js!(db_value);
        let converted_back = js_to_db!(js_safe);
        assert_eq!(db_value, converted_back);
        
        let opt_db = Some(67890i64);
        let opt_js = optional_db_to_js!(opt_db);
        let opt_converted_back = optional_js_to_db!(opt_js);
        assert_eq!(opt_db, opt_converted_back);
    }
}
