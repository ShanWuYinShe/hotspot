//! 错误类型定义
//!
//! 所有错误码与 API 规范 (api_spec_errors.mdc) 一一对应
//! 错误码是契约的唯一来源，禁止在此文件之外定义新的错误码

use axum::{
    http::StatusCode,
    response::{IntoResponse, Response},
    Json,
};
use thiserror::Error;

use crate::dto::response::ApiResponse;

#[derive(Debug, Error)]
pub enum AppError {
    // ============================================================
    // 10000-10099：请求与参数（客户端可修复）
    // ============================================================

    /// 10000 - 参数错误（通用）
    #[error("参数错误: {0}")]
    BadRequest(String),

    /// 10001 - 缺少必填参数
    #[error("缺少必填参数: {0}")]
    MissingParameter(String),

    /// 10002 - 参数格式错误
    #[error("参数格式错误: {0}")]
    InvalidFormat(String),

    /// 10003 - 参数值超出范围
    #[error("参数值超出范围: {0}")]
    OutOfRange(String),

    /// 10004 - JSON 解析失败
    #[error("JSON 解析失败: {0}")]
    JsonParseError(String),

    /// 10005 - 请求体过大
    #[error("请求体过大: {0}")]
    PayloadTooLarge(String),

    /// 10006 - 游标格式错误（解析/解码失败）
    #[error("游标格式错误: {0}")]
    CursorFormatError(String),

    /// 10007 - 游标无效或已过期
    #[error("游标无效或已过期: {0}")]
    CursorInvalidOrExpired(String),

    // ============================================================
    // 10100-10199：认证（需要登录/刷新/重新登录）
    // ============================================================

    /// 10100 - 认证失败（通用）
    #[error("认证失败: {0}")]
    Unauthorized(String),

    /// 10101 - Access Token 无效或过期
    #[error("Access Token 无效或过期: {0}")]
    AccessTokenInvalid(String),

    /// 10102 - Access Token 缺失
    #[error("Access Token 缺失")]
    AccessTokenMissing,

    /// 10103 - Refresh Token 无效或过期
    #[error("Refresh Token 无效或过期: {0}")]
    RefreshTokenInvalid(String),

    /// 10104 - Refresh Token 缺失
    #[error("Refresh Token 缺失")]
    RefreshTokenMissing,

    /// 10105 - 密码错误
    #[error("密码错误: {0}")]
    PasswordError(String),

    /// 10106 - 需要重新登录（会话已撤销/风控要求）
    #[error("需要重新登录: {0}")]
    ReLoginRequired(String),

    /// 10107 - 验证 Token 无效或已过期（verify_token）
    #[error("验证 Token 无效或已过期: {0}")]
    VerifyTokenInvalid(String),

    // ============================================================
    // 10200-10299：权限与访问控制（无权限/被限制）
    // ============================================================

    /// 10200 - 权限不足（通用）
    #[error("权限不足: {0}")]
    Forbidden(String),

    /// 10201 - 账号被封禁
    #[error("账号被封禁: {0}")]
    AccountBanned(String),

    /// 10204 - 非管理员
    #[error("非管理员")]
    NotAdmin,

    /// 10205 - 管理员权限不足
    #[error("管理员权限不足: {0}")]
    AdminPermissionDenied(String),

    /// 10206 - 无法操作管理员账户
    #[error("无法操作管理员账户: {0}")]
    CannotOperateAdmin(String),

    /// 10210 - 对方不允许接收私聊
    #[error("对方不允许接收私聊: {0}")]
    PrivateChatNotAllowed(String),

    /// 10211 - 已被对方拉黑
    #[error("已被对方拉黑: {0}")]
    BlockedByUser(String),

    /// 10212 - 群组密码错误
    #[error("群组密码错误: {0}")]
    GroupPasswordError(String),

    /// 10213 - 对方不允许被关注
    #[error("对方不允许被关注: {0}")]
    FollowNotAllowed(String),

    // ============================================================
    // 10300-10399：资源不存在 / 不可访问
    // ============================================================

    /// 10300 - 资源不存在（泛化，向后兼容）
    #[error("资源不存在: {0}")]
    NotFound(String),

    /// 10310 - 用户不存在
    #[error("用户不存在: {0}")]
    UserNotFound(String),

    /// 10320 - 群组不存在
    #[error("群组不存在: {0}")]
    GroupNotFound(String),

    /// 10330 - 生活瞬间不存在
    #[error("瞬间不存在: {0}")]
    MomentNotFound(String),

    /// 10340 - 评论不存在
    #[error("评论不存在: {0}")]
    CommentNotFound(String),

