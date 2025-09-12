/// HTTP控制器模块
///
/// 负责处理HTTP请求，调用应用服务，返回HTTP响应
/// 遵循单一职责原则，只负责协议转换，不包含业务逻辑
pub mod admin_controller;
pub mod group_controller;
pub mod health_controller;
pub mod enhanced_health_controller;
pub mod monitoring_controller;
pub mod metrics_controller;
// pub mod query_stats_controller;
// pub mod cache_monitor_controller;
pub mod unified_monitoring_controller;
pub mod message_controller;
pub mod moment_controller;
pub mod user_controller;
pub mod search_controller;
pub mod activity_controller;
pub mod upload_controller;
pub mod comment_controller;
pub mod friend_controller;

// 重新导出控制器类型
pub use admin_controller::*;
pub use group_controller::*;
pub use health_controller::*;
pub use enhanced_health_controller::*;
pub use monitoring_controller::*;
pub use metrics_controller::*;
// pub use query_stats_controller::*;
// pub use cache_monitor_controller::*;
pub use unified_monitoring_controller::*;
// pub use message_controller::MessageController; // 暂时禁用
pub use moment_controller::*;
pub use user_controller::UserController;
pub use search_controller::*;
pub use activity_controller::*;
pub use upload_controller::*;
pub use comment_controller::*;
// pub use friend_controller::*; // 暂时禁用

use crate::domain::DomainError;
use crate::application::ApplicationError;
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
        DomainError::UserNotFound
        | DomainError::GroupNotFound
        | DomainError::MessageNotFound
        | DomainError::MomentNotFound
        | DomainError::ConversationNotFound
        | DomainError::NotFound { .. } => {
            (1004, error.user_message())
        }

        // 权限错误 -> 1003
        DomainError::InsufficientPermissions
        | DomainError::NotMomentAuthor => {
            (1003, error.user_message())
        }

        // 验证错误 -> 1000
        DomainError::ValidationError { .. }
        | DomainError::ValidationFailed { errors: _ }
        | DomainError::InvalidFormat => {
            (1000, error.user_message())
        }

        // 冲突错误 -> 1005
        DomainError::LoginIdAlreadyExists
        | DomainError::GroupNameAlreadyExists
        | DomainError::UserAlreadyMember
        | DomainError::AlreadyGroupMember => {
            (1005, error.user_message())
        }

        // 认证错误 -> 1002
        DomainError::PasswordVerificationFailed
        | DomainError::InvalidCredentials => {
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

/// 应用层错误到HTTP响应的映射
pub fn map_application_error_to_http_response(error: crate::application::ApplicationError) -> Response {
    use crate::application::ApplicationError;
    
    match error {
        ApplicationError::Domain(domain_error) => map_domain_error_to_http_response(domain_error),
        ApplicationError::Infrastructure { message } => {
            let error_response = json!({
                "code": 5001,
                "msg": format!("基础设施错误: {}", message),
                "data": null
            });
            (StatusCode::OK, Json(error_response)).into_response()
        }
        ApplicationError::ConcurrencyConflict => {
            let error_response = json!({
                "code": 1005,
                "msg": "并发冲突，请重试",
                "data": null
            });
            (StatusCode::OK, Json(error_response)).into_response()
        }
        ApplicationError::TransactionFailed => {
            let error_response = json!({
                "code": 5000,
                "msg": "事务执行失败",
                "data": null
            });
            (StatusCode::OK, Json(error_response)).into_response()
        }
        ApplicationError::EventPublishFailed => {
            let error_response = json!({
                "code": 5000,
                "msg": "事件发布失败",
                "data": null
            });
            (StatusCode::OK, Json(error_response)).into_response()
        }
    }
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

    // 根据hotspot.md规范，统一返回HTTP 200状态码
    (StatusCode::OK, Json(response)).into_response()
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
    next_cursor: Option<String>,
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

/// 错误响应帮助函数 - 已弃用，请使用 map_domain_error_to_http_response
#[deprecated(note = "使用 map_domain_error_to_http_response 代替，以遵循统一的错误码规范")]
pub fn error_response(
    _status_code: StatusCode,
    message: &str,
    data: Option<serde_json::Value>,
) -> Response {
    let response = json!({
        "code": 5000, // 使用通用服务器错误码
        "msg": message,
        "data": data
    });

    // 根据hotspot.md规范，统一返回HTTP 200状态码
    (StatusCode::OK, Json(response)).into_response()
}
