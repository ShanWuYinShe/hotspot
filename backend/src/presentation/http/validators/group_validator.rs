/// 群组验证器
/// 
/// 迁移自传统validators/group_validator.rs，适配新架构
/// 提供群组相关的参数验证功能

use super::{ValidationError, ValidationResult, validate_string_length, validate_non_empty_string, validate_location_params};

/// 验证常量
const MAX_GROUP_NAME_LENGTH: usize = 100;
const MAX_GROUP_DESCRIPTION_LENGTH: usize = 500;
const MAX_GROUP_TAGS: usize = 10;
const MAX_TAG_LENGTH: usize = 20;

/// 验证群组名称
pub fn validate_group_name(name: &str) -> ValidationResult<()> {
    // 验证非空
    validate_non_empty_string(name, "群组名称")?;
    
    // 验证长度
    validate_string_length(name, "群组名称", 1, MAX_GROUP_NAME_LENGTH)?;
    
    // 验证字符格式（允许中文、字母、数字、空格、常见符号）
    if !name.chars().all(|c| {
        c.is_alphanumeric() || c.is_whitespace() || 
        c == '_' || c == '-' || c == '.' || c == '(' || c == ')' ||
        c == '[' || c == ']' || c == '#' || !c.is_ascii() // 允许非ASCII字符（如中文）
    }) {
        return Err(ValidationError::InvalidParameter {
            message: "群组名称包含不允许的字符".to_string(),
        });
    }
    
    Ok(())
}

/// 验证群组描述
pub fn validate_group_description(description: Option<&str>) -> ValidationResult<()> {
    if let Some(desc) = description {
        if !desc.trim().is_empty() {
            validate_string_length(desc, "群组描述", 1, MAX_GROUP_DESCRIPTION_LENGTH)?;
        }
    }
    Ok(())
}

/// 验证群组标签
pub fn validate_group_tags(tags: Option<&Vec<String>>) -> ValidationResult<()> {
    if let Some(tag_list) = tags {
        // 验证标签数量
        if tag_list.len() > MAX_GROUP_TAGS {
            return Err(ValidationError::InvalidParameter {
                message: format!("标签数量不能超过{}个", MAX_GROUP_TAGS),
            });
        }
        
        // 验证每个标签
        for tag in tag_list {
            validate_group_tag(tag)?;
        }
    }
    Ok(())
}

/// 验证单个群组标签
pub fn validate_group_tag(tag: &str) -> ValidationResult<()> {
    // 验证非空
    validate_non_empty_string(tag, "标签")?;
    
    // 验证长度
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

/// 验证群组创建请求
pub fn validate_group_creation_request(
    name: &str,
    description: Option<&str>,
    latitude: f64,
    longitude: f64,
    location_name: &str,
    tags: Option<&Vec<String>>,
) -> ValidationResult<()> {
    // 验证群组名称
    validate_group_name(name)?;
    
    // 验证群组描述
    validate_group_description(description)?;
    
    // 验证位置参数
    validate_location_params(latitude, longitude)?;
    
    // 验证位置名称
    super::validate_location_name(location_name)?;
    
    // 验证标签
    validate_group_tags(tags)?;
    
    Ok(())
}

/// 验证群组更新请求
pub fn validate_group_update_request(
    name: Option<&str>,
    description: Option<&str>,
    latitude: Option<f64>,
    longitude: Option<f64>,
    location_name: Option<&str>,
    tags: Option<&Vec<String>>,
) -> ValidationResult<()> {
    // 验证群组名称（如果提供）
    if let Some(group_name) = name {
        validate_group_name(group_name)?;
    }
    
    // 验证群组描述（如果提供）
    if description.is_some() {
        validate_group_description(description)?;
    }
    
    // 验证位置参数（如果提供）
    if let (Some(lat), Some(lng)) = (latitude, longitude) {
        validate_location_params(lat, lng)?;
    }
    
    // 验证位置名称（如果提供）
    if let Some(loc_name) = location_name {
        super::validate_location_name(loc_name)?;
    }
    
    // 验证标签（如果提供）
    if tags.is_some() {
        validate_group_tags(tags)?;
    }
    
    Ok(())
}

/// 验证群组ID
pub fn validate_group_id(group_id: &str) -> ValidationResult<uuid::Uuid> {
    super::validate_uuid(group_id, "群组ID")
}

/// 验证群组角色
pub fn validate_group_role(role: &str) -> ValidationResult<()> {
    match role {
        "owner" | "admin" | "member" => Ok(()),
        _ => Err(ValidationError::InvalidParameter {
            message: "群组角色必须是 owner、admin 或 member".to_string(),
        }),
    }
}

/// 验证群组密码（如果群组需要密码）
pub fn validate_group_password(password: Option<&str>) -> ValidationResult<()> {
    if let Some(pwd) = password {
        validate_string_length(pwd, "群组密码", 1, 50)?;
    }
    Ok(())
}
