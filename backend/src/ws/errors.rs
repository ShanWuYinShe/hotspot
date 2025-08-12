//! WebSocket错误处理

use serde::Serialize;
use std::fmt;
use crate::ws::constants::{error_codes, error_messages};

/// WebSocket错误类型
#[derive(Debug, Clone, Serialize)]
pub struct WsError {
    pub code: String,
    pub message: String,
    pub details: Option<String>,
}

impl WsError {
    /// 创建新的WebSocket错误
    pub fn new(code: &str, message: &str) -> Self {
        Self {
            code: code.to_string(),
            message: message.to_string(),
            details: None,
        }
    }
    
    /// 创建带详细信息的WebSocket错误
    pub fn with_details(code: &str, message: &str, details: &str) -> Self {
        Self {
            code: code.to_string(),
            message: message.to_string(),
            details: Some(details.to_string()),
        }
    }
    
    /// 无效消息错误
    pub fn invalid_message(details: &str) -> Self {
        Self::with_details(
            error_codes::INVALID_JSON,
            error_messages::INVALID_JSON_FORMAT,
            details,
        )
    }
    
    /// 无效令牌错误
    #[allow(dead_code)]
    pub fn invalid_token() -> Self {
        Self::new(
            error_codes::INVALID_TOKEN,
            error_messages::INVALID_TOKEN,
        )
    }
    
    /// 未授权错误
    #[allow(dead_code)]
    pub fn unauthorized() -> Self {
        Self::new(
            error_codes::UNAUTHORIZED,
            error_messages::UNAUTHORIZED_ACCESS,
        )
    }
    
    /// 禁止访问（权限不足）
    pub fn forbidden(details: &str) -> Self {
        Self::with_details(
            error_codes::FORBIDDEN,
            error_messages::FORBIDDEN,
            details,
        )
    }
    
    /// 房间不存在错误
    #[allow(dead_code)]
    pub fn room_not_found(room_id: &str) -> Self {
        Self::with_details(
            error_codes::ROOM_NOT_FOUND,
            error_messages::ROOM_NOT_EXISTS,
            &format!("房间ID: {}", room_id),
        )
    }
    
    /// 内部错误
    pub fn internal_error(details: &str) -> Self {
        Self::with_details(
            error_codes::INTERNAL_ERROR,
            error_messages::INTERNAL_SERVER_ERROR,
            details,
        )
    }
    
    /// 连接数量超限错误
    pub fn connection_limit_exceeded() -> Self {
        Self::new(
            error_codes::CONNECTION_LIMIT_EXCEEDED,
            error_messages::TOO_MANY_CONNECTIONS,
        )
    }
    
    /// 消息过大错误
    #[allow(dead_code)]
    pub fn message_too_large(size: usize, max_size: usize) -> Self {
        Self::with_details(
            error_codes::MESSAGE_TOO_LARGE,
            error_messages::MESSAGE_TOO_LARGE,
            &format!("消息大小: {}字节, 最大允许: {}字节", size, max_size),
        )
    }
}

impl fmt::Display for WsError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        if let Some(details) = &self.details {
            write!(f, "[{}] {}: {}", self.code, self.message, details)
        } else {
            write!(f, "[{}] {}", self.code, self.message)
        }
    }
}

impl std::error::Error for WsError {}

/// WebSocket结果类型
pub type WsResult<T> = Result<T, WsError>;
