use crate::utils::response::AppError;

/// 验证消息内容
pub fn validate_message_content(content: &str, message_type: &str) -> Result<(), AppError> {
    // 验证消息类型
    match message_type {
        "text" => {
            // 文本消息长度验证
            if content.is_empty() || content.len() > 2000 {
                return Err(AppError::BadRequest(
                    "文本消息长度应在1-2000个字符之间".to_string(),
                ));
            }
        }
        "image" => {
            // 图片消息URL验证
            if !content.starts_with("http") || content.len() > 500 {
                return Err(AppError::BadRequest("图片URL格式不正确".to_string()));
            }
        }
        "location" => {
            // 位置消息格式验证
            if content.is_empty() || content.len() > 500 {
                return Err(AppError::BadRequest("位置消息格式不正确".to_string()));
            }
        }
        _ => {
            return Err(AppError::BadRequest(format!(
                "不支持的消息类型: {}",
                message_type
            )));
        }
    }

    Ok(())
}

/// 验证会话ID是否提供
pub fn validate_conversation_id(conversation_id: Option<&String>) -> Result<(), AppError> {
    if conversation_id.is_none() || conversation_id.unwrap().is_empty() {
        return Err(AppError::BadRequest("会话ID不能为空".to_string()));
    }
    Ok(())
}

/// 验证接收者ID
pub fn validate_recipient_id(recipient_id: &str) -> Result<i64, AppError> {
    recipient_id
        .parse::<i64>()
        .map_err(|_| AppError::BadRequest("无效的用户ID格式".to_string()))
}