    /// 10350 - 消息/会话不存在
    #[error("消息/会话不存在: {0}")]
    ChatNotFound(String),

    /// 10360 - 文件/媒体不存在
    #[error("文件/媒体不存在: {0}")]
    FileNotFound(String),

    // ============================================================
    // 10400-10499：业务冲突（客户端通常不应重试）
    // ============================================================

    /// 10400 - 资源已存在
    #[error("资源已存在: {0}")]
    AlreadyExists(String),

    /// 10401 - 资源已被删除
    #[error("资源已被删除: {0}")]
    AlreadyDeleted(String),

    /// 10402 - 资源状态冲突
    #[error("资源状态冲突: {0}")]
    Conflict(String),

    /// 10403 - 版本冲突（乐观锁冲突）
    #[error("版本冲突: {0}")]
    VersionConflict(String),

    /// 10410 - 业务规则冲突（通用）
    #[error("业务规则冲突: {0}")]
    BusinessRuleViolation(String),

    /// 10411 - 不能对自己执行此操作
    #[error("不能对自己执行此操作")]
    CannotOperateOnSelf,

    /// 10412 - 已是好友关系（双方已互相关注）
    #[error("已是好友关系")]
    AlreadyFriends,

    /// 10413 - 关注请求已存在
    #[error("关注请求已存在: {0}")]
    FollowRequestAlreadyExists(String),

    /// 10414 - 群组已满员
    #[error("群组已满员")]
    GroupFull,

    /// 10415 - 已在群组中
    #[error("已在群组中")]
    AlreadyInGroup,

    /// 10416 - 不是群组成员
    #[error("不是群组成员: {0}")]
    NotGroupMember(String),

    /// 10417 - 消息已被撤回
    #[error("消息已被撤回")]
    MessageAlreadyRecalled,

    /// 10418 - 操作时间窗口已过期
    #[error("操作时间窗口已过期")]
    OperationWindowExpired,

    /// 10419 - 用户被禁言（群组内禁言状态）
    #[error("用户被禁言")]
    UserMuted,

    /// 10420 - 不是好友关系
    #[error("不是好友关系")]
    NotFriends,

    // ============================================================
    // 10500-10599：速率限制与配额（可重试）
    // ============================================================

    /// 10500 - 请求过于频繁（通用）
    #[error("请求过于频繁: {0}")]
    RateLimited(String),

    /// 10501 - 超过每分钟请求限制
    #[error("超过每分钟请求限制")]
    MinuteRateLimitExceeded,

    /// 10502 - 超过每小时请求限制
    #[error("超过每小时请求限制")]
    HourlyRateLimitExceeded,

    /// 10503 - 超过每日请求限制
    #[error("超过每日请求限制")]
    DailyRateLimitExceeded,

    /// 10504 - 并发请求数过多
    #[error("并发请求数过多")]
    TooManyConcurrentRequests,

    /// 10510 - 消息数量已达上限（陌生人消息限制）
    #[error("消息数量已达上限: {0}")]
    MessageLimitReached(String),

    // ============================================================
    // 10600-10699：服务端错误（可重试/稍后再试）
    // ============================================================

    /// 10600 - 服务器内部错误（通用）
    #[error("服务器内部错误: {0}")]
    InternalError(String),

    /// 10601 - 数据库错误
    #[error("数据库错误: {0}")]
    DatabaseError(String),

    /// 10602 - 缓存服务错误
    #[error("缓存服务错误: {0}")]
    CacheError(String),

    /// 10603 - 消息队列错误
    #[error("消息队列错误: {0}")]
    MessageQueueError(String),

    /// 10604 - 外部服务调用失败
    #[error("外部服务调用失败: {0}")]
    ExternalServiceError(String),

    /// 10610 - 服务暂时不可用
    #[error("服务暂时不可用")]
    ServiceUnavailable,

    /// 10611 - 系统维护中
    #[error("系统维护中")]
    SystemMaintenance,

    /// 10612 - 服务过载
    #[error("服务过载")]
    ServiceOverloaded,

    // ============================================================
    // 20000-20999：WebSocket 业务期错误
    // ============================================================

    /// 20000 - WebSocket 业务期错误（通用）
    #[error("WebSocket 错误: {0}")]
    WebSocketError(String),

    /// 20001 - 鉴权失败（Auth 事件）
    #[error("WebSocket 鉴权失败: {0}")]
    WebSocketAuthFailed(String),

    /// 20002 - 心跳超时
    #[error("WebSocket 心跳超时")]
    WebSocketHeartbeatTimeout,

