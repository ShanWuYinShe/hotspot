use axum::{
    http::StatusCode,
    response::{IntoResponse, Response},
    Json,
};
use serde_json::json;
use thiserror::Error;

#[derive(Debug, Error)]
pub enum AppError {
    // ============ 1000-1099：通用请求错误 ============
    #[error("参数错误: {0}")]
    BadRequest(String),

    #[error("参数验证失败: {0}")]
    InvalidInput(String),

    // ============ 1100-1199：认证与授权错误 ============
    #[error("认证失败: {0}")]
    Unauthorized(String),

    #[error("Token缺失: {0}")]
    TokenMissing(String),

    #[error("权限不足: {0}")]
    Forbidden(String),

    #[error("密码错误: {0}")]
    PasswordError(String),

    // ============ 1200-1299：资源错误 ============
    #[error("资源不存在: {0}")]
    NotFound(String),

    #[error("操作冲突: {0}")]
    Conflict(String),

    // ============ 1300-1399：业务逻辑错误 ============
    #[error("对方不允许接收私聊: {0}")]
    PrivateChatNotAllowed(String),

    #[error("已被对方拉黑: {0}")]
    BlockedByUser(String),

    #[error("群组密码错误: {0}")]
    GroupPasswordError(String),

    #[error("消息数量已达上限: {0}")]
    MessageLimitReached(String),

    #[error("对方不允许被关注: {0}")]
    FollowNotAllowed(String),

    // ============ 1400-1499：速率限制错误 ============
    #[error("请求过于频繁: {0}")]
    RateLimitExceeded(String),

    // ============ 5000-5099：服务器内部错误 ============
    #[error("内部错误: {0}")]
    InternalError(String),

    #[error("数据库错误: {0}")]
    Database(#[from] sqlx::Error),

    #[error("Redis错误: {0}")]
    Redis(#[from] redis::RedisError),

    #[error("JWT错误: {0}")]
    Jwt(#[from] jsonwebtoken::errors::Error),

    #[error("密码哈希错误")]
    BcryptError(#[from] bcrypt::BcryptError),

    #[error("序列化错误: {0}")]
    Serialization(#[from] serde_json::Error),

    #[error("内部服务器错误")]
    Internal,
}

impl AppError {
    fn error_code(&self) -> i32 {
        match self {
            // 1000-1099：通用请求错误
            Self::BadRequest(_) => 1000,              // 参数错误（通用）
            Self::InvalidInput(_) => 1001,            // 缺少必填参数
            // 1100-1199：认证与授权错误
            Self::Unauthorized(_) => 1100,            // 认证失败（通用）
            Self::Jwt(_) => 1101,                     // Token无效或过期
            Self::TokenMissing(_) => 1102,            // Token缺失
            Self::Forbidden(_) => 1103,               // 权限不足
            Self::PasswordError(_) => 1105,           // 密码错误
            // 1200-1299：资源错误
            Self::NotFound(_) => 1200,                // 资源不存在
            Self::Conflict(_) => 1203,                // 资源状态冲突
            // 1300-1399：业务逻辑错误
            Self::PrivateChatNotAllowed(_) => 1309,   // 对方不允许接收私聊
            Self::BlockedByUser(_) => 1310,           // 已被对方拉黑
            Self::GroupPasswordError(_) => 1311,      // 群组密码错误
            Self::MessageLimitReached(_) => 1312,     // 消息数量已达上限
            Self::FollowNotAllowed(_) => 1313,        // 对方不允许被关注
            // 1400-1499：速率限制错误
            Self::RateLimitExceeded(_) => 1400,       // 请求过于频繁（通用）
            // 5000-5099：服务器内部错误
            Self::InternalError(_) => 5000,           // 服务器内部错误（通用）
            Self::Database(_) => 5001,                // 数据库错误
            Self::Redis(_) => 5002,                   // 缓存服务错误
            Self::BcryptError(_) => 5000,
            Self::Serialization(_) => 5000,
            Self::Internal => 5000,
        }
    }
}

impl IntoResponse for AppError {
    fn into_response(self) -> Response {
        let status = StatusCode::OK; // 统一返回 200
        let code = self.error_code();
        let message = self.to_string();

        // 记录错误日志
        if code >= 5000 {
            tracing::error!(error = %self, "Internal server error");
        } else {
            tracing::warn!(error = %self, "Client error");
        }

        let body = Json(json!({
            "code": code,
            "message": message,
            "data": null,
        }));

        (status, body).into_response()
    }
}

pub type Result<T> = std::result::Result<T, AppError>;

