/// 瞬间验证器
/// 
/// 迁移自传统validators/moment_validator.rs，适配新架构
/// 提供瞬间相关的参数验证功能
use super::{ValidationError, ValidationResult, validate_string_length, validate_non_empty_string, validate_location_params, validate_location_name};

/// 验证常量
const MAX_MOMENT_DESCRIPTION_LENGTH: usize = 1000;
const MAX_MOMENT_TAGS: usize = 10;
const MAX_TAG_LENGTH: usize = 20;
const MAX_SEARCH_RADIUS: f64 = 50000.0; // 50公里

/// 验证瞬间内容
pub fn validate_moment_content(description: &str) -> ValidationResult<()> {
    validate_non_empty_string(description, "瞬间内容")?;
    validate_string_length(description, "瞬间内容", 1, MAX_MOMENT_DESCRIPTION_LENGTH)?;
    Ok(())
}

/// 验证瞬间标签
pub fn validate_moment_tags(tags: Option<&Vec<String>>) -> ValidationResult<()> {
    if let Some(tag_list) = tags {
        // 验证标签数量
        if tag_list.len() > MAX_MOMENT_TAGS {
            return Err(ValidationError::InvalidParameter {
                message: format!("标签数量不能超过{}个", MAX_MOMENT_TAGS),
            });
        }
        
        // 验证每个标签
        for tag in tag_list {
            validate_moment_tag(tag)?;
        }
    }
    Ok(())
}

/// 验证单个瞬间标签
pub fn validate_moment_tag(tag: &str) -> ValidationResult<()> {
    validate_non_empty_string(tag, "标签")?;
    validate_string_length(tag, "标签", 1, MAX_TAG_LENGTH)?;
    
    // 验证标签格式
    if !tag.chars().all(|c| {
        c.is_alphanumeric() || c.is_whitespace() || 
        c == '_' || c == '-' || c == '#' || !c.is_ascii() // 允许非ASCII字符
    }) {
        return Err(ValidationError::InvalidParameter {
            message: "标签只能包含字母、数字、空格和_-#符号".to_string(),
        });
    }
    
    Ok(())
}

/// 验证瞬间创建请求
pub fn validate_moment_creation_request(
    description: &str,
    latitude: f64,
    longitude: f64,
    location_name: &str,
    tags: Option<&Vec<String>>,
) -> ValidationResult<()> {
    // 验证瞬间内容
    validate_moment_content(description)?;
    
    // 验证位置参数
    validate_location_params(latitude, longitude)?;
    
    // 验证位置名称
    validate_location_name(location_name)?;
    
    // 验证标签
    validate_moment_tags(tags)?;
    
    Ok(())
}

/// 验证瞬间ID
pub fn validate_moment_id(moment_id: &str) -> ValidationResult<uuid::Uuid> {
    super::validate_uuid(moment_id, "瞬间ID")
}

/// 验证瞬间搜索半径
pub fn validate_moment_search_radius(radius: f64) -> ValidationResult<()> {
    if radius <= 0.0 || radius > MAX_SEARCH_RADIUS {
        return Err(ValidationError::InvalidParameter {
            message: format!("搜索半径必须在0-{}米之间", MAX_SEARCH_RADIUS),
        });
    }
    Ok(())
}

/// 验证根据位置搜索瞬间请求
pub fn validate_search_moments_by_location_request(
    latitude: f64,
    longitude: f64,
    radius: f64,
    cursor: Option<i64>,
    limit: Option<i64>,
) -> ValidationResult<()> {
    // 验证位置参数
    validate_location_params(latitude, longitude)?;
    
    // 验证搜索半径
    validate_moment_search_radius(radius)?;
    
    // 验证分页参数
    let cursor = cursor.unwrap_or(0);
    let limit = limit.unwrap_or(20);
    super::validate_pagination(cursor, limit, 100)?;
    
    Ok(())
}

/// 验证根据标签搜索瞬间请求
pub fn validate_search_moments_by_tags_request(
    tags: &Vec<String>,
    latitude: f64,
    longitude: f64,
    cursor: Option<i64>,
    limit: Option<i64>,
) -> ValidationResult<()> {
    // 验证标签
    if tags.is_empty() {
        return Err(ValidationError::InvalidParameter {
            message: "搜索标签不能为空".to_string(),
        });
    }
    
    validate_moment_tags(Some(tags))?;
    
    // 验证位置参数
    validate_location_params(latitude, longitude)?;
    
    // 验证分页参数
    let cursor = cursor.unwrap_or(0);
    let limit = limit.unwrap_or(20);
    super::validate_pagination(cursor, limit, 100)?;
    
    Ok(())
}

/// 验证用户瞬间历史请求
pub fn validate_user_moments_history_request(
    user_id: Option<i64>,
    cursor: Option<i64>,
    limit: Option<i64>,
) -> ValidationResult<()> {
    // 验证用户ID（如果提供）
    if let Some(id) = user_id {
        if id <= 0 {
            return Err(ValidationError::InvalidParameter {
                message: "用户ID必须为正数".to_string(),
            });
        }
    }
    
    // 验证分页参数
    let cursor = cursor.unwrap_or(0);
    let limit = limit.unwrap_or(20);
    super::validate_pagination(cursor, limit, 100)?;
    
    Ok(())
}

/// 验证瞬间评论内容
pub fn validate_moment_comment_content(content: &str) -> ValidationResult<()> {
    validate_non_empty_string(content, "评论内容")?;
    validate_string_length(content, "评论内容", 1, 500)?; // 评论最大500字符
    Ok(())
}

/// 验证添加瞬间评论请求
pub fn validate_add_moment_comment_request(
    moment_id: &str,
    content: &str,
) -> ValidationResult<()> {
    // 验证瞬间ID
    validate_moment_id(moment_id)?;
    
    // 验证评论内容
    validate_moment_comment_content(content)?;
    
    Ok(())
}