    /// 20003 - 消息格式错误
    #[error("WebSocket 消息格式错误: {0}")]
    WebSocketMessageFormatError(String),

    /// 20004 - 协议不兼容
    #[error("WebSocket 协议不兼容: {0}")]
    WebSocketProtocolIncompatible(String),

    /// 20100 - 订阅失败（通用）
    #[error("订阅失败: {0}")]
    SubscriptionFailed(String),

    /// 20101 - 订阅频道不存在
    #[error("订阅频道不存在: {0}")]
    SubscriptionChannelNotFound(String),

    /// 20102 - 订阅权限不足
    #[error("订阅权限不足: {0}")]
    SubscriptionPermissionDenied(String),

    /// 20103 - 订阅数量超出限制
    #[error("订阅数量超出限制")]
    SubscriptionLimitExceeded,

    /// 20104 - 无效的订阅项格式
    #[error("无效的订阅项格式: {0}")]
    InvalidSubscriptionFormat(String),

    /// 20105 - 批量项数超出限制
    #[error("批量项数超出限制")]
    BatchLimitExceeded,

    // ============================================================
    // 22000-22099：幂等性相关错误
    // ============================================================

    /// 22000 - 幂等键冲突
    #[error("幂等键冲突: {0}")]
    IdempotencyKeyConflict(String),

    /// 22001 - 幂等缓存失效
    #[error("幂等缓存失效")]
    IdempotencyCacheExpired,

    /// 22002 - 幂等操作正在处理中
    #[error("操作正在处理中")]
    IdempotencyOperationInProgress,

    // ============================================================
    // 框架/库错误转换（内部使用，统一映射到 10600）
    // ============================================================

