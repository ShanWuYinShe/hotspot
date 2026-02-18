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
    /// 10300 - 资源不存在（通用）
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

    // ============================================================
    // 10600-10699：服务端错误（可重试/稍后再试）
    // 安全说明：统一返回 10600，禁止向客户端暴露具体的技术栈细节
    // ============================================================
    /// 10600 - 服务器内部错误（通用）
    /// 注：数据库/缓存/消息队列/外部服务错误均统一映射到此错误码
    /// 详细错误信息仅记录在服务端日志中，通过 request_id 关联排查
    #[error("服务器内部错误: {0}")]
    InternalError(String),

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

    /// 20102 - 订阅权限不足
    #[error("订阅权限不足: {0}")]
    SubscriptionPermissionDenied(String),

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
    ///
    /// 错误码严格遵循 spec/api_spec_errors.mdc 定义：
    /// - 多个 enum variant 映射到同一个 spec 错误码
    /// - 具体业务场景通过 message 字段区分，客户端禁止解析 message
    /// - "编号间隙为预留空间，禁止随意填补"
    pub fn error_code(&self) -> i32 {
        match self {
            // 10000：参数错误（所有参数校验类 variant 统一映射）
            Self::BadRequest(_)
            | Self::MissingParameter(_)
            | Self::InvalidFormat(_)
            | Self::OutOfRange(_)
            | Self::JsonParseError(_)
            | Self::PayloadTooLarge(_)
            | Self::CursorFormatError(_)
            | Self::CursorInvalidOrExpired(_) => 10000,

            // 10100：认证失败（所有认证类 variant 统一映射）
            Self::Unauthorized(_)
            | Self::Jwt(_)
            | Self::AccessTokenMissing
            | Self::RefreshTokenInvalid(_)
            | Self::RefreshTokenMissing
            | Self::PasswordError(_)
            | Self::Bcrypt(_)
            | Self::VerifyTokenInvalid(_) => 10100,

            // 10200：权限不足/访问受限
            Self::Forbidden(_)
            | Self::NotAdmin
            | Self::AdminPermissionDenied(_)
            | Self::CannotOperateAdmin(_)
            | Self::BlockedByUser(_)
            | Self::GroupPasswordError(_)
            | Self::FollowNotAllowed(_) => 10200,

            // 10201：账号被封禁（独立保留：客户端需强制登出+封禁页）
            Self::AccountBanned(_) => 10201,

            // 10300：资源不存在（所有 NotFound 类 variant 统一映射）
            Self::NotFound(_)
            | Self::UserNotFound(_)
            | Self::GroupNotFound(_)
            | Self::MomentNotFound(_)
            | Self::CommentNotFound(_)
            | Self::ChatNotFound(_)
            | Self::FileNotFound(_) => 10300,

            // 10400：业务冲突（所有业务冲突类 variant 统一映射）
            Self::AlreadyExists(_)
            | Self::Conflict(_)
            | Self::VersionConflict(_)
            | Self::BusinessRuleViolation(_)
            | Self::CannotOperateOnSelf
            | Self::GroupFull
            | Self::AlreadyInGroup
            | Self::NotGroupMember(_)
            | Self::MessageAlreadyRecalled
            | Self::OperationWindowExpired
            | Self::UserMuted
            | Self::NotFriends => 10400,

            // 10500：请求过于频繁（所有限流类 variant 统一映射）
            Self::RateLimited(_)
            | Self::MinuteRateLimitExceeded
            | Self::HourlyRateLimitExceeded
            | Self::DailyRateLimitExceeded
            | Self::TooManyConcurrentRequests => 10500,

            // 10600：服务器错误（所有内部错误统一映射，不暴露技术细节）
            Self::InternalError(_)
            | Self::Sqlx(_)
            | Self::Redis(_)
            | Self::SerdeJson(_)
            | Self::ServiceUnavailable
            | Self::SystemMaintenance
            | Self::ServiceOverloaded => 10600,

            // 20000：WebSocket 连接/协议错误
            Self::WebSocketError(_)
            | Self::WebSocketAuthFailed(_)
            | Self::WebSocketHeartbeatTimeout
            | Self::WebSocketMessageFormatError(_)
            | Self::WebSocketProtocolIncompatible(_) => 20000,

            // 20100：WebSocket 订阅错误
            Self::SubscriptionFailed(_)
            | Self::SubscriptionPermissionDenied(_)
            | Self::InvalidSubscriptionFormat(_)
            | Self::BatchLimitExceeded => 20100,

            // 22000-22002：幂等性相关错误（各自驱动不同重试策略，不合并）
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

        // 预先提取速率限制 Retry-After（需在 self 引用结束前完成）
        let retry_after: Option<u32> = if (10500..10600).contains(&code) {
            Some(match &self {
                Self::MinuteRateLimitExceeded => 60,
                Self::HourlyRateLimitExceeded => 300,
                Self::DailyRateLimitExceeded => 3600,
                Self::RateLimited(_) | Self::TooManyConcurrentRequests => 60,
                _ => 60,
            })
        } else {
            None
        };

        // 10600 服务端错误：安全脱敏
        // - 完整错误信息（含数据库/Redis/序列化等技术细节）仅记录在服务端日志
        // - 客户端只收到通用 "服务器内部错误"，不暴露技术栈细节
        let client_message = if self.is_server_error() {
            tracing::error!(code = code, error = ?self, "服务端错误（客户端收到通用错误信息）");
            "服务器内部错误".to_string()
        } else {
            let message = self.to_string();
            tracing::warn!(code = code, message = %message, "request failed");
            message
        };

        let body = Json(ApiResponse::<()>::error(code, client_message));
        let mut resp = (status, body).into_response();

        // 速率限制错误：根据限流类型设置动态 Retry-After
        if let Some(seconds) = retry_after {
            if let Ok(val) = axum::http::HeaderValue::try_from(seconds.to_string()) {
                resp.headers_mut()
                    .insert(axum::http::header::RETRY_AFTER, val);
            }
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
    /// 创建内部错误
    pub fn internal(msg: impl Into<String>) -> Self {
        Self::InternalError(msg.into())
    }
}

/// 预热错误码映射（用于观测/告警，不影响业务逻辑）
pub fn warmup_error_catalog() {
    let _ = [
        AppError::VersionConflict("版本冲突".to_string()),
        AppError::TooManyConcurrentRequests,
        AppError::ServiceUnavailable,
        AppError::SystemMaintenance,
        AppError::ServiceOverloaded,
    ];
}
