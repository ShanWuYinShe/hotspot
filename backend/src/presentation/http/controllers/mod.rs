/// HTTP控制器模块
///
/// 负责处理HTTP请求，调用应用服务，返回HTTP响应
/// 遵循单一职责原则，只负责协议转换，不包含业务逻辑

pub mod admin_controller;
pub mod group_controller;
pub mod health_controller;
pub mod enhanced_health_controller;
pub mod monitoring_controller;
pub mod message_controller;
pub mod moment_controller;
pub mod user_controller;
pub mod search_controller;
pub mod activity_controller;
pub mod friend_controller;
pub mod upload_controller;

// 重新导出控制器类型
pub use admin_controller::*;
pub use group_controller::*;
pub use health_controller::*;
pub use enhanced_health_controller::*;
pub use monitoring_controller::*;
pub use message_controller::MessageController;
pub use moment_controller::*;
pub use user_controller::UserController;
pub use search_controller::*;
pub use activity_controller::*;
pub use friend_controller::*;
pub use upload_controller::*;

use crate::domain::DomainError;
use axum::Json;
use axum::http::StatusCode;
use axum::response::{IntoResponse, Response};
use serde_json::json;

/// 控制器基础trait
#[async_trait::async_trait]
pub trait Controller: Send + Sync {
    /// 控制器名称
    fn name(&self) -> &'static str;
}

/// HTTP错误响应处理 - 根据hotspot.md规范统一使用HTTP 200和业务错误码
pub fn map_domain_error_to_http_response(error: DomainError) -> Response {
    let (error_code, message) = match error {
        // 未找到错误 -> 1004
        DomainError::User(crate::domain::UserDomainError::UserNotFound { .. })
        | DomainError::Group(crate::domain::GroupDomainError::GroupNotFound { .. })
        | DomainError::Message(crate::domain::MessageDomainError::MessageNotFound { .. })
        | DomainError::Moment(crate::domain::MomentDomainError::MomentNotFound { .. }) => {
            (1004, error.user_message())
        }

        // 权限错误 -> 1003
        DomainError::Group(crate::domain::GroupDomainError::InsufficientPermission { .. })
        | DomainError::Message(crate::domain::MessageDomainError::NotMessageSender)
        | DomainError::Moment(crate::domain::MomentDomainError::NotMomentAuthor) => {
            (1003, error.user_message())
        }

        // 验证错误 -> 1000
        DomainError::User(crate::domain::UserDomainError::ValidationFailed { .. })
        | DomainError::Group(crate::domain::GroupDomainError::ValidationFailed { .. })
        | DomainError::Message(crate::domain::MessageDomainError::ValidationFailed { .. })
        | DomainError::Moment(crate::domain::MomentDomainError::ValidationFailed { .. }) => {
            (1000, error.user_message())
        }

        // 冲突错误 -> 1005
        DomainError::User(crate::domain::UserDomainError::LoginIdAlreadyExists { .. })
        | DomainError::Group(crate::domain::GroupDomainError::GroupNameAlreadyExists { .. })
        | DomainError::Group(crate::domain::GroupDomainError::UserAlreadyMember { .. }) => {
            (1005, error.user_message())
        }

        // 认证错误 -> 1002
        DomainError::User(crate::domain::UserDomainError::PasswordVerificationFailed) => {
            (1002, error.user_message())
        }

        // 业务规则违反 -> 1000
        DomainError::BusinessRuleViolation { .. } => {
            (1000, error.user_message())
        }

        // 并发冲突 -> 1005
        DomainError::ConcurrencyConflict { .. } => {
            (1005, error.user_message())
        }

        // 其他错误 -> 5000
        _ => {
            log::error!("Unhandled domain error: {:?}", error);
            (5000, "服务器内部错误".to_string())
        }
    };

    let error_response = json!({
        "code": error_code,
        "msg": message,
        "data": null
    });

    // 根据hotspot.md规范，统一返回HTTP 200状态码
    (StatusCode::OK, Json(error_response)).into_response()
}

/// 成功响应帮助函数
pub fn ok_response<T: serde::Serialize>(data: T) -> Response {
    let response = json!({
        "code": 0,
        "msg": "success",
        "data": data
    });

    (StatusCode::OK, Json(response)).into_response()
}

/// 创建的资源响应帮助函数
pub fn created_response<T: serde::Serialize>(data: T) -> Response {
    let response = json!({
        "code": 0,
        "msg": "created",
        "data": data
    });

    (StatusCode::CREATED, Json(response)).into_response()
}

/// 无内容响应帮助函数
pub fn no_content_response() -> Response {
    let response = json!({
        "code": 0,
        "msg": "success",
        "data": null
    });

    (StatusCode::OK, Json(response)).into_response()
}

/// 分页响应帮助函数
pub fn paginated_response<T: serde::Serialize>(
    items: Vec<T>,
    has_more: bool,
    next_cursor: Option<i64>,
) -> Response {
    let response = json!({
        "code": 0,
        "msg": "success",
        "data": {
            "items": items,
            "pagination": {
                "has_more": has_more,
                "next_cursor": next_cursor
            }
        }
    });

    (StatusCode::OK, Json(response)).into_response()
}

/// 错误响应帮助函数
pub fn error_response(
    status_code: StatusCode,
    message: &str,
    data: Option<serde_json::Value>,
) -> Response {
    let response = json!({
        "code": 1,
        "msg": message,
        "data": data
    });

    (status_code, Json(response)).into_response()
}
