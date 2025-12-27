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

    #[error("Token无效或过期: {0}")]
    TokenInvalid(String),

    #[error("权限不足: {0}")]
    Forbidden(String),

    #[error("密码错误: {0}")]
    PasswordError(String),

    #[error("需要重新登录: {0}")]
    ReLoginRequired(String),

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

    #[error("关注请求已存在: {0}")]
    FollowRequestAlreadyExists(String),

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
            Self::BadRequest(_) => 1000,   // 参数错误（通用）
            Self::InvalidInput(_) => 1001, // 缺少必填参数
            // 1100-1199：认证与授权错误
            Self::Unauthorized(_) => 1100,    // 认证失败（通用）
            Self::Jwt(_) => 1101,             // Token无效或过期
            Self::TokenMissing(_) => 1102,    // Token缺失
            Self::TokenInvalid(_) => 1101,    // Token无效或过期
            Self::Forbidden(_) => 1103,       // 权限不足
            Self::PasswordError(_) => 1105,   // 密码错误
            Self::ReLoginRequired(_) => 1106, // 需要重新登录
            // 1200-1299：资源错误
            Self::NotFound(_) => 1200, // 资源不存在
            Self::Conflict(_) => 1203, // 资源状态冲突
            // 1300-1399：业务逻辑错误
            Self::PrivateChatNotAllowed(_) => 1309, // 对方不允许接收私聊
            Self::BlockedByUser(_) => 1310,         // 已被对方拉黑
            Self::GroupPasswordError(_) => 1311,    // 群组密码错误
            Self::MessageLimitReached(_) => 1312,   // 消息数量已达上限
            Self::FollowNotAllowed(_) => 1313,      // 对方不允许被关注
            Self::FollowRequestAlreadyExists(_) => 1303, // 关注请求已存在
            // 1400-1499：速率限制错误
            Self::RateLimitExceeded(_) => 1400, // 请求过于频繁（通用）
            // 5000-5099：服务器内部错误
            Self::InternalError(_) => 5000, // 服务器内部错误（通用）
            Self::Database(_) => 5001,      // 数据库错误
            Self::Redis(_) => 5002,         // 缓存服务错误
            Self::BcryptError(_) => 5000,
            Self::Serialization(_) => 5000,
            Self::Internal => 5000,
        }
    }

    fn http_status(&self, code: i32) -> StatusCode {
        match code {
            0 => StatusCode::OK,
            c if (1000..=1099).contains(&c) => StatusCode::BAD_REQUEST,
            // 认证与授权
            1102 => StatusCode::UNAUTHORIZED,
            1103 => StatusCode::FORBIDDEN,
            1106 => StatusCode::UNAUTHORIZED,
            c if (1100..=1199).contains(&c) => StatusCode::UNAUTHORIZED,
            // 资源
            c if (1200..=1299).contains(&c) => match c {
                1200 => StatusCode::NOT_FOUND,
                1203 => StatusCode::CONFLICT,
                _ => StatusCode::NOT_FOUND,
            },
            // 业务规则
            c if (1300..=1399).contains(&c) => match c {
                1300 | 1303 | 1304 | 1305 | 1306 | 1307 | 1308 => StatusCode::CONFLICT,
                1312 => StatusCode::TOO_MANY_REQUESTS,
                _ => StatusCode::FORBIDDEN,
            },
            // 速率限制
            c if (1400..=1499).contains(&c) => StatusCode::TOO_MANY_REQUESTS,
            // 上传相关
            c if (1500..=1599).contains(&c) => StatusCode::BAD_REQUEST,
            // 幂等
            c if (2200..=2299).contains(&c) => StatusCode::CONFLICT,
            // 服务器内部
            _ => StatusCode::INTERNAL_SERVER_ERROR,
        }
    }
}

impl IntoResponse for AppError {
    fn into_response(self) -> Response {
        let code = self.error_code();
        let status = self.http_status(code);
        let message = self.to_string();

        // 记录错误日志（尽量简洁，避免把同一条信息重复打印多次）
        //
        // request_id 会由 TraceLayer span / request_id_middleware 自动注入到上下文与响应中，
        // 线上排查直接 grep request_id 即可。
        if code >= 5000 {
            tracing::error!(code = code, message = %message, "request failed");
        } else {
            tracing::warn!(code = code, message = %message, "request failed");
        }

        let body = Json(json!({
            "code": code,
            "message": message,
        }));

        let mut resp = (status, body).into_response();

        // 429：建议客户端稍后重试
        if status == StatusCode::TOO_MANY_REQUESTS {
            resp.headers_mut().insert(
                axum::http::header::RETRY_AFTER,
                axum::http::HeaderValue::from_static("60"),
            );
        }

        resp
    }
}

pub type Result<T> = std::result::Result<T, AppError>;
