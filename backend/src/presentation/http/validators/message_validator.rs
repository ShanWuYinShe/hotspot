/// 消息验证器
/// 
/// 迁移自传统validators/message_validator.rs，适配新架构
/// 提供消息相关的参数验证功能
use super::{ValidationError, ValidationResult, validate_string_length, validate_non_empty_string};

/// 验证常量
const MAX_TEXT_MESSAGE_LENGTH: usize = 2000;
const MAX_IMAGE_URL_LENGTH: usize = 500;
const MAX_LOCATION_MESSAGE_LENGTH: usize = 500;

/// 验证消息内容
pub fn validate_message_content(content: &str, message_type: &str) -> ValidationResult<()> {
    // 验证消息类型
    match message_type {
        "text" => {
            // 文本消息长度验证
            validate_non_empty_string(content, "消息内容")?;
            validate_string_length(content, "文本消息", 1, MAX_TEXT_MESSAGE_LENGTH)?;
        }
        "image" => {
            // 图片消息URL验证
            validate_non_empty_string(content, "图片URL")?;
            validate_string_length(content, "图片URL", 1, MAX_IMAGE_URL_LENGTH)?;
            
            if !content.starts_with("http://") && !content.starts_with("https://") {
                return Err(ValidationError::InvalidParameter {
                    message: "图片URL必须以http://或https://开头".to_string(),
                });
            }
        }
        "location" => {
            // 位置消息格式验证
            validate_non_empty_string(content, "位置信息")?;
            validate_string_length(content, "位置消息", 1, MAX_LOCATION_MESSAGE_LENGTH)?;
        }
        "file" => {
            // 文件消息URL验证
            validate_non_empty_string(content, "文件URL")?;
            validate_string_length(content, "文件URL", 1, MAX_IMAGE_URL_LENGTH)?;
            
            if !content.starts_with("http://") && !content.starts_with("https://") {
                return Err(ValidationError::InvalidParameter {
                    message: "文件URL必须以http://或https://开头".to_string(),
                });
            }
        }
        _ => {
            return Err(ValidationError::InvalidParameter {
                message: format!("不支持的消息类型: {}", message_type),
            });
        }
    }

    Ok(())
}

/// 验证消息类型
pub fn validate_message_type(message_type: &str) -> ValidationResult<()> {
    match message_type {
        "text" | "image" | "location" | "file" => Ok(()),
        _ => Err(ValidationError::InvalidParameter {
            message: format!("不支持的消息类型: {}", message_type),
        }),
    }
}

/// 验证会话ID
pub fn validate_conversation_id(conversation_id: Option<&String>) -> ValidationResult<()> {
    match conversation_id {
        Some(id) if !id.trim().is_empty() => Ok(()),
        _ => Err(ValidationError::InvalidParameter {
            message: "会话ID不能为空".to_string(),
        }),
    }
}

/// 验证接收者ID
pub fn validate_recipient_id(recipient_id: &str) -> ValidationResult<i64> {
    recipient_id
        .parse::<i64>()
        .map_err(|_| ValidationError::InvalidParameter {
            message: "无效的用户ID格式".to_string(),
        })
        .and_then(|id| {
            if id <= 0 {
                Err(ValidationError::InvalidParameter {
                    message: "用户ID必须为正数".to_string(),
                })
            } else {
                Ok(id)
            }
        })
}

/// 验证消息ID
pub fn validate_message_id(message_id: &str) -> ValidationResult<uuid::Uuid> {
    super::validate_uuid(message_id, "消息ID")
}

/// 验证群组消息发送请求
pub fn validate_send_group_message_request(
    group_id: &str,
    content: &str,
    message_type: &str,
    latitude: f64,
    longitude: f64,
) -> ValidationResult<()> {
    // 验证群组ID
    super::validate_uuid(group_id, "群组ID")?;
    
    // 验证消息内容和类型
    validate_message_content(content, message_type)?;
    
    // 验证位置参数
    super::validate_location_params(latitude, longitude)?;
    
    Ok(())
}

/// 验证私聊消息发送请求
pub fn validate_send_direct_message_request(
    recipient_id: &str,
    content: &str,
    message_type: &str,
    latitude: f64,
    longitude: f64,
) -> ValidationResult<()> {
    // 验证接收者ID
    validate_recipient_id(recipient_id)?;
    
    // 验证消息内容和类型
    validate_message_content(content, message_type)?;
    
    // 验证位置参数
    super::validate_location_params(latitude, longitude)?;
    
    Ok(())
}

/// 验证消息历史查询请求
pub fn validate_message_history_request(
    cursor: Option<i64>,
    limit: Option<i64>,
) -> ValidationResult<()> {
    let cursor = cursor.unwrap_or(0);
    let limit = limit.unwrap_or(20);
    
    // 验证分页参数
    super::validate_pagination(cursor, limit, 100)?;
    
    Ok(())
}