    /// SQLx 数据库错误
    #[error("数据库错误")]
    Sqlx(#[from] sqlx::Error),

    /// Redis 错误
    #[error("Redis 错误")]
    Redis(#[from] redis::RedisError),

    /// JWT 错误
    #[error("JWT 错误")]
    Jwt(#[from] jsonwebtoken::errors::Error),

    /// Bcrypt 错误
    #[error("密码哈希错误")]
    Bcrypt(#[from] bcrypt::BcryptError),

    /// Serde JSON 错误
    #[error("序列化错误")]
    SerdeJson(#[from] serde_json::Error),

}

impl AppError {
    /// 获取错误码
    pub fn error_code(&self) -> i32 {
        match self {
            // 10000-10099：请求与参数
            Self::BadRequest(_) => 10000,
            Self::MissingParameter(_) => 10001,
            Self::InvalidFormat(_) => 10002,
            Self::OutOfRange(_) => 10003,
            Self::JsonParseError(_) => 10004,
            Self::PayloadTooLarge(_) => 10005,
            Self::CursorFormatError(_) => 10006,
            Self::CursorInvalidOrExpired(_) => 10007,

            // 10100-10199：认证
            Self::Unauthorized(_) => 10100,
            Self::AccessTokenInvalid(_) | Self::Jwt(_) => 10101,
            Self::AccessTokenMissing => 10102,
            Self::RefreshTokenInvalid(_) => 10103,
            Self::RefreshTokenMissing => 10104,
            Self::PasswordError(_) | Self::Bcrypt(_) => 10105,
            Self::ReLoginRequired(_) => 10106,
            Self::VerifyTokenInvalid(_) => 10107,

            // 10200-10299：权限与访问控制
            Self::Forbidden(_) => 10200,
            Self::AccountBanned(_) => 10201,
            Self::NotAdmin => 10204,
            Self::AdminPermissionDenied(_) => 10205,
            Self::CannotOperateAdmin(_) => 10206,
            Self::PrivateChatNotAllowed(_) => 10210,
            Self::BlockedByUser(_) => 10211,
            Self::GroupPasswordError(_) => 10212,
            Self::FollowNotAllowed(_) => 10213,

            // 10300-10399：资源不存在
            Self::NotFound(_) => 10300,
            Self::UserNotFound(_) => 10310,
            Self::GroupNotFound(_) => 10320,
            Self::MomentNotFound(_) => 10330,
            Self::CommentNotFound(_) => 10340,
            Self::ChatNotFound(_) => 10350,
            Self::FileNotFound(_) => 10360,

            // 10400-10499：业务冲突
            Self::AlreadyExists(_) => 10400,
            Self::AlreadyDeleted(_) => 10401,
            Self::Conflict(_) => 10402,
            Self::VersionConflict(_) => 10403,
            Self::BusinessRuleViolation(_) => 10410,
            Self::CannotOperateOnSelf => 10411,
            Self::AlreadyFriends => 10412,
            Self::FollowRequestAlreadyExists(_) => 10413,
            Self::GroupFull => 10414,
            Self::AlreadyInGroup => 10415,
            Self::NotGroupMember(_) => 10416,
            Self::MessageAlreadyRecalled => 10417,
            Self::OperationWindowExpired => 10418,
            Self::UserMuted => 10419,
            Self::NotFriends => 10420,

            // 10500-10599：速率限制与配额
            Self::RateLimited(_) => 10500,
            Self::MinuteRateLimitExceeded => 10501,
            Self::HourlyRateLimitExceeded => 10502,
            Self::DailyRateLimitExceeded => 10503,
            Self::TooManyConcurrentRequests => 10504,
            Self::MessageLimitReached(_) => 10510,

            // 10600-10699：服务端错误
            Self::InternalError(_) => 10600,
            Self::DatabaseError(_) | Self::Sqlx(_) => 10601,
            Self::CacheError(_) | Self::Redis(_) => 10602,
            Self::MessageQueueError(_) => 10603,
            Self::ExternalServiceError(_) => 10604,
            Self::ServiceUnavailable => 10610,
            Self::SystemMaintenance => 10611,
            Self::ServiceOverloaded => 10612,
            Self::SerdeJson(_) => 10600,

            // 20000-20999：WebSocket 业务期错误
            Self::WebSocketError(_) => 20000,
            Self::WebSocketAuthFailed(_) => 20001,
            Self::WebSocketHeartbeatTimeout => 20002,
            Self::WebSocketMessageFormatError(_) => 20003,
            Self::WebSocketProtocolIncompatible(_) => 20004,
            Self::SubscriptionFailed(_) => 20100,
            Self::SubscriptionChannelNotFound(_) => 20101,
            Self::SubscriptionPermissionDenied(_) => 20102,
            Self::SubscriptionLimitExceeded => 20103,
            Self::InvalidSubscriptionFormat(_) => 20104,
            Self::BatchLimitExceeded => 20105,

            // 22000-22099：幂等性相关错误
            Self::IdempotencyKeyConflict(_) => 22000,
            Self::IdempotencyCacheExpired => 22001,
            Self::IdempotencyOperationInProgress => 22002,
        }
    }

    /// HTTP 状态码（API 固定返回 200）
    fn http_status(&self) -> StatusCode {
        // 根据 API 规范，所有业务错误统一返回 HTTP 200
        // 业务成功与否仅通过 ApiResponse.code 判断
        StatusCode::OK
    }

    /// 判断是否为服务端错误（用于日志级别判断）
    fn is_server_error(&self) -> bool {
        let code = self.error_code();
        (10600..10700).contains(&code)
    }
}

impl IntoResponse for AppError {
    fn into_response(self) -> Response {
        let code = self.error_code();
        let status = self.http_status();
        let message = self.to_string();

        // 根据错误类型决定日志级别
        if self.is_server_error() {
            tracing::error!(code = code, message = %message, "request failed");
        } else {
            tracing::warn!(code = code, message = %message, "request failed");
        }

        let body = Json(ApiResponse::<()>::error(code, message));

        let mut resp = (status, body).into_response();

        // 速率限制错误：添加 Retry-After 响应头
        if (10500..10600).contains(&code) {
            resp.headers_mut().insert(
                axum::http::header::RETRY_AFTER,
                axum::http::HeaderValue::from_static("60"),
            );
        }

        resp
    }
}

// ============================================================
// 便捷类型别名
// ============================================================

pub type Result<T> = std::result::Result<T, AppError>;


// ============================================================
// 便捷构造方法
// ============================================================

impl AppError {
    /// 创建参数错误
    pub fn bad_request(msg: impl Into<String>) -> Self {
        Self::BadRequest(msg.into())
    }

    /// 创建缺少参数错误
    pub fn missing_param(param: impl Into<String>) -> Self {
        Self::MissingParameter(param.into())
    }

    /// 创建格式错误
    pub fn invalid_format(msg: impl Into<String>) -> Self {
        Self::InvalidFormat(msg.into())
    }

    /// 创建权限不足错误
    pub fn forbidden(msg: impl Into<String>) -> Self {
        Self::Forbidden(msg.into())
    }

    /// 创建资源不存在错误
    pub fn not_found(msg: impl Into<String>) -> Self {
        Self::NotFound(msg.into())
    }

    /// 创建内部错误
    pub fn internal(msg: impl Into<String>) -> Self {
        Self::InternalError(msg.into())
    }

    /// 创建冲突错误
    pub fn conflict(msg: impl Into<String>) -> Self {
        Self::Conflict(msg.into())
    }
}
