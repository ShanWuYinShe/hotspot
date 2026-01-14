use axum::{
    http::StatusCode,
    response::{IntoResponse, Response},
    Json,
};
use serde_json::json;
use thiserror::Error;

#[derive(Debug, Error)]
pub enum AppError {
    // ============ 400xx：通用请求错误 ============
    #[error("参数错误: {0}")]
    BadRequest(String),

    #[error("参数格式错误: {0}")]
    InvalidFormat(String),

    #[error("参数值超出范围: {0}")]
    OutOfRange(String),

    #[error("参数验证失败: {0}")]
    InvalidInput(String),

    // ============ 401xx/403xx：认证与授权错误 ============
    #[error("认证失败: {0}")]
    Unauthorized(String),

    #[error("Token缺失: {0}")]
    TokenMissing(String),

    #[error("Token无效或过期: {0}")]
    TokenInvalid(String),

    #[error("权限不足: {0}")]
    Forbidden(String),

    #[error("非管理员")]
    NotAdmin,

    #[error("管理员权限不足: {0}")]
    AdminPermissionDenied(String),

    #[error("密码错误: {0}")]
    PasswordError(String),

    #[error("需要重新登录: {0}")]
    ReLoginRequired(String),

    // ============ 103xx：资源不存在 ============
    #[error("用户不存在: {0}")]
    UserNotFound(String),

    #[error("群组不存在: {0}")]
    GroupNotFound(String),

    #[error("瞬间不存在: {0}")]
    MomentNotFound(String),

    #[error("评论不存在: {0}")]
    CommentNotFound(String),

    #[error("消息/会话不存在: {0}")]
    ChatNotFound(String),

    #[error("文件/媒体不存在: {0}")]
    FileNotFound(String),

    #[error("资源不存在: {0}")]
    NotFound(String),

    #[error("游标无效或已过期: {0}")]
    CursorInvalidOrExpired(String),

    #[error("操作冲突: {0}")]
    Conflict(String),

    // ============ 403xx/409xx/429xx：业务逻辑与配额错误 ============
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

    // ============ 429xx：速率限制错误 ============
    #[error("请求过于频繁: {0}")]
    RateLimitExceeded(String),

    #[error("操作频率限制: {0}")]
    RateLimited(String),

    // ============ 409xx：资源状态冲突错误 ============
    #[error("资源状态冲突: {0}")]
    ResourceStateConflict(String),

    #[error("不是群组成员: {0}")]
    NotGroupMember(String),

    // ============ 500xx/503xx：服务器内部错误 ============
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
            // 100xx：通用请求错误
            Self::BadRequest(_) => 10000,
            Self::InvalidInput(_) => 10001,
            Self::InvalidFormat(_) => 10002,
            Self::OutOfRange(_) => 10003,

            // 101xx：认证失败
            Self::Unauthorized(_) => 10100,
            Self::Jwt(_) => 10101,
            Self::TokenInvalid(_) => 10101,
            Self::TokenMissing(_) => 10102,
            Self::PasswordError(_) => 10105,   // 密码错误
            Self::ReLoginRequired(_) => 10106, // 需要重新登录

            // 102xx：权限/访问控制
            Self::Forbidden(_) => 10200,             // 权限不足（通用）
            Self::NotAdmin => 10204,                 // 非管理员
            Self::AdminPermissionDenied(_) => 10205, // 管理员权限不足
            Self::PrivateChatNotAllowed(_) => 10210, // 对方不允许接收私聊
            Self::NotGroupMember(_) => 10416,        // 不是群组成员
            Self::BlockedByUser(_) => 10211,         // 已被对方拉黑
            Self::GroupPasswordError(_) => 10212,    // 群组密码错误
            Self::FollowNotAllowed(_) => 10213,      // 对方不允许被关注

            // 103xx：资源不存在
            Self::UserNotFound(_) => 10310,
            Self::GroupNotFound(_) => 10320,
            Self::MomentNotFound(_) => 10330,
            Self::CommentNotFound(_) => 10340,
            Self::ChatNotFound(_) => 10350,
            Self::FileNotFound(_) => 10360,
            Self::NotFound(_) => 10300,

            // 104xx：业务冲突
            Self::Conflict(_) => 10402,                   // 资源状态冲突
            Self::ResourceStateConflict(_) => 10402,      // 资源状态冲突
            Self::FollowRequestAlreadyExists(_) => 10413, // 关注请求已存在

            // 100xx：游标错误
            Self::CursorInvalidOrExpired(_) => 10007, // 游标无效或已过期

            // 105xx：速率限制/配额
            Self::RateLimitExceeded(_) => 10500, // 请求过于频繁（通用）
            Self::RateLimited(_) => 10500,       // 操作频率限制
            Self::MessageLimitReached(_) => 10510, // 消息数量已达上限

            // 500xx：服务器内部错误
            Self::InternalError(_) => 10600,
            Self::Database(_) => 10601,
            Self::Redis(_) => 10602,
            Self::BcryptError(_) => 10600,
            Self::Serialization(_) => 10600,
            Self::Internal => 10600,
        }
    }

    fn http_status(&self, _code: i32) -> StatusCode {
        // 所有业务错误统一返回 200 OK
        StatusCode::OK
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
        if code >= 50000 {
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
